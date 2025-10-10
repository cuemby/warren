# Warren CLI Reference

Complete command-line reference for Warren container orchestrator.

## Global Flags

These flags work with all commands:

```bash
--manager string       Manager API address (default "127.0.0.1:8080")
--log-level string     Log level: debug, info, warn, error (default "info")
--help, -h             Show help for command
--version              Show Warren version
```

**Environment Variables:**

```bash
export WARREN_MANAGER=192.168.1.10:8080   # Default manager address
export WARREN_LOG_LEVEL=debug              # Default log level
```

---

## warren cluster

Manage Warren clusters.

### warren cluster init

Initialize a new Warren cluster.

**Usage:**
```bash
warren cluster init [flags]
```

**Flags:**
```
--advertise-addr string    IP:port to advertise to other nodes (default: auto-detect)
--listen-addr string       IP:port to listen on (default "0.0.0.0:8080")
--data-dir string          Data directory for Raft and BoltDB (default "/var/lib/warren/data")
--metrics-addr string      Metrics endpoint address (default "127.0.0.1:9090")
--log-level string         Log level: debug, info, warn, error (default "info")
--enable-pprof            Enable profiling endpoints
```

**Examples:**

```bash
# Basic initialization (single node)
sudo warren cluster init

# Initialize with custom address
sudo warren cluster init --advertise-addr 192.168.1.10:8080

# Initialize with metrics on all interfaces
sudo warren cluster init --metrics-addr 0.0.0.0:9090

# Initialize with debug logging and profiling
sudo warren cluster init --log-level debug --enable-pprof
```

**Output:**
```
✓ Raft consensus initialized
✓ Manager started (Node ID: manager-abc123)
✓ API server listening on 127.0.0.1:8080
✓ Metrics available at http://127.0.0.1:9090/metrics

Cluster initialized successfully!
```

---

### warren cluster info

Show cluster information and status.

**Usage:**
```bash
warren cluster info [flags]
```

**Flags:**
```
--manager string    Manager API address
```

**Examples:**

```bash
# Show cluster info
warren cluster info --manager 127.0.0.1:8080

# With environment variable
export WARREN_MANAGER=127.0.0.1:8080
warren cluster info
```

**Output:**
```
Cluster ID: cluster-abc123
Raft Quorum: 3 managers (needs 2 for majority)
Leader: manager-1 (192.168.1.10:8080)

Managers:
  manager-1  192.168.1.10:8080  Leader    ready
  manager-2  192.168.1.11:8080  Follower  ready
  manager-3  192.168.1.12:8080  Follower  ready

Workers: 2 (3 tasks running)
```

---

### warren cluster join-token

Generate join tokens for managers or workers.

**Usage:**
```bash
warren cluster join-token [manager|worker] [flags]
```

**Flags:**
```
--manager string       Manager API address
--ttl duration         Token time-to-live (default 24h)
```

**Examples:**

```bash
# Generate manager join token
warren cluster join-token manager --manager 127.0.0.1:8080

# Generate worker join token
warren cluster join-token worker --manager 127.0.0.1:8080

# Generate token with custom TTL
warren cluster join-token worker --ttl 48h --manager 127.0.0.1:8080
```

**Output:**
```
Join Token (expires in 24h):
SWMTKN-1-5k7j9h3f2d1s9a7k5l3n1m9p7r5t3v1x9z7b5d3f1h9j7k5

On other nodes, run:
warren manager join --token SWMTKN-1-... --manager 192.168.1.10:8080
```

---

## warren manager

Manage manager nodes.

### warren manager join

Join an existing cluster as a manager.

**Usage:**
```bash
warren manager join [flags]
```

**Flags:**
```
--token string             Join token from cluster join-token manager
--manager string           Manager API address to join
--advertise-addr string    IP:port to advertise to other nodes (default: auto-detect)
--listen-addr string       IP:port to listen on (default "0.0.0.0:8080")
--data-dir string          Data directory (default "/var/lib/warren/data")
```

**Examples:**

```bash
# Join as manager
sudo warren manager join \
  --token SWMTKN-1-... \
  --manager 192.168.1.10:8080 \
  --advertise-addr 192.168.1.11:8080
```

**Output:**
```
✓ Connected to cluster leader at 192.168.1.10:8080
✓ Raft joined as follower
✓ Manager started (Node ID: manager-2-def456)
✓ API server listening on 192.168.1.11:8080

Manager joined successfully!
Raft quorum: 2/2 managers (needs 2 for majority)
```

---

## warren worker

Manage worker nodes.

### warren worker start

Start a worker node.

**Usage:**
```bash
warren worker start [flags]
```

**Flags:**
```
--manager string          Manager API address to connect to
--token string            Join token (optional, for authenticated join)
--data-dir string         Data directory (default "/var/lib/warren/worker")
--log-level string        Log level (default "info")
--enable-pprof           Enable profiling endpoints (port 6060)
```

**Examples:**

```bash
# Start worker (no authentication)
sudo warren worker start --manager 192.168.1.10:8080

# Start worker with join token
sudo warren worker start \
  --manager 192.168.1.10:8080 \
  --token SWMTKN-2-...

# Start worker with debug logging
sudo warren worker start \
  --manager 192.168.1.10:8080 \
  --log-level debug
```

**Output:**
```
✓ Worker started (Node ID: worker-xyz789)
✓ Connected to manager at 192.168.1.10:8080
✓ Heartbeat: 30s interval

Worker ready to accept tasks!
```

---

## warren node

Manage cluster nodes.

### warren node list

List all nodes in the cluster.

**Usage:**
```bash
warren node list [flags]
```

**Flags:**
```
--manager string    Manager API address
```

**Examples:**

```bash
warren node list --manager 127.0.0.1:8080
```

**Output:**
```
ID              ROLE      STATUS    ADDRESS           LABELS
manager-abc123  manager   ready     192.168.1.10:8080
manager-def456  manager   ready     192.168.1.11:8080
manager-ghi789  manager   ready     192.168.1.12:8080
worker-xyz111   worker    ready     192.168.1.20:0
worker-xyz222   worker    ready     192.168.1.21:0
```

---

## warren service

Manage services.

### warren service create

Create a new service.

**Usage:**
```bash
warren service create NAME [flags]
```

**Flags:**
```
--image string              Container image (required)
--replicas int              Number of replicas (default 1)
--mode string               Service mode: replicated or global (default "replicated")
--env stringArray           Environment variables (KEY=VALUE)
--secret stringArray        Secrets to mount
--volume stringArray        Volumes to mount (NAME:PATH)
--manager string            Manager API address
```

**Examples:**

```bash
# Basic service
warren service create nginx --image nginx:latest --replicas 3

# Service with environment variables
warren service create webapp \
  --image webapp:v1.0 \
  --replicas 5 \
  --env DATABASE_URL=postgres://db:5432/mydb \
  --env REDIS_URL=redis://cache:6379

# Service with secrets
warren service create api \
  --image api:latest \
  --replicas 3 \
  --secret db-password \
  --secret api-key

# Service with volume
warren service create postgres \
  --image postgres:15 \
  --replicas 1 \
  --volume db-data:/var/lib/postgresql/data \
  --env POSTGRES_PASSWORD=password

# Global service
warren service create node-exporter \
  --image prom/node-exporter:latest \
  --mode global
```

**Output:**
```
✓ Service 'nginx' created
✓ 3 tasks scheduled
Service ID: svc-nginx-123
```

---

### warren service list

List all services.

**Usage:**
```bash
warren service list [flags]
```

**Flags:**
```
--manager string    Manager API address
```

**Examples:**

```bash
warren service list --manager 127.0.0.1:8080
```

**Output:**
```
NAME     IMAGE          REPLICAS  MODE        CREATED
nginx    nginx:latest   3/3       replicated  5m ago
api      api:v1.0       5/5       replicated  2h ago
exporter prom/...:latest 2         global      1d ago
```

---

### warren service inspect

Inspect a service.

**Usage:**
```bash
warren service inspect NAME [flags]
```

**Flags:**
```
--manager string    Manager API address
```

**Examples:**

```bash
warren service inspect nginx --manager 127.0.0.1:8080
```

**Output:**
```
Service: nginx
ID: svc-nginx-123
Mode: replicated
Replicas: 3/3
Image: nginx:latest
Created: 2025-10-10 10:00:00

Environment:
  (none)

Secrets:
  (none)

Volumes:
  (none)

Tasks:
  task-nginx-1  worker-xyz111  running  nginx:latest  10.0.1.5  5m ago
  task-nginx-2  worker-xyz222  running  nginx:latest  10.0.1.6  5m ago
  task-nginx-3  worker-xyz111  running  nginx:latest  10.0.1.7  5m ago
```

---

### warren service scale

Scale a service to a specific number of replicas.

**Usage:**
```bash
warren service scale NAME --replicas N [flags]
```

**Flags:**
```
--replicas int      Number of replicas (required)
--manager string    Manager API address
```

**Examples:**

```bash
# Scale up
warren service scale nginx --replicas 5

# Scale down
warren service scale nginx --replicas 2
```

**Output:**
```
✓ Service 'nginx' scaled to 5 replicas
```

---

### warren service update

Update a service.

**Usage:**
```bash
warren service update NAME [flags]
```

**Flags:**
```
--image string              New container image
--replicas int              New replica count (same as scale)
--env stringArray           New environment variables
--manager string            Manager API address
```

**Examples:**

```bash
# Update image
warren service update nginx --image nginx:alpine

# Update environment
warren service update webapp --env NEW_VAR=value

# Update image and replicas
warren service update api --image api:v2.0 --replicas 10
```

**Output:**
```
✓ Service 'nginx' updated
✓ Rolling update in progress
```

---

### warren service delete

Delete a service.

**Usage:**
```bash
warren service delete NAME [flags]
```

**Flags:**
```
--manager string    Manager API address
```

**Examples:**

```bash
warren service delete nginx --manager 127.0.0.1:8080
```

**Output:**
```
✓ Service 'nginx' deleted
✓ All tasks stopped and removed
```

---

## warren secret

Manage secrets.

### warren secret create

Create a new secret.

**Usage:**
```bash
warren secret create NAME [flags]
```

**Flags:**
```
--from-literal string    Secret value as literal (KEY=VALUE)
--from-file string       Secret value from file
--from-stdin            Read secret value from stdin
--manager string        Manager API address
```

**Examples:**

```bash
# From literal
warren secret create db-password --from-literal password=secret123

# From file
warren secret create tls-cert --from-file /path/to/cert.pem

# From stdin
echo -n "mySecretValue" | warren secret create api-key --from-stdin
```

**Output:**
```
✓ Secret 'db-password' created
Secret ID: secret-db-password-abc123
```

---

### warren secret list

List all secrets.

**Usage:**
```bash
warren secret list [flags]
```

**Flags:**
```
--manager string    Manager API address
```

**Examples:**

```bash
warren secret list --manager 127.0.0.1:8080
```

**Output:**
```
NAME          SIZE    CREATED
db-password   23B     2d ago
api-key       64B     1h ago
tls-cert      1.2KB   5h ago
```

---

### warren secret inspect

Inspect a secret (metadata only).

**Usage:**
```bash
warren secret inspect NAME [flags]
```

**Flags:**
```
--manager string    Manager API address
```

**Examples:**

```bash
warren secret inspect db-password --manager 127.0.0.1:8080
```

**Output:**
```
Name: db-password
ID: secret-db-password-abc123
Size: 23 bytes
Created: 2025-10-08 10:00:00
Used By:
  - postgres (service)
  - api (service)
```

**Note:** Secret values are never displayed.

---

### warren secret delete

Delete a secret.

**Usage:**
```bash
warren secret delete NAME [flags]
```

**Flags:**
```
--force             Force delete (remove from using services)
--manager string    Manager API address
```

**Examples:**

```bash
warren secret delete db-password --manager 127.0.0.1:8080

# Force delete
warren secret delete db-password --force --manager 127.0.0.1:8080
```

**Output:**
```
✓ Secret 'db-password' deleted
```

---

## warren volume

Manage volumes.

### warren volume create

Create a new volume.

**Usage:**
```bash
warren volume create NAME [flags]
```

**Flags:**
```
--driver string         Volume driver (default "local")
--label stringArray     Labels (KEY=VALUE)
--manager string        Manager API address
```

**Examples:**

```bash
# Basic volume
warren volume create db-data --driver local

# Volume with labels
warren volume create app-data \
  --driver local \
  --label env=production \
  --label app=webapp
```

**Output:**
```
✓ Volume 'db-data' created
Volume ID: vol-db-data-abc123
```

---

### warren volume list

List all volumes.

**Usage:**
```bash
warren volume list [flags]
```

**Flags:**
```
--manager string    Manager API address
```

**Examples:**

```bash
warren volume list --manager 127.0.0.1:8080
```

**Output:**
```
NAME      DRIVER  SIZE    NODE       CREATED
db-data   local   1.2GB   worker-1   2d ago
app-data  local   500MB   worker-2   5h ago
```

---

### warren volume inspect

Inspect a volume.

**Usage:**
```bash
warren volume inspect NAME [flags]
```

**Flags:**
```
--manager string    Manager API address
```

**Examples:**

```bash
warren volume inspect db-data --manager 127.0.0.1:8080
```

**Output:**
```
Name: db-data
ID: vol-db-data-abc123
Driver: local
Mount Point: /var/lib/warren/volumes/db-data
Node: worker-1
Size: 1.2 GB
Used By:
  - postgres (service)
Created: 2025-10-08 10:00:00
```

---

### warren volume delete

Delete a volume.

**Usage:**
```bash
warren volume delete NAME [flags]
```

**Flags:**
```
--force             Force delete (stop using services)
--manager string    Manager API address
```

**Examples:**

```bash
warren volume delete db-data --manager 127.0.0.1:8080

# Force delete
warren volume delete db-data --force --manager 127.0.0.1:8080
```

**Output:**
```
✓ Volume 'db-data' deleted
```

---

## warren apply

Apply configuration from YAML files.

**Usage:**
```bash
warren apply -f FILE [flags]
```

**Flags:**
```
-f, --file stringArray    YAML file(s) or directory
--manager string          Manager API address
```

**Examples:**

```bash
# Apply single file
warren apply -f service.yaml

# Apply multiple files
warren apply -f service.yaml -f secret.yaml -f volume.yaml

# Apply directory
warren apply -f ./manifests/

# Apply from stdin
cat service.yaml | warren apply -f -
```

**Output:**
```
✓ Secret 'db-password' created
✓ Volume 'db-data' created
✓ Service 'postgres' created
```

---

## warren version

Show Warren version.

**Usage:**
```bash
warren version
```

**Output:**
```
warren version 1.0.0
commit: abc1234
built: 2025-10-10T10:00:00Z
go: go1.22.0
platform: linux/arm64
```

---

## Tab Completion

Generate shell completion scripts.

### Bash

```bash
# Generate completion script
warren completion bash > /etc/bash_completion.d/warren

# Or add to ~/.bashrc
warren completion bash >> ~/.bashrc
source ~/.bashrc
```

### Zsh

```bash
# Generate completion script
warren completion zsh > /usr/local/share/zsh/site-functions/_warren

# Or add to ~/.zshrc
warren completion zsh >> ~/.zshrc
source ~/.zshrc
```

### Fish

```bash
warren completion fish > ~/.config/fish/completions/warren.fish
```

### PowerShell

```powershell
warren completion powershell >> $PROFILE
```

---

## Exit Codes

Warren uses standard exit codes:

- **0** - Success
- **1** - General error
- **2** - Invalid arguments
- **3** - Connection error (cannot reach manager)
- **4** - Not found (service, secret, volume, etc.)
- **5** - Already exists (name conflict)
- **6** - Permission denied

---

## Examples by Use Case

### Deploy a Multi-Tier Web Application

```bash
# 1. Create secrets
warren secret create db-password --from-literal password=secret123
warren secret create api-key --from-literal key=abc123xyz

# 2. Create volumes
warren volume create db-data
warren volume create redis-data

# 3. Deploy database
warren service create postgres \
  --image postgres:15 \
  --replicas 1 \
  --volume db-data:/var/lib/postgresql/data \
  --secret db-password \
  --env POSTGRES_PASSWORD_FILE=/run/secrets/db-password

# 4. Deploy cache
warren service create redis \
  --image redis:7 \
  --replicas 1 \
  --volume redis-data:/data

# 5. Deploy API
warren service create api \
  --image myapi:v1.0 \
  --replicas 5 \
  --secret db-password \
  --secret api-key \
  --env DATABASE_URL=postgres://postgres@postgres:5432/mydb \
  --env REDIS_URL=redis://redis:6379

# 6. Deploy frontend
warren service create frontend \
  --image frontend:v1.0 \
  --replicas 3 \
  --env API_URL=http://api:8080
```

### Deploy Monitoring Stack

```bash
# 1. Create volumes
warren volume create prometheus-data
warren volume create grafana-data

# 2. Deploy Prometheus
warren service create prometheus \
  --image prom/prometheus:latest \
  --replicas 1 \
  --volume prometheus-data:/prometheus

# 3. Deploy node-exporter (global)
warren service create node-exporter \
  --image prom/node-exporter:latest \
  --mode global

# 4. Deploy Grafana
warren service create grafana \
  --image grafana/grafana:latest \
  --replicas 1 \
  --volume grafana-data:/var/lib/grafana \
  --env GF_SECURITY_ADMIN_PASSWORD=admin
```

---

## See Also

- [Getting Started](getting-started.md) - Quick start guide
- [Concepts](concepts/architecture.md) - Architecture and concepts
- [Migration from Docker Swarm](migration/from-docker-swarm.md)
- [Migration from Docker Compose](migration/from-docker-compose.md)
- [Troubleshooting](troubleshooting.md)

---

**Questions?** Ask in [GitHub Discussions](https://github.com/cuemby/warren/discussions) or email opensource@cuemby.com
