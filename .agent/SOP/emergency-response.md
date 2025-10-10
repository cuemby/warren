# Emergency Response Playbook

## Overview

This document defines procedures for handling production incidents and emergencies. Time is critical - follow this playbook to respond effectively.

## Incident Severity Levels

| Level | Response Time | Impact | Examples |
|-------|--------------|--------|----------|
| **SEV-1** | < 15 minutes | Critical | Production down, data loss, security breach |
| **SEV-2** | < 1 hour | High | Major feature broken, significant degradation |
| **SEV-3** | < 4 hours | Medium | Minor feature issue, partial functionality |
| **SEV-4** | < 24 hours | Low | Cosmetic issues, documentation errors |

---

## Emergency Response Process

### 1. ASSESS [0-5 minutes]

#### Determine Severity

```bash
# Check system health
$HEALTH_CHECK production

# Check error rates
$METRICS_TOOL errors --last=15m

# Check user reports
$SUPPORT_TOOL recent-tickets --status=open
```

#### Key Questions

- How many users affected?
- Is data at risk?
- Is security compromised?
- Can we contain the issue?
- Do we need external help?

#### Assign Severity

Use the table above to assign SEV-1 through SEV-4

---

### 2. COMMUNICATE [0-10 minutes]

#### For SEV-1 & SEV-2

**Immediate Actions**:

```text
1. Page on-call engineer
2. Notify incident commander
3. Create incident channel (#incident-YYYYMMDD-###)
4. Post to status page
5. Alert stakeholders
```

**Communication Template**:

```markdown
🚨 INCIDENT: [SEV-X] [Brief Description]

**Status**: Investigating
**Impact**: [User-facing impact]
**Started**: [Timestamp]
**Updates**: Every 15 minutes

**Team**:
- Incident Commander: @name
- Technical Lead: @name
- Communications: @name
```

#### For SEV-3 & SEV-4

- Create ticket
- Notify team via slack/email
- No status page update needed
- Provide realistic timeline

---

### 3. MITIGATE [Immediate]

#### Stop the Bleeding

**Priority Order**:

1. Protect data integrity
2. Restore service availability
3. Fix underlying issue

**Quick Mitigation Options**:

```bash
# Option 1: Feature flag disable
$FEATURE_FLAG_TOOL disable <feature-name>

# Option 2: Rollback deployment
$DEPLOY_TOOL rollback --to-version=<last-good-version>

# Option 3: Scale resources
$ORCHESTRATOR scale --replicas=10

# Option 4: Circuit breaker
$SERVICE_MESH break-circuit <service-name>

# Option 5: Traffic routing
$LOAD_BALANCER route-away <bad-instance>
```

**Verify Mitigation**:

```bash
# Check health
$HEALTH_CHECK production

# Check metrics
$METRICS_TOOL dashboard --last=5m

# Check logs
$LOGGING_TOOL tail --filter=error --last=1m
```

---

### 4. INVESTIGATE [Parallel to Mitigation]

#### Gather Evidence

```bash
# Recent deployments
$DEPLOY_TOOL history --last=24h

# Error logs
$LOGGING_TOOL search --level=error --last=1h

# Performance metrics
$METRICS_TOOL query "error_rate" --last=2h

# Database status
$DATABASE_TOOL status

# Infrastructure status
$CLOUD_PROVIDER status
```

#### Root Cause Analysis

- What changed recently?
- When did the issue start?
- What's different from normal behavior?
- Are there patterns in the errors?
- What do logs tell us?

---

### 5. RESOLVE [Target: < Response Time]

#### Implement Fix

**For Hotfix**:

```bash
# Create hotfix branch
$VCS_TOOL checkout -b hotfix/issue-description

# Implement minimal fix
# Edit files...

# Test fix
$TEST_RUNNER run

# Commit
$VCS_TOOL commit -m "fix: [SEV-X] resolve critical issue"

# Push
$VCS_TOOL push origin hotfix/issue-description

# Fast-track deployment
$DEPLOY_TOOL deploy --environment=production --fast-track
```

**For Configuration Change**:

```bash
# Update configuration
$CONFIG_TOOL set <key>=<value> --environment=production

# Verify change
$CONFIG_TOOL get <key> --environment=production

# Reload services if needed
$ORCHESTRATOR restart <service-name>
```

#### Verify Resolution

```bash
# Health check
$HEALTH_CHECK production

# Error rates back to normal?
$METRICS_TOOL errors --last=15m

# User reports stopped?
$SUPPORT_TOOL recent-tickets --last=15m

# Run smoke tests
$TEST_RUNNER run --suite=smoke --environment=production
```

---

### 6. REVIEW [Within 48 hours]

#### Post-Mortem Meeting

**Attendees**:

- Incident responders
- Engineering lead
- Product manager
- Customer support (if customer-facing)

**Agenda** (1 hour):

1. Timeline review (10 min)
2. Root cause analysis (15 min)
3. What went well (10 min)
4. What could be better (15 min)
5. Action items (10 min)

#### Post-Mortem Document

```markdown
# Post-Mortem: [Incident Title]

## Summary
Brief description of what happened

## Impact
- Users affected: [number/percentage]
- Duration: [start time] to [end time]
- Services impacted: [list]
- Data loss: [yes/no, details]

## Timeline
| Time | Event |
|------|-------|
| 14:23 | Initial detection |
| 14:25 | Incident declared SEV-1 |
| 14:30 | Mitigation started |
| 14:45 | Service restored |
| 15:00 | Incident resolved |

## Root Cause
Detailed explanation of what caused the incident

## Resolution
How the issue was resolved

## What Went Well
- Positive aspects of response
- What worked as planned

## What Could Be Better
- Areas for improvement
- Process gaps identified

## Action Items
- [ ] Action 1 - Owner - Due Date
- [ ] Action 2 - Owner - Due Date
- [ ] Action 3 - Owner - Due Date

## Lessons Learned
Key takeaways for future incidents
```

---

## Rollback Procedures

### When to Rollback

**Immediate Rollback**:

- Production completely down
- Data corruption detected
- Security breach identified
- Error rate > 10%

**Consider Rollback**:

- Major feature completely broken
- Performance degradation > 50%
- Multiple unrelated issues
- Unable to fix quickly

**Hotfix Instead**:

- Issue is well understood
- Fix is simple and quick
- Rollback has side effects
- Already partially mitigated

### Rollback Commands

```bash
# Check rollback target
$DEPLOY_TOOL history --environment=production

# Perform rollback
$DEPLOY_TOOL rollback --to-version=<version> --environment=production

# Verify rollback
$HEALTH_CHECK production

# Monitor metrics
$METRICS_TOOL watch --duration=15m

# Notify stakeholders
# Update status page
```

---

## Emergency Contacts

### Escalation Path

```text
Level 1: On-call Engineer
  ↓ (if unresolved in 30 min)
Level 2: Engineering Manager
  ↓ (if SEV-1 or unresolved in 1 hour)
Level 3: VP Engineering / CTO
  ↓ (if major impact or data breach)
Level 4: CEO / Legal (for legal/PR issues)
```

### Contact Methods

- Primary: PagerDuty / On-call system
- Secondary: Phone call
- Tertiary: Email + SMS

---

## Common Emergency Scenarios

### Scenario: Production Database Down

```bash
# 1. Check database status
$DATABASE_TOOL status

# 2. Try restart (if safe)
$DATABASE_TOOL restart

# 3. Check replica status
$DATABASE_TOOL replica-status

# 4. Failover to replica if needed
$DATABASE_TOOL failover --to-replica

# 5. Verify application connectivity
$HEALTH_CHECK production
```

### Scenario: API Rate Limit Exceeded

```bash
# 1. Disable non-critical features
$FEATURE_FLAG_TOOL disable feature1 feature2

# 2. Implement request throttling
$API_GATEWAY set-rate-limit --rps=100

# 3. Contact provider for limit increase
# 4. Scale horizontally if possible
$ORCHESTRATOR scale --replicas=10
```

### Scenario: Security Breach Suspected

```bash
# 1. Isolate affected systems immediately
$FIREWALL block-all <affected-service>

# 2. Preserve evidence
$LOGGING_TOOL archive --last=48h

# 3. Notify security team
# 4. Notify legal team
# 5. Follow data breach protocol
# 6. DO NOT destroy evidence
```

### Scenario: Payment Processing Failed

```bash
# 1. Enable maintenance mode for payments
$FEATURE_FLAG_TOOL disable payment-processing

# 2. Check payment provider status
$PAYMENT_PROVIDER status

# 3. Verify database transactions
$DATABASE_TOOL query "SELECT * FROM payments WHERE status='pending' LIMIT 10"

# 4. Contact payment provider support
# 5. Implement retry logic if needed
```

---

## Communication Templates

### Initial Alert

```text
🚨 [SEV-X] Production Incident

We're experiencing [brief description].
Team is investigating. Updates every 15 minutes.

Status: https://status.example.com
```

### Progress Update

```text
📊 [SEV-X] Update #[N] - [HH:MM]

Current Status: [Investigating/Mitigating/Resolved]
Actions Taken: [brief description]
Next Steps: [what's happening next]
ETA: [if known]
```

### Resolution Notice

```text
✅ [SEV-X] Incident Resolved - [HH:MM]

Issue: [brief description]
Resolution: [what fixed it]
Duration: [total time]
Impact: [brief summary]

Post-mortem: [link/date when available]
```

---

## Decision Trees

### Should I Declare an Incident?

```text
Are users affected? ──NO──▶ Monitor closely
    │ YES
    ▼
Is it impacting core functionality? ──NO──▶ SEV-3 or SEV-4
    │ YES
    ▼
Can we fix quickly (< 15 min)? ──YES──▶ Fix then document
    │ NO
    ▼
DECLARE INCIDENT: SEV-1 or SEV-2
```

### Rollback vs Hotfix?

```text
Can we fix in < 30 min? ──YES──▶ Attempt hotfix
    │ NO
    ▼
Is rollback safe? ──YES──▶ Rollback
    │ NO/UNSURE
    ▼
Mitigate with feature flags + plan fix
```

---

## Prevention

### Proactive Measures

- Monitor key metrics continuously
- Set up meaningful alerts
- Test rollback procedures regularly
- Keep runbooks updated
- Practice incident response (game days)
- Maintain spare capacity

### Alert Fatigue Prevention

- Tune alert thresholds
- Remove noisy alerts
- Alert on user impact, not internal metrics
- Use alert grouping
- Implement on-call rotation

---

## Related Documentation

- [deployment.md](deployment.md) - Deployment procedures
- [../System/monitoring.md](../System/monitoring.md) - Monitoring setup
- [../System/automation.md](../System/automation.md) - Automation tools
- [quality-gates.md](quality-gates.md) - Prevention through quality

---

## Emergency Checklist

### SEV-1 Response Checklist

- [ ] Severity assessed and confirmed
- [ ] Incident channel created
- [ ] On-call engineer paged
- [ ] Incident commander assigned
- [ ] Status page updated
- [ ] Mitigation attempted
- [ ] Stakeholders notified
- [ ] Logs and metrics captured
- [ ] Resolution implemented
- [ ] Service verified healthy
- [ ] Post-mortem scheduled

---

**Remember**: Stay calm, communicate clearly, mitigate first, investigate second. Document everything for the post-mortem.
