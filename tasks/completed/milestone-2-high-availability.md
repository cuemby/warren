## Milestone 2: High Availability

**Goal**: Multi-manager Raft cluster with edge resilience

**Priority**: [CRITICAL]
**Estimated Effort**: 2-3 weeks
**Status**: ✅ **COMPLETE** (2025-10-10)

**Note**: Simplified scope - focusing on core HA (multi-manager + containerd). WireGuard/DNS deferred to M3, worker autonomy and rolling updates optional for M2.

### Phase 2.1: Containerd Integration ✅ COMPLETE

- [x] **Real container runtime integration**
  - Created pkg/runtime/containerd.go (330 lines)
  - Full container lifecycle: Pull, Create, Start, Stop, Delete
  - Graceful shutdown (SIGTERM → SIGKILL)
  - Warren-specific namespace isolation
  - Test: Integration tests in test/integration/containerd_test.go
  - Commit: e7f5c7e

- [x] **Worker real execution**
  - Updated pkg/worker/worker.go to use containerd
  - Replaced simulated execution with real containers
  - Container status monitoring (5s interval)
  - Test: Manual test plan in docs/containerd-integration-test.md
  - Commit: e7f5c7e

### Phase 2.2: Multi-Manager Raft ✅ COMPLETE

- [x] **Raft cluster formation**
  - Implemented Manager.Join() method
  - Implemented Manager.AddVoter() for Raft expansion
  - Support 3 or 5 manager quorum
  - Token-based secure joining (64-char, 24h expiration)
  - Created pkg/manager/token.go for token management
  - Commit: 4fd6afe

- [x] **Leader election & failover**
  - Leader forwarding implemented (ensureLeader helper)
  - All write operations check leadership
  - Automatic Raft re-election on leader failure
  - Commit: 4fd6afe

- [x] **State replication**
  - Raft handles consistent state across managers
  - Reads from any node, writes to leader only
  - Commit: 4fd6afe

- [x] **gRPC API for cluster operations**
  - GenerateJoinToken RPC
  - JoinCluster RPC
  - GetClusterInfo RPC
  - Updated api/proto/warren.proto (28 methods total)
  - Commit: 4fd6afe

- [x] **CLI commands for cluster management**
  - `warren cluster join-token [worker|manager]`
  - `warren cluster info`
  - `warren manager join`
  - Client library methods added
  - Commit: 6ffc98d

### Phase 2.3: Testing & Validation ✅ COMPLETE

- [x] **Lima testing infrastructure**
  - Created Lima VM templates (warren.yaml)
  - Setup scripts for 5 VMs (3 managers + 2 workers)
  - Test scripts: test-cluster.sh, test-failover.sh
  - Full documentation in docs/testing/lima-setup.md
  - Commit: 292f721

- [x] **Test 3-manager cluster formation**
  - ✅ First manager bootstrap
  - ✅ Token generation with retry logic
  - ✅ Second and third managers join
  - ✅ Raft quorum verified (3 voters)
  - ✅ 2 workers registered
  - ✅ Service deployment tested (nginx with 2 replicas)
  - Test script: test/lima/test-cluster.sh - **PASSING**
  - Commit: 292f721

- [x] **Test leader failover**
  - ✅ Leader identification working
  - ✅ Leader kill triggers election
  - ⚠️ New leader election working but slower than 10s target
  - Note: Needs Raft config tuning (election timeout)
  - Test script: test/lima/test-failover.sh - **PARTIAL**

### Deferred Features (Optional for M2)

**Worker Autonomy (Partition Tolerance)** - Deferred to M3 or optional:
- Local state caching on workers
- Autonomous operation during partition
- Reconciliation on rejoin

**Advanced Networking** - Deferred to M3:
- Automatic WireGuard mesh
- DNS service

**Rolling Updates** - Optional for M2:
- Rolling update strategy
- Rollback functionality

### Milestone 2 Acceptance Criteria (Revised)

**Core HA (Required)**:

- [x] Multi-manager Raft implementation complete ✓
- [x] Token-based secure joining ✓
- [x] Leader forwarding for writes ✓
- [x] CLI commands for cluster management ✓
- [x] Containerd integration complete ✓
- [x] 3-manager cluster tested and operational ✓
- [x] Leader failover tested (works, needs tuning for <10s) ⚠️
- [x] End-to-end multi-manager workflow validated ✓

**Status**: 🎉 **MILESTONE 2 COMPLETE** 🎉

**Known Issues**:

- Leader failover works but takes >15s (target: <10s) - needs Raft election timeout tuning

**Optional (Nice to Have)**:


- [ ] Worker autonomy during partition
- [ ] Rolling updates with zero downtime
- [ ] Rollback functional

---

