# Project Architecture - Warren Container Orchestrator

## Overview

**Project Name**: Warren
**Type**: Container Orchestration Platform
**Purpose**: Simple yet feature-rich container orchestrator for edge computing, combining Docker Swarm's simplicity with Kubernetes-level features in a single binary with zero external dependencies.

**Last Updated**: 2025-10-09

---

## What This Is

Warren is a **production-ready container orchestration system** built from scratch that:

1. **Orchestrates containerized workloads** across distributed infrastructure
2. **Provides high availability** through Raft consensus (multi-manager cluster)
3. **Operates autonomously at the edge** with partition tolerance
4. **Ships as a single binary** (< 100MB) with no external dependencies
5. **Built for edge computing** with telco-grade reliability

**Philosophy**: "Docker Swarm simplicity + Kubernetes features - Kubernetes complexity"

---

## Project Structure

```
warren/
│
├── cmd/
│   └── warren/
│       └── main.go                    # CLI entry point, determines run mode
│
├── pkg/
│   ├── api/
│   │   ├── grpc.go                    # gRPC server (primary API)
│   │   ├── rest.go                    # REST gateway (grpc-gateway)
│   │   └── proto/                     # Protocol buffers definitions
│   │       ├── cluster.proto
│   │       ├── service.proto
│   │       └── task.proto
│   │
│   ├── manager/
│   │   ├── manager.go                 # Manager orchestration loop
│   │   ├── raft.go                    # Raft consensus integration
│   │   ├── scheduler.go               # Task placement scheduler
│   │   └── reconciler.go              # State reconciliation loop
│   │
│   ├── worker/
│   │   ├── agent.go                   # Worker agent
│   │   ├── runtime.go                 # Containerd integration
│   │   ├── healthcheck.go             # Container health checking
│   │   └── cache.go                   # Local state cache (partition tolerance)
│   │
│   ├── network/
│   │   ├── wireguard.go               # WireGuard mesh networking
│   │   ├── dns.go                     # Embedded DNS service
│   │   └── loadbalancer.go            # Service VIP & load balancing (iptables)
│   │
│   ├── security/
│   │   ├── ca.go                      # Certificate authority
│   │   ├── mtls.go                    # Mutual TLS implementation
│   │   └── secrets.go                 # Secrets encryption (AES-256-GCM)
│   │
│   ├── storage/
│   │   ├── state.go                   # Cluster state management
│   │   └── boltdb.go                  # BoltDB wrapper (Raft log store)
│   │
│   ├── deploy/
│   │   ├── rolling.go                 # Rolling update strategy
│   │   ├── bluegreen.go               # Blue/green deployment
│   │   └── canary.go                  # Canary deployment
│   │
│   └── types/
│       └── types.go                   # Core data types (Cluster, Node, Service, Task)
│
├── test/
│   ├── integration/                   # Integration tests
│   │   ├── cluster_test.go
│   │   ├── service_test.go
│   │   └── network_test.go
│   └── chaos/                         # Chaos/partition tests
│       └── partition_test.go
│
├── specs/
│   ├── prd.md                         # Product Requirements Document
│   └── tech.md                        # Technical Specification
│
├── tasks/
│   └── todo.md                        # Milestone-based development plan
│
├── docs/
│   ├── architecture.md                # Architecture deep-dive
│   ├── user-guide.md                  # User guide
│   └── api-reference.md               # API reference
│
├── .agent/                            # AI development framework
│   ├── README.md                      # Documentation index
│   ├── SOP/                           # Standard Operating Procedures
│   ├── System/                        # System documentation
│   └── Tasks/                         # Task templates
│
├── .claude/                           # Claude Code configuration
│   ├── settings.local.json
│   └── commands/                      # Custom slash commands
│
├── Makefile                           # Build automation
├── go.mod                             # Go module definition
├── go.sum                             # Go dependency checksums
├── CLAUDE.md                          # AI-specific instructions
├── LICENSE                            # Apache 2.0 (future)
└── README.md                          # Project README (future)
```

---

## Technology Stack

### Core Language & Runtime
- **Language**: Go 1.22+ (cross-platform, performance, cloud-native ecosystem)
- **Build**: CGO_ENABLED=0 (static binary), cross-compilation for multi-platform

### Distributed Systems
- **Consensus**: Raft via `hashicorp/raft` (leader election, log replication)
- **State Storage**: BoltDB via `hashicorp/raft-boltdb` (embedded key-value store)
- **Replication**: Raft log replication across manager quorum (3 or 5 nodes)

### Container Runtime
- **Runtime**: containerd 1.6+ (`github.com/containerd/containerd`)
- **Interface**: CRI-compatible (Container Runtime Interface)
- **Image Handling**: Pull, create, start, stop, delete via containerd gRPC API
- **Independence**: Docker-independent (works with standalone containerd)

### Networking
- **Overlay**: WireGuard (`golang.zx2c4.com/wireguard/wgctrl`)
- **Topology**: Full mesh (all nodes peer with all nodes)
- **Encryption**: ChaCha20-Poly1305 (WireGuard default)
- **DNS**: Embedded DNS server (resolves service names → VIPs)
- **Load Balancing**: iptables DNAT rules (round-robin to healthy replicas)

### Security
- **mTLS**: X.509 certificates via `crypto/x509` (manager-worker communication)
- **Certificate Authority**: Self-signed root CA, automatic cert issuance & rotation
- **Secrets Encryption**: AES-256-GCM via `crypto/aes` (secrets at rest)
- **Key Management**: Cluster encryption key in Raft, distributed securely

### APIs
- **Primary**: gRPC via `google.golang.org/grpc` (binary protocol, streaming)
- **Secondary**: REST via `grpc-ecosystem/grpc-gateway` (JSON/HTTP)
- **Protocol Buffers**: `google.golang.org/protobuf` (API schema)

### CLI
- **Framework**: Cobra (`github.com/spf13/cobra`)
- **Commands**: `warren cluster|service|node|secret|volume` + short alias `wrn`
- **Completion**: Bash, Zsh, Fish tab completion

### Observability
- **Metrics**: Prometheus via `github.com/prometheus/client_golang`
- **Logging**: Structured JSON via `github.com/rs/zerolog`
- **Endpoints**: `/metrics` (Prometheus), structured logs to stdout

### Build & Distribution
- **Build Tool**: Makefile + `go build`
- **Optimization**: `-ldflags="-s -w"` (strip debug), UPX compression
- **Cross-Compilation**: Linux (amd64, arm64), macOS (amd64, arm64), Windows (WSL2)
- **Distribution**: GitHub Releases, Homebrew, APT, Docker Hub

---

## System Architecture

### High-Level Design

Warren operates in two primary modes: **Manager** and **Worker**

```
┌─────────────────────────────────────────────────┐
│             Manager Nodes (Control Plane)        │
│                                                  │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐         │
│  │Manager 1│  │Manager 2│  │Manager 3│         │
│  │(Leader) │←→│(Follower)│→│(Follower)│         │
│  │         │  │         │  │         │         │
│  │  Raft   │  │  Raft   │  │  Raft   │         │
│  │ BoltDB  │  │ BoltDB  │  │ BoltDB  │         │
│  └─────────┘  └─────────┘  └─────────┘         │
│       ↓             ↓             ↓             │
│  ┌──────────────────────────────────────┐       │
│  │ API Server │ Scheduler │ Reconciler  │       │
│  └──────────────────────────────────────┘       │
└─────────────────────────────────────────────────┘
                      ↓
        ┌─────────────────────────────┐
        │   WireGuard Overlay Network │
        └─────────────────────────────┘
                      ↓
        ┌──────────────┬──────────────┐
        ↓              ↓              ↓
┌─────────────┐  ┌─────────────┐  ┌─────────────┐
│ Worker Node │  │ Worker Node │  │ Worker Node │
│             │  │             │  │             │
│   Agent     │  │   Agent     │  │   Agent     │
│ containerd  │  │ containerd  │  │ containerd  │
│             │  │             │  │             │
│[Containers] │  │[Containers] │  │[Containers] │
└─────────────┘  └─────────────┘  └─────────────┘
```

### Manager Components

1. **Raft Consensus**
   - Leader election (Raft protocol)
   - Log replication to followers (majority quorum)
   - State machine (FSM) applies log entries to cluster state
   - Snapshot/compaction every 10K entries

2. **API Server**
   - gRPC primary (`:2377`) - CLI, programmatic access
   - REST gateway (`:2378`) - HTTP/JSON for web UIs
   - mTLS enforcement (client cert validation)

3. **Scheduler**
   - Task placement algorithm (bin-packing with spread)
   - Resource-aware (CPU, memory, disk constraints)
   - Constraint-based (labels, affinity, node availability)
   - Runs every 5 seconds (unassigned tasks)

4. **Reconciler**
   - Ensures desired state = actual state
   - Detects failed tasks, creates replacements
   - Handles scale-up/scale-down
   - Runs every 10 seconds

### Worker Components

1. **Agent**
   - Heartbeat to manager (gRPC stream)
   - Watches for task assignments (event stream)
   - Local state cache (BoltDB) for partition tolerance
   - Autonomous operation during network partition

2. **Containerd Runtime**
   - Container lifecycle (pull, create, start, stop, delete)
   - Log streaming from containers
   - Resource tracking (CPU, memory usage)

3. **Health Checker**
   - HTTP probes (`GET /health`)
   - TCP probes (port connectivity)
   - Exec probes (command in container)
   - Reports to manager (unhealthy → task rescheduled)

### Networking Architecture

1. **WireGuard Overlay**
   - Each node: WireGuard interface (`wg0`) with cluster IP
   - Full mesh: All nodes peer with all nodes
   - Automatic peer configuration (keys distributed via Raft)

2. **Service VIP & Load Balancing**
   - Each service assigned Virtual IP (VIP) from cluster subnet
   - iptables DNAT rules route VIP → healthy task IPs
   - Round-robin load balancing (statistic mode nth)

3. **DNS Service**
   - Embedded DNS on managers (port 53)
   - Service name → VIP resolution
   - Workers configure managers as DNS servers

---

## Core Data Models

### Cluster State Entities

```go
// Cluster - root entity
type Cluster struct {
    ID            string
    CreatedAt     time.Time
    Managers      []*Node
    Workers       []*Node
    NetworkConfig *NetworkConfig
}

// Node - manager or worker
type Node struct {
    ID            string
    Role          string  // "manager" | "worker"
    Address       string  // Host IP
    OverlayIP     net.IP  // WireGuard IP
    Resources     *NodeResources
    Status        string  // "ready" | "down" | "draining"
    LastHeartbeat time.Time
}

// Service - user workload definition
type Service struct {
    ID             string
    Name           string
    Image          string
    Replicas       int
    DeployStrategy string  // "rolling" | "blue-green" | "canary"
    UpdateConfig   *UpdateConfig
    Networks       []string
    Secrets        []string
    Volumes        []*VolumeMount
}

// Task - single container instance
type Task struct {
    ID           string
    ServiceID    string
    NodeID       string
    ContainerID  string
    DesiredState string  // "running" | "shutdown"
    ActualState  string  // "pending" | "running" | "failed"
    Image        string
    HealthCheck  *HealthCheck
    RestartPolicy *RestartPolicy
}

// Secret - encrypted sensitive data
type Secret struct {
    ID        string
    Name      string
    Data      []byte  // AES-256-GCM encrypted
    CreatedAt time.Time
}

// Volume - persistent storage
type Volume struct {
    ID        string
    Name      string
    Driver    string  // "local", "nfs", etc.
    NodeID    string  // Node affinity (for local volumes)
    MountPath string
}
```

### Storage Layer

- **Raft Log**: Sequential log of all state changes (append-only)
- **BoltDB Buckets**: `nodes`, `services`, `tasks`, `secrets`, `volumes`, `networks`
- **FSM (Finite State Machine)**: Applies Raft log entries to in-memory state
- **Snapshots**: Periodic state snapshots (every 10K log entries), retains 3 snapshots

---

## Key Features & Capabilities

### 1. Cluster Management
- Single-command cluster init: `warren cluster init`
- Token-based worker join: `warren cluster join --token <token>`
- Multi-manager HA (3 or 5 manager quorum)
- Automatic leader election & failover (< 10s)

### 2. Service Orchestration
- Replicated services (N copies for availability)
- Global services (one per node - e.g., monitoring agents)
- Stateful services (with persistent volumes)
- Resource constraints (CPU, memory limits/reservations)

### 3. Deployment Strategies
- **Rolling Update**: Gradual replacement (configurable parallelism & delay)
- **Blue/Green**: Deploy new version, instant traffic switch, cleanup old
- **Canary**: Percentage-based traffic split (10% → 50% → 100%)

### 4. Networking
- Encrypted overlay (WireGuard mesh)
- Service discovery (DNS: service name → VIP)
- Load balancing (round-robin to healthy replicas)
- Zero-config (automatic peer setup)

### 5. Security
- mTLS by default (manager ↔ worker)
- Self-signed CA (automatic cert issuance & rotation)
- Secrets encryption at rest (AES-256-GCM)
- Secrets mounted as tmpfs (never written to disk)

### 6. Edge Resilience
- Autonomous worker operation during partition
- Local state caching (BoltDB on workers)
- Auto-restart failed containers (last-known desired state)
- Reconciliation on partition heal (conflict resolution)

### 7. Observability
- Prometheus metrics (`/metrics` endpoint)
- Structured JSON logs (component, context fields)
- Event streaming (real-time cluster events)

### 8. Multi-Platform
- Linux (amd64, arm64, armv7)
- macOS (amd64, arm64)
- Windows (WSL2)
- Architecture-aware scheduling (image arch matches node arch)

---

## API Specifications

### gRPC API (Primary)

**Services**:
- `WarrenAPI.CreateService` → Deploy service
- `WarrenAPI.ListServices` → Get all services
- `WarrenAPI.UpdateService` → Update service (triggers deploy strategy)
- `WarrenAPI.DeleteService` → Remove service
- `WarrenAPI.StreamEvents` → Real-time event stream

**Nodes**:
- `WarrenAPI.ListNodes` → Get cluster nodes
- `WarrenAPI.DrainNode` → Drain node (move tasks elsewhere)

**Secrets, Volumes**: Similar CRUD operations

**Transport**: mTLS-encrypted gRPC (client cert required)

### REST API (Secondary)

Via grpc-gateway:
```
POST   /v1/services              # Create service
GET    /v1/services              # List services
GET    /v1/services/{id}         # Get service
PUT    /v1/services/{id}         # Update service
DELETE /v1/services/{id}         # Delete service

GET    /v1/nodes                 # List nodes
POST   /v1/nodes/{id}/drain      # Drain node

POST   /v1/secrets               # Create secret
GET    /v1/secrets               # List secrets

GET    /v1/events                # SSE event stream
```

### CLI Commands

```bash
# Cluster management
warren cluster init                    # Start manager
warren cluster join-token worker       # Get join token
warren cluster join --token <token>    # Join as worker

# Service management
warren service create web --image nginx:latest --replicas 3
warren service list
warren service inspect web
warren service update web --replicas 5
warren service rollback web
warren service delete web

# Node management
warren node list
warren node inspect <id>
warren node drain <id>

# Secrets
warren secret create db-pass --from-file ./password.txt
warren secret list

# Volumes
warren volume create data --driver local
warren volume list

# YAML apply
warren apply -f warren.yaml
warren apply -f docker-compose.yaml  # Docker Compose compatibility

# Short alias
wrn service list  # Same as 'warren service list'
```

---

## Development Milestones

### Milestone 0: Foundation (1-2 weeks)
- POC: Raft consensus (3-node cluster)
- POC: Containerd integration (pull, create, start container)
- POC: WireGuard mesh (3 hosts, encrypted communication)
- POC: Binary size validation (< 50MB with all components)

### Milestone 1: Core Orchestration (3-4 weeks)
- Single-manager cluster functional
- Workers join via token
- Services deploy with replicas
- Basic scheduler (spread strategy)
- Health checking & auto-restart
- CLI basics (cluster, service, node commands)

### Milestone 2: High Availability (2-3 weeks)
- Multi-manager Raft cluster (3-5 nodes)
- Leader election & failover
- Worker partition tolerance (autonomous operation)
- Rolling updates & rollback
- Advanced networking (auto WireGuard mesh, DNS)

### Milestone 3: Advanced Deployment (2-3 weeks)
- Blue/green deployment
- Canary deployment
- Secrets management (encryption, distribution)
- Volume orchestration (local volumes, node affinity)
- Global services
- Docker Compose compatibility

### Milestone 4: Observability & Multi-Platform (2-3 weeks)
- Prometheus metrics
- Structured logging
- Multi-platform builds (Linux, macOS, Windows, ARM)
- Binary optimization (< 100MB)
- Memory optimization (< 256MB manager, < 128MB worker)
- Load testing (100-node cluster, 10K tasks)

### Milestone 5: Open Source & Ecosystem (2-4 weeks)
- Public GitHub release
- Documentation (user guide, API ref, architecture)
- CI/CD (automated releases)
- Package distribution (Homebrew, APT, Docker Hub)
- Community setup (Discord, GitHub Discussions)
- First 10 contributors onboarded

---

## Integration Points

### Container Runtime
- **containerd**: Container lifecycle, image management
- **Interface**: gRPC API (`/run/containerd/containerd.sock`)
- **Namespace**: Warren uses `warren` namespace in containerd

### Networking
- **WireGuard**: Kernel module (Linux 5.6+) or userspace fallback
- **iptables**: NAT rules for service VIPs and load balancing
- **DNS**: Resolv.conf updated to point to manager IPs

### Storage
- **Local Filesystem**: BoltDB files (`/var/lib/warren/`)
- **Volume Drivers**: Local (built-in), extensible interface for NFS, Ceph, etc.

### Observability
- **Prometheus**: Scrapes `/metrics` endpoint (port 9090)
- **Grafana**: Pre-built dashboards for Warren metrics
- **Log Aggregation**: Structured JSON to stdout, users ship to ELK, Loki, etc.

---

## Security Model

### mTLS Infrastructure
1. **Root CA**: Self-signed, created on cluster init
2. **Manager Certs**: Issued by root CA, 90-day expiry, auto-rotation
3. **Worker Certs**: Issued by root CA, 90-day expiry, auto-rotation
4. **Verification**: Mutual authentication (manager validates worker, worker validates manager)

### Secrets Encryption
1. **Cluster Key**: 32-byte AES key, generated on init, stored in Raft
2. **Encryption**: AES-256-GCM (authenticated encryption)
3. **Distribution**: Encrypted secrets replicated via Raft
4. **Mount**: Decrypted on worker, mounted as tmpfs (RAM-only, no disk write)

### Attack Surface
- **Minimal**: Single binary, no external services
- **Zero-Trust**: mTLS required for all communication
- **Least Privilege**: Workers only access assigned tasks, not cluster-wide state

---

## Performance Targets

| Metric | Target | Reasoning |
|--------|--------|-----------|
| Binary size | < 100MB compressed | Edge environments, slow networks |
| Manager memory | < 256MB | Shared hardware at edge sites |
| Worker memory | < 128MB | Resource-constrained edge nodes |
| API latency (p95) | < 100ms | Responsive CLI/UI experience |
| Scheduling latency | < 5s | Acceptable time to container start |
| Leader election | < 10s | Fast failover for HA |
| Overlay throughput | > 90% native | Minimal WireGuard overhead |
| 100-node cluster | Stable | Scalability for large deployments |

---

## Testing Strategy

### Unit Tests
- **Coverage**: 80%+ for core packages
- **Framework**: Go standard `testing`, `testify/assert`
- **Scope**: Individual functions, algorithms, data structures

### Integration Tests
- **Scenarios**: Cluster init, worker join, service deploy, failover
- **Environment**: Docker-in-Docker (run Warren in containers)
- **Automation**: GitHub Actions (run on every PR)

### Chaos Tests
- **Partitions**: Network partition workers, verify autonomous operation
- **Failures**: Kill leader, kill worker, verify recovery
- **Framework**: Custom (inspired by Jepsen)
- **Frequency**: Nightly (long-running soak tests)

### Load Tests
- **Scale**: 100-node cluster, 1000 services, 10K tasks
- **Metrics**: API latency, memory usage, scheduling latency
- **Tool**: Custom load generator + Prometheus

---

## Deployment Model

### Installation
```bash
# Download binary
curl -L https://github.com/cuemby/warren/releases/download/v1.0.0/warren-linux-amd64 -o warren
chmod +x warren
sudo mv warren /usr/local/bin/

# Or via package manager
brew install warren          # macOS
sudo apt install warren      # Debian/Ubuntu
```

### Cluster Setup
```bash
# Manager node
warren cluster init

# Worker nodes
warren cluster join --token <token-from-manager>

# Deploy service
warren service create web --image nginx:latest --replicas 3
```

### Upgrade Strategy
- **Binary replacement**: Stop Warren, replace binary, restart
- **Rolling manager upgrade**: Upgrade followers first, then leader
- **Raft compatibility**: Minor versions backward compatible
- **Zero-downtime**: Services keep running during manager upgrades (workers autonomous)

---

## Related Documentation

- **Product Requirements**: [specs/prd.md](../../specs/prd.md)
- **Technical Specification**: [specs/tech.md](../../specs/tech.md)
- **Milestone Plan**: [tasks/todo.md](../../tasks/todo.md)
- **Development SOPs**: [.agent/SOP/workflow.md](../SOP/workflow.md)

---

## Future Roadmap

### v1.1 - Built-in Ingress
- HTTP reverse proxy
- TLS termination (Let's Encrypt)
- Path/host-based routing

### v1.2 - Service Mesh
- Sidecar injection
- Service-to-service mTLS
- Traffic policies (retry, timeout, circuit breaking)

### v2.0 - Multi-Cluster & Plugins
- Cross-cluster service discovery
- Global load balancing
- Plugin SDK (custom schedulers, storage drivers)

---

**Version**: 1.0
**Maintained By**: Cuemby Engineering Team
**Last Updated**: 2025-10-09
