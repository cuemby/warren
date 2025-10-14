package reconciler

import (
	"fmt"
	"sync"
	"time"

	"github.com/cuemby/warren/pkg/log"
	"github.com/cuemby/warren/pkg/manager"
	"github.com/cuemby/warren/pkg/metrics"
	"github.com/cuemby/warren/pkg/types"
	"github.com/rs/zerolog"
)

// Reconciler ensures actual cluster state matches desired state
type Reconciler struct {
	manager *manager.Manager
	logger  zerolog.Logger
	mu      sync.RWMutex
	stopCh  chan struct{}
}

// NewReconciler creates a new reconciler
func NewReconciler(mgr *manager.Manager) *Reconciler {
	return &Reconciler{
		manager: mgr,
		logger:  log.WithComponent("reconciler"),
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

	r.logger.Info().Msg("Reconciler started")

	for {
		select {
		case <-ticker.C:
			if err := r.reconcile(); err != nil {
				// Log error but continue
				r.logger.Error().Err(err).Msg("Reconciliation cycle failed")
			}
		case <-r.stopCh:
			r.logger.Info().Msg("Reconciler stopped")
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
		r.logger.Error().Err(err).Msg("Failed to reconcile nodes")
	}

	// Reconcile containers
	if err := r.reconcileContainers(); err != nil {
		r.logger.Error().Err(err).Msg("Failed to reconcile containers")
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
				r.logger.Warn().
					Str("node_id", node.ID).
					Dur("no_heartbeat_duration", now.Sub(node.LastHeartbeat)).
					Msg("Node is down, marking as down")
				node.Status = types.NodeStatusDown
				if err := r.manager.UpdateNode(node); err != nil {
					r.logger.Error().
						Err(err).
						Str("node_id", node.ID).
						Msg("Failed to mark node as down")
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
			r.logger.Info().
				Str("container_id", container.ID).
				Str("node_id", container.NodeID).
				Msg("Container failed, marking for cleanup")

			// Mark container as shutdown (scheduler will create replacement)
			container.DesiredState = types.ContainerStateShutdown
			if err := r.manager.UpdateContainer(container); err != nil {
				r.logger.Error().
					Err(err).
					Str("container_id", container.ID).
					Msg("Failed to mark container for cleanup")
			}
		}

		// Handle unhealthy containers
		if container.ActualState == types.ContainerStateRunning && container.DesiredState == types.ContainerStateRunning {
			if container.HealthStatus != nil && !container.HealthStatus.Healthy {
				// Check if container has exceeded failure threshold
				// For now, we use a simple check: if unhealthy, mark as failed
				r.logger.Warn().
					Str("container_id", container.ID).
					Int("consecutive_failures", container.HealthStatus.ConsecutiveFailures).
					Str("health_message", container.HealthStatus.Message).
					Msg("Container is unhealthy, marking as failed")

				// Mark container as failed so it gets replaced
				container.ActualState = types.ContainerStateFailed
				container.Error = fmt.Sprintf("health check failed: %s", container.HealthStatus.Message)
				if err := r.manager.UpdateContainer(container); err != nil {
					r.logger.Error().
						Err(err).
						Str("container_id", container.ID).
						Msg("Failed to mark unhealthy container as failed")
				}
			}
		}

		// Handle containers on down nodes
		node, err := r.manager.GetNode(container.NodeID)
		if err != nil {
			r.logger.Debug().
				Err(err).
				Str("container_id", container.ID).
				Str("node_id", container.NodeID).
				Msg("Could not get node for container")
			continue
		}

		if node.Status == types.NodeStatusDown && container.DesiredState == types.ContainerStateRunning {
			r.logger.Info().
				Str("container_id", container.ID).
				Str("node_id", node.ID).
				Msg("Container on down node, marking for rescheduling")

			// Mark container as failed so scheduler can create replacement
			container.ActualState = types.ContainerStateFailed
			container.DesiredState = types.ContainerStateShutdown
			if err := r.manager.UpdateContainer(container); err != nil {
				r.logger.Error().
					Err(err).
					Str("container_id", container.ID).
					Msg("Failed to mark container as failed")
			}
		}

		// Clean up completed shutdown containers
		if container.DesiredState == types.ContainerStateShutdown && container.ActualState == types.ContainerStateComplete {
			// Container can be deleted after some grace period
			if time.Since(container.FinishedAt) > 5*time.Minute {
				r.logger.Debug().
					Str("container_id", container.ID).
					Msg("Deleting completed container")
				if err := r.manager.DeleteContainer(container.ID); err != nil {
					r.logger.Error().
						Err(err).
						Str("container_id", container.ID).
						Msg("Failed to delete completed container")
				}
			}
		}
	}

	return nil
}
