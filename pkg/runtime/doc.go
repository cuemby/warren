/*
Package runtime provides containerd integration for Warren's container lifecycle management.

The runtime package wraps containerd's client API to provide container operations
including image pulling, container creation with resource limits, lifecycle management,
and status monitoring. It handles the complexity of OCI spec generation, snapshot
management, and containerd namespace isolation.

# Architecture

Warren uses containerd as its container runtime, providing CRI-compatible container
operations with efficient resource management:

	┌─────────────────── CONTAINERD RUNTIME ────────────────────┐
	│                                                             │
	│  ┌──────────────────────────────────────────────┐         │
	│  │        ContainerdRuntime Client               │         │
	│  │  - Socket: /run/containerd/containerd.sock   │         │
	│  │  - Namespace: warren                          │         │
	│  └──────────────────┬───────────────────────────┘         │
	│                     │                                       │
	│  ┌──────────────────▼───────────────────────────┐         │
	│  │           Image Operations                    │         │
	│  │  - Pull images from registries                │         │
	│  │  - Unpack for snapshot creation                │         │
	│  │  - Verify and cache locally                   │         │
	│  └──────────────────┬───────────────────────────┘         │
	│                     │                                       │
	│  ┌──────────────────▼───────────────────────────┐         │
	│  │        Container Lifecycle                    │         │
	│  │  - Create: Generate OCI spec                  │         │
	│  │  - Start: Launch container process            │         │
	│  │  - Stop: Graceful shutdown (SIGTERM→SIGKILL) │         │
	│  │  - Delete: Cleanup container and snapshot     │         │
	│  └──────────────────┬───────────────────────────┘         │
	│                     │                                       │
	│  ┌──────────────────▼───────────────────────────┐         │
	│  │         Resource Management                   │         │
	│  │  - CPU: Shares (1024 = 1 core) + CFS quota   │         │
	│  │  - Memory: Hard limits in bytes               │         │
	│  │  - Applied via OCI spec modifications         │         │
	│  └──────────────────┬───────────────────────────┘         │
	│                     │                                       │
	│  ┌──────────────────▼───────────────────────────┐         │
	│  │           Mount Management                    │         │
	│  │  - Secrets: Bind mount to /run/secrets (ro)  │         │
	│  │  - Volumes: Persistent storage mounts         │         │
	│  │  - DNS: resolv.conf for service discovery     │         │
	│  └────────────────────────────────────────────────┘        │
	│                                                             │
	│  ┌──────────────────────────────────────────────┐         │
	│  │             Containerd Daemon                 │         │
	│  │  - Namespaces: Isolate Warren containers      │         │
	│  │  - Snapshotter: overlayfs for layers          │         │
	│  │  - Runtime: runc (io.containerd.runc.v2)      │         │
	│  └────────────────────────────────────────────────┘        │
	└─────────────────────────────────────────────────────────┘

# Core Components

ContainerdRuntime:
  - Main client wrapper for containerd operations
  - Manages socket connection and namespace isolation
  - Provides high-level container lifecycle methods
  - Thread-safe for concurrent operations

Socket Auto-Detection:
  - Default: /run/containerd/containerd.sock (Linux)
  - macOS Lima: ~/.lima/warren/sock/containerd.sock
  - Configurable via NewContainerdRuntime parameter
  - Auto-reconnect on connection loss (future)

Resource Limits:
  - CPU: CPULimit (cores) → CPU shares (1024 per core) + CFS quota
  - Memory: MemoryLimit (bytes) → cgroup memory.limit_in_bytes
  - Applied during container creation via OCI spec
  - Enforced by Linux cgroups (via containerd/runc)

# Container Lifecycle

Create Container:
  1. Validate image exists (pull if needed)
  2. Generate OCI runtime spec from task definition
  3. Apply resource limits (CPU/memory)
  4. Configure mounts (secrets, volumes, resolv.conf)
  5. Create container with snapshot
  6. Return container ID

Start Container:
  1. Load container by ID
  2. Create containerd task (running instance)
  3. Start task (execute entry point)
  4. Monitor for startup errors
  5. Return immediately (async)

Stop Container:
  1. Load container and get task
  2. Send SIGTERM for graceful shutdown
  3. Wait for exit with timeout (configurable)
  4. Send SIGKILL if timeout exceeded
  5. Delete task to free resources

Delete Container:
  1. Stop container if running
  2. Delete container object
  3. Cleanup snapshot layers
  4. Remove from containerd namespace

# Usage

Creating a Runtime Client:

	runtime, err := runtime.NewContainerdRuntime("")
	if err != nil {
		log.Fatal(err)
	}
	defer runtime.Close()

	// Custom socket path
	runtime, err := runtime.NewContainerdRuntime("/custom/containerd.sock")

Pulling an Image:

	ctx := context.Background()
	err := runtime.PullImage(ctx, "nginx:latest")
	if err != nil {
		log.Fatal(err)
	}

Creating and Starting a Container:

	task := &types.Task{
		ID:    "task-abc123",
		Image: "nginx:latest",
		Env:   []string{"ENV=production"},
		Resources: &types.Resources{
			CPULimit:    1.0,  // 1 CPU core
			MemoryLimit: 512 * 1024 * 1024,  // 512MB
		},
	}

	// Create container
	containerID, err := runtime.CreateContainer(ctx, task)
	if err != nil {
		log.Fatal(err)
	}

	// Start container
	err = runtime.StartContainer(ctx, containerID)
	if err != nil {
		log.Fatal(err)
	}

Creating Container with Mounts:

	// Prepare secret mount
	secretsPath := "/var/run/warren/secrets/task-abc123"

	// Prepare volume mounts
	volumeMounts := []specs.Mount{
		{
			Source:      "/var/lib/warren/volumes/data",
			Destination: "/data",
			Type:        "bind",
			Options:     []string{"bind", "rw"},
		},
	}

	// Prepare DNS configuration
	resolvConfPath := "/etc/warren/resolv.conf"

	containerID, err := runtime.CreateContainerWithMounts(
		ctx, task, secretsPath, volumeMounts, resolvConfPath,
	)

Stopping a Container:

	// Graceful shutdown with 30s timeout
	err := runtime.StopContainer(ctx, containerID, 30*time.Second)
	if err != nil {
		log.Fatal(err)
	}

Checking Container Status:

	status, err := runtime.GetContainerStatus(ctx, containerID)
	if err != nil {
		log.Fatal(err)
	}

	switch status {
	case types.TaskStateRunning:
		fmt.Println("Container is running")
	case types.TaskStateFailed:
		fmt.Println("Container failed")
	}

Getting Container IP Address:

	ip, err := runtime.GetContainerIP(ctx, containerID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Container IP: %s\n", ip)

# Integration Points

This package integrates with:

  - pkg/types: Task and resource definitions
  - pkg/worker: Task execution and monitoring
  - pkg/embedded: Containerd binary management
  - containerd: Low-level container runtime operations

# Resource Limits Implementation

CPU Limits:
  - CPULimit=1.0 → 1024 CPU shares (relative weight)
  - CPULimit=1.0 → CFS quota=100000µs per 100000µs period (100% of 1 core)
  - CPULimit=0.5 → 512 shares + 50000µs quota (50% of 1 core)
  - Enforced by Linux CFS scheduler via cgroups

Memory Limits:
  - Direct mapping: MemoryLimit bytes → cgroup memory.limit_in_bytes
  - Hard limit: OOM killer terminates if exceeded
  - No swap by default (cgroup memory.memsw.limit_in_bytes = memory.limit_in_bytes)
  - Recommendation: Set limits 10-20% below node capacity for system overhead

# Design Patterns

Namespace Isolation:
  - All Warren containers run in "warren" namespace
  - Prevents conflicts with other containerd users
  - Context automatically wrapped: namespaces.WithNamespace(ctx, "warren")
  - Cleanup operations scoped to namespace

Socket Management:
  - Single long-lived gRPC connection per client
  - Automatic reconnection on socket errors (future)
  - Thread-safe for concurrent operations
  - Close() releases connection resources

Error Handling:
  - Wrapped errors with context: fmt.Errorf("operation failed: %w", err)
  - Graceful degradation: Ignore cleanup errors during rollback
  - Container not found returns nil (idempotent delete)
  - Task not running returns nil for stop (idempotent stop)

# Performance Characteristics

Image Pull:
  - First pull: 10-60s (depends on image size and network)
  - Subsequent pulls: < 1s (cached layers)
  - Parallel pulls: Supported (containerd handles deduplication)

Container Create:
  - Typical: 100-500ms (snapshot creation + spec generation)
  - Large images: 1-2s (more layers to prepare)
  - Parallel creates: Supported (independent snapshots)

Container Start:
  - Typical: 50-200ms (process spawn + namespace setup)
  - Heavy init: 1-5s (application-dependent)
  - Concurrent starts: Limited by system resources (forks, PIDs)

Container Stop:
  - Graceful: 1-10s (depends on application signal handling)
  - Force kill: < 500ms (SIGKILL immediate)
  - Timeout configurable per stop operation

Memory Usage:
  - Client overhead: ~5MB per ContainerdRuntime instance
  - Per container: ~1-2MB metadata (containerd tracking)
  - Container memory: Defined by task.Resources.MemoryLimit

# Troubleshooting

Common Issues:

Cannot Connect to Containerd:
  - Symptom: "failed to connect to containerd" error
  - Check: Socket path exists and has correct permissions
  - Check: Containerd daemon is running (systemctl status containerd)
  - Solution: Verify socket path or restart containerd

Container Fails to Start:
  - Symptom: CreateContainer succeeds but StartContainer fails
  - Check: Image exists and is unpacked (PullImage first)
  - Check: OCI spec is valid (resource limits, mounts)
  - Check: Container logs (via containerd-shim logs)
  - Solution: Review task definition for invalid configuration

Resource Limit Enforcement:
  - Symptom: Container uses more resources than specified
  - Check: Cgroups v2 vs v1 (different APIs)
  - Check: CPU shares vs quota (shares are relative, quota is absolute)
  - Check: Memory limit includes page cache
  - Solution: Use absolute limits (CPU quota) for predictable behavior

Container IP Not Found:
  - Symptom: GetContainerIP returns "no IP address found"
  - Check: Container has network namespace (not host mode)
  - Check: Container eth0 interface exists
  - Check: CNI plugin configured correctly
  - Solution: Ensure container networking is initialized

# Monitoring

Key metrics to monitor:

Container Operations:
  - runtime_image_pull_duration: Time to pull images
  - runtime_container_create_duration: Time to create containers
  - runtime_container_start_duration: Time to start containers
  - runtime_containers_running: Current running container count

Resource Usage:
  - runtime_container_cpu_usage: Per-container CPU usage
  - runtime_container_memory_usage: Per-container memory usage
  - runtime_oom_kills_total: Containers killed for OOM

Containerd Health:
  - runtime_containerd_connected: Connection status (1=connected)
  - runtime_containerd_errors_total: Client operation errors
  - runtime_containerd_snapshots_total: Active snapshots

# Security

Namespace Isolation:
  - All Warren containers isolated in "warren" namespace
  - Prevents accidental operations on system containers
  - Cleanup scoped to namespace only

Socket Permissions:
  - Containerd socket typically requires root or containerd group
  - Warren agent must run with appropriate permissions
  - Socket access controls who can create/manage containers

Image Verification:
  - No built-in signature verification (future enhancement)
  - Recommendation: Use private registries with access controls
  - Consider: containerd image encryption for sensitive workloads

Resource Limits as Security:
  - CPU limits prevent denial-of-service (CPU exhaustion)
  - Memory limits prevent OOM on node
  - Enforced by kernel cgroups (not bypassable from container)

# See Also

  - pkg/worker for task execution orchestration
  - pkg/embedded for containerd binary management
  - pkg/types for Task and Resource definitions
  - containerd documentation: https://containerd.io/
  - OCI runtime spec: https://github.com/opencontainers/runtime-spec
*/
package runtime
