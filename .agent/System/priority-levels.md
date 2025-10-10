# Priority Levels

## Overview

Consistent priority levels ensure the team focuses on the right work at the right time.

## Priority Definitions

### [BLOCKER]

**Definition**: Work stops until resolved
**Response Time**: Immediate (< 15 minutes)
**Examples**:

- Production system down
- Data loss occurring
- Active security breach
- All users unable to access system

**Action**: Drop everything, all hands on deck

### [CRITICAL]

**Definition**: Must complete before moving forward
**Response Time**: < 1 hour
**Examples**:

- Build is broken
- Tests failing on main branch
- Blocking dependency for sprint
- Major feature completely non-functional

**Action**: Top priority, resolve today

### [REQUIRED]

**Definition**: Necessary for production readiness
**Response Time**: < 1 day
**Examples**:

- Core feature implementation
- Essential bug fixes
- Security vulnerabilities
- Required documentation

**Action**: Schedule in current sprint/iteration

### [RECOMMENDED]

**Definition**: Should do for quality/maintainability
**Response Time**: < 1 week
**Examples**:

- Code refactoring
- Performance improvements
- Technical debt reduction
- Nice-to-have features

**Action**: Include if capacity permits

### [OPTIONAL]

**Definition**: Nice to have enhancements
**Response Time**: Backlog
**Examples**:

- UI polish
- Minor convenience features
- Experimental features
- Future improvements

**Action**: Consider for future sprints

## Usage Guidelines

### Assigning Priorities

```yaml
questions_to_ask:
  - Does it block other work? → [BLOCKER/CRITICAL]
  - Is it required for launch? → [REQUIRED]
  - Does it improve quality? → [RECOMMENDED]
  - Is it just nice to have? → [OPTIONAL]
```

### Priority Escalation

Only escalate if impact increases:

- User reports increase
- Affects more users
- Blocks more work
- Risk increases

### Priority De-escalation

Consider lowering if:

- Workaround found
- Impact less than thought
- Can defer safely

## Related Documentation

- [../SOP/task-management.md](../SOP/task-management.md) - Task management
- [principles.md](principles.md) - Core principles
