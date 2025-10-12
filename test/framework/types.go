package framework

import (
	"context"
	"time"

	"github.com/cuemby/warren/pkg/client"
)

// RuntimeType defines the type of runtime for test clusters
type RuntimeType string

const (
	// RuntimeLima uses Lima VMs for testing
	RuntimeLima RuntimeType = "lima"
	// RuntimeDocker uses Docker containers for testing
	RuntimeDocker RuntimeType = "docker"
	// RuntimeLocal uses local processes for testing
	RuntimeLocal RuntimeType = "local"
)

// ClusterConfig defines the configuration for a test cluster
type ClusterConfig struct {
	// NumManagers is the number of manager nodes to create
	NumManagers int
	// NumWorkers is the number of worker nodes to create
	NumWorkers int
	// Runtime specifies which runtime to use (Lima, Docker, Local)
	Runtime RuntimeType
	// DataDir is the base directory for cluster data
	DataDir string
	// WarrenBinary is the path to the Warren binary
	WarrenBinary string
	// KeepOnFailure keeps the cluster running if tests fail (for debugging)
	KeepOnFailure bool
	// LogLevel sets the logging level for Warren processes
	LogLevel string
}

// Cluster represents a test Warren cluster
type Cluster struct {
	// Config is the cluster configuration
	Config *ClusterConfig
	// Managers contains all manager nodes in the cluster
	Managers []*Manager
	// Workers contains all worker nodes in the cluster
	Workers []*Worker
	// ctx is the context for cluster operations
	ctx context.Context
	// cancel cancels the cluster context
	cancel context.CancelFunc
}

// Manager represents a manager node in the test cluster
type Manager struct {
	// ID is the unique identifier for this manager
	ID string
	// VM is the virtual machine or container running this manager
	VM VM
	// APIAddr is the gRPC API address (host:port)
	APIAddr string
	// RaftAddr is the Raft consensus address (host:port)
	RaftAddr string
	// Client is the Warren client connected to this manager
	Client *client.Client
	// Process is the Warren process (if running locally)
	Process *Process
	// DataDir is the data directory for this manager
	DataDir string
	// IsLeader indicates if this manager is the Raft leader
	IsLeader bool
}

// Worker represents a worker node in the test cluster
type Worker struct {
	// ID is the unique identifier for this worker
	ID string
	// VM is the virtual machine or container running this worker
	VM VM
	// ManagerAddr is the address of the manager this worker connects to
	ManagerAddr string
	// Process is the Warren process (if running locally)
	Process *Process
	// DataDir is the data directory for this worker
	DataDir string
}

// Process is defined in process.go (to avoid duplication)

// VM represents a virtual machine or container for testing
type VM interface {
	// ID returns the unique identifier for this VM
	ID() string
	// Start starts the VM
	Start(ctx context.Context) error
	// Stop stops the VM gracefully
	Stop(ctx context.Context) error
	// Kill forcefully terminates the VM
	Kill(ctx context.Context) error
	// IsRunning returns true if the VM is currently running
	IsRunning() bool
	// Exec executes a command in the VM and returns the output
	Exec(ctx context.Context, command string, args ...string) (string, error)
	// CopyFile copies a file from the host to the VM
	CopyFile(ctx context.Context, src, dst string) error
	// GetIP returns the IP address of the VM
	GetIP() (string, error)
	// WaitForBoot waits for the VM to finish booting
	WaitForBoot(ctx context.Context) error
}

// TestContext provides utilities for test execution
type TestContext struct {
	// T is the testing.T instance
	T TestingT
	// Ctx is the context for test operations
	Ctx context.Context
	// Cancel cancels the test context
	Cancel context.CancelFunc
	// Timeout is the default timeout for operations
	Timeout time.Duration
	// Cleanup functions to run after test
	cleanup []func()
}

// TestingT is an interface matching testing.T
type TestingT interface {
	Logf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
	FailNow()
	Failed() bool
	Name() string
	Helper()
}

// ServiceSpec defines a service for testing
type ServiceSpec struct {
	Name     string
	Image    string
	Replicas int
	Env      map[string]string
	Ports    []ServicePort
}

// ServicePort defines a port mapping
type ServicePort struct {
	ContainerPort int
	Protocol      string
}

// IngressSpec defines an ingress rule for testing
type IngressSpec struct {
	Host     string
	Path     string
	PathType string // "Exact" or "Prefix"
	Backend  IngressBackend
	TLS      *IngressTLS
}

// IngressBackend defines the backend service for an ingress rule
type IngressBackend struct {
	Service string
	Port    int
}

// IngressTLS defines TLS configuration for an ingress
type IngressTLS struct {
	Enabled    bool
	SecretName string
}
