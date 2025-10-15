# Getting Started with Warren

Welcome to Warren! This guide will help you deploy your first containerized service in under 5 minutes.

## What is Warren?

Warren is a simple-yet-powerful container orchestration platform that combines the simplicity of Docker Swarm with the feature richness of Kubernetes. It's designed for edge computing with:

- **Single binary** - No external dependencies (< 100MB)
- **Built-in features** - Secrets, volumes, load balancing, HA out of the box
- **Edge-optimized** - Partition tolerance, low resource usage
- **Simple API** - Easy to learn, powerful enough for production

## Prerequisites

- **Linux or macOS** (ARM64 or AMD64)
- **containerd** - Container runtime
  - Ubuntu/Debian: `sudo apt install containerd`
  - macOS: Installed via Docker Desktop or Lima
- **Root/sudo access** - Required for container operations

## Installation

### Option 1: Download Binary

```bash
# Linux AMD64
curl -LO https://github.com/cuemby/warren/releases/latest/download/warren-linux-amd64
chmod +x warren-linux-amd64
sudo mv warren-linux-amd64 /usr/local/bin/warren

# Linux ARM64
curl -LO https://github.com/cuemby/warren/releases/latest/download/warren-linux-arm64
chmod +x warren-linux-arm64
sudo mv warren-linux-arm64 /usr/local/bin/warren

# macOS
curl -LO https://github.com/cuemby/warren/releases/latest/download/warren-darwin-arm64
chmod +x warren-darwin-arm64
sudo mv warren-darwin-arm64 /usr/local/bin/warren
```

### Option 2: Homebrew (macOS)

```bash
brew install cuemby/tap/warren
```

### Option 3: APT (Debian/Ubuntu)

```bash
curl -sL https://packagecloud.io/cuemby/warren/gpgkey | sudo apt-key add -
echo "deb https://packagecloud.io/cuemby/warren/ubuntu/ focal main" | sudo tee /etc/apt/sources.list.d/warren.list
sudo apt update
sudo apt install warren
```

### Verify Installation

```bash
warren --version
# Output: warren version 1.0.0
```

---

## Quick Start: Single-Node Cluster

This quick start guide will create a single-node cluster (manager + worker on same machine) and deploy an nginx service.

### Step 1: Initialize Cluster

```bash
# Initialize Warren cluster
sudo warren cluster init

# Output:
# ✓ Raft consensus initialized
# ✓ Manager started (Node ID: manager-abc123)
# ✓ API server listening on TCP: 127.0.0.1:8080 (mTLS)
# ✓ API server listening on Unix socket: /var/run/warren.sock (local, read-only)
# ✓ Metrics available at http://127.0.0.1:9090/metrics
#
# Cluster initialized successfully!
# Local CLI is ready to use immediately!
```

The manager node stores cluster state (using Raft) and provides the API for management operations.

> **💡 Warren v1.4.0 Simplicity**: CLI commands now work immediately via Unix socket - no setup required! Just run `warren node list` to verify. Remote access still requires `warren init` for security.

### Step 2: Start Worker

```bash
# In another terminal, start a worker on the same machine
sudo warren worker start --manager 127.0.0.1:8080

# Output:
# ✓ Worker started (Node ID: worker-xyz789)
# ✓ Connected to manager at 127.0.0.1:8080
# ✓ Heartbeat: 30s interval
#
# Worker ready to accept tasks!
```

The worker executes containers as assigned by the manager's scheduler.

### Step 3: Verify Cluster

```bash
# List nodes in the cluster (works immediately via Unix socket!)
warren node list

# Output:
# ID              ROLE      STATUS    ADDRESS         LABELS
# manager-abc123  manager   ready     127.0.0.1:8080
# worker-xyz789   worker    ready     127.0.0.1:0
```

### Step 4: Deploy Service

> **Note**: Service creation is a write operation. For local single-node clusters, you'll need to use `--manager` flag or set up mTLS certificates. See "[Remote CLI Setup](#remote-cli-setup)" section below for details.

```bash
# Deploy nginx with 2 replicas (requires --manager for write operations)
warren service create nginx \
  --image nginx:latest \
  --replicas 2 \
  --manager 127.0.0.1:8080

# Output:
# ✓ Service 'nginx' created
# ✓ 2 tasks scheduled
# Service ID: svc-nginx-123
```

### Step 5: Check Service Status

```bash
# List services (read operation - works via Unix socket!)
warren service list

# Output:
# NAME     IMAGE          REPLICAS  MODE        CREATED
# nginx    nginx:latest   2/2       replicated  30s ago

# Inspect service details (read operation - works via Unix socket!)
warren service inspect nginx

# Output:
# Service: nginx
# ID: svc-nginx-123
# Mode: replicated
# Replicas: 2/2
# Image: nginx:latest
#
# Tasks:
#   task-nginx-1  worker-xyz789  running  nginx:latest  10.0.1.5  1m ago
#   task-nginx-2  worker-xyz789  running  nginx:latest  10.0.1.6  1m ago
```

### Step 6: Scale Service

```bash
# Scale to 4 replicas (write operation - requires --manager)
warren service scale nginx --replicas 4 --manager 127.0.0.1:8080

# Output:
# ✓ Service 'nginx' scaled to 4 replicas

# Verify scaling (read operation - works via Unix socket!)
warren service list

# Output:
# NAME     IMAGE          REPLICAS  MODE        CREATED
# nginx    nginx:latest   4/4       replicated  2m ago
```

### Step 7: Update Service

```bash
# Update to nginx:alpine (write operation - requires --manager)
warren service update nginx --image nginx:alpine --manager 127.0.0.1:8080

# Output:
# ✓ Service 'nginx' updated
# ✓ Rolling update in progress (1 task at a time)
```

### Step 8: Clean Up

```bash
# Delete service (write operation - requires --manager)
warren service delete nginx --manager 127.0.0.1:8080

# Output:
# ✓ Service 'nginx' deleted
# ✓ All tasks stopped and removed

# Stop worker (Ctrl+C in worker terminal)
# Stop manager (Ctrl+C in manager terminal)
```

---

## Remote CLI Setup

Warren v1.4.0+ provides zero-config local access via Unix socket for read operations. However, you'll need to set up mTLS certificates in these scenarios:

1. **Write operations** on local manager (create, update, delete, scale)
2. **Remote access** to managers from another machine
3. **Automation scripts** that need write access

### When Do You Need This?

| Scenario | Unix Socket (No Setup) | mTLS Setup Required |
|----------|------------------------|---------------------|
| Local read operations (`warren node list`, `service list`) | ✅ Works | Not needed |
| Local write operations (`service create`, `scale`) | ❌ Blocked | ✅ Required |
| Remote access from another machine | ❌ N/A | ✅ Required |

### Setup mTLS for Local Write Operations

If you're on the manager node and need write access:

```bash
# Option 1: Use --manager flag (quick, per-command)
warren service create nginx --image nginx:latest --manager 127.0.0.1:8080

# Option 2: Initialize CLI with mTLS (permanent setup)
# Get the manager token
sudo cat /var/lib/warren/cluster/manager_token.txt

# Initialize CLI
warren init --manager 127.0.0.1:8080 --token <token-from-above>

# Output:
# ✓ CLI certificate requested and saved
# ✓ Certificates stored in ~/.warren/certs/cli/
# ✓ You can now use warren commands without --manager flag

# Now write operations work without --manager flag:
warren service create nginx --image nginx:latest
```

### Setup mTLS for Remote Access

If you're accessing Warren from a different machine:

**On the manager node:**
```bash
# Generate join token for CLI access
warren cluster join-token manager --manager 127.0.0.1:8080

# Output:
# Join Token (expires in 24h):
# SWMTKN-1-3x7h9f2k1p6v8w4q0n5m7j9l2b8d6f4g1h3k5m7n9p2r4t6
```

**On your local machine (remote):**
```bash
# Initialize CLI with manager address and token
warren init --manager 192.168.1.10:8080 --token SWMTKN-1-3x7h9f2k1p6v8w4q0n5m7j9l2b8d6f4g1h3k5m7n9p2r4t6

# Output:
# ✓ Connected to manager at 192.168.1.10:8080
# ✓ CLI certificate requested and saved
# ✓ Certificates stored in ~/.warren/certs/cli/

# Now all commands work:
warren node list
warren service list
warren service create web --image nginx:latest
```

### Using Environment Variables

Set the default manager to avoid repeating `--manager` flag:

```bash
# Add to ~/.bashrc or ~/.zshrc
export WARREN_MANAGER=192.168.1.10:8080

# Now commands work without --manager flag
warren service list
warren service create app --image myapp:latest
```

---

## Multi-Node Cluster

For production deployments, you'll want a multi-manager cluster (3 or 5 managers) for high availability.

### Architecture

```
┌─────────────┐  ┌─────────────┐  ┌─────────────┐
│  Manager 1  │  │  Manager 2  │  │  Manager 3  │
│   (Leader)  │◄─┤             │◄─┤             │
└──────┬──────┘  └──────┬──────┘  └──────┬──────┘
       │                │                │
       └────────────────┴────────────────┘
                        │
       ┌────────────────┴────────────────┐
       │                                  │
┌──────▼──────┐                    ┌──────▼──────┐
│   Worker 1  │                    │   Worker 2  │
└─────────────┘                    └─────────────┘
```

### Setup Multi-Node Cluster

**On Manager 1 (bootstrap):**

```bash
# Initialize first manager
sudo warren cluster init --advertise-addr 192.168.1.10:8080

# Generate join token for additional managers (write operation - requires --manager)
warren cluster join-token manager --manager 127.0.0.1:8080

# Output:
# Join Token (expires in 24h):
# SWMTKN-1-3x7h9f2k1p6v8w4q0n5m7j9l2b8d6f4g1h3k5m7n9p2r4t6
#
# On other nodes, run:
# warren manager join --token SWMTKN-1-... --manager 192.168.1.10:8080
```

**On Manager 2:**

```bash
# Join as second manager
sudo warren manager join \
  --token SWMTKN-1-3x7h9f2k1p6v8w4q0n5m7j9l2b8d6f4g1h3k5m7n9p2r4t6 \
  --manager 192.168.1.10:8080 \
  --advertise-addr 192.168.1.11:8080
```

**On Manager 3:**

```bash
# Join as third manager
sudo warren manager join \
  --token SWMTKN-1-3x7h9f2k1p6v8w4q0n5m7j9l2b8d6f4g1h3k5m7n9p2r4t6 \
  --manager 192.168.1.10:8080 \
  --advertise-addr 192.168.1.12:8080
```

**On Worker Nodes:**

```bash
# Generate worker join token (on any manager)
warren cluster join-token worker --manager 192.168.1.10:8080

# Output:
# Join Token (expires in 24h):
# SWMTKN-2-9k7j5h3f1d9s7a5k3l1n9m7p5r3t1v9x7z5b3d1f9h7j5k3
#
# On worker nodes, run:
# warren worker start --token SWMTKN-2-... --manager <manager-ip>:8080

# On worker 1
sudo warren worker start \
  --token SWMTKN-2-9k7j5h3f1d9s7a5k3l1n9m7p5r3t1v9x7z5b3d1f9h7j5k3 \
  --manager 192.168.1.10:8080

# On worker 2
sudo warren worker start \
  --token SWMTKN-2-9k7j5h3f1d9s7a5k3l1n9m7p5r3t1v9x7z5b3d1f9h7j5k3 \
  --manager 192.168.1.10:8080
```

**Verify Multi-Manager Cluster:**

From any manager node (using Unix socket):
```bash
# Check cluster info (read operation - works via Unix socket!)
warren cluster info

# Output:
# Cluster ID: cluster-abc123
# Raft Quorum: 3 managers (Leader: manager-1)
# Nodes: 5 total (3 managers, 2 workers)
#
# Managers:
#   manager-1  192.168.1.10:8080  Leader   ready
#   manager-2  192.168.1.11:8080  Follower ready
#   manager-3  192.168.1.12:8080  Follower ready
#
# Workers:
#   worker-1   ready  2 tasks
#   worker-2   ready  3 tasks
```

From a remote machine (using mTLS):
```bash
# First, set up remote access (see "Remote CLI Setup" section above)
warren init --manager 192.168.1.10:8080 --token <manager-token>

# Then check cluster info
warren cluster info
```

---

## Working with Secrets

> **Note**: `secret create` (write) requires `--manager` or mTLS setup. `secret list` (read) works via Unix socket.

```bash
# Create secret from literal (write operation - requires --manager)
warren secret create db-password \
  --from-literal password=mySecurePassword123 \
  --manager 127.0.0.1:8080

# Create secret from file (write operation - requires --manager)
warren secret create tls-cert \
  --from-file /path/to/cert.pem \
  --manager 127.0.0.1:8080

# List secrets (read operation - works via Unix socket!)
warren secret list

# Deploy service with secret (write operation - requires --manager)
warren service create webapp \
  --image myapp:latest \
  --secret db-password \
  --replicas 3 \
  --manager 127.0.0.1:8080

# Secret is mounted at /run/secrets/db-password in container
```

---

## Working with Volumes

> **Note**: `volume create` (write) requires `--manager` or mTLS setup. `volume list` (read) works via Unix socket.

```bash
# Create volume (write operation - requires --manager)
warren volume create db-data \
  --driver local \
  --manager 127.0.0.1:8080

# Deploy service with volume (write operation - requires --manager)
warren service create postgres \
  --image postgres:15 \
  --volume db-data:/var/lib/postgresql/data \
  --replicas 1 \
  --manager 127.0.0.1:8080

# List volumes (read operation - works via Unix socket!)
warren volume list
```

---

## Using YAML Files

Warren supports declarative configuration via YAML:

**nginx-service.yaml:**

```yaml
apiVersion: warren.io/v1
kind: Service
metadata:
  name: nginx
spec:
  image: nginx:latest
  replicas: 3
  mode: replicated
  env:
    - name: NGINX_PORT
      value: "80"
```

**Apply YAML:**

```bash
# Apply configuration
warren apply -f nginx-service.yaml --manager 127.0.0.1:8080

# Apply multiple files
warren apply -f app.yaml -f db.yaml -f cache.yaml --manager 127.0.0.1:8080

# Apply directory
warren apply -f ./manifests/ --manager 127.0.0.1:8080
```

---

## Next Steps

Now that you have Warren running, explore these topics:

- **[Concepts](concepts/architecture.md)** - Understand Warren's architecture
- **[CLI Reference](cli-reference.md)** - Complete command documentation
- **[Migration from Docker Swarm](migration/from-docker-swarm.md)** - Migrate existing workloads
- **[Migration from Docker Compose](migration/from-docker-compose.md)** - Convert Compose files
- **[Troubleshooting](troubleshooting.md)** - Common issues and solutions

---

## Quick Reference

### Common Commands

```bash
# Cluster management
warren cluster init                          # Initialize cluster
warren cluster info                          # Show cluster status (read - Unix socket!)
warren cluster join-token [manager|worker] \ # Generate join token (write)
  --manager <addr>

# Node management
warren node list                             # List all nodes (read - Unix socket!)
warren manager join --token <token> ...      # Join as manager
warren worker start --manager <addr>         # Start worker

# Service management (read operations work via Unix socket!)
warren service list                          # List services (read)
warren service inspect <name>                # Inspect service (read)
warren service create <name> --image <img> \ # Create service (write)
  --manager <addr>
warren service scale <name> --replicas <n> \ # Scale service (write)
  --manager <addr>
warren service update <name> --image <img> \ # Update service (write)
  --manager <addr>
warren service delete <name> \               # Delete service (write)
  --manager <addr>

# Secrets (read operations work via Unix socket!)
warren secret list                           # List secrets (read)
warren secret create <name> \                # Create secret (write)
  --from-literal key=value --manager <addr>

# Volumes (read operations work via Unix socket!)
warren volume list                           # List volumes (read)
warren volume create <name> \                # Create volume (write)
  --driver local --manager <addr>

# YAML (write operation)
warren apply -f <file.yaml> --manager <addr>
```

> **💡 Tip**: Read operations (list, inspect, info) work immediately via Unix socket. Write operations (create, update, delete) require `--manager` flag or `warren init` setup.

### Environment Variables

```bash
# Set default manager address (useful for write operations or remote access)
export WARREN_MANAGER=127.0.0.1:8080

# Now write commands work without --manager flag
warren service create web --image nginx:latest
warren service scale web --replicas 3

# Note: Read commands already work via Unix socket (no ENV needed)
```

---

## Getting Help

- **Documentation**: [https://github.com/cuemby/warren/docs](https://github.com/cuemby/warren/tree/main/docs)
- **GitHub Issues**: [https://github.com/cuemby/warren/issues](https://github.com/cuemby/warren/issues)
- **GitHub Discussions**: [https://github.com/cuemby/warren/discussions](https://github.com/cuemby/warren/discussions)
- **Email**: opensource@cuemby.com

---

**Congratulations!** You've deployed your first service with Warren. Welcome to simple, powerful container orchestration! 🎉
