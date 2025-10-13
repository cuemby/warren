package scheduler

import (
	"testing"
	"time"

	"github.com/cuemby/warren/pkg/manager"
	"github.com/cuemby/warren/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestGlobalServiceScheduling(t *testing.T) {
	// Create manager with temp data dir
	mgr, err := manager.NewManager(&manager.Config{
		NodeID:   "test-manager",
		BindAddr: "127.0.0.1:0",
		DataDir:  t.TempDir(),
	})
	assert.NoError(t, err)
	defer func() { _ = mgr.Shutdown() }()

	// Bootstrap cluster
	err = mgr.Bootstrap()
	assert.NoError(t, err)

	// Wait for leadership election (up to 5 seconds)
	for i := 0; i < 50; i++ {
		if mgr.IsLeader() {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !mgr.IsLeader() {
		t.Fatal("Manager failed to become leader")
	}

	// Create test nodes
	worker1 := &types.Node{
		ID:       "worker-1",
		Role:     types.NodeRoleWorker,
		Address:  "10.0.0.1",
		Hostname: "worker1",
		Status:   types.NodeStatusReady,
		Resources: &types.NodeResources{
			CPUCores:    4,
			MemoryBytes: 8 * 1024 * 1024 * 1024,
		},
		LastHeartbeat: time.Now(),
		CreatedAt:     time.Now(),
	}

	worker2 := &types.Node{
		ID:       "worker-2",
		Role:     types.NodeRoleWorker,
		Address:  "10.0.0.2",
		Hostname: "worker2",
		Status:   types.NodeStatusReady,
		Resources: &types.NodeResources{
			CPUCores:    4,
			MemoryBytes: 8 * 1024 * 1024 * 1024,
		},
		LastHeartbeat: time.Now(),
		CreatedAt:     time.Now(),
	}

	err = mgr.CreateNode(worker1)
	assert.NoError(t, err)
	err = mgr.CreateNode(worker2)
	assert.NoError(t, err)

	// Create global service
	service := &types.Service{
		ID:       "test-global-service",
		Name:     "monitoring",
		Image:    "prometheus/node-exporter:latest",
		Mode:     types.ServiceModeGlobal,
		Replicas: 0, // Not used for global services
		CreatedAt: time.Now(),
	}

	err = mgr.CreateService(service)
	assert.NoError(t, err)

	// Create scheduler
	sched := NewScheduler(mgr)

	// Run one scheduling cycle
	err = sched.schedule()
	assert.NoError(t, err)

	// Verify one task per node
	tasks, err := mgr.ListTasksByService(service.ID)
	assert.NoError(t, err)
	assert.Len(t, tasks, 2, "Should have exactly 2 tasks (one per worker)")

	// Verify each node has a task
	tasksByNode := make(map[string]bool)
	for _, task := range tasks {
		tasksByNode[task.NodeID] = true
		assert.Equal(t, service.ID, task.ServiceID)
		assert.Equal(t, types.TaskStatePending, task.ActualState)
		assert.Equal(t, types.TaskStateRunning, task.DesiredState)
	}

	assert.True(t, tasksByNode["worker-1"], "Worker 1 should have a task")
	assert.True(t, tasksByNode["worker-2"], "Worker 2 should have a task")

	// Simulate adding a new worker
	worker3 := &types.Node{
		ID:       "worker-3",
		Role:     types.NodeRoleWorker,
		Address:  "10.0.0.3",
		Hostname: "worker3",
		Status:   types.NodeStatusReady,
		Resources: &types.NodeResources{
			CPUCores:    4,
			MemoryBytes: 8 * 1024 * 1024 * 1024,
		},
		LastHeartbeat: time.Now(),
		CreatedAt:     time.Now(),
	}

	err = mgr.CreateNode(worker3)
	assert.NoError(t, err)

	// Run scheduling again
	err = sched.schedule()
	assert.NoError(t, err)

	// Verify auto-scaling to 3 tasks
	tasks, err = mgr.ListTasksByService(service.ID)
	assert.NoError(t, err)
	assert.Len(t, tasks, 3, "Should have auto-scaled to 3 tasks")

	tasksByNode = make(map[string]bool)
	for _, task := range tasks {
		tasksByNode[task.NodeID] = true
	}

	assert.True(t, tasksByNode["worker-1"])
	assert.True(t, tasksByNode["worker-2"])
	assert.True(t, tasksByNode["worker-3"], "New worker should have a task")

	// Simulate removing a worker (mark as down)
	worker2.Status = types.NodeStatusDown
	err = mgr.UpdateNode(worker2)
	assert.NoError(t, err)

	// Run scheduling again
	err = sched.schedule()
	assert.NoError(t, err)

	// Verify task for removed node is marked for shutdown
	tasks, err = mgr.ListTasksByService(service.ID)
	assert.NoError(t, err)

	for _, task := range tasks {
		if task.NodeID == "worker-2" {
			// Since worker-2 is down but still exists, task should remain
			// Only non-existent nodes trigger shutdown
			continue
		}
		assert.Equal(t, types.TaskStateRunning, task.DesiredState)
	}
}

func TestReplicatedServiceScheduling(t *testing.T) {
	// Create manager with temp data dir
	mgr, err := manager.NewManager(&manager.Config{
		NodeID:   "test-manager",
		BindAddr: "127.0.0.1:0",
		DataDir:  t.TempDir(),
	})
	assert.NoError(t, err)
	defer func() { _ = mgr.Shutdown() }()

	// Bootstrap cluster
	err = mgr.Bootstrap()
	assert.NoError(t, err)

	// Wait for leadership election (up to 5 seconds)
	for i := 0; i < 50; i++ {
		if mgr.IsLeader() {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !mgr.IsLeader() {
		t.Fatal("Manager failed to become leader")
	}

	// Create test nodes
	worker1 := &types.Node{
		ID:       "worker-1",
		Role:     types.NodeRoleWorker,
		Address:  "10.0.0.1",
		Hostname: "worker1",
		Status:   types.NodeStatusReady,
		Resources: &types.NodeResources{
			CPUCores:    4,
			MemoryBytes: 8 * 1024 * 1024 * 1024,
		},
		LastHeartbeat: time.Now(),
		CreatedAt:     time.Now(),
	}

	worker2 := &types.Node{
		ID:       "worker-2",
		Role:     types.NodeRoleWorker,
		Address:  "10.0.0.2",
		Hostname: "worker2",
		Status:   types.NodeStatusReady,
		Resources: &types.NodeResources{
			CPUCores:    4,
			MemoryBytes: 8 * 1024 * 1024 * 1024,
		},
		LastHeartbeat: time.Now(),
		CreatedAt:     time.Now(),
	}

	err = mgr.CreateNode(worker1)
	assert.NoError(t, err)
	err = mgr.CreateNode(worker2)
	assert.NoError(t, err)

	// Create replicated service with 3 replicas
	service := &types.Service{
		ID:       "test-replicated-service",
		Name:     "web",
		Image:    "nginx:latest",
		Mode:     types.ServiceModeReplicated,
		Replicas: 3,
		CreatedAt: time.Now(),
	}

	err = mgr.CreateService(service)
	assert.NoError(t, err)

	// Create scheduler
	sched := NewScheduler(mgr)

	// Run one scheduling cycle
	err = sched.schedule()
	assert.NoError(t, err)

	// Verify 3 tasks created
	tasks, err := mgr.ListTasksByService(service.ID)
	assert.NoError(t, err)
	assert.Len(t, tasks, 3, "Should have exactly 3 tasks")

	// Scale down to 2 replicas
	service.Replicas = 2
	err = mgr.UpdateService(service)
	assert.NoError(t, err)

	// Run scheduling again
	err = sched.schedule()
	assert.NoError(t, err)

	// Verify one task is marked for shutdown
	tasks, err = mgr.ListTasksByService(service.ID)
	assert.NoError(t, err)

	runningCount := 0
	shutdownCount := 0
	for _, task := range tasks {
		if task.DesiredState == types.TaskStateRunning {
			runningCount++
		} else if task.DesiredState == types.TaskStateShutdown {
			shutdownCount++
		}
	}

	assert.Equal(t, 2, runningCount, "Should have 2 running tasks")
	assert.Equal(t, 1, shutdownCount, "Should have 1 shutdown task")
}
