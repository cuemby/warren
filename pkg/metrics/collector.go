package metrics

import (
	"time"

	"github.com/cuemby/warren/pkg/manager"
	"github.com/cuemby/warren/pkg/types"
)

// Collector collects metrics from the manager
type Collector struct {
	manager *manager.Manager
	stopCh  chan struct{}
}

// NewCollector creates a new metrics collector
func NewCollector(mgr *manager.Manager) *Collector {
	return &Collector{
		manager: mgr,
		stopCh:  make(chan struct{}),
	}
}

// Start begins collecting metrics
func (c *Collector) Start() {
	ticker := time.NewTicker(15 * time.Second)
	go func() {
		// Collect immediately on start
		c.collect()

		for {
			select {
			case <-ticker.C:
				c.collect()
			case <-c.stopCh:
				ticker.Stop()
				return
			}
		}
	}()
}

// Stop stops the collector
func (c *Collector) Stop() {
	close(c.stopCh)
}

func (c *Collector) collect() {
	// Collect node metrics
	c.collectNodeMetrics()

	// Collect service metrics
	c.collectServiceMetrics()

	// Collect task metrics
	c.collectTaskMetrics()

	// Collect secret metrics
	c.collectSecretMetrics()

	// Collect volume metrics
	c.collectVolumeMetrics()

	// Collect Raft metrics
	c.collectRaftMetrics()
}

func (c *Collector) collectNodeMetrics() {
	nodes, err := c.manager.ListNodes()
	if err != nil {
		return
	}

	// Reset counters
	nodeCounts := make(map[string]map[string]int)

	for _, node := range nodes {
		role := string(node.Role)
		status := string(node.Status)

		if nodeCounts[role] == nil {
			nodeCounts[role] = make(map[string]int)
		}
		nodeCounts[role][status]++
	}

	// Update metrics
	for role, statuses := range nodeCounts {
		for status, count := range statuses {
			NodesTotal.WithLabelValues(role, status).Set(float64(count))
		}
	}
}

func (c *Collector) collectServiceMetrics() {
	services, err := c.manager.ListServices()
	if err != nil {
		return
	}

	ServicesTotal.Set(float64(len(services)))
}

func (c *Collector) collectTaskMetrics() {
	services, err := c.manager.ListServices()
	if err != nil {
		return
	}

	taskCounts := make(map[types.TaskState]int)

	for _, service := range services {
		tasks, err := c.manager.ListTasksByService(service.ID)
		if err != nil {
			continue
		}

		for _, task := range tasks {
			taskCounts[task.ActualState]++
		}
	}

	// Update metrics
	for state, count := range taskCounts {
		TasksTotal.WithLabelValues(string(state)).Set(float64(count))
	}
}

func (c *Collector) collectSecretMetrics() {
	secrets, err := c.manager.ListSecrets()
	if err != nil {
		return
	}

	SecretsTotal.Set(float64(len(secrets)))
}

func (c *Collector) collectVolumeMetrics() {
	volumes, err := c.manager.ListVolumes()
	if err != nil {
		return
	}

	VolumesTotal.Set(float64(len(volumes)))
}

func (c *Collector) collectRaftMetrics() {
	// Check if leader
	if c.manager.IsLeader() {
		RaftLeader.Set(1)
	} else {
		RaftLeader.Set(0)
	}

	// Get Raft stats
	stats := c.manager.GetRaftStats()
	if stats != nil {
		if lastIndex, ok := stats["last_log_index"].(uint64); ok {
			RaftLogIndex.Set(float64(lastIndex))
		}
		if appliedIndex, ok := stats["applied_index"].(uint64); ok {
			RaftAppliedIndex.Set(float64(appliedIndex))
		}
		// Peers count is harder to get, would need to expose from manager
		// For now, set to 1 (this node)
		RaftPeers.Set(1)
	}
}
