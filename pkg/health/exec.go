package health

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

// ExecChecker performs exec-based health checks by running a command
type ExecChecker struct {
	// Command is the command to execute (e.g., ["pg_isready", "-U", "postgres"])
	Command []string

	// Timeout is the command execution timeout (default: 10 seconds)
	Timeout time.Duration

	// ContainerID is the ID of the container to exec into
	// If empty, runs on host (useful for testing)
	ContainerID string
}

// NewExecChecker creates a new exec health checker
func NewExecChecker(command []string) *ExecChecker {
	return &ExecChecker{
		Command: command,
		Timeout: 10 * time.Second,
	}
}

// Check performs the exec health check
func (e *ExecChecker) Check(ctx context.Context) Result {
	start := time.Now()

	if len(e.Command) == 0 {
		return Result{
			Healthy:   false,
			Message:   "no command specified",
			CheckedAt: start,
			Duration:  time.Since(start),
		}
	}

	// Create context with timeout
	execCtx, cancel := context.WithTimeout(ctx, e.Timeout)
	defer cancel()

	// Build command
	var cmd *exec.Cmd
	if e.ContainerID != "" {
		// Execute in container using containerd/docker exec
		// For now, we'll use a placeholder - this will be updated when
		// we integrate with containerd runtime
		cmd = exec.CommandContext(execCtx, "echo", "exec-in-container-not-yet-implemented")
	} else {
		// Execute on host (for testing)
		cmd = exec.CommandContext(execCtx, e.Command[0], e.Command[1:]...)
	}

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute command
	err := cmd.Run()

	// Build result message
	message := fmt.Sprintf("Command: %v", e.Command)
	if err != nil {
		// Command failed
		message = fmt.Sprintf("%s, Error: %v", message, err)
		if stderr.Len() > 0 {
			message = fmt.Sprintf("%s, Stderr: %s", message, stderr.String())
		}

		return Result{
			Healthy:   false,
			Message:   message,
			CheckedAt: start,
			Duration:  time.Since(start),
		}
	}

	// Command succeeded (exit code 0)
	if stdout.Len() > 0 {
		// Include output in message (truncated if too long)
		output := stdout.String()
		if len(output) > 100 {
			output = output[:100] + "..."
		}
		message = fmt.Sprintf("%s, Output: %s", message, output)
	}

	return Result{
		Healthy:   true,
		Message:   message,
		CheckedAt: start,
		Duration:  time.Since(start),
	}
}

// Type returns the health check type
func (e *ExecChecker) Type() CheckType {
	return CheckTypeExec
}

// WithTimeout sets the execution timeout
func (e *ExecChecker) WithTimeout(timeout time.Duration) *ExecChecker {
	e.Timeout = timeout
	return e
}

// WithContainer sets the container ID for exec
func (e *ExecChecker) WithContainer(containerID string) *ExecChecker {
	e.ContainerID = containerID
	return e
}
