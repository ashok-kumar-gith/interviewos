package server

import (
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// requestIDHeader is the canonical correlation-ID header.
const requestIDHeader = "X-Request-ID"

// requestIDKey is the gin context key under which the request ID is stored.
const requestIDKey = "request_id"

// RequestID middleware ensures every request carries a correlation ID. It
// reuses an inbound X-Request-ID when present, otherwise generates a UUID, and
// echoes it back on the response so clients and logs can correlate.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader(requestIDHeader)
		if rid == "" {
			rid = uuid.NewString()
		}
		c.Set(requestIDKey, rid)
		c.Writer.Header().Set(requestIDHeader, rid)
		c.Next()
	}
}

// RequestIDFromContext returns the correlation ID stored on the gin context.
func RequestIDFromContext(c *gin.Context) string {
	if v, ok := c.Get(requestIDKey); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// Logger middleware emits one structured JSON log line per request carrying
// method, path, status, latency, client IP, and the request ID.
func Logger(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		if raw != "" {
			path = path + "?" + raw
		}

		fields := []zap.Field{
			zap.String("request_id", RequestIDFromContext(c)),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.Int("status", c.Writer.Status()),
			zap.Int64("latency_ms", latency.Milliseconds()),
			zap.String("client_ip", c.ClientIP()),
			zap.Int("bytes", c.Writer.Size()),
		}
		if len(c.Errors) > 0 {
			fields = append(fields, zap.String("errors", c.Errors.String()))
		}

		status := c.Writer.Status()
		switch {
		case status >= 500:
			log.Error("request", fields...)
		case status >= 400:
			log.Warn("request", fields...)
		default:
			log.Info("request", fields...)
		}
	}
}

// Recovery middleware converts panics into a 500 JSON error envelope without
// leaking internals, logging the panic with the request ID for diagnosis.
func Recovery(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.Error("panic recovered",
					zap.String("request_id", RequestIDFromContext(c)),
					zap.Any("panic", r),
					zap.String("path", c.Request.URL.Path),
				)
				c.AbortWithStatusJSON(500, gin.H{
					"error": gin.H{
						"code":       "INTERNAL",
						"message":    "internal server error",
						"request_id": RequestIDFromContext(c),
					},
				})
			}
		}()
		c.Next()
	}
}

// SecurityHeaders middleware sets conservative response security headers
// suitable for a JSON API that also serves the self-contained Swagger UI page.
// HSTS is only emitted when the request is served over TLS (or behind a TLS
// terminator that sets X-Forwarded-Proto: https), since asserting it on plain
// HTTP is meaningless and can lock out clients in dev.
//
// The Content-Security-Policy is permissive enough for the Swagger UI page,
// which loads swagger-ui-dist assets from a CDN and inlines bootstrap script;
// for API (JSON) responses these directives are inert.
func SecurityHeaders(isProd bool) gin.HandlerFunc {
	const csp = "default-src 'self'; " +
		"script-src 'self' 'unsafe-inline' https://unpkg.com https://cdn.jsdelivr.net; " +
		"style-src 'self' 'unsafe-inline' https://unpkg.com https://cdn.jsdelivr.net; " +
		"img-src 'self' data: https:; " +
		"font-src 'self' data: https:; " +
		"connect-src 'self'; " +
		"frame-ancestors 'none'; " +
		"base-uri 'self'"

	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Content-Security-Policy", csp)

		// HSTS only when the connection is actually secure.
		if c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https") {
			if isProd {
				h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
			} else {
				h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}
		}

		c.Next()
	}
}

// CORS middleware applies a strict origin allowlist. An empty allowlist falls
// back to denying cross-origin requests (no wildcard in production).
func CORS(origins []string) gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Idempotency-Key", requestIDHeader},
		ExposeHeaders:    []string{requestIDHeader},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})
}
