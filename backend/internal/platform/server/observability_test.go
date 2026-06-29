package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newInstrumentedEngine builds a minimal gin engine with the metrics and
// security-headers middleware installed plus a couple of routes, for testing
// the observability/security middleware in isolation.
func newInstrumentedEngine(t *testing.T) (*gin.Engine, *Metrics) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	m := NewMetrics()
	engine := gin.New()
	engine.Use(SecurityHeaders(false))
	engine.Use(m.Middleware())
	engine.GET("/metrics", m.Handler())
	engine.GET("/ping/:id", func(c *gin.Context) { c.String(http.StatusOK, "pong") })
	return engine, m
}

func TestMetrics_IncrementsCountersWithTemplatedPath(t *testing.T) {
	engine, _ := newInstrumentedEngine(t)

	// Hit the parameterized route twice with different IDs; both must collapse
	// onto the same templated label (/ping/:id) to bound cardinality.
	for _, id := range []string{"1", "2"} {
		req := httptest.NewRequest(http.MethodGet, "/ping/"+id, nil)
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)
	}

	// Scrape /metrics and assert the counter is present with the template path.
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	body := rec.Body.String()
	assert.Contains(t, body, "http_requests_total")
	assert.Contains(t, body, "http_request_duration_seconds")
	// Go runtime collector present.
	assert.Contains(t, body, "go_goroutines")
	// Templated path label, value 2 (raw IDs collapsed).
	assert.Contains(t, body,
		`http_requests_total{method="GET",path="/ping/:id",status="200"} 2`)
	// The raw IDs must NOT appear as label values.
	assert.NotContains(t, body, `path="/ping/1"`)
}

func TestSecurityHeaders_Present(t *testing.T) {
	engine, _ := newInstrumentedEngine(t)

	req := httptest.NewRequest(http.MethodGet, "/ping/1", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	h := rec.Header()
	assert.Equal(t, "nosniff", h.Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", h.Get("X-Frame-Options"))
	assert.Equal(t, "no-referrer", h.Get("Referrer-Policy"))
	assert.NotEmpty(t, h.Get("Content-Security-Policy"))
	// No TLS on the test request -> HSTS must be absent.
	assert.Empty(t, h.Get("Strict-Transport-Security"))
}

func TestRateLimit_429AfterThreshold(t *testing.T) {
	gin.SetMode(gin.TestMode)

	const limit = 3
	engine := gin.New()
	engine.Use(RequestID())
	// nil Redis -> in-memory limiter.
	engine.POST("/login", RateLimit(nil, limit, "test"), func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	do := func() (int, string) {
		req := httptest.NewRequest(http.MethodPost, "/login", nil)
		req.RemoteAddr = "203.0.113.7:5555"
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)
		return rec.Code, rec.Body.String()
	}

	// First `limit` requests pass.
	for i := 0; i < limit; i++ {
		code, _ := do()
		require.Equalf(t, http.StatusOK, code, "request %d should pass", i+1)
	}

	// The next one is throttled with the RATE_LIMITED envelope.
	code, body := do()
	require.Equal(t, http.StatusTooManyRequests, code)
	assert.Contains(t, body, CodeRateLimited)
	assert.True(t, strings.Contains(body, `"error"`), "expected error envelope, got: %s", body)
}

func TestRateLimit_DisabledWhenLimitZero(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.POST("/login", RateLimit(nil, 0, "test"), func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	for i := 0; i < 50; i++ {
		req := httptest.NewRequest(http.MethodPost, "/login", nil)
		req.RemoteAddr = "203.0.113.8:5555"
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)
	}
}

func TestRateLimit_PerIPIsolation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const limit = 2
	engine := gin.New()
	engine.Use(RequestID())
	engine.POST("/login", RateLimit(nil, limit, "test"), func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	hit := func(ip string) int {
		req := httptest.NewRequest(http.MethodPost, "/login", nil)
		req.RemoteAddr = ip + ":5555"
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)
		return rec.Code
	}

	// Exhaust IP A.
	require.Equal(t, http.StatusOK, hit("198.51.100.1"))
	require.Equal(t, http.StatusOK, hit("198.51.100.1"))
	require.Equal(t, http.StatusTooManyRequests, hit("198.51.100.1"))
	// A different IP is unaffected.
	require.Equal(t, http.StatusOK, hit("198.51.100.2"))
}
