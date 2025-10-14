# Milestone 8: Deployment Strategies - Implementation Plan

**Version**: 1.2.0 → 1.3.0
**Duration**: 2-3 weeks
**Status**: Planning
**Start Date**: 2025-10-13

---

## Overview

Milestone 8 adds production-grade deployment strategies to Warren, enabling zero-downtime deployments with multiple strategies:
- **Rolling Update** (Enhanced with parallelism, delays, failure handling)
- **Blue/Green Deployment** (Instant traffic switch)
- **Canary Deployment** (Gradual traffic migration)

### Goals

1. ✅ Enable zero-downtime deployments
2. ✅ Provide flexible deployment strategies
3. ✅ Integrate with existing health checks
4. ✅ Support automatic rollback on failures
5. ✅ Maintain Warren's simplicity (single binary, no external deps)

---

## Current State Analysis

### Existing Foundation ✅

**pkg/deploy/deploy.go** (Already exists):
- Deployer struct with manager reference
- Basic rolling update implementation
- Parallelism support
- Delay between batches
- Structured logging

**pkg/types/types.go** (Already defined):
```go
type DeployStrategy string
const (
    DeployStrategyRolling   = "rolling"
    DeployStrategyBlueGreen = "blue-green"
    DeployStrategyCanary    = "canary"
)

type UpdateConfig struct {
    Parallelism   int
    Delay         time.Duration
    FailureAction string        // "pause", "rollback", "continue"
    CanaryWeight  int           // 0-100 (for canary strategy)
}
```

**Service struct** (pkg/types/types.go:70):
- Has `UpdateConfig *UpdateConfig` field
- Already supports deployment configuration

### What Needs to be Built

1. **Service Versioning System**
   - Add version labels to services
   - Track deployment versions
   - Enable side-by-side versions (blue/green)

2. **Blue/Green Deployment**
   - Create green (new) version alongside blue (current)
   - Wait for green to be healthy
   - Switch traffic instantly
   - Keep blue for rollback

3. **Canary Deployment**
   - Create canary version (small % of replicas)
   - Gradually increase traffic weight
   - Monitor health and metrics
   - Promote or rollback

4. **Enhanced Rolling Updates**
   - Health check integration (wait for healthy before next batch)
   - Automatic rollback on failure
   - Max surge / max unavailable controls

5. **Traffic Management**
   - DNS-based traffic routing (leverage existing DNS server)
   - Service endpoint resolution
   - Version-aware routing

---

## Architecture Design

### Service Versioning

**Approach**: Use labels to track deployment versions

```go
type Service struct {
    // ... existing fields
    Labels map[string]string
}

// Version tracking labels:
// "warren.deployment.version": "v1", "v2", etc.
// "warren.deployment.strategy": "blue-green", "canary", "rolling"
// "warren.deployment.state": "active", "standby", "canary"
```

### Blue/Green Deployment

**Flow**:
1. Clone current service as "green" with new image
2. Schedule green containers
3. Wait for all green containers healthy
4. Update DNS/service endpoints to point to green
5. Mark blue as "standby" (keep for rollback)
6. After grace period, optionally delete blue

**Implementation**:
- Green service: `myservice-green` (temporary)
- Blue service: `myservice` (current)
- DNS update to point to green
- Single command rollback if needed

### Canary Deployment

**Flow**:
1. Create canary version (e.g., 10% of replicas)
2. Route 10% of traffic to canary
3. Monitor canary health + metrics
4. Gradually increase: 10% → 25% → 50% → 100%
5. Each step waits for stability window (e.g., 5 minutes)
6. Auto-rollback if failure threshold exceeded

**Implementation**:
- Canary containers tagged with version label
- DNS returns mix of stable + canary IPs (weighted)
- Metrics track canary vs stable error rates
- Configurable promotion steps

### Enhanced Rolling Updates

**Improvements**:
1. **Health Check Integration**:
   - Wait for container healthy before proceeding
   - Configurable health check timeout
   - Fail fast if health checks fail

2. **Automatic Rollback**:
   - Monitor error rates during rollout
   - Rollback if threshold exceeded (e.g., >5% errors)
   - Keep previous image for instant rollback

3. **Max Surge / Max Unavailable**:
   - Max surge: How many extra containers during update (default: 1)
   - Max unavailable: How many containers can be down (default: 0)
   - Ensures availability during rollout

### Traffic Management

**DNS-Based Routing** (leverage existing pkg/dns):
- Service resolution returns container IPs
- For canary: Return weighted mix of versions
- For blue/green: Switch all IPs atomically
- No external load balancer needed

**Algorithm**:
```go
func ResolveServiceWithStrategy(service *Service) []net.IP {
    if service.Strategy == "canary" {
        return weightedIPs(stableContainers, canaryContainers, canaryWeight)
    } else if service.Strategy == "blue-green" {
        return activeVersionIPs(service)
    } else {
        return allHealthyIPs(service)
    }
}
```

---

## Implementation Plan

### Week 1: Blue/Green Deployment

**Day 1-2: Service Versioning**
- [ ] Add deployment version tracking to Service struct
- [ ] Implement service cloning with version labels
- [ ] Add version queries (GetServiceVersion, ListVersions)
- [ ] Unit tests for versioning

**Day 3-4: Blue/Green Logic**
- [ ] Implement BlueGreenDeployer
- [ ] Create green version
- [ ] Wait for green healthy (health check integration)
- [ ] Switch traffic (DNS update)
- [ ] Rollback function
- [ ] Integration tests

**Day 5: CLI & API**
- [ ] Add `warren service update --strategy blue-green`
- [ ] Add `warren service rollback`
- [ ] gRPC API methods
- [ ] CLI documentation

**Deliverable**: Blue/green deployment working end-to-end

---

### Week 2: Canary Deployment

**Day 1-2: Canary Logic**
- [ ] Implement CanaryDeployer
- [ ] Create canary containers (% of replicas)
- [ ] Gradual promotion logic (10% → 25% → 50% → 100%)
- [ ] Stability window checks
- [ ] Unit tests

**Day 3-4: Weighted Traffic Routing**
- [ ] Enhance DNS resolver for weighted responses
- [ ] Implement traffic splitting algorithm
- [ ] Health-based exclusion (unhealthy containers removed from DNS)
- [ ] Integration tests

**Day 5: Monitoring & Auto-Rollback**
- [ ] Track canary error rates via metrics
- [ ] Auto-rollback on failure threshold
- [ ] CLI progress monitoring (`warren service canary status`)
- [ ] Documentation

**Deliverable**: Canary deployment with auto-rollback

---

### Week 3: Enhanced Rolling Updates & Polish

**Day 1-2: Rolling Update Enhancements**
- [ ] Health check integration (wait for healthy)
- [ ] Max surge / max unavailable controls
- [ ] Automatic rollback on failure
- [ ] Configurable failure thresholds
- [ ] Unit tests

**Day 3: Integration Testing**
- [ ] E2E test: Blue/green deployment
- [ ] E2E test: Canary deployment with promotion
- [ ] E2E test: Rolling update with auto-rollback
- [ ] E2E test: Failure scenarios
- [ ] Performance testing

**Day 4: Documentation**
- [ ] Deployment strategies guide (concepts)
- [ ] CLI reference updates
- [ ] API documentation
- [ ] Migration guide from rolling to blue/green
- [ ] Examples (YAML configs)

**Day 5: Polish & Release**
- [ ] Code review and refactoring
- [ ] Update observability guide (deployment metrics)
- [ ] Update operational runbooks (deployment procedures)
- [ ] Version bump to v1.3.0
- [ ] Release notes

**Deliverable**: Warren v1.3.0 with complete deployment strategies

---

## API Design

### Service Update API

```go
// UpdateServiceRequest with deployment strategy
message UpdateServiceRequest {
    string service_id = 1;
    string image = 2;
    DeploymentStrategy strategy = 3;
    UpdateConfig update_config = 4;
}

enum DeploymentStrategy {
    ROLLING = 0;
    BLUE_GREEN = 1;
    CANARY = 2;
}

message UpdateConfig {
    int32 parallelism = 1;
    int64 delay_seconds = 2;
    string failure_action = 3;  // "pause", "rollback", "continue"
    int32 canary_weight = 4;
    int32 max_surge = 5;
    int32 max_unavailable = 6;
    repeated int32 canary_steps = 7;  // [10, 25, 50, 100]
    int64 stability_window_seconds = 8;
}
```

### CLI Commands

```bash
# Blue/Green
warren service update myservice --image v2 --strategy blue-green

# Canary (default steps: 10, 25, 50, 100)
warren service update myservice --image v2 --strategy canary

# Canary (custom steps)
warren service update myservice --image v2 --strategy canary --canary-steps 5,10,50,100

# Enhanced Rolling
warren service update myservice --image v2 \
  --strategy rolling \
  --parallelism 2 \
  --max-surge 1 \
  --max-unavailable 0

# Monitor canary progress
warren service canary status myservice

# Promote canary early (skip remaining steps)
warren service canary promote myservice

# Rollback
warren service rollback myservice

# List deployment history
warren service history myservice
```

---

## Metrics

### New Metrics to Add

```go
// Deployment metrics
warren_deployments_total{strategy="rolling|blue-green|canary", status="success|failed"}
warren_deployment_duration_seconds{strategy}
warren_canary_promotion_steps_total
warren_rollbacks_total{reason="health_check|error_rate|manual"}

// Traffic split metrics (canary)
warren_canary_traffic_weight{service, version}
warren_canary_error_rate{service, version}
```

---

## Testing Strategy

### Unit Tests
- Service versioning logic
- Blue/green state machine
- Canary promotion algorithm
- Traffic weighted distribution
- Rollback logic

### Integration Tests
- Full blue/green deployment cycle
- Canary with multiple promotion steps
- Rollback scenarios
- Health check failure handling
- DNS traffic routing verification

### E2E Tests (test/e2e/)
- Deploy service with blue/green strategy
- Deploy service with canary strategy
- Simulate container failures during deployment
- Verify traffic routing
- Test automatic rollback

---

## Success Criteria

### Functional Requirements
- [ ] Blue/green deployment works end-to-end
- [ ] Canary deployment with gradual promotion
- [ ] Rolling update enhanced with health checks
- [ ] Automatic rollback on failures
- [ ] Traffic routing works correctly
- [ ] All deployment strategies tested

### Non-Functional Requirements
- [ ] Zero downtime during deployments
- [ ] Rollback completes in <10 seconds
- [ ] Deployment status visible in CLI/API
- [ ] Metrics track all deployment operations
- [ ] Documentation complete

### Production Readiness
- [ ] Comprehensive test coverage (>80%)
- [ ] Deployment strategies guide written
- [ ] Operational runbooks updated
- [ ] Performance validated (no regression)
- [ ] Backward compatible with existing services

---

## Risk Analysis

### Technical Risks

**Risk**: DNS caching causes stale routing during blue/green switch
**Mitigation**: Use short TTL (5-10s) for service records, document DNS behavior

**Risk**: Canary traffic split not precise (DNS round-robin not weighted)
**Mitigation**: Document limitation, consider nginx/envoy for precise splitting in future

**Risk**: Rollback leaves orphaned containers
**Mitigation**: Cleanup logic in rollback, garbage collection

**Risk**: Deployment interruption during manager failover
**Mitigation**: Store deployment state in Raft, resume on new leader

### Operational Risks

**Risk**: Users expect instant canary promotion
**Mitigation**: Provide `warren service canary promote` command

**Risk**: Confusion between blue/green and rolling
**Mitigation**: Clear documentation with decision guide

**Risk**: Automatic rollback too aggressive
**Mitigation**: Configurable failure thresholds, default to conservative values

---

## Dependencies

### External
- None (Warren is self-contained)

### Internal
- Existing health check system (pkg/health)
- DNS resolver (pkg/dns)
- Scheduler (pkg/scheduler)
- Metrics (pkg/metrics)

---

## Backward Compatibility

### Existing Services
- Services without strategy specified default to "rolling"
- Existing UpdateConfig fields remain compatible
- No breaking changes to API

### Migration Path
- Existing services continue to work
- Opt-in to new strategies via CLI flag
- Documentation provides migration guide

---

## Documentation Plan

1. **Deployment Strategies Guide** (docs/deployment-strategies.md)
   - Concepts (blue/green, canary, rolling)
   - When to use each strategy
   - Configuration reference
   - Examples and best practices

2. **CLI Reference Updates** (docs/cli-reference.md)
   - New flags and commands
   - Examples for each strategy

3. **API Documentation** (docs/api.md)
   - New gRPC methods
   - Message definitions

4. **Migration Guide** (docs/migration/deployment-strategies.md)
   - Upgrading from v1.2.0 to v1.3.0
   - Converting rolling to blue/green
   - Best practices

5. **Operational Runbooks Updates** (docs/operational-runbooks.md)
   - Blue/green deployment procedure
   - Canary deployment procedure
   - Rollback procedures

---

## Release Plan

### Version: v1.3.0

**Release Date**: ~2-3 weeks from start (early November 2025)

**Release Notes Highlights**:
- ✨ Blue/Green deployment strategy
- ✨ Canary deployment with gradual rollout
- ✨ Enhanced rolling updates with health check integration
- ✨ Automatic rollback on failures
- ✨ DNS-based traffic routing
- 📊 Deployment metrics and monitoring
- 📚 Comprehensive deployment strategies guide

**Breaking Changes**: None

**Migration Required**: No (opt-in features)

---

## Next Steps

1. Read and understand existing deploy.go implementation
2. Design service versioning data model
3. Start Week 1 implementation (blue/green)
4. Create initial tests
5. Iterate based on testing feedback

---

## References

- [Existing deploy.go](../pkg/deploy/deploy.go)
- [Service types](../pkg/types/types.go)
- [Kubernetes Deployment Strategies](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/)
- [Docker Swarm Update Behavior](https://docs.docker.com/engine/swarm/swarm-tutorial/rolling-update/)

---

**Status**: Ready to begin implementation
**Next**: Start Week 1 - Blue/Green Deployment
