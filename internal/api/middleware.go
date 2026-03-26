package api

import (
	"net/http"
	"strings"

	"github.com/statusy/statusy/config"
	"github.com/statusy/statusy/internal/auth"
	"github.com/statusy/statusy/internal/models"
	"gorm.io/gorm"
)

// Authenticate validates the Bearer JWT and stores claims in context.
func Authenticate(cfg *config.Config, db *gorm.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr := extractBearer(r)
			if tokenStr == "" {
				writeError(w, http.StatusUnauthorized, "missing authorization token")
				return
			}

			claims, err := auth.ParseToken(tokenStr, cfg)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			// Check blacklist
			blacklisted, err := auth.IsBlacklisted(db, claims.ID)
			if err != nil || blacklisted {
				writeError(w, http.StatusUnauthorized, "token has been revoked")
				return
			}

			ctx := auth.StoreClaimsInContext(r.Context(), claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// AuthenticateOptional tries to parse the Bearer JWT but never blocks the request.
// Claims are stored in context if valid; otherwise the request proceeds unauthenticated.
func AuthenticateOptional(cfg *config.Config, db *gorm.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr := extractBearer(r)
			if tokenStr != "" {
				if claims, err := auth.ParseToken(tokenStr, cfg); err == nil {
					blacklisted, err := auth.IsBlacklisted(db, claims.ID)
					if err == nil && !blacklisted {
						ctx := auth.StoreClaimsInContext(r.Context(), claims)
						r = r.WithContext(ctx)
					}
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireAdmin rejects requests from non-admin users.
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := claimsFromCtx(r)
		if claims == nil || claims.Role != models.RoleAdmin {
			writeError(w, http.StatusForbidden, "admin access required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// claimsFromCtx is a package-local helper.
func claimsFromCtx(r *http.Request) *auth.Claims {
	return auth.ClaimsFromContext(r.Context())
}

func extractBearer(r *http.Request) string {
	// X-Auth-Token takes priority — avoids Basic Auth header collision
	// when the app is behind a reverse proxy with HTTP Basic Auth.
	if t := r.Header.Get("X-Auth-Token"); t != "" {
		return t
	}
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return ""
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write([]byte(`{"error":"` + msg + `"}`)) //nolint:errcheck
}
