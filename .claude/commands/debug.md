---
description: Systematic debugging workflow with root cause analysis
tags: [debugging, troubleshooting, errors]
version: 1.0.0
---

# debug

## Context-Driven Debugging

Before diagnosing any code issue:

1. **Read `.agent/README.md`** to understand the current system state
2. **Check `.agent/System/`** for architecture, tech stack, and integration points that may relate to the error
3. **Review `.agent/SOP/`** for debugging best practices and known issue patterns
4. **Reference `.agent/Tasks/`** for recent changes that might have introduced the issue

## Diagnostic Process

Analyze [piece of code] causing [error] following this systematic workflow:

### 1. Error Analysis Phase

- Capture complete error message, stack trace, and reproduction steps
- Identify error type (syntax, runtime, logic, performance, integration)
- Determine scope of impact (local, module, system-wide)
- Check `.agent/System/` for related components and dependencies

### 2. Root Cause Investigation

- Step-by-step code walkthrough from error point backwards
- Trace data flow and state changes
- Identify logical mistakes, edge cases, or incorrect assumptions
- Review recent changes in `.agent/Tasks/` that might be related
- Check `.agent/SOP/` for similar known issues

### 3. Solution Design

- Propose minimal fix that addresses root cause
- Consider side effects and backward compatibility
- Verify solution against system architecture in `.agent/System/`
- Identify if this reveals a pattern worth documenting in `.agent/SOP/`

### 4. Implementation Phase

- Apply fix with minimal code changes
- Follow relevant debugging SOPs from `.agent/SOP/`
- Adhere to project's tech stack and patterns
- Add defensive code or validation if needed

### 5. Validation Phase

- Test fix against original error scenario
- Test edge cases and regression scenarios
- Verify no new issues introduced
- Validate against acceptance criteria

### 6. Prevention & Documentation

- Update `.agent/SOP/` with debugging pattern if applicable
- Document root cause in `.agent/Tasks/[feature-name].md` if part of ongoing work
- Add error handling pattern to `.agent/SOP/` if it's a common issue
- Update `.agent/System/` if the fix reveals architectural insights

### 7. Commit Phase

- Commit with clear fix description
- Use conventional commit format: `fix: resolve [brief description]`
- Reference error/issue number if applicable

## Analysis Output Format

Provide:

1. **Root Cause**: Clear explanation of what caused the error
2. **Code Walkthrough**: Step-by-step analysis of problematic code
3. **Proposed Fix**: Minimal code changes to resolve the issue
4. **Improvements**: Optional optimizations for performance, readability, or maintainability
5. **Prevention**: Suggestions to avoid similar issues (document in `.agent/SOP/` if pattern)

## Key Principles

- **Minimal Fix First**: Solve the immediate problem with least code change
- **Context Awareness**: Reference system architecture and existing patterns
- **Document Patterns**: Capture recurring issues in `.agent/SOP/`
- **Prevent Recurrence**: Add validation, tests, or documentation to prevent similar errors
- **Think Simply**: Complex fixes often indicate misunderstanding—seek simpler solutions

## Success Criteria

- [ ] Root cause identified and documented
- [ ] Fix applied with minimal code changes
- [ ] Original error no longer reproducible
- [ ] No regressions introduced
- [ ] Tests added to prevent recurrence
- [ ] Pattern documented in `.agent/SOP/` if applicable
- [ ] Changes committed with clear fix description

## Troubleshooting

### Common Debugging Mistakes

- **Treating symptoms, not root cause**: Always trace back to the actual source
- **Making too many changes at once**: Apply fixes incrementally
- **Not testing thoroughly**: Verify fix works and doesn't break anything else
- **Skipping documentation**: Future you (or teammates) will face this again

### When Stuck

1. Take a break and return with fresh eyes
2. Explain the problem out loud (rubber duck debugging)
3. Check `.agent/SOP/` for similar patterns
4. Review recent changes in `.agent/Tasks/`
5. Search for similar errors in documentation/issues

## Related Commands

- `/test` - Add tests to prevent regression
- `/code` - Implement the fix properly
- `/review` - Review the fix for quality and security
- `/explain` - Understand complex code causing the issue
