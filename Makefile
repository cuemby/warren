.PHONY: build clean test lint fmt help install dev proto download-containerd embed-deps act-test act-lint act-build act-all act-list act-install

# Build variables
VERSION ?= dev
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS := -X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildTime=$(BUILD_TIME)

# Build optimization for releases
RELEASE_LDFLAGS := $(LDFLAGS) -s -w

# Binaries
BINARY := warren
BUILD_DIR := bin

# Embedded dependencies
EMBED_DIR := pkg/embedded/binaries

# Protobuf
PROTO_DIR := api/proto
PROTO_OUT := api/proto

## help: Display this help message
help:
	@echo "Warren - Container Orchestrator"
	@echo ""
	@echo "Usage:"
	@echo "  make <target>"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

## proto: Generate Go code from protobuf definitions
proto:
	@echo "Generating protobuf code..."
	@if ! command -v protoc >/dev/null 2>&1; then \
		echo "Error: protoc not found. Install with:"; \
		echo "  brew install protobuf  # macOS"; \
		echo "  sudo apt install protobuf-compiler  # Ubuntu"; \
		exit 1; \
	fi
	@mkdir -p $(PROTO_OUT)
	protoc --go_out=$(PROTO_OUT) --go_opt=paths=source_relative \
		--go-grpc_out=$(PROTO_OUT) --go-grpc_opt=paths=source_relative \
		$(PROTO_DIR)/warren.proto
	@echo "✓ Protobuf code generated"

## download-containerd: Download containerd binaries for embedding
download-containerd:
	@echo "Downloading containerd binaries..."
	@./scripts/download-containerd.sh
	@echo "✓ Containerd binaries downloaded"

## embed-deps: Prepare all embedded dependencies (runs download-containerd)
embed-deps: download-containerd
	@echo "✓ All embedded dependencies prepared"

## build: Build Warren binary (without embedded containerd - for dev)
build:
	@echo "Building Warren (dev mode - no embedded containerd)..."
	@mkdir -p $(BUILD_DIR)
	go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY) ./cmd/warren
	@echo "✓ Built: $(BUILD_DIR)/$(BINARY)"
	@echo "⚠️  Note: This build does NOT include embedded containerd."
	@echo "   Use 'make build-embedded' to build with containerd, or"
	@echo "   Use --external-containerd flag to use system containerd."

## build-embedded: Build Warren with embedded containerd (requires download-containerd first)
build-embedded: embed-deps
	@echo "Building Warren with embedded containerd..."
	@if [ "$$(uname -s)" = "Darwin" ]; then \
		echo "⚠️  WARNING: You're on macOS!"; \
		echo "   Containerd does NOT provide macOS binaries."; \
		echo "   This will build Warren with Linux containerd embedded (for cross-compilation)."; \
		echo ""; \
		echo "   To run Warren on macOS, use:"; \
		echo "     1. make build (without embedded)"; \
		echo "     2. sudo warren cluster init --external-containerd"; \
		echo ""; \
		echo "   Or cross-compile for Linux:"; \
		echo "     make build-linux-amd64"; \
		echo ""; \
	fi
	@mkdir -p $(BUILD_DIR)
	go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY) ./cmd/warren
	@echo "✓ Built: $(BUILD_DIR)/$(BINARY)"
	@ls -lh $(BUILD_DIR)/$(BINARY)
	@if [ "$$(uname -s)" = "Darwin" ]; then \
		echo ""; \
		echo "ℹ️  This binary contains Linux containerd binaries (not usable on macOS)."; \
		echo "   For macOS dev: make build && sudo warren cluster init --external-containerd"; \
	fi

## build-release: Build optimized Warren binary for release
build-release:
	@echo "Building Warren (release)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -ldflags="$(RELEASE_LDFLAGS)" -o $(BUILD_DIR)/$(BINARY) ./cmd/warren
	@echo "✓ Built: $(BUILD_DIR)/$(BINARY)"
	@ls -lh $(BUILD_DIR)/$(BINARY)

## build-all: Build for all supported platforms
build-all: build-linux-amd64 build-linux-arm64 build-darwin-amd64 build-darwin-arm64
	@echo "✓ Built all platforms"
	@ls -lh $(BUILD_DIR)/

## build-linux-amd64: Build for Linux AMD64
build-linux-amd64:
	@echo "Building for Linux AMD64..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="$(RELEASE_LDFLAGS)" \
		-o $(BUILD_DIR)/$(BINARY)-linux-amd64 ./cmd/warren
	@echo "✓ Built: $(BUILD_DIR)/$(BINARY)-linux-amd64"

## build-linux-arm64: Build for Linux ARM64
build-linux-arm64:
	@echo "Building for Linux ARM64..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="$(RELEASE_LDFLAGS)" \
		-o $(BUILD_DIR)/$(BINARY)-linux-arm64 ./cmd/warren
	@echo "✓ Built: $(BUILD_DIR)/$(BINARY)-linux-arm64"

## build-darwin-amd64: Build for macOS AMD64
build-darwin-amd64:
	@echo "Building for macOS AMD64..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="$(RELEASE_LDFLAGS)" \
		-o $(BUILD_DIR)/$(BINARY)-darwin-amd64 ./cmd/warren
	@echo "✓ Built: $(BUILD_DIR)/$(BINARY)-darwin-amd64"

## build-darwin-arm64: Build for macOS ARM64 (Apple Silicon)
build-darwin-arm64:
	@echo "Building for macOS ARM64..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="$(RELEASE_LDFLAGS)" \
		-o $(BUILD_DIR)/$(BINARY)-darwin-arm64 ./cmd/warren
	@echo "✓ Built: $(BUILD_DIR)/$(BINARY)-darwin-arm64"

## install: Install Warren binary to /usr/local/bin
install: build
	@echo "Installing Warren..."
	sudo cp $(BUILD_DIR)/$(BINARY) /usr/local/bin/
	sudo ln -sf /usr/local/bin/$(BINARY) /usr/local/bin/wrn
	@echo "✓ Installed: /usr/local/bin/warren (alias: wrn)"

## dev: Run Warren in development mode
dev: build
	@$(BUILD_DIR)/$(BINARY) --help

## test: Run tests
test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...
	@echo "✓ Tests passed"

## test-integration: Run integration tests
test-integration:
	@echo "Running integration tests..."
	go test -v -tags=integration ./test/integration/...

## lint: Run linters
lint:
	@echo "Running linters..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
		echo "✓ Linting passed"; \
	else \
		echo "golangci-lint not installed. Install with:"; \
		echo "  brew install golangci-lint  # macOS"; \
		echo "  # or"; \
		echo "  go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

## fmt: Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	@echo "✓ Code formatted"

## tidy: Tidy go.mod
tidy:
	@echo "Tidying dependencies..."
	go mod tidy
	@echo "✓ Dependencies tidied"

## clean: Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out
	@echo "✓ Cleaned"

## clean-all: Clean build artifacts and embedded binaries
clean-all: clean
	@echo "Cleaning embedded binaries..."
	rm -f $(EMBED_DIR)/containerd-*
	@echo "✓ Cleaned all"

## check: Run all checks (fmt, lint, test)
check: fmt lint test
	@echo "✓ All checks passed"

## size: Check binary size
size: build-release
	@echo "Binary size check:"
	@SIZE=$$(stat -f%z $(BUILD_DIR)/$(BINARY) 2>/dev/null || stat -c%s $(BUILD_DIR)/$(BINARY) 2>/dev/null); \
	SIZE_MB=$$((SIZE / 1024 / 1024)); \
	echo "  Current: $${SIZE_MB}MB"; \
	if [ $$SIZE_MB -gt 100 ]; then \
		echo "  ⚠️  WARNING: Binary exceeds 100MB target!"; \
		exit 1; \
	else \
		echo "  ✓ Binary size OK (target: <100MB)"; \
	fi

#
# GitHub Actions Local Testing (act)
# https://nektosact.com/
#

## act-install: Install act (GitHub Actions local runner)
act-install:
	@echo "Installing act..."
	@if command -v brew >/dev/null 2>&1; then \
		brew install act; \
		echo "✓ act installed via Homebrew"; \
	else \
		echo "Installing act via script..."; \
		curl -s https://raw.githubusercontent.com/nektos/act/master/install.sh | sudo bash; \
		echo "✓ act installed"; \
	fi
	@echo ""
	@echo "Setup complete! Configuration in .actrc"
	@echo "Run 'make act-list' to see available GitHub Actions jobs"

## act-list: List all GitHub Actions workflows and jobs
act-list:
	@echo "GitHub Actions Workflows and Jobs:"
	@act --list 2>/dev/null || (echo "❌ act not found. Install with: make act-install" && exit 1)

## act-lint: Run linter locally with act
act-lint:
	@echo "Running linter with act (GitHub Actions locally)..."
	@act push --job lint --workflows .github/workflows/test.yml 2>/dev/null || \
		(echo "❌ act not found. Install with: make act-install" && exit 1)

## act-test: Run tests locally with act (Go 1.23)
act-test:
	@echo "Running tests with act (GitHub Actions locally)..."
	@act push --job "test (1.23)" --workflows .github/workflows/test.yml 2>/dev/null || \
		(echo "❌ act not found. Install with: make act-install" && exit 1)

## act-test-all: Run tests for all Go versions (1.22 and 1.23)
act-test-all:
	@echo "Running tests for Go 1.22..."
	@act push --job "test (1.22)" --workflows .github/workflows/test.yml
	@echo ""
	@echo "Running tests for Go 1.23..."
	@act push --job "test (1.23)" --workflows .github/workflows/test.yml

## act-build: Run build job locally with act
act-build:
	@echo "Running build with act (GitHub Actions locally)..."
	@act push --job build --workflows .github/workflows/test.yml 2>/dev/null || \
		(echo "❌ act not found. Install with: make act-install" && exit 1)

## act-all: Run all GitHub Actions workflows locally
act-all:
	@echo "Running all GitHub Actions workflows locally..."
	@act push --workflows .github/workflows/test.yml 2>/dev/null || \
		(echo "❌ act not found. Install with: make act-install" && exit 1)

## act-pr: Simulate pull request workflow locally
act-pr:
	@echo "Simulating pull request workflow..."
	@act pull_request --workflows .github/workflows/pr.yml 2>/dev/null || \
		(echo "❌ act not found. Install with: make act-install" && exit 1)

## act-debug: Run tests with act in interactive mode
act-debug:
	@echo "Running tests in interactive mode (attach to container)..."
	@echo "Use 'exit' to leave the container"
	@act push --job "test (1.23)" --workflows .github/workflows/test.yml --bind

## act-clean: Clean up act containers and images
act-clean:
	@echo "Cleaning up act containers..."
	@docker ps -a --filter "label=act" -q | xargs -r docker rm -f 2>/dev/null || true
	@echo "Cleaning up act images..."
	@docker images "catthehacker/ubuntu" -q | xargs -r docker rmi -f 2>/dev/null || true
	@echo "✓ Cleaned up act resources"

.DEFAULT_GOAL := help
