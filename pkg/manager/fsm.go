package manager

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/cuemby/warren/pkg/storage"
	"github.com/cuemby/warren/pkg/types"
	"github.com/hashicorp/raft"
)

// WarrenFSM implements the Raft Finite State Machine for Warren's cluster state
// It applies log entries to the cluster state and handles snapshots
type WarrenFSM struct {
	mu    sync.RWMutex
	store storage.Store
}

// NewWarrenFSM creates a new FSM instance
func NewWarrenFSM(store storage.Store) *WarrenFSM {
	return &WarrenFSM{
		store: store,
	}
}

// Command represents a state change operation in the Raft log
type Command struct {
	Op   string          `json:"op"`
	Data json.RawMessage `json:"data"`
}

// Apply applies a Raft log entry to the FSM
// This is called by Raft when a log entry is committed
func (f *WarrenFSM) Apply(log *raft.Log) interface{} {
	var cmd Command
	if err := json.Unmarshal(log.Data, &cmd); err != nil {
		return fmt.Errorf("failed to unmarshal command: %v", err)
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	switch cmd.Op {
	// Node operations
	case "create_node":
		var node types.Node
		if err := json.Unmarshal(cmd.Data, &node); err != nil {
			return err
		}
		return f.store.CreateNode(&node)

	case "update_node":
		var node types.Node
		if err := json.Unmarshal(cmd.Data, &node); err != nil {
			return err
		}
		return f.store.UpdateNode(&node)

	case "delete_node":
		var nodeID string
		if err := json.Unmarshal(cmd.Data, &nodeID); err != nil {
			return err
		}
		return f.store.DeleteNode(nodeID)

	// Service operations
	case "create_service":
		var service types.Service
		if err := json.Unmarshal(cmd.Data, &service); err != nil {
			return err
		}
		return f.store.CreateService(&service)

	case "update_service":
		var service types.Service
		if err := json.Unmarshal(cmd.Data, &service); err != nil {
			return err
		}
		return f.store.UpdateService(&service)

	case "delete_service":
		var serviceID string
		if err := json.Unmarshal(cmd.Data, &serviceID); err != nil {
			return err
		}
		return f.store.DeleteService(serviceID)

	// Task operations
	case "create_task":
		var task types.Task
		if err := json.Unmarshal(cmd.Data, &task); err != nil {
			return err
		}
		return f.store.CreateTask(&task)

	case "update_task":
		var task types.Task
		if err := json.Unmarshal(cmd.Data, &task); err != nil {
			return err
		}
		return f.store.UpdateTask(&task)

	case "delete_task":
		var taskID string
		if err := json.Unmarshal(cmd.Data, &taskID); err != nil {
			return err
		}
		return f.store.DeleteTask(taskID)

	// Secret operations
	case "create_secret":
		var secret types.Secret
		if err := json.Unmarshal(cmd.Data, &secret); err != nil {
			return err
		}
		return f.store.CreateSecret(&secret)

	case "delete_secret":
		var secretID string
		if err := json.Unmarshal(cmd.Data, &secretID); err != nil {
			return err
		}
		return f.store.DeleteSecret(secretID)

	// Volume operations
	case "create_volume":
		var volume types.Volume
		if err := json.Unmarshal(cmd.Data, &volume); err != nil {
			return err
		}
		return f.store.CreateVolume(&volume)

	case "delete_volume":
		var volumeID string
		if err := json.Unmarshal(cmd.Data, &volumeID); err != nil {
			return err
		}
		return f.store.DeleteVolume(volumeID)

	default:
		return fmt.Errorf("unknown command: %s", cmd.Op)
	}
}

// Snapshot creates a point-in-time snapshot of the FSM
// This is called periodically by Raft to compact the log
func (f *WarrenFSM) Snapshot() (raft.FSMSnapshot, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// Collect all state
	nodes, err := f.store.ListNodes()
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %v", err)
	}

	services, err := f.store.ListServices()
	if err != nil {
		return nil, fmt.Errorf("failed to list services: %v", err)
	}

	tasks, err := f.store.ListTasks()
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %v", err)
	}

	secrets, err := f.store.ListSecrets()
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets: %v", err)
	}

	volumes, err := f.store.ListVolumes()
	if err != nil {
		return nil, fmt.Errorf("failed to list volumes: %v", err)
	}

	networks, err := f.store.ListNetworks()
	if err != nil {
		return nil, fmt.Errorf("failed to list networks: %v", err)
	}

	snapshot := &WarrenSnapshot{
		Nodes:    nodes,
		Services: services,
		Tasks:    tasks,
		Secrets:  secrets,
		Volumes:  volumes,
		Networks: networks,
	}

	return snapshot, nil
}

// Restore restores the FSM from a snapshot
// This is called when a node restarts or joins the cluster
func (f *WarrenFSM) Restore(rc io.ReadCloser) error {
	defer rc.Close()

	var snapshot WarrenSnapshot
	if err := json.NewDecoder(rc).Decode(&snapshot); err != nil {
		return fmt.Errorf("failed to decode snapshot: %v", err)
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	// Restore all state
	for _, node := range snapshot.Nodes {
		if err := f.store.CreateNode(node); err != nil {
			return fmt.Errorf("failed to restore node: %v", err)
		}
	}

	for _, service := range snapshot.Services {
		if err := f.store.CreateService(service); err != nil {
			return fmt.Errorf("failed to restore service: %v", err)
		}
	}

	for _, task := range snapshot.Tasks {
		if err := f.store.CreateTask(task); err != nil {
			return fmt.Errorf("failed to restore task: %v", err)
		}
	}

	for _, secret := range snapshot.Secrets {
		if err := f.store.CreateSecret(secret); err != nil {
			return fmt.Errorf("failed to restore secret: %v", err)
		}
	}

	for _, volume := range snapshot.Volumes {
		if err := f.store.CreateVolume(volume); err != nil {
			return fmt.Errorf("failed to restore volume: %v", err)
		}
	}

	for _, network := range snapshot.Networks {
		if err := f.store.CreateNetwork(network); err != nil {
			return fmt.Errorf("failed to restore network: %v", err)
		}
	}

	return nil
}

// WarrenSnapshot represents a point-in-time snapshot of cluster state
type WarrenSnapshot struct {
	Nodes    []*types.Node
	Services []*types.Service
	Tasks    []*types.Task
	Secrets  []*types.Secret
	Volumes  []*types.Volume
	Networks []*types.Network
}

// Persist writes the snapshot to the given SnapshotSink
func (s *WarrenSnapshot) Persist(sink raft.SnapshotSink) error {
	err := func() error {
		// Encode snapshot as JSON
		if err := json.NewEncoder(sink).Encode(s); err != nil {
			return err
		}
		return sink.Close()
	}()

	if err != nil {
		sink.Cancel()
	}

	return err
}

// Release releases the snapshot resources
func (s *WarrenSnapshot) Release() {}
