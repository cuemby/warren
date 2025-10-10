.PHONY: build clean test lint fmt help install dev

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

## help: Display this help message
help:
	@echo "Warren - Container Orchestrator"
	@echo ""
	@echo "Usage:"
	@echo "  make <target>"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

## build: Build Warren binary
build:
	@echo "Building Warren..."
	@mkdir -p $(BUILD_DIR)
	go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY) ./cmd/warren
	@echo "✓ Built: $(BUILD_DIR)/$(BINARY)"

## build-release: Build optimized Warren binary for release
build-release:
	@echo "Building Warren (release)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -ldflags="$(RELEASE_LDFLAGS)" -o $(BUILD_DIR)/$(BINARY) ./cmd/warren
	@echo "✓ Built: $(BUILD_DIR)/$(BINARY)"
	@ls -lh $(BUILD_DIR)/$(BINARY)

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

## check: Run all checks (fmt, lint, test)
check: fmt lint test
	@echo "✓ All checks passed"

.DEFAULT_GOAL := help
