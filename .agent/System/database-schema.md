# Warren Database Schema

**Last Updated**: 2025-10-14
**Implementation Status**: Milestones 0-7 Complete ✅ | v1.3.1 Released | Phase 1 Complete ✅
**Database**: BoltDB (embedded key-value store)

---

## Overview

Warren uses **BoltDB** as its embedded database for cluster state persistence. BoltDB is a pure Go key-value store that:
- Provides ACID transactions
- Stores data in a single file (`warren.db`)
- Requires no external database server
- Integrates with Raft consensus for replication

**Location**: `/var/lib/warren/warren.db` (or specified `--data-dir`)

**File**: [pkg/storage/boltdb.go](../../pkg/storage/boltdb.go)

---

## Database Structure

BoltDB organizes data into **buckets** (similar to tables in SQL). Warren uses 6 primary buckets:

```
warren.db
├── nodes       # Cluster nodes (managers & workers)
├── services    # User-defined services
├── containers  # Individual container instances
├── secrets     # Encrypted sensitive data
├── volumes     # Persistent storage volumes
└── networks    # Overlay network definitions
```

---

## Bucket Schemas

### 1. Nodes Bucket

**Bucket Name**: `nodes`
**Key**: Node ID (string)
**Value**: JSON-serialized `Node` struct

**Purpose**: Tracks all cluster nodes (managers and workers)

**Schema**:
```go
type Node struct {
    ID            string                // Unique node identifier (UUID)
    Role          NodeRole              // "manager" | "worker"
    Address       string                // Host IP address
    OverlayIP     net.IP                // WireGuard overlay IP (future)
    Hostname      string                // Node hostname
    Labels        map[string]string     // User-defined labels
    Resources     *NodeResources        // Resource capacity
    Status        NodeStatus            // "ready" | "down" | "draining" | "unknown"
    LastHeartbeat time.Time             // Last heartbeat timestamp
    CreatedAt     time.Time             // Node registration time
}
```

**NodeResources**:
```go
type NodeResources struct {
    CPUCores           int     // Total CPU cores
    MemoryBytes        int64   // Total memory in bytes
    DiskBytes          int64   // Total disk in bytes
    CPUAllocated       float64 // Allocated CPU (cores)
    MemoryAllocated    int64   // Allocated memory (bytes)
    DiskAllocated      int64   // Allocated disk (bytes)
}
```

**Example Entry**:
```json
{
  "ID": "node-abc123",
  "Role": "worker",
  "Address": "192.168.1.10",
  "OverlayIP": null,
  "Hostname": "worker-01",
  "Labels": {"region": "us-west", "zone": "a"},
  "Resources": {
    "CPUCores": 8,
    "MemoryBytes": 16000000000,
    "DiskBytes": 500000000000,
    "CPUAllocated": 2.5,
    "MemoryAllocated": 4000000000,
    "DiskAllocated": 0
  },
  "Status": "ready",
  "LastHeartbeat": "2025-10-10T10:30:00Z",
  "CreatedAt": "2025-10-10T09:00:00Z"
}
```

**Access Patterns**:
- `GetNode(id)` - Direct lookup by node ID
- `ListNodes()` - Full scan of all nodes
- `UpdateNode(node)` - Upsert (create or update)
- `DeleteNode(id)` - Remove node

---

### 2. Services Bucket

**Bucket Name**: `services`
**Key**: Service ID (string)
**Value**: JSON-serialized `Service` struct

**Purpose**: Stores user-defined service definitions

**Schema**:
```go
type Service struct {
    ID             string                // Unique service ID (UUID)
    Name           string                // User-friendly name
    Image          string                // Container image (e.g., "nginx:latest")
    Replicas       int                   // Desired replica count
    Mode           ServiceMode           // "replicated" | "global"
    DeployStrategy DeployStrategy        // "rolling" | "blue-green" | "canary"
    UpdateConfig   *UpdateConfig         // Update strategy config
    Env            []string              // Environment variables
    Ports          []*PortMapping        // Port mappings (M6: with PublishMode)
    Networks       []string              // Network IDs
    Secrets        []string              // Secret IDs
    Volumes        []*VolumeMount        // Volume mounts
    Labels         map[string]string     // User-defined labels
    HealthCheck    *HealthCheck          // Health check config (M6: HTTP/TCP/Exec)
    RestartPolicy  *RestartPolicy        // Restart behavior
    Resources      *ResourceRequirements // CPU/memory limits (M6: implemented)
    StopTimeout    int                   // Graceful shutdown timeout in seconds (M6)
    CreatedAt      time.Time
    UpdatedAt      time.Time
}
```

**Example Entry**:
```json
{
  "ID": "svc-xyz789",
  "Name": "nginx-web",
  "Image": "nginx:latest",
  "Replicas": 3,
  "Mode": "replicated",
  "DeployStrategy": "rolling",
  "UpdateConfig": {
    "Parallelism": 1,
    "Delay": 5000000000,
    "FailureAction": "rollback",
    "CanaryWeight": 0
  },
  "Env": ["ENV=production"],
  "Ports": [
    {"ContainerPort": 80, "HostPort": 8080, "Protocol": "tcp"}
  ],
  "Networks": [],
  "Secrets": [],
  "Volumes": [],
  "Labels": {"app": "web", "tier": "frontend"},
  "HealthCheck": null,
  "RestartPolicy": {
    "Condition": "always",
    "MaxAttempts": 3,
    "Delay": 5000000000
  },
  "Resources": null,
  "CreatedAt": "2025-10-10T09:15:00Z",
  "UpdatedAt": "2025-10-10T09:15:00Z"
}
```

**Access Patterns**:
- `GetService(id)` - Direct lookup by service ID
- `GetServiceByName(name)` - Lookup by name (full scan)
- `ListServices()` - Full scan of all services
- `UpdateService(service)` - Upsert
- `DeleteService(id)` - Remove service

---

### 3. Containers Bucket

**Bucket Name**: `containers`
**Key**: Container ID (string)
**Value**: JSON-serialized `Container` struct

**Purpose**: Tracks individual container instances

**Schema**:
```go
type Container struct {
    ID            string                // Unique container ID (UUID)
    ServiceID     string                // Parent service ID
    ServiceName   string                // Service name (denormalized)
    NodeID        string                // Assigned worker node
    ContainerID   string                // Container runtime ID
    DesiredState  ContainerState        // "pending" | "running" | "shutdown"
    ActualState   ContainerState        // Actual state reported by worker
    HealthStatus  HealthStatus          // Health status (M6: healthy/unhealthy/unknown)
    Image         string                // Container image
    Env           []string              // Environment variables
    Ports         []*PortMapping        // Port mappings (M6: with PublishMode)
    Mounts        []*VolumeMount        // Volume mounts
    HealthCheck   *HealthCheck          // Health check config (M6: HTTP/TCP/Exec)
    RestartPolicy *RestartPolicy        // Restart policy
    Resources     *ResourceRequirements // Resource limits (M6: CPU/memory enforced)
    StopTimeout   int                   // Graceful shutdown timeout (M6)
    CreatedAt     time.Time
    StartedAt     time.Time
    FinishedAt    time.Time
    ExitCode      int
    Error         string
}
```

**ContainerState Values**:
- `pending` - Container created, awaiting scheduling
- `running` - Container assigned and running on worker
- `failed` - Container failed (non-zero exit or health check failure)
- `complete` - Container exited successfully (exit code 0)
- `shutdown` - Container stopped intentionally

**Example Entry**:
```json
{
  "ID": "container-123abc",
  "ServiceID": "svc-xyz789",
  "ServiceName": "nginx-web",
  "NodeID": "node-abc123",
  "ContainerID": "",
  "DesiredState": "running",
  "ActualState": "pending",
  "Image": "nginx:latest",
  "Env": ["ENV=production"],
  "Ports": [
    {"ContainerPort": 80, "HostPort": 8080, "Protocol": "tcp"}
  ],
  "Mounts": [],
  "HealthCheck": null,
  "RestartPolicy": {
    "Condition": "always",
    "MaxAttempts": 3,
    "Delay": 5000000000
  },
  "Resources": null,
  "CreatedAt": "2025-10-10T09:16:00Z",
  "StartedAt": "0001-01-01T00:00:00Z",
  "FinishedAt": "0001-01-01T00:00:00Z",
  "ExitCode": 0,
  "Error": ""
}
```

**Access Patterns**:
- `GetContainer(id)` - Direct lookup by container ID
- `ListContainers()` - Full scan of all containers
- `ListContainersByService(serviceID)` - Filter by service (in-memory filter)
- `ListContainersByNode(nodeID)` - Filter by node (in-memory filter)
- `UpdateContainer(container)` - Upsert
- `DeleteContainer(id)` - Remove container

---

### 4. Secrets Bucket

**Bucket Name**: `secrets`
**Key**: Secret ID (string)
**Value**: JSON-serialized `Secret` struct

**Purpose**: Stores encrypted sensitive data (passwords, keys, certificates)

**Schema**:
```go
type Secret struct {
    ID        string     // Unique secret ID (UUID)
    Name      string     // User-friendly name
    Data      []byte     // AES-256-GCM encrypted data
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

**Security Notes**:
- `Data` field is encrypted with AES-256-GCM before storage
- Encryption key stored in Raft cluster configuration
- Decrypted only when needed by workers
- **✅ Full encryption implementation complete (M3)**

**Example Entry**:
```json
{
  "ID": "secret-def456",
  "Name": "db-password",
  "Data": "base64-encoded-encrypted-data",
  "CreatedAt": "2025-10-10T09:00:00Z",
  "UpdatedAt": "2025-10-10T09:00:00Z"
}
```

**Access Patterns**:
- `GetSecret(id)` - Direct lookup by secret ID
- `GetSecretByName(name)` - Lookup by name (full scan)
- `ListSecrets()` - Full scan (returns encrypted data)
- `CreateSecret(secret)` - Insert secret
- `DeleteSecret(id)` - Remove secret

---

### 5. Volumes Bucket

**Bucket Name**: `volumes`
**Key**: Volume ID (string)
**Value**: JSON-serialized `Volume` struct

**Purpose**: Manages persistent storage volumes

**Schema**:
```go
type Volume struct {
    ID        string            // Unique volume ID (UUID)
    Name      string            // User-friendly name
    Driver    string            // "local", "nfs", etc.
    NodeID    string            // Node affinity (for local volumes)
    MountPath string            // Host mount path
    Options   map[string]string // Driver-specific options
    CreatedAt time.Time
}
```

**Driver Types**:
- `local` - Local filesystem volume (node-specific)
- `nfs` - Network File System (future)
- Custom drivers (pluggable, future)

**Example Entry**:
```json
{
  "ID": "vol-ghi789",
  "Name": "web-data",
  "Driver": "local",
  "NodeID": "node-abc123",
  "MountPath": "/var/lib/warren/volumes/web-data",
  "Options": {},
  "CreatedAt": "2025-10-10T09:00:00Z"
}
```

**Access Patterns**:
- `GetVolume(id)` - Direct lookup by volume ID
- `GetVolumeByName(name)` - Lookup by name (full scan)
- `ListVolumes()` - Full scan of all volumes
- `CreateVolume(volume)` - Insert volume
- `DeleteVolume(id)` - Remove volume

**✅ Volume orchestration complete (M3)**

---

### 6. Networks Bucket

**Bucket Name**: `networks`
**Key**: Network ID (string)
**Value**: JSON-serialized `Network` struct

**Purpose**: Defines overlay networks for service isolation

**Schema**:
```go
type Network struct {
    ID      string // Unique network ID (UUID)
    Name    string // User-friendly name
    Subnet  string // CIDR (e.g., "10.0.1.0/24")
    Gateway string // Gateway IP
    Driver  string // "wireguard"
}
```

**Example Entry**:
```json
{
  "ID": "net-jkl012",
  "Name": "frontend-net",
  "Subnet": "10.0.1.0/24",
  "Gateway": "10.0.1.1",
  "Driver": "wireguard"
}
```

**Access Patterns**:
- `GetNetwork(id)` - Direct lookup by network ID
- `ListNetworks()` - Full scan of all networks
- `CreateNetwork(network)` - Insert network
- `DeleteNetwork(id)` - Remove network

**⏳ WireGuard networking deferred (M6: DNS service discovery implemented)**

---

## Database Operations

### ACID Transactions

BoltDB provides ACID-compliant transactions:

**Read Transaction** (Shared lock):
```go
err := db.View(func(tx *bolt.Tx) error {
    b := tx.Bucket([]byte("nodes"))
    data := b.Get([]byte("node-id"))
    // Read-only operations
    return nil
})
```

**Write Transaction** (Exclusive lock):
```go
err := db.Update(func(tx *bolt.Tx) error {
    b := tx.Bucket([]byte("nodes"))
    data, _ := json.Marshal(node)
    return b.Put([]byte(node.ID), data)
})
```

### Serialization

- **Format**: JSON (`encoding/json`)
- **Encoding**: UTF-8
- **Storage**: Key-value pairs (key = ID, value = JSON bytes)

**Upsert Pattern**: Create and Update use the same operation (BoltDB `Put` overwrites)

---

## Raft Integration

Warren uses Raft consensus to replicate BoltDB state across manager nodes:

### Current Implementation (Milestone 1)

- **Single Manager**: BoltDB directly accessed by manager
- **No Replication**: Single copy of state
- **FSM**: Finite State Machine applies Raft log entries to state

**Raft FSM Apply**:
```go
func (fsm *FSM) Apply(log *raft.Log) interface{} {
    var cmd Command
    json.Unmarshal(log.Data, &cmd)

    switch cmd.Type {
    case "CreateService":
        store.CreateService(cmd.Service)
    case "UpdateTask":
        store.UpdateTask(cmd.Task)
    // ... other operations
    }
    return nil
}
```

### Future Implementation (Milestone 2)

- **Multi-Manager**: 3-5 manager quorum
- **Raft Replication**: All state changes replicated via Raft log
- **Consistency**: Strong consistency (linearizable reads/writes)
- **Leader Failover**: New leader takes over BoltDB access

---

## Schema Evolution

### Migration Strategy

**Current**: No migrations (Milestone 1, schema is stable)

**Future** (when schema changes required):
1. Version field in each struct
2. Backward-compatible JSON unmarshaling
3. Online migration during reads
4. Explicit migration command for large changes

**Example**:
```go
type Node struct {
    SchemaVersion int    // Add version field
    ID            string
    // ... other fields
}

// Read with migration
func GetNode(id string) (*Node, error) {
    node := unmarshal(data)
    if node.SchemaVersion < CurrentSchemaVersion {
        node = migrateNode(node)
    }
    return node
}
```

---

## Performance Characteristics

### Read Performance

- **Key lookup**: O(log n) (B+ tree)
- **Full scan**: O(n)
- **No secondary indexes**: Filtered queries require full scan

**Example**: `GetServiceByName` scans all services (acceptable for < 10K services)

### Write Performance

- **Single write**: < 1ms (SSD)
- **Batch write**: Atomic transaction, amortized cost
- **Lock**: Exclusive write lock (one writer at a time)

### Storage Size

**Estimated Storage** (Milestone 1 scale):
- 100 nodes × 1KB = 100KB
- 1,000 services × 2KB = 2MB
- 10,000 tasks × 1.5KB = 15MB
- **Total**: ~20MB for typical cluster

**Compaction**: BoltDB automatically reclaims freed space

---

## Backup and Recovery

### Backup

**Method 1**: File copy (while Warren stopped)
```bash
cp /var/lib/warren/warren.db /backup/warren-$(date +%Y%m%d).db
```

**Method 2**: Online backup (Raft snapshot)
```bash
warren cluster backup --output /backup/cluster-state.tar.gz
```
**⏳ Online backup deferred to Milestone 4**

### Recovery

**Restore from backup**:
```bash
# Stop Warren
systemctl stop warren

# Restore database
cp /backup/warren.db /var/lib/warren/warren.db

# Start Warren
systemctl start warren
```

**⏳ Point-in-time recovery deferred to Milestone 4**

---

## Debugging and Inspection

### BoltDB CLI Tool

Install `bbolt` CLI:
```bash
go install go.etcd.io/bbolt/cmd/bbolt@latest
```

**List buckets**:
```bash
bbolt buckets /var/lib/warren/warren.db
```

**Dump bucket**:
```bash
bbolt get /var/lib/warren/warren.db nodes node-abc123
```

**Inspect stats**:
```bash
bbolt stats /var/lib/warren/warren.db
```

### Warren CLI Inspection

```bash
# List all services
warren service list

# Inspect specific service (queries BoltDB)
warren service inspect nginx-web

# List all nodes
warren node list

# Get task details
warren task inspect task-123abc
```

---

## Security Considerations

### File Permissions

**BoltDB file**: `0600` (owner read/write only)
**Directory**: `/var/lib/warren/` (0700, owner only)

### Secrets Encryption

- **Algorithm**: AES-256-GCM (Galois/Counter Mode)
- **Key Storage**: Raft cluster configuration
- **Key Rotation**: Manual (future: automatic every 90 days)

**✅ Full secrets encryption complete (M3)**

### mTLS Security (M6)

- **Certificate Authority**: Self-signed root CA with 10-year validity
- **Node Certificates**: 365-day validity with IP SANs
- **TLS Version**: TLS 1.3 minimum
- **Certificate Storage**: File-based (0600 permissions)

**✅ mTLS implementation complete (M6)**

### Access Control

- **Manager only**: Only manager processes access BoltDB
- **Workers**: Read-only access via gRPC API (no direct DB access)
- **CLI**: Connects via gRPC API, not direct DB access

---

## Future Enhancements

### Milestone 6: Production Hardening ✅
- Health checks (HTTP/TCP/Exec probes)
- Resource limits (CPU/memory enforcement)
- Graceful shutdown (configurable timeout)
- mTLS security (CA, certificates, TLS 1.3)
- DNS service discovery (service/instance resolution)

### Future Milestones (M7+)
- Secondary indexes for faster lookups
- Online backup and restore
- Point-in-time recovery
- Automated schema migrations
- Certificate rotation automation

---

## Related Documentation

- **Storage Interface**: [pkg/storage/store.go](../../pkg/storage/store.go)
- **BoltDB Implementation**: [pkg/storage/boltdb.go](../../pkg/storage/boltdb.go)
- **Data Types**: [pkg/types/types.go](../../pkg/types/types.go)
- **API Reference**: [docs/api-reference.md](../../docs/api-reference.md)
- **Project Architecture**: [project-architecture.md](./project-architecture.md)

---

**Version**: 1.6
**Maintained By**: Cuemby Engineering Team
**Last Updated**: 2025-10-11
