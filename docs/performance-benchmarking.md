# Warren Performance Benchmarking Guide

This guide provides comprehensive performance benchmarking procedures for Warren v1.3.1, including baseline metrics, test scenarios, and performance targets.

## Table of Contents

- [Overview](#overview)
- [Performance Targets](#performance-targets)
- [Benchmarking Setup](#benchmarking-setup)
- [Core Metrics](#core-metrics)
- [Benchmark Scenarios](#benchmark-scenarios)
- [Performance Analysis](#performance-analysis)
- [Optimization Tips](#optimization-tips)

## Overview

Warren's performance characteristics are critical for production deployments. This guide helps you:
- Establish baseline performance metrics
- Identify performance bottlenecks
- Validate performance under load
- Compare performance across versions

## Performance Targets

### Latency Targets (v1.3.1)

| Operation | Target (p50) | Target (p95) | Target (p99) |
|-----------|--------------|--------------|--------------|
| Service Create | < 500ms | < 1s | < 2s |
| Service Update | < 500ms | < 1s | < 2s |
| Service Delete | < 200ms | < 500ms | < 1s |
| Task Schedule | < 2s | < 5s | < 10s |
| Task Start | < 5s | < 10s | < 15s |
| Raft Apply | < 50ms | < 100ms | < 200ms |
| API Request (read) | < 50ms | < 100ms | < 200ms |
| Reconciliation Cycle | < 5s | < 10s | < 20s |

### Throughput Targets

| Operation | Target |
|-----------|--------|
| API Requests/sec | > 100 |
| Service Creates/min | > 60 |
| Task Schedules/min | > 300 |
| Concurrent Tasks | > 1000 |

### Resource Targets

| Component | CPU | Memory |
|-----------|-----|--------|
| Manager Node | < 50% | < 512MB |
| Worker Node | < 30% | < 256MB |
| Per Task | < 5% | < 50MB |

## Benchmarking Setup

### Test Cluster Configuration

**Recommended**: 3 managers + 3 workers

```bash
# Manager nodes
- CPU: 2 cores
- RAM: 2GB
- Disk: 10GB SSD

# Worker nodes
- CPU: 4 cores
- RAM: 4GB
- Disk: 20GB SSD
```

### Prerequisites

```bash
# Install required tools
brew install hey     # HTTP load testing
brew install jq      # JSON parsing
brew install gnuplot # Plotting (optional)

# Or on Linux
apt-get install hey jq gnuplot
```

### Environment Preparation

```bash
# Clean cluster state
warren service ls | awk '{print $1}' | xargs -I{} warren service delete {}

# Verify clean state
warren service ls
warren task ls

# Enable debug logging (optional)
export WARREN_LOG_LEVEL=debug
```

## Core Metrics

### 1. Service Creation Latency

**Metric**: `warren_service_create_duration_seconds`

```bash
#!/bin/bash
# benchmark-service-create.sh

echo "Benchmarking Service Creation..."

ITERATIONS=100
TOTAL=0

for i in $(seq 1 $ITERATIONS); do
  START=$(date +%s%N)

  warren service create "bench-$i" \
    --image alpine:latest \
    --replicas 1 \
    --command "sleep,3600" \
    > /dev/null 2>&1

  END=$(date +%s%N)
  LATENCY=$(( (END - START) / 1000000 ))  # Convert to ms
  TOTAL=$((TOTAL + LATENCY))

  echo "Iteration $i: ${LATENCY}ms"
done

AVG=$((TOTAL / ITERATIONS))
echo ""
echo "Average Service Creation Latency: ${AVG}ms"

# Cleanup
for i in $(seq 1 $ITERATIONS); do
  warren service delete "bench-$i" > /dev/null 2>&1
done
```

**Expected**: Average < 500ms

### 2. Task Scheduling Latency

**Metric**: `warren_scheduling_latency_seconds`

```bash
#!/bin/bash
# benchmark-task-schedule.sh

echo "Benchmarking Task Scheduling..."

# Create service with multiple replicas
START=$(date +%s%N)

warren service create bench-sched \
  --image alpine:latest \
  --replicas 50 \
  --command "sleep,3600"

# Wait for all tasks to be scheduled
while true; do
  RUNNING=$(warren task ls --service bench-sched --format json | \
    jq '[.[] | select(.actual_state=="running")] | length')

  if [ "$RUNNING" -eq 50 ]; then
    break
  fi

  sleep 1
done

END=$(date +%s%N)
LATENCY=$(( (END - START) / 1000000000 ))  # Convert to seconds

echo "Scheduled 50 tasks in ${LATENCY}s"
echo "Average per task: $((LATENCY * 1000 / 50))ms"

# Cleanup
warren service delete bench-sched
```

**Expected**: < 100ms per task (5s total for 50 tasks)

### 3. API Request Latency

**Metric**: `warren_api_request_duration_seconds`

```bash
#!/bin/bash
# benchmark-api.sh

echo "Benchmarking API Performance..."

# Create some test data
warren service create api-test \
  --image alpine:latest \
  --replicas 5 \
  --command "sleep,3600"

sleep 10

# Benchmark list operations
echo ""
echo "=== List Services ==="
hey -n 1000 -c 10 -m GET http://localhost:8080/services

echo ""
echo "=== Get Service ==="
SERVICE_ID=$(warren service ls --format json | jq -r '.[0].id')
hey -n 1000 -c 10 -m GET http://localhost:8080/services/$SERVICE_ID

echo ""
echo "=== List Tasks ==="
hey -n 1000 -c 10 -m GET http://localhost:8080/tasks

# Cleanup
warren service delete api-test
```

**Expected**:
- p50 < 50ms
- p95 < 100ms
- p99 < 200ms

### 4. Raft Performance

**Metric**: `warren_raft_apply_duration_seconds`, `warren_raft_commit_duration_seconds`

```bash
#!/bin/bash
# benchmark-raft.sh

echo "Benchmarking Raft Performance..."

# Create many services quickly (generates Raft operations)
START=$(date +%s)

for i in $(seq 1 100); do
  warren service create "raft-test-$i" \
    --image alpine:latest \
    --replicas 1 \
    --command "sleep,10" \
    > /dev/null 2>&1 &

  # Limit concurrency
  if [ $((i % 10)) -eq 0 ]; then
    wait
  fi
done

wait

END=$(date +%s)
DURATION=$((END - START))

echo "Created 100 services in ${DURATION}s"
echo "Raft operations/sec: $((100 / DURATION))"

# Check Raft metrics
curl -s http://localhost:9090/metrics | grep warren_raft | grep -E "(apply|commit)"

# Cleanup
for i in $(seq 1 100); do
  warren service delete "raft-test-$i" > /dev/null 2>&1 &
done
wait
```

**Expected**: > 10 operations/second

### 5. Reconciliation Performance

**Metric**: `warren_reconciliation_duration_seconds`

```bash
#!/bin/bash
# benchmark-reconciliation.sh

echo "Benchmarking Reconciliation Performance..."

# Create load
warren service create recon-test \
  --image alpine:latest \
  --replicas 100 \
  --command "sleep,3600"

sleep 30

# Monitor reconciliation cycles
echo "Monitoring reconciliation for 60 seconds..."
for i in {1..12}; do
  DURATION=$(curl -s http://localhost:9090/metrics | \
    grep warren_reconciliation_duration_seconds_sum | \
    awk '{print $2}')

  CYCLES=$(curl -s http://localhost:9090/metrics | \
    grep warren_reconciliation_cycles_total | \
    awk '{print $2}')

  if [ ! -z "$CYCLES" ] && [ "$CYCLES" != "0" ]; then
    AVG=$(echo "scale=2; $DURATION / $CYCLES" | bc)
    echo "Cycle $i: Average reconciliation time: ${AVG}s"
  fi

  sleep 5
done

# Cleanup
warren service delete recon-test
```

**Expected**: < 5s per cycle under normal load

## Benchmark Scenarios

### Scenario 1: Baseline Performance

**Purpose**: Establish baseline metrics with minimal load

```bash
#!/bin/bash
# scenario-baseline.sh

echo "=== Baseline Performance Benchmark ==="
echo ""

# 1. Single service creation
echo "1. Service Creation (single):"
time warren service create baseline \
  --image alpine:latest \
  --replicas 1 \
  --command "sleep,3600"

# 2. Service scaling
echo ""
echo "2. Service Scaling:"
time warren service update baseline --replicas 10

sleep 10

# 3. API read operations
echo ""
echo "3. API Read Performance:"
hey -n 100 -c 1 -q 10 http://localhost:8080/services | \
  grep -E "(Requests/sec|Latencies)"

# 4. Metrics endpoint
echo ""
echo "4. Metrics Endpoint:"
time curl -s http://localhost:9090/metrics > /dev/null

# Cleanup
warren service delete baseline

echo ""
echo "Baseline benchmark complete"
```

### Scenario 2: Sustained Load

**Purpose**: Test performance under sustained load

```bash
#!/bin/bash
# scenario-sustained-load.sh

echo "=== Sustained Load Benchmark ==="
echo ""

# Deploy multiple services
echo "Deploying 20 services with 5 replicas each (100 tasks)..."
START=$(date +%s)

for i in $(seq 1 20); do
  warren service create "load-$i" \
    --image alpine:latest \
    --replicas 5 \
    --command "sleep,3600" \
    > /dev/null 2>&1 &

  if [ $((i % 5)) -eq 0 ]; then
    wait
  fi
done

wait

END=$(date +%s)
DEPLOY_TIME=$((END - START))

echo "Deployment completed in ${DEPLOY_TIME}s"
echo ""

# Monitor for 5 minutes
echo "Monitoring cluster under load for 5 minutes..."

for i in {1..60}; do
  # Check task count
  RUNNING=$(warren task ls --format json 2>/dev/null | \
    jq '[.[] | select(.actual_state=="running")] | length')

  # Check API latency
  LATENCY=$(curl -s -w "%{time_total}" http://localhost:8080/services -o /dev/null)

  # Check reconciliation time
  RECON=$(curl -s http://localhost:9090/metrics | \
    grep warren_reconciliation_duration_seconds_sum | \
    awk '{print $2}')

  echo "[$i/60] Tasks: $RUNNING, API: ${LATENCY}s, Recon: ${RECON}s"

  sleep 5
done

# Cleanup
echo ""
echo "Cleaning up..."
for i in $(seq 1 20); do
  warren service delete "load-$i" > /dev/null 2>&1 &
done
wait

echo "Sustained load benchmark complete"
```

### Scenario 3: Burst Load

**Purpose**: Test performance under burst traffic

```bash
#!/bin/bash
# scenario-burst-load.sh

echo "=== Burst Load Benchmark ==="
echo ""

# Create burst of services
echo "Creating burst of 50 services..."
START=$(date +%s)

for i in $(seq 1 50); do
  warren service create "burst-$i" \
    --image alpine:latest \
    --replicas 2 \
    --command "sleep,3600" \
    > /dev/null 2>&1 &
done

wait

END=$(date +%s)
BURST_TIME=$((END - START))

echo "Burst deployment: ${BURST_TIME}s"
echo "Services/sec: $((50 / BURST_TIME))"
echo ""

# Concurrent API requests
echo "Testing concurrent API requests..."
hey -n 1000 -c 50 http://localhost:8080/services | \
  grep -E "(Requests/sec|Latencies)"

# Cleanup
echo ""
echo "Cleaning up..."
for i in $(seq 1 50); do
  warren service delete "burst-$i" > /dev/null 2>&1 &
done
wait

echo "Burst load benchmark complete"
```

### Scenario 4: Failover Performance

**Purpose**: Measure leader failover time and impact

```bash
#!/bin/bash
# scenario-failover.sh

echo "=== Failover Performance Benchmark ==="
echo ""

# Deploy workload
echo "Deploying workload..."
warren service create failover-test \
  --image alpine:latest \
  --replicas 20 \
  --command "sleep,3600"

sleep 20

# Identify leader
LEADER=$(curl -s http://localhost:9090/metrics | \
  grep 'warren_raft_is_leader 1' | wc -l)

echo "Current leader identified"
echo ""

# Trigger failover (this is destructive!)
echo "WARNING: This will kill the leader process"
echo "Press Ctrl+C to cancel, or wait 5 seconds..."
sleep 5

START=$(date +%s)

# Kill leader (adjust based on your setup)
pkill -9 warren

echo "Leader killed at $(date)"

# Wait for new leader
while true; do
  NEW_LEADER=$(curl -s http://localhost:9090/metrics 2>/dev/null | \
    grep 'warren_raft_is_leader 1' | wc -l)

  if [ "$NEW_LEADER" -eq 1 ]; then
    END=$(date +%s)
    FAILOVER_TIME=$((END - START))
    echo "New leader elected in ${FAILOVER_TIME}s"
    break
  fi

  sleep 1
done

# Verify cluster still functional
echo ""
echo "Verifying cluster functionality..."
warren service ls > /dev/null 2>&1
if [ $? -eq 0 ]; then
  echo "✓ Cluster operational after failover"
else
  echo "✗ Cluster not responding"
fi

# Cleanup
warren service delete failover-test

echo "Failover benchmark complete"
```

## Performance Analysis

### Analyze Prometheus Metrics

```bash
#!/bin/bash
# analyze-metrics.sh

echo "=== Warren Performance Analysis ==="
echo ""

METRICS_URL="http://localhost:9090/metrics"

# Service operations
echo "Service Operations:"
curl -s $METRICS_URL | grep -E "warren_service_(create|update|delete)_duration" | \
  awk '{print $1, $2}' | column -t

echo ""

# Task operations
echo "Task Operations:"
curl -s $METRICS_URL | grep -E "warren_task_(create|start|stop)_duration" | \
  awk '{print $1, $2}' | column -t

echo ""

# Raft performance
echo "Raft Performance:"
curl -s $METRICS_URL | grep -E "warren_raft_(apply|commit)_duration" | \
  awk '{print $1, $2}' | column -t

echo ""

# Reconciliation
echo "Reconciliation:"
curl -s $METRICS_URL | grep warren_reconciliation | \
  awk '{print $1, $2}' | column -t

echo ""

# Resource utilization (if available)
echo "Resource Metrics:"
curl -s $METRICS_URL | grep -E "process_(cpu|memory)" | \
  awk '{print $1, $2}' | column -t
```

### Calculate Percentiles

```bash
#!/bin/bash
# calculate-percentiles.sh

# Extract histogram buckets for service creation
curl -s http://localhost:9090/metrics | \
  grep 'warren_service_create_duration_seconds_bucket' | \
  awk '{
    gsub(/le="/, "", $1)
    gsub(/".*/, "", $1)
    print $1, $2
  }' | \
  awk '
    BEGIN { total = 0 }
    {
      le = $1
      count = $2
      if (le != "+Inf") {
        if (total == 0) total = count
        pct = (count / total) * 100
        if (pct >= 50 && !p50) { p50 = le; print "p50:", p50 "s" }
        if (pct >= 95 && !p95) { p95 = le; print "p95:", p95 "s" }
        if (pct >= 99 && !p99) { p99 = le; print "p99:", p99 "s" }
      }
    }
  '
```

## Optimization Tips

### 1. Raft Performance

```bash
# Adjust Raft timing (in code)
# raftConfig.HeartbeatTimeout = 500 * time.Millisecond
# raftConfig.ElectionTimeout = 1000 * time.Millisecond
```

### 2. Reconciliation Frequency

```bash
# Adjust reconciler interval (in code)
# ticker := time.NewTicker(10 * time.Second)  // Default
# ticker := time.NewTicker(5 * time.Second)   // More frequent
```

### 3. Resource Limits

```bash
# Set resource limits for better performance
warren service create app \
  --image myapp:latest \
  --replicas 10 \
  --cpu-limit 0.5 \
  --memory-limit 512M
```

### 4. Logging Level

```bash
# Reduce logging overhead in production
export WARREN_LOG_LEVEL=info  # or warn
export WARREN_LOG_JSON=true   # Faster than console
```

### 5. Metrics Cardinality

```bash
# Be careful with high-cardinality labels
# Avoid: label per task_id
# Prefer: label per service_name
```

## Performance Reporting

### Create Performance Report

```bash
#!/bin/bash
# generate-performance-report.sh

REPORT="performance-report-$(date +%Y%m%d).txt"

echo "Warren Performance Report" > $REPORT
echo "Generated: $(date)" >> $REPORT
echo "" >> $REPORT

# System info
echo "=== System Information ===" >> $REPORT
uname -a >> $REPORT
echo "" >> $REPORT

# Cluster info
echo "=== Cluster Configuration ===" >> $REPORT
warren node ls >> $REPORT
echo "" >> $REPORT

# Run benchmarks
echo "=== Benchmark Results ===" >> $REPORT
./benchmark-service-create.sh >> $REPORT 2>&1
echo "" >> $REPORT
./benchmark-task-schedule.sh >> $REPORT 2>&1
echo "" >> $REPORT

# Metrics snapshot
echo "=== Metrics Snapshot ===" >> $REPORT
curl -s http://localhost:9090/metrics | \
  grep warren_ | \
  grep -v "#" >> $REPORT

echo ""
echo "Report saved to: $REPORT"
```

## References

- [Warren Metrics Guide](./logging.md)
- [Warren E2E Validation](./e2e-validation.md)
- [Prometheus Documentation](https://prometheus.io/docs/)
- [hey Load Testing Tool](https://github.com/rakyll/hey)
