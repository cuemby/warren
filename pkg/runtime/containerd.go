package runtime

import (
	"context"
	"fmt"
	"io"
	"syscall"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/oci"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/cuemby/warren/pkg/types"
)

const (
	// DefaultNamespace is the containerd namespace for Warren
	DefaultNamespace = "warren"

	// DefaultSocketPath is the default containerd socket
	DefaultSocketPath = "/run/containerd/containerd.sock"
)

// ContainerdRuntime implements container runtime using containerd
type ContainerdRuntime struct {
	client    *containerd.Client
	namespace string
}

// NewContainerdRuntime creates a new containerd runtime client
func NewContainerdRuntime(socketPath string) (*ContainerdRuntime, error) {
	if socketPath == "" {
		socketPath = DefaultSocketPath
	}

	client, err := containerd.New(socketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to containerd: %w", err)
	}

	return &ContainerdRuntime{
		client:    client,
		namespace: DefaultNamespace,
	}, nil
}

// Close closes the containerd client connection
func (r *ContainerdRuntime) Close() error {
	if r.client != nil {
		return r.client.Close()
	}
	return nil
}

// PullImage pulls a container image from a registry
func (r *ContainerdRuntime) PullImage(ctx context.Context, imageRef string) error {
	ctx = namespaces.WithNamespace(ctx, r.namespace)

	// Pull the image
	_, err := r.client.Pull(ctx, imageRef, containerd.WithPullUnpack)
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %w", imageRef, err)
	}

	return nil
}

// CreateContainer creates a container from a task specification
func (r *ContainerdRuntime) CreateContainer(ctx context.Context, task *types.Task) (string, error) {
	ctx = namespaces.WithNamespace(ctx, r.namespace)

	// Get the image
	image, err := r.client.GetImage(ctx, task.Image)
	if err != nil {
		return "", fmt.Errorf("failed to get image %s: %w", task.Image, err)
	}

	// Create container spec with environment variables
	opts := []oci.SpecOpts{
		oci.WithImageConfig(image),
		oci.WithEnv(task.Env),
	}

	// Create the container
	container, err := r.client.NewContainer(
		ctx,
		task.ID,
		containerd.WithImage(image),
		containerd.WithNewSnapshot(task.ID+"-snapshot", image),
		containerd.WithNewSpec(opts...),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	return container.ID(), nil
}

// CreateContainerWithSecrets creates a container with secret tmpfs mounts
func (r *ContainerdRuntime) CreateContainerWithSecrets(ctx context.Context, task *types.Task, secretsPath string) (string, error) {
	ctx = namespaces.WithNamespace(ctx, r.namespace)

	// Get the image
	image, err := r.client.GetImage(ctx, task.Image)
	if err != nil {
		return "", fmt.Errorf("failed to get image %s: %w", task.Image, err)
	}

	// Create container spec with environment variables and secrets mount
	opts := []oci.SpecOpts{
		oci.WithImageConfig(image),
		oci.WithEnv(task.Env),
	}

	// Add bind mount for secrets if provided
	if secretsPath != "" {
		opts = append(opts, oci.WithMounts([]specs.Mount{
			{
				Source:      secretsPath,
				Destination: "/run/secrets",
				Type:        "bind",
				Options:     []string{"ro", "bind"}, // Read-only bind mount
			},
		}))
	}

	// Create the container
	container, err := r.client.NewContainer(
		ctx,
		task.ID,
		containerd.WithImage(image),
		containerd.WithNewSnapshot(task.ID+"-snapshot", image),
		containerd.WithNewSpec(opts...),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	return container.ID(), nil
}

// StartContainer starts a container and returns its runtime ID
func (r *ContainerdRuntime) StartContainer(ctx context.Context, containerID string) error {
	ctx = namespaces.WithNamespace(ctx, r.namespace)

	// Get the container
	container, err := r.client.LoadContainer(ctx, containerID)
	if err != nil {
		return fmt.Errorf("failed to load container %s: %w", containerID, err)
	}

	// Create a task (running instance)
	task, err := container.NewTask(ctx, cio.NullIO)
	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	// Start the task
	if err := task.Start(ctx); err != nil {
		return fmt.Errorf("failed to start task: %w", err)
	}

	return nil
}

// StopContainer stops a running container
func (r *ContainerdRuntime) StopContainer(ctx context.Context, containerID string, timeout time.Duration) error {
	ctx = namespaces.WithNamespace(ctx, r.namespace)

	// Get the container
	container, err := r.client.LoadContainer(ctx, containerID)
	if err != nil {
		return fmt.Errorf("failed to load container %s: %w", containerID, err)
	}

	// Get the task
	task, err := container.Task(ctx, nil)
	if err != nil {
		// Task might not exist (container not running)
		return nil
	}

	// Create a context with timeout for graceful shutdown
	stopCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Try graceful shutdown first (SIGTERM)
	if err := task.Kill(stopCtx, syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to kill task: %w", err)
	}

	// Wait for task to exit
	statusC, err := task.Wait(stopCtx)
	if err != nil {
		return fmt.Errorf("failed to wait for task: %w", err)
	}

	// Wait for exit or timeout
	select {
	case <-statusC:
		// Task exited
	case <-stopCtx.Done():
		// Timeout - force kill (SIGKILL)
		if err := task.Kill(ctx, syscall.SIGKILL); err != nil {
			return fmt.Errorf("failed to force kill task: %w", err)
		}
	}

	// Delete the task
	if _, err := task.Delete(ctx); err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	return nil
}

// DeleteContainer removes a container and its snapshot
func (r *ContainerdRuntime) DeleteContainer(ctx context.Context, containerID string) error {
	ctx = namespaces.WithNamespace(ctx, r.namespace)

	// Get the container
	container, err := r.client.LoadContainer(ctx, containerID)
	if err != nil {
		// Container might not exist
		return nil
	}

	// Stop the container first if running
	if err := r.StopContainer(ctx, containerID, 10*time.Second); err != nil {
		// Log but continue with deletion
		fmt.Printf("Warning: failed to stop container before delete: %v\n", err)
	}

	// Delete the container
	if err := container.Delete(ctx, containerd.WithSnapshotCleanup); err != nil {
		return fmt.Errorf("failed to delete container: %w", err)
	}

	return nil
}

// GetContainerStatus returns the status of a container
func (r *ContainerdRuntime) GetContainerStatus(ctx context.Context, containerID string) (types.TaskState, error) {
	ctx = namespaces.WithNamespace(ctx, r.namespace)

	// Get the container
	container, err := r.client.LoadContainer(ctx, containerID)
	if err != nil {
		return types.TaskStateFailed, fmt.Errorf("failed to load container %s: %w", containerID, err)
	}

	// Get the task
	task, err := container.Task(ctx, nil)
	if err != nil {
		// No task means container is not running
		return types.TaskStatePending, nil
	}

	// Get task status
	status, err := task.Status(ctx)
	if err != nil {
		return types.TaskStateFailed, fmt.Errorf("failed to get task status: %w", err)
	}

	// Map containerd status to Warren status
	switch status.Status {
	case containerd.Running:
		return types.TaskStateRunning, nil
	case containerd.Stopped:
		// Check exit code
		if status.ExitStatus == 0 {
			return types.TaskStateComplete, nil
		}
		return types.TaskStateFailed, nil
	case containerd.Paused:
		return types.TaskStateRunning, nil
	default:
		return types.TaskStatePending, nil
	}
}

// GetContainerLogs streams container logs (simplified implementation)
func (r *ContainerdRuntime) GetContainerLogs(ctx context.Context, containerID string) (io.ReadCloser, error) {
	ctx = namespaces.WithNamespace(ctx, r.namespace)

	// Get the container
	container, err := r.client.LoadContainer(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to load container %s: %w", containerID, err)
	}

	// Get the task
	task, err := container.Task(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	// Get logs (this is simplified - real implementation would use cio.LogFile)
	_ = task // Use task for future log implementation

	// For now, return nil (logs implementation deferred)
	return nil, fmt.Errorf("logs not yet implemented")
}

// IsRunning checks if a container is currently running
func (r *ContainerdRuntime) IsRunning(ctx context.Context, containerID string) bool {
	status, err := r.GetContainerStatus(ctx, containerID)
	if err != nil {
		return false
	}
	return status == types.TaskStateRunning
}

// ListContainers returns all containers in the Warren namespace
func (r *ContainerdRuntime) ListContainers(ctx context.Context) ([]string, error) {
	ctx = namespaces.WithNamespace(ctx, r.namespace)

	containers, err := r.client.Containers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	ids := make([]string, 0, len(containers))
	for _, c := range containers {
		ids = append(ids, c.ID())
	}

	return ids, nil
}
