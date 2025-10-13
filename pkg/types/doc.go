/*
Package types defines the core data structures used throughout Warren.

This package contains all fundamental types that represent Warren's domain model,
including clusters, nodes, services, tasks, secrets, volumes, and ingress resources.
These types are used by all other packages for state management, API communication,
and orchestration logic.

# Architecture

The types package is the foundation of Warren's data model. It defines:

  - Cluster topology (managers, workers)
  - Service specifications and deployment strategies
  - Task execution state and lifecycle
  - Resource management (CPU, memory, disk)
  - Security primitives (secrets, certificates)
  - Storage primitives (volumes, mounts)
  - Networking primitives (ports, ingress, DNS)
  - Health check configurations

All types are designed to be:
  - Serializable (JSON, Protocol Buffers)
  - Immutable where possible (use pointers for updates)
  - Self-documenting (clear field names and comments)
  - Validated (constants for enums, validation helpers)

# Core Types

The main types in this package are:

Cluster Topology:
  - Cluster: Represents the entire Warren cluster
  - Node: Manager or worker node with resources and status
  - NodeRole: Manager or worker role
  - NodeStatus: Ready, down, draining, unknown

Service Management:
  - Service: User-defined containerized workload
  - ServiceMode: Replicated (N replicas) or global (one per node)
  - DeployStrategy: Rolling, blue-green, or canary updates
  - UpdateConfig: Update parallelism, delay, failure actions

Task Execution:
  - Task: Single instance of a service (container)
  - TaskState: Pending, assigned, running, failed, etc.
  - TaskStatus: Current state with timestamps and error details

Security:
  - Secret: Encrypted data (passwords, certificates, etc.)
  - SecretType: Opaque, TLS, certificate authority
  - Certificate: TLS certificate with domain and auto-renewal

Storage:
  - Volume: Persistent storage volume
  - VolumeDriver: Local, NFS, etc.
  - VolumeMount: Mount point in a container

Networking:
  - PortMapping: Container port to host/cluster port mapping
  - PublishMode: Host (node-local) or ingress (cluster-wide)
  - Ingress: HTTP/HTTPS routing rules with TLS
  - IngressRule: Host-based and path-based routing

Health & Resources:
  - HealthCheck: HTTP, TCP, or exec health probes
  - ResourceRequirements: CPU and memory limits/reservations
  - RestartPolicy: Always, on-failure, never

# Usage

Creating a Service:

	service := &types.Service{
		ID:       uuid.New().String(),
		Name:     "nginx",
		Image:    "nginx:latest",
		Replicas: 3,
		Mode:     types.ServiceModeReplicated,
		Ports: []*types.PortMapping{
			{
				ContainerPort: 80,
				HostPort:      8080,
				Protocol:      "tcp",
				PublishMode:   types.PublishModeHost,
			},
		},
		HealthCheck: &types.HealthCheck{
			Type:     types.HealthCheckTypeHTTP,
			Endpoint: "http://localhost:80/",
			Interval: 30 * time.Second,
			Timeout:  5 * time.Second,
			Retries:  3,
		},
		Resources: &types.ResourceRequirements{
			Limits: &types.ResourceSpec{
				CPUShares:   1024,
				MemoryBytes: 512 * 1024 * 1024, // 512MB
			},
		},
	}

Creating a Task from a Service:

	task := &types.Task{
		ID:          uuid.New().String(),
		ServiceID:   service.ID,
		ServiceName: service.Name,
		NodeID:      "",  // Assigned by scheduler
		Image:       service.Image,
		Env:         service.Env,
		Ports:       service.Ports,
		State:       types.TaskStatePending,
		DesiredState: types.TaskStateRunning,
		CreatedAt:   time.Now(),
	}

Creating a Secret:

	secret := &types.Secret{
		ID:        uuid.New().String(),
		Name:      "db-password",
		Type:      types.SecretTypeOpaque,
		Data:      []byte("super-secret-password"),
		CreatedAt: time.Now(),
	}

Creating an Ingress:

	ingress := &types.Ingress{
		ID:   uuid.New().String(),
		Name: "my-ingress",
		Rules: []*types.IngressRule{
			{
				Host: "api.example.com",
				Paths: []*types.IngressPath{
					{
						Path:        "/",
						PathType:    types.PathTypePrefix,
						ServiceName: "api-service",
						ServicePort: 80,
					},
				},
			},
		},
		TLS: &types.IngressTLS{
			Domains:    []string{"api.example.com"},
			AutoIssue:  true,
			ACMEEmail:  "admin@example.com",
		},
	}

# State Machine

Tasks follow a state machine:

	Pending → Assigned → Preparing → Starting → Running → Shutdown → Complete
	           ↓           ↓           ↓          ↓
	         Failed      Failed      Failed    Failed

Valid state transitions:
  - Pending → Assigned (scheduler assigns to node)
  - Assigned → Preparing (worker accepts task)
  - Preparing → Starting (secrets/volumes mounted)
  - Starting → Running (container started)
  - Running → Shutdown (graceful stop requested)
  - Shutdown → Complete (container stopped cleanly)
  - Any → Failed (error occurred)
  - Failed → Pending (retry via reconciler)

# Design Patterns

Enumeration Pattern:

	All enums use typed string constants for safety and clarity:
	  type TaskState string
	  const (
	      TaskStatePending  TaskState = "pending"
	      TaskStateRunning  TaskState = "running"
	  )

Resource Pattern:

	Resources follow a limit/reservation pattern:
	  - Limits: Hard caps (task killed if exceeded)
	  - Reservations: Guaranteed allocation (used for scheduling)

Optional Fields:

	Optional configurations use pointers:
	  - *HealthCheck: nil = no health checks
	  - *UpdateConfig: nil = use defaults
	  - *IngressTLS: nil = HTTP only

# Integration Points

This package integrates with:

  - pkg/storage: Persists all types to BoltDB
  - pkg/api: Converts to/from Protocol Buffer messages
  - pkg/manager: Orchestrates service and task lifecycle
  - pkg/scheduler: Uses resources for placement decisions
  - pkg/reconciler: Monitors task state transitions
  - pkg/worker: Executes tasks based on specifications
  - pkg/security: Encrypts secrets and manages certificates
  - pkg/volume: Mounts volumes according to specifications
  - pkg/health: Performs health checks per configuration
  - pkg/ingress: Routes traffic based on ingress rules

# Validation

Key validation rules:

Services:
  - Name must be unique within cluster
  - Replicas must be > 0 for replicated mode
  - Image must be a valid container image reference
  - Port mappings must have unique HostPort per node

Tasks:
  - Must have valid ServiceID reference
  - NodeID required when state >= Assigned
  - ContainerID required when state >= Running

Secrets:
  - Name must be unique within cluster
  - Data must be non-empty
  - Type must be valid SecretType

Volumes:
  - Name must be unique within cluster
  - Driver must be supported (local, etc.)

Ingress:
  - Rules must have unique Host + Path combinations
  - ServiceName must reference existing service
  - TLS domains must match rule hosts

# Thread Safety

All types in this package are designed to be:
  - Read-safe: Can be read concurrently from multiple goroutines
  - Write-unsafe: Mutations must be synchronized by callers
  - Immutable-preferred: Use new instances for updates where possible

The storage layer (pkg/storage) handles all synchronization for
persisted state. In-memory caches must implement their own locking.

# Performance Considerations

Memory Layout:
  - Small types (Node, Task) use value receivers where possible
  - Large types (Service, Ingress) use pointer receivers
  - Slices and maps are nil-checked before access

Serialization:
  - All types are JSON-serializable for storage
  - Protocol Buffer conversions in pkg/api for network efficiency
  - BoltDB stores types as JSON (human-readable, debuggable)

# See Also

  - pkg/storage for persistence layer
  - pkg/api for Protocol Buffer definitions
  - pkg/manager for orchestration logic
  - specs/tech.md for data model design rationale
  - .agent/System/database-schema.md for storage schema
*/
package types
