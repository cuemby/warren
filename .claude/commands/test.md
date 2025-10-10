---
description: Comprehensive test development with coverage and quality checks
tags: [testing, unit-tests, integration-tests, e2e, quality]
version: 1.0.0
---

# test

## Context-Driven Test Development

Before writing any tests:

1. **Read `.agent/README.md`** to understand the current system state
2. **Check `.agent/System/`** for tech stack, testing framework, and test architecture
3. **Review `.agent/SOP/`** for testing standards, patterns, and best practices
4. **Reference `.agent/Tasks/`** for acceptance criteria and feature requirements

## Test Development Process

Write comprehensive tests for [piece of code] using [testing framework] following this workflow:

### 1. Code Analysis Phase

- Understand the code's functionality completely
- Identify all inputs, outputs, and side effects
- Map dependencies and integration points (reference `.agent/System/`)
- Review acceptance criteria from `.agent/Tasks/`
- Check existing test coverage

### 2. Testing Strategy Phase

- **Test Levels**: Determine unit, integration, and e2e test needs
- **Test Coverage Goals**: Reference standards from `.agent/SOP/`
- **Critical Paths**: Identify must-test scenarios
- **Edge Cases**: List boundary conditions and error scenarios
- **Risk Areas**: Focus on bug-prone or critical code sections

### 3. Test Planning Phase

#### Test Types Needed

- [ ] **Unit Tests**: Individual functions/methods in isolation
- [ ] **Integration Tests**: Component interactions
- [ ] **E2E Tests**: Full user workflows
- [ ] **Performance Tests**: Load, stress, benchmark tests
- [ ] **Security Tests**: Vulnerability and authorization tests

#### Test Scenarios

- **Positive Tests**: Valid inputs and expected behavior
- **Negative Tests**: Invalid inputs and error handling
- **Edge Cases**: Boundary conditions and limits
- **Error Scenarios**: Failure modes and recovery
- **Integration**: Cross-component interactions

### 4. Test Environment Setup

- Configure testing framework per `.agent/System/`
- Set up test fixtures and mocks
- Prepare test data
- Configure test environment variables
- Set up CI/CD test pipeline

### 5. Test Implementation Phase

#### Unit Tests

Write isolated tests for each function/method using the testing framework from `.agent/System/`:

**Structure** (adapt to your language/framework):

- Setup phase (before each test)
- Test case with descriptive name
- AAA pattern: Arrange, Act, Assert
- Cleanup phase (after each test)

**Test Pattern**:

1. Arrange: Set up test data and dependencies
2. Act: Execute the function/method under test
3. Assert: Verify expected outcomes

**Coverage**:

- Happy path (valid inputs, expected behavior)
- Edge cases (boundaries, empty values, null/undefined)
- Error scenarios (invalid inputs, exceptions)

#### Integration Tests

Test component interactions using patterns from `.agent/SOP/`:

**Focus Areas**:

- Cross-component data flow
- API contract validation
- Database interactions
- External service integration
- Error propagation between components

#### E2E Tests

Test complete user workflows following scenarios from `.agent/Tasks/`:

**Characteristics**:

- End-to-end user journey
- Real or realistic test environment
- Full system integration
- Minimal mocking

### 6. Test Coverage Phase

- Run coverage analysis
- Identify untested paths
- Achieve coverage goals from `.agent/SOP/`
- Focus on critical paths first

### 7. Test Validation Phase

- Verify all tests pass
- Ensure tests are deterministic (no flaky tests)
- Validate test performance (execution time)
- Check for proper isolation (tests don't affect each other)
- Confirm tests fail when they should

### 8. Documentation Phase

- Document test strategy and patterns
- Update `.agent/SOP/` with new testing patterns
- Add testing notes to `.agent/Tasks/` if feature-specific
- Include inline comments for complex test logic

### 9. Commit Phase

- Commit tests with code changes
- Use conventional commit format: `test: add tests for [component]`
- Reference related feature or bug fix

## Test Output Format

### 📋 Test Strategy Summary

#### Component Under Test

- **Name**: [Component/Module]
- **Location**: File path
- **Functionality**: Brief description
- **Dependencies**: What it relies on

#### Test Approach

- **Framework**: [Testing framework] (from `.agent/System/`)
- **Test Levels**: Unit, Integration, E2E
- **Coverage Goal**: Target percentage (from `.agent/SOP/`)
- **Critical Areas**: High-priority test scenarios

---

### 🎯 Test Scenarios

#### Positive Test Cases (Happy Path)

1. **Scenario**: [Description]
   - **Input**: [Test data]
   - **Expected Output**: [Result]
   - **Rationale**: Why this matters

#### Negative Test Cases (Error Handling)

1. **Scenario**: [Invalid input handling]
   - **Input**: [Bad data]
   - **Expected**: [Error or exception]
   - **Rationale**: [Why test this]

#### Edge Cases

1. **Scenario**: [Boundary condition]
   - **Input**: [Edge case data]
   - **Expected**: [Behavior]
   - **Rationale**: [Why this matters]

#### Integration Scenarios

1. **Scenario**: [Cross-component test]
   - **Components**: [A, B, C]
   - **Flow**: [Interaction description]
   - **Expected**: [Result]

---

### 🧪 Test Implementation

Use the testing framework and patterns defined in `.agent/System/` and `.agent/SOP/`.

#### Unit Tests Example Structure

**Test Organization**:

- Group related tests together (e.g., by component, module, or functionality)
- Use descriptive test names that explain the scenario
- Follow AAA pattern: Arrange, Act, Assert

**Sample Unit Test Structure** (adapt to your framework):

```text
Test Suite: [Component/Function Name]
  ├── Test: should [expected behavior] when [valid condition]
  │   ├── Arrange: Set up valid test data
  │   ├── Act: Execute function under test
  │   └── Assert: Verify expected outcome
  │
  ├── Test: should handle [edge case] correctly
  │   ├── Arrange: Set up edge case data
  │   ├── Act: Execute function
  │   └── Assert: Verify edge case handling
  │
  └── Test: should throw [error] when [invalid condition]
      ├── Arrange: Set up invalid test data
      ├── Act & Assert: Verify error is thrown
```

#### Integration Tests Example Structure

**Test Organization**:

- Test interaction between components
- Verify data flow across boundaries
- Validate API contracts

**Sample Integration Test Structure**:

```text
Test Suite: [Integration Scenario]
  └── Test: should integrate [Component A] with [Component B]
      ├── Setup: Initialize both components
      ├── Act: Pass data through component chain
      └── Assert: Verify end-to-end data transformation
```

#### E2E Tests Example Structure

**Test Organization**:

- Test complete user workflows
- Use realistic test environment
- Minimize mocking

**Sample E2E Test Structure**:

```text
Test Suite: [User Workflow]
  └── Test: should complete [end-to-end scenario]
      ├── Setup: Prepare test environment
      ├── Act: Execute user journey steps
      ├── Assert: Verify final state
      └── Cleanup: Tear down test environment
```

---

### 📊 Test Coverage Report

#### Coverage Metrics

- **Lines**: X% (Goal: Y% from `.agent/SOP/`)
- **Branches**: X% (Goal: Y%)
- **Functions**: X% (Goal: Y%)
- **Statements**: X% (Goal: Y%)

#### Uncovered Areas

- [ ] [Function/path not covered] - Reason/plan
- [ ] [Edge case not tested] - Reason/plan

#### Critical Path Coverage

- [x] Main functionality: 100%
- [x] Error handling: 95%
- [x] Edge cases: 90%
- [ ] Integration points: 80% (plan to improve)

---

### ✅ Test Quality Checklist

#### Test Characteristics

- [ ] **Independent**: Tests don't depend on each other
- [ ] **Repeatable**: Same results every time
- [ ] **Fast**: Execute quickly (< 1s for unit tests)
- [ ] **Isolated**: No external dependencies
- [ ] **Clear**: Easy to understand what's being tested

#### AAA Pattern (Arrange-Act-Assert)

- [ ] Clear separation of test phases
- [ ] Setup (Arrange) is explicit
- [ ] Single action (Act) per test
- [ ] Assertions (Assert) are specific

#### Test Naming

- [ ] Descriptive: What's being tested
- [ ] Scenario: Under what conditions
- [ ] Expected: What should happen
- [ ] Format: `should [expected] when [condition]`

---

### 🔧 Test Fixtures & Helpers

Use patterns from `.agent/SOP/` for test data and helpers.

#### Test Data

Create reusable test data fixtures:

- **Valid data**: Represents happy path scenarios
- **Invalid data**: Used for negative testing
- **Edge case data**: Boundary conditions and special cases
- **Mock data**: Simulates external dependencies

#### Mock Objects

Create mocks for external dependencies:

- Database clients
- API services
- File system operations
- Third-party integrations

#### Test Helpers

Create reusable helper functions:

- Environment setup and teardown
- Common test data generators
- Assertion helpers
- Test utilities

---

### 🐛 Testing Edge Cases & Error Scenarios

#### Boundary Conditions

- Minimum values (0, empty, null)
- Maximum values (limits, overflow)
- Off-by-one errors
- Empty collections
- Single-item collections

#### Error Scenarios

- Network failures
- Database connection errors
- Invalid authentication
- Timeout scenarios
- Resource exhaustion

#### Concurrency Issues

- Race conditions
- Deadlocks
- Thread safety

---

### 📚 Related Testing Documentation

#### Internal Documentation

- [`.agent/System/testing.md`](.agent/System/testing.md) - Test architecture
- [`.agent/SOP/testing.md`](.agent/SOP/testing.md) - Testing standards and patterns
- [`.agent/Tasks/[feature].md`](.agent/Tasks/feature.md) - Acceptance criteria

#### Testing Framework Docs

- [Framework documentation link]
- [Best practices guide]
- [Common patterns]

---

## Testing Principles

- **Test-Driven Development**: Write tests before or with code
- **Comprehensive Coverage**: Test all critical paths
- **Isolation**: Tests should be independent
- **Clarity**: Tests are documentation
- **Fast Feedback**: Tests should run quickly
- **Reliable**: No flaky tests
- **Maintainable**: Easy to update as code changes
- **Pattern-Driven**: Follow patterns from `.agent/SOP/`

## Testing Best Practices from `.agent/SOP/`

### Unit Test Best Practices

- One assertion per test (when possible)
- Test behavior, not implementation
- Use meaningful test names
- Keep tests simple and readable
- Mock external dependencies
- Test edge cases and errors

### Integration Test Best Practices

- Test real interactions
- Use test databases/services
- Clean up after each test
- Test failure scenarios
- Verify data flow

### E2E Test Best Practices

- Test critical user journeys
- Keep E2E tests minimal (they're slow)
- Use page objects or similar patterns
- Handle async operations properly
- Make tests resilient to timing issues

### Performance Test Best Practices

- Establish baselines
- Test under realistic load
- Monitor resource usage
- Test scalability limits
- Document performance requirements

## Testing Anti-Patterns to Avoid

- ❌ Testing implementation details instead of behavior
- ❌ Flaky tests (non-deterministic)
- ❌ Tests that depend on execution order
- ❌ Overly complex test setup
- ❌ Testing framework code or libraries
- ❌ Ignoring edge cases and errors
- ❌ Tests that take too long to run
- ❌ Unclear test names
- ❌ Multiple unrelated assertions in one test

## Test Maintenance

### When to Update Tests

- Code behavior changes
- Bug fixes
- Refactoring
- New features added
- Edge cases discovered

### Keeping Tests Healthy

- Run tests regularly (CI/CD)
- Fix failing tests immediately
- Remove obsolete tests
- Update test data as needed
- Maintain test performance

### Documentation Updates

- Update `.agent/SOP/` with new test patterns
- Document complex test scenarios
- Keep test strategy current in `.agent/System/`

## CI/CD Integration

### Pre-Commit Hooks

Run tests before allowing commits using the test command from `.agent/System/`:

```bash
# Run test command defined in .agent/System/
[test_command]
```

### CI Pipeline Stages

1. **Lint & Format**: Code quality checks
2. **Unit Tests**: Fast, isolated tests
3. **Integration Tests**: Component interactions
4. **E2E Tests**: Full workflows
5. **Coverage Report**: Generate coverage metrics
6. **Performance Tests**: Benchmark critical paths

### Quality Gates

- Minimum coverage threshold (from `.agent/SOP/`)
- All tests must pass
- No critical security issues
- Performance benchmarks met

## Success Criteria

- [ ] All critical paths tested
- [ ] Coverage goals met (per `.agent/SOP/`)
- [ ] Edge cases covered
- [ ] Error scenarios tested
- [ ] Tests pass consistently
- [ ] Tests run fast (unit tests < 1s)
- [ ] Clear, descriptive test names
- [ ] Documentation updated
- [ ] CI/CD integration working
- [ ] Code review approved
