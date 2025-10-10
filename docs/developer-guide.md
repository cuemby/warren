# Warren Developer Guide

Deep dive into Warren's architecture, design decisions, and implementation details.

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Core Components](#core-components)
3. [Data Flow](#data-flow)
4. [State Management](#state-management)
5. [Scheduling Algorithm](#scheduling-algorithm)
6. [Reconciliation Loop](#reconciliation-loop)
7. [Code Organization](#code-organization)
8. [Development Workflow](#development-workflow)
9. [Testing Strategy](#testing-strategy)
10. [Contributing](#contributing)

---

## Architecture Overview

### High-Level Design

Warren follows a classic manager-worker architecture with distributed consensus:

```
┌─────────────────────── MANAGER NODE ───────────────────────┐
│                                                              │
│  ┌──────────────┐                                          │
│  │     CLI      │                                          │
│  │  (Cobra)     │                                          │
│  └──────┬───────┘                                          │
│         │                                                   │
│  ┌──────▼───────────────────────────────────────────┐     │
│  │              gRPC API Server                      │     │
│  │          (25+ methods, port 8080)                │     │
│  └──────┬───────────────────────────────────────────┘     │
│         │                                                   │
│  ┌──────▼─────────┐  ┌─────────────┐  ┌──────────────┐   │
│  │    Manager     │  │  Scheduler  │  │  Reconciler  │   │
│  │                │  │ (5s loop)   │  │  (10s loop)  │   │
│  │  - Raft Setup  │  │             │  │              │   │
│  │  - Bootstrap   │  │ - Round     │  │ - Health     │   │
│  │  - Commands    │  │   Robin     │  │   Check      │   │
│  └──────┬─────────┘  │ - Task      │  │ - Failure    │   │
│         │            │   Creation  │  │   Detection  │   │
│         │            └──────┬──────┘  └──────┬───────┘   │
│  ┌──────▼─────────────────┐│                │            │
│  │      Raft FSM          ││                │            │
│  │                        ││                │            │
│  │  - Apply()            ││                │            │
│  │  - Snapshot()         ││                │            │
│  │  - Restore()          ││                │            │
│  └──────┬──────────────────┘                │            │
│         │                                    │            │
│  ┌──────▼──────────────────┐                │            │
│  │  BoltDB Storage         │◄───────────────┘            │
│  │                         │                              │
│  │  - Nodes                │                              │
│  │  - Services             │                              │
│  │  - Tasks                │                              │
│  │  - Secrets              │                              │
│  │  - Volumes              │                              │
│  └─────────────────────────┘                              │
└──────────────────────────────────────────────────────────┘
                          │
                          │ gRPC (Heartbeat, Task Status)
                          ▼
┌─────────────────────── WORKER NODE ────────────────────────┐
│                                                              │
│  ┌──────────────────────────────────────────────┐          │
│  │            Worker Agent                       │          │
│  │                                               │          │
│  │  - Registration  - Heartbeat (5s)            │          │
│  │  - Task Sync (3s)  - Status Reporting        │          │
│  └──────┬───────────────────────────────────────┘          │
│         │                                                   │
│  ┌──────▼─────────┐                                        │
│  │  Task Map      │  (Local State Cache)                  │
│  │                │                                        │
│  │  task-1: RUNNING                                       │
│  │  task-2: PENDING                                       │
│  │  task-3: RUNNING                                       │
│  └──────┬─────────┘                                        │
│         │                                                   │
│  ┌──────▼─────────────────────────────┐                   │
│  │   Task Executor (Simulated)        │                   │
│  │                                     │                   │
│  │   - Start Task                      │                   │
│  │   - Monitor Task                    │                   │
│  │   - Stop Task                       │                   │
│  │   - Report Status                   │                   │
│  └─────────────────────────────────────┘                   │
└──────────────────────────────────────────────────────────┘
```

### Design Principles

1. **Simplicity First**: Minimize complexity, maximize clarity
2. **Single Binary**: Zero external dependencies
3. **Edge-First**: Designed for unreliable networks
4. **Raft Consensus**: Strong consistency over availability
5. **Declarative**: Desired state → actual state reconciliation

---

## Core Components

### 1. Manager ([pkg/manager/manager.go](../pkg/manager/manager.go))

**Responsibilities:**
- Initialize and manage Raft cluster
- Accept API requests
- Apply state changes via Raft
- Coordinate scheduler and reconciler

**Key Methods:**
```go
NewManager(cfg *Config) (*Manager, error)
Bootstrap() error                          // Initialize single-node cluster
Apply(cmd Command) error                   // Submit command to Raft
CreateService(service *Service) error      // Create service via Raft
GetService(id string) (*Service, error)    // Read from local store
```

**Lifecycle:**
1. Create manager with configuration
2. Bootstrap Raft (single-node initially)
3. Start API server
4. Start scheduler and reconciler
5. Process commands via Raft
6. Handle graceful shutdown

---

### 2. Raft FSM ([pkg/manager/fsm.go](../pkg/manager/fsm.go))

**Responsibilities:**
- Apply log entries to cluster state
- Create snapshots for log compaction
- Restore from snapshots

**Key Methods:**
```go
Apply(log *raft.Log) interface{}           // Apply command to state
Snapshot() (FSMSnapshot, error)            // Create snapshot
Restore(rc io.ReadCloser) error            // Restore from snapshot
```

**Command Types:**
- `create_node`, `update_node`, `delete_node`
- `create_service`, `update_service`, `delete_service`
- `create_task`, `update_task`, `delete_task`
- `create_secret`, `delete_secret`
- `create_volume`, `delete_volume`

**FSM Flow:**
```
Command → JSON Marshal → Raft.Apply() → FSM.Apply() → Store Update
```

---

### 3. Storage ([pkg/storage/boltdb.go](../pkg/storage/boltdb.go))

**Responsibilities:**
- Persist cluster state
- Provide CRUD operations
- Query and filter entities

**BoltDB Structure:**
```
warren.db
├── nodes/
│   ├── manager-1 → JSON(Node)
│   └── worker-1  → JSON(Node)
├── services/
│   └── nginx     → JSON(Service)
├── tasks/
│   ├── task-1    → JSON(Task)
│   ├── task-2    → JSON(Task)
│   └── task-3    → JSON(Task)
├── secrets/
├── volumes/
└── networks/
```

**Key Methods:**
```go
CreateNode(node *Node) error
GetNode(id string) (*Node, error)
ListNodes() ([]*Node, error)
UpdateNode(node *Node) error
DeleteNode(id string) error
// Similar for Service, Task, Secret, Volume, Network
```

---

### 4. Scheduler ([pkg/scheduler/scheduler.go](../pkg/scheduler/scheduler.go))

**Responsibilities:**
- Ensure services have correct number of tasks
- Assign tasks to workers
- Handle scale up/down

**Algorithm (Round-Robin Load Balancing):**
```go
func (s *Scheduler) selectNode(nodes []*Node, existingTasks []*Task) *Node {
    // Count tasks per node
    taskCounts := make(map[string]int)
    for _, task := range existingTasks {
        if task.DesiredState == TaskStateRunning {
            taskCounts[task.NodeID]++
        }
    }

    // Find node with fewest tasks
    selectedNode := nil
    minTasks := MaxInt
    for _, node := range nodes {
        count := taskCounts[node.ID]
        if count < minTasks {
            minTasks = count
            selectedNode = node
        }
    }

    return selectedNode
}
```

**Scheduling Loop:**
```
Every 5 seconds:
1. List all services
2. List all nodes (filter ready workers)
3. For each service:
   a. Count active tasks
   b. Calculate delta (desired - actual)
   c. If delta > 0: Create tasks
   d. If delta < 0: Mark tasks for shutdown
4. Sleep 5 seconds
```

---

### 5. Reconciler ([pkg/reconciler/reconciler.go](../pkg/reconciler/reconciler.go))

**Responsibilities:**
- Detect unhealthy nodes
- Replace failed tasks
- Clean up completed tasks

**Reconciliation Loop:**
```
Every 10 seconds:
1. Check node health:
   - If heartbeat > 30s old: Mark node as down
2. Check task health:
   - If task failed: Mark for cleanup
   - If task on down node: Mark for rescheduling
3. Clean up:
   - Delete completed tasks (after 5min grace period)
4. Sleep 10 seconds
```

**Failure Detection:**
```go
// Node failure
if time.Since(node.LastHeartbeat) > 30*time.Second {
    node.Status = NodeStatusDown
    manager.UpdateNode(node)
}

// Task failure
if task.ActualState == TaskStateFailed && task.DesiredState == TaskStateRunning {
    task.DesiredState = TaskStateShutdown  // Scheduler will replace
    manager.UpdateTask(task)
}
```

---

### 6. Worker ([pkg/worker/worker.go](../pkg/worker/worker.go))

**Responsibilities:**
- Register with manager
- Send heartbeat with task status
- Poll for task assignments
- Execute tasks

**Worker Loops:**
```go
// Heartbeat loop (every 5 seconds)
func (w *Worker) heartbeatLoop() {
    ticker := time.NewTicker(5 * time.Second)
    for {
        select {
        case <-ticker.C:
            w.sendHeartbeat()  // Send all task statuses
        case <-w.stopCh:
            return
        }
    }
}

// Task sync loop (every 3 seconds)
func (w *Worker) taskExecutorLoop() {
    ticker := time.NewTicker(3 * time.Second)
    for {
        select {
        case <-ticker.C:
            w.syncTasks()  // Poll for assignments
        case <-w.stopCh:
            return
        }
    }
}
```

---

### 7. gRPC API ([pkg/api/server.go](../pkg/api/server.go))

**Responsibilities:**
- Expose manager functionality via gRPC
- Handle CLI requests
- Process worker heartbeats
- Type conversion (protobuf ↔ internal types)

**Key Endpoints:**
- Node: RegisterNode, Heartbeat, ListNodes, GetNode
- Service: CreateService, UpdateService, DeleteService, GetService, ListServices
- Task: UpdateTaskStatus, ListTasks, GetTask
- Secret: CreateSecret, DeleteSecret, ListSecrets
- Volume: CreateVolume, DeleteVolume, ListVolumes

---

## Data Flow

### Service Creation Flow

```
1. User runs CLI command:
   $ warren service create nginx --image nginx:latest --replicas 3

2. CLI creates gRPC client:
   client.NewClient("127.0.0.1:8080")

3. CLI calls CreateService:
   client.CreateService("nginx", "nginx:latest", 3, nil)

4. gRPC server receives request:
   server.CreateService(ctx, req)

5. Server marshals service to JSON

6. Server submits to Raft:
   manager.Apply(Command{Op: "create_service", Data: json})

7. Raft replicates to cluster (single node in M1)

8. Raft calls FSM.Apply():
   fsm.Apply(log)

9. FSM unmarshals and stores:
   store.CreateService(service)

10. BoltDB writes to disk:
    bolt.Update(func(tx) { bucket.Put(id, json) })

11. Response returned to CLI:
    "✓ Service created: nginx"

12. Scheduler detects new service (within 5s):
    - Sees replicas=3, actual=0
    - Creates 3 tasks
    - Assigns to workers (round-robin)

13. Tasks saved via Raft:
    manager.CreateTask(task1)
    manager.CreateTask(task2)
    manager.CreateTask(task3)

14. Worker polls for tasks (within 3s):
    client.ListTasks(nodeID)

15. Worker detects new assignments:
    - Starts task execution
    - Updates local state map

16. Worker reports status in heartbeat (within 5s):
    client.Heartbeat({task1: running, task2: running, task3: running})

17. Manager updates task states via Raft
```

---

### Task Failure Recovery Flow

```
1. Task fails (crashes or reports failure)

2. Worker detects failure:
   task.ActualState = TaskStateFailed

3. Worker reports in next heartbeat (within 5s):
   Heartbeat({taskID: failed})

4. Manager updates task via Raft:
   task.ActualState = TaskStateFailed

5. Reconciler detects failed task (within 10s):
   if task.ActualState == Failed && task.DesiredState == Running {
       task.DesiredState = Shutdown
   }

6. Scheduler detects missing task (within 5s):
   - Service needs 3 replicas
   - Only 2 running tasks found
   - Creates replacement task

7. Replacement assigned to worker:
   New task created via Raft

8. Worker picks up new task (within 3s)

9. Worker starts replacement

10. Worker reports success (within 5s)

Total recovery time: 10s (reconciler) + 5s (scheduler) + 3s (worker) = ~18s
```

---

## State Management

### State Transitions

**Node States:**
```
      ┌──────────────┐
      │   READY      │
      └──────┬───────┘
             │
             │ no heartbeat (30s)
             │
      ┌──────▼───────┐
      │    DOWN      │
      └──────┬───────┘
             │
             │ heartbeat received
             │
      ┌──────▼───────┐
      │   READY      │
      └──────────────┘
```

**Task States:**
```
┌─────────┐  assign   ┌─────────┐  execute  ┌─────────┐
│ PENDING ├──────────→│ RUNNING ├──────────→│COMPLETE │
└────┬────┘           └────┬────┘           └─────────┘
     │                     │
     │ fail                │ fail
     │                     │
     └────────┬────────────┘
              │
       ┌──────▼──────┐
       │   FAILED    │
       └──────┬──────┘
              │
              │ cleanup
              │
       ┌──────▼──────┐
       │  SHUTDOWN   │
       └─────────────┘
```

### Consistency Model

**Strong Consistency (Raft):**
- All writes go through Raft
- Linearizable reads from leader
- Snapshot + log replication

**Eventual Consistency:**
- Worker task status (synced via heartbeat)
- Node health (synced via reconciler)

---

## Scheduling Algorithm

### Current: Round-Robin Load Balancing

**Pros:**
- Simple implementation
- Fair distribution
- Predictable behavior

**Cons:**
- Ignores actual resource usage
- No affinity/anti-affinity
- No placement constraints

**Code:**
```go
func (s *Scheduler) selectNode(nodes []*Node, tasks []*Task) *Node {
    taskCounts := make(map[string]int)
    for _, task := range tasks {
        if task.DesiredState == TaskStateRunning {
            taskCounts[task.NodeID]++
        }
    }

    var selectedNode *Node
    minTasks := math.MaxInt

    for _, node := range nodes {
        if node.Status == NodeStatusReady && node.Role == NodeRoleWorker {
            count := taskCounts[node.ID]
            if count < minTasks {
                minTasks = count
                selectedNode = node
            }
        }
    }

    return selectedNode
}
```

### Future: Resource-Aware Scheduling (Milestone 2)

Consider:
- CPU/memory availability
- Node labels and constraints
- Service affinity/anti-affinity
- Volume locality

---

## Reconciliation Loop

### Reconciliation Logic

```go
func (r *Reconciler) reconcile() error {
    // 1. Reconcile nodes
    nodes, _ := r.manager.ListNodes()
    for _, node := range nodes {
        if time.Since(node.LastHeartbeat) > 30*time.Second {
            node.Status = NodeStatusDown
            r.manager.UpdateNode(node)
        }
    }

    // 2. Reconcile tasks
    tasks, _ := r.manager.ListTasks()
    for _, task := range tasks {
        // Handle failures
        if task.ActualState == TaskStateFailed {
            task.DesiredState = TaskStateShutdown
            r.manager.UpdateTask(task)
        }

        // Handle tasks on down nodes
        node, _ := r.manager.GetNode(task.NodeID)
        if node.Status == NodeStatusDown {
            task.DesiredState = TaskStateShutdown
            r.manager.UpdateTask(task)
        }

        // Cleanup completed tasks
        if task.DesiredState == TaskStateShutdown &&
           task.ActualState == TaskStateComplete &&
           time.Since(task.FinishedAt) > 5*time.Minute {
            r.manager.DeleteTask(task.ID)
        }
    }

    return nil
}
```

---

## Code Organization

### Directory Structure

```
warren/
├── cmd/warren/              # CLI entry point
│   └── main.go             # Cobra commands, flags
│
├── pkg/
│   ├── api/                # gRPC API
│   │   └── server.go       # API implementation
│   │
│   ├── client/             # Go client library
│   │   └── client.go       # High-level client
│   │
│   ├── manager/            # Manager node
│   │   ├── manager.go      # Manager orchestration
│   │   └── fsm.go          # Raft FSM
│   │
│   ├── reconciler/         # Reconciliation loop
│   │   └── reconciler.go   # Health checking, cleanup
│   │
│   ├── scheduler/          # Task scheduler
│   │   └── scheduler.go    # Scheduling logic
│   │
│   ├── storage/            # Storage layer
│   │   ├── store.go        # Interface
│   │   └── boltdb.go       # BoltDB implementation
│   │
│   ├── types/              # Core types
│   │   └── types.go        # Node, Service, Task, etc.
│   │
│   └── worker/             # Worker agent
│       └── worker.go       # Task execution
│
├── api/proto/              # Protobuf definitions
│   ├── warren.proto        # API schema
│   ├── warren.pb.go        # Generated
│   └── warren_grpc.pb.go   # Generated
│
└── test/integration/       # Integration tests
    └── e2e_test.sh         # End-to-end test
```

### Naming Conventions

- **Packages**: lowercase, single word (`manager`, `scheduler`)
- **Files**: lowercase, underscore separated (`boltdb_store.go`)
- **Types**: PascalCase (`Service`, `Task`)
- **Functions**: camelCase (`createService`, `listNodes`)
- **Constants**: PascalCase or UPPER_SNAKE_CASE

---

## Development Workflow

### Building

```bash
# Development build
make build

# Release build (optimized)
make build-release

# Install to /usr/local/bin
make install
```

### Testing

```bash
# Unit tests
make test

# Integration tests
./test/integration/e2e_test.sh

# Specific package
go test ./pkg/scheduler -v
```

### Protobuf Generation

```bash
# Regenerate protobuf code
make proto

# Manual generation
protoc --go_out=. --go-grpc_out=. api/proto/warren.proto
```

### Adding a New Feature

1. **Update Types** ([pkg/types/types.go](../pkg/types/types.go))
   - Add new data structures

2. **Update Storage** ([pkg/storage/store.go](../pkg/storage/store.go))
   - Add storage interface methods
   - Implement in boltdb.go

3. **Update FSM** ([pkg/manager/fsm.go](../pkg/manager/fsm.go))
   - Add new command type
   - Handle in Apply()

4. **Update Manager** ([pkg/manager/manager.go](../pkg/manager/manager.go))
   - Add manager methods
   - Call via Raft.Apply()

5. **Update API** ([api/proto/warren.proto](../api/proto/warren.proto))
   - Add protobuf messages
   - Add RPC methods
   - Regenerate: `make proto`

6. **Implement API** ([pkg/api/server.go](../pkg/api/server.go))
   - Implement new RPC methods
   - Add type conversions

7. **Update CLI** ([cmd/warren/main.go](../cmd/warren/main.go))
   - Add new commands/flags

8. **Test**
   - Unit tests
   - Integration tests
   - Manual testing

---

## Testing Strategy

### Test Pyramid

```
                  ╱╲
                 ╱  ╲
                ╱ E2E╲          (10%) - Integration tests
               ╱──────╲
              ╱        ╲
             ╱Integration╲      (30%) - API tests
            ╱────────────╲
           ╱              ╲
          ╱  Unit Tests    ╲    (60%) - Package tests
         ╱──────────────────╲
```

### Unit Tests

Test individual packages in isolation:

```go
// pkg/scheduler/scheduler_test.go
func TestSelectNode_RoundRobin(t *testing.T) {
    nodes := []*types.Node{
        {ID: "node-1", Status: types.NodeStatusReady},
        {ID: "node-2", Status: types.NodeStatusReady},
    }

    tasks := []*types.Task{
        {NodeID: "node-1", DesiredState: types.TaskStateRunning},
    }

    scheduler := NewScheduler(nil)
    selected := scheduler.selectNode(nodes, tasks)

    assert.Equal(t, "node-2", selected.ID)
}
```

### Integration Tests

Test complete workflows:

```bash
# test/integration/e2e_test.sh
# 1. Start manager
# 2. Start worker
# 3. Create service
# 4. Verify tasks created
# 5. Scale service
# 6. Delete service
# 7. Verify cleanup
```

### Manual Testing

```bash
# Terminal 1: Manager with verbose logging
warren cluster init --data-dir /tmp/warren-test

# Terminal 2: Worker
warren worker start --node-id test-worker --data-dir /tmp/warren-worker

# Terminal 3: Commands
warren service create test --image test:latest --replicas 3
warren service list
warren node list
warren service scale test --replicas 5
warren service delete test
```

---

## Contributing

### Code Style

Follow standard Go conventions:
- `gofmt` for formatting
- `golangci-lint` for linting
- Comments for exported symbols
- Table-driven tests

### Commit Messages

Use conventional commits:
```
feat: add health checking for tasks
fix: handle nil pointer in scheduler
docs: update API reference
test: add integration test for scaling
```

### Pull Request Process

1. Fork the repository
2. Create feature branch: `git checkout -b feature/my-feature`
3. Make changes with tests
4. Run tests: `make test`
5. Commit with conventional format
6. Push and create PR
7. Address review feedback

### Development Environment

```bash
# Clone repo
git clone https://github.com/cuemby/warren.git
cd warren

# Install dependencies
go mod download

# Build
make build

# Run tests
make test

# Start developing!
```

---

## Architecture Decision Records

See [docs/adr/](../docs/adr/) for detailed rationale behind major decisions:

- [ADR-001: Why Raft?](../docs/adr/001-why-raft.md)
- [ADR-002: Why containerd?](../docs/adr/002-why-containerd.md)
- [ADR-003: Why WireGuard?](../docs/adr/003-why-wireguard.md)
- [ADR-004: Why BoltDB?](../docs/adr/004-why-boltdb.md)
- [ADR-005: Why Go?](../docs/adr/005-why-go.md)

---

## See Also

- [Quick Start Guide](quickstart.md) - Getting started
- [API Reference](api-reference.md) - Complete API documentation
- [PRD](../specs/prd.md) - Product requirements
- [Tech Spec](../specs/tech.md) - Technical specification
- [TODO](../tasks/todo.md) - Development roadmap

---

**Last Updated:** 2025-10-10 (Milestone 1)
