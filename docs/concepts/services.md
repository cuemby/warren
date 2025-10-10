# Services in Warren

A **service** is Warren's primary abstraction for running containerized applications. This guide explains service concepts, deployment modes, and lifecycle management.

## What is a Service?

A service defines how your application should run across the cluster:

- **What** - Container image to run (`nginx:latest`)
- **How many** - Number of replicas or deployment mode
- **Where** - Node placement constraints (optional)
- **Configuration** - Environment variables, secrets, volumes

**Key Concept**: A service is the desired state. Warren continuously reconciles to maintain this state.

## Service vs Task vs Container

```
Service (desired state)
   └── Task 1 (scheduled work)
          └── Container (running process)
   └── Task 2
          └── Container
   └── Task 3
          └── Container
```

- **Service**: High-level definition ("run 3 nginx instances")
- **Task**: Scheduled unit of work ("run nginx on worker-1")
- **Container**: Actual running process (containerd container)

## Service Modes

Warren supports two service modes:

### 1. Replicated Mode (default)

Deploy a specific number of task replicas across the cluster.

```bash
# Deploy 5 replicas of nginx
warren service create nginx \
  --image nginx:latest \
  --replicas 5 \
  --manager 127.0.0.1:8080
```

**Use Cases:**
- Web applications (load-balanced)
- API servers
- Background workers
- Stateless services

**Scheduling**: Warren distributes replicas across available workers using round-robin.

### 2. Global Mode

Deploy exactly one task on every worker node.

```bash
# Deploy monitoring agent on all nodes
warren service create node-exporter \
  --image prom/node-exporter:latest \
  --mode global \
  --manager 127.0.0.1:8080
```

**Use Cases:**
- Monitoring agents (node-exporter, datadog-agent)
- Log collectors (fluentd, filebeat)
- Security agents
- Local caching layers

**Scheduling**: Warren automatically adds/removes tasks as nodes join/leave the cluster.

## Creating Services

### Basic Service

```bash
warren service create myapp \
  --image myapp:v1.0 \
  --replicas 3 \
  --manager 127.0.0.1:8080
```

### Service with Environment Variables

```bash
warren service create webapp \
  --image webapp:latest \
  --replicas 3 \
  --env DATABASE_URL=postgres://db:5432/mydb \
  --env REDIS_URL=redis://cache:6379 \
  --manager 127.0.0.1:8080
```

### Service with Secrets

```bash
# Create secret first
warren secret create db-password \
  --from-literal password=secret123 \
  --manager 127.0.0.1:8080

# Deploy service with secret
warren service create api \
  --image api:latest \
  --replicas 5 \
  --secret db-password \
  --env DB_PASSWORD_FILE=/run/secrets/db-password \
  --manager 127.0.0.1:8080
```

### Service with Volumes

```bash
# Create volume first
warren volume create app-data \
  --driver local \
  --manager 127.0.0.1:8080

# Deploy stateful service
warren service create postgres \
  --image postgres:15 \
  --replicas 1 \
  --volume app-data:/var/lib/postgresql/data \
  --env POSTGRES_PASSWORD=password \
  --manager 127.0.0.1:8080
```

### Global Service

```bash
warren service create logging-agent \
  --image fluentd:latest \
  --mode global \
  --volume /var/log:/host/logs:ro \
  --manager 127.0.0.1:8080
```

## Service Lifecycle

### 1. Creation

```bash
warren service create nginx --image nginx:latest --replicas 3
```

**What Happens:**
1. API server validates request
2. Manager stores service definition in Raft
3. Scheduler creates 3 tasks, assigns to workers
4. Workers poll and fetch assigned tasks
5. Workers pull image and start containers
6. Service enters "running" state

### 2. Inspection

```bash
# List all services
warren service list --manager 127.0.0.1:8080

# Inspect specific service
warren service inspect nginx --manager 127.0.0.1:8080
```

**Output:**
```
Service: nginx
ID: svc-nginx-abc123
Mode: replicated
Replicas: 3/3
Image: nginx:latest
Created: 2025-10-10 10:00:00

Tasks:
  task-nginx-1  worker-1  running  nginx:latest  10.0.1.5  5m ago
  task-nginx-2  worker-2  running  nginx:latest  10.0.1.6  5m ago
  task-nginx-3  worker-3  running  nginx:latest  10.0.1.7  5m ago
```

### 3. Scaling

```bash
# Scale up
warren service scale nginx --replicas 5 --manager 127.0.0.1:8080

# Scale down
warren service scale nginx --replicas 2 --manager 127.0.0.1:8080
```

**What Happens (Scale Up):**
1. Update service replicas in Raft
2. Scheduler creates 2 new tasks
3. Assign tasks to workers (round-robin)
4. Workers start new containers
5. Service now has 5 running tasks

**What Happens (Scale Down):**
1. Update service replicas in Raft
2. Reconciler identifies excess tasks
3. Stop and remove 1 task (graceful shutdown)
4. Service now has 2 running tasks

### 4. Updating

```bash
# Update image
warren service update nginx \
  --image nginx:alpine \
  --manager 127.0.0.1:8080

# Update environment
warren service update webapp \
  --env NEW_VAR=value \
  --manager 127.0.0.1:8080
```

**Update Strategy: Rolling Update**

Warren performs rolling updates by default:

```
1. Stop task 1
2. Create new task 1 with updated spec
3. Wait for new task 1 healthy
4. Repeat for task 2, 3, ...
```

**Zero-downtime**: Old tasks keep running while new tasks start.

### 5. Deletion

```bash
warren service delete nginx --manager 127.0.0.1:8080
```

**What Happens:**
1. Mark service for deletion in Raft
2. Reconciler stops all tasks
3. Workers stop and remove containers
4. Delete service from storage
5. Service fully removed

## Deployment Strategies

Warren supports multiple deployment strategies (M3 feature):

### 1. Rolling Update (default)

Update tasks one at a time (or in batches).

```bash
warren service update myapp \
  --image myapp:v2.0 \
  --strategy rolling \
  --manager 127.0.0.1:8080
```

**Pros:**
- Zero downtime
- Gradual rollout
- Easy rollback

**Cons:**
- Slower (sequential)
- Mixed versions during update

### 2. Blue/Green Deployment

Deploy full new version alongside old, then switch traffic.

```bash
warren service update myapp \
  --image myapp:v2.0 \
  --strategy blue-green \
  --manager 127.0.0.1:8080
```

**Flow:**
```
1. Deploy "green" (v2.0) with full replicas
2. Wait for all green tasks healthy
3. Switch VIP from "blue" (v1.0) to "green"
4. Delete blue tasks after grace period
```

**Pros:**
- Instant cutover
- Easy rollback (keep blue running)
- No mixed versions

**Cons:**
- Requires 2x resources
- All-or-nothing switch

### 3. Canary Deployment

Deploy new version to subset of traffic, gradually increase.

```bash
warren service update myapp \
  --image myapp:v2.0 \
  --strategy canary \
  --canary-weight 10 \
  --manager 127.0.0.1:8080
```

**Flow:**
```
1. Deploy canary tasks (10% of replicas)
2. Route 10% traffic to canary
3. Monitor metrics, errors
4. Increase canary weight: 10% → 50% → 100%
5. Delete old tasks
```

**Pros:**
- Gradual rollout with validation
- Minimize blast radius
- Rollback at any stage

**Cons:**
- Complex traffic routing
- Requires monitoring

## Service Constraints

### Node Affinity

Pin service to specific nodes (useful for volumes):

```bash
# Service automatically pinned to node with volume
warren service create postgres \
  --image postgres:15 \
  --replicas 1 \
  --volume db-data:/var/lib/postgresql/data \
  --manager 127.0.0.1:8080
```

Warren enforces node affinity for volumes automatically.

### Resource Limits (Future)

```bash
# Coming in M6
warren service create webapp \
  --image webapp:latest \
  --replicas 3 \
  --cpu-limit 1.0 \
  --memory-limit 512M \
  --manager 127.0.0.1:8080
```

## Service Discovery

### Internal DNS (Future)

```bash
# Access service by name (M6)
curl http://nginx.warren.local
```

### VIP (Current)

Each service gets a virtual IP:

```bash
# Service "nginx" → VIP 10.1.0.5
# Traffic load-balanced to all replicas
```

Access via VIP from any worker node in the cluster.

## Health Checks (Future)

Warren will support health checks in M6:

```bash
warren service create webapp \
  --image webapp:latest \
  --replicas 3 \
  --health-cmd "curl -f http://localhost/health" \
  --health-interval 10s \
  --health-timeout 5s \
  --health-retries 3 \
  --manager 127.0.0.1:8080
```

**Unhealthy tasks** are automatically stopped and rescheduled.

## Service Logs (Future)

Aggregate logs from all tasks:

```bash
# Coming in M6
warren service logs nginx --follow --manager 127.0.0.1:8080
```

## Best Practices

### 1. Use Replicated Mode for Stateless Apps

```bash
# Good: Load-balanced web app
warren service create api --image api:latest --replicas 5
```

### 2. Use Global Mode for System Services

```bash
# Good: Monitoring on every node
warren service create node-exporter --mode global
```

### 3. Pin Stateful Services to Single Replica

```bash
# Good: Single database with persistent volume
warren service create postgres --replicas 1 --volume db-data:/var/lib/postgresql/data
```

Avoid multiple replicas for stateful services without replication support.

### 4. Use Secrets for Sensitive Data

```bash
# Bad: Password in environment variable
warren service create app --env DB_PASSWORD=secret123

# Good: Password in secret
warren secret create db-password --from-literal password=secret123
warren service create app --secret db-password
```

### 5. Tag Images with Versions

```bash
# Bad: :latest (ambiguous)
warren service create app --image app:latest

# Good: Explicit version
warren service create app --image app:v1.2.3
```

### 6. Test Updates on Staging First

```bash
# Deploy to staging cluster
warren service create app --image app:v2.0 --manager staging:8080

# After validation, deploy to production
warren service update app --image app:v2.0 --manager production:8080
```

### 7. Monitor Service Health

```bash
# Check service status regularly
warren service list --manager 127.0.0.1:8080

# Inspect specific service
warren service inspect app --manager 127.0.0.1:8080

# Check metrics endpoint
curl http://manager-ip:9090/metrics | grep warren_service_
```

## Common Patterns

### Web Application (Multi-Tier)

```bash
# Backend API
warren service create api \
  --image api:v1.0 \
  --replicas 5 \
  --env DATABASE_URL=postgres://db:5432/mydb \
  --secret db-password

# Frontend
warren service create frontend \
  --image frontend:v1.0 \
  --replicas 3 \
  --env API_URL=http://api.warren.local

# Database
warren volume create db-data
warren service create postgres \
  --image postgres:15 \
  --replicas 1 \
  --volume db-data:/var/lib/postgresql/data \
  --secret db-password
```

### Background Worker

```bash
warren service create worker \
  --image worker:v1.0 \
  --replicas 10 \
  --env QUEUE_URL=redis://queue:6379 \
  --secret api-key
```

### Monitoring Stack

```bash
# Prometheus (metrics)
warren volume create prometheus-data
warren service create prometheus \
  --image prom/prometheus:latest \
  --replicas 1 \
  --volume prometheus-data:/prometheus

# Node exporter (global)
warren service create node-exporter \
  --image prom/node-exporter:latest \
  --mode global

# Grafana (visualization)
warren volume create grafana-data
warren service create grafana \
  --image grafana/grafana:latest \
  --replicas 1 \
  --volume grafana-data:/var/lib/grafana
```

## Troubleshooting

### Service Not Starting

```bash
# Check service status
warren service inspect myapp --manager 127.0.0.1:8080

# Common issues:
# - Image not found → Check image name/tag
# - Task failed → Check container logs (future feature)
# - No available workers → Add workers or reduce replicas
```

### Slow Updates

```bash
# Check reconciler status
# Rolling updates are intentionally sequential for safety

# To speed up (future):
warren service update myapp --parallel 2 --delay 5s
```

### Service Unreachable

```bash
# Check if tasks are running
warren service inspect myapp

# Check VIP assignment (future CLI command)
# Check WireGuard connectivity
```

## YAML Configuration

Services can be defined declaratively:

**service.yaml:**

```yaml
apiVersion: warren.io/v1
kind: Service
metadata:
  name: webapp
spec:
  image: webapp:v1.0
  replicas: 5
  mode: replicated
  env:
    - name: DATABASE_URL
      value: postgres://db:5432/mydb
    - name: REDIS_URL
      value: redis://cache:6379
  secrets:
    - db-password
  volumes:
    - app-data:/app/data
```

**Apply:**

```bash
warren apply -f service.yaml --manager 127.0.0.1:8080
```

## Next Steps

- **[Networking](networking.md)** - Service VIPs and load balancing
- **[Storage](storage.md)** - Volumes for stateful services
- **[High Availability](high-availability.md)** - Service resilience
- **[CLI Reference](../cli-reference.md)** - Complete service commands

---

**Questions?** See [Troubleshooting](../troubleshooting.md) or ask in [GitHub Discussions](https://github.com/cuemby/warren/discussions).
