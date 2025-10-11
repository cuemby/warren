package health

import (
	"context"
	"time"
)

// CheckType represents the type of health check
type CheckType string

const (
	CheckTypeHTTP CheckType = "http"
	CheckTypeTCP  CheckType = "tcp"
	CheckTypeExec CheckType = "exec"
)

// Result represents the outcome of a health check
type Result struct {
	Healthy   bool
	Message   string
	CheckedAt time.Time
	Duration  time.Duration
}

// Checker is the interface that all health checkers must implement
type Checker interface {
	// Check performs the health check and returns the result
	Check(ctx context.Context) Result

	// Type returns the type of health check
	Type() CheckType
}

// Config contains common configuration for all health checks
type Config struct {
	// Interval is the time between health checks
	Interval time.Duration

	// Timeout is the maximum time to wait for a health check to complete
	Timeout time.Duration

	// Retries is the number of consecutive failures before marking as unhealthy
	Retries int

	// StartPeriod is the grace period before starting health checks
	// Used to allow slow-starting containers to initialize
	StartPeriod time.Duration
}

// DefaultConfig returns a Config with sensible defaults
func DefaultConfig() Config {
	return Config{
		Interval:    30 * time.Second,
		Timeout:     10 * time.Second,
		Retries:     3,
		StartPeriod: 0,
	}
}

// Status tracks the current health status of a container
type Status struct {
	// ConsecutiveFailures tracks the number of consecutive failed checks
	ConsecutiveFailures int

	// ConsecutiveSuccesses tracks the number of consecutive successful checks
	ConsecutiveSuccesses int

	// LastCheck is the timestamp of the last health check
	LastCheck time.Time

	// LastResult is the result of the last health check
	LastResult Result

	// Healthy indicates if the container is currently considered healthy
	Healthy bool

	// StartedAt is when health monitoring started for this container
	StartedAt time.Time
}

// NewStatus creates a new Status with default values
func NewStatus() *Status {
	return &Status{
		Healthy:   true, // Assume healthy until proven otherwise
		StartedAt: time.Now(),
	}
}

// Update updates the status based on a new health check result
func (s *Status) Update(result Result, config Config) {
	s.LastCheck = result.CheckedAt
	s.LastResult = result

	if result.Healthy {
		s.ConsecutiveSuccesses++
		s.ConsecutiveFailures = 0

		// Mark as healthy after first success
		s.Healthy = true
	} else {
		s.ConsecutiveFailures++
		s.ConsecutiveSuccesses = 0

		// Mark as unhealthy after reaching retry threshold
		if s.ConsecutiveFailures >= config.Retries {
			s.Healthy = false
		}
	}
}

// InStartPeriod returns true if we're still in the startup grace period
func (s *Status) InStartPeriod(config Config) bool {
	if config.StartPeriod == 0 {
		return false
	}
	return time.Since(s.StartedAt) < config.StartPeriod
}
