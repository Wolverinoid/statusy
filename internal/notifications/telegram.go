package notifications

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/statusy/statusy/config"
)

// TelegramConfig is stored as JSON in Notification.Config.
type TelegramConfig struct {
	ChatID string `json:"chat_id"`
	// BotToken can override the global one per-channel
	BotToken string `json:"bot_token,omitempty"`
}

// SendTelegram sends an alert message via the Telegram Bot API.
func SendTelegram(cfg *config.Config, tgCfg *TelegramConfig, alert Alert) error {
	token := tgCfg.BotToken
	if token == "" {
		token = cfg.Telegram.BotToken
	}
	if token == "" {
		return fmt.Errorf("telegram bot token not configured")
	}

	emoji := "🔴"
	if alert.Status == "UP" {
		emoji = "🟢"
	}

	text := fmt.Sprintf(
		"%s *%s* is *%s*\n\n`%s`",
		emoji, escapeMarkdown(alert.MonitorName), alert.Status, alert.Message,
	)

	payload := map[string]interface{}{
		"chat_id":    tgCfg.ChatID,
		"text":       text,
		"parse_mode": "Markdown",
	}

	body, _ := json.Marshal(payload)
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)

	resp, err := http.Post(url, "application/json", bytes.NewReader(body)) //nolint:gosec
	if err != nil {
		return fmt.Errorf("telegram API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API returned status %d", resp.StatusCode)
	}

	return nil
}

func escapeMarkdown(s string) string {
	// Escape special Markdown characters
	replacer := []string{"_", "\\_", "*", "\\*", "`", "\\`", "[", "\\["}
	r := s
	for i := 0; i < len(replacer)-1; i += 2 {
		r = replaceAll(r, replacer[i], replacer[i+1])
	}
	return r
}

func replaceAll(s, old, new string) string {
	result := ""
	for i := 0; i < len(s); i++ {
		if i+len(old) <= len(s) && s[i:i+len(old)] == old {
			result += new
			i += len(old) - 1
		} else {
			result += string(s[i])
		}
	}
	return result
}
