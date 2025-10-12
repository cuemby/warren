# Graceful Shutdown

Warren provides configurable graceful shutdown for containers, allowing applications to terminate cleanly and save state before being forcibly stopped.

## Overview

Graceful shutdown allows Warren to:
- **Stop containers cleanly** without data loss
- **Complete in-flight requests** before termination
- **Save state** to disk or databases
- **Release resources** properly (connections, file handles)
- **Prevent data corruption** from abrupt termination

## How Graceful Shutdown Works

Warren uses a two-phase shutdown process:

1. **SIGTERM (Graceful)**: Warren sends SIGTERM signal to the container
   - Container can catch this signal and begin shutdown
   - Application has time to finish work and clean up

2. **Wait Period**: Warren waits for the configured timeout
   - Container should exit voluntarily during this period
   - Default timeout: 10 seconds

3. **SIGKILL (Force)**: If container hasn't exited after timeout
   - Warren sends SIGKILL to force termination
   - Cannot be caught or ignored by the application
   - Container terminates immediately

```
Container Running
       │
       │ warren service delete/update
       ▼
Send SIGTERM ─────────────────┐
       │                      │
       │ Application handles  │
       │ signal and cleanups  │
       │                      │
       ▼                      │
Wait for stop-timeout         │
       │                      │
       │                      │
       ▼                      │
Container exited? ────NO──────┘
       │                      │
      YES                Send SIGKILL
       │                      │
       ▼                      ▼
   Cleanup            Force Terminate
```

## Configuring Stop Timeout

### Default Behavior

By default, Warren waits 10 seconds between SIGTERM and SIGKILL:

```bash
warren service create api --image myapi:latest
# Uses default 10 second stop timeout
```

### Custom Stop Timeout

Use the `--stop-timeout` flag to configure the timeout:

```bash
# Quick shutdown (5 seconds)
warren service create cache \
  --image redis:latest \
  --stop-timeout 5

# Standard shutdown (15 seconds)
warren service create api \
  --image myapi:latest \
  --stop-timeout 15

# Long shutdown for databases (30 seconds)
warren service create postgres \
  --image postgres:16 \
  --env POSTGRES_PASSWORD=secret \
  --stop-timeout 30

# Very long shutdown for batch jobs (60 seconds)
warren service create worker \
  --image worker:latest \
  --stop-timeout 60
```

### Viewing Stop Timeout

The stop timeout is displayed when creating a service:

```bash
$ warren service create api --image myapi:latest --stop-timeout 20

✓ Service created: api
  ID: api-abc123
  Image: myapi:latest
  Replicas: 1
  Stop Timeout: 20 seconds
```

## Application Integration

### Handling SIGTERM

Your application should handle SIGTERM to shut down gracefully:

#### Go Example

```go
package main

import (
    "context"
    "fmt"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"
)

func main() {
    server := &http.Server{Addr: ":8080"}

    // Handle SIGTERM
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGTERM)

    go func() {
        <-sigChan
        fmt.Println("Received SIGTERM, shutting down gracefully...")

        // Give requests 15 seconds to complete
        ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
        defer cancel()

        // Shutdown server
        if err := server.Shutdown(ctx); err != nil {
            fmt.Printf("Error during shutdown: %v\n", err)
        }
    }()

    // Start server
    if err := server.ListenAndServe(); err != http.ErrServerClosed {
        fmt.Printf("Server error: %v\n", err)
    }
}
```

#### Node.js Example

```javascript
const express = require('express');
const app = express();
const server = app.listen(3000);

// Track active requests
let activeRequests = 0;

app.use((req, res, next) => {
  activeRequests++;
  res.on('finish', () => activeRequests--);
  next();
});

// Handle SIGTERM
process.on('SIGTERM', () => {
  console.log('Received SIGTERM, shutting down gracefully...');

  // Stop accepting new connections
  server.close(() => {
    console.log('Server closed');

    // Close database connections, etc.
    process.exit(0);
  });

  // Force shutdown after 15 seconds
  setTimeout(() => {
    console.error('Forced shutdown after timeout');
    process.exit(1);
  }, 15000);
});
```

#### Python Example

```python
import signal
import sys
import time
from flask import Flask

app = Flask(__name__)
shutdown_event = False

def handle_sigterm(signum, frame):
    global shutdown_event
    print("Received SIGTERM, shutting down gracefully...")
    shutdown_event = True

    # Finish current work
    cleanup()

    # Exit cleanly
    sys.exit(0)

def cleanup():
    # Close database connections
    # Flush buffers
    # Save state
    print("Cleanup complete")

# Register signal handler
signal.signal(signal.SIGTERM, handle_sigterm)

if __name__ == '__main__':
    app.run()
```

### Best Practices for Applications

1. **Listen for SIGTERM**: Always register a SIGTERM handler
2. **Stop Accepting Work**: Don't accept new requests/jobs after SIGTERM
3. **Complete In-Flight Work**: Finish processing current requests
4. **Set Reasonable Timeout**: Internal timeout should be less than Warren's stop-timeout
5. **Exit with Code 0**: Exit cleanly after successful shutdown

## Choosing Stop Timeout Values

### By Application Type

| Application Type | Recommended Timeout | Reason |
|-----------------|--------------------|-----------------------------------------|
| Stateless API | 10-15 seconds | Finish current HTTP requests |
| Database | 30-60 seconds | Flush buffers, close connections |
| Message Queue Consumer | 30-60 seconds | Process current messages |
| Batch Job | 60-300 seconds | Complete current batch |
| Cache (Redis) | 5-10 seconds | Save RDB snapshot |
| Web Server (nginx) | 5-10 seconds | Finish serving files |

### Factors to Consider

1. **Request Duration**: How long do requests typically take?
   - Short requests (< 1s): 10-15 second timeout
   - Long requests (> 5s): 30-60 second timeout

2. **Data Persistence**: Does the app need to save state?
   - No state: 5-10 second timeout
   - Must save state: 30-60 second timeout

3. **External Dependencies**: How long to drain connections?
   - No external deps: 5-10 second timeout
   - Many connections: 15-30 second timeout

4. **Update Frequency**: How often do you update services?
   - Frequent updates: Keep timeout short (10-15s)
   - Rare updates: Can use longer timeout (30-60s)

### Example Configurations

```bash
# Nginx (fast shutdown, just finish serving files)
warren service create nginx \
  --image nginx:latest \
  --stop-timeout 5

# Node.js API (finish HTTP requests)
warren service create api \
  --image node-api:latest \
  --stop-timeout 15

# PostgreSQL (flush WAL, close connections)
warren service create postgres \
  --image postgres:16 \
  --stop-timeout 30

# Background worker (complete current job)
warren service create worker \
  --image worker:latest \
  --stop-timeout 60

# Batch processor (finish entire batch)
warren service create batch \
  --image batch-processor:latest \
  --stop-timeout 300
```

## Behavior During Updates

When updating a service, Warren uses graceful shutdown for old tasks:

```bash
$ warren service update api --image myapi:v2

# Warren will:
# 1. Create new tasks with v2 image
# 2. Wait for new tasks to be healthy
# 3. Send SIGTERM to old tasks
# 4. Wait for stop-timeout (configured value)
# 5. Send SIGKILL if tasks haven't exited
# 6. Delete old tasks
```

This ensures zero-downtime updates with clean task termination.

## Troubleshooting

### Tasks Take Too Long to Stop

**Symptom**: Task deletions take the full stop-timeout period

**Check**:
1. Verify application handles SIGTERM:
   ```bash
   # Send SIGTERM manually and check logs
   kill -TERM <pid>
   ```

2. Check application logs for shutdown messages
3. Verify no deadlocks or hung operations

**Solutions**:
- Implement SIGTERM handler in application
- Fix blocking operations in shutdown code
- Reduce cleanup work during shutdown

### Tasks Killed Before Finishing Work

**Symptom**: Data loss or incomplete requests during shutdown

**Check**:
1. Current stop timeout: `warren service inspect SERVICENAME`
2. How long shutdown actually takes (check app logs)

**Solutions**:
- Increase stop timeout: `warren service update api --stop-timeout 30`
- Optimize shutdown code to be faster
- Move long-running cleanup to background process

### Application Doesn't Handle SIGTERM

**Symptom**: Every task waits full stop-timeout before terminating

**Solutions**:
1. Add SIGTERM handler to application (see examples above)
2. Use exec form of CMD in Dockerfile:
   ```dockerfile
   # Wrong (shell form - shell doesn't forward signals)
   CMD node server.js

   # Correct (exec form - signals go directly to app)
   CMD ["node", "server.js"]
   ```

3. For shell scripts, use `exec` to replace shell:
   ```bash
   #!/bin/sh
   exec node server.js  # Replace shell with node process
   ```

### Stop Timeout Too Short

**Symptom**: Tasks being SIGKILL'd before finishing cleanup

**Diagnosis**:
- Check application logs for incomplete shutdown
- Look for "killed" messages before cleanup completes

**Solution**:
```bash
warren service update SERVICENAME --stop-timeout 30
```

### Stop Timeout Too Long

**Symptom**: Service updates take unnecessarily long

**Diagnosis**:
- Tasks exit quickly but Warren still waits full timeout
- Check task exit time in logs

**Solution**:
```bash
warren service update SERVICENAME --stop-timeout 10
```

## Advanced Topics

### Zero-Downtime Updates

Combine graceful shutdown with health checks for zero-downtime updates:

```bash
warren service create api \
  --image myapi:v1 \
  --replicas 3 \
  --health-http /health \
  --health-interval 10 \
  --stop-timeout 15
```

Update flow:
1. New task created with v2 image
2. Health check ensures new task is healthy
3. Old task receives SIGTERM
4. Old task finishes current requests (up to 15 seconds)
5. Old task exits cleanly
6. Repeat for remaining tasks

### Graceful Shutdown + Restart Policy

Graceful shutdown works with restart policies:

```bash
warren service create worker \
  --image worker:latest \
  --restart-condition on-failure \
  --stop-timeout 30
```

Behavior:
- **Manual Stop**: Uses graceful shutdown (SIGTERM → wait → SIGKILL)
- **Crash**: Restart policy applies
- **Update**: Old tasks use graceful shutdown, new tasks created

### Testing Graceful Shutdown

Test your application's shutdown behavior:

```bash
# 1. Run container locally
docker run -d --name test myapp:latest

# 2. Send SIGTERM
docker kill --signal=TERM test

# 3. Check logs for clean shutdown
docker logs test

# 4. Verify exit code
docker inspect test | grep ExitCode
# Should be 0 for clean shutdown
```

## Implementation Details

Warren implements graceful shutdown using containerd's task management:

1. **Worker receives stop command** from reconciler/scheduler
2. **Worker calls StopContainer()** on containerd runtime
3. **Runtime sends SIGTERM** to container process
4. **Runtime creates context** with stop-timeout deadline
5. **Runtime waits** for task exit or timeout
6. **If timeout expires**: Runtime sends SIGKILL
7. **Worker cleans up** task resources

The stop timeout flows through the system:
- Service specification → Task → Worker → Runtime → Container

## See Also

- [Service Management](./services.md)
- [Health Checks](./health-checks.md)
- [Rolling Updates](./updates.md)
- [CLI Reference](./cli-reference.md)
