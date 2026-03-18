package api

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/statusy/statusy/config"
	"github.com/statusy/statusy/internal/auth"
	"github.com/statusy/statusy/internal/metrics"
	"github.com/statusy/statusy/internal/monitors"
	"github.com/statusy/statusy/internal/notifications"
	"gorm.io/gorm"
)

// NewRouter builds and returns the main Chi router.
func NewRouter(
	cfg *config.Config,
	db *gorm.DB,
	metricsReg *metrics.Registry,
	dispatcher *notifications.Dispatcher,
	frontendFS embed.FS,
	logger *slog.Logger,
) http.Handler {
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Logger)

	authHandler := auth.NewHandler(db, cfg)
	monitorHandler := monitors.NewHandler(db, dispatcher, logger)

	// ── Setup status (always returns needed:false in normal mode) ─────────────
	r.Get("/api/setup/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"needed": false}) //nolint:errcheck
	})

	// ── Public auth routes ────────────────────────────────────────────────────
	r.Route("/api/auth", func(r chi.Router) {
		r.Post("/login", authHandler.Login)
		r.Post("/refresh", authHandler.Refresh)
	})

	// ── Authenticated API routes ──────────────────────────────────────────────
	r.Group(func(r chi.Router) {
		r.Use(Authenticate(cfg, db))

		r.Post("/api/auth/logout", authHandler.Logout)
		r.Get("/api/auth/me", authHandler.Me)

		// Monitors
		r.Route("/api/monitors", func(r chi.Router) {
			r.Get("/", monitorHandler.List)
			r.Post("/", monitorHandler.Create)
			r.Get("/{id}", monitorHandler.Get)
			r.Put("/{id}", monitorHandler.Update)
			r.Delete("/{id}", monitorHandler.Delete)
			r.Post("/{id}/pause", monitorHandler.Pause)
			r.Post("/{id}/resume", monitorHandler.Resume)
			r.Get("/{id}/history", monitorHandler.History)
		})

		// Notifications (admin only)
		r.Route("/api/notifications", func(r chi.Router) {
			r.Use(RequireAdmin)
			r.Get("/", monitorHandler.ListNotifications)
			r.Post("/", monitorHandler.CreateNotification)
			r.Put("/{id}", monitorHandler.UpdateNotification)
			r.Delete("/{id}", monitorHandler.DeleteNotification)
			r.Post("/{id}/test", monitorHandler.TestNotification)
		})

		// Users (admin only)
		r.Route("/api/users", func(r chi.Router) {
			r.Use(RequireAdmin)
			r.Get("/", listUsers(db))
			r.Post("/", createUser(db))
			r.Put("/{id}", updateUser(db))
			r.Delete("/{id}", deleteUser(db))
		})

		// LDAP test (admin only)
		r.With(RequireAdmin).Post("/api/admin/ldap/test", ldapTest(cfg))

		// Status pages
		r.Route("/api/status-pages", func(r chi.Router) {
			r.Get("/", listStatusPages(db))
			r.Post("/", createStatusPage(db))
			r.Get("/{id}", getStatusPage(db))
			r.Put("/{id}", updateStatusPage(db))
			r.Delete("/{id}", deleteStatusPage(db))
		})
	})

	// ── Prometheus metrics ────────────────────────────────────────────────────
	if cfg.Metrics.Enabled {
		r.Get("/metrics", metricsHandler(cfg, metricsReg))
	}

	// ── Health check ──────────────────────────────────────────────────────────
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`)) //nolint:errcheck
	})

	// ── Embedded React SPA ────────────────────────────────────────────────────
	distFS, err := fs.Sub(frontendFS, "frontend/dist")
	if err != nil {
		logger.Error("failed to sub frontend/dist", "error", err)
	} else {
		fileServer := http.FileServer(http.FS(distFS))
		r.Handle("/*", spaHandler(fileServer, distFS))
	}

	return r
}

// spaHandler serves static files and falls back to index.html for SPA routing.
func spaHandler(fileServer http.Handler, distFS fs.FS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Try to serve the file directly
		_, err := distFS.Open(r.URL.Path[1:]) // strip leading /
		if err != nil {
			// Fall back to index.html for client-side routing
			r2 := r.Clone(r.Context())
			r2.URL.Path = "/"
			fileServer.ServeHTTP(w, r2)
			return
		}
		fileServer.ServeHTTP(w, r)
	}
}

// metricsHandler serves Prometheus metrics with optional bearer token auth.
func metricsHandler(cfg *config.Config, reg *metrics.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if cfg.Metrics.BearerToken != "" {
			token := r.Header.Get("Authorization")
			if token != "Bearer "+cfg.Metrics.BearerToken {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		}
		reg.Handler().ServeHTTP(w, r)
	}
}
