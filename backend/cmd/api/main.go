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

	"github.com/interviewos/backend/internal/ai"
	"github.com/interviewos/backend/internal/analytics"
	"github.com/interviewos/backend/internal/auth"
	"github.com/interviewos/backend/internal/backendeng"
	"github.com/interviewos/backend/internal/behavioral"
	"github.com/interviewos/backend/internal/company"
	"github.com/interviewos/backend/internal/content"
	"github.com/interviewos/backend/internal/designproblems"
	"github.com/interviewos/backend/internal/intake"
	"github.com/interviewos/backend/internal/lld"
	"github.com/interviewos/backend/internal/mock"
	"github.com/interviewos/backend/internal/notification"
	"github.com/interviewos/backend/internal/platform/config"
	"github.com/interviewos/backend/internal/platform/database"
	"github.com/interviewos/backend/internal/platform/logger"
	"github.com/interviewos/backend/internal/platform/server"
	"github.com/interviewos/backend/internal/progress"
	"github.com/interviewos/backend/internal/resume"
	"github.com/interviewos/backend/internal/revision"
	"github.com/interviewos/backend/internal/roadmap"
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

	// 4. Compose feature modules (clean DI) and build the router.
	var registrars []server.RouteRegistrar
	if db != nil {
		// Shared JWT token manager (used by auth + every protected module's
		// RequireAuth middleware).
		tokens, terr := buildTokenManager(cfg)
		if terr != nil {
			return terr
		}

		registrars = append(registrars, buildAuthModule(cfg, log, db, rdb, tokens))

		// Content / Curriculum browsing module (read-only). The content repository
		// is shared with the Backend Engineering module below (which reuses it
		// rather than duplicating read logic).
		contentRepo := content.NewRepository(db)
		registrars = append(registrars,
			content.NewHandler(content.NewService(contentRepo)))

		// Backend Engineering depth catalog — read-only, public. Serves the
		// dedicated /backend-engineering/topics path over the shared content repo,
		// pinned to the backend_engineering pillar.
		registrars = append(registrars,
			backendeng.NewHandler(backendeng.NewService(contentRepo)))

		// Intake / profile module.
		registrars = append(registrars, intake.NewHandler(intake.HandlerConfig{
			Service: intake.NewService(intake.ServiceConfig{Repo: intake.NewRepository(db)}),
			Tokens:  tokens,
		}))

		// Behavioral (STAR stories) module.
		registrars = append(registrars, behavioral.NewHandler(behavioral.HandlerConfig{
			Service: behavioral.NewService(behavioral.ServiceConfig{Repo: behavioral.NewRepository(db)}),
			Tokens:  tokens,
		}))

		// Resume module.
		registrars = append(registrars, resume.NewHandler(resume.HandlerConfig{
			Service: resume.NewService(resume.ServiceConfig{Repo: resume.NewRepository(db)}),
			Tokens:  tokens,
		}))

		// Roadmap / Curriculum Engine module. The deterministic curriculum engine
		// (internal/curriculum) is composed here behind the roadmap service via the
		// ProfileReader + ContentReader read ports.
		registrars = append(registrars, roadmap.NewHandler(roadmap.HandlerConfig{
			Service: roadmap.NewService(roadmap.ServiceConfig{
				Repo:     roadmap.NewRepository(db),
				Profiles: roadmap.NewProfileReader(db),
				Content:  roadmap.NewContentReader(db),
			}),
			Tokens: tokens,
		}))

		// Revision Engine (spaced repetition). Built before progress so it can be
		// injected as the optional learning-completion scheduler (ADR D1/D2/D3,
		// SRS §6.1). It also serves GET /revision/due and POST /revision/{id}/recall.
		revisionSvc := revision.NewService(revision.ServiceConfig{Repo: revision.NewRepository(db)})
		registrars = append(registrars, revision.NewHandler(revision.HandlerConfig{
			Service: revisionSvc,
			Tokens:  tokens,
		}))

		// Progress / Today / Dashboard module. Owns task completion (transactional
		// progress + session + streak upserts), the Today view, and the dashboard
		// aggregate (readiness via the SRS multiplicative form, ADR D15). On
		// learning-task completion it schedules a revision via revisionSvc.
		registrars = append(registrars, progress.NewHandler(progress.HandlerConfig{
			Service: progress.NewService(progress.ServiceConfig{
				Repo:     progress.NewRepository(db),
				Revision: revisionSvc,
			}),
			Tokens: tokens,
		}))

		// Design Problems (HLD) catalog — read-only, public.
		registrars = append(registrars,
			designproblems.NewHandler(designproblems.NewService(designproblems.NewRepository(db))))

		// LLD problems catalog — read-only, public.
		registrars = append(registrars,
			lld.NewHandler(lld.NewService(lld.NewRepository(db))))

		// Mock Interview module (user-scoped).
		registrars = append(registrars, mock.NewHandler(mock.HandlerConfig{
			Service: mock.NewService(mock.ServiceConfig{Repo: mock.NewRepository(db)}),
			Tokens:  tokens,
		}))

		// Notifications module (user-scoped, in-app channel).
		registrars = append(registrars, notification.NewHandler(notification.HandlerConfig{
			Service: notification.NewService(notification.ServiceConfig{Repo: notification.NewRepository(db)}),
			Tokens:  tokens,
		}))

		// Analytics module (readiness, snapshots, weak/strong topics, time-spent).
		registrars = append(registrars, analytics.NewHandler(analytics.HandlerConfig{
			Service: analytics.NewService(analytics.ServiceConfig{Repo: analytics.NewRepository(db)}),
			Tokens:  tokens,
		}))

		// Company Mode (set/get target company; re-weights roadmap generation).
		registrars = append(registrars, company.NewHandler(company.HandlerConfig{
			Service: company.NewService(company.ServiceConfig{Repo: company.NewRepository(db)}),
			Tokens:  tokens,
		}))

		// AI Assistants (M4). The orchestrator calls Claude when an API key is
		// present and AI is enabled; otherwise — or on any error/timeout — it
		// serves the deterministic fallback (reusing the behavioral improver,
		// resume scorer, and mock weakness detector). Every call records an
		// ai_invocations row. AI is an augmentation, never a P0 dependency (§9).
		var aiClient ai.Client
		if cfg.AIEnabled && cfg.AnthropicAPIKey != "" {
			ac, aerr := ai.NewAnthropicClient(ai.AnthropicConfig{APIKey: cfg.AnthropicAPIKey, Model: cfg.AIModel})
			if aerr != nil {
				log.Warn("ai: anthropic client unavailable; using deterministic fallback", zap.Error(aerr))
			} else {
				aiClient = ac
				log.Info("ai: anthropic client enabled", zap.String("model", cfg.AIModel))
			}
		} else {
			log.Info("ai: running in deterministic-fallback mode (no api key or disabled)")
		}
		aiReaders := ai.NewReaders(db)
		registrars = append(registrars, ai.NewHandler(ai.HandlerConfig{
			Service: ai.NewService(ai.ServiceConfig{
				Client:   aiClient,
				Cache:    ai.NewRedisCache(rdb),
				Repo:     ai.NewRepository(db),
				Enabled:  cfg.AIEnabled,
				Model:    cfg.AIModel,
				Profiles: aiReaders,
				Plans:    aiReaders,
				Stories:  aiReaders,
				Resumes:  aiReaders,
				Mocks:    aiReaders,
				Topics:   aiReaders,
				Designs:  aiReaders,
			}),
			Tokens: tokens,
		}))
	} else {
		log.Warn("feature modules not mounted: database unavailable")
	}

	engine := server.New(server.Options{
		Config:     cfg,
		Logger:     log,
		DB:         db,
		Redis:      rdb,
		Registrars: registrars,
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

// buildTokenManager constructs the shared JWT token manager from configuration.
func buildTokenManager(cfg *config.Config) (*auth.TokenManager, error) {
	return auth.NewTokenManager(auth.TokenManagerConfig{
		Secret:     cfg.JWTSecret,
		AccessTTL:  cfg.AccessTokenTTL,
		RefreshTTL: cfg.RefreshTokenTTL,
		ResetTTL:   cfg.ResetTokenTTL,
		Issuer:     "interviewos",
	})
}

// buildAuthModule wires the auth repository, mailer, OAuth registry, service,
// and HTTP handler from configuration and the shared token manager.
func buildAuthModule(cfg *config.Config, log *zap.Logger, db *gorm.DB, rdb *redis.Client, tokens *auth.TokenManager) *auth.Handler {
	repo := auth.NewRepository(db)
	mailer := auth.NewLogMailer(log)
	// No live OAuth credentials locally: register unconfigured providers so the
	// callback route exists and returns a clear 501.
	oauthReg := auth.NewOAuthRegistry(
		auth.NewUnconfiguredProvider(auth.ProviderGoogle),
		auth.NewUnconfiguredProvider(auth.ProviderGitHub),
	)
	svc := auth.NewService(auth.ServiceConfig{
		Repo:   repo,
		Tokens: tokens,
		Mailer: mailer,
		OAuth:  oauthReg,
		Logger: log,
	})
	// Stricter per-IP rate limit on credential-sensitive auth endpoints. Uses
	// Redis when available (correct across replicas), in-memory otherwise.
	authRL := server.RateLimit(rdb, cfg.AuthRateLimitPerMin, "auth")

	return auth.NewHandler(auth.HandlerConfig{
		Service:       svc,
		Tokens:        tokens,
		SecureCookies: cfg.IsProduction(),
		RateLimit:     authRL,
	})
}
