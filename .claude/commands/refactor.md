---
description: Safe refactoring workflow with test coverage requirements
tags: [refactoring, code-quality, improvement]
version: 1.0.0
---

# refactor

## Context-Driven Code Refactoring

Before refactoring any code:

1. **Read `.agent/README.md`** to understand the current system state
2. **Check `.agent/System/`** for architecture patterns and tech stack constraints
3. **Review `.agent/SOP/`** for project-specific refactoring standards
4. **Reference `.agent/Tasks/`** to ensure refactoring aligns with feature goals

## Refactoring Process

Refactor [piece of code] to improve quality while preserving functionality, following this workflow:

### 1. Analysis Phase

- Understand current functionality completely
- Document existing behavior and edge cases
- Identify pain points: complexity, duplication, performance issues
- Check `.agent/System/` for architectural context
- Review `.agent/SOP/` for established patterns to follow

### 2. Testing Baseline Phase

- Write or verify existing tests cover current functionality
- Document all inputs, outputs, and edge cases
- Establish performance baseline if relevant
- Ensure tests pass before any changes
- **[CRITICAL]**: Never refactor without test coverage

### 3. Refactoring Goals Phase

- **Primary Goals**: What to improve (efficiency, readability, maintainability)
- **Constraints**: What must stay the same (API, behavior, performance)
- **Scope**: Minimal changes vs. comprehensive restructure
- **Patterns**: Reference `.agent/SOP/` for preferred patterns

### 4. Refactoring Strategy

- Break down into small, incremental changes
- Prioritize changes: [CRITICAL] → [REQUIRED] → [RECOMMENDED] → [OPTIONAL]
- Plan order of refactoring steps
- Identify risks and mitigation strategies

### 5. Implementation Phase

Apply refactoring techniques systematically:

#### Code Simplification

- Remove redundant code and dead code paths
- Simplify complex conditionals
- Extract nested logic into functions
- Reduce cognitive complexity

#### Structure Improvement

- Apply design patterns from `.agent/SOP/`
- Improve separation of concerns
- Enhance modularity and reusability
- Follow project architecture from `.agent/System/`

#### Performance Optimization

- Optimize algorithms (improve time/space complexity)
- Remove unnecessary operations
- Improve data structure usage
- Add caching where appropriate

#### Readability Enhancement

- Improve naming (variables, functions, classes)
- Add clear comments for complex logic
- Format consistently with project standards
- Make intent explicit

#### Best Practices Application

- Apply patterns from `.agent/SOP/`
- Follow tech stack conventions from `.agent/System/`
- Improve error handling
- Enhance type safety where applicable

### 6. Validation Phase

- Run all existing tests (must pass)
- Add new tests for edge cases discovered
- Verify performance hasn't regressed
- Test in expected scenarios and environments
- Compare against baseline established in Phase 2

### 7. Documentation Phase

- Document significant changes
- Update inline comments
- Add refactoring rationale to `.agent/Tasks/` if part of feature work
- Update `.agent/SOP/` if new patterns introduced
- Update `.agent/System/` if architectural changes made

### 8. Commit Phase

- Commit incremental changes frequently
- Use conventional commit format: `refactor: [brief description]`
- Keep commits atomic and focused
- Reference related tasks or issues

## Refactoring Output Format

Provide structured refactoring plan and results:

### 📋 Refactoring Summary

- **Current Issues**: What problems exist
- **Refactoring Goals**: What we're improving
- **Approach**: High-level strategy
- **Risk Level**: Low | Medium | High

### 🔍 Current State Analysis

- **Complexity**: Cyclomatic complexity, lines of code
- **Issues Identified**: Performance, readability, maintainability problems
- **Technical Debt**: What needs fixing
- **Test Coverage**: Current state

### 🎯 Refactoring Plan

#### Phase 1: [Description]

- Changes to make
- Expected improvements
- Risk mitigation

#### Phase 2: [Description]

- Next set of changes
- Dependencies on Phase 1

### 🔧 Refactoring Changes

#### Before

```[language]
// Original code
```

#### After

```[language]
// Refactored code
```

#### What Changed

- Specific improvements made
- Why these changes improve the code
- Pattern references from `.agent/SOP/`

### ✅ Improvements Achieved

- **Efficiency**: Performance gains (with metrics if applicable)
- **Readability**: Clarity improvements
- **Maintainability**: Structure enhancements
- **Complexity Reduction**: Before/after complexity comparison

### 🧪 Testing Results

- [ ] All existing tests pass
- [ ] New tests added for edge cases
- [ ] Performance baseline maintained or improved
- [ ] Behavior unchanged (functionality preserved)

### 📚 Documentation Updates Needed

- [ ] Update `.agent/System/` if architectural changes
- [ ] Add pattern to `.agent/SOP/` if reusable technique
- [ ] Update `.agent/Tasks/` if part of feature work
- [ ] Inline comments added for complex logic

## Refactoring Principles

- **Preserve Functionality**: Never change behavior while refactoring
- **Test-Driven**: Always have test coverage before refactoring
- **Incremental Changes**: Small, safe steps over big rewrites
- **Simplicity First**: Simpler is always better than clever
- **Context-Aware**: Follow patterns from `.agent/SOP/`
- **Measurable**: Track improvements objectively
- **Reversible**: Keep changes atomic for easy rollback
- **Pattern Recognition**: Document reusable patterns in `.agent/SOP/`

## Refactoring Techniques Checklist

### Code Simplification

- [ ] Extract method/function
- [ ] Inline unnecessary abstractions
- [ ] Remove dead code
- [ ] Simplify conditionals
- [ ] Reduce nesting depth

### Structure Improvement

- [ ] Separate concerns
- [ ] Apply design patterns from `.agent/SOP/`
- [ ] Improve naming
- [ ] Enhance modularity
- [ ] Follow project architecture

### Performance Optimization

- [ ] Optimize algorithms
- [ ] Improve data structures
- [ ] Remove redundant operations
- [ ] Add appropriate caching
- [ ] Reduce memory footprint

### Maintainability

- [ ] Improve readability
- [ ] Add documentation
- [ ] Enhance error handling
- [ ] Increase type safety
- [ ] Make intent explicit

## Risk Management

### Low Risk Refactoring

- Renaming variables/functions
- Extracting methods
- Simplifying expressions
- Removing dead code

### Medium Risk Refactoring

- Changing data structures
- Restructuring class hierarchies
- Modifying algorithms
- Updating dependencies

### High Risk Refactoring

- Architectural changes
- API modifications
- Performance-critical code changes
- Multi-system integrations

**[CRITICAL]**: For medium/high risk refactoring:

- Get approval before proceeding
- Implement feature flags if possible
- Plan rollback strategy
- Test exhaustively

## Refactoring Anti-Patterns to Avoid

- ❌ Refactoring without tests
- ❌ Changing functionality and refactoring simultaneously
- ❌ Over-engineering solutions
- ❌ Ignoring existing patterns from `.agent/SOP/`
- ❌ Making changes that break compatibility
- ❌ Refactoring for the sake of refactoring
- ❌ Large, monolithic refactoring commits

## Success Criteria

- [ ] Functionality preserved (all tests pass)
- [ ] Code quality improved (measurable metrics)
- [ ] Follows patterns from `.agent/SOP/`
- [ ] Aligns with architecture in `.agent/System/`
- [ ] No performance regression
- [ ] Documentation updated
- [ ] Code review approved
- [ ] Committed with clear messages
