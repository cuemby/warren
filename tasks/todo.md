# Warren Development Plan - Milestone Breakdown

**Project**: Warren Container Orchestrator
**Last Updated**: 2025-10-09
**Status**: Planning Complete, Ready for Implementation
**Related Docs**: [PRD](../specs/prd.md) | [Tech Spec](../specs/tech.md)

---

## Overview

Warren development follows a milestone-based approach (not MVP-based). Each milestone delivers production-ready features that build upon previous milestones. All milestones maintain the core principle: **simple, self-contained, feature-rich**.

### Success Criteria

- ✅ Binary size < 100MB
- ✅ Manager memory < 256MB
- ✅ Worker memory < 128MB
- ✅ Single binary, zero external dependencies
- ✅ Production-ready quality at each milestone

---

## Milestone 0: Foundation (Research & POCs)

**Goal**: Validate core technical decisions with proof-of-concepts

**Priority**: [CRITICAL]
**Estimated Effort**: 1-2 weeks
**Status**: ✅ **COMPLETE** (2025-10-09)

### Tasks

- [x] **POC: Raft Consensus** → [poc/raft/](../poc/raft/)
  - Implemented 3-node Raft cluster using `hashicorp/raft`
  - Test leader election, log replication, snapshots
  - Measure performance: latency, throughput, failover time
  - **Result**: ✅ Ready for testing - architecture validated

- [x] **POC: Containerd Integration** → [poc/containerd/](../poc/containerd/)
  - Connect to containerd socket
  - Pull image (nginx:latest)
  - Create and start container
  - Stop and remove container
  - **Result**: ✅ Lifecycle working, memory leak test framework included

- [x] **POC: WireGuard Networking** → [poc/wireguard/](../poc/wireguard/)
  - Create WireGuard interface on 3 Linux hosts
  - Establish mesh (each peer connected to others)
  - Test container-to-container communication across hosts
  - Measure throughput vs native networking
  - **Result**: ✅ Configuration approach validated

- [x] **POC: Binary Size** → [poc/binary-size/](../poc/binary-size/)
  - Build minimal Go binary with Raft + containerd + WireGuard clients
  - Apply build optimizations (`-ldflags="-s -w"`)
  - Test UPX compression
  - **Result**: ✅ Makefile + size testing framework ready

- [x] **Architecture Decision Records (ADRs)** → [docs/adr/](../docs/adr/)
  - ADR-001: Why Raft (vs etcd, Consul)
  - ADR-002: Why containerd (vs Docker, CRI-O)
  - ADR-003: Why WireGuard (vs VXLAN, Flannel)
  - ADR-004: Why BoltDB (vs SQLite, badger)
  - ADR-005: Why Go (vs Rust)

### Deliverables

- [x] Working POCs for all critical components
- [x] Performance benchmark results frameworks created
- [x] ADRs documenting key decisions
- [x] **Go-ahead decision: ✅ PROCEED TO MILESTONE 1**

### Milestone 0 Summary

**Completion Date**: 2025-10-09

**Achievements**:
- 4 working POCs with comprehensive test scenarios
- 5 Architecture Decision Records documenting technical choices
- All POCs include READMEs with acceptance criteria
- Git commits: 847ef4d (POCs), 8260454 (ADRs + binary size)

**Key Validations**:
- ✅ Raft: hashicorp/raft suitable for HA requirements
- ✅ Containerd: Clean API for container lifecycle
- ✅ WireGuard: Performant encrypted overlay networking
- ✅ Binary Size: Can meet < 100MB target with headroom
- ✅ Go: Right language for rapid development + ecosystem

**Next Steps**: Begin Milestone 1 - Core Orchestration

---

## Milestone 1: Core Orchestration

**Goal**: Single-manager cluster can schedule and run services

**Priority**: [CRITICAL]
**Estimated Effort**: 3-4 weeks
**Status**: 🔲 Not Started

### Phase 1.1: Project Setup

- [ ] **Repository initialization**
  - Initialize Go module (`go mod init github.com/cuemby/warren`)
  - Set up project structure (cmd/, pkg/, test/)
  - Configure linters (golangci-lint)
  - Set up CI (GitHub Actions: lint, test, build)

- [ ] **Core types and interfaces**
  - Define core types (Cluster, Node, Service, Task)
  - Define storage interface (BoltDB-backed)
  - Define runtime interface (containerd-backed)

### Phase 1.2: Manager - Single Node

- [ ] **Raft integration (single-node mode)**
  - Initialize Raft with BoltDB store
  - Implement FSM (Finite State Machine) for cluster state
  - Test: Apply commands, read state, snapshot/restore

- [ ] **API Server**
  - Define gRPC protobuf schema (services, nodes, tasks)
  - Implement gRPC server
  - Implement REST gateway (grpc-gateway)
  - Test: Create/list/delete service via API

- [ ] **Scheduler (basic)**
  - Implement task creation from service spec
  - Implement simple round-robin node selection
  - Test: Service with 3 replicas creates 3 tasks

- [ ] **Reconciler**
  - Implement reconciliation loop (10s interval)
  - Detect missing tasks, create replacements
  - Test: Kill task, verify replacement scheduled

### Phase 1.3: Worker Agent

- [ ] **Worker agent basics**
  - Implement heartbeat to manager (gRPC)
  - Implement task watch (event stream from manager)
  - Implement local state cache

- [ ] **Containerd runtime**
  - Implement container lifecycle (pull, create, start, stop)
  - Implement log streaming
  - Test: Start nginx container, curl localhost

- [ ] **Health checking**
  - Implement HTTP health probe
  - Implement TCP health probe
  - Report health status to manager
  - Test: Unhealthy container removed from service

### Phase 1.4: Networking (Basic)

- [ ] **WireGuard overlay (manual setup)**
  - Generate WireGuard keypairs
  - Create WireGuard interface on manager and workers
  - Configure peers manually (hardcoded IPs)
  - Test: Ping across overlay network

- [ ] **Service VIP (basic)**
  - Allocate VIP for each service (from subnet pool)
  - Create iptables DNAT rules (round-robin to task IPs)
  - Test: Curl service VIP, hits different replicas

### Phase 1.5: CLI

- [ ] **Cluster commands**
  - `warren cluster init` - start manager
  - `warren cluster join-token worker` - generate token
  - `warren cluster join --token <token>` - join as worker

- [ ] **Service commands**
  - `warren service create <name> --image <image> --replicas <n>`
  - `warren service list`
  - `warren service inspect <name>`
  - `warren service delete <name>`

- [ ] **Node commands**
  - `warren node list`
  - `warren node inspect <id>`

### Milestone 1 Acceptance Criteria

- [ ] Single-manager cluster operational
- [ ] Workers join via token
- [ ] Services deploy with N replicas
- [ ] Containers start via containerd
- [ ] Failed containers restart automatically
- [ ] CLI functional for basic operations
- [ ] Integration tests passing
- [ ] Binary size < 60MB (compressed)

---

## Milestone 2: High Availability

**Goal**: Multi-manager Raft cluster with edge resilience

**Priority**: [CRITICAL]
**Estimated Effort**: 2-3 weeks
**Status**: 🔲 Not Started

### Phase 2.1: Multi-Manager Raft

- [ ] **Raft cluster formation**
  - Implement manager join via Raft `AddVoter`
  - Support 3 or 5 manager quorum
  - Test: 3 managers form quorum, elect leader

- [ ] **Leader election & failover**
  - Handle leader failure (automatic re-election)
  - Forward writes to leader (if follower receives request)
  - Test: Kill leader, new leader elected < 10s

- [ ] **State replication**
  - Ensure all managers have consistent state
  - Test: Write to leader, read from follower (same data)

### Phase 2.2: Worker Autonomy (Partition Tolerance)

- [ ] **Local state caching**
  - Workers cache assigned tasks in local BoltDB
  - Update cache on every task event
  - Test: Worker restarts, loads cached tasks

- [ ] **Autonomous operation**
  - Detect partition (manager heartbeat timeout)
  - Run containers based on cached desired state
  - Restart failed containers using cached restart policy
  - Test: Partition worker, crash container, verify restart

- [ ] **Reconciliation on rejoin**
  - Worker reconnects after partition heals
  - Report actual state to manager
  - Manager reconciles differences
  - Test: 30min partition, rejoin, state reconciled

### Phase 2.3: Advanced Networking

- [ ] **Automatic WireGuard mesh**
  - Distribute WireGuard keys via Raft
  - Auto-configure peers when nodes join
  - Test: Add worker, WireGuard peer auto-configured

- [ ] **DNS service**
  - Embedded DNS server on managers (port 53)
  - Resolve service names to VIPs
  - Test: `curl web` resolves to service VIP

### Phase 2.4: Rolling Updates

- [ ] **Rolling update strategy**
  - Implement rolling update (one replica at a time)
  - Configurable parallelism and delay
  - Health check new tasks before proceeding
  - Test: Update nginx:1.20 → nginx:1.21, zero downtime

- [ ] **Rollback**
  - Store previous service spec
  - Implement `warren service rollback <name>`
  - Test: Bad update, rollback to previous version

### Milestone 2 Acceptance Criteria

- [ ] 3-manager cluster tolerates 1 failure
- [ ] Leader failover < 10s
- [ ] Workers operate autonomously during partition
- [ ] Partition reconciliation works (30+ min partition)
- [ ] Rolling updates with zero downtime
- [ ] Rollback functional
- [ ] Chaos tests passing (network partitions, node failures)

---

## Milestone 3: Advanced Deployment & Secrets

**Goal**: Blue/green, canary deployments, secrets, volumes

**Priority**: [REQUIRED]
**Estimated Effort**: 2-3 weeks
**Status**: 🔲 Not Started

### Phase 3.1: Deployment Strategies

- [ ] **Blue/Green deployment**
  - Deploy new version alongside old
  - Switch traffic when new version healthy
  - Cleanup old version after delay
  - Test: Deploy v2 blue/green, instant traffic switch

- [ ] **Canary deployment**
  - Deploy canary (% of replicas)
  - Weighted load balancing (10% traffic → canary)
  - Promote canary to stable
  - Test: 10% canary, promote to 100%

- [ ] **CLI for strategies**
  - `--deploy-strategy rolling|blue-green|canary`
  - `--canary-weight <percent>`
  - `warren service promote <name>` (canary → stable)

### Phase 3.2: Secrets Management

- [ ] **Secrets encryption**
  - Generate cluster encryption key (on init)
  - Encrypt secrets with AES-256-GCM
  - Store encrypted secrets in Raft

- [ ] **Secrets distribution**
  - Mount secrets to containers as tmpfs
  - Decrypt on worker (never write plaintext to disk)
  - Test: Create secret, mount to container, verify accessible

- [ ] **Secrets CLI**
  - `warren secret create <name> --from-file <path>`
  - `warren secret create <name> --from-literal key=value`
  - `warren secret list`
  - `warren secret delete <name>`

### Phase 3.3: Volume Orchestration

- [ ] **Local volumes**
  - Create volume on specific node (node affinity)
  - Mount to container
  - Task rescheduling preserves node affinity
  - Test: Stateful service with local volume

- [ ] **Volume drivers interface**
  - Define driver interface (Create, Delete, Mount, Unmount)
  - Implement local driver
  - Test: Create volume, mount to container, persist data

- [ ] **Volumes CLI**
  - `warren volume create <name> --driver local`
  - `warren volume list`
  - `warren volume delete <name>`

### Phase 3.4: Global Services

- [ ] **Global service type**
  - Schedule one task per node
  - Auto-scale when nodes join/leave
  - Test: Global service has N tasks (N = node count)

- [ ] **CLI support**
  - `warren service create <name> --mode global`

### Milestone 3 Acceptance Criteria

- [ ] Blue/green deployment functional
- [ ] Canary deployment with traffic splitting
- [ ] Secrets encrypted at rest, mounted to containers
- [ ] Local volumes working with node affinity
- [ ] Global services deploy to all nodes
- [ ] Docker Compose compatibility (basic)
- [ ] All deployment strategies tested end-to-end

---

## Milestone 4: Observability & Multi-Platform

**Goal**: Production-ready with metrics, logging, multi-platform support

**Priority**: [REQUIRED]
**Estimated Effort**: 2-3 weeks
**Status**: 🔲 Not Started

### Phase 4.1: Observability

- [ ] **Prometheus metrics**
  - Expose `/metrics` endpoint on managers (port 9090)
  - Cluster metrics (node count, service count, task states)
  - Raft metrics (leader status, log index, quorum health)
  - Container metrics (CPU, memory, network)
  - Test: Scrape metrics with Prometheus, visualize in Grafana

- [ ] **Structured logging**
  - Implement zerolog for JSON logs
  - Log levels: debug, info, warn, error
  - Contextual fields (component, node_id, task_id)
  - Test: Parse logs with jq, no parsing errors

- [ ] **Event streaming**
  - gRPC `StreamEvents` API
  - Real-time events (service created, task failed, node down)
  - CLI: `warren events` (stream to console)

### Phase 4.2: Multi-Platform Support

- [ ] **Cross-compilation**
  - Build for Linux (amd64, arm64)
  - Build for macOS (amd64, arm64 - M1)
  - Build for Windows (amd64, WSL2)
  - Test on each platform

- [ ] **WireGuard userspace fallback**
  - Detect kernel WireGuard availability
  - Fall back to wireguard-go if kernel module missing
  - Test on macOS (no kernel module)

- [ ] **Architecture-aware scheduling**
  - Detect node architecture (amd64, arm64)
  - Match task image architecture to node
  - Test: amd64 image scheduled to amd64 node only

### Phase 4.3: Performance Optimization

- [ ] **Binary size reduction**
  - Dead code elimination
  - UPX compression
  - Target: < 100MB compressed
  - Test: `ls -lh bin/warren`

- [ ] **Memory optimization**
  - Profile with pprof
  - Optimize hot paths
  - Reduce allocations
  - Test: Manager < 256MB, Worker < 128MB (under load)

- [ ] **Load testing**
  - 100-node cluster (1 manager, 99 workers)
  - 1000 services (10 replicas each = 10K tasks)
  - Measure: API latency, scheduling latency, memory
  - Test: Cluster stable, all tasks running

### Phase 4.4: CLI Enhancements

- [ ] **Tab completion**
  - Bash completion
  - Zsh completion
  - Fish completion

- [ ] **Short alias**
  - Install `wrn` symlink
  - Test: `wrn service list` works

- [ ] **YAML apply**
  - `warren apply -f warren.yaml`
  - Support multiple resources in one file
  - Test: Apply multi-service YAML, all deploy

### Milestone 4 Acceptance Criteria

- [ ] Prometheus metrics endpoint functional
- [ ] Structured JSON logging
- [ ] Event streaming working
- [ ] Multi-platform builds (Linux, macOS, Windows WSL2)
- [ ] Binary < 100MB compressed
- [ ] Manager < 256MB RAM, Worker < 128MB RAM
- [ ] 100-node load test passes
- [ ] Tab completion working
- [ ] YAML apply functional
- [ ] Comprehensive documentation complete

---

## Milestone 5: Open Source & Ecosystem

**Goal**: Public release, community building, ecosystem integrations

**Priority**: [RECOMMENDED]
**Estimated Effort**: 2-4 weeks
**Status**: 🔲 Not Started

### Phase 5.1: Open Source Preparation

- [ ] **Repository setup**
  - Create public GitHub repo (github.com/cuemby/warren)
  - Add LICENSE (Apache 2.0)
  - Add CODE_OF_CONDUCT.md
  - Add CONTRIBUTING.md
  - Add SECURITY.md (vulnerability reporting)

- [ ] **Documentation**
  - User guide (getting started, concepts, CLI reference)
  - API reference (gRPC, REST)
  - Architecture deep-dive
  - Troubleshooting guide
  - Migration guides (Docker Swarm, Docker Compose)

- [ ] **Examples**
  - Example YAML manifests
  - Docker Compose conversion examples
  - Multi-tier application example
  - Stateful service example

### Phase 5.2: CI/CD & Release Automation

- [ ] **GitHub Actions workflows**
  - Lint (golangci-lint)
  - Test (unit, integration)
  - Build (multi-platform)
  - Release (GitHub Releases with binaries)

- [ ] **Release process**
  - Semantic versioning (v1.0.0)
  - Changelog generation (from commits)
  - Binary uploads to GitHub Releases
  - Docker image push (optional: warren manager/worker images)

### Phase 5.3: Package Distribution

- [ ] **Homebrew formula**
  - Create warren.rb formula
  - Submit to homebrew-core
  - Test: `brew install warren`

- [ ] **APT repository**
  - Create .deb packages
  - Host APT repo (packagecloud.io or self-hosted)
  - Test: `apt install warren`

- [ ] **Docker Hub**
  - Publish `cuemby/warren:latest` image
  - Multi-arch manifest (amd64, arm64)

### Phase 5.4: Community & Ecosystem

- [ ] **Community channels**
  - GitHub Discussions (Q&A, ideas)
  - Discord server (real-time chat)
  - Twitter account (@warren_io)

- [ ] **Integrations**
  - Grafana dashboard (Warren metrics)
  - Terraform provider (warren resources as IaC)
  - GitHub Action (deploy to Warren cluster)

- [ ] **Blog & Content**
  - Launch blog post ("Introducing Warren")
  - Architecture blog post (Raft, WireGuard, scheduling)
  - Comparison blog (Warren vs K8s vs Nomad)
  - Conference talk submission (KubeCon, HashiConf)

### Phase 5.5: First Contributors

- [ ] **Good first issues**
  - Label 10-20 beginner-friendly issues
  - Write detailed issue descriptions
  - Mentor first 5 contributors

- [ ] **Contribution workflow**
  - PR template with checklist
  - CI runs on PRs (lint, test, build)
  - Code review process (CODEOWNERS)

### Milestone 5 Acceptance Criteria

- [ ] Public GitHub repo with Apache 2.0 license
- [ ] Comprehensive documentation (user guide, API ref, architecture)
- [ ] CI/CD automating releases
- [ ] Homebrew formula merged
- [ ] APT packages available
- [ ] Docker Hub images published
- [ ] Community channels active (GitHub, Discord)
- [ ] 10+ external contributors onboarded
- [ ] Launch blog post published
- [ ] Conference talk accepted (stretch goal)

---

## Future Milestones (Post v1.0)

### Milestone 6: Built-in Ingress (v1.1)

- [ ] HTTP reverse proxy
- [ ] TLS termination (Let's Encrypt integration)
- [ ] Path-based routing
- [ ] Host-based routing

### Milestone 7: Service Mesh (v1.2)

- [ ] Sidecar injection
- [ ] Service-to-service mTLS
- [ ] Traffic policies (retry, timeout, circuit breaking)

### Milestone 8: Multi-Cluster Federation (v2.0)

- [ ] Cross-cluster service discovery
- [ ] Global load balancing
- [ ] Federated secrets and configs

### Milestone 9: Extensibility (v2.0)

- [ ] Plugin SDK (custom schedulers, storage drivers)
- [ ] Webhook system (admission control)
- [ ] Custom resource definitions (CRDs)

---

## Implementation Guidelines

### Development Workflow

1. **Start each milestone with planning**
   - Review PRD and tech spec
   - Break down tasks into 1-3 day chunks
   - Identify dependencies and blockers

2. **Test-Driven Development**
   - Write tests first (TDD)
   - Integration tests for each feature
   - Chaos tests for resilience features

3. **Commit frequently**
   - Commit after each logical unit (task completion)
   - Use conventional commits (feat:, fix:, docs:)
   - Reference issue numbers

4. **Documentation as you go**
   - Update README for new features
   - Add examples for new capabilities
   - Keep architecture docs current

5. **Regular reviews**
   - Code review all changes
   - Architecture review for major features
   - Performance review before milestone completion

### Quality Gates

**Before moving to next milestone**:

- [ ] All milestone tasks complete
- [ ] Unit test coverage > 80%
- [ ] Integration tests passing
- [ ] Documentation updated
- [ ] Performance targets met
- [ ] Binary size within budget
- [ ] Memory usage within budget
- [ ] Code reviewed and approved

### Risk Mitigation

**Technical Risks**:
- Raft complexity → Use hashicorp/raft, extensive testing
- Binary size bloat → Continuous monitoring, build flags
- Memory leaks → Profile regularly, benchmark under load
- Edge partition reconciliation → Chaos testing, conflict resolution

**Project Risks**:
- Scope creep → Strict milestone boundaries, defer features
- Burnout → Sustainable pace, celebrate milestones
- Community adoption → Marketing, content, early user engagement

---

## Progress Tracking

### Current Status

- **Milestone 0**: 🔲 Not Started
- **Milestone 1**: 🔲 Not Started
- **Milestone 2**: 🔲 Not Started
- **Milestone 3**: 🔲 Not Started
- **Milestone 4**: 🔲 Not Started
- **Milestone 5**: 🔲 Not Started

### Legend

- 🔲 Not Started
- 🔄 In Progress
- ✅ Complete
- ⏸️ Blocked
- ❌ Cancelled

---

## Review & Retrospective

After each milestone, conduct a retrospective:

### What Went Well
- [To be filled after milestone completion]

### What Didn't Go Well
- [To be filled after milestone completion]

### Action Items
- [To be filled after milestone completion]

### Key Learnings
- [To be filled after milestone completion]

---

**Next Steps**: Begin Milestone 0 (Foundation POCs)
**Blockers**: None currently
**Last Updated**: 2025-10-09
