# Example-Driven Development Guide

## Overview

Learning from examples is one of the most effective ways to maintain code quality and consistency. This guide explains how to use good and bad examples to inform development decisions.

## Philosophy

> "Show, don't tell. An example is worth a thousand documentation pages."

### Benefits

- **Faster onboarding**: New developers learn by example
- **Consistent patterns**: Team follows established patterns
- **Fewer mistakes**: Learn from past errors
- **Better reviews**: Reference examples in feedback
- **Living documentation**: Examples evolve with codebase

---

## Example Repository Structure

```text
project-root/
└── examples/  (or .agent/examples/)
    ├── README.md              # Index of all examples
    ├── good/                  # Best practices
    │   ├── error-handling/
    │   ├── async-patterns/
    │   ├── testing/
    │   ├── api-design/
    │   └── performance/
    ├── bad/                   # Anti-patterns
    │   ├── security-issues/
    │   ├── performance-traps/
    │   ├── code-smells/
    │   └── common-bugs/
    └── templates/             # Starter templates
        ├── components/
        ├── services/
        └── tests/
```

---

## Good Examples

### What Makes a Good Example

✅ **Characteristics**:

- Solves a real problem
- Follows best practices
- Well-documented
- Tested thoroughly
- Production-ready quality

### Good Example Template

```markdown
# [Pattern Name] - Good Example

## Context
When to use this pattern and why.

## Code
\`\`\`language
// Well-structured, production-ready code
function handleUserInput(input) {
  // Input validation
  if (!input || typeof input !== 'string') {
    throw new ValidationError('Input must be a non-empty string');
  }

  // Sanitization
  const sanitized = sanitizeInput(input);

  // Processing
  return processInput(sanitized);
}
\`\`\`

## Why This Works
- Validates input before processing
- Clear error messages
- Separation of concerns
- Easy to test

## Tests
\`\`\`language
test('handles valid input', () => {
  expect(handleUserInput('valid')).toBe('processed');
});

test('throws on invalid input', () => {
  expect(() => handleUserInput(null)).toThrow(ValidationError);
});
\`\`\`

## When to Use
- User-facing input handling
- API endpoints
- Data processing pipelines

## Related Patterns
- [Error Handling](../error-handling/)
- [Input Validation](../validation/)
```

---

## Bad Examples

### What Makes a Bad Example

❌ **Characteristics**:

- Common mistake or anti-pattern
- Explains why it's problematic
- Shows consequences
- Provides correct alternative

### Bad Example Template

```markdown
# [Anti-Pattern Name] - Bad Example

## ❌ Problematic Code
\`\`\`language
// DO NOT DO THIS
function handleUserInput(input) {
  // No validation!
  return eval(input);  // SECURITY RISK!
}
\`\`\`

## Why This is Bad
- **Security**: eval() can execute arbitrary code
- **No validation**: Accepts any input
- **No error handling**: Crashes on unexpected input
- **Unpredictable**: Behavior depends on input content

## Consequences
- Code injection vulnerabilities
- Data breaches
- System compromise
- Production incidents

## ✅ Correct Alternative
See: [Good Example - Input Handling](../good/input-handling.md)

\`\`\`language
function handleUserInput(input) {
  if (!isValid(input)) {
    throw new ValidationError('Invalid input');
  }
  return safeProcess(input);
}
\`\`\`

## How to Detect
- Look for eval(), exec(), or similar functions
- Missing input validation
- Direct user input usage

## Prevention
- Always validate input
- Never use eval()
- Use safe parsing functions
- Add security linting rules
```

---

## Example Categories

### 1. Error Handling

**Good**:

```text
examples/good/error-handling/
├── try-catch-specific.md      # Catch specific error types
├── async-error-handling.md    # Proper promise rejection handling
├── error-messages.md          # User-friendly error messages
└── error-recovery.md          # Graceful degradation
```

**Bad**:

```text
examples/bad/error-handling/
├── empty-catch.md             # catch (e) {}
├── swallowing-errors.md       # Ignoring errors
├── generic-errors.md          # throw new Error('error')
└── missing-finally.md         # Resource leaks
```

### 2. Async Patterns

**Good**:

```text
examples/good/async/
├── promise-chains.md          # Proper promise chaining
├── async-await.md             # Clean async/await usage
├── parallel-execution.md      # Promise.all for parallel
└── cancellation.md            # AbortController usage
```

**Bad**:

```text
examples/bad/async/
├── callback-hell.md           # Nested callbacks
├── promise-constructor.md     # Anti-pattern usage
├── missing-await.md           # Forgot await keyword
└── unhandled-rejection.md     # Missing catch
```

### 3. Testing

**Good**:

```text
examples/good/testing/
├── aaa-pattern.md             # Arrange-Act-Assert
├── test-factories.md          # Test data factories
├── mocking.md                 # Proper mocking
└── integration-tests.md       # API testing
```

**Bad**:

```text
examples/bad/testing/
├── testing-implementation.md  # Testing internals
├── shared-state.md            # Tests depend on each other
├── magic-values.md            # Unclear test data
└── no-assertions.md           # Tests that don't assert
```

### 4. API Design

**Good**:

```text
examples/good/api/
├── restful-endpoints.md       # RESTful conventions
├── versioning.md              # API versioning
├── pagination.md              # Cursor/offset pagination
└── error-responses.md         # Consistent error format
```

**Bad**:

```text
examples/bad/api/
├── verb-in-url.md             # /getUser instead of GET /users
├── inconsistent-naming.md     # Mixed conventions
├── no-status-codes.md         # Always 200 OK
└── exposing-internals.md      # Leaking implementation
```

### 5. Performance

**Good**:

```text
examples/good/performance/
├── caching.md                 # Effective caching strategies
├── lazy-loading.md            # Load on demand
├── batch-operations.md        # Bulk processing
└── query-optimization.md      # Efficient queries
```

**Bad**:

```text
examples/bad/performance/
├── n-plus-one.md              # Query in loop
├── premature-optimization.md  # Optimizing too early
├── memory-leaks.md            # Uncleared references
└── blocking-operations.md     # Sync in async context
```

### 6. Security

**Good**:

```text
examples/good/security/
├── input-validation.md        # Sanitize and validate
├── authentication.md          # Secure auth patterns
├── secrets-management.md      # Environment variables
└── sql-queries.md             # Parameterized queries
```

**Bad**:

```text
examples/bad/security/
├── sql-injection.md           # String concatenation
├── xss-vulnerabilities.md     # Unescaped output
├── hardcoded-secrets.md       # Secrets in code
└── weak-passwords.md          # Poor password handling
```

---

## Using Examples in Development

### Before Implementation

```yaml
checklist:
  - [ ] Check examples/good/ for similar patterns
  - [ ] Review examples/bad/ for pitfalls
  - [ ] Use examples/templates/ as starting point
  - [ ] Validate approach matches good examples
```

### During Code Review

```markdown
## Code Review Comment

This looks like the anti-pattern in examples/bad/async/callback-hell.md

Consider refactoring to match examples/good/async/async-await.md instead:
[link to example]
```

### When Writing Tests

```markdown
Refer to examples/good/testing/aaa-pattern.md for test structure.

Your test should follow:
1. Arrange - Set up test data
2. Act - Execute the code
3. Assert - Verify the outcome
```

---

## Creating New Examples

### When to Add Examples

- Solved a tricky problem
- Found a better pattern
- Encountered repeated issue
- Code review discussions
- Onboarding feedback

### Example Creation Process

1. Identify the pattern or anti-pattern
2. Create minimal, focused example
3. Add clear explanation
4. Include tests (for good examples)
5. Add cross-references
6. Get team review

### Example Quality Checklist

- [ ] Code is minimal but complete
- [ ] Explanation is clear
- [ ] Context is provided
- [ ] Why/when is explained
- [ ] Tests included (if applicable)
- [ ] Related examples linked
- [ ] Language-agnostic where possible

---

## Language-Agnostic Examples

### Structure for Multi-Language Support

```text
examples/good/error-handling/
├── README.md                  # Concept explanation
├── javascript.md              # JS implementation
├── python.md                  # Python implementation
├── go.md                      # Go implementation
├── rust.md                    # Rust implementation
└── java.md                    # Java implementation
```

### Language-Agnostic Concepts

Focus on principles, not syntax:

- Error handling strategies
- Design patterns
- Architecture patterns
- Testing approaches
- Security principles

---

## Templates

### Component Template Example

```text
examples/templates/component-template.md:

# Component Template

## Structure
\`\`\`
ComponentName/
├── index.[ext]           # Main component
├── [name].test.[ext]     # Tests
├── [name].styles.[ext]   # Styles (if separate)
└── README.md             # Documentation
\`\`\`

## Implementation Pattern
[Code template with TODO comments]

## Required Tests
- [ ] Renders correctly
- [ ] Handles props
- [ ] Handles user interaction
- [ ] Handles error states

## Documentation Requirements
- Purpose and usage
- Props/parameters
- Examples
```

---

## Maintaining Examples

### Regular Review

- **Monthly**: Review for relevance
- **Quarterly**: Update with new patterns
- **Yearly**: Archive obsolete examples

### Keeping Examples Current

- Update when patterns change
- Add new examples as needed
- Archive outdated examples
- Cross-reference with real code

### Example Ownership

- Assign maintainers to categories
- Include in PR templates
- Review in team meetings

---

## Integration with Development

### IDE Integration

- Link examples in hover text
- Quick access from context menu
- Example snippets

### Documentation Links

```markdown
For error handling patterns, see examples/good/error-handling/
```

### Linting Rules

```yaml
# Add custom rules referencing examples
rules:
  custom/no-callback-hell:
    error: "Avoid callback hell. See examples/bad/async/callback-hell.md"
```

---

## Related Documentation

- [code-review.md](code-review.md) - Use examples in reviews
- [testing.md](testing.md) - Test examples
- [../System/anti-patterns.md](../System/anti-patterns.md) - System-level anti-patterns
- [../System/principles.md](../System/principles.md) - Core principles

---

**Remember**: Examples are living documentation. Keep them updated, relevant, and accessible. A good example can prevent hours of debugging and discussion.
