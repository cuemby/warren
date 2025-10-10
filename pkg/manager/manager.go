package manager

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/cuemby/warren/pkg/storage"
	"github.com/cuemby/warren/pkg/types"
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
)

// Manager represents a Warren cluster manager node
type Manager struct {
	nodeID   string
	bindAddr string
	dataDir  string

	raft  *raft.Raft
	fsm   *WarrenFSM
	store storage.Store
}

// Config holds configuration for creating a Manager
type Config struct {
	NodeID   string
	BindAddr string
	DataDir  string
}

// NewManager creates a new Manager instance
func NewManager(cfg *Config) (*Manager, error) {
	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %v", err)
	}

	// Create BoltDB store
	store, err := storage.NewBoltStore(cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create store: %v", err)
	}

	// Create FSM
	fsm := NewWarrenFSM(store)

	m := &Manager{
		nodeID:   cfg.NodeID,
		bindAddr: cfg.BindAddr,
		dataDir:  cfg.DataDir,
		fsm:      fsm,
		store:    store,
	}

	return m, nil
}

// Bootstrap initializes a new single-node Raft cluster
func (m *Manager) Bootstrap() error {
	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(m.nodeID)

	// Setup Raft communication
	addr, err := net.ResolveTCPAddr("tcp", m.bindAddr)
	if err != nil {
		return fmt.Errorf("failed to resolve bind address: %v", err)
	}

	transport, err := raft.NewTCPTransport(m.bindAddr, addr, 3, 10*time.Second, os.Stderr)
	if err != nil {
		return fmt.Errorf("failed to create transport: %v", err)
	}

	// Create snapshot store
	snapshotStore, err := raft.NewFileSnapshotStore(m.dataDir, 2, os.Stderr)
	if err != nil {
		return fmt.Errorf("failed to create snapshot store: %v", err)
	}

	// Create log store and stable store using BoltDB
	logStorePath := filepath.Join(m.dataDir, "raft-log.db")
	logStore, err := raftboltdb.NewBoltStore(logStorePath)
	if err != nil {
		return fmt.Errorf("failed to create log store: %v", err)
	}

	stableStorePath := filepath.Join(m.dataDir, "raft-stable.db")
	stableStore, err := raftboltdb.NewBoltStore(stableStorePath)
	if err != nil {
		return fmt.Errorf("failed to create stable store: %v", err)
	}

	// Create Raft instance
	r, err := raft.NewRaft(config, m.fsm, logStore, stableStore, snapshotStore, transport)
	if err != nil {
		return fmt.Errorf("failed to create raft: %v", err)
	}

	m.raft = r

	// Bootstrap cluster with this node as the only member
	configuration := raft.Configuration{
		Servers: []raft.Server{
			{
				ID:      config.LocalID,
				Address: transport.LocalAddr(),
			},
		},
	}

	future := m.raft.BootstrapCluster(configuration)
	if err := future.Error(); err != nil {
		return fmt.Errorf("failed to bootstrap cluster: %v", err)
	}

	return nil
}

// Join adds this manager to an existing cluster
func (m *Manager) Join(leaderAddr string) error {
	// TODO: Implement cluster join logic
	// This will be used when adding additional manager nodes
	return fmt.Errorf("join not yet implemented")
}

// IsLeader returns true if this manager is the Raft leader
func (m *Manager) IsLeader() bool {
	if m.raft == nil {
		return false
	}
	return m.raft.State() == raft.Leader
}

// LeaderAddr returns the address of the current Raft leader
func (m *Manager) LeaderAddr() string {
	if m.raft == nil {
		return ""
	}
	return string(m.raft.Leader())
}

// Apply submits a command to the Raft cluster
func (m *Manager) Apply(cmd Command) error {
	if m.raft == nil {
		return fmt.Errorf("raft not initialized")
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("failed to marshal command: %v", err)
	}

	future := m.raft.Apply(data, 5*time.Second)
	if err := future.Error(); err != nil {
		return fmt.Errorf("failed to apply command: %v", err)
	}

	// Check if apply returned an error
	if resp := future.Response(); resp != nil {
		if err, ok := resp.(error); ok && err != nil {
			return err
		}
	}

	return nil
}

// CreateNode adds a node to the cluster
func (m *Manager) CreateNode(node *types.Node) error {
	data, err := json.Marshal(node)
	if err != nil {
		return err
	}

	cmd := Command{
		Op:   "create_node",
		Data: data,
	}

	return m.Apply(cmd)
}

// UpdateNode updates a node in the cluster
func (m *Manager) UpdateNode(node *types.Node) error {
	data, err := json.Marshal(node)
	if err != nil {
		return err
	}

	cmd := Command{
		Op:   "update_node",
		Data: data,
	}

	return m.Apply(cmd)
}

// DeleteNode removes a node from the cluster
func (m *Manager) DeleteNode(id string) error {
	data, err := json.Marshal(id)
	if err != nil {
		return err
	}

	cmd := Command{
		Op:   "delete_node",
		Data: data,
	}

	return m.Apply(cmd)
}

// CreateService creates a new service
func (m *Manager) CreateService(service *types.Service) error {
	data, err := json.Marshal(service)
	if err != nil {
		return err
	}

	cmd := Command{
		Op:   "create_service",
		Data: data,
	}

	return m.Apply(cmd)
}

// UpdateService updates an existing service
func (m *Manager) UpdateService(service *types.Service) error {
	data, err := json.Marshal(service)
	if err != nil {
		return err
	}

	cmd := Command{
		Op:   "update_service",
		Data: data,
	}

	return m.Apply(cmd)
}

// DeleteService removes a service
func (m *Manager) DeleteService(id string) error {
	data, err := json.Marshal(id)
	if err != nil {
		return err
	}

	cmd := Command{
		Op:   "delete_service",
		Data: data,
	}

	return m.Apply(cmd)
}

// CreateTask creates a new task
func (m *Manager) CreateTask(task *types.Task) error {
	data, err := json.Marshal(task)
	if err != nil {
		return err
	}

	cmd := Command{
		Op:   "create_task",
		Data: data,
	}

	return m.Apply(cmd)
}

// UpdateTask updates a task
func (m *Manager) UpdateTask(task *types.Task) error {
	data, err := json.Marshal(task)
	if err != nil {
		return err
	}

	cmd := Command{
		Op:   "update_task",
		Data: data,
	}

	return m.Apply(cmd)
}

// DeleteTask removes a task
func (m *Manager) DeleteTask(id string) error {
	data, err := json.Marshal(id)
	if err != nil {
		return err
	}

	cmd := Command{
		Op:   "delete_task",
		Data: data,
	}

	return m.Apply(cmd)
}

// CreateSecret creates a new secret
func (m *Manager) CreateSecret(secret *types.Secret) error {
	data, err := json.Marshal(secret)
	if err != nil {
		return err
	}

	cmd := Command{
		Op:   "create_secret",
		Data: data,
	}

	return m.Apply(cmd)
}

// DeleteSecret removes a secret
func (m *Manager) DeleteSecret(id string) error {
	data, err := json.Marshal(id)
	if err != nil {
		return err
	}

	cmd := Command{
		Op:   "delete_secret",
		Data: data,
	}

	return m.Apply(cmd)
}

// CreateVolume creates a new volume
func (m *Manager) CreateVolume(volume *types.Volume) error {
	data, err := json.Marshal(volume)
	if err != nil {
		return err
	}

	cmd := Command{
		Op:   "create_volume",
		Data: data,
	}

	return m.Apply(cmd)
}

// DeleteVolume removes a volume
func (m *Manager) DeleteVolume(id string) error {
	data, err := json.Marshal(id)
	if err != nil {
		return err
	}

	cmd := Command{
		Op:   "delete_volume",
		Data: data,
	}

	return m.Apply(cmd)
}

// GetNode retrieves a node by ID (read from local store)
func (m *Manager) GetNode(id string) (*types.Node, error) {
	return m.store.GetNode(id)
}

// ListNodes returns all nodes (read from local store)
func (m *Manager) ListNodes() ([]*types.Node, error) {
	return m.store.ListNodes()
}

// GetService retrieves a service by ID (read from local store)
func (m *Manager) GetService(id string) (*types.Service, error) {
	return m.store.GetService(id)
}

// GetServiceByName retrieves a service by name (read from local store)
func (m *Manager) GetServiceByName(name string) (*types.Service, error) {
	return m.store.GetServiceByName(name)
}

// ListServices returns all services (read from local store)
func (m *Manager) ListServices() ([]*types.Service, error) {
	return m.store.ListServices()
}

// GetTask retrieves a task by ID (read from local store)
func (m *Manager) GetTask(id string) (*types.Task, error) {
	return m.store.GetTask(id)
}

// ListTasks returns all tasks (read from local store)
func (m *Manager) ListTasks() ([]*types.Task, error) {
	return m.store.ListTasks()
}

// ListTasksByService returns all tasks for a service (read from local store)
func (m *Manager) ListTasksByService(serviceID string) ([]*types.Task, error) {
	return m.store.ListTasksByService(serviceID)
}

// ListTasksByNode returns all tasks on a node (read from local store)
func (m *Manager) ListTasksByNode(nodeID string) ([]*types.Task, error) {
	return m.store.ListTasksByNode(nodeID)
}

// GetSecret retrieves a secret by ID (read from local store)
func (m *Manager) GetSecret(id string) (*types.Secret, error) {
	return m.store.GetSecret(id)
}

// GetSecretByName retrieves a secret by name (read from local store)
func (m *Manager) GetSecretByName(name string) (*types.Secret, error) {
	return m.store.GetSecretByName(name)
}

// ListSecrets returns all secrets (read from local store)
func (m *Manager) ListSecrets() ([]*types.Secret, error) {
	return m.store.ListSecrets()
}

// GetVolume retrieves a volume by ID (read from local store)
func (m *Manager) GetVolume(id string) (*types.Volume, error) {
	return m.store.GetVolume(id)
}

// GetVolumeByName retrieves a volume by name (read from local store)
func (m *Manager) GetVolumeByName(name string) (*types.Volume, error) {
	return m.store.GetVolumeByName(name)
}

// ListVolumes returns all volumes (read from local store)
func (m *Manager) ListVolumes() ([]*types.Volume, error) {
	return m.store.ListVolumes()
}

// GetNetwork retrieves a network by ID (read from local store)
func (m *Manager) GetNetwork(id string) (*types.Network, error) {
	return m.store.GetNetwork(id)
}

// ListNetworks returns all networks (read from local store)
func (m *Manager) ListNetworks() ([]*types.Network, error) {
	return m.store.ListNetworks()
}

// Shutdown gracefully shuts down the manager
func (m *Manager) Shutdown() error {
	if m.raft != nil {
		future := m.raft.Shutdown()
		if err := future.Error(); err != nil {
			return fmt.Errorf("failed to shutdown raft: %v", err)
		}
	}

	if m.store != nil {
		if err := m.store.Close(); err != nil {
			return fmt.Errorf("failed to close store: %v", err)
		}
	}

	return nil
}
