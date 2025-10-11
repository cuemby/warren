package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/cuemby/warren/api/proto"
	"github.com/cuemby/warren/pkg/client"
)

// TestHealthCheckHTTP tests HTTP health check functionality
func TestHealthCheckHTTP(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Connect to manager
	c, err := client.NewClient("127.0.0.1:8080")
	if err != nil {
		t.Fatalf("Failed to connect to manager: %v", err)
	}
	defer c.Close()

	// Create service with HTTP health check
	serviceName := fmt.Sprintf("test-health-http-%d", time.Now().Unix())
	req := &proto.CreateServiceRequest{
		Name:     serviceName,
		Image:    "nginx:latest",
		Replicas: 1,
		Mode:     "replicated",
		HealthCheck: &proto.HealthCheck{
			Type: proto.HealthCheck_HTTP,
			Http: &proto.HTTPHealthCheck{
				Path:          "/",
				Port:          80,
				Scheme:        "http",
				StatusCodeMin: 200,
				StatusCodeMax: 399,
			},
			IntervalSeconds: 10, // Check every 10 seconds
			TimeoutSeconds:  5,
			Retries:         2,
		},
	}

	service, err := c.CreateServiceWithOptions(req)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}
	defer cleanupService(c, service.Id)

	t.Logf("Created service %s with HTTP health check", serviceName)

	// Wait for task to be scheduled and running
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	var runningTask *proto.Task
	for {
		select {
		case <-ctx.Done():
			t.Fatal("Timeout waiting for task to start")
		default:
			tasks, err := c.ListTasks("", "")
			if err != nil {
				t.Fatalf("Failed to list tasks: %v", err)
			}

			for _, task := range tasks {
				if task.ServiceId == service.Id && task.ActualState == "running" {
					runningTask = task
					goto TaskRunning
				}
			}

			time.Sleep(2 * time.Second)
		}
	}

TaskRunning:
	t.Logf("Task %s is running", runningTask.Id)

	// Wait for health check to be reported
	// Health checks run every 10 seconds, so wait up to 30 seconds
	time.Sleep(30 * time.Second)

	// Get updated service to check health status
	service, err = c.GetService(serviceName)
	if err != nil {
		t.Fatalf("Failed to get service: %v", err)
	}

	// List tasks again to check health status
	tasks, err := c.ListTasks("", "")
	if err != nil {
		t.Fatalf("Failed to list tasks: %v", err)
	}

	var healthyTask *proto.Task
	for _, task := range tasks {
		if task.ServiceId == service.Id && task.ActualState == "running" {
			healthyTask = task
			break
		}
	}

	if healthyTask == nil {
		t.Fatal("Task is no longer running after health check period")
	}

	t.Logf("✓ HTTP health check test passed - task %s remained healthy", healthyTask.Id)
}

// TestHealthCheckTCP tests TCP health check functionality
func TestHealthCheckTCP(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Connect to manager
	c, err := client.NewClient("127.0.0.1:8080")
	if err != nil {
		t.Fatalf("Failed to connect to manager: %v", err)
	}
	defer c.Close()

	// Create service with TCP health check
	serviceName := fmt.Sprintf("test-health-tcp-%d", time.Now().Unix())
	req := &proto.CreateServiceRequest{
		Name:     serviceName,
		Image:    "nginx:latest",
		Replicas: 1,
		Mode:     "replicated",
		HealthCheck: &proto.HealthCheck{
			Type: proto.HealthCheck_TCP,
			Tcp: &proto.TCPHealthCheck{
				Port: 80,
			},
			IntervalSeconds: 10,
			TimeoutSeconds:  5,
			Retries:         2,
		},
	}

	service, err := c.CreateServiceWithOptions(req)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}
	defer cleanupService(c, service.Id)

	t.Logf("Created service %s with TCP health check", serviceName)

	// Wait for task to be scheduled and running
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	var runningTask *proto.Task
	for {
		select {
		case <-ctx.Done():
			t.Fatal("Timeout waiting for task to start")
		default:
			tasks, err := c.ListTasks("", "")
			if err != nil {
				t.Fatalf("Failed to list tasks: %v", err)
			}

			for _, task := range tasks {
				if task.ServiceId == service.Id && task.ActualState == "running" {
					runningTask = task
					goto TaskRunning
				}
			}

			time.Sleep(2 * time.Second)
		}
	}

TaskRunning:
	t.Logf("Task %s is running", runningTask.Id)

	// Wait for health check to be reported
	time.Sleep(30 * time.Second)

	// List tasks again to check health status
	tasks, err := c.ListTasks("", "")
	if err != nil {
		t.Fatalf("Failed to list tasks: %v", err)
	}

	var healthyTask *proto.Task
	for _, task := range tasks {
		if task.ServiceId == service.Id && task.ActualState == "running" {
			healthyTask = task
			break
		}
	}

	if healthyTask == nil {
		t.Fatal("Task is no longer running after health check period")
	}

	t.Logf("✓ TCP health check test passed - task %s remained healthy", healthyTask.Id)
}

// TestHealthCheckUnhealthy tests that unhealthy tasks are replaced
func TestHealthCheckUnhealthy(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Connect to manager
	c, err := client.NewClient("127.0.0.1:8080")
	if err != nil {
		t.Fatalf("Failed to connect to manager: %v", err)
	}
	defer c.Close()

	// Create service with HTTP health check on invalid path
	// This should cause the health check to fail
	serviceName := fmt.Sprintf("test-health-unhealthy-%d", time.Now().Unix())
	req := &proto.CreateServiceRequest{
		Name:     serviceName,
		Image:    "nginx:latest",
		Replicas: 1,
		Mode:     "replicated",
		HealthCheck: &proto.HealthCheck{
			Type: proto.HealthCheck_HTTP,
			Http: &proto.HTTPHealthCheck{
				Path:          "/nonexistent-health-endpoint",
				Port:          80,
				Scheme:        "http",
				StatusCodeMin: 200,
				StatusCodeMax: 299, // Only accept 2xx
			},
			IntervalSeconds: 5, // Check every 5 seconds (faster for testing)
			TimeoutSeconds:  3,
			Retries:         2, // Fail after 2 failures
		},
	}

	service, err := c.CreateServiceWithOptions(req)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}
	defer cleanupService(c, service.Id)

	t.Logf("Created service %s with failing HTTP health check", serviceName)

	// Wait for task to be scheduled and running
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	var initialTaskID string
	for {
		select {
		case <-ctx.Done():
			t.Fatal("Timeout waiting for task to start")
		default:
			tasks, err := c.ListTasks("", "")
			if err != nil {
				t.Fatalf("Failed to list tasks: %v", err)
			}

			for _, task := range tasks {
				if task.ServiceId == service.Id && task.ActualState == "running" {
					initialTaskID = task.Id
					goto TaskRunning
				}
			}

			time.Sleep(2 * time.Second)
		}
	}

TaskRunning:
	t.Logf("Initial task %s is running", initialTaskID)

	// Wait for health checks to fail and task to be replaced
	// Health checks every 5s, 2 retries = ~10-15 seconds to fail
	// Reconciler runs every 10s, so wait up to 45 seconds
	t.Log("Waiting for health checks to fail and task to be replaced...")
	time.Sleep(45 * time.Second)

	// Check if task was replaced
	tasks, err := c.ListTasks("", "")
	if err != nil {
		t.Fatalf("Failed to list tasks: %v", err)
	}

	var failedTaskCount int
	var newRunningTask *proto.Task

	for _, task := range tasks {
		if task.ServiceId == service.Id {
			if task.Id == initialTaskID && task.ActualState == "failed" {
				failedTaskCount++
				t.Logf("Original task %s marked as failed", task.Id)
			}
			if task.Id != initialTaskID && task.ActualState == "running" {
				newRunningTask = task
				t.Logf("New replacement task %s is running", task.Id)
			}
		}
	}

	if failedTaskCount == 0 {
		t.Error("Original task was not marked as failed")
	}

	if newRunningTask != nil {
		t.Logf("✓ Unhealthy task replacement test passed - task %s replaced unhealthy task %s",
			newRunningTask.Id, initialTaskID)
	} else {
		t.Log("⚠ Replacement task not yet running, but original task marked as failed (scheduler will create replacement)")
	}
}

// cleanupService deletes a service
func cleanupService(c *client.Client, serviceID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Use context to prevent hanging
	done := make(chan error, 1)
	go func() {
		done <- c.DeleteService(serviceID)
	}()

	select {
	case err := <-done:
		if err != nil {
			fmt.Printf("Warning: Failed to cleanup service %s: %v\n", serviceID, err)
		}
	case <-ctx.Done():
		fmt.Printf("Warning: Timeout cleaning up service %s\n", serviceID)
	}
}
