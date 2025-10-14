# Warren Observability Guide

**Version**: 1.1.1+
**Last Updated**: 2025-10-13
**Status**: Production Ready

---

## Table of Contents

- [Overview](#overview)
- [Metrics](#metrics)
- [Health Checks](#health-checks)
- [Structured Logging](#structured-logging)
- [Monitoring Setup](#monitoring-setup)
- [Alerting Guidelines](#alerting-guidelines)
- [Troubleshooting](#troubleshooting)

---

## Overview

Warren provides comprehensive observability through three pillars:

1. **Metrics** - Prometheus-compatible metrics for monitoring
2. **Health Checks** - Kubernetes-style liveness/readiness probes
3. **Structured Logging** - JSON-formatted logs with context

### Endpoints

By default, Warren exposes observability endpoints on `127.0.0.1:9090`:

```
http://127.0.0.1:9090/metrics   # Prometheus metrics
http://127.0.0.1:9090/health    # Health check (overall system health)
http://127.0.0.1:9090/ready     # Readiness check (ready to serve traffic)
http://127.0.0.1:9090/live      # Liveness check (process is alive)
```

---

## Metrics

Warren exposes 30+ Prometheus metrics covering all critical components.

### Cluster State Metrics

**Nodes**
```
warren_nodes_total{role="manager|worker", status="ready|down"}
```
- **Type**: Gauge
- **Description**: Total number of nodes by role and status
- **Labels**: role, status
- **Use Case**: Monitor cluster size and node health

**Services**
```
warren_services_total
```
- **Type**: Gauge
- **Description**: Total number of services in the cluster
- **Use Case**: Track service count over time

**Containers**
```
warren_containers_total{state="pending|running|shutdown|failed"}
```
- **Type**: Gauge
- **Description**: Total number of containers by state
- **Labels**: state
- **Use Case**: Monitor container lifecycle and detect stuck states

**Secrets & Volumes**
```
warren_secrets_total
warren_volumes_total
```
- **Type**: Gauge
- **Description**: Total secrets and volumes in the cluster
- **Use Case**: Track resource growth

### Raft Consensus Metrics

**Leadership**
```
warren_raft_is_leader
```
- **Type**: Gauge
- **Description**: 1 if this node is the Raft leader, 0 otherwise
- **Use Case**: Monitor leadership changes and failovers

**Cluster Size**
```
warren_raft_peers_total
```
- **Type**: Gauge
- **Description**: Total number of Raft peers in the cluster
- **Use Case**: Verify expected cluster size

**Log Indices**
```
warren_raft_log_index
warren_raft_applied_index
```
- **Type**: Gauge
- **Description**: Current and last applied Raft log index
- **Use Case**: Monitor replication lag (log_index - applied_index)

**Raft Operations**
```
warren_raft_commit_duration_seconds
warren_raft_apply_duration_seconds
```
- **Type**: Histogram
- **Description**: Time to commit and apply Raft log entries
- **Buckets**: [0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10]
- **Use Case**: Detect slow consensus operations

**Example Query**:
```promql
# Average Raft commit latency (p50)
histogram_quantile(0.5, rate(warren_raft_commit_duration_seconds_bucket[5m]))

# Raft replication lag
warren_raft_log_index - warren_raft_applied_index
```

### Scheduler Metrics

**Scheduling Operations** ✨ *New in Phase 1 Week 2*
```
warren_scheduling_latency_seconds
```
- **Type**: Histogram
- **Description**: Time taken to schedule each container
- **Buckets**: Default Prometheus buckets
- **Use Case**: Track scheduling performance

```
warren_containers_scheduled_total
warren_containers_failed_total
```
- **Type**: Counter
- **Description**: Total successful and failed container schedules
- **Use Case**: Calculate scheduling success rate

**Example Query**:
```promql
# Scheduling success rate (last 5m)
rate(warren_containers_scheduled_total[5m]) /
  (rate(warren_containers_scheduled_total[5m]) + rate(warren_containers_failed_total[5m]))

# p95 scheduling latency
histogram_quantile(0.95, rate(warren_scheduling_latency_seconds_bucket[5m]))
```

### API Metrics

**Request Metrics**
```
warren_api_requests_total{method="CreateService|UpdateService|...", status="success|error"}
warren_api_request_duration_seconds{method="CreateService|..."}
```
- **Type**: Counter (requests_total), Histogram (duration)
- **Description**: API request count and latency by method
- **Labels**: method, status (requests_total only)
- **Use Case**: Monitor API performance and error rates

**Example Query**:
```promql
# API error rate
rate(warren_api_requests_total{status="error"}[5m]) /
  rate(warren_api_requests_total[5m])

# p99 API latency
histogram_quantile(0.99, rate(warren_api_request_duration_seconds_bucket[5m]))
```

### Service Operations

**Service CRUD**
```
warren_service_create_duration_seconds
warren_service_update_duration_seconds
warren_service_delete_duration_seconds
```
- **Type**: Histogram
- **Description**: Time taken for service operations
- **Use Case**: Monitor service operation performance

### Container Operations

**Container Lifecycle**
```
warren_container_create_duration_seconds
warren_container_start_duration_seconds
warren_container_stop_duration_seconds
```
- **Type**: Histogram
- **Description**: Time taken for container lifecycle operations
- **Use Case**: Detect slow container runtime operations

### Reconciler Metrics

**Reconciliation**
```
warren_reconciliation_duration_seconds
warren_reconciliation_cycles_total
```
- **Type**: Histogram (duration), Counter (cycles)
- **Description**: Reconciliation cycle timing and count
- **Use Case**: Monitor reconciler performance

**Example Query**:
```promql
# Reconciliation cycles per minute
rate(warren_reconciliation_cycles_total[1m]) * 60

# Average reconciliation duration
rate(warren_reconciliation_duration_seconds_sum[5m]) /
  rate(warren_reconciliation_duration_seconds_count[5m])
```

### Ingress Metrics

**Ingress Operations**
```
warren_ingress_create_duration_seconds
warren_ingress_update_duration_seconds
```
- **Type**: Histogram
- **Description**: Time taken for ingress operations
- **Use Case**: Monitor ingress configuration changes

**Ingress Traffic**
```
warren_ingress_requests_total{host="example.com", backend="service-name"}
warren_ingress_request_duration_seconds{host="example.com", backend="service-name"}
```
- **Type**: Counter (requests_total), Histogram (duration)
- **Description**: Ingress request count and latency by host/backend
- **Labels**: host, backend
- **Use Case**: Monitor ingress traffic patterns and performance

---

## Health Checks

Warren implements Kubernetes-style health checks with three endpoints:

### 1. Liveness Check - `/live`

**Purpose**: Indicates if the process is alive and should not be restarted.

**Response** (always 200 OK if process is running):
```json
{
  "status": "alive",
  "uptime": "1h23m45s"
}
```

**Use Case**: Kubernetes liveness probe, process monitoring

**Curl Example**:
```bash
curl http://localhost:9090/live
```

### 2. Health Check - `/health`

**Purpose**: Overall system health - checks all components.

**Response** (200 OK if healthy, 503 if unhealthy):
```json
{
  "status": "healthy",
  "timestamp": "2025-10-13T10:30:00Z",
  "version": "1.1.1",
  "uptime": "2h15m30s",
  "components": {
    "raft": "healthy",
    "containerd": "healthy",
    "api": "healthy"
  }
}
```

**Unhealthy Response** (503 Service Unavailable):
```json
{
  "status": "unhealthy",
  "timestamp": "2025-10-13T10:30:00Z",
  "version": "1.1.1",
  "uptime": "2h15m30s",
  "components": {
    "raft": "healthy",
    "containerd": "unhealthy: connection failed",
    "api": "healthy"
  }
}
```

**Use Case**: General health monitoring, alerting

### 3. Readiness Check - `/ready`

**Purpose**: Indicates if the node is ready to serve traffic (all critical components are operational).

**Critical Components**:
- `raft` - Raft consensus initialized
- `containerd` - Container runtime connected
- `api` - gRPC API server running

**Ready Response** (200 OK):
```json
{
  "status": "ready",
  "timestamp": "2025-10-13T10:30:00Z",
  "version": "1.1.1",
  "uptime": "2h15m30s",
  "components": {
    "raft": "ready",
    "containerd": "ready",
    "api": "ready"
  }
}
```

**Not Ready Response** (503 Service Unavailable):
```json
{
  "status": "not_ready",
  "message": "waiting for containerd",
  "timestamp": "2025-10-13T10:30:00Z",
  "version": "1.1.1",
  "uptime": "30s",
  "components": {
    "raft": "ready",
    "containerd": "not ready: initializing",
    "api": "ready"
  }
}
```

**Use Case**: Load balancer health checks, Kubernetes readiness probe

### Kubernetes Integration

**Liveness Probe**:
```yaml
livenessProbe:
  httpGet:
    path: /live
    port: 9090
  initialDelaySeconds: 30
  periodSeconds: 10
  timeoutSeconds: 5
  failureThreshold: 3
```

**Readiness Probe**:
```yaml
readinessProbe:
  httpGet:
    path: /ready
    port: 9090
  initialDelaySeconds: 10
  periodSeconds: 5
  timeoutSeconds: 3
  failureThreshold: 3
```

---

## Structured Logging

Warren uses **zerolog** for structured JSON logging with context fields.

### Log Levels

- **DEBUG**: Detailed debugging information
- **INFO**: General informational messages
- **WARN**: Warning messages (non-critical issues)
- **ERROR**: Error messages (failures that don't stop execution)
- **FATAL**: Critical errors (process termination)

### Log Format

```json
{
  "level": "info",
  "component": "scheduler",
  "container_id": "abc123",
  "service_name": "nginx",
  "node_id": "worker-1",
  "time": "2025-10-13T10:30:00Z",
  "message": "Created container"
}
```

### Context Fields

Warren logs include relevant context fields:

**Scheduler**:
- `component`: "scheduler"
- `service_name`, `service_id`: Service being scheduled
- `container_id`: Container ID
- `node_id`: Selected node
- `volume_name`: Volume affinity (if applicable)

**Manager/Raft**:
- `component`: "manager"
- `operation`: Raft operation type

**API Server**:
- `component`: "api"
- `method`: gRPC method name
- `service_name`, `service_id`: Affected service

### Log Configuration

Set log level via environment variable:
```bash
export LOG_LEVEL=debug  # debug, info, warn, error
warren manager ...
```

### Parsing Logs

**jq Examples**:
```bash
# Filter errors only
warren manager ... 2>&1 | jq 'select(.level == "error")'

# Track container scheduling
warren manager ... 2>&1 | jq 'select(.component == "scheduler" and .message == "Created container")'

# Monitor specific service
warren manager ... 2>&1 | jq 'select(.service_name == "nginx")'
```

---

## Monitoring Setup

### Prometheus Configuration

**prometheus.yml**:
```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'warren-managers'
    static_configs:
      - targets:
          - 'manager-1:9090'
          - 'manager-2:9090'
          - 'manager-3:9090'
    relabel_configs:
      - source_labels: [__address__]
        target_label: instance

  - job_name: 'warren-workers'
    static_configs:
      - targets:
          - 'worker-1:9090'
          - 'worker-2:9090'
          - 'worker-3:9090'
```

### Grafana Dashboard

**Key Panels**:

1. **Cluster Overview**
   - Total nodes (by role/status)
   - Total services
   - Total containers (by state)

2. **Raft Health**
   - Leadership changes (counter)
   - Peer count
   - Replication lag
   - Commit/apply latency (p50, p95, p99)

3. **Scheduler Performance**
   - Scheduling latency (p50, p95, p99)
   - Scheduling success rate
   - Failed schedules (rate)

4. **API Performance**
   - Request rate (by method)
   - Error rate
   - Latency (p50, p95, p99)

5. **Service Operations**
   - Service create/update/delete latency
   - Container lifecycle latency

6. **Reconciler**
   - Reconciliation cycles/min
   - Reconciliation duration

**Example Queries**: See metric sections above for PromQL examples.

---

## Alerting Guidelines

### Critical Alerts

**No Raft Leader**:
```yaml
- alert: WarrenNoLeader
  expr: sum(warren_raft_is_leader) == 0
  for: 1m
  severity: critical
  annotations:
    summary: "No Raft leader elected"
    description: "Warren cluster has no leader for 1 minute"
```

**High Scheduling Failure Rate**:
```yaml
- alert: WarrenHighSchedulingFailureRate
  expr: |
    rate(warren_containers_failed_total[5m]) /
    (rate(warren_containers_scheduled_total[5m]) + rate(warren_containers_failed_total[5m])) > 0.1
  for: 5m
  severity: critical
  annotations:
    summary: "High container scheduling failure rate"
    description: "{{ $value | humanizePercentage }} of container schedules are failing"
```

### Warning Alerts

**Slow Raft Commits**:
```yaml
- alert: WarrenSlowRaftCommits
  expr: histogram_quantile(0.95, rate(warren_raft_commit_duration_seconds_bucket[5m])) > 1
  for: 5m
  severity: warning
  annotations:
    summary: "Slow Raft commit operations"
    description: "p95 Raft commit latency is {{ $value }}s"
```

**High API Error Rate**:
```yaml
- alert: WarrenHighAPIErrorRate
  expr: |
    rate(warren_api_requests_total{status="error"}[5m]) /
    rate(warren_api_requests_total[5m]) > 0.05
  for: 5m
  severity: warning
  annotations:
    summary: "High API error rate"
    description: "{{ $value | humanizePercentage }} of API requests are failing"
```

---

## Troubleshooting

### High Scheduling Latency

**Symptoms**: Containers take >5s to schedule
**Diagnosis**:
```promql
histogram_quantile(0.95, rate(warren_scheduling_latency_seconds_bucket[5m]))
```

**Possible Causes**:
1. Storage slow (BoltDB on slow disk)
2. High container churn
3. Complex volume affinity rules

**Solution**:
- Check storage latency
- Review service placement constraints
- Scale worker nodes

### Raft Replication Lag

**Symptoms**: `warren_raft_log_index` >> `warren_raft_applied_index`
**Diagnosis**:
```promql
warren_raft_log_index - warren_raft_applied_index
```

**Possible Causes**:
1. Slow disk I/O on managers
2. Network latency between managers
3. Heavy write load

**Solution**:
- Use SSDs for manager data directory
- Check network latency between managers
- Review Raft tuning parameters

### Container Scheduling Failures

**Symptoms**: `warren_containers_failed_total` increasing
**Diagnosis**:
```bash
# Check logs for scheduling errors
warren manager ... 2>&1 | jq 'select(.component == "scheduler" and .level == "error")'
```

**Possible Causes**:
1. No suitable nodes available
2. Volume affinity constraints
3. Resource exhaustion

**Solution**:
- Check node readiness: `warren node ls`
- Review service constraints
- Scale cluster

---

## Best Practices

1. **Metrics Retention**: Keep 15 days for troubleshooting
2. **Alert Fatigue**: Start with critical alerts only, add warnings gradually
3. **Dashboard Organization**: Separate manager and worker dashboards
4. **Log Aggregation**: Use ELK/Loki for centralized logs
5. **Baseline Metrics**: Establish baselines during normal operation
6. **Regular Review**: Review metrics weekly to detect trends

---

## Appendix: Metrics Reference

### Complete Metrics List

**Cluster State** (6 metrics):
- `warren_nodes_total`
- `warren_services_total`
- `warren_containers_total`
- `warren_secrets_total`
- `warren_volumes_total`
- (Future: `warren_networks_total`)

**Raft** (6 metrics):
- `warren_raft_is_leader`
- `warren_raft_peers_total`
- `warren_raft_log_index`
- `warren_raft_applied_index`
- `warren_raft_commit_duration_seconds`
- `warren_raft_apply_duration_seconds`

**Scheduler** (3 metrics):
- `warren_scheduling_latency_seconds`
- `warren_containers_scheduled_total`
- `warren_containers_failed_total`

**API** (2 metrics):
- `warren_api_requests_total`
- `warren_api_request_duration_seconds`

**Service Operations** (3 metrics):
- `warren_service_create_duration_seconds`
- `warren_service_update_duration_seconds`
- `warren_service_delete_duration_seconds`

**Container Operations** (3 metrics):
- `warren_container_create_duration_seconds`
- `warren_container_start_duration_seconds`
- `warren_container_stop_duration_seconds`

**Reconciler** (2 metrics):
- `warren_reconciliation_duration_seconds`
- `warren_reconciliation_cycles_total`

**Ingress** (4 metrics):
- `warren_ingress_create_duration_seconds`
- `warren_ingress_update_duration_seconds`
- `warren_ingress_requests_total`
- `warren_ingress_request_duration_seconds`

**Total**: 30+ metrics

---

## See Also

- [Performance Benchmarking Guide](performance-benchmarking.md)
- [Profiling Guide](profiling.md)
- [Troubleshooting Guide](troubleshooting.md)
- [Raft Tuning Guide](raft-tuning.md)
