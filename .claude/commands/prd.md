---
description: Create vibe-coding adapted Product Requirements Documents
tags: [prd, planning, requirements, product]
version: 1.0.0
---

# prd

## Context-Driven Product Requirements Document Creation

Before creating a PRD:

1. **Read `.agent/README.md`** to understand project context and standards
2. **Check `.agent/System/`** for current system state and architecture
3. **Review `.agent/SOP/`** for documentation best practices
4. **Reference `.agent/Tasks/prd-template.md`** for the complete template structure

## PRD Creation Process

Generate a comprehensive Product Requirements Document following the vibe-coding philosophy (context-driven, not time-driven) using this systematic workflow:

### 1. Context Discovery Phase

**Read Project Context**:
- Review `.agent/README.md` for project overview
- Check `.agent/System/principles.md` for core principles
- Review `.agent/System/priority-levels.md` for priority framework
- Scan existing PRDs in `docs/prd/` for patterns

**Gather Essential Information from User**:

**Feature/Product Basics**:
- Feature/Product Name
- Brief one-line description
- Is this: [New Feature | Enhancement | Refactor | Bug Fix | Other]
- Related to existing feature? [If yes, which one]

**Problem Understanding**:
- What problem are we solving?
- Who experiences this problem?
- How painful is it? [Low | Medium | High | Critical]
- What happens if we don't solve it?
- How is it currently handled (workarounds)?

**User Context**:
- Primary user type/persona
- Secondary user types (if any)
- Technical proficiency: [Low | Medium | High]
- Current behavior/workflow

**Scope & Priority**:
- Priority Level: [BLOCKER] | [CRITICAL] | [REQUIRED] | [RECOMMENDED] | [OPTIONAL]
- Context Complexity: XS | S | M | L | XL
  - XS: Trivial, single concept
  - S: Simple, few related concepts
  - M: Moderate, multiple interconnected concepts
  - L: Large, complex system with dependencies
  - XL: Extra large (needs breakdown)

---

### 2. Requirements Gathering Phase

**Must-Have Requirements** (MVP):
- Ask: "What are the absolute must-haves for this to work?"
- For each requirement:
  - Requirement description
  - Why is this essential?
  - Context complexity: [XS/S/M/L]
  - Dependencies

**Should-Have Requirements**:
- Ask: "What's important but not critical?"
- Capture with context complexity

**Could-Have Requirements**:
- Ask: "What would be nice to have?"
- Capture for future iterations

**Won't-Have (This Iteration)**:
- Ask: "What's explicitly out of scope?"
- Document to prevent scope creep

---

### 3. User Story Development

**For Each Epic/Story**:

Create user stories in this format:
```markdown
### Epic: [Epic Name]

#### Story: [Story Title]

**As a** [user type]
**I want** [capability]
**So that** [benefit/value]

**Acceptance Criteria**:
- [ ] **GIVEN** [context], **WHEN** [action], **THEN** [outcome]
- [ ] **GIVEN** [context], **WHEN** [action], **THEN** [outcome]
- [ ] **GIVEN** [context], **WHEN** [action], **THEN** [outcome]

**Priority**: P0 (Critical) | P1 (High) | P2 (Medium) | P3 (Low)
**Context Complexity**: XS | S | M | L | XL
**Dependencies**: [List]
```

**Ask user**:
- What are the main user workflows?
- What does success look like for the user?
- What edge cases should we handle?

---

### 4. Success Metrics Definition

**Ask user to define measurable success**:
- What metrics will we track?
- What's the current baseline (if known)?
- What's the target?
- How will we measure?

**Example metrics**:
- User engagement (usage frequency, time spent)
- Performance (response time, load time)
- Business (conversion rate, revenue impact)
- Quality (error rate, user satisfaction)

---

### 5. Context Complexity Assessment

**Overall Assessment**:
- Ask: "How much understanding is needed to build this?"
- Break down by area:
  - User Experience: [XS/S/M/L/XL]
  - Technical Implementation: [XS/S/M/L/XL]
  - Integration: [XS/S/M/L/XL]
  - Data Model: [XS/S/M/L/XL]
  - Testing: [XS/S/M/L/XL]

**Context Dependencies**:
- What existing systems need to be understood?
- What new concepts need to be learned?
- What external knowledge is required?

---

### 6. Iteration Phase Planning

**Instead of timeline milestones, define readiness phases**:

**Phase 0: Foundation**
- Readiness Criteria:
  - [ ] Core architecture decided
  - [ ] Key concepts understood
  - [ ] Dependencies identified
  - [ ] Team has necessary context
- Deliverables: ADRs, technical spikes, POCs

**Phase 1: Bootstrap**
- Readiness Criteria:
  - [ ] Basic functionality working
  - [ ] Core workflows implemented
  - [ ] Integration points proven
  - [ ] Happy path testable
- Deliverables: Working prototype, core features

**Phase 2: Enhancement**
- Readiness Criteria:
  - [ ] All Must-Have features complete
  - [ ] Edge cases handled
  - [ ] Performance acceptable
  - [ ] Should-Have features integrated
- Deliverables: Feature-complete state

**Phase 3: Refinement**
- Readiness Criteria:
  - [ ] Could-Have features evaluated
  - [ ] Polished UX
  - [ ] Production-ready
  - [ ] Team confident in deployment
- Deliverables: Production release

---

### 7. Risk Assessment

**Ask user about risks**:
- What could go wrong?
- What are we uncertain about?
- What external dependencies worry you?
- What's the learning curve?

**Document risks with**:
- Risk description
- Probability: [Low | Medium | High]
- Impact: [Low | Medium | High]
- Context Complexity involved
- Mitigation strategy

---

### 8. Vibe Check

**Ask user for gut feeling validation**:
- Does this feel right?
- Is the problem clearly defined?
- Does the solution make intuitive sense?
- Is the scope manageable?
- Does the team have necessary context?
- Are you excited to build this?

**Red flags to check**:
- Too much complexity for the value?
- Scope creeping during planning?
- Unclear user value?
- Team lacks critical context?

**Confidence Level**: 😰 Low | 😐 Medium | 😊 High | 🚀 Very High

---

### 9. Document Generation

**Generate PRD using `.agent/Tasks/prd-template.md` structure**:

1. Fill in all sections from gathered information
2. Use context complexity instead of time estimates
3. Use iteration phases instead of timeline milestones
4. Include vibe check section
5. Cross-reference `.agent/` documentation
6. Add all user stories with acceptance criteria
7. Include risk assessment
8. Document open questions

**File naming**: `docs/prd/[feature-name].md`
- Use kebab-case
- Make it descriptive
- Example: `user-authentication.md`, `payment-integration.md`

---

### 10. Review & Validation

**Present PRD to user for review**:
- Show generated PRD
- Highlight key sections
- Ask for feedback on:
  - Completeness
  - Accuracy
  - Clarity
  - Missing information
  - Scope correctness

**Iterate if needed**:
- Make adjustments based on feedback
- Add missing details
- Clarify ambiguities
- Adjust priorities or complexity

---

### 11. Cross-Referencing

**Update related documentation**:
- Add entry to `docs/README.md` (if exists)
- Reference in `.agent/Tasks/` if task-specific
- Note in `.agent/System/` if architecture impact
- Link to related PRDs or tech specs

**Create tracking**:
- Suggest creating tech spec: "Run `/tech-spec` to create technical specification"
- Suggest creating user stories: "Run `/user-stories` to extract detailed user stories"

---

## PRD Output Format

### Document Structure (following `.agent/Tasks/prd-template.md`)

```markdown
# Product Requirements Document (PRD)

**Document Version:** 1.0
**Last Updated:** [Today's Date]
**Status:** Draft
**Author:** [From context or prompt]
**Stakeholders:** [From context or prompt]
**Priority:** [User selected]
**Context Complexity:** [User/AI assessed]

---

## Executive Summary

[Generated from gathered information]

**Quick Facts**:
- **Problem**: [One-line]
- **Solution**: [One-line]
- **Impact**: [Expected outcome]
- **Context Complexity**: [XS/S/M/L/XL]

---

## Problem Statement

### The Problem
[Detailed problem description]

**Current Pain Points**:
- [Point 1]
- [Point 2]
- [Point 3]

**Who Experiences This**:
- [User group 1]: [Details]
- [User group 2]: [Details]

### Current Solution
[How problem is currently handled]

### Opportunity
[Value proposition]

---

## Goals and Objectives

### Business Goals
[From user input]

### User Goals
[From user input]

### Success Metrics
[Table with metrics, baseline, target, measurement]

---

## User Personas

[Primary and secondary personas from user input]

---

## User Stories

[All gathered user stories with acceptance criteria]

---

## Feature Requirements

### MoSCoW Prioritization

#### Must Have (MVP)
[All must-have requirements with context complexity]

#### Should Have
[Should-have requirements]

#### Could Have
[Nice-to-haves]

#### Won't Have (This Iteration)
[Out of scope]

### Non-Functional Requirements
[Performance, security, accessibility, compatibility]

---

## User Journey

[Primary scenarios and edge cases]

---

## Design & Interaction

[If provided by user, otherwise placeholder]

---

## Technical Considerations

[High-level technical notes]

**Suggestion**: Run `/tech-spec` to create detailed technical specification

---

## Iteration Phases

[Phase 0-3 with readiness criteria and deliverables]

---

## Context Complexity Breakdown

[Complexity assessment by area]

---

## Release Strategy

[Layer-based rollout strategy]

---

## Risks and Mitigations

[All identified risks with mitigations]

---

## Open Questions & Decisions Needed

[Unresolved questions with owners]

---

## Vibe Check ✨

[Gut feeling assessment and confidence level]

---

## Appendix

[Glossary, references, revision history]

---

## Approvals & Sign-off

[Approval section]

---

## Related Documentation

- **Next Steps**: Run `/tech-spec` to create technical specification
- **User Stories**: Run `/user-stories` to extract detailed user stories
- **Reference**: [`.agent/Tasks/prd-template.md`](.agent/Tasks/prd-template.md)
```

---

## Interactive Prompts

Use these prompts to gather information:

### Initial Prompt
```
I'll help you create a comprehensive PRD using our vibe-coding adapted template.

Let's start with the basics:

1. What's the name of this feature/product?
2. In one sentence, what does it do?
3. What priority level is this?
   [BLOCKER] | [CRITICAL] | [REQUIRED] | [RECOMMENDED] | [OPTIONAL]
```

### Problem Discovery Prompt
```
Let's understand the problem:

1. What problem are we solving?
2. Who experiences this problem most?
3. How painful is it? [Low | Medium | High | Critical]
4. What happens if we don't solve it?
5. How do users currently work around this?
```

### Requirements Prompt
```
Let's define the requirements:

MUST HAVE (MVP):
What are the absolute essentials? List them one by one.
[For each: Why is this essential? Context complexity?]

SHOULD HAVE:
What's important but not critical?

COULD HAVE:
What would be nice to have later?

WON'T HAVE:
What's explicitly out of scope?
```

### User Stories Prompt
```
Let's create user stories:

For the main workflow:
- As a [who], I want [what], so that [why]
- What are the acceptance criteria? (GIVEN/WHEN/THEN format)
- Priority? [P0/P1/P2/P3]
- Context complexity? [XS/S/M/L/XL]

Any other workflows or edge cases?
```

### Success Metrics Prompt
```
How will we measure success?

1. What metric(s) will we track?
2. What's the current state (baseline)?
3. What's the target?
4. How will we measure it?
```

### Context Complexity Prompt
```
Let's assess context complexity:

How much understanding is needed for:
- User Experience: [XS | S | M | L | XL]
- Technical Implementation: [XS | S | M | L | XL]
- Integration: [XS | S | M | L | XL]
- Data Model: [XS | S | M | L | XL]
- Testing: [XS | S | M | L | XL]

What existing systems need to be understood?
What new concepts need to be learned?
```

### Risk Assessment Prompt
```
What could go wrong?

For each risk:
- What's the risk?
- Probability: [Low | Medium | High]
- Impact: [Low | Medium | High]
- How do we mitigate it?
```

### Vibe Check Prompt
```
Final vibe check:

✓ Does this feel right?
✓ Is the problem clearly defined?
✓ Does the solution make sense?
✓ Is the scope manageable?
✓ Do we have the necessary context?
✓ Are you excited about this?

Any red flags?
- Too complex for the value?
- Scope creeping?
- Unclear value?
- Missing critical context?

Confidence level: 😰 Low | 😐 Medium | 😊 High | 🚀 Very High
```

---

## Best Practices

### DO ✅
- Use context complexity instead of time estimates
- Focus on understanding depth, not deadlines
- Ask clarifying questions
- Document assumptions
- Include vibe check
- Reference `.agent/` docs
- Create user stories with acceptance criteria
- Assess risks honestly
- Use iteration phases, not timelines
- Keep it practical and actionable

### DON'T ❌
- Use week/hour/day estimates
- Make up user needs
- Skip user stories
- Ignore edge cases
- Forget non-functional requirements
- Skip vibe check
- Make assumptions without documenting
- Over-scope the MVP
- Ignore risks
- Create PRD without user input

---

## Success Criteria

- [ ] PRD created using `.agent/Tasks/prd-template.md`
- [ ] All sections filled with meaningful content
- [ ] Context complexity assessed (no time estimates!)
- [ ] Iteration phases defined (not timeline milestones)
- [ ] User stories include acceptance criteria
- [ ] Risks identified with mitigations
- [ ] Vibe check completed
- [ ] File saved to `docs/prd/[feature-name].md`
- [ ] User reviewed and approved
- [ ] Next steps suggested (/tech-spec, /user-stories)

---

## Related Documentation

### Internal Documentation
- [`.agent/Tasks/prd-template.md`](.agent/Tasks/prd-template.md) - PRD template
- [`.agent/Tasks/README.md`](.agent/Tasks/README.md) - Tasks documentation index
- [`.agent/System/principles.md`](.agent/System/principles.md) - Core principles
- [`.agent/System/priority-levels.md`](.agent/System/priority-levels.md) - Priority definitions
- [`.agent/SOP/workflow.md`](.agent/SOP/workflow.md) - Development workflow

### Related Commands
- `/tech-spec` - Create technical specification from PRD
- `/user-stories` - Extract detailed user stories from PRD
- `/document` - Document existing features
- `/task` - Create task breakdown

---

## Example Usage

```bash
# Create new PRD
/prd

# Agent will guide you through:
1. Feature basics
2. Problem understanding
3. Requirements gathering
4. User stories
5. Success metrics
6. Context complexity
7. Risk assessment
8. Vibe check

# Output: docs/prd/feature-name.md

# Next steps:
/tech-spec  # Create technical specification
/user-stories  # Extract user stories for task breakdown
```

---

**Remember**: This is vibe coding! Focus on **understanding context**, not estimating time. Use **iteration readiness**, not deadlines. Trust your **gut feeling** (vibe check) to validate the plan.

---

## Success Criteria

- [ ] PRD document created in docs/prd/ directory
- [ ] Problem and solution clearly defined
- [ ] User stories documented with context complexity
- [ ] Success metrics defined
- [ ] Risks and dependencies identified
- [ ] Vibe check completed and documented
- [ ] Context complexity assessed (not time estimates!)

## Related Commands

- `/tech-spec` - Create technical specification from this PRD
- `/user-stories` - Extract and expand user stories from this PRD
- `/code` - Begin implementation after PRD and tech spec are complete
- `/update_docs` - Update .agent documentation after PRD changes
