package types

import (
	"net"
	"time"
)

// ServiceManager defines the interface for service management operations
// This interface is used to break the import cycle between deploy and manager packages
type ServiceManager interface {
	GetService(id string) (*Service, error)
	ListServices() ([]*Service, error)
	CreateService(service *Service) error
	UpdateService(service *Service) error
	DeleteService(id string) error
	ListContainersByService(serviceID string) ([]*Container, error)
	UpdateContainer(container *Container) error
}

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
	Address       string // Host IP address
	OverlayIP     net.IP // WireGuard overlay IP
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
	NodeRoleManager NodeRole = "manager" // Manager-only (control plane)
	NodeRoleWorker  NodeRole = "worker"  // Worker-only (workloads)
	NodeRoleHybrid  NodeRole = "hybrid"  // Manager + Worker combined
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

	// Currently allocated (reserved by containers)
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
	StopTimeout    int // Seconds to wait before force-killing containers (default: 10)
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

// DeploymentState represents the state of a deployment version
type DeploymentState string

const (
	DeploymentStateActive     DeploymentState = "active"      // Currently serving traffic
	DeploymentStateStandby    DeploymentState = "standby"     // Ready but not serving (blue in blue-green)
	DeploymentStateCanary     DeploymentState = "canary"      // Canary version receiving partial traffic
	DeploymentStateRolling    DeploymentState = "rolling"     // Rolling update in progress
	DeploymentStateFailed     DeploymentState = "failed"      // Deployment failed
	DeploymentStateRolledBack DeploymentState = "rolled-back" // Deployment was rolled back
)

// Deployment version label keys
const (
	LabelDeploymentVersion  = "warren.deployment.version"
	LabelDeploymentState    = "warren.deployment.state"
	LabelDeploymentStrategy = "warren.deployment.strategy"
	LabelOriginalService    = "warren.deployment.original-service"
)

// UpdateConfig controls how service updates are performed
type UpdateConfig struct {
	Parallelism             int           // How many containers to update simultaneously
	Delay                   time.Duration // Delay between batches
	FailureAction           string        // "pause", "rollback", "continue"
	MaxSurge                int           // Max extra containers during update (default: 1)
	MaxUnavailable          int           // Max containers that can be unavailable (default: 0)
	HealthCheckGracePeriod  time.Duration // Wait time for health checks (default: 30s)
	CanaryWeight            int           // 0-100 (current canary traffic weight)
	CanarySteps             []int         // Promotion steps, e.g., [10, 25, 50, 100]
	CanaryStabilityWindow   time.Duration // Wait time between canary steps (default: 5m)
	BlueGreenGracePeriod    time.Duration // Time to keep blue version after switch (default: 5m)
	AutoRollbackEnabled     bool          // Enable automatic rollback on failures
	FailureThresholdPercent int           // Error rate % to trigger rollback (default: 10)
}

// PortMapping defines port exposure
type PortMapping struct {
	Name          string
	ContainerPort int         // Port inside container (target port)
	HostPort      int         // Port on host/cluster (published port)
	Protocol      string      // "tcp" or "udp"
	PublishMode   PublishMode // "host" or "ingress"
}

// PublishMode defines how a port is published
type PublishMode string

const (
	// PublishModeHost publishes port only on the node running the container
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

// HealthStatus tracks the current health state of a container
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

// Container represents a single running container instance of a service
type Container struct {
	ID            string
	ServiceID     string
	ServiceName   string
	NodeID        string
	ContainerID   string // Runtime container ID (containerd)
	DesiredState  ContainerState
	ActualState   ContainerState
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

// ContainerState represents the state of a container
type ContainerState string

const (
	ContainerStatePending  ContainerState = "pending"
	ContainerStateRunning  ContainerState = "running"
	ContainerStateFailed   ContainerState = "failed"
	ContainerStateComplete ContainerState = "complete"
	ContainerStateShutdown ContainerState = "shutdown"
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
	Type        string
	Timestamp   time.Time
	NodeID      string
	ServiceID   string
	ContainerID string
	Message     string
	Data        map[string]string
}

// Ingress represents HTTP/HTTPS routing rules for external access
type Ingress struct {
	ID        string
	Name      string
	Rules     []*IngressRule
	TLS       *IngressTLS
	Labels    map[string]string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// IngressRule defines routing rules for an ingress
type IngressRule struct {
	Host  string         // Hostname to match (e.g., "api.example.com", "*.example.com")
	Paths []*IngressPath // Path-based routing rules
}

// IngressPath defines a path-based routing rule
type IngressPath struct {
	Path          string              // Path to match (e.g., "/api", "/web")
	PathType      PathType            // "Prefix" or "Exact"
	Backend       *IngressBackend     // Backend service to route to
	Rewrite       *PathRewrite        // Path rewriting configuration (M7.3)
	Headers       *HeaderManipulation // Header manipulation (M7.3)
	RateLimit     *RateLimit          // Rate limiting configuration (M7.3)
	AccessControl *AccessControl      // Access control rules (M7.3)
}

// PathType defines how paths are matched
type PathType string

const (
	PathTypePrefix PathType = "Prefix" // Prefix match (e.g., /api matches /api/*)
	PathTypeExact  PathType = "Exact"  // Exact match only
)

// IngressBackend defines the backend service for routing
type IngressBackend struct {
	ServiceName string // Service to route to
	Port        int    // Service port to connect to
}

// IngressTLS defines TLS configuration for HTTPS
type IngressTLS struct {
	Enabled    bool     // Enable HTTPS
	SecretName string   // Secret containing TLS cert/key (PEM format)
	Hosts      []string // Hosts covered by this TLS config
	AutoTLS    bool     // Enable Let's Encrypt automatic certificates (M7.3)
	Email      string   // Email for Let's Encrypt notifications (M7.3)
}

// PathRewrite defines path rewriting rules
type PathRewrite struct {
	StripPrefix string // Strip this prefix from the path (e.g., "/api/v1" → "/")
	ReplacePath string // Replace entire path with this (takes precedence over StripPrefix)
}

// HeaderManipulation defines header modification rules
type HeaderManipulation struct {
	Add    map[string]string // Headers to add (e.g., {"X-Custom": "value"})
	Set    map[string]string // Headers to set/overwrite (e.g., {"X-Frame-Options": "DENY"})
	Remove []string          // Headers to remove (e.g., ["X-Powered-By"])
}

// RateLimit defines rate limiting configuration
type RateLimit struct {
	RequestsPerSecond float64 // Requests allowed per second
	Burst             int     // Burst capacity (token bucket size)
}

// AccessControl defines IP-based access control
type AccessControl struct {
	AllowedIPs []string // IP whitelist (CIDR notation, e.g., "192.168.1.0/24")
	DeniedIPs  []string // IP blacklist (CIDR notation)
}

// TLSCertificate represents a TLS certificate for ingress
type TLSCertificate struct {
	ID        string            // Unique identifier
	Name      string            // Certificate name (e.g., "example-com-cert")
	Hosts     []string          // Hostnames covered by this cert (e.g., ["example.com", "*.example.com"])
	CertPEM   []byte            // Certificate in PEM format
	KeyPEM    []byte            // Private key in PEM format (encrypted in storage)
	Issuer    string            // Certificate issuer (e.g., "Let's Encrypt", "self-signed", "manual")
	NotBefore time.Time         // Certificate valid from
	NotAfter  time.Time         // Certificate valid until
	AutoRenew bool              // Enable automatic renewal (M7.3)
	Labels    map[string]string // Labels for organization
	CreatedAt time.Time
	UpdatedAt time.Time
}
