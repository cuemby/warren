# ADR-001: Use Raft Consensus for Multi-Manager HA

**Status**: Accepted
**Date**: 2025-10-09
**Decision Makers**: Warren Engineering Team
**Related**: [PRD](../../specs/prd.md), [Tech Spec](../../specs/tech.md)

## Context

Warren requires high availability through multi-manager clustering. When multiple managers exist, they must agree on cluster state (nodes, services, tasks) to prevent split-brain scenarios. We need a consensus algorithm that:

- Tolerates minority failure (e.g., 1 of 3 managers can fail)
- Ensures strong consistency (no conflicting state)
- Supports leader election (one manager handles writes)
- Persists state durably (survives crashes)
- Integrates well with Go (Warren's implementation language)

## Decision

**We will use the Raft consensus algorithm via the `hashicorp/raft` Go library.**

Raft will manage:
- Leader election among manager nodes
- Log replication of state changes (service create/update/delete, node joins, etc.)
- State machine (FSM) applying log entries to cluster state
- Snapshotting for compact state storage

## Alternatives Considered

### 1. **etcd Cluster**

**Pros**:
- Battle-tested in production (Kubernetes control plane)
- Strong consistency guarantees
- Rich client libraries

**Cons**:
- **External dependency**: Requires separate etcd cluster process
- **Violates single-binary principle**: Warren goal is zero external deps
- Operational complexity: Another system to manage, monitor, upgrade
- Resource overhead: etcd runs as separate process (memory, ports)

**Verdict**: ❌ Rejected - External dependency conflicts with Warren's core principle

### 2. **Consul**

**Pros**:
- Built-in service mesh features
- Strong consistency (Raft-based)
- Multi-datacenter support

**Cons**:
- **External dependency**: Separate Consul agent required
- **Overkill**: Service mesh features unnecessary for Warren (we build our own)
- Heavier weight than needed
- Additional operational burden

**Verdict**: ❌ Rejected - External dependency, unnecessary features

### 3. **Custom Raft Implementation**

**Pros**:
- Full control over implementation
- Tailored to Warren's exact needs

**Cons**:
- **Enormous engineering effort**: Raft is complex to implement correctly
- **High risk**: Distributed systems bugs are subtle and catastrophic
- **Time to market**: Months to implement and test rigorously
- **Maintenance burden**: Ongoing bug fixes and optimizations

**Verdict**: ❌ Rejected - Too risky, not core differentiation for Warren

### 4. **Paxos**

**Pros**:
- Theoretically proven consensus
- Used in some production systems (Google Chubby)

**Cons**:
- **More complex than Raft**: Harder to understand and implement
- **Fewer Go libraries**: Limited production-ready options
- **Raft is modern equivalent**: Raft designed as understandable Paxos

**Verdict**: ❌ Rejected - Raft is more practical choice

## Rationale for Raft (hashicorp/raft)

**Why Raft**:
1. **Well-understood**: Designed for understandability (vs Paxos)
2. **Strong consistency**: Linearizable reads/writes essential for orchestration
3. **Leader-based**: Simplifies write path (all writes go to leader)
4. **Proven fault tolerance**: Handles network partitions, node failures gracefully
5. **Efficient snapshots**: Compact state without full log replay

**Why hashicorp/raft Library**:
1. **Production-proven**: Used in Consul, Nomad, Vault (years in production)
2. **Pure Go**: No C dependencies, compiles to single binary
3. **Pluggable storage**: BoltDB integration via `raft-boltdb`
4. **Well-maintained**: Active development, security patches
5. **Documentation**: Comprehensive examples and guides

## Implementation Approach

### Manager Architecture

```
┌─────────────────────────────────┐
│     Warren Manager Node         │
│                                 │
│  ┌──────────────────────────┐  │
│  │   Raft Layer             │  │
│  │  - Leader Election       │  │
│  │  - Log Replication       │  │
│  │  - State Machine (FSM)   │  │
│  └──────────────────────────┘  │
│             ↓                   │
│  ┌──────────────────────────┐  │
│  │   BoltDB Store           │  │
│  │  - Raft log              │  │
│  │  - Stable storage        │  │
│  │  - Snapshots             │  │
│  └──────────────────────────┘  │
└─────────────────────────────────┘
```

### State Changes via Raft

All cluster state mutations flow through Raft:

```go
// User creates service
warren service create web --image nginx

// API server (on manager)
1. Receives gRPC request
2. Creates Raft command: {Op: "create_service", Service: {...}}
3. Submits to Raft: raft.Apply(command, timeout)
4. Raft replicates to majority
5. FSM applies command → state updated
6. Response returned to user
```

### Quorum Sizes

- **1 manager**: No HA (development only)
- **3 managers**: Tolerates 1 failure (recommended minimum)
- **5 managers**: Tolerates 2 failures (large deployments)

**Formula**: Quorum = (N/2) + 1
**Failure tolerance**: (N-1) / 2

## Consequences

### Positive

- ✅ **Zero external dependencies**: Raft embedded in Warren binary
- ✅ **Strong consistency**: Cluster state always coherent across managers
- ✅ **Automatic failover**: New leader elected in < 10s on leader failure
- ✅ **Proven reliability**: hashicorp/raft battle-tested in production
- ✅ **Go-native**: Clean integration, single binary distribution
- ✅ **Snapshot support**: Efficient state compaction (10K log entries → snapshot)

### Negative

- ⚠️ **Odd-number requirement**: Must run 1, 3, or 5 managers (not 2 or 4)
- ⚠️ **Write latency**: All writes go through leader (network round-trip for replication)
- ⚠️ **Learning curve**: Team must understand Raft concepts (leader, follower, candidate)
- ⚠️ **Partition sensitivity**: Minority partition cannot make progress (by design)

### Neutral

- **Leader-based**: Reads can be served from followers (with potential stale data) or leader (consistent but slower)
- **Network-dependent**: Requires stable network for quorum communication
- **State size**: Full state replicated to all managers (not sharded)

## Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Raft library bugs | Low | High | Use well-tested hashicorp/raft, comprehensive integration tests |
| Split-brain due to network partition | Medium | Critical | Raft prevents by design (minority cannot commit), test with chaos engineering |
| Performance issues with large state | Low | Medium | Implement snapshots, benchmark with 10K+ services |
| Operational complexity (quorum management) | Medium | Low | Document clearly, provide `warren cluster health` command |

## Performance Expectations

Based on hashicorp/raft benchmarks and Consul production data:

- **Write throughput**: 1,000-10,000 ops/sec (depends on network latency)
- **Replication latency**: 10-100ms (depends on quorum distance)
- **Leader election**: < 10s (configurable timeout)
- **Snapshot time**: < 5s for 10K services

**Warren-specific tuning**:
```go
raft.DefaultConfig()
config.HeartbeatTimeout = 1 * time.Second
config.ElectionTimeout = 1 * time.Second
config.LeaderLeaseTimeout = 500 * time.Millisecond
config.SnapshotInterval = 120 * time.Second
config.SnapshotThreshold = 10000
```

## Validation

See [poc/raft/](../../poc/raft/) for proof-of-concept implementation demonstrating:
- 3-node cluster formation
- Leader election
- Log replication
- Failover timing
- Snapshot creation/restoration

**Acceptance criteria**:
- ✅ Leader elected in < 10s
- ✅ Writes replicate to followers
- ✅ Failover successful
- ✅ Snapshots compress state effectively

## References

- [Raft Paper](https://raft.github.io/raft.pdf) - Original Raft consensus algorithm
- [hashicorp/raft](https://github.com/hashicorp/raft) - Go implementation
- [Raft Visualization](https://raft.github.io/) - Interactive Raft demo
- [Consul Architecture](https://www.consul.io/docs/architecture) - Production Raft usage

## Status History

- **2025-10-09**: Accepted - POC validated, aligns with zero-dependency principle
