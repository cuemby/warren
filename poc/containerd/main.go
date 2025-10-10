package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime"
	"syscall"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/oci"
)

const (
	containerdSocket = "/run/containerd/containerd.sock"
	namespace        = "warren-poc"
	testImage        = "docker.io/library/nginx:latest"
	testContainer    = "warren-test-nginx"
)

func main() {
	log.Println("=== Warren Containerd POC ===")
	log.Printf("Containerd socket: %s", containerdSocket)
	log.Printf("Namespace: %s", namespace)
	log.Println()

	// Check if containerd socket exists
	if _, err := os.Stat(containerdSocket); os.IsNotExist(err) {
		log.Fatalf("Containerd socket not found at %s\n"+
			"Please ensure containerd is running:\n"+
			"  macOS: brew install containerd && sudo containerd\n"+
			"  Linux: sudo systemctl start containerd\n", containerdSocket)
	}

	// Connect to containerd
	log.Println("1. Connecting to containerd...")
	client, err := containerd.New(containerdSocket)
	if err != nil {
		log.Fatalf("Failed to connect to containerd: %v", err)
	}
	defer client.Close()
	log.Println("✓ Connected to containerd")

	// Create context with namespace
	ctx := namespaces.WithNamespace(context.Background(), namespace)

	// Test 1: Pull image
	log.Printf("\n2. Pulling image: %s...", testImage)
	startTime := time.Now()
	image, err := pullImage(ctx, client, testImage)
	if err != nil {
		log.Fatalf("Failed to pull image: %v", err)
	}
	log.Printf("✓ Image pulled in %v", time.Since(startTime))
	log.Printf("  Image size: %d bytes", image.Size())

	// Test 2: Create container
	log.Printf("\n3. Creating container: %s...", testContainer)
	container, err := createContainer(ctx, client, testContainer, image)
	if err != nil {
		log.Fatalf("Failed to create container: %v", err)
	}
	defer func() {
		// Cleanup: delete container
		log.Println("\n7. Cleaning up: Deleting container...")
		if err := container.Delete(ctx, containerd.WithSnapshotCleanup); err != nil {
			log.Printf("Failed to delete container: %v", err)
		} else {
			log.Println("✓ Container deleted")
		}
	}()
	log.Println("✓ Container created")

	// Test 3: Start container
	log.Println("\n4. Starting container...")
	task, err := startContainer(ctx, container)
	if err != nil {
		log.Fatalf("Failed to start container: %v", err)
	}
	defer func() {
		// Cleanup: kill task
		log.Println("\n6. Stopping container...")
		if err := task.Kill(ctx, syscall.SIGTERM); err != nil {
			log.Printf("Failed to kill task: %v", err)
		}
		task.Wait(ctx)
		log.Println("✓ Container stopped")
	}()
	log.Printf("✓ Container started (PID: %d)", task.Pid())

	// Test 4: Check container status
	log.Println("\n5. Checking container status...")
	status, err := task.Status(ctx)
	if err != nil {
		log.Printf("Failed to get status: %v", err)
	} else {
		log.Printf("✓ Container status: %s", status.Status)
	}

	// Memory usage check
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	log.Printf("\n--- Memory Usage ---")
	log.Printf("Alloc: %d MB", m.Alloc/1024/1024)
	log.Printf("TotalAlloc: %d MB", m.TotalAlloc/1024/1024)
	log.Printf("Sys: %d MB", m.Sys/1024/1024)

	log.Println("\n✅ All tests passed!")
	log.Println("Container will run for 5 seconds before cleanup...")
	time.Sleep(5 * time.Second)
}

// pullImage pulls an image from a registry
func pullImage(ctx context.Context, client *containerd.Client, ref string) (containerd.Image, error) {
	image, err := client.Pull(ctx, ref, containerd.WithPullUnpack)
	if err != nil {
		return nil, fmt.Errorf("pull failed: %w", err)
	}
	return image, nil
}

// createContainer creates a new container from an image
func createContainer(ctx context.Context, client *containerd.Client, id string, image containerd.Image) (containerd.Container, error) {
	container, err := client.NewContainer(
		ctx,
		id,
		containerd.WithImage(image),
		containerd.WithNewSnapshot(id+"-snapshot", image),
		containerd.WithNewSpec(
			oci.WithImageConfig(image),
			oci.WithEnv([]string{
				"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
			}),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("create container failed: %w", err)
	}
	return container, nil
}

// startContainer starts a container and returns its task
func startContainer(ctx context.Context, container containerd.Container) (containerd.Task, error) {
	// Create task (the running instance of the container)
	task, err := container.NewTask(ctx, cio.NewCreator(cio.WithStdio))
	if err != nil {
		return nil, fmt.Errorf("create task failed: %w", err)
	}

	// Start the task
	if err := task.Start(ctx); err != nil {
		return nil, fmt.Errorf("start task failed: %w", err)
	}

	return task, nil
}
