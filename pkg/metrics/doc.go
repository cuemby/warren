/*
Package metrics provides Prometheus metrics collection and exposition for Warren.

The metrics package defines and registers all Warren metrics using the Prometheus
client library, providing observability into cluster health, resource utilization,
operation latency, and system performance. Metrics are exposed via HTTP endpoint
for scraping by Prometheus servers.

# Architecture

Warren's metrics system follows Prometheus best practices with comprehensive
instrumentation across all components:

	┌──────────────────── METRICS SYSTEM ──────────────────────┐
	│                                                            │
	│  ┌────────────────────────────────────────────┐          │
	│  │          Prometheus Registry                │          │
	│  │  - Global DefaultRegistry                   │          │
	│  │  - MustRegister at package init             │          │
	│  │  - Automatic Go runtime metrics             │          │
	│  └──────────────────┬─────────────────────────┘          │
	│                     │                                      │
	│  ┌──────────────────▼─────────────────────────┐          │
	│  │              Metric Types                   │          │
	│  │                                              │          │
	│  │  Gauge: Instant values (node count)         │          │
	│  │  Counter: Monotonic increases (requests)    │          │
	│  │  Histogram: Distributions (latency)         │          │
	│  │  Summary: Quantiles (percentiles)           │          │
	│  └──────────────────┬─────────────────────────┘          │
	│                     │                                      │
	│  ┌──────────────────▼─────────────────────────┐          │
	│  │           Metric Categories                 │          │
	│  │                                              │          │
	│  │  Cluster: Nodes, services, tasks            │          │
	│  │  Raft: Leader status, log index, peers      │          │
	│  │  API: Request count, duration               │          │
	│  │  Scheduler: Latency, scheduled count        │          │
	│  │  Operations: Create/update/delete duration  │          │
	│  │  Reconciler: Cycle duration, count          │          │
	│  │  Ingress: Request count, duration, errors   │          │
	│  └──────────────────┬─────────────────────────┘          │
	│                     │                                      │
	│  ┌──────────────────▼─────────────────────────┐          │
	│  │          HTTP Metrics Endpoint              │          │
	│  │  - Path: /metrics                           │          │
	│  │  - Format: Prometheus text exposition       │          │
	│  │  - Handler: promhttp.Handler()              │          │
	│  └──────────────────┬─────────────────────────┘          │
	│                     │                                      │
	│  ┌──────────────────▼─────────────────────────┐          │
	│  │         Prometheus Server                   │          │
	│  │  - Scrapes /metrics every 15s               │          │
	│  │  - Stores time series data                  │          │
	│  │  - Provides PromQL query interface          │          │
	│  └────────────────────────────────────────────┘           │
	└────────────────────────────────────────────────────────┘

# Core Components

Metric Registry:
  - Global Prometheus DefaultRegistry
  - All metrics registered at package init
  - Automatic collection of Go runtime metrics
  - Thread-safe for concurrent updates

Gauge Metrics:
  - Instant value that can go up or down
  - Examples: node count, task count, Raft leader status
  - Operations: Set, Inc, Dec, Add, Sub

Counter Metrics:
  - Monotonically increasing value
  - Examples: requests total, tasks scheduled total
  - Operations: Inc, Add (cannot decrease)

Histogram Metrics:
  - Distribution of observed values
  - Buckets for latency percentiles (p50, p95, p99)
  - Examples: API request duration, scheduling latency
  - Includes: sum, count, buckets

Timer Helper:
  - Convenience wrapper for timing operations
  - Start timer, observe duration to histogram
  - Supports label values for histogram vectors

# Metrics Catalog

Cluster Metrics:

warren_nodes_total{role, status}:
  - Type: Gauge
  - Description: Total nodes by role (manager/worker) and status (ready/down)
  - Labels: role, status
  - Example: warren_nodes_total{role="worker",status="ready"} 5

warren_services_total:
  - Type: Gauge
  - Description: Total number of services in cluster
  - Example: warren_services_total 10

warren_tasks_total{state}:
  - Type: Gauge
  - Description: Total tasks by state (pending/running/failed)
  - Labels: state
  - Example: warren_tasks_total{state="running"} 30

warren_secrets_total:
  - Type: Gauge
  - Description: Total number of secrets stored
  - Example: warren_secrets_total 5

warren_volumes_total:
  - Type: Gauge
  - Description: Total number of volumes
  - Example: warren_volumes_total 8

Raft Metrics:

warren_raft_is_leader:
  - Type: Gauge
  - Description: Whether this node is Raft leader (1=leader, 0=follower)
  - Example: warren_raft_is_leader 1

warren_raft_peers_total:
  - Type: Gauge
  - Description: Total Raft peers in cluster
  - Example: warren_raft_peers_total 3

warren_raft_log_index:
  - Type: Gauge
  - Description: Current Raft log index
  - Example: warren_raft_log_index 1543

warren_raft_applied_index:
  - Type: Gauge
  - Description: Last applied Raft log index
  - Example: warren_raft_applied_index 1543

API Metrics:

warren_api_requests_total{method, status}:
  - Type: Counter
  - Description: Total API requests by method and status
  - Labels: method, status
  - Example: warren_api_requests_total{method="CreateService",status="200"} 100

warren_api_request_duration_seconds{method}:
  - Type: Histogram
  - Description: API request duration in seconds
  - Labels: method
  - Buckets: 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10

Scheduler Metrics:

warren_scheduling_latency_seconds:
  - Type: Histogram
  - Description: Time to schedule tasks in seconds
  - Buckets: Default Prometheus buckets

warren_tasks_scheduled_total:
  - Type: Counter
  - Description: Total tasks successfully scheduled
  - Example: warren_tasks_scheduled_total 250

warren_tasks_failed_total:
  - Type: Counter
  - Description: Total tasks that failed
  - Example: warren_tasks_failed_total 5

Operation Latency Metrics:

warren_service_create_duration_seconds:
  - Type: Histogram
  - Description: Time to create a service

warren_service_update_duration_seconds:
  - Type: Histogram
  - Description: Time to update a service

warren_service_delete_duration_seconds:
  - Type: Histogram
  - Description: Time to delete a service

warren_task_create_duration_seconds:
  - Type: Histogram
  - Description: Time to create a task

warren_task_start_duration_seconds:
  - Type: Histogram
  - Description: Time to start a task container

warren_task_stop_duration_seconds:
  - Type: Histogram
  - Description: Time to stop a task container

Raft Operation Metrics:

warren_raft_apply_duration_seconds:
  - Type: Histogram
  - Description: Time to apply Raft log entry

warren_raft_commit_duration_seconds:
  - Type: Histogram
  - Description: Time to commit Raft log entry

Reconciler Metrics:

warren_reconciliation_duration_seconds:
  - Type: Histogram
  - Description: Reconciliation cycle duration

warren_reconciliation_cycles_total:
  - Type: Counter
  - Description: Total reconciliation cycles completed

Ingress Metrics:

warren_ingress_create_duration_seconds:
  - Type: Histogram
  - Description: Time to create ingress rule

warren_ingress_update_duration_seconds:
  - Type: Histogram
  - Description: Time to update ingress rule

warren_ingress_requests_total{host, backend}:
  - Type: Counter
  - Description: Total ingress requests by host and backend
  - Labels: host, backend

warren_ingress_request_duration_seconds{host, backend}:
  - Type: Histogram
  - Description: Ingress request duration by host and backend
  - Labels: host, backend

# Usage

Updating Gauge Metrics:

	import "github.com/cuemby/warren/pkg/metrics"

	// Set absolute value
	metrics.NodesTotal.WithLabelValues("worker", "ready").Set(5)

	// Increment/decrement
	metrics.ServicesTotal.Inc()
	metrics.ServicesTotal.Dec()

Updating Counter Metrics:

	// Increment by 1
	metrics.TasksScheduled.Inc()

	// Add arbitrary value
	metrics.APIRequestsTotal.WithLabelValues("CreateService", "200").Add(1)

Recording Histogram Observations:

	// Direct observation
	metrics.SchedulingLatency.Observe(0.125) // 125ms

	// Using Timer helper
	timer := metrics.NewTimer()
	// ... perform operation ...
	timer.ObserveDuration(metrics.ServiceCreateDuration)

Using Timer with Labels:

	timer := metrics.NewTimer()
	// ... perform operation ...
	timer.ObserveDurationVec(metrics.APIRequestDuration, "CreateService")

Complete Example:

	package main

	import (
		"net/http"
		"time"
		"github.com/cuemby/warren/pkg/metrics"
	)

	func main() {
		// Update cluster metrics
		metrics.NodesTotal.WithLabelValues("manager", "ready").Set(3)
		metrics.NodesTotal.WithLabelValues("worker", "ready").Set(5)
		metrics.ServicesTotal.Set(10)
		metrics.TasksTotal.WithLabelValues("running").Set(30)

		// Time an operation
		timer := metrics.NewTimer()
		createService()
		timer.ObserveDuration(metrics.ServiceCreateDuration)

		// Expose metrics endpoint
		http.Handle("/metrics", metrics.Handler())
		http.ListenAndServe(":9090", nil)
	}

	func createService() {
		// Service creation logic
		time.Sleep(100 * time.Millisecond)
	}

# Integration Points

This package integrates with:

  - pkg/manager: Updates cluster and Raft metrics
  - pkg/scheduler: Records scheduling latency
  - pkg/reconciler: Tracks reconciliation cycles
  - pkg/api: Instruments API request duration
  - pkg/worker: Reports task execution metrics
  - pkg/ingress: Tracks HTTP request metrics
  - Prometheus: Scrapes /metrics endpoint

# Design Patterns

Package Init Registration:
  - All metrics registered in init() function
  - MustRegister panics on duplicate registration
  - Ensures metrics available before main()
  - No runtime registration needed

Label Discipline:
  - Use WithLabelValues for cardinality-bounded labels
  - Avoid high-cardinality labels (IDs, timestamps)
  - Document label values in metric description
  - Keep label count low (< 5 per metric)

Timer Pattern:
  - Create timer at operation start
  - Defer or explicitly call ObserveDuration
  - Automatically calculates elapsed time
  - Supports both simple and vector histograms

Global Metrics:
  - Package-level variables for all metrics
  - Accessible from any Warren package
  - Thread-safe concurrent updates
  - No initialization required by callers

# Performance Characteristics

Metric Update Overhead:
  - Gauge set/inc: ~50ns per operation
  - Counter inc: ~50ns per operation
  - Histogram observe: ~200ns per operation
  - Labels: +100ns per label value
  - Negligible impact on hot path

Memory Usage:
  - Per metric: ~1KB baseline
  - Per label combination: ~100 bytes
  - Histogram buckets: ~50 bytes each
  - Total: ~1-5MB for typical Warren cluster

Scrape Performance:
  - Metrics gathering: ~1-5ms for full scrape
  - HTTP response: ~10ms for typical metric set
  - Recommendation: Scrape interval ≥ 15s
  - Concurrent scrapes: Safe (read-only)

Cardinality Management:
  - Low cardinality: role, status, state (< 10 values)
  - Medium cardinality: method, host (< 100 values)
  - Avoid: task IDs, timestamps (unbounded)
  - Best practice: Aggregate high-cardinality in logs

# Troubleshooting

Common Issues:

Missing Metrics:
  - Symptom: Metric not appearing in /metrics output
  - Check: Metric registered in init() function
  - Check: MustRegister called (panics if duplicate)
  - Solution: Verify metric variable is exported

High Cardinality:
  - Symptom: Prometheus memory usage grows
  - Cause: Using IDs or unbounded values as labels
  - Check: Label cardinality (count unique combinations)
  - Solution: Remove high-cardinality labels, aggregate differently

Histogram Bucket Mismatch:
  - Symptom: No data in desired percentiles
  - Cause: Buckets don't cover observed value range
  - Check: Histogram sum / count for average
  - Solution: Customize buckets for value range

Stale Metrics:
  - Symptom: Metrics not updating
  - Cause: Code not calling metric update methods
  - Check: Add logging around metric updates
  - Solution: Instrument code paths correctly

# Monitoring

Prometheus Queries (PromQL):

Node Health:
  - Total nodes: sum(warren_nodes_total)
  - Ready workers: warren_nodes_total{role="worker",status="ready"}
  - Down nodes: warren_nodes_total{status="down"}

Service Health:
  - Total services: warren_services_total
  - Running tasks: warren_tasks_total{state="running"}
  - Failed tasks: warren_tasks_total{state="failed"}
  - Task failure rate: rate(warren_tasks_failed_total[5m])

API Performance:
  - Request rate: rate(warren_api_requests_total[1m])
  - Error rate: rate(warren_api_requests_total{status=~"5.."}[1m])
  - p95 latency: histogram_quantile(0.95, warren_api_request_duration_seconds_bucket)
  - p99 latency: histogram_quantile(0.99, warren_api_request_duration_seconds_bucket)

Raft Health:
  - Has leader: max(warren_raft_is_leader) > 0
  - Leader changes: changes(warren_raft_is_leader[10m])
  - Log lag: warren_raft_log_index - warren_raft_applied_index
  - Peer count: warren_raft_peers_total

Scheduler Performance:
  - Scheduling rate: rate(warren_tasks_scheduled_total[1m])
  - p95 scheduling latency: histogram_quantile(0.95, warren_scheduling_latency_seconds_bucket)
  - Scheduling failures: rate(warren_tasks_failed_total[5m])

# Alerting Rules

Recommended Prometheus alerts:

High Task Failure Rate:
  - Alert: rate(warren_tasks_failed_total[5m]) > 0.1
  - Description: More than 0.1 tasks failing per second
  - Action: Check scheduler logs, node health, image availability

No Raft Leader:
  - Alert: max(warren_raft_is_leader) == 0
  - Description: Cluster has no Raft leader
  - Action: Check manager connectivity, quorum status

Frequent Leader Changes:
  - Alert: changes(warren_raft_is_leader[10m]) > 3
  - Description: Leader changed more than 3 times in 10 minutes
  - Action: Check network latency, manager load

High API Latency:
  - Alert: histogram_quantile(0.95, warren_api_request_duration_seconds_bucket) > 1
  - Description: p95 API latency > 1 second
  - Action: Check Raft performance, database size

# Grafana Dashboards

Recommended dashboard panels:

Cluster Overview:
  - Gauge: Total nodes (workers + managers)
  - Gauge: Total services
  - Time series: Tasks by state (running, pending, failed)
  - Time series: Task failure rate

API Performance:
  - Time series: Request rate by method
  - Time series: p95 and p99 latency
  - Time series: Error rate (5xx responses)

Raft Health:
  - Single stat: Leader status (yes/no)
  - Time series: Log index and applied index
  - Single stat: Peer count
  - Time series: Leader changes

Scheduler Performance:
  - Time series: Tasks scheduled per second
  - Heatmap: Scheduling latency distribution
  - Time series: Scheduling failures

# See Also

  - Prometheus documentation: https://prometheus.io/docs/
  - Prometheus client library: https://github.com/prometheus/client_golang
  - PromQL tutorial: https://prometheus.io/docs/prometheus/latest/querying/basics/
  - Histogram best practices: https://prometheus.io/docs/practices/histograms/
*/
package metrics
