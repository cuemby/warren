# Project Architecture - Warren Container Orchestrator

## Overview

**Project Name**: Warren
**Type**: Container Orchestration Platform
**Purpose**: Simple yet feature-rich container orchestrator for edge computing, combining Docker Swarm's simplicity with Kubernetes-level features in a single binary with zero external dependencies.

**Last Updated**: 2025-10-14
**Version**: v1.3.1
**Implementation Status**: Milestones 0-7 Complete ✅ | Phase 1 Stabilization Complete ✅

---

## What This Is

Warren is a **production-ready container orchestration system** for edge computing:

**Current Status (M0-M7 COMPLETE)**:
1. **Multi-manager HA cluster** - 3-5 node Raft quorum with 2-3s failover
2. **Real container runtime** - containerd integration with full lifecycle management
3. **Secrets & volumes** - AES-256-GCM encrypted secrets, local volume driver
4. **Advanced deployment** - Rolling updates, global services, deployment strategies
5. **Production observability** - Prometheus metrics, structured logging, profiling
6. **Open source ready** - Complete docs, CI/CD, package distribution setup
7. **Production hardening (M6)** - Health checks, mTLS, DNS, resource limits, graceful shutdown
8. **Built-in Ingress (M7)** - HTTP/HTTPS routing, Let's Encrypt, advanced routing, rate limiting

**Implemented Features (M0-M7)**:
- ✅ Multi-manager Raft cluster with automatic failover (< 3s)
- ✅ Containerd integration for real container execution
- ✅ Worker join and heartbeat with token-based security
- ✅ Service deployment (replicated and global modes)
- ✅ Container scheduler with volume affinity
- ✅ Reconciler for failure detection and auto-healing
- ✅ Secrets management (AES-256-GCM, tmpfs mounting)
- ✅ Volume orchestration (local driver, node affinity)
- ✅ Prometheus metrics and structured logging
- ✅ gRPC API with 37 RPC methods
- ✅ Complete CLI with YAML apply
- ✅ Multi-platform builds (Linux/macOS, AMD64/ARM64)
- ✅ **Platform Support**:
  - ✅ Linux: Embedded containerd binaries (AMD64/ARM64)
  - ✅ macOS: Lima VM integration with Alpine Linux + containerd
- ✅ Comprehensive documentation and CI/CD
- ✅ **M6 Production Hardening**:
  - ✅ Health checks (HTTP/TCP/Exec probes with auto-replacement)
  - ✅ Published ports (host mode with iptables)
  - ✅ DNS service discovery (service and instance resolution)
  - ✅ mTLS security (CA, certificate management, TLS 1.3)
  - ✅ Resource limits (CPU shares, CFS quota, memory limits)
  - ✅ Graceful shutdown (configurable SIGTERM timeout)
- ✅ **M7 Built-in Ingress**:
  - ✅ HTTP reverse proxy (port 8000)
  - ✅ HTTPS server (port 8443)
  - ✅ Host-based routing (api.example.com, *.example.com)
  - ✅ Path-based routing (/api, /web)
  - ✅ TLS certificate management (manual upload)
  - ✅ Let's Encrypt ACME integration (HTTP-01 challenge)
  - ✅ Auto-renewal (30-day threshold, daily job)
  - ✅ Advanced routing (headers, path rewriting, rate limiting, access control)
  - ✅ Load balancing across service replicas
  - ✅ Automatic proxy headers (X-Forwarded-For, X-Real-IP, etc.)

**Future Enhancements (M8+)**:
- ⏳ WireGuard mesh networking (deferred)
- ⏳ Blue/green and canary deployments (backlog)
- ⏳ Service mesh (mTLS between services, circuit breaking, tracing)
- ⏳ Multi-cluster federation

**Philosophy**: "Docker Swarm simplicity + Kubernetes features - Kubernetes complexity"

---

## Project Structure

```
warren/
│
├── cmd/
│   └── warren/
│       ├── main.go                    # ✅ CLI entry point with all commands
│       └── apply.go                   # ✅ YAML apply command (Service, Secret, Volume)
│
├── pkg/
│   ├── api/
│   │   └── server.go                  # ✅ gRPC server with 37 RPC methods
│   │
│   ├── manager/
│   │   ├── manager.go                 # ✅ Manager with Raft consensus & multi-manager support
│   │   ├── fsm.go                     # ✅ Finite State Machine for Raft
│   │   └── token.go                   # ✅ Join token management (workers & managers)
│   │
│   ├── scheduler/
│   │   └── scheduler.go               # ✅ Container scheduler with volume affinity (5s interval)
│   │
│   ├── reconciler/
│   │   └── reconciler.go              # ✅ Failure detection reconciler (10s interval, health-aware)
│   │
│   ├── worker/
│   │   ├── worker.go                  # ✅ Worker agent with heartbeat
│   │   ├── secrets.go                 # ✅ Secret fetching and tmpfs mounting
│   │   ├── volumes.go                 # ✅ Volume mounting to containers
│   │   ├── health_monitor.go          # ✅ Health check monitoring system (M6)
│   │   └── dns.go                     # ✅ Container DNS configuration (M6)
│   │
│   ├── runtime/
│   │   └── containerd.go              # ✅ Containerd integration with resource limits (M6)
│   │
│   ├── embedded/
│   │   ├── containerd.go              # ✅ Embedded containerd manager (Linux)
│   │   └── lima.go                    # ✅ Lima VM manager (macOS, build tag: darwin)
│   │
│   ├── security/
│   │   ├── secrets.go                 # ✅ AES-256-GCM encryption/decryption
│   │   ├── secrets_test.go            # ✅ Unit tests (10/10 passing)
│   │   ├── ca.go                      # ✅ Certificate Authority (M6)
│   │   ├── ca_test.go                 # ✅ CA unit tests
│   │   ├── certs.go                   # ✅ Certificate management (M6)
│   │   └── certs_test.go              # ✅ Certificate unit tests
│   │
│   ├── volume/
│   │   ├── local.go                   # ✅ Local volume driver with node affinity
│   │   └── local_test.go              # ✅ Unit tests (10/10 passing)
│   │
│   ├── health/
│   │   ├── health.go                  # ✅ Health check interface (M6)
│   │   ├── http.go                    # ✅ HTTP health probes (M6)
│   │   ├── http_test.go               # ✅ HTTP probe tests (7/7 passing)
│   │   ├── tcp.go                     # ✅ TCP health probes (M6)
│   │   └── exec.go                    # ✅ Exec health probes (M6)
│   │
│   ├── network/
│   │   └── hostports.go               # ✅ Host mode port publishing (M6)
│   │
│   ├── dns/
│   │   ├── server.go                  # ✅ Embedded DNS server (M6)
│   │   ├── server_test.go             # ✅ DNS server tests (9/9 passing)
│   │   ├── resolver.go                # ✅ Service/instance name resolver (M6)
│   │   └── instance.go                # ✅ Instance name parsing (M6)
│   │
│   ├── ingress/
│   │   ├── proxy.go                   # ✅ HTTP/HTTPS reverse proxy (M7)
│   │   ├── router.go                  # ✅ Host/path routing engine (M7)
│   │   ├── loadbalancer.go            # ✅ Backend selection and health checks (M7)
│   │   ├── middleware.go              # ✅ Headers, rewriting, rate limiting, access control (M7)
│   │   └── acme.go                    # ✅ Let's Encrypt ACME client (M7)
│   │
│   ├── deploy/
│   │   └── deploy.go                  # ✅ Deployment strategy foundation
│   │
│   ├── metrics/
│   │   ├── metrics.go                 # ✅ Prometheus metrics registration
│   │   └── collector.go               # ✅ Custom metric collectors
│   │
│   ├── log/
│   │   └── log.go                     # ✅ Structured logging (zerolog)
│   │
│   ├── events/
│   │   └── events.go                  # ✅ Event broker (pub/sub)
│   │
│   ├── storage/
│   │   ├── store.go                   # ✅ Cluster state management
│   │   └── boltdb.go                  # ✅ BoltDB wrapper for state persistence
│   │
│   ├── client/
│   │   └── client.go                  # ✅ gRPC client for CLI
│   │
│   └── types/
│       └── types.go                   # ✅ Core data types (Cluster, Node, Service, Container, Secret, Volume)
│
├── api/
│   └── proto/
│       └── warren.proto               # ✅ Protocol buffers definitions (37 RPC methods, M7 complete)
│
├── test/
│   ├── integration/
│   │   ├── containerd_test.go         # ✅ Containerd integration tests
│   │   └── health_check_test.go       # ✅ Health check integration tests (M6)
│   └── lima/
│       ├── setup.sh                   # ✅ Lima VM cluster setup
│       ├── test-cluster.sh            # ✅ Multi-manager cluster test
│       ├── test-failover.sh           # ✅ Leader failover test
│       ├── test-load.sh               # ✅ Load testing script
│       ├── test-mtls.sh               # ✅ mTLS security test (M6)
│       └── test-ports.sh              # ✅ Port publishing test (M6)
│
├── specs/
│   ├── prd.md                         # ✅ Product Requirements Document
│   └── tech.md                        # ✅ Technical Specification
│
├── tasks/
│   ├── todo.md                        # ✅ Milestone-based development plan (M0-M5)
│   └── m5-plan.md                     # ✅ Milestone 5 detailed implementation plan
│
├── docs/
│   ├── getting-started.md             # ✅ 5-minute tutorial (M5)
│   ├── cli-reference.md               # ✅ Complete CLI documentation (M5)
│   ├── troubleshooting.md             # ✅ Troubleshooting guide (M5)
│   ├── concepts/                      # ✅ Concept guides (M5)
│   │   ├── architecture.md
│   │   ├── services.md
│   │   ├── networking.md
│   │   ├── storage.md
│   │   └── high-availability.md
│   ├── migration/                     # ✅ Migration guides (M5)
│   │   ├── from-docker-swarm.md
│   │   └── from-docker-compose.md
│   ├── profiling.md                   # ✅ pprof usage guide (M4)
│   ├── load-testing.md                # ✅ Load testing guide (M4)
│   ├── raft-tuning.md                 # ✅ Raft configuration guide (M4)
│   ├── tab-completion.md              # ✅ Shell completion guide (M4)
│   ├── health-checks.md               # ✅ Health check configuration (M6)
│   ├── resource-limits.md             # ✅ CPU and memory limits (M6)
│   ├── graceful-shutdown.md           # ✅ Graceful shutdown patterns (M6)
│   ├── port-publishing.md             # ✅ Published ports documentation (M6)
│   ├── logging.md                     # ✅ Structured logging guide (M4)
│   ├── ingress.md                     # ✅ Built-in ingress controller guide (M7)
│   ├── e2e-validation.md              # ✅ End-to-end validation procedures (M7)
│   └── performance-benchmarking.md    # ✅ Performance benchmarking guide (M7)
│
├── examples/
│   ├── nginx-service.yaml             # ✅ Simple web service (M4)
│   ├── complete-app.yaml              # ✅ Complete app with secrets/volumes (M4)
│   ├── ingress-basic.yaml             # ✅ HTTP ingress with load balancing (M7)
│   ├── ingress-https.yaml             # ✅ HTTPS with Let's Encrypt (M7)
│   ├── health-checks.yaml             # ✅ HTTP/TCP/Exec probes (M6)
│   ├── resource-limits.yaml           # ✅ CPU/memory management (M6)
│   ├── secrets-volumes.yaml           # ✅ Secrets and persistent storage (M3)
│   ├── multi-service-app.yaml         # ✅ Full 3-tier application (M4)
│   ├── ha-cluster.yaml                # ✅ Production HA setup guide (M2)
│   └── advanced-routing.yaml          # ✅ Advanced ingress features (M7)
│
├── packaging/
│   ├── homebrew/                      # ✅ Homebrew formula and guide (M5)
│   └── apt/                           # ✅ APT packaging guide (M5)
│
├── .github/
│   ├── workflows/                     # ✅ CI/CD workflows (M5)
│   │   ├── test.yml
│   │   ├── release.yml
│   │   └── pr.yml
│   ├── PULL_REQUEST_TEMPLATE.md       # ✅ PR template (M5)
│   └── ISSUE_TEMPLATE/                # ✅ Issue templates (M5)
│
├── poc/
│   ├── raft/                          # ✅ Raft consensus POC (M0)
│   ├── containerd/                    # ✅ Containerd runtime POC (M0)
│   ├── wireguard/                     # ✅ WireGuard networking POC (M0)
│   └── binary-size/                   # ✅ Binary size POC (M0)
│
├── .agent/                            # ✅ AI development framework
│   ├── README.md                      # Documentation index
│   ├── SOP/                           # Standard Operating Procedures (10 files)
│   ├── System/                        # System documentation (5 files)
│   └── Tasks/                         # Task templates
│
├── .claude/                           # ✅ Claude Code configuration
│   └── commands/                      # Custom slash commands (13 commands)
│
├── Makefile                           # ✅ Build automation with multi-platform support
├── Dockerfile                         # ✅ Docker image (multi-stage build) (M5)
├── go.mod                             # ✅ Go module definition
├── go.sum                             # ✅ Go dependency checksums
├── CLAUDE.md                          # ✅ AI-specific instructions
├── README.md                          # ✅ Production-ready README (M5)
├── LICENSE                            # ✅ Apache 2.0 (M5)
├── CODE_OF_CONDUCT.md                 # ✅ Contributor Covenant (M5)
├── CONTRIBUTING.md                    # ✅ Contribution guidelines (M5)
└── SECURITY.md                        # ✅ Security policy (M5)
```

**Project Stats** (v1.1.1):
- **20 packages** in `pkg/` directory
- **77 Go files** total (45 source, 20 doc.go, 12 tests)
- **37 gRPC methods** in Protocol Buffers API
- **10 YAML examples** covering all major features
- **33 documentation files** (user guides, concepts, migrations)
- **181 commits** since October 2025

**Legend**:
- ✅ Implemented (M0-M7 complete)
- (M0-M7) indicates milestone where feature was added
- (M6) = Production Hardening, (M7) = Built-in Ingress

---

## Technology Stack

### Core Language & Runtime
- **Language**: Go 1.24+ (cross-platform, performance, cloud-native ecosystem)
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

### Milestone 0: Foundation ✅ **COMPLETE** (2025-10-09)
- ✅ POC: Raft consensus (3-node cluster) - [poc/raft/](../../poc/raft/)
- ✅ POC: Containerd integration - [poc/containerd/](../../poc/containerd/)
- ✅ POC: WireGuard mesh - [poc/wireguard/](../../poc/wireguard/)
- ✅ POC: Binary size validation - [poc/binary-size/](../../poc/binary-size/)
- ✅ 5 Architecture Decision Records - [docs/adr/](../../docs/adr/)

### Milestone 1: Core Orchestration ✅ **COMPLETE** (2025-10-09)
- ✅ Single-manager cluster with Raft consensus
- ✅ Worker join and heartbeat mechanism
- ✅ Service deployment with container creation
- ✅ Container scheduler (round-robin, 5s interval)
- ✅ Reconciler (failure detection, 10s interval)
- ✅ gRPC API (25+ methods)
- ✅ Full CLI (cluster, service, node commands)
- ✅ Integration tests
- ✅ 3,900+ lines of production code across 16 files

### Milestone 2: High Availability ✅ **COMPLETE** (2025-10-10)
- ✅ Multi-manager Raft cluster (3-5 nodes) - [pkg/manager/](../../pkg/manager/)
- ✅ Token-based secure joining - [pkg/manager/token.go](../../pkg/manager/token.go)
- ✅ Leader election & automatic failover (< 3s)
- ✅ Containerd integration - [pkg/runtime/containerd.go](../../pkg/runtime/containerd.go)
- ✅ Lima VM testing infrastructure - [test/lima/](../../test/lima/)
- ✅ 3-manager + 2-worker cluster validated

### Milestone 3: Advanced Deployment & Secrets ✅ **COMPLETE** (2025-10-10)
- ✅ Secrets management (AES-256-GCM) - [pkg/security/secrets.go](../../pkg/security/secrets.go)
- ✅ Secrets mounted as tmpfs - [pkg/worker/secrets.go](../../pkg/worker/secrets.go)
- ✅ Volume orchestration (local driver) - [pkg/volume/local.go](../../pkg/volume/local.go)
- ✅ Volume node affinity in scheduler
- ✅ Global services (one container per node)
- ✅ Deployment strategy foundation - [pkg/deploy/deploy.go](../../pkg/deploy/deploy.go)
- ✅ 11 commits with full test coverage

### Milestone 4: Observability & Multi-Platform ✅ **COMPLETE** (2025-10-10)
- ✅ Prometheus metrics (cluster, Raft, container) - [pkg/metrics/](../../pkg/metrics/)
- ✅ Structured JSON logging (zerolog) - [pkg/log/log.go](../../pkg/log/log.go)
- ✅ Event streaming foundation - [pkg/events/events.go](../../pkg/events/events.go)
- ✅ Multi-platform builds (Linux/macOS, AMD64/ARM64)
- ✅ Binary size: ~35MB (< 100MB target)
- ✅ Load testing infrastructure - [test/lima/test-load.sh](../../test/lima/test-load.sh)
- ✅ Performance validation: 10 svc/s, 66ms API latency
- ✅ Raft failover tuning: 2-3s (< 10s target)
- ✅ Tab completion (Bash, Zsh, Fish, PowerShell)
- ✅ YAML apply support - [cmd/warren/apply.go](../../cmd/warren/apply.go)
- ✅ pprof profiling support
- ✅ 4 comprehensive guides: profiling, load testing, Raft tuning, tab completion

### Milestone 5: Open Source & Ecosystem ✅ **COMPLETE** (2025-10-10)
- ✅ Public GitHub repository (https://github.com/cuemby/warren)
- ✅ LICENSE (Apache 2.0), CODE_OF_CONDUCT, CONTRIBUTING, SECURITY
- ✅ 14 comprehensive documentation files (~12,000+ lines)
  - Getting started, CLI reference, 5 concept guides
  - 2 migration guides (Swarm, Compose), troubleshooting
- ✅ GitHub Actions CI/CD (test, release, PR validation)
- ✅ Package distribution setup (Homebrew, APT, Docker Hub)
- ✅ Issue templates (bug, feature, docs)
- ✅ Production-ready README
- ✅ 10 commits for M5

**Total Achievement (M0-M5)**:
- 6,439 lines of Go code across 19 packages
- 14+ user-facing documentation files
- 10+ internal documentation files (.agent)
- Complete CI/CD automation
- Multi-platform build support
- Production-ready for edge deployment
- Open source community infrastructure

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

**Version**: 1.6 (Updated for Milestone 6 Progress)
**Maintained By**: Cuemby Engineering Team
**Last Updated**: 2025-10-11
