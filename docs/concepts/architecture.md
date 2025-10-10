# Warren Architecture

This document explains Warren's high-level architecture and how its components work together.

## Design Principles

Warren is built on three core principles:

1. **Simplicity** - Single binary, zero external dependencies, easy to understand
2. **Feature-Rich** - Built-in secrets, volumes, HA, load balancing (no add-ons required)
3. **Edge-First** - Optimized for resource constraints, partition tolerance, autonomous operation

## High-Level Architecture

```
┌──────────────────────────────────────────────────────────┐
│                     Warren Cluster                        │
│                                                            │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐      │
│  │  Manager 1  │  │  Manager 2  │  │  Manager 3  │      │
│  │  (Leader)   │◄─┤  (Follower) │◄─┤  (Follower) │      │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘      │
│         │                │                │               │
│         │      Raft Consensus (State)     │               │
│         │                                  │               │
│         └──────────────┬───────────────────┘              │
│                        │                                   │
│       ┌────────────────┴────────────────┐                 │
│       │         Worker Pool              │                 │
│       │                                  │                 │
│  ┌────▼─────┐  ┌──────────┐  ┌──────────┐               │
│  │ Worker 1 │  │ Worker 2 │  │ Worker N │               │
│  │          │  │          │  │          │               │
│  │ [nginx]  │  │ [redis]  │  │ [app]    │               │
│  └──────────┘  └──────────┘  └──────────┘               │
└──────────────────────────────────────────────────────────┘
```

## Core Components

### 1. Manager Node

The manager node runs the control plane and is responsible for:

- **Cluster State** - Maintains desired and current state via Raft
- **API Server** - gRPC API for client requests
- **Scheduler** - Assigns tasks to worker nodes
- **Reconciler** - Ensures desired state matches current state
- **Secrets Management** - Encrypts and distributes secrets

**Components:**

```
Manager
├── Raft (Consensus)
│   ├── Leader Election
│   ├── Log Replication
│   └── State Machine (FSM)
├── Storage (BoltDB)
│   ├── Services
│   ├── Tasks
│   ├── Nodes
│   ├── Secrets
│   └── Volumes
├── API Server (gRPC)
│   ├── Service API
│   ├── Node API
│   ├── Secret API
│   ├── Volume API
│   └── Cluster API
├── Scheduler
│   ├── Task Placement
│   ├── Load Balancing
│   └── Affinity Rules
└── Reconciler
    ├── Health Checking
    ├── Failure Detection
    └── Task Rescheduling
```

### 2. Worker Node

The worker node runs workloads (containers) and is responsible for:

- **Task Execution** - Runs containers via containerd
- **Heartbeat** - Reports health to manager every 30s
- **Task Polling** - Fetches assigned tasks from manager
- **Container Lifecycle** - Pull, create, start, stop, delete
- **Local State** - Caches task state for autonomous operation

**Components:**

```
Worker
├── Containerd Runtime
│   ├── Image Pulling
│   ├── Container Creation
│   ├── Container Execution
│   └── Container Monitoring
├── Task Executor
│   ├── Task Polling (5s)
│   ├── State Reporting
│   └── Lifecycle Management
├── Secret Handler
│   ├── Fetch from Manager
│   ├── Decrypt
│   └── Mount (tmpfs)
└── Volume Handler
    ├── Create Volume
    ├── Mount Volume
    └── Track Usage
```

### 3. Raft Consensus

Warren uses [Hashicorp Raft](https://github.com/hashicorp/raft) for distributed consensus among managers.

**Key Features:**

- **Leader Election** - One manager is elected as leader (handles writes)
- **Log Replication** - Leader replicates state changes to followers
- **Quorum** - Requires majority (2/3 for 3-manager, 3/5 for 5-manager)
- **Failover** - Automatic leader re-election on failure (~2-3s)

**State Machine (FSM):**

All cluster state changes go through Raft:

```go
// Example state transition
CreateService("nginx") → Raft Log → Apply to FSM → Update BoltDB → Return Success
```

**Raft guarantees:**
- **Strong consistency** - All managers see same state
- **Durability** - State survives node failures
- **Partition tolerance** - Operates with majority available

### 4. Scheduler

The scheduler assigns tasks to worker nodes based on:

- **Resource availability** - CPU, memory, disk (future)
- **Load balancing** - Round-robin across workers
- **Node affinity** - Pin tasks to specific nodes (volumes)
- **Service mode** - Replicated (N tasks) or global (1 per node)

**Scheduling Algorithm:**

```
1. Receive service spec (image, replicas, constraints)
2. Calculate desired tasks (replicas or global)
3. For each task:
   a. Filter eligible nodes (online, capacity)
   b. Apply affinity rules (volume constraints)
   c. Select node (round-robin or best-fit)
   d. Assign task to node
4. Store task assignments in Raft
5. Workers poll and execute assigned tasks
```

### 5. Reconciler

The reconciler ensures desired state matches current state:

**Reconciliation Loop (every 10s):**

```
1. Get desired state (services, tasks)
2. Get current state (task status from workers)
3. Detect drift:
   - Tasks failed → Reschedule
   - Nodes down → Mark tasks failed
   - Replicas mismatch → Scale up/down
4. Take corrective actions
5. Update state via Raft
```

**Failure Scenarios:**

| Failure | Detection | Action |
|---------|-----------|--------|
| Task crashes | Worker reports failure | Reschedule task on same/different node |
| Node down | Heartbeat timeout (30s) | Mark all node tasks failed, reschedule |
| Manager down | Raft timeout (500ms) | Elect new leader, continue operations |
| Network partition | Split-brain prevention | Minority partition stops processing writes |

### 6. Storage (BoltDB)

Warren uses [BoltDB](https://github.com/etcd-io/bbolt) for embedded key-value storage.

**Buckets:**

```
warren.db (BoltDB)
├── services/       → Service definitions
├── tasks/          → Task assignments and status
├── nodes/          → Node registry and health
├── secrets/        → Encrypted secrets
├── volumes/        → Volume metadata
└── cluster_config/ → Cluster-wide settings
```

**Why BoltDB?**

- ✅ Single-file database (no external process)
- ✅ ACID transactions
- ✅ Fast reads (B+tree index)
- ✅ Embedded in binary (zero dependencies)
- ✅ Small footprint (< 5MB for metadata)

## Data Flow

### Service Creation Flow

```
1. Client: warren service create nginx --image nginx:latest --replicas 3
2. API Server: Validate request
3. Manager: Create service via Raft log entry
4. FSM: Apply log entry → Store service in BoltDB
5. Scheduler: Create 3 tasks, assign to workers
6. Raft: Replicate task assignments to followers
7. Workers: Poll, fetch assigned tasks
8. Workers: Pull nginx:latest image
9. Workers: Create and start nginx containers
10. Workers: Report task status (running)
11. API Server: Return service ID to client
```

**Time to deploy**: < 5 seconds (excluding image pull)

### Service Update Flow (Rolling)

```
1. Client: warren service update nginx --image nginx:alpine
2. Manager: Update service spec via Raft
3. Reconciler: Detect image mismatch
4. Reconciler: Rolling update strategy:
   - Stop task 1 → Wait healthy → Start new task 1
   - Stop task 2 → Wait healthy → Start new task 2
   - Stop task 3 → Wait healthy → Start new task 3
5. Workers: Execute new tasks with nginx:alpine
6. Reconciler: Update complete
```

**Rolling update time**: ~30-60s (depends on health checks)

### Leader Failover Flow

```
1. Leader manager crashes
2. Followers detect missing heartbeats (500ms)
3. Followers start election (500ms-1s)
4. New leader elected (first to get quorum)
5. New leader takes over API requests
6. Clients retry failed requests to new leader
7. Cluster operations resume
```

**Failover time**: 2-3s (sub-10s)

## Network Architecture

Warren uses **WireGuard** for secure overlay networking:

```
┌───────────────┐          ┌───────────────┐
│   Manager 1   │          │   Worker 1    │
│  10.0.0.1     │◄────────►│  10.0.0.2     │
└───────────────┘   VPN    └───────────────┘
        ▲                          ▲
        │                          │
        │         WireGuard        │
        │                          │
        └──────────┬───────────────┘
                   │
            ┌──────▼──────┐
            │   Worker 2  │
            │  10.0.0.3   │
            └─────────────┘
```

**Features:**

- **Encryption** - All inter-node traffic encrypted (ChaCha20-Poly1305)
- **NAT Traversal** - Works across NAT boundaries
- **Low Overhead** - Minimal CPU/memory impact
- **Fast** - Line-rate performance on modern hardware

**Service VIP:**

Each service gets a virtual IP (VIP):

```bash
# Service "nginx" assigned VIP 10.1.0.5
# Traffic to 10.1.0.5 load-balanced to tasks:
nginx-task-1 → 10.0.1.10 (worker-1)
nginx-task-2 → 10.0.1.11 (worker-1)
nginx-task-3 → 10.0.1.12 (worker-2)
```

## Security Model

### Secrets Encryption

- **Algorithm** - AES-256-GCM
- **Key Derivation** - PBKDF2 from cluster initialization
- **Storage** - Encrypted at rest in BoltDB
- **Distribution** - Decrypted on worker, mounted via tmpfs

### Network Security

- **WireGuard** - All inter-node traffic encrypted
- **mTLS** - Optional mutual TLS for API (M6)
- **Join Tokens** - Time-limited (24h), single-use tokens

### Isolation

- **Namespace** - Warren uses dedicated containerd namespace
- **Filesystem** - Secrets on tmpfs (RAM, never disk)
- **Network** - WireGuard overlay isolates cluster traffic

## Resource Requirements

### Manager Node

- **CPU** - 1 core (2+ recommended for HA)
- **Memory** - 256MB (512MB for large clusters)
- **Disk** - 10GB (mostly for container images)
- **Network** - 1 Gbps recommended

### Worker Node

- **CPU** - 1 core (depends on workload)
- **Memory** - 512MB + workload requirements
- **Disk** - 20GB (for container images and volumes)
- **Network** - 1 Gbps recommended

### Edge Deployment

Warren runs on resource-constrained edge hardware:

- **Raspberry Pi 4** - 2GB RAM, ARM64
- **Intel NUC** - 4GB RAM, AMD64
- **Industrial IoT gateways** - 1GB RAM, ARM64

## Comparison to Other Orchestrators

| Feature | Warren | Docker Swarm | Kubernetes |
|---------|--------|--------------|------------|
| **Binary Size** | 35MB | 50MB | N/A (distributed) |
| **Manager Memory** | 256MB | 200MB | 2GB+ (etcd, controller) |
| **Setup Time** | < 5 min | < 5 min | 30+ min |
| **Built-in HA** | ✅ Raft | ✅ Raft | ✅ etcd |
| **Built-in Secrets** | ✅ AES-256 | ✅ | ✅ |
| **Built-in Volumes** | ✅ | ✅ | ❌ (CSI drivers) |
| **Built-in LB** | ✅ VIP | ✅ VIP | ❌ (ingress controller) |
| **Built-in Metrics** | ✅ Prometheus | ❌ | ❌ (add-on) |
| **Edge Optimized** | ✅ | ❌ | ❌ |
| **Active Development** | ✅ Open Source | ❌ Closed | ✅ Open Source |

## Performance Characteristics

### Throughput

- **Service creation**: 10+ services/sec
- **API latency**: < 100ms (50ms typical)
- **Heartbeat overhead**: < 1% CPU
- **State replication**: < 10ms (3-manager)

### Scalability

- **Managers**: 3 or 5 (recommended 3 for edge)
- **Workers**: Tested up to 100 nodes
- **Services**: 1000+ per cluster
- **Tasks**: 10,000+ total

### Failover

- **Manager failover**: 2-3s (sub-10s)
- **Task reschedule**: 30s (heartbeat timeout)
- **Network partition**: Majority partition continues

## Next Steps

- **[Services](services.md)** - Learn about service types and deployment
- **[Networking](networking.md)** - Understand WireGuard and VIPs
- **[Storage](storage.md)** - Volumes and secrets
- **[High Availability](high-availability.md)** - Multi-manager clusters

---

**Questions?** See [Troubleshooting](../troubleshooting.md) or ask in [GitHub Discussions](https://github.com/cuemby/warren/discussions).
