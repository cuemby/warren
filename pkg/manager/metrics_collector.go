package manager

import (
	"time"

	"github.com/cuemby/warren/pkg/metrics"
	"github.com/cuemby/warren/pkg/types"
)

// MetricsCollector collects metrics from the manager
type MetricsCollector struct {
	manager *Manager
	stopCh  chan struct{}
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(mgr *Manager) *MetricsCollector {
	return &MetricsCollector{
		manager: mgr,
		stopCh:  make(chan struct{}),
	}
}

// Start begins collecting metrics
func (c *MetricsCollector) Start() {
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
func (c *MetricsCollector) Stop() {
	close(c.stopCh)
}

func (c *MetricsCollector) collect() {
	// Collect node metrics
	c.collectNodeMetrics()

	// Collect service metrics
	c.collectServiceMetrics()

	// Collect task metrics
	c.collectContainerMetrics()

	// Collect secret metrics
	c.collectSecretMetrics()

	// Collect volume metrics
	c.collectVolumeMetrics()

	// Collect Raft metrics
	c.collectRaftMetrics()
}

func (c *MetricsCollector) collectNodeMetrics() {
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
			metrics.NodesTotal.WithLabelValues(role, status).Set(float64(count))
		}
	}
}

func (c *MetricsCollector) collectServiceMetrics() {
	services, err := c.manager.ListServices()
	if err != nil {
		return
	}

	metrics.ServicesTotal.Set(float64(len(services)))
}

func (c *MetricsCollector) collectContainerMetrics() {
	services, err := c.manager.ListServices()
	if err != nil {
		return
	}

	containerCounts := make(map[types.ContainerState]int)

	for _, service := range services {
		containers, err := c.manager.ListContainersByService(service.ID)
		if err != nil {
			continue
		}

		for _, container := range containers {
			containerCounts[container.ActualState]++
		}
	}

	// Update metrics
	for state, count := range containerCounts {
		metrics.ContainersTotal.WithLabelValues(string(state)).Set(float64(count))
	}
}

func (c *MetricsCollector) collectSecretMetrics() {
	secrets, err := c.manager.ListSecrets()
	if err != nil {
		return
	}

	metrics.SecretsTotal.Set(float64(len(secrets)))
}

func (c *MetricsCollector) collectVolumeMetrics() {
	volumes, err := c.manager.ListVolumes()
	if err != nil {
		return
	}

	metrics.VolumesTotal.Set(float64(len(volumes)))
}

func (c *MetricsCollector) collectRaftMetrics() {
	// Check if leader
	if c.manager.IsLeader() {
		metrics.RaftLeader.Set(1)
	} else {
		metrics.RaftLeader.Set(0)
	}

	// Get Raft stats
	stats := c.manager.GetRaftStats()
	if stats != nil {
		if lastIndex, ok := stats["last_log_index"].(uint64); ok {
			metrics.RaftLogIndex.Set(float64(lastIndex))
		}
		if appliedIndex, ok := stats["applied_index"].(uint64); ok {
			metrics.RaftAppliedIndex.Set(float64(appliedIndex))
		}
		if peers, ok := stats["peers"].(uint64); ok {
			metrics.RaftPeers.Set(float64(peers))
		}
	}
}
