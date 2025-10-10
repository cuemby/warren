# Code Review Standards

## Overview

Code review is a critical quality gate that ensures code quality, knowledge sharing, and maintainability. This document defines standards and best practices for effective code reviews.

## Review Checklist

### Functionality

- [ ] Does the code solve the stated problem?
- [ ] Are all acceptance criteria met?
- [ ] Are edge cases handled?
- [ ] Are error conditions properly managed?
- [ ] Does it integrate well with existing features?

### Tests

- [ ] Are there adequate unit tests?
- [ ] Are integration tests included where needed?
- [ ] Do tests cover edge cases?
- [ ] Are tests readable and maintainable?
- [ ] Do all tests pass?
- [ ] Is test coverage maintained or improved?

### Security

- [ ] Is user input validated?
- [ ] Are security best practices followed?
- [ ] Are there no hardcoded secrets?
- [ ] Is sensitive data properly handled?
- [ ] Are authentication/authorization checks in place?
- [ ] Are dependencies secure and up-to-date?

### Performance

- [ ] Are there any obvious bottlenecks?
- [ ] Are database queries optimized?
- [ ] Is caching used appropriately?
- [ ] Are large operations async where appropriate?
- [ ] Does it meet performance benchmarks?

### Documentation

- [ ] Is the code self-documenting?
- [ ] Are complex algorithms explained?
- [ ] Is public API documented?
- [ ] Are README/docs updated?
- [ ] Are breaking changes documented?

### Code Quality

- [ ] Follows project coding standards?
- [ ] Is code DRY (Don't Repeat Yourself)?
- [ ] Are functions/methods focused and small?
- [ ] Are names clear and descriptive?
- [ ] Is error handling consistent?
- [ ] Are dependencies minimal?

---

## Review Etiquette

### For Reviewers

#### DO ✅

- **Focus on code, not person**: Comment on the work, not the developer
- **Be constructive**: Suggest improvements, not just criticisms
- **Explain why**: Help the author understand the reasoning
- **Ask questions**: "Could we..." instead of "You should..."
- **Acknowledge good work**: Recognize clever solutions
- **Be timely**: Respond within 24 hours
- **Be thorough**: Check all aspects, not just syntax

#### DON'T ❌

- Use judgmental language ("This is stupid", "Obviously wrong")
- Be vague ("This doesn't look right")
- Nitpick trivial issues while missing major problems
- Request changes without explanation
- Approve without actually reviewing
- Review your own code (except in emergencies)
- Block on personal preference issues

### For Authors

#### DO ✅

- **Provide context**: Explain the problem and approach
- **Keep changes focused**: One logical change per PR
- **Self-review first**: Check your own code before requesting review
- **Respond promptly**: Address feedback within 24 hours
- **Be open to feedback**: Assume good intent
- **Ask clarifying questions**: If feedback is unclear
- **Update docs**: Keep documentation in sync

#### DON'T ❌

- Submit massive PRs (> 400 lines)
- Mix unrelated changes
- Take feedback personally
- Argue without listening
- Mark conversations resolved without addressing them
- Push changes without re-requesting review
- Rush reviewers

---

## Review Process

### 1. Pre-Review (Author)

```bash
# Self-review checklist
$LINTER check
$TEST_RUNNER run
$VCS_TOOL diff --cached

# Create PR with proper template
$VCS_PLATFORM pr create --title "type: description"
```

### 2. Initial Review (Reviewer)

- Read PR description and linked issues
- Understand the context and goals
- Check CI/CD status
- Review files changed
- Test locally if needed

### 3. Provide Feedback

**Comment Types**:

- **[BLOCKER]**: Must fix before merge
- **[SUGGESTION]**: Consider changing
- **[QUESTION]**: Need clarification
- **[PRAISE]**: Good work!
- **[NITPICK]**: Optional improvement

**Example Comments**:

```text
[BLOCKER] This endpoint doesn't validate user input, which
could lead to SQL injection. Consider using parameterized queries.

[SUGGESTION] Could we extract this into a helper function?
It might make the code more reusable.

[QUESTION] What happens if the API returns an error here?

[PRAISE] Nice use of the strategy pattern here!

[NITPICK] Minor: could use const instead of let here since
it's never reassigned.
```

### 4. Address Feedback (Author)

- Fix [BLOCKER] issues immediately
- Respond to questions with answers or code changes
- Consider suggestions and implement or explain why not
- Optional: implement nitpicks if time allows

### 5. Re-Review (Reviewer)

- Verify blockers are resolved
- Check that changes address feedback
- Approve if all concerns addressed

### 6. Merge

- Ensure CI/CD passes
- Use appropriate merge strategy
- Delete source branch after merge

---

## Review Sizes

### Small (< 100 lines)

- **Review time**: 15-30 minutes
- **Approval needed**: 1 reviewer
- **Turnaround**: Same day

### Medium (100-400 lines)

- **Review time**: 30-60 minutes
- **Approval needed**: 1-2 reviewers
- **Turnaround**: Within 24 hours

### Large (> 400 lines)

- **Review time**: 1-2 hours
- **Approval needed**: 2+ reviewers
- **Recommendation**: Break into smaller PRs

---

## Review Anti-Patterns

### Rubber Stamping

❌ **Problem**: Approving without actually reviewing
✅ **Solution**: Take time to understand changes

### Bike-Shedding

❌ **Problem**: Arguing about trivial details
✅ **Solution**: Focus on significant issues, defer style to linters

### Design in Review

❌ **Problem**: Major architectural feedback late in process
✅ **Solution**: Design review before implementation

### Ghost Reviewers

❌ **Problem**: Assigned but never responds
✅ **Solution**: Respond or reassign within 24 hours

### Defensive Author

❌ **Problem**: Author argues against all feedback
✅ **Solution**: Assume good intent, learn from feedback

---

## Language-Agnostic Review Commands

```bash
# Check out PR locally
$VCS_TOOL fetch origin pull/123/head:pr-123
$VCS_TOOL checkout pr-123

# Run tests
$TEST_RUNNER run

# Run linter
$LINTER check

# Check diff
$VCS_TOOL diff main...pr-123

# Check files changed
$VCS_TOOL diff --name-only main...pr-123

# Approve PR
$VCS_PLATFORM pr review 123 --approve

# Request changes
$VCS_PLATFORM pr review 123 --request-changes
```

---

## Review Templates

### Approval Comment

```markdown
## Review Summary
✅ Functionality: Code solves the problem effectively
✅ Tests: Good coverage including edge cases
✅ Security: No concerns identified
✅ Performance: Meets benchmarks
✅ Documentation: Clear and updated

**Approval**: LGTM! 🚀
```

### Request Changes Comment

```markdown
## Review Summary
⚠️ Functionality: Missing error handling in API call
⚠️ Tests: Need integration test for error case
✅ Security: No concerns
✅ Performance: Looks good
⚠️ Documentation: Update README with new endpoint

**Status**: Requesting changes. Please address the items above.
```

---

## Pair Review

For complex or critical changes:

- **Schedule**: 30-60 minute session
- **Format**: Screen share or in-person
- **Roles**: Author explains, reviewer asks questions
- **Benefits**: Faster feedback, knowledge transfer
- **When**: Complex logic, security-critical, new patterns

---

## Related Documentation

- [quality-gates.md](quality-gates.md) - Quality checkpoints
- [commits.md](commits.md) - Commit standards
- [../System/anti-patterns.md](../System/anti-patterns.md) - What to avoid
- [examples-guide.md](examples-guide.md) - Good and bad examples
- [../System/principles.md](../System/principles.md) - Core principles

---

## Review Metrics

Track these to improve process:

- Average review time
- Number of review rounds
- Time to first review
- PR size distribution
- Defects found in review vs production

**Goal**: Fast, thorough reviews that catch issues early

---

**Remember**: Code review is about improving code quality AND sharing knowledge. Be kind, be thorough, be timely.
