# Warren Logging Guide

Warren uses [zerolog](https://github.com/rs/zerolog) for structured, high-performance logging. This guide covers best practices for using logging throughout the Warren codebase.

## Table of Contents

- [Quick Start](#quick-start)
- [Log Levels](#log-levels)
- [Structured Fields](#structured-fields)
- [Context Loggers](#context-loggers)
- [Best Practices](#best-practices)
- [Examples](#examples)
- [Configuration](#configuration)

## Quick Start

```go
import "github.com/cuemby/warren/pkg/log"

// Simple logging
log.Info("Cluster initialized")
log.Error("Failed to start service")

// Structured logging with fields
log.Logger.Info().
    Str("service", "nginx").
    Int("replicas", 3).
    Msg("Service created")

// Error logging with error object
log.Logger.Error().
    Err(err).
    Str("service_id", serviceID).
    Msg("Failed to create service")
```

## Log Levels

Warren supports four log levels, from most to least verbose:

| Level | When to Use | Example |
|-------|-------------|---------|
| **Debug** | Development, detailed troubleshooting | Internal state changes, loop iterations |
| **Info** | Normal operations, important events | Service created, node joined, reconciliation completed |
| **Warn** | Recoverable errors, degraded performance | Retry attempts, slow operations, deprecated features |
| **Error** | Failures requiring attention | Service creation failed, API errors, data corruption |

### Choosing the Right Level

**Debug**: Verbose information useful during development
```go
log.Logger.Debug().
    Str("task_id", task.ID).
    Str("state", string(task.ActualState)).
    Msg("Task state transition")
```

**Info**: Normal operational messages
```go
log.Logger.Info().
    Str("node_id", nodeID).
    Str("role", "manager").
    Msg("Node joined cluster")
```

**Warn**: Something unexpected but handled
```go
log.Logger.Warn().
    Str("service", serviceName).
    Dur("latency", latency).
    Msg("Service creation took longer than expected")
```

**Error**: Failures that need investigation
```go
log.Logger.Error().
    Err(err).
    Str("service_id", service.ID).
    Msg("Failed to schedule tasks")
```

## Structured Fields

Always use structured fields instead of string formatting. This enables better querying and analysis.

### ❌ Bad: String Formatting
```go
log.Info(fmt.Sprintf("Created service %s with %d replicas", name, replicas))
```

### ✅ Good: Structured Fields
```go
log.Logger.Info().
    Str("service", name).
    Int("replicas", replicas).
    Msg("Service created")
```

### Common Field Types

```go
// Strings
Str("component", "scheduler")
Str("node_id", nodeID)

// Numbers
Int("replicas", 3)
Int32("port", 8080)
Float64("cpu_usage", 0.75)

// Booleans
Bool("leader", true)
Bool("healthy", false)

// Durations
Dur("latency", duration)
Dur("timeout", 30*time.Second)

// Times
Time("created_at", time.Now())
Time("last_heartbeat", node.LastHeartbeat)

// Errors
Err(err)

// Objects (marshaled to JSON)
Interface("config", config)
```

## Context Loggers

Create child loggers with persistent context for related operations:

### Component Logger
```go
// Create once at component initialization
logger := log.WithComponent("scheduler")

// Use throughout the component
logger.Info().
    Str("task_id", taskID).
    Msg("Task scheduled")

logger.Debug().
    Int("pending_tasks", len(queue)).
    Msg("Processing task queue")
```

### Request-Scoped Logger
```go
// Create per-request with relevant context
logger := log.Logger.With().
    Str("request_id", requestID).
    Str("user", userID).
    Logger()

logger.Info().Msg("Processing request")
// ... handle request ...
logger.Info().Dur("duration", elapsed).Msg("Request completed")
```

### Multi-Field Context
```go
// Combine multiple context fields
logger := log.Logger.With().
    Str("component", "reconciler").
    Str("node_id", nodeID).
    Logger()

// All subsequent logs include component and node_id
logger.Info().Msg("Starting reconciliation")
logger.Debug().Int("tasks", len(tasks)).Msg("Reconciling tasks")
```

## Best Practices

### 1. Use Descriptive Messages

Messages should be clear and actionable without the fields:

**❌ Bad**
```go
log.Logger.Info().Str("s", svc).Msg("done")
```

**✅ Good**
```go
log.Logger.Info().
    Str("service", svc.Name).
    Msg("Service creation completed")
```

### 2. Include Relevant Context

Always include IDs and context needed for debugging:

```go
log.Logger.Error().
    Err(err).
    Str("service_id", serviceID).
    Str("task_id", taskID).
    Str("node_id", nodeID).
    Msg("Task failed to start")
```

### 3. Log Errors Properly

Always include the error object and context:

**❌ Bad**
```go
log.Error("Error occurred")
```

**✅ Good**
```go
log.Logger.Error().
    Err(err).
    Str("operation", "CreateService").
    Str("service", name).
    Msg("Service creation failed")
```

### 4. Avoid Logging in Hot Paths

Skip expensive logging in performance-critical code:

```go
// Use debug level for hot paths
if log.Logger.GetLevel() == zerolog.DebugLevel {
    log.Logger.Debug().
        Int("task_count", len(tasks)).
        Msg("Processing task batch")
}
```

### 5. Use Events for Metrics, Logs for Debugging

Don't log what should be a metric:

**❌ Bad**
```go
log.Logger.Info().Msg("Task scheduled") // For every task!
```

**✅ Good**
```go
metrics.TasksScheduled.Inc()  // Use metrics counter
log.Logger.Debug().
    Str("task_id", taskID).
    Msg("Task scheduled")  // Debug only
```

### 6. Log State Transitions

Important state changes should always be logged:

```go
log.Logger.Info().
    Str("node_id", node.ID).
    Str("old_status", string(oldStatus)).
    Str("new_status", string(newStatus)).
    Msg("Node status changed")
```

### 7. Don't Log Secrets

Never log sensitive information:

**❌ Bad**
```go
log.Logger.Debug().
    Str("password", password).
    Str("token", apiToken).
    Msg("Authenticating")
```

**✅ Good**
```go
log.Logger.Debug().
    Str("username", username).
    Bool("authenticated", true).
    Msg("Authentication successful")
```

## Examples

### Service Creation

```go
func (s *Server) CreateService(ctx context.Context, req *proto.CreateServiceRequest) (*proto.CreateServiceResponse, error) {
    logger := log.Logger.With().
        Str("component", "api").
        Str("operation", "CreateService").
        Str("service", req.Name).
        Logger()

    logger.Info().
        Int32("replicas", req.Replicas).
        Str("image", req.Image).
        Msg("Creating service")

    service := &types.Service{
        ID:    uuid.New().String(),
        Name:  req.Name,
        Image: req.Image,
        Replicas: int(req.Replicas),
    }

    if err := s.manager.CreateService(service); err != nil {
        logger.Error().
            Err(err).
            Str("service_id", service.ID).
            Msg("Failed to create service")
        return nil, err
    }

    logger.Info().
        Str("service_id", service.ID).
        Msg("Service created successfully")

    return &proto.CreateServiceResponse{
        Service: serviceToProto(service),
    }, nil
}
```

### Reconciliation Loop

```go
func (r *Reconciler) reconcile() error {
    timer := metrics.NewTimer()
    logger := log.WithComponent("reconciler")

    logger.Debug().Msg("Starting reconciliation cycle")

    // Reconcile nodes
    if err := r.reconcileNodes(); err != nil {
        logger.Warn().
            Err(err).
            Msg("Failed to reconcile nodes")
    }

    // Reconcile tasks
    if err := r.reconcileTasks(); err != nil {
        logger.Warn().
            Err(err).
            Msg("Failed to reconcile tasks")
    }

    duration := timer.Duration()
    logger.Info().
        Dur("duration", duration).
        Msg("Reconciliation cycle completed")

    if duration > 10*time.Second {
        logger.Warn().
            Dur("duration", duration).
            Msg("Reconciliation cycle took longer than expected")
    }

    return nil
}
```

### Task Scheduling

```go
func (s *Scheduler) scheduleTask(task *types.Task) error {
    logger := log.Logger.With().
        Str("component", "scheduler").
        Str("task_id", task.ID).
        Str("service_id", task.ServiceID).
        Logger()

    logger.Debug().
        Str("image", task.Image).
        Msg("Scheduling task")

    node, err := s.selectNode(task)
    if err != nil {
        logger.Error().
            Err(err).
            Msg("Failed to select node for task")
        return err
    }

    logger.Info().
        Str("node_id", node.ID).
        Float64("node_cpu", node.Resources.CPU).
        Int64("node_memory", node.Resources.Memory).
        Msg("Task scheduled to node")

    metrics.TasksScheduled.Inc()
    return nil
}
```

## Configuration

### Console Output (Development)

```go
log.Init(log.Config{
    Level:      log.DebugLevel,
    JSONOutput: false,  // Human-readable console output
})
```

Output:
```
2025-10-12T19:30:00Z INF Service created service=nginx replicas=3
2025-10-12T19:30:01Z DBG Task scheduled task_id=task-123 node_id=node-1
```

### JSON Output (Production)

```go
log.Init(log.Config{
    Level:      log.InfoLevel,
    JSONOutput: true,  // Structured JSON for log aggregation
})
```

Output:
```json
{"level":"info","time":"2025-10-12T19:30:00Z","message":"Service created","service":"nginx","replicas":3}
{"level":"debug","time":"2025-10-12T19:30:01Z","message":"Task scheduled","task_id":"task-123","node_id":"node-1"}
```

### Environment-Based Configuration

In `cmd/warren/main.go`:

```go
func initLogging() {
    level := log.InfoLevel
    if os.Getenv("WARREN_LOG_LEVEL") == "debug" {
        level = log.DebugLevel
    }

    jsonOutput := os.Getenv("WARREN_LOG_JSON") == "true"

    log.Init(log.Config{
        Level:      level,
        JSONOutput: jsonOutput,
    })
}
```

### Log File Output

```go
logFile, err := os.OpenFile("/var/log/warren.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
if err != nil {
    log.Fatal("Failed to open log file")
}

log.Init(log.Config{
    Level:      log.InfoLevel,
    JSONOutput: true,
    Output:     logFile,
})
```

## Integration with Observability

### Correlation with Metrics

```go
timer := metrics.NewTimer()

log.Logger.Info().
    Str("service", serviceName).
    Msg("Creating service")

// ... perform operation ...

duration := timer.Duration()
timer.ObserveDuration(metrics.ServiceCreateDuration)

log.Logger.Info().
    Str("service", serviceName).
    Dur("duration", duration).
    Msg("Service created")
```

### Correlation with Tracing

If adding distributed tracing in the future:

```go
span := trace.StartSpan(ctx, "CreateService")
defer span.End()

log.Logger.Info().
    Str("trace_id", span.TraceID()).
    Str("span_id", span.SpanID()).
    Str("service", serviceName).
    Msg("Creating service")
```

## Troubleshooting

### Enable Debug Logging

```bash
# Set environment variable
export WARREN_LOG_LEVEL=debug

# Or use command-line flag
warren cluster init --log-level=debug
```

### Query JSON Logs

Using `jq` to query JSON logs:

```bash
# Find all errors
cat warren.log | jq 'select(.level=="error")'

# Find logs for specific service
cat warren.log | jq 'select(.service=="nginx")'

# Find slow operations
cat warren.log | jq 'select(.duration > 1000000000)'  # > 1 second in nanoseconds
```

### Common Issues

**Issue**: Too many debug logs in production
- **Solution**: Set `WARREN_LOG_LEVEL=info` or `warn`

**Issue**: Can't parse logs
- **Solution**: Enable JSON output with `WARREN_LOG_JSON=true`

**Issue**: Missing context in logs
- **Solution**: Use context loggers with `.With()` to add persistent fields

## References

- [zerolog Documentation](https://github.com/rs/zerolog)
- [Warren Metrics Guide](./metrics.md)
- [Warren Health Checks](./health-checks.md)
