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

	tasks  map[string]*types.Task
	taskMu sync.RWMutex

	stopCh chan struct{}
}

// Config holds worker configuration
type Config struct {
	NodeID            string
	ManagerAddr       string
	DataDir           string
	Resources         *types.NodeResources
	EncryptionKey     []byte // Cluster-wide encryption key for secrets
	ContainerdSocket  string // Containerd socket path (empty = auto-detect)
	JoinToken         string // Join token for initial authentication
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
		tasks:       make(map[string]*types.Task),
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

	return w, nil
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
	go w.taskExecutorLoop()

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

// sendHeartbeat sends a heartbeat with task status to manager
func (w *Worker) sendHeartbeat() error {
	w.taskMu.RLock()
	taskStatuses := make([]*proto.TaskStatus, 0, len(w.tasks))
	for _, task := range w.tasks {
		taskStatuses = append(taskStatuses, &proto.TaskStatus{
			TaskId:      task.ID,
			ActualState: string(task.ActualState),
			ContainerId: task.ContainerID,
			Error:       task.Error,
		})
	}
	w.taskMu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := w.client.Heartbeat(ctx, &proto.HeartbeatRequest{
		NodeId:       w.nodeID,
		TaskStatuses: taskStatuses,
	})

	return err
}

// taskExecutorLoop polls for task assignments and executes them
func (w *Worker) taskExecutorLoop() {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := w.syncTasks(); err != nil {
				fmt.Printf("Task sync error: %v\n", err)
			}
		case <-w.stopCh:
			return
		}
	}
}

// syncTasks fetches assigned tasks from manager and executes them
func (w *Worker) syncTasks() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get all tasks assigned to this node
	resp, err := w.client.ListTasks(ctx, &proto.ListTasksRequest{
		NodeId: w.nodeID,
	})
	if err != nil {
		return fmt.Errorf("failed to list tasks: %v", err)
	}

	// Process each task
	for _, protoTask := range resp.Tasks {
		taskID := protoTask.Id

		w.taskMu.Lock()
		existingTask, exists := w.tasks[taskID]
		w.taskMu.Unlock()

		// New task - start it
		if !exists && protoTask.DesiredState == "running" {
			// Convert proto volume mounts to types.VolumeMount
			var mounts []*types.VolumeMount
			for _, pv := range protoTask.Volumes {
				mounts = append(mounts, &types.VolumeMount{
					Source:   pv.Source,
					Target:   pv.Target,
					ReadOnly: pv.ReadOnly,
				})
			}

			task := &types.Task{
				ID:           protoTask.Id,
				ServiceID:    protoTask.ServiceId,
				ServiceName:  protoTask.ServiceName,
				NodeID:       protoTask.NodeId,
				DesiredState: types.TaskState(protoTask.DesiredState),
				ActualState:  types.TaskStatePending,
				Image:        protoTask.Image,
				Secrets:      protoTask.Secrets,
				Mounts:       mounts,
			}

			w.taskMu.Lock()
			w.tasks[taskID] = task
			w.taskMu.Unlock()

			go w.executeTask(task)
		}

		// Existing task - handle shutdown
		if exists && protoTask.DesiredState == "shutdown" {
			go w.stopTask(existingTask)
		}
	}

	return nil
}

// executeTask executes a single task using containerd
func (w *Worker) executeTask(task *types.Task) {
	ctx := context.Background()
	fmt.Printf("Starting task %s (service: %s, image: %s)\n", task.ID, task.ServiceName, task.Image)

	// Pull the image first
	fmt.Printf("Pulling image %s...\n", task.Image)
	if err := w.runtime.PullImage(ctx, task.Image); err != nil {
		w.taskMu.Lock()
		task.ActualState = types.TaskStateFailed
		task.Error = fmt.Sprintf("failed to pull image: %v", err)
		w.taskMu.Unlock()
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
			w.taskMu.Lock()
			task.ActualState = types.TaskStateFailed
			task.Error = fmt.Sprintf("failed to mount secrets: %v", err)
			w.taskMu.Unlock()
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
			w.taskMu.Lock()
			task.ActualState = types.TaskStateFailed
			task.Error = fmt.Sprintf("failed to prepare volumes: %v", err)
			w.taskMu.Unlock()
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
		w.taskMu.Lock()
		task.ActualState = types.TaskStateFailed
		task.Error = fmt.Sprintf("failed to create container: %v", err)
		w.taskMu.Unlock()
		fmt.Printf("Task %s failed to create container: %v\n", task.ID, err)
		return
	}
	fmt.Printf("Container %s created\n", containerID)

	// Start the container
	if err := w.runtime.StartContainer(ctx, containerID); err != nil {
		w.taskMu.Lock()
		task.ActualState = types.TaskStateFailed
		task.Error = fmt.Sprintf("failed to start container: %v", err)
		w.taskMu.Unlock()
		fmt.Printf("Task %s failed to start container: %v\n", task.ID, err)
		return
	}

	// Update task state to running
	w.taskMu.Lock()
	task.ActualState = types.TaskStateRunning
	task.ContainerID = containerID
	task.StartedAt = time.Now()
	w.taskMu.Unlock()
	fmt.Printf("Task %s is running (container: %s)\n", task.ID, containerID)

	// Monitor container status
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Check if task should be stopped
			w.taskMu.RLock()
			currentTask := w.tasks[task.ID]
			w.taskMu.RUnlock()

			if currentTask == nil || currentTask.DesiredState == types.TaskStateShutdown {
				return
			}

			// Check container status
			status, err := w.runtime.GetContainerStatus(ctx, containerID)
			if err != nil {
				fmt.Printf("Failed to get container status: %v\n", err)
				continue
			}

			// Update task state if container failed
			if status == types.TaskStateFailed || status == types.TaskStateComplete {
				w.taskMu.Lock()
				task.ActualState = status
				if status == types.TaskStateFailed {
					task.Error = "container exited unexpectedly"
				}
				w.taskMu.Unlock()
				fmt.Printf("Task %s container stopped (status: %s)\n", task.ID, status)
				return
			}

		case <-w.stopCh:
			return
		}
	}
}

// stopTask stops a running task
func (w *Worker) stopTask(task *types.Task) {
	ctx := context.Background()
	fmt.Printf("Stopping task %s (container: %s)\n", task.ID, task.ContainerID)

	// Stop the container
	if task.ContainerID != "" {
		if err := w.runtime.StopContainer(ctx, task.ContainerID, 10*time.Second); err != nil {
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

	w.taskMu.Lock()
	task.ActualState = types.TaskStateComplete
	task.FinishedAt = time.Now()
	w.taskMu.Unlock()

	// Remove from local task map after reporting
	time.Sleep(2 * time.Second)
	w.taskMu.Lock()
	delete(w.tasks, task.ID)
	w.taskMu.Unlock()

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

	conn, err := grpc.Dial(w.managerAddr, grpc.WithTransportCredentials(creds))
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
	conn, err := grpc.Dial(w.managerAddr, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("failed to dial manager: %w", err)
	}

	return conn, nil
}
