# High Availability in Warren

High Availability (HA) ensures your cluster continues operating even when components fail. This guide explains Warren's HA features and best practices.

## Overview

Warren achieves high availability through:

1. **Multi-Manager Raft Cluster** - Distributed consensus for control plane
2. **Automatic Failover** - Leader re-election on manager failure
3. **Task Rescheduling** - Automatic recovery from worker/task failures
4. **Partition Tolerance** - Operations continue during network splits (majority partition)

## Architecture for HA

### Single Manager (Development)

```
┌──────────────┐
│  Manager 1   │ ← Single point of failure
└──────┬───────┘
       │
   ┌───┴───┐
   │       │
Worker-1 Worker-2
```

**Characteristics:**
- ❌ Manager failure = cluster down
- ✅ Simple setup
- ✅ Low resource usage
- **Use Case**: Development, testing

### Three Managers (Production)

```
┌─────────────┐  ┌─────────────┐  ┌─────────────┐
│  Manager 1  │  │  Manager 2  │  │  Manager 3  │
│   (Leader)  │◄─┤  (Follower) │◄─┤  (Follower) │
└──────┬──────┘  └──────┬──────┘  └──────┬──────┘
       │                │                │
       └────────────────┴────────────────┘
                        │
       ┌────────────────┴────────────────┐
       │                                  │
    Worker-1                          Worker-2
```

**Characteristics:**
- ✅ Tolerates 1 manager failure
- ✅ Automatic leader election
- ✅ No downtime during failover (~2-3s)
- **Use Case**: Production, small clusters

**Quorum**: 2 out of 3 managers required (majority)

### Five Managers (Large Production)

```
     Manager-1 (Leader)
         ↙ ↓ ↓ ↘
    Mgr-2 Mgr-3 Mgr-4 Mgr-5
```

**Characteristics:**
- ✅ Tolerates 2 manager failures
- ✅ Higher redundancy
- ⚠️ More network overhead (5-way replication)
- **Use Case**: Critical production, large clusters

**Quorum**: 3 out of 5 managers required (majority)

## Raft Consensus

Warren uses [Hashicorp Raft](https://github.com/hashicorp/raft) for distributed consensus.

### How Raft Works

**Leader Election:**
```
1. Managers start in Follower state
2. If no heartbeat from leader (500ms timeout)
3. Followers become Candidates
4. Candidates request votes from peers
5. First to get majority becomes Leader
6. Leader sends heartbeats to maintain leadership
```

**Election Time**: ~500ms - 1s

**Log Replication:**
```
1. Client sends request to any manager
2. If not leader, forward to leader
3. Leader appends to Raft log
4. Leader replicates to followers
5. Once majority acknowledges, commit
6. Leader responds to client
```

**Commit Latency**: ~10-50ms (3 managers, LAN)

### Raft Guarantees

- **Strong Consistency** - All managers see same state
- **Durability** - Committed entries survive failures
- **Availability** - Operates with majority available
- **Partition Tolerance** - Minority partitions stop processing writes

## Setting Up Multi-Manager Cluster

### Step 1: Initialize First Manager

```bash
# On manager-1 (192.168.1.10)
sudo warren cluster init \
  --advertise-addr 192.168.1.10:8080 \
  --listen-addr 0.0.0.0:8080

# Output:
# ✓ Raft consensus initialized (single-node)
# ✓ Manager started (Node ID: manager-1-abc123)
# ✓ API server listening on 192.168.1.10:8080
# ✓ Metrics available at http://192.168.1.10:9090/metrics
#
# Cluster initialized successfully!
# To add managers, generate join token:
#   warren cluster join-token manager
```

### Step 2: Generate Manager Join Token

```bash
# On manager-1
warren cluster join-token manager --manager 192.168.1.10:8080

# Output:
# Join Token (expires in 24h):
# SWMTKN-1-5k7j9h3f2d1s9a7k5l3n1m9p7r5t3v1x9z7b5d3f1h9j7k5
#
# On other nodes, run:
# warren manager join \
#   --token SWMTKN-1-5k7j9h3f2d1s9a7k5l3n1m9p7r5t3v1x9z7b5d3f1h9j7k5 \
#   --manager 192.168.1.10:8080 \
#   --advertise-addr <this-node-ip>:8080
```

### Step 3: Join Second Manager

```bash
# On manager-2 (192.168.1.11)
sudo warren manager join \
  --token SWMTKN-1-5k7j9h3f2d1s9a7k5l3n1m9p7r5t3v1x9z7b5d3f1h9j7k5 \
  --manager 192.168.1.10:8080 \
  --advertise-addr 192.168.1.11:8080

# Output:
# ✓ Connected to cluster leader at 192.168.1.10:8080
# ✓ Raft joined as follower
# ✓ Manager started (Node ID: manager-2-def456)
# ✓ API server listening on 192.168.1.11:8080
#
# Manager joined successfully!
# Raft quorum: 2/2 managers (needs 2 for majority)
```

### Step 4: Join Third Manager

```bash
# On manager-3 (192.168.1.12)
sudo warren manager join \
  --token SWMTKN-1-5k7j9h3f2d1s9a7k5l3n1m9p7r5t3v1x9z7b5d3f1h9j7k5 \
  --manager 192.168.1.10:8080 \
  --advertise-addr 192.168.1.12:8080

# Output:
# ✓ Connected to cluster leader at 192.168.1.10:8080
# ✓ Raft joined as follower
# ✓ Manager started (Node ID: manager-3-ghi789)
# ✓ API server listening on 192.168.1.12:8080
#
# Manager joined successfully!
# Raft quorum: 3/3 managers (needs 2 for majority)
```

### Step 5: Verify Cluster

```bash
# Check cluster info
warren cluster info --manager 192.168.1.10:8080

# Output:
# Cluster ID: cluster-abc123
# Raft Quorum: 3 managers (needs 2 for majority)
# Leader: manager-1 (192.168.1.10:8080)
#
# Managers:
#   manager-1  192.168.1.10:8080  Leader    ready
#   manager-2  192.168.1.11:8080  Follower  ready
#   manager-3  192.168.1.12:8080  Follower  ready
#
# Workers: 0
```

## Failover Scenarios

### Manager Failure (Leader)

**Scenario**: Leader manager crashes or becomes unreachable.

```
Before:
  Manager-1 (Leader) ✓
  Manager-2 (Follower) ✓
  Manager-3 (Follower) ✓

Manager-1 crashes:
  Manager-1 (Leader) ✗ [CRASHED]
  Manager-2 (Follower) ✓
  Manager-3 (Follower) ✓

Election (500ms-1s):
  Manager-2 (Candidate) → Requests votes
  Manager-3 (Candidate) → Requests votes
  First to get majority (2 votes) wins

After:
  Manager-1 ✗ [DOWN]
  Manager-2 (Leader) ✓ [ELECTED]
  Manager-3 (Follower) ✓
```

**Timeline:**
1. **T+0ms**: Manager-1 crashes
2. **T+500ms**: Managers 2 & 3 detect missing heartbeats
3. **T+500-1000ms**: Election happens
4. **T+1000ms**: Manager-2 elected as new leader
5. **T+1100ms**: Cluster operational, clients retry requests

**Total Downtime**: ~2-3 seconds

**Client Impact**:
- Requests in-flight during failover may fail (retry required)
- New requests succeed after new leader elected

### Manager Failure (Follower)

**Scenario**: Follower manager crashes.

```
Before:
  Manager-1 (Leader) ✓
  Manager-2 (Follower) ✓
  Manager-3 (Follower) ✓

Manager-3 crashes:
  Manager-1 (Leader) ✓
  Manager-2 (Follower) ✓
  Manager-3 (Follower) ✗ [CRASHED]

After (no election needed):
  Manager-1 (Leader) ✓
  Manager-2 (Follower) ✓
  Manager-3 ✗ [DOWN]

Quorum: 2/3 (still have majority)
```

**Impact**: None - cluster continues operating normally.

**Writes**: Continue (2 of 3 managers = majority)

**Reads**: Continue from any manager

### Worker Failure

**Scenario**: Worker node crashes or becomes unreachable.

```
Detection:
1. Worker stops sending heartbeats
2. Manager marks worker unhealthy after 30s timeout
3. Reconciler detects failed tasks
4. Reconciler reschedules tasks to other workers

Timeline:
  T+0s:   Worker-1 crashes
  T+30s:  Manager marks worker-1 down
  T+31s:  Reconciler marks all tasks failed
  T+32s:  Scheduler assigns tasks to worker-2, worker-3
  T+40s:  New containers running on other workers
```

**Recovery Time**: ~30-40 seconds

**Service Impact**:
- Tasks on failed worker become unavailable
- Remaining tasks (other workers) continue serving traffic
- New tasks scheduled within 30-40s

### Task Failure

**Scenario**: Container crashes or exits.

```
Detection:
1. Worker detects container exit
2. Worker reports task failed to manager
3. Reconciler reschedules task
4. New container starts on same or different worker

Timeline:
  T+0s:   Container crashes
  T+1s:   Worker detects exit
  T+2s:   Manager notified
  T+3s:   Scheduler assigns new task
  T+10s:  New container running
```

**Recovery Time**: ~10 seconds

### Network Partition

**Scenario**: Network splits cluster into two partitions.

```
Before:
  Partition A: Manager-1 (Leader), Manager-2, Worker-1
  Partition B: Manager-3, Worker-2

After partition:
  Partition A (3 nodes, 2 managers = majority):
    - Manager-1 (Leader) continues
    - Manager-2 (Follower)
    - Worker-1 continues operating
    - Writes succeed ✓

  Partition B (2 nodes, 1 manager = minority):
    - Manager-3 (Follower)
    - Worker-2 continues operating locally
    - Writes blocked ✗ (no quorum)
    - Reads succeed ✓ (stale)
```

**Split-Brain Prevention**: Raft's quorum requirement prevents split-brain.

**Partition Heals**:
```
1. Network connectivity restored
2. Manager-3 detects leader (Manager-1)
3. Manager-3 replicates missed log entries
4. Partition B catches up
5. Cluster fully operational
```

## Worker Autonomy (Partition Tolerance)

Workers can operate autonomously during partitions:

**During Partition:**
- Workers cache task state locally
- Existing containers continue running
- Health checks continue locally
- Failed tasks restarted locally (best-effort)

**Limitations:**
- No new task assignments (requires manager)
- No service scaling
- No new services deployed

**After Reconnection:**
- Worker reports current state
- Manager reconciles differences
- Tasks aligned with desired state

## High Availability Best Practices

### 1. Use 3 Managers for Production

```bash
# Good: 3 managers (tolerates 1 failure)
Managers: 3
Quorum: 2
Fault Tolerance: 1 manager failure

# Avoid: 2 managers (no fault tolerance)
Managers: 2
Quorum: 2
Fault Tolerance: 0 manager failures (any failure = no quorum)
```

**Never use 2 managers** - requires both for quorum, no fault tolerance.

### 2. Distribute Managers Across Failure Domains

```bash
# Good: Managers in different racks/datacenters
Manager-1: Rack A
Manager-2: Rack B
Manager-3: Rack C

# Bad: All managers in same rack
Manager-1: Rack A
Manager-2: Rack A
Manager-3: Rack A
# Risk: Rack power failure = entire cluster down
```

### 3. Run Managers on Dedicated Nodes

```bash
# Good: Manager-only nodes
Manager-1: No worker role
Manager-2: No worker role
Manager-3: No worker role

# Avoid: Combined manager+worker
# Workload can affect control plane performance
```

### 4. Use Multiple Replicas for Services

```bash
# Good: Multiple replicas (survives worker failure)
warren service create api --replicas 5

# Bad: Single replica (no redundancy)
warren service create api --replicas 1
```

### 5. Monitor Manager Health

```bash
# Check Raft quorum
warren cluster info --manager 127.0.0.1:8080

# Check manager metrics
curl http://manager:9090/metrics | grep raft_

# Alert on:
# - Manager down
# - No leader
# - Quorum lost
# - High log replication lag
```

### 6. Test Failover Regularly

```bash
# Chaos testing script
# 1. Kill leader manager
# 2. Verify new leader elected
# 3. Verify services continue
# 4. Verify writes succeed
# 5. Restart failed manager
# 6. Verify rejoins cluster
```

### 7. Configure Appropriate Timeouts

Current (optimized for edge/LAN):
```
HeartbeatTimeout:    500ms
ElectionTimeout:     500ms
LeaderLeaseTimeout:  250ms
```

Adjust for WAN deployments:
```bash
# Future: Custom Raft timeouts (M6)
warren cluster init \
  --raft-heartbeat-timeout 1s \
  --raft-election-timeout 1s
```

### 8. Use Load Balancer for API Access

```bash
# Instead of hardcoding manager IP:
export WARREN_MANAGER=192.168.1.10:8080

# Use load balancer:
export WARREN_MANAGER=lb.warren.local:8080

# Load balancer distributes across all managers
# Automatic failover if manager down
```

## Backup and Disaster Recovery

### Backup Manager State

```bash
# Stop manager (on each manager node)
sudo systemctl stop warren-manager

# Backup BoltDB
sudo tar czf warren-backup-$(date +%Y%m%d).tar.gz \
  /var/lib/warren/data/warren.db

# Start manager
sudo systemctl start warren-manager

# Copy to safe location
scp warren-backup-*.tar.gz backup-server:/backups/
```

**Frequency**: Daily or after significant changes

### Restore from Backup

**Complete cluster loss:**

```bash
# On new manager-1
sudo systemctl stop warren-manager
sudo rm -rf /var/lib/warren/data/
sudo tar xzf warren-backup-20251010.tar.gz -C /
sudo warren cluster init --restore
```

**Single manager loss (3-manager cluster):**

```bash
# No restore needed - just re-join
# On replacement manager
sudo warren manager join \
  --token <new-token> \
  --manager <working-manager-ip> \
  --advertise-addr <this-node-ip>

# Raft replicates state automatically
```

### Testing Disaster Recovery

```bash
# 1. Backup cluster state
# 2. Destroy entire cluster (all managers)
# 3. Restore from backup
# 4. Verify services and data intact
# 5. Re-join workers
# 6. Verify full functionality
```

## Monitoring HA Health

### Key Metrics

```bash
# Raft metrics (Prometheus format)
curl http://manager:9090/metrics | grep raft_

# Important metrics:
# - raft_leader (1 = this node is leader, 0 = follower)
# - raft_peers (number of peers in cluster)
# - raft_commit_index (log index committed)
# - raft_applied_index (log index applied)
# - raft_last_contact (time since last leader contact)
```

### Health Checks

```bash
# Check cluster health
warren cluster info --manager <any-manager>

# Expected output:
# - Leader exists
# - All managers ready
# - Quorum present (e.g., 3/3 managers)
```

### Alerts to Configure

1. **Manager Down** - Any manager unreachable
2. **No Leader** - Cluster has no leader (election in progress)
3. **Quorum Lost** - Less than majority of managers available
4. **High Log Lag** - Follower replication falling behind
5. **Frequent Elections** - Leader instability (network issues)

## Performance Tuning for HA

### Raft Configuration

Current settings (edge/LAN optimized):

```go
HeartbeatTimeout:    500ms   // How often leader sends heartbeats
ElectionTimeout:     500ms   // How long followers wait before election
CommitTimeout:       50ms    // Max time to wait for commit
LeaderLeaseTimeout:  250ms   // Leader lease duration
```

**Trade-offs:**

- **Lower timeouts** = Faster failover, more network traffic
- **Higher timeouts** = Slower failover, less network traffic

### Network Requirements

- **Latency**: < 50ms RTT (LAN/edge deployments)
- **Bandwidth**: 1 Mbps per manager (heartbeats + log replication)
- **Packet Loss**: < 1%

For WAN deployments, increase timeouts to accommodate latency.

## Comparison to Other Orchestrators

| Feature | Warren | Docker Swarm | Kubernetes |
|---------|--------|--------------|------------|
| **Consensus** | Raft | Raft | etcd (Raft) |
| **Quorum** | 2/3 or 3/5 | 2/3 | etcd quorum |
| **Failover Time** | 2-3s | 10-15s | 30-60s |
| **Worker Autonomy** | Partial | No | No |
| **Setup Complexity** | Low | Low | High |
| **Backup** | Single file | Single file | Multi-component |

## Next Steps

- **[Architecture](architecture.md)** - Overall system design
- **[Services](services.md)** - Resilient service deployment
- **[Troubleshooting](../troubleshooting.md)** - Debugging HA issues

---

**Questions?** See [Troubleshooting](../troubleshooting.md) or ask in [GitHub Discussions](https://github.com/cuemby/warren/discussions).
