# Testing Standards

## Overview

Testing is non-negotiable. No untested code reaches production. This document defines testing standards, patterns, and best practices.

## Testing Pyramid

```text
        ╱╲
       ╱  ╲      E2E Tests (10%)
      ╱────╲     < 30 seconds
     ╱      ╲
    ╱────────╲   Integration Tests (20%)
   ╱          ╲  < 5 seconds
  ╱────────────╲
 ╱              ╲ Unit Tests (70%)
╱────────────────╲ < 100ms
```

### Distribution Guidelines

- **Unit Tests (70%)**: Fast, isolated, test single units
- **Integration Tests (20%)**: Test component interactions
- **E2E Tests (10%)**: Test complete user workflows

---

## Test Principles

### AAA Pattern

All tests should follow Arrange-Act-Assert:

```text
// Arrange - Set up test conditions
const user = createTestUser();

// Act - Execute the behavior
const result = authenticateUser(user);

// Assert - Verify the outcome
expect(result.isAuthenticated).toBe(true);
```

### FIRST Principles

- **Fast**: Tests run quickly
- **Independent**: Tests don't depend on each other
- **Repeatable**: Same result every time
- **Self-validating**: Pass or fail, no manual verification
- **Timely**: Written with or before production code

---

## Unit Tests [CRITICAL]

### What to Test

- ✅ Business logic
- ✅ Edge cases
- ✅ Error handling
- ✅ Input validation
- ✅ Utility functions
- ✅ Pure functions

### What NOT to Test

- ❌ Framework code
- ❌ Third-party libraries
- ❌ Getters/setters without logic
- ❌ Configuration files

### Best Practices

#### Test Structure

```text
describe('ComponentName', () => {
  describe('methodName', () => {
    it('should handle valid input', () => {
      // Arrange
      const input = createValidInput();

      // Act
      const result = methodName(input);

      // Assert
      expect(result).toBe(expected);
    });

    it('should throw error for invalid input', () => {
      // Arrange
      const input = createInvalidInput();

      // Act & Assert
      expect(() => methodName(input)).toThrow(ErrorType);
    });
  });
});
```

#### Naming Convention

```text
it('should [expected behavior] when [condition]')

Examples:
it('should return user when valid ID provided')
it('should throw error when user not found')
it('should cache result after first call')
```

#### Test Data

```text
// ✅ Good - Descriptive test data
const validUser = { id: 1, name: 'John', email: 'john@example.com' };
const userWithoutEmail = { id: 2, name: 'Jane' };

// ❌ Bad - Magic values
const user = { id: 1, name: 'a', email: 'b' };
```

### Language-Agnostic Examples

#### JavaScript/TypeScript

```javascript
describe('Calculator', () => {
  it('should add two numbers correctly', () => {
    const calc = new Calculator();
    expect(calc.add(2, 3)).toBe(5);
  });
});
```

#### Python

```python
def test_calculator_add():
    """Should add two numbers correctly"""
    calc = Calculator()
    assert calc.add(2, 3) == 5
```

#### Go

```go
func TestCalculatorAdd(t *testing.T) {
    calc := NewCalculator()
    result := calc.Add(2, 3)
    if result != 5 {
        t.Errorf("Expected 5, got %d", result)
    }
}
```

#### Rust

```rust
#[test]
fn test_calculator_add() {
    let calc = Calculator::new();
    assert_eq!(calc.add(2, 3), 5);
}
```

#### Java

```java
@Test
public void shouldAddTwoNumbersCorrectly() {
    Calculator calc = new Calculator();
    assertEquals(5, calc.add(2, 3));
}
```

---

## Integration Tests [REQUIRED]

### What to Test

- Database interactions
- API endpoints
- External service integrations
- Component interactions
- Configuration loading

### Best Practices

- Use test databases (never production!)
- Clean up after each test
- Use test containers when possible
- Mock external dependencies
- Test realistic scenarios

### Example Structure

```text
describe('User API Integration', () => {
  beforeEach(async () => {
    await setupTestDatabase();
  });

  afterEach(async () => {
    await cleanupTestDatabase();
  });

  it('should create user and return 201', async () => {
    const response = await api.post('/users', validUserData);
    expect(response.status).toBe(201);
    expect(response.body).toHaveProperty('id');
  });
});
```

---

## E2E Tests [REQUIRED]

### What to Test

- Critical user workflows
- Authentication flows
- Payment processes
- Core business features

### Best Practices

- Test real user scenarios
- Keep number of E2E tests low
- Run in isolated environments
- Use page object pattern
- Make tests resilient to UI changes

### Example

```text
test('User can complete checkout process', async () => {
  await loginPage.login(testUser);
  await productsPage.addToCart(product);
  await cartPage.proceedToCheckout();
  await checkoutPage.enterPaymentDetails(testCard);
  await checkoutPage.confirmOrder();

  expect(await confirmationPage.getOrderNumber()).toBeTruthy();
});
```

---

## Test-Driven Development (TDD)

### Red-Green-Refactor Cycle

```text
1. 🔴 RED: Write failing test
2. 🟢 GREEN: Write minimal code to pass
3. 🔵 REFACTOR: Improve code quality
```

### When to Use TDD

- ✅ Complex business logic
- ✅ Bug fixes (write test first)
- ✅ New features with clear requirements
- ✅ Refactoring existing code

### Example TDD Workflow

```bash
# 1. Write failing test
$TEST_RUNNER run
# Test fails ❌

# 2. Write minimal implementation
# Edit code...

$TEST_RUNNER run
# Test passes ✅

# 3. Refactor
# Improve code structure

$TEST_RUNNER run
# Tests still pass ✅

# 4. Commit
$VCS_TOOL commit -m "feat: implement user validation"
```

---

## Test Coverage

### Targets by Profile

| Profile | Minimum Coverage | Target Coverage |
|---------|-----------------|-----------------|
| MVP/Prototype | 40% | 50% |
| Startup | 60% | 70% |
| Enterprise | 80% | 85% |
| Open Source | 70% | 80% |

### Coverage Types

- **Line Coverage**: Percentage of lines executed
- **Branch Coverage**: Percentage of branches taken
- **Function Coverage**: Percentage of functions called

### Running Coverage

```bash
# Generate coverage report
$TEST_RUNNER run --coverage

# View coverage report
# Check coverage/index.html or coverage output
```

### What Coverage Doesn't Tell You

- ❌ Quality of tests
- ❌ Edge cases covered
- ❌ Meaningful assertions
- ✅ Only which code was executed

---

## Test Organization

### File Structure

```text
project/
├── src/
│   ├── user/
│   │   ├── user.service.js
│   │   ├── user.service.test.js      # Unit tests
│   │   └── user.service.integration.test.js  # Integration tests
├── tests/
│   ├── unit/                          # Additional unit tests
│   ├── integration/                   # Integration tests
│   └── e2e/                          # E2E tests
```

### Naming Conventions

- Unit: `*.test.*` or `*.spec.*`
- Integration: `*.integration.test.*`
- E2E: `*.e2e.test.*`

---

## Mocking and Stubbing

### When to Mock

- External APIs
- Databases (in unit tests)
- File system operations
- Time-dependent operations
- Slow operations

### Mocking Strategies

```text
// Test double types:
- Dummy: Placeholder objects
- Stub: Returns predetermined values
- Mock: Verifies interactions
- Spy: Records how it was used
- Fake: Working implementation (simplified)
```

### Example

```javascript
// Mock external API
const mockAPI = {
  getUser: jest.fn().mockResolvedValue({ id: 1, name: 'John' })
};

// Test
const user = await service.getUser(1);
expect(mockAPI.getUser).toHaveBeenCalledWith(1);
```

---

## Test Data Management

### Best Practices

- Use factories or builders for test data
- Keep test data minimal
- Use realistic but not real data
- Never use production data

### Factory Pattern

```javascript
// User factory
function createTestUser(overrides = {}) {
  return {
    id: 1,
    name: 'Test User',
    email: 'test@example.com',
    role: 'user',
    ...overrides
  };
}

// Usage
const adminUser = createTestUser({ role: 'admin' });
const userWithoutEmail = createTestUser({ email: null });
```

---

## Error Testing

### Test Error Conditions

```javascript
// Synchronous errors
it('should throw error for invalid input', () => {
  expect(() => parseData(null)).toThrow(ValidationError);
});

// Async errors
it('should reject promise for network error', async () => {
  await expect(fetchData()).rejects.toThrow(NetworkError);
});
```

---

## Performance Testing

### When to Test Performance

- After optimization work
- Before major releases
- For performance-critical code
- Regression testing

### Example

```javascript
it('should complete operation within time limit', async () => {
  const start = Date.now();
  await performOperation();
  const duration = Date.now() - start;

  expect(duration).toBeLessThan(1000); // < 1 second
});
```

---

## Continuous Testing

### Local Development

```bash
# Run tests on file changes
$TEST_RUNNER run --watch

# Run specific test file
$TEST_RUNNER run path/to/test.test.js

# Run tests matching pattern
$TEST_RUNNER run --grep "authentication"
```

### CI/CD Integration

```yaml
pipeline:
  - name: "Test"
    steps:
      - run: $TEST_RUNNER run
      - run: $TEST_RUNNER run --coverage
      - condition: coverage < threshold
        action: fail_build
```

See [../System/automation.md](../System/automation.md)

---

## Debugging Tests

### Common Commands

```bash
# Run single test
$TEST_RUNNER run --test-name="specific test name"

# Run with verbose output
$TEST_RUNNER run --verbose

# Run in debug mode
$TEST_RUNNER run --inspect

# Update snapshots
$TEST_RUNNER run --update-snapshots
```

### Debugging Tips

1. Use `console.log()` strategically
2. Run single test in isolation
3. Check test data setup
4. Verify mocks are configured correctly
5. Check async handling

---

## Anti-Patterns

### ❌ DON'T

- Write tests after finding bugs in production
- Skip tests for "simple" code
- Share state between tests
- Test implementation details
- Make tests depend on execution order
- Ignore flaky tests

### ✅ DO

- Write tests with or before code
- Test all code paths
- Make tests independent
- Test behavior, not implementation
- Run tests in random order
- Fix flaky tests immediately

---

## Related Documentation

- [workflow.md](workflow.md) - Development workflow
- [quality-gates.md](quality-gates.md) - Quality checkpoints
- [../System/automation.md](../System/automation.md) - CI/CD setup
- [../System/principles.md](../System/principles.md) - Core principles

---

## Language-Specific Testing Tools

### JavaScript/TypeScript

```bash
$TEST_RUNNER="jest|vitest|mocha"
Coverage: "istanbul|c8"
Assertions: "expect|chai|assert"
```

### Python

```bash
$TEST_RUNNER="pytest|unittest|nose2"
Coverage: "pytest-cov|coverage.py"
Mocking: "unittest.mock|pytest-mock"
```

### Go

```bash
$TEST_RUNNER="go test"
Coverage: "go test -cover"
Mocking: "gomock|testify"
```

### Rust

```bash
$TEST_RUNNER="cargo test"
Coverage: "tarpaulin|grcov"
Mocking: "mockall|mocktopus"
```

### Java

```bash
$TEST_RUNNER="junit|testng"
Coverage: "jacoco|cobertura"
Mocking: "mockito|easymock"
```

---

**Remember**: Tests are first-class code. Write them well, maintain them carefully, and run them frequently.
