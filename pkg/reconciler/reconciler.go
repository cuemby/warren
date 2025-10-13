package reconciler

import (
	"fmt"
	"sync"
	"time"

	"github.com/cuemby/warren/pkg/manager"
	"github.com/cuemby/warren/pkg/metrics"
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
	// Start timing the reconciliation cycle
	timer := metrics.NewTimer()
	defer func() {
		timer.ObserveDuration(metrics.ReconciliationDuration)
		metrics.ReconciliationCyclesTotal.Inc()
	}()

	r.mu.Lock()
	defer r.mu.Unlock()

	// Reconcile nodes
	if err := r.reconcileNodes(); err != nil {
		fmt.Printf("Failed to reconcile nodes: %v\n", err)
	}

	// Reconcile containers
	if err := r.reconcileContainers(); err != nil {
		fmt.Printf("Failed to reconcile containers: %v\n", err)
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

// reconcileContainers ensures failed containers are replaced
func (r *Reconciler) reconcileContainers() error {
	containers, err := r.manager.ListContainers()
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	for _, container := range containers {
		// Handle failed containers
		if container.ActualState == types.ContainerStateFailed && container.DesiredState == types.ContainerStateRunning {
			fmt.Printf("Container %s failed on node %s, marking for cleanup\n", container.ID, container.NodeID)

			// Mark container as shutdown (scheduler will create replacement)
			container.DesiredState = types.ContainerStateShutdown
			if err := r.manager.UpdateContainer(container); err != nil {
				fmt.Printf("Failed to mark container %s for cleanup: %v\n", container.ID, err)
			}
		}

		// Handle unhealthy containers
		if container.ActualState == types.ContainerStateRunning && container.DesiredState == types.ContainerStateRunning {
			if container.HealthStatus != nil && !container.HealthStatus.Healthy {
				// Check if container has exceeded failure threshold
				// For now, we use a simple check: if unhealthy, mark as failed
				fmt.Printf("Container %s is unhealthy (%d consecutive failures): %s\n",
					container.ID, container.HealthStatus.ConsecutiveFailures, container.HealthStatus.Message)

				// Mark container as failed so it gets replaced
				container.ActualState = types.ContainerStateFailed
				container.Error = fmt.Sprintf("health check failed: %s", container.HealthStatus.Message)
				if err := r.manager.UpdateContainer(container); err != nil {
					fmt.Printf("Failed to mark unhealthy container %s as failed: %v\n", container.ID, err)
				}
			}
		}

		// Handle containers on down nodes
		node, err := r.manager.GetNode(container.NodeID)
		if err != nil {
			continue
		}

		if node.Status == types.NodeStatusDown && container.DesiredState == types.ContainerStateRunning {
			fmt.Printf("Container %s on down node %s, marking for rescheduling\n", container.ID, node.ID)

			// Mark container as failed so scheduler can create replacement
			container.ActualState = types.ContainerStateFailed
			container.DesiredState = types.ContainerStateShutdown
			if err := r.manager.UpdateContainer(container); err != nil {
				fmt.Printf("Failed to mark container %s as failed: %v\n", container.ID, err)
			}
		}

		// Clean up completed shutdown containers
		if container.DesiredState == types.ContainerStateShutdown && container.ActualState == types.ContainerStateComplete {
			// Container can be deleted after some grace period
			if time.Since(container.FinishedAt) > 5*time.Minute {
				if err := r.manager.DeleteContainer(container.ID); err != nil {
					fmt.Printf("Failed to delete completed container %s: %v\n", container.ID, err)
				}
			}
		}
	}

	return nil
}
