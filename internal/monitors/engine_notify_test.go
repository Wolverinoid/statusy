package monitors

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/statusy/statusy/internal/models"
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
