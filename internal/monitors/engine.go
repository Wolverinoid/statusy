package monitors

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/statusy/statusy/internal/metrics"
	"github.com/statusy/statusy/internal/models"
	"github.com/statusy/statusy/internal/notifications"
	"gorm.io/gorm"
)

// Engine manages all monitor goroutines.
type Engine struct {
	db         *gorm.DB
	dispatcher *notifications.Dispatcher
	metrics    *metrics.Registry
	logger     *slog.Logger

	mu      sync.Mutex
	workers map[uint]*worker // monitor ID → worker
	stopCh  chan struct{}

	// certNotifyMu guards certNotifiedOn
	certNotifyMu  sync.Mutex
	certNotifiedOn map[uint]time.Time // monitor ID → last cert-expiry notification date (UTC day)
}

type worker struct {
	cancel context.CancelFunc
}

// NewEngine creates a new monitor engine.
func NewEngine(db *gorm.DB, dispatcher *notifications.Dispatcher, reg *metrics.Registry, logger *slog.Logger) *Engine {
	return &Engine{
		db:             db,
		dispatcher:     dispatcher,
		metrics:        reg,
		logger:         logger,
		workers:        make(map[uint]*worker),
		stopCh:         make(chan struct{}),
		certNotifiedOn: make(map[uint]time.Time),
	}
}

// Start loads all active monitors from DB and starts their goroutines.
// Also runs a periodic reload to pick up new/updated monitors.
func (e *Engine) Start() {
	e.reload()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			e.reload()
		case <-e.stopCh:
			e.stopAll()
			return
		}
	}
}

// Stop signals the engine to shut down.
func (e *Engine) Stop() {
	close(e.stopCh)
}

// StartMonitor starts a single monitor worker (called after create/resume).
func (e *Engine) StartMonitor(m *models.Monitor) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.startWorker(m)
}

// StopMonitor stops a single monitor worker (called after pause/delete).
func (e *Engine) StopMonitor(id uint) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if w, ok := e.workers[id]; ok {
		w.cancel()
		delete(e.workers, id)
	}
}

// reload syncs running workers with the DB state.
func (e *Engine) reload() {
	var monitors []models.Monitor
	if err := e.db.Where("active = ?", true).Find(&monitors).Error; err != nil {
		e.logger.Error("engine: failed to load monitors", "error", err)
		return
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	// Build set of active IDs
	activeIDs := make(map[uint]bool, len(monitors))
	for _, m := range monitors {
		activeIDs[m.ID] = true
	}

	// Stop workers for removed/paused monitors
	for id, w := range e.workers {
		if !activeIDs[id] {
			w.cancel()
			delete(e.workers, id)
		}
	}

	// Start workers for new monitors
	for i := range monitors {
		if _, running := e.workers[monitors[i].ID]; !running {
			e.startWorker(&monitors[i])
		}
	}
}

func (e *Engine) startWorker(m *models.Monitor) {
	ctx, cancel := context.WithCancel(context.Background())
	e.workers[m.ID] = &worker{cancel: cancel}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				e.logger.Error("monitor worker panicked", "monitor_id", m.ID, "panic", r)
			}
		}()

		// Run immediately on start
		e.runCheck(ctx, m)

		interval := time.Duration(m.IntervalSeconds) * time.Second
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// Reload monitor from DB to get latest config
				var latest models.Monitor
				if err := e.db.First(&latest, m.ID).Error; err != nil {
					e.logger.Error("monitor: failed to reload", "monitor_id", m.ID, "error", err)
					return
				}
				e.runCheck(ctx, &latest)
			case <-ctx.Done():
				return
			}
		}
	}()
}

// retryDelay is the pause between immediate re-checks of a failed check.
// (var, not const, so tests can shorten it.)
var retryDelay = 2 * time.Second

// checkWithRetries runs the check and, if it fails, immediately re-runs it up to
// m.Retries more times (retryDelay apart) before reporting DOWN. A single success
// wins, so one-off failures (e.g. "connection reset by peer" from a CDN edge)
// no longer flip the monitor to DOWN. Retries stop early when the monitor
// interval would be exceeded or the context is cancelled.
// Returns the final status, message and the response time of the last attempt.
func (e *Engine) checkWithRetries(ctx context.Context, m *models.Monitor) (models.MonitorStatus, string, int64) {
	timeout := time.Duration(m.TimeoutSeconds) * time.Second
	interval := time.Duration(m.IntervalSeconds) * time.Second
	began := time.Now()

	var (
		status       models.MonitorStatus
		message      string
		responseTime int64
	)
	for attempt := 0; ; attempt++ {
		attemptStart := time.Now()
		checkCtx, cancel := context.WithTimeout(ctx, timeout)
		status, message = RunCheck(checkCtx, m)
		cancel()
		responseTime = time.Since(attemptStart).Milliseconds()

		if status != models.StatusDown || attempt >= m.Retries {
			if status == models.StatusDown && attempt > 0 {
				message = fmt.Sprintf("%s (failed %d attempts)", message, attempt+1)
			}
			return status, message, responseTime
		}
		// Don't let retries overlap with the next scheduled check.
		if interval > 0 && time.Since(began)+retryDelay >= interval {
			if attempt > 0 {
				message = fmt.Sprintf("%s (failed %d attempts)", message, attempt+1)
			}
			return status, message, responseTime
		}

		e.logger.Info("monitor check failed, retrying",
			"monitor_id", m.ID, "attempt", attempt+1, "of", m.Retries+1, "message", message)
		select {
		case <-time.After(retryDelay):
		case <-ctx.Done():
			return status, message, responseTime
		}
	}
}

func (e *Engine) runCheck(ctx context.Context, m *models.Monitor) {
	status, message, responseTime := e.checkWithRetries(ctx, m)

	result := models.CheckResult{
		MonitorID:      m.ID,
		Status:         status,
		ResponseTimeMs: responseTime,
		Message:        message,
		CheckedAt:      time.Now(),
	}
	e.db.Create(&result)

	// Update Prometheus metrics
	e.metrics.UpdateMonitor(m, status, responseTime)

	// Always persist last message (for UI display)
	e.db.Model(m).UpdateColumn("last_message", message)

	// Detect status change
	previousStatus := m.Status
	// Snapshot BEFORE the UpdateColumn calls below: GORM writes the updated value
	// back into the model struct, so m.ConsecutiveFailures would already be the
	// new value (0 on recovery) and the UP alert would be wrongly suppressed.
	previousFailures := m.ConsecutiveFailures
	// newConsecutiveFailures is the updated count after this check
	newConsecutiveFailures := consecutiveFailures(status, previousFailures)

	// Always persist the failure counter (absolute value, not an increment, so it
	// stays correct even if the in-memory monitor is stale).
	if newConsecutiveFailures != previousFailures {
		e.db.Model(m).UpdateColumn("consecutive_failures", newConsecutiveFailures)
	}

	if status != previousStatus {
		e.db.Model(m).UpdateColumn("status", status)
	}

	// Alerts are decided per channel on every check (not only on status change):
	// a channel with NotifyAfterFail=N must fire when the N-th consecutive failure
	// happens, which is usually a check *after* the status already flipped to DOWN.
	if status == models.StatusDown || status != previousStatus {
		var notifs []models.Notification
		e.db.Model(m).Association("Notifications").Find(&notifs)

		for _, n := range notifs {
			if !shouldNotify(n, status, previousStatus, previousFailures, newConsecutiveFailures) {
				continue
			}
			e.dispatcher.Send(notifications.Alert{
				MonitorID:   m.ID,
				MonitorName: m.Name,
				Status:      string(status),
				Message:     message,
				Channel:     n,
			})
		}
	}

	// TLS cert expiry warning: fire daily notifications for HTTPS monitors
	// when cert expires within 7 days (independent of UP/DOWN status change).
	if m.Type == models.MonitorHTTP && strings.HasPrefix(strings.ToLower(m.URL), "https://") {
		e.maybeSendCertExpiryNotification(m, message)
	}
}

// maybeSendCertExpiryNotification sends a daily cert-expiry warning notification
// when the TLS cert expires within 7 days. It fires at most once per UTC day.
func (e *Engine) maybeSendCertExpiryNotification(m *models.Monitor, message string) {
	// The cert expiry info is embedded in the check message by checkHTTP.
	// We only act when the message contains "TLS cert valid until" with ≤7 days,
	// or when the monitor is DOWN due to an expired cert.
	const warnDays = 7

	// Parse days remaining from message — look for "(%d days)" pattern.
	var daysRemaining int
	var certWarning bool
	if strings.Contains(message, "TLS cert valid until") {
		// e.g. "HTTP 200, TLS cert valid until 2025-06-01 (3 days)"
		n, err := parseDaysFromMessage(message)
		if err == nil && n <= warnDays {
			daysRemaining = n
			certWarning = true
		}
	} else if strings.Contains(message, "TLS certificate expired") {
		daysRemaining = 0
		certWarning = true
	}

	if !certWarning {
		return
	}

	// Check if we already sent a notification today (UTC day).
	today := time.Now().UTC().Truncate(24 * time.Hour)
	e.certNotifyMu.Lock()
	lastSent := e.certNotifiedOn[m.ID]
	if !lastSent.Before(today) {
		e.certNotifyMu.Unlock()
		return // already notified today
	}
	e.certNotifiedOn[m.ID] = today
	e.certNotifyMu.Unlock()

	// Build warning message
	var warnMsg string
	if daysRemaining <= 0 {
		warnMsg = message
	} else {
		warnMsg = fmt.Sprintf("TLS certificate expires in %d day(s) — please renew soon! (%s)", daysRemaining, message)
	}

	// Load notification channels and send
	var notifs []models.Notification
	e.db.Model(m).Association("Notifications").Find(&notifs)
	for _, n := range notifs {
		if !n.Active {
			continue
		}
		e.dispatcher.Send(notifications.Alert{
			MonitorID:   m.ID,
			MonitorName: m.Name,
			Status:      "CERT_EXPIRY",
			Message:     warnMsg,
			Channel:     n,
		})
	}
}

// parseDaysFromMessage extracts the days number from a message like
// "HTTP 200, TLS cert valid until 2025-06-01 (3 days)".
func parseDaysFromMessage(msg string) (int, error) {
	// Find "(%d days)" suffix
	start := strings.LastIndex(msg, "(")
	end := strings.LastIndex(msg, " days)")
	if start < 0 || end < 0 || end <= start {
		return 0, fmt.Errorf("pattern not found")
	}
	var days int
	_, err := fmt.Sscanf(msg[start+1:end], "%d", &days)
	return days, err
}

// shouldNotify decides whether channel n gets an alert after a check.
//
//   - DOWN: sent exactly once per outage, on the check where the consecutive
//     failure count reaches the channel's NotifyAfterFail (min 1). Later failed
//     checks keep counting but do not re-alert.
//   - UP: a recovery is sent only if the matching DOWN alert was actually sent
//     to this channel (prevFailures >= threshold). A short blip that never
//     reached the threshold produces no "DOWN" and therefore no "UP".
//     First check after creation (PENDING -> UP) keeps notifying as before.
func shouldNotify(n models.Notification, status, prevStatus models.MonitorStatus, prevFailures, newFailures int) bool {
	if !n.Active {
		return false
	}
	threshold := n.NotifyAfterFail
	if threshold < 1 {
		threshold = 1
	}

	switch status {
	case models.StatusDown:
		return n.NotifyOnDown && newFailures == threshold
	case models.StatusUp:
		if status == prevStatus || !n.NotifyOnUp {
			return false
		}
		if prevStatus == models.StatusDown {
			return prevFailures >= threshold
		}
		return true
	}
	return false
}

func consecutiveFailures(status models.MonitorStatus, current int) int {
	if status == models.StatusDown {
		return current + 1
	}
	return 0
}

func (e *Engine) stopAll() {
	e.mu.Lock()
	defer e.mu.Unlock()
	for id, w := range e.workers {
		w.cancel()
		delete(e.workers, id)
	}
}
