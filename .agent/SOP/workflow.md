# Development Workflow

## Overview

This document defines the standard development workflow and the Think-Tool Loop pattern for consistent, high-quality software delivery.

## The Think-Tool Loop

```text
┌─────────┐           ┌─────────┐           ┌─────────┐
│ Do Task │ ──────┐   │  Think  │   ┌────── │  File   │
└─────────┘       │   └─────────┘   │       └─────────┘
                  │        ▲ ▼       │
                  │   ┌─────────┐   │       ┌─────────┐
                  └──▶│Tool Call│◀──┘────── │  Text   │
                      └─────────┘            └─────────┘
                                             ┌─────────┐
                                             │ Command │
                                             └─────────┘
```

## Development Cycle

```text
┌─────────┐     ┌─────────────┐     ┌─────────┐     ┌──────────┐
│ Get task│────▶│Add to task  │────▶│ Do Task │────▶│Reflect on│
└─────────┘     │    list     │     └─────────┘     │  output  │
                └─────────────┘           ▲          └─────┬────┘
                       ▲                  │                │
                       └──────────────────┴────────────────┘
                                                           │
                                                           ▼
                                                      ┌────────┐
                                                      │ Output │
                                                      └────────┘
```

## Workflow Steps

### 1. Get Task

**Purpose**: Understand what needs to be done

**Actions**:

- Read task description thoroughly
- Identify stakeholders and users
- Understand success criteria
- Note any constraints or dependencies

**Questions to Ask**:

- What problem are we solving?
- Who are the users?
- What does success look like?
- What are the constraints?

**Output**: Clear understanding of requirements

---

### 2. Add to Task List

**Purpose**: Break down work into manageable units

**Actions**:

- Use [../Tasks/task-template.md](../Tasks/task-template.md)
- Break complex tasks into subtasks
- Assign priority levels ([BLOCKER]/[CRITICAL]/[REQUIRED]/[RECOMMENDED]/[OPTIONAL])
- Estimate effort (max 8 hours per task)

**Best Practices**:

- Each task should be completable in one session
- Tasks should be testable independently
- Dependencies should be explicit
- Use action verbs (Implement, Fix, Refactor, Test)

**Output**: Structured task list with clear acceptance criteria

---

### 3. Do Task

**Purpose**: Execute the work with continuous feedback

**The Think-Tool Cycle**:

1. **Think**: Plan the next small step
2. **Tool Call**: Execute using appropriate tool (File/Text/Command)
3. **Evaluate**: Check result and adjust
4. **Repeat**: Continue until task complete

**Key Principles**:

- Make the smallest change possible
- Test early and often
- Commit frequently
- Ask for help when stuck (> 30 minutes)

**Tools Reference**:

- **File operations**: Read, edit, create code
- **Text operations**: Document, communicate
- **Command operations**: Build, test, deploy

**Output**: Working, tested code

---

### 4. Reflect on Output

**Purpose**: Verify quality and completeness

**Verification Checklist**:

- [ ] Acceptance criteria met?
- [ ] Tests written and passing?
- [ ] Code follows standards?
- [ ] Documentation updated?
- [ ] No breaking changes?
- [ ] Performance acceptable?

**Quality Gates**: See [quality-gates.md](quality-gates.md)

**Output**: Validated, production-ready work

---

### 5. Output/Iterate

**Purpose**: Deliver value or improve based on feedback

**Decision Point**:

- ✅ **Ready**: Proceed to commit and deploy
- ⚠️ **Issues Found**: Return to step 3 (Do Task)
- 🔄 **Requirements Changed**: Return to step 1 (Get Task)

**Actions if Ready**:

- Commit with conventional format (see [commits.md](commits.md))
- Update documentation
- Mark task as complete
- Notify stakeholders

**Output**: Delivered value + updated documentation

---

## Phase-Based Workflow

### Phase 1: Discover [REQUIRED]

#### Explore Codebase

```bash
# Search patterns
$SEARCH_TOOL "pattern_to_find"

# Review history
$VCS_TOOL log --oneline --graph

# Check dependencies
$PACKAGE_MANAGER list

# Understand architecture
# Read architectural decision records
```

#### Context Gathering

- Read `/specs/prd.md` for product context
- Read `/specs/tech.md` for technical context
- Review related code and tests
- Check for existing patterns

---

### Phase 2: Design [CRITICAL]

#### Planning Mode

- Use [../Tasks/planning-template.md](../Tasks/planning-template.md)
- Create comprehensive specs before execution
- Validate approach with examples
- Get stakeholder approval

#### Key Decisions

- Architecture approach
- Technology choices
- Performance requirements
- Security considerations

**Document**: Use [decision-making.md](decision-making.md) for ADRs

---

### Phase 3: Build [CRITICAL]

#### Implementation Standards

- **[CRITICAL]** Test-driven development when possible
- **[REQUIRED]** Error handling on all external calls
- **[REQUIRED]** Input validation on all user data
- **[RECOMMENDED]** Logging for debugging

#### Testing Pyramid

- **Unit Tests (70%)**: < 100ms execution
- **Integration Tests (20%)**: < 5 seconds
- **E2E Tests (10%)**: < 30 seconds

See [testing.md](testing.md) for details

---

### Phase 4: Deploy [REQUIRED]

#### Pre-Deploy Checklist

- [ ] All tests passing
- [ ] Security scan completed
- [ ] Performance benchmarks met
- [ ] Documentation updated
- [ ] Rollback plan ready

See [deployment.md](deployment.md) for details

---

## Orchestration Patterns

### Sequential Pattern

Use when tasks depend on each other:

```text
Research → Design → Implement → Test → Deploy
```

### Parallel Pattern

Use when tasks are independent:

```text
┌─ Research ─┐
├─ Examples  ├─→ Synthesis → Implementation
└─ Standards ─┘
```

### Iterative Pattern

Use when feedback drives improvement:

```text
Plan → Execute → Validate ─┐
  ↑                        │
  └────── Refine ←─────────┘
```

---

## Time-Boxing Guidelines

| Activity | Max Time | Action if Exceeded |
|----------|----------|-------------------|
| Understanding task | 30 min | Ask clarifying questions |
| Planning approach | 1 hour | Get peer review |
| Bug investigation | 30 min | Escalate to team |
| Implementation | 4 hours | Break into smaller tasks |
| Debugging | 1 hour | Pair program |

---

## Best Practices

### DO ✅

- Understand before coding
- Break work into small tasks
- Test continuously
- Commit frequently
- Document decisions
- Ask when unclear

### DON'T ❌

- Start coding without understanding
- Create large, complex changes
- Skip tests "for now"
- Leave uncommitted work
- Make assumptions
- Struggle alone for hours

---

## Workflow Automation

### Git Hooks

```yaml
pre-commit:
  - lint
  - format
  - security-scan

pre-push:
  - test
  - build

post-merge:
  - dependency-update
  - notify-team
```

See [../System/automation.md](../System/automation.md) for setup

---

## Related Documentation

- [task-management.md](task-management.md) - Task lifecycle details
- [quality-gates.md](quality-gates.md) - Quality checkpoints
- [testing.md](testing.md) - Testing practices
- [commits.md](commits.md) - Commit conventions
- [deployment.md](deployment.md) - Deployment process
- [../System/orchestration.md](../System/orchestration.md) - Agent patterns

---

**Remember**: The goal is continuous delivery of value through small, tested, incremental changes.
