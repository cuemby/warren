/*
Package scheduler provides container scheduling and orchestration for Warren clusters.

The scheduler is responsible for assigning pending containers to healthy worker nodes
based on resource availability, volume affinity, and load balancing requirements.
It runs as a continuous background process, ensuring that service replica counts
match their desired state and that containers are evenly distributed across the cluster.

# Architecture

The scheduler operates on a fixed 5-second interval, processing all services and
their associated containers in each cycle:

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
	│     • List existing containers                             │
	│     • Compare actual vs desired state                      │
	│     • Create missing containers OR remove excess           │
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

Scheduler: The main scheduling engine that orchestrates container placement.

	scheduler := NewScheduler(manager)
	scheduler.Start()  // Begins 5-second scheduling loop
	defer scheduler.Stop()

The scheduler maintains no internal state beyond the manager reference - all
cluster state is read from the manager on each cycle, making it stateless and
resilient to restarts.

# Scheduling Algorithms

## Replicated Service Scheduling

Replicated services specify a desired replica count. The scheduler ensures
exactly that many containers are running:

	Service: nginx (replicas=3)
	Current containers: 2 running
	Action: Create 1 new container

Node selection uses a simple round-robin algorithm with container counting:

 1. Count containers per node (only running containers)
 2. Select node with fewest containers
 3. Create container on selected node

## Global Service Scheduling

Global services run exactly one container per worker node:

	Service: monitoring-agent (mode=global)
	Worker nodes: 5
	Action: Ensure 1 container per node

The scheduler automatically creates containers when new nodes join and removes containers
when nodes are decommissioned.

## Volume Affinity

When a service uses volumes, containers must be scheduled on the node where the
volume resides:

	Service: database (volume=db-data)
	Volume: db-data (nodeID=worker-1)
	Constraint: Container MUST run on worker-1

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

	// Scheduler will create 3 containers distributed across 2 nodes
	// Expected: 2 containers on one node, 1 container on the other

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

	// Scheduler will place container on worker-2 due to volume affinity

# Integration Points

## Manager Integration

The scheduler depends on the manager package for all cluster state operations:

  - ListServices() - Get all services
  - ListNodes() - Get all worker nodes
  - ListContainersByService(serviceID) - Get containers for a service
  - CreateContainer(container) - Create new container
  - UpdateContainer(container) - Update container state
  - GetVolumeByName(name) - Check volume affinity

## Reconciler Coordination

The scheduler works in tandem with the reconciler:

  - Scheduler: Creates containers to meet desired state
  - Reconciler: Detects failures and marks containers for replacement
  - Scheduler: Sees failed containers and creates replacements

This separation of concerns ensures clean boundaries:

	Scheduler: "Make it happen" (proactive)
	Reconciler: "Fix what's broken" (reactive)

## Worker Integration

Workers pull containers assigned to them via the manager:

 1. Scheduler assigns container to node-1
 2. Worker on node-1 polls for containers (via manager)
 3. Worker starts runtime container
 4. Worker reports container state back to manager

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

The scheduler only creates and removes containers. It does NOT:

  - Start/stop runtime containers (worker's job)
  - Monitor container health (reconciler's job)
  - Update container runtime state (worker's job)

This clear separation prevents coupling and makes each component testable.

# Performance Characteristics

## Time Complexity

Per scheduling cycle (N services, M nodes, C containers):

  - List services: O(N)
  - List nodes: O(M)
  - List containers per service: O(C)
  - Node selection: O(M * C) worst case (counting containers per node)
  - Overall: O(N * (C + M))

For a typical cluster (100 services, 10 nodes, 500 containers):
  - ~0.5-1 second per scheduling cycle
  - Well within 5-second interval

## Memory Usage

Minimal memory footprint:

  - No caching (reads from manager each cycle)
  - Temporary allocations for node/container lists
  - ~10-20 MB for typical cluster sizes

## Scheduling Latency

Time from service creation to container running:

  - Best case: 5 seconds (next scheduler cycle)
  - Worst case: 10 seconds (just missed previous cycle)
  - Average: 7.5 seconds

For faster scheduling, reduce the ticker interval in the run() method, but be
aware of increased CPU usage and API load on the manager.

# Troubleshooting

## Containers Not Being Created

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

## Containers Stuck in "Pending"

The scheduler creates containers but workers start them. Debug:

1. Check worker logs:
  - Worker should see assigned containers
  - Look for containerd/image pull errors

2. Check container details:
  - Run: warren service ps <service-name>
  - Look at container error messages

## Uneven Container Distribution

If containers are not evenly distributed:

1. Check container state filtering:
  - Only running containers count toward load balancing
  - Failed/completed containers don't affect placement

2. Verify node readiness:
  - Scheduler only uses "Ready" worker nodes
  - Down nodes are excluded from scheduling

## Volume Affinity Not Working

If containers aren't being pinned to volume nodes:

1. Verify volume exists:
  - Run: warren volume ls
  - Check volume.NodeID field

2. Check service volume configuration:
  - Ensure volume name matches exactly
  - Verify volume driver is "local"

# Monitoring Metrics

The scheduler doesn't currently export Prometheus metrics, but you can monitor:

## Log-based Metrics

  - "Created container" - New container created
  - "Scheduler error" - Scheduling failure
  - "Selecting node X for service Y (volume affinity)" - Volume pinning

## Manager Metrics

  - Containers created per service (via manager API)
  - Container state distribution (pending/running/failed)
  - Node utilization (containers per node)

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
  - Scheduler respects container resource limits (CPU/memory)
  - Ensure nodes have sufficient capacity for all replicas
  - Monitor node resource usage to prevent over-subscription

# See Also

  - pkg/reconciler - Failure detection and auto-healing
  - pkg/manager - Cluster state management
  - pkg/worker - Container execution on worker nodes
  - pkg/volume - Volume lifecycle management
  - docs/concepts/services.md - Service modes and scaling
*/
package scheduler
