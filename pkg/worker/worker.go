package worker

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"sync"
	"time"

	specs "github.com/opencontainers/runtime-spec/specs-go"

	"github.com/cuemby/warren/api/proto"
	"github.com/cuemby/warren/pkg/network"
	"github.com/cuemby/warren/pkg/runtime"
	"github.com/cuemby/warren/pkg/security"
	"github.com/cuemby/warren/pkg/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Worker represents a Warren worker node
type Worker struct {
	nodeID      string
	managerAddr string
	dataDir     string

	client         proto.WarrenAPIClient
	conn           *grpc.ClientConn
	runtime        *runtime.ContainerdRuntime
	secretsHandler *SecretsHandler
	volumesHandler *VolumesHandler
	healthMonitor  *HealthMonitor
	dnsHandler     *DNSHandler
	portPublisher  *network.HostPortPublisher

	containers   map[string]*types.Container
	containersMu sync.RWMutex

	stopCh chan struct{}
}

// Config holds worker configuration
type Config struct {
	NodeID           string
	ManagerAddr      string
	DataDir          string
	Resources        *types.NodeResources
	EncryptionKey    []byte // Cluster-wide encryption key for secrets
	ContainerdSocket string // Containerd socket path (empty = auto-detect)
	JoinToken        string // Join token for initial authentication
}

// NewWorker creates a new worker instance
func NewWorker(cfg *Config) (*Worker, error) {
	// Initialize containerd runtime
	rt, err := runtime.NewContainerdRuntime(cfg.ContainerdSocket)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize containerd runtime: %w", err)
	}

	w := &Worker{
		nodeID:      cfg.NodeID,
		managerAddr: cfg.ManagerAddr,
		dataDir:     cfg.DataDir,
		runtime:     rt,
		containers:  make(map[string]*types.Container),
		stopCh:      make(chan struct{}),
	}

	// Initialize secrets handler if encryption key provided
	if len(cfg.EncryptionKey) > 0 {
		sh, err := NewSecretsHandler(w, cfg.EncryptionKey)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize secrets handler: %w", err)
		}
		w.secretsHandler = sh

		// Ensure secrets base directory exists
		if err := EnsureSecretsBaseDir(); err != nil {
			return nil, fmt.Errorf("failed to ensure secrets directory: %w", err)
		}
	}

	// Initialize volumes handler
	vh, err := NewVolumesHandler(w)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize volumes handler: %w", err)
	}
	w.volumesHandler = vh

	// Initialize DNS handler
	managerIP := ExtractManagerIP(cfg.ManagerAddr)
	dh, err := NewDNSHandler(w, managerIP)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize DNS handler: %w", err)
	}
	w.dnsHandler = dh

	// Initialize health monitor
	w.healthMonitor = NewHealthMonitor(w)

	// Initialize port publisher for host mode port publishing
	w.portPublisher = network.NewHostPortPublisher()

	return w, nil
}

// NewEmbeddedWorker creates a worker optimized for in-process embedding with a manager (hybrid mode)
// This is identical to NewWorker but documents the intended use case for embedded workers
func NewEmbeddedWorker(cfg *Config) (*Worker, error) {
	// Embedded workers work exactly like regular workers, but they:
	// 1. Connect to localhost manager (same process)
	// 2. Share the same node ID as the manager
	// 3. Don't need separate certificate request (same process, trusted)
	return NewWorker(cfg)
}

// Start starts the worker and connects to manager
func (w *Worker) Start(resources *types.NodeResources, joinToken string) error {
	// Ensure worker has a certificate
	certDir, err := security.GetCertDir("worker", w.nodeID)
	if err != nil {
		return fmt.Errorf("failed to get cert directory: %w", err)
	}

	// Request certificate if not exists
	if !security.CertExists(certDir) {
		fmt.Println("Worker certificate not found, requesting from manager...")
		if err := w.requestCertificate(joinToken); err != nil {
			return fmt.Errorf("failed to request certificate: %w", err)
		}
		fmt.Printf("✓ Certificate obtained and saved to %s\n", certDir)
	} else {
		fmt.Printf("✓ Using existing certificate from %s\n", certDir)
	}

	// Connect to manager with mTLS
	conn, err := w.connectWithMTLS(certDir)
	if err != nil {
		return fmt.Errorf("failed to connect to manager: %w", err)
	}
	w.conn = conn
	w.client = proto.NewWarrenAPIClient(conn)

	// Register with manager
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := w.client.RegisterNode(ctx, &proto.RegisterNodeRequest{
		Id:      w.nodeID,
		Role:    "worker",
		Address: "localhost", // TODO: Get actual address
		Resources: &proto.NodeResources{
			CpuCores:    int64(resources.CPUCores),
			MemoryBytes: resources.MemoryBytes,
			DiskBytes:   resources.DiskBytes,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to register with manager: %w", err)
	}

	fmt.Printf("Worker registered with manager\n")
	fmt.Printf("  Node ID: %s\n", resp.Node.Id)
	fmt.Printf("  Overlay IP: %s\n", resp.OverlayIp)

	// Start heartbeat loop
	go w.heartbeatLoop()

	// Start task executor loop
	go w.containerExecutorLoop()

	// Start health monitor
	w.healthMonitor.Start()

	return nil
}

// Stop stops the worker
func (w *Worker) Stop() error {
	close(w.stopCh)

	// Stop health monitor
	if w.healthMonitor != nil {
		w.healthMonitor.Stop()
	}

	if w.conn != nil {
		w.conn.Close()
	}

	if w.runtime != nil {
		w.runtime.Close()
	}

	return nil
}

// heartbeatLoop sends periodic heartbeats to manager
func (w *Worker) heartbeatLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := w.sendHeartbeat(); err != nil {
				fmt.Printf("Heartbeat error: %v\n", err)
			}
		case <-w.stopCh:
			return
		}
	}
}

// sendHeartbeat sends a heartbeat with container status to manager
func (w *Worker) sendHeartbeat() error {
	w.containersMu.RLock()
	containerStatuses := make([]*proto.ContainerStatus, 0, len(w.containers))
	for _, container := range w.containers {
		containerStatuses = append(containerStatuses, &proto.ContainerStatus{
			ContainerId:        container.ID,
			ActualState:        string(container.ActualState),
			RuntimeContainerId: container.ContainerID,
			Error:              container.Error,
		})
	}
	w.containersMu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := w.client.Heartbeat(ctx, &proto.HeartbeatRequest{
		NodeId:            w.nodeID,
		ContainerStatuses: containerStatuses,
	})

	return err
}

// containerExecutorLoop polls for container assignments and executes them
func (w *Worker) containerExecutorLoop() {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := w.syncContainers(); err != nil {
				fmt.Printf("Container sync error: %v\n", err)
			}
		case <-w.stopCh:
			return
		}
	}
}

// syncContainers fetches assigned containers from manager and executes them
func (w *Worker) syncContainers() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get all containers assigned to this node
	resp, err := w.client.ListContainers(ctx, &proto.ListContainersRequest{
		NodeId: w.nodeID,
	})
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	// Process each container
	for _, protoContainer := range resp.Containers {
		containerID := protoContainer.Id

		w.containersMu.Lock()
		existingContainer, exists := w.containers[containerID]
		w.containersMu.Unlock()

		// New container - start it
		if !exists && protoContainer.DesiredState == "running" {
			// Convert proto volume mounts to types.VolumeMount
			var mounts []*types.VolumeMount
			for _, pv := range protoContainer.Volumes {
				mounts = append(mounts, &types.VolumeMount{
					Source:   pv.Source,
					Target:   pv.Target,
					ReadOnly: pv.ReadOnly,
				})
			}

			container := &types.Container{
				ID:           protoContainer.Id,
				ServiceID:    protoContainer.ServiceId,
				ServiceName:  protoContainer.ServiceName,
				NodeID:       protoContainer.NodeId,
				DesiredState: types.ContainerState(protoContainer.DesiredState),
				ActualState:  types.ContainerStatePending,
				Image:        protoContainer.Image,
				Secrets:      protoContainer.Secrets,
				Mounts:       mounts,
			}

			w.containersMu.Lock()
			w.containers[containerID] = container
			w.containersMu.Unlock()

			go w.executeContainer(container)
		}

		// Existing container - handle shutdown
		if exists && protoContainer.DesiredState == "shutdown" {
			go w.stopContainer(existingContainer)
		}
	}

	return nil
}

// executeContainer executes a single task using containerd
func (w *Worker) executeContainer(task *types.Container) {
	ctx := context.Background()
	fmt.Printf("Starting task %s (service: %s, image: %s)\n", task.ID, task.ServiceName, task.Image)

	// Pull the image first
	fmt.Printf("Pulling image %s...\n", task.Image)
	if err := w.runtime.PullImage(ctx, task.Image); err != nil {
		w.containersMu.Lock()
		task.ActualState = types.ContainerStateFailed
		task.Error = fmt.Sprintf("failed to pull image: %v", err)
		w.containersMu.Unlock()
		fmt.Printf("Task %s failed to pull image: %v\n", task.ID, err)
		return
	}
	fmt.Printf("Image %s pulled successfully\n", task.Image)

	// Mount secrets if task has them
	var secretsPath string
	if len(task.Secrets) > 0 && w.secretsHandler != nil {
		fmt.Printf("Mounting %d secret(s) for task %s...\n", len(task.Secrets), task.ID)
		var err error
		secretsPath, err = w.secretsHandler.MountSecretsForTask(task)
		if err != nil {
			w.containersMu.Lock()
			task.ActualState = types.ContainerStateFailed
			task.Error = fmt.Sprintf("failed to mount secrets: %v", err)
			w.containersMu.Unlock()
			fmt.Printf("Task %s failed to mount secrets: %v\n", task.ID, err)
			return
		}
		fmt.Printf("Secrets mounted at %s\n", secretsPath)

		// Ensure cleanup on exit
		defer func() {
			if err := w.secretsHandler.CleanupSecretsForTask(task.ID); err != nil {
				fmt.Printf("Warning: failed to cleanup secrets for task %s: %v\n", task.ID, err)
			}
		}()
	}

	// Prepare volumes if task has them
	var volumeMounts []specs.Mount
	if len(task.Mounts) > 0 && w.volumesHandler != nil {
		fmt.Printf("Preparing %d volume(s) for task %s...\n", len(task.Mounts), task.ID)
		var err error
		volumeMounts, err = w.volumesHandler.PrepareVolumesForTask(task)
		if err != nil {
			w.containersMu.Lock()
			task.ActualState = types.ContainerStateFailed
			task.Error = fmt.Sprintf("failed to prepare volumes: %v", err)
			w.containersMu.Unlock()
			fmt.Printf("Task %s failed to prepare volumes: %v\n", task.ID, err)
			return
		}
		fmt.Printf("Volumes prepared: %d mount(s)\n", len(volumeMounts))

		// Ensure cleanup on exit
		defer func() {
			if err := w.volumesHandler.CleanupVolumesForTask(task); err != nil {
				fmt.Printf("Warning: failed to cleanup volumes for task %s: %v\n", task.ID, err)
			}
		}()
	}

	// Get DNS configuration (resolv.conf path)
	var resolvConfPath string
	var err error
	if w.dnsHandler != nil {
		resolvConfPath, err = w.dnsHandler.GetResolvConfPath()
		if err != nil {
			fmt.Printf("Warning: failed to get DNS config for task %s: %v (continuing without DNS)\n", task.ID, err)
			resolvConfPath = "" // Continue without DNS if it fails
		} else {
			fmt.Printf("Using DNS config from %s\n", resolvConfPath)
		}
	}

	// Create the container with secrets, volumes, and DNS config
	var containerID string
	if secretsPath != "" || len(volumeMounts) > 0 || resolvConfPath != "" {
		containerID, err = w.runtime.CreateContainerWithMounts(ctx, task, secretsPath, volumeMounts, resolvConfPath)
	} else {
		containerID, err = w.runtime.CreateContainer(ctx, task)
	}

	if err != nil {
		w.containersMu.Lock()
		task.ActualState = types.ContainerStateFailed
		task.Error = fmt.Sprintf("failed to create container: %v", err)
		w.containersMu.Unlock()
		fmt.Printf("Task %s failed to create container: %v\n", task.ID, err)
		return
	}
	fmt.Printf("Container %s created\n", containerID)

	// Start the container
	if err := w.runtime.StartContainer(ctx, containerID); err != nil {
		w.containersMu.Lock()
		task.ActualState = types.ContainerStateFailed
		task.Error = fmt.Sprintf("failed to start container: %v", err)
		w.containersMu.Unlock()
		fmt.Printf("Task %s failed to start container: %v\n", task.ID, err)
		return
	}

	// Update task state to running
	w.containersMu.Lock()
	task.ActualState = types.ContainerStateRunning
	task.ContainerID = containerID
	task.StartedAt = time.Now()
	w.containersMu.Unlock()
	fmt.Printf("Task %s is running (container: %s)\n", task.ID, containerID)

	// Publish ports if task has any
	if len(task.Ports) > 0 && w.portPublisher != nil {
		// Get container IP from runtime
		containerIP, err := w.runtime.GetContainerIP(ctx, containerID)
		if err != nil {
			fmt.Printf("Warning: failed to get container IP for port publishing: %v\n", err)
		} else {
			// Convert []*PortMapping to []PortMapping for publisher
			var ports []types.PortMapping
			for _, p := range task.Ports {
				if p != nil {
					ports = append(ports, *p)
				}
			}

			fmt.Printf("Publishing %d port(s) for task %s (container IP: %s)\n",
				len(ports), task.ID, containerIP)

			if err := w.portPublisher.PublishPorts(task.ID, containerIP, ports); err != nil {
				fmt.Printf("Warning: failed to publish ports for task %s: %v\n", task.ID, err)
				// Don't fail the task if port publishing fails - log and continue
			} else {
				fmt.Printf("✓ Ports published for task %s\n", task.ID)
			}
		}
	}

	// Monitor container status
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Check if task should be stopped
			w.containersMu.RLock()
			currentTask := w.containers[task.ID]
			w.containersMu.RUnlock()

			if currentTask == nil || currentTask.DesiredState == types.ContainerStateShutdown {
				return
			}

			// Check container status
			status, err := w.runtime.GetContainerStatus(ctx, containerID)
			if err != nil {
				fmt.Printf("Failed to get container status: %v\n", err)
				continue
			}

			// Update task state if container failed
			if status == types.ContainerStateFailed || status == types.ContainerStateComplete {
				w.containersMu.Lock()
				task.ActualState = status
				if status == types.ContainerStateFailed {
					task.Error = "container exited unexpectedly"
				}
				w.containersMu.Unlock()
				fmt.Printf("Task %s container stopped (status: %s)\n", task.ID, status)
				return
			}

		case <-w.stopCh:
			return
		}
	}
}

// stopContainer stops a running task
func (w *Worker) stopContainer(task *types.Container) {
	ctx := context.Background()
	fmt.Printf("Stopping task %s (container: %s)\n", task.ID, task.ContainerID)

	// Determine stop timeout (default: 10 seconds)
	stopTimeout := 10 * time.Second
	if task.StopTimeout > 0 {
		stopTimeout = time.Duration(task.StopTimeout) * time.Second
	}

	// Stop the container
	if task.ContainerID != "" {
		fmt.Printf("Sending SIGTERM to container %s (timeout: %v)\n", task.ContainerID, stopTimeout)
		if err := w.runtime.StopContainer(ctx, task.ContainerID, stopTimeout); err != nil {
			fmt.Printf("Failed to stop container %s: %v\n", task.ContainerID, err)
		}

		// Delete the container
		if err := w.runtime.DeleteContainer(ctx, task.ContainerID); err != nil {
			fmt.Printf("Failed to delete container %s: %v\n", task.ContainerID, err)
		}
	}

	// Cleanup secrets if task had any
	if len(task.Secrets) > 0 && w.secretsHandler != nil {
		if err := w.secretsHandler.CleanupSecretsForTask(task.ID); err != nil {
			fmt.Printf("Warning: failed to cleanup secrets for task %s: %v\n", task.ID, err)
		} else {
			fmt.Printf("Secrets cleaned up for task %s\n", task.ID)
		}
	}

	// Cleanup published ports if task had any
	if len(task.Ports) > 0 && w.portPublisher != nil {
		if err := w.portPublisher.UnpublishPorts(task.ID); err != nil {
			fmt.Printf("Warning: failed to unpublish ports for task %s: %v\n", task.ID, err)
		} else {
			fmt.Printf("Ports unpublished for task %s\n", task.ID)
		}
	}

	w.containersMu.Lock()
	task.ActualState = types.ContainerStateComplete
	task.FinishedAt = time.Now()
	w.containersMu.Unlock()

	// Remove from local task map after reporting
	time.Sleep(2 * time.Second)
	w.containersMu.Lock()
	delete(w.containers, task.ID)
	w.containersMu.Unlock()

	fmt.Printf("Task %s stopped\n", task.ID)
}

// requestCertificate requests a certificate from the manager using a join token
func (w *Worker) requestCertificate(token string) error {
	// Connect with TLS but without client certificate (token provides authentication)
	// Skip server verification temporarily since we don't have the CA cert yet
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true, // Skip server cert verification for initial connection
		MinVersion:         tls.VersionTLS13,
	}
	creds := credentials.NewTLS(tlsConfig)

	conn, err := grpc.NewClient(w.managerAddr, grpc.WithTransportCredentials(creds))
	if err != nil {
		return fmt.Errorf("failed to connect to manager: %w", err)
	}
	defer conn.Close()

	client := proto.NewWarrenAPIClient(conn)

	// Request certificate
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.RequestCertificate(ctx, &proto.RequestCertificateRequest{
		NodeId: w.nodeID,
		Token:  token,
	})
	if err != nil {
		return fmt.Errorf("failed to request certificate: %w", err)
	}

	// Get certificate directory
	certDir, err := security.GetCertDir("worker", w.nodeID)
	if err != nil {
		return fmt.Errorf("failed to get cert directory: %w", err)
	}

	// Create directory
	if err := os.MkdirAll(certDir, 0700); err != nil {
		return fmt.Errorf("failed to create cert directory: %w", err)
	}

	// Save certificate files
	certPath := certDir + "/node.crt"
	keyPath := certDir + "/node.key"
	caPath := certDir + "/ca.crt"

	if err := os.WriteFile(certPath, resp.Certificate, 0600); err != nil {
		return fmt.Errorf("failed to write certificate: %w", err)
	}

	if err := os.WriteFile(keyPath, resp.PrivateKey, 0600); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	if err := os.WriteFile(caPath, resp.CaCert, 0644); err != nil {
		return fmt.Errorf("failed to write CA certificate: %w", err)
	}

	return nil
}

// connectWithMTLS establishes a gRPC connection with mTLS
func (w *Worker) connectWithMTLS(certDir string) (*grpc.ClientConn, error) {
	// Load worker certificate
	cert, err := security.LoadCertFromFile(certDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load worker certificate: %w", err)
	}

	// Load CA certificate for server verification
	caCert, err := security.LoadCACertFromFile(certDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load CA certificate: %w", err)
	}

	// Create cert pool for server verification
	certPool := x509.NewCertPool()
	certPool.AddCert(caCert)

	// Configure TLS
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{*cert},
		RootCAs:      certPool,
		MinVersion:   tls.VersionTLS13,
	}

	// Create gRPC connection with TLS
	creds := credentials.NewTLS(tlsConfig)
	conn, err := grpc.NewClient(w.managerAddr, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("failed to dial manager: %w", err)
	}

	return conn, nil
}
