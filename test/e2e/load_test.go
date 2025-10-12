package e2e

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/cuemby/warren/test/framework"
)

// TestLoadSmall tests basic load handling with 50 services
// This replaces test/lima/test-load.sh --scale small
func TestLoadSmall(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	testLoad(t, LoadConfig{
		Name:            "Small",
		NumManagers:     1,
		NumWorkers:      2,
		NumServices:     50,
		ReplicasPerSvc:  2,
		MaxCreationTime: 2 * time.Minute,
	})
}

// TestLoadMedium tests moderate load handling with 200 services
func TestLoadMedium(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping medium load test in short mode")
	}

	testLoad(t, LoadConfig{
		Name:            "Medium",
		NumManagers:     3,
		NumWorkers:      5,
		NumServices:     200,
		ReplicasPerSvc:  3,
		MaxCreationTime: 5 * time.Minute,
	})
}

// TestLoadLarge tests high load handling with 1000 services
// This is a stress test and should be run manually
func TestLoadLarge(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large load test in short mode")
	}

	t.Skip("Large load test disabled by default - run manually with go test -run TestLoadLarge")

	testLoad(t, LoadConfig{
		Name:            "Large",
		NumManagers:     3,
		NumWorkers:      10,
		NumServices:     1000,
		ReplicasPerSvc:  3,
		MaxCreationTime: 15 * time.Minute,
	})
}

// LoadConfig defines load test parameters
type LoadConfig struct {
	Name            string
	NumManagers     int
	NumWorkers      int
	NumServices     int
	ReplicasPerSvc  int
	MaxCreationTime time.Duration
}

// testLoad executes a load test with given configuration
func testLoad(t *testing.T, config LoadConfig) {
	t.Logf("Starting %s load test: %d services × %d replicas = %d tasks",
		config.Name, config.NumServices, config.ReplicasPerSvc,
		config.NumServices*config.ReplicasPerSvc)

	// Create cluster
	clusterConfig := &framework.ClusterConfig{
		NumManagers: config.NumManagers,
		NumWorkers:  config.NumWorkers,
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

	cluster, err := framework.NewCluster(clusterConfig)
	if err != nil {
		t.Fatalf("Failed to create cluster: %v", err)
	}
	defer cluster.Cleanup()

	if err := cluster.Start(); err != nil {
		t.Fatalf("Failed to start cluster: %v", err)
	}
	defer cluster.Stop()

	waiter := framework.DefaultWaiter()
	ctx := context.Background()

	// Wait for cluster ready
	t.Run("SetupCluster", func(t *testing.T) {
		if err := waiter.WaitForLeaderElection(ctx, cluster); err != nil {
			t.Fatalf("Leader election failed: %v", err)
		}

		leader, err := cluster.GetLeader()
		if err != nil {
			t.Fatalf("Failed to get leader: %v", err)
		}

		expectedNodes := config.NumManagers + config.NumWorkers
		if err := waiter.WaitForNodeCount(ctx, leader.Client, expectedNodes); err != nil {
			t.Fatalf("Expected %d nodes: %v", expectedNodes, err)
		}

		t.Logf("✓ Cluster ready: %d managers, %d workers", config.NumManagers, config.NumWorkers)
	})

	// Measure service creation throughput
	t.Run("CreateServices", func(t *testing.T) {
		leader, err := cluster.GetLeader()
		if err != nil {
			t.Fatalf("Failed to get leader: %v", err)
		}

		t.Logf("Creating %d services with %d replicas each...", config.NumServices, config.ReplicasPerSvc)

		createStart := time.Now()
		failures := 0

		// Create services in batches for better performance
		batchSize := 50
		numBatches := (config.NumServices + batchSize - 1) / batchSize

		for batch := 0; batch < numBatches; batch++ {
			startIdx := batch * batchSize
			endIdx := (batch + 1) * batchSize
			if endIdx > config.NumServices {
				endIdx = config.NumServices
			}

			batchStart := time.Now()
			batchFailures := createServiceBatch(t, leader.Client, startIdx, endIdx, config.ReplicasPerSvc)
			batchDuration := time.Since(batchStart)

			failures += batchFailures
			batchSize := endIdx - startIdx

			if batchFailures == 0 {
				rate := float64(batchSize) / batchDuration.Seconds()
				t.Logf("  Batch %d/%d: Created %d services in %v (%.1f svc/s)",
					batch+1, numBatches, batchSize, batchDuration, rate)
			} else {
				t.Logf("  Batch %d/%d: Created %d/%d services (%d failed)",
					batch+1, numBatches, batchSize-batchFailures, batchSize, batchFailures)
			}
		}

		createDuration := time.Since(createStart)
		successCount := config.NumServices - failures

		rate := float64(successCount) / createDuration.Seconds()
		t.Logf("✓ Service creation complete:")
		t.Logf("  Total time: %v", createDuration)
		t.Logf("  Success: %d/%d services", successCount, config.NumServices)
		t.Logf("  Throughput: %.2f services/s", rate)

		if failures > 0 {
			failureRate := float64(failures) / float64(config.NumServices) * 100
			if failureRate > 5.0 {
				t.Errorf("High failure rate: %.1f%% (%d/%d)", failureRate, failures, config.NumServices)
			} else {
				t.Logf("⚠ Failures: %d (%.1f%%)", failures, failureRate)
			}
		}

		if createDuration > config.MaxCreationTime {
			t.Errorf("Service creation took too long: %v (max: %v)", createDuration, config.MaxCreationTime)
		}
	})

	// Measure API performance under load
	t.Run("APIPerformance", func(t *testing.T) {
		leader, err := cluster.GetLeader()
		if err != nil {
			t.Fatalf("Failed to get leader: %v", err)
		}

		t.Log("Measuring API latency with load...")

		numRequests := 100
		latencies := make([]time.Duration, numRequests)

		for i := 0; i < numRequests; i++ {
			start := time.Now()
			_, err := leader.Client.ListServices()
			latencies[i] = time.Since(start)

			if err != nil {
				t.Logf("Request %d failed: %v", i, err)
			}
		}

		// Calculate statistics
		var sum time.Duration
		min := time.Hour
		max := time.Duration(0)

		for _, lat := range latencies {
			sum += lat
			if lat < min {
				min = lat
			}
			if lat > max {
				max = lat
			}
		}

		avg := sum / time.Duration(numRequests)

		t.Logf("✓ API latency under load:")
		t.Logf("  Requests: %d", numRequests)
		t.Logf("  Average: %v", avg)
		t.Logf("  Min: %v", min)
		t.Logf("  Max: %v", max)

		// Check if API is still responsive
		if avg > 2*time.Second {
			t.Errorf("API too slow under load: avg latency %v", avg)
		}

		if max > 10*time.Second {
			t.Errorf("API max latency too high: %v", max)
		}
	})

	// Verify cluster stability
	t.Run("ClusterStability", func(t *testing.T) {
		leader, err := cluster.GetLeader()
		if err != nil {
			t.Fatalf("Failed to get leader: %v", err)
		}

		// List all services
		services, err := leader.Client.ListServices()
		if err != nil {
			t.Fatalf("Failed to list services: %v", err)
		}

		t.Logf("Cluster has %d services", len(services))

		// Verify all nodes still healthy
		nodes, err := leader.Client.ListNodes()
		if err != nil {
			t.Fatalf("Failed to list nodes: %v", err)
		}

		healthyNodes := 0
		for _, node := range nodes {
			if node.Status == "ready" {
				healthyNodes++
			}
		}

		expectedNodes := config.NumManagers + config.NumWorkers
		if healthyNodes < expectedNodes {
			t.Errorf("Not all nodes healthy: %d/%d", healthyNodes, expectedNodes)
		} else {
			t.Logf("✓ All %d nodes healthy", healthyNodes)
		}

		// Check leader still stable
		info, err := leader.Client.GetClusterInfo()
		if err != nil {
			t.Fatalf("Failed to get cluster info: %v", err)
		}

		if info.LeaderId == "" {
			t.Error("No leader after load test")
		} else {
			t.Logf("✓ Leader stable: %s", info.LeaderId)
		}
	})

	// Cleanup services
	t.Run("Cleanup", func(t *testing.T) {
		leader, err := cluster.GetLeader()
		if err != nil {
			t.Fatalf("Failed to get leader: %v", err)
		}

		t.Logf("Cleaning up %d test services...", config.NumServices)

		cleanupStart := time.Now()
		failures := 0

		// Delete in batches
		batchSize := 50
		numBatches := (config.NumServices + batchSize - 1) / batchSize

		for batch := 0; batch < numBatches; batch++ {
			startIdx := batch * batchSize
			endIdx := (batch + 1) * batchSize
			if endIdx > config.NumServices {
				endIdx = config.NumServices
			}

			batchFailures := deleteServiceBatch(leader.Client, startIdx, endIdx)
			failures += batchFailures

			if batch%5 == 0 {
				t.Logf("  Progress: %d/%d services deleted", endIdx, config.NumServices)
			}
		}

		cleanupDuration := time.Since(cleanupStart)

		if failures > 0 {
			t.Logf("⚠ Cleanup: %d failures", failures)
		} else {
			t.Logf("✓ Cleanup complete in %v", cleanupDuration)
		}
	})
}

// createServiceBatch creates a batch of services concurrently
func createServiceBatch(t *testing.T, client *framework.Client, startIdx, endIdx, replicas int) int {
	failures := 0
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i := startIdx; i < endIdx; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			serviceName := fmt.Sprintf("load-test-%d", idx)
			if err := client.CreateService(serviceName, "nginx:alpine", replicas); err != nil {
				mu.Lock()
				failures++
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()
	return failures
}

// deleteServiceBatch deletes a batch of services concurrently
func deleteServiceBatch(client *framework.Client, startIdx, endIdx int) int {
	failures := 0
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i := startIdx; i < endIdx; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			serviceName := fmt.Sprintf("load-test-%d", idx)
			if err := client.DeleteService(serviceName); err != nil {
				mu.Lock()
				failures++
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()
	return failures
}

// TestSchedulerPerformance tests scheduler throughput and task distribution
func TestSchedulerPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping scheduler performance test in short mode")
	}

	config := &framework.ClusterConfig{
		NumManagers: 1,
		NumWorkers:  3,
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
	defer cluster.Cleanup()

	if err := cluster.Start(); err != nil {
		t.Fatalf("Failed to start cluster: %v", err)
	}
	defer cluster.Stop()

	waiter := framework.DefaultWaiter()
	ctx := context.Background()

	// Wait for cluster
	if err := waiter.WaitForLeaderElection(ctx, cluster); err != nil {
		t.Fatalf("Leader election failed: %v", err)
	}

	leader, err := cluster.GetLeader()
	if err != nil {
		t.Fatalf("Failed to get leader: %v", err)
	}

	if err := waiter.WaitForNodeCount(ctx, leader.Client, 4); err != nil {
		t.Fatalf("Expected 4 nodes: %v", err)
	}

	t.Run("TaskDistribution", func(t *testing.T) {
		// Create service with many replicas
		numReplicas := 30 // 10 per worker ideally
		serviceName := "distribution-test"

		t.Logf("Creating service with %d replicas...", numReplicas)
		if err := leader.Client.CreateService(serviceName, "nginx:alpine", numReplicas); err != nil {
			t.Fatalf("Failed to create service: %v", err)
		}
		defer leader.Client.DeleteService(serviceName)

		// Wait for tasks to be scheduled
		t.Log("Waiting for scheduler to assign tasks...")
		time.Sleep(10 * time.Second)

		// Get service and tasks
		svc, err := leader.Client.GetService(serviceName)
		if err != nil {
			t.Fatalf("Failed to get service: %v", err)
		}

		tasks, err := leader.Client.ListTasks(svc.Id, "")
		if err != nil {
			t.Fatalf("Failed to list tasks: %v", err)
		}

		// Analyze task distribution across nodes
		nodeTaskCount := make(map[string]int)
		runningTasks := 0

		for _, task := range tasks {
			if task.ActualState == "running" {
				runningTasks++
			}
			if task.NodeId != "" {
				nodeTaskCount[task.NodeId]++
			}
		}

		t.Logf("Task distribution:")
		for nodeID, count := range nodeTaskCount {
			percentage := float64(count) / float64(len(tasks)) * 100
			t.Logf("  Node %s: %d tasks (%.1f%%)", nodeID, count, percentage)
		}

		// Check distribution quality
		if len(nodeTaskCount) < 2 {
			t.Error("Poor task distribution: all tasks on single node")
		}

		// Check for severe imbalance (more than 60% on one node)
		for nodeID, count := range nodeTaskCount {
			percentage := float64(count) / float64(len(tasks)) * 100
			if percentage > 60.0 {
				t.Errorf("Unbalanced distribution: node %s has %.1f%% of tasks", nodeID, percentage)
			}
		}

		t.Logf("✓ Scheduler distributed %d tasks across %d workers", len(tasks), len(nodeTaskCount))
	})
}
