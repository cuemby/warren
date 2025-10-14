package deploy

import (
	"fmt"
	"time"

	"github.com/cuemby/warren/pkg/log"
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
	// Get all containers for the service
	containers, err := d.manager.ListContainersByService(service.ID)
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	// Filter running containers
	var runningContainers []*types.Container
	for _, container := range containers {
		if container.DesiredState == types.ContainerStateRunning {
			runningContainers = append(runningContainers, container)
		}
	}

	if len(runningContainers) == 0 {
		return fmt.Errorf("no running containers to update")
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

	log.Logger.Info().
		Str("service", service.Name).
		Str("service_id", service.ID).
		Str("current_image", service.Image).
		Str("new_image", newImage).
		Int("containers_to_update", len(runningContainers)).
		Int("parallelism", parallelism).
		Dur("delay", delay).
		Msg("Starting rolling update")

	// Update service image
	service.Image = newImage
	service.UpdatedAt = time.Now()
	if err := d.manager.UpdateService(service); err != nil {
		return fmt.Errorf("failed to update service: %w", err)
	}

	// Update containers in batches
	for i := 0; i < len(runningContainers); i += parallelism {
		end := i + parallelism
		if end > len(runningContainers) {
			end = len(runningContainers)
		}

		batch := runningContainers[i:end]
		batchNum := (i / parallelism) + 1
		totalBatches := (len(runningContainers) + parallelism - 1) / parallelism
		log.Logger.Info().
			Int("batch", batchNum).
			Int("total_batches", totalBatches).
			Int("containers", len(batch)).
			Msg("Updating batch")

		// Shutdown old containers
		for _, container := range batch {
			container.DesiredState = types.ContainerStateShutdown
			if err := d.manager.UpdateContainer(container); err != nil {
				log.Logger.Warn().
					Err(err).
					Str("container_id", container.ID).
					Msg("Failed to shutdown container")
				continue
			}
			log.Logger.Info().
				Str("container_id", container.ID[:8]).
				Str("node_id", container.NodeID).
				Msg("Shutting down container")
		}

		// The scheduler will automatically create new containers with the updated image
		// Wait for the delay before processing next batch
		if delay > 0 && end < len(runningContainers) {
			log.Logger.Info().Dur("delay", delay).Msg("Waiting before next batch")
			time.Sleep(delay)
		}
	}

	log.Logger.Info().
		Str("service", service.Name).
		Str("service_id", service.ID).
		Msg("Rolling update complete")
	return nil
}

// GetDeploymentStatus returns the status of a deployment
func (d *Deployer) GetDeploymentStatus(serviceID string) (*DeploymentStatus, error) {
	service, err := d.manager.GetService(serviceID)
	if err != nil {
		return nil, err
	}

	containers, err := d.manager.ListContainersByService(serviceID)
	if err != nil {
		return nil, err
	}

	status := &DeploymentStatus{
		ServiceID:   serviceID,
		ServiceName: service.Name,
		Image:       service.Image,
		Strategy:    string(service.DeployStrategy),
		Containers:  make(map[string]int),
	}

	for _, container := range containers {
		status.Containers[string(container.ActualState)]++
		if container.ActualState == types.ContainerStateRunning {
			status.ReadyContainers++
		}
	}

	status.TotalContainers = len(containers)
	status.DesiredContainers = service.Replicas

	return status, nil
}

// DeploymentStatus represents the current status of a deployment
type DeploymentStatus struct {
	ServiceID         string
	ServiceName       string
	Image             string
	Strategy          string
	TotalContainers   int
	DesiredContainers int
	ReadyContainers   int
	Containers        map[string]int // State -> Count
}
