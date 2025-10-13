package runtime

import (
	"context"
	"fmt"
	"io"
	"net"
	"os/exec"
	"strings"
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

// CreateContainer creates a container from a container specification
func (r *ContainerdRuntime) CreateContainer(ctx context.Context, container *types.Container) (string, error) {
	ctx = namespaces.WithNamespace(ctx, r.namespace)

	// Get the image
	image, err := r.client.GetImage(ctx, container.Image)
	if err != nil {
		return "", fmt.Errorf("failed to get image %s: %w", container.Image, err)
	}

	// Create container spec with environment variables
	opts := []oci.SpecOpts{
		oci.WithImageConfig(image),
		oci.WithEnv(container.Env),
	}

	// Apply resource limits if specified
	if container.Resources != nil {
		if container.Resources.CPULimit > 0 {
			// Convert CPULimit (cores) to CPU shares and quota
			// CPU shares: relative weight (1024 = 1 core)
			// CPU quota: period=100000 (100ms), quota=CPULimit*100000
			shares := uint64(container.Resources.CPULimit * 1024)
			quota := int64(container.Resources.CPULimit * 100000)
			period := uint64(100000)

			opts = append(opts, oci.WithCPUShares(shares))
			opts = append(opts, oci.WithCPUCFS(quota, period))
		}

		if container.Resources.MemoryLimit > 0 {
			// Apply memory limit in bytes
			opts = append(opts, oci.WithMemoryLimit(uint64(container.Resources.MemoryLimit)))
		}
	}

	// Create the container
	ctrdContainer, err := r.client.NewContainer(
		ctx,
		container.ID,
		containerd.WithImage(image),
		containerd.WithNewSnapshot(container.ID+"-snapshot", image),
		containerd.WithNewSpec(opts...),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	return ctrdContainer.ID(), nil
}

// CreateContainerWithMounts creates a container with secret and volume mounts
func (r *ContainerdRuntime) CreateContainerWithMounts(ctx context.Context, container *types.Container, secretsPath string, volumeMounts []specs.Mount, resolvConfPath string) (string, error) {
	ctx = namespaces.WithNamespace(ctx, r.namespace)

	// Get the image
	image, err := r.client.GetImage(ctx, container.Image)
	if err != nil {
		return "", fmt.Errorf("failed to get image %s: %w", container.Image, err)
	}

	// Create container spec with environment variables
	opts := []oci.SpecOpts{
		oci.WithImageConfig(image),
		oci.WithEnv(container.Env),
	}

	// Apply resource limits if specified
	if container.Resources != nil {
		if container.Resources.CPULimit > 0 {
			// Convert CPULimit (cores) to CPU shares and quota
			// CPU shares: relative weight (1024 = 1 core)
			// CPU quota: period=100000 (100ms), quota=CPULimit*100000
			shares := uint64(container.Resources.CPULimit * 1024)
			quota := int64(container.Resources.CPULimit * 100000)
			period := uint64(100000)

			opts = append(opts, oci.WithCPUShares(shares))
			opts = append(opts, oci.WithCPUCFS(quota, period))
		}

		if container.Resources.MemoryLimit > 0 {
			// Apply memory limit in bytes
			opts = append(opts, oci.WithMemoryLimit(uint64(container.Resources.MemoryLimit)))
		}
	}

	// Collect all mounts
	var mounts []specs.Mount

	// Add bind mount for secrets if provided
	if secretsPath != "" {
		mounts = append(mounts, specs.Mount{
			Source:      secretsPath,
			Destination: "/run/secrets",
			Type:        "bind",
			Options:     []string{"ro", "bind"}, // Read-only bind mount
		})
	}

	// Add volume mounts
	mounts = append(mounts, volumeMounts...)

	// Add DNS configuration (resolv.conf) if provided
	if resolvConfPath != "" {
		mounts = append(mounts, specs.Mount{
			Source:      resolvConfPath,
			Destination: "/etc/resolv.conf",
			Type:        "bind",
			Options:     []string{"ro", "bind"}, // Read-only bind mount
		})
	}

	// Apply all mounts if any
	if len(mounts) > 0 {
		opts = append(opts, oci.WithMounts(mounts))
	}

	// Create the container
	ctrdContainer, err := r.client.NewContainer(
		ctx,
		container.ID,
		containerd.WithImage(image),
		containerd.WithNewSnapshot(container.ID+"-snapshot", image),
		containerd.WithNewSpec(opts...),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	return ctrdContainer.ID(), nil
}

// CreateContainerWithSecrets creates a container with secret tmpfs mounts (deprecated, use CreateContainerWithMounts)
func (r *ContainerdRuntime) CreateContainerWithSecrets(ctx context.Context, container *types.Container, secretsPath string) (string, error) {
	return r.CreateContainerWithMounts(ctx, container, secretsPath, nil, "")
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
func (r *ContainerdRuntime) GetContainerStatus(ctx context.Context, containerID string) (types.ContainerState, error) {
	ctx = namespaces.WithNamespace(ctx, r.namespace)

	// Get the container
	container, err := r.client.LoadContainer(ctx, containerID)
	if err != nil {
		return types.ContainerStateFailed, fmt.Errorf("failed to load container %s: %w", containerID, err)
	}

	// Get the task
	task, err := container.Task(ctx, nil)
	if err != nil {
		// No task means container is not running
		return types.ContainerStatePending, nil
	}

	// Get task status
	status, err := task.Status(ctx)
	if err != nil {
		return types.ContainerStateFailed, fmt.Errorf("failed to get task status: %w", err)
	}

	// Map containerd status to Warren status
	switch status.Status {
	case containerd.Running:
		return types.ContainerStateRunning, nil
	case containerd.Stopped:
		// Check exit code
		if status.ExitStatus == 0 {
			return types.ContainerStateComplete, nil
		}
		return types.ContainerStateFailed, nil
	case containerd.Paused:
		return types.ContainerStateRunning, nil
	default:
		return types.ContainerStatePending, nil
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
	return status == types.ContainerStateRunning
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

// GetContainerIP returns the IP address of a container
func (r *ContainerdRuntime) GetContainerIP(ctx context.Context, containerID string) (string, error) {
	ctx = namespaces.WithNamespace(ctx, r.namespace)

	// Get the container
	container, err := r.client.LoadContainer(ctx, containerID)
	if err != nil {
		return "", fmt.Errorf("failed to load container %s: %w", containerID, err)
	}

	// Get the task
	task, err := container.Task(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get task: %w", err)
	}

	// Get task status to ensure it's running
	status, err := task.Status(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get task status: %w", err)
	}

	if status.Status != containerd.Running {
		return "", fmt.Errorf("container is not running")
	}

	// Get the PID of the container task
	pid := task.Pid()
	if pid == 0 {
		return "", fmt.Errorf("container task has no PID")
	}

	// Use nsenter to execute ip command in the container's network namespace
	// This extracts the IP address from the eth0 interface
	cmd := exec.CommandContext(ctx, "nsenter", "-t", fmt.Sprintf("%d", pid), "-n", "ip", "-4", "addr", "show", "eth0")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get container IP: %w (output: %s)", err, string(output))
	}

	// Parse the output to extract IP address
	// Example output:
	// 2: eth0@if3: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc noqueue state UP group default
	//     inet 10.88.0.2/16 brd 10.88.255.255 scope global eth0
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "inet ") {
			// Extract IP from "inet 10.88.0.2/16 brd ..."
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				// Remove CIDR notation (e.g., "10.88.0.2/16" -> "10.88.0.2")
				ipWithCIDR := parts[1]
				ip, _, err := net.ParseCIDR(ipWithCIDR)
				if err != nil {
					return "", fmt.Errorf("failed to parse IP address %s: %w", ipWithCIDR, err)
				}
				return ip.String(), nil
			}
		}
	}

	return "", fmt.Errorf("no IP address found for container")
}
