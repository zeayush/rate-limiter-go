// Package metrics exposes Prometheus metrics for the rate limiter.
//
// Three counters are tracked:
//
//	rate_limiter_requests_total{key, algorithm, status}
//	  status = "allowed" | "denied"
//
//	rate_limiter_errors_total{key, algorithm}
//	  counts limiter errors (Redis timeouts, factory failures, etc.)
//
//	rate_limiter_active_keys (Gauge)
//	  number of keys currently tracked by the in-memory store
//
// Register the metrics once at startup:
//
//	metrics.Register(prometheus.DefaultRegisterer)
//
// Expose via the standard /metrics endpoint:
//
//	http.Handle("/metrics", promhttp.Handler())
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics holds all Prometheus instruments for the rate limiter.
type Metrics struct {
	RequestsTotal *prometheus.CounterVec
	ErrorsTotal   *prometheus.CounterVec
	ActiveKeys    prometheus.Gauge
}

// New constructs the metric descriptors. Call Register() before observing.
func New() *Metrics {
	return &Metrics{
		RequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "rate_limiter_requests_total",
				Help: "Total rate limit checks, partitioned by key, algorithm, and status.",
			},
			[]string{"key", "algorithm", "status"},
		),
		ErrorsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "rate_limiter_errors_total",
				Help: "Total rate limiter errors by key and algorithm.",
			},
			[]string{"key", "algorithm"},
		),
		ActiveKeys: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "rate_limiter_active_keys",
			Help: "Number of unique keys tracked by the in-memory store.",
		}),
	}
}

// Register registers all metrics with the given registerer.
// Use prometheus.DefaultRegisterer for the global default registry.
func (m *Metrics) Register(reg prometheus.Registerer) error {
	for _, c := range []prometheus.Collector{
		m.RequestsTotal,
		m.ErrorsTotal,
		m.ActiveKeys,
	} {
		if err := reg.Register(c); err != nil {
			return err
		}
	}
	return nil
}

// RecordAllow records a single allowed or denied request.
//
//	algorithm: "token_bucket" | "sliding_window" | "fixed_window"
//	allowed:   true → status="allowed", false → status="denied"
func (m *Metrics) RecordAllow(key, algorithm string, allowed bool) {
	status := "allowed"
	if !allowed {
		status = "denied"
	}
	m.RequestsTotal.WithLabelValues(key, algorithm, status).Inc()
}

// RecordError records a limiter error.
func (m *Metrics) RecordError(key, algorithm string) {
	m.ErrorsTotal.WithLabelValues(key, algorithm).Inc()
}

// SetActiveKeys updates the active-keys gauge.
func (m *Metrics) SetActiveKeys(n int) {
	m.ActiveKeys.Set(float64(n))
}
