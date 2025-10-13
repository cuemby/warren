/*
Package health provides health check mechanisms for monitoring container health in Warren clusters.

This package implements three types of health checks: HTTP, TCP, and Exec. Health checks
enable automatic detection of unhealthy containers and trigger automatic replacement via
the reconciler, ensuring service availability and reliability without manual intervention.

# Architecture

Warren's health check system follows a modular checker design:

	┌─────────────────────────────────────────────────────────────┐
	│                   Health Check System                       │
	└─────┬──────────────────────────────────────────────────────┘
	      │
	      ▼
	┌──────────────────────────────────────────────────────────────┐
	│                     Checker Interface                        │
	│  • Check(ctx) Result                                         │
	│  • Type() CheckType                                          │
	└────────┬─────────────────────────────────────────────────────┘
	         │
	    ┌────┴──────┬──────────┐
	    ▼           ▼          ▼
	┌────────┐  ┌──────┐  ┌────────┐
	│  HTTP  │  │ TCP  │  │  Exec  │
	│Checker │  │Checker│ │Checker │
	└────────┘  └──────┘  └────────┘
	     │          │          │
	     ▼          ▼          ▼
	  GET /    Connect     Run cmd
	  /health    :port      in container

## Health Check Flow

 1. Task starts → Worker creates health checker
 2. Wait for StartPeriod (grace period for slow apps)
 3. Every Interval: Run health check
 4. If check fails: Increment consecutive failures
 5. If failures >= Retries: Mark task unhealthy
 6. Reconciler detects unhealthy task → Replaces it

# Health Check Types

## HTTP Health Checks

HTTP checks perform HTTP requests to verify application health:

	Check Type: HTTP
	Configuration:
	├── URL: http://container-ip:8080/health
	├── Method: GET, POST, HEAD
	├── Headers: Custom HTTP headers
	├── Expected Status: 200-399 (configurable)
	└── Timeout: 10 seconds

Example responses:
  - 200 OK → Healthy
  - 503 Service Unavailable → Unhealthy
  - Connection timeout → Unhealthy
  - Connection refused → Unhealthy

## TCP Health Checks

TCP checks verify that a port is listening and accepting connections:

	Check Type: TCP
	Configuration:
	├── Address: container-ip:6379
	├── Timeout: 5 seconds
	└── Connection test only (no data sent)

Use cases:
  - Database health (PostgreSQL, MySQL, Redis)
  - Message queue health (RabbitMQ, Kafka)
  - Any service with TCP listener

## Exec Health Checks

Exec checks run commands inside the container and check exit codes:

	Check Type: Exec
	Configuration:
	├── Command: ["pg_isready", "-U", "postgres"]
	├── Timeout: 10 seconds
	├── Exit code 0 → Healthy
	└── Exit code != 0 → Unhealthy

Use cases:
  - Database-specific checks (pg_isready, mysqladmin ping)
  - Custom health scripts
  - File system checks
  - Process checks

# Core Components

## Checker Interface

All health checkers implement this interface:

	type Checker interface {
		Check(ctx context.Context) Result
		Type() CheckType
	}

This allows polymorphic health checking - workers don't need to know the
check type, just call Check() and interpret the Result.

## Result Structure

All checks return a standardized Result:

	type Result struct {
		Healthy   bool          // Check passed?
		Message   string        // Human-readable message
		CheckedAt time.Time     // When check ran
		Duration  time.Duration // How long check took
	}

## Status Tracking

Status tracks health over time:

	type Status struct {
		ConsecutiveFailures  int    // Failure streak
		ConsecutiveSuccesses int    // Success streak
		LastCheck            time.Time
		LastResult           Result
		Healthy              bool   // Current health state
		StartedAt            time.Time
	}

The status implements hysteresis - multiple failures required before marking
unhealthy, preventing flapping from transient issues.

## Configuration

Health checks are configured per service:

	type Config struct {
		Interval    time.Duration  // Time between checks (default: 30s)
		Timeout     time.Duration  // Max check duration (default: 10s)
		Retries     int            // Failures before unhealthy (default: 3)
		StartPeriod time.Duration  // Grace period for slow startup (default: 0)
	}

# Usage Examples

## HTTP Health Check

	import "github.com/cuemby/warren/pkg/health"

	// Create HTTP checker
	checker := health.NewHTTPChecker("http://192.168.1.10:8080/health")

	// Customize (optional)
	checker.WithMethod("GET").
		WithHeader("User-Agent", "Warren-Health/1.0").
		WithStatusRange(200, 299).  // Only 2xx is healthy
		WithTimeout(5 * time.Second)

	// Perform check
	ctx := context.Background()
	result := checker.Check(ctx)

	if result.Healthy {
		fmt.Printf("✓ Healthy: %s (took %v)\n", result.Message, result.Duration)
	} else {
		fmt.Printf("✗ Unhealthy: %s\n", result.Message)
	}

	// Output:
	// ✓ Healthy: HTTP 200 OK (took 12ms)

## TCP Health Check

	// Create TCP checker for Redis
	checker := health.NewTCPChecker("192.168.1.10:6379")
	checker.WithTimeout(3 * time.Second)

	// Check if Redis is listening
	result := checker.Check(ctx)

	if result.Healthy {
		fmt.Println("Redis is accepting connections")
	} else {
		fmt.Printf("Redis unreachable: %s\n", result.Message)
	}

	// Output:
	// Redis is accepting connections

## Exec Health Check

	// Create exec checker for PostgreSQL
	checker := health.NewExecChecker([]string{
		"pg_isready",
		"-U", "postgres",
		"-d", "mydb",
	})
	checker.WithTimeout(5 * time.Second)
	checker.WithContainer("container-abc123")  // Run in this container

	// Check database
	result := checker.Check(ctx)

	if result.Healthy {
		fmt.Println("PostgreSQL is ready")
	} else {
		fmt.Printf("PostgreSQL not ready: %s\n", result.Message)
	}

## Health Status Tracking

	// Create status tracker
	status := health.NewStatus()

	// Configure health check
	config := health.Config{
		Interval:    10 * time.Second,
		Timeout:     5 * time.Second,
		Retries:     3,
		StartPeriod: 30 * time.Second,
	}

	// Simulate health check loop
	checker := health.NewHTTPChecker("http://app:8080/health")

	for {
		// Check if in startup grace period
		if status.InStartPeriod(config) {
			fmt.Println("In startup period, skipping health check")
			time.Sleep(config.Interval)
			continue
		}

		// Run health check
		ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
		result := checker.Check(ctx)
		cancel()

		// Update status
		status.Update(result, config)

		// Check if unhealthy
		if !status.Healthy {
			fmt.Printf("Container unhealthy after %d failures\n",
				status.ConsecutiveFailures)
			// Trigger replacement...
			break
		}

		time.Sleep(config.Interval)
	}

## Service with Health Check

	// Define service with HTTP health check
	service := &types.Service{
		ID:    "svc-api",
		Name:  "api",
		Image: "myapp:v1",
		HealthCheck: &types.HealthCheck{
			Type:     types.HealthCheckTypeHTTP,
			HTTP: &types.HTTPHealthCheck{
				Path:   "/health",
				Port:   8080,
				Scheme: "http",
			},
			Interval:    15 * time.Second,
			Timeout:     5 * time.Second,
			Retries:     3,
			StartPeriod: 60 * time.Second,  // Allow 60s for startup
		},
	}

	// Warren will:
	// 1. Start container
	// 2. Wait 60s (StartPeriod)
	// 3. Check /health every 15s
	// 4. After 3 failures, mark unhealthy
	// 5. Reconciler replaces unhealthy task

# Integration Points

## Worker Integration

Workers manage health check execution:

 1. Task assigned to worker
 2. Worker starts container
 3. Worker creates appropriate health checker
 4. Worker runs checks on configured interval
 5. Worker updates task.HealthStatus via manager
 6. Reconciler reads HealthStatus, triggers replacement if needed

## Reconciler Integration

The reconciler uses health status to detect failures:

	// Check task health
	if task.ActualState == types.TaskStateRunning {
		if task.HealthStatus != nil && !task.HealthStatus.Healthy {
			// Mark task as failed
			task.ActualState = types.TaskStateFailed
			task.Error = fmt.Sprintf("health check failed: %s",
				task.HealthStatus.Message)
			manager.UpdateTask(task)
		}
	}

## Scheduler Integration

The scheduler considers health when placing tasks:

  - Unhealthy tasks don't count toward active replicas
  - Scheduler creates replacement tasks
  - Load balancer excludes unhealthy backends

## Manager Integration

The manager stores health status:

	Task {
		ID:          "task-abc123"
		ActualState: "running"
		HealthStatus: {
			Healthy:             false
			ConsecutiveFailures: 3
			Message:            "HTTP 503 Service Unavailable"
			LastCheck:          "2024-01-15T10:30:00Z"
		}
	}

# Design Patterns

## Strategy Pattern

Different checkers implement the Checker interface:

	Checker (interface)
	├── HTTPChecker (HTTP strategy)
	├── TCPChecker (TCP strategy)
	└── ExecChecker (Exec strategy)

This allows runtime selection of check type without code changes.

## Builder Pattern

Checkers use fluent builders for configuration:

	checker := NewHTTPChecker(url).
		WithMethod("POST").
		WithHeader("Auth", "token").
		WithTimeout(5 * time.Second)

This provides clean, readable configuration with optional parameters.

## Hysteresis Pattern

Status tracking implements hysteresis to prevent flapping:

	Healthy → 1 failure → Still healthy
	Healthy → 2 failures → Still healthy
	Healthy → 3 failures → Unhealthy!

	Unhealthy → 1 success → Healthy!

This prevents oscillation from transient issues while still responding to
persistent problems.

## Context-Based Cancellation

All checks respect context deadlines:

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result := checker.Check(ctx)  // Respects timeout

This enables proper timeout handling and resource cleanup.

# Performance Characteristics

## HTTP Check Performance

HTTP checks are network-bound:

  - Latency: 1-100ms (depends on network + app)
  - Memory: ~10KB per check (HTTP client)
  - CPU: Minimal (mostly waiting for I/O)

For 100 checks/second:
  - ~1% CPU usage
  - ~1MB memory

## TCP Check Performance

TCP checks are very lightweight:

  - Latency: 1-10ms (just TCP handshake)
  - Memory: ~1KB per check
  - CPU: Negligible

TCP checks are ideal for high-frequency monitoring.

## Exec Check Performance

Exec checks are most expensive:

  - Latency: 10-1000ms (depends on command)
  - Memory: Command output size
  - CPU: Command execution

Use exec checks sparingly and increase check interval.

## Recommended Check Intervals

  - HTTP: 10-30 seconds
  - TCP: 5-15 seconds
  - Exec: 30-60 seconds

# Troubleshooting

## False Positive Failures

If healthy containers are marked unhealthy:

1. Check timeout settings:
  - Timeout too short for slow responses?
  - Network latency accounted for?
  - Increase timeout to 2x expected duration

2. Check retry count:
  - Retries = 1 → Very sensitive to transients
  - Retries = 3 → More tolerant (recommended)
  - Increase retries for flaky networks

3. Check StartPeriod:
  - App takes 60s to start but StartPeriod = 10s?
  - Set StartPeriod > app startup time
  - Monitor app startup logs

## Health Checks Not Running

If health checks aren't being performed:

1. Verify configuration:
  - Check service.HealthCheck is set
  - Verify Interval > 0
  - Ensure worker is running

2. Check worker logs:
  - Look for "health check" messages
  - Check for errors creating checker
  - Verify container IP/port reachable

3. Check network connectivity:
  - Can worker reach container IP?
  - Firewall blocking health check port?
  - Container actually listening on port?

## Health Checks Too Slow

If health checks impact performance:

1. Optimize check endpoint:
  - Health check should be lightweight
  - Don't hit database on every check
  - Cache health status if expensive to compute

2. Tune check interval:
  - Reduce check frequency
  - Balance detection speed vs. overhead
  - 30s interval is usually sufficient

3. Use appropriate check type:
  - TCP faster than HTTP
  - HTTP faster than Exec
  - Choose lightest check that's still reliable

## Container Flapping

If containers restart repeatedly:

1. Check application health:
  - Is app actually healthy?
  - Check application logs for errors
  - Test health endpoint manually

2. Tune health check parameters:
  - Increase retries (tolerate transients)
  - Increase interval (reduce check frequency)
  - Increase timeout (allow slower responses)
  - Increase StartPeriod (slow startup)

3. Review health check logic:
  - Is check too strict?
  - Is check testing right thing?
  - Consider application-specific checks

# Monitoring Metrics

Key health check metrics:

  - Health checks performed per second
  - Health check success rate
  - Health check latency (p50, p95, p99)
  - Consecutive failures per task
  - Tasks marked unhealthy per hour

# Best Practices

1. Health Check Design
  - Check critical dependencies (database, cache, etc.)
  - Return quickly (< 1 second ideal)
  - Don't overwhelm backend services
  - Cache expensive computations
  - Return detailed status in response

2. Configuration Tuning
  - Set Interval = 10-30s (balance detection vs. overhead)
  - Set Timeout = 5-10s (2x expected response time)
  - Set Retries = 3 (tolerate transients)
  - Set StartPeriod = 2x app startup time

3. Application Integration
  - Implement /health endpoint in all services
  - Return 200 when healthy, 503 when not
  - Include dependency status in health response
  - Test health endpoint in development

4. HTTP Health Endpoints
  - Keep checks lightweight
  - Don't require authentication
  - Return JSON with status details
  - Include version and uptime
  - Test with curl before deploying

5. Progressive Readiness
  - Use StartPeriod for slow-starting apps
  - Consider separate readiness vs. liveness checks (future)
  - Gradual health restoration (don't kill on first failure)

# Security Considerations

## HTTP Health Checks

  - Health endpoints should not require authentication
  - Don't expose sensitive information in health responses
  - Use internal networks only (not public internet)
  - Rate limit health check endpoints

## Exec Health Checks

  - Validate command arguments (prevent injection)
  - Run commands as non-root user
  - Limit command execution time
  - Monitor for command abuse

# Future Enhancements

Planned health check features:

  - gRPC health checks (gRPC health protocol)
  - Custom health check scripts
  - Readiness vs. liveness checks (Kubernetes-style)
  - Health check metrics export (Prometheus)
  - Dependency health aggregation
  - Circuit breaker integration

# See Also

  - pkg/reconciler - Uses health status for failure detection
  - pkg/worker - Executes health checks on containers
  - pkg/scheduler - Uses health for load balancing decisions
  - docs/health-checks.md - Health check configuration guide
*/
package health
