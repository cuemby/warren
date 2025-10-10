# Conventional Commit Standards

## Overview

Conventional commits provide a consistent format for commit messages, enabling automated tooling, clear history, and better collaboration.

## Commit Format

```text
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

### Examples

```text
feat: add user authentication
fix: resolve memory leak in cache handler
docs: update API documentation
refactor: simplify error handling logic
test: add integration tests for payment flow
```

---

## Commit Types

### Primary Types

#### feat

**Purpose**: New feature for the user
**When to use**: Adding functionality
**Example**:

```bash
$VCS_TOOL commit -m "feat: add password reset functionality"
```

#### fix

**Purpose**: Bug fix for the user
**When to use**: Fixing a defect
**Example**:

```bash
$VCS_TOOL commit -m "fix: resolve null pointer in user service"
```

#### docs

**Purpose**: Documentation changes only
**When to use**: README, code comments, documentation files
**Example**:

```bash
$VCS_TOOL commit -m "docs: update installation instructions"
```

### Secondary Types

#### refactor

**Purpose**: Code change that neither fixes a bug nor adds a feature
**When to use**: Improving code structure without changing behavior
**Example**:

```bash
$VCS_TOOL commit -m "refactor: extract validation logic to separate class"
```

#### test

**Purpose**: Adding or correcting tests
**When to use**: Test files only, no production code changes
**Example**:

```bash
$VCS_TOOL commit -m "test: add unit tests for authentication module"
```

#### chore

**Purpose**: Changes to build process or auxiliary tools
**When to use**: Dependencies, configuration, tooling
**Example**:

```bash
$VCS_TOOL commit -m "chore: update build dependencies"
```

#### style

**Purpose**: Code style changes (formatting, white-space, etc.)
**When to use**: Cosmetic changes only, no logic changes
**Example**:

```bash
$VCS_TOOL commit -m "style: format code with prettier"
```

#### perf

**Purpose**: Performance improvements
**When to use**: Optimizing code for speed or efficiency
**Example**:

```bash
$VCS_TOOL commit -m "perf: optimize database query performance"
```

#### ci

**Purpose**: CI/CD configuration changes
**When to use**: Pipeline, workflow, or automation changes
**Example**:

```bash
$VCS_TOOL commit -m "ci: add automated security scanning"
```

#### build

**Purpose**: Changes affecting the build system
**When to use**: Build scripts, compilation, packaging
**Example**:

```bash
$VCS_TOOL commit -m "build: update webpack configuration"
```

#### revert

**Purpose**: Reverting a previous commit
**When to use**: Undoing a commit
**Example**:

```bash
$VCS_TOOL commit -m "revert: undo changes to authentication flow"
```

---

## Scope (Optional)

Scope provides additional context about which part of codebase is affected.

### Format

```text
<type>(<scope>): <description>
```

### Examples

```bash
feat(auth): add OAuth2 support
fix(api): handle timeout errors
docs(readme): add setup instructions
test(payment): add integration tests
```

### Common Scopes

- Component names: `(auth)`, `(api)`, `(ui)`
- Module names: `(parser)`, `(validator)`, `(router)`
- Package names: `(core)`, `(utils)`, `(models)`

---

## Description

### Guidelines

- Use imperative mood: "add" not "added" or "adds"
- Don't capitalize first letter
- No period at the end
- Keep under 72 characters
- Be specific and descriptive

### Good Examples ✅

```bash
add user profile editing
fix race condition in cache
update API documentation
remove deprecated endpoints
```

### Bad Examples ❌

```bash
Added stuff                    # Too vague
Fix.                          # Not descriptive
Updated files                 # What files? What changed?
FIXED THE BUG!!!             # Unprofessional, capitalized
This fixes the thing that was wrong  # Too wordy
```

---

## Body (Optional)

Use body to provide additional context:

- Motivation for the change
- Contrast with previous behavior
- Side effects or breaking changes

### Format

- Separate from description with blank line
- Wrap at 72 characters
- Use bullet points for multiple items

### Example

```bash
$VCS_TOOL commit -m "$(cat <<'EOF'
feat: add caching layer for API responses

Implement Redis-based caching to improve API performance:
- Cache GET requests for 5 minutes
- Invalidate cache on POST/PUT/DELETE
- Add cache-control headers
- Reduce database load by ~60%

Resolves performance issues reported in #123
EOF
)"
```

---

## Footer (Optional)

### Breaking Changes

Prefix with `BREAKING CHANGE:` to indicate breaking changes

```bash
$VCS_TOOL commit -m "$(cat <<'EOF'
feat: update API authentication

BREAKING CHANGE: API now requires Bearer token instead of API key.
Clients must update authentication headers.
EOF
)"
```

### Issue References

Reference related issues or pull requests

```bash
$VCS_TOOL commit -m "$(cat <<'EOF'
fix: resolve memory leak

Fixes #123
Closes #456
Related to #789
EOF
)"
```

---

## AI-Assisted Commits

When using AI assistants like Claude Code:

### Standard Format

```bash
$VCS_TOOL commit -m "$(cat <<'EOF'
<type>: <description>

🤖 Generated with Claude Code

Co-Authored-By: Claude <noreply@anthropic.com>
EOF
)"
```

### Example

```bash
$VCS_TOOL commit -m "$(cat <<'EOF'
feat: implement user notification system

Added email and in-app notifications for:
- Account activity alerts
- Payment confirmations
- System updates

🤖 Generated with Claude Code

Co-Authored-By: Claude <noreply@anthropic.com>
EOF
)"
```

---

## Commit Frequency

### DO ✅

- Commit after each logical unit of work
- Commit working, tested code
- Commit frequently (multiple times per session)
- Commit before taking breaks
- Commit before switching tasks

### DON'T ❌

- Commit broken code
- Commit untested code
- Batch multiple unrelated changes
- Commit generated files (build artifacts, dependencies)
- Commit sensitive data (secrets, credentials)

---

## Git Workflow

### Basic Workflow

```bash
# Check status
$VCS_TOOL status

# Stage specific files
$VCS_TOOL add path/to/file1 path/to/file2

# Review changes
$VCS_TOOL diff --cached

# Commit with message
$VCS_TOOL commit -m "feat: add new feature"

# Push to remote
$VCS_TOOL push origin branch-name
```

### Amending Commits

Only amend commits that haven't been pushed:

```bash
# Add forgotten files
$VCS_TOOL add forgotten-file.txt

# Amend last commit
$VCS_TOOL commit --amend --no-edit

# Or change commit message
$VCS_TOOL commit --amend -m "new message"
```

⚠️ **Warning**: Never amend pushed commits on shared branches!

---

## Pre-Commit Checks

### Automated Checks

```yaml
pre-commit:
  - lint: "Ensure code style compliance"
  - format: "Auto-format code"
  - test: "Run unit tests"
  - secrets: "Scan for secrets"
  - types: "Check type safety"
```

### Manual Checks

- [ ] Tests pass locally
- [ ] No debugging code left in
- [ ] No commented-out code
- [ ] No secrets or credentials
- [ ] Documentation updated if needed

See [../System/automation.md](../System/automation.md) for git hooks setup

---

## Commit Message Templates

### Feature Commit

```bash
$VCS_TOOL commit -m "$(cat <<'EOF'
feat(<scope>): <brief description>

<detailed explanation of what was added>

- Feature detail 1
- Feature detail 2
- Feature detail 3

Closes #<issue-number>
EOF
)"
```

### Bug Fix Commit

```bash
$VCS_TOOL commit -m "$(cat <<'EOF'
fix(<scope>): <brief description>

<explanation of the bug and the fix>

Root cause: <what caused the bug>
Solution: <how it was fixed>

Fixes #<issue-number>
EOF
)"
```

### Refactoring Commit

```bash
$VCS_TOOL commit -m "$(cat <<'EOF'
refactor(<scope>): <brief description>

<explanation of what was refactored and why>

- No functional changes
- Improved code structure
- Better maintainability

Related to #<issue-number>
EOF
)"
```

---

## Commit History Best Practices

### Keep History Clean

- Use feature branches
- Rebase before merging (if team policy allows)
- Squash trivial commits
- Write clear, descriptive messages

### Review Before Push

```bash
# View commit history
$VCS_TOOL log --oneline -10

# View detailed commit
$VCS_TOOL show <commit-hash>

# View all changes in branch
$VCS_TOOL log main..feature-branch --oneline
```

---

## Common Mistakes

### ❌ Vague Messages

```bash
fix stuff
update code
changes
wip
```

### ❌ Too Much in One Commit

Mixing feature + refactor + docs + fix

### ❌ Incomplete Work

Committing non-working code or failing tests

### ❌ Generated Content

Committing build artifacts, node_modules, etc.

### ✅ Good Practices

```bash
feat(auth): add OAuth2 login support
fix(api): handle null response from payment gateway
docs(readme): add deployment instructions
test(cart): add unit tests for discount calculation
```

---

## Language-Agnostic Examples

### JavaScript/TypeScript

```bash
feat(hooks): add useLocalStorage hook
fix(api): resolve promise rejection in fetch
test(utils): add tests for date formatter
```

### Python

```bash
feat(ml): add new prediction model
fix(parser): handle edge case in JSON parser
refactor(models): simplify user model structure
```

### Go

```bash
feat(http): add middleware for request logging
fix(db): resolve connection pool leak
test(handlers): add integration tests for API
```

### Rust

```bash
feat(cli): add new command for data export
fix(parser): resolve lifetime issue in parser
perf(compute): optimize matrix multiplication
```

### Java

```bash
feat(service): add user notification service
fix(repository): resolve N+1 query issue
refactor(controller): extract validation logic
```

---

## Related Documentation

- [workflow.md](workflow.md) - Development workflow
- [code-review.md](code-review.md) - Review standards
- [../System/automation.md](../System/automation.md) - Git hooks setup
- [../System/anti-patterns.md](../System/anti-patterns.md) - What to avoid

---

## Tools and Automation

### Commit Message Validation

```bash
# Install commitlint (example for Node.js projects)
$PACKAGE_MANAGER install --save-dev @commitlint/cli @commitlint/config-conventional
```

### Commit Message Templates

```bash
# Set up commit template
$VCS_TOOL config --global commit.template ~/.gitmessage
```

### Interactive Commit Tools

- `git commit -v` - Shows diff in commit message editor
- `git add -p` - Interactively stage changes
- `$VCS_PLATFORM pr create` - Create PR with commit messages

---

**Remember**: Good commit messages help everyone understand the history and evolution of the codebase. Take the extra minute to write them well.
