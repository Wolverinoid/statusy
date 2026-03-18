package main

import (
	"context"
	"embed"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/statusy/statusy/config"
	"github.com/statusy/statusy/internal/api"
	"github.com/statusy/statusy/internal/database"
	"github.com/statusy/statusy/internal/metrics"
	"github.com/statusy/statusy/internal/monitors"
	"github.com/statusy/statusy/internal/notifications"
	"github.com/statusy/statusy/internal/setup"
)

//go:embed frontend/dist
var frontendFS embed.FS

func main() {
	// Logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// ── Determine startup mode ────────────────────────────────────────────────
	// If setup is needed (no config file, no env-var DSN+secret), start in
	// "setup mode": serve only the setup API + frontend so the user can
	// configure the app via the browser wizard.
	if setup.IsWebSetupNeeded() {
		slog.Info("no configuration found — starting in web setup mode")
		runSetupMode(logger)
		return
	}

	// ── Normal startup ────────────────────────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	db, err := database.Connect(cfg)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	if err := database.Migrate(db); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	// Seed admin user if none exists (Docker / env-var deployments)
	if err := setup.EnsureAdminExists(db); err != nil {
		slog.Error("failed to seed admin user", "error", err)
		os.Exit(1)
	}

	metricsRegistry := metrics.NewRegistry()
	dispatcher := notifications.NewDispatcher(cfg, logger)
	go dispatcher.Run()

	engine := monitors.NewEngine(db, dispatcher, metricsRegistry, logger)
	go engine.Start()

	router := api.NewRouter(cfg, db, metricsRegistry, dispatcher, frontendFS, logger)

	srv := &http.Server{
		Addr:         cfg.Server.Address,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("server starting", "address", cfg.Server.Address)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-quit
	slog.Info("shutting down...")

	engine.Stop()
	dispatcher.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server shutdown error", "error", err)
	}

	slog.Info("shutdown complete")
}

// runSetupMode starts a minimal HTTP server that serves only the setup wizard.
// Once the user completes setup via the browser, the process exits so the
// container/supervisor can restart it in normal mode.
func runSetupMode(logger *slog.Logger) {
	addr := os.Getenv("STATUSY_SERVER_ADDRESS")
	if addr == "" {
		addr = ":3000"
	}

	router := setup.NewSetupRouter(frontendFS, logger)

	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Setup completion triggers a clean exit so the container restarts in normal mode.
	setup.OnSetupComplete(func() {
		slog.Info("setup complete — restarting in normal mode")
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		srv.Shutdown(ctx) //nolint:errcheck
	})

	go func() {
		slog.Info("setup server starting", "address", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("setup server error", "error", err)
			os.Exit(1)
		}
	}()

	select {
	case <-quit:
		slog.Info("shutting down setup server...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(ctx) //nolint:errcheck
	case <-setup.SetupDone():
		// already shut down inside OnSetupComplete callback
	}
}
