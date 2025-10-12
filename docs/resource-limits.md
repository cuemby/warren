# Resource Limits

Warren supports CPU and memory resource limits to constrain container resource usage and prevent resource contention.

## Overview

Resource limits enforce hard constraints on CPU and memory usage for containers. These limits are implemented using Linux cgroups and are enforced by the kernel.

### Benefits

- **Resource Isolation**: Prevent noisy neighbors from consuming all resources
- **Predictable Performance**: Guarantee baseline resources for critical services
- **Cost Control**: Prevent runaway resource usage
- **Multi-tenancy**: Safely run multiple workloads on shared infrastructure

## CPU Limits

CPU limits control how much CPU time a container can use.

### How CPU Limits Work

Warren uses two Linux cgroup mechanisms for CPU limiting:

1. **CPU Shares** (relative weight):
   - Proportional CPU allocation when CPU is contended
   - 1024 shares = 1 core's worth of relative weight
   - Example: Task with 2048 shares gets 2x CPU vs task with 1024 shares

2. **CFS Quota** (hard limit):
   - Absolute CPU time limit per period
   - Period: 100ms (100,000 microseconds)
   - Quota: `cpus * 100,000` microseconds per period
   - Example: 0.5 CPUs = 50,000us of CPU time per 100ms

### CPU Limit Examples

```bash
# Half a CPU core
warren service create api --image myapi:latest --cpus 0.5

# One full CPU core
warren service create worker --image worker:latest --cpus 1.0

# Two CPU cores
warren service create db --image postgres:latest --cpus 2.0

# Fractional CPUs for light workloads
warren service create sidecar --image logging:latest --cpus 0.25
```

### CPU Limit Behavior

- **Under Contention**: CFS quota enforces hard limit
  - Container throttled when quota exhausted
  - Must wait until next period (100ms) to get more CPU time

- **No Contention**: Container can use more than its share
  - If no other containers are competing, can burst above limit
  - CPU shares determine relative priority

## Memory Limits

Memory limits control how much RAM a container can use.

### How Memory Limits Work

Memory limits use the Linux cgroup memory controller:

- **Hard Limit**: Enforced by the kernel
- **OOM Kill**: Container is killed if it exceeds the limit
- **No Overcommit**: Container cannot allocate more than the limit

### Memory Limit Examples

```bash
# 512 megabytes
warren service create cache --image redis:latest --memory 512m

# 1 gigabyte
warren service create api --image myapi:latest --memory 1g

# 2 gigabytes
warren service create db --image postgres:latest --memory 2g

# Kilobytes (rare)
warren service create tiny --image alpine:latest --memory 10240k
```

### Supported Memory Units

- **b** or **B**: Bytes
- **k** or **kb** or **KB**: Kilobytes (1024 bytes)
- **m** or **mb** or **MB**: Megabytes (1024² bytes)
- **g** or **gb** or **GB**: Gigabytes (1024³ bytes)

Examples:
- `512m` = 536,870,912 bytes
- `1g` = 1,073,741,824 bytes
- `2048m` = 2g = 2,147,483,648 bytes

### Memory Limit Behavior

- **Within Limit**: Container runs normally
- **At Limit**: Memory allocations start failing
- **Exceeds Limit**: Container is OOM-killed by kernel
  - Warren's restart policy determines if container restarts
  - Check container logs for OOM messages

## Combined CPU and Memory Limits

You can specify both CPU and memory limits together:

```bash
# API server: 1 CPU, 512MB memory
warren service create api \
  --image myapi:latest \
  --cpus 1.0 \
  --memory 512m

# Database: 2 CPUs, 4GB memory
warren service create postgres \
  --image postgres:15 \
  --cpus 2.0 \
  --memory 4g \
  --env POSTGRES_PASSWORD=secret

# Microservice: 0.5 CPU, 256MB memory
warren service create svc \
  --image mysvc:latest \
  --cpus 0.5 \
  --memory 256m \
  --replicas 3
```

## Viewing Resource Limits

Resource limits are displayed when creating a service:

```bash
$ warren service create api --image myapi:latest --cpus 1.0 --memory 512m

✓ Service created: api
  ID: api-abc123
  Image: myapi:latest
  Replicas: 1
  CPU Limit: 1.00 cores
  Memory Limit: 512.0 MB
```

To view limits for existing services:

```bash
$ warren service inspect api
```

## Best Practices

### Choosing CPU Limits

1. **Start Conservative**: Begin with lower limits and increase based on monitoring
2. **Profile Your Application**: Measure actual CPU usage under load
3. **Consider Burst**: Applications with bursty CPU patterns may need higher limits
4. **Fractional CPUs**: Most applications don't need full cores (0.5, 0.25 often sufficient)

Typical CPU limits:
- **Web APIs**: 0.5 - 1.0 cores
- **Background Workers**: 0.25 - 0.5 cores
- **Databases**: 2.0 - 4.0 cores
- **Batch Processing**: 1.0 - 2.0 cores

### Choosing Memory Limits

1. **Baseline + Headroom**: Set limit to typical usage + 20-30% headroom
2. **Monitor OOM Kills**: Increase limit if containers are being OOM-killed
3. **Application Type**:
   - Stateless: Lower memory (256MB - 512MB)
   - Stateful: Higher memory (1GB - 4GB)
   - Caches: Very high memory (4GB+)

Typical memory limits:
- **Stateless APIs**: 256MB - 512MB
- **Node.js Apps**: 512MB - 1GB
- **Java Apps**: 1GB - 2GB (adjust for JVM heap)
- **Databases**: 2GB - 8GB+
- **Caches (Redis)**: 1GB - 16GB+

### Resource Limit Recommendations

1. **Always Set Limits**: Even generous limits prevent runaway processes
2. **Test Under Load**: Verify limits are appropriate under realistic load
3. **Monitor Resource Usage**: Use metrics to understand actual vs. configured limits
4. **Account for Overhead**: JVM, runtimes add memory overhead beyond app data
5. **Plan for Bursts**: CPU limits should accommodate temporary spikes
6. **Match Node Capacity**: Don't set limits higher than node resources

## Troubleshooting

### Container Being OOM-Killed

**Symptom**: Container restarts frequently with exit code 137

```bash
$ warren service ps api
ID          STATE     EXIT_CODE
api-1       failed    137
api-2       running   0
```

**Diagnosis**:
- Exit code 137 = 128 + 9 (SIGKILL from OOM killer)
- Container exceeded memory limit

**Solutions**:
1. Increase memory limit: `warren service update api --memory 1g`
2. Optimize application memory usage
3. Check for memory leaks in application code

### CPU Throttling

**Symptom**: Application slow despite low overall CPU usage

**Diagnosis**:
- Container hitting CPU quota (CFS throttling)
- Check cgroup CPU stats

**Solutions**:
1. Increase CPU limit: `warren service update api --cpus 2.0`
2. Optimize CPU-intensive code paths
3. Scale horizontally (more replicas with same limits)

### Setting Appropriate Limits

**Too Low**:
- Containers OOM-killed or CPU-throttled
- Poor performance
- Frequent restarts

**Too High**:
- Resource waste
- Poor bin-packing (fewer tasks per node)
- False sense of resource availability

**Just Right**:
- Containers run reliably
- Resources efficiently utilized
- Headroom for bursts

## Implementation Details

### Linux Cgroups

Warren uses cgroups v1 (compatible with most Linux distributions):

CPU cgroup files:
- `cpu.shares`: Relative CPU weight (1024 per core)
- `cpu.cfs_quota_us`: CPU time quota per period
- `cpu.cfs_period_us`: Period duration (100ms)

Memory cgroup files:
- `memory.limit_in_bytes`: Hard memory limit

### Containerd Integration

Warren uses containerd's OCI runtime spec to configure cgroups:

```go
// CPU limits
oci.WithCPUShares(uint64(cpus * 1024))
oci.WithCPUCFS(int64(cpus * 100000), 100000)

// Memory limits
oci.WithMemoryLimit(uint64(memoryBytes))
```

These options are translated into cgroup settings by containerd's runc runtime.

## Future Enhancements

### Planned Features

- **Memory Reservations**: Soft limits for memory
- **CPU Reservations**: Guaranteed minimum CPU
- **Resource-Aware Scheduling**: Place tasks on nodes with sufficient resources
- **Resource Monitoring**: Expose actual CPU/memory usage via metrics
- **Burst Credits**: Allow temporary over-limit usage

### Not Planned (Out of Scope)

- **Device Limits**: GPU, network bandwidth limiting
- **Storage I/O Limits**: Disk IOPS/throughput limiting
- **PID Limits**: Process count limiting (can be added if needed)

## See Also

- [Service Management](./services.md)
- [Networking](./networking.md)
- [Health Checks](./health-checks.md)
