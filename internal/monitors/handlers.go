package monitors

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/statusy/statusy/internal/models"
	"github.com/statusy/statusy/internal/notifications"
	"gorm.io/gorm"
)

// Handler provides HTTP handlers for monitor CRUD and notification management.
type Handler struct {
	db         *gorm.DB
	dispatcher *notifications.Dispatcher
	logger     *slog.Logger
}

func NewHandler(db *gorm.DB, dispatcher *notifications.Dispatcher, logger *slog.Logger) *Handler {
	return &Handler{db: db, dispatcher: dispatcher, logger: logger}
}

// GET /api/monitors
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	var monitors []models.Monitor
	if err := h.db.Find(&monitors).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch monitors")
		return
	}
	writeJSON(w, http.StatusOK, monitors)
}

// POST /api/monitors
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var m models.Monitor
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	m.Status = models.StatusPending
	if err := h.db.Create(&m).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create monitor")
		return
	}
	writeJSON(w, http.StatusCreated, m)
}

// GET /api/monitors/{id}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))
	var m models.Monitor
	if err := h.db.Preload("Notifications").First(&m, id).Error; err != nil {
		writeError(w, http.StatusNotFound, "monitor not found")
		return
	}
	writeJSON(w, http.StatusOK, m)
}

// PUT /api/monitors/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))
	var body models.Monitor
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.db.Model(&models.Monitor{}).Where("id = ?", id).Updates(&body).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update monitor")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "updated"})
}

// DELETE /api/monitors/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))
	if err := h.db.Delete(&models.Monitor{}, id).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete monitor")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

// POST /api/monitors/{id}/pause
func (h *Handler) Pause(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))
	if err := h.db.Model(&models.Monitor{}).Where("id = ?", id).
		Updates(map[string]interface{}{"active": false, "status": models.StatusMaintenance}).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "failed to pause monitor")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "paused"})
}

// POST /api/monitors/{id}/resume
func (h *Handler) Resume(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))
	if err := h.db.Model(&models.Monitor{}).Where("id = ?", id).
		Updates(map[string]interface{}{"active": true, "status": models.StatusPending}).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resume monitor")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "resumed"})
}

// GET /api/monitors/{id}/history
func (h *Handler) History(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))
	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 1000 {
			limit = n
		}
	}

	var results []models.CheckResult
	if err := h.db.Where("monitor_id = ?", id).
		Order("checked_at DESC").
		Limit(limit).
		Find(&results).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch history")
		return
	}
	writeJSON(w, http.StatusOK, results)
}

// ─── Notification channel handlers ───────────────────────────────────────────

// GET /api/notifications
func (h *Handler) ListNotifications(w http.ResponseWriter, r *http.Request) {
	var notifs []models.Notification
	if err := h.db.Find(&notifs).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch notifications")
		return
	}
	writeJSON(w, http.StatusOK, notifs)
}

// POST /api/notifications
func (h *Handler) CreateNotification(w http.ResponseWriter, r *http.Request) {
	var n models.Notification
	if err := json.NewDecoder(r.Body).Decode(&n); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.db.Create(&n).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create notification")
		return
	}
	writeJSON(w, http.StatusCreated, n)
}

// PUT /api/notifications/{id}
func (h *Handler) UpdateNotification(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))
	var body models.Notification
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.db.Model(&models.Notification{}).Where("id = ?", id).Updates(&body).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update notification")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "updated"})
}

// DELETE /api/notifications/{id}
func (h *Handler) DeleteNotification(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))
	if err := h.db.Delete(&models.Notification{}, id).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete notification")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

// POST /api/notifications/{id}/test — sends a test alert
func (h *Handler) TestNotification(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))
	var n models.Notification
	if err := h.db.First(&n, id).Error; err != nil {
		writeError(w, http.StatusNotFound, "notification not found")
		return
	}

	h.dispatcher.Send(notifications.Alert{
		MonitorID:   0,
		MonitorName: "Test Monitor",
		Status:      "UP",
		Message:     "This is a test notification from Statusy",
		Channel:     n,
	})

	writeJSON(w, http.StatusOK, map[string]string{"message": "test alert sent"})
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
