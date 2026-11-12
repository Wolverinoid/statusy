package monitors

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/statusy/statusy/internal/models"
)

// flakyServer drops the first `failFirst` connections without answering
// (client sees EOF / reset), then answers 200.
func flakyServer(t *testing.T, failFirst int32) (*httptest.Server, *int32) {
	t.Helper()
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) <= failFirst {
			conn, _, err := w.(http.Hijacker).Hijack()
			if err == nil {
				conn.Close()
			}
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	return srv, &calls
}

func testEngine() *Engine {
	return &Engine{logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
}

func testMonitor(url string, retries int) *models.Monitor {
	return &models.Monitor{
		Type:            models.MonitorHTTP,
		URL:             url,
		Retries:         retries,
		TimeoutSeconds:  5,
		IntervalSeconds: 60,
	}
}

func withFastRetries(t *testing.T) {
	old := retryDelay
	retryDelay = 10 * time.Millisecond
	t.Cleanup(func() { retryDelay = old })
}

// TC1 (R1): a one-off failure is absorbed by retries -> UP, 2 requests made.
func TestTC1_RetryAbsorbsSingleFailure(t *testing.T) {
	withFastRetries(t)
	srv, calls := flakyServer(t, 1)

	status, _, _ := testEngine().checkWithRetries(context.Background(), testMonitor(srv.URL, 2))

	if status != models.StatusUp {
		t.Fatalf("want UP, got %s", status)
	}
	if got := atomic.LoadInt32(calls); got != 2 {
		t.Fatalf("want 2 requests, got %d", got)
	}
}

// TC2 (R2): failure persists beyond Retries -> DOWN after exactly Retries+1 attempts.
func TestTC2_DownAfterRetriesExhausted(t *testing.T) {
	withFastRetries(t)
	srv, calls := flakyServer(t, 100)

	status, msg, _ := testEngine().checkWithRetries(context.Background(), testMonitor(srv.URL, 2))

	if status != models.StatusDown {
		t.Fatalf("want DOWN, got %s (%s)", status, msg)
	}
	if got := atomic.LoadInt32(calls); got != 3 {
		t.Fatalf("want 3 attempts, got %d", got)
	}
}

// TC3 (R3): Retries=0 -> single attempt, DOWN right away.
func TestTC3_NoRetriesWhenZero(t *testing.T) {
	withFastRetries(t)
	srv, calls := flakyServer(t, 1)

	status, _, _ := testEngine().checkWithRetries(context.Background(), testMonitor(srv.URL, 0))

	if status != models.StatusDown {
		t.Fatalf("want DOWN, got %s", status)
	}
	if got := atomic.LoadInt32(calls); got != 1 {
		t.Fatalf("want 1 attempt, got %d", got)
	}
}
