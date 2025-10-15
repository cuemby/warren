package framework

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/cuemby/warren/pkg/client"
)

// DefaultClusterConfig returns a default cluster configuration
func DefaultClusterConfig() *ClusterConfig {
	warrenBinary := os.Getenv("WARREN_BINARY")
	if warrenBinary == "" {
		warrenBinary = "bin/warren"
	}

	dataDir := os.Getenv("WARREN_TEST_DATA_DIR")
	if dataDir == "" {
		dataDir = "/tmp/warren-test"
	}

	return &ClusterConfig{
		NumManagers:   3,
		NumWorkers:    2,
		Runtime:       RuntimeLima,
		DataDir:       dataDir,
		WarrenBinary:  warrenBinary,
		KeepOnFailure: false,
		LogLevel:      "info",
	}
}

// NewCluster creates a new test cluster with the given configuration
func NewCluster(config *ClusterConfig) (*Cluster, error) {
	if config == nil {
		config = DefaultClusterConfig()
	}

	// Validate configuration
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid cluster config: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	cluster := &Cluster{
		Config:   config,
		Managers: make([]*Manager, 0, config.NumManagers),
		Workers:  make([]*Worker, 0, config.NumWorkers),
		ctx:      ctx,
		cancel:   cancel,
	}

	return cluster, nil
}

// Start starts the entire cluster (managers and workers)
func (c *Cluster) Start() error {
	// Start managers first
	for i := 0; i < c.Config.NumManagers; i++ {
		if err := c.startManager(i); err != nil {
			return fmt.Errorf("failed to start manager-%d: %w", i+1, err)
		}
	}

	// Wait for Raft quorum
	if err := c.WaitForQuorum(); err != nil {
		return fmt.Errorf("failed to establish quorum: %w", err)
	}

	// Start workers
	for i := 0; i < c.Config.NumWorkers; i++ {
		if err := c.startWorker(i); err != nil {
			return fmt.Errorf("failed to start worker-%d: %w", i+1, err)
		}
	}

	return nil
}

// Stop stops the entire cluster gracefully
func (c *Cluster) Stop() error {
	// Stop workers first
	for _, worker := range c.Workers {
		if err := c.stopWorker(worker); err != nil {
			return fmt.Errorf("failed to stop worker %s: %w", worker.ID, err)
		}
	}

	// Stop managers
	for _, manager := range c.Managers {
		if err := c.stopManager(manager); err != nil {
			return fmt.Errorf("failed to stop manager %s: %w", manager.ID, err)
		}
	}

	return nil
}

// Cleanup cleans up all cluster resources
func (c *Cluster) Cleanup() error {
	// Stop cluster if running
	if err := c.Stop(); err != nil {
		// Log but don't fail cleanup on stop errors
		fmt.Printf("Warning: error during stop: %v\n", err)
	}

	// Cancel context
	if c.cancel != nil {
		c.cancel()
	}

	// Clean up VMs if using Lima/Docker
	if c.Config.Runtime == RuntimeLima || c.Config.Runtime == RuntimeDocker {
		for _, manager := range c.Managers {
			if manager.VM != nil {
				if err := manager.VM.Kill(c.ctx); err != nil {
					fmt.Printf("Warning: failed to kill manager VM %s: %v\n", manager.ID, err)
				}
			}
		}
		for _, worker := range c.Workers {
			if worker.VM != nil {
				if err := worker.VM.Kill(c.ctx); err != nil {
					fmt.Printf("Warning: failed to kill worker VM %s: %v\n", worker.ID, err)
				}
			}
		}
	}

	// Clean up data directories
	if !c.Config.KeepOnFailure {
		if err := os.RemoveAll(c.Config.DataDir); err != nil {
			return fmt.Errorf("failed to remove data dir: %w", err)
		}
	}

	return nil
}

// GetLeader returns the current Raft leader manager
func (c *Cluster) GetLeader() (*Manager, error) {
	for _, manager := range c.Managers {
		if manager.Client == nil {
			continue
		}

		info, err := manager.Client.GetClusterInfo()
		if err != nil {
			continue
		}

		leaderID := info.LeaderId
		for _, m := range c.Managers {
			if m.ID == leaderID {
				m.IsLeader = true
				return m, nil
			}
		}
	}

	return nil, fmt.Errorf("no leader found in cluster")
}

// WaitForQuorum waits for Raft quorum to be established
func (c *Cluster) WaitForQuorum() error {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for quorum: %w", ctx.Err())
		case <-ticker.C:
			if c.hasQuorum() {
				return nil
			}
		}
	}
}

// WaitForWorkers waits for all workers to connect to the cluster
func (c *Cluster) WaitForWorkers() error {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for workers: %w", ctx.Err())
		case <-ticker.C:
			leader, err := c.GetLeader()
			if err != nil {
				continue
			}

			nodes, err := leader.Client.ListNodes()
			if err != nil {
				continue
			}

			workerCount := 0
			for _, node := range nodes {
				if node.Role == "worker" {
					workerCount++
				}
			}

			if workerCount >= c.Config.NumWorkers {
				return nil
			}
		}
	}
}

// KillManager kills a specific manager (simulates crash)
func (c *Cluster) KillManager(id string) error {
	for _, manager := range c.Managers {
		if manager.ID == id {
			if manager.Process != nil {
				return manager.Process.Kill()
			}
			if manager.VM != nil {
				// Kill the Warren process in the VM
				_, err := manager.VM.Exec(c.ctx, "pkill", "-9", "warren")
				return err
			}
			return fmt.Errorf("manager %s has no process or VM", id)
		}
	}
	return fmt.Errorf("manager %s not found", id)
}

// RestartManager restarts a specific manager
func (c *Cluster) RestartManager(id string) error {
	// Find the manager
	var managerIndex int
	found := false
	for i, manager := range c.Managers {
		if manager.ID == id {
			managerIndex = i
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("manager %s not found", id)
	}

	// Stop the manager
	if err := c.stopManager(c.Managers[managerIndex]); err != nil {
		return fmt.Errorf("failed to stop manager: %w", err)
	}

	// Wait a moment
	time.Sleep(2 * time.Second)

	// Restart the manager
	if err := c.startManager(managerIndex); err != nil {
		return fmt.Errorf("failed to restart manager: %w", err)
	}

	return nil
}

// Private helper methods

func (c *Cluster) startManager(index int) error {
	managerID := fmt.Sprintf("manager-%d", index+1)
	dataDir := filepath.Join(c.Config.DataDir, managerID)

	// Create data directory
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data dir: %w", err)
	}

	manager := &Manager{
		ID:       managerID,
		DataDir:  dataDir,
		APIAddr:  fmt.Sprintf("localhost:%d", 8080+index),
		RaftAddr: fmt.Sprintf("localhost:%d", 7946+index),
	}

	// Start based on runtime type
	switch c.Config.Runtime {
	case RuntimeLima:
		return c.startManagerLima(manager, index)
	case RuntimeDocker:
		return c.startManagerDocker(manager, index)
	case RuntimeLocal:
		return c.startManagerLocal(manager, index)
	default:
		return fmt.Errorf("unsupported runtime: %s", c.Config.Runtime)
	}
}

func (c *Cluster) startManagerLocal(manager *Manager, index int) error {
	// Start Warren process locally
	process := NewProcess(c.Config.WarrenBinary)

	args := []string{
		"cluster", "init",
		"--node-id=" + manager.ID,
		"--bind-addr=" + manager.RaftAddr,
		"--api-addr=" + manager.APIAddr,
		"--data-dir=" + manager.DataDir,
		"--log-level=" + c.Config.LogLevel,
	}

	// For managers after the first, join the cluster
	if index > 0 {
		firstManager := c.Managers[0]
		token, err := c.getJoinToken(firstManager, "manager")
		if err != nil {
			return fmt.Errorf("failed to get join token: %w", err)
		}

		args = []string{
			"cluster", "join",
			"--token=" + token,
			"--node-id=" + manager.ID,
			"--bind-addr=" + manager.RaftAddr,
			"--api-addr=" + manager.APIAddr,
			"--data-dir=" + manager.DataDir,
			"--log-level=" + c.Config.LogLevel,
		}
	}

	process.Args = args

	if err := process.Start(); err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}

	manager.Process = process

	// Wait for API to be ready
	if err := c.waitForAPI(manager.APIAddr, 30*time.Second); err != nil {
		return fmt.Errorf("API not ready: %w", err)
	}

	// Create client
	warrenClient, err := client.NewClient(manager.APIAddr)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	manager.Client = NewClient(warrenClient)
	c.Managers = append(c.Managers, manager)

	return nil
}

func (c *Cluster) startManagerLima(manager *Manager, index int) error {
	// TODO: Implement Lima VM management
	return fmt.Errorf("Lima runtime not yet implemented")
}

func (c *Cluster) startManagerDocker(manager *Manager, index int) error {
	// TODO: Implement Docker container management
	return fmt.Errorf("Docker runtime not yet implemented")
}

func (c *Cluster) startWorker(index int) error {
	workerID := fmt.Sprintf("worker-%d", index+1)
	dataDir := filepath.Join(c.Config.DataDir, workerID)

	// Create data directory
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data dir: %w", err)
	}

	worker := &Worker{
		ID:      workerID,
		DataDir: dataDir,
	}

	// Get join token from leader
	leader, err := c.GetLeader()
	if err != nil {
		return fmt.Errorf("failed to get leader: %w", err)
	}

	token, err := c.getJoinToken(leader, "worker")
	if err != nil {
		return fmt.Errorf("failed to get join token: %w", err)
	}

	worker.ManagerAddr = leader.APIAddr

	// Start based on runtime type
	switch c.Config.Runtime {
	case RuntimeLocal:
		return c.startWorkerLocal(worker, token)
	case RuntimeLima:
		return c.startWorkerLima(worker, token)
	case RuntimeDocker:
		return c.startWorkerDocker(worker, token)
	default:
		return fmt.Errorf("unsupported runtime: %s", c.Config.Runtime)
	}
}

func (c *Cluster) startWorkerLocal(worker *Worker, token string) error {
	process := NewProcess(c.Config.WarrenBinary)

	process.Args = []string{
		"cluster", "join",
		"--token=" + token,
		"--node-id=" + worker.ID,
		"--data-dir=" + worker.DataDir,
		"--log-level=" + c.Config.LogLevel,
	}

	if err := process.Start(); err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}

	worker.Process = process
	c.Workers = append(c.Workers, worker)

	return nil
}

func (c *Cluster) startWorkerLima(worker *Worker, token string) error {
	// TODO: Implement Lima worker
	return fmt.Errorf("Lima runtime not yet implemented")
}

func (c *Cluster) startWorkerDocker(worker *Worker, token string) error {
	// TODO: Implement Docker worker
	return fmt.Errorf("Docker runtime not yet implemented")
}

func (c *Cluster) stopManager(manager *Manager) error {
	if manager.Client != nil {
		manager.Client.Close()
	}

	if manager.Process != nil {
		return manager.Process.Stop()
	}

	if manager.VM != nil {
		return manager.VM.Stop(c.ctx)
	}

	return nil
}

func (c *Cluster) stopWorker(worker *Worker) error {
	if worker.Process != nil {
		return worker.Process.Stop()
	}

	if worker.VM != nil {
		return worker.VM.Stop(c.ctx)
	}

	return nil
}

func (c *Cluster) hasQuorum() bool {
	leader, err := c.GetLeader()
	if err != nil {
		return false
	}

	info, err := leader.Client.GetClusterInfo()
	if err != nil {
		return false
	}

	// Check if we have a leader and quorum
	return info.LeaderId != "" && len(info.Servers) >= (c.Config.NumManagers/2+1)
}

func (c *Cluster) waitForAPI(addr string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(c.ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for API at %s: %w", addr, ctx.Err())
		case <-ticker.C:
			// Try to connect
			client, err := client.NewClient(addr)
			if err != nil {
				continue
			}
			client.Close()
			return nil
		}
	}
}

func (c *Cluster) getJoinToken(manager *Manager, role string) (string, error) {
	if manager.Client == nil {
		return "", fmt.Errorf("manager client is nil")
	}

	// TODO: Implement token generation via client
	// For now, return a placeholder
	return "test-token-" + role, nil
}

func validateConfig(config *ClusterConfig) error {
	if config.NumManagers < 1 {
		return fmt.Errorf("NumManagers must be >= 1, got %d", config.NumManagers)
	}

	if config.NumManagers%2 == 0 {
		return fmt.Errorf("NumManagers should be odd for Raft quorum, got %d", config.NumManagers)
	}

	if config.WarrenBinary == "" {
		return fmt.Errorf("WarrenBinary cannot be empty")
	}

	if config.DataDir == "" {
		return fmt.Errorf("DataDir cannot be empty")
	}

	return nil
}
