/*
Package manager implements the Warren cluster manager node with Raft consensus.

The manager package is the control plane of Warren, responsible for cluster
coordination, state management, and orchestration decisions. Managers form a
highly-available quorum using the Raft consensus protocol, ensuring consistent
cluster state even during network partitions or node failures.

# Architecture

A Warren cluster consists of 1-7 manager nodes that form a Raft quorum:

	┌─────────────────────── MANAGER NODE ───────────────────────┐
	│                                                              │
	│  ┌──────────────────────────────────────────────┐          │
	│  │           gRPC API Server (port 8080)        │          │
	│  │  - 30+ methods for cluster operations        │          │
	│  └──────────────────┬───────────────────────────┘          │
	│                     │                                        │
	│  ┌──────────────────▼───────────────────────────┐          │
	│  │              Manager                          │          │
	│  │  - Handles API requests                       │          │
	│  │  - Proposes Raft commands                     │          │
	│  │  - Coordinates scheduler & reconciler         │          │
	│  │  - Manages tokens, secrets, DNS, ingress      │          │
	│  └──────────────────┬───────────────────────────┘          │
	│                     │                                        │
	│  ┌──────────────────▼───────────────────────────┐          │
	│  │          Raft Consensus Layer                 │          │
	│  │  - Leader election (2-3s failover)            │          │
	│  │  - Log replication across managers            │          │
	│  │  - FSM applies committed commands             │          │
	│  └──────────────────┬───────────────────────────┘          │
	│                     │                                        │
	│  ┌──────────────────▼───────────────────────────┐          │
	│  │         WarrenFSM (Finite State Machine)      │          │
	│  │  - Apply(): Process committed commands        │          │
	│  │  - Snapshot(): Create state snapshots         │          │
	│  │  - Restore(): Recover from snapshots          │          │
	│  └──────────────────┬───────────────────────────┘          │
	│                     │                                        │
	│  ┌──────────────────▼───────────────────────────┐          │
	│  │              BoltDB Store                      │          │
	│  │  - Nodes, Services, Tasks                     │          │
	│  │  - Secrets, Volumes, Certificates             │          │
	│  │  - Raft log and snapshots                     │          │
	│  └────────────────────────────────────────────────┘         │
	└──────────────────────────────────────────────────────────┘

# Core Components

Manager:
  - Main orchestration coordinator
  - Handles gRPC API requests
  - Proposes Raft commands for state changes
  - Manages scheduler and reconciler lifecycle
  - Coordinates DNS server and ingress proxy

WarrenFSM:
  - Raft finite state machine implementation
  - Applies committed log entries to cluster state
  - Implements snapshot/restore for fast recovery

TokenManager:
  - Generates and validates join tokens
  - Separate tokens for workers and managers
  - Time-limited tokens with rotation support

Command:
  - Encapsulates state change operations
  - Types: CreateService, UpdateTask, AddNode, etc.
  - Serialized as JSON in Raft log

# Raft Consensus

Warren uses HashiCorp's Raft library for distributed consensus.

Cluster Sizes:
  - 1 manager: Development only (no HA)
  - 3 managers: Production (tolerates 1 failure)
  - 5 managers: High availability (tolerates 2 failures)
  - 7 managers: Maximum recommended (tolerates 3 failures)

Quorum Requirements:
  - Write operations require majority quorum
  - Read operations served by leader (linearizable)
  - Leader election typically completes in 2-3 seconds
  - Network partition: Minority partition becomes read-only

Data Replication:
  - All state changes replicated via Raft log
  - Log entries applied to FSM in order
  - Snapshots created periodically for compaction
  - New managers sync via snapshot + log replay

# Usage

Creating a Manager:

	cfg := &manager.Config{
		NodeID:   "manager-1",
		BindAddr: "192.168.1.10:8080",
		DataDir:  "/var/lib/warren/manager-1",
	}

	mgr, err := manager.NewManager(cfg)
	if err != nil {
		log.Fatal(err)
	}

Initializing a Cluster:

	// First manager initializes the cluster
	err := mgr.InitCluster()
	if err != nil {
		log.Fatal(err)
	}

Joining Additional Managers:

	// Additional managers join existing cluster
	token := "manager-join-token-abc123"
	err := mgr.JoinCluster("192.168.1.10:8080", token)
	if err != nil {
		log.Fatal(err)
	}

Generating Join Tokens:

	// Leader generates tokens for new nodes
	workerToken, err := mgr.GenerateWorkerToken()
	if err != nil {
		log.Fatal(err)
	}

	managerToken, err := mgr.GenerateManagerToken()
	if err != nil {
		log.Fatal(err)
	}

Proposing State Changes:

	// All state changes go through Raft
	cmd := &manager.Command{
		Type: "create_service",
		Data: serviceJSON,
	}

	err := mgr.ProposeCommand(cmd)
	if err != nil {
		log.Fatal(err)
	}

# Leadership

Only the Raft leader can:
  - Accept write operations (state changes)
  - Schedule new tasks
  - Generate join tokens
  - Coordinate cluster operations

Followers:
  - Forward writes to leader automatically
  - Serve read operations (eventually consistent)
  - Participate in leader election
  - Replicate log entries from leader

When leader fails:
  - New leader elected in 2-3 seconds
  - Scheduler and reconciler start on new leader
  - Workers reconnect to new leader automatically
  - No service disruption (workers cache state)

# State Machine Commands

The FSM processes these command types:

Node Operations:
  - AddNode: Register new worker node
  - UpdateNodeStatus: Update node health and resources
  - RemoveNode: Decommission node

Service Operations:
  - CreateService: Deploy new service
  - UpdateService: Modify service (triggers rolling update)
  - DeleteService: Remove service and all tasks

Task Operations:
  - CreateTask: Create new task instance
  - UpdateTask: Update task state and metadata
  - DeleteTask: Remove completed/failed task

Secret Operations:
  - CreateSecret: Store encrypted secret
  - DeleteSecret: Remove secret (if not in use)

Volume Operations:
  - CreateVolume: Create persistent volume
  - DeleteVolume: Remove volume (if not in use)

Certificate Operations:
  - CreateCertificate: Upload TLS certificate
  - UpdateCertificate: Renew certificate (Let's Encrypt)
  - DeleteCertificate: Remove certificate

Ingress Operations:
  - CreateIngress: Create HTTP/HTTPS ingress rule
  - UpdateIngress: Modify routing rules
  - DeleteIngress: Remove ingress

# Failure Scenarios

Manager Failure:
  - If follower fails: No impact (quorum maintained)
  - If leader fails: New election (2-3s downtime)
  - Raft handles seamlessly

Network Partition:
  - Majority partition: Continues operating (elects leader)
  - Minority partition: Read-only mode (no writes accepted)
  - Partition heals: Minority syncs from majority

Data Corruption:
  - BoltDB checksums detect corruption
  - Restore from latest snapshot
  - Sync missing log entries from peers

# Performance Characteristics

Raft Operations:
  - Write latency: ~10ms (local quorum)
  - Snapshot interval: Every 10,000 log entries
  - Max log size: Configurable (default 1GB)
  - Heartbeat interval: 1 second

API Throughput:
  - Service creation: 10/sec (linearizable writes)
  - Task updates: 100/sec (batched FSM applies)
  - Read operations: 1000/sec (leader serving)

Memory Usage:
  - Base manager: 50MB
  - Per service: ~10KB
  - Per task: ~5KB
  - Per node: ~2KB
  - Typical 3-manager cluster: ~256MB total

# Integration Points

This package integrates with:

  - pkg/api: Provides gRPC server implementation
  - pkg/storage: Persists cluster state to BoltDB
  - pkg/scheduler: Coordinates task scheduling
  - pkg/reconciler: Coordinates failure detection
  - pkg/security: Manages secrets encryption and CA
  - pkg/dns: Provides DNS server for service discovery
  - pkg/ingress: Provides HTTP/HTTPS ingress controller
  - pkg/events: Publishes cluster events

# Design Patterns

Command Pattern:
  - All state changes encapsulated as commands
  - Commands serialized and replicated via Raft
  - FSM applies commands to achieve state transitions

Leader Pattern:
  - Single leader coordinates operations
  - Followers forward writes to leader
  - Automatic failover on leader failure

Token Pattern:
  - Time-limited join tokens for authentication
  - Separate tokens for workers and managers
  - Tokens rotated periodically for security

# Security

Join Token Security:
  - Tokens generated with cryptographic randomness
  - Time-limited validity (default 1 hour)
  - Separate tokens for workers and managers
  - Tokens never logged or exposed in API

mTLS Support:
  - Manager-to-manager: Raft over mTLS (future)
  - Manager-to-worker: gRPC with TLS (future)
  - Certificate rotation: Automated (future)

Secrets Encryption:
  - AES-256-GCM for secret data
  - Encryption key derived from cluster ID
  - Keys never stored on disk unencrypted

# High Availability

3-Manager Cluster (Production):
  - Tolerates 1 manager failure
  - Requires 2/3 quorum for writes
  - Recommended for production workloads

5-Manager Cluster (High Availability):
  - Tolerates 2 manager failures
  - Requires 3/5 quorum for writes
  - Recommended for critical workloads

Best Practices:
  - Deploy managers across availability zones
  - Use fast, reliable network between managers
  - Monitor Raft metrics (leader elections, log size)
  - Regular snapshot backups

# Troubleshooting

Common Issues:

Leader Election Storms:
  - Symptom: Frequent leader changes
  - Cause: Network latency or clock skew
  - Solution: Check network and NTP sync

Split Brain:
  - Symptom: Multiple leaders claimed
  - Cause: Network partition without quorum
  - Solution: Ensure quorum is maintained (3+ managers)

Slow Writes:
  - Symptom: High write latency
  - Cause: Slow follower or network
  - Solution: Check follower health and network latency

Large Raft Log:
  - Symptom: High memory usage
  - Cause: Snapshot interval too high
  - Solution: Reduce snapshot threshold

# Monitoring

Key metrics to monitor:

Raft Health:
  - raft_leader_changes: Should be low (< 1/hour)
  - raft_log_entries: Should be bounded (snapshots working)
  - raft_commit_time: Should be low (< 10ms)
  - raft_quorum: Should always be true

Manager Health:
  - manager_proposals_total: Write throughput
  - manager_proposal_duration: Write latency
  - manager_fsm_apply_duration: State machine performance

Resource Usage:
  - process_resident_memory: Memory usage
  - go_goroutines: Goroutine count
  - process_cpu_seconds: CPU usage

# See Also

  - pkg/api for gRPC server implementation
  - pkg/storage for state persistence
  - pkg/scheduler for task scheduling logic
  - pkg/reconciler for failure detection
  - docs/concepts/high-availability.md for HA setup
  - docs/raft-tuning.md for Raft configuration
*/
package manager
