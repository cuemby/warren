package health

import (
	"context"
	"fmt"
	"net"
	"time"
)

// TCPChecker performs TCP-based health checks
type TCPChecker struct {
	// Address is the TCP address to connect to (e.g., "container-ip:6379")
	Address string

	// Timeout is the connection timeout (default: 5 seconds)
	Timeout time.Duration
}

// NewTCPChecker creates a new TCP health checker
func NewTCPChecker(address string) *TCPChecker {
	return &TCPChecker{
		Address: address,
		Timeout: 5 * time.Second,
	}
}

// Check performs the TCP health check
func (t *TCPChecker) Check(ctx context.Context) Result {
	start := time.Now()

	// Create a dialer with timeout
	dialer := &net.Dialer{
		Timeout: t.Timeout,
	}

	// Attempt to connect
	conn, err := dialer.DialContext(ctx, "tcp", t.Address)
	if err != nil {
		return Result{
			Healthy:   false,
			Message:   fmt.Sprintf("connection failed: %v", err),
			CheckedAt: start,
			Duration:  time.Since(start),
		}
	}
	defer conn.Close()

	// Connection successful
	return Result{
		Healthy:   true,
		Message:   fmt.Sprintf("TCP connection to %s successful", t.Address),
		CheckedAt: start,
		Duration:  time.Since(start),
	}
}

// Type returns the health check type
func (t *TCPChecker) Type() CheckType {
	return CheckTypeTCP
}

// WithTimeout sets the connection timeout
func (t *TCPChecker) WithTimeout(timeout time.Duration) *TCPChecker {
	t.Timeout = timeout
	return t
}
