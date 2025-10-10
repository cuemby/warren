# Migrating from Docker Swarm to Warren

This guide helps you migrate from Docker Swarm to Warren with minimal disruption.

## Why Migrate?

- **Docker Swarm is now closed source** - No longer open-source community development
- **Warren is actively developed** - Open-source, modern architecture
- **Similar API** - Warren concepts map directly to Swarm
- **More features** - Built-in metrics, structured logging, edge optimizations
- **Better performance** - Faster failover (~2-3s vs 10-15s)

## Conceptual Mapping

### Core Concepts

| Docker Swarm | Warren | Notes |
|--------------|--------|-------|
| **Stack** | Collection of Services | Use `warren apply` for multi-service deployments |
| **Service** | Service | Direct equivalent |
| **Task** | Task | Direct equivalent |
| **Replicated Service** | Replicated Service | Same concept (N replicas) |
| **Global Service** | Global Service | Same concept (1 per node) |
| **Manager Node** | Manager Node | Raft-based consensus |
| **Worker Node** | Worker Node | Executes containers |
| **Secret** | Secret | AES-256-GCM encrypted |
| **Volume** | Volume | Local driver (NFS/Ceph coming) |
| **Network** | WireGuard Overlay | Encrypted by default |
| **Service VIP** | Service VIP | Load-balanced virtual IP |

### Commands

| Docker Swarm | Warren | Notes |
|--------------|--------|-------|
| `docker swarm init` | `warren cluster init` | Initialize cluster |
| `docker swarm join-token` | `warren cluster join-token` | Generate join token |
| `docker swarm join` | `warren manager/worker join` | Join cluster |
| `docker node ls` | `warren node list` | List nodes |
| `docker service create` | `warren service create` | Create service |
| `docker service ls` | `warren service list` | List services |
| `docker service ps` | `warren service inspect` | Inspect service tasks |
| `docker service scale` | `warren service scale` | Scale service |
| `docker service update` | `warren service update` | Update service |
| `docker service rm` | `warren service delete` | Remove service |
| `docker secret create` | `warren secret create` | Create secret |
| `docker secret ls` | `warren secret list` | List secrets |
| `docker volume create` | `warren volume create` | Create volume |
| `docker stack deploy` | `warren apply -f` | Deploy stack |

## Migration Strategy

### Option 1: Blue/Green Migration (Recommended)

Deploy Warren alongside Swarm, migrate workloads gradually.

**Timeline**: 1-2 weeks
**Downtime**: None
**Risk**: Low

**Steps:**

1. **Set up Warren cluster** (separate from Swarm)
2. **Migrate one service at a time**
3. **Validate each migration**
4. **Update DNS/load balancers**
5. **Decommission Swarm when complete**

### Option 2: In-Place Migration

Replace Swarm with Warren on same infrastructure.

**Timeline**: 1-2 days
**Downtime**: ~1-2 hours
**Risk**: Medium

**Steps:**

1. **Backup Swarm state**
2. **Document current services**
3. **Schedule maintenance window**
4. **Stop Swarm**
5. **Initialize Warren**
6. **Recreate services**
7. **Validate and restore traffic**

## Step-by-Step Migration

### Phase 1: Preparation

#### 1. Inventory Swarm Resources

```bash
# List all services
docker service ls > swarm-services.txt

# Inspect each service (capture config)
for svc in $(docker service ls --format '{{.Name}}'); do
  docker service inspect $svc > swarm-service-$svc.json
done

# List secrets
docker secret ls > swarm-secrets.txt

# List volumes
docker volume ls > swarm-volumes.txt

# List networks
docker network ls > swarm-networks.txt
```

#### 2. Document Dependencies

Create a dependency map:

```
frontend → api → database
         ↘     ↗
          cache
```

**Migration Order**: database → cache → api → frontend

#### 3. Install Warren

On new nodes or existing (after Swarm removal):

```bash
# Download Warren
curl -LO https://github.com/cuemby/warren/releases/latest/download/warren-linux-amd64
chmod +x warren-linux-amd64
sudo mv warren-linux-amd64 /usr/local/bin/warren

# Verify installation
warren --version
```

### Phase 2: Initialize Warren Cluster

#### 1. Initialize First Manager

```bash
# On manager-1
sudo warren cluster init --advertise-addr 192.168.1.10:8080
```

#### 2. Add Additional Managers

```bash
# Generate token on manager-1
warren cluster join-token manager --manager 192.168.1.10:8080

# On manager-2
sudo warren manager join \
  --token SWMTKN-1-... \
  --manager 192.168.1.10:8080 \
  --advertise-addr 192.168.1.11:8080

# On manager-3
sudo warren manager join \
  --token SWMTKN-1-... \
  --manager 192.168.1.10:8080 \
  --advertise-addr 192.168.1.12:8080
```

#### 3. Add Workers

```bash
# Generate worker token
warren cluster join-token worker --manager 192.168.1.10:8080

# On each worker
sudo warren worker start \
  --manager 192.168.1.10:8080 \
  --token SWMTKN-2-...
```

### Phase 3: Migrate Resources

#### 1. Migrate Secrets

```bash
# Swarm
docker secret create db-password /path/to/password.txt

# Warren
warren secret create db-password --from-file /path/to/password.txt
```

**Bulk Migration Script:**

```bash
#!/bin/bash
# migrate-secrets.sh

for secret in $(docker secret ls --format '{{.Name}}'); do
  echo "Migrating secret: $secret"

  # Export secret value (requires Swarm access)
  docker secret inspect $secret --pretty > /tmp/$secret.txt

  # Create in Warren
  warren secret create $secret --from-file /tmp/$secret.txt

  # Clean up
  rm /tmp/$secret.txt
done
```

#### 2. Migrate Volumes

```bash
# For local volumes, data must be copied manually

# Swarm volume location
/var/lib/docker/volumes/<volume-name>/_data/

# Warren volume location
/var/lib/warren/volumes/<volume-name>/

# Migration:
sudo rsync -av \
  /var/lib/docker/volumes/db-data/_data/ \
  /var/lib/warren/volumes/db-data/
```

#### 3. Migrate Services

**Swarm Service:**

```bash
docker service create \
  --name nginx \
  --replicas 3 \
  --publish 80:80 \
  --env NGINX_PORT=80 \
  --secret db-password \
  --mount type=volume,source=logs,target=/var/log/nginx \
  nginx:latest
```

**Warren Service:**

```bash
# Create volume first
warren volume create logs

# Create service
warren service create nginx \
  --image nginx:latest \
  --replicas 3 \
  --env NGINX_PORT=80 \
  --secret db-password \
  --volume logs:/var/log/nginx
```

**Note**: Warren doesn't support `--publish` yet (M6). Use WireGuard VIP for internal access.

### Phase 4: Convert Docker Compose Stacks

**Swarm Stack (docker-compose.yml):**

```yaml
version: "3.8"

services:
  web:
    image: nginx:latest
    replicas: 3
    ports:
      - "80:80"
    environment:
      NGINX_PORT: 80
    secrets:
      - db-password
    volumes:
      - logs:/var/log/nginx

secrets:
  db-password:
    external: true

volumes:
  logs:
    driver: local
```

**Warren YAML (warren.yaml):**

```yaml
apiVersion: warren.io/v1
kind: Secret
metadata:
  name: db-password
spec:
  external: true
---
apiVersion: warren.io/v1
kind: Volume
metadata:
  name: logs
spec:
  driver: local
---
apiVersion: warren.io/v1
kind: Service
metadata:
  name: web
spec:
  image: nginx:latest
  replicas: 3
  mode: replicated
  env:
    - name: NGINX_PORT
      value: "80"
  secrets:
    - db-password
  volumes:
    - name: logs
      mountPath: /var/log/nginx
```

**Deploy:**

```bash
warren apply -f warren.yaml --manager 192.168.1.10:8080
```

### Phase 5: Validation

#### 1. Verify Services

```bash
# List services
warren service list

# Inspect each service
warren service inspect nginx

# Check task status
warren service inspect nginx | grep -A 10 "Tasks:"
```

#### 2. Test Connectivity

```bash
# From within cluster
curl http://<service-vip>

# Check service endpoints
warren service inspect nginx
```

#### 3. Monitor Logs

```bash
# Warren structured logs
journalctl -u warren-manager -f
journalctl -u warren-worker -f

# Check for errors
journalctl -u warren-manager | grep ERROR
```

### Phase 6: Cutover

#### 1. Update DNS/Load Balancers

```bash
# Old: Point to Swarm VIPs
A record: app.example.com → 10.255.0.5 (Swarm)

# New: Point to Warren VIPs
A record: app.example.com → 10.1.0.5 (Warren)
```

#### 2. Drain Traffic from Swarm

```bash
# Swarm: Scale down services
docker service scale nginx --replicas 0

# Or remove services
docker service rm nginx
```

#### 3. Decommission Swarm

```bash
# On each worker
docker swarm leave

# On managers
docker swarm leave --force
```

## Feature Comparison

### Supported in Warren

- ✅ Replicated services
- ✅ Global services
- ✅ Service scaling
- ✅ Rolling updates
- ✅ Secrets (AES-256-GCM)
- ✅ Volumes (local driver)
- ✅ Service VIPs
- ✅ Multi-manager HA
- ✅ Overlay networking (WireGuard)
- ✅ YAML declarative config

### Not Yet Supported (Roadmap)

- 🔜 Published ports (M6)
- 🔜 Health checks (M6)
- 🔜 Service logs aggregation (M6)
- 🔜 Blue/green deployments (implemented, needs docs)
- 🔜 Canary deployments (implemented, needs docs)
- 🔜 Custom networks (M6)
- 🔜 Service constraints (node labels) (M6)
- 🔜 Resource limits (CPU/memory) (M6)

### Warren Advantages

- ✅ **Open source** - Active community development
- ✅ **Faster failover** - 2-3s vs 10-15s (Swarm)
- ✅ **Built-in metrics** - Prometheus endpoint included
- ✅ **Structured logging** - JSON logs with context
- ✅ **Edge optimized** - Partition tolerance, low resource usage
- ✅ **Encrypted by default** - WireGuard overlay
- ✅ **Smaller binary** - 35MB vs 50MB

## Common Migration Issues

### Issue 1: Published Ports Not Supported

**Swarm:**
```bash
docker service create --publish 80:80 nginx
```

**Workaround:**
- Use external load balancer pointing to Warren nodes
- Access via service VIP from within cluster
- Wait for M6 (port publishing support)

### Issue 2: Custom Networks

**Swarm:**
```bash
docker network create --driver overlay mynet
docker service create --network mynet nginx
```

**Warren:**
- Single WireGuard overlay for entire cluster
- All services can communicate
- Custom networks coming in M6

### Issue 3: Placement Constraints

**Swarm:**
```bash
docker service create --constraint 'node.labels.type==database' postgres
```

**Warren:**
- Automatic node affinity for volumes
- Manual constraints coming in M6

### Issue 4: Health Checks

**Swarm:**
```bash
docker service create \
  --health-cmd "curl -f http://localhost/health" \
  --health-interval 30s \
  nginx
```

**Warren:**
- Health checks coming in M6
- Currently relies on container exit code

## Rollback Plan

If migration fails:

### 1. Keep Swarm Running

During blue/green migration, Swarm remains operational:

```bash
# Switch DNS back to Swarm
A record: app.example.com → <swarm-vip>

# Scale up Swarm services
docker service scale nginx --replicas 3
```

### 2. Backup Warren State

```bash
# On each manager
sudo systemctl stop warren-manager
sudo cp /var/lib/warren/data/warren.db /backup/
sudo systemctl start warren-manager
```

### 3. Document Issues

Create detailed incident report:

- What failed?
- Error messages
- Logs
- Configuration

Share with Warren community for support.

## Migration Checklist

- [ ] **Pre-Migration**
  - [ ] Inventory Swarm resources
  - [ ] Document dependencies
  - [ ] Test Warren on staging
  - [ ] Create migration scripts
  - [ ] Schedule maintenance window
  - [ ] Communicate to stakeholders

- [ ] **Migration**
  - [ ] Install Warren on nodes
  - [ ] Initialize cluster (managers + workers)
  - [ ] Migrate secrets
  - [ ] Migrate volumes (copy data)
  - [ ] Migrate services (one by one)
  - [ ] Validate each service

- [ ] **Cutover**
  - [ ] Update DNS/load balancers
  - [ ] Monitor for errors
  - [ ] Validate end-to-end
  - [ ] Drain Swarm traffic

- [ ] **Post-Migration**
  - [ ] Monitor Warren metrics
  - [ ] Document changes
  - [ ] Decommission Swarm (after 7 days)
  - [ ] Update runbooks

## Getting Help

- **Documentation**: [Warren Docs](https://github.com/cuemby/warren/docs)
- **GitHub Discussions**: [Community Q&A](https://github.com/cuemby/warren/discussions)
- **GitHub Issues**: [Bug Reports](https://github.com/cuemby/warren/issues)
- **Email**: opensource@cuemby.com

---

**Migration successful?** Share your story in [GitHub Discussions](https://github.com/cuemby/warren/discussions)!
