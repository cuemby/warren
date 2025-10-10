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
# ✓ API server listening on 127.0.0.1:8080
# ✓ Metrics available at http://127.0.0.1:9090/metrics
#
# Cluster initialized successfully!
```

The manager node stores cluster state (using Raft) and provides the API for management operations.

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
# List nodes in the cluster
warren node list --manager 127.0.0.1:8080

# Output:
# ID              ROLE      STATUS    ADDRESS         LABELS
# manager-abc123  manager   ready     127.0.0.1:8080
# worker-xyz789   worker    ready     127.0.0.1:0
```

### Step 4: Deploy Service

```bash
# Deploy nginx with 2 replicas
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
# List services
warren service list --manager 127.0.0.1:8080

# Output:
# NAME     IMAGE          REPLICAS  MODE        CREATED
# nginx    nginx:latest   2/2       replicated  30s ago

# Inspect service details
warren service inspect nginx --manager 127.0.0.1:8080

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
# Scale to 4 replicas
warren service scale nginx --replicas 4 --manager 127.0.0.1:8080

# Output:
# ✓ Service 'nginx' scaled to 4 replicas

# Verify scaling
warren service list --manager 127.0.0.1:8080

# Output:
# NAME     IMAGE          REPLICAS  MODE        CREATED
# nginx    nginx:latest   4/4       replicated  2m ago
```

### Step 7: Update Service

```bash
# Update to nginx:alpine
warren service update nginx --image nginx:alpine --manager 127.0.0.1:8080

# Output:
# ✓ Service 'nginx' updated
# ✓ Rolling update in progress (1 task at a time)
```

### Step 8: Clean Up

```bash
# Delete service
warren service delete nginx --manager 127.0.0.1:8080

# Output:
# ✓ Service 'nginx' deleted
# ✓ All tasks stopped and removed

# Stop worker (Ctrl+C in worker terminal)
# Stop manager (Ctrl+C in manager terminal)
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

# Generate join token for additional managers
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

```bash
# Check cluster info (shows Raft quorum)
warren cluster info --manager 192.168.1.10:8080

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

---

## Working with Secrets

```bash
# Create secret from literal
warren secret create db-password \
  --from-literal password=mySecurePassword123 \
  --manager 127.0.0.1:8080

# Create secret from file
warren secret create tls-cert \
  --from-file /path/to/cert.pem \
  --manager 127.0.0.1:8080

# List secrets
warren secret list --manager 127.0.0.1:8080

# Deploy service with secret
warren service create webapp \
  --image myapp:latest \
  --secret db-password \
  --replicas 3 \
  --manager 127.0.0.1:8080

# Secret is mounted at /run/secrets/db-password in container
```

---

## Working with Volumes

```bash
# Create volume
warren volume create db-data \
  --driver local \
  --manager 127.0.0.1:8080

# Deploy service with volume
warren service create postgres \
  --image postgres:15 \
  --volume db-data:/var/lib/postgresql/data \
  --replicas 1 \
  --manager 127.0.0.1:8080

# List volumes
warren volume list --manager 127.0.0.1:8080
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
warren cluster info --manager <addr>         # Show cluster status
warren cluster join-token [manager|worker]   # Generate join token

# Node management
warren node list --manager <addr>            # List all nodes
warren manager join --token <token> ...      # Join as manager
warren worker start --manager <addr>         # Start worker

# Service management
warren service create <name> --image <img>   # Create service
warren service list --manager <addr>         # List services
warren service scale <name> --replicas <n>   # Scale service
warren service update <name> --image <img>   # Update service
warren service delete <name>                 # Delete service

# Secrets
warren secret create <name> --from-literal key=value
warren secret list --manager <addr>

# Volumes
warren volume create <name> --driver local
warren volume list --manager <addr>

# YAML
warren apply -f <file.yaml> --manager <addr>
```

### Environment Variables

```bash
# Set default manager address
export WARREN_MANAGER=127.0.0.1:8080

# Now commands work without --manager flag
warren service list
warren node list
```

---

## Getting Help

- **Documentation**: [https://github.com/cuemby/warren/docs](https://github.com/cuemby/warren/tree/main/docs)
- **GitHub Issues**: [https://github.com/cuemby/warren/issues](https://github.com/cuemby/warren/issues)
- **GitHub Discussions**: [https://github.com/cuemby/warren/discussions](https://github.com/cuemby/warren/discussions)
- **Email**: opensource@cuemby.com

---

**Congratulations!** You've deployed your first service with Warren. Welcome to simple, powerful container orchestration! 🎉
