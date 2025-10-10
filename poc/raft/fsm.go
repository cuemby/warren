package main

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/hashicorp/raft"
)

// KeyValueFSM is a simple finite state machine for testing Raft
// It implements a basic key-value store
type KeyValueFSM struct {
	mu   sync.RWMutex
	data map[string]string
}

// Command represents a state change operation
type Command struct {
	Op    string `json:"op"`    // "set" or "delete"
	Key   string `json:"key"`
	Value string `json:"value"`
}

// NewKeyValueFSM creates a new FSM instance
func NewKeyValueFSM() *KeyValueFSM {
	return &KeyValueFSM{
		data: make(map[string]string),
	}
}

// Apply applies a Raft log entry to the FSM
// This is called by Raft when a log entry is committed
func (f *KeyValueFSM) Apply(log *raft.Log) interface{} {
	var cmd Command
	if err := json.Unmarshal(log.Data, &cmd); err != nil {
		return fmt.Errorf("failed to unmarshal command: %v", err)
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	switch cmd.Op {
	case "set":
		f.data[cmd.Key] = cmd.Value
		return nil
	case "delete":
		delete(f.data, cmd.Key)
		return nil
	default:
		return fmt.Errorf("unknown operation: %s", cmd.Op)
	}
}

// Snapshot creates a point-in-time snapshot of the FSM
func (f *KeyValueFSM) Snapshot() (raft.FSMSnapshot, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// Deep copy the data
	snapshot := make(map[string]string)
	for k, v := range f.data {
		snapshot[k] = v
	}

	return &KeyValueSnapshot{data: snapshot}, nil
}

// Restore restores the FSM from a snapshot
func (f *KeyValueFSM) Restore(rc io.ReadCloser) error {
	defer rc.Close()

	var data map[string]string
	if err := json.NewDecoder(rc).Decode(&data); err != nil {
		return err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.data = data
	return nil
}

// Get retrieves a value (read-only operation, not via Raft)
func (f *KeyValueFSM) Get(key string) (string, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	val, ok := f.data[key]
	return val, ok
}

// KeyValueSnapshot represents a snapshot of the FSM
type KeyValueSnapshot struct {
	data map[string]string
}

// Persist writes the snapshot to the given SnapshotSink
func (s *KeyValueSnapshot) Persist(sink raft.SnapshotSink) error {
	err := func() error {
		// Encode data as JSON
		b, err := json.Marshal(s.data)
		if err != nil {
			return err
		}

		// Write to sink
		if _, err := sink.Write(b); err != nil {
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
func (s *KeyValueSnapshot) Release() {}
