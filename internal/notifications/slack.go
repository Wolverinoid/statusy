package notifications

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// SlackConfig is stored as JSON in Notification.Config.
type SlackConfig struct {
	WebhookURL string `json:"webhook_url"`
	Channel    string `json:"channel,omitempty"`  // optional override
	Username   string `json:"username,omitempty"` // optional override
}

// SendSlack sends an alert to a Slack channel via incoming webhook.
func SendSlack(slackCfg *SlackConfig, alert Alert) error {
	if slackCfg.WebhookURL == "" {
		return fmt.Errorf("slack webhook URL not configured")
	}

	color := "danger"
	emoji := ":red_circle:"
	if alert.Status == "UP" {
		color = "good"
		emoji = ":large_green_circle:"
	}

	username := slackCfg.Username
	if username == "" {
		username = "Statusy"
	}

	payload := map[string]interface{}{
		"username": username,
		"attachments": []map[string]interface{}{
			{
				"color":    color,
				"fallback": fmt.Sprintf("%s %s is %s: %s", emoji, alert.MonitorName, alert.Status, alert.Message),
				"fields": []map[string]interface{}{
					{"title": "Monitor", "value": alert.MonitorName, "short": true},
					{"title": "Status", "value": fmt.Sprintf("%s %s", emoji, alert.Status), "short": true},
					{"title": "Details", "value": alert.Message, "short": false},
				},
			},
		},
	}

	if slackCfg.Channel != "" {
		payload["channel"] = slackCfg.Channel
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling slack payload: %w", err)
	}

	resp, err := http.Post(slackCfg.WebhookURL, "application/json", bytes.NewReader(body)) //nolint:gosec
	if err != nil {
		return fmt.Errorf("slack webhook request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack webhook returned status %d", resp.StatusCode)
	}

	return nil
}
