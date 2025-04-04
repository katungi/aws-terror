package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// AWS API metrics
	awsAPICallsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "awsterror_aws_api_calls_total",
			Help: "Total number of AWS API calls",
		},
		[]string{"api", "status"},
	)

	awsAPILatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "awsterror_aws_api_latency_seconds",
			Help:    "Latency of AWS API calls",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"api"},
	)

	// Drift detection metrics
	driftChecksTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "awsterror_drift_checks_total",
			Help: "Total number of drift checks performed",
		},
	)

	driftDetectedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "awsterror_drift_detected_total",
			Help: "Total number of drifts detected",
		},
		[]string{"attribute"},
	)

	driftCheckLatency = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "awsterror_drift_check_latency_seconds",
			Help:    "Latency of drift checks",
			Buckets: prometheus.DefBuckets,
		},
	)
)

// RecordAWSAPICall records metrics for an AWS API call
func RecordAWSAPICall(api string, status string, latency float64) {
	awsAPICallsTotal.WithLabelValues(api, status).Inc()
	awsAPILatency.WithLabelValues(api).Observe(latency)
}

// RecordDriftCheck records metrics for a drift check
func RecordDriftCheck(latency float64) {
	driftChecksTotal.Inc()
	driftCheckLatency.Observe(latency)
}

// RecordDriftDetected records a detected drift for a specific attribute
func RecordDriftDetected(attribute string) {
	driftDetectedTotal.WithLabelValues(attribute).Inc()
}