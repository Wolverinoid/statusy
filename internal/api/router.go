package api

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/statusy/statusy/config"
	"github.com/statusy/statusy/internal/auth"
	"github.com/statusy/statusy/internal/metrics"
	"github.com/statusy/statusy/internal/models"
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
			r.Put("/{id}/notifications", monitorHandler.SetMonitorNotifications)
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

		// Integrations (admin only)
		r.Route("/api/admin/integrations", func(r chi.Router) {
			r.Use(RequireAdmin)
			r.Get("/prometheus", getPrometheusIntegration(db))
			r.Put("/prometheus", putPrometheusIntegration(db))
		})

		// Status pages (admin only for management)
		r.Route("/api/status-pages", func(r chi.Router) {
			r.Use(RequireAdmin)
			r.Get("/", listStatusPages(db))
			r.Post("/", createStatusPage(db))
			r.Get("/{id}", getStatusPage(db))
			r.Put("/{id}", updateStatusPage(db))
			r.Delete("/{id}", deleteStatusPage(db))
			r.Put("/{id}/monitors", setStatusPageMonitors(db))
			r.Put("/{id}/users", setStatusPageUsers(db))
		})

	})

	// ── Prometheus metrics ────────────────────────────────────────────────────
	// Public status page — optional auth (private pages need token)
	r.With(AuthenticateOptional(cfg, db)).Get("/api/status/{slug}", publicStatusPage(db))

	if cfg.Metrics.Enabled {
		r.Get("/metrics", metricsHandler(cfg, metricsReg, db))
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

// metricsHandler serves Prometheus metrics with optional bearer token auth (config)
// and optional HTTP basic auth (stored in DB via integrations).
func metricsHandler(cfg *config.Config, reg *metrics.Registry, db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Bearer token auth from config file (legacy).
		// Only enforced when the request uses Bearer scheme; Basic auth requests
		// are allowed to fall through to the basic-auth check below.
		if cfg.Metrics.BearerToken != "" {
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				if authHeader != "Bearer "+cfg.Metrics.BearerToken {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}
			} else if !strings.HasPrefix(authHeader, "Basic ") {
				// No recognised auth scheme provided — reject
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			// "Basic ..." header: fall through to basic-auth check below
		}

		// Basic auth from DB integration settings.
		// Also gates the endpoint: if the integration record exists but is disabled, return 404.
		if db != nil {
			var integration models.Integration
			err := db.Where("name = ?", "prometheus").First(&integration).Error
			if err == nil {
				// Record exists — check if disabled
				if !integration.Enabled {
					http.NotFound(w, r)
					return
				}
				// Enabled — enforce basic auth if credentials are set
				var promCfg PrometheusConfig
				if jsonErr := json.Unmarshal([]byte(integration.Config), &promCfg); jsonErr == nil {
					if promCfg.BasicAuthUser != "" || promCfg.BasicAuthPass != "" {
						user, pass, ok := r.BasicAuth()
						if !ok || user != promCfg.BasicAuthUser || pass != promCfg.BasicAuthPass {
							w.Header().Set("WWW-Authenticate", `Basic realm="Statusy Metrics"`)
							http.Error(w, "unauthorized", http.StatusUnauthorized)
							return
						}
					}
				}
			}
			// If no record exists at all — allow access (not configured yet)
		}

		reg.Handler().ServeHTTP(w, r)
	}
}

// loadPrometheusIntegrationConfig reads the Prometheus integration config from DB.
func loadPrometheusIntegrationConfig(db *gorm.DB) (*PrometheusConfig, error) {
	var integration models.Integration
	if err := db.Where("name = ?", "prometheus").First(&integration).Error; err != nil {
		return nil, err
	}
	if !integration.Enabled {
		return nil, nil
	}
	var cfg PrometheusConfig
	if err := json.Unmarshal([]byte(integration.Config), &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
