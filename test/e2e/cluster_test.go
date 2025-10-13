package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/cuemby/warren/test/framework"
)

// TestBasicCluster tests basic cluster initialization and operations
func TestBasicCluster(t *testing.T) {
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

	// Start the cluster
	if err := cluster.Start(); err != nil {
		t.Fatalf("Failed to start cluster: %v", err)
	}
	defer func() { _ = cluster.Stop() }()

	// Get test utilities
	assert := framework.NewAssertions(t)
	waiter := framework.DefaultWaiter()
	ctx := context.Background()

	// Get leader manager
	leader, err := cluster.GetLeader()
	if err != nil {
		t.Fatalf("Failed to get leader: %v", err)
	}

	t.Run("VerifyClusterState", func(t *testing.T) {
		// Wait for cluster to be ready
		if err := waiter.WaitForLeaderElection(ctx, cluster); err != nil {
			t.Fatalf("Leader election failed: %v", err)
		}

		// Verify quorum
		assert.QuorumSize(1, cluster)

		// Verify node count (1 manager + 1 worker)
		if err := waiter.WaitForNodeCount(ctx, leader.Client, 2); err != nil {
			t.Fatalf("Expected 2 nodes: %v", err)
		}

		// Verify worker node
		if err := waiter.WaitForWorkerNodes(ctx, leader.Client, 1); err != nil {
			t.Fatalf("Expected 1 worker node: %v", err)
		}
	})

	t.Run("CreateAndRunService", func(t *testing.T) {
		// Create a simple nginx service
		service := &framework.ServiceSpec{
			Name:     "test-nginx",
			Image:    "nginx:alpine",
			Replicas: 1,
		}

		if err := leader.Client.CreateService(service.Name, service.Image, service.Replicas); err != nil {
			t.Fatalf("Failed to create service: %v", err)
		}
		defer func() { _ = leader.Client.DeleteService(service.Name) }()

		// Wait for service to have running tasks
		if err := waiter.WaitForServiceRunning(ctx, leader.Client, service.Name); err != nil {
			t.Fatalf("Service failed to start: %v", err)
		}

		// Verify replicas
		assert.ServiceReplicas(service.Name, service.Replicas, leader.Client)

		// Get service details
		svc, err := leader.Client.GetService(service.Name)
		if err != nil {
			t.Fatalf("Failed to get service: %v", err)
		}

		// Verify service properties
		if svc.Image != service.Image {
			t.Errorf("Expected image %s, got %s", service.Image, svc.Image)
		}
		if int(svc.Replicas) != service.Replicas {
			t.Errorf("Expected %d replicas, got %d", service.Replicas, svc.Replicas)
		}
	})

	t.Run("ServiceScaling", func(t *testing.T) {
		// Create service with 1 replica
		service := &framework.ServiceSpec{
			Name:     "test-scaling",
			Image:    "nginx:alpine",
			Replicas: 1,
		}

		if err := leader.Client.CreateService(service.Name, service.Image, service.Replicas); err != nil {
			t.Fatalf("Failed to create service: %v", err)
		}
		defer func() { _ = leader.Client.DeleteService(service.Name) }()

		// Wait for initial replica
		if err := waiter.WaitForReplicas(ctx, leader.Client, service.Name, 1); err != nil {
			t.Fatalf("Failed to start initial replica: %v", err)
		}

		// Scale to 2 replicas
		svc, err := leader.Client.GetService(service.Name)
		if err != nil {
			t.Fatalf("Failed to get service: %v", err)
		}
		if _, err := leader.Client.UpdateService(svc.Id, 2); err != nil {
			t.Fatalf("Failed to scale service: %v", err)
		}

		// Wait for scaled replicas
		if err := waiter.WaitForReplicas(ctx, leader.Client, service.Name, 2); err != nil {
			t.Fatalf("Failed to scale to 2 replicas: %v", err)
		}

		// Verify final state
		assert.ServiceReplicas(service.Name, 2, leader.Client)
	})

	t.Run("ServiceDeletion", func(t *testing.T) {
		// Create service
		service := &framework.ServiceSpec{
			Name:     "test-deletion",
			Image:    "nginx:alpine",
			Replicas: 1,
		}

		if err := leader.Client.CreateService(service.Name, service.Image, service.Replicas); err != nil {
			t.Fatalf("Failed to create service: %v", err)
		}

		// Wait for service to be running
		if err := waiter.WaitForServiceRunning(ctx, leader.Client, service.Name); err != nil {
			t.Fatalf("Service failed to start: %v", err)
		}

		// Delete service
		if err := leader.Client.DeleteService(service.Name); err != nil {
			t.Fatalf("Failed to delete service: %v", err)
		}

		// Wait for service to be deleted
		if err := waiter.WaitForServiceDeleted(ctx, leader.Client, service.Name); err != nil {
			t.Fatalf("Service not deleted: %v", err)
		}
	})
}

// TestMultiManagerCluster tests a 3-manager HA cluster
func TestMultiManagerCluster(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping multi-manager test in short mode")
	}

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

	if err := cluster.Start(); err != nil {
		t.Fatalf("Failed to start cluster: %v", err)
	}
	defer func() { _ = cluster.Stop() }()

	assert := framework.NewAssertions(t)
	waiter := framework.DefaultWaiter()
	ctx := context.Background()

	t.Run("VerifyHACluster", func(t *testing.T) {
		// Wait for leader election
		if err := waiter.WaitForLeaderElection(ctx, cluster); err != nil {
			t.Fatalf("Leader election failed: %v", err)
		}

		// Verify quorum (3 managers)
		assert.QuorumSize(3, cluster)

		// Verify total nodes (3 managers + 2 workers)
		leader, err := cluster.GetLeader()
		if err != nil {
			t.Fatalf("Failed to get leader: %v", err)
		}

		if err := waiter.WaitForNodeCount(ctx, leader.Client, 5); err != nil {
			t.Fatalf("Expected 5 nodes: %v", err)
		}

		if err := waiter.WaitForManagerNodes(ctx, leader.Client, 3); err != nil {
			t.Fatalf("Expected 3 manager nodes: %v", err)
		}

		if err := waiter.WaitForWorkerNodes(ctx, leader.Client, 2); err != nil {
			t.Fatalf("Expected 2 worker nodes: %v", err)
		}
	})

	t.Run("LeaderFailover", func(t *testing.T) {
		// Get current leader
		originalLeader, err := cluster.GetLeader()
		if err != nil {
			t.Fatalf("Failed to get leader: %v", err)
		}

		t.Logf("Original leader: %s", originalLeader.ID)

		// Kill the leader
		if err := cluster.KillManager(originalLeader.ID); err != nil {
			t.Fatalf("Failed to kill leader: %v", err)
		}

		// Wait for new leader election
		time.Sleep(5 * time.Second) // Give time for failure detection

		if err := waiter.WaitForLeaderElection(ctx, cluster); err != nil {
			t.Fatalf("New leader not elected: %v", err)
		}

		// Verify new leader is different
		newLeader, err := cluster.GetLeader()
		if err != nil {
			t.Fatalf("Failed to get new leader: %v", err)
		}

		if newLeader.ID == originalLeader.ID {
			t.Errorf("Leader did not change after failover")
		}

		t.Logf("New leader: %s", newLeader.ID)

		// Verify cluster still functional
		service := &framework.ServiceSpec{
			Name:     "test-after-failover",
			Image:    "nginx:alpine",
			Replicas: 1,
		}

		if err := newLeader.Client.CreateService(service.Name, service.Image, service.Replicas); err != nil {
			t.Fatalf("Failed to create service after failover: %v", err)
		}
		defer func() { _ = newLeader.Client.DeleteService(service.Name) }()

		if err := waiter.WaitForServiceRunning(ctx, newLeader.Client, service.Name); err != nil {
			t.Fatalf("Service failed to start after failover: %v", err)
		}
	})
}
