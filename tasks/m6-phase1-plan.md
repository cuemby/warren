# Milestone 6 - Phase 6.1: Health Checks Implementation Plan

**Phase**: 6.1 - Health Checks
**Priority**: CRITICAL
**Estimated Effort**: 3-5 days
**Status**: IN PROGRESS
**Start Date**: 2025-10-10

---

## Overview

Implement comprehensive health checking for containers to ensure reliability and automatic failure recovery. Health checks are critical for production deployments as they enable:
- Automatic detection of unhealthy containers
- Auto-replacement of failed tasks
- Prevention of traffic to unhealthy replicas
- Graceful startup and shutdown handling

---

## Architecture

### Health Check Types

1. **HTTP Health Check**
   - Send HTTP GET request to container endpoint
   - Check response status code (e.g., 200-299 = healthy)
   - Most common for web services
   - Example: `GET /health` → 200 OK

2. **TCP Health Check**
   - Attempt TCP connection to container port
   - Connection success = healthy
   - Useful for non-HTTP services (databases, caches)
   - Example: Connect to Redis on port 6379

3. **Exec Health Check**
   - Execute command inside container
   - Exit code 0 = healthy, non-zero = unhealthy
   - Most flexible, can check internal state
   - Example: `pg_isready` for PostgreSQL

### Health Check Phases

1. **Startup Probe**
   - Delays other probes until container is ready
   - Protects slow-starting applications
   - Higher failure threshold

2. **Liveness Probe**
   - Determines if container needs restart
   - Detects deadlocks, infinite loops
   - Failed = kill and restart container

3. **Readiness Probe**
   - Determines if container can receive traffic
   - Temporarily unhealthy = remove from load balancing
   - Failed = don't send traffic, but don't restart

---

## Data Structures

### Protobuf Schema (api/proto/warren.proto)

```protobuf
// HealthCheck defines container health monitoring
message HealthCheck {
  // Type of health check
  enum Type {
    HTTP = 0;
    TCP = 1;
    EXEC = 2;
  }

  Type type = 1;

  // HTTP-specific config
  HTTPHealthCheck http = 2;

  // TCP-specific config
  TCPHealthCheck tcp = 3;

  // Exec-specific config
  ExecHealthCheck exec = 4;

  // Common configuration
  int32 interval_seconds = 5;      // Time between checks (default: 30s)
  int32 timeout_seconds = 6;       // Check timeout (default: 10s)
  int32 retries = 7;               // Failures before unhealthy (default: 3)
  int32 start_period_seconds = 8;  // Grace period for startup (default: 0s)
}

message HTTPHealthCheck {
  string path = 1;              // HTTP path (e.g., "/health")
  int32 port = 2;               // Container port
  string scheme = 3;            // "http" or "https"
  repeated Header headers = 4;  // Custom headers
  int32 status_code_min = 5;    // Min acceptable status (default: 200)
  int32 status_code_max = 6;    // Max acceptable status (default: 399)
}

message Header {
  string key = 1;
  string value = 2;
}

message TCPHealthCheck {
  int32 port = 1;  // Container port to check
}

message ExecHealthCheck {
  repeated string command = 1;  // Command to execute (e.g., ["pg_isready"])
}
```

### Go Types (pkg/types/health.go)

```go
package types

import "time"

type HealthCheckType string

const (
	HealthCheckTypeHTTP HealthCheckType = "http"
	HealthCheckTypeTCP  HealthCheckType = "tcp"
	HealthCheckTypeExec HealthCheckType = "exec"
)

type HealthCheckPhase string

const (
	HealthCheckPhaseStartup   HealthCheckPhase = "startup"
	HealthCheckPhaseLiveness  HealthCheckPhase = "liveness"
	HealthCheckPhaseReadiness HealthCheckPhase = "readiness"
)

type HealthCheckResult struct {
	Healthy   bool
	Message   string
	CheckedAt time.Time
}

type HealthCheckStatus struct {
	Phase              HealthCheckPhase
	ConsecutiveFailures int
	ConsecutiveSuccesses int
	LastCheck          time.Time
	LastResult         HealthCheckResult
}
```

---

## Implementation Steps

### Step 1: Create Health Check Package Structure (30 min)
- [ ] Create pkg/health/ directory
- [ ] Create pkg/health/health.go (interfaces and common types)
- [ ] Create pkg/health/http.go (HTTP health checker)
- [ ] Create pkg/health/tcp.go (TCP health checker)
- [ ] Create pkg/health/exec.go (Exec health checker)

### Step 2: Update Protobuf Schema (30 min)
- [ ] Add HealthCheck message types to warren.proto
- [ ] Add health field to ServiceSpec
- [ ] Regenerate protobuf code: `make proto`
- [ ] Update types/service.go with health check fields

### Step 3: Implement HTTP Health Checker (1 hour)
- [ ] HTTPChecker struct with http.Client
- [ ] Check() method implementing probe logic
- [ ] Timeout handling
- [ ] Status code validation
- [ ] Custom headers support
- [ ] Unit tests (healthy, unhealthy, timeout scenarios)

### Step 4: Implement TCP Health Checker (45 min)
- [ ] TCPChecker struct
- [ ] Check() method with net.DialTimeout
- [ ] Connection attempt logic
- [ ] Unit tests (connection success/failure/timeout)

### Step 5: Implement Exec Health Checker (1 hour)
- [ ] ExecChecker struct with containerd client
- [ ] Check() method executing command in container
- [ ] Exit code validation
- [ ] Output capture for debugging
- [ ] Unit tests (mock containerd exec)

### Step 6: Worker Health Monitoring (2 hours)
- [ ] Create pkg/worker/health.go
- [ ] HealthMonitor struct managing active health checks
- [ ] StartMonitoring() - launch goroutines per task
- [ ] StopMonitoring() - graceful shutdown
- [ ] checkLoop() - periodic health check execution
- [ ] Report health status to manager via gRPC
- [ ] Handle startup/liveness/readiness phases

### Step 7: Manager Health Tracking (1.5 hours)
- [ ] Update pkg/types/task.go with HealthStatus field
- [ ] Add ReportTaskHealth() to gRPC API
- [ ] Store health status in BoltDB (tasks bucket)
- [ ] Reconciler queries health status
- [ ] Mark unhealthy tasks as failed

### Step 8: Reconciler Integration (1 hour)
- [ ] Update pkg/reconciler/reconciler.go
- [ ] Check task health status in reconciliation loop
- [ ] Mark tasks unhealthy after N consecutive failures
- [ ] Trigger task replacement for unhealthy tasks
- [ ] Respect grace periods and retries

### Step 9: CLI Support (1 hour)
- [ ] Update cmd/warren/main.go service create flags:
  - --health-http <path:port>
  - --health-tcp <port>
  - --health-cmd <command>
  - --health-interval <seconds>
  - --health-timeout <seconds>
  - --health-retries <count>
  - --health-start-period <seconds>
- [ ] Parse flags and populate ServiceSpec.HealthCheck
- [ ] Update service inspect to show health config

### Step 10: Integration Testing (2 hours)
- [ ] Test: nginx with HTTP health check on /
- [ ] Test: unhealthy container (return 500) gets replaced
- [ ] Test: redis with TCP health check on 6379
- [ ] Test: postgres with exec health check (pg_isready)
- [ ] Test: startup period delays liveness checks
- [ ] Test: readiness probe affects load balancing (future)

---

## Test Scenarios

### Scenario 1: Healthy HTTP Service
```bash
warren service create nginx \
  --image nginx:latest \
  --replicas 3 \
  --health-http path=/,port=80 \
  --health-interval 10 \
  --health-timeout 5 \
  --health-retries 2

# Expected: All 3 replicas healthy, no restarts
warren service inspect nginx
# Health: 3/3 healthy
```

### Scenario 2: Unhealthy HTTP Service
```bash
# Deploy service that returns 500
warren service create broken-app \
  --image broken-app:latest \
  --replicas 2 \
  --health-http path=/health,port=8080

# Expected: After 3 failures (retries), tasks marked unhealthy
# Reconciler replaces unhealthy tasks
warren service inspect broken-app
# Health: 0/2 healthy, 2/2 restarting
```

### Scenario 3: TCP Health Check
```bash
warren service create redis \
  --image redis:latest \
  --replicas 1 \
  --health-tcp port=6379 \
  --health-interval 5

# Expected: TCP connection succeeds, task healthy
```

### Scenario 4: Exec Health Check
```bash
warren service create postgres \
  --image postgres:latest \
  --replicas 1 \
  --health-cmd pg_isready \
  --health-interval 10 \
  --health-start-period 30

# Expected: 30s startup grace period, then liveness checks
```

---

## Success Criteria

- [ ] HTTP health checks detect unhealthy containers
- [ ] TCP health checks validate port connectivity
- [ ] Exec health checks run commands inside containers
- [ ] Unhealthy tasks automatically replaced
- [ ] Startup grace periods respected
- [ ] CLI flags functional
- [ ] Unit test coverage >80%
- [ ] Integration tests pass
- [ ] Documentation updated

---

## Dependencies

**External Libraries**:
- None (use stdlib: net/http, net, os/exec)

**Containerd Integration**:
- Need containerd client for exec health checks
- Use existing pkg/runtime/containerd.go client

**Protobuf**:
- Update api/proto/warren.proto
- Regenerate with `make proto`

---

## Risks & Mitigations

**Risk**: Health checks overwhelming workers with requests
**Mitigation**: Configurable intervals (default 30s), max concurrent checks

**Risk**: False positives from slow startup
**Mitigation**: Startup grace periods, configurable retries

**Risk**: Health check commands consuming resources
**Mitigation**: Timeouts, resource limits on exec commands

**Risk**: Flapping (rapid healthy/unhealthy transitions)
**Mitigation**: Consecutive failure/success counters, debouncing

---

## Next Steps After Phase 6.1

1. Phase 6.2: Published Ports
2. Phase 6.3: DNS Service Discovery
3. Phase 6.4: mTLS Security
4. Phase 6.5: Resource Limits

---

**Last Updated**: 2025-10-10
**Assignee**: Claude (AI Assistant)
**Reviewer**: User
