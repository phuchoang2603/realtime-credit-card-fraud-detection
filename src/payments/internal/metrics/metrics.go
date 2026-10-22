package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	Registry          *prometheus.Registry
	attemptLatency    prometheus.Histogram
	attemptFailures   *prometheus.CounterVec
	pendingOutcomeAge prometheus.Gauge
	redisHits         prometheus.Counter
	redisMisses       prometheus.Counter
	redisErrors       prometheus.Counter
}

func New() *Metrics {
	m := &Metrics{
		Registry: prometheus.NewRegistry(),
		attemptLatency: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name: "payments_attempt_duration_seconds", Help: "Duration of StartAttempt orchestration.",
			Buckets: []float64{0.025, 0.05, 0.1, 0.25, 0.5, 1, 2, 3, 5, 10},
		}),
		attemptFailures: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "payments_attempt_failures_total", Help: "Attempts that did not succeed, by reason.",
		}, []string{"reason"}),
		pendingOutcomeAge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "payments_pending_outcome_age_seconds", Help: "Age of the oldest attempt awaiting a processor outcome.",
		}),
		redisHits:   prometheus.NewCounter(prometheus.CounterOpts{Name: "payments_redis_idempotency_hits_total", Help: "Idempotency lookups answered by Redis."}),
		redisMisses: prometheus.NewCounter(prometheus.CounterOpts{Name: "payments_redis_idempotency_misses_total", Help: "Idempotency lookups that found no Redis record."}),
		redisErrors: prometheus.NewCounter(prometheus.CounterOpts{Name: "payments_redis_idempotency_errors_total", Help: "Redis operations that failed or timed out."}),
	}
	m.Registry.MustRegister(m.attemptLatency, m.attemptFailures, m.pendingOutcomeAge, m.redisHits, m.redisMisses, m.redisErrors)
	return m
}

func (m *Metrics) ObserveAttempt(duration time.Duration) {
	m.attemptLatency.Observe(duration.Seconds())
}

func (m *Metrics) AttemptFailure(reason string) {
	m.attemptFailures.WithLabelValues(reason).Inc()
}

func (m *Metrics) SetPendingOutcomeAge(age time.Duration) {
	m.pendingOutcomeAge.Set(age.Seconds())
}

func (m *Metrics) CacheHit()   { m.redisHits.Inc() }
func (m *Metrics) CacheMiss()  { m.redisMisses.Inc() }
func (m *Metrics) CacheError() { m.redisErrors.Inc() }
