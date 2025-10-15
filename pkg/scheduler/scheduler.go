package scheduler

import (
	"fmt"
	"sync"
	"time"

	"github.com/cuemby/warren/pkg/log"
	"github.com/cuemby/warren/pkg/manager"
	"github.com/cuemby/warren/pkg/metrics"
	"github.com/cuemby/warren/pkg/types"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// Scheduler assigns containers to nodes based on resource availability
type Scheduler struct {
	manager *manager.Manager
	logger  zerolog.Logger
	mu      sync.RWMutex
	stopCh  chan struct{}
}

// NewScheduler creates a new scheduler
func NewScheduler(mgr *manager.Manager) *Scheduler {
	return &Scheduler{
		manager: mgr,
		logger:  log.WithComponent("scheduler"),
		stopCh:  make(chan struct{}),
	}
}

// Start begins the scheduler loop
func (s *Scheduler) Start() {
	go s.run()
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	close(s.stopCh)
}

// run is the main scheduler loop
func (s *Scheduler) run() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := s.schedule(); err != nil {
				// Log error but continue
				s.logger.Error().Err(err).Msg("Scheduling cycle failed")
			}
		case <-s.stopCh:
			return
		}
	}
}

// schedule performs one scheduling cycle
func (s *Scheduler) schedule() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get all services
	services, err := s.manager.ListServices()
	if err != nil {
		return fmt.Errorf("failed to list services: %w", err)
	}

	// Get all nodes
	nodes, err := s.manager.ListNodes()
	if err != nil {
		return fmt.Errorf("failed to list nodes: %w", err)
	}

	// Filter schedulable nodes (workers and hybrid nodes)
	readyNodes := filterSchedulableNodes(nodes)
	if len(readyNodes) == 0 {
		// No schedulable nodes available
		s.logger.Warn().Msg("No schedulable nodes available. If this is a new cluster, ensure 'warren cluster init' completed (hybrid mode enabled by default)")
		return nil
	}

	// Schedule each service
	for _, service := range services {
		if err := s.scheduleService(service, readyNodes); err != nil {
			s.logger.Error().
				Err(err).
				Str("service_name", service.Name).
				Str("service_id", service.ID).
				Msg("Failed to schedule service")
			continue
		}
	}

	return nil
}

// scheduleService ensures the service has the correct number of containers
func (s *Scheduler) scheduleService(service *types.Service, nodes []*types.Node) error {
	// Get existing containers for this service
	containers, err := s.manager.ListContainersByService(service.ID)
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	if service.Mode == types.ServiceModeGlobal {
		return s.scheduleGlobalService(service, nodes, containers)
	}
	return s.scheduleReplicatedService(service, nodes, containers)
}

// scheduleGlobalService ensures one container per node for global services
func (s *Scheduler) scheduleGlobalService(service *types.Service, nodes []*types.Node, containers []*types.Container) error {
	// Build map of active containers by node
	nodeContainerMap := make(map[string]*types.Container)
	for _, container := range containers {
		if container.DesiredState == types.ContainerStateRunning &&
			(container.ActualState == types.ContainerStatePending || container.ActualState == types.ContainerStateRunning) {
			nodeContainerMap[container.NodeID] = container
		}
	}

	// Ensure each node has exactly one container
	for _, node := range nodes {
		if _, exists := nodeContainerMap[node.ID]; !exists {
			// Create container for this node
			timer := metrics.NewTimer()
			container := &types.Container{
				ID:            uuid.New().String(),
				ServiceID:     service.ID,
				ServiceName:   service.Name,
				NodeID:        node.ID,
				DesiredState:  types.ContainerStateRunning,
				ActualState:   types.ContainerStatePending,
				Image:         service.Image,
				Env:           service.Env,
				Mounts:        service.Volumes,
				Secrets:       service.Secrets,
				Resources:     service.Resources,
				HealthCheck:   service.HealthCheck,
				RestartPolicy: service.RestartPolicy,
				StopTimeout:   service.StopTimeout,
				CreatedAt:     time.Now(),
			}

			if err := s.manager.CreateContainer(container); err != nil {
				metrics.ContainersFailed.Inc()
				return fmt.Errorf("failed to create container: %w", err)
			}

			timer.ObserveDuration(metrics.SchedulingLatency)
			metrics.ContainersScheduled.Inc()

			s.logger.Info().
				Str("container_id", container.ID).
				Str("service_name", service.Name).
				Str("node_id", node.ID).
				Msg("Created global container")
		}
	}

	// Remove containers for nodes that no longer exist
	for _, container := range containers {
		if container.DesiredState != types.ContainerStateRunning {
			continue
		}

		nodeExists := false
		for _, node := range nodes {
			if node.ID == container.NodeID {
				nodeExists = true
				break
			}
		}

		if !nodeExists {
			container.DesiredState = types.ContainerStateShutdown
			if err := s.manager.UpdateContainer(container); err != nil {
				s.logger.Error().Err(err).Str("container_id", container.ID).Msg("Failed to shutdown container")
				continue
			}
			s.logger.Info().
				Str("container_id", container.ID).
				Str("node_id", container.NodeID).
				Msg("Removed global container (node no longer exists)")
		}
	}

	return nil
}

// scheduleReplicatedService handles replicated service scheduling
func (s *Scheduler) scheduleReplicatedService(service *types.Service, nodes []*types.Node, containers []*types.Container) error {
	// Count running/pending containers
	activeContainers := 0
	for _, container := range containers {
		if container.DesiredState == types.ContainerStateRunning &&
			(container.ActualState == types.ContainerStatePending || container.ActualState == types.ContainerStateRunning) {
			activeContainers++
		}
	}

	desiredContainers := service.Replicas
	containersToCreate := desiredContainers - activeContainers

	// Create missing containers
	if containersToCreate > 0 {
		for i := 0; i < containersToCreate; i++ {
			timer := metrics.NewTimer()

			// Check if service has volume requirements
			node, err := s.selectNodeForService(service, nodes, containers)
			if err != nil {
				metrics.ContainersFailed.Inc()
				return fmt.Errorf("failed to select node: %w", err)
			}
			if node == nil {
				metrics.ContainersFailed.Inc()
				return fmt.Errorf("no suitable node found")
			}

			container := &types.Container{
				ID:            uuid.New().String(),
				ServiceID:     service.ID,
				ServiceName:   service.Name,
				NodeID:        node.ID,
				DesiredState:  types.ContainerStateRunning,
				ActualState:   types.ContainerStatePending,
				Image:         service.Image,
				Env:           service.Env,
				Mounts:        service.Volumes,
				Secrets:       service.Secrets,
				Resources:     service.Resources,
				HealthCheck:   service.HealthCheck,
				RestartPolicy: service.RestartPolicy,
				StopTimeout:   service.StopTimeout,
				CreatedAt:     time.Now(),
			}

			if err := s.manager.CreateContainer(container); err != nil {
				metrics.ContainersFailed.Inc()
				return fmt.Errorf("failed to create container: %w", err)
			}

			timer.ObserveDuration(metrics.SchedulingLatency)
			metrics.ContainersScheduled.Inc()

			s.logger.Info().
				Str("container_id", container.ID).
				Str("service_name", service.Name).
				Str("node_id", node.ID).
				Msg("Created container")
		}
	}

	// Remove excess containers
	if containersToCreate < 0 {
		containersToRemove := -containersToCreate
		removed := 0
		for _, container := range containers {
			if removed >= containersToRemove {
				break
			}
			if container.DesiredState == types.ContainerStateRunning {
				container.DesiredState = types.ContainerStateShutdown
				if err := s.manager.UpdateContainer(container); err != nil {
					s.logger.Error().Err(err).Str("container_id", container.ID).Msg("Failed to shutdown container")
					continue
				}
				removed++
			}
		}
	}

	return nil
}

// selectNodeForService selects a node for a service, considering volume affinity
func (s *Scheduler) selectNodeForService(service *types.Service, nodes []*types.Node, existingContainers []*types.Container) (*types.Node, error) {
	// Check if service has volume requirements
	if len(service.Volumes) > 0 {
		// Find node with volume affinity
		for _, volumeMount := range service.Volumes {
			volume, err := s.manager.GetVolumeByName(volumeMount.Source)
			if err != nil {
				// Volume doesn't exist yet, will be created on selected node
				continue
			}

			// If volume exists and has node affinity, use that node
			if volume.NodeID != "" {
				for _, node := range nodes {
					if node.ID == volume.NodeID {
						s.logger.Debug().
							Str("node_id", node.ID).
							Str("service_name", service.Name).
							Str("volume_name", volume.Name).
							Msg("Selected node for service (volume affinity)")
						return node, nil
					}
				}
				return nil, fmt.Errorf("volume %s requires node %s which is not available", volume.Name, volume.NodeID)
			}
		}
	}

	// No volume affinity, use standard selection
	return s.selectNode(nodes, existingContainers), nil
}

// selectNode implements simple round-robin node selection
func (s *Scheduler) selectNode(nodes []*types.Node, existingContainers []*types.Container) *types.Node {
	if len(nodes) == 0 {
		return nil
	}

	// Count containers per node
	containerCounts := make(map[string]int)
	for _, container := range existingContainers {
		if container.DesiredState == types.ContainerStateRunning {
			containerCounts[container.NodeID]++
		}
	}

	// Find node with fewest containers (simple load balancing)
	var selectedNode *types.Node
	minContainers := int(^uint(0) >> 1) // Max int

	for _, node := range nodes {
		count := containerCounts[node.ID]
		if count < minContainers {
			minContainers = count
			selectedNode = node
		}
	}

	return selectedNode
}

// filterSchedulableNodes returns nodes that can run workloads (workers and hybrid nodes)
func filterSchedulableNodes(nodes []*types.Node) []*types.Node {
	var ready []*types.Node
	for _, node := range nodes {
		// Include workers AND hybrid nodes (managers that can run workloads)
		if (node.Role == types.NodeRoleWorker || node.Role == types.NodeRoleHybrid) &&
			node.Status == types.NodeStatusReady {
			ready = append(ready, node)
		}
	}
	return ready
}
