---
description: Context-driven code implementation with .agent integration
tags: [development, implementation, code]
version: 1.0.0
---

# code

## Context-Driven Development

Before writing any code:

1. **Read `.agent/README.md`** to understand the current system state
2. **Check `.agent/System/`** for relevant architecture, tech stack, and integration points
3. **Review `.agent/SOP/`** for best practices specific to this task type
4. **Reference `.agent/Tasks/`** for related PRD and implementation plans

## Implementation Process

Write code in [programming language] to [perform action] following this workflow:

### 1. Planning Phase

- Create/update task plan in `.agent/Tasks/[feature-name].md`
- Break down into simple, atomic changes
- Identify files to modify (prefer editing over creating new files)
- List acceptance criteria

### 2. Implementation Phase

- Follow relevant SOPs from `.agent/SOP/`
- Keep changes minimal and focused
- Adhere to project's tech stack and patterns documented in `.agent/System/`
- Write efficient, well-structured code optimized for performance
- Follow best practices and industry standards

### 3. Testing Phase

- Test thoroughly to ensure functionality meets requirements
- Verify against acceptance criteria from task plan

### 4. Documentation Phase

- Update `.agent/System/` docs to reflect new system state
- Update or create relevant SOP in `.agent/SOP/` if introducing new patterns
- Update `.agent/Tasks/[feature-name].md` with implementation summary
- Update `.agent/README.md` index if adding new documentation

### 5. Commit Phase

- Commit frequently after each logical unit of work
- Use conventional commit format (e.g., `feat:`, `fix:`, `docs:`)

## Key Principles

- **Simplicity First**: Impact as little code as possible
- **Context Awareness**: Always reference existing patterns and documentation
- **Incremental Updates**: Keep `.agent/` docs synchronized with code changes
- **Documentation as Code**: Treat `.agent/` docs as living documentation

## Success Criteria

- [ ] Code implements required functionality
- [ ] All tests pass (unit, integration, e2e)
- [ ] Code follows patterns from `.agent/SOP/`
- [ ] Documentation updated in `.agent/System/` and `.agent/Tasks/`
- [ ] Changes committed with conventional commit format
- [ ] Code reviewed (if team workflow requires it)
- [ ] Acceptance criteria from task plan met

## Related Commands

- `/test` - Write comprehensive tests for your implementation
- `/review` - Review code quality and security
- `/refactor` - Improve code structure while maintaining functionality
- `/document` - Create comprehensive documentation
- `/debug` - Troubleshoot issues in implementation
