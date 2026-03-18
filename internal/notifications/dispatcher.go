package notifications

import (
	"encoding/json"
	"log/slog"

	"github.com/statusy/statusy/config"
	"github.com/statusy/statusy/internal/models"
)

// Alert is the payload sent to notification channels.
type Alert struct {
	MonitorID   uint
	MonitorName string
	Status      string // "UP" | "DOWN"
	Message     string
	Channel     models.Notification
}

// Dispatcher receives alerts and routes them to the correct sender.
type Dispatcher struct {
	cfg    *config.Config
	logger *slog.Logger
	queue  chan Alert
	stopCh chan struct{}
}

// NewDispatcher creates a dispatcher with a buffered queue.
func NewDispatcher(cfg *config.Config, logger *slog.Logger) *Dispatcher {
	return &Dispatcher{
		cfg:    cfg,
		logger: logger,
		queue:  make(chan Alert, 256),
		stopCh: make(chan struct{}),
	}
}

// Send enqueues an alert (non-blocking; drops if queue is full).
func (d *Dispatcher) Send(alert Alert) {
	select {
	case d.queue <- alert:
	default:
		d.logger.Warn("notification queue full, dropping alert", "monitor", alert.MonitorName)
	}
}

// Run processes alerts from the queue until Stop is called.
func (d *Dispatcher) Run() {
	for {
		select {
		case alert := <-d.queue:
			d.dispatch(alert)
		case <-d.stopCh:
			// Drain remaining alerts
			for {
				select {
				case alert := <-d.queue:
					d.dispatch(alert)
				default:
					return
				}
			}
		}
	}
}

// Stop signals the dispatcher to shut down after draining the queue.
func (d *Dispatcher) Stop() {
	close(d.stopCh)
}

func (d *Dispatcher) dispatch(alert Alert) {
	switch alert.Channel.Type {
	case models.NotifEmail:
		var cfg EmailConfig
		if err := json.Unmarshal([]byte(alert.Channel.Config), &cfg); err != nil {
			d.logger.Error("invalid email config", "error", err)
			return
		}
		if err := SendEmail(d.cfg, &cfg, alert); err != nil {
			d.logger.Error("email send failed", "error", err, "monitor", alert.MonitorName)
		}

	case models.NotifTelegram:
		var cfg TelegramConfig
		if err := json.Unmarshal([]byte(alert.Channel.Config), &cfg); err != nil {
			d.logger.Error("invalid telegram config", "error", err)
			return
		}
		if err := SendTelegram(d.cfg, &cfg, alert); err != nil {
			d.logger.Error("telegram send failed", "error", err, "monitor", alert.MonitorName)
		}

	case models.NotifSlack:
		var cfg SlackConfig
		if err := json.Unmarshal([]byte(alert.Channel.Config), &cfg); err != nil {
			d.logger.Error("invalid slack config", "error", err)
			return
		}
		if err := SendSlack(&cfg, alert); err != nil {
			d.logger.Error("slack send failed", "error", err, "monitor", alert.MonitorName)
		}

	default:
		d.logger.Warn("unknown notification type", "type", alert.Channel.Type)
	}
}
