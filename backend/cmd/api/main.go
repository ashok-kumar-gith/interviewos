// Command api is the composition root for the InterviewOS backend. It loads
// configuration, initializes the structured logger, connects the backing
// stores (PostgreSQL + Redis) with graceful degradation, builds the Gin
// router, and serves HTTP with graceful shutdown on SIGINT/SIGTERM.
package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/redis/go-redis/v9"

	"github.com/interviewos/backend/internal/platform/config"
	"github.com/interviewos/backend/internal/platform/database"
	"github.com/interviewos/backend/internal/platform/logger"
	"github.com/interviewos/backend/internal/platform/server"
)

func main() {
	if err := run(); err != nil {
		// Logger may not be available yet; write to stderr and exit non-zero.
		os.Stderr.WriteString("fatal: " + err.Error() + "\n")
		os.Exit(1)
	}
}

func run() error {
	// 1. Configuration (fail fast on invalid/missing required values).
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// 2. Structured logger.
	log, err := logger.New(cfg.Env, cfg.LogLevel)
	if err != nil {
		return err
	}
	defer func() { _ = log.Sync() }()

	log.Info("starting interviewos api",
		zap.String("env", cfg.Env),
		zap.String("port", cfg.Port),
	)

	// 3. Backing stores with graceful degradation: a failure to connect is
	// logged as a warning and the process continues. Readiness will report the
	// dependency as down until it recovers.
	ctx := context.Background()

	var db *gorm.DB
	if conn, derr := database.NewPostgres(ctx, database.DefaultPostgresConfig(cfg.DatabaseURL)); derr != nil {
		log.Warn("postgres unavailable; continuing in degraded mode", zap.Error(derr))
	} else {
		db = conn
		log.Info("connected to postgres")
	}

	var rdb *redis.Client
	if conn, rerr := database.NewRedis(ctx, cfg.RedisURL); rerr != nil {
		log.Warn("redis unavailable; continuing in degraded mode", zap.Error(rerr))
	} else {
		rdb = conn
		log.Info("connected to redis")
	}

	// 4. Build the router.
	engine := server.New(server.Options{
		Config: cfg,
		Logger: log,
		DB:     db,
		Redis:  rdb,
	})

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           engine,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// 5. Serve with graceful shutdown.
	serverErr := make(chan error, 1)
	go func() {
		log.Info("http server listening", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		return err
	case sig := <-quit:
		log.Info("shutdown signal received; draining", zap.String("signal", sig.String()))
	}

	// Drain in-flight requests, then close backing connections.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("graceful shutdown failed", zap.Error(err))
		_ = srv.Close()
	}

	if db != nil {
		if err := database.ClosePostgres(db); err != nil {
			log.Warn("closing postgres", zap.Error(err))
		}
	}
	if rdb != nil {
		if err := rdb.Close(); err != nil {
			log.Warn("closing redis", zap.Error(err))
		}
	}

	log.Info("shutdown complete")
	return nil
}
