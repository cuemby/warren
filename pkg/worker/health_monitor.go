package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/cuemby/warren/api/proto"
	"github.com/cuemby/warren/pkg/health"
	"github.com/cuemby/warren/pkg/types"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// HealthMonitor manages health checks for containers
type HealthMonitor struct {
	worker    *Worker
	monitors  map[string]*containerHealthMonitor
	cancelFns map[string]context.CancelFunc
	stopCh    chan struct{}
}

// containerHealthMonitor tracks health check state for a single task
type containerHealthMonitor struct {
	container *types.Container
	checker   health.Checker
	status    *health.Status
	config    health.Config
}

// NewHealthMonitor creates a new health monitor
func NewHealthMonitor(w *Worker) *HealthMonitor {
	return &HealthMonitor{
		worker:    w,
		monitors:  make(map[string]*containerHealthMonitor),
		cancelFns: make(map[string]context.CancelFunc),
		stopCh:    make(chan struct{}),
	}
}

// Start starts the health monitor
func (hm *HealthMonitor) Start() {
	go hm.monitorLoop()
}

// Stop stops the health monitor
func (hm *HealthMonitor) Stop() {
	close(hm.stopCh)
	// Cancel all running health checks
	for _, cancel := range hm.cancelFns {
		cancel()
	}
}

// monitorLoop monitors tasks and starts/stops health checks as needed
func (hm *HealthMonitor) monitorLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			hm.syncHealthChecks()
		case <-hm.stopCh:
			return
		}
	}
}

// syncHealthChecks syncs health checks with current tasks
func (hm *HealthMonitor) syncHealthChecks() {
	hm.worker.containersMu.RLock()
	currentTasks := make(map[string]*types.Container)
	for id, task := range hm.worker.containers {
		currentTasks[id] = task
	}
	hm.worker.containersMu.RUnlock()

	// Stop health checks for tasks that no longer exist
	for taskID, cancel := range hm.cancelFns {
		if _, exists := currentTasks[taskID]; !exists {
			cancel()
			delete(hm.cancelFns, taskID)
			delete(hm.monitors, taskID)
		}
	}

	// Start health checks for new tasks that have health checks configured
	for taskID, task := range currentTasks {
		if _, exists := hm.monitors[taskID]; exists {
			continue // Already monitoring
		}

		if task.HealthCheck == nil {
			continue // No health check configured
		}

		if task.ActualState != types.ContainerStateRunning {
			continue // Only monitor running tasks
		}

		// Start monitoring this task
		if err := hm.startHealthCheck(task); err != nil {
			fmt.Printf("Failed to start health check for task %s: %v\n", taskID, err)
		}
	}
}

// startHealthCheck starts a health check goroutine for a task
func (hm *HealthMonitor) startHealthCheck(task *types.Container) error {
	// Create health checker based on type
	checker, err := hm.createChecker(task)
	if err != nil {
		return fmt.Errorf("failed to create health checker: %w", err)
	}

	// Create health config
	config := health.Config{
		Interval:    task.HealthCheck.Interval,
		Timeout:     task.HealthCheck.Timeout,
		Retries:     task.HealthCheck.Retries,
		StartPeriod: 0, // TODO: Get from task.HealthCheck if we add StartPeriod field
	}

	// Create monitor
	monitor := &containerHealthMonitor{
		container: task,
		checker:   checker,
		status: &health.Status{
			StartedAt: time.Now(),
			Healthy:   true, // Assume healthy initially
		},
		config: config,
	}

	hm.monitors[task.ID] = monitor

	// Start health check loop
	ctx, cancel := context.WithCancel(context.Background())
	hm.cancelFns[task.ID] = cancel

	go hm.healthCheckLoop(ctx, monitor)

	return nil
}

// healthCheckLoop runs health checks for a task
func (hm *HealthMonitor) healthCheckLoop(ctx context.Context, monitor *containerHealthMonitor) {
	ticker := time.NewTicker(monitor.config.Interval)
	defer ticker.Stop()

	// Run initial check immediately
	hm.runHealthCheck(ctx, monitor)

	for {
		select {
		case <-ticker.C:
			hm.runHealthCheck(ctx, monitor)
		case <-ctx.Done():
			return
		case <-hm.stopCh:
			return
		}
	}
}

// runHealthCheck performs a single health check and reports the result
func (hm *HealthMonitor) runHealthCheck(ctx context.Context, monitor *containerHealthMonitor) {
	// Create context with timeout
	checkCtx, cancel := context.WithTimeout(ctx, monitor.config.Timeout)
	defer cancel()

	// Perform health check
	result := monitor.checker.Check(checkCtx)

	// Update status
	monitor.status.Update(result, monitor.config)

	// Report to manager
	if err := hm.reportHealth(monitor); err != nil {
		fmt.Printf("Failed to report health for container %s: %v\n", monitor.container.ID, err)
	}
}

// reportHealth reports health status to the manager
func (hm *HealthMonitor) reportHealth(monitor *containerHealthMonitor) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := hm.worker.client.ReportContainerHealth(ctx, &proto.ReportContainerHealthRequest{
		ContainerId:          monitor.container.ID,
		Healthy:              monitor.status.Healthy,
		Message:              monitor.status.LastResult.Message,
		CheckedAt:            timestamppb.New(monitor.status.LastCheck),
		ConsecutiveFailures:  int32(monitor.status.ConsecutiveFailures),
		ConsecutiveSuccesses: int32(monitor.status.ConsecutiveSuccesses),
	})

	return err
}

// createChecker creates the appropriate health checker for a task
func (hm *HealthMonitor) createChecker(task *types.Container) (health.Checker, error) {
	switch task.HealthCheck.Type {
	case types.HealthCheckHTTP:
		// Parse endpoint to get URL
		// For now, construct URL from container IP and endpoint
		url := fmt.Sprintf("http://localhost%s", task.HealthCheck.Endpoint)
		return health.NewHTTPChecker(url), nil

	case types.HealthCheckTCP:
		// Parse endpoint to get address
		address := fmt.Sprintf("localhost%s", task.HealthCheck.Endpoint)
		return health.NewTCPChecker(address), nil

	case types.HealthCheckExec:
		// Create exec checker with command and container ID
		return health.NewExecChecker(task.HealthCheck.Command).WithContainer(task.ContainerID), nil

	default:
		return nil, fmt.Errorf("unsupported health check type: %s", task.HealthCheck.Type)
	}
}
