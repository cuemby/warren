---
description: Comprehensive code review with quality and security checks
tags: [code-review, quality, security]
version: 1.0.0
---

# review

## Context-Driven Code Review

Before reviewing any code:

1. **Read `.agent/README.md`** to understand the current system state
2. **Check `.agent/System/`** for architecture, tech stack, and integration points
3. **Review `.agent/SOP/`** for project-specific standards and best practices
4. **Reference `.agent/Tasks/`** for the feature context and acceptance criteria

## Review Process

Perform comprehensive code review on [piece of code] following this systematic workflow:

### 1. Context Analysis Phase

- Understand the code's purpose and intended functionality
- Review related `.agent/Tasks/` documentation for requirements
- Identify scope: new feature, bug fix, refactor, or optimization
- Check how code fits within system architecture from `.agent/System/`

### 2. Code Quality Assessment

- **Functionality**: Does it solve the problem correctly?
- **Correctness**: Are there logical errors or edge cases missed?
- **Simplicity**: Is it the minimal code needed? Any over-engineering?
- **Readability**: Is the code clear and self-documenting?
- **Maintainability**: Can others easily understand and modify it?

### 3. Standards Compliance Check

- Verify adherence to patterns documented in `.agent/SOP/`
- Check consistency with tech stack in `.agent/System/`
- Validate coding style and conventions
- Ensure proper error handling and validation
- Review naming conventions and code organization

### 4. Security & Performance Review

- **Security**: Identify vulnerabilities (injection, XSS, auth issues, data exposure)
- **Performance**: Spot bottlenecks (N+1 queries, inefficient algorithms, memory leaks)
- **Resource Management**: Check for proper cleanup and resource handling
- **Scalability**: Assess if code scales with growth

### 5. Testing & Documentation Review

- Are tests adequate and meaningful?
- Is the code testable and properly structured?
- Are edge cases covered?
- Is documentation clear and up-to-date?
- Should this pattern be added to `.agent/SOP/`?

### 6. Improvement Recommendations

- Suggest specific, actionable improvements
- Provide examples or references from `.agent/SOP/` or `.agent/System/`
- Prioritize suggestions: [CRITICAL], [REQUIRED], [RECOMMENDED], [OPTIONAL]
- Explain the "why" behind each suggestion for educational value

### 7. Documentation Updates

- Identify if new patterns should be added to `.agent/SOP/`
- Note if `.agent/System/` needs updates based on changes
- Suggest improvements to `.agent/Tasks/` documentation if incomplete

## Review Output Format

Provide structured feedback:

### Summary

- Overall assessment: [Approve | Approve with suggestions | Request changes]
- Key strengths (what's done well)
- Critical issues (must fix before merge)

### Detailed Findings

#### 🔴 Critical Issues [BLOCKER]

- Issue with code location/line reference
- Why it's critical
- Specific fix recommendation

#### 🟡 Important Issues [REQUIRED]

- Issue with code location/line reference
- Impact and reasoning
- Suggested improvement with example

#### 🟢 Suggestions [RECOMMENDED]

- Optimization opportunity
- Reasoning and benefits
- Optional improvement with example

#### 💡 Best Practices [OPTIONAL]

- Long-term improvements
- References to `.agent/SOP/` or industry standards
- Educational insights

### Documentation Needs

- [ ] Add pattern to `.agent/SOP/[topic].md`
- [ ] Update `.agent/System/[component].md`
- [ ] Improve `.agent/Tasks/[feature].md` acceptance criteria

## Review Principles

- **Constructive Mentorship**: Educate, don't just criticize
- **Context-Aware**: Reference project standards from `.agent/`
- **Actionable Feedback**: Provide specific fixes, not vague comments
- **Prioritized**: Distinguish blockers from nice-to-haves
- **Example-Driven**: Show good alternatives when suggesting changes
- **Simplicity Focused**: Favor simpler solutions over complex ones
- **Pattern Recognition**: Identify reusable patterns for `.agent/SOP/`

## Code Review Checklist

- [ ] Functionality matches requirements in `.agent/Tasks/`
- [ ] Follows architecture patterns in `.agent/System/`
- [ ] Adheres to SOPs in `.agent/SOP/`
- [ ] No security vulnerabilities
- [ ] No performance bottlenecks
- [ ] Adequate test coverage
- [ ] Clear and maintainable code
- [ ] Proper error handling
- [ ] Documentation updated
- [ ] No breaking changes (or properly documented)

## Success Criteria

- [ ] All critical issues identified and documented
- [ ] Security vulnerabilities flagged
- [ ] Performance concerns highlighted
- [ ] Actionable feedback provided with examples
- [ ] Suggestions prioritized ([BLOCKER], [REQUIRED], [RECOMMENDED], [OPTIONAL])
- [ ] Documentation needs identified
- [ ] Overall assessment provided (Approve/Request Changes)

## Related Commands

- `/code` - Implement recommended improvements
- `/refactor` - Apply structural improvements suggested in review
- `/test` - Add missing test coverage
- `/debug` - Fix issues identified in review
- `/document` - Add documentation improvements
