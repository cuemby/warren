package metrics

import (
	"net/http"

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

	TasksTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "warren_tasks_total",
			Help: "Total number of tasks by state",
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
			Help:    "Time taken to schedule tasks in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)

	TasksScheduled = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "warren_tasks_scheduled_total",
			Help: "Total number of tasks scheduled",
		},
	)

	TasksFailed = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "warren_tasks_failed_total",
			Help: "Total number of failed tasks",
		},
	)
)

func init() {
	// Register all metrics
	prometheus.MustRegister(NodesTotal)
	prometheus.MustRegister(ServicesTotal)
	prometheus.MustRegister(TasksTotal)
	prometheus.MustRegister(SecretsTotal)
	prometheus.MustRegister(VolumesTotal)
	prometheus.MustRegister(RaftLeader)
	prometheus.MustRegister(RaftPeers)
	prometheus.MustRegister(RaftLogIndex)
	prometheus.MustRegister(RaftAppliedIndex)
	prometheus.MustRegister(APIRequestsTotal)
	prometheus.MustRegister(APIRequestDuration)
	prometheus.MustRegister(SchedulingLatency)
	prometheus.MustRegister(TasksScheduled)
	prometheus.MustRegister(TasksFailed)
}

// Handler returns the Prometheus HTTP handler
func Handler() http.Handler {
	return promhttp.Handler()
}
