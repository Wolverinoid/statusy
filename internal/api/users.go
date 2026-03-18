package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/statusy/statusy/config"
	"github.com/statusy/statusy/internal/auth"
	"github.com/statusy/statusy/internal/models"
	"gorm.io/gorm"
)

func listUsers(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var users []models.User
		if err := db.Find(&users).Error; err != nil {
			writeError(w, http.StatusInternalServerError, "failed to fetch users")
			return
		}
		// Strip password hashes from response
		type userResp struct {
			ID          uint             `json:"id"`
			Username    string           `json:"username"`
			Email       string           `json:"email"`
			DisplayName string           `json:"display_name"`
			Role        models.UserRole  `json:"role"`
			AuthSource  models.AuthSource `json:"auth_source"`
			Active      bool             `json:"active"`
		}
		resp := make([]userResp, len(users))
		for i, u := range users {
			resp[i] = userResp{
				ID: u.ID, Username: u.Username, Email: u.Email,
				DisplayName: u.DisplayName, Role: u.Role,
				AuthSource: u.AuthSource, Active: u.Active,
			}
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

func createUser(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Username string          `json:"username"`
			Email    string          `json:"email"`
			Password string          `json:"password"`
			Role     models.UserRole `json:"role"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		hash, err := auth.HashPassword(body.Password)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to hash password")
			return
		}
		user := models.User{
			Username:   body.Username,
			Email:      body.Email,
			Password:   hash,
			Role:       body.Role,
			AuthSource: models.AuthLocal,
			Active:     true,
		}
		if err := db.Create(&user).Error; err != nil {
			writeError(w, http.StatusConflict, "user already exists or invalid data")
			return
		}
		writeJSON(w, http.StatusCreated, map[string]uint{"id": user.ID})
	}
}

func updateUser(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.Atoi(chi.URLParam(r, "id"))
		var body struct {
			Email    string          `json:"email"`
			Role     models.UserRole `json:"role"`
			Active   *bool           `json:"active"`
			Password string          `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		updates := map[string]interface{}{}
		if body.Email != "" {
			updates["email"] = body.Email
		}
		if body.Role != "" {
			updates["role"] = body.Role
		}
		if body.Active != nil {
			updates["active"] = *body.Active
		}
		if body.Password != "" {
			hash, err := auth.HashPassword(body.Password)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "failed to hash password")
				return
			}
			updates["password"] = hash
		}
		if err := db.Model(&models.User{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			writeError(w, http.StatusInternalServerError, "failed to update user")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"message": "updated"})
	}
}

func deleteUser(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.Atoi(chi.URLParam(r, "id"))
		if err := db.Delete(&models.User{}, id).Error; err != nil {
			writeError(w, http.StatusInternalServerError, "failed to delete user")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
	}
}

func ldapTest(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := auth.TestConnection(&cfg.LDAP); err != nil {
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"message": "LDAP connection successful"})
	}
}

// writeJSON is defined in middleware.go — reuse it here via same package.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}
