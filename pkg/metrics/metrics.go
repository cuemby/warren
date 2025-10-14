package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// Cluster metrics
	NodesTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "warren_nodes_total",
			Help: "Total number of nodes by role and status",
		},
		[]string{"role", "status"},
	)

	ServicesTotal = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "warren_services_total",
			Help: "Total number of services",
		},
	)

	ContainersTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "warren_containers_total",
			Help: "Total number of containers by state",
		},
		[]string{"state"},
	)

	SecretsTotal = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "warren_secrets_total",
			Help: "Total number of secrets",
		},
	)

	VolumesTotal = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "warren_volumes_total",
			Help: "Total number of volumes",
		},
	)

	// Raft metrics
	RaftLeader = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "warren_raft_is_leader",
			Help: "Whether this node is the Raft leader (1 = leader, 0 = follower)",
		},
	)

	RaftPeers = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "warren_raft_peers_total",
			Help: "Total number of Raft peers in the cluster",
		},
	)

	RaftLogIndex = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "warren_raft_log_index",
			Help: "Current Raft log index",
		},
	)

	RaftAppliedIndex = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "warren_raft_applied_index",
			Help: "Last applied Raft log index",
		},
	)

	// API metrics
	APIRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "warren_api_requests_total",
			Help: "Total number of API requests by method and status",
		},
		[]string{"method", "status"},
	)

	APIRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "warren_api_request_duration_seconds",
			Help:    "API request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method"},
	)

	// Scheduler metrics
	SchedulingLatency = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "warren_scheduling_latency_seconds",
			Help:    "Time taken to schedule containers in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)

	ContainersScheduled = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "warren_containers_scheduled_total",
			Help: "Total number of containers scheduled",
		},
	)

	ContainersFailed = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "warren_containers_failed_total",
			Help: "Total number of failed containers",
		},
	)

	// Service operation metrics
	ServiceCreateDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "warren_service_create_duration_seconds",
			Help:    "Time taken to create a service in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)

	ServiceUpdateDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "warren_service_update_duration_seconds",
			Help:    "Time taken to update a service in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)

	ServiceDeleteDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "warren_service_delete_duration_seconds",
			Help:    "Time taken to delete a service in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)

	// Container operation metrics
	ContainerCreateDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "warren_container_create_duration_seconds",
			Help:    "Time taken to create a container in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)

	ContainerStartDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "warren_container_start_duration_seconds",
			Help:    "Time taken to start a container in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)

	ContainerStopDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "warren_container_stop_duration_seconds",
			Help:    "Time taken to stop a container in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)

	// Raft operation metrics
	RaftApplyDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "warren_raft_apply_duration_seconds",
			Help:    "Time taken to apply a Raft log entry in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)

	RaftCommitDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "warren_raft_commit_duration_seconds",
			Help:    "Time taken to commit a Raft log entry in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)

	// Reconciler metrics
	ReconciliationDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "warren_reconciliation_duration_seconds",
			Help:    "Time taken for a reconciliation cycle in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)

	ReconciliationCyclesTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "warren_reconciliation_cycles_total",
			Help: "Total number of reconciliation cycles completed",
		},
	)

	// Ingress metrics
	IngressCreateDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "warren_ingress_create_duration_seconds",
			Help:    "Time taken to create an ingress rule in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)

	IngressUpdateDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "warren_ingress_update_duration_seconds",
			Help:    "Time taken to update an ingress rule in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)

	IngressRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "warren_ingress_requests_total",
			Help: "Total number of ingress requests by host and backend",
		},
		[]string{"host", "backend"},
	)

	IngressRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "warren_ingress_request_duration_seconds",
			Help:    "Ingress request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"host", "backend"},
	)

	// Deployment metrics
	DeploymentsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "warren_deployments_total",
			Help: "Total number of deployments by strategy and status",
		},
		[]string{"strategy", "status"},
	)

	DeploymentDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "warren_deployment_duration_seconds",
			Help:    "Deployment duration in seconds by strategy",
			Buckets: []float64{1, 5, 10, 30, 60, 120, 300, 600, 1800}, // 1s to 30min
		},
		[]string{"strategy"},
	)

	RolledBackDeploymentsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "warren_deployments_rolled_back_total",
			Help: "Total number of deployments that were rolled back",
		},
		[]string{"strategy", "reason"},
	)
)

func init() {
	// Register all metrics
	prometheus.MustRegister(NodesTotal)
	prometheus.MustRegister(ServicesTotal)
	prometheus.MustRegister(ContainersTotal)
	prometheus.MustRegister(SecretsTotal)
	prometheus.MustRegister(VolumesTotal)
	prometheus.MustRegister(RaftLeader)
	prometheus.MustRegister(RaftPeers)
	prometheus.MustRegister(RaftLogIndex)
	prometheus.MustRegister(RaftAppliedIndex)
	prometheus.MustRegister(APIRequestsTotal)
	prometheus.MustRegister(APIRequestDuration)
	prometheus.MustRegister(SchedulingLatency)
	prometheus.MustRegister(ContainersScheduled)
	prometheus.MustRegister(ContainersFailed)

	// Register operation latency metrics
	prometheus.MustRegister(ServiceCreateDuration)
	prometheus.MustRegister(ServiceUpdateDuration)
	prometheus.MustRegister(ServiceDeleteDuration)
	prometheus.MustRegister(ContainerCreateDuration)
	prometheus.MustRegister(ContainerStartDuration)
	prometheus.MustRegister(ContainerStopDuration)
	prometheus.MustRegister(RaftApplyDuration)
	prometheus.MustRegister(RaftCommitDuration)
	prometheus.MustRegister(ReconciliationDuration)
	prometheus.MustRegister(ReconciliationCyclesTotal)
	prometheus.MustRegister(IngressCreateDuration)
	prometheus.MustRegister(IngressUpdateDuration)
	prometheus.MustRegister(IngressRequestsTotal)
	prometheus.MustRegister(IngressRequestDuration)

	// Register deployment metrics
	prometheus.MustRegister(DeploymentsTotal)
	prometheus.MustRegister(DeploymentDuration)
	prometheus.MustRegister(RolledBackDeploymentsTotal)
}

// Handler returns the Prometheus HTTP handler
func Handler() http.Handler {
	return promhttp.Handler()
}

// Timer is a helper for timing operations
type Timer struct {
	start time.Time
}

// NewTimer creates a new timer
func NewTimer() *Timer {
	return &Timer{start: time.Now()}
}

// ObserveDuration records the duration to a histogram
func (t *Timer) ObserveDuration(histogram prometheus.Histogram) {
	duration := time.Since(t.start).Seconds()
	histogram.Observe(duration)
}

// ObserveDurationVec records the duration to a histogram vec with labels
func (t *Timer) ObserveDurationVec(histogram prometheus.ObserverVec, labels ...string) {
	duration := time.Since(t.start).Seconds()
	histogram.WithLabelValues(labels...).Observe(duration)
}

// Duration returns the elapsed time since timer started
func (t *Timer) Duration() time.Duration {
	return time.Since(t.start)
}
