# Task → Container Terminology Refactor - Status

**Status:** 🚧 IN PROGRESS
**Branch:** `refactor/container-terminology`
**Started:** 2025-10-13
**Target:** v2.0.0 (Breaking Change)

---

## ✅ Completed

### Planning Phase
- ✅ Comprehensive audit of codebase (57 Go files, 464+ occurrences)
- ✅ Created detailed refactor plan ([terminology-refactor-plan.md](terminology-refactor-plan.md))
- ✅ Created feature branch `refactor/container-terminology`
- ✅ Identified all breaking changes
- ✅ Estimated effort: 26 hours (4-5 days)

### Documentation
- ✅ Fixed terminology inconsistency in pkg/ingress/doc.go (instance → task)
- ✅ Added comprehensive Go package documentation (20 doc.go files)
- ✅ Created 10 production-ready YAML examples

---

## 🚧 Next Steps

This is a **LARGE refactor** that should be done systematically. Here's the recommended approach:

### Immediate Next Steps (Do in order)

**Step 1: Update Core Types** (`pkg/types/types.go`)
```go
// Changes needed:
- Task → Container
- TaskState → ContainerState
- TaskStatus → ContainerStatus
- TaskID field → ContainerID (in Event type)
```

**Step 2: Update Protocol Buffers** (`api/proto/warren.proto`)
```protobuf
// All Task messages → Container
// All RPC methods with "Task" → "Container"
// Regenerate: make proto
```

**Step 3: Update Storage Layer** (`pkg/storage/boltdb.go`)
```go
// Bucket rename: "tasks" → "containers"
// All query methods
// Create migration tool
```

**Step 4: Update Packages (in dependency order)**
1. pkg/manager/ (18 occurrences)
2. pkg/scheduler/ (36 occurrences)
3. pkg/reconciler/
4. pkg/worker/ (56 occurrences - largest!)
5. pkg/api/
6. pkg/client/
7. pkg/ingress/
8. pkg/health/
9. Others

**Step 5: Update CLI** (`cmd/warren/main.go`)
```bash
# Commands to update:
warren task → warren container
Add aliases: warren ps, warren inspect, warren logs
```

**Step 6: Update Tests** (12 test files)

**Step 7: Update Documentation** (20 doc.go + user docs)

**Step 8: Create Migration Tool** (for existing deployments)

**Step 9: Testing & Validation**

**Step 10: Release v2.0.0**

---

## 🎯 Recommended Approach

Given the scope, here are **three options**:

### Option A: Do It Now (Aggressive)
- Continue in this session
- Use automated search/replace with validation
- 4-5 day focused effort
- **Risk:** Might introduce bugs if rushed

### Option B: Systematic Refactor (Recommended)
- **Do in dedicated sessions, one phase at a time**
- Test after each phase
- More careful and thorough
- **Timeline:** 1-2 weeks
- **Benefit:** Lower risk, higher quality

### Option C: Create Automated Refactor Script
- Write a refactor script that handles all renames
- Review and test the script
- Run it in one go
- **Timeline:** 2-3 days (script + validation)
- **Benefit:** Consistency, repeatability

---

## 💡 My Recommendation

**Option B: Systematic, phased approach**

**Why:**
1. This touches 57 files across the entire codebase
2. gRPC API changes are breaking (requires careful handling)
3. Storage migration affects existing deployments
4. Need thorough testing at each stage
5. Warren is production software now

**How to proceed:**
1. **Today:** Update core types (types.go) + proto file
2. **Day 2:** Update storage + create migration tool
3. **Day 3:** Update manager, scheduler, reconciler packages
4. **Day 4:** Update worker, API, client packages
5. **Day 5:** Update CLI + documentation
6. **Day 6:** Testing + validation
7. **Day 7:** Release prep (v2.0.0, CHANGELOG, migration guide)

---

## 📋 Current Branch State

```bash
git branch
# * refactor/container-terminology

git status
# On branch refactor/container-terminology
# Clean (plan committed)

git log --oneline -3
# a6b281c docs: add comprehensive Task→Container terminology refactor plan
# 8a001a5 fix(docs): use consistent 'task' terminology
# 66275b6 docs: add comprehensive Go package documentation
```

---

## 🔧 Tools & Scripts Needed

### 1. Search/Replace Validation Script
```bash
#!/bin/bash
# validate-refactor.sh
# Ensures all Task→Container renames are complete

echo "Checking for remaining 'Task' occurrences..."
rg "Task" pkg/ -g "*.go" | grep -v "// Task" | grep -v doc.go
# Should return 0 results when done
```

### 2. Proto Regeneration Script
```bash
#!/bin/bash
# regenerate-proto.sh
cd api/proto
protoc --go_out=. --go-grpc_out=. warren.proto
cd ../..
go mod tidy
```

### 3. Database Migration Tool
```go
// cmd/warren-migrate/main.go
// Migrates BoltDB from "tasks" bucket to "containers" bucket
// Preserves all data, transforms field names
```

### 4. Test Runner
```bash
#!/bin/bash
# test-all.sh
echo "Running unit tests..."
go test ./... -v

echo "Running integration tests..."
go test ./test/integration/... -v

echo "Running E2E tests..."
./test/lima/test-cluster.sh
```

---

## ⚠️ Important Notes

1. **This is a v2.0.0 breaking change** - All existing clients/deployments must update

2. **Data migration required** - Existing Warren clusters need migration tool

3. **No rollback after release** - Once v2.0.0 is out, no going back

4. **Communication needed** - If Warren has users, announce this change early

5. **Testing is critical** - Must have comprehensive tests before release

---

## 🤔 Decision Point

**Before proceeding further, confirm:**

1. ✅ **Approve systematic approach?** (Option B - phased refactor)
2. ⏸️ **Schedule refactor work?** (Dedicated time blocks)
3. ⏸️ **Create refactor scripts first?** (Automation for consistency)
4. ⏸️ **Announcement needed?** (If Warren has production users)

---

## 📝 Commit Message Template

For when refactor is complete:

```
refactor!: replace Task with Container throughout codebase

BREAKING CHANGE: Complete terminology refactor from Task to Container

Replace all "Task" and "Instance" terminology with "Container" and
"Replica" for clarity and consistency.

Changes:
- Renamed Task → Container in all types
- Renamed TaskState → ContainerState
- Updated gRPC API: all Task RPCs → Container RPCs
- Updated storage: "tasks" bucket → "containers" bucket
- Updated CLI: warren task → warren container
- Updated documentation (20 doc.go + user docs)
- Created migration tool for existing deployments

Breaking Changes:
- gRPC API methods renamed
- Storage schema changed (requires migration)
- CLI commands renamed
- All client code must update

Migration:
- Run migration tool: warren-migrate --from-v1 --to-v2
- See docs/migration/v1-to-v2.md for full instructions

Files changed: 57 Go files, 12 tests, 20+ docs
Lines changed: ~464 occurrences replaced

This change improves clarity by using descriptive "Container" instead
of abstract "Task" terminology. "Replica" is used in scaling contexts
("service with 3 replicas").

Closes #XXX

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>
```

---

**Status:** 🟡 PAUSED - Awaiting decision on approach

**Next Action:** Choose Option A, B, or C above and proceed accordingly
