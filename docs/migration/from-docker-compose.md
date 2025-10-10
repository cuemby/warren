# Migrating from Docker Compose to Warren

This guide helps you convert Docker Compose applications to Warren for production orchestration.

## When to Migrate

**Use Docker Compose for:**
- Local development
- Single-node deployments
- Testing environments

**Use Warren for:**
- Multi-node production clusters
- High availability requirements
- Auto-scaling and self-healing
- Service discovery and load balancing
- Distributed edge deployments

## Key Differences

| Feature | Docker Compose | Warren |
|---------|----------------|--------|
| **Deployment** | Single node | Multi-node cluster |
| **High Availability** | No | Yes (multi-manager) |
| **Auto-healing** | No | Yes (reconciler) |
| **Scaling** | Manual | Manual (auto-scaling M6) |
| **Load Balancing** | No | Yes (service VIPs) |
| **Secrets** | File-based | Encrypted (AES-256-GCM) |
| **State Management** | Restart policy | Desired state reconciliation |

## Conceptual Mapping

| Docker Compose | Warren | Notes |
|----------------|--------|-------|
| **Service** | Service | Direct mapping |
| **Container** | Task | Task → Container |
| **Replicas** | Replicas | Same concept |
| **Ports** | Service VIP | No host port publishing yet (M6) |
| **Environment** | Environment | Same format |
| **Secrets** | Secrets | Encrypted at rest |
| **Volumes** | Volumes | Local driver (NFS coming M7) |
| **Networks** | WireGuard Overlay | Single encrypted overlay |
| **Depends On** | Manual ordering | No dependency management (M6) |
| **Health Check** | Coming M6 | Container exit code currently |

## Conversion Process

### Step 1: Analyze Compose File

Take this example Docker Compose application:

**docker-compose.yml:**

```yaml
version: "3.8"

services:
  frontend:
    image: frontend:v1.0
    ports:
      - "80:80"
    environment:
      API_URL: http://api:8080
    depends_on:
      - api
    deploy:
      replicas: 3

  api:
    image: api:v1.0
    ports:
      - "8080:8080"
    environment:
      DATABASE_URL: postgres://db:5432/mydb
      REDIS_URL: redis://cache:6379
    secrets:
      - db_password
    depends_on:
      - db
      - cache
    deploy:
      replicas: 5

  db:
    image: postgres:15
    environment:
      POSTGRES_PASSWORD_FILE: /run/secrets/db_password
    secrets:
      - db_password
    volumes:
      - db_data:/var/lib/postgresql/data

  cache:
    image: redis:7
    volumes:
      - redis_data:/data

secrets:
  db_password:
    file: ./secrets/db_password.txt

volumes:
  db_data:
  redis_data:
```

### Step 2: Create Warren Resources

#### 1. Create Secrets

```bash
# Create secret from file
warren secret create db_password \
  --from-file ./secrets/db_password.txt \
  --manager 127.0.0.1:8080
```

#### 2. Create Volumes

```bash
# Create volumes
warren volume create db_data --driver local --manager 127.0.0.1:8080
warren volume create redis_data --driver local --manager 127.0.0.1:8080
```

#### 3. Create Services (Bottom-Up)

**Database (no dependencies):**

```bash
warren service create db \
  --image postgres:15 \
  --replicas 1 \
  --secret db_password \
  --volume db_data:/var/lib/postgresql/data \
  --env POSTGRES_PASSWORD_FILE=/run/secrets/db_password \
  --manager 127.0.0.1:8080
```

**Cache (no dependencies):**

```bash
warren service create cache \
  --image redis:7 \
  --replicas 1 \
  --volume redis_data:/data \
  --manager 127.0.0.1:8080
```

**API (depends on db, cache):**

```bash
# Note: Update DATABASE_URL to use Warren service VIP or DNS
warren service create api \
  --image api:v1.0 \
  --replicas 5 \
  --secret db_password \
  --env DATABASE_URL=postgres://db:5432/mydb \
  --env REDIS_URL=redis://cache:6379 \
  --manager 127.0.0.1:8080
```

**Frontend (depends on api):**

```bash
warren service create frontend \
  --image frontend:v1.0 \
  --replicas 3 \
  --env API_URL=http://api:8080 \
  --manager 127.0.0.1:8080
```

### Step 3: Convert to Warren YAML

**warren.yaml:**

```yaml
# Secrets
apiVersion: warren.io/v1
kind: Secret
metadata:
  name: db_password
spec:
  fromFile: ./secrets/db_password.txt

---
# Volumes
apiVersion: warren.io/v1
kind: Volume
metadata:
  name: db_data
spec:
  driver: local

---
apiVersion: warren.io/v1
kind: Volume
metadata:
  name: redis_data
spec:
  driver: local

---
# Services
apiVersion: warren.io/v1
kind: Service
metadata:
  name: db
spec:
  image: postgres:15
  replicas: 1
  mode: replicated
  env:
    - name: POSTGRES_PASSWORD_FILE
      value: /run/secrets/db_password
  secrets:
    - db_password
  volumes:
    - name: db_data
      mountPath: /var/lib/postgresql/data

---
apiVersion: warren.io/v1
kind: Service
metadata:
  name: cache
spec:
  image: redis:7
  replicas: 1
  mode: replicated
  volumes:
    - name: redis_data
      mountPath: /data

---
apiVersion: warren.io/v1
kind: Service
metadata:
  name: api
spec:
  image: api:v1.0
  replicas: 5
  mode: replicated
  env:
    - name: DATABASE_URL
      value: postgres://db:5432/mydb
    - name: REDIS_URL
      value: redis://cache:6379
  secrets:
    - db_password

---
apiVersion: warren.io/v1
kind: Service
metadata:
  name: frontend
spec:
  image: frontend:v1.0
  replicas: 3
  mode: replicated
  env:
    - name: API_URL
      value: http://api:8080
```

**Deploy:**

```bash
warren apply -f warren.yaml --manager 127.0.0.1:8080
```

## Common Patterns

### Pattern 1: Web App + Database

**Docker Compose:**

```yaml
version: "3.8"

services:
  web:
    image: webapp:latest
    ports:
      - "80:80"
    environment:
      DB_HOST: db
    depends_on:
      - db

  db:
    image: postgres:15
    environment:
      POSTGRES_PASSWORD: secret123
    volumes:
      - db_data:/var/lib/postgresql/data

volumes:
  db_data:
```

**Warren:**

```bash
# Create secret
warren secret create db_password --from-literal password=secret123

# Create volume
warren volume create db_data

# Deploy database
warren service create db \
  --image postgres:15 \
  --replicas 1 \
  --volume db_data:/var/lib/postgresql/data \
  --secret db_password \
  --env POSTGRES_PASSWORD_FILE=/run/secrets/db_password

# Deploy web app
warren service create web \
  --image webapp:latest \
  --replicas 3 \
  --env DB_HOST=db
```

### Pattern 2: Microservices

**Docker Compose:**

```yaml
version: "3.8"

services:
  gateway:
    image: gateway:latest
    ports:
      - "80:80"
    depends_on:
      - users
      - orders
      - products

  users:
    image: users-service:latest
    deploy:
      replicas: 3

  orders:
    image: orders-service:latest
    deploy:
      replicas: 3

  products:
    image: products-service:latest
    deploy:
      replicas: 3
```

**Warren:**

```bash
# Deploy all services (no strict dependency order needed)
warren service create users --image users-service:latest --replicas 3
warren service create orders --image orders-service:latest --replicas 3
warren service create products --image products-service:latest --replicas 3
warren service create gateway --image gateway:latest --replicas 2
```

### Pattern 3: Background Workers

**Docker Compose:**

```yaml
version: "3.8"

services:
  worker:
    image: worker:latest
    environment:
      QUEUE_URL: redis://queue:6379
    depends_on:
      - queue
    deploy:
      replicas: 10

  queue:
    image: redis:7
```

**Warren:**

```bash
warren service create queue --image redis:7 --replicas 1
warren service create worker \
  --image worker:latest \
  --replicas 10 \
  --env QUEUE_URL=redis://queue:6379
```

### Pattern 4: Monitoring Stack

**Docker Compose:**

```yaml
version: "3.8"

services:
  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - prometheus_data:/prometheus

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    volumes:
      - grafana_data:/var/lib/grafana

  node-exporter:
    image: prom/node-exporter:latest
    deploy:
      mode: global

volumes:
  prometheus_data:
  grafana_data:
```

**Warren:**

```bash
# Create volumes
warren volume create prometheus_data
warren volume create grafana_data

# Deploy services
warren service create prometheus \
  --image prom/prometheus:latest \
  --replicas 1 \
  --volume prometheus_data:/prometheus

warren service create grafana \
  --image grafana/grafana:latest \
  --replicas 1 \
  --volume grafana_data:/var/lib/grafana

# Global service (one per node)
warren service create node-exporter \
  --image prom/node-exporter:latest \
  --mode global
```

## Feature Gaps and Workarounds

### 1. Published Ports

**Compose:**
```yaml
ports:
  - "80:80"
  - "443:443"
```

**Warren (M6):**
- Coming in M6
- **Workaround**: Use external load balancer or access via service VIP from within cluster

### 2. Depends On

**Compose:**
```yaml
depends_on:
  - db
  - cache
```

**Warren:**
- No strict dependency management
- **Workaround**: Deploy services in dependency order (manual)

### 3. Health Checks

**Compose:**
```yaml
healthcheck:
  test: ["CMD", "curl", "-f", "http://localhost/health"]
  interval: 30s
  timeout: 10s
  retries: 3
```

**Warren (M6):**
- Coming in M6
- **Workaround**: Rely on container exit code

### 4. Build Context

**Compose:**
```yaml
build:
  context: ./app
  dockerfile: Dockerfile
```

**Warren:**
- No build support (use pre-built images)
- **Workaround**: Build images separately and push to registry

```bash
# Build and push image
docker build -t myapp:v1.0 ./app
docker push myapp:v1.0

# Deploy to Warren
warren service create myapp --image myapp:v1.0
```

### 5. Environment File

**Compose:**
```yaml
env_file:
  - .env
```

**Warren:**
- No direct env file support
- **Workaround**: Use secrets or explicit `--env` flags

```bash
# Convert .env to secret
warren secret create app-env --from-file .env

# Or explicit variables
warren service create app \
  --env VAR1=value1 \
  --env VAR2=value2
```

### 6. Custom Networks

**Compose:**
```yaml
networks:
  frontend:
  backend:
```

**Warren:**
- Single WireGuard overlay (all services can communicate)
- **Workaround**: Use network policies (M7) for isolation

### 7. Resource Limits

**Compose:**
```yaml
deploy:
  resources:
    limits:
      cpus: '0.50'
      memory: 512M
```

**Warren (M6):**
- Coming in M6
- **Workaround**: OS-level cgroups (manual)

## Conversion Tools

### Automated Conversion Script

```bash
#!/bin/bash
# compose-to-warren.sh
# Converts docker-compose.yml to Warren commands

set -e

COMPOSE_FILE=${1:-docker-compose.yml}
MANAGER=${2:-127.0.0.1:8080}

echo "Converting $COMPOSE_FILE to Warren..."

# Parse services (requires yq)
services=$(yq eval '.services | keys | .[]' $COMPOSE_FILE)

for svc in $services; do
  image=$(yq eval ".services.$svc.image" $COMPOSE_FILE)
  replicas=$(yq eval ".services.$svc.deploy.replicas // 1" $COMPOSE_FILE)

  echo "Creating service: $svc"
  warren service create $svc \
    --image $image \
    --replicas $replicas \
    --manager $MANAGER

  echo "✓ Service $svc created"
done

echo "Conversion complete!"
```

**Usage:**
```bash
chmod +x compose-to-warren.sh
./compose-to-warren.sh docker-compose.yml 192.168.1.10:8080
```

## Migration Checklist

- [ ] **Pre-Migration**
  - [ ] Review Docker Compose file
  - [ ] Identify dependencies
  - [ ] List secrets and volumes
  - [ ] Test on Warren staging cluster

- [ ] **Conversion**
  - [ ] Create secrets in Warren
  - [ ] Create volumes in Warren
  - [ ] Convert services (bottom-up dependency order)
  - [ ] Update environment variables (service names, URLs)
  - [ ] Create Warren YAML manifest

- [ ] **Deployment**
  - [ ] Deploy secrets
  - [ ] Deploy volumes
  - [ ] Deploy services (dependency order)
  - [ ] Verify all services running

- [ ] **Validation**
  - [ ] Test service connectivity
  - [ ] Verify data persistence (volumes)
  - [ ] Check logs for errors
  - [ ] Load test application

- [ ] **Documentation**
  - [ ] Update deployment docs
  - [ ] Document manual steps
  - [ ] Create runbook for Warren

## Example: Full Application Migration

### Original Compose File

```yaml
version: "3.8"

services:
  nginx:
    image: nginx:latest
    ports:
      - "80:80"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
    depends_on:
      - api

  api:
    image: api:v1.0
    environment:
      DB_HOST: postgres
      REDIS_HOST: cache
    secrets:
      - db_password
    depends_on:
      - postgres
      - cache

  postgres:
    image: postgres:15
    environment:
      POSTGRES_PASSWORD_FILE: /run/secrets/db_password
    secrets:
      - db_password
    volumes:
      - db_data:/var/lib/postgresql/data

  cache:
    image: redis:7
    volumes:
      - redis_data:/data

secrets:
  db_password:
    file: ./secrets/db_password.txt

volumes:
  db_data:
  redis_data:
```

### Migration Steps

```bash
# 1. Create Warren cluster (if not exists)
sudo warren cluster init

# 2. Start worker
sudo warren worker start --manager 127.0.0.1:8080

# 3. Create secrets
warren secret create db_password --from-file ./secrets/db_password.txt

# 4. Create volumes
warren volume create db_data
warren volume create redis_data

# 5. Deploy services (bottom-up)
warren service create postgres \
  --image postgres:15 \
  --replicas 1 \
  --secret db_password \
  --volume db_data:/var/lib/postgresql/data \
  --env POSTGRES_PASSWORD_FILE=/run/secrets/db_password

warren service create cache \
  --image redis:7 \
  --replicas 1 \
  --volume redis_data:/data

warren service create api \
  --image api:v1.0 \
  --replicas 3 \
  --secret db_password \
  --env DB_HOST=postgres \
  --env REDIS_HOST=cache

warren service create nginx \
  --image nginx:latest \
  --replicas 2

# 6. Verify deployment
warren service list
warren service inspect api
```

### Warren YAML Equivalent

```yaml
apiVersion: warren.io/v1
kind: Secret
metadata:
  name: db_password
spec:
  fromFile: ./secrets/db_password.txt
---
apiVersion: warren.io/v1
kind: Volume
metadata:
  name: db_data
spec:
  driver: local
---
apiVersion: warren.io/v1
kind: Volume
metadata:
  name: redis_data
spec:
  driver: local
---
apiVersion: warren.io/v1
kind: Service
metadata:
  name: postgres
spec:
  image: postgres:15
  replicas: 1
  secrets:
    - db_password
  volumes:
    - name: db_data
      mountPath: /var/lib/postgresql/data
  env:
    - name: POSTGRES_PASSWORD_FILE
      value: /run/secrets/db_password
---
apiVersion: warren.io/v1
kind: Service
metadata:
  name: cache
spec:
  image: redis:7
  replicas: 1
  volumes:
    - name: redis_data
      mountPath: /data
---
apiVersion: warren.io/v1
kind: Service
metadata:
  name: api
spec:
  image: api:v1.0
  replicas: 3
  secrets:
    - db_password
  env:
    - name: DB_HOST
      value: postgres
    - name: REDIS_HOST
      value: cache
---
apiVersion: warren.io/v1
kind: Service
metadata:
  name: nginx
spec:
  image: nginx:latest
  replicas: 2
```

**Deploy:**
```bash
warren apply -f warren.yaml
```

## Getting Help

- **Documentation**: [Warren Docs](https://github.com/cuemby/warren/docs)
- **GitHub Discussions**: [Community Q&A](https://github.com/cuemby/warren/discussions)
- **GitHub Issues**: [Bug Reports](https://github.com/cuemby/warren/issues)
- **Email**: opensource@cuemby.com

---

**Migration successful?** Share your experience in [GitHub Discussions](https://github.com/cuemby/warren/discussions)!
