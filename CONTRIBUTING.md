# Contributing to Warren

Thank you for your interest in contributing to Warren! We welcome contributions from the community and are excited to work with you.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [How to Contribute](#how-to-contribute)
- [Submitting Issues](#submitting-issues)
- [Submitting Pull Requests](#submitting-pull-requests)
- [Code Review Process](#code-review-process)
- [Coding Standards](#coding-standards)
- [Testing Requirements](#testing-requirements)
- [Documentation Requirements](#documentation-requirements)
- [Commit Message Guidelines](#commit-message-guidelines)
- [Community](#community)

---

## Code of Conduct

This project adheres to the Contributor Covenant [Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code. Please report unacceptable behavior to opensource@cuemby.com.

---

## Getting Started

### Prerequisites

Before you begin, ensure you have the following installed:

- **Go 1.22 or later** - [Download Go](https://go.dev/dl/)
- **containerd** - Container runtime (for integration testing)
  - Linux: `apt install containerd` or `yum install containerd`
  - macOS: Install via Docker Desktop or Lima
- **Protocol Buffers compiler (protoc)** - For gRPC code generation
  - `brew install protobuf` (macOS)
  - `apt install protobuf-compiler` (Ubuntu)
- **Git** - Version control

### Optional Tools

- **golangci-lint** - Linting: `brew install golangci-lint`
- **Lima** - For multi-node testing on macOS: `brew install lima`
- **Make** - Build automation (usually pre-installed)

---

## Development Setup

### 1. Fork and Clone

```bash
# Fork the repository on GitHub first, then:
git clone https://github.com/YOUR_USERNAME/warren.git
cd warren
```

### 2. Install Dependencies

```bash
# Download Go dependencies
go mod download

# Install protobuf Go plugins
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

### 3. Build Warren

```bash
# Build for your platform
make build

# Or use go directly
go build -o bin/warren ./cmd/warren

# Verify build
./bin/warren --version
```

### 4. Run Tests

```bash
# Run unit tests
go test ./...

# Run with coverage
go test -cover ./...

# Run integration tests (requires containerd)
cd test/integration
./e2e_test.sh
```

### 5. Generate Protobuf Code (if needed)

```bash
# If you modify api/proto/warren.proto
make proto

# Or manually
protoc --go_out=. --go-grpc_out=. api/proto/warren.proto
```

---

## How to Contribute

### Finding Work

1. **Good First Issues**: Look for issues labeled [`good first issue`](https://github.com/cuemby/warren/labels/good%20first%20issue)
2. **Help Wanted**: Check [`help wanted`](https://github.com/cuemby/warren/labels/help%20wanted) for community contributions
3. **Propose New Features**: Open a discussion in [GitHub Discussions](https://github.com/cuemby/warren/discussions)

### Types of Contributions

- **Bug Fixes**: Fix reported issues
- **Features**: Implement new functionality
- **Documentation**: Improve or add documentation
- **Tests**: Add test coverage
- **Examples**: Create usage examples
- **Performance**: Optimize existing code
- **Refactoring**: Improve code quality

---

## Submitting Issues

### Bug Reports

When reporting a bug, please include:

1. **Warren version**: `warren --version`
2. **Operating system**: Linux/macOS, version
3. **Go version**: `go version`
4. **Steps to reproduce**: Detailed steps
5. **Expected behavior**: What should happen
6. **Actual behavior**: What actually happens
7. **Logs**: Relevant log output (use `--log-level debug`)
8. **Configuration**: Any relevant config files

**Use the [Bug Report Template](https://github.com/cuemby/warren/issues/new?template=bug_report.md)**

### Feature Requests

When requesting a feature:

1. **Use case**: Describe the problem you're trying to solve
2. **Proposed solution**: How you think it should work
3. **Alternatives**: Other solutions you've considered
4. **Additional context**: Examples, mockups, references

**Use the [Feature Request Template](https://github.com/cuemby/warren/issues/new?template=feature_request.md)**

### Questions

For usage questions, please use [GitHub Discussions](https://github.com/cuemby/warren/discussions) instead of issues.

---

## Submitting Pull Requests

### Before You Submit

1. **Search existing PRs**: Ensure no duplicate work
2. **Open an issue first**: For significant changes, discuss in an issue
3. **Follow coding standards**: See [Coding Standards](#coding-standards)
4. **Add tests**: All new code must have tests
5. **Update documentation**: Keep docs in sync with code changes

### PR Process

1. **Create a branch**:
   ```bash
   git checkout -b feature/my-feature
   # or
   git checkout -b fix/my-bugfix
   ```

2. **Make your changes**:
   - Write code following project conventions
   - Add tests for new functionality
   - Update documentation as needed

3. **Test your changes**:
   ```bash
   # Run all tests
   go test ./...

   # Run linter
   golangci-lint run

   # Build to ensure no errors
   make build
   ```

4. **Commit your changes**:
   ```bash
   # Use conventional commit format (see below)
   git commit -m "feat: add new scheduler policy"
   ```

5. **Push to your fork**:
   ```bash
   git push origin feature/my-feature
   ```

6. **Open a Pull Request**:
   - Fill out the PR template completely
   - Link related issues (e.g., "Closes #123")
   - Provide clear description of changes
   - Include screenshots/examples if applicable

### PR Checklist

Before submitting, ensure:

- [ ] Code follows project style guidelines
- [ ] Tests added/updated and passing
- [ ] Documentation updated (if applicable)
- [ ] Commit messages follow conventional commits format
- [ ] PR description is clear and complete
- [ ] All CI checks passing
- [ ] Branch is up-to-date with main

---

## Code Review Process

### What to Expect

1. **Initial Review**: A maintainer will review within 1-3 business days
2. **Feedback**: You may receive comments requesting changes
3. **Iteration**: Address feedback and push new commits
4. **Approval**: Once approved by at least one maintainer, PR can be merged
5. **Merge**: Maintainers will merge using "Squash and Merge"

### Review Criteria

Reviewers will check:

- **Correctness**: Does it work as intended?
- **Testing**: Adequate test coverage?
- **Documentation**: Changes documented?
- **Style**: Follows coding standards?
- **Performance**: No obvious performance issues?
- **Security**: No security vulnerabilities?
- **Simplicity**: Is the solution simple and maintainable?

---

## Coding Standards

### Go Style Guide

- Follow [Effective Go](https://go.dev/doc/effective_go)
- Follow [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Use `gofmt` for formatting (automatically handled by `golangci-lint`)

### Code Organization

- **Package structure**: Follow existing structure in `pkg/`, `cmd/`, `internal/`
- **File naming**: Use lowercase with underscores (e.g., `my_file.go`)
- **Function naming**: Use camelCase for private, PascalCase for public
- **Constants**: Use ALL_CAPS or camelCase depending on scope

### Best Practices

- **Keep functions small**: Aim for < 50 lines per function
- **Single responsibility**: Each function/struct should do one thing
- **Error handling**: Always check and handle errors appropriately
- **Comments**: Document public APIs, complex logic, and non-obvious code
- **Avoid premature optimization**: Prioritize clarity over performance until profiling proves otherwise

### Example Code

```go
// Good: Clear, documented, well-structured
package scheduler

// Schedule assigns tasks to nodes based on available resources.
// Returns an error if no suitable node is found.
func (s *Scheduler) Schedule(task *types.Task) (*types.Node, error) {
    if task == nil {
        return nil, fmt.Errorf("task cannot be nil")
    }

    nodes, err := s.getAvailableNodes()
    if err != nil {
        return nil, fmt.Errorf("failed to get nodes: %w", err)
    }

    node := s.selectBestNode(nodes, task)
    if node == nil {
        return nil, fmt.Errorf("no suitable node found for task %s", task.ID)
    }

    return node, nil
}
```

---

## Testing Requirements

### Test Coverage

- **Minimum coverage**: 80% for new code
- **Critical paths**: 100% coverage for scheduler, reconciler, FSM
- **Edge cases**: Test error conditions and boundary cases

### Types of Tests

1. **Unit Tests**: Test individual functions/methods
   ```go
   func TestSchedulerSelectNode(t *testing.T) {
       // Arrange
       scheduler := NewScheduler()
       nodes := []*types.Node{...}
       task := &types.Task{...}

       // Act
       node, err := scheduler.Schedule(task)

       // Assert
       assert.NoError(t, err)
       assert.NotNil(t, node)
   }
   ```

2. **Integration Tests**: Test component interactions
   - Located in `test/integration/`
   - May require external dependencies (containerd)

3. **End-to-End Tests**: Test full workflows
   - Shell scripts in `test/lima/` or `test/integration/`
   - Test cluster formation, service deployment, failover

### Running Tests

```bash
# Unit tests
go test ./pkg/...

# Specific package
go test ./pkg/scheduler

# With coverage
go test -cover ./...

# Integration tests
cd test/integration && ./e2e_test.sh

# Load tests (requires Lima VMs)
cd test/lima && ./test-load.sh
```

---

## Documentation Requirements

### When Documentation is Required

- **New features**: Always document new functionality
- **API changes**: Update API reference
- **Breaking changes**: Clearly document migration path
- **Configuration**: Document new flags or config options

### Documentation Types

1. **Code Comments**: Document public APIs
   ```go
   // NewManager creates a new Manager instance with the given configuration.
   // It initializes Raft consensus, storage, and the API server.
   // Returns an error if initialization fails.
   func NewManager(config *Config) (*Manager, error) {
       // ...
   }
   ```

2. **README Updates**: Update main README.md if needed

3. **User Guides**: Add to `docs/` for user-facing features
   - `docs/getting-started.md`
   - `docs/concepts/`
   - `docs/cli-reference.md`

4. **Architecture Docs**: Update `docs/developer-guide.md` for internal changes

5. **Examples**: Add YAML examples to `examples/`

---

## Commit Message Guidelines

### Conventional Commits Format

We follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Types

- **feat**: New feature
- **fix**: Bug fix
- **docs**: Documentation only
- **style**: Code style changes (formatting, no logic change)
- **refactor**: Code refactoring (no functional change)
- **perf**: Performance improvement
- **test**: Adding or updating tests
- **chore**: Maintenance tasks (dependencies, build, etc.)
- **ci**: CI/CD changes

### Examples

```bash
# Feature
feat(scheduler): add node affinity support for volumes

# Bug fix
fix(worker): prevent goroutine leak in task polling

# Documentation
docs: add migration guide from Docker Swarm

# Breaking change
feat(api)!: change CreateService signature to support secrets

BREAKING CHANGE: CreateService now requires SecretRefs parameter
```

### Commit Best Practices

- **Atomic commits**: Each commit should be a logical unit
- **Descriptive**: Clearly explain what and why
- **Reference issues**: Use "Closes #123" or "Fixes #123"
- **Sign commits**: Use GPG signing for verification (optional but recommended)

---

## Community

### Communication Channels

- **GitHub Issues**: Bug reports and feature requests
- **GitHub Discussions**: Questions, ideas, show-and-tell
- **Discord** (coming soon): Real-time chat
- **Mailing List**: opensource@cuemby.com

### Getting Help

- **Documentation**: Check [docs/](docs/)
- **Discussions**: Ask in [GitHub Discussions](https://github.com/cuemby/warren/discussions)
- **Examples**: See [examples/](examples/)
- **Issue Search**: Search existing issues first

### Maintainer Team

Current maintainers:

- **Angel Ramirez** (@ar4mirez) - Lead Maintainer
- **Cuemby Team** - Core contributors

### Recognition

We recognize contributors in the following ways:

- Listed in release notes
- Featured in monthly contributor spotlight (coming soon)
- Maintainer status for consistent, high-quality contributions

---

## Thank You!

Warren is built by the community, for the community. Every contribution, no matter how small, makes a difference. We appreciate your time and effort!

**Happy coding!** 🚀

---

**Questions?** Open a [Discussion](https://github.com/cuemby/warren/discussions) or email opensource@cuemby.com
