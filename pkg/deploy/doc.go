/*
Package deploy implements deployment strategies for Warren services.

The deploy package provides orchestrated service updates with multiple deployment
strategies including rolling updates, blue/green deployments, and canary releases.
It coordinates with the manager to safely transition services from old to new
versions with configurable parallelism, delays, and failure handling.

# Architecture

Warren's deployment system coordinates safe service updates across the cluster:

	┌─────────────── DEPLOYMENT STRATEGIES ────────────────────┐
	│                                                            │
	│  ┌────────────────────────────────────────────┐          │
	│  │              Deployer                       │          │
	│  │  - Coordinates service updates              │          │
	│  │  - Implements deployment strategies         │          │
	│  │  - Integrates with manager API              │          │
	│  └──────────────────┬─────────────────────────┘          │
	│                     │                                      │
	│  ┌──────────────────▼─────────────────────────┐          │
	│  │         Deployment Strategies               │          │
	│  │                                              │          │
	│  │  Rolling Update:                            │          │
	│  │    Old → New (one by one)                   │          │
	│  │    - Parallelism: 1-N tasks                 │          │
	│  │    - Delay: Configurable between batches    │          │
	│  │    - Rollback: Automatic on failure         │          │
	│  │                                              │          │
	│  │  Blue/Green (Future):                       │          │
	│  │    Old (Blue) + New (Green) → Switch        │          │
	│  │    - Full deployment before traffic switch  │          │
	│  │    - Instant rollback capability            │          │
	│  │    - Zero downtime                          │          │
	│  │                                              │          │
	│  │  Canary (Future):                           │          │
	│  │    Old + New (weighted) → Gradual          │          │
	│  │    - Traffic split: 10% → 50% → 100%       │          │
	│  │    - Metrics-driven progression             │          │
	│  │    - Automatic rollback on errors           │          │
	│  └──────────────────┬─────────────────────────┘          │
	│                     │                                      │
	│  ┌──────────────────▼─────────────────────────┐          │
	│  │          Update Flow                        │          │
	│  │                                              │          │
	│  │  1. Get current service state               │          │
	│  │  2. List running tasks                      │          │
	│  │  3. Update service image                    │          │
	│  │  4. Shutdown old tasks (batched)            │          │
	│  │  5. Scheduler creates new tasks             │          │
	│  │  6. Wait for health checks                  │          │
	│  │  7. Repeat until all tasks updated          │          │
	│  └────────────────────────────────────────────┘           │
	│                                                            │
	│  ┌────────────────────────────────────────────┐          │
	│  │         Failure Handling                    │          │
	│  │                                              │          │
	│  │  Pause: Stop deployment, investigate        │          │
	│  │  Rollback: Revert to previous version       │          │
	│  │  Continue: Proceed despite failures         │          │
	│  └────────────────────────────────────────────┘           │
	└────────────────────────────────────────────────────────┘

# Core Components

Deployer:
  - Main orchestrator for service updates
  - Holds reference to manager for state access
  - Implements deployment strategy selection
  - Provides status reporting

DeploymentStatus:
  - Current deployment progress information
  - Task counts by state
  - Ready vs desired task comparison
  - Used for monitoring and CLI output

UpdateConfig:
  - Parallelism: How many tasks to update simultaneously
  - Delay: Wait time between update batches
  - FailureAction: pause, rollback, or continue
  - CanaryWeight: Percentage of traffic to new version (future)

# Deployment Strategies

Rolling Update:

Strategy:
  - Update tasks one by one (or in batches)
  - Old task shutdown → Scheduler creates new task
  - Wait for health check before next batch
  - Configurable parallelism and delay

Flow:
 1. Get all running tasks for service
 2. Determine batch size (parallelism)
 3. For each batch:
    a. Shutdown old tasks (set DesiredState=Shutdown)
    b. Scheduler automatically creates replacements
    c. Wait for configured delay
 4. Monitor until all tasks updated

Configuration:
  - Parallelism: 1 (serial), 2-N (parallel batches)
  - Delay: 0s (immediate), 10s, 30s, etc.
  - FailureAction: rollback (default), pause, continue

Advantages:
  - Zero downtime (old tasks run until new ready)
  - Resource efficient (gradual replacement)
  - Easy rollback (reverse the process)

Disadvantages:
  - Slower than blue/green (gradual)
  - Mixed versions during update
  - Potential compatibility issues between versions

Blue/Green Deployment (Future - Milestone 4):

Strategy:
  - Deploy full new version (Green) alongside old (Blue)
  - Test Green environment before switching traffic
  - Instant cutover via load balancer update
  - Keep Blue for instant rollback

Flow:
 1. Deploy Green version (same replica count as Blue)
 2. Wait for all Green tasks healthy
 3. Update service VIP to point to Green
 4. Monitor for issues
 5. Remove Blue tasks after soak period

Configuration:
  - SoakTime: How long to monitor before cleanup
  - FailureAction: rollback (switch back to Blue)

Advantages:
  - Zero downtime
  - Full testing before switch
  - Instant rollback
  - Single version at a time (after switch)

Disadvantages:
  - Requires 2x resources temporarily
  - Longer deployment time (full parallel deployment)
  - Database migrations require coordination

Canary Deployment (Future - Milestone 4):

Strategy:
  - Deploy new version with small percentage of traffic
  - Gradually increase traffic weight (10% → 50% → 100%)
  - Monitor metrics and automatically rollback on errors
  - Final step removes old version

Flow:
 1. Deploy canary tasks (e.g., 10% of replicas)
 2. Route 10% of traffic to canary
 3. Monitor error rates, latency, etc.
 4. If healthy, increase to 50%, then 100%
 5. Remove old version tasks

Configuration:
  - CanaryWeight: 10, 25, 50, 100 (progression)
  - HealthThreshold: Error rate < 1%, p95 latency < 500ms
  - AutoPromote: Automatic or manual progression

Advantages:
  - Minimal risk (small traffic percentage)
  - Metrics-driven validation
  - Automatic rollback on issues
  - Production validation with real traffic

Disadvantages:
  - Longer deployment time
  - Requires metrics integration
  - Complex traffic routing
  - May need session affinity

# Usage

Creating a Deployer:

	import (
		"github.com/cuemby/warren/pkg/deploy"
		"github.com/cuemby/warren/pkg/manager"
	)

	// Create deployer with manager reference
	mgr := manager.NewManager(...)
	deployer := deploy.NewDeployer(mgr)

Rolling Update:

	// Update service to new image
	err := deployer.UpdateService("service-xyz789", "nginx:1.21")
	if err != nil {
		log.Fatal(err)
	}

Getting Deployment Status:

	status, err := deployer.GetDeploymentStatus("service-xyz789")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Service: %s\n", status.ServiceName)
	fmt.Printf("Image: %s\n", status.Image)
	fmt.Printf("Ready: %d/%d\n", status.ReadyTasks, status.DesiredTasks)
	fmt.Printf("Total: %d tasks\n", status.TotalTasks)
	fmt.Printf("By state: %+v\n", status.Tasks)

Complete Example:

	package main

	import (
		"fmt"
		"time"
		"github.com/cuemby/warren/pkg/deploy"
		"github.com/cuemby/warren/pkg/manager"
		"github.com/cuemby/warren/pkg/types"
	)

	func main() {
		// Initialize manager and deployer
		mgr := manager.NewManager(...)
		defer mgr.Close()

		deployer := deploy.NewDeployer(mgr)

		// Create service with update configuration
		service := &types.Service{
			Name:     "web",
			Image:    "nginx:1.20",
			Replicas: 5,
			UpdateConfig: &types.UpdateConfig{
				Parallelism: 2,       // Update 2 tasks at a time
				Delay:       10 * time.Second, // Wait 10s between batches
			},
		}

		err := mgr.CreateService(service)
		if err != nil {
			panic(err)
		}

		fmt.Println("Service created, waiting for stable state...")
		time.Sleep(30 * time.Second)

		// Update to new version
		fmt.Println("Starting rolling update to nginx:1.21")
		err = deployer.UpdateService(service.ID, "nginx:1.21")
		if err != nil {
			panic(err)
		}

		// Monitor deployment status
		for i := 0; i < 30; i++ {
			status, _ := deployer.GetDeploymentStatus(service.ID)
			fmt.Printf("Progress: %d/%d ready\n",
				status.ReadyTasks, status.DesiredTasks)

			if status.ReadyTasks == status.DesiredTasks {
				fmt.Println("Deployment complete!")
				break
			}

			time.Sleep(2 * time.Second)
		}
	}

# Integration Points

This package integrates with:

  - pkg/manager: Service and task state management
  - pkg/scheduler: Automatic task creation for replacements
  - pkg/reconciler: Ensures desired state achieved
  - pkg/types: Service, Task, and UpdateConfig definitions
  - pkg/api: CLI commands for service updates

# Update Configuration

UpdateConfig Structure:

	type UpdateConfig struct {
		Parallelism   int           // Concurrent updates (default: 1)
		Delay         time.Duration // Delay between batches (default: 0s)
		FailureAction string        // "pause", "rollback", "continue"
		CanaryWeight  int           // 0-100 for canary strategy
	}

Parallelism:
  - 1: Serial updates (safest, slowest)
  - 2-5: Small batches (balanced)
  - N: Update all at once (fastest, risky)
  - Recommendation: 20-30% of replicas

Delay:
  - 0s: No delay (fastest)
  - 10s: Short delay (allow health checks)
  - 30s-60s: Conservative (monitor metrics)
  - Recommendation: 10-30s for production

FailureAction:
  - "pause": Stop deployment for manual investigation
  - "rollback": Automatically revert to previous version
  - "continue": Proceed despite failures
  - Recommendation: "rollback" for production

# Design Patterns

Orchestration Pattern:
  - Deployer coordinates, doesn't execute
  - Delegates to scheduler for task creation
  - Relies on reconciler for convergence
  - Separation of concerns

Declarative Updates:
  - Update service spec (image)
  - Shutdown old tasks (DesiredState=Shutdown)
  - Scheduler creates new tasks automatically
  - System converges to desired state

Batch Processing:
  - Process tasks in configurable batches
  - Wait between batches for stability
  - Fail-fast on errors (configurable)
  - Predictable update duration

Status Polling:
  - Clients poll GetDeploymentStatus
  - Real-time progress tracking
  - Task state aggregation
  - CLI progress bars

# Performance Characteristics

Rolling Update Timing:
  - Per task: ~5-15s (stop old + start new)
  - Parallelism=1: 5-15s × replica count
  - Parallelism=2: ~50% faster
  - Delay adds directly to total time

Example (10 replicas, parallelism=2, delay=10s):
  - Batches: 5 (10 replicas / 2 parallelism)
  - Per batch: ~15s (task update) + 10s (delay) = 25s
  - Total: 25s × 5 = ~125 seconds (~2 minutes)

Resource Usage:
  - Rolling: No additional resources (1:1 replacement)
  - Blue/Green: 2x resources temporarily
  - Canary: 1.1-1.5x resources during rollout

Rollback Speed:
  - Rolling: Same as forward (restart old tasks)
  - Blue/Green: Instant (switch VIP)
  - Canary: Gradual decrease of new version

# Troubleshooting

Common Issues:

Update Stalls:
  - Symptom: Deployment doesn't progress
  - Cause: New tasks failing health checks
  - Check: Task state, logs, image availability
  - Solution: Fix health check or image, retry update

Wrong Image Version:
  - Symptom: Tasks still running old version
  - Cause: Image cache, wrong tag
  - Check: Actual running image (docker inspect)
  - Solution: Force pull, use specific tag

Rollback Fails:
  - Symptom: Cannot revert to previous version
  - Cause: Previous image unavailable
  - Check: Image registry, local cache
  - Solution: Ensure previous version images retained

Mixed Versions:
  - Symptom: Both old and new versions running
  - Cause: Update in progress or stalled
  - Check: Deployment status, task states
  - Solution: Wait for completion or rollback

# Monitoring

Key metrics to monitor:

Deployment Progress:
  - deploy_tasks_total: Total tasks in deployment
  - deploy_tasks_ready: Tasks in ready state
  - deploy_progress_percent: (ready / total) × 100

Deployment Duration:
  - deploy_duration_seconds: Time from start to completion
  - deploy_batch_duration: Time per batch
  - deploy_delay_total: Total delay time

Deployment Outcomes:
  - deploy_success_total: Successful deployments
  - deploy_failed_total: Failed deployments
  - deploy_rollback_total: Rollbacks executed

# Rollback Procedure

Manual Rollback:

	// Get previous image version (from service history)
	previousImage := "nginx:1.20"

	// Update back to previous version
	err := deployer.UpdateService(serviceID, previousImage)
	if err != nil {
		log.Fatal(err)
	}

Automatic Rollback (Future):

	service.UpdateConfig.FailureAction = "rollback"

	// If new tasks fail, automatic rollback triggered
	err := deployer.UpdateService(serviceID, newImage)
	// Returns error but rollback already in progress

# Best Practices

Do:
  - Test updates in staging first
  - Use health checks for readiness
  - Set appropriate parallelism (20-30% of replicas)
  - Add delay between batches (10-30s)
  - Monitor metrics during deployment
  - Keep previous image versions available
  - Document rollback procedures

Don't:
  - Update all tasks at once (parallelism=N)
  - Skip health checks (deployment validation)
  - Use "latest" tag (not reproducible)
  - Deploy during high traffic (if possible)
  - Ignore deployment errors
  - Delete old image versions immediately

# Deployment Checklist

Pre-Deployment:
  - [ ] Test new version in staging
  - [ ] Review service health checks
  - [ ] Check image availability
  - [ ] Verify UpdateConfig parameters
  - [ ] Notify team of deployment window

During Deployment:
  - [ ] Monitor deployment status
  - [ ] Watch task state transitions
  - [ ] Check application metrics
  - [ ] Monitor error logs
  - [ ] Verify new version behavior

Post-Deployment:
  - [ ] Confirm all tasks healthy
  - [ ] Verify application functionality
  - [ ] Monitor for delayed issues
  - [ ] Document any issues encountered
  - [ ] Clean up old versions (after soak period)

# Future Enhancements

Planned for Milestone 4:
  - Blue/green deployment implementation
  - Canary deployment with traffic splitting
  - Automatic rollback on metrics threshold
  - Deployment history and versioning
  - Pause/resume deployment capability

Planned for Milestone 5:
  - Multi-phase canary (10% → 50% → 100%)
  - A/B testing integration
  - Deployment webhooks (notifications)
  - Custom deployment strategies (plugins)
  - Database migration coordination

# See Also

  - pkg/manager for service state management
  - pkg/scheduler for task scheduling
  - pkg/reconciler for state convergence
  - pkg/types for UpdateConfig definitions
  - docs/concepts/deployments.md for deployment guide
  - Kubernetes deployments: https://kubernetes.io/docs/concepts/workloads/controllers/deployment/
  - Blue/Green deployments: https://martinfowler.com/bliki/BlueGreenDeployment.html
  - Canary releases: https://martinfowler.com/bliki/CanaryRelease.html
*/
package deploy
