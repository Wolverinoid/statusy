package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/statusy/statusy/internal/models"
)

// Registry holds all Prometheus metrics for Statusy.
type Registry struct {
	reg *prometheus.Registry

	monitorUp           *prometheus.GaugeVec
	monitorResponseTime *prometheus.GaugeVec
	monitorSSLDays      *prometheus.GaugeVec
	monitorDomainDays   *prometheus.GaugeVec
	checkTotal          *prometheus.CounterVec
	checkFailuresTotal  *prometheus.CounterVec
}

// NewRegistry creates and registers all Statusy metrics.
func NewRegistry() *Registry {
	reg := prometheus.NewRegistry()

	labels := []string{"monitor_id", "monitor_name", "monitor_type"}

	r := &Registry{
		reg: reg,

		monitorUp: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "statusy_monitor_up",
			Help: "1 if the monitor is UP, 0 if DOWN",
		}, labels),

		monitorResponseTime: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "statusy_monitor_response_time_ms",
			Help: "Last response time in milliseconds",
		}, labels),

		monitorSSLDays: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "statusy_monitor_ssl_days_remaining",
			Help: "Days until SSL certificate expiry",
		}, labels),

		monitorDomainDays: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "statusy_monitor_domain_days_remaining",
			Help: "Days until domain expiry",
		}, labels),

		checkTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "statusy_check_total",
			Help: "Total number of checks performed",
		}, labels),

		checkFailuresTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "statusy_check_failures_total",
			Help: "Total number of failed checks",
		}, labels),
	}

	reg.MustRegister(
		r.monitorUp,
		r.monitorResponseTime,
		r.monitorSSLDays,
		r.monitorDomainDays,
		r.checkTotal,
		r.checkFailuresTotal,
	)

	return r
}

// UpdateMonitor updates all relevant metrics after a check run.
func (r *Registry) UpdateMonitor(m *models.Monitor, status models.MonitorStatus, responseTimeMs int64) {
	id := string(rune(m.ID)) // convert uint to string label
	labels := prometheus.Labels{
		"monitor_id":   id,
		"monitor_name": m.Name,
		"monitor_type": string(m.Type),
	}

	// Up/Down gauge
	upVal := 0.0
	if status == models.StatusUp {
		upVal = 1.0
	}
	r.monitorUp.With(labels).Set(upVal)

	// Response time
	r.monitorResponseTime.With(labels).Set(float64(responseTimeMs))

	// Total checks counter
	r.checkTotal.With(labels).Inc()

	// Failure counter
	if status == models.StatusDown {
		r.checkFailuresTotal.With(labels).Inc()
	}
}

// UpdateSSLDays sets the SSL days remaining gauge.
func (r *Registry) UpdateSSLDays(m *models.Monitor, days int) {
	r.monitorSSLDays.With(prometheus.Labels{
		"monitor_id":   string(rune(m.ID)),
		"monitor_name": m.Name,
		"monitor_type": string(m.Type),
	}).Set(float64(days))
}

// UpdateDomainDays sets the domain expiry days remaining gauge.
func (r *Registry) UpdateDomainDays(m *models.Monitor, days int) {
	r.monitorDomainDays.With(prometheus.Labels{
		"monitor_id":   string(rune(m.ID)),
		"monitor_name": m.Name,
		"monitor_type": string(m.Type),
	}).Set(float64(days))
}

// Handler returns an HTTP handler for the /metrics endpoint.
func (r *Registry) Handler() http.Handler {
	return promhttp.HandlerFor(r.reg, promhttp.HandlerOpts{})
}
