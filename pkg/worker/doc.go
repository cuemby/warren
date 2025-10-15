/*
Package worker implements the Warren worker node that executes containerized tasks.

The worker package is the data plane of Warren, responsible for running containers,
reporting health status, and maintaining connectivity with the manager cluster.
Workers are stateless agents that receive task assignments from managers and
execute them using containerd.

# Architecture

A Warren worker is a single-purpose agent that bridges managers and containers:

	┌─────────────────────── WORKER NODE ────────────────────────┐
	│                                                              │
	│  ┌──────────────────────────────────────────────┐          │
	│  │              Worker Agent                     │          │
	│  │  - gRPC client to manager                     │          │
	│  │  - Heartbeat loop (5s)                        │          │
	│  │  - Task sync loop (3s)                        │          │
	│  │  - Status reporting                           │          │
	│  └──────┬──────────────────────────┬─────────────┘          │
	│         │                          │                         │
	│  ┌──────▼───────┐          ┌──────▼───────────┐            │
	│  │  Handlers    │          │  Local Cache     │            │
	│  │  - Secrets   │          │  - Task map      │            │
	│  │  - Volumes   │          │  - Container IDs │            │
	│  │  - DNS       │          │  - Status        │            │
	│  │  - Health    │          └──────────────────┘            │
	│  │  - Ports     │                                           │
	│  └──────┬───────┘                                           │
	│         │                                                    │
	│  ┌──────▼──────────────────────────────────────┐           │
	│  │          Containerd Runtime                  │           │
	│  │  - Pull images                               │           │
	│  │  - Create containers                         │           │
	│  │  - Start/stop containers                     │           │
	│  │  - Monitor container status                  │           │
	│  │  - Apply resource limits                     │           │
	│  └──────────────────────────────────────────────┘           │
	└──────────────────────────────────────────────────────────┘

# Core Components

Worker:
  - Main worker agent
  - Maintains gRPC connection to manager
  - Executes heartbeat and sync loops
  - Coordinates all handlers

SecretsHandler:
  - Fetches encrypted secrets from manager
  - Decrypts using cluster encryption key
  - Mounts secrets as tmpfs in containers
  - Cleans up on task removal

VolumesHandler:
  - Manages volume lifecycle
  - Mounts volumes into containers
  - Ensures volume affinity (local volumes)
  - Tracks volume usage

HealthMonitor:
  - Executes health checks (HTTP/TCP/Exec)
  - Reports health status to manager
  - Triggers task replacement on failure
  - Integrates with reconciler

DNSHandler:
  - Configures container DNS
  - Points containers to manager DNS server
  - Enables service discovery

HostPortPublisher:
  - Publishes container ports on host
  - Manages iptables rules (Linux)
  - Handles port conflicts
  - Cleans up on task removal

# Worker Lifecycle

Registration:

 1. Worker starts with join token
 2. Connects to manager via gRPC
 3. Registers with node resources (CPU, memory)
 4. Receives unique node ID
 5. Begins heartbeat loop

Heartbeat Loop (5 seconds):

 1. Send heartbeat to manager
 2. Report node resources and status
 3. Receive acknowledgment
 4. Update last heartbeat timestamp

Task Sync Loop (3 seconds):

 1. Fetch assigned tasks from manager
 2. Compare with local task cache
 3. Start new tasks
 4. Stop removed tasks
 5. Report task status updates

Task Execution:

 1. Receive task assignment
 2. Prepare: Mount secrets and volumes
 3. Pull container image (if not cached)
 4. Create container with runtime
 5. Configure DNS, network, resources
 6. Start container
 7. Monitor health checks
 8. Report running status

Task Removal:

 1. Receive stop command
 2. Stop container (SIGTERM, grace period)
 3. Force kill if timeout exceeded
 4. Unmount secrets and volumes
 5. Remove iptables rules
 6. Clean up container
 7. Report complete status

# Usage

Creating a Worker:

	cfg := &worker.Config{
		NodeID:           "worker-1",
		ManagerAddr:      "192.168.1.10:8080",
		DataDir:          "/var/lib/warren/worker-1",
		JoinToken:        "worker-join-token-xyz789",
		EncryptionKey:    clusterKey,
		ContainerdSocket: "", // Auto-detect
		Resources: &types.NodeResources{
			CPUCores:    4,
			MemoryBytes: 8 * 1024 * 1024 * 1024, // 8GB
			DiskBytes:   100 * 1024 * 1024 * 1024, // 100GB
		},
	}

	w, err := worker.NewWorker(cfg)
	if err != nil {
		log.Fatal(err)
	}

Starting the Worker:

	// Connects to manager and begins loops
	err := w.Start()
	if err != nil {
		log.Fatal(err)
	}

Stopping the Worker:

	// Graceful shutdown with task cleanup
	err := w.Stop()
	if err != nil {
		log.Fatal(err)
	}

# Task Execution

The worker executes tasks through multiple phases:

Preparing Phase:

  - Fetch and decrypt secrets from manager
  - Mount secrets as tmpfs at /run/secrets/<name>
  - Ensure volumes exist (create if local driver)
  - Prepare volume mount points

Starting Phase:

  - Pull container image if not present
  - Create container with:
  - Environment variables
  - Secret mounts (tmpfs)
  - Volume mounts (bind or named)
  - DNS configuration (manager IP)
  - Resource limits (CPU, memory)
  - Health check configuration
  - Configure host port publishing (iptables)
  - Start container process

Running Phase:

  - Monitor container status
  - Execute health checks periodically
  - Report status updates to manager
  - Handle container restarts (restart policy)

Stopping Phase:

  - Send SIGTERM to container
  - Wait for grace period (default 10s)
  - Send SIGKILL if timeout exceeded
  - Unmount secrets (tmpfs)
  - Remove iptables rules
  - Clean up container

# Secrets Handling

Workers handle secrets securely:

Fetch and Decrypt:

  - Fetch encrypted secret data from manager
  - Decrypt using cluster encryption key
  - Store decrypted data in memory only

Mount as tmpfs:

  - Create tmpfs mount at /run/secrets/<name>
  - Write secret data to tmpfs
  - Set permissions (0400, container user)
  - tmpfs is memory-only (never touches disk)

Container Access:

  - Container mounts /run/secrets/<name>
  - Application reads secret as regular file
  - Secret data never written to disk
  - tmpfs cleared on unmount

Cleanup:

  - Unmount tmpfs when task stops
  - Memory automatically cleared
  - No disk cleanup required

# Volume Handling

Workers manage volume lifecycle:

Local Volumes:

  - Created at /var/lib/warren/volumes/<volume-name>
  - Mounted as bind mount into container
  - Persists across task restarts
  - Affinity ensures same node (local storage)

Volume Mounts:

  - Source: Volume name (e.g., "db-data")
  - Target: Container path (e.g., "/var/lib/postgresql")
  - ReadOnly: Optional read-only mount
  - UID/GID mapping handled by runtime

Volume Cleanup:

  - Volumes persist after task stops
  - Manual deletion via "warren volume delete"
  - Prevents accidental data loss

# Health Monitoring

Workers execute health checks and report results:

HTTP Health Checks:

  - Send HTTP GET to specified endpoint
  - Expected status code: 200-399
  - Timeout and retry configuration
  - Reports healthy/unhealthy to manager

TCP Health Checks:

  - Attempt TCP connection to port
  - Connection success = healthy
  - Connection failure = unhealthy
  - Useful for databases, caches

Exec Health Checks:

  - Run command inside container
  - Exit code 0 = healthy
  - Non-zero exit = unhealthy
  - Useful for custom health logic

Health Failure:

  - After N failed checks, mark unhealthy
  - Report to manager
  - Reconciler replaces unhealthy task
  - Old task stops, new task starts

# Port Publishing

Workers publish container ports to host:

Host Mode (PublishModeHost):

  - Maps container port to host port
  - Creates iptables rules:
  - PREROUTING: DNAT to container IP
  - POSTROUTING: MASQUERADE for responses
  - Port available only on hosting node
  - Used for health checks, ingress backends

Ingress Mode (PublishModeIngress):

  - Future: Routing mesh (not yet implemented)
  - Will route to any task replica
  - Load balancing across tasks

Port Conflicts:

  - Worker detects port conflicts
  - Reports error to manager
  - Scheduler avoids conflicting placements

# Failure Scenarios

Manager Disconnection:

  - Worker continues running tasks
  - Heartbeat loop retries connection
  - Exponential backoff (up to 30s)
  - Tasks keep running (autonomy)

Container Failure:

  - Worker detects exit via containerd
  - Restarts based on RestartPolicy
  - Reports failure to manager
  - Reconciler may reschedule

Containerd Failure:

  - Worker cannot execute new tasks
  - Reports error to manager
  - Existing containers may continue (containerd recovery)
  - Worker marked unhealthy

Worker Crash:

  - Containers keep running (containerd daemon)
  - Worker restart re-syncs state
  - Orphaned containers detected and cleaned

# Performance Characteristics

Resource Usage:

  - Base worker: 20MB memory
  - Per task: ~5MB memory
  - Typical worker (10 tasks): ~70MB total

Loop Frequencies:

  - Heartbeat: Every 5 seconds
  - Task sync: Every 3 seconds
  - Health checks: Per service config (30s typical)

Task Operations:

  - Task start time: 2-5s (image cached)
  - Task start time: 10-60s (image pull)
  - Task stop time: <10s (grace period)
  - Task cleanup: <1s

# Integration Points

This package integrates with:

  - pkg/runtime: Executes containers via containerd
  - pkg/security: Decrypts secrets and handles certificates
  - pkg/volume: Manages volume mounts
  - pkg/health: Executes health check probes
  - pkg/network: Publishes ports via iptables
  - pkg/dns: Configures container DNS
  - api/proto: Communicates with manager via gRPC

# Design Patterns

Agent Pattern:

  - Stateless agent design
  - All state stored in manager
  - Worker restarts are transparent
  - Task cache for performance only

Handler Pattern:

  - Separate handlers for concerns
  - Secrets, volumes, DNS, health, ports
  - Each handler has specific lifecycle
  - Coordinated by main Worker

Reconciliation Pattern:

  - Desired state from manager
  - Current state from containerd
  - Reconcile: Start new, stop removed
  - Eventually consistent

# Security

Join Token Authentication:

  - Worker authenticates with join token
  - Token validated by manager
  - Token single-use (optional)
  - Connection uses gRPC (TLS ready)

Secrets Encryption:

  - Secrets encrypted at rest in manager
  - Decrypted in worker memory only
  - Mounted as tmpfs (no disk write)
  - Cleared on unmount

Container Isolation:

  - Containers run as non-root (when specified)
  - Linux namespaces (PID, network, mount)
  - Cgroups for resource limits
  - Seccomp profiles (future)

# Troubleshooting

Common Issues:

Worker Won't Connect:

  - Check manager address reachable
  - Verify join token is valid
  - Check firewall allows port 8080
  - Review worker logs

Tasks Not Starting:

  - Check containerd is running
  - Verify image can be pulled
  - Check disk space for volumes
  - Review task logs in containerd

Health Checks Failing:

  - Verify container is running
  - Test endpoint manually (HTTP)
  - Check network connectivity
  - Adjust timeout/retries

Ports Not Accessible:

  - Verify iptables rules created
  - Check container listening on port
  - Test from host machine first
  - Review firewall rules

# Monitoring

Key metrics to monitor:

Worker Health:

  - worker_heartbeat_failures: Connection issues
  - worker_tasks_running: Active task count
  - worker_task_start_duration: Performance
  - worker_task_failures: Task reliability

Resource Usage:

  - node_cpu_used: CPU utilization
  - node_memory_used: Memory utilization
  - node_disk_used: Disk utilization

Container Health:

  - container_restarts: Restart frequency
  - health_check_failures: Health check issues
  - container_oom_kills: Memory limit hits

# See Also

  - pkg/runtime for containerd integration
  - pkg/security for secrets handling
  - pkg/health for health check execution
  - docs/concepts/services.md for service concepts
  - docs/troubleshooting.md for common issues
*/
package worker
