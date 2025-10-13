/*
Package scheduler provides task scheduling and orchestration for Warren clusters.

The scheduler is responsible for assigning pending tasks to healthy worker nodes
based on resource availability, volume affinity, and load balancing requirements.
It runs as a continuous background process, ensuring that service replica counts
match their desired state and that tasks are evenly distributed across the cluster.

# Architecture

The scheduler operates on a fixed 5-second interval, processing all services and
their associated tasks in each cycle:

	┌────────────────────────────────────────────────────────────┐
	│                    Scheduler Loop                          │
	│                   (Every 5 seconds)                        │
	└────────────────┬───────────────────────────────────────────┘
	                 │
	                 ▼
	┌────────────────────────────────────────────────────────────┐
	│  1. List all services and worker nodes                    │
	│  2. Filter nodes: Ready + Worker role only                │
	│  3. For each service:                                      │
	│     • List existing tasks                                  │
	│     • Compare actual vs desired state                      │
	│     • Create missing tasks OR remove excess tasks          │
	└────────────────┬───────────────────────────────────────────┘
	                 │
	    ┌────────────┴────────────┐
	    │                         │
	    ▼                         ▼
	┌─────────────┐       ┌──────────────┐
	│  Replicated │       │    Global    │
	│   Services  │       │   Services   │
	└─────┬───────┘       └──────┬───────┘
	      │                      │
	      ▼                      ▼
	  Round-robin            One per node
	  with volume            assignment
	  affinity

# Core Components

Scheduler: The main scheduling engine that orchestrates task placement.

	scheduler := NewScheduler(manager)
	scheduler.Start()  // Begins 5-second scheduling loop
	defer scheduler.Stop()

The scheduler maintains no internal state beyond the manager reference - all
cluster state is read from the manager on each cycle, making it stateless and
resilient to restarts.

# Scheduling Algorithms

## Replicated Service Scheduling

Replicated services specify a desired replica count. The scheduler ensures
exactly that many tasks are running:

	Service: nginx (replicas=3)
	Current tasks: 2 running
	Action: Create 1 new task

Node selection uses a simple round-robin algorithm with task counting:

 1. Count tasks per node (only running tasks)
 2. Select node with fewest tasks
 3. Create task on selected node

## Global Service Scheduling

Global services run exactly one task per worker node:

	Service: monitoring-agent (mode=global)
	Worker nodes: 5
	Action: Ensure 1 task per node

The scheduler automatically creates tasks when new nodes join and removes tasks
when nodes are decommissioned.

## Volume Affinity

When a service uses volumes, tasks must be scheduled on the node where the
volume resides:

	Service: database (volume=db-data)
	Volume: db-data (nodeID=worker-1)
	Constraint: Task MUST run on worker-1

This ensures data locality for stateful workloads. If a volume doesn't exist
yet, the scheduler selects a node using standard load balancing, and the volume
is created on that node.

# Usage Examples

## Basic Scheduler Setup

	import (
		"github.com/cuemby/warren/pkg/scheduler"
		"github.com/cuemby/warren/pkg/manager"
	)

	// Create manager (provides cluster state access)
	mgr := manager.NewManager(store, raftNode)

	// Create and start scheduler
	sched := scheduler.NewScheduler(mgr)
	sched.Start()

	// Scheduler runs automatically every 5 seconds
	// ...

	// Gracefully stop scheduler
	sched.Stop()

## Testing Scheduler Behavior

	// Create test fixtures
	service := &types.Service{
		ID:       "svc-1",
		Name:     "nginx",
		Mode:     types.ServiceModeReplicated,
		Replicas: 3,
		Image:    "nginx:latest",
	}

	nodes := []*types.Node{
		{ID: "node-1", Role: types.NodeRoleWorker, Status: types.NodeStatusReady},
		{ID: "node-2", Role: types.NodeRoleWorker, Status: types.NodeStatusReady},
	}

	// Scheduler will create 3 tasks distributed across 2 nodes
	// Expected: 2 tasks on one node, 1 task on the other

## Volume-Aware Scheduling

	// Service with volume requirement
	service := &types.Service{
		ID:      "db-1",
		Name:    "postgres",
		Image:   "postgres:15",
		Replicas: 1,
		Volumes: []*types.VolumeMount{
			{
				Source:      "postgres-data",
				Target:      "/var/lib/postgresql/data",
				Driver:      "local",
			},
		},
	}

	// Volume exists on specific node
	volume := &types.Volume{
		ID:     "vol-1",
		Name:   "postgres-data",
		NodeID: "worker-2",  // Must run on worker-2
		Driver: "local",
	}

	// Scheduler will place task on worker-2 due to volume affinity

# Integration Points

## Manager Integration

The scheduler depends on the manager package for all cluster state operations:

  - ListServices() - Get all services
  - ListNodes() - Get all worker nodes
  - ListTasksByService(serviceID) - Get tasks for a service
  - CreateTask(task) - Create new task
  - UpdateTask(task) - Update task state
  - GetVolumeByName(name) - Check volume affinity

## Reconciler Coordination

The scheduler works in tandem with the reconciler:

  - Scheduler: Creates tasks to meet desired state
  - Reconciler: Detects failures and marks tasks for replacement
  - Scheduler: Sees failed tasks and creates replacements

This separation of concerns ensures clean boundaries:

	Scheduler: "Make it happen" (proactive)
	Reconciler: "Fix what's broken" (reactive)

## Worker Integration

Workers pull tasks assigned to them via the manager:

 1. Scheduler assigns task to node-1
 2. Worker on node-1 polls for tasks (via manager)
 3. Worker starts container for task
 4. Worker reports task state back to manager

# Design Patterns

## Stateless Design

The scheduler maintains no persistent state. All decisions are made based on
current cluster state read from the manager. This makes the scheduler:

  - Resilient to crashes (no state to lose)
  - Easy to reason about (no hidden state)
  - Simple to test (just mock the manager)

## Reconciliation Loop Pattern

The scheduler implements the reconciliation loop pattern common in orchestrators:

	forever {
		actual_state = read_cluster_state()
		desired_state = read_service_specs()
		diff = desired_state - actual_state
		apply(diff)
		sleep(5_seconds)
	}

## Separation of Concerns

The scheduler only creates and removes tasks. It does NOT:

  - Start/stop containers (worker's job)
  - Monitor task health (reconciler's job)
  - Update task runtime state (worker's job)

This clear separation prevents coupling and makes each component testable.

# Performance Characteristics

## Time Complexity

Per scheduling cycle (N services, M nodes, T tasks):

  - List services: O(N)
  - List nodes: O(M)
  - List tasks per service: O(T)
  - Node selection: O(M * T) worst case (counting tasks per node)
  - Overall: O(N * (T + M))

For a typical cluster (100 services, 10 nodes, 500 tasks):
  - ~0.5-1 second per scheduling cycle
  - Well within 5-second interval

## Memory Usage

Minimal memory footprint:

  - No caching (reads from manager each cycle)
  - Temporary allocations for node/task lists
  - ~10-20 MB for typical cluster sizes

## Scheduling Latency

Time from service creation to task running:

  - Best case: 5 seconds (next scheduler cycle)
  - Worst case: 10 seconds (just missed previous cycle)
  - Average: 7.5 seconds

For faster scheduling, reduce the ticker interval in the run() method, but be
aware of increased CPU usage and API load on the manager.

# Troubleshooting

## Tasks Not Being Created

Check these common issues:

1. No worker nodes available:
  - Run: warren node ls
  - Ensure nodes are in "Ready" state
  - Check node heartbeats (should be < 30s ago)

2. Scheduler not running:
  - Check manager logs for "scheduler" component
  - Verify scheduler.Start() was called

3. Service configuration issues:
  - Ensure replica count > 0
  - Check volume constraints (volume must exist or be creatable)

## Tasks Stuck in "Pending"

The scheduler creates tasks but workers start them. Debug:

1. Check worker logs:
  - Worker should see assigned tasks
  - Look for containerd/image pull errors

2. Check task details:
  - Run: warren service ps <service-name>
  - Look at task error messages

## Uneven Task Distribution

If tasks are not evenly distributed:

1. Check task state filtering:
  - Only running tasks count toward load balancing
  - Failed/completed tasks don't affect placement

2. Verify node readiness:
  - Scheduler only uses "Ready" worker nodes
  - Down nodes are excluded from scheduling

## Volume Affinity Not Working

If tasks aren't being pinned to volume nodes:

1. Verify volume exists:
  - Run: warren volume ls
  - Check volume.NodeID field

2. Check service volume configuration:
  - Ensure volume name matches exactly
  - Verify volume driver is "local"

# Monitoring Metrics

The scheduler doesn't currently export Prometheus metrics, but you can monitor:

## Log-based Metrics

  - "Created task" - New task created
  - "Scheduler error" - Scheduling failure
  - "Selecting node X for service Y (volume affinity)" - Volume pinning

## Manager Metrics

  - Tasks created per service (via manager API)
  - Task state distribution (pending/running/failed)
  - Node utilization (tasks per node)

# Best Practices

1. Scheduler Interval Tuning
  - Default 5s balances responsiveness vs. overhead
  - For large clusters (>100 nodes), consider 10s
  - For dev/test, can reduce to 1s for faster feedback

2. Service Replica Planning
  - Set replicas <= number of worker nodes (for even distribution)
  - Use global mode for node-level services (monitoring, logging)
  - Consider resource limits when scaling replicas

3. Volume-Backed Services
  - Use replicas=1 for stateful services with local volumes
  - Pin volume to specific node by creating it first
  - Consider node labels for volume placement control (future feature)

4. Resource Constraints
  - Scheduler respects task resource limits (CPU/memory)
  - Ensure nodes have sufficient capacity for all replicas
  - Monitor node resource usage to prevent over-subscription

# See Also

  - pkg/reconciler - Failure detection and auto-healing
  - pkg/manager - Cluster state management
  - pkg/worker - Task execution on worker nodes
  - pkg/volume - Volume lifecycle management
  - docs/concepts/services.md - Service modes and scaling
*/
package scheduler
