package database

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// NewRedis parses a Redis URL, constructs a client, and verifies connectivity
// with a Ping. It returns an error if the URL is invalid or the server is
// unreachable, so the caller can decide whether to degrade gracefully.
func NewRedis(ctx context.Context, url string) (*redis.Client, error) {
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("database: parse redis url: %w", err)
	}
	client := redis.NewClient(opt)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		// Close the client we just opened so we do not leak a connection.
		_ = client.Close()
		return nil, fmt.Errorf("database: ping redis: %w", err)
	}
	return client, nil
}

// PingRedis verifies the Redis server is reachable. Used by the readiness probe.
func PingRedis(ctx context.Context, client *redis.Client) error {
	if client == nil {
		return fmt.Errorf("database: redis client is nil")
	}
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return client.Ping(pingCtx).Err()
}
