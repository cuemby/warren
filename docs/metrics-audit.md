# Metrics Audit - Phase 1 Week 2

**Date**: 2025-10-13
**Status**: Audit Complete
**Goal**: Identify gaps and enhance Prometheus metrics coverage

---

## Executive Summary

Warren has a **solid metrics foundation** with 30+ metrics defined in `pkg/metrics/metrics.go`. However, **metric instrumentation is sparse** - only 2 out of 10+ critical components actively emit metrics.

**Current Coverage**:
- ✅ API Server: Service create/update/delete duration
- ✅ Reconciler: Reconciliation duration and cycle count
- ❌ Scheduler: No instrumentation
- ❌ Manager/Raft: No instrumentation
- ❌ Worker: No instrumentation
- ❌ Runtime (containerd): No instrumentation
- ❌ DNS: No instrumentation
- ❌ Ingress: No instrumentation

**Recommendation**: Instrument scheduler, manager, and worker components as high priority.

---

## Existing Metrics Inventory

### Cluster State Metrics (Gauges)
```
warren_nodes_total{role, status}              - Node count by role/status
warren_services_total                          - Total services
warren_containers_total{state}                 - Container count by state
warren_secrets_total                           - Total secrets
warren_volumes_total                           - Total volumes
```

**Status**: ✅ Defined, ❌ Not instrumented
**Location**: pkg/metrics/metrics.go:13-48

### Raft Metrics (Gauges)
```
warren_raft_is_leader                          - Leader status (1/0)
warren_raft_peers_total                        - Raft peer count
warren_raft_log_index                          - Current log index
warren_raft_applied_index                      - Last applied index
```

**Status**: ✅ Defined, ❌ Not instrumented
**Location**: pkg/metrics/metrics.go:50-77

### API Metrics (Counter + Histogram)
```
warren_api_requests_total{method, status}     - Request count
warren_api_request_duration_seconds{method}    - Request latency
```

**Status**: ✅ Defined, ❌ Not fully instrumented
**Location**: pkg/metrics/metrics.go:79-95

### Scheduler Metrics (Histogram + Counters)
```
warren_scheduling_latency_seconds              - Scheduling time
warren_containers_scheduled_total              - Successful schedules
warren_containers_failed_total                 - Failed schedules
```

**Status**: ✅ Defined, ❌ NOT instrumented
**Location**: pkg/metrics/metrics.go:97-118

### Service Operation Metrics (Histograms)
```
warren_service_create_duration_seconds         - Service creation time
warren_service_update_duration_seconds         - Service update time
warren_service_delete_duration_seconds         - Service deletion time
```

**Status**: ✅ Defined, ✅ INSTRUMENTED (pkg/api/server.go:250,336,377)
**Location**: pkg/metrics/metrics.go:120-143

### Container Operation Metrics (Histograms)
```
warren_container_create_duration_seconds       - Container creation time
warren_container_start_duration_seconds        - Container start time
warren_container_stop_duration_seconds         - Container stop time
```

**Status**: ✅ Defined, ❌ Not instrumented
**Location**: pkg/metrics/metrics.go:145-168

### Raft Operation Metrics (Histograms)
```
warren_raft_apply_duration_seconds             - Log apply time
warren_raft_commit_duration_seconds            - Log commit time
```

**Status**: ✅ Defined, ❌ Not instrumented
**Location**: pkg/metrics/metrics.go:170-185

### Reconciler Metrics (Histogram + Counter)
```
warren_reconciliation_duration_seconds         - Reconciliation cycle time
warren_reconciliation_cycles_total             - Total cycles completed
```

**Status**: ✅ Defined, ✅ INSTRUMENTED (pkg/reconciler/reconciler.go:59-62)
**Location**: pkg/metrics/metrics.go:187-201

### Ingress Metrics (Counter + Histogram)
```
warren_ingress_create_duration_seconds         - Ingress creation time
warren_ingress_update_duration_seconds         - Ingress update time
warren_ingress_requests_total{host, backend}   - Request count
warren_ingress_request_duration_seconds{host, backend} - Request latency
```

**Status**: ✅ Defined, ❌ Not instrumented
**Location**: pkg/metrics/metrics.go:203-235

---

## Current Instrumentation Analysis

### ✅ Instrumented Components

#### 1. API Server (pkg/api/server.go)
**Lines**: 250, 336, 377
**Metrics Used**:
- ServiceCreateDuration
- ServiceUpdateDuration
- ServiceDeleteDuration

**Coverage**: Service operations only (3/37 API methods)

#### 2. Reconciler (pkg/reconciler/reconciler.go)
**Lines**: 59-62
**Metrics Used**:
- ReconciliationDuration
- ReconciliationCyclesTotal

**Coverage**: Basic timing + count

---

### ❌ Missing Instrumentation

#### 1. Scheduler (pkg/scheduler/scheduler.go)
**Priority**: HIGH
**Metrics Available**:
- SchedulingLatency
- ContainersScheduled
- ContainersFailed

**Missing Instrumentation**:
- [ ] Track scheduling latency per container
- [ ] Increment ContainersScheduled on success
- [ ] Increment ContainersFailed on failure
- [ ] Add scheduling reason labels (resource constraints, affinity, etc.)

**Estimated Effort**: 30-45 minutes

#### 2. Manager/Raft (pkg/manager/*.go)
**Priority**: HIGH
**Metrics Available**:
- RaftLeader
- RaftPeers
- RaftLogIndex
- RaftAppliedIndex
- RaftApplyDuration
- RaftCommitDuration

**Missing Instrumentation**:
- [ ] Update Raft state gauges (leader, peers, log index)
- [ ] Track Raft operation latencies
- [ ] Monitor leadership changes
- [ ] Track snapshot operations

**Estimated Effort**: 1-2 hours

#### 3. Worker/Runtime (pkg/worker/*.go, pkg/runtime/*.go)
**Priority**: MEDIUM
**Metrics Available**:
- ContainerCreateDuration
- ContainerStartDuration
- ContainerStopDuration

**Missing Instrumentation**:
- [ ] Track container lifecycle operations
- [ ] Monitor containerd interactions
- [ ] Track health check execution
- [ ] Volume mount operations

**Estimated Effort**: 1-2 hours

#### 4. DNS (pkg/dns/*.go)
**Priority**: LOW
**Metrics Needed** (not yet defined):
- DNS query count by type
- DNS query latency
- DNS resolution failures

**Missing Instrumentation**:
- [ ] Define DNS metrics
- [ ] Instrument resolver
- [ ] Track query patterns

**Estimated Effort**: 1 hour

#### 5. Ingress (pkg/ingress/*.go)
**Priority**: LOW
**Metrics Available**:
- IngressCreateDuration
- IngressUpdateDuration
- IngressRequestsTotal
- IngressRequestDuration

**Missing Instrumentation**:
- [ ] Track ingress CRUD operations
- [ ] Monitor proxy request metrics
- [ ] Track backend health
- [ ] Monitor ACME certificate operations

**Estimated Effort**: 1-2 hours

---

## Metrics Collection Status

Warren has a **metrics collector** (pkg/metrics/collector.go) that periodically updates cluster state metrics. Let me verify its implementation...

**Status**: To be investigated

---

## Recommendations

### Phase 1 Week 2 (This Week)

#### Priority 1: Scheduler Instrumentation ⭐
**Impact**: HIGH
**Effort**: 30-45 minutes

Add metrics to scheduler to track:
- Scheduling latency per container
- Success/failure counts
- Scheduling decisions (node selection)

#### Priority 2: Manager/Raft Instrumentation ⭐
**Impact**: HIGH
**Effort**: 1-2 hours

Add metrics to manager to track:
- Raft state (leader, peers, log indices)
- Raft operation latencies
- Leadership changes

#### Priority 3: Metrics Collector Verification
**Impact**: MEDIUM
**Effort**: 30 minutes

Verify that metrics collector is:
- Running periodically
- Updating cluster state metrics
- Handling errors properly

#### Priority 4: Health Check Endpoints
**Impact**: HIGH
**Effort**: 1 hour

Add `/health` and `/ready` endpoints to API server:
- `/health`: Basic liveness check
- `/ready`: Readiness check (Raft status, storage connectivity)

### Future Work (Phase 1 Week 3+)

- Worker/Runtime instrumentation
- DNS metrics definition and instrumentation
- Ingress proxy metrics
- Custom buckets for histograms (optimize for Warren's performance characteristics)

---

## Testing Plan

1. **Unit Tests**: Verify metrics are incremented
2. **Integration Tests**: Validate metrics via /metrics endpoint
3. **Manual Validation**:
   - Start Warren cluster
   - Perform operations
   - Query Prometheus metrics
   - Verify counters/histograms update correctly

---

## Expected Outcomes

After Phase 1 Week 2:
- ✅ Scheduler fully instrumented
- ✅ Manager/Raft state metrics updated
- ✅ Health check endpoints available
- ✅ Metrics collector verified
- ✅ Documentation for operators

**Coverage Increase**: 2/10 → 5/10 critical components instrumented (150% increase)

---

## Notes

- All metrics are already registered in init() function
- Timer helper makes instrumentation straightforward
- No breaking changes required
- Backward compatible (metrics are additive)

**Next Step**: Begin scheduler instrumentation
