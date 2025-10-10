package worker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cuemby/warren/api/proto"
	"github.com/cuemby/warren/pkg/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Worker represents a Warren worker node
type Worker struct {
	nodeID      string
	managerAddr string
	dataDir     string

	client proto.WarrenAPIClient
	conn   *grpc.ClientConn

	tasks  map[string]*types.Task
	taskMu sync.RWMutex

	stopCh chan struct{}
}

// Config holds worker configuration
type Config struct {
	NodeID      string
	ManagerAddr string
	DataDir     string
	Resources   *types.NodeResources
}

// NewWorker creates a new worker instance
func NewWorker(cfg *Config) (*Worker, error) {
	w := &Worker{
		nodeID:      cfg.NodeID,
		managerAddr: cfg.ManagerAddr,
		dataDir:     cfg.DataDir,
		tasks:       make(map[string]*types.Task),
		stopCh:      make(chan struct{}),
	}

	return w, nil
}

// Start starts the worker and connects to manager
func (w *Worker) Start(resources *types.NodeResources) error {
	// Connect to manager
	conn, err := grpc.Dial(w.managerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to manager: %v", err)
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
		return fmt.Errorf("failed to register with manager: %v", err)
	}

	fmt.Printf("Worker registered with manager\n")
	fmt.Printf("  Node ID: %s\n", resp.Node.Id)
	fmt.Printf("  Overlay IP: %s\n", resp.OverlayIp)

	// Start heartbeat loop
	go w.heartbeatLoop()

	// Start task executor loop
	go w.taskExecutorLoop()

	return nil
}

// Stop stops the worker
func (w *Worker) Stop() error {
	close(w.stopCh)

	if w.conn != nil {
		w.conn.Close()
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
			task := &types.Task{
				ID:           protoTask.Id,
				ServiceID:    protoTask.ServiceId,
				ServiceName:  protoTask.ServiceName,
				NodeID:       protoTask.NodeId,
				DesiredState: types.TaskState(protoTask.DesiredState),
				ActualState:  types.TaskStatePending,
				Image:        protoTask.Image,
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

// executeTask executes a single task (simulated for now)
func (w *Worker) executeTask(task *types.Task) {
	fmt.Printf("Starting task %s (service: %s, image: %s)\n", task.ID, task.ServiceName, task.Image)

	// Update task state to running
	w.taskMu.Lock()
	task.ActualState = types.TaskStateRunning
	task.ContainerID = fmt.Sprintf("container-%s", task.ID[:8])
	task.StartedAt = time.Now()
	w.taskMu.Unlock()

	// Simulate container execution
	// TODO: Replace with actual containerd integration
	fmt.Printf("Task %s is running (simulated)\n", task.ID)

	// Keep task running until stopped
	ticker := time.NewTicker(10 * time.Second)
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

			// Simulate random failures (10% chance every 10 seconds)
			// TODO: Remove this simulation
			// if rand.Intn(10) == 0 {
			// 	w.taskMu.Lock()
			// 	task.ActualState = types.TaskStateFailed
			// 	task.Error = "Simulated failure"
			// 	w.taskMu.Unlock()
			// 	fmt.Printf("Task %s failed (simulated)\n", task.ID)
			// 	return
			// }

		case <-w.stopCh:
			return
		}
	}
}

// stopTask stops a running task
func (w *Worker) stopTask(task *types.Task) {
	fmt.Printf("Stopping task %s\n", task.ID)

	w.taskMu.Lock()
	task.ActualState = types.TaskStateComplete
	task.FinishedAt = time.Now()
	w.taskMu.Unlock()

	// TODO: Actually stop container via containerd

	// Remove from local task map after reporting
	time.Sleep(2 * time.Second)
	w.taskMu.Lock()
	delete(w.tasks, task.ID)
	w.taskMu.Unlock()

	fmt.Printf("Task %s stopped\n", task.ID)
}
