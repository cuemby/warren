# Decision Making Framework

## Overview

This document provides frameworks for making technical decisions quickly and effectively, with appropriate documentation for future reference.

## Time-Boxing Rules

Prevent analysis paralysis by setting time limits on decisions:

| Decision Type | Max Time | Action if Exceeded |
|--------------|----------|-------------------|
| Tool Selection | 2 hours | Use most popular/standard |
| Bug Investigation | 30 minutes | Escalate to team/pair program |
| Architecture Choice | 4 hours | Prototype both options |
| Performance Optimization | 1 hour | Profile first, then optimize |
| Design Pattern | 1 hour | Use simplest solution |
| Naming | 15 minutes | Use descriptive, refactor later |
| Refactoring Approach | 2 hours | Break into smaller steps |
| Third-party Integration | 3 hours | Try official SDK first |

---

## Decision Framework

### 1. Define the Problem

- What are we trying to solve?
- Why is this decision important?
- What constraints exist?
- Who is affected?

### 2. Identify Options

- List 2-5 viable options
- Include "do nothing" as an option
- Research quickly, don't overanalyze

### 3. Evaluate Trade-offs

```yaml
Option A:
  pros:
    - Fast to implement
    - Well documented
  cons:
    - Limited flexibility
    - Higher cost
  risk: Low

Option B:
  pros:
    - Highly flexible
    - Open source
  cons:
    - Steeper learning curve
    - Less mature
  risk: Medium
```

### 4. Make Decision

- Use weighted scoring if needed
- Consider reversibility
- Favor simple over complex
- Choose based on data + intuition

### 5. Document Decision

- Use ADR (Architecture Decision Record)
- Capture context and reasoning
- Note alternatives considered
- Record expected consequences

---

## Architecture Decision Records (ADR)

### When to Create ADR

- Architectural changes
- Technology choices
- Design pattern decisions
- Process changes
- Significant refactoring

### ADR Template

```markdown
# ADR-### [Title]

## Status
[Proposed | Accepted | Deprecated | Superseded by ADR-XXX]

## Context
What is the issue we're facing in our current situation?
What factors are at play? What are the constraints?

## Decision
What decision have we made? Be specific.

## Consequences
What becomes easier or more difficult as a result?

### Positive
- Benefit 1
- Benefit 2

### Negative
- Trade-off 1
- Trade-off 2

### Neutral
- Side effect 1

## Alternatives Considered
What other options did we look at?

### Option 1: [Name]
- Pros: ...
- Cons: ...
- Why rejected: ...

### Option 2: [Name]
- Pros: ...
- Cons: ...
- Why rejected: ...

## Implementation Notes
Any details about how to implement this decision.

## Related Decisions
- ADR-XXX: Related decision
- Supersedes: ADR-YYY

## References
- [Link to discussion](url)
- [Relevant documentation](url)

---
Date: YYYY-MM-DD
Author: @username
Reviewers: @reviewer1, @reviewer2
```

### ADR Example

```markdown
# ADR-003: Use PostgreSQL for Primary Database

## Status
Accepted

## Context
We need to choose a database for our user management system.
Requirements:
- ACID compliance
- Strong consistency
- Relational data model
- JSON support for flexible schemas
- Good performance up to 10M users

## Decision
We will use PostgreSQL 15+ as our primary database.

## Consequences

### Positive
- Battle-tested with 25+ years of development
- Excellent ACID compliance and data integrity
- Native JSON support for flexible fields
- Rich ecosystem of tools and extensions
- Strong community support

### Negative
- More complex to scale horizontally than NoSQL
- Requires more careful query optimization
- Higher operational complexity than managed NoSQL

### Neutral
- Team needs to learn PostgreSQL-specific features
- Need to set up replication and backups

## Alternatives Considered

### Option 1: MongoDB
- Pros: Simpler scaling, flexible schema
- Cons: Eventual consistency, less mature transactions
- Why rejected: ACID compliance is critical for user data

### Option 2: MySQL
- Pros: Similar to PostgreSQL, wide adoption
- Cons: Weaker JSON support, less feature-rich
- Why rejected: PostgreSQL has better JSON capabilities

## Implementation Notes
- Use PostgreSQL 15+ for JSONB improvements
- Set up read replicas for scaling reads
- Use connection pooling (PgBouncer)
- Implement automated backups

## Related Decisions
- ADR-002: Use row-level security
- Supersedes: ADR-001 (initial database choice)

## References
- [PostgreSQL Documentation](https://postgresql.org/docs)
- [Team Discussion](link-to-discussion)

---
Date: 2025-01-15
Author: @eng-lead
Reviewers: @backend-team
```

---

## Decision-Making Principles

### 1. Reversible vs Irreversible

**Reversible** (low-risk): Make quickly, can change later

- Code structure
- Variable names
- Library choices (within ecosystem)

**Irreversible** (high-risk): Take time, hard to change

- Database choice
- Cloud provider
- Core architecture
- Data schemas

### 2. YAGNI (You Aren't Gonna Need It)

- Don't build for hypothetical future needs
- Solve today's problems today
- Add complexity only when required

### 3. Simplicity Wins

```text
Simple > Complex
Boring > Exciting
Proven > Novel
Standard > Custom
```

### 4. Data-Driven

- Measure before optimizing
- A/B test when possible
- Use metrics to validate decisions
- Question assumptions

---

## Common Decision Patterns

### Technology Selection

```yaml
evaluation_criteria:
  must_have:
    - Meets functional requirements
    - Active maintenance/community
    - Compatible with existing stack
    - Team can learn in reasonable time

  nice_to_have:
    - Strong documentation
    - Good performance characteristics
    - Existing team expertise
    - Cost-effective

scoring:
  weight_by_importance:
    must_have: 10x
    nice_to_have: 1x

decision:
  if total_score > threshold:
    proceed: true
  else:
    consider_alternatives: true
```

### Build vs Buy

```text
Consider Building If:
- Core business differentiator
- Existing solutions don't fit
- Long-term cost savings clear
- Team has expertise

Consider Buying If:
- Not core business value
- Good solutions exist
- Time to market critical
- Maintenance burden high
```

### Refactor vs Rewrite

```text
Refactor If:
- Core logic is sound
- Can improve incrementally
- Low risk of regression
- Team understands code

Rewrite If:
- Fundamental design issues
- Technology is obsolete
- Can't test existing code
- Cost of maintenance > rewrite
```

---

## Decision Velocity

### Fast Decisions (< 1 hour)

- Minor code structure changes
- Variable/function naming
- Small refactoring
- Local optimizations

### Medium Decisions (1-8 hours)

- Library selection
- API design
- Database schema changes
- Architecture patterns

### Slow Decisions (> 1 day)

- Technology stack changes
- Major architectural decisions
- Cloud provider selection
- Framework choices

---

## Escalation Guidelines

### When to Escalate

- Decision impacts multiple teams
- Significant cost implications (> $X)
- Security or compliance concerns
- Architectural implications
- No clear answer after time-box

### Escalation Path

```text
Developer → Tech Lead → Engineering Manager → CTO
```

### Escalation Template

```markdown
## Decision Needed

**Problem**: Brief description

**Options Considered**:
1. Option A (pros/cons)
2. Option B (pros/cons)

**My Recommendation**: Option X because...

**Timeline**: Decision needed by [date]

**Impact if delayed**: [consequences]

**Questions for Leadership**:
- Question 1
- Question 2
```

---

## Group Decision Making

### Consensus Building

1. Present problem and options
2. Each person shares perspective
3. Discuss trade-offs
4. Vote or reach consensus
5. Document decision

### When Consensus Fails

- Tech lead makes final call
- Document dissenting opinions
- Set review date to revisit

### Decision-Making Meetings

**Timeboxed Agenda**:

- 5 min: Context and constraints
- 10 min: Options and trade-offs
- 10 min: Discussion
- 5 min: Decision and next steps

---

## Post-Decision

### Implementation

- Create tasks
- Assign owners
- Set timeline
- Define success metrics

### Review Schedule

- Small decisions: Review in 1 month
- Medium decisions: Review in 3 months
- Large decisions: Review in 6 months

### Learning

- Did the decision work out?
- What would we do differently?
- Update ADR with learnings

---

## Decision Anti-Patterns

### ❌ DON'T

- Analysis paralysis
- Decision by committee
- Ignoring data
- Following hype
- Not documenting
- Reopening settled decisions
- Making decisions in isolation

### ✅ DO

- Time-box decisions
- Use clear criteria
- Gather relevant data
- Document reasoning
- Consider reversibility
- Include stakeholders
- Learn from outcomes

---

## Related Documentation

- [../System/principles.md](../System/principles.md) - Core principles
- [../System/anti-patterns.md](../System/anti-patterns.md) - What to avoid
- [workflow.md](workflow.md) - Development workflow

---

## ADR Storage

Store ADRs in:

```text
project-root/
└── docs/
    └── decisions/
        ├── README.md          # Index of all ADRs
        ├── ADR-001-title.md
        ├── ADR-002-title.md
        └── ADR-003-title.md
```

Or in:

```text
project-root/
└── .agent/
    └── System/
        └── decisions/
```

---

**Remember**: Perfect is the enemy of good. Make the best decision you can with available information, document it, and move forward. You can always adjust later.
