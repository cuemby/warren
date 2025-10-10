# Quality Gates

## Overview

Quality gates are checkpoints that ensure code meets standards before progressing to the next stage. This document defines the gates and requirements at each stage.

## Three-Stage Gate System

```text
Code → Pre-Commit → Pre-Merge → Pre-Deploy → Production
         ├─GATE 1   ├─GATE 2      ├─GATE 3
         │          │              │
         ├─Local    ├─PR Review    ├─Staging
         ├─Fast     ├─Thorough     ├─Complete
         └─Auto     └─Manual+Auto  └─Auto+Manual
```

---

## Gate 1: Pre-Commit [CRITICAL]

**Purpose**: Prevent broken code from entering version control

**When**: Before every commit

**Duration**: < 2 minutes

### Automated Checks

```bash
# Linting
$LINTER check

# Formatting
$FORMATTER check

# Unit tests
$TEST_RUNNER run --unit

# Type checking (if applicable)
$TYPE_CHECKER check

# Secret scanning
$SECRET_SCANNER scan

# Todo/Fixme detection
grep -r "FIXME\|TODO" src/
```

### Manual Checks

- [ ] Code compiles/runs
- [ ] No debugging code left in
- [ ] No commented-out code
- [ ] No console.log / print statements
- [ ] Tests written for new code
- [ ] Self-reviewed changes

### Failure Action

- **MUST FIX** before committing
- Do not bypass pre-commit hooks
- Do not commit with `--no-verify`

### Setup

```bash
# Install pre-commit hooks
$VCS_TOOL config core.hooksPath .git/hooks

# Make hook executable
chmod +x .git/hooks/pre-commit
```

See [../System/automation.md](../System/automation.md)

---

## Gate 2: Pre-Merge [REQUIRED]

**Purpose**: Ensure production-ready code quality

**When**: Before merging PR to main branch

**Duration**: Varies by PR size

### Automated Checks

```yaml
linting:
  status: REQUIRED
  tools: [$LINTER]

testing:
  unit_tests:
    status: REQUIRED
    coverage_minimum: 60-80%  # Per profile

  integration_tests:
    status: REQUIRED

  e2e_tests:
    status: RECOMMENDED

security:
  dependency_scan:
    status: REQUIRED
    block_on: [critical, high]

  code_scan:
    status: REQUIRED
    block_on: [critical]

  secret_scan:
    status: REQUIRED
    block_on: [any]

performance:
  benchmarks:
    status: RECOMMENDED
    regression_threshold: 10%

build:
  status: REQUIRED
  artifacts: REQUIRED
```

### Manual Checks

#### Code Review

- [ ] 1-2 approvals (depends on team size)
- [ ] All conversations resolved
- [ ] No [BLOCKER] comments unresolved

See [code-review.md](code-review.md)

#### Functional Review

- [ ] Acceptance criteria met
- [ ] Edge cases handled
- [ ] Error messages user-friendly
- [ ] Breaking changes documented

#### Documentation Review

- [ ] README updated (if needed)
- [ ] API docs updated (if needed)
- [ ] Changelog updated
- [ ] Migration guide (if breaking changes)

### Failure Action

- **MUST FIX** before merging
- Address all [BLOCKER] feedback
- Re-request review after fixes

### CI/CD Pipeline Example

```yaml
name: Pre-Merge Checks

on:
  pull_request:
    branches: [main]

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - run: $LINTER check

  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - run: $TEST_RUNNER run
      - run: $TEST_RUNNER run --coverage

  security:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - run: $SECURITY_SCANNER scan

  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - run: $BUILD_TOOL build
```

---

## Gate 3: Pre-Deploy [REQUIRED]

**Purpose**: Verify production readiness

**When**: Before deploying to production

**Duration**: Varies (typically 10-30 minutes)

### Automated Checks on Staging

```yaml
smoke_tests:
  status: REQUIRED
  suite: critical_paths

integration_tests:
  status: REQUIRED
  environment: staging

load_tests:
  status: RECOMMENDED
  scenarios: [normal, peak]
  duration: 10min

security_scan:
  status: REQUIRED
  full_scan: true

health_checks:
  status: REQUIRED
  endpoints: [/health, /ready]

database_migrations:
  status: REQUIRED
  tested_on: staging
  rollback_tested: true
```

### Manual Checks

#### Staging Verification

- [ ] Smoke tests passed
- [ ] Critical workflows tested manually
- [ ] Performance acceptable
- [ ] No errors in logs
- [ ] Monitoring dashboards healthy

#### Deployment Readiness

- [ ] Rollback plan documented
- [ ] Feature flags configured (if needed)
- [ ] Alerts configured
- [ ] On-call person identified
- [ ] Stakeholders notified

#### Documentation

- [ ] Runbook updated
- [ ] Deployment notes prepared
- [ ] Known issues documented
- [ ] Rollback procedure tested

### Failure Action

- **MUST FIX** critical issues
- **CAN DEFER** minor issues with tracking ticket
- **CANNOT DEPLOY** if any [BLOCKER] items fail

### Deployment Checklist

See [deployment.md](deployment.md)

---

## Coverage Requirements by Profile

| Profile | Unit | Integration | E2E | Security | Performance |
|---------|------|-------------|-----|----------|-------------|
| **MVP** | 40% | Basic | Optional | Basic | Best effort |
| **Startup** | 60% | Required | Smoke | Regular | 95% |
| **Enterprise** | 80% | Required | Required | Continuous | 99.9% |
| **Open Source** | 70% | Required | Smoke | Community | Variable |

---

## Quality Metrics

### Track These Metrics

```yaml
build_metrics:
  - build_success_rate
  - build_duration
  - flaky_test_count

code_quality:
  - test_coverage_trend
  - code_duplication
  - cyclomatic_complexity
  - technical_debt_ratio

deployment_metrics:
  - deployment_frequency
  - lead_time_for_changes
  - mean_time_to_recovery
  - change_failure_rate

defect_metrics:
  - bugs_found_in_review
  - bugs_found_in_staging
  - bugs_found_in_production
  - bug_escape_rate
```

### Quality Thresholds

```yaml
build:
  success_rate_minimum: 95%
  duration_maximum: 10min

test:
  flaky_test_maximum: 0
  coverage_minimum: "per profile"

deployment:
  rollback_rate_maximum: 5%
  mean_time_to_recovery_maximum: 1hour
```

---

## Bypassing Quality Gates

### When Allowed

- ⚠️ **Emergency hotfix** (SEV-1 incident)
- ⚠️ **Security patch** (critical vulnerability)
- ⚠️ **Data loss prevention** (imminent risk)

### Bypass Process

```text
1. Get approval from Engineering Lead/Manager
2. Document reason in commit message
3. Create follow-up ticket for proper fix
4. Monitor deployment intensively
5. Post-mortem within 24 hours
```

### Bypass Template

```bash
$VCS_TOOL commit -m "$(cat <<'EOF'
fix: [EMERGENCY] resolve critical security issue

BYPASS QUALITY GATES: SEV-1 incident
Approval: @engineering-lead
Reason: Active security exploit in production
Follow-up: TICKET-123

🚨 Emergency deployment - monitoring required
EOF
)"
```

---

## Common Gate Failures

### Lint Failures

```bash
# Fix automatically
$FORMATTER fix

# Verify
$LINTER check
```

### Test Failures

```bash
# Run specific test
$TEST_RUNNER run path/to/failing-test

# Fix and re-run all
$TEST_RUNNER run
```

### Coverage Failures

```bash
# Check coverage report
$TEST_RUNNER run --coverage

# Add missing tests
# Re-run verification
```

### Security Failures

```bash
# Update vulnerable dependency
$PACKAGE_MANAGER update package-name

# Re-scan
$SECURITY_SCANNER scan
```

---

## Gate Maintenance

### Regular Review

- Weekly: Review flaky tests
- Monthly: Review quality metrics
- Quarterly: Adjust thresholds if needed

### Continuous Improvement

- Add gates for recurring issues
- Remove noisy/unhelpful gates
- Tune thresholds based on data
- Automate more manual checks

---

## Integration with Tools

### Git Hooks

```bash
# .git/hooks/pre-commit
#!/bin/bash
$LINTER check || exit 1
$TEST_RUNNER run --unit || exit 1
$SECRET_SCANNER scan || exit 1
```

### CI/CD Status Checks

```yaml
# Required status checks in GitHub/GitLab
branch_protection:
  required_status_checks:
    - lint
    - test
    - security-scan
    - build

  required_reviews: 1

  dismiss_stale_reviews: true
```

### IDE Integration

- Lint on save
- Test on change
- Format on save
- Show coverage inline

---

## Anti-Patterns

### ❌ DON'T

- Skip gates "just this once"
- Commit with `--no-verify`
- Approve PRs without reviewing
- Deploy with failing tests
- Ignore security warnings
- Rush through gates

### ✅ DO

- Respect all quality gates
- Fix failures immediately
- Review code thoroughly
- Test before deploying
- Address security issues
- Take time for quality

---

## Related Documentation

- [testing.md](testing.md) - Testing standards
- [code-review.md](code-review.md) - Review process
- [deployment.md](deployment.md) - Deployment checklist
- [../System/automation.md](../System/automation.md) - Automation setup
- [../System/principles.md](../System/principles.md) - Core principles

---

**Remember**: Quality gates exist to protect users, the team, and the product. They're not obstacles—they're safety nets.
