# Technical Specification - Warren Container Orchestrator

**Document Version:** 1.0
**Last Updated:** 2025-10-09
**Status:** Approved
**Author:** Cuemby Engineering Team
**Related PRD:** [specs/prd.md](prd.md)

---

## Executive Summary

Warren is a distributed container orchestration system built in Go, combining Raft consensus, containerd integration, and WireGuard networking into a single binary. This technical specification defines the architecture, component design, APIs, and implementation approach for building Warren from scratch.

**Technical Overview**:

- **Language**: Go 1.22+
- **Architecture**: Manager-worker distributed system with Raft consensus
- **Container Runtime**: containerd (CRI-compatible)
- **Networking**: WireGuard overlay mesh
- **Storage**: BoltDB-backed Raft log
- **Packaging**: Single static binary < 100MB
- **Target Platforms**: Linux (primary), macOS, Windows (WSL2), ARM64

---

## Table of Contents

1. [System Architecture](#system-architecture)
2. [Component Design](#component-design)
3. [Data Models](#data-models)
4. [API Specifications](#api-specifications)
5. [Networking Architecture](#networking-architecture)
6. [Security Model](#security-model)
7. [Deployment Strategies](#deployment-strategies)
8. [Storage & State Management](#storage--state-management)
9. [Observability](#observability)
10. [Build & Distribution](#build--distribution)
11. [Testing Strategy](#testing-strategy)
12. [Performance Targets](#performance-targets)

---

## System Architecture

### High-Level Overview

```
┌─────────────────────────────────────────────────────────────┐
│                     Warren Cluster                          │
│                                                             │
│  ┌──────────────────────────────────────────────────┐     │
│  │              Manager Nodes (Raft Quorum)          │     │
│  │                                                    │     │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐       │     │
│  │  │ Manager  │  │ Manager  │  │ Manager  │       │     │
│  │  │  (Leader)│←→│  (Follower)│→│(Follower)│       │     │
│  │  │          │  │          │  │          │       │     │
│  │  │ ┌──────┐ │  │ ┌──────┐ │  │ ┌──────┐ │       │     │
│  │  │ │ Raft │ │  │ │ Raft │ │  │ │ Raft │ │       │     │
│  │  │ └──────┘ │  │ └──────┘ │  │ └──────┘ │       │     │
│  │  │ ┌──────┐ │  │ ┌──────┐ │  │ ┌──────┐ │       │     │
│  │  │ │BoltDB│ │  │ │BoltDB│ │  │ │BoltDB│ │       │     │
│  │  │ └──────┘ │  │ └──────┘ │  │ └──────┘ │       │     │
│  │  └──────────┘  └──────────┘  └──────────┘       │     │
│  │                                                    │     │
│  │  API Server │ Scheduler │ Reconciler             │     │
│  └──────────────────────────────────────────────────┘     │
│                         │                                  │
│                         ▼                                  │
│           ┌──────────────────────────────┐                │
│           │    WireGuard Overlay Network  │                │
│           └──────────────────────────────┘                │
│                         │                                  │
│          ┌──────────────┴──────────────┐                  │
│          ▼                              ▼                  │
│  ┌──────────────┐              ┌──────────────┐           │
│  │ Worker Node  │              │ Worker Node  │           │
│  │              │              │              │           │
│  │ ┌──────────┐ │              │ ┌──────────┐ │           │
│  │ │  Agent   │ │              │ │  Agent   │ │           │
│  │ └──────────┘ │              │ └──────────┘ │           │
│  │ ┌──────────┐ │              │ ┌──────────┐ │           │
│  │ │containerd│ │              │ │containerd│ │           │
│  │ └──────────┘ │              │ └──────────┘ │           │
│  │              │              │              │           │
│  │ [Containers] │              │ [Containers] │           │
│  └──────────────┘              └──────────────┘           │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Design Principles

1. **Single Binary Philosophy**: All components compiled into one executable, mode determined by CLI flags
2. **Raft-First State Management**: All cluster state flows through Raft consensus for consistency
3. **Autonomous Workers**: Workers cache state and operate independently during partitions
4. **Zero-Config Security**: mTLS established automatically, manual override available
5. **Built-In Observability**: Prometheus metrics and structured logging with no external dependencies

### Process Modes

Warren binary operates in different modes based on invocation:

```go
// CLI determines mode
warren cluster init          // Starts manager mode
warren cluster join ...      // Starts worker mode
warren service create ...    // CLI client mode (API calls to manager)
```

**Manager Mode**:
- Runs Raft consensus participant
- Hosts API server (gRPC + HTTP)
- Executes scheduler and reconciler loops
- Manages cluster state in BoltDB

**Worker Mode**:
- Runs worker agent
- Connects to managers via gRPC
- Executes tasks via containerd
- Reports health and metrics

**CLI Mode**:
- Parses commands and flags
- Connects to manager API
- Streams output to user

---

## Component Design

### 1. Manager Components

#### 1.1 Raft Consensus Module

**Purpose**: Distributed state consistency across manager nodes

**Implementation**:
```go
// Using hashicorp/raft library
type RaftNode struct {
    raft      *raft.Raft
    fsm       *WarrenFSM        // Finite State Machine
    transport *raft.NetworkTransport
    store     *raftboltdb.BoltStore
    config    *raft.Config
}

type WarrenFSM struct {
    mu    sync.RWMutex
    state *ClusterState  // In-memory state
}

// FSM applies Raft log entries to state
func (f *WarrenFSM) Apply(log *raft.Log) interface{} {
    var cmd Command
    if err := json.Unmarshal(log.Data, &cmd); err != nil {
        return err
    }

    f.mu.Lock()
    defer f.mu.Unlock()

    switch cmd.Type {
    case "create_service":
        return f.state.CreateService(cmd.Service)
    case "update_service":
        return f.state.UpdateService(cmd.Service)
    case "delete_service":
        return f.state.DeleteService(cmd.ServiceID)
    // ... other commands
    }
}
```

**Key Features**:
- Leader election on cluster init
- Log replication to followers (majority quorum)
- Snapshot/compaction every 10K entries
- Automatic failover (< 10s on leader crash)

**Dependencies**:
- `github.com/hashicorp/raft` (consensus)
- `github.com/hashicorp/raft-boltdb` (log store)

#### 1.2 API Server

**Purpose**: Expose gRPC and REST APIs for CLI and external clients

**Implementation**:
```go
type APIServer struct {
    grpcServer *grpc.Server
    httpServer *http.Server
    raft       *RaftNode
    auth       *AuthManager  // mTLS validation
}

// gRPC service definition (proto)
service WarrenAPI {
    rpc CreateService(CreateServiceRequest) returns (Service);
    rpc ListServices(ListServicesRequest) returns (ServiceList);
    rpc UpdateService(UpdateServiceRequest) returns (Service);
    rpc DeleteService(DeleteServiceRequest) returns (Empty);
    rpc StreamEvents(StreamEventsRequest) returns (stream Event);
    // ... node, task, secret, volume RPCs
}

// REST gateway for HTTP clients
func (s *APIServer) StartHTTPGateway() {
    mux := runtime.NewServeMux()
    opts := []grpc.DialOption{grpc.WithTransportCredentials(s.auth.ClientCreds)}

    pb.RegisterWarrenAPIHandlerFromEndpoint(ctx, mux, grpcEndpoint, opts)
    s.httpServer = &http.Server{Handler: mux}
}
```

**Endpoints**:
- gRPC: `manager:2377` (binary protocol, CLI default)
- HTTP/REST: `manager:2378` (JSON, for web UIs/scripts)
- Metrics: `manager:9090/metrics` (Prometheus)

**Authentication**:
- mTLS required (client cert validation)
- Token-based for CLI (stored in `~/.warren/config`)

#### 1.3 Scheduler

**Purpose**: Place tasks on worker nodes based on resources and constraints

**Implementation**:
```go
type Scheduler struct {
    state     *ClusterState
    raft      *RaftNode
    stopCh    chan struct{}
}

// Scheduling loop
func (s *Scheduler) Run() {
    ticker := time.NewTicker(5 * time.Second)
    for {
        select {
        case <-ticker.C:
            s.scheduleUnassignedTasks()
        case <-s.stopCh:
            return
        }
    }
}

func (s *Scheduler) scheduleUnassignedTasks() {
    tasks := s.state.GetUnassignedTasks()

    for _, task := range tasks {
        node := s.selectNode(task)
        if node == nil {
            continue  // No suitable node
        }

        // Submit via Raft
        cmd := Command{
            Type: "assign_task",
            TaskID: task.ID,
            NodeID: node.ID,
        }
        s.raft.Apply(cmd, 5*time.Second)
    }
}

// Bin-packing algorithm with spread
func (s *Scheduler) selectNode(task *Task) *Node {
    nodes := s.state.GetHealthyNodes()

    // Filter: resources, labels, affinity
    candidates := s.filterNodes(nodes, task)
    if len(candidates) == 0 {
        return nil
    }

    // Score: prefer less-utilized nodes (spread)
    scored := s.scoreNodes(candidates, task)
    sort.Slice(scored, func(i, j int) bool {
        return scored[i].Score > scored[j].Score
    })

    return scored[0].Node
}
```

**Algorithms**:
- **Spread**: Distribute replicas across nodes for availability
- **Bin-packing**: Efficient resource utilization (future)
- **Affinity/Anti-affinity**: Place tasks near/away from others (future)

#### 1.4 Reconciler

**Purpose**: Ensure actual cluster state matches desired state

**Implementation**:
```go
type Reconciler struct {
    state  *ClusterState
    raft   *RaftNode
}

func (r *Reconciler) Run() {
    ticker := time.NewTicker(10 * time.Second)
    for {
        select {
        case <-ticker.C:
            r.reconcile()
        }
    }
}

func (r *Reconciler) reconcile() {
    services := r.state.GetServices()

    for _, svc := range services {
        desired := svc.Replicas
        actual := r.state.GetRunningTaskCount(svc.ID)

        if actual < desired {
            // Scale up
            for i := 0; i < (desired - actual); i++ {
                task := NewTask(svc)
                r.raft.Apply(Command{Type: "create_task", Task: task})
            }
        } else if actual > desired {
            // Scale down
            excess := r.state.GetExcessTasks(svc.ID, actual - desired)
            for _, task := range excess {
                r.raft.Apply(Command{Type: "delete_task", TaskID: task.ID})
            }
        }

        // Check for failed tasks
        failed := r.state.GetFailedTasks(svc.ID)
        for _, task := range failed {
            if task.RestartPolicy.ShouldRestart() {
                replacement := NewTask(svc)
                r.raft.Apply(Command{Type: "create_task", Task: replacement})
                r.raft.Apply(Command{Type: "delete_task", TaskID: task.ID})
            }
        }
    }
}
```

### 2. Worker Components

#### 2.1 Worker Agent

**Purpose**: Execute tasks, report status, health check containers

**Implementation**:
```go
type WorkerAgent struct {
    id           string
    managerAddrs []string
    client       pb.WarrenAPIClient  // gRPC to manager
    runtime      *ContainerdRuntime
    healthChecker *HealthChecker
    cache        *LocalStateCache    // For partition tolerance
}

func (w *WorkerAgent) Run() {
    // Heartbeat to manager
    go w.heartbeatLoop()

    // Watch for task assignments
    go w.watchTasks()

    // Health check assigned tasks
    go w.healthChecker.Run()
}

func (w *WorkerAgent) watchTasks() {
    stream, err := w.client.StreamEvents(ctx, &pb.StreamEventsRequest{
        NodeID: w.id,
        EventTypes: []string{"task_assigned", "task_deleted"},
    })

    for {
        event, err := stream.Recv()
        if err != nil {
            // Network partition? Use cached state
            w.runAutonomous()
            continue
        }

        switch event.Type {
        case "task_assigned":
            w.startTask(event.Task)
        case "task_deleted":
            w.stopTask(event.TaskID)
        }
    }
}

// Autonomous mode during partition
func (w *WorkerAgent) runAutonomous() {
    tasks := w.cache.GetAssignedTasks()

    for _, task := range tasks {
        // Ensure task still running per last-known desired state
        if !w.runtime.IsRunning(task.ContainerID) {
            if task.RestartPolicy.ShouldRestart() {
                w.runtime.StartContainer(task)
            }
        }
    }
}
```

#### 2.2 Containerd Runtime Integration

**Purpose**: Interface with containerd for container lifecycle

**Implementation**:
```go
type ContainerdRuntime struct {
    client    *containerd.Client
    namespace string
}

func NewContainerdRuntime() (*ContainerdRuntime, error) {
    client, err := containerd.New("/run/containerd/containerd.sock")
    if err != nil {
        return nil, err
    }

    return &ContainerdRuntime{
        client:    client,
        namespace: "warren",
    }, nil
}

func (r *ContainerdRuntime) StartContainer(task *Task) (string, error) {
    ctx := namespaces.WithNamespace(context.Background(), r.namespace)

    // Pull image
    image, err := r.client.Pull(ctx, task.Image, containerd.WithPullUnpack)
    if err != nil {
        return "", err
    }

    // Create container
    container, err := r.client.NewContainer(
        ctx,
        task.ID,
        containerd.WithImage(image),
        containerd.WithNewSnapshot(task.ID+"-snapshot", image),
        containerd.WithNewSpec(
            oci.WithImageConfig(image),
            oci.WithEnv(task.Env),
            oci.WithMounts(task.Mounts),
        ),
    )

    // Start task (containerd task, not Warren task)
    cTask, err := container.NewTask(ctx, cio.NewCreator(cio.WithStdio))
    if err != nil {
        return "", err
    }

    if err := cTask.Start(ctx); err != nil {
        return "", err
    }

    return container.ID(), nil
}

func (r *ContainerdRuntime) StopContainer(containerID string) error {
    ctx := namespaces.WithNamespace(context.Background(), r.namespace)

    container, err := r.client.LoadContainer(ctx, containerID)
    if err != nil {
        return err
    }

    task, err := container.Task(ctx, nil)
    if err != nil {
        return err
    }

    // Graceful shutdown
    if err := task.Kill(ctx, syscall.SIGTERM); err != nil {
        return err
    }

    // Wait with timeout
    _, err = task.Wait(ctx)
    return err
}
```

#### 2.3 Health Checker

**Purpose**: Monitor container health via HTTP/TCP/exec probes

**Implementation**:
```go
type HealthChecker struct {
    runtime *ContainerdRuntime
    tasks   map[string]*HealthCheck
    mu      sync.RWMutex
}

type HealthCheck struct {
    TaskID       string
    Type         string  // http, tcp, exec
    Endpoint     string  // URL or address
    Interval     time.Duration
    Timeout      time.Duration
    Retries      int
    FailureCount int
}

func (hc *HealthChecker) Run() {
    ticker := time.NewTicker(5 * time.Second)
    for {
        <-ticker.C
        hc.checkAll()
    }
}

func (hc *HealthChecker) checkAll() {
    hc.mu.RLock()
    defer hc.mu.RUnlock()

    for _, check := range hc.tasks {
        go hc.check(check)
    }
}

func (hc *HealthChecker) check(check *HealthCheck) {
    var healthy bool

    switch check.Type {
    case "http":
        healthy = hc.httpCheck(check.Endpoint, check.Timeout)
    case "tcp":
        healthy = hc.tcpCheck(check.Endpoint, check.Timeout)
    case "exec":
        healthy = hc.execCheck(check.TaskID, check.Endpoint)
    }

    if !healthy {
        check.FailureCount++
        if check.FailureCount >= check.Retries {
            // Report to manager via gRPC
            // Manager will reschedule task
        }
    } else {
        check.FailureCount = 0
    }
}
```

### 3. CLI Components

#### 3.1 Command Structure

**Implementation** (using Cobra):
```go
// Root command
var rootCmd = &cobra.Command{
    Use:   "warren",
    Short: "Warren container orchestrator",
}

// Cluster commands
var clusterCmd = &cobra.Command{
    Use:   "cluster",
    Short: "Manage Warren cluster",
}

var clusterInitCmd = &cobra.Command{
    Use:   "init",
    Short: "Initialize a new cluster",
    Run: func(cmd *cobra.Command, args []string) {
        // Start manager mode
        mgr := manager.New(config)
        mgr.Run()
    },
}

// Service commands
var serviceCmd = &cobra.Command{
    Use:   "service",
    Short: "Manage services",
}

var serviceCreateCmd = &cobra.Command{
    Use:   "create [NAME]",
    Short: "Create a new service",
    Run: func(cmd *cobra.Command, args []string) {
        client := newAPIClient()
        svc, err := client.CreateService(ctx, &pb.CreateServiceRequest{
            Name:     args[0],
            Image:    image,
            Replicas: replicas,
        })
        // ...
    },
}

// Short alias
func init() {
    rootCmd.AddCommand(clusterCmd, serviceCmd, nodeCmd, secretCmd)
    clusterCmd.AddCommand(clusterInitCmd, clusterJoinCmd)

    // Register "wrn" as alias binary (symlink or hardlink)
}
```

---

## Data Models

### Core Entities

```go
// Cluster represents the entire Warren cluster
type Cluster struct {
    ID              string
    CreatedAt       time.Time
    Managers        []*Node
    Workers         []*Node
    RaftLeader      string
    NetworkConfig   *NetworkConfig
}

// Node represents a manager or worker
type Node struct {
    ID            string
    Role          string  // "manager" or "worker"
    Address       string
    Hostname      string
    Labels        map[string]string
    Resources     *NodeResources
    Status        string  // "ready", "down", "unknown"
    LastHeartbeat time.Time
}

type NodeResources struct {
    CPUCores      int
    MemoryBytes   int64
    DiskBytes     int64

    // Allocated (reserved by tasks)
    CPUAllocated    float64
    MemoryAllocated int64
    DiskAllocated   int64
}

// Service represents a user-defined workload
type Service struct {
    ID              string
    Name            string
    Image           string
    Replicas        int
    DeployStrategy  string  // "rolling", "blue-green", "canary"
    UpdateConfig    *UpdateConfig
    Networks        []string
    Secrets         []string
    Volumes         []*VolumeMount
    Labels          map[string]string
    CreatedAt       time.Time
    UpdatedAt       time.Time
}

type UpdateConfig struct {
    Parallelism   int           // How many replicas to update simultaneously
    Delay         time.Duration // Delay between updates
    FailureAction string        // "pause", "rollback", "continue"
    CanaryWeight  int           // 0-100 (for canary strategy)
}

// Task represents a single container instance
type Task struct {
    ID            string
    ServiceID     string
    NodeID        string
    ContainerID   string
    DesiredState  string  // "running", "shutdown"
    ActualState   string  // "pending", "running", "failed", "complete"
    Image         string
    Env           []string
    Mounts        []*Mount
    RestartPolicy *RestartPolicy
    HealthCheck   *HealthCheck
    CreatedAt     time.Time
    StartedAt     time.Time
    FinishedAt    time.Time
}

type RestartPolicy struct {
    Condition    string  // "none", "on-failure", "always"
    MaxAttempts  int
    Delay        time.Duration
}

// Secret represents encrypted sensitive data
type Secret struct {
    ID          string
    Name        string
    Data        []byte  // Encrypted
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

// Volume represents persistent storage
type Volume struct {
    ID         string
    Name       string
    Driver     string  // "local", "nfs", etc.
    NodeID     string  // For local volumes, which node
    MountPath  string
    Options    map[string]string
    CreatedAt  time.Time
}

// Network represents overlay network
type Network struct {
    ID       string
    Name     string
    Subnet   string  // CIDR
    Gateway  string
    Driver   string  // "wireguard"
}
```

### State Storage

All state stored in BoltDB with following buckets:

```go
var (
    BucketNodes     = []byte("nodes")
    BucketServices  = []byte("services")
    BucketTasks     = []byte("tasks")
    BucketSecrets   = []byte("secrets")
    BucketVolumes   = []byte("volumes")
    BucketNetworks  = []byte("networks")
)

type ClusterState struct {
    db *bolt.DB
}

func (s *ClusterState) CreateService(svc *Service) error {
    return s.db.Update(func(tx *bolt.Tx) error {
        b := tx.Bucket(BucketServices)
        data, err := json.Marshal(svc)
        if err != nil {
            return err
        }
        return b.Put([]byte(svc.ID), data)
    })
}

func (s *ClusterState) GetService(id string) (*Service, error) {
    var svc Service
    err := s.db.View(func(tx *bolt.Tx) error {
        b := tx.Bucket(BucketServices)
        data := b.Get([]byte(id))
        if data == nil {
            return ErrNotFound
        }
        return json.Unmarshal(data, &svc)
    })
    return &svc, err
}
```

---

## API Specifications

### gRPC API (Primary)

**Protocol Buffers Definition**:

```protobuf
syntax = "proto3";

package warren.v1;

// Service management
service WarrenAPI {
    // Services
    rpc CreateService(CreateServiceRequest) returns (Service);
    rpc GetService(GetServiceRequest) returns (Service);
    rpc ListServices(ListServicesRequest) returns (ListServicesResponse);
    rpc UpdateService(UpdateServiceRequest) returns (Service);
    rpc DeleteService(DeleteServiceRequest) returns (google.protobuf.Empty);
    rpc ScaleService(ScaleServiceRequest) returns (Service);

    // Nodes
    rpc ListNodes(ListNodesRequest) returns (ListNodesResponse);
    rpc GetNode(GetNodeRequest) returns (Node);
    rpc UpdateNode(UpdateNodeRequest) returns (Node);
    rpc DrainNode(DrainNodeRequest) returns (google.protobuf.Empty);

    // Tasks
    rpc ListTasks(ListTasksRequest) returns (ListTasksResponse);
    rpc GetTask(GetTaskRequest) returns (Task);

    // Secrets
    rpc CreateSecret(CreateSecretRequest) returns (Secret);
    rpc ListSecrets(ListSecretsRequest) returns (ListSecretsResponse);
    rpc DeleteSecret(DeleteSecretRequest) returns (google.protobuf.Empty);

    // Volumes
    rpc CreateVolume(CreateVolumeRequest) returns (Volume);
    rpc ListVolumes(ListVolumesRequest) returns (ListVolumesResponse);
    rpc DeleteVolume(DeleteVolumeRequest) returns (google.protobuf.Empty);

    // Events (streaming)
    rpc StreamEvents(StreamEventsRequest) returns (stream Event);
}

message Service {
    string id = 1;
    string name = 2;
    string image = 3;
    int32 replicas = 4;
    string deploy_strategy = 5;
    UpdateConfig update_config = 6;
    repeated string networks = 7;
    repeated string secrets = 8;
    map<string, string> labels = 9;
    google.protobuf.Timestamp created_at = 10;
    google.protobuf.Timestamp updated_at = 11;
}

message CreateServiceRequest {
    string name = 1;
    string image = 2;
    int32 replicas = 3;
    string deploy_strategy = 4;
    UpdateConfig update_config = 5;
    repeated Port ports = 6;
    repeated string networks = 7;
    map<string, string> labels = 8;
}

message UpdateConfig {
    int32 parallelism = 1;
    google.protobuf.Duration delay = 2;
    string failure_action = 3;
    int32 canary_weight = 4;
}

message Event {
    string type = 1;  // "service_created", "task_failed", etc.
    google.protobuf.Timestamp timestamp = 2;
    google.protobuf.Any payload = 3;
}
```

### REST API (via gRPC-Gateway)

```
POST   /v1/services
GET    /v1/services
GET    /v1/services/{id}
PUT    /v1/services/{id}
DELETE /v1/services/{id}
POST   /v1/services/{id}/scale

GET    /v1/nodes
GET    /v1/nodes/{id}
PUT    /v1/nodes/{id}
POST   /v1/nodes/{id}/drain

GET    /v1/tasks
GET    /v1/tasks/{id}

POST   /v1/secrets
GET    /v1/secrets
DELETE /v1/secrets/{id}

POST   /v1/volumes
GET    /v1/volumes
DELETE /v1/volumes/{id}

GET    /v1/events (SSE - Server-Sent Events)
```

### YAML Manifest Format

```yaml
# warren.yaml
version: "1"

services:
  web:
    image: nginx:latest
    replicas: 3
    deploy:
      strategy: rolling
      update_config:
        parallelism: 1
        delay: 10s
        failure_action: rollback
    ports:
      - target: 80
        published: 8080
        protocol: tcp
    networks:
      - frontend
    health_check:
      type: http
      endpoint: /health
      interval: 10s
      timeout: 5s
      retries: 3
    resources:
      limits:
        cpus: "0.5"
        memory: 512M
      reservations:
        cpus: "0.25"
        memory: 256M
    labels:
      com.example.team: "platform"

  api:
    image: myapp:v1.0
    replicas: 5
    deploy:
      strategy: canary
      update_config:
        canary_weight: 10
    secrets:
      - db_password
    volumes:
      - api-data:/var/lib/api
    networks:
      - frontend
      - backend

networks:
  frontend:
    driver: wireguard
    subnet: 10.0.1.0/24
  backend:
    driver: wireguard
    subnet: 10.0.2.0/24

secrets:
  db_password:
    external: true  # Created separately via CLI

volumes:
  api-data:
    driver: local
```

---

## Networking Architecture

### WireGuard Overlay Network

**Design**:
- Each node gets WireGuard interface (`wg0`) with unique IP in cluster subnet
- Mesh topology: all nodes peer with all other nodes
- Encryption: ChaCha20-Poly1305 (WireGuard default)

**Implementation**:

```go
type NetworkManager struct {
    iface      string  // "wg0"
    subnet     *net.IPNet
    privateKey wgtypes.Key
    publicKey  wgtypes.Key
    peers      map[string]*Peer
}

type Peer struct {
    NodeID     string
    PublicKey  wgtypes.Key
    Endpoint   string  // IP:port
    AllowedIPs []net.IPNet
}

func (nm *NetworkManager) Initialize(clusterSubnet string) error {
    // Generate key pair
    privateKey, err := wgtypes.GeneratePrivateKey()
    if err != nil {
        return err
    }
    nm.privateKey = privateKey
    nm.publicKey = privateKey.PublicKey()

    // Parse subnet
    _, subnet, err := net.ParseCIDR(clusterSubnet)
    if err != nil {
        return err
    }
    nm.subnet = subnet

    // Create WireGuard interface
    return nm.createInterface()
}

func (nm *NetworkManager) createInterface() error {
    client, err := wgctrl.New()
    if err != nil {
        return err
    }
    defer client.Close()

    // Create interface (via netlink)
    link := &netlink.Link{
        Name: nm.iface,
        Type: "wireguard",
    }
    if err := netlink.LinkAdd(link); err != nil {
        return err
    }

    // Configure WireGuard
    cfg := wgtypes.Config{
        PrivateKey: &nm.privateKey,
        ListenPort: 51820,
    }
    return client.ConfigureDevice(nm.iface, cfg)
}

func (nm *NetworkManager) AddPeer(node *Node) error {
    peer := wgtypes.PeerConfig{
        PublicKey: node.WireGuardPublicKey,
        Endpoint: &net.UDPAddr{
            IP:   net.ParseIP(node.Address),
            Port: 51820,
        },
        AllowedIPs: []net.IPNet{
            {IP: node.OverlayIP, Mask: net.CIDRMask(32, 32)},
        },
    }

    client, _ := wgctrl.New()
    defer client.Close()

    return client.ConfigureDevice(nm.iface, wgtypes.Config{
        Peers: []wgtypes.PeerConfig{peer},
    })
}
```

### DNS Service

**Purpose**: Resolve service names to VIPs

**Implementation**:

```go
// Embedded DNS server (port 53 on managers)
type DNSServer struct {
    state *ClusterState
}

func (d *DNSServer) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
    msg := dns.Msg{}
    msg.SetReply(r)

    for _, q := range r.Question {
        if q.Qtype == dns.TypeA {
            // Look up service by name
            svc, err := d.state.GetServiceByName(q.Name)
            if err != nil {
                continue
            }

            // Return service VIP
            rr := &dns.A{
                Hdr: dns.RR_Header{
                    Name:   q.Name,
                    Rrtype: dns.TypeA,
                    Class:  dns.ClassINET,
                    Ttl:    30,
                },
                A: svc.VIP,
            }
            msg.Answer = append(msg.Answer, rr)
        }
    }

    w.WriteMsg(&msg)
}
```

### Service VIP & Load Balancing

**Design**:
- Each service assigned a Virtual IP (VIP) from cluster subnet
- iptables rules on each node route VIP traffic to healthy replicas
- Round-robin load balancing

**Implementation**:

```go
func (nm *NetworkManager) CreateServiceVIP(svc *Service, tasks []*Task) error {
    vip := nm.allocateVIP(svc.ID)

    // iptables NAT rules for load balancing
    for i, task := range tasks {
        if task.ActualState != "running" {
            continue
        }

        // DNAT rule: VIP:port -> task_container_ip:port
        rule := fmt.Sprintf(
            "-t nat -A WARREN-LB -d %s -p tcp --dport %d -m statistic --mode nth --every %d --packet 0 -j DNAT --to-destination %s:%d",
            vip, svc.Port, len(tasks)-i, task.ContainerIP, task.ContainerPort,
        )
        exec.Command("iptables", strings.Fields(rule)...).Run()
    }

    return nil
}
```

---

## Security Model

### mTLS Certificate Infrastructure

**Design**:
- Self-signed root CA created on first manager
- Manager certificates signed by root CA
- Worker certificates signed by root CA
- Automatic rotation every 90 days

**Implementation**:

```go
type CertAuthority struct {
    rootCert    *x509.Certificate
    rootKey     *rsa.PrivateKey
    certCache   map[string]*tls.Certificate
}

func (ca *CertAuthority) Initialize() error {
    // Generate root CA
    rootKey, _ := rsa.GenerateKey(rand.Reader, 4096)

    template := &x509.Certificate{
        SerialNumber: big.NewInt(1),
        Subject: pkix.Name{
            Organization: []string{"Warren Cluster"},
            CommonName:   "Warren Root CA",
        },
        NotBefore:             time.Now(),
        NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
        KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
        IsCA:                  true,
        BasicConstraintsValid: true,
    }

    certDER, _ := x509.CreateCertificate(rand.Reader, template, template, &rootKey.PublicKey, rootKey)
    ca.rootCert, _ = x509.ParseCertificate(certDER)
    ca.rootKey = rootKey

    return nil
}

func (ca *CertAuthority) IssueNodeCertificate(nodeID, role string) (*tls.Certificate, error) {
    nodeKey, _ := rsa.GenerateKey(rand.Reader, 2048)

    template := &x509.Certificate{
        SerialNumber: big.NewInt(time.Now().Unix()),
        Subject: pkix.Name{
            Organization: []string{"Warren Cluster"},
            CommonName:   fmt.Sprintf("%s-%s", role, nodeID),
        },
        NotBefore:    time.Now(),
        NotAfter:     time.Now().Add(90 * 24 * time.Hour),  // 90 days
        KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
        ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
    }

    certDER, _ := x509.CreateCertificate(rand.Reader, template, ca.rootCert, &nodeKey.PublicKey, ca.rootKey)

    cert := &tls.Certificate{
        Certificate: [][]byte{certDER},
        PrivateKey:  nodeKey,
    }

    return cert, nil
}
```

### Secrets Encryption

**Design**:
- Secrets encrypted at rest with AES-256-GCM
- Encryption key derived from cluster init token (stored in Raft)
- Secrets mounted to containers as tmpfs (memory-only)

**Implementation**:

```go
type SecretsManager struct {
    encryptionKey []byte  // 32 bytes for AES-256
}

func (sm *SecretsManager) EncryptSecret(plaintext []byte) ([]byte, error) {
    block, _ := aes.NewCipher(sm.encryptionKey)
    gcm, _ := cipher.NewGCM(block)

    nonce := make([]byte, gcm.NonceSize())
    io.ReadFull(rand.Reader, nonce)

    ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
    return ciphertext, nil
}

func (sm *SecretsManager) DecryptSecret(ciphertext []byte) ([]byte, error) {
    block, _ := aes.NewCipher(sm.encryptionKey)
    gcm, _ := cipher.NewGCM(block)

    nonceSize := gcm.NonceSize()
    nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

    plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
    return plaintext, err
}

// Mount to container
func (sm *SecretsManager) MountSecret(containerID, secretName string) error {
    secret, _ := sm.state.GetSecret(secretName)
    plaintext, _ := sm.DecryptSecret(secret.Data)

    // Write to tmpfs
    tmpfsPath := fmt.Sprintf("/run/secrets/%s", containerID)
    os.MkdirAll(tmpfsPath, 0700)

    secretPath := filepath.Join(tmpfsPath, secretName)
    return ioutil.WriteFile(secretPath, plaintext, 0400)  // Read-only
}
```

---

## Deployment Strategies

### 1. Rolling Update

**Algorithm**:
```go
func (d *Deployer) RollingUpdate(svc *Service, newImage string) error {
    tasks := d.state.GetServiceTasks(svc.ID)
    parallelism := svc.UpdateConfig.Parallelism
    delay := svc.UpdateConfig.Delay

    for i := 0; i < len(tasks); i += parallelism {
        batch := tasks[i:min(i+parallelism, len(tasks))]

        for _, oldTask := range batch {
            // Create new task with updated image
            newTask := oldTask.Clone()
            newTask.Image = newImage
            d.raft.Apply(Command{Type: "create_task", Task: newTask})

            // Wait for new task to be healthy
            if err := d.waitHealthy(newTask.ID, 60*time.Second); err != nil {
                // Rollback on failure
                if svc.UpdateConfig.FailureAction == "rollback" {
                    return d.rollback(svc)
                }
                return err
            }

            // Stop old task
            d.raft.Apply(Command{Type: "delete_task", TaskID: oldTask.ID})
        }

        // Delay before next batch
        time.Sleep(delay)
    }

    return nil
}
```

### 2. Blue/Green Deployment

**Algorithm**:
```go
func (d *Deployer) BlueGreenUpdate(svc *Service, newImage string) error {
    // Deploy "green" (new version) alongside "blue" (current)
    greenTasks := make([]*Task, svc.Replicas)
    for i := 0; i < svc.Replicas; i++ {
        task := NewTask(svc)
        task.Image = newImage
        task.Labels["warren.deploy.color"] = "green"

        d.raft.Apply(Command{Type: "create_task", Task: task})
        greenTasks[i] = task
    }

    // Wait for all green tasks healthy
    for _, task := range greenTasks {
        if err := d.waitHealthy(task.ID, 120*time.Second); err != nil {
            // Cleanup green tasks on failure
            for _, t := range greenTasks {
                d.raft.Apply(Command{Type: "delete_task", TaskID: t.ID})
            }
            return err
        }
    }

    // Switch traffic: update service VIP to point to green tasks
    d.networkManager.UpdateServiceVIP(svc.ID, greenTasks)

    // Delay for monitoring
    time.Sleep(30 * time.Second)

    // Cleanup blue tasks
    blueTasks := d.state.GetTasksByLabel(svc.ID, "warren.deploy.color", "blue")
    for _, task := range blueTasks {
        d.raft.Apply(Command{Type: "delete_task", TaskID: task.ID})
    }

    return nil
}
```

### 3. Canary Deployment

**Algorithm**:
```go
func (d *Deployer) CanaryUpdate(svc *Service, newImage string, weight int) error {
    // Deploy canary tasks (weight% of total replicas)
    canaryCount := (svc.Replicas * weight) / 100
    if canaryCount == 0 {
        canaryCount = 1
    }

    canaryTasks := make([]*Task, canaryCount)
    for i := 0; i < canaryCount; i++ {
        task := NewTask(svc)
        task.Image = newImage
        task.Labels["warren.deploy.canary"] = "true"

        d.raft.Apply(Command{Type: "create_task", Task: task})
        canaryTasks[i] = task
    }

    // Wait for canary health
    for _, task := range canaryTasks {
        if err := d.waitHealthy(task.ID, 60*time.Second); err != nil {
            return err
        }
    }

    // Update load balancer weights
    stableTasks := d.state.GetTasksByLabel(svc.ID, "warren.deploy.canary", "false")
    d.networkManager.UpdateServiceVIPWeighted(svc.ID, stableTasks, canaryTasks, weight)

    return nil
}

// Promote canary to stable
func (d *Deployer) PromoteCanary(svcID string) error {
    canaryTasks := d.state.GetTasksByLabel(svcID, "warren.deploy.canary", "true")

    // Update labels
    for _, task := range canaryTasks {
        task.Labels["warren.deploy.canary"] = "false"
        d.raft.Apply(Command{Type: "update_task", Task: task})
    }

    // Scale to full replicas with new image
    svc, _ := d.state.GetService(svcID)
    return d.RollingUpdate(svc, canaryTasks[0].Image)
}
```

---

## Storage & State Management

### Raft Log Compaction

**Snapshot Strategy**:
```go
func (r *RaftNode) snapshotLoop() {
    ticker := time.NewTicker(5 * time.Minute)
    for {
        <-ticker.C

        // Snapshot if log exceeds 10K entries
        if r.raft.Stats()["last_log_index"] > "10000" {
            r.snapshot()
        }
    }
}

func (r *RaftNode) snapshot() error {
    // Serialize current state
    snapshot, err := r.fsm.Snapshot()
    if err != nil {
        return err
    }

    // Persist snapshot
    future := r.raft.Snapshot()
    return future.Error()
}

// FSM Snapshot implementation
func (f *WarrenFSM) Snapshot() (raft.FSMSnapshot, error) {
    f.mu.RLock()
    defer f.mu.RUnlock()

    // Deep copy state
    clone := f.state.Clone()

    return &WarrenSnapshot{state: clone}, nil
}

type WarrenSnapshot struct {
    state *ClusterState
}

func (s *WarrenSnapshot) Persist(sink raft.SnapshotSink) error {
    data, err := json.Marshal(s.state)
    if err != nil {
        sink.Cancel()
        return err
    }

    if _, err := sink.Write(data); err != nil {
        sink.Cancel()
        return err
    }

    return sink.Close()
}
```

### Backup & Restore

**Backup**:
```bash
# Backup creates Raft snapshot + BoltDB dump
warren cluster backup --output /path/to/backup.tar.gz
```

**Implementation**:
```go
func (m *Manager) Backup(outputPath string) error {
    // Force Raft snapshot
    future := m.raft.Snapshot()
    if err := future.Error(); err != nil {
        return err
    }

    // Copy snapshot files
    snapshotPath := filepath.Join(m.dataDir, "snapshots")
    boltPath := filepath.Join(m.dataDir, "raft.db")

    // Create tarball
    tarball, _ := os.Create(outputPath)
    defer tarball.Close()

    gw := gzip.NewWriter(tarball)
    defer gw.Close()

    tw := tar.NewWriter(gw)
    defer tw.Close()

    // Add snapshot and BoltDB to tar
    addFileToTar(tw, snapshotPath)
    addFileToTar(tw, boltPath)

    return nil
}
```

---

## Observability

### Prometheus Metrics

**Exposed Metrics**:

```go
var (
    // Cluster metrics
    nodeCount = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "warren_nodes_total",
            Help: "Total number of nodes by role and status",
        },
        []string{"role", "status"},
    )

    serviceCount = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name: "warren_services_total",
            Help: "Total number of services",
        },
    )

    taskCount = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "warren_tasks_total",
            Help: "Total number of tasks by state",
        },
        []string{"state"},
    )

    // Raft metrics
    raftLeader = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name: "warren_raft_is_leader",
            Help: "1 if this node is Raft leader, 0 otherwise",
        },
    )

    raftLogIndex = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name: "warren_raft_last_log_index",
            Help: "Last Raft log index",
        },
    )

    // Container metrics
    containerCPU = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "warren_container_cpu_usage_percent",
            Help: "Container CPU usage percentage",
        },
        []string{"task_id", "service_name"},
    )

    containerMemory = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "warren_container_memory_usage_bytes",
            Help: "Container memory usage in bytes",
        },
        []string{"task_id", "service_name"},
    )
)

func init() {
    prometheus.MustRegister(nodeCount, serviceCount, taskCount)
    prometheus.MustRegister(raftLeader, raftLogIndex)
    prometheus.MustRegister(containerCPU, containerMemory)
}

// Metrics server
func (m *Manager) startMetricsServer() {
    http.Handle("/metrics", promhttp.Handler())
    http.ListenAndServe(":9090", nil)
}
```

### Structured Logging

**Implementation** (using zerolog):

```go
import "github.com/rs/zerolog/log"

func (s *Scheduler) scheduleTask(task *Task) {
    log.Info().
        Str("component", "scheduler").
        Str("task_id", task.ID).
        Str("service_id", task.ServiceID).
        Str("node_id", task.NodeID).
        Msg("scheduling task")

    if err := s.placeTask(task); err != nil {
        log.Error().
            Err(err).
            Str("component", "scheduler").
            Str("task_id", task.ID).
            Msg("failed to schedule task")
        return
    }

    log.Info().
        Str("component", "scheduler").
        Str("task_id", task.ID).
        Str("node_id", task.NodeID).
        Msg("task scheduled successfully")
}

// Configure log level
func main() {
    zerolog.SetGlobalLevel(zerolog.InfoLevel)

    if os.Getenv("WARREN_LOG_LEVEL") == "debug" {
        zerolog.SetGlobalLevel(zerolog.DebugLevel)
    }

    // JSON output
    log.Logger = log.Output(os.Stdout)
}
```

---

## Build & Distribution

### Build Configuration

**Makefile**:

```makefile
VERSION := $(shell git describe --tags --always --dirty)
COMMIT := $(shell git rev-parse --short HEAD)
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')

LDFLAGS := -X main.Version=$(VERSION) \
           -X main.Commit=$(COMMIT) \
           -X main.BuildTime=$(BUILD_TIME) \
           -s -w  # Strip debug info

.PHONY: build
build:
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/warren ./cmd/warren

.PHONY: build-all
build-all:
	# Linux amd64
	GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o bin/warren-linux-amd64 ./cmd/warren

	# Linux arm64
	GOOS=linux GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o bin/warren-linux-arm64 ./cmd/warren

	# macOS amd64
	GOOS=darwin GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o bin/warren-darwin-amd64 ./cmd/warren

	# macOS arm64
	GOOS=darwin GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o bin/warren-darwin-arm64 ./cmd/warren

.PHONY: compress
compress: build-all
	upx --best --lzma bin/warren-*

.PHONY: install
install: build
	cp bin/warren /usr/local/bin/warren
	ln -sf /usr/local/bin/warren /usr/local/bin/wrn
```

### Binary Size Optimization

**Techniques**:

1. **Dead code elimination**: `-ldflags="-s -w"` strips debug symbols
2. **Compression**: UPX compression reduces binary ~60%
3. **Build tags**: Exclude unnecessary features at compile time
4. **Dependency audit**: Avoid bloated libraries

```bash
# Before optimization
-rwxr-xr-x  1 user  staff   142M warren

# After -ldflags="-s -w"
-rwxr-xr-x  1 user  staff    98M warren

# After UPX --best
-rwxr-xr-x  1 user  staff    42M warren
```

### Package Distribution

**Homebrew (macOS)**:

```ruby
# warren.rb
class Warren < Formula
  desc "Simple yet powerful container orchestrator"
  homepage "https://github.com/cuemby/warren"
  url "https://github.com/cuemby/warren/releases/download/v1.0.0/warren-darwin-amd64.tar.gz"
  sha256 "..."

  def install
    bin.install "warren"
    bin.install_symlink "warren" => "wrn"
  end
end
```

**APT (Debian/Ubuntu)**:

```bash
# Build .deb package
fpm -s dir -t deb -n warren -v 1.0.0 \
    --description "Warren container orchestrator" \
    --url "https://warren.io" \
    --license "Apache-2.0" \
    bin/warren=/usr/local/bin/warren
```

---

## Testing Strategy

### Unit Tests

**Coverage Target**: 80%+

```go
// scheduler_test.go
func TestScheduler_SelectNode(t *testing.T) {
    tests := []struct {
        name      string
        task      *Task
        nodes     []*Node
        wantNode  string
        wantError bool
    }{
        {
            name: "selects node with most available resources",
            task: &Task{
                Resources: &ResourceRequirements{
                    CPUCores: 1,
                    Memory:   1 * GB,
                },
            },
            nodes: []*Node{
                {ID: "node1", Resources: &NodeResources{CPUCores: 4, MemoryBytes: 8*GB, CPUAllocated: 2}},
                {ID: "node2", Resources: &NodeResources{CPUCores: 4, MemoryBytes: 8*GB, CPUAllocated: 1}},
            },
            wantNode: "node2",
        },
        {
            name: "returns error if no node has capacity",
            task: &Task{
                Resources: &ResourceRequirements{
                    CPUCores: 4,
                    Memory:   16 * GB,
                },
            },
            nodes: []*Node{
                {ID: "node1", Resources: &NodeResources{CPUCores: 2, MemoryBytes: 4*GB}},
            },
            wantError: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            s := &Scheduler{state: &ClusterState{nodes: tt.nodes}}
            node := s.selectNode(tt.task)

            if tt.wantError {
                require.Nil(t, node)
            } else {
                require.Equal(t, tt.wantNode, node.ID)
            }
        })
    }
}
```

### Integration Tests

**Test Scenarios**:

1. **Cluster Initialization**: Manager starts, Raft initializes, BoltDB created
2. **Worker Join**: Worker joins, heartbeat established, appears in `warren node list`
3. **Service Deployment**: Service created, tasks scheduled, containers start
4. **Failover**: Kill leader, new leader elected, cluster remains operational
5. **Network Partition**: Partition worker, verify autonomous operation, verify reconciliation on rejoin

```go
// integration_test.go
func TestClusterInitAndJoin(t *testing.T) {
    // Start manager
    mgr := startManager(t, 2377)
    defer mgr.Stop()

    // Wait for leader election
    waitForLeader(t, mgr, 10*time.Second)

    // Start worker
    token := mgr.GetJoinToken("worker")
    worker := startWorker(t, token, mgr.Address())
    defer worker.Stop()

    // Verify worker appears in cluster
    require.Eventually(t, func() bool {
        nodes := mgr.ListNodes()
        return len(nodes) == 2 && nodes[1].Role == "worker"
    }, 30*time.Second, 1*time.Second)
}

func TestServiceDeployment(t *testing.T) {
    cluster := startCluster(t, 1, 2)  // 1 manager, 2 workers
    defer cluster.Stop()

    // Create service
    svc := cluster.CreateService(&Service{
        Name:     "web",
        Image:    "nginx:latest",
        Replicas: 3,
    })

    // Verify tasks scheduled and running
    require.Eventually(t, func() bool {
        tasks := cluster.ListTasks(svc.ID)
        running := 0
        for _, task := range tasks {
            if task.ActualState == "running" {
                running++
            }
        }
        return running == 3
    }, 60*time.Second, 2*time.Second)
}
```

### Chaos/Partition Testing

**Jepsen-style tests**:

```go
func TestNetworkPartition(t *testing.T) {
    cluster := startCluster(t, 3, 3)  // 3 managers, 3 workers
    defer cluster.Stop()

    svc := cluster.CreateService(&Service{Name: "test", Replicas: 3})
    waitForRunning(t, svc.ID, 3)

    // Partition one worker from managers
    worker := cluster.Workers[0]
    cluster.PartitionNode(worker.ID)

    // Verify worker continues running tasks
    time.Sleep(30 * time.Second)
    tasks := worker.GetLocalTasks()
    require.Equal(t, 1, len(tasks))
    require.Equal(t, "running", tasks[0].ActualState)

    // Crash container on partitioned worker
    worker.StopContainer(tasks[0].ContainerID)

    // Verify worker restarts container (autonomous)
    require.Eventually(t, func() bool {
        tasks := worker.GetLocalTasks()
        return tasks[0].ActualState == "running"
    }, 20*time.Second, 1*time.Second)

    // Heal partition
    cluster.HealPartition(worker.ID)

    // Verify reconciliation
    time.Sleep(10 * time.Second)
    clusterState := cluster.GetState()
    require.Equal(t, 3, len(clusterState.Tasks))
}
```

---

## Performance Targets

### Benchmarks

| Metric | Target | Measurement |
|--------|--------|-------------|
| **Binary size** | < 100MB | `ls -lh bin/warren` |
| **Manager memory** | < 256MB | `ps aux \| grep warren` (manager) |
| **Worker memory** | < 128MB | `ps aux \| grep warren` (worker) |
| **API latency (p95)** | < 100ms | `hey -z 60s http://manager:2378/v1/services` |
| **Scheduling latency** | < 5s | Time from `create_service` to `task.ActualState=running` |
| **Leader election** | < 10s | Time from leader kill to new leader ready |
| **Network throughput** | > 90% host speed | `iperf3` across WireGuard overlay |
| **100-node cluster** | Stable | 1 manager, 99 workers, 1000 tasks total |

### Load Testing

**Scenario**: Deploy 1000 services (10 replicas each) on 100-node cluster

```go
func BenchmarkMassiveDeployment(b *testing.B) {
    cluster := startCluster(b, 3, 100)  // 3 managers, 100 workers
    defer cluster.Stop()

    b.ResetTimer()

    for i := 0; i < 1000; i++ {
        cluster.CreateService(&Service{
            Name:     fmt.Sprintf("svc-%d", i),
            Image:    "alpine:latest",
            Replicas: 10,
        })
    }

    // Wait for all tasks running
    waitForTaskCount(b, cluster, 10000, "running")

    b.StopTimer()

    // Measure metrics
    b.ReportMetric(float64(cluster.GetManagerMemory()/1024/1024), "MB/manager")
    b.ReportMetric(float64(cluster.GetAvgWorkerMemory()/1024/1024), "MB/worker")
}
```

---

## Appendix

### Project Structure

```
warren/
├── cmd/
│   └── warren/
│       └── main.go              # CLI entry point
├── pkg/
│   ├── api/
│   │   ├── grpc.go              # gRPC server
│   │   ├── rest.go              # REST gateway
│   │   └── proto/               # Protocol buffers
│   ├── manager/
│   │   ├── manager.go           # Manager orchestration
│   │   ├── raft.go              # Raft integration
│   │   ├── scheduler.go         # Task scheduler
│   │   └── reconciler.go        # State reconciliation
│   ├── worker/
│   │   ├── agent.go             # Worker agent
│   │   ├── runtime.go           # Containerd integration
│   │   └── healthcheck.go       # Health checking
│   ├── network/
│   │   ├── wireguard.go         # WireGuard mesh
│   │   ├── dns.go               # DNS service
│   │   └── loadbalancer.go      # Service VIP/LB
│   ├── security/
│   │   ├── ca.go                # Certificate authority
│   │   ├── mtls.go              # mTLS implementation
│   │   └── secrets.go           # Secrets encryption
│   ├── storage/
│   │   ├── state.go             # Cluster state
│   │   └── boltdb.go            # BoltDB wrapper
│   ├── deploy/
│   │   ├── rolling.go           # Rolling updates
│   │   ├── bluegreen.go         # Blue/green deployment
│   │   └── canary.go            # Canary deployment
│   └── types/
│       └── types.go             # Core data types
├── test/
│   ├── integration/             # Integration tests
│   └── chaos/                   # Chaos/partition tests
├── docs/
│   ├── architecture.md
│   ├── api-reference.md
│   └── user-guide.md
├── Makefile
├── go.mod
└── go.sum
```

### Dependencies

**Core**:
- `github.com/hashicorp/raft` - Raft consensus
- `github.com/hashicorp/raft-boltdb` - BoltDB store for Raft
- `github.com/containerd/containerd` - Container runtime
- `golang.zx2c4.com/wireguard/wgctrl` - WireGuard control
- `github.com/spf13/cobra` - CLI framework
- `google.golang.org/grpc` - gRPC framework
- `github.com/grpc-ecosystem/grpc-gateway` - REST gateway

**Utilities**:
- `github.com/rs/zerolog` - Structured logging
- `github.com/prometheus/client_golang` - Prometheus metrics
- `github.com/vishvananda/netlink` - Linux networking
- `gopkg.in/yaml.v3` - YAML parsing
- `github.com/stretchr/testify` - Testing utilities

### Open Questions

1. **Windows support**: Full Windows container support or WSL2-only initially?
   - **Decision**: WSL2-only for v1.0, native Windows deferred to v2.0

2. **Plugin architecture**: Support custom schedulers/storage drivers?
   - **Decision**: Built-in only for v1.0, plugin SDK in v2.0 roadmap

3. **Federation**: Multi-cluster coordination needed initially?
   - **Decision**: Single cluster only for v1.0, federation explored for v2.0

---

**Document Status**: Approved for implementation
**Next Steps**: Begin Phase 0 (Foundation) - POCs for Raft, containerd, WireGuard
**Implementation Start**: Ready to begin milestone planning and coding

---

*This technical specification provides the blueprint for building Warren. All architectural decisions are documented with implementation details. Refer to [specs/prd.md](prd.md) for product context and user requirements.*
