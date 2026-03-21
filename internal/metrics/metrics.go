package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// All Prometheus metric registrations - this is the ONLY file in the project
// that should call prometheus.NewCounter, prometheus.NewHistogram, etc.
// This is enforced by the project's critical rules for metrics.

var (
	// RequestTotal counts total HTTP requests with app, method, and status labels
	RequestTotal *prometheus.CounterVec = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "backd_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"app", "method", "status"},
	)

	// RequestDuration tracks HTTP request duration in seconds with app and method labels
	RequestDuration *prometheus.HistogramVec = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "backd_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"app", "method"},
	)

	// FnInvocations counts function invocations with app, function, and status labels
	FnInvocations *prometheus.CounterVec = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "backd_function_invocations_total",
			Help: "Total number of function invocations",
		},
		[]string{"app", "function", "status"},
	)

	// FnDuration tracks function execution duration in seconds with app and function labels
	FnDuration *prometheus.HistogramVec = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "backd_function_duration_seconds",
			Help:    "Function execution duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"app", "function"},
	)

	// JobsEnqueued counts jobs enqueued with app, function, and trigger labels
	JobsEnqueued *prometheus.CounterVec = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "backd_jobs_enqueued_total",
			Help: "Total number of jobs enqueued",
		},
		[]string{"app", "function", "trigger"},
	)

	// JobsCompleted counts jobs completed successfully with app, function, and trigger labels
	JobsCompleted *prometheus.CounterVec = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "backd_jobs_completed_total",
			Help: "Total number of jobs completed successfully",
		},
		[]string{"app", "function", "trigger"},
	)

	// JobsFailed counts jobs that failed with app, function, and trigger labels
	JobsFailed *prometheus.CounterVec = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "backd_jobs_failed_total",
			Help: "Total number of jobs that failed",
		},
		[]string{"app", "function", "trigger"},
	)

	// JobsDuration tracks job execution duration in seconds with app and function labels
	JobsDuration *prometheus.HistogramVec = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "backd_job_duration_seconds",
			Help:    "Job execution duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"app", "function"},
	)

	// DBPoolConns tracks database pool connection count with app label
	DBPoolConns *prometheus.GaugeVec = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "backd_db_pool_connections",
			Help: "Number of database pool connections",
		},
		[]string{"app"},
	)

	// ActiveSessions tracks active user sessions with app label
	ActiveSessions *prometheus.GaugeVec = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "backd_active_sessions",
			Help: "Number of active user sessions",
		},
		[]string{"app"},
	)
)

// RegisterMetrics registers all Prometheus metrics with the default registry.
// This should be called once during application startup.
func RegisterMetrics() {
	prometheus.MustRegister(
		RequestTotal,
		RequestDuration,
		FnInvocations,
		FnDuration,
		JobsEnqueued,
		JobsCompleted,
		JobsFailed,
		JobsDuration,
		DBPoolConns,
		ActiveSessions,
	)
}

// Metrics interface defines the contract for metrics collection
type Metrics interface {
	// RecordRequest records an HTTP request with method, path, and status
	RecordRequest(app, method, status string)

	// RecordFunctionCall records a function invocation with app name, function name, and success status
	RecordFunctionCall(app, fn string, success bool)

	// RecordStorageOperation records a storage operation with app name, operation type, and success status
	RecordStorageOperation(app, operation string, success bool)
}

// metrics implements the Metrics interface
type metrics struct{}

// NewMetrics creates a new Metrics instance
func NewMetrics() Metrics {
	return &metrics{}
}

// RecordRequest records HTTP request metrics
func (m *metrics) RecordRequest(app, method, status string) {
	RequestTotal.WithLabelValues(app, method, status).Inc()
}

// RecordFunctionCall records function invocation metrics
func (m *metrics) RecordFunctionCall(app, fn string, success bool) {
	status := "success"
	if !success {
		status = "error"
	}
	FnInvocations.WithLabelValues(app, fn, status).Inc()
}

// RecordStorageOperation records storage operation metrics
func (m *metrics) RecordStorageOperation(app, operation string, success bool) {
	// Storage operations are tracked as function calls for now
	// This could be extended with dedicated storage metrics if needed
	status := "success"
	if !success {
		status = "error"
	}
	FnInvocations.WithLabelValues(app, "storage_"+operation, status).Inc()
}
