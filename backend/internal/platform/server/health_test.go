package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/interviewos/backend/internal/platform/config"
)

func newTestEngine(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	return New(Options{
		Config: &config.Config{
			Env:         "development",
			CORSOrigins: []string{"http://localhost:3000"},
		},
		Logger: zap.NewNop(),
		DB:     nil, // graceful degradation: no DB in unit test
		Redis:  nil, // graceful degradation: no Redis in unit test
	})
}

func TestHealthz_OK(t *testing.T) {
	engine := newTestEngine(t)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "ok", body["status"])
}

func TestReadyz_NotReadyWhenDependenciesMissing(t *testing.T) {
	engine := newTestEngine(t)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	// With nil DB and Redis, readiness must report 503.
	require.Equal(t, http.StatusServiceUnavailable, rec.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "not_ready", body["status"])
}
