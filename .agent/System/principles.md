# Core Development Principles

## Overview

These principles guide all development work, regardless of technology or team size. Follow these to ensure consistent, high-quality output.

## The Three Pillars

### 1. Code Minimalism [CRITICAL]

**Principle**: Write the least amount of code necessary

**Guidelines**:

- Remove code before adding code
- Each function/method does one thing well
- Prefer composition over inheritance
- No premature optimization
- If you can't explain it simply, it's too complex

**Example**:

```text
❌ Complex: 100 lines with multiple responsibilities
✅ Simple: 20 lines, single responsibility
```

### 2. Quality Gates [CRITICAL]

**Principle**: Every change passes through quality checkpoints

**Gates**:

- Pre-commit: Local validation
- Pre-merge: Peer review + CI
- Pre-deploy: Staging validation

**Non-negotiable**:

- Tests must pass
- Code must be reviewed
- No secrets in code
- Performance baselines met

See [../SOP/quality-gates.md](../SOP/quality-gates.md)

### 3. Clarity First [CRITICAL]

**Principle**: Understanding beats speed

**Practice**:

- Ask clarification questions until you understand
- Confirm requirements before implementation
- Question ambiguous specifications immediately
- [BLOCKER]: No coding without clear specifications

**When Unclear**:

1. Ask stakeholder
2. Check documentation
3. Review similar code
4. Discuss with team
5. **Never**: Make assumptions

---

## Development Workflow Principles

### Understand → Plan → Execute → Verify

```text
1. GET TASK: Understand requirements fully
2. PLAN: Break down into manageable units
3. EXECUTE: Build with continuous feedback
4. VERIFY: Test, review, validate
5. DOCUMENT: Record decisions and changes
```

See [../SOP/workflow.md](../SOP/workflow.md)

### The Think-Tool Loop

- **Think**: Plan next small step
- **Tool**: Execute (code, test, commit)
- **Evaluate**: Check result
- **Repeat**: Until complete

---

## Code Quality Principles

### Readability

- Code is read 10x more than written
- Write for humans first, computers second
- Clear names > clever code
- Comments explain "why", not "what"

### Testability

- If it's hard to test, it's poorly designed
- Test-driven development when possible
- Test behavior, not implementation
- Coverage is a metric, not a goal

### Maintainability

- Future you will thank present you
- Every TODO has a ticket
- Delete unused code immediately
- Refactor continuously, not in sprints

### Performance

- Measure before optimizing
- Profile to find bottlenecks
- Simple fast > complex faster
- Performance is a feature

---

## Collaboration Principles

### Communication

- Over-communicate rather than under
- Asynchronous by default
- Document decisions publicly
- Status updates proactively

### Code Review

- Review promptly (< 24 hours)
- Focus on code, not person
- Explain "why" in feedback
- Block only on critical issues

### Knowledge Sharing

- Document as you learn
- Share discoveries widely
- Mentor actively
- Learn from failures

---

## Decision-Making Principles

### Time-Boxing

- Set time limits on decisions
- Good decision now > perfect decision later
- Document and move forward
- Can always adjust later

### Reversibility

- Favor reversible decisions
- Use feature flags
- Make changes incrementally
- Test rollback procedures

### Simplicity

- **Simple > Complex**
- **Boring > Exciting**
- **Proven > Novel**
- **Standard > Custom**

See [../SOP/decision-making.md](../SOP/decision-making.md)

---

## Technical Principles

### YAGNI (You Aren't Gonna Need It)

- Build for today's requirements
- Don't speculate future needs
- Add complexity only when required
- Remove unused features

### DRY (Don't Repeat Yourself)

- Extract common patterns
- But: Don't abstract too early
- Duplication is better than wrong abstraction
- Three strikes and you refactor

### SOLID (for OOP)

- **S**ingle Responsibility
- **O**pen/Closed
- **L**iskov Substitution
- **I**nterface Segregation
- **D**ependency Inversion

### Separation of Concerns

- Each module has one job
- Clear boundaries between layers
- Minimize coupling
- Maximize cohesion

---

## Security Principles [CRITICAL]

- **Never trust user input**: Validate and sanitize everything
- **Principle of least privilege**: Minimal permissions required
- **Defense in depth**: Multiple security layers
- **Fail securely**: Errors don't expose vulnerabilities
- **No secrets in code**: Use environment variables
- **Keep dependencies updated**: Regular security scans

See [compliance.md](compliance.md)

---

## Testing Principles

### Testing Pyramid

- 70% Unit Tests: Fast, isolated
- 20% Integration Tests: Components together
- 10% E2E Tests: Full workflows

### Test Quality

- Tests are first-class code
- One assertion per test (usually)
- AAA pattern: Arrange-Act-Assert
- Independent tests (no shared state)
- Fast tests (< 100ms for unit)

See [../SOP/testing.md](../SOP/testing.md)

---

## Deployment Principles

### Continuous Delivery

- Main branch is always deployable
- Small, frequent releases
- Automate everything
- Monitor intensively post-deploy

### Zero Downtime

- Backward-compatible changes
- Feature flags for new features
- Rollback plan always ready
- Database migrations separate from code

### Observability

- Log everything important
- Monitor key metrics
- Alert on user impact
- Track business metrics

See [../SOP/deployment.md](../SOP/deployment.md)

---

## Documentation Principles

### Living Documentation

- Code is primary documentation
- Update docs with code changes
- Examples over explanations
- Keep it close to code

### What to Document

- **Why**: Architecture decisions (ADRs)
- **How to**: Setup, deployment procedures
- **What**: API documentation, interfaces
- **When**: Emergency procedures

### What NOT to Document

- Implementation details (use code)
- Obvious information
- Outdated information (delete it)
- Things that change frequently

---

## Priority Framework

Use these consistently:

| Level | Meaning | Response Time |
|-------|---------|---------------|
| [BLOCKER] | Stop everything | < 15 minutes |
| [CRITICAL] | Must complete first | < 1 hour |
| [REQUIRED] | Necessary for production | < 1 day |
| [RECOMMENDED] | Should do for quality | < 1 week |
| [OPTIONAL] | Nice to have | Backlog |

See [priority-levels.md](priority-levels.md)

---

## Continuous Improvement

### Retrospectives

- Regular reflection on what works
- Action items with owners
- Track improvements
- Celebrate wins

### Metrics

- Track to improve, not to judge
- Velocity, quality, cycle time
- Bug escape rate
- Team happiness

### Learning

- Share knowledge actively
- Post-mortems for incidents
- Examples for patterns
- Mentorship for growth

---

## Anti-Patterns to Avoid

### ❌ Never Do This

1. Commit directly to main
2. Skip tests "for now"
3. Leave TODO without ticket
4. Deploy Friday afternoon
5. Ignore failing tests
6. Store secrets in code
7. Make assumptions
8. Refactor without tests

See [anti-patterns.md](anti-patterns.md)

---

## Related Documentation

- [../SOP/workflow.md](../SOP/workflow.md) - Development workflow
- [../SOP/quality-gates.md](../SOP/quality-gates.md) - Quality checkpoints
- [priority-levels.md](priority-levels.md) - Priority definitions
- [anti-patterns.md](anti-patterns.md) - What to avoid

---

**Remember**: Principles are guidelines, not rules. Use judgment, but favor adherence. When breaking a principle, document why.
