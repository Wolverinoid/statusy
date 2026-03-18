package api

import (
	"encoding/json"
	"net/http"

	"github.com/statusy/statusy/internal/models"
	"gorm.io/gorm"
)

// PrometheusConfig is the JSON stored in Integration.Config for the "prometheus" integration.
type PrometheusConfig struct {
	URL           string `json:"url"`
	BasicAuthUser string `json:"basic_auth_user"`
	BasicAuthPass string `json:"basic_auth_pass"`
}

// GET /api/admin/integrations/prometheus
func getPrometheusIntegration(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var integration models.Integration
		err := db.Where("name = ?", "prometheus").First(&integration).Error
		if err == gorm.ErrRecordNotFound {
			// Return empty/default config
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"enabled":         false,
				"url":             "",
				"basic_auth_user": "",
				"basic_auth_pass": "",
			})
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to fetch integration")
			return
		}

		var cfg PrometheusConfig
		_ = json.Unmarshal([]byte(integration.Config), &cfg)

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"enabled":         integration.Enabled,
			"url":             cfg.URL,
			"basic_auth_user": cfg.BasicAuthUser,
			"basic_auth_pass": cfg.BasicAuthPass,
		})
	}
}

// PUT /api/admin/integrations/prometheus
func putPrometheusIntegration(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Enabled       bool   `json:"enabled"`
			URL           string `json:"url"`
			BasicAuthUser string `json:"basic_auth_user"`
			BasicAuthPass string `json:"basic_auth_pass"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		cfgJSON, err := json.Marshal(PrometheusConfig{
			URL:           body.URL,
			BasicAuthUser: body.BasicAuthUser,
			BasicAuthPass: body.BasicAuthPass,
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to encode config")
			return
		}

		var integration models.Integration
		result := db.Where("name = ?", "prometheus").First(&integration)
		if result.Error == gorm.ErrRecordNotFound {
			integration = models.Integration{
				Name:    "prometheus",
				Enabled: body.Enabled,
				Config:  string(cfgJSON),
			}
			if err := db.Create(&integration).Error; err != nil {
				writeError(w, http.StatusInternalServerError, "failed to save integration")
				return
			}
		} else {
			integration.Enabled = body.Enabled
			integration.Config = string(cfgJSON)
			if err := db.Save(&integration).Error; err != nil {
				writeError(w, http.StatusInternalServerError, "failed to save integration")
				return
			}
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"enabled":         integration.Enabled,
			"url":             body.URL,
			"basic_auth_user": body.BasicAuthUser,
			"basic_auth_pass": body.BasicAuthPass,
		})
	}
}
