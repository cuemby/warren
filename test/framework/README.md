# Warren Test Framework

**Status**: Foundation Complete (Week 1) ✅
**Version**: 1.0.0
**Last Updated**: 2025-10-12

## Overview

The Warren test framework provides reusable infrastructure for writing Go-based end-to-end, integration, and chaos tests for Warren. It replaces bash-based testing with type-safe, maintainable Go tests.

## Features

- ✅ **Cluster Management**: Create and manage multi-node test clusters
- ✅ **Multiple Runtimes**: Support for Lima VMs, Docker containers, and local processes
- ✅ **Process Lifecycle**: Start, stop, restart, and monitor Warren processes
- ✅ **Rich Assertions**: Comprehensive test assertion helpers
- ✅ **Polling Utilities**: Wait for conditions with timeouts
- ✅ **Log Capture**: Automatic log capture with searching and filtering
- ✅ **Type Safety**: Compile-time checks prevent test breakage
- ✅ **Parallel Execution**: Tests can run in parallel with `t.Parallel()`

## Architecture

```
test/framework/
├── types.go         # Core type definitions
├── cluster.go       # Cluster management
├── process.go       # Process lifecycle & logging
├── assertions.go    # Test assertions
├── waiters.go       # Polling utilities
├── vm.go           # VM interface (TODO)
├── lima_vm.go      # Lima implementation (TODO)
└── docker_vm.go    # Docker implementation (TODO)
```

## Quick Start

### 1. Basic Cluster Test

```go
package e2e

import (
    "context"
    "testing"
    "time"

    "github.com/cuemby/warren/test/framework"
)

func TestBasicCluster(t *testing.T) {
    // Skip in short mode
    if testing.Short() {
        t.Skip("Skipping e2e test in short mode")
    }

    // Create context
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()

    // Configure cluster
    config := framework.DefaultClusterConfig()
    config.NumManagers = 3
    config.NumWorkers = 2
    config.Runtime = framework.RuntimeLocal

    // Create cluster
    cluster, err := framework.NewCluster(config)
    if err != nil {
        t.Fatalf("Failed to create cluster: %v", err)
    }
    defer cluster.Cleanup()

    // Start cluster
    if err := cluster.Start(); err != nil {
        t.Fatalf("Failed to start cluster: %v", err)
    }

    // Create assertions helper
    assert := framework.NewAssertions(t)
    assert.HasLeader(cluster)
    assert.QuorumSize(3, cluster)

    t.Log("✓ Cluster started successfully")
}
```

### 2. Service Lifecycle Test

```go
func TestServiceLifecycle(t *testing.T) {
    // Setup cluster (omitted for brevity)
    cluster, _ := setupTestCluster(t)
    defer cluster.Cleanup()

    // Get leader
    leader, err := cluster.GetLeader()
    if err != nil {
        t.Fatalf("No leader: %v", err)
    }

    // Create assertions and waiter
    assert := framework.NewAssertions(t)
    waiter := framework.DefaultWaiter()

    // Create service
    svc, err := leader.Client.CreateService("nginx", "nginx:alpine", 3, nil)
    assert.NoError(err, "Failed to create service")

    // Wait for replicas to be running
    ctx := context.Background()
    err = waiter.WaitForReplicas(ctx, leader.Client, "nginx", 3)
    assert.NoError(err, "Failed to wait for replicas")

    // Verify service is running
    assert.ServiceRunning("nginx", leader.Client)
    assert.ServiceReplicas("nginx", 3, leader.Client)

    // Delete service
    err = leader.Client.DeleteService(svc.Id)
    assert.NoError(err, "Failed to delete service")

    // Wait for deletion
    err = waiter.WaitForServiceDeleted(ctx, leader.Client, "nginx")
    assert.NoError(err, "Service not deleted")

    t.Log("✓ Service lifecycle test passed")
}
```

### 3. Failover Test

```go
func TestLeaderFailover(t *testing.T) {
    cluster, _ := setupTestCluster(t)
    defer cluster.Cleanup()

    assert := framework.NewAssertions(t)
    waiter := framework.DefaultWaiter()

    // Get current leader
    leader, err := cluster.GetLeader()
    assert.NoError(err, "Failed to get leader")

    oldLeaderID := leader.ID
    t.Logf("Current leader: %s", oldLeaderID)

    // Kill the leader
    err = cluster.KillManager(oldLeaderID)
    assert.NoError(err, "Failed to kill leader")

    // Wait for new leader election
    ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
    defer cancel()

    err = waiter.WaitForLeaderElection(ctx, cluster)
    assert.NoError(err, "New leader not elected")

    // Verify new leader
    newLeader, err := cluster.GetLeader()
    assert.NoError(err, "Failed to get new leader")
    assert.NotEqual(oldLeaderID, newLeader.ID, "Leader should have changed")

    t.Logf("✓ Leader failover: %s → %s", oldLeaderID, newLeader.ID)
}
```

## API Reference

### Cluster Management

#### Creating a Cluster

```go
// Default configuration (3 managers, 2 workers, Lima runtime)
config := framework.DefaultClusterConfig()

// Custom configuration
config := &framework.ClusterConfig{
    NumManagers:   3,
    NumWorkers:    2,
    Runtime:       framework.RuntimeLocal,
    DataDir:       "/tmp/warren-test",
    WarrenBinary:  "bin/warren",
    KeepOnFailure: false,  // Keep cluster running if test fails (for debugging)
    LogLevel:      "info",
}

cluster, err := framework.NewCluster(config)
```

#### Cluster Operations

```go
// Start entire cluster
err := cluster.Start()

// Stop cluster gracefully
err := cluster.Stop()

// Clean up all resources
err := cluster.Cleanup()

// Get current leader
leader, err := cluster.GetLeader()

// Wait for quorum
err := cluster.WaitForQuorum()

// Wait for workers to connect
err := cluster.WaitForWorkers()

// Kill a manager (simulate crash)
err := cluster.KillManager("manager-1")

// Restart a manager
err := cluster.RestartManager("manager-1")
```

### Process Management

```go
// Create process
process := framework.NewProcess("bin/warren")
process.Args = []string{"cluster", "init", "--node-id=test"}
process.Env = []string{"LOG_LEVEL=debug"}
process.LogFile = "/tmp/warren.log"

// Start process
err := process.Start()

// Stop gracefully (SIGTERM)
err := process.Stop()

// Kill forcefully (SIGKILL)
err := process.Kill()

// Restart
err := process.Restart()

// Check if running
isRunning := process.IsRunning()

// Get logs
logs := process.Logs()

// Wait for specific log line
err := process.WaitForLog("cluster started", 30*time.Second)
```

### Assertions

```go
assert := framework.NewAssertions(t)

// Service assertions
assert.ServiceExists("nginx", client)
assert.ServiceRunning("nginx", client)
assert.ServiceReplicas("nginx", 3, client)
assert.ServiceDeleted("nginx", client)

// Task assertions
assert.TaskRunning("task-123", client)
assert.TaskHealthy("task-123", client)

// Cluster assertions
assert.HasLeader(cluster)
assert.QuorumSize(3, cluster)
assert.NodeCount(5, client)

// Generic assertions
assert.NoError(err, "Operation failed")
assert.Error(err, "Expected error")
assert.Equal(3, count, "Replica count mismatch")
assert.True(condition, "Condition should be true")
assert.Contains(logs, "started", "Logs should contain 'started'")

// Eventually (polling assertion)
assert.Eventually(func() bool {
    svc, _ := client.GetService("nginx")
    return svc.Status == "running"
}, 30*time.Second, 1*time.Second, "service should be running")

// Logging (non-failing)
assert.Step("Creating service")
assert.Info("Starting manager-1")
assert.Success("Cluster started")
assert.Warning("High memory usage detected")
```

### Waiters

```go
waiter := framework.DefaultWaiter()  // 30s timeout, 1s interval

// Custom waiter
waiter := framework.NewWaiter(60*time.Second, 2*time.Second)

// Wait for service
ctx := context.Background()
err := waiter.WaitForServiceRunning(ctx, client, "nginx")
err := waiter.WaitForServiceDeleted(ctx, client, "nginx")

// Wait for replicas
err := waiter.WaitForReplicas(ctx, client, "nginx", 3)

// Wait for task
err := waiter.WaitForTaskRunning(ctx, client, "task-123")
err := waiter.WaitForTaskHealthy(ctx, client, "task-123")

// Wait for cluster state
err := waiter.WaitForLeaderElection(ctx, cluster)
err := waiter.WaitForQuorum(ctx, cluster)
err := waiter.WaitForNodeCount(ctx, client, 5)
err := waiter.WaitForWorkerNodes(ctx, client, 2)
err := waiter.WaitForClusterHealthy(ctx, client)

// Wait for secrets/volumes
err := waiter.WaitForSecret(ctx, client, "my-secret")
err := waiter.WaitForVolume(ctx, client, "my-volume")

// Generic wait with condition
err := waiter.WaitFor(ctx, func() bool {
    // Custom condition
    return someCondition()
}, "custom condition description")

// Retry with exponential backoff
err := framework.Retry(ctx, 5, time.Second, func() error {
    return someOperation()
})
```

## Test Patterns

### Table-Driven Tests

```go
func TestServiceCreation(t *testing.T) {
    cluster, _ := setupTestCluster(t)
    defer cluster.Cleanup()

    leader, _ := cluster.GetLeader()

    tests := []struct {
        name     string
        image    string
        replicas int32
        wantErr  bool
    }{
        {"nginx", "nginx:alpine", 1, false},
        {"redis", "redis:alpine", 3, false},
        {"invalid", "nonexistent:image", 1, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            _, err := leader.Client.CreateService(tt.name, tt.image, tt.replicas, nil)
            if (err != nil) != tt.wantErr {
                t.Errorf("CreateService() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### Parallel Tests

```go
func TestParallelServices(t *testing.T) {
    cluster, _ := setupTestCluster(t)
    defer cluster.Cleanup()

    services := []string{"nginx", "redis", "postgres"}

    for _, svc := range services {
        svc := svc  // Capture range variable
        t.Run(svc, func(t *testing.T) {
            t.Parallel()  // Run in parallel

            leader, _ := cluster.GetLeader()
            // Create and test service
        })
    }
}
```

### Subtests

```go
func TestServiceLifecycle(t *testing.T) {
    cluster, _ := setupTestCluster(t)
    defer cluster.Cleanup()

    leader, _ := cluster.GetLeader()
    assert := framework.NewAssertions(t)

    t.Run("create", func(t *testing.T) {
        _, err := leader.Client.CreateService("nginx", "nginx:alpine", 3, nil)
        assert.NoError(err, "Failed to create service")
    })

    t.Run("scale", func(t *testing.T) {
        _, err := leader.Client.UpdateService("nginx", 5)
        assert.NoError(err, "Failed to scale service")
    })

    t.Run("delete", func(t *testing.T) {
        err := leader.Client.DeleteService("nginx")
        assert.NoError(err, "Failed to delete service")
    })
}
```

## Configuration

### Environment Variables

```bash
# Warren binary path
export WARREN_BINARY="bin/warren"

# Test data directory
export WARREN_TEST_DATA_DIR="/tmp/warren-test"

# Runtime type
export WARREN_TEST_RUNTIME="local"  # local, lima, or docker
```

### Test Flags

```bash
# Run all tests
go test ./test/e2e/...

# Run specific test
go test -run TestClusterFormation ./test/e2e/...

# Skip long-running tests
go test -short ./test/e2e/...

# Enable verbose output
go test -v ./test/e2e/...

# Run with race detector
go test -race ./test/e2e/...

# Set timeout
go test -timeout 10m ./test/e2e/...

# Run in parallel
go test -parallel 4 ./test/e2e/...
```

## Best Practices

### 1. Always Use defer for Cleanup

```go
cluster, err := framework.NewCluster(config)
if err != nil {
    t.Fatalf("Failed to create cluster: %v", err)
}
defer cluster.Cleanup()  // Ensure cleanup even if test fails
```

### 2. Use Context with Timeout

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()

err := waiter.WaitForServiceRunning(ctx, client, "nginx")
```

### 3. Skip Long Tests in Short Mode

```go
if testing.Short() {
    t.Skip("Skipping e2e test in short mode")
}
```

### 4. Use t.Helper() in Helper Functions

```go
func setupTestCluster(t *testing.T) *framework.Cluster {
    t.Helper()  // Mark as helper for better error reporting

    cluster, err := framework.NewCluster(framework.DefaultClusterConfig())
    if err != nil {
        t.Fatalf("Failed to create cluster: %v", err)
    }

    return cluster
}
```

### 5. Log Test Steps

```go
assert := framework.NewAssertions(t)
assert.Step("Step 1: Creating cluster")
// ...
assert.Step("Step 2: Starting services")
// ...
assert.Success("Test completed successfully")
```

### 6. Keep Tests Fast

- Use `t.Parallel()` where possible
- Use smaller clusters for unit-like tests (1 manager, 1 worker)
- Use timeouts to fail fast
- Consider using local runtime over Lima/Docker for speed

### 7. Test in CI

```yaml
# .github/workflows/e2e.yml
name: E2E Tests

on: [push, pull_request]

jobs:
  e2e:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Build Warren
        run: make build

      - name: Run E2E Tests
        run: go test -v -timeout 30m ./test/e2e/...
```

## Troubleshooting

### Test Hangs

- Check timeouts in Context and Waiter
- Use `t.Log()` to add visibility
- Run with `-v` flag for verbose output
- Check process logs: `cluster.Managers[0].Process.Logs()`

### Port Conflicts

- Ensure no other Warren instances running
- Use unique ports for each test
- Clean up previous test resources

### Permission Errors

- Check file permissions on data directories
- Ensure Warren binary is executable
- Run with appropriate privileges if needed

### Cleanup Failures

- Check `KeepOnFailure` option for debugging
- Manually clean up: `rm -rf /tmp/warren-test*`
- Kill lingering processes: `pkill warren`

## Future Enhancements

- [ ] Lima VM implementation (vm.go, lima_vm.go)
- [ ] Docker container implementation (docker_vm.go)
- [ ] Chaos testing utilities (chaos/)
- [ ] Benchmark helpers (benchmark/)
- [ ] Test fixtures and factories
- [ ] Network utilities (port allocation, TCP checks)
- [ ] File utilities (temp dirs, file ops)
- [ ] Enhanced logging (structured, colored)

## Contributing

When adding new framework features:

1. Add tests for framework code (framework/*_test.go)
2. Update this README with examples
3. Follow existing patterns and conventions
4. Ensure backward compatibility
5. Document any breaking changes

## Resources

- **Main Repo**: [github.com/cuemby/warren](https://github.com/cuemby/warren)
- **Test Migration Plan**: `tasks/todo.md` (Phase 1 Stabilization)
- **Example Tests**: `test/e2e/` (coming soon)
- **Integration Tests**: `test/integration/` (existing Go tests)

---

**Maintained by**: Warren Test Infrastructure Team
**Version**: 1.0.0 (Foundation - Week 1 Complete)
**Last Updated**: 2025-10-12
