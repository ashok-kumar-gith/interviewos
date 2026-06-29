package ai

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache is the optional response cache port (§9: identical asks don't re-bill).
// A nil Cache disables caching entirely.
type Cache interface {
	Get(ctx context.Context, key string) (string, bool)
	Set(ctx context.Context, key, value string, ttl time.Duration)
}

// redisCache is a Redis-backed Cache. It degrades gracefully: any Redis error is
// treated as a miss / no-op so a Redis outage never breaks an AI feature.
type redisCache struct {
	rdb *redis.Client
}

// NewRedisCache returns a Redis-backed Cache, or nil when rdb is nil (caching
// disabled).
func NewRedisCache(rdb *redis.Client) Cache {
	if rdb == nil {
		return nil
	}
	return &redisCache{rdb: rdb}
}

func (c *redisCache) Get(ctx context.Context, key string) (string, bool) {
	v, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		return "", false
	}
	return v, true
}

func (c *redisCache) Set(ctx context.Context, key, value string, ttl time.Duration) {
	_ = c.rdb.Set(ctx, key, value, ttl).Err()
}

// cacheKey builds a stable cache key from the feature and the normalized prompt
// inputs (§9: key = hash(template + normalized_inputs)).
func cacheKey(feature Feature, model, system, prompt string) string {
	h := sha256.New()
	h.Write([]byte(model))
	h.Write([]byte{0})
	h.Write([]byte(system))
	h.Write([]byte{0})
	h.Write([]byte(prompt))
	return "ai:" + string(feature) + ":" + hex.EncodeToString(h.Sum(nil))
}
