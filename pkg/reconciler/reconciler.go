package reconciler

import (
	"fmt"
	"sync"
	"time"

	"github.com/cuemby/warren/pkg/manager"
	"github.com/cuemby/warren/pkg/types"
)

// Reconciler ensures actual cluster state matches desired state
type Reconciler struct {
	manager *manager.Manager
	mu      sync.RWMutex
	stopCh  chan struct{}
}

// NewReconciler creates a new reconciler
func NewReconciler(mgr *manager.Manager) *Reconciler {
	return &Reconciler{
		manager: mgr,
		stopCh:  make(chan struct{}),
	}
}

// Start begins the reconciliation loop
func (r *Reconciler) Start() {
	go r.run()
}

// Stop stops the reconciler
func (r *Reconciler) Stop() {
	close(r.stopCh)
}

// run is the main reconciliation loop
func (r *Reconciler) run() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := r.reconcile(); err != nil {
				// Log error but continue
				fmt.Printf("Reconciler error: %v\n", err)
			}
		case <-r.stopCh:
			return
		}
	}
}

// reconcile performs one reconciliation cycle
func (r *Reconciler) reconcile() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Reconcile nodes
	if err := r.reconcileNodes(); err != nil {
		fmt.Printf("Failed to reconcile nodes: %v\n", err)
	}

	// Reconcile tasks
	if err := r.reconcileTasks(); err != nil {
		fmt.Printf("Failed to reconcile tasks: %v\n", err)
	}

	return nil
}

// reconcileNodes checks node health and updates status
func (r *Reconciler) reconcileNodes() error {
	nodes, err := r.manager.ListNodes()
	if err != nil {
		return fmt.Errorf("failed to list nodes: %w", err)
	}

	now := time.Now()
	for _, node := range nodes {
		// Check if node is down (no heartbeat in 30 seconds)
		if now.Sub(node.LastHeartbeat) > 30*time.Second {
			if node.Status != types.NodeStatusDown {
				fmt.Printf("Node %s is down (no heartbeat for %v)\n", node.ID, now.Sub(node.LastHeartbeat))
				node.Status = types.NodeStatusDown
				if err := r.manager.UpdateNode(node); err != nil {
					fmt.Printf("Failed to mark node %s as down: %v\n", node.ID, err)
				}
			}
		}
	}

	return nil
}

// reconcileTasks ensures failed tasks are replaced
func (r *Reconciler) reconcileTasks() error {
	tasks, err := r.manager.ListTasks()
	if err != nil {
		return fmt.Errorf("failed to list tasks: %w", err)
	}

	for _, task := range tasks {
		// Handle failed tasks
		if task.ActualState == types.TaskStateFailed && task.DesiredState == types.TaskStateRunning {
			fmt.Printf("Task %s failed on node %s, marking for cleanup\n", task.ID, task.NodeID)

			// Mark task as shutdown (scheduler will create replacement)
			task.DesiredState = types.TaskStateShutdown
			if err := r.manager.UpdateTask(task); err != nil {
				fmt.Printf("Failed to mark task %s for cleanup: %v\n", task.ID, err)
			}
		}

		// Handle unhealthy tasks
		if task.ActualState == types.TaskStateRunning && task.DesiredState == types.TaskStateRunning {
			if task.HealthStatus != nil && !task.HealthStatus.Healthy {
				// Check if task has exceeded failure threshold
				// For now, we use a simple check: if unhealthy, mark as failed
				fmt.Printf("Task %s is unhealthy (%d consecutive failures): %s\n",
					task.ID, task.HealthStatus.ConsecutiveFailures, task.HealthStatus.Message)

				// Mark task as failed so it gets replaced
				task.ActualState = types.TaskStateFailed
				task.Error = fmt.Sprintf("health check failed: %s", task.HealthStatus.Message)
				if err := r.manager.UpdateTask(task); err != nil {
					fmt.Printf("Failed to mark unhealthy task %s as failed: %v\n", task.ID, err)
				}
			}
		}

		// Handle tasks on down nodes
		node, err := r.manager.GetNode(task.NodeID)
		if err != nil {
			continue
		}

		if node.Status == types.NodeStatusDown && task.DesiredState == types.TaskStateRunning {
			fmt.Printf("Task %s on down node %s, marking for rescheduling\n", task.ID, node.ID)

			// Mark task as failed so scheduler can create replacement
			task.ActualState = types.TaskStateFailed
			task.DesiredState = types.TaskStateShutdown
			if err := r.manager.UpdateTask(task); err != nil {
				fmt.Printf("Failed to mark task %s as failed: %v\n", task.ID, err)
			}
		}

		// Clean up completed shutdown tasks
		if task.DesiredState == types.TaskStateShutdown && task.ActualState == types.TaskStateComplete {
			// Task can be deleted after some grace period
			if time.Since(task.FinishedAt) > 5*time.Minute {
				if err := r.manager.DeleteTask(task.ID); err != nil {
					fmt.Printf("Failed to delete completed task %s: %v\n", task.ID, err)
				}
			}
		}
	}

	return nil
}
