# Containerd POC - Container Lifecycle

## Overview

This POC validates that Warren can control containers via the containerd API:
- Connect to containerd daemon
- Pull images
- Create containers
- Start/stop containers
- No memory leaks

## Prerequisites

### macOS
```bash
# Install containerd
brew install containerd

# Start containerd (requires sudo)
sudo containerd

# In another terminal, verify it's running
sudo ctr version
```

### Linux
```bash
# Install containerd (usually pre-installed)
sudo apt install containerd  # Ubuntu/Debian
# or
sudo yum install containerd  # RHEL/CentOS

# Start containerd
sudo systemctl start containerd
sudo systemctl status containerd
```

## Running the POC

```bash
cd poc/containerd

# Download dependencies
go mod download

# Run (requires sudo for containerd socket access)
sudo go run .
```

## Expected Output

```
=== Warren Containerd POC ===
Containerd socket: /run/containerd/containerd.sock
Namespace: warren-poc

1. Connecting to containerd...
✓ Connected to containerd

2. Pulling image: docker.io/library/nginx:latest...
✓ Image pulled in 5.2s
  Image size: 71234567 bytes

3. Creating container: warren-test-nginx...
✓ Container created

4. Starting container...
✓ Container started (PID: 12345)

5. Checking container status...
✓ Container status: running

--- Memory Usage ---
Alloc: 15 MB
TotalAlloc: 25 MB
Sys: 35 MB

✅ All tests passed!
Container will run for 5 seconds before cleanup...

6. Stopping container...
✓ Container stopped

7. Cleaning up: Deleting container...
✓ Container deleted
```

## Test Scenarios

### 1. Containerd Connection

**Expected**: Successfully connect to `/run/containerd/containerd.sock`
**Result**: ✅ PASS / ❌ FAIL

**Observed**:
```
# Connection output
```

### 2. Image Pull

**Expected**: Pull nginx:latest image successfully
**Result**: ✅ PASS / ❌ FAIL
**Pull Time**: ___ seconds
**Image Size**: ___ MB

**Observed**:
```
# Pull output and timing
```

### 3. Container Lifecycle

**Expected**: Create → Start → Status=running → Stop → Delete
**Result**: ✅ PASS / ❌ FAIL

**Observed**:
```
# Lifecycle output
```

### 4. Memory Leak Test

**Test**: Run create/delete cycle 100 times, measure memory growth
**Expected**: Memory usage stable (< 10% growth)

**Modify main.go**:
```go
// Add this test after main tests
log.Println("\n8. Memory leak test (100 cycles)...")
startMem := getMemUsage()

for i := 0; i < 100; i++ {
    container, _ := createContainer(ctx, client, fmt.Sprintf("test-%d", i), image)
    task, _ := startContainer(ctx, container)
    task.Kill(ctx, syscall.SIGTERM)
    task.Wait(ctx)
    container.Delete(ctx, containerd.WithSnapshotCleanup)

    if i % 10 == 0 {
        log.Printf("Cycle %d/100", i)
    }
}

endMem := getMemUsage()
growth := ((endMem - startMem) / startMem) * 100
log.Printf("✓ Memory growth: %.2f%%", growth)
```

**Result**: ✅ PASS / ❌ FAIL
**Memory Growth**: ____%

**Observed**:
```
# Memory test output
```

### 5. Resource Constraints

**Test**: Create container with CPU/memory limits
**Expected**: Limits enforced by containerd

**Add to createContainer**:
```go
containerd.WithNewSpec(
    oci.WithImageConfig(image),
    oci.WithMemoryLimit(128 * 1024 * 1024), // 128MB
    oci.WithCPUCFS(50000, 100000),           // 50% of one core
)
```

**Result**: ✅ PASS / ❌ FAIL

**Observed**:
```
# Resource constraint output
```

## Performance Measurements

### Image Pull Time

**Test**: Pull nginx:latest (cold cache)
**Expected**: < 10 seconds

**Results**:
- Time: ___ seconds
- Size: ___ MB

### Container Start Time

**Test**: Create + Start container
**Expected**: < 2 seconds

**Results**:
- Create: ___ ms
- Start: ___ ms
- Total: ___ ms

### Memory Overhead

**Test**: Process memory after 100 container cycles
**Expected**: < 100MB process memory

**Results**:
- Initial: ___ MB
- After 100 cycles: ___ MB
- Growth: ____%

## Conclusions

### Success Criteria

- [ ] Successfully connect to containerd
- [ ] Pull images from registry
- [ ] Create containers from images
- [ ] Start and stop containers
- [ ] Check container status
- [ ] Clean up (delete) containers and snapshots
- [ ] No memory leaks (< 10% growth over 100 cycles)
- [ ] Resource constraints working

### Go/No-Go Decision

**Decision**: ✅ GO / ❌ NO-GO

**Rationale**:
```
# Why containerd meets (or doesn't meet) Warren's requirements
```

### Issues Discovered

```
# Problems, workarounds, concerns
```

### Recommendations for Warren Implementation

```
# Lessons learned:
# - containerd namespace strategy
# - Error handling patterns
# - Resource cleanup best practices
# - Image caching considerations
```

## Next Steps

If GO:
- [ ] Proceed to WireGuard POC
- [ ] Design Warren's container runtime interface
- [ ] Document containerd configuration (namespace, socket path)

If NO-GO:
- [ ] Investigate alternatives (CRI-O, custom runtime)
- [ ] Document blockers
