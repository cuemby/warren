# ADR-004: Use BoltDB for Raft Storage

**Status**: Accepted
**Date**: 2025-10-09

## Decision

**Use BoltDB as the storage backend for Raft logs and stable storage**, via `hashicorp/raft-boltdb`.

## Context

Raft needs persistent storage for:
- **Log store**: Sequential log of all commands
- **Stable store**: Raft metadata (current term, voted for)
- **Snapshots**: Compacted state

Options:
1. **BoltDB** - Embedded key-value store
2. **SQLite** - Embedded SQL database
3. **BadgerDB** - LSM-tree KV store

## Rationale

**Chose BoltDB because**:

✅ **Raft integration**: `hashicorp/raft-boltdb` official adapter
✅ **Embedded**: No external database process
✅ **Single file**: Simple backup/restore
✅ **ACID**: Full transaction support
✅ **Pure Go**: Zero C dependencies
✅ **Battle-tested**: Used in etcd, Consul
✅ **B+tree**: Efficient range queries

## Alternatives Rejected

### SQLite
❌ CGO dependency (requires C compiler)
❌ Heavier than needed (SQL features unnecessary)
✓ More features, but Warren doesn't need SQL

### BadgerDB
❌ No official raft-badgerdb adapter
❌ LSM-tree more complex than needed
❌ Write amplification concerns
✓ Faster writes, but Raft log is append-mostly

## Implementation

```go
// Create BoltDB stores
logStore, _ := raftboltdb.NewBoltStore("/var/lib/warren/raft-log.db")
stableStore, _ := raftboltdb.NewBoltStore("/var/lib/warren/raft-stable.db")

// Use with Raft
raft.NewRaft(config, fsm, logStore, stableStore, snapshots, transport)
```

**Storage layout**:
```
/var/lib/warren/
├── raft-log.db      # Raft log entries
├── raft-stable.db   # Raft metadata
└── snapshots/       # State snapshots
    ├── 1-12345-*.snap
    └── 2-23456-*.snap
```

## Consequences

✅ Zero external database dependency
✅ Single-file simplicity (easy backup)
✅ ACID transactions (no corruption)
✅ Official Raft integration
⚠️ Single-threaded writes (Raft serializes anyway)
⚠️ Full replication (not sharded, acceptable for Warren's scale)

## Performance

- **Write throughput**: 10K-100K ops/sec (depends on fsync)
- **Read latency**: < 1ms (in-memory B+tree)
- **File size**: Grows with log, compacted by snapshots

**Tuning**:
```go
// Snapshot every 10K entries
config.SnapshotThreshold = 10000
config.SnapshotInterval = 120 * time.Second
```

## Validation

Validated in [poc/raft/](../../poc/raft/) - BoltDB handles Raft workload effectively.

**Status**: Accepted - Mature, proven choice
