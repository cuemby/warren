# Deployment Standards

## Overview

Deployment is the process of releasing code to production environments. This document defines deployment best practices, checklists, and procedures.

## Pre-Deploy Checklist [REQUIRED]

### Code Quality

- [ ] All tests passing (unit, integration, E2E)
- [ ] Code review approved
- [ ] No linter warnings or errors
- [ ] Code coverage meets threshold
- [ ] No known bugs or blockers

### Security

- [ ] Security scan completed (no critical/high issues)
- [ ] Dependencies updated and scanned
- [ ] No secrets in code
- [ ] Authentication/authorization verified
- [ ] Input validation implemented

### Performance

- [ ] Performance benchmarks met
- [ ] Load testing completed (if applicable)
- [ ] Database migrations tested
- [ ] No N+1 query issues
- [ ] Caching configured

### Documentation

- [ ] README updated
- [ ] API documentation current
- [ ] Changelog updated
- [ ] Breaking changes documented
- [ ] Runbook created/updated

### Monitoring & Rollback

- [ ] Monitoring configured
- [ ] Alerts set up
- [ ] Logs configured
- [ ] Rollback plan documented
- [ ] Feature flags configured (if applicable)

---

## Deployment Strategies

### Blue-Green Deployment

```text
1. Deploy to "green" environment
2. Test green environment
3. Switch traffic from "blue" to "green"
4. Keep blue as rollback option
```

**Use when**: Zero-downtime requirement

### Canary Deployment

```text
1. Deploy to small subset of servers
2. Monitor for errors
3. Gradually increase traffic
4. Full rollout or rollback
```

**Use when**: Risk mitigation needed

### Rolling Deployment

```text
1. Deploy to one server at a time
2. Verify each deployment
3. Continue to next server
4. Rollback if issues detected
```

**Use when**: Limited resources

### Feature Flags

```text
1. Deploy code with feature disabled
2. Enable for internal users first
3. Enable for subset of users
4. Full rollout
```

**Use when**: Gradual feature release needed

---

## Environment Strategy

### Standard Environments

```text
Development → Staging → Production

- Development: Feature development and testing
- Staging: Pre-production validation
- Production: Live user environment
```

### Environment Parity

Keep environments as similar as possible:

- Same infrastructure
- Same configuration structure
- Same monitoring
- Same deployment process

---

## Deployment Workflow

### 1. Pre-Deploy

```bash
# Run full test suite
$TEST_RUNNER run

# Run linter
$LINTER check

# Run security scan
$SECURITY_SCANNER scan

# Build artifacts
$BUILD_TOOL build

# Tag release
$VCS_TOOL tag -a v1.2.3 -m "Release v1.2.3"
$VCS_TOOL push origin v1.2.3
```

### 2. Deploy to Staging

```bash
# Deploy to staging
$DEPLOY_TOOL deploy --environment=staging

# Run smoke tests
$TEST_RUNNER run --suite=smoke

# Verify deployment
$HEALTH_CHECK staging

# Run integration tests
$TEST_RUNNER run --suite=integration
```

### 3. Deploy to Production

```bash
# Create deployment record
$DEPLOY_TOOL start-deployment --environment=production

# Deploy
$DEPLOY_TOOL deploy --environment=production

# Health check
$HEALTH_CHECK production

# Monitor metrics
$METRICS_TOOL watch --duration=15m

# Mark deployment complete
$DEPLOY_TOOL complete-deployment
```

### 4. Post-Deploy

```bash
# Verify critical paths
$TEST_RUNNER run --suite=smoke --environment=production

# Check error rates
$METRICS_TOOL errors --last=1h

# Verify monitoring
$ALERT_TOOL verify

# Update status page (if applicable)
$STATUS_PAGE update "Deployed v1.2.3"
```

---

## Rollback Procedures

### When to Rollback

- Critical errors in production
- Performance degradation > 20%
- Security vulnerability introduced
- Data corruption detected
- Monitoring alerts firing

### Rollback Steps

```bash
# 1. Identify issue severity
# 2. Decide: rollback vs. hotfix

# Quick rollback
$DEPLOY_TOOL rollback --to-version=v1.2.2

# Or use feature flag
$FEATURE_FLAG_TOOL disable feature-name

# 3. Verify rollback
$HEALTH_CHECK production

# 4. Notify stakeholders
# 5. Post-mortem within 48 hours
```

### Rollback Checklist

- [ ] Rollback decision made and approved
- [ ] Rollback executed
- [ ] Health checks passing
- [ ] Monitoring verified
- [ ] Stakeholders notified
- [ ] Incident ticket created
- [ ] Post-mortem scheduled

---

## Database Migrations

### Migration Best Practices

- Make migrations reversible
- Test migrations on staging
- Run migrations separately from code deploy
- Have rollback migration ready
- Avoid data transformations in migrations

### Migration Workflow

```bash
# 1. Create migration
$MIGRATION_TOOL create add_user_email_index

# 2. Test locally
$MIGRATION_TOOL up

# 3. Test rollback
$MIGRATION_TOOL down

# 4. Deploy to staging
$MIGRATION_TOOL up --environment=staging

# 5. Verify staging
# Run tests, check data

# 6. Deploy to production
$MIGRATION_TOOL up --environment=production

# 7. Verify production
# Check data integrity
```

### Dangerous Operations

⚠️ **Require extra care**:

- Dropping tables
- Dropping columns
- Changing column types
- Large data transformations

---

## Zero-Downtime Deployments

### Strategies

1. **Backward-compatible changes**: New code works with old schema
2. **Feature flags**: Deploy disabled, enable after verification
3. **Blue-green**: Switch traffic only when ready
4. **Rolling updates**: Never all servers down simultaneously

### Multi-Phase Deployments

```text
Phase 1: Add new column (nullable)
Deploy: ✅ App continues working

Phase 2: Start writing to new column
Deploy: ✅ Both old and new columns updated

Phase 3: Backfill old data
Deploy: ✅ All data in new column

Phase 4: Start reading from new column
Deploy: ✅ App uses new column

Phase 5: Remove old column
Deploy: ✅ Cleanup complete
```

---

## Deployment Timing

### Best Times to Deploy

- ✅ Monday-Thursday (early in work day)
- ✅ After code freeze period
- ✅ When team is available for monitoring
- ✅ Low-traffic periods (if possible)

### Avoid Deploying

- ❌ Friday afternoons
- ❌ Before holidays/weekends
- ❌ During peak traffic
- ❌ When key people unavailable
- ❌ Multiple changes simultaneously

---

## Monitoring Post-Deploy

### Metrics to Watch

```yaml
error_rates:
  - 4xx errors
  - 5xx errors
  - Application exceptions

performance:
  - Response times (p50, p95, p99)
  - Database query times
  - API latency
  - Resource utilization

business:
  - User sign-ups
  - Transactions
  - Key feature usage
  - Conversion rates
```

### Monitoring Duration

- **First 15 minutes**: Intensive monitoring
- **First hour**: Regular checks
- **First day**: Periodic verification
- **First week**: Watch for trends

See [../System/monitoring.md](../System/monitoring.md)

---

## Deployment Artifacts

### What to Archive

- Built artifacts (binaries, containers)
- Configuration files (sanitized)
- Database migration scripts
- Deployment logs
- Test results
- Security scan results

### Artifact Storage

```bash
# Tag and push container
$CONTAINER_RUNTIME tag app:latest app:v1.2.3
$CONTAINER_RUNTIME push app:v1.2.3

# Store in artifact repository
$ARTIFACT_REPO upload build-artifacts.tar.gz

# Archive logs
$LOGGING_TOOL archive deployment-v1.2.3
```

---

## Communication

### Pre-Deploy

- Notify team of deployment window
- Share deployment plan
- Identify on-call person
- Brief stakeholders on changes

### During Deploy

- Update status page
- Monitor chat channels
- Watch metrics dashboard
- Be ready to rollback

### Post-Deploy

- Announce completion
- Share metrics
- Document any issues
- Thank the team

---

## Emergency Deploys

### When Justified

- Security vulnerabilities
- Data loss prevention
- Critical bug fixes
- Legal/compliance issues

### Emergency Process

```text
1. Assess severity [BLOCKER]
2. Get approval from lead/manager
3. Create hotfix branch
4. Implement minimal fix
5. Fast-track testing
6. Deploy to production
7. Monitor intensively
8. Post-mortem within 24 hours
```

See [emergency-response.md](emergency-response.md)

---

## Deployment Anti-Patterns

### ❌ DON'T

- Deploy untested code
- Deploy without rollback plan
- Deploy complex changes all at once
- Deploy when you can't monitor
- Skip documentation updates
- Ignore test failures
- Deploy Friday afternoon

### ✅ DO

- Test thoroughly before deploying
- Have rollback plan ready
- Deploy incrementally
- Monitor intensively post-deploy
- Update all documentation
- Fix all test failures
- Deploy early in week

---

## CI/CD Integration

### Automated Pipeline

```yaml
stages:
  - build:
      - compile code
      - run linters
      - run security scans

  - test:
      - unit tests
      - integration tests
      - coverage check

  - staging:
      - deploy to staging
      - run smoke tests
      - run E2E tests

  - production:
      - manual approval required
      - deploy to production
      - health checks
      - monitor metrics
```

See [../System/automation.md](../System/automation.md)

---

## Configuration Management

### Best Practices

- Store config in environment variables
- Never commit secrets
- Use secret management tools
- Validate configuration on startup
- Document all config options

### Config Structure

```bash
# Development
export DATABASE_URL="postgresql://localhost/app_dev"
export API_KEY="dev_key_xxx"

# Staging
export DATABASE_URL="postgresql://staging.db/app_staging"
export API_KEY="staging_key_xxx"

# Production
export DATABASE_URL="postgresql://prod.db/app_prod"
export API_KEY="${SECRET_MANAGER_API_KEY}"
```

---

## Related Documentation

- [emergency-response.md](emergency-response.md) - Incident response
- [quality-gates.md](quality-gates.md) - Pre-deploy checks
- [../System/automation.md](../System/automation.md) - CI/CD setup
- [../System/monitoring.md](../System/monitoring.md) - Monitoring setup

---

**Remember**: Deployment is not complete until monitoring confirms success. Always have a rollback plan ready.
