package framework

import (
	"context"
	"strings"
	"time"
)

// Assertions provides test assertion helpers
type Assertions struct {
	t TestingT
}

// NewAssertions creates a new Assertions instance
func NewAssertions(t TestingT) *Assertions {
	return &Assertions{t: t}
}

// ServiceExists asserts that a service exists
func (a *Assertions) ServiceExists(name string, client *Client) {
	a.t.Helper()

	svc, err := client.Client.GetService(name)
	if err != nil {
		a.t.Fatalf("Service %s does not exist: %v", name, err)
	}

	if svc == nil {
		a.t.Fatalf("Service %s is nil", name)
	}
}

// ServiceRunning asserts that a service has at least one running task
func (a *Assertions) ServiceRunning(name string, client *Client) {
	a.t.Helper()

	svc, err := client.Client.GetService(name)
	if err != nil {
		a.t.Fatalf("Failed to get service %s: %v", name, err)
	}

	// Check via tasks (Service proto has no status field)
	tasks, err := client.Client.ListTasks(svc.Id, "")
	if err != nil {
		a.t.Fatalf("Failed to list tasks for service %s: %v", name, err)
	}

	for _, task := range tasks {
		if task.ActualState == "running" {
			return // At least one task running
		}
	}

	a.t.Fatalf("Service %s has no running tasks", name)
}

// ServiceReplicas asserts that a service has the expected number of running replicas
func (a *Assertions) ServiceReplicas(name string, expected int, client *Client) {
	a.t.Helper()

	svc, err := client.Client.GetService(name)
	if err != nil {
		a.t.Fatalf("Failed to get service %s: %v", name, err)
	}

	tasks, err := client.Client.ListTasks(svc.Id, "")
	if err != nil {
		a.t.Fatalf("Failed to list tasks for service %s: %v", name, err)
	}

	running := 0
	for _, task := range tasks {
		if task.ActualState == "running" {
			running++
		}
	}

	if running != expected {
		a.t.Fatalf("Service %s has %d running replicas, expected %d", name, running, expected)
	}
}

// ServiceDeleted asserts that a service no longer exists
func (a *Assertions) ServiceDeleted(name string, client *Client) {
	a.t.Helper()

	_, err := client.Client.GetService(name)
	if err == nil {
		a.t.Fatalf("Service %s still exists, expected it to be deleted", name)
	}

	// Check if error indicates service not found
	if !strings.Contains(err.Error(), "not found") {
		a.t.Fatalf("Unexpected error checking service %s: %v", name, err)
	}
}

// TaskRunning asserts that a task is running
func (a *Assertions) TaskRunning(taskID string, client *Client) {
	a.t.Helper()

	tasks, err := client.Client.ListTasks("", "")
	if err != nil {
		a.t.Fatalf("Failed to list tasks: %v", err)
	}

	for _, task := range tasks {
		if task.Id == taskID {
			if task.ActualState != "running" {
				a.t.Fatalf("Task %s is not running (state: %s)", taskID, task.ActualState)
			}
			return
		}
	}

	a.t.Fatalf("Task %s not found", taskID)
}

// TaskHealthy asserts that a task is healthy
// TODO: Task proto doesn't have health_status field yet - using actual_state as proxy
func (a *Assertions) TaskHealthy(taskID string, client *Client) {
	a.t.Helper()

	tasks, err := client.Client.ListTasks("", "")
	if err != nil {
		a.t.Fatalf("Failed to list tasks: %v", err)
	}

	for _, task := range tasks {
		if task.Id == taskID {
			// For now, check if task is running as proxy for healthy
			// TODO: Use actual health_status when added to proto
			if task.ActualState != "running" {
				a.t.Fatalf("Task %s is not healthy (state: %s)", taskID, task.ActualState)
			}
			return
		}
	}

	a.t.Fatalf("Task %s not found", taskID)
}

// HasLeader asserts that the cluster has a leader
func (a *Assertions) HasLeader(cluster *Cluster) {
	a.t.Helper()

	leader, err := cluster.GetLeader()
	if err != nil {
		a.t.Fatalf("Cluster has no leader: %v", err)
	}

	if leader == nil {
		a.t.Fatalf("Leader is nil")
	}
}

// QuorumSize asserts that the cluster has the expected quorum size
func (a *Assertions) QuorumSize(expected int, cluster *Cluster) {
	a.t.Helper()

	// Simply check the number of managers in the cluster
	if len(cluster.Managers) != expected {
		a.t.Fatalf("Cluster has %d managers, expected %d", len(cluster.Managers), expected)
	}
}

// NodeCount asserts that the cluster has the expected number of nodes
func (a *Assertions) NodeCount(expected int, client *Client) {
	a.t.Helper()

	nodes, err := client.Client.ListNodes()
	if err != nil {
		a.t.Fatalf("Failed to list nodes: %v", err)
	}

	if len(nodes) != expected {
		a.t.Fatalf("Cluster has %d nodes, expected %d", len(nodes), expected)
	}
}

// NodeRole asserts that a node has a specific role
func (a *Assertions) NodeRole(nodeID, expectedRole string, client *Client) {
	a.t.Helper()

	node, err := client.Client.GetNode(nodeID)
	if err != nil {
		a.t.Fatalf("Failed to get node %s: %v", nodeID, err)
	}

	if node.Role != expectedRole {
		a.t.Fatalf("Node %s has role %s, expected %s", nodeID, node.Role, expectedRole)
	}
}

// Eventually repeatedly runs a condition until it returns true or timeout occurs
func (a *Assertions) Eventually(condition func() bool, timeout, interval time.Duration, msg string) {
	a.t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			a.t.Fatalf("Timeout waiting for condition: %s (timeout: %v)", msg, timeout)
		case <-ticker.C:
			if condition() {
				return
			}
		}
	}
}

// EventuallyWithContext is like Eventually but uses a provided context
func (a *Assertions) EventuallyWithContext(ctx context.Context, condition func() bool, interval time.Duration, msg string) {
	a.t.Helper()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			a.t.Fatalf("Context cancelled waiting for condition: %s (error: %v)", msg, ctx.Err())
		case <-ticker.C:
			if condition() {
				return
			}
		}
	}
}

// NoError asserts that the error is nil
func (a *Assertions) NoError(err error, msg string) {
	a.t.Helper()

	if err != nil {
		a.t.Fatalf("%s: %v", msg, err)
	}
}

// Error asserts that the error is not nil
func (a *Assertions) Error(err error, msg string) {
	a.t.Helper()

	if err == nil {
		a.t.Fatalf("%s: expected error but got nil", msg)
	}
}

// Equal asserts that two values are equal
func (a *Assertions) Equal(expected, actual interface{}, msg string) {
	a.t.Helper()

	if expected != actual {
		a.t.Fatalf("%s: expected %v, got %v", msg, expected, actual)
	}
}

// NotEqual asserts that two values are not equal
func (a *Assertions) NotEqual(expected, actual interface{}, msg string) {
	a.t.Helper()

	if expected == actual {
		a.t.Fatalf("%s: expected values to be different, but both are %v", msg, expected)
	}
}

// True asserts that a condition is true
func (a *Assertions) True(condition bool, msg string) {
	a.t.Helper()

	if !condition {
		a.t.Fatalf("%s: expected true, got false", msg)
	}
}

// False asserts that a condition is false
func (a *Assertions) False(condition bool, msg string) {
	a.t.Helper()

	if condition {
		a.t.Fatalf("%s: expected false, got true", msg)
	}
}

// Contains asserts that a string contains a substring
func (a *Assertions) Contains(haystack, needle, msg string) {
	a.t.Helper()

	if !strings.Contains(haystack, needle) {
		a.t.Fatalf("%s: expected %q to contain %q", msg, haystack, needle)
	}
}

// NotContains asserts that a string does not contain a substring
func (a *Assertions) NotContains(haystack, needle, msg string) {
	a.t.Helper()

	if strings.Contains(haystack, needle) {
		a.t.Fatalf("%s: expected %q not to contain %q", msg, haystack, needle)
	}
}

// Len asserts that a slice or map has a specific length
func (a *Assertions) Len(obj interface{}, expected int, msg string) {
	a.t.Helper()

	var length int

	switch v := obj.(type) {
	case []interface{}:
		length = len(v)
	case map[string]interface{}:
		length = len(v)
	case string:
		length = len(v)
	default:
		a.t.Fatalf("%s: unsupported type for Len assertion: %T", msg, obj)
		return
	}

	if length != expected {
		a.t.Fatalf("%s: expected length %d, got %d", msg, expected, length)
	}
}

// Nil asserts that a value is nil
func (a *Assertions) Nil(obj interface{}, msg string) {
	a.t.Helper()

	if obj != nil {
		a.t.Fatalf("%s: expected nil, got %v", msg, obj)
	}
}

// NotNil asserts that a value is not nil
func (a *Assertions) NotNil(obj interface{}, msg string) {
	a.t.Helper()

	if obj == nil {
		a.t.Fatalf("%s: expected non-nil value", msg)
	}
}

// Logf logs a formatted message (non-failing)
func (a *Assertions) Logf(format string, args ...interface{}) {
	a.t.Helper()
	a.t.Logf(format, args...)
}

// Log logs a message (non-failing)
func (a *Assertions) Log(msg string) {
	a.t.Helper()
	a.t.Logf("%s", msg)
}

// Step logs a test step (for visibility in test output)
func (a *Assertions) Step(step string) {
	a.t.Helper()
	a.t.Logf("\n==> %s", step)
}

// Success logs a success message
func (a *Assertions) Success(msg string) {
	a.t.Helper()
	a.t.Logf("✓ %s", msg)
}

// Info logs an informational message
func (a *Assertions) Info(msg string) {
	a.t.Helper()
	a.t.Logf("ℹ %s", msg)
}

// Warning logs a warning message
func (a *Assertions) Warning(msg string) {
	a.t.Helper()
	a.t.Logf("⚠ %s", msg)
}

// Errorf logs an error and fails the test
func (a *Assertions) Errorf(format string, args ...interface{}) {
	a.t.Helper()
	a.t.Errorf(format, args...)
}

// Fatalf logs a fatal error and stops the test immediately
func (a *Assertions) Fatalf(format string, args ...interface{}) {
	a.t.Helper()
	a.t.Fatalf(format, args...)
}

// FailNow fails the test immediately without logging
func (a *Assertions) FailNow() {
	a.t.Helper()
	a.t.FailNow()
}

// Fail marks the test as failed but continues execution
func (a *Assertions) Fail(msg string) {
	a.t.Helper()
	a.t.Errorf("Test failed: %s", msg)
}
