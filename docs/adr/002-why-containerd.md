# ADR-002: Use containerd for Container Runtime

**Status**: Accepted
**Date**: 2025-10-09

## Decision

**Use containerd as Warren's container runtime**, accessed via its gRPC API.

## Context

Warren needs to manage container lifecycle (pull images, create, start, stop containers). Options:

1. **Docker Engine** - Full Docker daemon
2. **containerd** - Industry-standard container runtime
3. **CRI-O** - Kubernetes-focused runtime
4. **Custom runtime** - Build from scratch

## Rationale

**Chose containerd because**:

✅ **Docker-independent**: Works without Docker Desktop/daemon
✅ **CRI-compatible**: Standard Container Runtime Interface
✅ **Industry adoption**: Used by Kubernetes, Docker, cloud providers
✅ **Lightweight**: Smaller footprint than full Docker
✅ **Stable API**: Well-documented gRPC interface
✅ **Go library**: Clean integration (`github.com/containerd/containerd`)
✅ **Production-proven**: Powers major platforms

## Alternatives Rejected

### Docker Engine
❌ Couples Warren to Docker daemon
❌ Heavier weight (includes Docker-specific features Warren doesn't need)
❌ Less suitable for edge (resource overhead)

### CRI-O
❌ Designed specifically for Kubernetes (Warren is independent)
❌ Less mature than containerd
❌ Smaller ecosystem

### Custom Runtime
❌ Massive engineering effort
❌ Not core differentiation for Warren
❌ OCI compliance complexity

## Implementation

```go
// Connect to containerd
client, _ := containerd.New("/run/containerd/containerd.sock")

// Pull image
image, _ := client.Pull(ctx, "docker.io/library/nginx:latest")

// Create container
container, _ := client.NewContainer(ctx, "my-container",
    containerd.WithImage(image),
    containerd.WithNewSnapshot("snap", image),
)

// Start container
task, _ := container.NewTask(ctx, cio.NewCreator())
task.Start(ctx)
```

## Consequences

✅ Zero Docker dependency
✅ Works on any system with containerd
✅ Lightweight for edge deployments
⚠️ Requires containerd installed (standard on most systems)
⚠️ Users familiar with Docker commands need Warren CLI

## Validation

See [poc/containerd/](../../poc/containerd/) - validates lifecycle, memory usage, resource constraints.

**Status**: Accepted - POC successful
