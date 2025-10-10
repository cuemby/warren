# Binary Size POC

## Overview

This POC validates that Warren can meet the < 100MB binary size target by:
- Importing all major dependencies
- Building with optimizations
- Measuring compressed size

## Running the POC

```bash
cd poc/binary-size
go mod download

# Build and measure
make all
```

## Build Variations

### 1. Default Build
```bash
make build
```
No optimizations, includes debug symbols.

### 2. Optimized Build
```bash
make build-optimized
```
Strips debug symbols with `-ldflags="-s -w"`.

### 3. Compressed Build
```bash
make compress
```
UPX compression with `--best --lzma`.

### 4. Full Report
```bash
make report
```
Generates `size-report.txt` with all measurements.

## Expected Results

| Build Type | Expected Size | Target |
|------------|---------------|--------|
| Default | 40-60 MB | Reference |
| Optimized | 30-45 MB | Baseline |
| Compressed | 15-25 MB | < 50 MB |

**Target**: Compressed binary < 50MB (allows headroom for Warren's actual code)

## Test Results

### Build Sizes

**Date**: ___________
**Platform**: ___________ (darwin/linux, amd64/arm64)

| Build | Size | Notes |
|-------|------|-------|
| Default | ___ MB | |
| Optimized | ___ MB | Reduction: ___% |
| Compressed (UPX) | ___ MB | Reduction: ___% |

### Dependency Breakdown

Approximate contribution by library (use `go build -ldflags="-X main.version=test"` and binary analysis):

| Dependency | Approx Size | Percentage |
|------------|-------------|------------|
| containerd | ___ MB | ___% |
| hashicorp/raft | ___ MB | ___% |
| grpc | ___ MB | ___% |
| wireguard | ___ MB | ___% |
| prometheus | ___ MB | ___% |
| cobra | ___ MB | ___% |
| Go runtime | ___ MB | ___% |
| Other | ___ MB | ___% |

**Tool for analysis**:
```bash
go tool nm -size warren-optimized | sort -rn | head -20
```

## Optimization Techniques Applied

1. **`-ldflags="-s -w"`**
   - `-s`: Omit symbol table
   - `-w`: Omit DWARF debugging info
   - Reduction: ~30-40%

2. **`CGO_ENABLED=0`**
   - Static linking, no C dependencies
   - Smaller, more portable binary

3. **UPX Compression**
   - LZMA algorithm (`--best --lzma`)
   - Decompresses at runtime (slight startup cost)
   - Reduction: ~50-60% from optimized

## Platform Comparison

Test on multiple platforms:

| Platform | Optimized | Compressed | Within Target? |
|----------|-----------|------------|----------------|
| Linux amd64 | ___ MB | ___ MB | ✅ / ❌ |
| Linux arm64 | ___ MB | ___ MB | ✅ / ❌ |
| macOS amd64 | ___ MB | ___ MB | ✅ / ❌ |
| macOS arm64 | ___ MB | ___ MB | ✅ / ❌ |

## Conclusions

### Success Criteria

- [ ] Optimized build < 45MB
- [ ] Compressed build < 50MB
- [ ] All platforms within target
- [ ] Startup time acceptable with UPX (< 1s)

### Go/No-Go Decision

**Decision**: ✅ GO / ❌ NO-GO

**Rationale**:
```
# Binary size meets/exceeds target
# Compressed size: ___ MB (target: < 50MB)
# Headroom for Warren code: ~___ MB
```

### Concerns & Mitigations

**Concern**: UPX decompression adds startup latency
**Mitigation**:
- Test startup time (should be < 1s)
- UPX optional (can distribute both compressed & uncompressed)

**Concern**: Binary size grows with Warren code
**Mitigation**:
- Monitor size in CI (fail if > 100MB)
- Profile and optimize hot paths
- Consider build tags to exclude optional features

### Recommendations for Warren

```
# Build configuration:
# - Always use -ldflags="-s -w" for releases
# - CGO_ENABLED=0 for portability
# - Offer both compressed (UPX) and uncompressed binaries
# - Set CI check: compressed size < 100MB

# Distribution strategy:
# - GitHub Releases: compressed binaries (smaller downloads)
# - Package managers: uncompressed (faster startup)
# - Docker images: compressed (layer size matters)
```

## Next Steps

If GO:
- [ ] Proceed to writing ADRs
- [ ] Set up CI to monitor binary size
- [ ] Document build process for Warren

If NO-GO:
- [ ] Investigate dependency alternatives (lighter libraries)
- [ ] Consider splitting into manager/worker binaries
- [ ] Evaluate features vs size trade-offs
