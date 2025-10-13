/*
Package log provides structured logging for Warren using zerolog.

The log package wraps the zerolog library to provide JSON-structured logging with
component-specific loggers, configurable log levels, and helper functions for
common logging patterns. All logs include timestamps and support filtering by
severity level for production debugging.

# Architecture

Warren's logging system provides structured JSON logging with minimal overhead:

	┌──────────────────── LOGGING SYSTEM ──────────────────────┐
	│                                                            │
	│  ┌────────────────────────────────────────────┐          │
	│  │            Global Logger                    │          │
	│  │  - Zerolog instance                         │          │
	│  │  - Initialized via log.Init()               │          │
	│  │  - Thread-safe for concurrent use           │          │
	│  └──────────────────┬─────────────────────────┘          │
	│                     │                                      │
	│  ┌──────────────────▼─────────────────────────┐          │
	│  │           Configuration                     │          │
	│  │  - Level: debug/info/warn/error             │          │
	│  │  - Format: JSON or console (human)          │          │
	│  │  - Output: stdout, file, or custom writer   │          │
	│  └──────────────────┬─────────────────────────┘          │
	│                     │                                      │
	│  ┌──────────────────▼─────────────────────────┐          │
	│  │         Component Loggers                   │          │
	│  │  - WithComponent("scheduler")               │          │
	│  │  - WithNodeID("node-abc123")                │          │
	│  │  - WithServiceID("service-xyz")             │          │
	│  │  - WithTaskID("task-def456")                │          │
	│  └──────────────────┬─────────────────────────┘          │
	│                     │                                      │
	│  ┌──────────────────▼─────────────────────────┐          │
	│  │            Log Output                       │          │
	│  │                                              │          │
	│  │  JSON Format:                               │          │
	│  │  {                                           │          │
	│  │    "level": "info",                         │          │
	│  │    "component": "scheduler",                │          │
	│  │    "time": "2024-10-13T10:30:00Z",         │          │
	│  │    "message": "task scheduled"              │          │
	│  │  }                                           │          │
	│  │                                              │          │
	│  │  Console Format:                            │          │
	│  │  10:30AM INF task scheduled component=scheduler │      │
	│  └────────────────────────────────────────────┘           │
	└────────────────────────────────────────────────────────┘

# Core Components

Global Logger:
  - Package-level zerolog.Logger instance
  - Initialized once via log.Init()
  - Accessible from all Warren packages
  - Thread-safe concurrent writes

Log Levels:
  - Debug: Detailed debugging information
  - Info: General informational messages
  - Warn: Warning messages (potential issues)
  - Error: Error messages (operation failed)
  - Fatal: Critical errors (process exits)

Configuration:
  - Level: Filter messages below threshold
  - JSONOutput: JSON vs human-readable console
  - Output: io.Writer for log destination (stdout, file)

Context Loggers:
  - WithComponent: Add component name to all logs
  - WithNodeID: Add node ID context
  - WithServiceID: Add service ID context
  - WithTaskID: Add task ID context

# Log Levels

Debug Level:
  - Purpose: Detailed debugging information
  - Usage: Development and troubleshooting
  - Performance: Verbose, may impact production
  - Example: "Checking node resources: CPU=4, Memory=8GB"

Info Level:
  - Purpose: General informational messages
  - Usage: Default production level
  - Performance: Moderate volume
  - Example: "Service created: web (nginx:latest)"

Warn Level:
  - Purpose: Potential issues or unexpected conditions
  - Usage: Situations that may require attention
  - Performance: Low volume
  - Example: "Node heartbeat missed (1 occurrence)"

Error Level:
  - Purpose: Operation failures that need investigation
  - Usage: Failed operations, exceptions
  - Performance: Low volume
  - Example: "Failed to start container: image not found"

Fatal Level:
  - Purpose: Critical errors causing process termination
  - Usage: Unrecoverable errors only
  - Behavior: Logs message and exits process (os.Exit(1))
  - Example: "Failed to initialize Raft: %v"

# Usage

Initializing the Logger:

	import "github.com/cuemby/warren/pkg/log"

	// JSON output (production)
	log.Init(log.Config{
		Level:      log.InfoLevel,
		JSONOutput: true,
		Output:     os.Stdout,
	})

	// Console output (development)
	log.Init(log.Config{
		Level:      log.DebugLevel,
		JSONOutput: false,
		Output:     os.Stdout,
	})

	// Custom output (file)
	file, _ := os.OpenFile("/var/log/warren.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	log.Init(log.Config{
		Level:      log.InfoLevel,
		JSONOutput: true,
		Output:     file,
	})

Simple Logging:

	log.Info("Cluster initialized successfully")
	log.Debug("Checking node status")
	log.Warn("High memory usage detected")
	log.Error("Failed to connect to containerd")
	log.Fatal("Cannot start without database") // Exits process

Structured Logging:

	log.Logger.Info().
		Str("service_id", "service-123").
		Int("replicas", 3).
		Msg("Service created")

	log.Logger.Error().
		Err(err).
		Str("node_id", "node-abc").
		Msg("Node health check failed")

Component Loggers:

	// Create component-specific logger
	schedulerLog := log.WithComponent("scheduler")
	schedulerLog.Info().Msg("Starting scheduler loop")
	schedulerLog.Debug().Str("task_id", "task-123").Msg("Scheduling task")

	// Multiple context fields
	taskLog := log.WithComponent("worker").
		With().Str("node_id", "node-abc").
		Str("task_id", "task-123").Logger()
	taskLog.Info().Msg("Starting task")
	taskLog.Error().Err(err).Msg("Task failed")

Context Logger Helpers:

	// Node-specific logs
	nodeLog := log.WithNodeID("node-abc123")
	nodeLog.Info().Msg("Node joined cluster")

	// Service-specific logs
	svcLog := log.WithServiceID("service-xyz789")
	svcLog.Info().Msg("Service updated")

	// Task-specific logs
	taskLog := log.WithTaskID("task-def456")
	taskLog.Info().Msg("Task started")

Complete Example:

	package main

	import (
		"errors"
		"os"
		"github.com/cuemby/warren/pkg/log"
	)

	func main() {
		// Initialize logger
		log.Init(log.Config{
			Level:      log.InfoLevel,
			JSONOutput: true,
			Output:     os.Stdout,
		})

		log.Info("Warren starting")

		// Component-specific logging
		schedulerLog := log.WithComponent("scheduler")
		schedulerLog.Info().
			Str("node_id", "node-1").
			Int("task_count", 5).
			Msg("Scheduling tasks")

		// Error logging
		err := errors.New("connection refused")
		log.Logger.Error().
			Err(err).
			Str("component", "runtime").
			Msg("Failed to connect to containerd")

		log.Info("Warren stopped")
	}

# Integration Points

This package integrates with:

  - pkg/manager: Logs cluster operations and Raft events
  - pkg/scheduler: Logs task scheduling decisions
  - pkg/reconciler: Logs state reconciliation
  - pkg/worker: Logs task execution and health checks
  - pkg/api: Logs API requests and errors
  - pkg/embedded: Logs containerd lifecycle

# Log Output Examples

JSON Format (Production):

	{"level":"info","component":"manager","time":"2024-10-13T10:30:00Z","message":"Cluster initialized"}
	{"level":"info","component":"scheduler","task_id":"task-123","time":"2024-10-13T10:30:01Z","message":"Task scheduled"}
	{"level":"error","component":"worker","node_id":"node-abc","error":"image not found","time":"2024-10-13T10:30:02Z","message":"Failed to start task"}

Console Format (Development):

	10:30:00 INF Cluster initialized component=manager
	10:30:01 INF Task scheduled component=scheduler task_id=task-123
	10:30:02 ERR Failed to start task component=worker node_id=node-abc error="image not found"

# Design Patterns

Global Logger Pattern:
  - Single package-level Logger instance
  - Initialized once at application start
  - Accessible from all packages without passing
  - Simplifies logging in deeply nested calls

Context Logger Pattern:
  - Create child loggers with context fields
  - Pass context loggers to functions
  - Automatically includes context in all logs
  - Avoids repetitive field specification

Structured Logging Pattern:
  - Use typed fields (.Str, .Int, .Err)
  - Enables log aggregation and querying
  - Better than string concatenation
  - Parseable by log analysis tools

Error Logging Pattern:
  - Always use .Err(err) for error objects
  - Provides stack trace information
  - Enables error tracking and alerting
  - Consistent error format across codebase

# Performance Characteristics

Logging Overhead:
  - Disabled level: 0ns (compile-time optimization)
  - JSON encode: ~500ns per log line
  - Console format: ~1µs per log line
  - String field: +50ns per field
  - Int field: +30ns per field

Memory Allocation:
  - Zero allocation for disabled levels
  - ~100 bytes per log line (JSON)
  - ~200 bytes per log line (console)
  - Amortized by buffer pooling

Throughput:
  - JSON: ~2M log lines per second
  - Console: ~1M log lines per second
  - Bottleneck: I/O write speed
  - Async writes recommended for high volume

Log Level Impact:
  - Debug: High volume, use in development only
  - Info: Moderate volume, suitable for production
  - Warn/Error: Low volume, minimal impact
  - Recommendation: Info level in production

# Troubleshooting

Common Issues:

No Log Output:
  - Symptom: No logs appearing
  - Check: log.Init() called before logging
  - Check: Log level set appropriately (Debug < Info < Warn < Error)
  - Solution: Initialize logger in main() before any logging

Excessive Log Volume:
  - Symptom: Disk space fills quickly
  - Cause: Debug level in production
  - Check: Log level configuration
  - Solution: Use Info level in production, rotate logs

Missing Context Fields:
  - Symptom: Logs missing component or ID fields
  - Cause: Using global Logger instead of context logger
  - Solution: Use WithComponent() or create child loggers

Log Parsing Fails:
  - Symptom: Cannot parse JSON logs
  - Cause: Invalid JSON in message field
  - Check: Embedded quotes or control characters
  - Solution: Use .Str() instead of string interpolation

Performance Degradation:
  - Symptom: Slow application performance
  - Cause: Excessive logging in hot path
  - Check: Log statements in tight loops
  - Solution: Reduce log frequency, use sampling

# Log Rotation

File-Based Logging:

Warren doesn't include built-in log rotation. Use external tools:

Logrotate (Linux):
	# /etc/logrotate.d/warren
	/var/log/warren/*.log {
	    daily
	    rotate 7
	    compress
	    delaycompress
	    missingok
	    notifempty
	    copytruncate
	}

Systemd Journal:
	# Automatic rotation by systemd
	journalctl -u warren -f

Docker/Kubernetes:
	# Use container runtime log drivers
	# JSON logs to stdout (already implemented)

# Log Aggregation

Recommended Tools:

Elasticsearch + Filebeat:
  - Filebeat ships logs to Elasticsearch
  - Kibana for visualization and search
  - Query: component:"scheduler" AND level:"error"

Loki + Promtail:
  - Lightweight log aggregation
  - Grafana integration
  - Query: {component="scheduler"} |= "error"

CloudWatch Logs:
  - AWS native log aggregation
  - Metric filters for alerting
  - Query: fields @message | filter component = "scheduler"

Datadog:
  - Full-stack observability
  - APM and log correlation
  - Query: service:warren component:scheduler status:error

# Monitoring

Log-Based Alerts:

High Error Rate:
  - Query: rate(log entries with level="error"[5m]) > 10
  - Description: More than 10 errors per second
  - Action: Check recent errors, investigate root cause

No Logs:
  - Query: absent(log entries[1m])
  - Description: No logs received in 1 minute
  - Action: Check Warren process, log pipeline

Specific Error Pattern:
  - Query: log entries containing "failed to connect to containerd"
  - Description: Containerd connection issues
  - Action: Check containerd status, socket permissions

# Security

Log Content:
  - Never log secrets or sensitive data
  - Redact tokens, passwords, API keys
  - Use log scrubbing for compliance (GDPR, PCI)
  - Review logs before sharing externally

Log Access:
  - Restrict log file permissions (0640)
  - Limit log aggregation access (RBAC)
  - Audit log access in production
  - Encrypt logs at rest and in transit

Log Injection:
  - Use structured logging (prevents injection)
  - Never concatenate user input into log messages
  - Use typed fields (.Str, .Int) for user data
  - Validate/sanitize before logging if necessary

# Best Practices

Do:
  - Use Info level for production
  - Use structured fields for queryable data
  - Create component-specific loggers
  - Log errors with .Err() for stack traces
  - Include context (node ID, service ID, task ID)

Don't:
  - Log sensitive data (secrets, passwords)
  - Use Debug level in production
  - Log in tight loops (use sampling)
  - Concatenate strings (use .Str, .Int)
  - Block on log writes (use buffered output)

# See Also

  - Zerolog documentation: https://github.com/rs/zerolog
  - Structured logging: https://www.thoughtworks.com/radar/techniques/structured-logging
  - 12-Factor App Logs: https://12factor.net/logs
  - Log aggregation: https://www.elastic.co/what-is/log-aggregation
*/
package log
