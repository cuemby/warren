package manager

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/cuemby/warren/pkg/client"
	"github.com/cuemby/warren/pkg/dns"
	"github.com/cuemby/warren/pkg/events"
	"github.com/cuemby/warren/pkg/security"
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

	raft           *raft.Raft
	fsm            *WarrenFSM
	store          storage.Store
	tokenManager   *TokenManager
	secretsManager *security.SecretsManager
	ca             *security.CertAuthority
	eventBroker    *events.Broker
	dnsServer      *dns.Server
	dnsCtx         context.Context
	dnsCancel      context.CancelFunc
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

	// Create token manager
	tokenManager := NewTokenManager()

	// Create secrets manager with cluster-derived key
	clusterKey := security.DeriveKeyFromClusterID(cfg.NodeID) // Using node ID as cluster ID for now
	secretsManager, err := security.NewSecretsManager(clusterKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create secrets manager: %v", err)
	}

	// Set cluster encryption key for CA encryption
	if err := security.SetClusterEncryptionKey(clusterKey); err != nil {
		return nil, fmt.Errorf("failed to set cluster encryption key: %v", err)
	}

	// Create Certificate Authority
	ca := security.NewCertAuthority(store)

	// Create event broker
	eventBroker := events.NewBroker()
	eventBroker.Start()

	// Create DNS server
	dnsServer := dns.NewServer(store, nil) // Use default config
	dnsCtx, dnsCancel := context.WithCancel(context.Background())

	m := &Manager{
		nodeID:         cfg.NodeID,
		bindAddr:       cfg.BindAddr,
		dataDir:        cfg.DataDir,
		fsm:            fsm,
		store:          store,
		secretsManager: secretsManager,
		ca:             ca,
		tokenManager:   tokenManager,
		eventBroker:    eventBroker,
		dnsServer:      dnsServer,
		dnsCtx:         dnsCtx,
		dnsCancel:      dnsCancel,
	}

	return m, nil
}

// Bootstrap initializes a new single-node Raft cluster
func (m *Manager) Bootstrap() error {
	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(m.nodeID)

	// Tune Raft timeouts for faster failover (target: <10s)
	// Hashicorp Raft defaults are conservative for WAN deployments
	// We're optimizing for edge/LAN with lower latency expectations
	//
	// Defaults: HeartbeatTimeout=1s, ElectionTimeout=1s, LeaderLeaseTimeout=500ms
	// For <10s failover, we need faster detection and election
	config.HeartbeatTimeout = 500 * time.Millisecond   // Reduced from 1s - faster failure detection
	config.ElectionTimeout = 500 * time.Millisecond    // Reduced from 1s - faster elections
	config.CommitTimeout = 50 * time.Millisecond        // Keep default - not critical for failover
	config.LeaderLeaseTimeout = 250 * time.Millisecond  // Reduced from 500ms - faster lease timeout

	// These settings mean:
	// - Leader sends heartbeats every ~250ms (HeartbeatTimeout/2)
	// - Followers wait 500ms without heartbeat before election
	// - Election completes in ~500ms-1s
	// - Total failover time: ~2-3s (well under 10s target)

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

	// Initialize Certificate Authority
	if err := m.initializeCA(); err != nil {
		return fmt.Errorf("failed to initialize CA: %v", err)
	}

	// Start DNS server
	go func() {
		if err := m.dnsServer.Start(m.dnsCtx); err != nil {
			fmt.Printf("Failed to start DNS server: %v\n", err)
		}
	}()

	// Give DNS server time to start
	time.Sleep(100 * time.Millisecond)

	return nil
}

// Join adds this manager to an existing cluster
func (m *Manager) Join(leaderAddr string, token string) error {
	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(m.nodeID)

	// Tune Raft timeouts for faster failover (target: <10s)
	// Same configuration as Bootstrap for consistency across cluster
	config.HeartbeatTimeout = 500 * time.Millisecond   // Reduced from 1s - faster failure detection
	config.ElectionTimeout = 500 * time.Millisecond    // Reduced from 1s - faster elections
	config.CommitTimeout = 50 * time.Millisecond        // Keep default - not critical for failover
	config.LeaderLeaseTimeout = 250 * time.Millisecond  // Reduced from 500ms - faster lease timeout

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

	// Contact the leader to add this node to the cluster via RPC
	fmt.Printf("Contacting leader at %s to join cluster...\n", leaderAddr)
	fmt.Printf("Node ID: %s, Bind Addr: %s, Token: %s\n", m.nodeID, m.bindAddr, token)

	// Create client to connect to leader
	c, err := client.NewClient(leaderAddr)
	if err != nil {
		return fmt.Errorf("failed to connect to leader: %v", err)
	}
	defer c.Close()

	// Send JoinCluster RPC to leader
	if err := c.JoinCluster(m.nodeID, m.bindAddr, token); err != nil {
		return fmt.Errorf("failed to join cluster via RPC: %v", err)
	}

	fmt.Println("✓ Successfully joined cluster")

	// Load Certificate Authority from storage (already initialized by bootstrap node)
	if err := m.ca.LoadFromStore(); err != nil {
		return fmt.Errorf("failed to load CA: %v", err)
	}
	fmt.Println("✓ Loaded Certificate Authority from cluster")

	// Start DNS server
	go func() {
		if err := m.dnsServer.Start(m.dnsCtx); err != nil {
			fmt.Printf("Failed to start DNS server: %v\n", err)
		}
	}()

	// Give DNS server time to start
	time.Sleep(100 * time.Millisecond)

	return nil
}

// AddVoter adds a new manager node to the Raft cluster
func (m *Manager) AddVoter(nodeID, address string) error {
	if m.raft == nil {
		return fmt.Errorf("raft not initialized")
	}

	if !m.IsLeader() {
		return fmt.Errorf("not the leader, current leader: %s", m.LeaderAddr())
	}

	fmt.Printf("Adding voter: ID=%s, Address=%s\n", nodeID, address)

	future := m.raft.AddVoter(raft.ServerID(nodeID), raft.ServerAddress(address), 0, 10*time.Second)
	if err := future.Error(); err != nil {
		return fmt.Errorf("failed to add voter: %v", err)
	}

	fmt.Printf("Successfully added voter %s to cluster\n", nodeID)
	return nil
}

// RemoveServer removes a server from the Raft cluster
func (m *Manager) RemoveServer(nodeID string) error {
	if m.raft == nil {
		return fmt.Errorf("raft not initialized")
	}

	if !m.IsLeader() {
		return fmt.Errorf("not the leader")
	}

	future := m.raft.RemoveServer(raft.ServerID(nodeID), 0, 10*time.Second)
	if err := future.Error(); err != nil {
		return fmt.Errorf("failed to remove server: %v", err)
	}

	return nil
}

// GetClusterServers returns information about all servers in the Raft cluster
func (m *Manager) GetClusterServers() ([]raft.Server, error) {
	if m.raft == nil {
		return nil, fmt.Errorf("raft not initialized")
	}

	future := m.raft.GetConfiguration()
	if err := future.Error(); err != nil {
		return nil, fmt.Errorf("failed to get configuration: %v", err)
	}

	return future.Configuration().Servers, nil
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

// GetRaftStats returns Raft statistics
func (m *Manager) GetRaftStats() map[string]interface{} {
	if m.raft == nil {
		return nil
	}

	stats := make(map[string]interface{})
	stats["state"] = m.raft.State().String()
	stats["last_log_index"] = m.raft.LastIndex()
	stats["applied_index"] = m.raft.AppliedIndex()
	stats["leader"] = string(m.raft.Leader())

	return stats
}

// GetEventBroker returns the event broker
func (m *Manager) GetEventBroker() *events.Broker {
	return m.eventBroker
}

// PublishEvent publishes an event to all subscribers
func (m *Manager) PublishEvent(event *events.Event) {
	if m.eventBroker != nil {
		m.eventBroker.Publish(event)
	}
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

// EncryptSecret encrypts plaintext secret data
func (m *Manager) EncryptSecret(plaintext []byte) ([]byte, error) {
	return m.secretsManager.EncryptSecret(plaintext)
}

// CreateSecret creates a new secret (data should already be encrypted)
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

// GenerateJoinToken generates a new join token for adding nodes
func (m *Manager) GenerateJoinToken(role string) (*JoinToken, error) {
	if !m.IsLeader() {
		return nil, fmt.Errorf("not the leader, tokens can only be generated by the leader")
	}

	// Token valid for 24 hours
	return m.tokenManager.GenerateToken(role, 24*time.Hour)
}

// ValidateJoinToken validates a join token
func (m *Manager) ValidateJoinToken(token string) (string, error) {
	return m.tokenManager.ValidateToken(token)
}

// Shutdown gracefully shuts down the manager
func (m *Manager) Shutdown() error {
	// Stop DNS server first
	if m.dnsServer != nil {
		if err := m.dnsServer.Stop(); err != nil {
			fmt.Printf("Warning: failed to stop DNS server: %v\n", err)
		}
	}

	// Cancel DNS context
	if m.dnsCancel != nil {
		m.dnsCancel()
	}

	// Stop event broker
	if m.eventBroker != nil {
		m.eventBroker.Stop()
	}

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

// initializeCA initializes the Certificate Authority for a new cluster
func (m *Manager) initializeCA() error {
	// Check if CA already exists in storage
	if m.ca.IsInitialized() {
		fmt.Println("✓ Certificate Authority already initialized")
		return nil
	}

	// Try to load existing CA from storage
	err := m.ca.LoadFromStore()
	if err == nil {
		fmt.Println("✓ Loaded existing Certificate Authority")
		return nil
	}

	// CA doesn't exist, create new one
	fmt.Println("Initializing new Certificate Authority...")
	if err := m.ca.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize CA: %v", err)
	}

	// Save CA to storage
	if err := m.ca.SaveToStore(); err != nil {
		return fmt.Errorf("failed to save CA: %v", err)
	}

	fmt.Println("✓ Certificate Authority initialized and saved")

	// Issue certificate for this manager node
	certDir, err := security.GetCertDir("manager", m.nodeID)
	if err != nil {
		return fmt.Errorf("failed to get cert directory: %v", err)
	}

	// Check if certificate already exists
	if !security.CertExists(certDir) {
		fmt.Printf("Issuing certificate for manager %s...\n", m.nodeID)

		// Extract IP from bind address for certificate SAN
		host, _, err := net.SplitHostPort(m.bindAddr)
		if err != nil {
			return fmt.Errorf("failed to parse bind address: %v", err)
		}
		ip := net.ParseIP(host)
		var ipAddresses []net.IP
		if ip != nil {
			ipAddresses = []net.IP{ip}
		}

		// DNS names for the manager
		dnsNames := []string{
			fmt.Sprintf("manager-%s", m.nodeID),
			"localhost",
		}

		cert, err := m.ca.IssueNodeCertificate(m.nodeID, "manager", dnsNames, ipAddresses)
		if err != nil {
			return fmt.Errorf("failed to issue node certificate: %v", err)
		}

		// Save certificate to file
		if err := security.SaveCertToFile(cert, certDir); err != nil {
			return fmt.Errorf("failed to save certificate: %v", err)
		}

		// Save CA certificate
		caCert := m.ca.GetRootCACert()
		if err := security.SaveCACertToFile(caCert, certDir); err != nil {
			return fmt.Errorf("failed to save CA certificate: %v", err)
		}

		fmt.Printf("✓ Certificate issued and saved to %s\n", certDir)
	} else {
		fmt.Printf("✓ Certificate already exists at %s\n", certDir)
	}

	return nil
}

// IssueCertificate issues a certificate for a node
func (m *Manager) IssueCertificate(nodeID, role string) (*tls.Certificate, error) {
	if !m.ca.IsInitialized() {
		return nil, fmt.Errorf("CA not initialized")
	}

	// For workers and CLI, no DNS names or IP addresses needed (client certs)
	// They connect to the manager, not vice versa
	return m.ca.IssueNodeCertificate(nodeID, role, nil, nil)
}

// CertToPEM converts a TLS certificate to PEM format
func (m *Manager) CertToPEM(cert *tls.Certificate) (certPEM, keyPEM []byte, err error) {
	if cert == nil {
		return nil, nil, fmt.Errorf("certificate is nil")
	}

	// Encode certificate
	certPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Certificate[0],
	})

	// Encode private key
	privateKey, ok := cert.PrivateKey.(*rsa.PrivateKey)
	if !ok {
		return nil, nil, fmt.Errorf("private key is not RSA")
	}

	keyPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	return certPEM, keyPEM, nil
}

// GetCACertPEM returns the CA certificate in PEM format
func (m *Manager) GetCACertPEM() []byte {
	if !m.ca.IsInitialized() {
		return nil
	}

	caCertDER := m.ca.GetRootCACert()
	return pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caCertDER,
	})
}

// ValidateToken validates a join token and returns the role
func (m *Manager) ValidateToken(token string) (string, error) {
	return m.tokenManager.ValidateToken(token)
}

// NodeID returns the manager's node ID
func (m *Manager) NodeID() string {
	return m.nodeID
}

// --- Ingress Operations ---

// CreateIngress creates a new ingress via Raft
func (m *Manager) CreateIngress(ingress *types.Ingress) error {
	return m.applyRaftCommand("CreateIngress", ingress)
}

// UpdateIngress updates an ingress via Raft
func (m *Manager) UpdateIngress(ingress *types.Ingress) error {
	return m.applyRaftCommand("UpdateIngress", ingress)
}

// DeleteIngress deletes an ingress via Raft
func (m *Manager) DeleteIngress(ingressID string) error {
	return m.applyRaftCommand("DeleteIngress", map[string]string{"id": ingressID})
}
