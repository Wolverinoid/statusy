package monitors

import (
	"context"
	"log/slog"
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
}

type worker struct {
	cancel context.CancelFunc
}

// NewEngine creates a new monitor engine.
func NewEngine(db *gorm.DB, dispatcher *notifications.Dispatcher, reg *metrics.Registry, logger *slog.Logger) *Engine {
	return &Engine{
		db:         db,
		dispatcher: dispatcher,
		metrics:    reg,
		logger:     logger,
		workers:    make(map[uint]*worker),
		stopCh:     make(chan struct{}),
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

func (e *Engine) runCheck(ctx context.Context, m *models.Monitor) {
	start := time.Now()
	timeout := time.Duration(m.TimeoutSeconds) * time.Second
	checkCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	status, message := RunCheck(checkCtx, m)
	responseTime := time.Since(start).Milliseconds()

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

	// Detect status change
	previousStatus := m.Status
	if status != previousStatus {
		e.db.Model(m).Updates(map[string]interface{}{
			"status":               status,
			"consecutive_failures": consecutiveFailures(status, m.ConsecutiveFailures),
		})

		// Load notification channels for this monitor
		var notifs []models.Notification
		e.db.Model(m).Association("Notifications").Find(&notifs)

		for _, n := range notifs {
			if !n.Active {
				continue
			}
			// Respect noise-reduction: only alert after N consecutive failures
			if status == models.StatusDown && m.ConsecutiveFailures < n.NotifyAfterFail {
				continue
			}
			if status == models.StatusDown && !n.NotifyOnDown {
				continue
			}
			if status == models.StatusUp && !n.NotifyOnUp {
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
	} else if status == models.StatusDown {
		// Increment consecutive failures counter
		e.db.Model(m).UpdateColumn("consecutive_failures", gorm.Expr("consecutive_failures + 1"))
	}
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
