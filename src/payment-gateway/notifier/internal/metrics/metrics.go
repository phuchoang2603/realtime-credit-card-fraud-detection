package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/phuchoang2603/payment-gateway-fraud-detection/notifier/internal/domain"
)

type Metrics struct {
	registry       *prometheus.Registry
	deliveries     *prometheus.CounterVec
	backlog        prometheus.Gauge
	oldestPending  prometheus.Gauge
	consumed       *prometheus.CounterVec
	consumerErrors *prometheus.CounterVec
	lastEventAge   prometheus.Gauge
	lastConsumedAt prometheus.Gauge
}

func New() *Metrics {
	m := &Metrics{
		registry: prometheus.NewRegistry(),
		deliveries: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "notifier_deliveries_total",
			Help: "Webhook delivery attempts by outcome (delivered, retry, failed).",
		}, []string{"outcome"}),
		backlog: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "notifier_deliveries_pending",
			Help: "Webhook deliveries waiting to be delivered.",
		}),
		oldestPending: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "notifier_oldest_pending_age_seconds",
			Help: "Age of the oldest pending webhook delivery.",
		}),
		consumed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "notifier_consumed_events_total",
			Help: "Consumed ledger events by result (recorded, duplicate, skipped, malformed).",
		}, []string{"result"}),
		consumerErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "notifier_consumer_errors_total",
			Help: "Consumer errors by stage (poll, handle, commit).",
		}, []string{"stage"}),
		lastEventAge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "notifier_consumer_last_event_age_seconds",
			Help: "Age of the most recently consumed event when it was processed.",
		}),
		lastConsumedAt: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "notifier_consumer_last_commit_timestamp_seconds",
			Help: "Unix time of the last successful offset commit.",
		}),
	}
	m.registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		m.deliveries, m.backlog, m.oldestPending, m.consumed, m.consumerErrors, m.lastEventAge, m.lastConsumedAt,
	)
	return m
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

func (m *Metrics) Delivery(outcome string) { m.deliveries.WithLabelValues(outcome).Inc() }

func (m *Metrics) Backlog(stats domain.Stats) {
	m.backlog.Set(float64(stats.Pending))
	m.oldestPending.Set(stats.OldestPending.Seconds())
}

func (m *Metrics) Consumed(result string) { m.consumed.WithLabelValues(result).Inc() }

func (m *Metrics) ConsumerError(stage string) { m.consumerErrors.WithLabelValues(stage).Inc() }

func (m *Metrics) EventAge(age time.Duration) { m.lastEventAge.Set(age.Seconds()) }

func (m *Metrics) Committed(at time.Time) { m.lastConsumedAt.Set(float64(at.Unix())) }
