package deploy

import (
	"fmt"
	"time"

	"github.com/cuemby/warren/pkg/manager"
	"github.com/cuemby/warren/pkg/types"
)

// Deployer handles service deployment strategies
type Deployer struct {
	manager *manager.Manager
}

// NewDeployer creates a new deployer
func NewDeployer(mgr *manager.Manager) *Deployer {
	return &Deployer{
		manager: mgr,
	}
}

// UpdateService updates a service with the specified strategy
func (d *Deployer) UpdateService(serviceID string, newImage string) error {
	service, err := d.manager.GetService(serviceID)
	if err != nil {
		return fmt.Errorf("failed to get service: %w", err)
	}

	// For now, always use rolling update
	// Blue/green and canary will be added in Milestone 4
	return d.rollingUpdate(service, newImage)
}

// rollingUpdate performs a rolling update of the service
func (d *Deployer) rollingUpdate(service *types.Service, newImage string) error {
	// Get all tasks for the service
	tasks, err := d.manager.ListTasksByService(service.ID)
	if err != nil {
		return fmt.Errorf("failed to list tasks: %w", err)
	}

	// Filter running tasks
	var runningTasks []*types.Task
	for _, task := range tasks {
		if task.DesiredState == types.TaskStateRunning {
			runningTasks = append(runningTasks, task)
		}
	}

	if len(runningTasks) == 0 {
		return fmt.Errorf("no running tasks to update")
	}

	// Determine update parallelism
	parallelism := 1
	if service.UpdateConfig != nil && service.UpdateConfig.Parallelism > 0 {
		parallelism = service.UpdateConfig.Parallelism
	}

	// Determine delay between batches
	delay := 0 * time.Second
	if service.UpdateConfig != nil {
		delay = service.UpdateConfig.Delay
	}

	fmt.Printf("Starting rolling update for service %s:\n", service.Name)
	fmt.Printf("  Current image: %s\n", service.Image)
	fmt.Printf("  New image: %s\n", newImage)
	fmt.Printf("  Tasks to update: %d\n", len(runningTasks))
	fmt.Printf("  Parallelism: %d\n", parallelism)
	fmt.Printf("  Delay: %v\n", delay)

	// Update service image
	service.Image = newImage
	service.UpdatedAt = time.Now()
	if err := d.manager.UpdateService(service); err != nil {
		return fmt.Errorf("failed to update service: %w", err)
	}

	// Update tasks in batches
	for i := 0; i < len(runningTasks); i += parallelism {
		end := i + parallelism
		if end > len(runningTasks) {
			end = len(runningTasks)
		}

		batch := runningTasks[i:end]
		fmt.Printf("\nUpdating batch %d/%d (%d tasks)...\n",
			(i/parallelism)+1,
			(len(runningTasks)+parallelism-1)/parallelism,
			len(batch))

		// Shutdown old tasks
		for _, task := range batch {
			task.DesiredState = types.TaskStateShutdown
			if err := d.manager.UpdateTask(task); err != nil {
				fmt.Printf("  Warning: failed to shutdown task %s: %v\n", task.ID, err)
				continue
			}
			fmt.Printf("  Shutting down task %s on node %s\n", task.ID[:8], task.NodeID)
		}

		// The scheduler will automatically create new tasks with the updated image
		// Wait for the delay before processing next batch
		if delay > 0 && end < len(runningTasks) {
			fmt.Printf("  Waiting %v before next batch...\n", delay)
			time.Sleep(delay)
		}
	}

	fmt.Printf("\nRolling update complete for service %s\n", service.Name)
	return nil
}

// GetDeploymentStatus returns the status of a deployment
func (d *Deployer) GetDeploymentStatus(serviceID string) (*DeploymentStatus, error) {
	service, err := d.manager.GetService(serviceID)
	if err != nil {
		return nil, err
	}

	tasks, err := d.manager.ListTasksByService(serviceID)
	if err != nil {
		return nil, err
	}

	status := &DeploymentStatus{
		ServiceID:   serviceID,
		ServiceName: service.Name,
		Image:       service.Image,
		Strategy:    string(service.DeployStrategy),
		Tasks:       make(map[string]int),
	}

	for _, task := range tasks {
		status.Tasks[string(task.ActualState)]++
		if task.ActualState == types.TaskStateRunning {
			status.ReadyTasks++
		}
	}

	status.TotalTasks = len(tasks)
	status.DesiredTasks = service.Replicas

	return status, nil
}

// DeploymentStatus represents the current status of a deployment
type DeploymentStatus struct {
	ServiceID    string
	ServiceName  string
	Image        string
	Strategy     string
	TotalTasks   int
	DesiredTasks int
	ReadyTasks   int
	Tasks        map[string]int // State -> Count
}
