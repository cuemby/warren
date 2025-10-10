# Binary Size Optimization

## Current Status

**Target**: < 100MB
**Current**: ~19MB (optimized release build)
**Headroom**: 81% under budget ✓

## Build Optimizations

### Enabled Optimizations

1. **CGO Disabled** (`CGO_ENABLED=0`)
   - Produces static binaries
   - No C library dependencies
   - Easier deployment

2. **Strip Debug Symbols** (`-ldflags="-s -w"`)
   - `-s`: Omit symbol table
   - `-w`: Omit DWARF debug info
   - Reduces size by ~30-40%

3. **Cross-Platform Builds**
   - Linux AMD64: ~20MB
   - Linux ARM64: ~19MB
   - macOS AMD64: ~20MB
   - macOS ARM64: ~19MB

## Size by Platform

```bash
make build-all
```

Output:
```
-rwxr-xr-x  warren-linux-amd64     20M
-rwxr-xr-x  warren-linux-arm64     19M
-rwxr-xr-x  warren-darwin-amd64    20M
-rwxr-xr-x  warren-darwin-arm64    19M
```

## Size Check

Run automated size check:

```bash
make size
```

This builds a release binary and verifies it's under 100MB.

## Size Breakdown

Warren includes:
- Raft consensus (hashicorp/raft)
- Containerd client (containerd/containerd)
- gRPC + Protocol Buffers
- Prometheus metrics
- Zerolog logging
- BoltDB embedded storage
- Scheduler, reconciler, API server

Despite all these features, we maintain a small footprint through:
- Careful dependency management
- No unnecessary frameworks
- Efficient Go stdlib usage

## Future Optimizations

If needed (currently not necessary):

1. **UPX Compression** - Can reduce by another 50-60%
   ```bash
   upx --best bin/warren
   ```

2. **Module Pruning** - Review and remove unused dependencies

3. **Dead Code Elimination** - Use Go 1.21+ with more aggressive DCE

## Monitoring

Binary size is tracked in CI and reported on each PR.
Alert triggers if size exceeds 80MB (80% of budget).

## Comparison

| System       | Binary Size |
|--------------|-------------|
| Warren       | 19MB        |
| Nomad        | ~40MB       |
| K3s          | ~50MB       |
| Docker       | 60MB+       |
| Kubernetes   | 100MB+ (per component) |

Warren achieves full orchestration in a fraction of the size! 🚀
