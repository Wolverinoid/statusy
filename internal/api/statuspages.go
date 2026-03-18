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
		if err := db.Preload("Monitors").Find(&pages).Error; err != nil {
			writeError(w, http.StatusInternalServerError, "failed to fetch status pages")
			return
		}
		writeJSON(w, http.StatusOK, pages)
	}
}

func createStatusPage(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var page models.StatusPage
		if err := json.NewDecoder(r.Body).Decode(&page); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if err := db.Create(&page).Error; err != nil {
			writeError(w, http.StatusConflict, "failed to create status page")
			return
		}
		writeJSON(w, http.StatusCreated, page)
	}
}

func getStatusPage(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.Atoi(chi.URLParam(r, "id"))
		var page models.StatusPage
		if err := db.Preload("Monitors").First(&page, id).Error; err != nil {
			writeError(w, http.StatusNotFound, "status page not found")
			return
		}
		writeJSON(w, http.StatusOK, page)
	}
}

func updateStatusPage(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.Atoi(chi.URLParam(r, "id"))
		var body models.StatusPage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if err := db.Model(&models.StatusPage{}).Where("id = ?", id).Updates(&body).Error; err != nil {
			writeError(w, http.StatusInternalServerError, "failed to update status page")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"message": "updated"})
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
