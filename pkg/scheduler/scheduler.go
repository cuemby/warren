package scheduler

import (
	"fmt"
	"sync"
	"time"

	"github.com/cuemby/warren/pkg/log"
	"github.com/cuemby/warren/pkg/manager"
	"github.com/cuemby/warren/pkg/types"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// Scheduler assigns tasks to nodes based on resource availability
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
				fmt.Printf("Scheduler error: %v\n", err)
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
		return fmt.Errorf("failed to list services: %v", err)
	}

	// Get all nodes
	nodes, err := s.manager.ListNodes()
	if err != nil {
		return fmt.Errorf("failed to list nodes: %v", err)
	}

	// Filter ready worker nodes
	readyNodes := filterReadyWorkers(nodes)
	if len(readyNodes) == 0 {
		// No workers available, skip scheduling
		return nil
	}

	// Schedule each service
	for _, service := range services {
		if err := s.scheduleService(service, readyNodes); err != nil {
			fmt.Printf("Failed to schedule service %s: %v\n", service.Name, err)
			continue
		}
	}

	return nil
}

// scheduleService ensures the service has the correct number of tasks
func (s *Scheduler) scheduleService(service *types.Service, nodes []*types.Node) error {
	// Get existing tasks for this service
	tasks, err := s.manager.ListTasksByService(service.ID)
	if err != nil {
		return fmt.Errorf("failed to list tasks: %v", err)
	}

	if service.Mode == types.ServiceModeGlobal {
		return s.scheduleGlobalService(service, nodes, tasks)
	}
	return s.scheduleReplicatedService(service, nodes, tasks)
}

// scheduleGlobalService ensures one task per node for global services
func (s *Scheduler) scheduleGlobalService(service *types.Service, nodes []*types.Node, tasks []*types.Task) error {
	// Build map of active tasks by node
	nodeTaskMap := make(map[string]*types.Task)
	for _, task := range tasks {
		if task.DesiredState == types.TaskStateRunning &&
			(task.ActualState == types.TaskStatePending || task.ActualState == types.TaskStateRunning) {
			nodeTaskMap[task.NodeID] = task
		}
	}

	// Ensure each node has exactly one task
	for _, node := range nodes {
		if _, exists := nodeTaskMap[node.ID]; !exists {
			// Create task for this node
			task := &types.Task{
				ID:            uuid.New().String(),
				ServiceID:     service.ID,
				ServiceName:   service.Name,
				NodeID:        node.ID,
				DesiredState:  types.TaskStateRunning,
				ActualState:   types.TaskStatePending,
				Image:         service.Image,
				Env:           service.Env,
				Mounts:        service.Volumes,
				Secrets:       service.Secrets,
				Resources:     service.Resources,
				HealthCheck:   service.HealthCheck,
				RestartPolicy: service.RestartPolicy,
				CreatedAt:     time.Now(),
			}

			if err := s.manager.CreateTask(task); err != nil {
				return fmt.Errorf("failed to create task: %v", err)
			}

			s.logger.Info().
				Str("task_id", task.ID).
				Str("service_name", service.Name).
				Str("node_id", node.ID).
				Msg("Created global task")
		}
	}

	// Remove tasks for nodes that no longer exist
	for _, task := range tasks {
		if task.DesiredState != types.TaskStateRunning {
			continue
		}

		nodeExists := false
		for _, node := range nodes {
			if node.ID == task.NodeID {
				nodeExists = true
				break
			}
		}

		if !nodeExists {
			task.DesiredState = types.TaskStateShutdown
			if err := s.manager.UpdateTask(task); err != nil {
				s.logger.Error().Err(err).Str("task_id", task.ID).Msg("Failed to shutdown task")
				continue
			}
			s.logger.Info().
				Str("task_id", task.ID).
				Str("node_id", task.NodeID).
				Msg("Removed global task (node no longer exists)")
		}
	}

	return nil
}

// scheduleReplicatedService handles replicated service scheduling
func (s *Scheduler) scheduleReplicatedService(service *types.Service, nodes []*types.Node, tasks []*types.Task) error {
	// Count running/pending tasks
	activeTasks := 0
	for _, task := range tasks {
		if task.DesiredState == types.TaskStateRunning &&
			(task.ActualState == types.TaskStatePending || task.ActualState == types.TaskStateRunning) {
			activeTasks++
		}
	}

	desiredTasks := service.Replicas
	tasksToCreate := desiredTasks - activeTasks

	// Create missing tasks
	if tasksToCreate > 0 {
		for i := 0; i < tasksToCreate; i++ {
			// Check if service has volume requirements
			node, err := s.selectNodeForService(service, nodes, tasks)
			if err != nil {
				return fmt.Errorf("failed to select node: %v", err)
			}
			if node == nil {
				return fmt.Errorf("no suitable node found")
			}

			task := &types.Task{
				ID:            uuid.New().String(),
				ServiceID:     service.ID,
				ServiceName:   service.Name,
				NodeID:        node.ID,
				DesiredState:  types.TaskStateRunning,
				ActualState:   types.TaskStatePending,
				Image:         service.Image,
				Env:           service.Env,
				Mounts:        service.Volumes,
				Secrets:       service.Secrets,
				Resources:     service.Resources,
				HealthCheck:   service.HealthCheck,
				RestartPolicy: service.RestartPolicy,
				CreatedAt:     time.Now(),
			}

			if err := s.manager.CreateTask(task); err != nil {
				return fmt.Errorf("failed to create task: %v", err)
			}

			s.logger.Info().
				Str("task_id", task.ID).
				Str("service_name", service.Name).
				Str("node_id", node.ID).
				Msg("Created task")
		}
	}

	// Remove excess tasks
	if tasksToCreate < 0 {
		tasksToRemove := -tasksToCreate
		removed := 0
		for _, task := range tasks {
			if removed >= tasksToRemove {
				break
			}
			if task.DesiredState == types.TaskStateRunning {
				task.DesiredState = types.TaskStateShutdown
				if err := s.manager.UpdateTask(task); err != nil {
					s.logger.Error().Err(err).Str("task_id", task.ID).Msg("Failed to shutdown task")
					continue
				}
				removed++
			}
		}
	}

	return nil
}

// selectNodeForService selects a node for a service, considering volume affinity
func (s *Scheduler) selectNodeForService(service *types.Service, nodes []*types.Node, existingTasks []*types.Task) (*types.Node, error) {
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
						fmt.Printf("Selecting node %s for service %s (volume affinity: %s)\n",
							node.ID, service.Name, volume.Name)
						return node, nil
					}
				}
				return nil, fmt.Errorf("volume %s requires node %s which is not available", volume.Name, volume.NodeID)
			}
		}
	}

	// No volume affinity, use standard selection
	return s.selectNode(nodes, existingTasks), nil
}

// selectNode implements simple round-robin node selection
func (s *Scheduler) selectNode(nodes []*types.Node, existingTasks []*types.Task) *types.Node {
	if len(nodes) == 0 {
		return nil
	}

	// Count tasks per node
	taskCounts := make(map[string]int)
	for _, task := range existingTasks {
		if task.DesiredState == types.TaskStateRunning {
			taskCounts[task.NodeID]++
		}
	}

	// Find node with fewest tasks (simple load balancing)
	var selectedNode *types.Node
	minTasks := int(^uint(0) >> 1) // Max int

	for _, node := range nodes {
		count := taskCounts[node.ID]
		if count < minTasks {
			minTasks = count
			selectedNode = node
		}
	}

	return selectedNode
}

// filterReadyWorkers returns only worker nodes that are ready
func filterReadyWorkers(nodes []*types.Node) []*types.Node {
	var ready []*types.Node
	for _, node := range nodes {
		if node.Role == types.NodeRoleWorker && node.Status == types.NodeStatusReady {
			ready = append(ready, node)
		}
	}
	return ready
}
