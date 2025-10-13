# Warren Docker Image
# Multi-stage build for minimal image size

# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make protobuf-dev

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o warren \
    ./cmd/warren

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata

# Create non-root user
RUN addgroup -g 1000 warren && \
    adduser -D -u 1000 -G warren warren

# Create data directory
RUN mkdir -p /var/lib/warren/data && \
    chown -R warren:warren /var/lib/warren

# Copy binary from builder
COPY --from=builder /build/warren /usr/local/bin/warren

# Set user
USER warren

# Expose ports
EXPOSE 8080 9090 51820/udp

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD warren --version || exit 1

# Default command
ENTRYPOINT ["/usr/local/bin/warren"]
CMD ["--help"]
