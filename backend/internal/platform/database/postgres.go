// Package database provides connectors for the backing data stores:
// a pooled GORM/PostgreSQL handle and a Redis client wrapper. Both expose
// health checks (Ping) used by the readiness probe.
package database

import (
	"context"
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// PostgresConfig tunes the GORM connection pool.
type PostgresConfig struct {
	// DSN is the PostgreSQL connection URL.
	DSN string
	// MaxOpenConns caps the total open connections to Postgres.
	MaxOpenConns int
	// MaxIdleConns caps the idle connections retained in the pool.
	MaxIdleConns int
	// ConnMaxLifetime bounds how long a connection may be reused.
	ConnMaxLifetime time.Duration
	// LogLevel controls GORM's SQL logging verbosity.
	LogLevel gormlogger.LogLevel
}

// DefaultPostgresConfig returns pool defaults sized for a single API replica.
func DefaultPostgresConfig(dsn string) PostgresConfig {
	return PostgresConfig{
		DSN:             dsn,
		MaxOpenConns:    25,
		MaxIdleConns:    10,
		ConnMaxLifetime: 30 * time.Minute,
		LogLevel:        gormlogger.Warn,
	}
}

// NewPostgres opens a pooled GORM connection to PostgreSQL, applies pool
// settings, and verifies connectivity with a Ping. It returns an error if the
// connection cannot be established or pinged.
func NewPostgres(ctx context.Context, cfg PostgresConfig) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.DSN), &gorm.Config{
		Logger:                 gormlogger.Default.LogMode(cfg.LogLevel),
		SkipDefaultTransaction: true,
	})
	if err != nil {
		return nil, fmt.Errorf("database: open postgres: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("database: access sql.DB: %w", err)
	}
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(pingCtx); err != nil {
		return nil, fmt.Errorf("database: ping postgres: %w", err)
	}
	return db, nil
}

// PingPostgres verifies the database is reachable. Used by the readiness probe.
func PingPostgres(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database: postgres handle is nil")
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("database: access sql.DB: %w", err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return sqlDB.PingContext(pingCtx)
}

// ClosePostgres releases the underlying connection pool.
func ClosePostgres(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
