package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chiware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/contextpilot/backend/internal/appshell"
	"github.com/contextpilot/backend/internal/auth"
	"github.com/contextpilot/backend/internal/briefing"
	"github.com/contextpilot/backend/internal/config"
	"github.com/contextpilot/backend/internal/handler"
	"github.com/contextpilot/backend/internal/memory"
	"github.com/contextpilot/backend/internal/middleware"
	"github.com/contextpilot/backend/internal/migrate"
	"github.com/contextpilot/backend/internal/meeting"
	"github.com/contextpilot/backend/internal/upcoming"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	cfg := config.Load()

	// Connect to PostgreSQL if DATABASE_URL is provided.
	// Phase 1 can run without DB for the app shell; meeting import requires it.
	var pool *pgxpool.Pool
	if cfg.DatabaseURL != "" {
		var err error
		// ARCH_OK: database connection setup — root context for pgxpool initialization, not a per-request context
		pool, err = pgxpool.New(context.Background(), cfg.DatabaseURL)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to connect to postgres")
		}
		defer pool.Close()
		log.Info().Msg("connected to postgres")
		// Hardcode absolute path: binary lives at /app/server, migrations at /app/db/migrations.
		migrationsPath := "/app/db/migrations"
		// ARCH_OK: startup-only migration — no request context available, short-lived with timeout in migrate.Run
		if os.Getenv("SKIP_MIGRATIONS") != "1" {
			if err := migrate.Run(
				// ARCH_OK: startup-only, bounded by migrate.Run's internal timeout
				context.Background(), cfg.DatabaseURL, migrationsPath); err != nil {
				log.Fatal().Err(err).Msg("failed to run migrations")
			}
		} else {
			log.Info().Msg("skipping migrations (SKIP_MIGRATIONS=1)")
		}
	} else {
		log.Info().Msg("DATABASE_URL not set; running without DB (app-shell only)")
	}

	// CSRF middleware (CWE-352) — Origin/Referer allowlist applied to all
	// state-changing methods (POST/PUT/PATCH/DELETE) at the chi router
	// layer. Defense-in-depth alongside the SvelteKit /api/* hooks-layer
	// check. Only installed when CSRF_ALLOWED_ORIGINS is non-empty
	// (empty = disabled for dev/fallback).
	var csrfMiddleware func(http.Handler) http.Handler
	if len(cfg.CSRFAllowedOrigins) > 0 {
		csrfMiddleware = middleware.NewCSRF(
			log.Logger,
			middleware.CSRFConfig{
				AllowedOrigins: cfg.CSRFAllowedOrigins,
				Enabled:        true,
			},
		)
		log.Info().
			Strs("allowed_origins", cfg.CSRFAllowedOrigins).
			Msg("CSRF middleware enabled")
	}

	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.WithRequestID)
	r.Use(middleware.Recover(log.Logger))
	r.Use(chiware.Logger)
	r.Use(chiware.Timeout(30 * time.Second))

	// CSRF middleware — only active when CSRF_ALLOWED_ORIGINS is set.
	// GET/HEAD/OPTIONS pass through untouched (zero impact on read traffic).
	if csrfMiddleware != nil {
		r.Use(csrfMiddleware)
	}

	// Health endpoints
	r.Get("/healthz", handler.Healthz)
	r.Get("/live", handler.Live)
	r.Get("/ready", func(w http.ResponseWriter, r *http.Request) {
		handler.Ready(r.Context()).ServeHTTP(w, r)
	})

	// App shell routes — all gated by FF_ENABLE_APP_SHELL (defaults false).
	r.Mount("/api/v1/appshell", appshell.Handler(&log.Logger))

	// Auth routes — login, signup, logout. Gated by FF_ENABLE_APP_SHELL.
	r.Mount("/api/v1/auth", auth.Handler(&log.Logger))

	// Briefing background worker — starts when FF_ENABLE_PRE_CALL_BRIEFING=true.
	// Worker goroutine is stopped via briefingWorker.Stop() during graceful shutdown.
	var briefingWorker *briefing.Worker
	// Memory background worker — starts when FF_ENABLE_MEETING_MEMORY_PROCESSING=true.
	// Worker goroutine is stopped via memWorker.Stop() during graceful shutdown.
	var memWorker *memory.Worker

	// Rate limiting middleware for POST /meetings (100 req/min per IP by default).
	// Configured via RATE_LIMIT_IMPORT_MAX_REQUESTS and RATE_LIMIT_IMPORT_WINDOW_SECS.
	//
	// SECURITY (CWE-770): the middleware is ALWAYS installed for the import
	// path when the config value is positive. config.Load already coerces
	// 0 to the documented default, but this gate is the second layer of
	// defense: a negative value means "explicitly disabled" (escape hatch
	// for incident response); anything else installs the middleware. This
	// prevents a future regression where an operator sets 0 and silently
	// disables rate limiting, as observed in audit task t_6cc2471a.
	var importRateLimitMiddleware func(http.Handler) http.Handler
	if cfg.RateLimitImportMaxRequests < 0 {
		log.Warn().
			Int("max_requests", cfg.RateLimitImportMaxRequests).
			Msg("RATE_LIMIT_IMPORT_MAX_REQUESTS is negative — import rate limiting is EXPLICITLY DISABLED (incident-response escape hatch)")
	} else {
		if cfg.RateLimitImportMaxRequests == 0 {
			// Should be impossible because config.Load coerces 0 to the
			// default, but log it loudly if it ever happens so we notice.
			log.Warn().Msg("RATE_LIMIT_IMPORT_MAX_REQUESTS=0 reached main gate; defaulting to documented default")
		}
		importRateLimitMiddleware = middleware.NewRateLimiter(
			log.Logger,
			middleware.RateLimiterConfig{
				MaxRequests: cfg.RateLimitImportMaxRequests,
				Window:      time.Duration(cfg.RateLimitImportWindowSecs) * time.Second,
				Enabled:     true,
				KeyPrefix:   "rl:import:",
			},
			cfg.RedisURL,
			middleware.DefaultClientIPKey,
		)
	}

	// Upcoming meetings — gated by FF_ENABLE_UPCOMING_MEETINGS (defaults false).
	// Only mounted if pool != nil (DB available).
	if pool != nil {
		// Memory processing — gated by FF_ENABLE_MEETING_MEMORY_PROCESSING
		// (defaults false). Start the background worker BEFORE building the
		// meeting router so we can wire the worker into the memory sub-router
		// that lives at /api/v1/meetings/{id}/memory/.... The memory sub-router
		// is nested inside the meeting sub-router (NOT mounted at /api/v1) so
		// chi's longest-prefix mount does not shadow the /meetings/{id}/...
		// sub-paths. The previous approach — r.Mount("/api/v1", memory.Handler)
		// — was unreachable at runtime because the meeting mount at
		// /api/v1/meetings captured every /api/v1/meetings/... request and
		// 404'd inside the meeting router before the flag-mismatch guard ran.
		if memory.IsFeatureFlagEnabled() {
			memRepo := memory.NewRepository(pool)
			memWorker = memory.NewWorker(
				memRepo,
				&memory.DefaultMemoryProcessor{},
				&log.Logger,
				memory.DefaultWorkerConfig(),
			)
			// ARCH_OK: server-lifetime context for background worker startup — same pattern as pgxpool.New
			memWorker.Start(context.Background())
		}

		// Build the meeting router and nest the memory sub-router when the
		// feature flag is on. When the flag is off, mountSubRoute is nil and
		// the meeting router is identical to the pre-fix shape.
		var mountSubRoute func(chi.Router)
		if memWorker != nil {
			mountSubRoute = func(sub chi.Router) {
				memory.RegisterRoutesOnRouter(sub, &log.Logger, pool, memWorker)
			}
		}
		r.Mount("/api/v1/meetings", meeting.HandlerWithSubRoute(&log.Logger, pool, importRateLimitMiddleware, mountSubRoute))

		// Briefing worker — gated by FF_ENABLE_PRE_CALL_BRIEFING (defaults false).
		// Instantiates after BriefingService is ready to process jobs from
		// briefing_processing_jobs queue. Created BEFORE the upcoming router
		// so we can pass the handler to upcoming's sub-route callback.
		if briefing.IsFeatureFlagEnabled() {
			// briefingSourceAdapter implements briefing.MeetingSourceRepo using the same pool,
			// bridging the briefing service's source-selection queries to the meeting tables.
			briefingSrcRepo := &briefingSourceAdapter{pool: pool}
			briefingRepo := briefing.NewRepository(pool)
			prodService := briefing.NewProductionBriefingService(briefingRepo, briefingSrcRepo, &log.Logger)
			briefingWorker = briefing.NewWorker(
				briefingRepo,
				prodService,
				&log.Logger,
				briefing.DefaultWorkerConfig(),
			)
			// ARCH_OK: server-lifetime context for background worker startup — same pattern as pgxpool.New
			briefingWorker.Start(context.Background())
		}

		// Build the upcoming router and nest the briefing sub-router when the
		// briefing feature flag is on. This solves chi's route-shadowing problem:
		// mounting briefing at /api/v1 (outside upcoming) would shadow every
		// /api/v1/upcoming/{meetingId}/briefing/... request (404) because chi's
		// radix tree matches the more-specific /api/v1/upcoming mount first and
		// the briefing handler is never reached. By registering briefing paths
		// directly on the upcoming router (not inside a Route sub-scope), both
		// upcoming and briefing routes coexist without shadowing.
		var upcomingSubRoute func(chi.Router)
		if briefingWorker != nil {
			upcomingSubRoute = func(r chi.Router) {
				// Register briefing paths directly on the upcoming router at
				// /{meetingId}/briefing/... so both upcoming ({id}, /{id}/cancel)
				// and briefing coexist without chi's Route sub-scope causing
				// path-segment duplication (which produces 404).
				briefing.RegisterRoutesOnRouter(r, &log.Logger, pool, briefingWorker)
			}
		}
		r.Mount("/api/v1/upcoming", upcoming.HandlerWithSubRoute(&log.Logger, pool, upcomingSubRoute))
	}

	addr := fmt.Sprintf(":%s", cfg.ServerPort)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		// ARCH_OK: signal-handler scope; not a request context
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if briefingWorker != nil {
			briefingWorker.Stop()
		}
		if memWorker != nil {
			memWorker.Stop()
		}
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Error().Err(err).Msg("server shutdown error")
		}
	}()

	log.Info().Str("addr", addr).Msg("ContextPilot server starting")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Error().Err(err).Msg("server error")
		os.Exit(1)
	}
}
