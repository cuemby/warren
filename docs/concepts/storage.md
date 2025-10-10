# Storage in Warren

Warren provides two types of storage for containerized applications:

1. **Volumes** - Persistent data storage that survives container restarts
2. **Secrets** - Encrypted sensitive data (passwords, certificates, keys)

This guide explains how storage works in Warren.

## Volumes

### What are Volumes?

Volumes provide persistent storage for stateful applications. Data in volumes persists even when containers are stopped or restarted.

**Use Cases:**
- Database data directories
- Application state
- Uploaded files
- Logs
- Cache directories

### Volume Lifecycle

```
1. Create Volume
   └── warren volume create db-data

2. Attach to Service
   └── warren service create postgres --volume db-data:/var/lib/postgresql/data

3. Task Scheduled with Volume
   └── Scheduler pins task to node with volume

4. Container Mounts Volume
   └── containerd bind-mounts volume path

5. Data Persists
   └── Container restarts, data remains
```

### Creating Volumes

```bash
# Create volume with local driver
warren volume create my-data \
  --driver local \
  --manager 127.0.0.1:8080

# Create volume with optional labels
warren volume create db-data \
  --driver local \
  --label env=production \
  --label app=postgres \
  --manager 127.0.0.1:8080
```

### Listing Volumes

```bash
warren volume list --manager 127.0.0.1:8080

# Output:
# NAME      DRIVER  SIZE    NODE       CREATED
# db-data   local   1.2GB   worker-1   2d ago
# app-data  local   500MB   worker-2   5h ago
```

### Inspecting Volumes

```bash
warren volume inspect db-data --manager 127.0.0.1:8080

# Output:
# Name: db-data
# ID: vol-db-data-abc123
# Driver: local
# Mount Point: /var/lib/warren/volumes/db-data
# Node: worker-1
# Size: 1.2 GB
# Used By:
#   - postgres (service)
#   - postgres-task-1 (task)
# Created: 2025-10-08 10:00:00
```

### Using Volumes with Services

```bash
# Attach volume to service
warren service create postgres \
  --image postgres:15 \
  --replicas 1 \
  --volume db-data:/var/lib/postgresql/data \
  --env POSTGRES_PASSWORD=password \
  --manager 127.0.0.1:8080
```

**What happens:**
1. Scheduler checks if `db-data` volume exists
2. If exists, scheduler pins task to node with volume (node affinity)
3. If doesn't exist, scheduler picks a node and creates volume
4. Worker mounts volume to container at specified path
5. Container writes data to volume

### Deleting Volumes

```bash
# Delete volume (only if not in use)
warren volume delete db-data --manager 127.0.0.1:8080

# Force delete (stops using services first)
warren volume delete db-data --force --manager 127.0.0.1:8080
```

**Safety:** Warren prevents deleting volumes in use unless `--force` is specified.

## Volume Drivers

### Local Driver (Current)

The local driver stores volumes on the worker node's filesystem.

**Characteristics:**
- **Storage**: Local disk on worker node
- **Performance**: Direct filesystem access (fast)
- **Durability**: Tied to node (data lost if node fails)
- **Sharing**: Single node only (no cross-node access)

**Mount Points:**
```bash
# Volumes stored at:
/var/lib/warren/volumes/<volume-name>/

# Example:
/var/lib/warren/volumes/db-data/
```

**Best For:**
- Single-replica databases
- Node-local caching
- Logs and temporary data
- Development environments

### Future: Distributed Drivers (M7)

Warren will support distributed volume drivers:

**NFS Driver:**
```bash
warren volume create shared-data \
  --driver nfs \
  --option server=192.168.1.100 \
  --option path=/exports/shared
```

**Ceph Driver:**
```bash
warren volume create ceph-data \
  --driver ceph \
  --option pool=warren \
  --option size=10G
```

**Benefits:**
- Multi-node access (shared volumes)
- Redundancy (data replicated)
- Migration (move services across nodes)

## Node Affinity

### How Node Affinity Works

When a service uses a volume, Warren enforces **node affinity** to ensure the task runs on the same node as the volume.

**Example:**

```bash
# Create volume (stored on worker-1)
warren volume create db-data --manager 127.0.0.1:8080

# Create service with volume
warren service create postgres \
  --replicas 1 \
  --volume db-data:/var/lib/postgresql/data

# Scheduler ALWAYS schedules postgres task to worker-1
```

**Why?**
- Local volumes can't be accessed from other nodes
- Ensures data consistency
- Prevents split-brain scenarios

### Multi-Replica Services with Volumes

**Single volume (not recommended):**
```bash
# Bad: Multiple replicas with same volume
warren service create app --replicas 3 --volume data:/data
```

**Problem**: All replicas scheduled to same node (defeats load balancing).

**Solution 1: Use volume per replica**
```bash
# Better: One volume per replica
warren volume create app-data-1
warren volume create app-data-2
warren volume create app-data-3

# Deploy with separate volumes (requires manual task assignment - M6)
```

**Solution 2: Use distributed volume driver (M7)**
```bash
# Best: Shared distributed volume
warren volume create shared-data --driver nfs
warren service create app --replicas 3 --volume shared-data:/data
```

**Solution 3: Use database replication**
```bash
# Best for databases: Built-in replication
warren service create postgres-primary --volume primary-data:/data
warren service create postgres-replica --volume replica-data:/data
# Configure PostgreSQL streaming replication
```

## Secrets

### What are Secrets?

Secrets store sensitive data (passwords, API keys, certificates) in encrypted form.

**Features:**
- **Encrypted at rest** - AES-256-GCM encryption in BoltDB
- **Encrypted in transit** - Transferred over WireGuard/mTLS
- **Mounted to containers** - Available at `/run/secrets/<secret-name>`
- **Temporary storage** - Stored in tmpfs (RAM, never disk)

### Creating Secrets

**From literal value:**
```bash
warren secret create db-password \
  --from-literal password=mySecurePassword123 \
  --manager 127.0.0.1:8080
```

**From file:**
```bash
warren secret create tls-cert \
  --from-file /path/to/cert.pem \
  --manager 127.0.0.1:8080

warren secret create tls-key \
  --from-file /path/to/key.pem \
  --manager 127.0.0.1:8080
```

**From stdin:**
```bash
echo -n "mySecurePassword" | warren secret create db-password \
  --from-stdin \
  --manager 127.0.0.1:8080
```

### Listing Secrets

```bash
warren secret list --manager 127.0.0.1:8080

# Output:
# NAME          SIZE    CREATED
# db-password   23B     2d ago
# api-key       64B     1h ago
# tls-cert      1.2KB   5h ago
```

**Note:** Secret values are never shown, only metadata.

### Inspecting Secrets

```bash
warren secret inspect db-password --manager 127.0.0.1:8080

# Output:
# Name: db-password
# ID: secret-db-password-abc123
# Size: 23 bytes
# Created: 2025-10-08 10:00:00
# Used By:
#   - postgres (service)
#   - api (service)
```

**Security:** Actual secret value is never displayed.

### Using Secrets with Services

```bash
# Create secret
warren secret create db-password \
  --from-literal password=secret123

# Deploy service with secret
warren service create api \
  --image api:latest \
  --replicas 5 \
  --secret db-password \
  --env DB_PASSWORD_FILE=/run/secrets/db-password \
  --manager 127.0.0.1:8080
```

**Inside container:**
```bash
# Secret mounted at /run/secrets/db-password
cat /run/secrets/db-password
# Output: secret123

# Use in application
export DB_PASSWORD=$(cat /run/secrets/db-password)
psql -U user -d mydb
```

### Multiple Secrets

```bash
warren service create webapp \
  --image webapp:latest \
  --secret db-password \
  --secret api-key \
  --secret tls-cert \
  --secret tls-key \
  --manager 127.0.0.1:8080

# Mounted at:
# /run/secrets/db-password
# /run/secrets/api-key
# /run/secrets/tls-cert
# /run/secrets/tls-key
```

### Deleting Secrets

```bash
# Delete secret (only if not in use)
warren secret delete db-password --manager 127.0.0.1:8080

# Force delete (removes from using services)
warren secret delete db-password --force --manager 127.0.0.1:8080
```

**Safety:** Warren prevents deleting secrets in use unless `--force` is specified.

## Secret Encryption

### Encryption Algorithm

Warren uses **AES-256-GCM** (Galois/Counter Mode):

- **Key Size**: 256 bits
- **Mode**: GCM (Authenticated Encryption with Associated Data)
- **Authentication**: Included (prevents tampering)
- **IV**: Random per secret (128 bits)

### Key Derivation

The encryption key is derived during cluster initialization:

```bash
# Initialize cluster (generates encryption key)
warren cluster init

# Key derived from:
# - Cluster ID (unique per cluster)
# - Random salt (generated at init)
# - PBKDF2 (100,000 iterations)
```

**Key Storage**: Stored securely in BoltDB (encrypted metadata).

**Key Rotation**: Manual key rotation coming in M6.

### Encryption Flow

```
1. User creates secret: "password=secret123"
2. Manager encrypts with AES-256-GCM
3. Stores encrypted blob + IV in BoltDB
4. Worker requests secret for task
5. Manager sends encrypted secret (over WireGuard)
6. Worker decrypts secret
7. Worker mounts to tmpfs (RAM)
8. Container reads from /run/secrets/
```

**At Rest**: Encrypted in BoltDB (AES-256-GCM)
**In Transit**: Encrypted via WireGuard (ChaCha20-Poly1305)
**In Use**: Plaintext in tmpfs (RAM only, never disk)

### Security Properties

- **Confidentiality**: Secret values never stored in plaintext
- **Integrity**: GCM authentication prevents tampering
- **Forward Secrecy**: Rotating WireGuard keys protect past traffic
- **Ephemeral**: Secrets cleared from RAM on task stop

## Backup and Recovery

### Backing Up Volumes

**Manual backup:**
```bash
# On worker node with volume
sudo tar czf db-data-backup.tar.gz /var/lib/warren/volumes/db-data/

# Copy backup to safe location
scp db-data-backup.tar.gz backup-server:/backups/
```

**Future: Automatic snapshots (M7)**
```bash
warren volume snapshot db-data --name snapshot-20251010
warren volume restore db-data --from snapshot-20251010
```

### Backing Up Secrets

Secrets are stored encrypted in BoltDB. Backup the entire database:

```bash
# On manager node
sudo systemctl stop warren-manager
sudo cp /var/lib/warren/data/warren.db /backups/warren.db.backup
sudo systemctl start warren-manager
```

**Future: Secret export (M6)**
```bash
warren secret export --all --output secrets-backup.json
warren secret import --file secrets-backup.json
```

### Disaster Recovery

**Restore volume:**
```bash
# On worker node
sudo systemctl stop warren-worker
sudo rm -rf /var/lib/warren/volumes/db-data/
sudo tar xzf db-data-backup.tar.gz -C /
sudo systemctl start warren-worker
```

**Restore secrets (restore BoltDB):**
```bash
# On manager node
sudo systemctl stop warren-manager
sudo cp /backups/warren.db.backup /var/lib/warren/data/warren.db
sudo systemctl start warren-manager
```

## Storage Best Practices

### 1. Use Volumes for Stateful Data

```bash
# Bad: Store data in container (ephemeral)
warren service create postgres --image postgres:15

# Good: Use volume for persistent data
warren volume create db-data
warren service create postgres \
  --volume db-data:/var/lib/postgresql/data
```

### 2. Use Secrets for Sensitive Data

```bash
# Bad: Password in environment variable (visible)
warren service create app --env DB_PASSWORD=secret123

# Good: Password in secret (encrypted)
warren secret create db-password --from-literal password=secret123
warren service create app --secret db-password
```

### 3. Single Replica for Stateful Services

```bash
# Good: Single replica for database
warren service create postgres \
  --replicas 1 \
  --volume db-data:/var/lib/postgresql/data
```

Avoid multiple replicas with local volumes (causes node affinity conflicts).

### 4. Regular Backups

```bash
# Backup volumes regularly
0 2 * * * /usr/local/bin/backup-warren-volumes.sh

# Backup BoltDB regularly
0 3 * * * /usr/local/bin/backup-warren-db.sh
```

### 5. Monitor Volume Usage

```bash
# Check volume size
warren volume inspect db-data

# Monitor disk usage on workers
df -h /var/lib/warren/volumes/
```

Set alerts when volumes reach 80% capacity.

### 6. Rotate Secrets

```bash
# Update secret value (future M6)
warren secret update db-password --from-literal password=newPassword

# Restart services to pick up new secret
warren service update postgres --force-recreate
```

### 7. Use Read-Only Volumes

```bash
# Mount volume as read-only (future M6)
warren service create app \
  --volume config-data:/etc/config:ro
```

Prevents accidental data corruption.

## Storage Limits and Quotas

### Current Limits

- **Volume Size**: Limited by node disk space
- **Secret Size**: 1 MB max per secret
- **Secrets per Service**: Unlimited
- **Volumes per Service**: Unlimited

### Future: Quotas (M7)

```bash
# Set volume size limit
warren volume create db-data --size 10G

# Set namespace quota
warren namespace create prod --volume-quota 100G --secret-quota 100
```

## YAML Configuration

Define volumes and secrets declaratively:

**volume.yaml:**
```yaml
apiVersion: warren.io/v1
kind: Volume
metadata:
  name: db-data
spec:
  driver: local
  labels:
    env: production
    app: postgres
```

**secret.yaml:**
```yaml
apiVersion: warren.io/v1
kind: Secret
metadata:
  name: db-password
spec:
  data:
    password: c2VjcmV0MTIz  # base64 encoded
```

**service.yaml:**
```yaml
apiVersion: warren.io/v1
kind: Service
metadata:
  name: postgres
spec:
  image: postgres:15
  replicas: 1
  volumes:
    - name: db-data
      mountPath: /var/lib/postgresql/data
  secrets:
    - db-password
  env:
    - name: POSTGRES_PASSWORD_FILE
      value: /run/secrets/db-password
```

**Apply:**
```bash
warren apply -f volume.yaml -f secret.yaml -f service.yaml
```

## Troubleshooting

### Volume Mount Fails

```bash
# Check volume exists
warren volume inspect db-data

# Check volume path on worker
sudo ls -la /var/lib/warren/volumes/db-data/

# Check volume permissions
sudo stat /var/lib/warren/volumes/db-data/

# Common fix: Ensure directory exists and has correct permissions
sudo chown -R 999:999 /var/lib/warren/volumes/db-data/  # postgres UID/GID
```

### Secret Not Mounted

```bash
# Check secret exists
warren secret inspect db-password

# Check if service references secret
warren service inspect myapp

# Check tmpfs mount inside container
sudo ctr -n warren tasks exec -t --exec-id debug <task-id> sh
mount | grep /run/secrets
cat /run/secrets/db-password
```

### Volume Full

```bash
# Check volume size
warren volume inspect db-data

# Check disk usage on worker
df -h /var/lib/warren/volumes/

# Clean up old data or expand disk
```

### Task Stuck on Wrong Node

```bash
# If task scheduled to node without volume
# This indicates a scheduler bug

# Workaround: Delete and recreate service
warren service delete myapp
warren service create myapp --volume db-data:/data
```

## Next Steps

- **[High Availability](high-availability.md)** - Resilient storage strategies
- **[Networking](networking.md)** - Network security for secrets
- **[Services](services.md)** - Using storage with services

---

**Questions?** See [Troubleshooting](../troubleshooting.md) or ask in [GitHub Discussions](https://github.com/cuemby/warren/discussions).
