/*
Package reconciler provides failure detection and automatic healing for Warren clusters.

The reconciler continuously monitors the cluster for failures and deviations from
desired state, automatically triggering corrective actions to maintain system health.
It runs as a background process that detects failed tasks, down nodes, and unhealthy
containers, ensuring that services remain available despite infrastructure failures.

# Architecture

The reconciler operates on a fixed 10-second interval, monitoring all cluster
components and triggering remediation when problems are detected:

	┌────────────────────────────────────────────────────────────┐
	│                  Reconciliation Loop                       │
	│                   (Every 10 seconds)                       │
	└────────────────┬───────────────────────────────────────────┘
	                 │
	    ┌────────────┴────────────┐
	    │                         │
	    ▼                         ▼
	┌─────────────────┐   ┌──────────────────┐
	│ Reconcile Nodes │   │ Reconcile Tasks  │
	└─────┬───────────┘   └──────┬───────────┘
	      │                      │
	      ▼                      ▼
	  Check                 Check health
	  heartbeats            and state
	      │                      │
	      ▼                      ▼
	  Mark down             Replace failed
	  nodes                 tasks

# Failure Detection

## Node Failure Detection

Nodes send heartbeats every 5 seconds. If no heartbeat is received for 30
seconds, the node is marked as "Down":

	Last heartbeat: 2024-01-15 10:30:00
	Current time:   2024-01-15 10:30:35  (35 seconds elapsed)
	Status: NodeStatusDown

When a node is marked down:
  - All running tasks on that node are marked as failed
  - Scheduler creates replacement tasks on healthy nodes
  - Node becomes eligible for scheduling again after recovery

## Task Failure Detection

The reconciler detects multiple failure scenarios:

1. Task explicitly failed (worker reported failure):
  - ActualState = TaskStateFailed
  - Error message available in task.Error
  - Action: Mark for cleanup (DesiredState = Shutdown)

2. Task health check failed:
  - HealthStatus.Healthy = false
  - ConsecutiveFailures > threshold
  - Action: Mark as failed, scheduler creates replacement

3. Task on down node:
  - Node.Status = NodeStatusDown
  - Task.DesiredState = Running
  - Action: Mark as failed for rescheduling

## Health-Aware Reconciliation

The reconciler integrates with health checks to detect unhealthy containers:

	Task running: nginx-1
	Health check: HTTP GET /health → 503 Service Unavailable
	Consecutive failures: 3 (exceeds threshold)
	Action: Mark task as failed
	Result: Scheduler creates replacement task

# Core Components

Reconciler: The main reconciliation engine that monitors cluster health.

	reconciler := NewReconciler(manager)
	reconciler.Start()  // Begins 10-second reconciliation loop
	defer reconciler.Stop()

Like the scheduler, the reconciler is stateless - it reads cluster state from
the manager on each cycle, making decisions based only on current observations.

# Reconciliation Strategies

## Node Reconciliation

For each node in the cluster:

 1. Read node.LastHeartbeat timestamp
 2. Calculate time since last heartbeat
 3. If > 30 seconds:
    • Update node.Status = NodeStatusDown
    • Persist to storage
 4. If < 30 seconds and currently down:
    • Update node.Status = NodeStatusReady
    • Node returns to scheduling pool

## Task Reconciliation

For each task in the cluster:

 1. Check if task is failed but desired to be running:
    • Mark DesiredState = Shutdown
    • Scheduler will create replacement

 2. Check if task is running but unhealthy:
    • Evaluate HealthStatus
    • If unhealthy, mark ActualState = Failed
    • Scheduler will create replacement

 3. Check if task is on a down node:
    • Mark ActualState = Failed
    • Mark DesiredState = Shutdown
    • Scheduler will create replacement on healthy node

 4. Garbage collect completed tasks:
    • If DesiredState = Shutdown and ActualState = Complete
    • Wait 5 minutes, then delete task

# Usage Examples

## Basic Reconciler Setup

	import (
		"github.com/cuemby/warren/pkg/reconciler"
		"github.com/cuemby/warren/pkg/manager"
	)

	// Create manager (provides cluster state access)
	mgr := manager.NewManager(store, raftNode)

	// Create and start reconciler
	rec := reconciler.NewReconciler(mgr)
	rec.Start()

	// Reconciler runs automatically every 10 seconds
	// ...

	// Gracefully stop reconciler
	rec.Stop()

## Simulating Node Failure

	// Simulate node going down (stop heartbeats)
	node := &types.Node{
		ID:            "worker-1",
		Status:        types.NodeStatusReady,
		LastHeartbeat: time.Now().Add(-35 * time.Second),  // 35s ago
	}

	// On next reconciliation cycle (within 10 seconds):
	// 1. Reconciler detects: now - LastHeartbeat > 30s
	// 2. Sets node.Status = NodeStatusDown
	// 3. All tasks on worker-1 marked as failed
	// 4. Scheduler creates replacement tasks on healthy nodes

## Health Check Integration

	// Service with health check configured
	service := &types.Service{
		ID:    "svc-1",
		Name:  "api",
		Image: "myapp:v1",
		HealthCheck: &types.HealthCheck{
			Type:     types.HealthCheckTypeHTTP,
			HTTP:     &types.HTTPHealthCheck{Path: "/health"},
			Interval: 10 * time.Second,
			Timeout:  5 * time.Second,
			Retries:  3,
		},
	}

	// Task with failed health check
	task := &types.Task{
		ID:           "task-1",
		ServiceID:    "svc-1",
		ActualState:  types.TaskStateRunning,
		DesiredState: types.TaskStateRunning,
		HealthStatus: &types.HealthStatus{
			Healthy:             false,
			ConsecutiveFailures: 3,
			Message:            "HTTP 503 Service Unavailable",
		},
	}

	// Reconciler will:
	// 1. Detect task is running but unhealthy
	// 2. Set task.ActualState = TaskStateFailed
	// 3. Set task.Error = "health check failed: HTTP 503..."
	// 4. Scheduler creates new healthy task

# Integration Points

## Manager Integration

The reconciler depends on the manager package for cluster state:

  - ListNodes() - Get all nodes to check heartbeats
  - UpdateNode(node) - Mark nodes as down/ready
  - ListTasks() - Get all tasks for health checks
  - UpdateTask(task) - Mark tasks as failed
  - DeleteTask(taskID) - Garbage collect old tasks
  - GetNode(nodeID) - Get node details for task reconciliation

## Scheduler Coordination

The reconciler and scheduler form a feedback loop:

	Reconciler: "This task is broken, mark it for cleanup"
	Scheduler: "I see a service needs more replicas, create task"
	Reconciler: "New task is healthy, no action needed"

This separation ensures:
  - Reconciler focuses on detection
  - Scheduler focuses on placement
  - Clean, testable boundaries

## Health Check Integration

The reconciler reads health status from tasks:

 1. Worker runs health checks (pkg/health)
 2. Worker updates task.HealthStatus via manager
 3. Reconciler reads task.HealthStatus
 4. Reconciler marks unhealthy tasks as failed

## Worker Integration

Workers interact with reconciliation indirectly:

 1. Worker reports task state to manager
 2. Reconciler detects failed state
 3. Reconciler marks task for cleanup
 4. Worker sees DesiredState = Shutdown
 5. Worker stops container and marks task complete

# Design Patterns

## Stateless Reconciliation

Like the scheduler, the reconciler maintains no state between cycles:

  - All decisions based on current cluster state
  - No memory of previous failures
  - Crash-safe and simple to reason about

## Level-Triggered Reconciliation

The reconciler uses level-triggered (not edge-triggered) logic:

	Edge-triggered: "Node just went down, react"
	Level-triggered: "Node is down, ensure proper state"

This means:
  - Reconciler doesn't need to track state changes
  - System converges even if reconciliation cycles are missed
  - More robust to timing issues and race conditions

## Garbage Collection Pattern

Completed tasks are not deleted immediately:

 1. Task finishes (DesiredState = Shutdown, ActualState = Complete)
 2. Wait 5 minutes (grace period for debugging)
 3. Delete task from storage

This provides a debugging window while preventing unbounded storage growth.

# Performance Characteristics

## Time Complexity

Per reconciliation cycle (N nodes, T tasks):

  - List nodes: O(N)
  - Check heartbeats: O(N)
  - List tasks: O(T)
  - Check health: O(T)
  - Get node per task: O(T * log N) if manager uses index
  - Overall: O(N + T)

For a typical cluster (10 nodes, 500 tasks):
  - ~100-200ms per reconciliation cycle
  - Well within 10-second interval

## Memory Usage

Minimal memory footprint:

  - No state caching
  - Temporary allocations for node/task lists
  - ~5-10 MB for typical cluster sizes

## Failure Detection Latency

Time from failure to remediation:

  - Node failure: 30-40 seconds (30s timeout + 10s cycle)
  - Task failure: 0-10 seconds (detected in next cycle)
  - Health check failure: (retries * interval) + 10s cycle
  - Example: 3 retries * 10s + 10s = ~40 seconds

# Troubleshooting

## Tasks Not Being Replaced

If failed tasks are not being replaced:

1. Check reconciler is running:
  - Look for "Reconciler error" in manager logs
  - Verify reconciler.Start() was called

2. Check scheduler is running:
  - Reconciler marks tasks failed
  - Scheduler creates replacements
  - Both must be running!

3. Verify task state:
  - Run: warren service ps <service-name>
  - Check DesiredState vs ActualState
  - Look for error messages

## False Positive Node Failures

If nodes are incorrectly marked as down:

1. Check heartbeat interval:
  - Workers should heartbeat every 5s
  - Network latency < 25s to avoid false positives

2. Check manager clock skew:
  - Reconciler uses manager's clock
  - NTP synchronization recommended

3. Verify network connectivity:
  - Workers must reach manager node
  - Check firewalls, security groups

## Tasks Flapping (Constant Restarts)

If tasks are constantly being killed and recreated:

1. Check health check configuration:
  - Retries too low? (minimum 2-3)
  - Timeout too short? (minimum 5s)
  - StartPeriod too short for slow apps?

2. Check application logs:
  - Is app actually healthy?
  - Look for crash loops

3. Review reconciliation logs:
  - Look for patterns in failure messages
  - Check timing of failures

## Slow Failure Recovery

If recovery takes too long:

1. Tune reconciliation interval:
  - Default 10s balances overhead vs. speed
  - Can reduce to 5s for faster recovery
  - Be aware of increased CPU usage

2. Tune node heartbeat timeout:
  - Default 30s prevents false positives
  - Can reduce to 15s for faster detection
  - Risk of false positives on slow networks

3. Tune health check parameters:
  - Reduce interval for faster detection
  - Reduce retries for quicker action
  - Balance against false positives

# Monitoring Metrics

The reconciler exports Prometheus metrics:

	# Reconciliation cycle metrics
	reconciliation_duration_seconds - Time to complete cycle
	reconciliation_cycles_total - Total reconciliation cycles

## Additional Observable Metrics

  - Failed tasks detected (log: "Task X failed")
  - Unhealthy tasks detected (log: "Task X is unhealthy")
  - Nodes marked down (log: "Node X is down")
  - Tasks cleaned up (log: "Task X cleaned up after completion")

# Best Practices

1. Reconciliation Interval Tuning
  - Default 10s provides good balance
  - For critical workloads, reduce to 5s
  - For large clusters (>100 nodes), consider 15s
  - Always < scheduler interval for proper coordination

2. Health Check Configuration
  - Set Retries = 3 (prevents false positives)
  - Set Interval = 10-30s (not too aggressive)
  - Set StartPeriod for slow-starting apps
  - Monitor health check logs for tuning

3. Node Failure Tolerance
  - Plan for 1-2 minute recovery time
  - Ensure sufficient spare capacity for rescheduling
  - Use persistent volumes for stateful workloads
  - Test failure scenarios in staging

4. Task Cleanup
  - Default 5-minute grace period is usually sufficient
  - For high-churn workloads, reduce to 1 minute
  - For debugging, increase to 15-30 minutes
  - Monitor task count growth over time

5. Resource Planning
  - Account for double resource usage during failover
  - Ensure (N-1) nodes can handle full load
  - Monitor node resource usage
  - Plan for cascading failures

# See Also

  - pkg/scheduler - Task placement and orchestration
  - pkg/manager - Cluster state management
  - pkg/health - Health check implementation
  - pkg/worker - Task execution and state reporting
  - docs/troubleshooting.md - Debugging cluster issues
*/
package reconciler
