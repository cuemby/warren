# Deployment Strategies

Warren v1.3.0 introduces production-grade deployment strategies to enable zero-downtime deployments with different risk/speed tradeoffs.

## Overview

Warren supports three deployment strategies:

1. **Rolling Update** (Default) - Update containers one-at-a-time or in small batches
2. **Blue-Green** - Deploy full new version, switch traffic instantly
3. **Canary** - Gradually shift traffic from old to new version

All strategies integrate with Warren's health check system and support automatic rollback on failures.

---

## Rolling Update

**When to use**: Default strategy for most deployments. Balances speed with safety.

**How it works**:
1. Update containers in batches (configurable parallelism)
2. Wait for each container to be healthy before proceeding
3. Continue until all containers are updated
4. Automatic rollback if failures exceed threshold

**Example**:
```bash
# Basic rolling update (default strategy)
warren service update web --image nginx:1.21

# With custom parallelism and delay
warren service update web --image nginx:1.21 \
  --strategy rolling \
  --parallelism 2 \
  --delay 10s
```

**Configuration**:
```yaml
# Service YAML
updateConfig:
  parallelism: 2              # Update 2 containers at a time
  delay: 10s                  # Wait 10s between batches
  failureAction: rollback     # rollback | pause | continue
  maxSurge: 1                 # Max extra containers during update
  maxUnavailable: 0           # Max containers that can be down
  healthCheckGracePeriod: 30s # Wait for health checks
```

**Pros**:
- Gradual rollout minimizes risk
- Uses existing infrastructure
- Resource efficient (no extra capacity needed)

**Cons**:
- Slower than blue-green (sequential updates)
- Mixed versions running during deployment
- Rollback requires reverse rolling update

---

## Blue-Green Deployment

**When to use**: When you need instant traffic switching and can afford temporary 2x capacity.

**How it works**:
1. Create full "green" environment with new version
2. Wait for all green containers to be healthy
3. Switch traffic from "blue" (old) to "green" (new) instantly
4. Keep blue version as standby for quick rollback
5. Clean up blue version after grace period

**Example**:
```bash
# Blue-green deployment
warren service update api --image myapi:v2 --strategy blue-green

# Rollback if needed (instant switch back to blue)
warren service rollback api
```

**Architecture**:
```
Before:
  Blue (v1) ────> 100% traffic

During:
  Blue (v1) ────> 0% traffic (standby)
  Green (v2) ───> 100% traffic (active)

Rollback:
  Blue (v1) ────> 100% traffic (active)
  Green (v2) ───> 0% traffic (rolled back)
```

**Configuration**:
```yaml
updateConfig:
  strategy: blue-green
  healthCheckGracePeriod: 30s    # Wait before switching
  blueGreenGracePeriod: 5m       # Keep blue for rollback
  autoRollbackEnabled: true      # Auto rollback on failure
  failureThresholdPercent: 10    # Error rate trigger
```

**Pros**:
- Instant traffic switch (zero downtime)
- Quick rollback (instant switch back)
- Test green environment before switching
- Clean separation of old/new versions

**Cons**:
- Requires 2x capacity temporarily
- More resource intensive
- Database migrations need special handling

**Best Practices**:
- Ensure health checks are comprehensive
- Test green environment before switching
- Monitor error rates after switch
- Keep blue version for grace period
- Use for critical services with good health checks

---

## Canary Deployment

**When to use**: When you want gradual rollout with real traffic validation.

**How it works**:
1. Deploy small "canary" version (e.g., 1 container)
2. Route small % of traffic to canary (10%)
3. Monitor canary health and metrics
4. Gradually increase traffic (10% → 25% → 50% → 100%)
5. Wait for stability window between steps
6. Auto-rollback if canary fails health checks

**Example**:
```bash
# Canary with default steps (10%, 25%, 50%, 100%)
warren service update web --image nginx:1.21 --strategy canary

# Custom canary steps
warren service update web --image nginx:1.21 \
  --strategy canary \
  --canary-steps 10,50,100 \
  --canary-window 300
```

**Traffic Progression**:
```
Step 1: Old (90%) + Canary (10%)  ──> Wait 5min
Step 2: Old (75%) + Canary (25%)  ──> Wait 5min
Step 3: Old (50%) + Canary (50%)  ──> Wait 5min
Step 4: Old (0%)  + Canary (100%) ──> Done
```

**Configuration**:
```yaml
updateConfig:
  strategy: canary
  canarySteps: [10, 25, 50, 100]     # Traffic % at each step
  canaryStabilityWindow: 5m          # Wait between steps
  healthCheckGracePeriod: 30s        # Initial health wait
  autoRollbackEnabled: true          # Rollback on failure
  failureThresholdPercent: 10        # Error rate trigger
```

**Pros**:
- Gradual rollout with real traffic
- Early problem detection with minimal impact
- Automatic rollback on failures
- Fine-grained control over risk

**Cons**:
- Slower deployment (stability windows)
- More complex than rolling update
- Requires traffic routing infrastructure
- Need good monitoring to detect issues

**Best Practices**:
- Start with small canary (10%)
- Monitor error rates and latency
- Use stability windows to detect issues
- Have comprehensive health checks
- Set appropriate failure thresholds

---

## Comparison

| Feature | Rolling | Blue-Green | Canary |
|---------|---------|------------|--------|
| Speed | Medium | Fast | Slow |
| Risk | Medium | Low | Very Low |
| Resource Usage | Low | High (2x) | Medium |
| Rollback Speed | Slow | Instant | Medium |
| Complexity | Low | Medium | High |
| Best For | Default | Critical services | New features |

---

## Health Check Integration

All strategies wait for containers to pass health checks before proceeding:

```yaml
healthCheck:
  type: HTTP
  http:
    path: /health
    port: 8080
  intervalSeconds: 10
  timeoutSeconds: 5
  retries: 3
```

**Grace Period**: Wait time before starting health checks (default: 30s)

```bash
# Adjust grace period
warren service create web --image nginx \
  --health-http /health \
  --health-interval 10 \
  --health-timeout 5 \
  --health-retries 3
```

---

## Automatic Rollback

Warren can automatically rollback failed deployments:

**Triggers**:
- Health check failures during deployment
- Error rate exceeds threshold (if configured)
- Container start failures

**Configuration**:
```yaml
updateConfig:
  autoRollbackEnabled: true
  failureThresholdPercent: 10  # Rollback if >10% error rate
```

**Manual Rollback**:
```bash
# Rollback to previous version
warren service rollback web

# Check deployment status
warren service inspect web
```

---

## Versioning & Labels

Warren tracks deployments using labels:

```yaml
labels:
  warren.deployment.version: "a1b2c3d4"    # Unique version ID
  warren.deployment.state: "active"        # active | standby | canary
  warren.deployment.strategy: "blue-green" # Deployment strategy used
  warren.deployment.original-service: "abc123"  # Original service ID
```

**States**:
- `active` - Currently serving traffic
- `standby` - Available for rollback
- `canary` - Receiving % of traffic
- `rolling` - Being updated
- `failed` - Deployment failed
- `rolled-back` - Was rolled back

---

## Metrics

Monitor deployments via Prometheus metrics:

```promql
# Total deployments by strategy and status
warren_deployments_total{strategy="blue-green", status="success"}

# Deployment duration
histogram_quantile(0.95, warren_deployment_duration_seconds{strategy="canary"})

# Rollback rate
rate(warren_deployments_rolled_back_total[5m])
```

**Available Metrics**:
- `warren_deployments_total` - Counter by strategy and status
- `warren_deployment_duration_seconds` - Histogram by strategy
- `warren_deployments_rolled_back_total` - Counter by strategy and reason

---

## Examples

### Simple Web Application
```bash
# Create service with rolling update (default)
warren service create web \
  --image nginx:1.20 \
  --replicas 3 \
  --publish 80:80

# Update with rolling strategy
warren service update web --image nginx:1.21
```

### Critical API with Blue-Green
```bash
# Create API service
warren service create api \
  --image myapi:v1 \
  --replicas 5 \
  --health-http /health

# Blue-green deployment
warren service update api --image myapi:v2 --strategy blue-green

# Instant rollback if issues
warren service rollback api
```

### New Feature with Canary
```bash
# Create service
warren service create app \
  --image myapp:stable \
  --replicas 10

# Canary deployment with custom steps
warren service update app \
  --image myapp:new-feature \
  --strategy canary \
  --canary-steps 5,10,25,100 \
  --canary-window 600
```

---

## Troubleshooting

### Deployment Stuck
```bash
# Check service status
warren service inspect web

# Check container health
warren service ps web

# View logs
docker logs <container-id>
```

### Health Checks Failing
```bash
# Increase grace period
warren service update web --image nginx:1.21 \
  --health-grace-period 60

# Check health endpoint manually
curl http://<container-ip>:8080/health
```

### Rollback Not Working
```bash
# List all service versions
warren service list | grep web

# Check standby version exists
warren service inspect web-<version>

# Manual cleanup if needed
warren service delete web-<old-version>
```

---

## Best Practices

1. **Always use health checks** - Essential for safe deployments
2. **Start conservative** - Use rolling updates by default
3. **Test before production** - Try blue-green/canary in staging first
4. **Monitor metrics** - Watch error rates during deployments
5. **Set grace periods** - Allow time for containers to start
6. **Use labels** - Tag deployments for tracking
7. **Plan rollback** - Always have rollback strategy ready
8. **Resource planning** - Blue-green needs 2x capacity
9. **Database migrations** - Handle separately from code deploys
10. **Document strategy** - Record why you chose specific strategy

---

## Migration Guide

### From v1.2 to v1.3

All existing services continue to use rolling updates by default. No action required.

**To adopt new strategies**:

```bash
# Existing service (continues with rolling)
warren service update web --image nginx:1.21

# Opt-in to blue-green
warren service update web --image nginx:1.21 --strategy blue-green

# Opt-in to canary
warren service update web --image nginx:1.21 --strategy canary
```

**Breaking Changes**: None - fully backward compatible

---

## See Also

- [Health Checks](health-checks.md) - Configure health checking
- [Observability](observability.md) - Monitor deployments
- [Operational Runbooks](operational-runbooks.md) - Deployment procedures
- [CLI Reference](cli-reference.md) - Complete command reference
