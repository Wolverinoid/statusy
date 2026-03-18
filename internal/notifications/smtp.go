package notifications

import (
	"fmt"
	"strings"

	"github.com/statusy/statusy/config"
	"gopkg.in/gomail.v2"
)

// EmailConfig is stored as JSON in Notification.Config.
type EmailConfig struct {
	Recipients []string `json:"recipients"`
	Subject    string   `json:"subject"` // optional override
}

// SendEmail sends an alert via SMTP using the global SMTP config.
func SendEmail(cfg *config.Config, emailCfg *EmailConfig, alert Alert) error {
	if cfg.SMTP.Host == "" {
		return fmt.Errorf("SMTP host not configured")
	}

	subject := emailCfg.Subject
	if subject == "" {
		subject = fmt.Sprintf("[Statusy] %s is %s", alert.MonitorName, alert.Status)
	}

	body := buildEmailBody(alert)

	m := gomail.NewMessage()
	m.SetHeader("From", cfg.SMTP.From)
	m.SetHeader("To", emailCfg.Recipients...)
	m.SetHeader("Subject", subject)
	m.SetBody("text/plain", body)

	d := gomail.NewDialer(cfg.SMTP.Host, cfg.SMTP.Port, cfg.SMTP.Username, cfg.SMTP.Password)
	d.SSL = cfg.SMTP.TLS

	if err := d.DialAndSend(m); err != nil {
		return fmt.Errorf("sending email: %w", err)
	}

	return nil
}

func buildEmailBody(alert Alert) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Monitor: %s\n", alert.MonitorName))
	sb.WriteString(fmt.Sprintf("Status:  %s\n", alert.Status))
	sb.WriteString(fmt.Sprintf("Details: %s\n", alert.Message))
	sb.WriteString("\n-- Statusy Monitoring --\n")
	return sb.String()
}
