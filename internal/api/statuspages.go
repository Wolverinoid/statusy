package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/statusy/statusy/internal/models"
	"gorm.io/gorm"
)

func listStatusPages(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var pages []models.StatusPage
		if err := db.Preload("Monitors").Preload("Users").Find(&pages).Error; err != nil {
			writeError(w, http.StatusInternalServerError, "failed to fetch status pages")
			return
		}
		writeJSON(w, http.StatusOK, pages)
	}
}

func createStatusPage(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Name        string `json:"name"`
			Slug        string `json:"slug"`
			Description string `json:"description"`
			Public      bool   `json:"public"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if body.Name == "" || body.Slug == "" {
			writeError(w, http.StatusBadRequest, "name and slug are required")
			return
		}
		page := models.StatusPage{
			Name:        body.Name,
			Slug:        body.Slug,
			Description: body.Description,
			Public:      body.Public,
		}
		if err := db.Create(&page).Error; err != nil {
			writeError(w, http.StatusConflict, "failed to create status page (slug may already exist)")
			return
		}
		writeJSON(w, http.StatusCreated, page)
	}
}

func getStatusPage(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.Atoi(chi.URLParam(r, "id"))
		var page models.StatusPage
		if err := db.Preload("Monitors").Preload("Users").First(&page, id).Error; err != nil {
			writeError(w, http.StatusNotFound, "status page not found")
			return
		}
		writeJSON(w, http.StatusOK, page)
	}
}

func updateStatusPage(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.Atoi(chi.URLParam(r, "id"))
		var body struct {
			Name        string `json:"name"`
			Slug        string `json:"slug"`
			Description string `json:"description"`
			Public      bool   `json:"public"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if err := db.Model(&models.StatusPage{}).Where("id = ?", id).Updates(map[string]interface{}{
			"name":        body.Name,
			"slug":        body.Slug,
			"description": body.Description,
			"public":      body.Public,
		}).Error; err != nil {
			writeError(w, http.StatusInternalServerError, "failed to update status page")
			return
		}
		var page models.StatusPage
		db.Preload("Monitors").Preload("Users").First(&page, id)
		writeJSON(w, http.StatusOK, page)
	}
}

func deleteStatusPage(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.Atoi(chi.URLParam(r, "id"))
		if err := db.Delete(&models.StatusPage{}, id).Error; err != nil {
			writeError(w, http.StatusInternalServerError, "failed to delete status page")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
	}
}

// PUT /api/status-pages/{id}/monitors — replace monitor list
func setStatusPageMonitors(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.Atoi(chi.URLParam(r, "id"))
		var body struct {
			MonitorIDs []uint `json:"monitor_ids"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		var page models.StatusPage
		if err := db.First(&page, id).Error; err != nil {
			writeError(w, http.StatusNotFound, "status page not found")
			return
		}
		var monitors []models.Monitor
		if len(body.MonitorIDs) > 0 {
			db.Find(&monitors, body.MonitorIDs)
		}
		if err := db.Model(&page).Association("Monitors").Replace(monitors); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to update monitors")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"message": "updated"})
	}
}

// PUT /api/status-pages/{id}/users — replace user access list
func setStatusPageUsers(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.Atoi(chi.URLParam(r, "id"))
		var body struct {
			UserIDs []uint `json:"user_ids"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		var page models.StatusPage
		if err := db.First(&page, id).Error; err != nil {
			writeError(w, http.StatusNotFound, "status page not found")
			return
		}
		var users []models.User
		if len(body.UserIDs) > 0 {
			db.Find(&users, body.UserIDs)
		}
		if err := db.Model(&page).Association("Users").Replace(users); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to update users")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"message": "updated"})
	}
}

// GET /status/{slug} — public status page view (no auth required)
func publicStatusPage(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "slug")
		var page models.StatusPage
		if err := db.Preload("Monitors").Where("slug = ?", slug).First(&page).Error; err != nil {
			writeError(w, http.StatusNotFound, "status page not found")
			return
		}

		// Check access: public pages are open; private pages require auth
		if !page.Public {
			// Try to authenticate via Bearer token
			claims := claimsFromCtx(r)
			if claims == nil {
				writeError(w, http.StatusUnauthorized, "this status page is private")
				return
			}
			// Admins always have access
			if claims.Role != models.RoleAdmin {
				// Check if user is in the page's user list
				var count int64
				db.Table("status_page_users").
					Where("status_page_id = ? AND user_id = ?", page.ID, claims.UserID).
					Count(&count)
				if count == 0 {
					writeError(w, http.StatusForbidden, "access denied")
					return
				}
			}
		}

		writeJSON(w, http.StatusOK, page)
	}
}
