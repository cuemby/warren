# Warren Terminology Refactor: Task → Container/Replica

**Date:** 2025-10-13
**Status:** 📋 PLANNING
**Priority:** [CRITICAL]
**Impact:** 🔥 BREAKING CHANGE
**Version:** Requires v2.0.0

---

## Executive Summary

Replace Warren's confusing "Task" and "Instance" terminology with clear, descriptive terms:
- **Container** = Single running container (replaces "Task")
- **Replica** = Used in scaling context ("3 replicas")
- **NO "Task" or "Instance"** anywhere

This is a **full, consistent replacement** across code, API, storage, CLI, and documentation.

---

## Why This Change?

### Current Problem

**"Task" is abstract Kubernetes/Swarm jargon:**
- ❌ Users don't know what a "task" is
- ❌ Requires explanation
- ❌ Not descriptive
- ❌ Inconsistent with Docker terminology

**Evidence from Warren's own code:**
```go
type Task struct {
    ContainerID string  // ← Warren calls it a container!
}
```

### New Terminology

**Container** (noun) = Single running container
- ✅ Clear and concrete
- ✅ Docker users understand immediately
- ✅ No explanation needed
- ✅ Matches Warren's internal implementation

**Replica** (noun) = Instance of a service, used in scaling context
- ✅ Kubernetes users understand
- ✅ Clear for "3 replicas of nginx"
- ✅ Less technical than "container"

### Usage Examples

**Before:**
- "Service has 3 tasks"
- "Task failed"
- "Schedule tasks to nodes"

**After:**
- "Service has 3 replicas" (scaling context)
- "Container failed" (runtime context)
- "Schedule containers to nodes"

---

## Scope Analysis

### 1. Core Code Changes

**Go Type Definitions:**
```
Type Changes:
- Task → Container
- TaskState → ContainerState
- TaskStatus → ContainerStatus
- TaskEvent → ContainerEvent

Files Affected:
- pkg/types/types.go (main definition)
- All packages importing types.Task
```

**Function Names (~50 functions):**
```
Examples:
- CreateTask() → CreateContainer()
- UpdateTaskStatus() → UpdateContainerStatus()
- ListTasks() → ListContainers()
- GetTask() → GetContainer()
- DeleteTask() → DeleteContainer()
- ScheduleTask() → ScheduleContainer()
- ReconcileTasks() → ReconcileContainers()
```

**Variables (~464 occurrences):**
```
Examples:
- task → container
- tasks → containers
- taskID → containerID
- taskState → containerState
```

**Files to Modify:**
- 45 Go source files (pkg/*/*.go, excluding doc.go and tests)
- 12 Test files (pkg/*/*_test.go)
- Total: ~57 Go files

**Key Packages:**
- pkg/types/ (core definitions)
- pkg/manager/ (18 occurrences)
- pkg/worker/ (56 occurrences)
- pkg/scheduler/ (36 occurrences)
- pkg/reconciler/
- pkg/api/
- pkg/storage/
- pkg/client/
- pkg/ingress/
- pkg/health/

### 2. API Changes (Protocol Buffers)

**Proto Message Changes:**
```protobuf
// Before
message Task { ... }
message TaskStatus { ... }
message TaskEvent { ... }
message UpdateTaskStatusRequest { ... }
message ListTasksRequest { ... }

// After
message Container { ... }
message ContainerStatus { ... }
message ContainerEvent { ... }
message UpdateContainerStatusRequest { ... }
message ListContainersRequest { ... }
```

**RPC Method Changes:**
```protobuf
// Before
rpc UpdateTaskStatus(UpdateTaskStatusRequest) returns (UpdateTaskStatusResponse);
rpc ListTasks(ListTasksRequest) returns (ListTasksResponse);
rpc GetTask(GetTaskRequest) returns (GetTaskResponse);
rpc WatchTasks(WatchTasksRequest) returns (stream TaskEvent);
rpc ReportTaskHealth(ReportTaskHealthRequest) returns (ReportTaskHealthResponse);

// After
rpc UpdateContainerStatus(UpdateContainerStatusRequest) returns (UpdateContainerStatusResponse);
rpc ListContainers(ListContainersRequest) returns (ListContainersResponse);
rpc GetContainer(GetContainerRequest) returns (GetContainerResponse);
rpc WatchContainers(WatchContainersRequest) returns (stream ContainerEvent);
rpc ReportContainerHealth(ReportContainerHealthRequest) returns (ReportContainerHealthResponse);
```

**Impact:** 🔥 **BREAKING - Wire format changes, clients must update**

### 3. Storage/Database Changes

**BoltDB Bucket Names:**
```go
// Before
bucketTasks = []byte("tasks")

// After
bucketContainers = []byte("containers")
```

**JSON Field Names:**
```json
// Before
{
  "task_id": "...",
  "task_state": "running",
  "tasks": [...]
}

// After
{
  "container_id": "...",
  "container_state": "running",
  "containers": [...]
}
```

**Impact:** 🔥 **BREAKING - Requires data migration**

### 4. CLI Changes

**Command Structure:**
```bash
# Before
warren task list
warren task inspect <task-id>
warren task logs <task-id>

# After
warren container list       # or just: warren ps
warren container inspect <container-id>
warren container logs <container-id>
```

**Aliases to Consider:**
```bash
warren ps            # Alias for: warren container list
warren inspect       # Alias for: warren container inspect
warren logs          # Alias for: warren container logs
```

**Impact:** 🔥 **BREAKING - Users' scripts break**

### 5. Documentation Changes

**User Documentation (~388 occurrences):**
- docs/*.md files
- Need search and replace with context-aware changes

**Package Documentation:**
- 20 pkg/*/doc.go files
- Already done in our recent docs commit
- Need updates

**Examples:**
- YAML example files (currently 0 occurrences)
- No changes needed

**Total Lines:** ~388+ to review and update

---

## Terminology Decision Matrix

| Context | Use "Container" | Use "Replica" |
|---------|----------------|---------------|
| **Runtime** | ✅ "Container is running" | ❌ |
| **Scaling** | ❌ | ✅ "Scale to 5 replicas" |
| **Health** | ✅ "Container health failed" | ❌ |
| **Scheduling** | ✅ "Schedule container to node" | ❌ |
| **Service Spec** | ❌ | ✅ "Service with 3 replicas" |
| **Load Balancing** | ✅ "Route to containers" | ✅ "3 replicas available" |
| **API/Code** | ✅ Always use Container | ⚠️ Only in service.Replicas field |

**Rule:** Use "container" everywhere except when talking about scaling ("replicas: 3")

---

## Breaking Changes Summary

| Component | Impact | Migration Required |
|-----------|--------|-------------------|
| **Go API** | 🔥 HIGH | Code must update imports |
| **gRPC API** | 🔥 HIGH | Clients must update proto |
| **Storage** | 🔥 HIGH | Data migration script |
| **CLI** | 🔥 HIGH | User scripts break |
| **Documentation** | ⚠️ MEDIUM | Links may break |

**Version Bump Required:** v1.1.0 → **v2.0.0**

---

## Migration Strategy

### Option 1: Big Bang (Recommended)

**Approach:** Change everything at once with v2.0.0 release

**Pros:**
- Clean, consistent from day one
- No confusing transition period
- Simpler codebase (no aliases)
- Warren is early enough (v1.1.0)

**Cons:**
- All users must update at once
- Scripts break
- Requires data migration

**Timeline:** 2-3 days of work

### Option 2: Gradual Migration (Not Recommended)

**Approach:** Support both terminologies for 2-3 releases

**Pros:**
- Smooth user transition
- Less immediate breakage

**Cons:**
- Confusing (task vs container)
- Complex code with aliases
- Longer migration period
- Still breaks eventually

**Why Not:** Defeats the purpose of consistent terminology

---

## Implementation Plan

### Phase 1: Preparation (4 hours)

**1.1 Create Migration Scripts**
- Data migration script (BoltDB: tasks → containers)
- Proto regeneration script
- Search/replace validation script

**1.2 Create Feature Branch**
```bash
git checkout -b refactor/container-terminology
```

**1.3 Document Breaking Changes**
- CHANGELOG.md entry
- Migration guide (docs/migration/v1-to-v2.md)
- Deprecation notices

### Phase 2: Core Code Changes (8 hours)

**2.1 Update Type Definitions** (1 hour)
```
File: pkg/types/types.go
Changes:
- type Task → type Container
- type TaskState → type ContainerState
- type TaskStatus → type ContainerStatus
- All related types and constants
```

**2.2 Update All Package Code** (4 hours)
```
Packages (in order):
1. pkg/types/        (core)
2. pkg/storage/      (persistence)
3. pkg/manager/      (orchestration)
4. pkg/scheduler/    (scheduling)
5. pkg/reconciler/   (reconciliation)
6. pkg/worker/       (execution)
7. pkg/api/          (API server)
8. pkg/client/       (Go client)
9. pkg/ingress/      (load balancing)
10. pkg/health/      (health checks)
11. Other packages
```

**2.3 Update Protocol Buffers** (1 hour)
```
File: api/proto/warren.proto
- Rename all Task messages to Container
- Rename all RPC methods
- Regenerate Go code: make proto
```

**2.4 Update Storage Layer** (1 hour)
```
File: pkg/storage/boltdb.go
- Rename bucket: tasks → containers
- Create migration function
- Update all queries
```

**2.5 Update Tests** (1 hour)
```
Files: pkg/*/*_test.go (12 files)
- Update all test names
- Update mock data
- Update assertions
```

### Phase 3: CLI Changes (2 hours)

**3.1 Update CLI Commands** (1 hour)
```
File: cmd/warren/main.go
Commands to update:
- warren task → warren container
- Add aliases: warren ps, warren inspect, warren logs
```

**3.2 Update CLI Output** (1 hour)
```
- Update table headers
- Update status messages
- Update help text
```

### Phase 4: Documentation Changes (4 hours)

**4.1 Update Package Documentation** (2 hours)
```
Files: pkg/*/doc.go (20 files)
- Search/replace with context
- Review and validate
- Update code examples
```

**4.2 Update User Documentation** (2 hours)
```
Files: docs/*.md (~388 occurrences)
- Search/replace with context
- Update command examples
- Update screenshots/diagrams
- Update migration guides
```

### Phase 5: Data Migration (2 hours)

**5.1 Create Migration Tool** (1 hour)
```
File: cmd/warren-migrate/main.go
Function:
- Read old BoltDB with "tasks" bucket
- Create new "containers" bucket
- Copy and transform all data
- Validate migration
```

**5.2 Test Migration** (1 hour)
```
- Test on sample data
- Test on large datasets
- Verify data integrity
- Test rollback procedure
```

### Phase 6: Testing & Validation (4 hours)

**6.1 Unit Tests** (1 hour)
```bash
go test ./... -v
# All tests must pass
```

**6.2 Integration Tests** (1 hour)
```bash
# Run integration test suite
go test ./test/integration/... -v
```

**6.3 E2E Tests** (1 hour)
```bash
# Full cluster test
./test/lima/test-cluster.sh
./test/lima/test-failover.sh
```

**6.4 Manual Validation** (1 hour)
```bash
# Create cluster
warren cluster init
# Deploy service
warren service create nginx --image nginx --replicas 3
# List containers
warren container list
# Check health
warren container inspect <id>
```

### Phase 7: Release Preparation (2 hours)

**7.1 Update Version** (15 min)
```
- Version bump: v1.1.0 → v2.0.0
- Update version constants
- Update README badges
```

**7.2 Update CHANGELOG** (30 min)
```markdown
## [2.0.0] - 2025-10-XX

### BREAKING CHANGES

- Renamed "Task" to "Container" throughout codebase
- Renamed "Instance" to "Container" in user-facing documentation
- Changed API endpoints: /tasks → /containers
- Changed CLI commands: warren task → warren container
- Changed database schema: requires migration

### Migration Guide

See docs/migration/v1-to-v2.md for complete migration instructions.
```

**7.3 Create Migration Guide** (45 min)
```
File: docs/migration/v1-to-v2.md
Sections:
- Breaking changes summary
- Code migration (Go clients)
- CLI migration (scripts)
- Data migration (database)
- Troubleshooting
```

**7.4 Update README** (30 min)
```
- Update terminology throughout
- Update command examples
- Add migration notice
```

---

## Total Effort Estimate

| Phase | Hours | Days (8h) |
|-------|-------|-----------|
| 1. Preparation | 4 | 0.5 |
| 2. Core Code | 8 | 1.0 |
| 3. CLI | 2 | 0.25 |
| 4. Documentation | 4 | 0.5 |
| 5. Data Migration | 2 | 0.25 |
| 6. Testing | 4 | 0.5 |
| 7. Release Prep | 2 | 0.25 |
| **TOTAL** | **26** | **3.25** |

**Realistic Timeline:** 4-5 days (including buffer)

---

## Risk Assessment

### Risk 1: Data Migration Failures

**Risk:** BoltDB migration corrupts data or fails

**Mitigation:**
- Backup before migration
- Validate after migration
- Test on sample data first
- Provide rollback procedure

### Risk 2: Breaking External Integrations

**Risk:** External tools/scripts break

**Mitigation:**
- Clear migration guide
- Announce breaking change early
- Provide migration examples
- Consider deprecation warnings (if gradual)

### Risk 3: Incomplete Replacement

**Risk:** Miss some "task" occurrences

**Mitigation:**
- Comprehensive grep audit (done)
- Automated search/replace validation
- Code review checklist
- Integration tests

### Risk 4: Proto Version Conflicts

**Risk:** gRPC clients have version mismatch

**Mitigation:**
- Version proto package
- Document wire format changes
- Provide proto file for external clients

---

## Validation Checklist

Before merging, verify:

### Code
- [ ] All "Task" renamed to "Container" in types
- [ ] All "task" variables renamed to "container"
- [ ] All functions with "Task" renamed
- [ ] Proto file updated and regenerated
- [ ] Storage bucket renamed
- [ ] Migration script created and tested
- [ ] All tests updated and passing

### CLI
- [ ] Commands renamed (task → container)
- [ ] Aliases added (ps, inspect, logs)
- [ ] Help text updated
- [ ] Output messages updated

### Documentation
- [ ] All 20 doc.go files updated
- [ ] User docs updated (~388 occurrences)
- [ ] Migration guide created
- [ ] CHANGELOG updated
- [ ] README updated

### Testing
- [ ] Unit tests passing (go test ./...)
- [ ] Integration tests passing
- [ ] E2E tests passing
- [ ] Manual testing completed
- [ ] Migration tested with real data

### Release
- [ ] Version bumped to v2.0.0
- [ ] Git tag created
- [ ] Release notes written
- [ ] Breaking changes documented

---

## Terminology Reference

### Quick Reference Card

| Old Term | New Term | Context |
|----------|----------|---------|
| Task | Container | Always |
| Tasks | Containers | Always |
| TaskID | ContainerID | Always |
| TaskState | ContainerState | Always |
| Instance | Container | Always (runtime) |
| Instance | Replica | Only in scaling context |
| task.go | container.go | File names |
| ListTasks() | ListContainers() | Functions |
| /tasks | /containers | API endpoints |
| bucketTasks | bucketContainers | Storage |

### Writing Guidelines

**Do:**
- ✅ "Container is running on node-1"
- ✅ "Service has 3 replicas"
- ✅ "Scale to 5 replicas"
- ✅ "Container health check failed"
- ✅ "List all containers"

**Don't:**
- ❌ "Task is running"
- ❌ "Service has 3 tasks"
- ❌ "Instance failed"
- ❌ "3 instances of the service"

---

## Post-Migration Tasks

### Immediate (Week 1)
- Monitor for bug reports
- Update any missed documentation
- Answer migration questions
- Fix critical issues

### Short-term (Month 1)
- Gather feedback
- Update any external tools
- Blog post about terminology change
- Update videos/tutorials

### Long-term (Quarter 1)
- Ensure ecosystem adoption
- Update third-party integrations
- Archive old documentation versions

---

## Decision

**Recommendation:** Proceed with **Option 1: Big Bang** approach

**Rationale:**
1. Warren is early (v1.1.0) - few production users
2. Consistency is critical for long-term success
3. Clean break better than confusing transition
4. 4-5 days of work is reasonable
5. Benefits outweigh migration pain

**Next Steps:**
1. Get approval for breaking change
2. Announce to users (if any)
3. Create feature branch
4. Begin Phase 1: Preparation

---

**Status:** 📋 Awaiting Approval
**Author:** Claude (Anthropic)
**Reviewer:** TBD
**Approved:** TBD
