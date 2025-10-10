# Product Requirements Document (PRD) - Warren

**Document Version:** 1.0
**Last Updated:** 2025-10-09
**Status:** Approved
**Author:** Cuemby Engineering Team
**Stakeholders:** Platform Team, Edge Operations, Open Source Community
**Priority:** [CRITICAL]
**Context Complexity:** XL

---

## Executive Summary

Warren is a next-generation container orchestration platform that combines the simplicity of Docker Swarm with the feature richness of Kubernetes, delivered as a single binary with zero external dependencies. Built for edge computing with telco-grade reliability, Warren enables teams to deploy and manage containerized workloads across distributed infrastructure without the operational complexity of existing solutions.

**Quick Facts**:

- **Problem**: Existing orchestrators are either too simple (Docker Swarm - now closed source) or too complex (Kubernetes requires extensive add-ons and expertise)
- **Solution**: Self-contained, feature-rich orchestrator with built-in security, observability, and multi-deployment strategies, shipped as a single binary
- **Impact**: Reduce orchestration complexity by 80% while maintaining enterprise-grade features for edge deployments
- **Context Complexity**: XL - Distributed systems, consensus algorithms, container runtimes, networking, edge constraints

---

## Problem Statement

### The Problem

Container orchestration has bifurcated into two unsatisfactory extremes: overly simplistic tools that lack production features, and overly complex platforms that require dedicated teams to operate. Edge computing deployments face additional challenges with resource constraints, network intermittency, and geographic distribution that existing solutions don't adequately address.

**Current Pain Points**:

- **Docker Swarm closure**: Docker Swarm went closed source, leaving users without a simple open-source orchestration option
- **Kubernetes complexity**: K8s requires 15+ add-ons for basic production features (ingress, monitoring, secrets, service mesh, etc.), each adding operational overhead
- **Edge incompatibility**: Existing orchestrators aren't optimized for edge constraints (limited resources, intermittent connectivity, geographic distribution)
- **Dependency hell**: Current solutions require external databases (etcd), load balancers, monitoring stacks, and complex networking plugins
- **Binary bloat**: Solutions that do bundle features create 500MB+ binaries with excessive memory footprints

**Who Experiences This**:

- **Telco edge operators**: Managing distributed compute at cell towers and edge data centers, need reliability with minimal operational overhead (daily impact)
- **IoT platform teams**: Running orchestration on resource-constrained edge devices, struggle with existing tools' resource requirements (daily impact)
- **SMB infrastructure teams**: Small teams (2-5 engineers) need production-grade orchestration without K8s complexity (daily impact)
- **Platform engineers**: Building internal platforms, frustrated by external dependencies and integration complexity (daily impact)

### Current Solution

**Workarounds**:

- **Kubernetes + 15 add-ons**: Teams cobble together K8s, Prometheus, Grafana, cert-manager, ingress-nginx, etc. - requires dedicated SRE team
- **Docker Swarm (abandoned)**: Teams stuck on old versions or forced to migrate to proprietary alternatives
- **Cloud-specific solutions**: Lock into AWS ECS/Fargate, Azure Container Instances, losing cloud portability
- **Manual container management**: Some teams give up on orchestration entirely, manage containers with systemd/docker-compose

**Limitations**:

- **Operational complexity**: Requires specialized knowledge across multiple systems (K8s + ecosystem)
- **Resource overhead**: Minimal K8s cluster needs 4GB+ RAM just for control plane
- **Vendor lock-in**: Cloud-specific solutions prevent multi-cloud or hybrid deployments
- **No edge optimization**: Solutions designed for data center, not edge constraints
- **Update complexity**: Rolling updates, canary deployments require additional tooling

### Opportunity

**Value Proposition**:

- **For edge operators**: Deploy production-grade orchestration on resource-constrained edge hardware with autonomous operation during network partitions
- **For platform teams**: Build internal platforms with zero-dependency orchestrator that "just works" - single binary deployment, no ecosystem assembly required
- **For the industry**: Fill the gap between Docker Swarm (too simple) and Kubernetes (too complex) with modern, open-source alternative
- **For Cuemby**: Establish thought leadership in edge orchestration, build community around simple-yet-powerful tooling (rabbit/warren branding alignment)

---

## Goals and Objectives

### Business Goals

- **Market positioning**: Become the default orchestrator for edge computing deployments within 12 months of launch (measured by GitHub stars, Docker Hub pulls)
- **Open source adoption**: Achieve 10K+ GitHub stars and 50+ active contributors within first year
- **Community building**: Establish Warren as the "sane alternative" to Kubernetes, similar to how Caddy became the alternative to nginx/Apache

### User Goals

- **Simplicity**: Deploy production orchestration in under 5 minutes with single binary and 3 CLI commands
- **Feature completeness**: Access rolling updates, canary deployments, secrets management, service mesh, and observability without installing add-ons
- **Edge reliability**: Run workloads on edge nodes that autonomously recover from network partitions and hardware failures
- **Migration path**: Migrate from Docker Swarm or Docker Compose with minimal config changes (days, not months)

### Success Metrics

| Metric | Current Baseline | Target | Measurement Method |
|--------|-----------------|--------|-------------------|
| Binary size | N/A | < 100MB | CI build artifacts |
| Memory footprint (control plane) | N/A | < 256MB per manager node | Runtime profiling |
| Time to first deployment | N/A | < 5 minutes (cluster init → service running) | End-to-end test automation |
| GitHub stars | 0 | 10,000 in 12 months | GitHub API |
| Production deployments | 0 | 100+ reported in 12 months | Telemetry opt-in |
| Docker Hub pulls | 0 | 1M+ in 12 months | Docker Hub stats |

**Success Criteria**:

- [ ] Binary < 100MB, memory < 256MB per manager
- [ ] All three deployment strategies (rolling, blue/green, canary) work without plugins
- [ ] Edge nodes recover autonomously from 30+ minute network partitions
- [ ] Users migrate from Docker Compose in < 1 day
- [ ] Community contributions (10+ external PRs in first 6 months)
- [ ] Zero CVEs in first 6 months

---

## User Personas

### Primary Persona: Edge Platform Engineer

**Role**: Platform/DevOps Engineer managing edge infrastructure
**Technical Proficiency**: High
**Context Familiarity**: Expert in containers, intermediate in orchestration

**Goals**:

- Deploy orchestration across 50-500 edge locations without dedicated SRE team
- Ensure services stay running during network partitions (cell towers, remote sites)
- Minimize resource usage on edge hardware (often shared with customer workloads)
- Achieve 99.9% uptime without complex multi-region failover setups

**Pain Points**:

- K8s control plane eats 4GB+ RAM, unacceptable for edge hardware
- Managing etcd, ingress controllers, cert-manager across hundreds of sites is operationally prohibitive
- Docker Swarm was perfect but now closed source and abandoned
- Need built-in observability but Prometheus/Grafana stack adds complexity

**Behaviors**:

- Prefers infrastructure-as-code (YAML manifests) but needs CLI for troubleshooting
- Monitors health via metrics scraping (Prometheus ecosystem familiar)
- Expects zero-touch operations - services should self-heal automatically
- Values simplicity over features - will sacrifice advanced features for reliability

**Quote**: *"I need Kubernetes reliability without Kubernetes complexity. Docker Swarm was close, but it's dead and lacked features. Warren needs to be that perfect middle ground."*

### Secondary Persona: SMB Infrastructure Lead

**Role**: Infrastructure lead at 20-200 person company
**Technical Proficiency**: Medium-High
**Context Familiarity**: Familiar with Docker, new to orchestration

**Goals**:

- Run production workloads with 2-3 person ops team (no dedicated SRE)
- Avoid cloud vendor lock-in while maintaining cloud deployment option
- Implement modern deployment practices (canary, blue/green) without specialized tools
- Keep infrastructure costs low (avoid managed K8s, reduce operational overhead)

**Pain Points**:

- K8s requires hiring specialists or months of training
- Cloud-specific solutions lock into single vendor
- Docker Compose doesn't scale beyond single host
- Existing "simple" orchestrators lack production features (secrets, rolling updates)

**Behaviors**:

- Starts with Docker Compose for local dev, needs orchestration for production scaling
- Prefers managed solutions but budget constraints force self-hosted
- CLI-first workflow, YAML for repeatability
- Needs comprehensive documentation and examples - limited time for trial-and-error

**Quote**: *"We're a small team running production services. We need something between Docker Compose and Kubernetes - powerful enough for production, simple enough for 2 people to manage."*

### Tertiary Persona: Open Source Contributor

**Role**: Distributed systems enthusiast, Go/Rust developer
**Technical Proficiency**: High
**Context Familiarity**: Expert in distributed systems, consensus algorithms

**Goals**:

- Contribute to well-architected orchestration platform
- Learn distributed systems patterns (Raft, gossip protocols, schedulers)
- Build reputation in cloud-native ecosystem
- Influence design of next-generation tooling

**Pain Points**:

- K8s codebase too large/complex to contribute meaningfully
- Smaller projects lack architectural rigor or production adoption
- Want to work on "real" distributed systems, not toy projects

**Behaviors**:

- Reviews architecture before considering contribution
- Expects clear contribution guidelines, architectural docs
- Values code quality, comprehensive testing
- Motivated by technical challenges (consensus, scheduling algorithms)

**Quote**: *"I want to contribute to a real distributed system that people actually use. Warren's architecture needs to be clean enough to understand and robust enough to matter."*

---

## User Stories

### Epic 1: Cluster Management

**Epic Context Complexity**: L
**Priority**: [CRITICAL]

#### Story 1.1: Initialize First Manager Node

**As a** platform engineer
**I want** to initialize a Warren cluster with a single command
**So that** I can get started without complex configuration files or prerequisites

**Acceptance Criteria**:

- [ ] **GIVEN** a Linux/macOS/Windows host with containerd installed, **WHEN** I run `warren cluster init`, **THEN** cluster initializes in < 30 seconds with TLS certificates auto-generated
- [ ] **GIVEN** successful initialization, **WHEN** I run `warren node list`, **THEN** I see the manager node in "Ready" status
- [ ] **GIVEN** cluster initialization, **WHEN** I check memory usage, **THEN** manager process uses < 128MB RAM

**Priority**: P0 (Critical)
**Context Complexity**: M
**Dependencies**: Raft consensus, certificate generation, BoltDB storage

#### Story 1.2: Join Worker Nodes

**As a** platform engineer
**I want** to join worker nodes with a single token
**So that** I can scale my cluster without managing certificates manually

**Acceptance Criteria**:

- [ ] **GIVEN** initialized cluster, **WHEN** I run `warren cluster join-token worker`, **THEN** I receive a time-limited token and join command
- [ ] **GIVEN** join token and command, **WHEN** I run join command on new host, **THEN** worker joins cluster in < 15 seconds with automatic mTLS setup
- [ ] **GIVEN** joined worker, **WHEN** network partition occurs, **THEN** worker maintains existing services and reconnects automatically when network restored

**Priority**: P0 (Critical)
**Context Complexity**: L
**Dependencies**: mTLS implementation, Raft membership changes, WireGuard networking

#### Story 1.3: Multi-Manager HA

**As a** platform engineer
**I want** to run multiple manager nodes for high availability
**So that** cluster survives manager node failures without downtime

**Acceptance Criteria**:

- [ ] **GIVEN** single-manager cluster, **WHEN** I join additional managers (3 or 5 total), **THEN** Raft quorum established and leader elected
- [ ] **GIVEN** 3-manager cluster, **WHEN** leader fails, **THEN** new leader elected in < 10 seconds and services continue running
- [ ] **GIVEN** manager failure, **WHEN** failed node rejoins, **THEN** it syncs state and rejoins quorum without manual intervention

**Priority**: P0 (Critical)
**Context Complexity**: XL
**Dependencies**: Raft consensus, leader election, state replication

---

### Epic 2: Service Deployment

**Epic Context Complexity**: L
**Priority**: [CRITICAL]

#### Story 2.1: Deploy Replicated Service

**As a** platform engineer
**I want** to deploy a service with multiple replicas
**So that** workload is distributed and survives node failures

**Acceptance Criteria**:

- [ ] **GIVEN** running cluster, **WHEN** I run `warren service create web --image nginx:latest --replicas 3`, **THEN** 3 containers start across available nodes in < 30 seconds
- [ ] **GIVEN** deployed service, **WHEN** I run `warren service list`, **THEN** I see service status with 3/3 replicas running
- [ ] **GIVEN** replicated service, **WHEN** one replica fails, **THEN** scheduler starts replacement replica in < 10 seconds

**Priority**: P0 (Critical)
**Context Complexity**: M
**Dependencies**: Scheduler, containerd integration, health checking

#### Story 2.2: Service Discovery and Load Balancing

**As a** developer
**I want** services to discover each other by name
**So that** I don't hardcode IP addresses in configurations

**Acceptance Criteria**:

- [ ] **GIVEN** service named "web" running, **WHEN** another service queries "web" via DNS, **THEN** DNS resolves to healthy replica IPs
- [ ] **GIVEN** multiple replicas, **WHEN** clients connect to service VIP, **THEN** traffic distributes across healthy replicas (round-robin)
- [ ] **GIVEN** replica failure, **WHEN** clients connect, **THEN** failed replica excluded from load balancing within 5 seconds

**Priority**: P0 (Critical)
**Context Complexity**: L
**Dependencies**: DNS service, WireGuard mesh, health checking

#### Story 2.3: YAML-Based Deployment

**As a** platform engineer
**I want** to define services in YAML files
**So that** I can version control configurations and reproduce deployments

**Acceptance Criteria**:

- [ ] **GIVEN** warren.yaml file with service definition, **WHEN** I run `warren apply -f warren.yaml`, **THEN** service deploys matching specification
- [ ] **GIVEN** existing service, **WHEN** I modify warren.yaml and reapply, **THEN** Warren performs rolling update to new spec
- [ ] **GIVEN** docker-compose.yaml file, **WHEN** I run `warren apply -f docker-compose.yaml`, **THEN** Warren converts and deploys services (migration path)

**Priority**: P1 (High)
**Context Complexity**: M
**Dependencies**: YAML parser, schema validation, Docker Compose compatibility layer

---

### Epic 3: Deployment Strategies

**Epic Context Complexity**: L
**Priority**: [REQUIRED]

#### Story 3.1: Rolling Updates

**As a** developer
**I want** to update services without downtime
**So that** users don't experience service interruptions

**Acceptance Criteria**:

- [ ] **GIVEN** service v1 with 5 replicas, **WHEN** I update to v2 with `--update-strategy rolling`, **THEN** replicas update one-by-one with 10s delay between updates
- [ ] **GIVEN** rolling update in progress, **WHEN** new replica fails health check, **THEN** rollout pauses and previous version remains active
- [ ] **GIVEN** rolling update, **WHEN** I run `warren service rollback web`, **THEN** service reverts to previous version using same rolling strategy

**Priority**: P0 (Critical)
**Context Complexity**: M
**Dependencies**: Health checking, scheduler, version tracking

#### Story 3.2: Blue/Green Deployment

**As a** platform engineer
**I want** to deploy new version alongside old version
**So that** I can switch traffic instantly or rollback with zero downtime

**Acceptance Criteria**:

- [ ] **GIVEN** service v1 (blue), **WHEN** I deploy v2 with `--update-strategy blue-green`, **THEN** v2 deploys fully before traffic switches
- [ ] **GIVEN** both versions running, **WHEN** I run `warren service promote web`, **THEN** traffic switches to v2 instantly and v1 scales down after 30s
- [ ] **GIVEN** v2 receiving traffic, **WHEN** I detect issues and run `warren service rollback web`, **THEN** traffic switches back to v1 instantly

**Priority**: P1 (High)
**Context Complexity**: L
**Dependencies**: Service versioning, traffic routing, health validation

#### Story 3.3: Canary Deployment

**As a** SRE
**I want** to route percentage of traffic to new version
**So that** I can validate changes with subset of users before full rollout

**Acceptance Criteria**:

- [ ] **GIVEN** service v1, **WHEN** I deploy v2 with `--canary-weight 10`, **THEN** 10% of requests route to v2, 90% to v1
- [ ] **GIVEN** canary deployment, **WHEN** I gradually increase canary weight (10→50→100), **THEN** traffic shifts progressively and old version scales down
- [ ] **GIVEN** canary showing errors, **WHEN** I run `warren service rollback web`, **THEN** all traffic returns to v1 and v2 terminates

**Priority**: P1 (High)
**Context Complexity**: L
**Dependencies**: Traffic splitting, metrics integration, weighted load balancing

---

### Epic 4: Edge Resilience

**Epic Context Complexity**: XL
**Priority**: [CRITICAL]

#### Story 4.1: Autonomous Operation During Partition

**As an** edge operator
**I want** edge nodes to continue running services during network partition
**So that** customer workloads stay online even when control plane unreachable

**Acceptance Criteria**:

- [ ] **GIVEN** healthy service on edge node, **WHEN** network to managers partitioned for 30+ minutes, **THEN** service continues running and restarts on failure
- [ ] **GIVEN** network partition, **WHEN** container crashes, **THEN** local Warren agent restarts it based on last-known desired state
- [ ] **GIVEN** extended partition (24+ hours), **WHEN** network restored, **THEN** node syncs state, reconciles differences, and rejoins cluster without data loss

**Priority**: P0 (Critical)
**Context Complexity**: XL
**Dependencies**: Local state caching, Raft partition handling, conflict resolution

#### Story 4.2: Resource-Constrained Scheduling

**As an** edge operator
**I want** Warren to respect resource limits on edge hardware
**So that** orchestration doesn't starve customer workloads

**Acceptance Criteria**:

- [ ] **GIVEN** node with 2GB RAM, **WHEN** I schedule service with 1.5GB limit, **THEN** Warren reserves resources and prevents over-scheduling
- [ ] **GIVEN** resource-constrained node, **WHEN** scheduler evaluates placement, **THEN** services only placed if resources available (CPU, memory, disk)
- [ ] **GIVEN** node resource exhaustion, **WHEN** services need placement, **THEN** Warren schedules to other nodes and alerts via metrics

**Priority**: P0 (Critical)
**Context Complexity**: M
**Dependencies**: Resource accounting, scheduler bin-packing, node capacity tracking

---

### Epic 5: Built-in Observability

**Epic Context Complexity**: M
**Priority**: [REQUIRED]

#### Story 5.1: Prometheus Metrics Endpoint

**As a** platform engineer
**I want** Warren to expose Prometheus metrics
**So that** I can scrape metrics without deploying separate monitoring stack

**Acceptance Criteria**:

- [ ] **GIVEN** running Warren manager, **WHEN** I scrape `/metrics` endpoint, **THEN** I receive cluster health metrics (node count, service count, task states)
- [ ] **GIVEN** service deployed, **WHEN** I scrape service metrics, **THEN** I receive container metrics (CPU, memory, network, restart count)
- [ ] **GIVEN** Raft cluster, **WHEN** I scrape manager metrics, **THEN** I receive consensus metrics (leader status, quorum health, replication lag)

**Priority**: P1 (High)
**Context Complexity**: S
**Dependencies**: Prometheus client library, metrics collection

#### Story 5.2: Structured Logging

**As a** platform engineer
**I want** Warren to output structured JSON logs
**So that** I can parse logs programmatically and send to my logging backend

**Acceptance Criteria**:

- [ ] **GIVEN** Warren running, **WHEN** events occur, **THEN** logs output in JSON format with timestamp, level, component, message
- [ ] **GIVEN** log output, **WHEN** I configure log level (debug/info/warn/error), **THEN** Warren respects level and filters appropriately
- [ ] **GIVEN** service events, **WHEN** state changes occur, **THEN** structured logs include service ID, node ID, task ID for correlation

**Priority**: P1 (High)
**Context Complexity**: XS
**Dependencies**: Structured logging library (zerolog, zap)

---

## Feature Requirements

### MoSCoW Prioritization

#### Must Have (Core Features) - [CRITICAL]

- **REQ-001: Single Binary Distribution**
  - **Rationale**: Core differentiator - zero external dependencies
  - **Context Complexity**: L
  - **Dependencies**: Go build system, embedded assets (certs, defaults)
  - **Acceptance Criteria**: Single binary < 100MB runs on Linux/macOS/Windows/ARM without external dependencies

- **REQ-002: Raft-Based Consensus**
  - **Rationale**: Multi-manager HA requires distributed consensus
  - **Context Complexity**: XL
  - **Dependencies**: hashicorp/raft library, BoltDB storage backend
  - **Acceptance Criteria**: 3-5 manager quorum survives minority failure (1 of 3, 2 of 5), elects leader < 10s

- **REQ-003: Containerd Integration**
  - **Rationale**: CRI-compatible container runtime, Docker-independent
  - **Context Complexity**: M
  - **Dependencies**: containerd API client
  - **Acceptance Criteria**: Pull images, create/destroy containers, stream logs via containerd

- **REQ-004: WireGuard Overlay Network**
  - **Rationale**: Secure, performant mesh networking for cross-node communication
  - **Context Complexity**: L
  - **Dependencies**: WireGuard kernel module (Linux 5.6+)
  - **Acceptance Criteria**: Containers on different nodes communicate via encrypted overlay, DNS resolution works

- **REQ-005: Service Scheduler**
  - **Rationale**: Core orchestration - place tasks on appropriate nodes
  - **Context Complexity**: L
  - **Dependencies**: Resource tracking, node health, placement algorithms
  - **Acceptance Criteria**: Schedule replicas across nodes, respect resource limits, rebalance on node failure

- **REQ-006: Zero-Config mTLS**
  - **Rationale**: Security by default, no manual certificate management
  - **Context Complexity**: M
  - **Dependencies**: Certificate generation (crypto/x509), automatic rotation
  - **Acceptance Criteria**: Manager-worker communication encrypted, certificates auto-generated and rotated

- **REQ-007: Health Checking & Auto-Recovery**
  - **Rationale**: Services must self-heal automatically
  - **Context Complexity**: M
  - **Dependencies**: HTTP/TCP health probes, restart policies
  - **Acceptance Criteria**: Failed containers restart per policy, unhealthy replicas removed from load balancing

#### Should Have (Important) - [REQUIRED]

- **REQ-008: Docker Compose Compatibility**
  - **Rationale**: Migration path from existing Docker Compose users
  - **Context Complexity**: M
  - **Dependencies**: Compose file parser, schema mapping to Warren format
  - **Acceptance Criteria**: `warren apply -f docker-compose.yaml` deploys services correctly

- **REQ-009: Global Services**
  - **Rationale**: Edge use case - run agent/daemon on every node
  - **Context Complexity**: S
  - **Dependencies**: Scheduler, node tracking
  - **Acceptance Criteria**: Global service deploys one replica per node, scales automatically with cluster

- **REQ-010: Secrets Management**
  - **Rationale**: Secure credential distribution to containers
  - **Context Complexity**: M
  - **Dependencies**: Encrypted storage, tmpfs mounting in containers
  - **Acceptance Criteria**: Secrets stored encrypted, mounted to containers as files, rotated without downtime

- **REQ-011: Volume Orchestration**
  - **Rationale**: Stateful services need persistent storage
  - **Context Complexity**: L
  - **Dependencies**: Volume drivers, node affinity (local volumes), distributed storage interface
  - **Acceptance Criteria**: Create/attach/detach volumes, local and network-backed volumes supported

- **REQ-012: CLI with Short Alias**
  - **Rationale**: Kubectl-style UX with convenience alias
  - **Context Complexity**: S
  - **Dependencies**: Cobra CLI framework
  - **Acceptance Criteria**: `warren service create` works, `wrn` alias also works, tab completion available

#### Could Have (Nice to Have) - [RECOMMENDED]

- **REQ-013: Multi-Architecture Support**
  - **Rationale**: Edge includes ARM devices (Raspberry Pi, ARM servers)
  - **Context Complexity**: M
  - **Dependencies**: Cross-compilation, architecture-aware scheduling
  - **Acceptance Criteria**: Binary runs on amd64, arm64, armv7, schedule respects arch constraints

- **REQ-014: Built-in Ingress Controller**
  - **Rationale**: HTTP routing without external load balancer
  - **Context Complexity**: L
  - **Dependencies**: HTTP reverse proxy, TLS termination, routing rules
  - **Acceptance Criteria**: Route external traffic to services based on hostname/path, TLS termination

- **REQ-015: Service Mesh (mTLS between services)**
  - **Rationale**: Zero-trust security between application services
  - **Context Complexity**: XL
  - **Dependencies**: Sidecar injection, certificate distribution, traffic interception
  - **Acceptance Criteria**: Service-to-service communication encrypted, mutual authentication enforced

#### Won't Have (This Iteration)

- **Multi-cluster federation**: Single cluster spanning geos is sufficient initially, federation adds significant complexity
- **Kubernetes API compatibility**: Warren has its own API/UX, K8s compatibility would compromise simplicity
- **Custom schedulers/plugins**: Built-in scheduler sufficient for initial use cases, extensibility deferred
- **Windows container support**: Linux containers only initially, Windows adds complexity
- **Built-in CI/CD**: Orchestration only, users integrate with external CI/CD (GitHub Actions, GitLab CI, etc.)

---

### Non-Functional Requirements

#### Performance

- **Control plane latency**: < 100ms API response time (p95) for cluster < 100 nodes
- **Scheduling latency**: < 5 seconds to schedule and start task on available node
- **State replication**: Raft log replication < 100ms between managers (p95)
- **Resource overhead**: Manager process < 256MB RAM, worker agent < 128MB RAM
- **Network throughput**: Overlay network within 10% of host network performance (WireGuard overhead)
- **Context**: Edge deployments share hardware with customer workloads, overhead must be minimal

#### Security

- **mTLS by default**: All manager-worker communication encrypted with mutual authentication
- **Certificate rotation**: Automatic rotation every 90 days, no service disruption
- **Secrets encryption**: Secrets encrypted at rest (AES-256), in transit (TLS), in memory (tmpfs)
- **Least privilege**: Worker nodes can't access cluster-wide state, only their assigned tasks
- **Audit logging**: All administrative actions logged with user/timestamp/action
- **Context**: Telco edge requires defense-in-depth, zero-trust security model

#### Reliability

- **High availability**: 99.9% uptime with 3-manager setup (tolerates 1 manager failure)
- **Data durability**: Raft ensures committed state survives minority failure, no data loss
- **Partition tolerance**: Edge nodes operate autonomously during network partition, reconcile on reconnect
- **Graceful degradation**: Read-only mode during quorum loss, full recovery on quorum restore
- **Context**: Edge environments have variable network quality, system must handle partitions gracefully

#### Compatibility

- **Operating Systems**: Linux (kernel 5.6+), macOS (12+), Windows (WSL2) - native Windows deferred
- **Architectures**: amd64, arm64 (primary), armv7 (stretch goal)
- **Container Runtimes**: containerd 1.6+, Docker Engine support via containerd shim
- **Networking**: WireGuard kernel module required (Linux), userspace fallback for macOS/Windows
- **Context**: Cross-platform support critical for developer experience, but edge deployments primarily Linux

---

## User Journey

### Primary Scenario: Deploy First Service on New Cluster

**Context Complexity**: M

```text
1. Entry Point
   → User has downloaded `warren` binary (GitHub releases, package manager)
   → Context: Familiar with Docker, new to Warren
   → Environment: Single Linux VM or laptop

2. Cluster Initialization
   → User runs: `warren cluster init`
   → Action: Warren generates certificates, initializes Raft, starts manager
   → System: Outputs join token for workers, cluster ID, API endpoint
   → Feedback: "✓ Manager initialized at 192.168.1.10:2377"

3. Service Deployment (CLI)
   → User runs: `warren service create web --image nginx:latest --replicas 3 --port 80:8080`
   → Action: Warren schedules 3 tasks, pulls image, starts containers
   → System: Creates overlay network, service VIP, DNS entries
   → Feedback: "✓ Service 'web' created, 3/3 replicas running"

4. Verification
   → User runs: `warren service list`
   → Action: Lists services with replica status
   → System: Shows: "web | 3/3 | nginx:latest | 8080:80"
   → User curls: `http://localhost:8080`
   → System: Returns nginx welcome page

5. Success State
   → User achieves: Running service in < 5 minutes
   → System: Service healthy, load balanced across replicas
   → Next: Deploy additional services, scale cluster, configure rolling updates
```

### Edge Case: Network Partition at Edge Node

**Context Complexity**: L

```text
1. Initial State
   → Cluster: 3 managers (cloud), 10 workers (edge sites)
   → Service: "sensor-processor" running on each edge worker (global service)

2. Partition Occurs
   → Edge site loses WAN connection to cloud managers
   → Worker node detects partition (heartbeat timeout)
   → System: Enters autonomous mode, caches last-known state

3. Autonomous Operation
   → Container crashes on partitioned worker
   → Worker detects crash via health check
   → System: Restarts container per restart policy (last-known desired state)
   → User sees: Service continues processing sensor data locally

4. Partition Resolves
   → WAN connection restored after 2 hours
   → Worker reconnects to Raft leader
   → System: Syncs state delta, reconciles any drift
   → Feedback: Logs show "reconnected to cluster, state synced"

5. Recovery Complete
   → Service continues running, no data loss
   → Metrics show partition duration and auto-recovery
   → User: No manual intervention required
```

### User Journey Map

| Stage | User Action | System Response | Emotions | Opportunities |
|-------|------------|-----------------|----------|---------------|
| Discovery | Download Warren binary | Single file, no installer | 😊 Simple, fast | Clear docs on quick start |
| Setup | Run `warren cluster init` | Cluster ready in 30s | 😊 Surprisingly easy | Show join commands for workers |
| First Deploy | Create service via CLI | Replicas running in < 1 min | 🚀 It just works! | Suggest YAML for repeatability |
| Scale | Add worker nodes | Auto-join, tasks rebalance | 😊 Seamless scaling | Highlight built-in load balancing |
| Update | Deploy new version | Rolling update, zero downtime | 🎯 Professional grade | Promote canary for advanced users |
| Monitor | Check metrics endpoint | Rich metrics without setup | 😮 Built-in observability | Link to Grafana dashboard examples |
| Troubleshoot | View logs, service status | Clear structured output | 😌 Debuggable | Improve error messages, suggest fixes |

---

## Technical Considerations

### Architecture Impact

**New Components** (Warren from scratch):

- **Warren Manager** (complexity: XL): Raft-based control plane, API server, scheduler, state store
- **Warren Worker Agent** (complexity: L): Task executor, health checker, metrics collector
- **Warren CLI** (complexity: M): User interface, API client, manifest parser
- **Overlay Network Manager** (complexity: L): WireGuard mesh setup, DNS service, service VIP management
- **Storage Layer** (complexity: M): BoltDB-backed Raft log, state snapshots, backup/restore

**External Dependencies** (embedded in binary):

- hashicorp/raft (consensus)
- containerd client (container runtime interface)
- WireGuard Go library (fallback for userspace networking)
- Cobra (CLI framework)
- Prometheus client (metrics)

**Context Complexity**: XL - Distributed systems, consensus, scheduling, networking all in one system

### Integration Points

| System | Purpose | Integration Type | Complexity |
|--------|---------|-----------------|------------|
| containerd | Container lifecycle management | gRPC API client | M |
| WireGuard | Overlay networking | Kernel module (netlink) + userspace fallback | L |
| Prometheus | Metrics scraping | HTTP endpoint (/metrics) | S |
| Logging backends | Log aggregation | Structured JSON to stdout/file | XS |
| Docker Compose | Migration path | File format parser + converter | M |
| Certificate Authority | mTLS cert generation | Embedded (crypto/x509) | M |

### Data Requirements

**New Entities**:

- **Cluster**: ID, managers[], workers[], network_config, secrets[], volumes[]
- **Node**: ID, role (manager/worker), resources (CPU/RAM/disk), health_status, labels{}
- **Service**: ID, name, image, replicas, deploy_strategy, networks[], secrets[], volumes[]
- **Task**: ID, service_id, node_id, container_id, desired_state, actual_state, health_status
- **Secret**: ID, name, encrypted_data, created_at, updated_at
- **Volume**: ID, name, driver, node_affinity, mount_path

**Relationships**:

- Cluster → Nodes (1:N)
- Service → Tasks (1:N)
- Node → Tasks (1:N)
- Service → Secrets, Volumes (M:N)

**Data Volume**:

- Small clusters (< 20 nodes): ~10MB state size
- Medium clusters (< 100 nodes): ~50MB state size
- Large clusters (< 500 nodes): ~250MB state size

**Data Retention**:

- Raft log: Compacted/snapshotted every 10K entries, retain 3 snapshots
- Metrics: Warren emits, users choose retention (Prometheus scrapes)
- Logs: Structured output to stdout/file, users ship to aggregator

**Data Privacy**:

- Secrets encrypted at rest (AES-256-GCM)
- TLS-encrypted in transit
- No telemetry collection without explicit opt-in

**Data Context Complexity**: L - Relational model with clear entity boundaries

---

## Iteration Phases

### Phase 0: Foundation (Research & Design)

**Readiness Criteria**:

- [ ] Raft consensus strategy decided (library vs custom)
- [ ] Containerd integration proven with POC
- [ ] WireGuard networking tested on Linux/macOS/Windows
- [ ] Binary size estimates validated (< 100MB target)
- [ ] Scheduling algorithm designed (bin-packing + spread)

**Deliverables**:

- Architecture Decision Records (Raft choice, storage backend, networking approach)
- POC: containerd client pulling image + starting container
- POC: 3-node Raft cluster with BoltDB
- POC: WireGuard mesh between 3 hosts
- Technical specification document

### Phase 1: Core Orchestration (Milestone 1)

**Readiness Criteria**:

- [ ] Single manager can schedule tasks to workers
- [ ] Replicated services deploy successfully
- [ ] Basic health checking and restart policies work
- [ ] CLI supports: cluster init, node join, service create/list/delete
- [ ] Overlay network allows container-to-container communication

**Deliverables**:

- Warren binary (Linux amd64 only)
- Core CLI commands working
- Single-manager orchestration functional
- Basic integration tests passing
- Documentation: quick start guide

**Key Features**:

- Cluster initialization (single manager)
- Worker join with token
- Replicated service deployment
- Basic scheduler (spread strategy)
- Overlay networking (WireGuard)
- Health checks (HTTP/TCP)

### Phase 2: High Availability (Milestone 2)

**Readiness Criteria**:

- [ ] 3-manager Raft quorum functional
- [ ] Leader election and failover < 10s
- [ ] State replication working across managers
- [ ] Edge nodes operate autonomously during partition
- [ ] Rolling updates implemented

**Deliverables**:

- Multi-manager HA cluster
- Raft-based state replication
- Partition tolerance for edge nodes
- Rolling update strategy
- Enhanced CLI: manager promotion, cluster health

**Key Features**:

- Raft consensus with BoltDB
- Multi-manager quorum (3-5 nodes)
- Leader election and failover
- Autonomous edge operation
- Rolling update strategy
- Service rollback capability

### Phase 3: Advanced Deployment (Milestone 3)

**Readiness Criteria**:

- [ ] Blue/green and canary deployments functional
- [ ] Secrets management working
- [ ] Volume orchestration for stateful services
- [ ] Docker Compose compatibility tested
- [ ] Global services implemented

**Deliverables**:

- All three deployment strategies (rolling, blue/green, canary)
- Secrets encrypted storage and distribution
- Volume lifecycle management
- Docker Compose migration path
- Enhanced CLI: secrets, volumes, advanced deploy options

**Key Features**:

- Blue/green deployment
- Canary deployment with traffic splitting
- Secrets management (encrypted at rest)
- Volume orchestration (local + drivers)
- Docker Compose compatibility
- Global services (one per node)

### Phase 4: Observability & Polish (Milestone 4)

**Readiness Criteria**:

- [ ] Prometheus metrics endpoint complete
- [ ] Structured logging implemented
- [ ] Multi-platform support (Linux, macOS, ARM)
- [ ] Binary size optimized (< 100MB)
- [ ] Memory footprint optimized (< 256MB manager)
- [ ] Comprehensive documentation complete

**Deliverables**:

- Production-ready Warren v1.0
- Prometheus metrics integration
- Structured JSON logging
- Multi-platform binaries (amd64, arm64)
- Complete documentation (user guide, API reference, architecture)
- Package manager distributions (brew, apt, yum)

**Key Features**:

- Prometheus metrics endpoint
- Structured logging (JSON)
- Multi-architecture support (amd64, arm64, armv7)
- Performance optimization
- CLI tab completion
- Short alias (`wrn`)
- Comprehensive docs + examples

### Phase 5: Community & Ecosystem (Milestone 5)

**Readiness Criteria**:

- [ ] Open source release (GitHub, Apache 2.0)
- [ ] Contribution guidelines published
- [ ] CI/CD pipelines for community PRs
- [ ] Roadmap published with community input
- [ ] First 10 external contributors onboarded

**Deliverables**:

- Public GitHub repository
- Contribution guidelines (CONTRIBUTING.md)
- Code of conduct
- Issue templates (bug, feature request)
- PR template with checklist
- CI/CD for testing + releases
- Community roadmap

**Key Features**:

- Open source release
- Contribution workflows
- Community governance model
- Plugin/extension architecture (future-proofing)
- Public roadmap with voting

---

## Release Strategy

### Iteration-Based Rollout

**Layer 0: Internal Alpha** (Milestone 1-2)

- **Audience**: Cuemby internal team
- **Context**: Core orchestration + HA functional
- **Feedback**: Internal dogfooding, identify critical bugs
- **Duration**: 2-4 weeks of internal use

**Layer 1: Private Beta** (Milestone 3)

- **Audience**: 10-20 friendly users (telco partners, early adopters)
- **Context**: Advanced deployment strategies complete
- **Feedback**: Edge use case validation, performance testing
- **Duration**: 4-8 weeks with weekly releases

**Layer 2: Public Beta** (Milestone 4)

- **Audience**: Open source early adopters (GitHub stars, social media)
- **Context**: Observability complete, multi-platform support
- **Feedback**: Broad compatibility testing, docs validation
- **Duration**: 8-12 weeks, bi-weekly releases

**Layer 3: General Availability v1.0** (Milestone 5)

- **Audience**: Public, production-ready
- **Context**: Battle-tested, comprehensive docs, community established
- **Feedback**: Production deployments, community contributions
- **Cadence**: Monthly minor releases, weekly patches

### Feature Flags

Warren supports runtime feature toggles via CLI flags or environment variables:

```bash
# Enable experimental features
warren cluster init --experimental-ingress --experimental-service-mesh

# Environment-based toggles
WARREN_ENABLE_CANARY=true warren service update web --canary-weight 10
```

**Flag Configuration**:

- `--experimental-ingress`: Built-in ingress controller (Phase 4+)
- `--experimental-service-mesh`: mTLS between services (Phase 5+)
- `--enable-telemetry`: Opt-in usage telemetry (privacy-first)

**Rollout Strategy**:

- New features start behind flags (disabled by default)
- Graduate to stable after 2+ releases without issues
- Deprecated flags removed after 3 releases with warnings

### Rollback Plan

**Monitoring** (built into Warren):

- Metrics to watch: Raft leader elections, task failure rate, API latency p95
- Alert thresholds:
  - Leader elections > 3 in 10 minutes (quorum instability)
  - Task failure rate > 10% (scheduler/runtime issues)
  - API latency p95 > 500ms (control plane overload)
- Context: Dashboards provided for Grafana with pre-built alerts

**Rollback Triggers**:

- Critical bug (data loss, security vulnerability): Immediate rollback + patch release
- Performance regression > 50%: Rollback + investigation
- Community reports of widespread issues: Rollback within 24h

**Rollback Process**:

1. Release advisory: GitHub issue, social media, mailing list
2. Binary swap: Users download previous version, restart managers
3. State compatibility: Raft log backward compatible within minor versions
4. Investigation: Root cause analysis, regression tests added
5. Re-release: Patched version with extended testing

---

## Risks and Mitigations

| Risk | Probability | Impact | Context Complexity | Mitigation Strategy |
|------|------------|--------|-------------------|-------------------|
| Raft consensus bugs (split brain, data loss) | Medium | **Critical** | XL | Extensive testing with Jepsen-style chaos, use battle-tested hashicorp/raft library |
| Binary size exceeds 100MB | Medium | High | M | Aggressive dead code elimination, compression (UPX), profile-guided optimization |
| Memory footprint exceeds 256MB | High | High | L | Memory profiling (pprof), optimize data structures, limit cache sizes |
| WireGuard kernel dependency breaks compat | Low | Medium | M | Userspace WireGuard fallback (wireguard-go), compatibility matrix testing |
| Containerd API changes break integration | Low | Medium | M | Pin to stable containerd versions, abstract runtime interface for future-proofing |
| Edge autonomy creates state divergence | High | High | XL | Conflict-free replicated data types (CRDTs) for reconciliation, comprehensive merge logic |
| Community adoption fails to materialize | Medium | High | S | Marketing: blogs, conference talks, tutorials; partnerships with edge/telco companies |
| Kubernetes ecosystem pulls users away | High | Medium | S | Focus on simplicity differentiator, highlight TCO savings, target non-K8s users |

**Context Risks**:

- **Unknown unknowns in distributed systems**: Distributed systems have emergent behaviors difficult to predict. Mitigation: chaos engineering (network partitions, clock skew, disk failures), fuzzing, long-running soak tests.
- **Edge hardware variability**: Telco edge hardware varies widely (ARM, x86, old kernels). Mitigation: Compatibility matrix, test on real hardware (Raspberry Pi, Nvidia Jetson, edge servers), kernel version checks.
- **Learning curve for contributors**: Distributed systems intimidate new contributors. Mitigation: Comprehensive architecture docs, "good first issue" labels, pairing sessions, mentorship program.

---

## Open Questions & Decisions Needed

### Critical Questions

1. **Question**: Should Warren support Windows containers (not just Linux containers on WSL2)?
   - **Owner**: Platform team
   - **Context Needed**: User research - do edge deployments use Windows containers?
   - **Blocking**: Windows runtime integration (Phase 4)
   - **Priority**: [RECOMMENDED]
   - **Decision**: Defer to Phase 5, prioritize Linux (edge is primarily Linux)

2. **Question**: Custom scheduler plugins vs built-in algorithms only?
   - **Owner**: Architecture lead
   - **Context Needed**: Community feedback on scheduling needs (affinity, GPU, specialized hardware)
   - **Blocking**: Scheduler interface design (Phase 1)
   - **Priority**: [OPTIONAL]
   - **Decision**: Built-in only for v1.0, plugin architecture in v2.0 roadmap

3. **Question**: Support distributed storage drivers (Ceph, GlusterFS) or local volumes only?
   - **Owner**: Platform team
   - **Context Needed**: Edge deployments storage requirements, latency tolerance
   - **Blocking**: Volume orchestration design (Phase 3)
   - **Priority**: [REQUIRED]
   - **Decision**: Local volumes Phase 3, driver interface (CSI-like) Phase 4, actual drivers Phase 5

### Assumptions to Validate

- [ ] **Assumption**: WireGuard kernel module available on target edge devices (Linux 5.6+)
  - **Validation**: Survey target hardware, kernel version distribution
  - **Fallback**: Userspace wireguard-go (slower but compatible)

- [ ] **Assumption**: 100MB binary is acceptable to users (vs 20MB Docker binary)
  - **Validation**: User interviews, download metrics from beta
  - **Fallback**: Split into warren-manager + warren-worker binaries if needed

- [ ] **Assumption**: Edge nodes can tolerate 30+ minute partitions without data loss
  - **Validation**: Network reliability data from telco partners
  - **Fallback**: Adjust reconciliation window, local state retention policies

- [ ] **Assumption**: Users prefer zero-config mTLS over manual certificate management
  - **Validation**: Beta feedback, compare to Swarm/K8s UX
  - **Fallback**: Provide manual cert mode for advanced users (BYOC - Bring Your Own Certs)

---

## Vibe Check ✨

### Gut Feeling Assessment

**Does this feel right?**

- [x] Problem is clearly defined (Swarm too simple, K8s too complex, edge unaddressed)
- [x] Solution makes intuitive sense (simple like Swarm, features like K8s, zero dependencies)
- [x] Scope feels manageable (phased approach, clear milestones)
- [x] Team has necessary context (Kubernetes expertise, distributed systems background)
- [x] Excited to build this (solving real pain point, technical depth, community impact)

**Red Flags**:

- [ ] Too much complexity for value? **No** - complexity is inherent in distributed orchestration, we're just making it invisible to users
- [ ] Scope creeping during planning? **No** - clear "won't have" list, deferred features to later phases
- [ ] Unclear user value? **No** - clear personas, pain points, migration path from Swarm/Compose
- [ ] Team lacks critical context? **No** - Kubernetes expertise translates directly, Go proficiency present

**Confidence Level**: 🚀 **Very High**

**Why high confidence**:

1. **Clear market gap**: Swarm abandoned, K8s too complex, edge underserved
2. **Proven approach**: Taking best of Swarm (simplicity) + K8s (features), not inventing new paradigms
3. **Technical feasibility**: All components proven (Raft via hashicorp, containerd mature, WireGuard stable)
4. **Team expertise**: Kubernetes background means understanding of orchestration patterns, pitfalls
5. **Community timing**: Cloud-native fatigue, interest in simpler alternatives (see Deno vs Node, Caddy vs nginx)

**Risks to monitor**:

- Binary size creep (mitigate with aggressive optimization)
- Edge partition reconciliation complexity (lots of testing needed)
- Community adoption (requires marketing, outreach, content creation)

---

## Appendix

### Glossary

- **Raft**: Consensus algorithm for distributed systems, ensures state consistency across nodes
- **Containerd**: Industry-standard container runtime, used by Kubernetes and Docker
- **WireGuard**: Modern VPN protocol, used here for encrypted overlay networking
- **mTLS**: Mutual TLS, both client and server authenticate each other with certificates
- **Edge computing**: Running compute workloads close to data source (IoT devices, cell towers) vs centralized cloud
- **Replicated service**: Service with N identical copies for availability and load distribution
- **Global service**: Service with one instance per node (e.g., monitoring agents)
- **BoltDB**: Embedded key-value database, used by etcd and other distributed systems
- **Quorum**: Majority of nodes required for consensus (e.g., 2 of 3 managers)
- **Control plane**: Manager nodes that make scheduling decisions, vs data plane (workers running containers)

### References

- Docker Swarm documentation: https://docs.docker.com/engine/swarm/ (archived)
- Kubernetes architecture: https://kubernetes.io/docs/concepts/architecture/
- HashiCorp Raft: https://github.com/hashicorp/raft
- Containerd: https://containerd.io/
- WireGuard: https://www.wireguard.com/
- Edge computing overview: https://www.cncf.io/blog/2020/09/03/edge-computing/

### Revision History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2025-10-09 | Cuemby Engineering | Initial PRD based on planning session |

---

## Approvals & Sign-off

**Approval Status**: Approved (solo project, open source)

| Role | Name | Feedback | Approved |
|------|------|----------|----------|
| Product Lead | Cuemby Team | PRD aligns with vision | ☑ |
| Engineering Lead | Cuemby Team | Technical approach sound | ☑ |
| Community Lead | TBD | - | ☐ (Post open-source) |

**Final Approval Date**: 2025-10-09

---

## Related Documentation

- **Technical Spec**: [specs/tech.md](tech.md) (to be created)
- **Milestone Plan**: [tasks/todo.md](../tasks/todo.md) (to be created)
- **Architecture Docs**: [.agent/System/project-architecture.md](../.agent/System/project-architecture.md) (to be updated)

---

*This PRD follows context-driven planning principles. Focus is on understanding depth, iteration readiness, and clear scope boundaries rather than arbitrary deadlines. Warren will be built iteratively with continuous validation against user needs and technical constraints.*
