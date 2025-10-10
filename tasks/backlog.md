# Warren Development Backlog

**Last Updated**: 2025-10-10
**Status**: Active - Features deferred from milestones or recommended for future work

---

## Overview

This document tracks optional features, enhancements, and recommendations that have been deferred from the main milestone plan. Items are organized by priority and category.

### Priority Levels

- **[HIGH]** - Important for production readiness, should be addressed soon
- **[MEDIUM]** - Valuable enhancements, address when capacity allows
- **[LOW]** - Nice-to-have features, consider for future versions
- **[FUTURE]** - Post-v1.0 features for major version updates

---

## Deferred from Milestone 1

### Phase 1.3: Worker Agent

#### Containerd Runtime Enhancements
**Priority**: [MEDIUM]
**Status**: Basic containerd integration complete, enhancements deferred

- [ ] **Container log streaming**
  - Stream container logs via gRPC to manager
  - CLI: `warren service logs <service-name> [--follow]`
  - Aggregated logs from all replicas
  - Test: Deploy service, stream logs in real-time

- [ ] **Health checking enhancements**
  - HTTP health probe with custom paths/headers
  - TCP health probe with timeout
  - gRPC health checking protocol
  - Exec-based health checks (run command in container)
  - Report health status to manager with retry logic
  - Test: Unhealthy container removed from service rotation

**Rationale**: Basic container execution working. Log streaming and advanced health checks are valuable but not critical for core functionality.

---

### Phase 1.4: Networking (Basic)

#### Manual WireGuard Setup
**Priority**: [MEDIUM]
**Status**: Deferred to M4/M5

- [ ] **WireGuard overlay (manual setup)**
  - Generate WireGuard keypairs per node
  - Create WireGuard interface on manager and workers
  - Configure peers manually with hardcoded IPs
  - Test: Ping across overlay network

- [ ] **Service VIP (basic load balancing)**
  - Allocate VIP for each service from subnet pool
  - Create iptables DNAT rules (round-robin to task IPs)
  - Update rules on task creation/deletion
  - Test: Curl service VIP, hits different replicas

**Rationale**: Not required for basic orchestration. Can use host networking initially. Full automated networking planned for later milestone.

---

### Phase 1.5: CLI & Integration Testing

#### CLI Commands
**Priority**: [LOW]
**Status**: Core commands complete, inspect deferred

- [ ] **Node inspect command**
  - `warren node inspect <id>` - show detailed node info
  - Display: resources, running tasks, labels, health
  - Test: Inspect node shows accurate information

#### Integration Testing
**Priority**: [HIGH]
**Status**: Basic tests done, real-world testing needed

- [ ] **Worker failure test automation**
  - Automated test: Kill worker → Tasks rescheduled
  - Verify reconciler behavior
  - Measure failover time

- [ ] **Chaos engineering tests**
  - Network partition between manager and workers
  - Manager failure scenarios
  - Disk full scenarios
  - Test recovery and data consistency

**Rationale**: `node inspect` is nice-to-have. Chaos testing critical for production but deferred until more features stable.

---

## Deferred from Milestone 2

### Phase 2.2: Multi-Manager Raft

#### Performance Tuning
**Priority**: [HIGH]
**Status**: Works but needs optimization

- [ ] **Raft election timeout tuning**
  - Current: Leader failover takes >15s (target: <10s)
  - Tune Raft configuration parameters:
    - HeartbeatTimeout
    - ElectionTimeout
    - LeaderLeaseTimeout
  - Test: Failover consistently under 10s

**Rationale**: Leader failover works but slower than desired. Not blocking but important for production HA.

---

### Optional Features (Nice to Have)

#### Worker Autonomy (Partition Tolerance)
**Priority**: [MEDIUM]
**Status**: Deferred - basic HA sufficient for now

- [ ] **Local state caching on workers**
  - Workers cache task assignments locally
  - Continue running existing tasks during partition
  - Queue state changes for reconciliation

- [ ] **Autonomous operation during partition**
  - Workers operate independently when disconnected
  - Maintain container health and restarts
  - No new task assignment during partition

- [ ] **Reconciliation on rejoin**
  - Worker reconnects to manager after partition heals
  - Sync state: report actual vs desired
  - Manager reconciles differences
  - Test: Partition network → Worker continues → Rejoin → State syncs

**Rationale**: Edge resilience is valuable but adds complexity. Current design requires manager connectivity for new scheduling, which is acceptable for most use cases.

---

#### Advanced Networking
**Priority**: [MEDIUM]
**Status**: Deferred to M4 or later

- [ ] **Automatic WireGuard mesh**
  - Managers distribute WireGuard configs
  - Automatic peer discovery and connection
  - Dynamic IP allocation for overlay network
  - Test: Node joins → Auto-configures WireGuard → Can reach other nodes

- [ ] **DNS service**
  - Built-in DNS server on managers
  - Service discovery via DNS (servicename.warren)
  - SRV records for port discovery
  - Test: Container resolves service name → Gets service VIP

**Rationale**: Manual networking setup acceptable for now. Automated mesh and DNS are quality-of-life improvements.

---

#### Rolling Updates
**Priority**: [LOW]
**Status**: Basic deployment strategies in M3, rolling updates partially implemented

- [ ] **Enhanced rolling update strategy**
  - Health-check based progression (wait for healthy)
  - Automatic rollback on failure threshold
  - Configurable failure policies
  - Test: Rolling update with failing health checks → Auto-rollback

**Rationale**: Basic rolling update foundation exists. Enhanced version with health-aware progression is nice-to-have.

---

## Deferred from Milestone 3

### Phase 3.4: Deployment Strategies

#### Blue/Green Deployment
**Priority**: [MEDIUM]
**Status**: Foundation ready, implementation deferred

- [ ] **Implement blue/green deployer (pkg/deploy/bluegreen.go)**
  - Deploy green version with full replicas
  - Wait for all tasks healthy
  - Switch VIP/DNS routing to green
  - Cleanup blue after configurable grace period
  - Test: Zero-downtime version switch

- [ ] **Service versioning support**
  - Track service version in labels
  - Label tasks as blue/green with version
  - Rollback switches back to blue version
  - Test: Update → Rollback → Original version restored

- [ ] **CLI commands**
  - `warren service update <name> --image <img> --strategy blue-green`
  - `warren service rollback <name>` (switch back to previous)
  - Test: Complete blue/green workflow via CLI

**Rationale**: Rolling updates sufficient for MVP. Blue/green valuable for critical services but adds complexity.

---

#### Canary Deployment
**Priority**: [MEDIUM]
**Status**: Foundation ready, implementation deferred

- [ ] **Implement canary deployer (pkg/deploy/canary.go)**
  - Deploy canary tasks (percentage of replicas)
  - Gradual weight increase: 10% → 50% → 100%
  - Health-check based promotion
  - Full promotion triggers rolling update for rest
  - Test: 10% → 50% → 100% promotion workflow

- [ ] **Weighted load balancing (pkg/network/loadbalancer.go)**
  - Support traffic weight in routing rules
  - iptables probability matching for weight distribution
  - Dynamic weight updates
  - Test: Traffic split matches configured weights (10% to canary)

- [ ] **CLI commands**
  - `warren service update <name> --image <img> --strategy canary --weight 10`
  - `warren service promote <name> --weight 50` (increase gradually)
  - `warren service promote <name>` (full promotion)
  - Test: Complete canary workflow via CLI

**Rationale**: Advanced deployment strategy. Valuable for gradual rollouts but not essential for initial release.

---

### Phase 3.5: Documentation & Testing

#### End-to-End Tests
**Priority**: [HIGH]
**Status**: Unit tests done, integration tests needed

- [ ] **Secrets integration test**
  - Deploy nginx with TLS cert from secret
  - Verify HTTPS works with cert
  - Update secret, verify rotation in containers
  - Test: End-to-end secret workflow

- [ ] **Volumes integration test**
  - Deploy postgres with persistent volume
  - Write data to database
  - Kill container, let scheduler restart
  - Verify data persists across restarts
  - Test: End-to-end stateful service

- [ ] **Global service integration test**
  - Deploy node-exporter as global service
  - Verify 1 task per node
  - Add new worker node to cluster
  - Verify task auto-scheduled to new node
  - Test: Global service scales with cluster

- [ ] **Blue/green integration test**
  - Deploy app v1 (multiple replicas)
  - Update to v2 using blue/green strategy
  - Verify zero downtime (continuous requests)
  - Rollback to v1
  - Test: Complete deployment workflow

- [ ] **Canary integration test**
  - Deploy app v1 (5 replicas)
  - Deploy canary v2 (10% weight)
  - Verify ~10% traffic to v2
  - Promote to 100%
  - Test: Weighted traffic distribution

**Rationale**: Critical for production confidence but time-consuming. Prioritize after core features stable.

---

#### Documentation Updates
**Priority**: [HIGH]
**Status**: Core docs complete, M3/M4 updates needed

- [ ] **Update API documentation**
  - Document secrets/volumes gRPC APIs
  - Document deployment strategy APIs
  - Add request/response examples
  - Document error codes

- [ ] **Update CLI reference**
  - Add secret commands with examples
  - Add volume commands with examples
  - Add deployment strategy flags
  - Add tab completion instructions

- [ ] **Update quickstart guide**
  - Add secrets usage example
  - Add volumes usage example
  - Add deployment strategies example
  - Add YAML apply example

- [ ] **Update .agent documentation**
  - Update project-architecture.md with M3/M4 features
  - Update database-schema.md (secrets, volumes buckets)
  - Update api-documentation.md (new endpoints)
  - Add deployment-strategies.md

**Rationale**: Documentation critical for adoption. Prioritize alongside feature releases.

---

### Additional M3 Features

#### Advanced Features
**Priority**: [LOW]
**Status**: Deferred to post-v1.0

- [ ] **Docker Compose compatibility**
  - Parse Docker Compose YAML format
  - Convert Compose services to Warren services
  - Support common Compose features (depends_on, networks, volumes)
  - Test: `warren apply -f docker-compose.yml`

- [ ] **Distributed volume drivers**
  - NFS volume driver
  - Ceph/RBD volume driver
  - Volume driver plugin interface
  - Test: Deploy service with NFS volume, accessible from any node

- [ ] **Secret rotation automation**
  - Automatic secret rotation on expiry
  - Rolling restart of services using rotated secret
  - Webhook support for external secret providers (Vault)
  - Test: Secret rotates → Services restart with new secret

**Rationale**: Advanced features for specialized use cases. Not needed for v1.0.

---

## Deferred from Milestone 4

### Phase 4.2: Multi-Platform Support

#### WireGuard Userspace Fallback
**Priority**: [MEDIUM]
**Status**: Deferred - kernel WireGuard working on Linux

- [ ] **WireGuard userspace mode**
  - Detect kernel WireGuard availability on startup
  - Fall back to wireguard-go (userspace) if unavailable
  - Test on macOS (no kernel WireGuard)
  - Test on Windows WSL2
  - Performance comparison: kernel vs userspace

**Rationale**: Kernel WireGuard works on Linux (primary target). Userspace fallback enables development on macOS/Windows but lower priority.

---

#### Architecture-Aware Scheduling
**Priority**: [MEDIUM]
**Status**: Deferred - homogeneous clusters common

- [ ] **Architecture detection**
  - Detect node architecture (amd64, arm64) on registration
  - Store in node metadata
  - Display in `warren node list`

- [ ] **Image architecture matching**
  - Detect image architecture from manifest
  - Match task image arch to node arch in scheduler
  - Fail scheduling if no compatible nodes
  - Test: amd64 image → Only scheduled to amd64 nodes

**Rationale**: Most clusters are homogeneous. Mixed-arch support valuable for edge but not critical.

---

### Phase 4.3: Performance Optimization

#### Memory Optimization
**Priority**: [HIGH]
**Status**: Not started - needed for production

- [ ] **Profile manager with pprof**
  - Enable pprof HTTP endpoint
  - Profile heap allocations under load
  - Identify hot paths and optimization opportunities
  - Test: Manager stable under load

- [ ] **Profile worker with pprof**
  - Profile worker memory usage
  - Optimize container status polling
  - Reduce allocations in hot paths
  - Test: Worker < 128MB under load

- [ ] **Optimize reconciler**
  - Reduce reconciliation loop allocations
  - Efficient diff calculation
  - Batch state updates
  - Test: Reconciler efficient with 1000+ tasks

**Rationale**: Critical for production deployments. Current implementation functional but not optimized.

---

#### Load Testing
**Priority**: [HIGH]
**Status**: Not started - needed to validate scale targets

- [ ] **Large cluster test (100 nodes)**
  - Set up: 3 managers + 97 workers
  - Deploy 1000 services (10 replicas each = 10,000 tasks)
  - Measure: API latency, scheduling latency, memory usage
  - Monitor: Raft performance, BoltDB read/write latency
  - Test: Cluster stable, all tasks running, <5s scheduling time

- [ ] **Stress test - rapid deployments**
  - Deploy 100 services simultaneously
  - Measure: API throughput, scheduler queue depth
  - Monitor: CPU usage, memory growth
  - Test: No scheduler starvation, all services deployed

- [ ] **Stress test - rapid scaling**
  - Create service with 1 replica
  - Scale to 1000 replicas
  - Measure: Time to fully scaled
  - Test: <1 minute to full scale

- [ ] **Chaos test - manager failures**
  - Kill random managers during load test
  - Verify: Cluster recovers, no task loss
  - Measure: Recovery time, leader election time
  - Test: < 10s failover, zero task loss

**Rationale**: Essential to validate Warren meets scale and performance targets before production use.

---

### Phase 4.4: CLI Enhancements

#### Short Alias
**Priority**: [LOW]
**Status**: Deferred - convenience feature

- [ ] **Install `wrn` symlink**
  - Create symlink during installation
  - Update installation scripts (Homebrew, APT)
  - Add to PATH automatically
  - Test: `wrn service list` works like `warren service list`

**Rationale**: Nice quality-of-life feature but very low priority.

---

## Future Enhancements (Post-M4)

### Observability Enhancements
**Priority**: [MEDIUM]
**Status**: Post-v1.0

- [ ] **Distributed tracing**
  - OpenTelemetry integration
  - Trace API requests across manager cluster
  - Trace task scheduling and execution
  - Export to Jaeger or Zipkin

- [ ] **Enhanced event streaming**
  - Complete gRPC streaming implementation (protobuf fix needed)
  - CLI: `warren events --follow` (real-time event stream)
  - Event filtering by type, service, node
  - Event history retention in BoltDB

- [ ] **Audit logging**
  - Log all API operations (who, what, when)
  - Store audit logs in BoltDB
  - CLI: `warren audit list` (query audit log)
  - Export audit logs to external systems

**Rationale**: Valuable for production operations and compliance but not blocking for v1.0.

---

### Multi-Document YAML Support
**Priority**: [LOW]
**Status**: Deferred

- [ ] **Multi-resource YAML files**
  - Support multiple resources in single YAML file
  - Separate by `---` document separator
  - Apply all resources in order
  - Test: Single file with service + secret + volume → All created

- [ ] **YAML directory apply**
  - `warren apply -f ./manifests/` (apply all YAML in directory)
  - Recursive option for nested directories
  - Test: Apply entire application stack from directory

**Rationale**: YAML apply works for single resources. Multi-doc support nice-to-have but not critical.

---

## Milestone 5: Open Source & Ecosystem

**Status**: Not started - planned for after M4 complete

All M5 features are future work. See main todo.md for full details. Key items:

### High Priority for Public Release

- [ ] **Repository setup** (LICENSE, CODE_OF_CONDUCT, CONTRIBUTING)
- [ ] **Comprehensive documentation** (user guide, API reference, architecture)
- [ ] **CI/CD automation** (GitHub Actions for lint, test, build, release)
- [ ] **Package distribution** (Homebrew formula, APT packages, Docker Hub)

### Medium Priority

- [ ] **Community channels** (GitHub Discussions, Discord, social media)
- [ ] **Example applications** (Multi-tier apps, stateful services)
- [ ] **Migration guides** (Docker Swarm → Warren, Compose → Warren)

### Nice to Have

- [ ] **Integrations** (Grafana dashboards, Terraform provider, GitHub Actions)
- [ ] **Blog content** (Launch post, architecture deep-dive, comparisons)
- [ ] **Conference talks** (KubeCon, HashiConf submissions)

---

## Future Milestones (Post-v1.0)

These are major feature additions planned for future versions. See main todo.md for details.

### Milestone 6: Built-in Ingress (v1.1)
**Priority**: [FUTURE]

- HTTP reverse proxy with path/host-based routing
- TLS termination with Let's Encrypt integration
- Rate limiting and request filtering

### Milestone 7: Service Mesh (v1.2)
**Priority**: [FUTURE]

- Sidecar injection for service-to-service communication
- Automatic mTLS between services
- Traffic policies (retry, timeout, circuit breaking)
- Observability (request tracing, metrics)

### Milestone 8: Multi-Cluster Federation (v2.0)
**Priority**: [FUTURE]

- Cross-cluster service discovery
- Global load balancing across clusters
- Federated secrets and configurations
- Cluster failover and disaster recovery

### Milestone 9: Extensibility (v2.0)
**Priority**: [FUTURE]

- Plugin SDK (custom schedulers, volume drivers, network drivers)
- Webhook system (admission control, validation, mutation)
- Custom Resource Definitions (CRDs) for extending Warren

---

## Backlog Management

### Adding Items

When deferring features from active milestones:

1. Add to appropriate category in this document
2. Include priority level and rationale
3. Link to related issues/discussions if applicable
4. Remove from active todo.md milestone

### Prioritization Criteria

**HIGH Priority**:
- Required for production stability (memory optimization, load testing)
- Critical for adoption (documentation, common use cases)
- Security or reliability issues

**MEDIUM Priority**:
- Valuable enhancements (blue/green deployment, canary)
- Platform compatibility (WireGuard userspace, arch-aware scheduling)
- Quality-of-life improvements (better health checks, log streaming)

**LOW Priority**:
- Convenience features (short alias, multi-doc YAML)
- Edge cases and specialized use cases

**FUTURE Priority**:
- Major feature additions for post-v1.0 versions
- Breaking changes or architectural enhancements
- Advanced enterprise features

### Review Schedule

- **Weekly**: Review HIGH priority items, promote to active work if capacity
- **Monthly**: Review MEDIUM priority items, plan for inclusion in upcoming milestones
- **Quarterly**: Review LOW and FUTURE priority items, archive if no longer relevant

---

## Notes

**Last Reviewed**: 2025-10-10
**Next Review**: 2025-10-17

**Key Decisions**:
- Focus on completing M4 optimization tasks before moving to M5
- Prioritize load testing and memory optimization over optional features
- Defer deployment strategies (blue/green, canary) until M5 or later
- Keep backlog lean - only track features that have been explicitly discussed

**Changes Since Last Review**:
- Initial backlog creation
- Moved all deferred M1-M4 features to backlog
- Categorized by milestone and priority
