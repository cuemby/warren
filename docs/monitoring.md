# Warren Monitoring Guide

**Version**: 1.3.1
**Last Updated**: 2025-10-14
**Audience**: Platform Engineers, SREs, Operations Teams

---

## Overview

Warren provides built-in observability through HTTP health check endpoints and Prometheus metrics. This guide covers how to monitor Warren clusters in production.

---

## Health Check Endpoints

Warren exposes three HTTP endpoints for monitoring:

### `/health` - Liveness Check

**Purpose**: Basic liveness probe - returns 200 if the Warren process is alive.

**Use Case**: Kubernetes liveness probes, uptime monitoring

**Request**:
```bash
curl http://localhost:9090/health
```

**Response** (HTTP 200):
```json
{
  "status": "healthy",
  "timestamp": "2025-10-14T12:30:00Z",
  "version": "1.3.1"
}
```

**Fields**:
- `status`: Always "healthy" if process is running
- `timestamp`: Current server time
- `version`: Warren version

**Recommended Check**:
- Interval: 10 seconds
- Timeout: 5 seconds
- Failure Threshold: 3 consecutive failures

---

### `/ready` - Readiness Check

**Purpose**: Comprehensive readiness probe - checks if Warren can accept traffic.

**Use Case**: Kubernetes readiness probes, load balancer health checks

**Request**:
```bash
curl http://localhost:9090/ready
```

**Response (Ready)** (HTTP 200):
```json
{
  "status": "ready",
  "timestamp": "2025-10-14T12:30:00Z",
  "checks": {
    "raft": "leader",
    "storage": "ok"
  }
}
```

**Response (Not Ready)** (HTTP 503):
```json
{
  "status": "not ready",
  "timestamp": "2025-10-14T12:30:00Z",
  "checks": {
    "raft": "no leader elected",
    "storage": "not initialized"
  },
  "message": "Waiting for leader election"
}
```

**Checks Performed**:
1. **Raft Status**: Leader/follower state, leader availability
2. **Storage**: Database accessibility
3. **Event Broker**: (Future) Event system health

**Recommended Check**:
- Interval: 5 seconds
- Timeout: 3 seconds
- Failure Threshold: 2 consecutive failures

---

### `/metrics` - Prometheus Metrics

**Purpose**: Detailed operational metrics in Prometheus format.

**Use Case**: Prometheus scraping, Grafana dashboards, alerting

**Request**:
```bash
curl http://localhost:9090/metrics
```

**Response**: Prometheus text format
```
# HELP warren_nodes_total Total number of nodes by role and status
# TYPE warren_nodes_total gauge
warren_nodes_total{role="manager",status="ready"} 3
warren_nodes_total{role="worker",status="ready"} 5
...
```

**Recommended Scrape**:
- Interval: 15 seconds (default Prometheus)
- Timeout: 10 seconds

---

## Prometheus Metrics

### Cluster Metrics

| Metric | Type | Description | Labels |
|--------|------|-------------|--------|
| `warren_nodes_total` | Gauge | Total nodes in cluster | `role`, `status` |
| `warren_services_total` | Gauge | Total services | - |
| `warren_containers_total` | Gauge | Total containers | `state` |
| `warren_secrets_total` | Gauge | Total secrets | - |
| `warren_volumes_total` | Gauge | Total volumes | - |

**Example Queries**:
```promql
# Total nodes by role
sum(warren_nodes_total) by (role)

# Running containers
warren_containers_total{state="running"}

# Worker nodes that are ready
warren_nodes_total{role="worker",status="ready"}
```

---

### Raft Consensus Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `warren_raft_is_leader` | Gauge | 1 if leader, 0 if follower |
| `warren_raft_peers_total` | Gauge | Number of Raft peers |
| `warren_raft_log_index` | Gauge | Current Raft log index |
| `warren_raft_applied_index` | Gauge | Last applied log index |
| `warren_raft_apply_duration_seconds` | Histogram | Raft apply latency |
| `warren_raft_commit_duration_seconds` | Histogram | Raft commit latency |

**Example Queries**:
```promql
# Current leader
warren_raft_is_leader == 1

# Raft replication lag
warren_raft_log_index - warren_raft_applied_index

# Raft apply latency (p95)
histogram_quantile(0.95, rate(warren_raft_apply_duration_seconds_bucket[5m]))
```

---

### Service Operation Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `warren_service_create_duration_seconds` | Histogram | Service creation time |
| `warren_service_update_duration_seconds` | Histogram | Service update time |
| `warren_service_delete_duration_seconds` | Histogram | Service deletion time |

**Example Queries**:
```promql
# Service creation latency (p99)
histogram_quantile(0.99, rate(warren_service_create_duration_seconds_bucket[5m]))

# Service operations per second
rate(warren_service_create_duration_seconds_count[1m])
```

---

### Container Operation Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `warren_containers_scheduled_total` | Counter | Total containers scheduled |
| `warren_containers_failed_total` | Counter | Total failed containers |
| `warren_container_create_duration_seconds` | Histogram | Container creation time |
| `warren_container_start_duration_seconds` | Histogram | Container start time |
| `warren_container_stop_duration_seconds` | Histogram | Container stop time |

**Example Queries**:
```promql
# Container failure rate
rate(warren_containers_failed_total[5m])

# Container creation latency (p50)
histogram_quantile(0.50, rate(warren_container_create_duration_seconds_bucket[5m]))

# Containers scheduled per minute
rate(warren_containers_scheduled_total[1m]) * 60
```

---

### Scheduler Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `warren_scheduling_latency_seconds` | Histogram | Scheduling cycle duration |

**Example Queries**:
```promql
# Scheduling latency (p95)
histogram_quantile(0.95, rate(warren_scheduling_latency_seconds_bucket[5m]))
```

---

### Reconciler Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `warren_reconciliation_duration_seconds` | Histogram | Reconciliation cycle time |
| `warren_reconciliation_cycles_total` | Counter | Total reconciliation cycles |

**Example Queries**:
```promql
# Reconciliation cycles per minute
rate(warren_reconciliation_cycles_total[1m]) * 60

# Reconciliation latency (p99)
histogram_quantile(0.99, rate(warren_reconciliation_duration_seconds_bucket[5m]))
```

---

### Ingress Metrics

| Metric | Type | Description | Labels |
|--------|------|-------------|--------|
| `warren_ingress_requests_total` | Counter | Total ingress requests | `host`, `backend` |
| `warren_ingress_request_duration_seconds` | Histogram | Ingress request latency | `host`, `backend` |
| `warren_ingress_create_duration_seconds` | Histogram | Ingress rule creation time | - |
| `warren_ingress_update_duration_seconds` | Histogram | Ingress rule update time | - |

**Example Queries**:
```promql
# Requests per second by host
rate(warren_ingress_requests_total[1m])

# Request latency (p95) by host
histogram_quantile(0.95, rate(warren_ingress_request_duration_seconds_bucket[5m])) by (host)

# Top backends by request volume
topk(5, sum(rate(warren_ingress_requests_total[5m])) by (backend))
```

---

### Deployment Metrics

| Metric | Type | Description | Labels |
|--------|------|-------------|--------|
| `warren_deployments_total` | Counter | Total deployments | `strategy`, `status` |
| `warren_deployment_duration_seconds` | Histogram | Deployment duration | `strategy` |
| `warren_deployments_rolled_back_total` | Counter | Rolled back deployments | `strategy`, `reason` |

**Example Queries**:
```promql
# Deployment success rate
rate(warren_deployments_total{status="success"}[5m])
/
rate(warren_deployments_total[5m])

# Rolling update duration (p95)
histogram_quantile(0.95, rate(warren_deployment_duration_seconds_bucket{strategy="rolling"}[5m]))

# Rollback rate
rate(warren_deployments_rolled_back_total[5m])
```

---

## Kubernetes Integration

### Liveness Probe

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 9090
  initialDelaySeconds: 10
  periodSeconds: 10
  timeoutSeconds: 5
  failureThreshold: 3
```

### Readiness Probe

```yaml
readinessProbe:
  httpGet:
    path: /ready
    port: 9090
  initialDelaySeconds: 5
  periodSeconds: 5
  timeoutSeconds: 3
  failureThreshold: 2
```

### Prometheus ServiceMonitor

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: warren-metrics
  namespace: warren-system
spec:
  selector:
    matchLabels:
      app: warren-manager
  endpoints:
  - port: http-metrics
    path: /metrics
    interval: 15s
    scrapeTimeout: 10s
```

---

## Alert Rules

### Recommended Prometheus Alerts

```yaml
groups:
- name: warren
  rules:
  # Cluster Health
  - alert: WarrenNoLeader
    expr: sum(warren_raft_is_leader) == 0
    for: 1m
    labels:
      severity: critical
    annotations:
      summary: "No Raft leader elected"
      description: "Warren cluster has no leader for 1 minute"

  - alert: WarrenQuorumLost
    expr: warren_raft_peers_total < 2
    for: 30s
    labels:
      severity: critical
    annotations:
      summary: "Raft quorum lost"
      description: "Warren has fewer than 2 peers (no quorum)"

  # Performance
  - alert: WarrenHighSchedulingLatency
    expr: histogram_quantile(0.95, rate(warren_scheduling_latency_seconds_bucket[5m])) > 10
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "High scheduling latency"
      description: "P95 scheduling latency > 10s"

  - alert: WarrenHighContainerFailureRate
    expr: rate(warren_containers_failed_total[5m]) > 0.1
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "High container failure rate"
      description: "Container failure rate > 0.1/sec"

  # Capacity
  - alert: WarrenNoReadyWorkers
    expr: warren_nodes_total{role="worker",status="ready"} == 0
    for: 1m
    labels:
      severity: critical
    annotations:
      summary: "No ready worker nodes"
      description: "All worker nodes are down or not ready"
```

---

## Grafana Dashboard

### Sample Dashboard JSON

See [grafana-dashboard.json](./grafana-dashboard.json) for a pre-built Warren monitoring dashboard with:

- Cluster overview (nodes, services, containers)
- Raft health (leader status, log indices, replication lag)
- Performance metrics (latency histograms, throughput)
- Container lifecycle (create/start/stop duration)
- Ingress traffic (requests/sec, latency by host)
- Deployment tracking (success rate, rollbacks)

### Key Panels

1. **Cluster Health**: Node status, Raft leader, quorum state
2. **Service Overview**: Total services, containers by state
3. **Performance**: P50/P95/P99 latency for key operations
4. **Throughput**: Operations per second (scheduling, reconciliation)
5. **Errors**: Failed containers, rollbacks, API errors

---

## Monitoring Best Practices

### 1. Alert on Symptoms, Not Causes

❌ Don't: Alert on high CPU usage
✅ Do: Alert on high scheduling latency or container failure rate

### 2. Use Multi-Window Multi-Burn-Rate Alerts

For SLOs, use multiple time windows (short + long) to catch both fast and slow burns.

### 3. Monitor Golden Signals

1. **Latency**: Service creation, scheduling, API response time
2. **Traffic**: Requests/sec, containers scheduled/min
3. **Errors**: Container failures, Raft errors, API errors
4. **Saturation**: Node capacity, scheduling queue depth

### 4. Set Realistic Thresholds

Based on Phase 1 benchmarking:
- Service creation: <2s (p99)
- Task scheduling: <5s (p99)
- API latency: <100ms (p99)

### 5. Dashboard Organization

Organize dashboards by role:
- **Overview**: High-level cluster health (for on-call)
- **Performance**: Deep-dive into latency and throughput (for SREs)
- **Capacity**: Resource utilization and scaling (for capacity planning)

---

## Troubleshooting with Metrics

### Problem: Slow Service Creation

**Check**:
```promql
histogram_quantile(0.99, rate(warren_service_create_duration_seconds_bucket[5m]))
```

**If high**:
1. Check Raft apply latency (disk I/O?)
2. Check scheduler latency (resource constraints?)
3. Check API latency (network issues?)

### Problem: High Container Failure Rate

**Check**:
```promql
rate(warren_containers_failed_total[5m])
```

**Investigate**:
1. Which nodes? (Node resource exhaustion?)
2. Which services? (Image pull failures? Health check issues?)
3. Timing pattern? (Deployment related? Network partition?)

### Problem: Raft Replication Lag

**Check**:
```promql
warren_raft_log_index - warren_raft_applied_index
```

**If growing**:
1. Check Raft apply duration (slow FSM?)
2. Check disk I/O (BoltDB performance?)
3. Check network latency (inter-manager communication?)

---

## Next Steps

1. **Set up Prometheus**: Configure scraping of Warren `/metrics` endpoint
2. **Import Grafana Dashboard**: Use provided JSON template
3. **Configure Alerts**: Start with critical alerts (no leader, quorum loss)
4. **Baseline Metrics**: Observe normal behavior for 1 week
5. **Tune Thresholds**: Adjust alerts based on actual usage patterns

---

## See Also

- [E2E Validation Guide](./e2e-validation.md) - Testing procedures
- [Performance Benchmarking Guide](./performance-benchmarking.md) - Benchmarking Warren
- [Troubleshooting Guide](./troubleshooting.md) - Common issues and solutions
- [Warren Architecture](./.agent/System/project-architecture.md) - System design

---

**Questions or Issues?**
Report monitoring issues at: https://github.com/cuemby/warren/issues
