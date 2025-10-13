# Warren API Documentation

**Last Updated**: 2025-10-13
**Implementation Status**: Milestones 0-7 Complete ✅ | v1.1.1 Released
**API Version**: v1
**Protocol**: gRPC

---

## Overview

Warren exposes a **gRPC API** for all cluster operations. The API is used by:
- **CLI** (`warren` command) - Primary user interface
- **Workers** - Register nodes, send heartbeats, watch tasks
- **External tools** - Programmatic cluster management

**API Server**:
- **Port**: `:2377` (default, configurable)
- **Protocol**: gRPC over HTTP/2
- **Security**: ✅ mTLS with TLS 1.3 (M6 complete)
- **File**: [pkg/api/server.go](../../pkg/api/server.go)
- **Proto**: [api/proto/warren.proto](../../api/proto/warren.proto)

---

## API Service

### WarrenAPI Service

The main gRPC service with 30+ methods organized by resource type:

```protobuf
service WarrenAPI {
  // Node operations (5 methods)
  rpc RegisterNode(RegisterNodeRequest) returns (RegisterNodeResponse);
  rpc Heartbeat(HeartbeatRequest) returns (HeartbeatResponse);
  rpc ListNodes(ListNodesRequest) returns (ListNodesResponse);
  rpc GetNode(GetNodeRequest) returns (GetNodeResponse);
  rpc RemoveNode(RemoveNodeRequest) returns (RemoveNodeResponse);

  // Service operations (5 methods)
  rpc CreateService(CreateServiceRequest) returns (CreateServiceResponse);
  rpc UpdateService(UpdateServiceRequest) returns (UpdateServiceResponse);
  rpc DeleteService(DeleteServiceRequest) returns (DeleteServiceResponse);
  rpc GetService(GetServiceRequest) returns (GetServiceResponse);
  rpc ListServices(ListServicesRequest) returns (ListServicesResponse);

  // Container operations (5 methods - M6: added ReportContainerHealth)
  rpc UpdateContainerStatus(UpdateContainerStatusRequest) returns (UpdateContainerStatusResponse);
  rpc ListContainers(ListContainersRequest) returns (ListContainersResponse);
  rpc GetContainer(GetContainerRequest) returns (GetContainerResponse);
  rpc WatchContainers(WatchContainersRequest) returns (stream ContainerEvent);
  rpc ReportContainerHealth(ReportContainerHealthRequest) returns (ReportContainerHealthResponse);  // M6

  // Secret operations (4 methods - M6: added GetSecret)
  rpc CreateSecret(CreateSecretRequest) returns (CreateSecretResponse);
  rpc DeleteSecret(DeleteSecretRequest) returns (DeleteSecretResponse);
  rpc ListSecrets(ListSecretsRequest) returns (ListSecretsResponse);
  rpc GetSecret(GetSecretRequest) returns (GetSecretResponse);  // M6

  // Volume operations (4 methods - M6: added GetVolume)
  rpc CreateVolume(CreateVolumeRequest) returns (CreateVolumeResponse);
  rpc DeleteVolume(DeleteVolumeRequest) returns (DeleteVolumeResponse);
  rpc ListVolumes(ListVolumesRequest) returns (ListVolumesResponse);
  rpc GetVolume(GetVolumeRequest) returns (GetVolumeResponse);  // M6

  // Cluster operations (5 methods - M2+M6)
  rpc GenerateJoinToken(GenerateJoinTokenRequest) returns (GenerateJoinTokenResponse);
  rpc JoinCluster(JoinClusterRequest) returns (JoinClusterResponse);
  rpc GetClusterInfo(GetClusterInfoRequest) returns (GetClusterInfoResponse);
  rpc RequestCertificate(RequestCertificateRequest) returns (RequestCertificateResponse);  // M6
  rpc StreamEvents(StreamEventsRequest) returns (stream Event);
}
```

**Total**: 28 unary RPCs + 2 server-streaming RPCs = **30 methods**

**M6 Additions**:
- ReportContainerHealth (health check monitoring)
- RequestCertificate (mTLS certificate management)
- GetSecret, GetVolume (complete CRUD operations)

---

## Node Operations

### 1. RegisterNode

**Purpose**: Register a new node (manager or worker) with the cluster

**Method**: `RegisterNode(RegisterNodeRequest) → RegisterNodeResponse`

**Request**:
```protobuf
message RegisterNodeRequest {
  string id = 1;                      // Unique node ID (UUID)
  string role = 2;                    // "manager" or "worker"
  string address = 3;                 // Host IP address
  NodeResources resources = 4;        // Available resources
  map<string, string> labels = 5;     // User-defined labels
}

message NodeResources {
  int64 cpu_cores = 1;       // Total CPU cores
  int64 memory_bytes = 2;    // Total memory (bytes)
  int64 disk_bytes = 3;      // Total disk (bytes)
}
```

**Response**:
```protobuf
message RegisterNodeResponse {
  Node node = 1;              // Registered node details
  string overlay_ip = 2;      // Assigned overlay IP (future)
}
```

**CLI Usage**:
```bash
warren worker start --address 192.168.1.10
```

**Go Client Example**:
```go
client := NewWarrenClient(conn)
resp, err := client.RegisterNode(ctx, &RegisterNodeRequest{
    Id:      "node-abc123",
    Role:    "worker",
    Address: "192.168.1.10",
    Resources: &NodeResources{
        CpuCores:    8,
        MemoryBytes: 16 * 1024 * 1024 * 1024, // 16GB
        DiskBytes:   500 * 1024 * 1024 * 1024, // 500GB
    },
    Labels: map[string]string{"zone": "us-west-1a"},
})
```

---

### 2. Heartbeat

**Purpose**: Worker sends periodic heartbeat to manager (every 5s)

**Method**: `Heartbeat(HeartbeatRequest) → HeartbeatResponse`

**Request**:
```protobuf
message HeartbeatRequest {
  string node_id = 1;                               // Node ID
  NodeResources available_resources = 2;            // Current available resources
  repeated ContainerStatus container_statuses = 3;  // Status of assigned containers
}

message ContainerStatus {
  string container_id = 1;  // Container ID
  string actual_state = 2;  // "running", "failed", "stopped"
  string runtime_id = 3;    // Container runtime ID
  string error = 4;         // Error message (if failed)
}
```

**Response**:
```protobuf
message HeartbeatResponse {
  string status = 1;  // "ok" or error
}
```

**Worker Loop**:
```go
ticker := time.NewTicker(5 * time.Second)
for range ticker.C {
    _, err := client.Heartbeat(ctx, &HeartbeatRequest{
        NodeId: nodeID,
        AvailableResources: getCurrentResources(),
        ContainerStatuses:  getContainerStatuses(),
    })
}
```

---

### 3. ListNodes

**Purpose**: Get all cluster nodes

**Method**: `ListNodes(ListNodesRequest) → ListNodesResponse`

**Request**:
```protobuf
message ListNodesRequest {
  string role_filter = 1;  // Optional: "manager" or "worker"
}
```

**Response**:
```protobuf
message ListNodesResponse {
  repeated Node nodes = 1;
}

message Node {
  string id = 1;
  string role = 2;
  string address = 3;
  string overlay_ip = 4;
  NodeResources resources = 5;
  string status = 6;  // "ready", "down", "unknown"
  google.protobuf.Timestamp last_heartbeat = 7;
  google.protobuf.Timestamp created_at = 8;
  map<string, string> labels = 9;
}
```

**CLI Usage**:
```bash
warren node list
warren node list --role worker
```

---

### 4. GetNode

**Purpose**: Get details of a specific node

**Method**: `GetNode(GetNodeRequest) → GetNodeResponse`

**Request**:
```protobuf
message GetNodeRequest {
  string id = 1;
}
```

**Response**:
```protobuf
message GetNodeResponse {
  Node node = 1;
}
```

**CLI Usage**:
```bash
warren node inspect node-abc123
```

---

### 5. RemoveNode

**Purpose**: Remove a node from the cluster

**Method**: `RemoveNode(RemoveNodeRequest) → RemoveNodeResponse`

**Request**:
```protobuf
message RemoveNodeRequest {
  string id = 1;
}
```

**Response**:
```protobuf
message RemoveNodeResponse {
  string status = 1;
}
```

**CLI Usage**:
```bash
warren node remove node-abc123
```

---

## Service Operations

### 6. CreateService

**Purpose**: Deploy a new service

**Method**: `CreateService(CreateServiceRequest) → CreateServiceResponse`

**Request**:
```protobuf
message CreateServiceRequest {
  string name = 1;                          // Service name
  string image = 2;                         // Container image
  int32 replicas = 3;                       // Desired replica count
  string mode = 4;                          // "replicated" or "global"
  string deploy_strategy = 5;               // "rolling", "blue-green", "canary"
  UpdateConfig update_config = 6;           // Update configuration
  HealthCheck health_check = 7;             // Health check config
  RestartPolicy restart_policy = 8;         // Restart policy
  ResourceRequirements resources = 9;       // Resource limits
  repeated string networks = 10;            // Network IDs
  repeated VolumeMount volumes = 11;        // Volume mounts
  map<string, string> env = 12;             // Environment variables
  repeated string command = 13;             // Container command
}
```

**Response**:
```protobuf
message CreateServiceResponse {
  Service service = 1;
}
```

**CLI Usage**:
```bash
warren service create web --image nginx:latest --replicas 3
warren service create api --image myapp:v1 --replicas 5 --env "ENV=prod"
```

**Go Client Example**:
```go
resp, err := client.CreateService(ctx, &CreateServiceRequest{
    Name:     "nginx-web",
    Image:    "nginx:latest",
    Replicas: 3,
    Mode:     "replicated",
    Env: map[string]string{
        "NGINX_PORT": "80",
    },
})
```

---

### 7. UpdateService

**Purpose**: Update an existing service (replicas, image, env)

**Method**: `UpdateService(UpdateServiceRequest) → UpdateServiceResponse`

**Request**:
```protobuf
message UpdateServiceRequest {
  string id = 1;                   // Service ID
  int32 replicas = 2;              // New replica count (optional)
  string image = 3;                // New image (optional)
  map<string, string> env = 4;     // New environment variables (optional)
}
```

**Response**:
```protobuf
message UpdateServiceResponse {
  Service service = 1;
}
```

**CLI Usage**:
```bash
warren service update web --replicas 5
warren service update web --image nginx:1.25
```

---

### 8. DeleteService

**Purpose**: Remove a service and all its tasks

**Method**: `DeleteService(DeleteServiceRequest) → DeleteServiceResponse`

**Request**:
```protobuf
message DeleteServiceRequest {
  string id = 1;
}
```

**Response**:
```protobuf
message DeleteServiceResponse {
  string status = 1;
}
```

**CLI Usage**:
```bash
warren service delete web
```

---

### 9. GetService

**Purpose**: Get details of a specific service

**Method**: `GetService(GetServiceRequest) → GetServiceResponse`

**Request**:
```protobuf
message GetServiceRequest {
  string id = 1;      // Service ID
  string name = 2;    // Or service name
}
```

**Response**:
```protobuf
message GetServiceResponse {
  Service service = 1;
}
```

**CLI Usage**:
```bash
warren service inspect web
```

---

### 10. ListServices

**Purpose**: Get all services

**Method**: `ListServices(ListServicesRequest) → ListServicesResponse`

**Request**:
```protobuf
message ListServicesRequest {}
```

**Response**:
```protobuf
message ListServicesResponse {
  repeated Service services = 1;
}
```

**CLI Usage**:
```bash
warren service list
```

---

## Container Operations

### 11. UpdateContainerStatus

**Purpose**: Worker reports container status changes

**Method**: `UpdateContainerStatus(UpdateContainerStatusRequest) → UpdateContainerStatusResponse`

**Request**:
```protobuf
message UpdateContainerStatusRequest {
  string container_id = 1;   // Container ID
  string node_id = 2;        // Reporting node
  string actual_state = 3;   // "running", "failed", "stopped"
  string runtime_id = 4;     // Container runtime ID
  string error = 5;          // Error message (if failed)
}
```

**Response**:
```protobuf
message UpdateContainerStatusResponse {
  string status = 1;
}
```

**Worker Usage**:
```go
_, err := client.UpdateContainerStatus(ctx, &UpdateContainerStatusRequest{
    ContainerId: container.ID,
    NodeId:      nodeID,
    ActualState: "running",
    RuntimeId:   runtimeID,
})
```

---

### 12. ListTasks

**Purpose**: Get all tasks (optionally filtered)

**Method**: `ListTasks(ListTasksRequest) → ListTasksResponse`

**Request**:
```protobuf
message ListTasksRequest {
  string service_id = 1;  // Optional filter by service
  string node_id = 2;     // Optional filter by node
}
```

**Response**:
```protobuf
message ListTasksResponse {
  repeated Task tasks = 1;
}
```

**CLI Usage**:
```bash
warren task list
warren task list --service web
warren task list --node node-abc123
```

---

### 13. GetTask

**Purpose**: Get details of a specific task

**Method**: `GetTask(GetTaskRequest) → GetTaskResponse`

**Request**:
```protobuf
message GetTaskRequest {
  string id = 1;
}
```

**Response**:
```protobuf
message GetTaskResponse {
  Task task = 1;
}

message Task {
  string id = 1;
  string service_id = 2;
  string service_name = 3;
  string node_id = 4;
  string container_id = 5;
  string desired_state = 6;  // "running", "shutdown"
  string actual_state = 7;   // "pending", "running", "failed", "stopped"
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

**CLI Usage**:
```bash
warren task inspect task-123abc
```

---

### 14. WatchTasks (Server Streaming)

**Purpose**: Worker watches for task assignments (streaming)

**Method**: `WatchTasks(WatchTasksRequest) → stream TaskEvent`

**Request**:
```protobuf
message WatchTasksRequest {
  string node_id = 1;  // Node ID to watch tasks for
}
```

**Response Stream**:
```protobuf
message TaskEvent {
  string type = 1;  // "add", "update", "delete"
  Task task = 2;    // Task details
}
```

**Worker Loop**:
```go
stream, err := client.WatchTasks(ctx, &WatchTasksRequest{
    NodeId: nodeID,
})

for {
    event, err := stream.Recv()
    if err == io.EOF {
        break
    }

    switch event.Type {
    case "add":
        startTask(event.Task)
    case "update":
        updateTask(event.Task)
    case "delete":
        stopTask(event.Task)
    }
}
```

---

## Secret Operations

### 15. CreateSecret

**Purpose**: Create an encrypted secret

**Method**: `CreateSecret(CreateSecretRequest) → CreateSecretResponse`

**Request**:
```protobuf
message CreateSecretRequest {
  string name = 1;
  bytes data = 2;  // Secret data (will be encrypted)
}
```

**Response**:
```protobuf
message CreateSecretResponse {
  Secret secret = 1;
}
```

**CLI Usage**:
```bash
warren secret create db-password --from-file ./password.txt
```

**⏳ Full implementation in Milestone 3**

---

### 16. DeleteSecret

**Purpose**: Delete a secret

**Method**: `DeleteSecret(DeleteSecretRequest) → DeleteSecretResponse`

**Request**:
```protobuf
message DeleteSecretRequest {
  string id = 1;
}
```

**Response**:
```protobuf
message DeleteSecretResponse {
  string status = 1;
}
```

**CLI Usage**:
```bash
warren secret delete db-password
```

---

### 17. ListSecrets

**Purpose**: List all secrets (metadata only, no data)

**Method**: `ListSecrets(ListSecretsRequest) → ListSecretsResponse`

**Request**:
```protobuf
message ListSecretsRequest {}
```

**Response**:
```protobuf
message ListSecretsResponse {
  repeated Secret secrets = 1;
}

message Secret {
  string id = 1;
  string name = 2;
  google.protobuf.Timestamp created_at = 3;
  // Note: data field excluded from list response
}
```

**CLI Usage**:
```bash
warren secret list
```

---

## Volume Operations

### 18. CreateVolume

**Purpose**: Create a persistent volume

**Method**: `CreateVolume(CreateVolumeRequest) → CreateVolumeResponse`

**Request**:
```protobuf
message CreateVolumeRequest {
  string name = 1;
  string driver = 2;                      // "local", "nfs", etc.
  map<string, string> driver_opts = 3;    // Driver-specific options
  map<string, string> labels = 4;         // User-defined labels
}
```

**Response**:
```protobuf
message CreateVolumeResponse {
  Volume volume = 1;
}
```

**CLI Usage**:
```bash
warren volume create web-data --driver local
```

**⏳ Full implementation in Milestone 3**

---

### 19. DeleteVolume

**Purpose**: Delete a volume

**Method**: `DeleteVolume(DeleteVolumeRequest) → DeleteVolumeResponse`

**Request**:
```protobuf
message DeleteVolumeRequest {
  string id = 1;
}
```

**Response**:
```protobuf
message DeleteVolumeResponse {
  string status = 1;
}
```

**CLI Usage**:
```bash
warren volume delete web-data
```

---

### 20. ListVolumes

**Purpose**: List all volumes

**Method**: `ListVolumes(ListVolumesRequest) → ListVolumesResponse`

**Request**:
```protobuf
message ListVolumesRequest {}
```

**Response**:
```protobuf
message ListVolumesResponse {
  repeated Volume volumes = 1;
}

message Volume {
  string id = 1;
  string name = 2;
  string driver = 3;
  map<string, string> driver_opts = 4;
  map<string, string> labels = 5;
  google.protobuf.Timestamp created_at = 6;
}
```

**CLI Usage**:
```bash
warren volume list
```

---

## API Client Implementation

### Go Client

**Connection**:
```go
import (
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
    pb "github.com/cuemby/warren/api/proto"
)

conn, err := grpc.Dial("localhost:2377",
    grpc.WithTransportCredentials(insecure.NewCredentials()))
if err != nil {
    log.Fatal(err)
}
defer conn.Close()

client := pb.NewWarrenAPIClient(conn)
```

**Example Usage**:
```go
// Create service
resp, err := client.CreateService(ctx, &pb.CreateServiceRequest{
    Name:     "nginx",
    Image:    "nginx:latest",
    Replicas: 3,
})

// List services
services, err := client.ListServices(ctx, &pb.ListServicesRequest{})
for _, svc := range services.Services {
    fmt.Printf("Service: %s (%d replicas)\n", svc.Name, svc.Replicas)
}
```

---

### Python Client

**Installation**:
```bash
pip install grpcio grpcio-tools
python -m grpc_tools.protoc -I. --python_out=. --grpc_python_out=. warren.proto
```

**Client Code**:
```python
import grpc
from warren_pb2 import CreateServiceRequest
from warren_pb2_grpc import WarrenAPIStub

channel = grpc.insecure_channel('localhost:2377')
client = WarrenAPIStub(channel)

# Create service
response = client.CreateService(CreateServiceRequest(
    name='nginx',
    image='nginx:latest',
    replicas=3
))
print(f"Service created: {response.service.id}")

# List services
services = client.ListServices(ListServicesRequest())
for svc in services.services:
    print(f"Service: {svc.name} ({svc.replicas} replicas)")
```

---

## API Design Patterns

### 1. Resource CRUD

Standard CRUD operations for all resources:
- **Create**: `CreateService`, `CreateSecret`, `CreateVolume`
- **Read**: `GetService`, `ListServices`
- **Update**: `UpdateService`
- **Delete**: `DeleteService`

### 2. Idempotent Operations

- `RegisterNode`: Re-registering same node updates metadata
- `UpdateTaskStatus`: Repeating same status is safe

### 3. Filtering

List operations support optional filters:
```protobuf
message ListNodesRequest {
  string role_filter = 1;  // "manager" or "worker"
}

message ListTasksRequest {
  string service_id = 1;   // Filter by service
  string node_id = 2;      // Filter by node
}
```

### 4. Streaming for Events

`WatchTasks` uses server streaming for real-time updates:
```protobuf
rpc WatchTasks(WatchTasksRequest) returns (stream TaskEvent);
```

---

## Error Handling

### gRPC Status Codes

Warren uses standard gRPC status codes:

| Code | Status | Usage |
|------|--------|-------|
| 0 | OK | Success |
| 3 | INVALID_ARGUMENT | Invalid request parameters |
| 5 | NOT_FOUND | Resource not found |
| 6 | ALREADY_EXISTS | Duplicate resource (e.g., service name) |
| 13 | INTERNAL | Server error |
| 14 | UNAVAILABLE | Manager unavailable |

**Example Error Response**:
```go
if err != nil {
    st, ok := status.FromError(err)
    if ok {
        fmt.Printf("Error: %s (code: %d)\n", st.Message(), st.Code())
    }
}
```

---

## Performance Considerations

### Request Latency

**Target Latency** (p95):
- Unary RPCs: < 100ms
- Streaming: < 10ms per event

**Current Performance** (Milestone 1):
- CreateService: ~10ms (includes BoltDB write)
- ListServices: ~5ms (small clusters < 100 services)
- Heartbeat: < 5ms

### Throughput

**Single Manager Limits**:
- Heartbeats: 1,000 workers × 0.2 req/s = 200 req/s
- Task updates: ~500 req/s
- Service operations: ~100 req/s

### Scalability

**⏳ Future (Milestone 2+)**:
- Multi-manager HA cluster
- Load balancing across managers
- Raft leader handles writes, followers can serve reads

---

## Health Check Operations (M6)

### ReportTaskHealth

**Purpose**: Worker reports task health status to manager

**Method**: `ReportTaskHealth(ReportTaskHealthRequest) → ReportTaskHealthResponse`

**Request**:
```protobuf
message ReportTaskHealthRequest {
  string task_id = 1;         // Task ID
  string health_status = 2;   // "healthy", "unhealthy", or "unknown"
}
```

**Response**:
```protobuf
message ReportTaskHealthResponse {
  string status = 1;          // Acknowledgment
}
```

**Worker Usage**:
```go
// Health monitor reports status
_, err := client.ReportTaskHealth(ctx, &ReportTaskHealthRequest{
    TaskId:       task.ID,
    HealthStatus: "healthy",
})
```

**Health Check Types** (defined in Service/Task):
- **HTTP**: GET request with expected status code
- **TCP**: Port connectivity check
- **Exec**: Command execution with exit code check

**Reconciler Integration**:
- Reconciler monitors task health status (every 10s)
- Unhealthy tasks marked as failed
- Replacement tasks automatically created

---

## Security

### ✅ Current Implementation (M6 Complete)

- **mTLS**: Mutual TLS for all gRPC connections (TLS 1.3)
- **Certificate Authority**: Self-signed root CA (RSA 4096, 10-year validity)
- **Node Certificates**: 365-day validity with IP SANs
- **Token-based auth**: Bootstrap tokens for worker, manager, and CLI
- **RequestClientCert**: TLS mode for bootstrap flow

**mTLS Flow**:
1. Cluster init generates root CA
2. Bootstrap tokens printed after init (worker, manager, CLI)
3. Workers/CLI request certificate using token
4. Manager issues signed certificate
5. All subsequent gRPC calls use mTLS

**Example mTLS Setup**:
```go
creds, err := credentials.NewServerTLSFromFile(certFile, keyFile)
server := grpc.NewServer(grpc.Creds(creds))
```

### M6 Security RPCs

#### RequestCertificate

**Purpose**: Request signed certificate from CA using bootstrap token

**Method**: `RequestCertificate(RequestCertificateRequest) → RequestCertificateResponse`

**Request**:
```protobuf
message RequestCertificateRequest {
  string token = 1;         // Bootstrap token
  string node_id = 2;       // Node ID
  string role = 3;          // "worker", "manager", or "cli"
  string address = 4;       // Node address (for IP SAN)
}
```

**Response**:
```protobuf
message RequestCertificateResponse {
  bytes certificate = 1;    // PEM-encoded certificate
  bytes ca_cert = 2;        // PEM-encoded CA certificate
}
```

**Usage**:
```bash
# Worker requests certificate
warren worker start --token <bootstrap-token>
```

---

## Testing the API

### grpcurl

Install grpcurl:
```bash
brew install grpcurl  # macOS
```

**List services**:
```bash
grpcurl -plaintext -import-path ./api/proto -proto warren.proto \
  localhost:2377 warren.v1.WarrenAPI/ListServices
```

**Create service**:
```bash
grpcurl -plaintext -import-path ./api/proto -proto warren.proto \
  -d '{"name":"nginx","image":"nginx:latest","replicas":3}' \
  localhost:2377 warren.v1.WarrenAPI/CreateService
```

---

## Related Documentation

- **API Implementation**: [pkg/api/server.go](../../pkg/api/server.go)
- **Protocol Buffers**: [api/proto/warren.proto](../../api/proto/warren.proto)
- **CLI Client**: [pkg/client/client.go](../../pkg/client/client.go)
- **Complete API Reference**: [docs/api-reference.md](../../docs/api-reference.md)
- **Database Schema**: [database-schema.md](./database-schema.md)
- **Project Architecture**: [project-architecture.md](./project-architecture.md)

---

**Version**: 1.6
**Maintained By**: Cuemby Engineering Team
**Last Updated**: 2025-10-11
