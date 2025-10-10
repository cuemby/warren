# Warren API Reference

Complete reference for Warren's gRPC API.

## Overview

Warren exposes a gRPC API on port 8080 (configurable via `--api-addr`). All operations are performed through the `WarrenAPI` service.

**Base URL:** `localhost:8080` (default)

**Protocol:** gRPC (HTTP/2)

**Authentication:** None (Milestone 1) - mTLS in Milestone 3

## Table of Contents

- [Node Operations](#node-operations)
- [Service Operations](#service-operations)
- [Task Operations](#task-operations)
- [Secret Operations](#secret-operations)
- [Volume Operations](#volume-operations)
- [Data Models](#data-models)
- [Error Handling](#error-handling)
- [Usage Examples](#usage-examples)

---

## Node Operations

### RegisterNode

Register a new node (worker or manager) with the cluster.

**Method:** `RegisterNode`

**Request:**
```protobuf
message RegisterNodeRequest {
  string id = 1;                    // Unique node ID
  string role = 2;                  // "manager" or "worker"
  string address = 3;               // Node IP address
  NodeResources resources = 4;      // Available resources
  map<string, string> labels = 5;   // Node labels
}
```

**Response:**
```protobuf
message RegisterNodeResponse {
  Node node = 1;          // Registered node information
  string overlay_ip = 2;  // Assigned overlay IP (future)
}
```

**Errors:**
- `InvalidArgument`: Invalid role or missing required fields
- `AlreadyExists`: Node ID already registered

---

### Heartbeat

Send periodic heartbeat with task status updates.

**Method:** `Heartbeat`

**Request:**
```protobuf
message HeartbeatRequest {
  string node_id = 1;                     // Node ID
  NodeResources available_resources = 2;  // Current available resources
  repeated TaskStatus task_statuses = 3;  // Status of all tasks
}
```

**Response:**
```protobuf
message HeartbeatResponse {
  string status = 1;  // "ok"
}
```

**Notes:**
- Workers should send heartbeat every 5 seconds
- Manager marks nodes as down after 30 seconds without heartbeat
- Include status of ALL tasks assigned to this node

---

### ListNodes

List all nodes in the cluster.

**Method:** `ListNodes`

**Request:**
```protobuf
message ListNodesRequest {
  string role_filter = 1;  // Optional: "manager" or "worker"
}
```

**Response:**
```protobuf
message ListNodesResponse {
  repeated Node nodes = 1;
}
```

**Example:**
```bash
# List all nodes
warren node list

# Via grpcurl
grpcurl -plaintext localhost:8080 warren.v1.WarrenAPI/ListNodes
```

---

### GetNode

Get details of a specific node.

**Method:** `GetNode`

**Request:**
```protobuf
message GetNodeRequest {
  string id = 1;  // Node ID
}
```

**Response:**
```protobuf
message GetNodeResponse {
  Node node = 1;
}
```

**Errors:**
- `NotFound`: Node ID does not exist

---

### RemoveNode

Remove a node from the cluster.

**Method:** `RemoveNode`

**Request:**
```protobuf
message RemoveNodeRequest {
  string id = 1;  // Node ID
}
```

**Response:**
```protobuf
message RemoveNodeResponse {
  string status = 1;  // "ok"
}
```

**Notes:**
- Tasks on removed node are marked for rescheduling
- Use for graceful node decommission

---

## Service Operations

### CreateService

Create a new service.

**Method:** `CreateService`

**Request:**
```protobuf
message CreateServiceRequest {
  string name = 1;                          // Service name (unique)
  string image = 2;                         // Container image
  int32 replicas = 3;                       // Desired replica count
  string mode = 4;                          // "replicated" or "global"
  string deploy_strategy = 5;               // "rolling", "blue-green", "canary"
  UpdateConfig update_config = 6;           // Update configuration
  HealthCheck health_check = 7;             // Health check config
  RestartPolicy restart_policy = 8;         // Restart policy
  ResourceRequirements resources = 9;       // Resource requirements
  repeated string networks = 10;            // Network names
  repeated VolumeMount volumes = 11;        // Volume mounts
  map<string, string> env = 12;            // Environment variables
  repeated string command = 13;             // Command override
}
```

**Response:**
```protobuf
message CreateServiceResponse {
  Service service = 1;  // Created service
}
```

**Example:**
```bash
warren service create nginx \
  --image nginx:latest \
  --replicas 3 \
  --env PORT=8080
```

**Notes:**
- Service name must be unique
- Scheduler creates tasks within 5 seconds
- Default mode is "replicated"

---

### UpdateService

Update an existing service (scale, image, env vars).

**Method:** `UpdateService`

**Request:**
```protobuf
message UpdateServiceRequest {
  string id = 1;                // Service ID
  int32 replicas = 2;           // New replica count (optional)
  string image = 3;             // New image (optional)
  map<string, string> env = 4;  // New environment vars (optional)
}
```

**Response:**
```protobuf
message UpdateServiceResponse {
  Service service = 1;  // Updated service
}
```

**Example:**
```bash
warren service scale nginx --replicas 5
```

---

### DeleteService

Delete a service and all its tasks.

**Method:** `DeleteService`

**Request:**
```protobuf
message DeleteServiceRequest {
  string id = 1;  // Service ID
}
```

**Response:**
```protobuf
message DeleteServiceResponse {
  string status = 1;  // "ok"
}
```

**Notes:**
- All tasks are marked for shutdown
- Workers stop tasks gracefully
- Tasks are deleted after completion

---

### GetService

Get details of a specific service.

**Method:** `GetService`

**Request:**
```protobuf
message GetServiceRequest {
  string id = 1;    // Service ID (optional)
  string name = 2;  // Service name (optional)
}
```

**Response:**
```protobuf
message GetServiceResponse {
  Service service = 1;
}
```

**Notes:**
- Can query by either ID or name
- Returns first match

---

### ListServices

List all services in the cluster.

**Method:** `ListServices`

**Request:**
```protobuf
message ListServicesRequest {}
```

**Response:**
```protobuf
message ListServicesResponse {
  repeated Service services = 1;
}
```

**Example:**
```bash
warren service list
```

---

## Task Operations

### UpdateTaskStatus

Update task status (called by workers).

**Method:** `UpdateTaskStatus`

**Request:**
```protobuf
message UpdateTaskStatusRequest {
  string task_id = 1;       // Task ID
  string node_id = 2;       // Node ID
  string actual_state = 3;  // "pending", "running", "failed", "complete"
  string container_id = 4;  // Container ID
  string error = 5;         // Error message (if failed)
}
```

**Response:**
```protobuf
message UpdateTaskStatusResponse {
  string status = 1;  // "ok"
}
```

**Notes:**
- Workers call this when task state changes
- Typically included in heartbeat instead
- Use for explicit state updates

---

### ListTasks

List tasks with optional filters.

**Method:** `ListTasks`

**Request:**
```protobuf
message ListTasksRequest {
  string service_id = 1;  // Filter by service (optional)
  string node_id = 2;     // Filter by node (optional)
}
```

**Response:**
```protobuf
message ListTasksResponse {
  repeated Task tasks = 1;
}
```

**Example:**
```bash
# List all tasks for worker-1
grpcurl -plaintext -d '{"node_id":"worker-1"}' \
  localhost:8080 warren.v1.WarrenAPI/ListTasks
```

---

### GetTask

Get details of a specific task.

**Method:** `GetTask`

**Request:**
```protobuf
message GetTaskRequest {
  string id = 1;  // Task ID
}
```

**Response:**
```protobuf
message GetTaskResponse {
  Task task = 1;
}
```

---

### WatchTasks

Stream task assignments for a node (not yet implemented).

**Method:** `WatchTasks`

**Request:**
```protobuf
message WatchTasksRequest {
  string node_id = 1;  // Node ID to watch
}
```

**Response (Stream):**
```protobuf
message TaskEvent {
  string type = 1;  // "add", "update", "delete"
  Task task = 2;    // Task data
}
```

**Status:** 🚧 Placeholder for Milestone 2

---

## Secret Operations

### CreateSecret

Create a new secret.

**Method:** `CreateSecret`

**Request:**
```protobuf
message CreateSecretRequest {
  string name = 1;  // Secret name
  bytes data = 2;   // Secret data (encrypted)
}
```

**Response:**
```protobuf
message CreateSecretResponse {
  Secret secret = 1;  // Created secret (without data)
}
```

**Notes:**
- Data is encrypted with AES-256-GCM (Milestone 3)
- Secret data never returned in list/get operations

---

### DeleteSecret

Delete a secret.

**Method:** `DeleteSecret`

**Request:**
```protobuf
message DeleteSecretRequest {
  string id = 1;  // Secret ID
}
```

**Response:**
```protobuf
message DeleteSecretResponse {
  string status = 1;  // "ok"
}
```

---

### ListSecrets

List all secrets (without data).

**Method:** `ListSecrets`

**Request:**
```protobuf
message ListSecretsRequest {}
```

**Response:**
```protobuf
message ListSecretsResponse {
  repeated Secret secrets = 1;
}
```

**Notes:**
- Only metadata returned (ID, name, created_at)
- Data field always empty

---

## Volume Operations

### CreateVolume

Create a new volume.

**Method:** `CreateVolume`

**Request:**
```protobuf
message CreateVolumeRequest {
  string name = 1;                          // Volume name
  string driver = 2;                        // "local", "nfs", etc.
  map<string, string> driver_opts = 3;      // Driver-specific options
  map<string, string> labels = 4;           // Volume labels
}
```

**Response:**
```protobuf
message CreateVolumeResponse {
  Volume volume = 1;
}
```

---

### DeleteVolume

Delete a volume.

**Method:** `DeleteVolume`

**Request:**
```protobuf
message DeleteVolumeRequest {
  string id = 1;  // Volume ID
}
```

**Response:**
```protobuf
message DeleteVolumeResponse {
  string status = 1;  // "ok"
}
```

---

### ListVolumes

List all volumes.

**Method:** `ListVolumes`

**Request:**
```protobuf
message ListVolumesRequest {}
```

**Response:**
```protobuf
message ListVolumesResponse {
  repeated Volume volumes = 1;
}
```

---

## Data Models

### Node

```protobuf
message Node {
  string id = 1;
  string role = 2;                        // "manager" or "worker"
  string address = 3;
  string overlay_ip = 4;
  NodeResources resources = 5;
  string status = 6;                      // "ready", "down", "unknown"
  google.protobuf.Timestamp last_heartbeat = 7;
  google.protobuf.Timestamp created_at = 8;
  map<string, string> labels = 9;
}
```

### NodeResources

```protobuf
message NodeResources {
  int64 cpu_cores = 1;      // Number of CPU cores
  int64 memory_bytes = 2;   // Memory in bytes
  int64 disk_bytes = 3;     // Disk space in bytes
}
```

### Service

```protobuf
message Service {
  string id = 1;
  string name = 2;
  string image = 3;
  int32 replicas = 4;
  string mode = 5;                        // "replicated" or "global"
  string deploy_strategy = 6;             // "rolling", "blue-green", "canary"
  UpdateConfig update_config = 7;
  HealthCheck health_check = 8;
  RestartPolicy restart_policy = 9;
  ResourceRequirements resources = 10;
  repeated string networks = 11;
  repeated VolumeMount volumes = 12;
  map<string, string> env = 13;
  repeated string command = 14;
  google.protobuf.Timestamp created_at = 15;
  google.protobuf.Timestamp updated_at = 16;
}
```

### Task

```protobuf
message Task {
  string id = 1;
  string service_id = 2;
  string service_name = 3;
  string node_id = 4;
  string container_id = 5;
  string desired_state = 6;               // "running", "shutdown"
  string actual_state = 7;                // "pending", "running", "failed", "complete"
  string image = 8;
  repeated string command = 9;
  map<string, string> env = 10;
  ResourceRequirements resources = 11;
  repeated VolumeMount volumes = 12;
  HealthCheck health_check = 13;
  RestartPolicy restart_policy = 14;
  google.protobuf.Timestamp created_at = 15;
  google.protobuf.Timestamp updated_at = 16;
  string error = 17;
}
```

### UpdateConfig

```protobuf
message UpdateConfig {
  int32 parallelism = 1;      // How many tasks to update at once
  int32 delay_seconds = 2;    // Delay between batches
  string failure_action = 3;  // "pause", "continue", "rollback"
}
```

### HealthCheck

```protobuf
message HealthCheck {
  string type = 1;            // "http", "tcp", "exec"
  string endpoint = 2;        // URL or address
  int32 interval_seconds = 3;
  int32 timeout_seconds = 4;
  int32 retries = 5;
}
```

### RestartPolicy

```protobuf
message RestartPolicy {
  string condition = 1;       // "none", "on-failure", "any"
  int32 max_attempts = 2;
  int32 delay_seconds = 3;
}
```

### ResourceRequirements

```protobuf
message ResourceRequirements {
  int64 cpu_shares = 1;                 // CPU shares (1024 = 1 core)
  int64 memory_bytes = 2;               // Memory limit
  int64 memory_reservation_bytes = 3;   // Memory reservation
}
```

---

## Error Handling

### Standard Error Codes

Warren uses standard gRPC status codes:

| Code | Description | Example |
|------|-------------|---------|
| `OK` (0) | Success | Operation completed |
| `INVALID_ARGUMENT` (3) | Bad request | Invalid node role, missing required field |
| `NOT_FOUND` (5) | Resource not found | Node/Service/Task ID doesn't exist |
| `ALREADY_EXISTS` (6) | Duplicate resource | Node ID or service name already exists |
| `UNAVAILABLE` (14) | Service unavailable | Manager not ready, Raft not leader |
| `INTERNAL` (13) | Internal error | Storage failure, Raft apply failed |

### Error Response Format

```json
{
  "error": {
    "code": 3,
    "message": "invalid node role: must be 'worker' or 'manager'",
    "details": []
  }
}
```

### Handling Errors

**Go Client:**
```go
_, err := client.CreateService(ctx, req)
if err != nil {
    if status.Code(err) == codes.AlreadyExists {
        // Handle duplicate service name
    }
}
```

**CLI:**
```bash
warren service create nginx --image nginx:latest
# Error: failed to create service: service name already exists
```

---

## Usage Examples

### Using grpcurl

**List Services:**
```bash
grpcurl -plaintext localhost:8080 warren.v1.WarrenAPI/ListServices
```

**Create Service:**
```bash
grpcurl -plaintext -d '{
  "name": "nginx",
  "image": "nginx:latest",
  "replicas": 3,
  "mode": "replicated"
}' localhost:8080 warren.v1.WarrenAPI/CreateService
```

**Get Service:**
```bash
grpcurl -plaintext -d '{"name":"nginx"}' \
  localhost:8080 warren.v1.WarrenAPI/GetService
```

### Using Go Client

```go
import (
    "github.com/cuemby/warren/pkg/client"
)

// Connect to manager
c, err := client.NewClient("localhost:8080")
if err != nil {
    log.Fatal(err)
}
defer c.Close()

// Create service
service, err := c.CreateService("nginx", "nginx:latest", 3, nil)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Created service: %s\n", service.Id)

// List services
services, err := c.ListServices()
if err != nil {
    log.Fatal(err)
}

for _, svc := range services {
    fmt.Printf("- %s: %d replicas\n", svc.Name, svc.Replicas)
}
```

### Using Python Client

```python
import grpc
from proto import warren_pb2, warren_pb2_grpc

# Connect
channel = grpc.insecure_channel('localhost:8080')
client = warren_pb2_grpc.WarrenAPIStub(channel)

# Create service
request = warren_pb2.CreateServiceRequest(
    name='nginx',
    image='nginx:latest',
    replicas=3,
    mode='replicated'
)

response = client.CreateService(request)
print(f"Created service: {response.service.id}")
```

---

## Rate Limits

**Milestone 1:** No rate limiting

**Future:**
- Per-IP rate limits
- Authenticated user quotas
- Burst allowances

---

## Versioning

**Current Version:** `v1`

**API Stability:** Alpha (Milestone 1)
- Breaking changes possible
- Backward compatibility not guaranteed

**Stable API:** Milestone 6+

---

## Performance Considerations

### Request Timeouts

Default client timeouts:
- Node operations: 10 seconds
- Service operations: 10 seconds
- Task operations: 5 seconds
- Heartbeat: 5 seconds

### Batch Operations

For bulk operations, use multiple concurrent requests:

```go
// Create 10 services concurrently
var wg sync.WaitGroup
for i := 0; i < 10; i++ {
    wg.Add(1)
    go func(n int) {
        defer wg.Done()
        c.CreateService(fmt.Sprintf("service-%d", n), "nginx:latest", 1, nil)
    }(i)
}
wg.Wait()
```

### Caching

- Node list: Cache for 5 seconds (heartbeat interval)
- Service list: Cache for 10 seconds (reconciler interval)
- Task list: No caching recommended (rapid state changes)

---

## Security

**Milestone 1:** No authentication or encryption

**Milestone 3:**
- mTLS for all gRPC connections
- Certificate-based authentication
- Role-based access control (RBAC)

---

## API Evolution

### Planned Additions (Milestone 2+)

- `warren.v1.WarrenAPI/WatchTasks` - Task event streaming
- `warren.v1.WarrenAPI/GetClusterInfo` - Cluster metadata
- `warren.v1.WarrenAPI/CreateNetwork` - Network management
- `warren.v1.WarrenAPI/GetLogs` - Container log streaming

### Deprecation Policy

- Deprecated APIs marked in documentation
- 2 milestone grace period before removal
- Migration guides provided

---

## See Also

- [Quick Start Guide](quickstart.md) - Getting started
- [Developer Guide](developer-guide.md) - Architecture deep-dive
- [Protobuf Definitions](../api/proto/warren.proto) - Source definitions
- [Go Client Documentation](../pkg/client/client.go) - Client library

---

**Last Updated:** 2025-10-10 (Milestone 1)
