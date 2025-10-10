# Raft POC - 3-Node Cluster

## Overview

This POC validates that `hashicorp/raft` can meet Warren's high availability requirements:
- Leader election
- Log replication
- Failover (< 10s)
- Snapshots

## Components

- `fsm.go` - Finite State Machine implementing simple key-value store
- `main.go` - Raft node implementation with bootstrap/join logic

## Running the POC

### Prerequisites

```bash
cd poc/raft
go mod download
```

### Start 3-Node Cluster

**Terminal 1 - Node 1 (Bootstrap as leader)**:
```bash
go run . -id node1 -addr 127.0.0.1:8001
```

**Terminal 2 - Node 2**:
```bash
go run . -id node2 -addr 127.0.0.1:8002 -join 127.0.0.1:8001
```

**Terminal 3 - Node 3**:
```bash
go run . -id node3 -addr 127.0.0.1:8003 -join 127.0.0.1:8001
```

### Manual Join (Production Approach)

Since this is a POC, nodes 2 and 3 need to be manually added to the cluster.

On Node 1 (leader), use the Raft API to add nodes:

```go
// Pseudo-code for production implementation
r.AddVoter(raft.ServerID("node2"), raft.ServerAddress("127.0.0.1:8002"), 0, 0)
r.AddVoter(raft.ServerID("node3"), raft.ServerAddress("127.0.0.1:8003"), 0, 0)
```

**Note**: In Warren's actual implementation, this will be handled by the manager's API server when workers join via `warren cluster join`.

## Test Scenarios

### 1. Leader Election

**Expected**: One node becomes leader, others become followers
**Result**: ✅ PASS / ❌ FAIL
**Time**: ___ seconds

**Observed**:
```
# Output from test
```

### 2. Log Replication

**Expected**: Write to leader, read from follower (same value)
**Test Steps**:
1. Apply command on leader: `set test-key=test-value`
2. Wait 1 second for replication
3. Read from follower FSM

**Result**: ✅ PASS / ❌ FAIL

**Observed**:
```
# Output showing replication
```

### 3. Leader Failover

**Expected**: Kill leader → new leader elected < 10s → cluster operational
**Test Steps**:
1. Identify current leader (e.g., node1)
2. Kill leader process (Ctrl+C)
3. Observe followers
4. Measure time to new leader election

**Result**: ✅ PASS / ❌ FAIL
**Failover Time**: ___ seconds

**Observed**:
```
# Output showing election
```

### 4. Snapshot & Restore

**Expected**: Create snapshot after 10K writes, restore on restart
**Test Steps**:
1. Modify main.go to write 10,000 key-value pairs
2. Wait for snapshot (check `data/snapshots/` directory)
3. Restart node
4. Verify data restored from snapshot

**Result**: ✅ PASS / ❌ FAIL

**Observed**:
```
# Snapshot file details
# Restoration log output
```

## Performance Measurements

### Write Latency

**Test**: 1000 sequential writes to leader
**Expected**: < 10ms per write (p95)

**Results**:
- Mean: ___ ms
- P50: ___ ms
- P95: ___ ms
- P99: ___ ms

### Replication Lag

**Test**: Write to leader, measure time until follower has entry
**Expected**: < 100ms

**Results**:
- Mean: ___ ms
- Max: ___ ms

## Conclusions

### Success Criteria

- [ ] 3-node cluster forms correctly
- [ ] One node becomes leader
- [ ] Log replication working (writes propagate to followers)
- [ ] Leader failover < 10s
- [ ] Snapshots created and restored successfully
- [ ] Write latency acceptable (< 10ms p95)
- [ ] Replication lag acceptable (< 100ms)

### Go/No-Go Decision

**Decision**: ✅ GO / ❌ NO-GO

**Rationale**:
```
# Summary of why Raft meets (or doesn't meet) Warren's requirements
```

### Issues Discovered

```
# List any problems, workarounds, or concerns
```

### Recommendations for Warren Implementation

```
# Lessons learned, configuration tuning, best practices
```

## Next Steps

If GO:
- [ ] Proceed to containerd POC
- [ ] Document Raft configuration for Warren (timeouts, quorum size)
- [ ] Design manager join workflow (how workers get added as voters)

If NO-GO:
- [ ] Investigate alternatives (etcd, Consul, custom Raft)
- [ ] Document blockers and mitigation strategies
