# Warren Testing

This directory contains the Warren testing framework and test suites.

## Test Framework

Warren includes a comprehensive Go-based testing framework for e2e and integration tests. The framework provides:

- **Cluster Management**: Start/stop multi-node Warren clusters using Lima VMs
- **Process Lifecycle**: Manage Warren processes with automatic log capture
- **Rich Assertions**: Type-safe test assertions for services, tasks, nodes, etc.
- **Polling Utilities**: Wait for conditions with timeouts and retries
- **Parallel Execution**: Run tests concurrently for faster feedback

See [framework/README.md](framework/README.md) for detailed documentation.

## E2E Tests

End-to-end tests validate complete workflows across multi-node clusters.

### Running E2E Tests

```bash
# Run all e2e tests
go test -v ./test/e2e/...

# Run specific test
go test -v ./test/e2e -run TestBasicCluster

# Run in short mode (skip long tests)
go test -v -short ./test/e2e/...

# Run with race detector
go test -v -race ./test/e2e/...
```

### Available E2E Tests

- **TestBasicCluster**: Single manager + worker cluster with service operations
- **TestMultiManagerCluster**: 3-manager HA cluster with leader failover

## Integration Tests

Integration tests validate specific Warren features and components.

## Health Check Tests

The health check integration tests validate the end-to-end health monitoring system.

### Prerequisites

1. Build Warren binary:
   ```bash
   make build
   ```

2. Start a Warren cluster:
   ```bash
   ./warren cluster init --node-id test-manager --data-dir /tmp/warren-test
   ```

### Running Tests

Run all health check tests:
```bash
go test -v ./test -run TestHealthCheck
```

Run specific test:
```bash
# Test HTTP health checks
go test -v ./test -run TestHealthCheckHTTP

# Test TCP health checks
go test -v ./test -run TestHealthCheckTCP

# Test unhealthy task replacement
go test -v ./test -run TestHealthCheckUnhealthy
```

### Test Coverage

#### TestHealthCheckHTTP
- Creates service with HTTP health check on nginx
- Validates task starts and runs successfully
- Confirms health checks pass and task remains healthy
- **Duration**: ~30 seconds

#### TestHealthCheckTCP
- Creates service with TCP health check on nginx port 80
- Validates TCP connectivity checks succeed
- Confirms task remains running with passing health checks
- **Duration**: ~30 seconds

#### TestHealthCheckUnhealthy
- Creates service with intentionally failing HTTP health check
- Validates reconciler detects unhealthy task
- Confirms unhealthy task is marked as failed
- Validates scheduler creates replacement task
- **Duration**: ~45 seconds

### Test Architecture

```
┌─────────────────┐
│   Test Suite    │
└────────┬────────┘
         │
         │ 1. Create Service with Health Check
         ▼
┌─────────────────┐
│    Manager      │
└────────┬────────┘
         │
         │ 2. Schedule Task
         ▼
┌─────────────────┐
│     Worker      │──────┐
└────────┬────────┘      │
         │               │ 4. Report Health
         │ 3. Run        │
         │ Health        │
         │ Checks        ▼
         │        ┌─────────────────┐
         │        │  Health Monitor │
         │        └────────┬────────┘
         │                 │
         │                 │ 5. Update HealthStatus
         ▼                 ▼
┌─────────────────────────────┐
│      Reconciler             │
│  (Detects Unhealthy Tasks)  │
└────────┬────────────────────┘
         │
         │ 6. Mark as Failed
         ▼
┌─────────────────┐
│    Scheduler    │
│ (Creates        │
│  Replacement)   │
└─────────────────┘
```

### Cleanup

The tests automatically clean up services after completion. If tests fail and leave resources:

```bash
# List services
./warren service list --manager 127.0.0.1:8080

# Delete specific service
./warren service delete SERVICE_NAME --manager 127.0.0.1:8080
```

### Troubleshooting

**Test hangs waiting for task to start:**
- Check that Warren cluster is running
- Verify containerd is working: `./warren service create test --image nginx --manager 127.0.0.1:8080`
- Check manager logs for errors

**Health checks not reported:**
- Verify worker is connected: `./warren node list --manager 127.0.0.1:8080`
- Check worker logs for health check errors
- Ensure network connectivity between worker and containers

**Unhealthy task not replaced:**
- Check reconciler is running (should log every 10 seconds)
- Verify task is marked as unhealthy: `./warren service inspect SERVICE_NAME`
- Check scheduler logs for replacement task creation
