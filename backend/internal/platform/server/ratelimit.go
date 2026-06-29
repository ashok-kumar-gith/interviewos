package server

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// rateLimiter is the backend-agnostic limiter contract: Allow reports whether a
// request from key may proceed within the current fixed window, and returns the
// number of seconds until the window resets (for Retry-After).
type rateLimiter interface {
	Allow(ctx context.Context, key string) (allowed bool, retryAfter int)
}

// RateLimit builds a gin middleware enforcing a fixed-window per-IP budget. When
// rdb is non-nil it uses a Redis-backed limiter (correct across replicas);
// otherwise it falls back to a process-local in-memory limiter. A limit <= 0
// disables the middleware (returns a no-op).
func RateLimit(rdb *redis.Client, limit int, scope string) gin.HandlerFunc {
	if limit <= 0 {
		return func(c *gin.Context) { c.Next() }
	}
	window := time.Minute

	var lim rateLimiter
	if rdb != nil {
		lim = &redisLimiter{rdb: rdb, limit: limit, window: window, scope: scope}
	} else {
		lim = newMemoryLimiter(limit, window, scope)
	}

	return func(c *gin.Context) {
		key := c.ClientIP()
		allowed, retryAfter := lim.Allow(c.Request.Context(), key)
		if !allowed {
			if retryAfter > 0 {
				c.Header("Retry-After", strconv.Itoa(retryAfter))
			}
			AbortError(c, http.StatusTooManyRequests, CodeRateLimited,
				"too many requests; please retry later", nil)
			return
		}
		c.Next()
	}
}

// ---- Redis-backed fixed-window limiter ----

type redisLimiter struct {
	rdb    *redis.Client
	limit  int
	window time.Duration
	scope  string
}

func (r *redisLimiter) Allow(ctx context.Context, key string) (bool, int) {
	bucket := time.Now().UnixNano() / int64(r.window)
	redisKey := "ratelimit:" + r.scope + ":" + key + ":" + strconv.FormatInt(bucket, 10)

	pipe := r.rdb.TxPipeline()
	incr := pipe.Incr(ctx, redisKey)
	pipe.Expire(ctx, redisKey, r.window)
	if _, err := pipe.Exec(ctx); err != nil {
		// Fail open: never let a Redis hiccup take down auth endpoints.
		return true, 0
	}

	count := incr.Val()
	if count > int64(r.limit) {
		// Seconds remaining in the current window.
		elapsed := time.Now().UnixNano() % int64(r.window)
		retryAfter := int((r.window - time.Duration(elapsed)).Seconds())
		if retryAfter < 1 {
			retryAfter = 1
		}
		return false, retryAfter
	}
	return true, 0
}

// ---- In-memory fixed-window limiter (fallback / tests) ----

type memoryLimiter struct {
	limit  int
	window time.Duration

	mu      sync.Mutex
	buckets map[string]*memBucket
	lastGC  time.Time
}

type memBucket struct {
	count   int
	resetAt time.Time
}

func newMemoryLimiter(limit int, window time.Duration, _ string) *memoryLimiter {
	return &memoryLimiter{
		limit:   limit,
		window:  window,
		buckets: make(map[string]*memBucket),
		lastGC:  time.Now(),
	}
}

func (m *memoryLimiter) Allow(_ context.Context, key string) (bool, int) {
	now := time.Now()

	m.mu.Lock()
	defer m.mu.Unlock()

	m.gcLocked(now)

	b, ok := m.buckets[key]
	if !ok || now.After(b.resetAt) {
		m.buckets[key] = &memBucket{count: 1, resetAt: now.Add(m.window)}
		return true, 0
	}

	b.count++
	if b.count > m.limit {
		retryAfter := int(time.Until(b.resetAt).Seconds())
		if retryAfter < 1 {
			retryAfter = 1
		}
		return false, retryAfter
	}
	return true, 0
}

// gcLocked drops expired buckets at most once per window to bound memory.
func (m *memoryLimiter) gcLocked(now time.Time) {
	if now.Sub(m.lastGC) < m.window {
		return
	}
	for k, b := range m.buckets {
		if now.After(b.resetAt) {
			delete(m.buckets, k)
		}
	}
	m.lastGC = now
}
