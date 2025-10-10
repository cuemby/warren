.PHONY: build clean test lint fmt help install dev proto

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

.DEFAULT_GOAL := help
