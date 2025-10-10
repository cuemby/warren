# Task Management

## Overview

Effective task management ensures work is tracked, prioritized, and completed systematically. This document defines task lifecycle, tracking, and management practices.

## Task Lifecycle

```text
┌──────────┐
│ Created  │
└────┬─────┘
     │
┌────▼──────┐
│ Planned   │
└────┬──────┘
     │
┌────▼──────────┐
│ In Progress   │
└────┬──────────┘
     │
┌────▼─────┐
│ Review   │
└────┬─────┘
     │
┌────▼───────┐
│ Completed  │
└────────────┘
```

---

## Task Structure

### Required Fields

- **ID**: Unique identifier (TASK-001, #123, etc.)
- **Title**: Brief, action-oriented description
- **Priority**: [BLOCKER] / [CRITICAL] / [REQUIRED] / [RECOMMENDED] / [OPTIONAL]
- **Status**: pending / in_progress / completed
- **Description**: Detailed explanation

### Optional Fields

- **Assignee**: Who's responsible
- **Estimate**: Time estimate (max 8 hours)
- **Due Date**: When it's needed
- **Dependencies**: What must be done first
- **Tags**: Categories (bug, feature, refactor)

### Template

See [../Tasks/task-template.md](../Tasks/task-template.md)

---

## Priority Levels

### [BLOCKER]

- **Definition**: Stops all work
- **Response Time**: Immediate (< 15 min)
- **Examples**: Production down, data loss, security breach
- **Action**: Drop everything, fix now

### [CRITICAL]

- **Definition**: Must complete before moving forward
- **Response Time**: < 1 hour
- **Examples**: Test failures, blocking dependencies
- **Action**: Top priority, complete today

### [REQUIRED]

- **Definition**: Necessary for production readiness
- **Response Time**: < 1 day
- **Examples**: Core features, essential fixes
- **Action**: Schedule in current sprint

### [RECOMMENDED]

- **Definition**: Should do for quality
- **Response Time**: < 1 week
- **Examples**: Refactoring, improvements
- **Action**: Include if time permits

### [OPTIONAL]

- **Definition**: Nice to have enhancements
- **Response Time**: Backlog
- **Examples**: UI polish, minor features
- **Action**: Consider for future sprints

See [../System/priority-levels.md](../System/priority-levels.md)

---

## Task Breakdown

### Good Task Characteristics

- ✅ Completable in one session (< 8 hours)
- ✅ Single, clear objective
- ✅ Testable outcome
- ✅ Independent or clear dependencies
- ✅ Action-oriented title

### Breaking Down Large Tasks

**Before** (too large):

```text
TASK-001: Implement user authentication
Estimate: 40 hours
```

**After** (well-broken-down):

```text
TASK-001: Create user database schema
  Estimate: 2 hours

TASK-002: Implement password hashing
  Estimate: 3 hours

TASK-003: Create login endpoint
  Estimate: 4 hours

TASK-004: Add JWT token generation
  Estimate: 3 hours

TASK-005: Implement session management
  Estimate: 4 hours

TASK-006: Add authentication middleware
  Estimate: 3 hours

TASK-007: Write authentication tests
  Estimate: 5 hours
```

---

## Task Tracking

### Where to Track

- Issue trackers (GitHub Issues, Jira, Linear)
- Project boards (Kanban, Scrum)
- Text files (tasks/todo.md)
- Task management tools

### Board Structure

```text
┌──────────┐  ┌─────────────┐  ┌────────┐  ┌───────────┐
│ Backlog  │  │ In Progress │  │ Review │  │ Completed │
├──────────┤  ├─────────────┤  ├────────┤  ├───────────┤
│ TASK-010 │  │ TASK-001 👤 │  │ TASK-  │  │ TASK-005  │
│ TASK-011 │  │ TASK-002 👤 │  │ 003 🔍 │  │ TASK-006  │
│ TASK-012 │  └─────────────┘  │ TASK-  │  │ TASK-007  │
│   ...    │                    │ 004 🔍 │  └───────────┘
└──────────┘                    └────────┘
```

### Status Definitions

- **Backlog**: Not yet started
- **In Progress**: Actively being worked on
- **Review**: Awaiting code review
- **Completed**: Done and merged

---

## Task Workflow

### 1. Create Task

```markdown
**Task**: Add user email validation

**Priority**: [REQUIRED]

**Description**:
Validate email format on user registration and update endpoints.

**Acceptance Criteria**:
- [ ] Reject invalid email formats
- [ ] Show clear error messages
- [ ] Handle edge cases (unicode, special chars)
- [ ] Add unit tests

**Estimate**: 2 hours
```

### 2. Plan Task

- Break into subtasks if needed
- Identify dependencies
- Estimate effort
- Assign to sprint/milestone

### 3. Start Task

```bash
# Create feature branch
$VCS_TOOL checkout -b feature/TASK-001-email-validation

# Move task to "In Progress"
# Add your name as assignee
```

### 4. Work on Task

- Follow [workflow.md](workflow.md)
- Commit frequently
- Update task with progress
- Ask for help if stuck > 30 min

### 5. Complete Task

```bash
# Run tests
$TEST_RUNNER run

# Commit changes
$VCS_TOOL commit -m "feat: add email validation"

# Push branch
$VCS_TOOL push origin feature/TASK-001-email-validation

# Create PR
$VCS_PLATFORM pr create --title "feat: add email validation"

# Move task to "Review"
```

### 6. Close Task

- After PR merged
- Update documentation if needed
- Move to "Completed"
- Celebrate! 🎉

---

## Task Estimation

### T-Shirt Sizing

- **XS**: < 1 hour
- **S**: 1-2 hours
- **M**: 2-4 hours
- **L**: 4-8 hours
- **XL**: > 8 hours (break it down!)

### Story Points (Optional)

- **1 point**: Trivial change
- **2 points**: Simple task
- **3 points**: Moderate complexity
- **5 points**: Complex task
- **8 points**: Very complex (consider breaking down)
- **13+ points**: Too large, must break down

### Estimation Guidelines

- Include time for:
  - Writing tests
  - Code review
  - Documentation
  - Debugging
- Multiply initial estimate by 1.5x (padding for unknowns)
- Re-estimate if actual time > 2x estimate

---

## Daily Task Management

### Morning

1. Review task list
2. Identify top 1-3 priorities
3. Check for blockers
4. Start with highest priority

### Throughout Day

1. Update task status as you progress
2. Mark tasks complete when done
3. Add new tasks as they arise
4. Re-prioritize if needed

### End of Day

1. Commit all work
2. Update task statuses
3. Note any blockers
4. Plan tomorrow's priorities

---

## Sprint/Iteration Planning

### Planning Meeting

1. Review backlog
2. Prioritize tasks
3. Assign to sprint
4. Estimate capacity
5. Commit to scope

### Sprint Structure (Example: 2 weeks)

- **Day 1**: Planning
- **Days 2-9**: Development
- **Day 10**: Final testing, docs
- **Day 10 (end)**: Sprint review & retrospective

### Capacity Planning

```text
Team Member Hours per Day:
- 8 hours total
- 2 hours (meetings, breaks, interruptions)
= 6 hours productive time

Sprint Capacity:
- 6 hours/day × 10 days = 60 hours per person
- 3 team members = 180 hours total
- 20% buffer = 144 hours planned capacity
```

---

## Task Dependencies

### Handling Dependencies

```text
TASK-001: Create database schema
  ↓
TASK-002: Implement API endpoints (depends on TASK-001)
  ↓
TASK-003: Add frontend integration (depends on TASK-002)
```

### Parallel Work

```text
TASK-001: Backend API
TASK-002: Frontend UI  } Can work in parallel
TASK-003: Documentation
```

---

## Task Anti-Patterns

### ❌ DON'T

- Create vague tasks ("Fix stuff")
- Leave tasks in limbo
- Have multiple in-progress tasks
- Skip updating status
- Create tasks > 8 hours
- Work on wrong priority
- Ignore dependencies

### ✅ DO

- Write clear, actionable tasks
- Keep status current
- Focus on one task at a time
- Update frequently
- Break down large tasks
- Work by priority
- Track dependencies

---

## Task Communication

### Updating Stakeholders

```markdown
## Task Update: TASK-001

**Status**: In Progress (75% complete)

**Completed**:
- ✅ Database schema created
- ✅ Basic validation logic implemented

**In Progress**:
- 🔄 Adding edge case handling
- 🔄 Writing unit tests

**Blockers**:
- None currently

**ETA**: End of day
```

### Asking for Help

```markdown
## Help Needed: TASK-002

**Problem**: Not sure which validation library to use

**What I've Tried**:
- Researched 3 options (validator.js, joi, yup)
- Read documentation

**Questions**:
- Do we have a preferred validation library?
- Are there security considerations?

**Urgency**: Moderate (blocking progress)
```

---

## Metrics & Tracking

### Team Metrics

- **Velocity**: Tasks completed per sprint
- **Cycle Time**: Time from start to complete
- **Lead Time**: Time from created to complete
- **Throughput**: Tasks completed per week

### Individual Metrics

- Tasks completed
- Estimate accuracy
- Time to first review
- Rework rate

### Use Metrics For

- ✅ Improving processes
- ✅ Capacity planning
- ✅ Identifying bottlenecks

### Don't Use Metrics For

- ❌ Individual performance reviews
- ❌ Comparing team members
- ❌ Punishment

---

## Tools & Integration

### Issue Trackers

```bash
# GitHub
gh issue list
gh issue create --title "Title" --body "Description"
gh issue view 123

# Command line tools
$TASK_MANAGER list
$TASK_MANAGER create "Task title"
$TASK_MANAGER update 123 --status in_progress
```

### Automation

```yaml
# Automatic task updates
on:
  pull_request:
    types: [opened]
  action: link_issue

on:
  pull_request:
    types: [closed]
  action: close_issue
```

---

## Related Documentation

- [../Tasks/task-template.md](../Tasks/task-template.md) - Task template
- [../Tasks/planning-template.md](../Tasks/planning-template.md) - Planning template
- [workflow.md](workflow.md) - Development workflow
- [../System/priority-levels.md](../System/priority-levels.md) - Priority definitions

---

**Remember**: Tasks are tools for organization, not bureaucracy. Keep them lightweight, actionable, and up-to-date. The goal is to ship value, not to maintain task lists.
