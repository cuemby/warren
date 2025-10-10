---
description: Extract and create detailed user stories with acceptance criteria
tags: [user-stories, agile, requirements, planning]
version: 1.0.0
---

# user-stories

## Context-Driven User Stories Creation

Before creating user stories:

1. **Read `.agent/README.md`** to understand project context
2. **Check `.agent/Tasks/README.md`** for user story guidelines
3. **Review `.agent/System/priority-levels.md`** for priority framework
4. **Reference PRD (if available)** for feature context

## User Stories Creation Process

Generate detailed user stories from a PRD or create new ones using the vibe-coding philosophy (context complexity, not time estimates).

### Mode Selection

**Prompt user**:
```
How would you like to create user stories?

1. Extract from existing PRD
2. Create new user stories from scratch
3. Expand existing user stories
```

---

## Mode A: Extract from PRD

### 1. PRD Discovery

**Locate PRD**:
- Prompt: "Enter PRD file path or I can search"
- Search in:
  - `docs/prd/*.md`
  - `.agent/Tasks/*.md`
  - Root `*.md`
- Present list if multiple found

**Read and Parse PRD**:
- Extract executive summary for context
- Find "User Stories" section
- Identify all epics and stories
- Note acceptance criteria
- Review user personas
- Understand success metrics
- Check context complexity assessments

---

### 2. Story Extraction & Expansion

**For each story found in PRD**:

#### Extract Core Information
- Story title
- User type ("As a...")
- Desired capability ("I want...")
- Value/benefit ("So that...")
- Priority level
- Context complexity (if present)
- Acceptance criteria

#### Expand with Details
- **Add detailed acceptance criteria** using GIVEN/WHEN/THEN format
- **Assess context complexity** if not in PRD
- **Identify dependencies** on other stories
- **Define context requirements** (what understanding needed)
- **Add technical notes** (from tech spec if available)
- **Include test scenarios**

#### Example Expansion
```
Original (from PRD):
Story: User login
As a user, I want to log in, so that I can access my account.
Priority: P0
Acceptance Criteria:
- User can enter credentials
- User is authenticated

Expanded:
Story: User Login Authentication
As a registered user
I want to securely log in with my email and password
So that I can access my personalized account dashboard

Priority: P0 (Critical) / [CRITICAL]
Context Complexity: M (Moderate)
Dependencies:
- User registration system
- Session management
- Database access

Acceptance Criteria:
- [ ] GIVEN I am on the login page
    WHEN I enter valid email and password
    THEN I am authenticated and redirected to dashboard

- [ ] GIVEN I am on the login page
    WHEN I enter invalid credentials
    THEN I see a clear error message "Invalid email or password"

- [ ] GIVEN I have entered wrong password 3 times
    WHEN I attempt to login again
    THEN my account is temporarily locked for 15 minutes

- [ ] GIVEN I am successfully logged in
    WHEN I close the browser and return within session timeout
    THEN I remain logged in

- [ ] GIVEN I am on the login page
    WHEN I click "Forgot Password"
    THEN I am taken to password reset flow

Context Requirements:
- Understanding of authentication flows (OAuth, JWT, etc.)
- Knowledge of session management
- Security best practices for auth
- Database schema for users table

Technical Notes:
- Use bcrypt for password hashing
- JWT tokens with 24h expiry
- Refresh token strategy needed
- Rate limiting on login endpoint

Test Scenarios:
- Happy path: Valid credentials → Success
- Invalid email format → Validation error
- Wrong password → Auth error
- Account locked → Locked message
- Session expiry → Re-authenticate
- CSRF protection → Token validation
```

---

### 3. Story Organization

**Group by Epic**:
- Epic: [Name from PRD]
  - Story 1
  - Story 2
  - Story 3

**Order by Priority + Complexity**:
- Use matrix from `.agent/Tasks/README.md`:
  ```
  Priority ↓ / Complexity →
  [BLOCKER] + XS = Do immediately
  [CRITICAL] + M = Plan and execute
  [REQUIRED] + L = Break down first
  ```

**Check for Story Size**:
- If complexity is XL → Break down into smaller stories
- If story has > 8 acceptance criteria → Consider splitting
- If multiple user types → Create separate stories

---

### 4. Generate Output

**Create structured document**:

```markdown
# User Stories: [Feature Name]

**Source PRD:** [Link to PRD]
**Generated:** [Date]
**Total Stories:** [Count]
**Total Complexity:** [Sum or average]

---

## Epic: [Epic Name]

**Epic Priority:** [BLOCKER/CRITICAL/REQUIRED/RECOMMENDED/OPTIONAL]
**Epic Complexity:** [XS/S/M/L/XL]
**Epic Goal:** [What this epic achieves]

---

### Story 1: [Story Title]

**ID:** US-001
**Priority:** P0 (Critical) / [CRITICAL]
**Context Complexity:** M (Moderate)

**User Story:**
As a [user type]
I want [capability]
So that [benefit]

**Acceptance Criteria:**
- [ ] GIVEN [context], WHEN [action], THEN [outcome]
- [ ] GIVEN [context], WHEN [action], THEN [outcome]
- [ ] GIVEN [context], WHEN [action], THEN [outcome]

**Context Requirements:**
- [Understanding needed]
- [Knowledge required]
- [Concepts to grasp]

**Dependencies:**
- US-XXX: [Description]
- System: [Existing system]

**Technical Notes:**
- [Implementation hints]
- [Architecture considerations]
- [Security requirements]

**Test Scenarios:**
- Happy path: [Description]
- Error cases: [Description]
- Edge cases: [Description]

**Definition of Done:**
- [ ] Code implemented
- [ ] Unit tests passing
- [ ] Integration tests passing
- [ ] Code reviewed
- [ ] Documentation updated
- [ ] Deployed to staging
- [ ] Acceptance criteria verified

---

[Repeat for each story]

---

## Story Breakdown by Priority

### [BLOCKER] Stories
- US-XXX: [Title] - Complexity: [X]

### [CRITICAL] Stories
- US-XXX: [Title] - Complexity: [X]

### [REQUIRED] Stories
- US-XXX: [Title] - Complexity: [X]

### [RECOMMENDED] Stories
- US-XXX: [Title] - Complexity: [X]

### [OPTIONAL] Stories
- US-XXX: [Title] - Complexity: [X]

---

## Complexity Analysis

| Epic | Stories | XS | S | M | L | XL | Total Complexity |
|------|---------|----|----|----|----|----|--------------------|
| Epic 1 | 5 | 2 | 2 | 1 | 0 | 0 | Low-Medium |
| Epic 2 | 3 | 0 | 1 | 2 | 0 | 0 | Medium |
| Total | 8 | 2 | 3 | 3 | 0 | 0 | Medium |

---

## Dependency Graph

```
US-001 (Foundation)
  ↓
US-002 (Core) → US-003 (Core)
  ↓                ↓
US-004 (Enhancement)
```

---

## Implementation Suggestion

**Layer 0 (Foundation):**
- US-001: [Story with foundational context]

**Layer 1 (Core):**
- US-002, US-003: [Core functionality stories]

**Layer 2 (Enhancement):**
- US-004, US-005: [Enhancement stories]

**Layer 3 (Refinement):**
- US-006: [Polish and optimization stories]

---

## Related Documentation

- **PRD:** [Link]
- **Tech Spec:** [Link if exists]
- **Tasks:** [Link to .agent/Tasks/]
```

**Save to:** `docs/user-stories/[feature-name].md`

---

## Mode B: Create New User Stories

### 1. Context Gathering

**Ask user**:
- What feature are we writing stories for?
- Do you have a PRD? (If yes, switch to Mode A)
- Who are the users?
- What problem are we solving?

---

### 2. Guided Story Creation

**For each story, guide through**:

#### Story Basics
```
Let's create a user story.

1. Who is the user? (role/persona)
2. What do they want to do? (capability)
3. Why do they want this? (benefit/value)
```

#### Priority Assignment
```
What's the priority?

[BLOCKER] - Stops everything, must fix now
[CRITICAL] - Must complete before moving forward
[REQUIRED] - Necessary for production
[RECOMMENDED] - Should do for quality
[OPTIONAL] - Nice to have

Or use: P0 (Critical) | P1 (High) | P2 (Medium) | P3 (Low)
```

#### Complexity Assessment
```
How much understanding is needed?

XS - Trivial, single concept, obvious approach
S  - Simple, few related concepts, clear path
M  - Moderate, multiple interconnected concepts
L  - Large, complex system with dependencies
XL - Extra large, needs breakdown into smaller stories
```

#### Acceptance Criteria
```
What are the acceptance criteria?

Use GIVEN/WHEN/THEN format:

1. GIVEN [starting context]
   WHEN [user action]
   THEN [expected outcome]

Add as many as needed to fully define "done"
```

#### Dependencies
```
Does this story depend on anything?

- Other user stories?
- Existing systems?
- External services?
- Technical infrastructure?
```

#### Context Requirements
```
What needs to be understood to implement this?

- Domain knowledge required
- Technical concepts
- System architecture
- Integration points
```

---

### 3. Story Validation

**Check story quality**:

#### INVEST Criteria
- **Independent**: Can be worked on independently?
- **Negotiable**: Details can be refined?
- **Valuable**: Delivers user value?
- **Estimable**: Complexity can be assessed?
- **Small**: Completable in one iteration?
- **Testable**: Clear acceptance criteria?

#### Story Smells
- Too vague? → Add more acceptance criteria
- Too large (XL)? → Break into smaller stories
- Too technical? → Rephrase from user perspective
- No clear value? → Clarify the "so that"

---

### 4. Generate Story Document

**Same format as Mode A extraction**

**Save to:** `docs/user-stories/[feature-name].md`

---

## Mode C: Expand Existing Stories

### 1. Story Selection

**Prompt**:
- Enter file path with existing stories, or
- Enter story text to expand

---

### 2. Analysis & Expansion

**For each story**:

#### Add Missing Elements
- Fill in context complexity if missing
- Add GIVEN/WHEN/THEN acceptance criteria
- Identify dependencies
- Document context requirements
- Add technical notes
- Define test scenarios
- Create definition of done

#### Enhance Acceptance Criteria
- Convert high-level criteria to GIVEN/WHEN/THEN
- Add edge cases
- Add error scenarios
- Add security considerations

#### Break Down Large Stories
- If complexity is L or XL
- If > 8 acceptance criteria
- If multiple concerns mixed
- Suggest smaller stories

---

### 3. Generate Enhanced Document

**Output enhanced stories** with all details filled in

---

## Story Templates

### Minimal Story Template
```markdown
### Story: [Title]

As a [user type]
I want [capability]
So that [benefit]

**Priority:** [P0-P3 or BLOCKER/CRITICAL/etc.]
**Context Complexity:** [XS/S/M/L/XL]

**Acceptance Criteria:**
- [ ] GIVEN [context], WHEN [action], THEN [outcome]
```

### Complete Story Template
```markdown
### Story: [Title]

**ID:** US-###
**Priority:** P0 (Critical) / [CRITICAL]
**Context Complexity:** M (Moderate)
**Epic:** [Epic name]

**User Story:**
As a [user type]
I want [capability]
So that [benefit]

**Acceptance Criteria:**
- [ ] GIVEN [context], WHEN [action], THEN [outcome]
- [ ] GIVEN [context], WHEN [action], THEN [outcome]
- [ ] GIVEN [context], WHEN [action], THEN [outcome]

**Context Requirements:**
- [Understanding needed]
- [Knowledge required]

**Dependencies:**
- [List dependencies]

**Technical Notes:**
- [Implementation considerations]

**Test Scenarios:**
- Happy path: [Description]
- Error cases: [Description]
- Edge cases: [Description]

**Definition of Done:**
- [ ] Code implemented
- [ ] Tests passing (unit, integration)
- [ ] Code reviewed
- [ ] Documentation updated
- [ ] Acceptance criteria verified
```

---

## Story Size Guidelines

### XS (Extra Small) - Trivial
- Single file change
- No new concepts
- Obvious implementation
- < 2 acceptance criteria
- Example: "Fix typo in error message"

### S (Small) - Simple
- 2-3 files involved
- Familiar patterns
- Clear dependencies
- 2-4 acceptance criteria
- Example: "Add input validation to form field"

### M (Medium) - Moderate
- Multiple components
- Some new concepts
- Few integration points
- 4-6 acceptance criteria
- Example: "Implement user profile editing"

### L (Large) - Complex
- System-wide changes
- New architecture needed
- Multiple dependencies
- 6-8 acceptance criteria
- Example: "Add real-time notifications system"

### XL (Extra Large) - Too Big
- **MUST BREAK DOWN**
- Multiple sub-systems
- Many unknowns
- > 8 acceptance criteria
- Example: "Implement complete payment system" → Break into: setup, checkout, processing, refunds, reporting

---

## Priority + Complexity Matrix

Decision guide for story ordering (from `.agent/Tasks/README.md`):

```
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

## Export Formats

### Markdown (Default)
Standard markdown file as shown above

### YAML (for task trackers)
```yaml
user_stories:
  - id: US-001
    title: "Story title"
    description: |
      As a [user]
      I want [capability]
      So that [benefit]
    priority: P0
    complexity: M
    acceptance_criteria:
      - "GIVEN ... WHEN ... THEN ..."
    dependencies:
      - US-002
    context_requirements:
      - "Understanding needed"
```

### JSON (for APIs/tools)
```json
{
  "user_stories": [
    {
      "id": "US-001",
      "title": "Story title",
      "user_type": "user",
      "capability": "capability",
      "benefit": "benefit",
      "priority": "P0",
      "complexity": "M",
      "acceptance_criteria": ["..."],
      "dependencies": ["US-002"]
    }
  ]
}
```

### Task Tracker Format (GitHub Issues, Jira, Linear)
```markdown
**GitHub Issues Format:**

Title: [Story Title]

Body:
## User Story
As a [user], I want [capability], so that [benefit]

## Priority
P0 (Critical) / [CRITICAL]

## Complexity
M (Moderate)

## Acceptance Criteria
- [ ] GIVEN ... WHEN ... THEN ...

## Dependencies
- #123 (Other issue)

Labels: user-story, priority:critical, complexity:medium
```

---

## Interactive Prompts

### Mode Selection Prompt
```
How would you like to create user stories?

1. Extract and expand from existing PRD
   → I'll read the PRD and generate detailed stories

2. Create new stories from scratch
   → I'll guide you through creating stories step-by-step

3. Expand existing stories
   → I'll take your stories and add missing details

Which mode? [1/2/3]
```

### PRD Selection Prompt (Mode 1)
```
Let's extract user stories from a PRD.

Option 1: Enter PRD file path
Option 2: I'll search for PRDs in docs/prd/

Your choice? [1/2]

[If search]: Found these PRDs:
1. docs/prd/user-authentication.md
2. docs/prd/payment-integration.md

Which one? [1/2]
```

### New Story Prompt (Mode 2)
```
Let's create a user story.

Story #1:

1. Who is the user? (e.g., "registered user", "admin", "guest")
   →

2. What capability do they want? (e.g., "log in", "view dashboard")
   →

3. Why do they want this? (the benefit)
   →

4. Priority? [BLOCKER | CRITICAL | REQUIRED | RECOMMENDED | OPTIONAL]
   →

5. Complexity? [XS | S | M | L | XL]
   →

Now let's define acceptance criteria...
```

### Acceptance Criteria Prompt
```
Define acceptance criteria (GIVEN/WHEN/THEN format):

Criterion #1:
GIVEN: [starting context]
WHEN: [user action]
THEN: [expected outcome]

Add another criterion? [y/n]
```

### Export Format Prompt
```
How would you like to export these stories?

1. Markdown file (default)
   → docs/user-stories/[name].md

2. YAML format
   → For task trackers, automation

3. JSON format
   → For APIs, programmatic access

4. GitHub Issues format
   → Ready to paste into GitHub

5. All formats
   → Generate all export formats

Your choice? [1/2/3/4/5]
```

---

## Best Practices

### DO ✅
- Write from user perspective ("As a user...")
- Focus on user value ("So that...")
- Use GIVEN/WHEN/THEN for acceptance criteria
- Assess context complexity (not time!)
- Identify dependencies explicitly
- Break down XL stories
- Include edge cases and errors
- Make stories testable
- Keep stories independent when possible
- Reference PRD and tech spec

### DON'T ❌
- Write technical tasks disguised as stories
- Estimate in hours/days/weeks
- Create stories without acceptance criteria
- Mix multiple concerns in one story
- Make stories too large (> L complexity)
- Forget about error scenarios
- Skip context requirements
- Ignore dependencies
- Write vague acceptance criteria
- Lose traceability to PRD

---

## Success Criteria

- [ ] Stories extracted/created
- [ ] All stories have acceptance criteria (GIVEN/WHEN/THEN)
- [ ] Context complexity assessed for each
- [ ] Priority assigned to each
- [ ] Dependencies identified
- [ ] Context requirements documented
- [ ] Stories grouped by epic
- [ ] Stories ordered by priority + complexity
- [ ] XL stories broken down
- [ ] File saved to `docs/user-stories/[feature-name].md`
- [ ] Export format generated if requested
- [ ] Ready for implementation

---

## Related Documentation

### Internal Documentation
- [`.agent/Tasks/README.md`](.agent/Tasks/README.md) - Task management and story guidelines
- [`.agent/Tasks/prd-template.md`](.agent/Tasks/prd-template.md) - PRD reference
- [`.agent/System/priority-levels.md`](.agent/System/priority-levels.md) - Priority framework
- [`.agent/SOP/task-management.md`](.agent/SOP/task-management.md) - Task lifecycle

### Related Commands
- `/prd` - Create product requirements document
- `/tech-spec` - Create technical specification
- `/task` - Create individual task from story

---

## Example Usage

### Mode 1: Extract from PRD
```bash
/user-stories

# Select mode 1: Extract from PRD
# Enter PRD path: docs/prd/user-authentication.md

# Agent will:
1. Read PRD
2. Extract all user stories
3. Expand with detailed acceptance criteria
4. Assess context complexity
5. Identify dependencies
6. Organize by epic and priority
7. Generate docs/user-stories/user-authentication.md

# Output: Complete user stories document
```

### Mode 2: Create from Scratch
```bash
/user-stories

# Select mode 2: Create new stories
# Agent guides through:
1. Story basics (As a/I want/So that)
2. Priority selection
3. Complexity assessment
4. Acceptance criteria (GIVEN/WHEN/THEN)
5. Dependencies
6. Context requirements

# Repeat for each story
# Output: Complete user stories document
```

### Mode 3: Expand Existing
```bash
/user-stories

# Select mode 3: Expand existing
# Enter file path or paste stories
# Agent will:
1. Parse existing stories
2. Add missing GIVEN/WHEN/THEN
3. Assess complexity
4. Add context requirements
5. Identify dependencies
6. Break down large stories

# Output: Enhanced user stories document
```

---

**Remember**: User stories describe **what users want to achieve**, not **how to implement**. Focus on user value and context complexity, not technical implementation details or time estimates.

---

## Success Criteria

- [ ] User stories extracted or created
- [ ] All stories have acceptance criteria (GIVEN/WHEN/THEN format)
- [ ] Context complexity assessed for each story (XS/S/M/L/XL)
- [ ] Priority assigned to each story
- [ ] Dependencies identified
- [ ] Stories grouped by epic
- [ ] XL stories broken down into smaller pieces
- [ ] Document saved to docs/user-stories/

## Related Commands

- `/prd` - Create or reference the PRD these stories are based on
- `/tech-spec` - Create technical specification from stories
- `/code` - Begin implementation of user stories
- `/test` - Write tests based on acceptance criteria
