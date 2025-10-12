# Testing GitHub Actions Locally with act

**Last Updated**: 2025-10-12
**Status**: Recommended for Warren development workflow

## Overview

[`act`](https://github.com/nektos/act) is a tool that allows you to run GitHub Actions workflows locally using Docker containers. This enables faster iteration, debugging, and testing of CI/CD workflows without pushing commits to GitHub.

## Why Use act for Warren?

### Benefits

1. **Faster Iteration**: Test workflow changes locally before pushing to GitHub
2. **Debug CI Failures**: Reproduce and fix CI issues on your local machine
3. **Cost Savings**: Reduce GitHub Actions runner minutes usage
4. **Offline Development**: Work on CI/CD without internet connectivity
5. **Pre-commit Validation**: Catch workflow errors before they reach CI

### Use Cases for Warren

- **Testing Workflow Changes**: Modify `.github/workflows/*.yml` and test immediately
- **Debugging Flaky Tests**: Reproduce scheduler/integration test failures locally
- **Linter Configuration**: Test golangci-lint configuration changes
- **Multi-Go Version Testing**: Verify compatibility with Go 1.22 and 1.23
- **Build Verification**: Ensure binaries build correctly across platforms

## Prerequisites

- **Docker**: Must be installed and running
  ```bash
  docker --version  # Should show Docker 20.10+
  ```

- **Disk Space**: ~2-5GB for Docker images (depends on runner size)

## Installation

### macOS (Homebrew)

```bash
brew install act
```

### Linux

```bash
# Download latest release
curl -s https://raw.githubusercontent.com/nektos/act/master/install.sh | sudo bash

# Or with Homebrew on Linux
brew install act
```

### Verification

```bash
act --version
# Should show: act version 0.2.x or higher
```

## Configuration

### 1. First-Time Setup

When you first run `act`, it will prompt you to choose a runner image size:

```
? Please choose the default image you want to use with act:
  - Large (~17GB): Mirrors GitHub-hosted runners (most compatible)
  - Medium (~500MB): Includes essential tools ⭐ RECOMMENDED
  - Micro (~200MB): Contains only Node.js (limited compatibility)
```

**Recommendation for Warren**: Choose **Medium** image

The medium image (`catthehacker/ubuntu:act-latest`) includes:
- Go toolchain
- Common build tools (make, gcc)
- Docker CLI (for nested Docker operations)
- Git, curl, wget
- ~500MB download, 1.5GB disk space

### 2. Create `.actrc` Configuration

Create a `.actrc` file in the Warren project root:

```bash
# Warren project .actrc configuration

# Use medium image for balance of size and compatibility
-P ubuntu-latest=catthehacker/ubuntu:act-latest

# Enable verbose output for debugging
--verbose

# Use host Docker daemon (for containerd tests if needed)
--container-daemon-socket -

# Set default event to push (most common for Warren)
--defaultbranch main
```

### 3. Create `.secrets` File (Optional)

For workflows requiring secrets (like Docker Hub publishing):

```bash
# .secrets - DO NOT COMMIT THIS FILE
DOCKERHUB_USERNAME=your_username
DOCKERHUB_TOKEN=your_token
CODECOV_TOKEN=your_codecov_token
```

**Important**: Add `.secrets` to `.gitignore`:

```bash
echo ".secrets" >> .gitignore
```

## Usage

### Run All Workflows

```bash
# Run all workflows triggered by push event
act push

# Dry run (show what would execute without running)
act push --dry-run
```

### Run Specific Workflow

```bash
# Run only test workflow
act push --workflows .github/workflows/test.yml

# Run only lint job from test workflow
act push --workflows .github/workflows/test.yml --job lint
```

### Run Specific Job

```bash
# Run just the lint job
act --job lint

# Run tests with Go 1.23
act --job "test (1.23)"

# Run tests with Go 1.22
act --job "test (1.22)"
```

### Common Options

```bash
# List all workflows and jobs
act --list

# Run with secrets file
act push --secret-file .secrets

# Run with specific event
act pull_request

# Run with custom environment variable
act push --env GO_VERSION=1.23

# Interactive mode (attach to container)
act push --job test --bind

# Rebuild Docker images (after changes)
act push --pull

# Use host network (for local services)
act --container-architecture linux/amd64
```

## Warren-Specific Examples

### 1. Test Certificate Signature Changes

```bash
# Run security tests after modifying IssueNodeCertificate
act push --job "test (1.23)" \
  --workflows .github/workflows/test.yml
```

### 2. Verify Go Version Compatibility

```bash
# Test with Go 1.22
act push --job "test (1.22)"

# Test with Go 1.23
act push --job "test (1.23)"
```

### 3. Debug Flaky Scheduler Tests

```bash
# Run tests multiple times to reproduce race condition
for i in {1..5}; do
  echo "Run $i:"
  act push --job "test (1.23)"
done
```

### 4. Test Lint Configuration

```bash
# Run golangci-lint locally before pushing
act push --job lint

# Test with specific golangci-lint version
act push --job lint --env GOLANGCI_LINT_VERSION=v1.55.2
```

### 5. Verify Build on All Platforms

```bash
# Run build job (only after tests pass in CI)
act push --job build
```

### 6. Test PR Validation

```bash
# Simulate PR event
act pull_request --workflows .github/workflows/pr.yml
```

## Limitations & Workarounds

### 1. Multi-Platform Builds

**Issue**: act runs on your host architecture (e.g., arm64 on M1 Mac)

**Workaround**:
```bash
# Specify target architecture
act push --container-architecture linux/amd64

# For release workflow (multi-platform builds), use GitHub Actions
# act has limited QEMU/buildx support
```

### 2. Containerd Integration Tests

**Issue**: Warren's integration tests require containerd, which needs privileged access

**Workaround**:
```bash
# Skip integration tests in act
act push --job test --env SKIP_INTEGRATION_TESTS=true

# Or run integration tests directly on host
go test -v ./test/integration/...
```

### 3. Docker Hub Publishing

**Issue**: act may not have full Docker buildx support for multi-arch images

**Recommendation**: Use act for testing build steps, but rely on GitHub Actions for actual releases

### 4. GitHub API Rate Limits

**Issue**: act may trigger rate limits when pulling action definitions

**Workaround**:
```bash
# Use GitHub token for higher rate limits
act push --secret GITHUB_TOKEN=$(gh auth token)
```

## Troubleshooting

### Issue: "command not found" errors in act

**Solution**: Use medium or large runner image instead of micro:

```bash
# Update .actrc
-P ubuntu-latest=catthehacker/ubuntu:full-latest  # Large image
```

### Issue: "Cannot connect to Docker daemon"

**Solution**: Ensure Docker Desktop is running:

```bash
docker ps  # Should list running containers
```

### Issue: Tests fail in act but pass in GitHub Actions

**Solution**: Check for platform-specific differences:

```bash
# Compare environment variables
act push --job test --dry-run | grep -A 50 "ENV:"

# Check Go version
act push --job test --bind
# Then inside container:
go version
```

### Issue: "Out of disk space" errors

**Solution**: Clean up Docker images and containers:

```bash
# Remove act images
docker rmi catthehacker/ubuntu:act-latest

# Clean up Docker system
docker system prune -af

# Re-pull images
act --pull
```

## Best Practices

### 1. Use act in Pre-commit Hook

Add to `.git/hooks/pre-commit`:

```bash
#!/bin/bash
# Run linter before commit
echo "Running linter locally..."
act --job lint --quiet

if [ $? -ne 0 ]; then
  echo "❌ Linter failed. Fix issues before committing."
  exit 1
fi

echo "✅ Linter passed"
```

### 2. Create Makefile Targets

Add to `Makefile`:

```makefile
.PHONY: act-test act-lint act-all

# Run tests locally with act
act-test:
	act push --job "test (1.23)" --workflows .github/workflows/test.yml

# Run linter locally with act
act-lint:
	act push --job lint --workflows .github/workflows/test.yml

# Run all CI checks locally
act-all:
	act push --workflows .github/workflows/test.yml

# List all available jobs
act-list:
	act --list
```

Usage:
```bash
make act-lint   # Run linter
make act-test   # Run tests
make act-all    # Run everything
```

### 3. Document Workflow Changes

When modifying `.github/workflows/*.yml`:

1. Test locally with `act` first
2. Commit workflow changes
3. Document changes in PR description
4. Mention "Tested with act" in commit message

### 4. Use act for Debugging Only

- ✅ Use act for: Testing workflows, debugging failures, validating changes
- ❌ Don't use act for: Official releases, security-sensitive operations, multi-arch builds

### 5. Keep .actrc in Version Control

Add `.actrc` to git for team consistency:

```bash
git add .actrc
git commit -m "chore: add act configuration for local CI testing"
```

But keep `.secrets` private:

```bash
# .gitignore
.secrets
```

## Integration with Warren Workflow

### Development Workflow

```
1. Make code changes
   ↓
2. Run act locally
   ├─ make act-lint     # Quick linter check
   ├─ make act-test     # Run tests
   └─ make act-all      # Full CI validation
   ↓
3. If act passes:
   ├─ git commit
   └─ git push
   ↓
4. GitHub Actions runs
   └─ Should pass (already validated locally)
```

### Debugging CI Failures

```
1. CI fails on GitHub
   ↓
2. Reproduce locally:
   └─ act push --job <failed-job>
   ↓
3. Debug in act:
   ├─ act push --job <failed-job> --bind  # Interactive
   └─ Inspect logs, modify code
   ↓
4. Fix and re-test:
   └─ act push --job <failed-job>
   ↓
5. Push fix to GitHub
   └─ CI should pass now
```

## Resources

- **Official Documentation**: https://nektosact.com/
- **GitHub Repository**: https://github.com/nektos/act
- **Runner Images**: https://github.com/catthehacker/docker_images
- **VS Code Extension**: https://marketplace.visualstudio.com/items?itemName=me-dutour-mathieu.vscode-github-actions

## Next Steps

1. **Install act**: `brew install act` (macOS) or see installation section above
2. **Configure**: Create `.actrc` and `.secrets` files
3. **Test**: Run `act --list` to see available jobs
4. **Integrate**: Add `make act-*` targets to Makefile
5. **Document**: Add act usage to team onboarding docs

## Feedback

If you encounter issues with act in the Warren project, please:

1. Check the troubleshooting section above
2. Consult the [act documentation](https://nektosact.com/)
3. Open an issue in the Warren repository with:
   - act version (`act --version`)
   - Docker version (`docker --version`)
   - Workflow file being tested
   - Error messages or logs

---

**Maintained by**: Warren DevOps Team
**Questions?**: Open a GitHub Discussion or Slack #warren-dev
