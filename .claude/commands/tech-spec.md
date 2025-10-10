---
description: Generate technical specifications from PRD with context complexity
tags: [tech-spec, architecture, implementation, planning]
version: 1.0.0
---

# tech-spec

## Context-Driven Technical Specification from PRD

Before creating a technical specification:

1. **Read `.agent/README.md`** to understand project context
2. **Check `.agent/System/`** for current tech stack, architecture, and patterns
3. **Review `.agent/SOP/`** for technical best practices and standards
4. **Reference PRD** that this spec is based on
5. **Check existing tech specs** in `docs/tech-spec/` for patterns

## Technical Specification Creation Process

Generate a comprehensive technical specification based on an existing PRD, using context complexity instead of time estimates.

### 1. PRD Discovery Phase

**Locate the PRD**:
- Prompt user: "Enter PRD file path or I can search for you"
- If searching, look in:
  - `docs/prd/*.md`
  - `.agent/Tasks/*.md`
  - Root directory `*.md`
- Present list if multiple PRDs found

**Read and Analyze PRD Thoroughly**:
- Extract executive summary
- Understand problem statement
- Review all feature requirements (Must/Should/Could/Won't)
- Note user stories and acceptance criteria
- Understand success metrics
- Review context complexity assessment
- Identify technical considerations mentioned
- Note risks that need technical mitigations

---

### 2. System Context Discovery

**Read Current System State** (from `.agent/System/`):
- Current architecture
- Tech stack and frameworks
- Database and data storage
- API patterns and conventions
- Authentication/authorization approach
- Deployment and infrastructure
- Monitoring and observability
- Testing frameworks and patterns

**Identify Integration Points**:
- What existing systems does this touch?
- What new components are needed?
- What services need to be created/modified?
- What data models need changes?

**Check Best Practices** (from `.agent/SOP/`):
- Code standards
- Testing requirements
- Security practices
- Performance guidelines
- Documentation standards

---

### 3. Technical Requirements Extraction

**Derive Technical Requirements from PRD**:

For each functional requirement:
- What's the technical approach?
- What components are involved?
- What complexity level? [XS/S/M/L/XL]
- What dependencies exist?

**Non-Functional Requirements**:
- Performance: Response times, throughput, scalability
- Security: Authentication, authorization, encryption
- Reliability: Uptime, disaster recovery, backups
- Maintainability: Code quality, documentation
- Compliance: Data privacy, accessibility

---

### 4. Architecture Design

**High-Level Architecture**:
- What's the overall system structure?
- What new components are needed?
- How do components interact?
- What patterns are we using? (MVC, microservices, etc.)

**Component Architecture**:
- Frontend (if applicable):
  - Framework: [React/Vue/Angular/etc.] from `.agent/System/`
  - State management
  - Styling approach
  - Build tools

- Backend (if applicable):
  - Language/runtime from `.agent/System/`
  - Framework
  - API design (REST/GraphQL/gRPC)
  - Validation

- Data Layer:
  - Database choice from `.agent/System/`
  - Data models
  - Caching strategy
  - Data migration approach

- Infrastructure:
  - Cloud provider from `.agent/System/`
  - Container strategy
  - Orchestration
  - CI/CD pipeline

**Architecture Complexity**: [XS/S/M/L/XL]

---

### 5. Data Design

**Data Models**:
- What entities are needed?
- What are the relationships?
- What are the constraints?
- What indexes are needed?

**Data Flow**:
- How does data move through the system?
- What transformations occur?
- Where is data cached?
- What's the persistence strategy?

**Data Complexity**: [XS/S/M/L/XL]

---

### 6. API Design

**For REST APIs**:
- Endpoints needed
- HTTP methods
- Request/response schemas
- Error responses
- Authentication/authorization
- Versioning strategy

**For GraphQL**:
- Schema definition
- Queries and mutations
- Resolvers needed
- Error handling

**API Complexity**: [XS/S/M/L/XL]

---

### 7. Implementation Layers (Not Time-Based Phases!)

**Layer 0: Foundation**
- Architecture decisions needed
- Core infrastructure setup
- Database schema design
- API contract definition
- **Complexity**: [XS/S/M/L]
- **Context Needed**: [What must be understood]
- **Deliverables**: ADRs, schemas, contracts

**Layer 1: Core Implementation**
- Essential components
- Basic functionality
- Critical integration points
- Happy path implementation
- **Complexity**: [XS/S/M/L]
- **Context Needed**: [What must be understood]
- **Deliverables**: Working core features

**Layer 2: Enhancement**
- Additional features
- Edge case handling
- Performance optimization
- Error handling improvements
- **Complexity**: [XS/S/M/L]
- **Context Needed**: [What must be understood]
- **Deliverables**: Feature-complete implementation

**Layer 3: Refinement**
- Polish and optimization
- Monitoring and alerts
- Documentation completion
- Production readiness
- **Complexity**: [XS/S/M/L]
- **Context Needed**: [What must be understood]
- **Deliverables**: Production-ready system

---

### 8. Testing Strategy

**Test Coverage Requirements** (from `.agent/SOP/testing.md`):
- Unit Tests: [X]% minimum (from project standards)
- Integration Tests: Critical paths
- E2E Tests: User journeys from PRD
- Performance Tests: Load scenarios

**Test Approach by Layer**:
- Layer 0: Schema validation, contract tests
- Layer 1: Unit tests for core logic, integration tests
- Layer 2: Edge case tests, performance tests
- Layer 3: E2E tests, load tests

**Testing Complexity**: [XS/S/M/L/XL]

---

### 9. Security Considerations

**Security Requirements**:
- Authentication approach
- Authorization model (RBAC, ABAC, etc.)
- Data encryption (at rest, in transit)
- Input validation
- Rate limiting
- Security scanning

**Threat Model**:
- Identify threats from PRD risks
- Assess impact and likelihood
- Define mitigations
- Document security measures

**Security Complexity**: [XS/S/M/L/XL]

---

### 10. Deployment & Operations

**Deployment Strategy**:
- Environment progression (from `.agent/System/automation.md`)
- Deployment approach (blue-green, canary, rolling)
- Rollback procedures
- Feature flags

**Monitoring & Observability**:
- Metrics to track (from PRD success metrics)
- Logging strategy
- Alert thresholds
- Dashboards needed

**Operations Complexity**: [XS/S/M/L/XL]

---

### 11. Component Complexity Breakdown

**Create complexity table**:

| Component | Complexity | Reasoning | Dependencies |
|-----------|-----------|-----------|-------------|
| Frontend | [XS-XL] | [Why] | [List] |
| Backend API | [XS-XL] | [Why] | [List] |
| Data Layer | [XS-XL] | [Why] | [List] |
| Integration | [XS-XL] | [Why] | [List] |
| Testing | [XS-XL] | [Why] | [List] |
| Deployment | [XS-XL] | [Why] | [List] |

**Overall Technical Complexity**: [XS/S/M/L/XL]

---

### 12. Architecture Decision Records (ADRs)

**For significant decisions, create ADRs**:
- Decision: What was decided
- Context: Why we faced this decision
- Alternatives: What else we considered
- Consequences: Trade-offs of this choice

**Suggested ADRs** (reference `.agent/SOP/decision-making.md`):
- Technology choices
- Architecture patterns
- Database schema approach
- API design decisions
- Security approach

---

### 13. Risks & Dependencies

**Technical Risks** (from PRD + technical analysis):
- Complexity risks
- Integration risks
- Performance risks
- Security risks
- Dependency risks

**Technical Dependencies**:
- External services/APIs
- Third-party libraries
- Infrastructure requirements
- Team knowledge/skills needed

---

### 14. Document Generation

**Generate tech spec with structure**:

```markdown
# Technical Specification: [Feature Name]

**Document Version:** 1.0
**Last Updated:** [Date]
**Author:** [Name]
**Related PRD:** [Link to PRD]
**Overall Complexity:** [XS/S/M/L/XL]

## Executive Summary
[Technical overview]

## Context and Scope
[From PRD + system context]

## Technical Requirements
[Functional + Non-Functional]

## System Architecture
[High-level + component architecture]

## Data Design
[Models + flow]

## API Design
[Endpoints/schemas]

## Implementation Layers
[Layer 0-3 with complexity assessments]

## Testing Strategy
[Coverage + approach]

## Security Considerations
[Threat model + measures]

## Deployment Strategy
[Environments + rollout]

## Component Complexity Breakdown
[Table with all components]

## Architecture Decision Records
[ADRs or references]

## Risks and Mitigations
[Technical risks]

## Dependencies
[External + internal]

## Related Documentation
[Links to PRD, .agent docs, etc.]
```

**File naming**: `docs/tech-spec/[feature-name].md`

---

### 15. Cross-Reference

**Link to PRD**:
- Reference PRD file path
- Map PRD requirements to technical components
- Ensure all PRD requirements are addressed

**Update `.agent/` if needed**:
- If new patterns: Add to `.agent/SOP/`
- If architecture changes: Update `.agent/System/`
- If new decisions: Create ADRs in `docs/decisions/`

---

## Technical Specification Output Format

### Document Structure

```markdown
# Technical Specification: [Feature Name]

**Document Version:** 1.0
**Last Updated:** [Today's Date]
**Author:** [From context]
**Reviewers:** [Team leads]
**Related PRD:** [Link to PRD file]
**Overall Complexity:** [XS/S/M/L/XL]

---

## Executive Summary

[Brief technical overview of the solution, architecture approach, and key technical decisions]

**Technical Overview**:
- **Approach**: [Architecture pattern]
- **Tech Stack**: [From `.agent/System/`]
- **Complexity**: [Overall assessment]
- **Key Decisions**: [Major technical choices]

---

## Context and Scope

### Problem Context
[Technical problem from engineering perspective]

### In Scope
- [What this spec covers technically]
- [Components to be built/modified]

### Out of Scope
- [What's not covered]
- [Future technical considerations]

### Assumptions
- [Technical assumptions]
- [Infrastructure assumptions]

---

## Technical Requirements

### Functional Requirements

| ID | Requirement (from PRD) | Technical Approach | Complexity |
|----|------------------------|-------------------|-----------|
| FR-001 | [Requirement] | [Approach] | [XS-XL] |
| FR-002 | [Requirement] | [Approach] | [XS-XL] |

### Non-Functional Requirements

#### Performance
- **Latency**: p50 < Xms, p95 < Yms, p99 < Zms
- **Throughput**: X requests/second
- **Storage**: X GB initial, scalable to Y TB
- **Memory**: X MB/GB per instance
- **Complexity**: [XS/S/M/L]

#### Scalability
- **Horizontal scaling**: Auto-scale X-Y instances
- **Database**: [Scaling strategy]
- **Caching**: [Cache strategy]
- **Complexity**: [XS/S/M/L]

#### Reliability
- **Availability**: X.X% uptime (from PRD)
- **Disaster Recovery**: RPO < X, RTO < Y
- **Backup**: [Strategy and frequency]
- **Complexity**: [XS/S/M/L]

#### Security
- **Encryption**: [At rest, in transit]
- **Authentication**: [Method]
- **Authorization**: [RBAC/ABAC model]
- **Compliance**: [Requirements]
- **Complexity**: [XS/S/M/L]

---

## System Architecture

### High-Level Architecture

```text
[ASCII diagram or description of system components]

┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Client    │────▶│  API Gateway│────▶│   Service   │
│   (Web/App) │     │             │     │   Layer     │
└─────────────┘     └─────────────┘     └─────────────┘
                                               │
                    ┌─────────────┐     ┌─────────────┐
                    │   Database  │◀────│    Cache    │
                    │             │     │             │
                    └─────────────┘     └─────────────┘
```

**Architecture Complexity**: [XS/S/M/L/XL]

### Component Architecture

#### Frontend (if applicable)
- **Framework**: [From `.agent/System/`]
- **State Management**: [Approach]
- **Styling**: [Method]
- **Build Tool**: [Tool]
- **Complexity**: [XS/S/M/L]

#### Backend
- **Language/Runtime**: [From `.agent/System/`]
- **Framework**: [From `.agent/System/`]
- **ORM**: [Tool]
- **Validation**: [Library]
- **Complexity**: [XS/S/M/L]

#### Data Layer
- **Database**: [From `.agent/System/`]
- **Caching**: [Strategy]
- **Search**: [If needed]
- **Complexity**: [XS/S/M/L]

#### Infrastructure
- **Cloud Provider**: [From `.agent/System/`]
- **Container**: [Docker, etc.]
- **Orchestration**: [K8s, etc.]
- **CI/CD**: [Pipeline]
- **Complexity**: [XS/S/M/L]

---

## Data Design

### Data Models

#### Entity 1: [Name]
```language
[Schema definition in your language/framework]
```

**Relationships**: [Describe]
**Indexes**: [List]
**Complexity**: [XS/S/M/L]

#### Entity 2: [Name]
[Repeat for each entity]

### Database Schema

```sql
-- Example for SQL databases
CREATE TABLE [table_name] (
  -- columns
);

-- Indexes
CREATE INDEX [index_name] ON [table];
```

### Data Flow

```text
[Sequence diagram or description]

Client → API → Service → Cache?
                      ↓ (miss)
                   Database → Cache (update)
```

**Data Flow Complexity**: [XS/S/M/L]

---

## API Design

### REST Endpoints (or GraphQL Schema)

#### Endpoint 1: [Name]
```
[Method] [Path]
Headers: [Required headers]
Authentication: [Requirement]

Request:
{
  // Schema
}

Response: [Status Code]
{
  // Schema
}

Errors:
- [Status]: [When/why]
```

**Complexity**: [XS/S/M/L]

#### Endpoint 2: [Name]
[Repeat for each endpoint]

**API Complexity**: [XS/S/M/L]

---

## Implementation Layers

### Layer 0: Foundation

**Context Understanding Needed**:
- [Domain knowledge required]
- [System components to understand]
- [Technical concepts to learn]

**Components**:
- [ ] Database schema design
- [ ] API contract definition
- [ ] Core data models
- [ ] Authentication/authorization framework

**Complexity**: [XS/S/M/L]

**Deliverables**:
- Architecture Decision Records
- Database schema
- API specifications
- Core abstractions

---

### Layer 1: Core Implementation

**Context Building On**:
- Foundation layer complete
- [Additional context needed]

**Components**:
- [ ] Core API endpoints
- [ ] Basic business logic
- [ ] Data persistence
- [ ] Essential integrations

**Complexity**: [XS/S/M/L]

**Deliverables**:
- Working core features
- Basic test coverage
- Integration with existing systems

---

### Layer 2: Enhancement

**Context Building On**:
- Core implementation complete
- [Additional context needed]

**Components**:
- [ ] Additional features
- [ ] Edge case handling
- [ ] Performance optimization
- [ ] Enhanced error handling

**Complexity**: [XS/S/M/L]

**Deliverables**:
- Feature-complete implementation
- Comprehensive tests
- Performance benchmarks

---

### Layer 3: Refinement

**Context Building On**:
- Enhanced implementation complete
- [Additional context needed]

**Components**:
- [ ] UI/UX polish
- [ ] Monitoring and alerting
- [ ] Documentation
- [ ] Production hardening

**Complexity**: [XS/S/M/L]

**Deliverables**:
- Production-ready system
- Complete documentation
- Runbooks and playbooks

---

## Testing Strategy

### Test Coverage Requirements
[From `.agent/SOP/testing.md` or project standards]

- **Unit Tests**: X% minimum
- **Integration Tests**: All critical paths
- **E2E Tests**: Key user journeys
- **Performance Tests**: Load scenarios

### Testing by Layer

**Layer 0**: Schema validation, contract tests
**Layer 1**: Unit tests, basic integration tests
**Layer 2**: Edge case tests, performance tests
**Layer 3**: E2E tests, load tests, chaos tests

### Test Approach

```language
// Example test structure for your framework
describe('[Component]', () => {
  it('should [behavior]', () => {
    // Arrange
    // Act
    // Assert
  });
});
```

**Testing Complexity**: [XS/S/M/L]

---

## Security Considerations

### Threat Model

| Threat | Impact | Likelihood | Mitigation |
|--------|--------|-----------|-----------|
| [Threat 1] | [High/Med/Low] | [High/Med/Low] | [Strategy] |
| [Threat 2] | [High/Med/Low] | [High/Med/Low] | [Strategy] |

### Security Measures

**Authentication**: [Approach]
**Authorization**: [RBAC/ABAC details]
**Encryption**: [Methods]
**Input Validation**: [Strategy]
**Rate Limiting**: [Limits]
**Secrets Management**: [Tool/approach]

**Security Complexity**: [XS/S/M/L]

---

## Deployment Strategy

### Environments
- **Development**: [Details]
- **Staging**: [Details]
- **Production**: [Details]

### Deployment Approach
[Blue-green, canary, rolling, etc.]

### Feature Flags
- Flag: `feature_[name]_enabled`
- Default: Disabled
- Rollout: Gradual

### Rollback Plan
- Triggers: [When to rollback]
- Process: [How to rollback]
- Verification: [How to verify]

**Deployment Complexity**: [XS/S/M/L]

---

## Monitoring & Observability

### Metrics
[From PRD success metrics + technical metrics]

- Response time (p50, p95, p99)
- Error rate
- Request rate
- Resource usage (CPU, memory)
- Business metrics

### Logging
- Log level strategy
- Structured logging format
- Log retention policy

### Alerts
- Threshold definitions
- Alert routing
- On-call procedures

**Observability Complexity**: [XS/S/M/L]

---

## Component Complexity Breakdown

| Component | Complexity | Reasoning | Dependencies |
|-----------|-----------|-----------|-------------|
| Frontend | [XS-XL] | [Why this level] | [List] |
| Backend API | [XS-XL] | [Why this level] | [List] |
| Data Layer | [XS-XL] | [Why this level] | [List] |
| Integration | [XS-XL] | [Why this level] | [List] |
| Testing | [XS-XL] | [Why this level] | [List] |
| Security | [XS-XL] | [Why this level] | [List] |
| Deployment | [XS-XL] | [Why this level] | [List] |
| Monitoring | [XS-XL] | [Why this level] | [List] |

**Overall Technical Complexity**: [XS/S/M/L/XL]

---

## Architecture Decision Records

### ADR-001: [Decision Title]
**Status**: Proposed | Accepted
**Context**: [Why this decision needed]
**Decision**: [What was decided]
**Consequences**: [Trade-offs]
**Alternatives**: [What else considered]

[Repeat for each major decision]

---

## Risks and Mitigations

| Risk | Probability | Impact | Complexity | Mitigation |
|------|------------|--------|-----------|-----------|
| [Tech risk 1] | [L/M/H] | [L/M/H] | [XS-XL] | [Strategy] |
| [Tech risk 2] | [L/M/H] | [L/M/H] | [XS-XL] | [Strategy] |

---

## Dependencies

### External Dependencies
- [Service/API 1]: [Purpose, version]
- [Library 1]: [Purpose, version]

### Internal Dependencies
- [System 1]: [Integration point]
- [Service 1]: [Dependency]

### Infrastructure Dependencies
- [Cloud service]: [Requirement]
- [Database]: [Version, configuration]

**Dependency Complexity**: [XS/S/M/L]

---

## Related Documentation

### Internal Documentation
- **PRD**: [Link to PRD file]
- **`.agent/System/`**: [Relevant system docs]
- **`.agent/SOP/`**: [Relevant practices]
- **Architecture Decisions**: [Link to ADRs]

### External Resources
- [Framework documentation]
- [API documentation]
- [Best practices guides]

---

## Next Steps

1. **Review & Approval**:
   - Technical review by team
   - Architecture review
   - Security review

2. **Implementation Planning**:
   - Run `/user-stories` to break down into tasks
   - Create task breakdown in `.agent/Tasks/`
   - Assign components to team members

3. **Documentation Updates**:
   - Update `.agent/System/` if architecture changes
   - Add new patterns to `.agent/SOP/` if applicable
   - Create ADRs for major decisions

---

## Approval

| Role | Name | Date | Status |
|------|------|------|--------|
| Tech Lead | | | ☐ |
| Architect | | | ☐ |
| Security | | | ☐ |
| DevOps | | | ☐ |

---

*This technical specification uses context-driven planning. Focus on understanding complexity and implementation layers, not arbitrary time estimates.*
```

---

## Interactive Prompts

### PRD Selection Prompt
```
Let's create a technical specification.

Do you have a PRD file path, or should I search for PRDs?
1. I have the path: [Enter path]
2. Search for PRDs
3. Start without PRD (manual input)
```

### Architecture Analysis Prompt
```
Based on the PRD, I need to understand the technical approach.

Current system (from .agent/System/):
- Tech stack: [List]
- Architecture: [Description]
- Patterns: [List]

For this feature:
1. What new components are needed?
2. What existing components need modification?
3. What's the integration approach?
4. What's your gut feeling on complexity? [XS/S/M/L/XL]
```

### Component Complexity Prompt
```
Let's assess complexity for each component:

Frontend complexity: [XS | S | M | L | XL] - Why?
Backend complexity: [XS | S | M | L | XL] - Why?
Data layer complexity: [XS | S | M | L | XL] - Why?
Integration complexity: [XS | S | M | L | XL] - Why?
Testing complexity: [XS | S | M | L | XL] - Why?

Any components I missed?
```

### Implementation Layers Prompt
```
Let's define implementation layers (not time-based phases):

Layer 0 (Foundation):
- What architecture decisions are needed?
- What needs to be designed upfront?
- Complexity: [XS/S/M/L]

Layer 1 (Core):
- What's the essential functionality?
- What must work for the happy path?
- Complexity: [XS/S/M/L]

Layer 2 (Enhancement):
- What additional features are needed?
- What edge cases to handle?
- Complexity: [XS/S/M/L]

Layer 3 (Refinement):
- What polish is needed?
- What production hardening?
- Complexity: [XS/S/M/L]
```

---

## Best Practices

### DO ✅
- Start from PRD requirements
- Reference `.agent/System/` for tech stack
- Follow patterns from `.agent/SOP/`
- Use context complexity assessments
- Define implementation layers (not phases)
- Create ADRs for decisions
- Consider security from the start
- Plan for monitoring and observability
- Document all dependencies
- Cross-reference to PRD

### DON'T ❌
- Use time estimates (weeks/days/hours)
- Ignore existing architecture
- Skip security considerations
- Forget about testing strategy
- Make architecture decisions without ADRs
- Over-engineer the solution
- Forget monitoring and alerts
- Ignore deployment strategy
- Skip complexity assessment
- Lose traceability to PRD

---

## Success Criteria

- [ ] Tech spec created from PRD
- [ ] All PRD requirements addressed
- [ ] Architecture designed
- [ ] Component complexity assessed
- [ ] Implementation layers defined (not timelines!)
- [ ] Testing strategy included
- [ ] Security considerations documented
- [ ] Deployment strategy defined
- [ ] ADRs created for major decisions
- [ ] Cross-referenced to PRD
- [ ] File saved to `docs/tech-spec/[feature-name].md`
- [ ] Ready for implementation planning

---

## Related Documentation

### Internal Documentation
- [`.agent/Tasks/tech-spec-template.md`](.agent/Tasks/tech-spec-template.md) - Tech spec template
- [`.agent/System/`](.agent/System/) - Current system architecture
- [`.agent/SOP/`](.agent/SOP/) - Technical best practices
- [`.agent/Tasks/prd-template.md`](.agent/Tasks/prd-template.md) - PRD reference

### Related Commands
- `/prd` - Create product requirements document
- `/user-stories` - Extract user stories for implementation
- `/document` - Document technical decisions

---

## Example Usage

```bash
# Create tech spec from existing PRD
/tech-spec

# Agent will:
1. Ask for PRD file path (or search)
2. Read and analyze PRD
3. Check current system architecture
4. Generate technical specification
5. Assess component complexity
6. Define implementation layers
7. Create ADRs for decisions
8. Save to docs/tech-spec/feature-name.md

# Output: Complete technical specification

# Next steps:
/user-stories  # Break down into implementable tasks
```

---

**Remember**: Technical complexity is about **understanding depth**, not time to build. Use **implementation layers** to organize work by context, not deadlines.

---

## Success Criteria

- [ ] Technical specification created in docs/tech-spec/ directory
- [ ] Architecture and design clearly documented
- [ ] Component complexity assessed (XS/S/M/L/XL)
- [ ] Implementation layers defined
- [ ] ADRs created for key decisions
- [ ] Integration points identified
- [ ] Testing strategy defined
- [ ] System documentation (.agent/System/) updated

## Related Commands

- `/prd` - Reference the product requirements this spec is based on
- `/user-stories` - Extract user stories for implementation
- `/code` - Begin implementation following this spec
- `/update_docs` - Update .agent/System/ with architectural changes
- `/document` - Create additional technical documentation
