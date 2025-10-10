# Warren Load Testing Guide

This guide explains how to run load tests against Warren clusters to validate performance, scalability, and stability.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Test Scales](#test-scales)
- [Running Load Tests](#running-load-tests)
- [Interpreting Results](#interpreting-results)
- [Performance Targets](#performance-targets)
- [Advanced Testing](#advanced-testing)
- [Troubleshooting](#troubleshooting)

---

## Overview

Warren includes load testing infrastructure to validate:

- **API throughput and latency** - How fast can Warren handle service operations
- **Scheduler performance** - How quickly can tasks be scheduled and placed
- **Memory usage under load** - Does memory stay within targets (Manager < 256MB, Worker < 128MB)
- **Cluster stability** - Does the cluster remain stable under load
- **Scale limits** - How many services/tasks can a cluster handle

Load tests use Lima VMs to create realistic multi-node clusters and deploy hundreds or thousands of services.

---

## Prerequisites

### System Requirements

**For Small Scale Tests (50 services, 150 tasks):**
- 3 VMs (1 manager + 2 workers)
- ~6GB RAM total
- ~10GB disk space

**For Medium Scale Tests (200 services, 600 tasks):**
- 8 VMs (3 managers + 5 workers)
- ~16GB RAM total
- ~20GB disk space

**For Large Scale Tests (1000 services, 3000 tasks):**
- 13 VMs (3 managers + 10 workers)
- ~26GB RAM total
- ~40GB disk space

**Note:** For true 100-node tests (10,000 tasks), you'll need cloud infrastructure or a large bare-metal cluster.

### Software Requirements

- **Lima** - VM management (`brew install lima` on macOS)
- **Warren binary** - Built from source (`make build`)
- **Go tools** - For profiling (`go tool pprof`)
- **bc** - For statistics calculations (usually pre-installed)

---

## Quick Start

### 1. Set Up Test Environment

```bash
# Create test VMs (3 managers + 2 workers by default)
./test/lima/setup.sh

# Or create more workers for larger tests
./test/lima/setup.sh --managers 3 --workers 10
```

### 2. Start Warren Cluster

```bash
# Start manager on VM 1 (with profiling for load test)
limactl shell warren-manager-1
sudo /tmp/lima/warren/bin/warren cluster init --enable-pprof

# In new terminal, start workers
limactl shell warren-worker-1
sudo /tmp/lima/warren/bin/warren worker start \
  --manager lima-warren-manager-1.internal:8080 \
  --enable-pprof

# Repeat for additional workers (warren-worker-2, etc.)
```

### 3. Run Load Test

```bash
# Run small load test (50 services)
./test/lima/test-load.sh --scale small

# Run with profiling enabled
./test/lima/test-load.sh --scale small --profile
```

---

## Test Scales

The load testing script supports three predefined scales:

### Small Scale (Development Testing)

```bash
./test/lima/test-load.sh --scale small
```

**Configuration:**
- Services: 50
- Replicas per service: 3
- Total tasks: 150
- Recommended cluster: 1 manager, 2 workers
- Duration: ~2-3 minutes
- Memory target: Manager < 256MB

**Use for:**
- Development and debugging
- Quick smoke tests
- CI/CD validation

### Medium Scale (Integration Testing)

```bash
./test/lima/test-load.sh --scale medium
```

**Configuration:**
- Services: 200
- Replicas per service: 3
- Total tasks: 600
- Recommended cluster: 3 managers, 5 workers
- Duration: ~5-10 minutes
- Memory target: Manager < 256MB

**Use for:**
- Integration testing
- Performance validation
- Pre-release testing

### Large Scale (Stress Testing)

```bash
./test/lima/test-load.sh --scale large
```

**Configuration:**
- Services: 1000
- Replicas per service: 3
- Total tasks: 3000
- Recommended cluster: 3 managers, 10+ workers
- Duration: ~15-30 minutes
- Memory target: Manager < 512MB (under high load)

**Use for:**
- Stress testing
- Capacity planning
- Performance optimization
- Finding scale limits

### Custom Scale

```bash
# Custom number of services
./test/lima/test-load.sh --services 500 --replicas 5

# This creates 500 services with 5 replicas each = 2500 tasks
```

---

## Running Load Tests

### Basic Load Test

```bash
# Run small scale test
./test/lima/test-load.sh --scale small
```

**What it does:**
1. Checks that manager and workers are running
2. Creates N services with M replicas each
3. Waits for scheduler to process tasks
4. Measures API latency (100 requests)
5. Reports memory usage and performance
6. Cleans up test services

### Load Test with Profiling

```bash
# Enable profiling to capture memory/CPU profiles
./test/lima/test-load.sh --scale medium --profile
```

**Captures:**
- Manager heap profile
- Manager CPU profile (30s sample)
- Worker heap profile (if profiling enabled)

Profiles saved to: `test/load-profiles-YYYYMMDD-HHMMSS/`

Analyze with:
```bash
go tool pprof test/load-profiles-*/manager_heap.prof
```

### Sustained Load Test

For longer-running tests to check for memory leaks:

```bash
# Create services but don't clean up immediately
./test/lima/test-load.sh --scale large --profile

# Let cluster run for 1 hour
sleep 3600

# Check memory usage
limactl shell warren-manager-1 ps aux | grep warren

# Capture another profile to compare
limactl shell warren-manager-1 curl http://127.0.0.1:9090/debug/pprof/heap > heap_after_1h.prof

# Compare profiles
go tool pprof -base heap_initial.prof heap_after_1h.prof
```

---

## Interpreting Results

### Sample Output

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Warren Load Test Results
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Test Configuration:
  Scale: medium
  Services Created: 200
  Replicas per Service: 3
  Total Tasks: 600
  Profiling: true

Results:
  Total Duration: 127s
  Service Creation Rate: 1.57 services/s
  Failed Creations: 0

Memory Usage:
  Manager: 184MB
✓ Manager memory within target

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

### Key Metrics

#### Service Creation Rate

**Target:** > 1 service/s for small clusters, > 0.5 service/s for large

Measures how fast Warren can process service creation requests through the API and Raft consensus.

**If below target:**
- Check API latency
- Profile manager CPU usage
- Check Raft commit latency
- Verify network between manager nodes

#### API Latency

**Target:** < 100ms average for service list operations

Measures responsiveness of the gRPC API under load.

**If above target:**
- Profile manager CPU (likely scheduler or reconciler overhead)
- Check if database reads are slow (BoltDB)
- Verify network latency to manager

#### Manager Memory

**Target:** < 256MB for normal load, < 512MB for heavy load

Measures manager resident memory usage.

**If above target:**
- Capture heap profile: `go tool pprof http://127.0.0.1:9090/debug/pprof/heap`
- Look for memory leaks (continuously growing allocations)
- Check for excessive caching or buffering
- Profile allocations: `go tool pprof -alloc_space heap.prof`

#### Worker Memory

**Target:** < 128MB for normal load (excluding container memory)

Measures worker resident memory usage (not including containers).

**If above target:**
- Capture heap profile: `go tool pprof http://127.0.0.1:6060/debug/pprof/heap`
- Check for container status polling overhead
- Verify containerd client isn't leaking connections

#### Failed Creations

**Target:** 0 failures

Any failed service creations indicate issues.

**If failures occur:**
- Check manager logs for errors
- Verify Raft consensus is healthy
- Check disk space (BoltDB writes)
- Verify API authentication/authorization

---

## Performance Targets

Warren's design targets (from PRD):

| Metric | Target | Acceptable | Critical |
|--------|--------|------------|----------|
| Manager Memory | < 256MB | < 512MB | > 512MB |
| Worker Memory | < 128MB | < 256MB | > 256MB |
| API Latency (p50) | < 50ms | < 100ms | > 200ms |
| API Latency (p99) | < 200ms | < 500ms | > 1s |
| Service Creation | > 1/s | > 0.5/s | < 0.5/s |
| Scheduler Latency | < 5s | < 10s | > 30s |
| Cluster Size | 100 nodes | 50 nodes | < 10 nodes |
| Services per Cluster | 1000+ | 500+ | < 100 |
| Tasks per Cluster | 10,000+ | 5,000+ | < 1,000 |

**Color Coding:**
- **Target** (Green): Warren should achieve this under normal conditions
- **Acceptable** (Yellow): May occur under heavy load, investigate if persistent
- **Critical** (Red): Unacceptable, requires optimization

---

## Advanced Testing

### Test with Real Containers

By default, Warren may simulate container execution. For realistic load tests:

```bash
# Deploy real nginx containers
./test/lima/test-load.sh --scale medium --profile

# Services will be created with nginx:latest image
# Workers will pull and run real containers via containerd
```

**Resource requirements increase:**
- Each nginx container: ~2-5MB memory
- 600 tasks (medium scale) = ~1.2-3GB container memory
- Ensure workers have sufficient memory

### Test Scheduler Performance

```bash
# Rapid service creation to stress scheduler
for i in {1..100}; do
  limactl shell warren-manager-1 sudo /tmp/lima/warren/bin/warren service create \
    rapid-test-$i --image nginx --replicas 5 --manager localhost:8080 &
done
wait

# Check scheduler handled the burst
limactl shell warren-manager-1 sudo /tmp/lima/warren/bin/warren service list
```

### Test Raft Consensus Under Load

```bash
# Create services while killing/restarting managers
./test/lima/test-load.sh --scale large --profile &
LOAD_PID=$!

# After 30s, kill leader
sleep 30
LEADER=$(limactl shell warren-manager-1 sudo /tmp/lima/warren/bin/warren cluster info | grep Leader | awk '{print $2}')
limactl shell "$LEADER" sudo pkill warren

# Cluster should re-elect and continue
# Wait for load test to complete
wait $LOAD_PID
```

### Test Worker Failures

```bash
# Start load test
./test/lima/test-load.sh --scale medium &
LOAD_PID=$!

# Kill worker after 1 minute
sleep 60
limactl shell warren-worker-1 sudo pkill warren

# Tasks should be rescheduled to other workers
# Check reconciler behavior
wait $LOAD_PID
```

### Memory Leak Detection

```bash
# Capture baseline
limactl shell warren-manager-1 curl http://127.0.0.1:9090/debug/pprof/heap > baseline.prof

# Run load test
./test/lima/test-load.sh --scale large

# Let cluster run for 30 minutes
sleep 1800

# Capture second profile
limactl shell warren-manager-1 curl http://127.0.0.1:9090/debug/pprof/heap > after_30min.prof

# Compare - look for growth
go tool pprof -base baseline.prof after_30min.prof
```

In pprof:
```
(pprof) top
# Should see what's growing
(pprof) list <function>
# Examine source
```

### API Throughput Test

```bash
# Measure max API throughput
ab -n 1000 -c 10 http://127.0.0.1:8080/service/list

# Or with parallel clients
for i in {1..10}; do
  (for j in {1..100}; do
    limactl shell warren-manager-1 sudo /tmp/lima/warren/bin/warren service list &> /dev/null
  done) &
done
wait
```

---

## Troubleshooting

### Issue: "Manager VM not running"

**Solution:**
```bash
# Check VM status
limactl list

# If VMs don't exist, create them
./test/lima/setup.sh

# If VMs stopped, start them
limactl start warren-manager-1
```

### Issue: "Warren manager process not running"

**Solution:**
```bash
# Start manager
limactl shell warren-manager-1
sudo /tmp/lima/warren/bin/warren cluster init --enable-pprof
```

### Issue: "No workers registered"

**Solution:**
```bash
# Start workers
limactl shell warren-worker-1
sudo /tmp/lima/warren/bin/warren worker start \
  --manager lima-warren-manager-1.internal:8080
```

### Issue: Load test hangs during service creation

**Possible causes:**
- Manager overloaded (CPU at 100%)
- Raft not committing (check logs)
- Network issues between VMs
- Disk full (check /tmp and data directories)

**Debug:**
```bash
# Check manager CPU
limactl shell warren-manager-1 top

# Check manager logs
limactl shell warren-manager-1 sudo journalctl -fu warren

# Check disk space
limactl shell warren-manager-1 df -h

# Check Raft status
limactl shell warren-manager-1 sudo /tmp/lima/warren/bin/warren cluster info
```

### Issue: High memory usage during test

**Expected behavior:**
- Memory should stabilize after initial service creation
- Minor growth is normal (caching, buffers)
- Memory should not continuously grow

**If memory keeps growing:**
```bash
# Capture heap profile
limactl shell warren-manager-1 curl http://127.0.0.1:9090/debug/pprof/heap > leak.prof

# Analyze
go tool pprof leak.prof
(pprof) top
(pprof) list <function>
```

### Issue: Services created but tasks not scheduled

**Possible causes:**
- No workers available (all down or not registered)
- Workers at capacity (resources exhausted)
- Scheduler not running

**Debug:**
```bash
# Check nodes
limactl shell warren-manager-1 sudo /tmp/lima/warren/bin/warren node list

# Check scheduler logs
limactl shell warren-manager-1 sudo journalctl -fu warren | grep -i scheduler

# Check worker resources
limactl shell warren-worker-1 sudo /tmp/lima/warren/bin/warren node inspect worker-1
```

### Issue: Failed to capture profiles

**Solution:**
```bash
# Ensure profiling is enabled
limactl shell warren-manager-1 sudo /tmp/lima/warren/bin/warren cluster init --enable-pprof

# Check profiling endpoint is accessible
limactl shell warren-manager-1 curl http://127.0.0.1:9090/debug/pprof/

# Should return HTML page with profile links
```

---

## Best Practices

### Before Running Load Tests

1. **Build latest binary** - `make build` to ensure you're testing current code
2. **Clean environment** - `./test/lima/cleanup.sh && ./test/lima/setup.sh` for fresh start
3. **Enable profiling** - Always use `--enable-pprof` when starting manager/workers
4. **Baseline profiles** - Capture profiles before load test for comparison
5. **Monitor externally** - Keep `htop` or `top` running to watch resource usage

### During Load Tests

1. **Don't interfere** - Let test run to completion without manual operations
2. **Watch for errors** - Monitor manager/worker logs for errors or warnings
3. **Check stability** - Ensure processes don't crash or restart
4. **Note anomalies** - Document any unexpected behavior for investigation

### After Load Tests

1. **Capture final profiles** - Get heap and CPU profiles after load
2. **Check for leaks** - Compare initial and final memory usage
3. **Save results** - Keep test output and profiles for trending analysis
4. **Clean up** - Delete test services to restore cluster to clean state
5. **Document findings** - Note performance metrics and any issues found

---

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Load Test

on:
  schedule:
    - cron: '0 2 * * *'  # Daily at 2am
  workflow_dispatch:

jobs:
  load-test:
    runs-on: macos-latest
    steps:
      - uses: actions/checkout@v3

      - name: Install Lima
        run: brew install lima

      - name: Build Warren
        run: make build

      - name: Setup test environment
        run: ./test/lima/setup.sh --managers 1 --workers 2

      - name: Start cluster
        run: |
          limactl shell warren-manager-1 sudo /tmp/lima/warren/bin/warren cluster init --enable-pprof &
          sleep 10
          limactl shell warren-worker-1 sudo /tmp/lima/warren/bin/warren worker start --manager lima-warren-manager-1.internal:8080 &
          sleep 5

      - name: Run load test
        run: ./test/lima/test-load.sh --scale small --profile

      - name: Upload profiles
        uses: actions/upload-artifact@v3
        with:
          name: load-test-profiles
          path: test/load-profiles-*

      - name: Cleanup
        if: always()
        run: ./test/lima/cleanup.sh
```

---

## Related Documentation

- [Profiling Guide](profiling.md) - How to analyze profiles captured during load tests
- [Performance Tuning](performance.md) - Optimization techniques
- [Lima Testing](../test/lima/README.md) - Lima test environment documentation
- [Metrics](metrics.md) - Understanding Warren metrics

---

**Last Updated**: 2025-10-10
