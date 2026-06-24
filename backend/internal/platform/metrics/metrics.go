package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	HTTPRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "indieforge_http_requests_total",
		Help: "Total HTTP requests by method, route pattern, and status code.",
	}, []string{"method", "path", "status"})

	HTTPDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "indieforge_http_request_duration_seconds",
		Help:    "HTTP request duration in seconds (P50/P95/P99 via histogram_quantile).",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path"})

	// PurchasesTotal counts confirmed payments broken down by kind
	// (purchase | subscription | friend-pack).
	PurchasesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "indieforge_purchases_total",
		Help: "Confirmed purchases by payment kind.",
	}, []string{"kind"})

	// DAU is updated every minute by the metrics ticker in main.
	DAU = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "indieforge_dau",
		Help: "Daily active users: distinct users with a session in the last 24 hours.",
	})

	// MAU is updated every minute by the metrics ticker in main.
	MAU = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "indieforge_mau",
		Help: "Monthly active users: distinct users with a session in the last 30 days.",
	})

	ActiveSubscriptions = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "indieforge_active_subscriptions_total",
		Help: "Total currently active subscriptions.",
	})
)
