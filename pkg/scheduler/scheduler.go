package scheduler

import (
	"fmt"
	"sync"
	"time"

	"github.com/cuemby/warren/pkg/manager"
	"github.com/cuemby/warren/pkg/types"
	"github.com/google/uuid"
)

// Scheduler assigns tasks to nodes based on resource availability
type Scheduler struct {
	manager *manager.Manager
	mu      sync.RWMutex
	stopCh  chan struct{}
}

// NewScheduler creates a new scheduler
func NewScheduler(mgr *manager.Manager) *Scheduler {
	return &Scheduler{
		manager: mgr,
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

	// Count running/pending tasks
	activeTasks := 0
	for _, task := range tasks {
		if task.DesiredState == types.TaskStateRunning &&
			(task.ActualState == types.TaskStatePending || task.ActualState == types.TaskStateRunning) {
			activeTasks++
		}
	}

	// Calculate how many tasks we need
	var desiredTasks int
	if service.Mode == types.ServiceModeReplicated {
		desiredTasks = service.Replicas
	} else if service.Mode == types.ServiceModeGlobal {
		desiredTasks = len(nodes)
	}

	// Create missing tasks
	tasksToCreate := desiredTasks - activeTasks
	if tasksToCreate > 0 {
		for i := 0; i < tasksToCreate; i++ {
			node := s.selectNode(nodes, tasks)
			if node == nil {
				return fmt.Errorf("no suitable node found")
			}

			task := &types.Task{
				ID:          uuid.New().String(),
				ServiceID:   service.ID,
				ServiceName: service.Name,
				NodeID:      node.ID,
				DesiredState: types.TaskStateRunning,
				ActualState:  types.TaskStatePending,
				Image:       service.Image,
				Env:         service.Env,
				Resources:   service.Resources,
				HealthCheck: service.HealthCheck,
				RestartPolicy: service.RestartPolicy,
				CreatedAt:   time.Now(),
			}

			if err := s.manager.CreateTask(task); err != nil {
				return fmt.Errorf("failed to create task: %v", err)
			}

			fmt.Printf("Created task %s for service %s on node %s\n", task.ID, service.Name, node.ID)
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
					fmt.Printf("Failed to shutdown task %s: %v\n", task.ID, err)
					continue
				}
				removed++
			}
		}
	}

	return nil
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
