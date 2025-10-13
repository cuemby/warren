package framework

import (
	"context"
	"fmt"
	"time"
)

// Waiter provides utilities for waiting on conditions with timeouts
type Waiter struct {
	timeout  time.Duration
	interval time.Duration
}

// NewWaiter creates a new Waiter with the given timeout and polling interval
func NewWaiter(timeout, interval time.Duration) *Waiter {
	return &Waiter{
		timeout:  timeout,
		interval: interval,
	}
}

// DefaultWaiter returns a waiter with sensible defaults (30s timeout, 1s interval)
func DefaultWaiter() *Waiter {
	return NewWaiter(30*time.Second, 1*time.Second)
}

// WaitFor waits for a condition to become true
func (w *Waiter) WaitFor(ctx context.Context, condition func() bool, description string) error {
	ctx, cancel := context.WithTimeout(ctx, w.timeout)
	defer cancel()

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// Check immediately
	if condition() {
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for: %s (timeout: %v)", description, w.timeout)
		case <-ticker.C:
			if condition() {
				return nil
			}
		}
	}
}

// WaitForServiceRunning waits for a service to have at least one running task
func (w *Waiter) WaitForServiceRunning(ctx context.Context, client *Client, name string) error {
	return w.WaitFor(ctx, func() bool {
		svc, err := client.Client.GetService(name)
		if err != nil {
			return false
		}

		tasks, err := client.Client.ListTasks(svc.Id, "")
		if err != nil {
			return false
		}

		for _, task := range tasks {
			if task.ActualState == "running" {
				return true
			}
		}
		return false
	}, fmt.Sprintf("service %s to have running tasks", name))
}

// WaitForServiceDeleted waits for a service to be deleted
func (w *Waiter) WaitForServiceDeleted(ctx context.Context, client *Client, name string) error {
	return w.WaitFor(ctx, func() bool {
		_, err := client.Client.GetService(name)
		return err != nil // Service not found means it's deleted
	}, fmt.Sprintf("service %s to be deleted", name))
}

// WaitForReplicas waits for a service to have a specific number of running replicas
func (w *Waiter) WaitForReplicas(ctx context.Context, client *Client, serviceName string, count int) error {
	return w.WaitFor(ctx, func() bool {
		svc, err := client.Client.GetService(serviceName)
		if err != nil {
			return false
		}

		tasks, err := client.Client.ListTasks(svc.Id, "")
		if err != nil {
			return false
		}

		running := 0
		for _, task := range tasks {
			if task.ActualState == "running" {
				running++
			}
		}

		return running == count
	}, fmt.Sprintf("service %s to have %d running replicas", serviceName, count))
}

// WaitForTask waits for a specific task to reach a status
func (w *Waiter) WaitForTask(ctx context.Context, client *Client, taskID string, status string) error {
	return w.WaitFor(ctx, func() bool {
		tasks, err := client.Client.ListTasks("", "")
		if err != nil {
			return false
		}

		for _, task := range tasks {
			if task.Id == taskID {
				return task.ActualState == status
			}
		}

		return false
	}, fmt.Sprintf("task %s to reach status %s", taskID, status))
}

// WaitForTaskRunning waits for a task to be running
func (w *Waiter) WaitForTaskRunning(ctx context.Context, client *Client, taskID string) error {
	return w.WaitForTask(ctx, client, taskID, "running")
}

// WaitForTaskHealthy waits for a task to become healthy
// TODO: Use actual health_status field when added to proto
func (w *Waiter) WaitForTaskHealthy(ctx context.Context, client *Client, taskID string) error {
	return w.WaitFor(ctx, func() bool {
		tasks, err := client.Client.ListTasks("", "")
		if err != nil {
			return false
		}

		for _, task := range tasks {
			if task.Id == taskID {
				// For now, use ActualState as a proxy for health
				// When health_status is added to proto, replace with: task.HealthStatus == "healthy"
				return task.ActualState == "running"
			}
		}

		return false
	}, fmt.Sprintf("task %s to become healthy", taskID))
}

// WaitForLeaderElection waits for a leader to be elected in the cluster
func (w *Waiter) WaitForLeaderElection(ctx context.Context, cluster *Cluster) error {
	return w.WaitFor(ctx, func() bool {
		_, err := cluster.GetLeader()
		return err == nil
	}, "leader election to complete")
}

// WaitForQuorum waits for Raft quorum to be established
func (w *Waiter) WaitForQuorum(ctx context.Context, cluster *Cluster) error {
	return w.WaitFor(ctx, func() bool {
		return cluster.hasQuorum()
	}, "Raft quorum to be established")
}

// WaitForNodeCount waits for a specific number of nodes to join the cluster
func (w *Waiter) WaitForNodeCount(ctx context.Context, client *Client, count int) error {
	return w.WaitFor(ctx, func() bool {
		nodes, err := client.Client.ListNodes()
		if err != nil {
			return false
		}
		return len(nodes) == count
	}, fmt.Sprintf("cluster to have %d nodes", count))
}

// WaitForWorkerNodes waits for a specific number of worker nodes
func (w *Waiter) WaitForWorkerNodes(ctx context.Context, client *Client, count int) error {
	return w.WaitFor(ctx, func() bool {
		nodes, err := client.Client.ListNodes()
		if err != nil {
			return false
		}

		workers := 0
		for _, node := range nodes {
			if node.Role == "worker" {
				workers++
			}
		}

		return workers == count
	}, fmt.Sprintf("cluster to have %d worker nodes", count))
}

// WaitForManagerNodes waits for a specific number of manager nodes
func (w *Waiter) WaitForManagerNodes(ctx context.Context, client *Client, count int) error {
	return w.WaitFor(ctx, func() bool {
		nodes, err := client.Client.ListNodes()
		if err != nil {
			return false
		}

		managers := 0
		for _, node := range nodes {
			if node.Role == "manager" {
				managers++
			}
		}

		return managers == count
	}, fmt.Sprintf("cluster to have %d manager nodes", count))
}

// WaitForClusterHealthy waits for all nodes in the cluster to be healthy
func (w *Waiter) WaitForClusterHealthy(ctx context.Context, client *Client) error {
	return w.WaitFor(ctx, func() bool {
		nodes, err := client.Client.ListNodes()
		if err != nil {
			return false
		}

		for _, node := range nodes {
			if node.Status != "ready" {
				return false
			}
		}

		return len(nodes) > 0
	}, "all cluster nodes to be healthy")
}

// WaitForSecret waits for a secret to exist
func (w *Waiter) WaitForSecret(ctx context.Context, client *Client, name string) error {
	return w.WaitFor(ctx, func() bool {
		_, err := client.Client.GetSecretByName(name)
		return err == nil
	}, fmt.Sprintf("secret %s to exist", name))
}

// WaitForSecretDeleted waits for a secret to be deleted
func (w *Waiter) WaitForSecretDeleted(ctx context.Context, client *Client, name string) error {
	return w.WaitFor(ctx, func() bool {
		_, err := client.Client.GetSecretByName(name)
		return err != nil
	}, fmt.Sprintf("secret %s to be deleted", name))
}

// WaitForVolume waits for a volume to exist
func (w *Waiter) WaitForVolume(ctx context.Context, client *Client, name string) error {
	return w.WaitFor(ctx, func() bool {
		_, err := client.Client.GetVolumeByName(name)
		return err == nil
	}, fmt.Sprintf("volume %s to exist", name))
}

// WaitForVolumeDeleted waits for a volume to be deleted
func (w *Waiter) WaitForVolumeDeleted(ctx context.Context, client *Client, name string) error {
	return w.WaitFor(ctx, func() bool {
		_, err := client.Client.GetVolumeByName(name)
		return err != nil
	}, fmt.Sprintf("volume %s to be deleted", name))
}

// WaitForConditionWithRetry waits for a condition with exponential backoff retry
func (w *Waiter) WaitForConditionWithRetry(ctx context.Context, condition func() (bool, error), description string) error {
	ctx, cancel := context.WithTimeout(ctx, w.timeout)
	defer cancel()

	interval := w.interval
	maxInterval := 10 * time.Second

	for {
		ok, err := condition()
		if err != nil {
			return fmt.Errorf("error checking condition '%s': %w", description, err)
		}

		if ok {
			return nil
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for: %s (timeout: %v)", description, w.timeout)
		case <-time.After(interval):
			// Exponential backoff
			interval = interval * 2
			if interval > maxInterval {
				interval = maxInterval
			}
		}
	}
}

// PollUntil polls a condition until it returns true or context is cancelled
func PollUntil(ctx context.Context, interval time.Duration, condition func() bool) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Check immediately
	if condition() {
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if condition() {
				return nil
			}
		}
	}
}

// PollUntilWithError polls a condition that can return an error
func PollUntilWithError(ctx context.Context, interval time.Duration, condition func() (bool, error)) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Check immediately
	if ok, err := condition(); err != nil {
		return err
	} else if ok {
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if ok, err := condition(); err != nil {
				return err
			} else if ok {
				return nil
			}
		}
	}
}

// Retry retries an operation with exponential backoff
func Retry(ctx context.Context, attempts int, initialDelay time.Duration, operation func() error) error {
	var err error
	delay := initialDelay

	for i := 0; i < attempts; i++ {
		err = operation()
		if err == nil {
			return nil
		}

		if i < attempts-1 {
			select {
			case <-ctx.Done():
				return fmt.Errorf("retry cancelled: %w", ctx.Err())
			case <-time.After(delay):
				delay = delay * 2
			}
		}
	}

	return fmt.Errorf("operation failed after %d attempts: %w", attempts, err)
}
