package types

import (
	"net"
	"time"
)

// Cluster represents the entire Warren cluster
type Cluster struct {
	ID            string
	CreatedAt     time.Time
	Managers      []*Node
	Workers       []*Node
	NetworkConfig *NetworkConfig
}

// Node represents a manager or worker node in the cluster
type Node struct {
	ID            string
	Role          NodeRole
	Address       string    // Host IP address
	OverlayIP     net.IP    // WireGuard overlay IP
	Hostname      string
	Labels        map[string]string
	Resources     *NodeResources
	Status        NodeStatus
	LastHeartbeat time.Time
	CreatedAt     time.Time
}

// NodeRole defines the role of a node
type NodeRole string

const (
	NodeRoleManager NodeRole = "manager"
	NodeRoleWorker  NodeRole = "worker"
)

// NodeStatus represents the current state of a node
type NodeStatus string

const (
	NodeStatusReady    NodeStatus = "ready"
	NodeStatusDown     NodeStatus = "down"
	NodeStatusDraining NodeStatus = "draining"
	NodeStatusUnknown  NodeStatus = "unknown"
)

// NodeResources tracks resource capacity and allocation
type NodeResources struct {
	// Total capacity
	CPUCores    int
	MemoryBytes int64
	DiskBytes   int64

	// Currently allocated (reserved by tasks)
	CPUAllocated    float64
	MemoryAllocated int64
	DiskAllocated   int64
}

// Service represents a user-defined workload
type Service struct {
	ID             string
	Name           string
	Image          string
	Replicas       int
	Mode           ServiceMode
	DeployStrategy DeployStrategy
	UpdateConfig   *UpdateConfig
	Env            []string
	Ports          []*PortMapping
	Networks       []string
	Secrets        []string
	Volumes        []*VolumeMount
	Labels         map[string]string
	HealthCheck    *HealthCheck
	RestartPolicy  *RestartPolicy
	Resources      *ResourceRequirements
	StopTimeout    int // Seconds to wait before force-killing tasks (default: 10)
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// ServiceMode defines how a service is scheduled
type ServiceMode string

const (
	ServiceModeReplicated ServiceMode = "replicated" // N replicas
	ServiceModeGlobal     ServiceMode = "global"     // One per node
)

// DeployStrategy defines how updates are deployed
type DeployStrategy string

const (
	DeployStrategyRolling   DeployStrategy = "rolling"
	DeployStrategyBlueGreen DeployStrategy = "blue-green"
	DeployStrategyCanary    DeployStrategy = "canary"
)

// UpdateConfig controls how service updates are performed
type UpdateConfig struct {
	Parallelism   int           // How many tasks to update simultaneously
	Delay         time.Duration // Delay between batches
	FailureAction string        // "pause", "rollback", "continue"
	CanaryWeight  int           // 0-100 (for canary strategy)
}

// PortMapping defines port exposure
type PortMapping struct {
	Name          string
	ContainerPort int      // Port inside container (target port)
	HostPort      int      // Port on host/cluster (published port)
	Protocol      string   // "tcp" or "udp"
	PublishMode   PublishMode // "host" or "ingress"
}

// PublishMode defines how a port is published
type PublishMode string

const (
	// PublishModeHost publishes port only on the node running the task
	PublishModeHost PublishMode = "host"

	// PublishModeIngress publishes port on all nodes with routing mesh
	PublishModeIngress PublishMode = "ingress"
)

// VolumeMount defines a volume mount point
type VolumeMount struct {
	Source   string // Volume name
	Target   string // Container path
	ReadOnly bool
}

// HealthCheck defines container health checking
type HealthCheck struct {
	Type     HealthCheckType // "http", "tcp", "exec"
	Endpoint string          // URL or address
	Command  []string        // For exec type
	Interval time.Duration
	Timeout  time.Duration
	Retries  int
}

// HealthCheckType defines the type of health check
type HealthCheckType string

const (
	HealthCheckHTTP HealthCheckType = "http"
	HealthCheckTCP  HealthCheckType = "tcp"
	HealthCheckExec HealthCheckType = "exec"
)

// HealthStatus tracks the current health state of a task
type HealthStatus struct {
	Healthy              bool
	Message              string
	CheckedAt            time.Time
	ConsecutiveFailures  int
	ConsecutiveSuccesses int
}

// RestartPolicy defines container restart behavior
type RestartPolicy struct {
	Condition   RestartCondition
	MaxAttempts int
	Delay       time.Duration
}

// RestartCondition defines when to restart
type RestartCondition string

const (
	RestartNever     RestartCondition = "never"
	RestartOnFailure RestartCondition = "on-failure"
	RestartAlways    RestartCondition = "always"
)

// ResourceRequirements defines resource limits and reservations
type ResourceRequirements struct {
	// Limits (maximum allowed)
	CPULimit    float64 // Cores (e.g., 0.5 = 50% of one core)
	MemoryLimit int64   // Bytes

	// Reservations (guaranteed minimum)
	CPUReservation    float64
	MemoryReservation int64
}

// Task represents a single running instance of a service
type Task struct {
	ID            string
	ServiceID     string
	ServiceName   string
	NodeID        string
	ContainerID   string
	DesiredState  TaskState
	ActualState   TaskState
	Image         string
	Env           []string
	Ports         []*PortMapping
	Mounts        []*VolumeMount
	Secrets       []string // Secret names to mount
	HealthCheck   *HealthCheck
	HealthStatus  *HealthStatus // Current health check status
	RestartPolicy *RestartPolicy
	Resources     *ResourceRequirements
	StopTimeout   int // Seconds to wait before force-killing (default: 10)
	CreatedAt     time.Time
	StartedAt     time.Time
	FinishedAt    time.Time
	ExitCode      int
	Error         string
}

// TaskState represents the state of a task
type TaskState string

const (
	TaskStatePending  TaskState = "pending"
	TaskStateRunning  TaskState = "running"
	TaskStateFailed   TaskState = "failed"
	TaskStateComplete TaskState = "complete"
	TaskStateShutdown TaskState = "shutdown"
)

// Secret represents encrypted sensitive data
type Secret struct {
	ID        string
	Name      string
	Data      []byte // Encrypted with AES-256-GCM
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Volume represents persistent storage
type Volume struct {
	ID        string
	Name      string
	Driver    string            // "local", "nfs", etc.
	NodeID    string            // Node affinity (for local volumes)
	MountPath string            // Host mount path
	Options   map[string]string // Driver-specific options
	CreatedAt time.Time
}

// Network represents an overlay network
type Network struct {
	ID      string
	Name    string
	Subnet  string // CIDR (e.g., "10.0.1.0/24")
	Gateway string
	Driver  string // "wireguard"
}

// NetworkConfig represents cluster-wide network configuration
type NetworkConfig struct {
	ClusterSubnet string            // Overall cluster subnet (e.g., "10.0.0.0/16")
	ServiceSubnet string            // Subnet for service VIPs (e.g., "10.0.1.0/24")
	NodeIPs       map[string]net.IP // Node ID -> Overlay IP mapping
}

// Event represents a cluster event (for streaming API)
type Event struct {
	Type      string
	Timestamp time.Time
	NodeID    string
	ServiceID string
	TaskID    string
	Message   string
	Data      map[string]string
}
