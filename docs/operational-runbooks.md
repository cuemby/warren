# Warren Operational Runbooks

**Version**: 1.1.1+
**Last Updated**: 2025-10-13
**Audience**: Operations/SRE Teams

---

## Table of Contents

- [Overview](#overview)
- [Common Operations](#common-operations)
- [Incident Response](#incident-response)
- [Maintenance Procedures](#maintenance-procedures)
- [Disaster Recovery](#disaster-recovery)
- [Capacity Planning](#capacity-planning)

---

## Overview

This document provides step-by-step operational procedures for managing Warren in production.

### Quick Reference

**Emergency Contacts**: [Add your team's contact info]
**Monitoring Dashboard**: [Add Grafana URL]
**Alert Manager**: [Add AlertManager URL]

### Before You Begin

- ✅ Familiarize yourself with [Observability Guide](observability.md)
- ✅ Have access to manager nodes
- ✅ Can query metrics at port 9090
- ✅ Warren CLI installed locally

---

## Common Operations

### Checking Cluster Health

**Purpose**: Verify cluster is healthy before/after changes

**Steps**:

1. **Check overall health** (any manager):
   ```bash
   curl http://manager-1:9090/health | jq .
   ```

   **Expected**: `"status": "healthy"`, all components healthy

2. **Check Raft leadership**:
   ```bash
   curl -s http://manager-1:9090/metrics | grep warren_raft_is_leader
   ```

   **Expected**: Exactly one manager has value `1`

3. **Check node count**:
   ```bash
   warren node ls
   ```

   **Expected**: All expected nodes listed with status "Ready"

4. **Check service status**:
   ```bash
   warren service ls
   ```

   **Expected**: All services with desired replica count

**Verification**:
- [ ] Health endpoint returns 200 OK
- [ ] One and only one Raft leader
- [ ] All nodes in Ready state
- [ ] No stuck containers (pending for >5 minutes)

**Related Metrics**:
```promql
# Cluster has leader
sum(warren_raft_is_leader) == 1

# All nodes ready
warren_nodes_total{status="ready"}

# No stuck containers
warren_containers_total{state="pending"}
```

---

### Deploying a New Service

**Purpose**: Deploy a new service to the cluster

**Prerequisites**:
- [ ] Container image available in registry
- [ ] Resource requirements determined
- [ ] Health check endpoint configured (if HTTP service)

**Steps**:

1. **Create service YAML**:
   ```yaml
   name: myapp
   image: myregistry.com/myapp:v1.0.0
   replicas: 3
   mode: replicated
   resources:
     cpu_limit: 1.0
     memory_limit: 512Mi
   health_check:
     type: http
     endpoint: http://localhost:8080/health
     interval: 10s
     timeout: 5s
     retries: 3
   ```

2. **Deploy service**:
   ```bash
   warren service create -f myapp.yaml
   ```

3. **Verify deployment**:
   ```bash
   # Check service created
   warren service ls | grep myapp

   # Wait for containers to be running (max 2 minutes)
   timeout 120 bash -c 'until [ $(warren service ps myapp | grep -c "Running") -eq 3 ]; do sleep 5; done'

   # Check container distribution
   warren service ps myapp
   ```

4. **Monitor deployment metrics**:
   ```bash
   # Check scheduling latency
   curl -s http://manager-1:9090/metrics | \
     grep 'warren_scheduling_latency_seconds' | grep myapp

   # Check if any scheduling failures
   curl -s http://manager-1:9090/metrics | \
     grep warren_containers_failed_total
   ```

5. **Test service health**:
   ```bash
   # If service exposes port, test connectivity
   curl http://<container-ip>:8080/health
   ```

**Rollback** (if deployment fails):
```bash
warren service delete myapp
```

**Verification Checklist**:
- [ ] Service created successfully
- [ ] All replicas running within 2 minutes
- [ ] Containers distributed across workers
- [ ] Health checks passing
- [ ] Service accessible (if applicable)
- [ ] No error spikes in logs

---

### Scaling a Service

**Purpose**: Increase or decrease service replica count

**Steps**:

1. **Check current state**:
   ```bash
   warren service inspect myapp | jq '.replicas'
   warren service ps myapp | grep Running | wc -l
   ```

2. **Update replica count**:
   ```bash
   # Scale up to 5 replicas
   warren service update myapp --replicas 5

   # Or scale down to 2
   warren service update myapp --replicas 2
   ```

3. **Monitor scaling progress**:
   ```bash
   # Watch container count
   watch -n 2 'warren service ps myapp | grep Running | wc -l'

   # Check metrics
   curl -s http://manager-1:9090/metrics | \
     grep 'warren_containers_total{state="running"}'
   ```

4. **Verify new replicas**:
   ```bash
   # List all containers
   warren service ps myapp

   # Check distribution
   warren service ps myapp | awk '{print $4}' | sort | uniq -c
   ```

**Expected Timeline**:
- Scale up: ~10-30 seconds per new container
- Scale down: Immediate (graceful shutdown may take up to 10s)

**Verification**:
- [ ] Desired replica count reached
- [ ] All new containers healthy
- [ ] Load balanced across workers
- [ ] No scheduling failures

---

### Updating a Service (Rolling Update)

**Purpose**: Deploy new version with zero downtime

**Prerequisites**:
- [ ] New image built and pushed to registry
- [ ] Health checks configured on service
- [ ] Rollback plan prepared

**Steps**:

1. **Pre-update verification**:
   ```bash
   # Record current image
   warren service inspect myapp | jq '.image'

   # Ensure all replicas healthy
   warren service ps myapp | grep -v Running && echo "WARNING: Unhealthy containers"
   ```

2. **Perform rolling update**:
   ```bash
   warren service update myapp --image myregistry.com/myapp:v1.1.0
   ```

3. **Monitor update progress**:
   ```bash
   # Watch container versions
   watch -n 5 'warren service ps myapp'

   # Check update metrics
   curl -s http://manager-1:9090/metrics | \
     grep warren_service_update_duration_seconds
   ```

4. **Monitor for errors**:
   ```bash
   # Check logs on manager for update events
   journalctl -u warren-manager -f | grep -i "update\|error"

   # Check container failures
   curl -s http://manager-1:9090/metrics | \
     grep warren_containers_failed_total
   ```

5. **Verify update completed**:
   ```bash
   # All containers on new image
   warren service ps myapp | grep v1.1.0 | wc -l

   # Health checks passing
   curl http://<container-ip>:8080/health
   ```

**Rollback** (if update fails):
```bash
# Immediately rollback to previous image
warren service update myapp --image myregistry.com/myapp:v1.0.0
```

**Verification**:
- [ ] All containers updated to new image
- [ ] No container failures during update
- [ ] Health checks passing
- [ ] Service responding correctly
- [ ] Update duration within expected range

---

## Incident Response

### Incident: No Raft Leader

**Symptoms**:
- Alert: `WarrenNoLeader` firing
- API requests failing with "no leader" errors
- Cluster operations blocked

**Diagnosis**:
```bash
# Check leadership status on all managers
for host in manager-1 manager-2 manager-3; do
  echo "=== $host ==="
  curl -s http://$host:9090/metrics | grep warren_raft_is_leader
  curl -s http://$host:9090/health | jq .components.raft
done

# Check Raft peer count
curl -s http://manager-1:9090/metrics | grep warren_raft_peers_total

# Check manager logs
ssh manager-1 'journalctl -u warren-manager -n 100 | grep -i raft'
```

**Common Causes**:
1. **Network partition**: Managers can't communicate
2. **Majority down**: 2 of 3 managers offline
3. **Disk I/O issues**: Raft can't commit to disk

**Resolution**:

**If 2+ managers are online**:
```bash
# 1. Check network connectivity between managers
for host in manager-1 manager-2 manager-3; do
  ssh $host 'warren node ls'
done

# 2. Check Raft logs for election issues
ssh manager-1 'journalctl -u warren-manager | grep -i "election\|heartbeat"'

# 3. If logs show election timeouts, restart managers one by one
ssh manager-1 'systemctl restart warren-manager'
# Wait 30 seconds for re-election
sleep 30
ssh manager-2 'systemctl restart warren-manager'
```

**If only 1 manager online** (emergency recovery):
```bash
# WARNING: This resets the cluster. Use only as last resort.

# 1. Stop all managers
for host in manager-1 manager-2 manager-3; do
  ssh $host 'systemctl stop warren-manager'
done

# 2. On one manager, bootstrap new cluster
ssh manager-1 'warren cluster recover --force --data-dir /var/lib/warren'

# 3. Start recovered manager
ssh manager-1 'systemctl start warren-manager'

# 4. Re-join other managers
ssh manager-2 'warren cluster join --leader manager-1:8080 --token <token> --role manager'
ssh manager-3 'warren cluster join --leader manager-1:8080 --token <token> --role manager'
```

**Post-Resolution**:
- [ ] Verify one leader elected
- [ ] All managers show leader in health check
- [ ] Test API operation (create test service)
- [ ] Review root cause (network logs, disk I/O)

**Prevention**:
- Monitor network latency between managers
- Use dedicated NICs for Raft traffic
- Monitor disk I/O and ensure SSDs for data directory

---

### Incident: High Container Scheduling Failures

**Symptoms**:
- Alert: `WarrenHighSchedulingFailureRate` firing
- Services stuck in pending state
- `warren_containers_failed_total` increasing

**Diagnosis**:
```bash
# Check scheduling failure rate
curl -s http://manager-1:9090/metrics | grep warren_containers_failed_total

# Check worker node status
warren node ls

# Check recent scheduler logs
ssh manager-1 'journalctl -u warren-manager | grep "scheduler" | grep -i "error\|failed" | tail -20'

# Check resource availability
warren node ls --format '{{.ID}}\t{{.CPU}}\t{{.Memory}}'
```

**Common Causes**:
1. **No ready workers**: All workers offline or draining
2. **Resource exhaustion**: Workers at capacity
3. **Volume affinity**: Volume pinned to offline node

**Resolution**:

**If no ready workers**:
```bash
# Check worker status
warren node ls | grep worker

# Check worker health
for host in worker-1 worker-2 worker-3; do
  echo "=== $host ==="
  curl -s http://$host:9090/health | jq .
  ssh $host 'systemctl status warren-worker'
done

# Restart unhealthy workers
ssh worker-1 'systemctl restart warren-worker'
```

**If resource exhaustion**:
```bash
# List containers by node
warren node ls --format '{{.ID}}' | while read node; do
  echo "=== $node ==="
  warren service ps | grep $node | wc -l
done

# Scale down or remove services
warren service delete <unused-service>
warren service update <service> --replicas 2  # scale down
```

**If volume affinity issues**:
```bash
# Identify service with volume
warren service ls | grep volumes

# Check volume location
warren volume ls

# If volume on offline node, migrate volume
# (requires manual data migration)
```

**Post-Resolution**:
- [ ] Scheduling success rate back to >95%
- [ ] Pending containers resolved
- [ ] Workers at healthy utilization (<80%)
- [ ] Alert cleared

---

### Incident: Slow Raft Commits

**Symptoms**:
- Alert: `WarrenSlowRaftCommits` firing
- API operations taking >2 seconds
- `warren_raft_commit_duration_seconds` p95 > 1s

**Diagnosis**:
```bash
# Check Raft commit latency
curl -s http://manager-1:9090/metrics | \
  grep 'warren_raft_commit_duration_seconds' | head -20

# Check disk I/O on managers
for host in manager-1 manager-2 manager-3; do
  echo "=== $host ==="
  ssh $host 'iostat -x 1 3 | grep -A 3 "Device"'
done

# Check Raft log size
ssh manager-1 'ls -lh /var/lib/warren/raft/*.db'

# Check network latency between managers
for host in manager-2 manager-3; do
  ping -c 5 $host
done
```

**Common Causes**:
1. **Slow disk I/O**: HDD or overloaded disk
2. **Network latency**: High RTT between managers
3. **Large log backlog**: Raft log not compacting

**Resolution**:

**If slow disk I/O**:
```bash
# Check disk type
ssh manager-1 'lsblk -o NAME,ROTA,TYPE,SIZE,MOUNTPOINT | grep warren'

# If ROTA=1 (HDD), migrate to SSD:
# 1. Add new manager with SSD
# 2. Demote old manager
# 3. Remove old manager
```

**If network latency**:
```bash
# Check MTU settings
for host in manager-1 manager-2 manager-3; do
  ssh $host 'ip link show | grep mtu'
done

# Tune MTU if needed (requires network admin)
```

**If large Raft log**:
```bash
# Check log size
ssh manager-1 'du -sh /var/lib/warren/raft/'

# Trigger manual snapshot
ssh manager-1 'kill -USR1 $(pidof warren)'

# Wait for snapshot to complete
sleep 10

# Verify log reduced
ssh manager-1 'du -sh /var/lib/warren/raft/'
```

**Post-Resolution**:
- [ ] Raft commit p95 < 500ms
- [ ] API operations responsive
- [ ] Disk I/O wait <20%
- [ ] Alert cleared

---

## Maintenance Procedures

### Upgrading Warren

**Purpose**: Upgrade Warren to new version with zero downtime

**Prerequisites**:
- [ ] New Warren binary tested in staging
- [ ] Backup of manager data directories
- [ ] Maintenance window scheduled (optional)

**Steps**:

1. **Backup cluster state**:
   ```bash
   # On each manager
   ssh manager-1 'tar -czf /backup/warren-manager-1-$(date +%Y%m%d).tar.gz /var/lib/warren/'
   ```

2. **Upgrade managers one at a time**:
   ```bash
   # Manager 1 (not leader)
   ssh manager-1 'systemctl stop warren-manager'
   scp warren-v1.2.0 manager-1:/usr/local/bin/warren
   ssh manager-1 'systemctl start warren-manager'

   # Wait for manager to rejoin
   sleep 30
   curl -s http://manager-1:9090/health | jq .status

   # Repeat for manager-2 and manager-3
   ```

3. **Upgrade workers** (can be done in parallel):
   ```bash
   # Drain worker before upgrade
   warren node update worker-1 --availability drain

   # Wait for containers to migrate (1-2 minutes)
   sleep 120

   # Upgrade worker
   ssh worker-1 'systemctl stop warren-worker'
   scp warren-v1.2.0 worker-1:/usr/local/bin/warren
   ssh worker-1 'systemctl start warren-worker'

   # Reactivate worker
   warren node update worker-1 --availability active
   ```

4. **Verify upgrade**:
   ```bash
   # Check versions
   warren version

   # All nodes should report new version
   warren node ls
   ```

**Rollback** (if issues occur):
```bash
# Stop new version
systemctl stop warren-manager

# Restore old binary
cp /usr/local/bin/warren-v1.1.1 /usr/local/bin/warren

# Restore data (if needed)
tar -xzf /backup/warren-manager-1-<date>.tar.gz -C /

# Start old version
systemctl start warren-manager
```

**Verification**:
- [ ] All nodes running new version
- [ ] Raft cluster stable
- [ ] All services running
- [ ] Metrics reporting correctly

---

### Adding a New Manager Node

**Purpose**: Scale manager quorum from 3 to 5 (or replace failed manager)

**Prerequisites**:
- [ ] New server provisioned
- [ ] Network connectivity to existing managers
- [ ] containerd installed

**Steps**:

1. **Generate join token** (on existing manager):
   ```bash
   warren cluster token create --role manager
   # Save token output
   ```

2. **Install Warren on new node**:
   ```bash
   ssh manager-4 'curl -L https://warren.io/install.sh | sh'
   ```

3. **Join new manager**:
   ```bash
   ssh manager-4 'warren cluster join \
     --node-id manager-4 \
     --bind-addr <manager-4-ip>:7946 \
     --data-dir /var/lib/warren \
     --api-addr 0.0.0.0:8080 \
     --leader manager-1:8080 \
     --token <manager-token> \
     --role manager'
   ```

4. **Verify manager joined**:
   ```bash
   # Check peer count (should increase by 1)
   curl -s http://manager-1:9090/metrics | grep warren_raft_peers_total

   # Check new manager health
   curl -s http://manager-4:9090/health | jq .

   # List all nodes
   warren node ls | grep manager
   ```

5. **Monitor Raft replication**:
   ```bash
   # Check replication lag
   for host in manager-1 manager-2 manager-3 manager-4; do
     echo "=== $host ==="
     curl -s http://$host:9090/metrics | \
       grep 'warren_raft_log_index\|warren_raft_applied_index'
   done
   ```

**Verification**:
- [ ] New manager shows in `warren node ls`
- [ ] Raft peer count increased
- [ ] New manager health checks passing
- [ ] Replication lag <100 entries

---

### Removing a Manager Node

**Purpose**: Scale down manager quorum or remove failed node

**WARNING**: Maintaining quorum is critical. Never remove managers simultaneously.

**Prerequisites**:
- [ ] At least 3 managers will remain
- [ ] Node to remove is not the current leader

**Steps**:

1. **Check if node is leader**:
   ```bash
   curl -s http://manager-2:9090/metrics | grep warren_raft_is_leader
   ```

   If leader (value=1), trigger re-election:
   ```bash
   # Force leadership transfer
   ssh manager-2 'systemctl restart warren-manager'
   # Wait for new leader election (30 seconds)
   sleep 30
   ```

2. **Remove node from cluster**:
   ```bash
   warren node rm manager-2
   ```

3. **Stop Warren on removed node**:
   ```bash
   ssh manager-2 'systemctl stop warren-manager'
   ssh manager-2 'systemctl disable warren-manager'
   ```

4. **Verify removal**:
   ```bash
   # Check peer count decreased
   curl -s http://manager-1:9090/metrics | grep warren_raft_peers_total

   # Verify node not in list
   warren node ls | grep manager-2
   ```

**Verification**:
- [ ] Node removed from cluster
- [ ] Raft peer count decreased
- [ ] Cluster still has leader
- [ ] API operations working

---

## Disaster Recovery

### Scenario: All Managers Lost

**Situation**: All manager nodes failed, data intact

**Recovery Steps**:

1. **Assess data integrity**:
   ```bash
   # Check if any manager data directories intact
   ssh manager-1 'ls -lh /var/lib/warren/raft/'
   ```

2. **Restore from most recent manager**:
   ```bash
   # Identify most recent data (highest log index)
   for host in manager-1 manager-2 manager-3; do
     ssh $host 'ls -lth /var/lib/warren/raft/ | head -5'
   done
   ```

3. **Bootstrap single-node cluster**:
   ```bash
   # On manager with most recent data
   ssh manager-1 'warren cluster recover --force --data-dir /var/lib/warren'
   ```

4. **Start recovered manager**:
   ```bash
   ssh manager-1 'systemctl start warren-manager'

   # Verify it's leader
   curl -s http://manager-1:9090/metrics | grep warren_raft_is_leader
   ```

5. **Re-add other managers**:
   ```bash
   # Generate new tokens
   warren cluster token create --role manager

   # Join managers
   ssh manager-2 'warren cluster join --leader manager-1:8080 ...'
   ssh manager-3 'warren cluster join --leader manager-1:8080 ...'
   ```

6. **Verify cluster state**:
   ```bash
   warren node ls
   warren service ls
   warren volume ls
   ```

**Expected Recovery Time**: 5-10 minutes

---

## Capacity Planning

### Monitoring Resource Usage

**Purpose**: Track resource trends to plan capacity

**Metrics to Monitor**:

```promql
# Node count by status
warren_nodes_total{status="ready"}

# Service count
warren_services_total

# Container count by state
warren_containers_total

# Scheduling latency trend
histogram_quantile(0.95, rate(warren_scheduling_latency_seconds_bucket[1h]))

# Raft commit latency trend
histogram_quantile(0.95, rate(warren_raft_commit_duration_seconds_bucket[1h]))
```

**Capacity Thresholds**:
- **Workers**: Add workers when average container density >30 per worker
- **Managers**: Keep 3 managers (5 for very large clusters >100 nodes)
- **Storage**: Alert when Raft log directory >10GB

**Scaling Guidelines**:

| Metric | Warning | Critical | Action |
|--------|---------|----------|--------|
| Worker utilization | 70% | 85% | Add worker |
| Scheduling latency p95 | 1s | 5s | Add worker or optimize |
| Raft commit latency p95 | 500ms | 1s | Check disk I/O, add manager |
| Manager storage | 8GB | 10GB | Clean up logs, expand disk |

---

## See Also

- [Observability Guide](observability.md) - Metrics and monitoring
- [E2E Validation](e2e-validation.md) - Testing procedures
- [Performance Benchmarking](performance-benchmarking.md) - Performance testing
- [Troubleshooting Guide](troubleshooting.md) - Common issues
