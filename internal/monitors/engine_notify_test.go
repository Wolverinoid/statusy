package monitors

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/statusy/statusy/config"
	"github.com/statusy/statusy/internal/metrics"
	"github.com/statusy/statusy/internal/models"
	"github.com/statusy/statusy/internal/notifications"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func channel(after int) models.Notification {
	n := models.Notification{Active: true, NotifyOnDown: true, NotifyOnUp: true, NotifyAfterFail: after}
	return n
}

// TC4 (R4): DOWN alert fires once, when the failure count reaches NotifyAfterFail;
// UP alert only follows a DOWN alert that was really sent.
func TestTC4_NotifyAfterFail(t *testing.T) {
	up, down, pending := models.StatusUp, models.StatusDown, models.StatusPending

	tests := []struct {
		name         string
		n            models.Notification
		status, prev models.MonitorStatus
		prevF, newF  int
		want         bool
	}{
		{"after=1 first failure alerts", channel(1), down, up, 0, 1, true},
		{"after=3 first failure silent", channel(3), down, up, 0, 1, false},
		{"after=3 second failure silent", channel(3), down, down, 1, 2, false},
		{"after=3 third failure alerts (status already DOWN)", channel(3), down, down, 2, 3, true},
		{"after=3 fourth failure does not re-alert", channel(3), down, down, 3, 4, false},
		{"after=0 treated as 1", channel(0), down, up, 0, 1, true},
		{"recovery after alert sent", channel(3), up, down, 5, 0, true},
		{"recovery after blip below threshold is silent", channel(3), up, down, 2, 0, false},
		{"UP while already UP is silent", channel(1), up, up, 0, 0, false},
		{"first check PENDING->UP still notifies", channel(3), up, pending, 0, 0, true},
		{"inactive channel is silent", func() models.Notification { n := channel(1); n.Active = false; return n }(), down, up, 0, 1, false},
		{"NotifyOnDown=false", func() models.Notification { n := channel(1); n.NotifyOnDown = false; return n }(), down, up, 0, 1, false},
		{"NotifyOnUp=false", func() models.Notification { n := channel(1); n.NotifyOnUp = false; return n }(), up, down, 1, 0, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldNotify(tc.n, tc.status, tc.prev, tc.prevF, tc.newF); got != tc.want {
				t.Fatalf("shouldNotify = %v, want %v", got, tc.want)
			}
		})
	}
}

// TC5 (R5): HTTP error details include CF-RAY when the response has it.
func TestTC5_CFRayInErrorDetails(t *testing.T) {
	withRay := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("CF-RAY", "8a1b2c3d4e5f6a7b-FRA")
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer withRay.Close()
	noRay := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer noRay.Close()

	status, msg := checkHTTP(context.Background(), testMonitor(withRay.URL, 0))
	if status != models.StatusDown || !strings.Contains(msg, "[CF-RAY: 8a1b2c3d4e5f6a7b-FRA]") {
		t.Fatalf("want DOWN with CF-RAY, got %s %q", status, msg)
	}

	status, msg = checkHTTP(context.Background(), testMonitor(noRay.URL, 0))
	if status != models.StatusDown || strings.Contains(msg, "CF-RAY") {
		t.Fatalf("want DOWN without CF-RAY, got %s %q", status, msg)
	}
}

// TC6 (R4): end-to-end through runCheck — after a DOWN alert, recovery must send UP.
// Regression: GORM's UpdateColumn wrote the new failure counter (0) back into the
// in-memory monitor before shouldNotify ran, so prevFailures was 0 and UP was dropped.
func TestTC6_RecoveryAfterDownSendsUp(t *testing.T) {
	var healthy atomic.Bool // target: false -> 500, true -> 200
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if healthy.Load() {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer target.Close()

	var mu sync.Mutex
	var alerts []string // Slack webhook bodies
	hook := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		mu.Lock()
		alerts = append(alerts, string(b))
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer hook.Close()

	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1) // each :memory: connection is its own database
	if err := db.AutoMigrate(&models.Monitor{}, &models.Notification{}, &models.CheckResult{}); err != nil {
		t.Fatal(err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	disp := notifications.NewDispatcher(&config.Config{}, logger)
	go disp.Run()
	defer disp.Stop()
	e := NewEngine(db, disp, metrics.NewRegistry(), logger)

	mon := testMonitor(target.URL, 0)
	mon.Status = models.StatusUp
	if err := db.Create(mon).Error; err != nil {
		t.Fatal(err)
	}
	ch := channel(1)
	ch.Name, ch.Type = "slack", models.NotifSlack
	ch.Config = fmt.Sprintf(`{"webhook_url":%q}`, hook.URL)
	if err := db.Model(mon).Association("Notifications").Append(&ch); err != nil {
		t.Fatal(err)
	}

	count := func(sub string) int {
		mu.Lock()
		defer mu.Unlock()
		n := 0
		for _, a := range alerts {
			if strings.Contains(a, sub) {
				n++
			}
		}
		return n
	}
	waitFor := func(sub string) {
		for i := 0; i < 100 && count(sub) == 0; i++ {
			time.Sleep(20 * time.Millisecond)
		}
	}
	reload := func() *models.Monitor { // the worker reloads the monitor on every tick
		var m models.Monitor
		if err := db.First(&m, mon.ID).Error; err != nil {
			t.Fatal(err)
		}
		return &m
	}

	e.runCheck(context.Background(), reload())
	waitFor("is DOWN")
	if count("is DOWN") != 1 {
		t.Fatalf("want 1 DOWN alert, got %d", count("is DOWN"))
	}

	healthy.Store(true)
	e.runCheck(context.Background(), reload())
	waitFor("is UP")
	if count("is UP") != 1 {
		t.Fatalf("want 1 UP alert after recovery, got %d (alerts: %v)", count("is UP"), alerts)
	}
}
