package deploy

import (
	"fmt"
	"time"

	"github.com/cuemby/warren/pkg/log"
	"github.com/cuemby/warren/pkg/metrics"
	"github.com/cuemby/warren/pkg/types"
	"github.com/google/uuid"
)

// Deployer handles service deployment strategies
type Deployer struct {
	manager types.ServiceManager
}

// NewDeployer creates a new deployer
func NewDeployer(mgr types.ServiceManager) *Deployer {
	return &Deployer{
		manager: mgr,
	}
}

// UpdateService updates a service with the specified strategy
func (d *Deployer) UpdateService(serviceID string, newImage string, strategy types.DeployStrategy) error {
	// Start deployment timer
	startTime := time.Now()

	service, err := d.manager.GetService(serviceID)
	if err != nil {
		return fmt.Errorf("failed to get service: %w", err)
	}

	// Use specified strategy or service's default
	if strategy == "" {
		strategy = service.DeployStrategy
	}
	if strategy == "" {
		strategy = types.DeployStrategyRolling // Default to rolling
	}

	log.Logger.Info().
		Str("service", service.Name).
		Str("strategy", string(strategy)).
		Str("old_image", service.Image).
		Str("new_image", newImage).
		Msg("Starting service update")

	var deployErr error
	switch strategy {
	case types.DeployStrategyBlueGreen:
		deployErr = d.blueGreenUpdate(service, newImage)
	case types.DeployStrategyCanary:
		deployErr = d.canaryUpdate(service, newImage)
	case types.DeployStrategyRolling:
		deployErr = d.rollingUpdate(service, newImage)
	default:
		deployErr = fmt.Errorf("unknown deployment strategy: %s", strategy)
	}

	// Record deployment metrics
	duration := time.Since(startTime).Seconds()
	status := "success"
	if deployErr != nil {
		status = "failed"
	}

	metrics.DeploymentsTotal.WithLabelValues(string(strategy), status).Inc()
	metrics.DeploymentDuration.WithLabelValues(string(strategy)).Observe(duration)

	return deployErr
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

// blueGreenUpdate performs a blue-green deployment of the service
func (d *Deployer) blueGreenUpdate(service *types.Service, newImage string) error {
	log.Logger.Info().
		Str("service", service.Name).
		Str("service_id", service.ID).
		Str("current_image", service.Image).
		Str("new_image", newImage).
		Msg("Starting blue-green deployment")

	// Generate version identifier
	version := uuid.New().String()[:8]

	// Clone service as "green" version with new image
	greenService := d.cloneServiceForDeployment(service, newImage, version, types.DeploymentStateStandby)

	// Create the green service
	if err := d.manager.CreateService(greenService); err != nil {
		return fmt.Errorf("failed to create green service: %w", err)
	}

	log.Logger.Info().
		Str("green_service_id", greenService.ID).
		Str("version", version).
		Msg("Created green service")

	// Wait for green containers to become healthy
	if err := d.waitForHealthyContainers(greenService); err != nil {
		// Cleanup green service on failure
		_ = d.manager.DeleteService(greenService.ID)
		return fmt.Errorf("green service failed health checks: %w", err)
	}

	log.Logger.Info().
		Str("green_service_id", greenService.ID).
		Msg("Green service is healthy, switching traffic")

	// Switch traffic: mark green as active, blue as standby
	greenService.Labels[types.LabelDeploymentState] = string(types.DeploymentStateActive)
	if err := d.manager.UpdateService(greenService); err != nil {
		return fmt.Errorf("failed to activate green service: %w", err)
	}

	service.Labels[types.LabelDeploymentState] = string(types.DeploymentStateStandby)
	service.UpdatedAt = time.Now()
	if err := d.manager.UpdateService(service); err != nil {
		log.Logger.Warn().Err(err).Msg("Failed to mark blue service as standby")
	}

	log.Logger.Info().
		Str("service", service.Name).
		Str("blue_service_id", service.ID).
		Str("green_service_id", greenService.ID).
		Str("version", version).
		Msg("Blue-green deployment complete")

	// TODO: After grace period, cleanup blue service
	// This should be handled by a background cleanup job

	return nil
}

// cloneServiceForDeployment creates a copy of a service for deployment
func (d *Deployer) cloneServiceForDeployment(original *types.Service, newImage string, version string, state types.DeploymentState) *types.Service {
	clone := &types.Service{
		ID:             uuid.New().String(),
		Name:           original.Name + "-" + version,
		Image:          newImage,
		Replicas:       original.Replicas,
		Mode:           original.Mode,
		DeployStrategy: original.DeployStrategy,
		UpdateConfig:   original.UpdateConfig,
		Env:            original.Env,
		Ports:          original.Ports,
		Networks:       original.Networks,
		Secrets:        original.Secrets,
		Volumes:        original.Volumes,
		Labels:         make(map[string]string),
		HealthCheck:    original.HealthCheck,
		RestartPolicy:  original.RestartPolicy,
		Resources:      original.Resources,
		StopTimeout:    original.StopTimeout,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Copy original labels
	for k, v := range original.Labels {
		clone.Labels[k] = v
	}

	// Add deployment tracking labels
	clone.Labels[types.LabelDeploymentVersion] = version
	clone.Labels[types.LabelDeploymentState] = string(state)
	clone.Labels[types.LabelDeploymentStrategy] = string(original.DeployStrategy)
	clone.Labels[types.LabelOriginalService] = original.ID

	return clone
}

// waitForHealthyContainers waits for all containers of a service to become healthy
func (d *Deployer) waitForHealthyContainers(service *types.Service) error {
	gracePeriod := 30 * time.Second
	if service.UpdateConfig != nil && service.UpdateConfig.HealthCheckGracePeriod > 0 {
		gracePeriod = service.UpdateConfig.HealthCheckGracePeriod
	}

	timeout := time.After(5 * time.Minute) // Overall timeout
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	log.Logger.Info().
		Str("service_id", service.ID).
		Dur("grace_period", gracePeriod).
		Msg("Waiting for containers to become healthy")

	// Initial grace period
	time.Sleep(gracePeriod)

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for healthy containers")
		case <-ticker.C:
			containers, err := d.manager.ListContainersByService(service.ID)
			if err != nil {
				return fmt.Errorf("failed to list containers: %w", err)
			}

			if len(containers) == 0 {
				continue
			}

			// Check if all containers are running and healthy
			allHealthy := true
			runningCount := 0

			for _, container := range containers {
				if container.DesiredState != types.ContainerStateRunning {
					continue
				}

				if container.ActualState != types.ContainerStateRunning {
					allHealthy = false
					break
				}

				runningCount++

				// If health check is configured, verify health status
				if container.HealthCheck != nil && container.HealthStatus != nil {
					if !container.HealthStatus.Healthy {
						allHealthy = false
						break
					}
				}
			}

			if allHealthy && runningCount == service.Replicas {
				log.Logger.Info().
					Str("service_id", service.ID).
					Int("healthy_containers", runningCount).
					Msg("All containers are healthy")
				return nil
			}

			log.Logger.Debug().
				Str("service_id", service.ID).
				Int("running", runningCount).
				Int("desired", service.Replicas).
				Bool("all_healthy", allHealthy).
				Msg("Waiting for containers to become healthy")
		}
	}
}

// canaryUpdate performs a canary deployment of the service
func (d *Deployer) canaryUpdate(service *types.Service, newImage string) error {
	log.Logger.Info().
		Str("service", service.Name).
		Str("service_id", service.ID).
		Str("current_image", service.Image).
		Str("new_image", newImage).
		Msg("Starting canary deployment")

	// Generate version identifier
	version := uuid.New().String()[:8]

	// Determine canary steps (default: [10, 25, 50, 100])
	canarySteps := []int{10, 25, 50, 100}
	if service.UpdateConfig != nil && len(service.UpdateConfig.CanarySteps) > 0 {
		canarySteps = service.UpdateConfig.CanarySteps
	}

	// Determine stability window between steps
	stabilityWindow := 5 * time.Minute
	if service.UpdateConfig != nil && service.UpdateConfig.CanaryStabilityWindow > 0 {
		stabilityWindow = service.UpdateConfig.CanaryStabilityWindow
	}

	// Clone service as canary version
	canaryService := d.cloneServiceForDeployment(service, newImage, version, types.DeploymentStateCanary)

	// Start with 1 replica for canary
	canaryService.Replicas = 1

	if err := d.manager.CreateService(canaryService); err != nil {
		return fmt.Errorf("failed to create canary service: %w", err)
	}

	log.Logger.Info().
		Str("canary_service_id", canaryService.ID).
		Str("version", version).
		Msg("Created canary service")

	// Wait for initial canary to become healthy
	if err := d.waitForHealthyContainers(canaryService); err != nil {
		_ = d.manager.DeleteService(canaryService.ID)
		return fmt.Errorf("canary service failed initial health checks: %w", err)
	}

	// Progressive canary rollout
	for i, weight := range canarySteps {
		log.Logger.Info().
			Str("canary_service_id", canaryService.ID).
			Int("step", i+1).
			Int("weight", weight).
			Msg("Progressing canary deployment")

		// Calculate desired replicas based on weight
		totalReplicas := service.Replicas
		canaryReplicas := (totalReplicas * weight) / 100
		if canaryReplicas < 1 {
			canaryReplicas = 1
		}
		stableReplicas := totalReplicas - canaryReplicas

		// Update canary service replicas
		canaryService.Replicas = canaryReplicas
		if err := d.manager.UpdateService(canaryService); err != nil {
			return fmt.Errorf("failed to scale canary service: %w", err)
		}

		// Update stable service replicas
		service.Replicas = stableReplicas
		if err := d.manager.UpdateService(service); err != nil {
			return fmt.Errorf("failed to scale stable service: %w", err)
		}

		// Wait for new canary replicas to become healthy
		if err := d.waitForHealthyContainers(canaryService); err != nil {
			log.Logger.Error().Err(err).Msg("Canary health check failed, initiating rollback")
			return d.rollbackCanary(service, canaryService, totalReplicas)
		}

		// Update canary weight in config
		if service.UpdateConfig == nil {
			service.UpdateConfig = &types.UpdateConfig{}
		}
		service.UpdateConfig.CanaryWeight = weight
		_ = d.manager.UpdateService(service)

		// Wait for stability window before next step (unless this is the final step)
		if i < len(canarySteps)-1 {
			log.Logger.Info().
				Dur("stability_window", stabilityWindow).
				Msg("Waiting for stability window before next canary step")
			time.Sleep(stabilityWindow)
		}
	}

	// Canary is fully rolled out, mark it as active
	canaryService.Labels[types.LabelDeploymentState] = string(types.DeploymentStateActive)
	canaryService.Name = service.Name // Restore original name
	if err := d.manager.UpdateService(canaryService); err != nil {
		return fmt.Errorf("failed to activate canary service: %w", err)
	}

	// Mark old service as standby
	service.Labels[types.LabelDeploymentState] = string(types.DeploymentStateStandby)
	service.Replicas = 0
	if err := d.manager.UpdateService(service); err != nil {
		log.Logger.Warn().Err(err).Msg("Failed to mark old service as standby")
	}

	log.Logger.Info().
		Str("service", service.Name).
		Str("old_service_id", service.ID).
		Str("new_service_id", canaryService.ID).
		Str("version", version).
		Msg("Canary deployment complete")

	return nil
}

// rollbackCanary rolls back a failed canary deployment
func (d *Deployer) rollbackCanary(stableService, canaryService *types.Service, originalReplicas int) error {
	log.Logger.Warn().
		Str("stable_service_id", stableService.ID).
		Str("canary_service_id", canaryService.ID).
		Msg("Rolling back canary deployment")

	// Track rollback metric
	metrics.RolledBackDeploymentsTotal.WithLabelValues("canary", "health_check_failed").Inc()

	// Restore stable service to full replicas
	stableService.Replicas = originalReplicas
	if err := d.manager.UpdateService(stableService); err != nil {
		return fmt.Errorf("failed to restore stable service replicas: %w", err)
	}

	// Delete canary service
	if err := d.manager.DeleteService(canaryService.ID); err != nil {
		log.Logger.Error().Err(err).Str("canary_service_id", canaryService.ID).Msg("Failed to delete canary service")
	}

	// Mark as rolled back
	canaryService.Labels[types.LabelDeploymentState] = string(types.DeploymentStateRolledBack)

	log.Logger.Info().
		Str("stable_service_id", stableService.ID).
		Msg("Canary rollback complete")

	return fmt.Errorf("canary deployment rolled back due to health check failures")
}

// RollbackDeployment rolls back a deployment to the previous version
func (d *Deployer) RollbackDeployment(serviceID string) error {
	service, err := d.manager.GetService(serviceID)
	if err != nil {
		return fmt.Errorf("failed to get service: %w", err)
	}

	log.Logger.Info().
		Str("service", service.Name).
		Str("service_id", service.ID).
		Msg("Starting deployment rollback")

	// Find standby version (previous deployment)
	services, err := d.manager.ListServices()
	if err != nil {
		return fmt.Errorf("failed to list services: %w", err)
	}

	var standbyService *types.Service
	for _, svc := range services {
		if svc.Labels[types.LabelOriginalService] == service.ID &&
			svc.Labels[types.LabelDeploymentState] == string(types.DeploymentStateStandby) {
			standbyService = svc
			break
		}
	}

	if standbyService == nil {
		return fmt.Errorf("no standby version found to rollback to")
	}

	log.Logger.Info().
		Str("standby_service_id", standbyService.ID).
		Str("standby_image", standbyService.Image).
		Msg("Found standby version for rollback")

	// Switch traffic back to standby version
	standbyService.Labels[types.LabelDeploymentState] = string(types.DeploymentStateActive)
	standbyService.Replicas = service.Replicas
	if err := d.manager.UpdateService(standbyService); err != nil {
		return fmt.Errorf("failed to activate standby service: %w", err)
	}

	// Mark current version as rolled back
	service.Labels[types.LabelDeploymentState] = string(types.DeploymentStateRolledBack)
	service.Replicas = 0
	if err := d.manager.UpdateService(service); err != nil {
		log.Logger.Warn().Err(err).Msg("Failed to mark current service as rolled back")
	}

	// Track manual rollback metric
	strategy := standbyService.Labels[types.LabelDeploymentStrategy]
	if strategy == "" {
		strategy = "unknown"
	}
	metrics.RolledBackDeploymentsTotal.WithLabelValues(strategy, "manual").Inc()

	log.Logger.Info().
		Str("service", service.Name).
		Str("rolled_back_to", standbyService.Image).
		Msg("Deployment rollback complete")

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
