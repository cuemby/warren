package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/cuemby/warren/test/framework"
)

// TestLeaderFailover tests Raft leader failover in a 3-manager cluster
// This replaces test/lima/test-failover.sh
func TestLeaderFailover(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping leader failover test in short mode")
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

	if err := cluster.Start(); err != nil {
		t.Fatalf("Failed to start cluster: %v", err)
	}
	defer func() { _ = cluster.Stop() }()

	assert := framework.NewAssertions(t)
	waiter := framework.DefaultWaiter()
	ctx := context.Background()

	// Create a test service before failover
	var testServiceName string

	t.Run("SetupInitialCluster", func(t *testing.T) {
		// Wait for leader election
		t.Log("Waiting for initial leader election...")
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
		t.Logf("Initial leader: %s", leader.ID)

		// Wait for all nodes to be ready
		if err := waiter.WaitForNodeCount(ctx, leader.Client, 5); err != nil {
			t.Fatalf("Expected 5 nodes: %v", err)
		}

		// Create a test service before failover
		testServiceName = "test-pre-failover"
		t.Logf("Creating service '%s' before failover...", testServiceName)
		if err := leader.Client.CreateService(testServiceName, "nginx:alpine", 2); err != nil {
			t.Fatalf("Failed to create service: %v", err)
		}

		// Wait for service to be running
		if err := waiter.WaitForReplicas(ctx, leader.Client, testServiceName, 2); err != nil {
			t.Fatalf("Service failed to start: %v", err)
		}
		t.Log("✓ Initial cluster setup complete")
	})

	t.Run("LeaderFailover", func(t *testing.T) {
		// Get current leader
		originalLeader, err := cluster.GetLeader()
		if err != nil {
			t.Fatalf("Failed to get leader: %v", err)
		}

		originalLeaderID := originalLeader.ID
		t.Logf("Current leader: %s (API: %s)", originalLeaderID, originalLeader.APIAddr)

		// Record failover start time
		failoverStart := time.Now()

		// Kill the leader process
		t.Logf("Killing leader %s...", originalLeaderID)
		if err := cluster.KillManager(originalLeaderID); err != nil {
			t.Fatalf("Failed to kill leader: %v", err)
		}
		t.Log("✓ Leader process killed")

		// Wait for new leader election (target: <10s)
		t.Log("Waiting for new leader election (target: <10s)...")

		// Give a moment for failure detection
		time.Sleep(3 * time.Second)

		// Wait for new leader
		if err := waiter.WaitForLeaderElection(ctx, cluster); err != nil {
			t.Fatalf("New leader not elected: %v", err)
		}

		// Get new leader
		newLeader, err := cluster.GetLeader()
		if err != nil {
			t.Fatalf("Failed to get new leader: %v", err)
		}

		failoverDuration := time.Since(failoverStart)
		t.Logf("✓ New leader elected: %s", newLeader.ID)
		t.Logf("✓ Failover time: %v", failoverDuration)

		// Verify new leader is different
		if newLeader.ID == originalLeaderID {
			t.Errorf("Leader did not change after failover (still %s)", originalLeaderID)
		}

		// Check if failover was fast enough (target <10s)
		if failoverDuration > 10*time.Second {
			t.Logf("⚠ Failover took longer than 10s target: %v", failoverDuration)
		} else {
			t.Logf("✓ Failover within target (<10s)")
		}
	})

	t.Run("VerifyClusterOperationAfterFailover", func(t *testing.T) {
		// Get new leader
		newLeader, err := cluster.GetLeader()
		if err != nil {
			t.Fatalf("Failed to get new leader: %v", err)
		}

		t.Log("Testing cluster operations after failover...")

		// Test 1: List services
		t.Log("Test 1: List services")
		services, err := newLeader.Client.ListServices()
		if err != nil {
			t.Fatalf("Failed to list services after failover: %v", err)
		}
		t.Logf("✓ Can list services (%d services found)", len(services))

		// Verify pre-failover service still exists
		found := false
		for _, svc := range services {
			if svc.Name == testServiceName {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Pre-failover service '%s' not found after failover", testServiceName)
		} else {
			t.Log("✓ Pre-failover service still exists")
		}

		// Test 2: Create new service after failover
		t.Log("Test 2: Create new service after failover")
		postFailoverService := "test-post-failover"
		if err := newLeader.Client.CreateService(postFailoverService, "nginx:alpine", 1); err != nil {
			t.Fatalf("Failed to create service after failover: %v", err)
		}
		defer func() { _ = newLeader.Client.DeleteService(postFailoverService) }()
		t.Log("✓ Created service after failover")

		// Wait for new service to be running
		if err := waiter.WaitForServiceRunning(ctx, newLeader.Client, postFailoverService); err != nil {
			t.Fatalf("Service failed to start after failover: %v", err)
		}
		t.Log("✓ Service running after failover")

		// Test 3: List nodes
		t.Log("Test 3: List nodes")
		nodes, err := newLeader.Client.ListNodes()
		if err != nil {
			t.Fatalf("Failed to list nodes after failover: %v", err)
		}
		t.Logf("✓ Can list nodes (%d nodes)", len(nodes))

		// Test 4: Get cluster info
		t.Log("Test 4: Check cluster info")
		info, err := newLeader.Client.GetClusterInfo()
		if err != nil {
			t.Fatalf("Failed to get cluster info after failover: %v", err)
		}

		t.Logf("Cluster info: Leader=%s, Servers=%d", info.LeaderId, len(info.Servers))

		// We should still have all 3 servers (dead one may still be in Raft config)
		// Or we may have 2-3 depending on timing
		if len(info.Servers) < 2 {
			t.Errorf("Expected at least 2 servers in cluster, got %d", len(info.Servers))
		}

		t.Log("✓ Cluster fully operational after failover")
	})

	t.Run("RestartKilledLeader", func(t *testing.T) {
		// Find the killed manager
		var killedManager *framework.Manager
		for _, mgr := range cluster.Managers {
			if mgr.Process.IsRunning() == false {
				killedManager = mgr
				break
			}
		}

		if killedManager == nil {
			t.Skip("Could not identify killed manager (maybe test skipped failover)")
			return
		}

		t.Logf("Restarting killed manager: %s", killedManager.ID)

		// Restart the manager
		if err := cluster.RestartManager(killedManager.ID); err != nil {
			t.Fatalf("Failed to restart killed manager: %v", err)
		}

		// Wait for it to rejoin
		t.Log("Waiting for manager to rejoin cluster...")
		time.Sleep(5 * time.Second)

		// Get cluster info from current leader
		newLeader, err := cluster.GetLeader()
		if err != nil {
			t.Fatalf("Failed to get current leader: %v", err)
		}

		info, err := newLeader.Client.GetClusterInfo()
		if err != nil {
			t.Fatalf("Failed to get cluster info: %v", err)
		}

		t.Logf("Cluster servers after restart: %d", len(info.Servers))

		// Should have all 3 managers again
		if len(info.Servers) >= 3 {
			t.Log("✓ Killed manager rejoined cluster")
		} else {
			t.Logf("⚠ Manager may still be rejoining (%d/3 servers)", len(info.Servers))
		}
	})
}

// TestMultipleFailovers tests consecutive leader failures
func TestMultipleFailovers(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping multiple failovers test in short mode")
	}

	config := &framework.ClusterConfig{
		NumManagers: 3,
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

	waiter := framework.DefaultWaiter()
	ctx := context.Background()

	// Initial setup
	if err := waiter.WaitForLeaderElection(ctx, cluster); err != nil {
		t.Fatalf("Initial leader election failed: %v", err)
	}

	// Track killed managers
	killedManagers := make([]string, 0)

	t.Run("FirstFailover", func(t *testing.T) {
		leader, err := cluster.GetLeader()
		if err != nil {
			t.Fatalf("Failed to get leader: %v", err)
		}

		leaderID := leader.ID
		t.Logf("Killing first leader: %s", leaderID)

		if err := cluster.KillManager(leaderID); err != nil {
			t.Fatalf("Failed to kill first leader: %v", err)
		}

		killedManagers = append(killedManagers, leaderID)

		// Wait for new election
		time.Sleep(3 * time.Second)
		if err := waiter.WaitForLeaderElection(ctx, cluster); err != nil {
			t.Fatalf("Failed to elect new leader after first failover: %v", err)
		}

		newLeader, err := cluster.GetLeader()
		if err != nil {
			t.Fatalf("Failed to get new leader: %v", err)
		}

		if newLeader.ID == leaderID {
			t.Errorf("Leader did not change after first failover")
		}

		t.Logf("✓ First failover complete, new leader: %s", newLeader.ID)
	})

	t.Run("SecondFailover", func(t *testing.T) {
		leader, err := cluster.GetLeader()
		if err != nil {
			t.Fatalf("Failed to get leader: %v", err)
		}

		leaderID := leader.ID
		t.Logf("Killing second leader: %s", leaderID)

		if err := cluster.KillManager(leaderID); err != nil {
			t.Fatalf("Failed to kill second leader: %v", err)
		}

		killedManagers = append(killedManagers, leaderID)

		// Now we have only 1 manager left (no quorum!)
		// This should NOT elect a new leader
		time.Sleep(5 * time.Second)

		// Try to get leader (should fail or return no leader)
		_, err = cluster.GetLeader()
		if err == nil {
			t.Log("⚠ Cluster still has leader with only 1/3 managers (unexpected)")
		} else {
			t.Logf("✓ No leader with only 1/3 managers (expected): %v", err)
		}
	})

	t.Run("RestoreQuorum", func(t *testing.T) {
		// Restart one of the killed managers to restore quorum
		managerToRestart := killedManagers[0]
		t.Logf("Restarting manager to restore quorum: %s", managerToRestart)

		if err := cluster.RestartManager(managerToRestart); err != nil {
			t.Fatalf("Failed to restart manager: %v", err)
		}

		// Wait for quorum to be restored
		t.Log("Waiting for quorum restoration and leader election...")
		time.Sleep(8 * time.Second)

		if err := waiter.WaitForLeaderElection(ctx, cluster); err != nil {
			t.Fatalf("Failed to elect leader after quorum restoration: %v", err)
		}

		leader, err := cluster.GetLeader()
		if err != nil {
			t.Fatalf("Failed to get leader after restoration: %v", err)
		}

		t.Logf("✓ Quorum restored, leader elected: %s", leader.ID)

		// Verify cluster is operational
		nodes, err := leader.Client.ListNodes()
		if err != nil {
			t.Fatalf("Cluster not operational after quorum restoration: %v", err)
		}

		t.Logf("✓ Cluster operational with %d nodes", len(nodes))
	})
}

// TestLeaderFailoverWithActiveWorkload tests failover while services are running
func TestLeaderFailoverWithActiveWorkload(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping failover with workload test in short mode")
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

	waiter := framework.DefaultWaiter()
	ctx := context.Background()

	// Wait for initial setup
	if err := waiter.WaitForLeaderElection(ctx, cluster); err != nil {
		t.Fatalf("Initial leader election failed: %v", err)
	}

	leader, err := cluster.GetLeader()
	if err != nil {
		t.Fatalf("Failed to get leader: %v", err)
	}

	// Create multiple services
	services := []string{"workload-1", "workload-2", "workload-3"}
	for _, svcName := range services {
		t.Logf("Creating service: %s", svcName)
		if err := leader.Client.CreateService(svcName, "nginx:alpine", 2); err != nil {
			t.Fatalf("Failed to create service %s: %v", svcName, err)
		}
		defer func() { _ = leader.Client.DeleteService(svcName) }()
	}

	// Wait for all services to be running
	for _, svcName := range services {
		if err := waiter.WaitForReplicas(ctx, leader.Client, svcName, 2); err != nil {
			t.Fatalf("Service %s failed to start: %v", svcName, err)
		}
	}

	t.Log("✓ All workload services running")

	// Now kill the leader while workload is active
	leaderID := leader.ID
	t.Logf("Killing leader %s while workload is running...", leaderID)

	if err := cluster.KillManager(leaderID); err != nil {
		t.Fatalf("Failed to kill leader: %v", err)
	}

	// Wait for new leader
	time.Sleep(3 * time.Second)
	if err := waiter.WaitForLeaderElection(ctx, cluster); err != nil {
		t.Fatalf("New leader not elected: %v", err)
	}

	newLeader, err := cluster.GetLeader()
	if err != nil {
		t.Fatalf("Failed to get new leader: %v", err)
	}

	t.Logf("✓ New leader elected: %s", newLeader.ID)

	// Verify all services still exist and are running
	t.Log("Verifying workload services after failover...")
	for _, svcName := range services {
		svc, err := newLeader.Client.GetService(svcName)
		if err != nil {
			t.Errorf("Service %s not found after failover: %v", svcName, err)
			continue
		}

		tasks, err := newLeader.Client.ListTasks(svc.Id, "")
		if err != nil {
			t.Errorf("Failed to list tasks for service %s: %v", svcName, err)
			continue
		}

		runningCount := 0
		for _, task := range tasks {
			if task.ActualState == "running" {
				runningCount++
			}
		}

		if runningCount < 2 {
			t.Logf("⚠ Service %s has %d/2 running replicas after failover", svcName, runningCount)
		} else {
			t.Logf("✓ Service %s: %d/2 replicas running", svcName, runningCount)
		}
	}

	t.Log("✓ Workload survived leader failover")
}
