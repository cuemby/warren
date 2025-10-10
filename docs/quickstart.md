# Warren Quick Start Guide

Get Warren up and running in 5 minutes.

## What is Warren?

Warren is a container orchestration platform that combines Docker Swarm's simplicity with enterprise-level features. It's perfect for edge computing, IoT deployments, and teams who want Kubernetes-like capabilities without the complexity.

**Key Features:**
- 🚀 Single binary deployment (< 20MB)
- 🔄 Automatic task scheduling and recovery
- 📊 Built-in load balancing
- 🔐 Secure by default
- 🌐 Distributed consensus (Raft)

## Prerequisites

- Linux, macOS, or Windows with WSL2
- Go 1.22+ (for building from source)
- Network connectivity between nodes

## Installation

### Option 1: Build from Source

```bash
git clone https://github.com/cuemby/warren.git
cd warren
make build
```

The binary will be at `./bin/warren`.

### Option 2: Install Directly

```bash
make install
# Installs to /usr/local/bin/warren
```

## Your First Cluster

### Step 1: Start a Manager

On your first node, initialize a cluster:

```bash
warren cluster init
```

**Output:**
```
Initializing Warren cluster...
  Node ID: manager-1
  Raft Address: 127.0.0.1:7946
  API Address: 127.0.0.1:8080
  Data Directory: ./warren-data

✓ Cluster initialized successfully
✓ Scheduler started
✓ Reconciler started
gRPC API listening on 127.0.0.1:8080

Manager is running. Press Ctrl+C to stop.
```

**What just happened?**
- Manager started with Raft consensus
- API server listening on port 8080
- Scheduler running (creates tasks every 5s)
- Reconciler running (checks health every 10s)

### Step 2: Add a Worker

In a **new terminal** (or on another machine):

```bash
warren worker start --manager 127.0.0.1:8080
```

**Output:**
```
Starting Warren worker...
  Node ID: worker-1
  Manager: 127.0.0.1:8080
  Resources: 4 cores, 8 GB memory

Worker registered with manager
  Node ID: worker-1
  Overlay IP: 10.0.0.1

Worker is running. Press Ctrl+C to stop.
```

### Step 3: Verify the Cluster

In a **third terminal**:

```bash
warren node list
```

**Output:**
```
ID              ROLE       STATUS          CPU
manager-1       manager    ready           8
worker-1        worker     ready           4
```

🎉 **Congratulations!** You have a working Warren cluster!

## Deploy Your First Service

### Create a Service

```bash
warren service create nginx \
  --image nginx:latest \
  --replicas 3
```

**Output:**
```
✓ Service created: nginx
  ID: abc123def456
  Image: nginx:latest
  Replicas: 3
```

### Check Service Status

```bash
warren service list
```

**Output:**
```
NAME                 REPLICAS     IMAGE                          ID
nginx                3            nginx:latest                   abc123def456
```

### Inspect Service Details

```bash
warren service inspect nginx
```

**Output:**
```
Service: nginx
  ID: abc123def456789
  Image: nginx:latest
  Replicas: 3
  Mode: replicated
```

**What's happening behind the scenes?**
1. Manager saves service to Raft (distributed storage)
2. Scheduler detects new service (within 5 seconds)
3. Scheduler creates 3 tasks
4. Tasks assigned to workers using round-robin
5. Workers execute tasks (simulated in Milestone 1)
6. Workers report status every 5 seconds

## Scale a Service

Need more replicas?

```bash
warren service scale nginx --replicas 5
```

**Output:**
```
✓ Service scaled: nginx
  Replicas: 3 → 5
```

Warren automatically:
- Creates 2 new tasks
- Assigns them to workers
- Starts execution
- Reports status

Need fewer replicas?

```bash
warren service scale nginx --replicas 2
```

Warren automatically:
- Marks 3 tasks for shutdown
- Workers stop tasks gracefully
- Cleans up after grace period

## Clean Up

### Delete a Service

```bash
warren service delete nginx
```

**Output:**
```
✓ Service deleted: nginx
```

All tasks are automatically stopped and cleaned up.

### Stop the Cluster

**Stop Worker:**
```bash
# In worker terminal, press Ctrl+C
```

**Stop Manager:**
```bash
# In manager terminal, press Ctrl+C
```

## Common Commands Reference

### Cluster Management

```bash
# Initialize cluster
warren cluster init

# Custom configuration
warren cluster init \
  --node-id manager-1 \
  --bind-addr 0.0.0.0:7946 \
  --api-addr 0.0.0.0:8080 \
  --data-dir /var/lib/warren
```

### Worker Management

```bash
# Start worker
warren worker start --manager <MANAGER_IP>:8080

# Custom resources
warren worker start \
  --manager 192.168.1.100:8080 \
  --node-id worker-2 \
  --cpu 8 \
  --memory 16
```

### Service Management

```bash
# Create service
warren service create <NAME> --image <IMAGE> --replicas <N>

# With environment variables
warren service create webapp \
  --image myapp:latest \
  --replicas 3 \
  --env DATABASE_URL=postgres://... \
  --env REDIS_URL=redis://...

# List services
warren service list

# Inspect service
warren service inspect <NAME>

# Scale service
warren service scale <NAME> --replicas <N>

# Delete service
warren service delete <NAME>
```

### Node Management

```bash
# List all nodes
warren node list

# Connect to remote manager
warren node list --manager 192.168.1.100:8080
```

## Understanding Warren's Architecture

```
┌─────────────────────────────────────────────────────┐
│                    Manager Node                      │
│                                                       │
│  ┌─────────────┐  ┌──────────────┐  ┌─────────────┐│
│  │    Raft     │  │  Scheduler   │  │ Reconciler  ││
│  │  Consensus  │  │  (5s loop)   │  │  (10s loop) ││
│  └─────────────┘  └──────────────┘  └─────────────┘│
│         │                 │                 │        │
│         └─────────────────┴─────────────────┘        │
│                          │                           │
│                   ┌──────▼──────┐                    │
│                   │  BoltDB     │                    │
│                   │  Storage    │                    │
│                   └─────────────┘                    │
│                          │                           │
│                   ┌──────▼──────┐                    │
│                   │  gRPC API   │                    │
│                   │  :8080      │                    │
│                   └──────┬──────┘                    │
└──────────────────────────┼──────────────────────────┘
                           │
                           │ Heartbeat + Task Status
                           │ Task Assignments
                           │
        ┌──────────────────┼──────────────────┐
        │                  │                  │
┌───────▼────────┐  ┌──────▼────────┐  ┌─────▼────────┐
│  Worker Node   │  │  Worker Node  │  │ Worker Node  │
│                │  │               │  │              │
│  ┌──────────┐  │  │ ┌──────────┐ │  │┌──────────┐  │
│  │   Task   │  │  │ │   Task   │ │  ││   Task   │  │
│  │ Executor │  │  │ │ Executor │ │  ││ Executor │  │
│  └──────────┘  │  │ └──────────┘ │  │└──────────┘  │
└────────────────┘  └───────────────┘  └──────────────┘
```

## What Happens When Things Fail?

Warren is designed to be resilient:

### Task Failure
1. Task crashes or reports failure
2. Worker sends failure status in heartbeat
3. Reconciler detects failure (within 10 seconds)
4. Reconciler marks task for cleanup
5. Scheduler creates replacement (within 5 seconds)
6. New task assigned to healthy worker

### Worker Failure
1. Worker stops sending heartbeat
2. Manager marks node as down (after 30 seconds)
3. Reconciler detects tasks on down node
4. All tasks marked for rescheduling
5. Scheduler creates replacements on healthy nodes

### Manager Failure (Future: Milestone 2)
- Multiple managers with Raft consensus
- Automatic leader election
- Worker reconnects to new leader
- Zero downtime failover

## Configuration Options

### Manager Flags

```bash
warren cluster init [flags]

Flags:
  --node-id string      Unique node ID (default "manager-1")
  --bind-addr string    Raft communication address (default "127.0.0.1:7946")
  --api-addr string     gRPC API address (default "127.0.0.1:8080")
  --data-dir string     Data directory (default "./warren-data")
```

### Worker Flags

```bash
warren worker start [flags]

Flags:
  --node-id string      Unique node ID (default "worker-1")
  --manager string      Manager gRPC address (default "127.0.0.1:8080")
  --data-dir string     Data directory (default "./warren-worker-data")
  --cpu int            CPU cores (default 4)
  --memory int         Memory in GB (default 8)
```

### Service Creation Flags

```bash
warren service create NAME [flags]

Flags:
  --image string           Container image (required)
  --replicas int          Number of replicas (default 1)
  --env strings           Environment variables (KEY=VALUE)
  --manager string        Manager address (default "127.0.0.1:8080")
```

## Troubleshooting

### Worker Can't Connect to Manager

**Symptom:** Worker fails to register

**Solution:**
1. Check manager is running: `ps aux | grep warren`
2. Verify API address: Manager should show "gRPC API listening on X.X.X.X:8080"
3. Check firewall: Port 8080 must be accessible
4. Try with explicit IP: `warren worker start --manager 192.168.1.100:8080`

### Service Not Creating Tasks

**Symptom:** `warren service list` shows service, but no tasks running

**Possible causes:**
1. No workers registered: Run `warren node list` to verify
2. Scheduler not running: Check manager logs for "Scheduler started"
3. Wait 5 seconds: Scheduler runs every 5 seconds

### Tasks Stuck in Pending

**Symptom:** Tasks show as "pending" in worker logs

**This is normal in Milestone 1:**
- Task execution is simulated (no real containers)
- Tasks should transition to "running" within 3 seconds
- Check worker logs for "Task X is running (simulated)"

## Performance Tuning

### Scheduler Interval

Default: 5 seconds
- Lower = faster response, more CPU
- Higher = less CPU, slower response

### Reconciler Interval

Default: 10 seconds
- Lower = faster failure detection, more CPU
- Higher = less CPU, slower failure detection

### Heartbeat Interval

Default: 5 seconds (workers)
- Lower = fresher status, more network traffic
- Higher = less traffic, stale status

## Next Steps

Now that you have Warren running:

1. **Learn More**: Read [Developer Guide](developer-guide.md)
2. **API Reference**: See [API Reference](api-reference.md)
3. **Production Setup**: Follow [Deployment Guide](.agent/System/deployment.md) (coming soon)
4. **Contribute**: Check [Contributing Guide](../CONTRIBUTING.md) (coming soon)

## Getting Help

- **Documentation**: Check [docs/](.)
- **Issues**: Report at [GitHub Issues](https://github.com/cuemby/warren/issues)
- **Discussions**: Join [GitHub Discussions](https://github.com/cuemby/warren/discussions)

## What's Next?

Warren Milestone 1 focuses on core orchestration. Future milestones will add:

- **Milestone 2**: Multi-manager HA, worker partition tolerance
- **Milestone 3**: Real container execution (containerd)
- **Milestone 4**: WireGuard overlay networking
- **Milestone 5**: Advanced deployments (blue/green, canary)
- **Milestone 6**: Observability and monitoring

See [tasks/todo.md](../tasks/todo.md) for the complete roadmap.

---

**Happy Orchestrating!** 🚀
