package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/statusy/statusy/config"
	"github.com/statusy/statusy/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Handler struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewHandler(db *gorm.DB, cfg *config.Config) *Handler {
	return &Handler{db: db, cfg: cfg}
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// POST /api/auth/login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "username and password required")
		return
	}

	var user *models.User

	// Try LDAP first if enabled
	if h.cfg.LDAP.Enabled {
		ldapUser, err := AuthenticateLDAP(req.Username, req.Password, &h.cfg.LDAP)
		if err == nil {
			// Auto-provision or sync user
			user, err = h.upsertLDAPUser(ldapUser)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "failed to provision user")
				return
			}
		}
	}

	// Fall back to local auth
	if user == nil {
		if !h.cfg.Auth.AllowLocalLogin {
			writeError(w, http.StatusUnauthorized, "local login is disabled")
			return
		}
		var dbUser models.User
		if err := h.db.Where("username = ? AND auth_source = ?", req.Username, models.AuthLocal).
			First(&dbUser).Error; err != nil {
			writeError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		if err := bcrypt.CompareHashAndPassword([]byte(dbUser.Password), []byte(req.Password)); err != nil {
			writeError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		if !dbUser.Active {
			writeError(w, http.StatusForbidden, "account is disabled")
			return
		}
		user = &dbUser
	}

	tokens, err := IssueTokenPair(user, h.cfg)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to issue token")
		return
	}

	writeJSON(w, http.StatusOK, tokens)
}

// POST /api/auth/logout
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	tokenStr := extractBearerToken(r)
	if tokenStr == "" {
		writeError(w, http.StatusBadRequest, "missing token")
		return
	}

	claims, err := ParseToken(tokenStr, h.cfg)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid token")
		return
	}

	if err := BlacklistToken(h.db, claims.ID, claims.ExpiresAt.Time); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to invalidate token")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "logged out"})
}

// POST /api/auth/refresh
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	claims, err := ParseToken(body.RefreshToken, h.cfg)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}

	if claims.Issuer != "statusy-refresh" {
		writeError(w, http.StatusUnauthorized, "not a refresh token")
		return
	}

	blacklisted, err := IsBlacklisted(h.db, claims.ID)
	if err != nil || blacklisted {
		writeError(w, http.StatusUnauthorized, "token has been revoked")
		return
	}

	var user models.User
	if err := h.db.First(&user, claims.UserID).Error; err != nil {
		writeError(w, http.StatusUnauthorized, "user not found")
		return
	}

	// Blacklist old refresh token
	_ = BlacklistToken(h.db, claims.ID, claims.ExpiresAt.Time)

	tokens, err := IssueTokenPair(&user, h.cfg)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to issue token")
		return
	}

	writeJSON(w, http.StatusOK, tokens)
}

// GET /api/auth/me
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	claims := ClaimsFromContext(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	var user models.User
	if err := h.db.First(&user, claims.UserID).Error; err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":           user.ID,
		"username":     user.Username,
		"email":        user.Email,
		"display_name": user.DisplayName,
		"role":         user.Role,
		"auth_source":  user.AuthSource,
	})
}

// upsertLDAPUser creates or updates a user record from LDAP attributes.
func (h *Handler) upsertLDAPUser(lu *LDAPUser) (*models.User, error) {
	var user models.User
	result := h.db.Where("username = ? AND auth_source = ?", lu.Username, models.AuthLDAP).First(&user)

	if result.Error == gorm.ErrRecordNotFound {
		user = models.User{
			Username:    lu.Username,
			Email:       lu.Email,
			DisplayName: lu.DisplayName,
			Role:        models.UserRole(h.cfg.LDAP.DefaultRole),
			AuthSource:  models.AuthLDAP,
			Active:      true,
		}
		return &user, h.db.Create(&user).Error
	}

	// Sync attributes
	user.Email = lu.Email
	user.DisplayName = lu.DisplayName
	return &user, h.db.Save(&user).Error
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func extractBearerToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if strings.HasPrefix(header, "Bearer ") {
		return strings.TrimPrefix(header, "Bearer ")
	}
	return ""
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// HashPassword creates a bcrypt hash of a plaintext password.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// contextKey is unexported to avoid collisions.
type contextKey string

const claimsKey contextKey = "claims"

// ClaimsFromContext retrieves JWT claims stored by the auth middleware.
func ClaimsFromContext(ctx interface{ Value(interface{}) interface{} }) *Claims {
	v := ctx.Value(claimsKey)
	if v == nil {
		return nil
	}
	c, _ := v.(*Claims)
	return c
}

// StoreClaimsInContext returns a new context with claims stored under the auth package key.
// Use this in middleware so Me handler can retrieve them.
func StoreClaimsInContext(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}

// unused import guard
var _ = time.Now
