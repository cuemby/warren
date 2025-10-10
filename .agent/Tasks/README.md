# Tasks Documentation Index

## Overview
This directory contains templates and guides for planning, documenting, and executing work in a context-driven, flexible manner aligned with "vibe coding" philosophy.

**Key Principle**: We estimate **context complexity**, not time. Work is organized by **understanding depth** and **priority**, not artificial deadlines.

---

## Quick Reference

### 🎯 Planning New Work
1. Start with [feature-brief-template.md](feature-brief-template.md) for quick specs
2. Use [prd-template.md](prd-template.md) for comprehensive product planning
3. Create [tech-spec-template.md](tech-spec-template.md) for implementation details
4. Reference [user-stories-guide.md](user-stories-guide.md) for writing stories

### 📋 Managing Tasks
1. Use [task-template.md](task-template.md) for individual task definition
2. Follow [planning-template.md](planning-template.md) for complex work
3. Apply [acceptance-criteria.md](acceptance-criteria.md) for validation
4. Conduct [retrospective-template.md](retrospective-template.md) after completion

### 🗺️ Strategic Planning
1. Use [roadmap-template.md](roadmap-template.md) for context-layered planning
2. Reference [requirements-template.md](requirements-template.md) for gathering needs

---

## Template Categories

### Product Planning Templates

#### [prd-template.md](prd-template.md)
**Purpose**: Comprehensive product requirements document
**When to Use**:
- New major features
- New products
- Significant functionality changes
- Need stakeholder alignment

**Context Complexity**: L-XL (requires substantial context)
**Key Sections**:
- Problem statement & opportunity
- User personas & stories
- Feature requirements (Must/Should/Could/Won't)
- Success metrics & acceptance criteria
- Context complexity estimates
- Iteration phases (not timelines!)

---

#### [feature-brief-template.md](feature-brief-template.md)
**Purpose**: Quick one-page feature specification
**When to Use**:
- Small to medium features
- Quick alignment needed
- Iterative development
- Proof of concepts

**Context Complexity**: XS-M
**Key Sections**:
- Problem → Solution → Impact
- Context requirements
- Dependencies
- Acceptance criteria

---

#### [user-stories-guide.md](user-stories-guide.md)
**Purpose**: Guide for writing effective user stories
**When to Use**:
- Breaking down features
- Sprint planning
- Backlog refinement

**Content**:
- User story format (As a/I want/So that)
- Acceptance criteria patterns
- Context estimation guide
- Priority assignment rules
- Examples (good & bad)

---

### Technical Planning Templates

#### [tech-spec-template.md](tech-spec-template.md)
**Purpose**: Detailed technical implementation specification
**When to Use**:
- Complex technical work
- Architecture decisions needed
- Team needs implementation guidance
- Integration with multiple systems

**Context Complexity**: M-XL
**Key Sections**:
- Architecture design
- Data models & API design
- Implementation layers (not phases!)
- Testing strategy
- Security & performance requirements
- Complexity assessments

---

#### [requirements-template.md](requirements-template.md)
**Purpose**: Structured requirements gathering
**When to Use**:
- Starting new projects
- Unclear requirements
- Multiple stakeholders
- Complex domain

**Context Complexity**: M-L
**Key Sections**:
- User requirements
- System requirements
- Technical requirements
- Context mapping

---

### Execution Templates

#### [task-template.md](task-template.md)
**Purpose**: Individual task definition
**When to Use**: Every task!

**Context Complexity**: XS-L (if > L, break down)
**Key Fields**:
- Title (action-oriented)
- Priority ([BLOCKER] to [OPTIONAL])
- Context Complexity (XS/S/M/L/XL)
- Description
- Acceptance criteria
- Dependencies

---

#### [planning-template.md](planning-template.md)
**Purpose**: Planning mode protocol for complex tasks
**When to Use**:
- Task complexity > M
- Multiple file changes
- Cross-system integration
- Unclear approach

**Content**:
- Understanding (current → desired state)
- Approach (step by step)
- Risks & mitigations
- Success criteria

---

#### [acceptance-criteria.md](acceptance-criteria.md)
**Purpose**: Validation and completion criteria
**When to Use**: Defining "done"

**Templates for**:
- Functional criteria
- Non-functional criteria
- Performance benchmarks
- Security requirements

---

### Strategic Templates

#### [roadmap-template.md](roadmap-template.md)
**Purpose**: Context-layered strategic planning
**When to Use**:
- Quarterly planning
- Product strategy
- Feature prioritization
- Dependency mapping

**Key Concept**: **Context Layers**, not timelines!
- Layer 0: Foundation (core understanding)
- Layer 1: Core (essential features)
- Layer 2: Extended (enhancements)
- Layer 3: Future (explorations)

---

#### [retrospective-template.md](retrospective-template.md)
**Purpose**: Learning and continuous improvement
**When to Use**:
- After major milestones
- End of iterations
- After incidents
- Regular team reflection

**Key Sections**:
- What went well
- What could be better
- Action items with owners

---

## Vibe Coding Concepts

### Context Complexity Levels

Replace time estimates with complexity understanding:

| Level | Definition | Characteristics |
|-------|------------|----------------|
| **XS** | Trivial | Single concept, obvious approach |
| **S** | Simple | Few related concepts, clear path |
| **M** | Moderate | Multiple interconnected concepts |
| **L** | Large | Complex system with dependencies |
| **XL** | Extra Large | **MUST break down** into smaller contexts |

### Priority Levels

From [../System/priority-levels.md](../System/priority-levels.md):

| Level | Response | Example |
|-------|----------|---------|
| **[BLOCKER]** | < 15 min | Production down |
| **[CRITICAL]** | < 1 hour | Build broken |
| **[REQUIRED]** | < 1 day | Core features |
| **[RECOMMENDED]** | < 1 week | Quality improvements |
| **[OPTIONAL]** | Backlog | Nice-to-haves |

### Iteration Phases

Instead of time-based milestones, use readiness phases:

```text
Phase 0: Foundation
  → Core architecture decided
  → Key concepts understood
  → Dependencies identified

Phase 1: Bootstrap
  → Basic functionality working
  → Core workflows implemented
  → Integration points proven

Phase 2: Enhancement
  → Feature-complete state
  → Edge cases handled
  → Performance acceptable

Phase 3: Refinement
  → Polished and optimized
  → Documentation complete
  → Production-ready
```

---

## Workflow Guide

### Starting New Feature

```text
1. Quick spec → feature-brief-template.md
2. Get feedback
3. If approved:
   → Complex? Use prd-template.md
   → Technical? Add tech-spec-template.md
4. Break into tasks → task-template.md
5. Execute with planning-template.md
6. Validate with acceptance-criteria.md
7. Learn with retrospective-template.md
```

### Context Complexity Assessment

Ask these questions:

**Trivial (XS)**:
- [ ] Single file change?
- [ ] No new concepts?
- [ ] Obvious solution?

**Simple (S)**:
- [ ] 2-3 files involved?
- [ ] Familiar patterns?
- [ ] Clear dependencies?

**Moderate (M)**:
- [ ] Multiple components?
- [ ] Some new concepts?
- [ ] Few integration points?

**Large (L)**:
- [ ] System-wide changes?
- [ ] New architecture needed?
- [ ] Multiple dependencies?

**Extra Large (XL)**:
- [ ] If yes to > 3 above → **BREAK IT DOWN!**

---

## Priority + Complexity Matrix

Decision guide for work ordering:

```text
Priority ↓ / Complexity →  │  XS  │  S   │  M   │  L   │  XL
───────────────────────────┼──────┼──────┼──────┼──────┼─────
[BLOCKER]                  │ NOW  │ NOW  │ NOW  │BREAK │BREAK
[CRITICAL]                 │TODAY │TODAY │TODAY │PLAN  │BREAK
[REQUIRED]                 │TODAY │WEEK  │WEEK  │PLAN  │DEFER
[RECOMMENDED]              │WEEK  │WEEK  │PLAN  │DEFER │DEFER
[OPTIONAL]                 │BACK  │BACK  │BACK  │DEFER │DEFER

NOW   = Drop everything
TODAY = Within current session
WEEK  = When capacity available
PLAN  = Needs planning before starting
BREAK = Break into smaller pieces
BACK  = Add to backlog
DEFER = Defer or descope
```

---

## Template Selection Guide

### "I need to..."

**...quickly propose a feature**
→ [feature-brief-template.md](feature-brief-template.md)

**...plan a major product**
→ [prd-template.md](prd-template.md)

**...design technical solution**
→ [tech-spec-template.md](tech-spec-template.md)

**...write better user stories**
→ [user-stories-guide.md](user-stories-guide.md)

**...gather requirements**
→ [requirements-template.md](requirements-template.md)

**...plan work breakdown**
→ [task-template.md](task-template.md)

**...tackle complex task**
→ [planning-template.md](planning-template.md)

**...define "done"**
→ [acceptance-criteria.md](acceptance-criteria.md)

**...plan strategic direction**
→ [roadmap-template.md](roadmap-template.md)

**...learn from work completed**
→ [retrospective-template.md](retrospective-template.md)

---

## Anti-Patterns

### ❌ DON'T
- Use time estimates (hours/days/weeks)
- Create rigid deadlines for exploration
- Skip requirements for "quick" features
- Write XL tasks without breaking down
- Ignore context complexity
- Start coding without understanding

### ✅ DO
- Estimate context complexity
- Use priority levels consistently
- Write clear acceptance criteria
- Break large work into smaller contexts
- Validate understanding before coding
- Learn and adapt from retrospectives

---

## Related Documentation

### SOP (Standard Operating Procedures)
- [../SOP/workflow.md](../SOP/workflow.md) - Development workflow
- [../SOP/task-management.md](../SOP/task-management.md) - Task lifecycle
- [../SOP/decision-making.md](../SOP/decision-making.md) - Making decisions

### System Documentation
- [../System/principles.md](../System/principles.md) - Core principles
- [../System/priority-levels.md](../System/priority-levels.md) - Priority definitions

---

## Contributing to Templates

### Adding New Templates
1. Identify gap in current templates
2. Create template following existing patterns
3. Use context complexity (not time)
4. Include examples
5. Update this README
6. Cross-reference related docs

### Improving Existing Templates
1. Use templates in real work
2. Note pain points
3. Suggest improvements
4. Update based on learnings
5. Keep templates lean and practical

---

**Remember**: These templates serve you, not the other way around. Use what's helpful, adapt what's not, skip what's overkill. The goal is **clarity and shared understanding**, not bureaucracy.

*Last Updated: 2025-10-08*
