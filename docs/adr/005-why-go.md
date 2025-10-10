# ADR-005: Use Go for Warren Implementation

**Status**: Accepted
**Date**: 2025-10-09

## Decision

**Implement Warren in Go 1.22+**.

## Context

Need to choose implementation language for a distributed container orchestrator. Requirements:
- Single binary distribution
- Cross-platform (Linux, macOS, Windows, ARM)
- Strong concurrency (goroutines)
- Mature ecosystem (networking, distributed systems)
- Fast development velocity

Primary candidates: **Go** vs **Rust**

## Rationale

**Chose Go because**:

✅ **Cloud-native ecosystem**: containerd, Kubernetes, Docker all in Go
✅ **Rapid development**: Fast iteration, simple syntax
✅ **Team expertise**: Existing Go knowledge (Kubernetes background)
✅ **Standard library**: Excellent net, http, crypto packages
✅ **Concurrency**: Goroutines perfect for orchestrator workload
✅ **Cross-compilation**: Trivial cross-platform builds
✅ **Static binaries**: Single binary with no runtime dependencies
✅ **Mature libraries**: Raft, gRPC, containerd clients available

## Why Not Rust?

Rust considered seriously but ultimately deferred:

**Rust Pros**:
- ✓ Better memory safety (borrow checker)
- ✓ Slightly better performance
- ✓ Smaller binaries
- ✓ Strong type system

**Rust Cons (for Warren v1.0)**:
- ❌ **Steeper learning curve**: Slower initial development
- ❌ **Smaller ecosystem**: Fewer distributed systems libraries
- ❌ **containerd integration**: Go client more mature
- ❌ **Team ramp-up**: Time to proficiency longer
- ❌ **AI assistance**: Less effective for complex Rust (borrow checker)

**Decision**: Go for v1.0, re-evaluate Rust for v2.0 or performance-critical components

## Trade-offs Accepted

**Memory Safety**:
- Go: Garbage collected (minor overhead, pauses)
- Mitigation: GC tuning, memory pools for hot paths

**Performance**:
- Go: ~95% of Rust performance for most workloads
- Mitigation: Profile and optimize hot paths

**Binary Size**:
- Go: ~10-20% larger than equivalent Rust
- Mitigation: Build optimizations (`-ldflags="-s -w"`), UPX compression

## Implementation Standards

**Go Version**: 1.22+
- Generics (1.18+)
- Improved performance (1.21+)
- Standard library enhancements

**Code Style**:
- `gofmt` for formatting
- `golangci-lint` for linting
- Effective Go conventions

**Build**:
```bash
# Static binary
CGO_ENABLED=0 go build -ldflags="-s -w" -o warren ./cmd/warren

# Cross-compile
GOOS=linux GOARCH=arm64 go build...
```

## Consequences

✅ **Fast development**: Ship v1.0 faster
✅ **Rich ecosystem**: Leverage existing Go libraries
✅ **Team productivity**: Work in familiar language
✅ **Easy debugging**: Simple stack traces, pprof profiling
✅ **Community**: Large Go community for hiring, support

⚠️ **GC pauses**: Acceptable for orchestrator (milliseconds)
⚠️ **Memory usage**: Slightly higher than Rust (acceptable trade-off)

## Future Considerations

**Rust for specific components**:
- Network data plane (if performance critical)
- Security-sensitive code (secrets encryption)
- High-throughput components

**Hybrid approach** possible: Go control plane, Rust data plane

## Validation

All POCs implemented in Go successfully:
- [poc/raft/](../../poc/raft/)
- [poc/containerd/](../../poc/containerd/)
- [poc/wireguard/](../../poc/wireguard/)
- [poc/binary-size/](../../poc/binary-size/)

**Status**: Accepted - Go meets all requirements, enables fast development
