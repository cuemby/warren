package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
)

func main() {
	var (
		nodeID   = flag.String("id", "node1", "Node ID")
		raftAddr = flag.String("addr", "127.0.0.1:8001", "Raft bind address")
		joinAddr = flag.String("join", "", "Address of leader to join (empty for bootstrap)")
		dataDir  = flag.String("data", "", "Data directory (defaults to /tmp/raft-<id>)")
	)
	flag.Parse()

	if *dataDir == "" {
		*dataDir = filepath.Join(os.TempDir(), fmt.Sprintf("raft-%s", *nodeID))
	}

	// Create data directory
	if err := os.MkdirAll(*dataDir, 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	log.Printf("Starting Raft node %s at %s", *nodeID, *raftAddr)
	log.Printf("Data directory: %s", *dataDir)

	// Setup Raft configuration
	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(*nodeID)
	config.Logger = log.New(os.Stdout, fmt.Sprintf("[%s] ", *nodeID), log.LstdFlags)

	// Create FSM
	fsm := NewKeyValueFSM()

	// Setup Raft communication
	addr, err := net.ResolveTCPAddr("tcp", *raftAddr)
	if err != nil {
		log.Fatalf("Failed to resolve address: %v", err)
	}

	transport, err := raft.NewTCPTransport(*raftAddr, addr, 3, 10*time.Second, os.Stdout)
	if err != nil {
		log.Fatalf("Failed to create transport: %v", err)
	}

	// Create snapshot store
	snapshots, err := raft.NewFileSnapshotStore(*dataDir, 3, os.Stdout)
	if err != nil {
		log.Fatalf("Failed to create snapshot store: %v", err)
	}

	// Create log store and stable store
	logStore, err := raftboltdb.NewBoltStore(filepath.Join(*dataDir, "raft-log.db"))
	if err != nil {
		log.Fatalf("Failed to create log store: %v", err)
	}

	stableStore, err := raftboltdb.NewBoltStore(filepath.Join(*dataDir, "raft-stable.db"))
	if err != nil {
		log.Fatalf("Failed to create stable store: %v", err)
	}

	// Create Raft instance
	r, err := raft.NewRaft(config, fsm, logStore, stableStore, snapshots, transport)
	if err != nil {
		log.Fatalf("Failed to create Raft instance: %v", err)
	}

	// Bootstrap or join cluster
	if *joinAddr == "" {
		// Bootstrap as single-node cluster
		configuration := raft.Configuration{
			Servers: []raft.Server{
				{
					ID:      config.LocalID,
					Address: transport.LocalAddr(),
				},
			},
		}
		future := r.BootstrapCluster(configuration)
		if err := future.Error(); err != nil {
			log.Fatalf("Failed to bootstrap cluster: %v", err)
		}
		log.Printf("Bootstrapped single-node cluster")
	} else {
		// Join existing cluster
		// Note: In production, this would be done via an API call to the leader
		log.Printf("Joining cluster via %s (manual configuration required)", *joinAddr)
		log.Printf("On leader, run: AddVoter(%s, %s)", *nodeID, *raftAddr)
	}

	// Wait for leader election
	log.Println("Waiting for leader election...")
	for {
		if r.State() == raft.Leader {
			log.Println("✓ This node is the LEADER")
			break
		}
		if leaderAddr, _ := r.LeaderWithID(); leaderAddr != "" {
			log.Printf("✓ Leader is at: %s", leaderAddr)
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	// Test: Write some data if leader
	if r.State() == raft.Leader {
		log.Println("\n--- Testing Raft operations ---")

		// Write operation
		cmd := Command{Op: "set", Key: "test-key", Value: "test-value"}
		data, _ := json.Marshal(cmd)

		log.Println("Applying command: set test-key=test-value")
		applyFuture := r.Apply(data, 5*time.Second)
		if err := applyFuture.Error(); err != nil {
			log.Printf("Failed to apply command: %v", err)
		} else {
			log.Println("✓ Command applied successfully")
		}

		// Wait a bit for replication
		time.Sleep(1 * time.Second)

		// Read from FSM
		if val, ok := fsm.Get("test-key"); ok {
			log.Printf("✓ Read from FSM: test-key=%s", val)
		}
	}

	// Keep running
	log.Println("\nNode running. Press Ctrl+C to stop.")
	log.Printf("State: %s", r.State())
	log.Printf("Leader: %v", r.Leader())

	// Print stats every 10 seconds
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		stats := r.Stats()
		log.Printf("--- Stats ---")
		log.Printf("State: %s", r.State())
		log.Printf("Leader: %s", stats["leader"])
		log.Printf("Last Log Index: %s", stats["last_log_index"])
		log.Printf("Commit Index: %s", stats["commit_index"])
	}
}
