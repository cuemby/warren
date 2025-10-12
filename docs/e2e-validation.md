# Warren End-to-End Deployment Validation

This guide covers the complete end-to-end validation process for Warren v1.1.0, including deployment, testing, and verification of all features.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Deployment Steps](#deployment-steps)
- [Validation Checklist](#validation-checklist)
- [Test Scenarios](#test-scenarios)
- [Performance Verification](#performance-verification)
- [Health & Monitoring Checks](#health--monitoring-checks)
- [Troubleshooting](#troubleshooting)

## Overview

This validation ensures Warren is production-ready by testing:
- ✅ Cluster formation (3 managers + 3 workers)
- ✅ High availability and leader failover
- ✅ Service deployment and scheduling
- ✅ Task lifecycle management
- ✅ Health monitoring
- ✅ Secrets and volumes
- ✅ Built-in ingress
- ✅ Performance under load

## Prerequisites

### System Requirements

**Manager Nodes** (minimum 3 for HA):
- 2 CPU cores
- 2 GB RAM
- 10 GB disk
- Network connectivity between all nodes

**Worker Nodes** (minimum 3 for testing):
- 2 CPU cores
- 2 GB RAM
- 20 GB disk (for container images)
- containerd installed

### Software Requirements

- Warren binary (v1.1.0)
- containerd (v1.7+)
- Go 1.22+ (for running Go-based E2E tests)
- curl/jq (for API testing)

### Network Requirements

- Ports 8080 (API)
- Ports 9090 (Metrics)
- Ports 8000 (HTTP Ingress)
- Ports 8443 (HTTPS Ingress)
- Raft communication between managers

## Deployment Steps

### 1. Build Warren

```bash
# Build for your platform
make build

# Or build embedded (includes containerd)
make build-embedded

# Verify build
./bin/warren version
```

### 2. Initialize First Manager

```bash
# On manager-1
warren cluster init \
  --node-id manager-1 \
  --bind-addr 192.168.1.10:7946 \
  --data-dir /var/lib/warren \
  --api-addr 0.0.0.0:8080

# Save the join tokens displayed
```

### 3. Join Additional Managers

```bash
# On manager-2
warren cluster join \
  --node-id manager-2 \
  --bind-addr 192.168.1.11:7946 \
  --data-dir /var/lib/warren \
  --api-addr 0.0.0.0:8080 \
  --leader 192.168.1.10:8080 \
  --token <manager-token> \
  --role manager

# On manager-3
warren cluster join \
  --node-id manager-3 \
  --bind-addr 192.168.1.12:7946 \
  --data-dir /var/lib/warren \
  --api-addr 0.0.0.0:8080 \
  --leader 192.168.1.10:8080 \
  --token <manager-token> \
  --role manager
```

### 4. Join Worker Nodes

```bash
# On worker-1
warren cluster join \
  --node-id worker-1 \
  --bind-addr 192.168.1.20:7946 \
  --data-dir /var/lib/warren \
  --leader 192.168.1.10:8080 \
  --token <worker-token> \
  --role worker

# Repeat for worker-2 and worker-3
```

## Validation Checklist

### Phase 1: Cluster Health

- [ ] All 3 managers are in "ready" state
- [ ] One manager is elected as leader
- [ ] All 3 workers are in "ready" state
- [ ] Raft consensus is functioning (check logs)
- [ ] Health endpoints responding on all managers

**Verify**:
```bash
# Check cluster status
warren node ls

# Check health endpoints
curl http://localhost:9090/health
curl http://localhost:9090/ready
curl http://localhost:9090/live

# Check metrics
curl http://localhost:9090/metrics | grep warren_raft_is_leader
```

**Expected Output**:
```
NODE ID      ROLE     STATUS  ADDRESS
manager-1    manager  ready   192.168.1.10:7946
manager-2    manager  ready   192.168.1.11:7946
manager-3    manager  ready   192.168.1.12:7946
worker-1     worker   ready   192.168.1.20:7946
worker-2     worker   ready   192.168.1.21:7946
worker-3     worker   ready   192.168.1.22:7946
```

### Phase 2: Service Deployment

- [ ] Create a simple service (nginx)
- [ ] Service tasks are scheduled across workers
- [ ] Tasks reach "running" state
- [ ] Tasks are healthy (if health check configured)
- [ ] Service is accessible

**Test**:
```bash
# Create nginx service
warren service create nginx \
  --image nginx:latest \
  --replicas 3 \
  --ports 80:8080

# Check service status
warren service ls
warren service inspect nginx

# Check tasks
warren task ls --service nginx

# Test connectivity (from worker nodes)
curl http://localhost:8080
```

**Expected**: 3 tasks running across workers, nginx serving content

### Phase 3: Scaling Operations

- [ ] Scale service up (replicas: 3 → 6)
- [ ] New tasks are scheduled
- [ ] Scale down (replicas: 6 → 2)
- [ ] Tasks are gracefully terminated

**Test**:
```bash
# Scale up
warren service update nginx --replicas 6
sleep 10
warren task ls --service nginx | wc -l  # Should be 6

# Scale down
warren service update nginx --replicas 2
sleep 10
warren task ls --service nginx | wc -l  # Should be 2
```

### Phase 4: Leader Failover

- [ ] Identify current leader
- [ ] Stop leader process
- [ ] New leader is elected (< 10 seconds)
- [ ] Cluster remains operational
- [ ] Services continue running

**Test**:
```bash
# Find leader
LEADER=$(curl -s http://localhost:9090/metrics | \
  grep 'warren_raft_is_leader 1' | \
  head -1 | \
  awk '{print $NF}')

echo "Current leader: $LEADER"

# Stop leader (on that node)
# pkill warren

# Wait for new leader election
sleep 15

# Verify new leader elected
curl http://localhost:9090/metrics | grep warren_raft_is_leader

# Verify services still operational
warren service ls
```

**Expected**: New leader elected within 10 seconds, no service disruption

### Phase 5: Secrets Management

- [ ] Create a secret
- [ ] Deploy service with secret mount
- [ ] Secret is available in container
- [ ] Secret cannot be read via API (encrypted)
- [ ] Delete secret

**Test**:
```bash
# Create secret
echo "supersecret123" | warren secret create db-password -

# Create service with secret
warren service create app \
  --image alpine:latest \
  --replicas 1 \
  --secret db-password \
  --command "sh,-c,cat /run/secrets/db-password && sleep 3600"

# Verify secret mounted
warren task logs <task-id>  # Should show "supersecret123"

# Try to read secret via API (should be encrypted)
warren secret inspect db-password  # Data should be encrypted

# Cleanup
warren service delete app
warren secret delete db-password
```

### Phase 6: Volumes

- [ ] Create a volume
- [ ] Mount volume in service
- [ ] Write data to volume
- [ ] Restart service
- [ ] Data persists after restart

**Test**:
```bash
# Create volume
warren volume create data-vol

# Create service with volume
warren service create writer \
  --image alpine:latest \
  --replicas 1 \
  --volume data-vol:/data \
  --command "sh,-c,echo 'test data' > /data/test.txt && sleep 3600"

# Wait for task to run
sleep 10

# Read data from volume (via another task)
warren service create reader \
  --image alpine:latest \
  --replicas 1 \
  --volume data-vol:/data \
  --command "sh,-c,cat /data/test.txt && sleep 10"

warren task logs <reader-task-id>  # Should show "test data"

# Cleanup
warren service delete writer
warren service delete reader
warren volume delete data-vol
```

### Phase 7: Built-in Ingress

- [ ] Create ingress rule
- [ ] HTTP traffic is routed correctly
- [ ] Host-based routing works
- [ ] Path-based routing works
- [ ] HTTPS/TLS works (if configured)

**Test**:
```bash
# Create backend service
warren service create backend \
  --image nginx:latest \
  --replicas 2 \
  --ports 80

# Create ingress rule
warren ingress create web \
  --host example.local \
  --path / \
  --backend backend:80

# Test HTTP routing
curl -H "Host: example.local" http://localhost:8000/

# Check ingress status
warren ingress ls
warren ingress inspect web

# Cleanup
warren ingress delete web
warren service delete backend
```

### Phase 8: Health Monitoring

- [ ] Service with health check defined
- [ ] Unhealthy tasks are restarted
- [ ] Health status reflected in task list

**Test**:
```bash
# Create service with health check
warren service create web \
  --image nginx:latest \
  --replicas 2 \
  --ports 80 \
  --health-cmd "curl -f http://localhost/ || exit 1" \
  --health-interval 10s \
  --health-timeout 5s \
  --health-retries 3

# Wait for health checks
sleep 30

# Check task health status
warren task ls --service web
# Should show healthy: true

# Cleanup
warren service delete web
```

## Test Scenarios

### Scenario 1: Basic Service Deployment

**Objective**: Verify end-to-end service deployment

```bash
# 1. Deploy simple web service
warren service create hello \
  --image hashicorp/http-echo:latest \
  --replicas 3 \
  --ports 5678:8080 \
  --env TEXT="Hello from Warren"

# 2. Wait for deployment
sleep 15

# 3. Verify all tasks running
RUNNING=$(warren task ls --service hello --format json | jq '[.[] | select(.actual_state=="running")] | length')
if [ "$RUNNING" -eq 3 ]; then
  echo "✓ All tasks running"
else
  echo "✗ Expected 3 running tasks, got $RUNNING"
  exit 1
fi

# 4. Test service connectivity
for i in {1..3}; do
  curl http://localhost:8080
done

# 5. Cleanup
warren service delete hello
```

### Scenario 2: Rolling Update

**Objective**: Verify rolling update functionality

```bash
# 1. Deploy initial version
warren service create app \
  --image nginx:1.24 \
  --replicas 6 \
  --ports 80:8080

# 2. Wait for deployment
sleep 20

# 3. Perform rolling update
warren service update app \
  --image nginx:1.25 \
  --update-parallelism 2 \
  --update-delay 5s

# 4. Monitor update progress
watch -n 2 'warren task ls --service app'

# 5. Verify all tasks updated
IMAGE_COUNT=$(warren task ls --service app --format json | jq '[.[] | select(.image=="nginx:1.25")] | length')
if [ "$IMAGE_COUNT" -eq 6 ]; then
  echo "✓ Rolling update successful"
fi

# 6. Cleanup
warren service delete app
```

### Scenario 3: Load Test

**Objective**: Verify cluster handles load

```bash
# 1. Deploy load test target
warren service create load-target \
  --image nginx:latest \
  --replicas 10 \
  --ports 80:8080

# 2. Wait for full deployment
sleep 30

# 3. Generate load
for i in {1..1000}; do
  curl -s http://localhost:8080 > /dev/null &
done
wait

# 4. Check cluster health during load
curl http://localhost:9090/metrics | grep warren_tasks_total

# 5. Verify no task failures
FAILED=$(warren task ls --service load-target --format json | jq '[.[] | select(.actual_state=="failed")] | length')
if [ "$FAILED" -eq 0 ]; then
  echo "✓ No task failures under load"
fi

# 6. Cleanup
warren service delete load-target
```

## Performance Verification

### Service Creation Latency

**Target**: < 1 second for service creation

```bash
# Measure service creation time
time warren service create perf-test \
  --image alpine:latest \
  --replicas 1 \
  --command "sleep,3600"

warren service delete perf-test
```

### Task Scheduling Latency

**Target**: < 5 seconds from service creation to task running

```bash
START=$(date +%s)
warren service create sched-test \
  --image alpine:latest \
  --replicas 1 \
  --command "sleep,3600"

# Wait for running state
while [ "$(warren task ls --service sched-test --format json | jq -r '.[0].actual_state')" != "running" ]; do
  sleep 0.5
done

END=$(date +%s)
LATENCY=$((END - START))

echo "Task scheduling latency: ${LATENCY}s"

warren service delete sched-test
```

### API Response Time

**Target**: < 100ms for read operations

```bash
# Measure API latency
for i in {1..10}; do
  time warren service ls > /dev/null
done
```

### Metrics Export

```bash
# Check metrics response time
time curl -s http://localhost:9090/metrics > /dev/null

# Verify key metrics present
curl -s http://localhost:9090/metrics | grep -E "(warren_services_total|warren_tasks_total|warren_nodes_total)"
```

## Health & Monitoring Checks

### Health Endpoints

```bash
# Overall health
curl http://localhost:9090/health | jq .

# Readiness check
curl http://localhost:9090/ready | jq .

# Liveness check
curl http://localhost:9090/live | jq .
```

**Expected Health Response**:
```json
{
  "status": "healthy",
  "timestamp": "2025-10-12T22:00:00Z",
  "components": {
    "raft": "ready",
    "containerd": "ready",
    "api": "ready"
  },
  "version": "1.1.0"
}
```

### Metrics Verification

```bash
# Cluster metrics
curl -s http://localhost:9090/metrics | grep warren_nodes_total
curl -s http://localhost:9090/metrics | grep warren_services_total
curl -s http://localhost:9090/metrics | grep warren_tasks_total

# Raft metrics
curl -s http://localhost:9090/metrics | grep warren_raft_is_leader
curl -s http://localhost:9090/metrics | grep warren_raft_peers_total

# Latency metrics (added in Phase 1)
curl -s http://localhost:9090/metrics | grep warren_service_create_duration
curl -s http://localhost:9090/metrics | grep warren_reconciliation_duration
```

### Logging Verification

```bash
# Check log output (JSON or console)
tail -f /var/log/warren.log

# Search for errors
grep -i error /var/log/warren.log

# Check specific components
grep 'component":"scheduler' /var/log/warren.log | tail -20
```

## Troubleshooting

### Cluster Won't Form

**Symptoms**: Nodes can't join, Raft won't form

**Debug**:
```bash
# Check Raft logs
grep raft /var/log/warren.log

# Check network connectivity
ping <manager-ip>
telnet <manager-ip> 8080

# Check node status
warren node ls
```

### Tasks Not Scheduling

**Symptoms**: Tasks stuck in "pending" state

**Debug**:
```bash
# Check scheduler logs
grep scheduler /var/log/warren.log

# Check node resources
warren node inspect <node-id>

# Check task assignment
warren task inspect <task-id>
```

### Services Not Accessible

**Symptoms**: Can't reach service endpoints

**Debug**:
```bash
# Check service ports
warren service inspect <service-name>

# Check task state
warren task ls --service <service-name>

# Check containerd
crictl ps
crictl logs <container-id>
```

### Health Checks Failing

**Symptoms**: `/ready` returns 503

**Debug**:
```bash
# Check component health
curl http://localhost:9090/health | jq .components

# Check Raft status
curl http://localhost:9090/metrics | grep warren_raft

# Check logs for errors
grep -i "health" /var/log/warren.log
```

## Success Criteria

Warren v1.1.0 is considered production-ready if:

- ✅ **Cluster Formation**: 3+3 cluster forms successfully
- ✅ **High Availability**: Leader failover < 10 seconds
- ✅ **Service Deployment**: Services deploy and scale reliably
- ✅ **Task Management**: Tasks reach running state consistently
- ✅ **Health Monitoring**: Health checks work correctly
- ✅ **Secrets & Volumes**: Data management functions properly
- ✅ **Ingress**: HTTP/HTTPS routing works
- ✅ **Performance**: Meets latency targets
- ✅ **Observability**: Metrics and logs are useful
- ✅ **Stability**: No crashes or data corruption under load

## Next Steps

After successful validation:

1. **Document Results**: Record metrics and observations
2. **Performance Baseline**: Document baseline performance metrics
3. **Create Runbook**: Operational procedures for production
4. **Deploy to Production**: Follow deployment guide
5. **Monitor**: Set up Prometheus/Grafana for monitoring

## References

- [Warren Deployment Guide](./deployment.md)
- [Warren Logging Guide](./logging.md)
- [Warren Metrics Reference](./metrics.md)
- [Warren Troubleshooting Guide](./troubleshooting.md)
