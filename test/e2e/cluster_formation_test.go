package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/cuemby/warren/test/framework"
)

// TestClusterFormation tests the formation of a 3-manager + 2-worker cluster
// This replaces test/lima/test-cluster.sh
func TestClusterFormation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping cluster formation test in short mode")
	}

	// Configure 3-manager + 2-worker cluster
	config := &framework.ClusterConfig{
		NumManagers: 3,
		NumWorkers:  2,
		UseLima:     true,
		ManagerVMConfig: &framework.VMConfig{
			CPUs:   2,
			Memory: "2GiB",
			Disk:   "10GiB",
		},
		WorkerVMConfig: &framework.VMConfig{
			CPUs:   2,
			Memory: "2GiB",
			Disk:   "10GiB",
		},
	}

	cluster, err := framework.NewCluster(config)
	if err != nil {
		t.Fatalf("Failed to create cluster: %v", err)
	}
	defer func() { _ = cluster.Cleanup() }()

	// Start the cluster (bootstraps manager-1, joins manager-2 & manager-3, starts workers)
	if err := cluster.Start(); err != nil {
		t.Fatalf("Failed to start cluster: %v", err)
	}
	defer func() { _ = cluster.Stop() }()

	assert := framework.NewAssertions(t)
	waiter := framework.DefaultWaiter()
	ctx := context.Background()

	t.Run("VerifyManagerClusterFormation", func(t *testing.T) {
		// Wait for leader election
		t.Log("Waiting for Raft leader election...")
		if err := waiter.WaitForLeaderElection(ctx, cluster); err != nil {
			t.Fatalf("Leader election failed: %v", err)
		}

		// Get leader
		leader, err := cluster.GetLeader()
		if err != nil {
			t.Fatalf("Failed to get leader: %v", err)
		}
		t.Logf("Leader elected: %s", leader.ID)

		// Verify quorum (3 managers)
		t.Log("Verifying Raft quorum...")
		assert.QuorumSize(3, cluster)
		t.Log("✓ Raft quorum established (3 voters)")

		// Get cluster info from leader
		info, err := leader.Client.GetClusterInfo()
		if err != nil {
			t.Fatalf("Failed to get cluster info: %v", err)
		}

		// Verify all 3 managers are in the cluster
		if len(info.Servers) != 3 {
			t.Errorf("Expected 3 Raft servers, got %d", len(info.Servers))
		}

		// Verify leader is set
		if info.LeaderId == "" {
			t.Error("No leader ID in cluster info")
		}

		t.Logf("Cluster info: Leader=%s, Servers=%d", info.LeaderId, len(info.Servers))
	})

	t.Run("VerifyWorkerRegistration", func(t *testing.T) {
		leader, err := cluster.GetLeader()
		if err != nil {
			t.Fatalf("Failed to get leader: %v", err)
		}

		// Wait for all nodes to register (3 managers + 2 workers = 5 nodes)
		t.Log("Waiting for all nodes to register...")
		if err := waiter.WaitForNodeCount(ctx, leader.Client, 5); err != nil {
			t.Fatalf("Expected 5 nodes to register: %v", err)
		}

		// Verify manager node count
		if err := waiter.WaitForManagerNodes(ctx, leader.Client, 3); err != nil {
			t.Fatalf("Expected 3 manager nodes: %v", err)
		}
		t.Log("✓ All 3 managers registered")

		// Verify worker node count
		if err := waiter.WaitForWorkerNodes(ctx, leader.Client, 2); err != nil {
			t.Fatalf("Expected 2 worker nodes: %v", err)
		}
		t.Log("✓ All 2 workers registered")

		// List all nodes
		nodes, err := leader.Client.ListNodes()
		if err != nil {
			t.Fatalf("Failed to list nodes: %v", err)
		}

		t.Logf("Cluster nodes:")
		for _, node := range nodes {
			t.Logf("  - ID: %s, Role: %s, Status: %s", node.Id, node.Role, node.Status)
		}
	})

	t.Run("DeployTestService", func(t *testing.T) {
		leader, err := cluster.GetLeader()
		if err != nil {
			t.Fatalf("Failed to get leader: %v", err)
		}

		// Create nginx service with 2 replicas
		serviceName := "nginx-test"
		serviceImage := "nginx:alpine"
		replicas := 2

		t.Logf("Creating service '%s' with %d replicas...", serviceName, replicas)
		if err := leader.Client.CreateService(serviceName, serviceImage, replicas); err != nil {
			t.Fatalf("Failed to create service: %v", err)
		}
		defer func() { _ = leader.Client.DeleteService(serviceName) }()

		// Wait for service to have running tasks
		t.Log("Waiting for service tasks to be running...")
		if err := waiter.WaitForReplicas(ctx, leader.Client, serviceName, replicas); err != nil {
			t.Fatalf("Service tasks failed to start: %v", err)
		}
		t.Log("✓ Service running with 2 replicas")

		// Verify service details
		svc, err := leader.Client.GetService(serviceName)
		if err != nil {
			t.Fatalf("Failed to get service: %v", err)
		}

		if svc.Image != serviceImage {
			t.Errorf("Expected image %s, got %s", serviceImage, svc.Image)
		}

		if int(svc.Replicas) != replicas {
			t.Errorf("Expected %d replicas, got %d", replicas, svc.Replicas)
		}

		// List tasks
		tasks, err := leader.Client.ListTasks(svc.Id, "")
		if err != nil {
			t.Fatalf("Failed to list tasks: %v", err)
		}

		t.Logf("Service tasks:")
		for _, task := range tasks {
			t.Logf("  - Task: %s, Node: %s, State: %s", task.Id, task.NodeId, task.ActualState)
		}

		// Verify tasks are distributed (not all on same node)
		nodeMap := make(map[string]int)
		for _, task := range tasks {
			if task.ActualState == "running" {
				nodeMap[task.NodeId]++
			}
		}

		if len(nodeMap) == 1 {
			t.Log("⚠ All tasks scheduled on same node (load balancing may need improvement)")
		} else {
			t.Logf("✓ Tasks distributed across %d nodes", len(nodeMap))
		}
	})

	t.Run("VerifyClusterHealth", func(t *testing.T) {
		leader, err := cluster.GetLeader()
		if err != nil {
			t.Fatalf("Failed to get leader: %v", err)
		}

		// Wait for all nodes to be healthy
		t.Log("Verifying cluster health...")
		if err := waiter.WaitForClusterHealthy(ctx, leader.Client); err != nil {
			t.Fatalf("Cluster not healthy: %v", err)
		}
		t.Log("✓ All cluster nodes are healthy")
	})
}

// TestClusterFormationSingleManager tests basic single-manager cluster
// This is a simpler, faster variant for quick validation
func TestClusterFormationSingleManager(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping single manager test in short mode")
	}

	// Configure 1-manager + 1-worker cluster
	config := &framework.ClusterConfig{
		NumManagers: 1,
		NumWorkers:  1,
		UseLima:     true,
		ManagerVMConfig: &framework.VMConfig{
			CPUs:   2,
			Memory: "2GiB",
			Disk:   "10GiB",
		},
		WorkerVMConfig: &framework.VMConfig{
			CPUs:   2,
			Memory: "2GiB",
			Disk:   "10GiB",
		},
	}

	cluster, err := framework.NewCluster(config)
	if err != nil {
		t.Fatalf("Failed to create cluster: %v", err)
	}
	defer func() { _ = cluster.Cleanup() }()

	if err := cluster.Start(); err != nil {
		t.Fatalf("Failed to start cluster: %v", err)
	}
	defer func() { _ = cluster.Stop() }()

	assert := framework.NewAssertions(t)
	waiter := framework.DefaultWaiter()
	ctx := context.Background()

	t.Run("VerifyBasicCluster", func(t *testing.T) {
		// Get leader (only manager)
		leader, err := cluster.GetLeader()
		if err != nil {
			t.Fatalf("Failed to get leader: %v", err)
		}

		// Verify quorum (single node)
		assert.QuorumSize(1, cluster)

		// Wait for worker to register
		if err := waiter.WaitForNodeCount(ctx, leader.Client, 2); err != nil {
			t.Fatalf("Expected 2 nodes: %v", err)
		}

		if err := waiter.WaitForWorkerNodes(ctx, leader.Client, 1); err != nil {
			t.Fatalf("Expected 1 worker node: %v", err)
		}

		t.Log("✓ Single-manager cluster initialized")
		t.Log("✓ Worker registered")
	})

	t.Run("DeploySimpleService", func(t *testing.T) {
		leader, err := cluster.GetLeader()
		if err != nil {
			t.Fatalf("Failed to get leader: %v", err)
		}

		// Create simple service
		serviceName := "test-simple"
		if err := leader.Client.CreateService(serviceName, "nginx:alpine", 1); err != nil {
			t.Fatalf("Failed to create service: %v", err)
		}
		defer func() { _ = leader.Client.DeleteService(serviceName) }()

		// Wait for service to be running
		if err := waiter.WaitForServiceRunning(ctx, leader.Client, serviceName); err != nil {
			t.Fatalf("Service failed to start: %v", err)
		}

		t.Log("✓ Service deployed and running")
	})
}

// TestClusterFormationManagerOnly tests manager-only cluster (no workers)
// Useful for testing control plane without containers
func TestClusterFormationManagerOnly(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping manager-only test in short mode")
	}

	config := &framework.ClusterConfig{
		NumManagers: 3,
		NumWorkers:  0, // No workers
		UseLima:     true,
		ManagerVMConfig: &framework.VMConfig{
			CPUs:   2,
			Memory: "2GiB",
			Disk:   "10GiB",
		},
	}

	cluster, err := framework.NewCluster(config)
	if err != nil {
		t.Fatalf("Failed to create cluster: %v", err)
	}
	defer func() { _ = cluster.Cleanup() }()

	if err := cluster.Start(); err != nil {
		t.Fatalf("Failed to start cluster: %v", err)
	}
	defer func() { _ = cluster.Stop() }()

	assert := framework.NewAssertions(t)
	waiter := framework.DefaultWaiter()
	ctx := context.Background()

	t.Run("VerifyManagerOnlyCluster", func(t *testing.T) {
		// Wait for leader election
		if err := waiter.WaitForLeaderElection(ctx, cluster); err != nil {
			t.Fatalf("Leader election failed: %v", err)
		}

		// Verify quorum
		assert.QuorumSize(3, cluster)

		// Get leader
		leader, err := cluster.GetLeader()
		if err != nil {
			t.Fatalf("Failed to get leader: %v", err)
		}

		// Verify only manager nodes (3 managers, 0 workers)
		if err := waiter.WaitForNodeCount(ctx, leader.Client, 3); err != nil {
			t.Fatalf("Expected 3 manager nodes: %v", err)
		}

		nodes, err := leader.Client.ListNodes()
		if err != nil {
			t.Fatalf("Failed to list nodes: %v", err)
		}

		workerCount := 0
		for _, node := range nodes {
			if node.Role == "worker" {
				workerCount++
			}
		}

		if workerCount != 0 {
			t.Errorf("Expected 0 workers, found %d", workerCount)
		}

		t.Log("✓ Manager-only cluster verified (3 managers, 0 workers)")
	})

	t.Run("ServiceCreationWithoutWorkers", func(t *testing.T) {
		leader, err := cluster.GetLeader()
		if err != nil {
			t.Fatalf("Failed to get leader: %v", err)
		}

		// Create service (will be pending since no workers)
		serviceName := "test-no-workers"
		if err := leader.Client.CreateService(serviceName, "nginx:alpine", 1); err != nil {
			t.Fatalf("Failed to create service: %v", err)
		}
		defer func() { _ = leader.Client.DeleteService(serviceName) }()

		// Service should be created but tasks will be pending
		time.Sleep(3 * time.Second) // Give scheduler time to run

		svc, err := leader.Client.GetService(serviceName)
		if err != nil {
			t.Fatalf("Failed to get service: %v", err)
		}

		if svc.Name != serviceName {
			t.Errorf("Expected service name %s, got %s", serviceName, svc.Name)
		}

		tasks, err := leader.Client.ListTasks(svc.Id, "")
		if err != nil {
			t.Fatalf("Failed to list tasks: %v", err)
		}

		// Tasks should exist but be in pending state (no workers available)
		runningCount := 0
		for _, task := range tasks {
			if task.ActualState == "running" {
				runningCount++
			}
		}

		if runningCount > 0 {
			t.Logf("⚠ Found %d running tasks (expected 0 since no workers)", runningCount)
		} else {
			t.Log("✓ Service created but tasks pending (no workers available)")
		}
	})
}
