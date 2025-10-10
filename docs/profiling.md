# Warren Performance Profiling Guide

This guide explains how to profile Warren's memory and CPU usage using Go's built-in pprof tool.

## Table of Contents

- [Overview](#overview)
- [Enabling Profiling](#enabling-profiling)
- [Profiling Manager](#profiling-manager)
- [Profiling Worker](#profiling-worker)
- [Common Profiling Tasks](#common-profiling-tasks)
- [Analyzing Profiles](#analyzing-profiles)
- [Best Practices](#best-practices)

---

## Overview

Warren includes built-in support for Go's `pprof` profiling tool, which allows you to:

- **Profile memory usage** - Identify memory leaks and high allocation hot spots
- **Profile CPU usage** - Find CPU-intensive code paths
- **Monitor goroutines** - Debug goroutine leaks and concurrency issues
- **Analyze blocking** - Identify synchronization bottlenecks
- **Track allocations** - See where memory is being allocated

Profiling is disabled by default and can be enabled with the `--enable-pprof` flag.

---

## Enabling Profiling

### Manager Profiling

When starting a manager (cluster init or manager join), add the `--enable-pprof` flag:

```bash
# Initialize cluster with profiling enabled
warren cluster init --enable-pprof

# Join as manager with profiling enabled
warren manager join --token <token> --leader <addr> --enable-pprof
```

**Profiling endpoints will be available at:**
- Manager: `http://127.0.0.1:9090/debug/pprof/`

### Worker Profiling

When starting a worker, add the `--enable-pprof` flag:

```bash
# Start worker with profiling enabled
warren worker start --manager 127.0.0.1:8080 --enable-pprof
```

**Profiling endpoints will be available at:**
- Worker: `http://127.0.0.1:6060/debug/pprof/`

---

## Profiling Manager

### Available Endpoints

When a manager is started with `--enable-pprof`, the following endpoints are available at `http://127.0.0.1:9090/debug/pprof/`:

| Endpoint | Description |
|----------|-------------|
| `/debug/pprof/` | Index page with all available profiles |
| `/debug/pprof/heap` | Memory allocation profile |
| `/debug/pprof/profile` | CPU profile (30s sample by default) |
| `/debug/pprof/goroutine` | Stack traces of all current goroutines |
| `/debug/pprof/threadcreate` | Stack traces that led to creation of OS threads |
| `/debug/pprof/block` | Stack traces that led to blocking on synchronization |
| `/debug/pprof/mutex` | Stack traces of holders of contended mutexes |
| `/debug/pprof/allocs` | All past memory allocations |

### Example: Profile Manager Memory

```bash
# Capture heap profile
curl http://127.0.0.1:9090/debug/pprof/heap > manager_heap.prof

# Analyze with pprof
go tool pprof manager_heap.prof

# Or analyze directly from URL
go tool pprof http://127.0.0.1:9090/debug/pprof/heap
```

### Example: Profile Manager CPU

```bash
# Capture 30-second CPU profile
curl http://127.0.0.1:9090/debug/pprof/profile?seconds=30 > manager_cpu.prof

# Analyze with pprof
go tool pprof manager_cpu.prof
```

### Example: Check Goroutines

```bash
# View goroutine count
curl http://127.0.0.1:9090/debug/pprof/goroutine?debug=1

# Capture goroutine profile
go tool pprof http://127.0.0.1:9090/debug/pprof/goroutine
```

---

## Profiling Worker

### Available Endpoints

When a worker is started with `--enable-pprof`, the same endpoints are available at `http://127.0.0.1:6060/debug/pprof/`.

### Example: Profile Worker Memory

```bash
# Capture heap profile from worker
curl http://127.0.0.1:6060/debug/pprof/heap > worker_heap.prof

# Analyze with pprof
go tool pprof worker_heap.prof

# Or analyze directly
go tool pprof http://127.0.0.1:6060/debug/pprof/heap
```

### Example: Profile Worker CPU

```bash
# Capture CPU profile from worker
curl http://127.0.0.1:6060/debug/pprof/profile?seconds=30 > worker_cpu.prof

# Analyze
go tool pprof worker_cpu.prof
```

---

## Common Profiling Tasks

### 1. Find Memory Leaks

```bash
# Capture baseline heap profile
go tool pprof -sample_index=alloc_objects http://127.0.0.1:9090/debug/pprof/heap

# Wait for memory to grow (simulate load)
# ...

# Capture second heap profile and compare
go tool pprof -sample_index=alloc_objects -base heap_baseline.prof http://127.0.0.1:9090/debug/pprof/heap
```

### 2. Identify Hot Paths

```bash
# Capture CPU profile during load
go tool pprof http://127.0.0.1:9090/debug/pprof/profile?seconds=60

# In pprof interactive mode:
(pprof) top10          # Show top 10 functions by CPU time
(pprof) list <function> # Show source code for function
(pprof) web            # Generate call graph (requires graphviz)
```

### 3. Detect Goroutine Leaks

```bash
# Check goroutine count over time
watch -n 5 'curl -s http://127.0.0.1:9090/debug/pprof/goroutine?debug=1 | head -1'

# If count grows, capture profile
go tool pprof http://127.0.0.1:9090/debug/pprof/goroutine

# In pprof:
(pprof) top           # See which functions create most goroutines
(pprof) list <func>   # Examine source
```

### 4. Analyze Lock Contention

```bash
# Capture mutex profile
go tool pprof http://127.0.0.1:9090/debug/pprof/mutex

# In pprof:
(pprof) top           # See most contended mutexes
(pprof) web           # Visualize contention graph
```

### 5. Track Allocations

```bash
# Capture allocation profile
go tool pprof -sample_index=alloc_space http://127.0.0.1:9090/debug/pprof/allocs

# In pprof:
(pprof) top20         # Top 20 allocation sites
(pprof) list <func>   # Source for specific function
```

---

## Analyzing Profiles

### Interactive pprof Commands

Once in `go tool pprof <profile>` interactive mode:

| Command | Description |
|---------|-------------|
| `top` | Show top entries by current sample count |
| `top10` | Show top 10 entries |
| `top -cum` | Sort by cumulative count |
| `list <func>` | Show source code with annotations |
| `web` | Generate SVG call graph (requires graphviz) |
| `peek <func>` | Show callers and callees |
| `traces` | Show all sample stack traces |
| `help` | Show all commands |

### Generate Visual Reports

```bash
# Generate SVG call graph
go tool pprof -svg http://127.0.0.1:9090/debug/pprof/heap > heap.svg

# Generate flame graph (requires go-torch)
go-torch http://127.0.0.1:9090/debug/pprof/profile

# Generate PDF report
go tool pprof -pdf http://127.0.0.1:9090/debug/pprof/profile > profile.pdf
```

### Compare Profiles

```bash
# Capture baseline
curl http://127.0.0.1:9090/debug/pprof/heap > baseline.prof

# Run load test or wait for memory growth
# ...

# Capture second profile
curl http://127.0.0.1:9090/debug/pprof/heap > after_load.prof

# Compare
go tool pprof -base baseline.prof after_load.prof
```

---

## Best Practices

### When to Enable Profiling

**DO enable profiling when:**
- Investigating performance issues
- Optimizing memory usage
- Diagnosing memory leaks
- Debugging goroutine leaks
- Running load tests
- Doing capacity planning

**DON'T enable profiling when:**
- Running in production (unless investigating specific issues)
- Benchmarking performance (profiling adds overhead)
- Not actively investigating issues

### Performance Impact

- **Heap profiling**: Negligible impact (~1-2% overhead)
- **CPU profiling**: ~5% overhead during profiling
- **Goroutine profiling**: Negligible impact
- **Mutex profiling**: Can be significant (10-20%) if many mutexes

### Memory Targets

Based on Warren's design goals:

**Manager:**
- Target: < 256MB resident memory
- Under load: < 512MB
- Profile if exceeding these targets

**Worker:**
- Target: < 128MB resident memory (excluding containers)
- Under load: < 256MB
- Profile if exceeding these targets

### Profiling Workflow

1. **Establish baseline** - Profile under normal load
2. **Simulate load** - Run realistic workload or load test
3. **Capture profiles** - Memory, CPU, goroutines
4. **Analyze** - Use pprof to identify hot spots
5. **Optimize** - Fix issues
6. **Verify** - Profile again to confirm improvements
7. **Repeat** - Continue until targets met

### Common Issues

**Issue: "No data available"**
- Heap profiles need allocations to occur
- CPU profiles need actual CPU usage
- Wait for activity or generate load

**Issue: "Address already in use"**
- Another process using port 9090 (manager) or 6060 (worker)
- Check: `lsof -i :9090` or `lsof -i :6060`
- Stop conflicting process or change Warren ports

**Issue: "Profile appears empty"**
- Not enough sample data collected
- Increase profiling duration: `?seconds=60`
- Generate more load on the system

---

## Examples

### Example 1: Investigate Manager Memory Growth

```bash
# Start manager with profiling
warren cluster init --enable-pprof

# Capture baseline
curl http://127.0.0.1:9090/debug/pprof/heap > baseline.prof

# Deploy 100 services to simulate load
for i in {1..100}; do
  warren service create nginx-$i --image nginx:latest --replicas 3
done

# Wait 5 minutes for potential leak
sleep 300

# Capture after load
curl http://127.0.0.1:9090/debug/pprof/heap > after_load.prof

# Compare
go tool pprof -base baseline.prof after_load.prof

# In pprof:
(pprof) top20          # See what grew
(pprof) list <func>    # Examine source
```

### Example 2: Profile Worker Under Load

```bash
# Start worker with profiling
warren worker start --enable-pprof

# From another terminal, deploy services
warren service create stress --image stress-ng --replicas 10

# Capture CPU profile (60 seconds)
go tool pprof http://127.0.0.1:6060/debug/pprof/profile?seconds=60

# In pprof:
(pprof) top           # See CPU hot spots
(pprof) web           # Generate call graph
```

### Example 3: Check for Goroutine Leaks

```bash
# Start manager with profiling
warren cluster init --enable-pprof

# Monitor goroutine count every 10 seconds
watch -n 10 'curl -s http://127.0.0.1:9090/debug/pprof/goroutine?debug=1 | head -5'

# If count grows steadily, investigate
go tool pprof http://127.0.0.1:9090/debug/pprof/goroutine

# In pprof:
(pprof) top           # See which functions create goroutines
(pprof) traces        # See full stack traces
```

---

## Additional Resources

- [Go pprof Documentation](https://pkg.go.dev/net/http/pprof)
- [Profiling Go Programs](https://go.dev/blog/pprof)
- [Go Memory Management](https://go.dev/doc/gc-guide)
- [Diagnostics Guide](https://go.dev/doc/diagnostics)

---

## Troubleshooting

### Q: Can I profile multiple managers in a cluster?

Yes, each manager runs its own profiling server. Use different ports for each:

```bash
# Manager 1 (default)
warren cluster init --enable-pprof --api-addr 127.0.0.1:8080

# Manager 2
warren manager join --enable-pprof --api-addr 127.0.0.1:8081 --token <token> --leader <addr>

# Profile Manager 1
go tool pprof http://127.0.0.1:9090/debug/pprof/heap

# Profile Manager 2
go tool pprof http://127.0.0.1:9091/debug/pprof/heap  # (if metrics on different port)
```

**Note:** Currently all managers use port 9090 for metrics/profiling. This may need adjustment for co-located managers.

### Q: How do I profile in production?

Enable profiling only on specific nodes and for limited time:

```bash
# Enable on one manager temporarily
# Capture profile
# Disable by restarting without flag
```

Consider:
- Enable on a single manager (not all)
- Capture profiles quickly
- Disable after investigation
- Store profiles for offline analysis

### Q: What if I can't install graphviz for `web` command?

Use alternative visualizations:

```bash
# Text-based tree
(pprof) tree

# Generate SVG and view in browser
go tool pprof -svg <profile> > graph.svg
open graph.svg

# Or use online tools
go tool pprof -raw <profile> > raw.txt
# Upload to pprof.me or similar
```

---

**Last Updated**: 2025-10-10
**Related Docs**: [Metrics](metrics.md) | [Performance](performance.md)
