package integration

import (
	"context"
	"testing"
	"time"

	"github.com/cuemby/warren/pkg/runtime"
	"github.com/cuemby/warren/pkg/types"
	"github.com/google/uuid"
)

// TestContainerdBasicWorkflow tests the basic containerd workflow:
// pull image → create container → start → check status → stop → delete
func TestContainerdBasicWorkflow(t *testing.T) {
	// Skip if containerd is not available
	rt, err := runtime.NewContainerdRuntime("")
	if err != nil {
		t.Skipf("Containerd not available: %v", err)
	}
	defer rt.Close()

	ctx := context.Background()
	taskID := uuid.New().String()

	// Test task with nginx
	task := &types.Task{
		ID:    taskID,
		Image: "docker.io/library/nginx:alpine",
		Env:   []string{"TEST=integration"},
	}

	t.Log("Step 1: Pulling nginx:alpine image...")
	if err := rt.PullImage(ctx, task.Image); err != nil {
		t.Fatalf("Failed to pull image: %v", err)
	}
	t.Log("✓ Image pulled successfully")

	t.Log("Step 2: Creating container...")
	containerID, err := rt.CreateContainer(ctx, task)
	if err != nil {
		t.Fatalf("Failed to create container: %v", err)
	}
	t.Logf("✓ Container created: %s", containerID)

	// Ensure cleanup
	defer func() {
		t.Log("Cleanup: Deleting container...")
		if err := rt.DeleteContainer(ctx, containerID); err != nil {
			t.Logf("Warning: Failed to delete container: %v", err)
		}
	}()

	t.Log("Step 3: Starting container...")
	if err := rt.StartContainer(ctx, containerID); err != nil {
		t.Fatalf("Failed to start container: %v", err)
	}
	t.Log("✓ Container started")

	// Wait a moment for container to fully start
	time.Sleep(2 * time.Second)

	t.Log("Step 4: Checking container status...")
	status, err := rt.GetContainerStatus(ctx, containerID)
	if err != nil {
		t.Fatalf("Failed to get container status: %v", err)
	}
	t.Logf("✓ Container status: %s", status)

	if status != types.TaskStateRunning {
		t.Errorf("Expected container to be running, got: %s", status)
	}

	t.Log("Step 5: Verifying container is running...")
	if !rt.IsRunning(ctx, containerID) {
		t.Error("Container should be running but IsRunning returned false")
	}
	t.Log("✓ Container is running")

	t.Log("Step 6: Stopping container...")
	if err := rt.StopContainer(ctx, containerID, 10*time.Second); err != nil {
		t.Fatalf("Failed to stop container: %v", err)
	}
	t.Log("✓ Container stopped")

	t.Log("Step 7: Verifying container stopped...")
	if rt.IsRunning(ctx, containerID) {
		t.Error("Container should be stopped but IsRunning returned true")
	}
	t.Log("✓ Container is not running")

	t.Log("✅ All steps completed successfully!")
}

// TestContainerdListContainers tests listing containers
func TestContainerdListContainers(t *testing.T) {
	rt, err := runtime.NewContainerdRuntime("")
	if err != nil {
		t.Skipf("Containerd not available: %v", err)
	}
	defer rt.Close()

	ctx := context.Background()

	t.Log("Listing containers in Warren namespace...")
	containers, err := rt.ListContainers(ctx)
	if err != nil {
		t.Fatalf("Failed to list containers: %v", err)
	}

	t.Logf("Found %d containers in Warren namespace", len(containers))
	for _, id := range containers {
		t.Logf("  - %s", id)
	}
}

// TestContainerdPullMultipleImages tests pulling multiple images
func TestContainerdPullMultipleImages(t *testing.T) {
	rt, err := runtime.NewContainerdRuntime("")
	if err != nil {
		t.Skipf("Containerd not available: %v", err)
	}
	defer rt.Close()

	ctx := context.Background()

	images := []string{
		"docker.io/library/nginx:alpine",
		"docker.io/library/redis:alpine",
	}

	for _, img := range images {
		t.Logf("Pulling image: %s", img)
		if err := rt.PullImage(ctx, img); err != nil {
			t.Errorf("Failed to pull image %s: %v", img, err)
		} else {
			t.Logf("✓ Successfully pulled: %s", img)
		}
	}
}
