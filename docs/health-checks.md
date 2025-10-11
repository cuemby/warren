# Health Checks

Warren provides comprehensive health check monitoring for all containers, ensuring reliability and automatic recovery from failures.

## Overview

Health checks allow Warren to:
- **Monitor container health** continuously
- **Detect failures** before users notice
- **Auto-replace unhealthy containers** without manual intervention
- **Prevent cascading failures** in distributed systems

## Health Check Types

Warren supports three types of health checks:

### 1. HTTP Health Checks

HTTP checks make HTTP/HTTPS requests to a container endpoint.

```bash
warren service create web \
  --image nginx:latest \
  --health-http / \
  --health-interval 30 \
  --health-timeout 10 \
  --health-retries 3
```

**Parameters:**
- **Path**: HTTP path to check (e.g., `/health`, `/api/status`)
- **Port**: Container port (default: 80)
- **Scheme**: `http` or `https` (default: http)
- **Status Codes**: Acceptable range (default: 200-399)

**Use Cases:**
- Web servers (nginx, Apache)
- HTTP APIs and microservices
- Application health endpoints

### 2. TCP Health Checks

TCP checks verify port connectivity.

```bash
warren service create db \
  --image postgres:16 \
  --health-tcp 5432 \
  --health-interval 30
```

**Parameters:**
- **Port**: Container port to check

**Use Cases:**
- Databases (PostgreSQL, MySQL, Redis)
- Message queues (RabbitMQ, Kafka)
- Any TCP service

### 3. Exec Health Checks

Exec checks run commands inside the container.

```bash
warren service create postgres \
  --image postgres:16 \
  --health-cmd pg_isready \
  --health-interval 30
```

**Parameters:**
- **Command**: Command and arguments to execute

**Exit Codes:**
- `0` = Healthy
- Non-zero = Unhealthy

**Use Cases:**
- Database readiness (pg_isready, mysqladmin ping)
- Application-specific checks
- File existence checks

## Health Check Configuration

### Common Parameters

All health check types support these parameters:

| Flag | Default | Description |
|------|---------|-------------|
| `--health-interval` | 30s | Time between checks |
| `--health-timeout` | 10s | Check timeout |
| `--health-retries` | 3 | Failures before unhealthy |

### Examples

**Web service with custom interval:**
```bash
warren service create api \
  --image myapi:latest \
  --health-http /health \
  --health-interval 15 \
  --health-timeout 5
```

**Database with fast failure detection:**
```bash
warren service create redis \
  --image redis:7 \
  --health-tcp 6379 \
  --health-interval 10 \
  --health-retries 2
```

**Application with exec check:**
```bash
warren service create app \
  --image myapp:latest \
  --health-cmd /app/healthcheck.sh \
  --health-interval 60
```

## Health Check Flow

```
┌─────────────┐
│   Service   │
│  (replicas) │
└──────┬──────┘
       │
       │ 1. Task Scheduled
       ▼
┌─────────────┐
│   Worker    │
└──────┬──────┘
       │
       │ 2. Container Started
       ▼
┌─────────────────┐
│ Health Monitor  │
│  (per task)     │
└──────┬──────────┘
       │
       │ 3. Periodic Checks
       ▼
┌─────────────────┐
│  HTTP/TCP/Exec  │
│    Checker      │
└──────┬──────────┘
       │
       │ 4. Report Status
       ▼
┌─────────────────┐
│    Manager      │
│ (Update Status) │
└──────┬──────────┘
       │
       │ 5. Check Health
       ▼
┌─────────────────┐
│   Reconciler    │
│ (Every 10s)     │
└──────┬──────────┘
       │
       │ 6. Unhealthy?
       ▼
┌─────────────────┐
│  Mark Failed    │
└──────┬──────────┘
       │
       │ 7. Trigger
       ▼
┌─────────────────┐
│   Scheduler     │
│ (Create New)    │
└─────────────────┘
```

## Behavior

### Health Check Lifecycle

1. **Container Starts**: Worker begins health monitoring
2. **Initial Check**: First check runs immediately
3. **Periodic Checks**: Checks run every `interval` seconds
4. **Status Tracking**: Consecutive failures/successes counted
5. **Failure Threshold**: After `retries` failures, task marked unhealthy
6. **Auto-Replacement**: Reconciler marks unhealthy tasks as failed
7. **New Task**: Scheduler creates replacement on healthy node

### Failure Detection

- **Consecutive Failures**: Must fail `retries` times in a row
- **No Flapping**: Single failure doesn't trigger replacement
- **Debouncing**: Protects against transient network issues

### Recovery

- **Automatic**: No manual intervention required
- **Fast**: Detection time = `interval` × `retries` seconds
- **Safe**: Old container keeps running until replacement is healthy

## Best Practices

### 1. Choose the Right Type

- **HTTP**: For web services and APIs (most common)
- **TCP**: For databases and simple port checks
- **Exec**: For complex application-specific logic

### 2. Set Appropriate Intervals

```bash
# Fast detection for critical services
--health-interval 10 --health-retries 2  # 20s to failure

# Balanced for most services
--health-interval 30 --health-retries 3  # 90s to failure

# Slow for expensive checks
--health-interval 60 --health-retries 2  # 120s to failure
```

### 3. Design Robust Health Endpoints

**Good health endpoint:**
```go
func healthHandler(w http.ResponseWriter, r *http.Request) {
    // Check dependencies
    if !db.Ping() {
        http.Error(w, "database unreachable", 503)
        return
    }

    if !cache.IsConnected() {
        http.Error(w, "cache unreachable", 503)
        return
    }

    // All checks passed
    w.WriteHeader(200)
    w.Write([]byte("healthy"))
}
```

**Bad health endpoint:**
```go
func healthHandler(w http.ResponseWriter, r *http.Request) {
    // Always returns 200 - useless!
    w.WriteHeader(200)
}
```

### 4. Consider Resource Usage

- HTTP checks are lightweight (~1ms)
- TCP checks are fastest (~0.1ms)
- Exec checks can be expensive (depends on command)

### 5. Avoid Check Dependencies

❌ **Bad**: Health check depends on external service
```bash
# Don't do this - creates circular dependency
curl https://api.example.com/health
```

✅ **Good**: Health check is self-contained
```bash
# Check only local resources
pg_isready -h localhost
```

## Troubleshooting

### Health Checks Not Running

**Symptoms**: No health status updates

**Check**:
1. Verify worker is running: `warren node list`
2. Check task is running: `warren service inspect SERVICENAME`
3. Review worker logs for health check errors

### Tasks Keep Getting Replaced

**Symptoms**: Task fails health checks repeatedly

**Check**:
1. Test health endpoint manually:
   ```bash
   # For HTTP
   curl http://container-ip:port/path

   # For TCP
   nc -zv container-ip port

   # For Exec
   docker exec container-id command
   ```

2. Increase timeout if checks are slow:
   ```bash
   --health-timeout 30
   ```

3. Check container logs:
   ```bash
   warren logs SERVICE_NAME
   ```

### False Positives

**Symptoms**: Healthy containers marked as unhealthy

**Solutions**:
- Increase retries: `--health-retries 5`
- Increase interval: `--health-interval 60`
- Simplify health check logic
- Check for network issues between worker and container

### Slow Startup Containers

**Symptoms**: Container marked unhealthy before it finishes starting

**Solutions**:
- Use longer initial interval
- Implement `/ready` endpoint separate from `/health`
- Add initialization logic to health endpoint

## Integration Tests

Warren includes comprehensive health check integration tests:

```bash
# Run all health check tests
go test -v ./test/integration -run TestHealthCheck

# Run specific test
go test -v ./test/integration -run TestHealthCheckHTTP
```

See [test/README.md](../test/README.md) for details.

## Examples

### Nginx Web Server

```bash
warren service create nginx \
  --image nginx:latest \
  --replicas 3 \
  --health-http / \
  --health-interval 30
```

### PostgreSQL Database

```bash
warren service create postgres \
  --image postgres:16 \
  --env POSTGRES_PASSWORD=secret \
  --health-cmd pg_isready \
  --health-interval 10
```

### Redis Cache

```bash
warren service create redis \
  --image redis:7 \
  --health-tcp 6379 \
  --health-interval 15
```

### Custom Application

```bash
warren service create myapp \
  --image myapp:v1.0 \
  --health-http /api/health \
  --health-interval 20 \
  --health-timeout 10 \
  --health-retries 3
```

## Advanced Topics

### Multiple Health Checks

Currently, Warren supports one health check per service. For multiple checks:

1. Create a wrapper script that runs all checks
2. Use as exec health check
3. Return 0 only if all checks pass

### Health vs Readiness

- **Health**: Is the container working?
- **Readiness**: Is the container ready for traffic?

Warren currently uses a single health check that serves both purposes. Future versions may add separate readiness probes for load balancing.

### Monitoring Health Status

View health status via:

```bash
# Service level
warren service inspect SERVICENAME

# Task level (TODO: add health status to task inspect)
warren service inspect SERVICENAME
```

## See Also

- [Services Documentation](concepts/services.md)
- [Reconciler Documentation](.agent/System/reconciler.md)
- [Integration Tests](../test/README.md)
