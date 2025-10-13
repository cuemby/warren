/*
Package storage provides BoltDB-backed state persistence for Warren's cluster data.

The storage package implements the Store interface using BoltDB as the underlying
database, providing ACID transactions for cluster state including nodes, services,
tasks, secrets, volumes, certificates, and ingresses. All data is serialized as
JSON and stored in separate buckets for efficient querying and isolation.

# Architecture

Warren uses BoltDB (bbolt) for embedded, transactional storage with zero external
dependencies:

	┌──────────────────── BOLTDB STORAGE ──────────────────────┐
	│                                                            │
	│  ┌────────────────────────────────────────────┐          │
	│  │            BoltStore                        │          │
	│  │  - File: <dataDir>/warren.db                │          │
	│  │  - Format: B+tree with MVCC                 │          │
	│  │  - Transactions: ACID with fsync            │          │
	│  └──────────────────┬─────────────────────────┘          │
	│                     │                                      │
	│  ┌──────────────────▼─────────────────────────┐          │
	│  │              Bucket Structure                │          │
	│  │  ┌────────────────────────────┐             │          │
	│  │  │ nodes          (Node ID)   │             │          │
	│  │  │ services       (Service ID)│             │          │
	│  │  │ tasks          (Task ID)   │             │          │
	│  │  │ secrets        (Secret ID) │             │          │
	│  │  │ volumes        (Volume ID) │             │          │
	│  │  │ networks       (Network ID)│             │          │
	│  │  │ ca             (fixed key) │             │          │
	│  │  │ ingresses      (Ingress ID)│             │          │
	│  │  │ tls_certificates (Cert ID) │             │          │
	│  │  └────────────────────────────┘             │          │
	│  └──────────────────┬─────────────────────────┘          │
	│                     │                                      │
	│  ┌──────────────────▼─────────────────────────┐          │
	│  │        Transaction Management                │          │
	│  │  - Read: db.View() - Concurrent reads       │          │
	│  │  - Write: db.Update() - Serialized writes   │          │
	│  │  - Rollback: Automatic on error             │          │
	│  │  - Commit: Automatic on success + fsync     │          │
	│  └──────────────────┬─────────────────────────┘          │
	│                     │                                      │
	│  ┌──────────────────▼─────────────────────────┐          │
	│  │          JSON Serialization                  │          │
	│  │  - Marshal: Go struct → JSON bytes          │          │
	│  │  - Unmarshal: JSON bytes → Go struct        │          │
	│  │  - Validation: Type safety via Go types     │          │
	│  └────────────────────────────────────────────┘           │
	│                                                            │
	│  ┌────────────────────────────────────────────┐          │
	│  │           BoltDB File                        │          │
	│  │  - Copy-on-write B+tree                      │          │
	│  │  - Page size: 4KB                            │          │
	│  │  - mmap for reads                            │          │
	│  │  - Atomic writes with fsync                  │          │
	│  └────────────────────────────────────────────┘           │
	└────────────────────────────────────────────────────────┘

# Core Components

BoltStore:
  - Implements Store interface using BoltDB
  - Single database file per manager node
  - Automatic bucket creation on initialization
  - Thread-safe via BoltDB's transaction model

Buckets:
  - nodes: Worker and manager node registrations
  - services: Service definitions and configurations
  - tasks: Task instances and their current state
  - secrets: Encrypted secret data
  - volumes: Persistent volume metadata
  - networks: Overlay network configurations
  - ca: Certificate authority data (single entry)
  - ingresses: HTTP/HTTPS ingress rules
  - tls_certificates: TLS certificate data for ingress

Transaction Model:
  - Read transactions: db.View() - Concurrent, consistent snapshots
  - Write transactions: db.Update() - Serialized, atomic commits
  - Isolation: Snapshot isolation (MVCC)
  - Durability: fsync on commit ensures crash recovery

# CRUD Operations

Node Operations:

Create Node:
  - Insert node metadata with ID as key
  - JSON serialization of Node struct
  - Atomic commit via transaction

Get Node:
  - Key lookup by node ID
  - Unmarshal JSON to Node struct
  - Returns error if not found

List Nodes:
  - Cursor iteration over nodes bucket
  - Deserialize all entries to []*Node
  - Empty slice if no nodes

Update Node:
  - Upsert operation (same as Create)
  - Overwrites existing key with new value
  - Atomic replacement

Delete Node:
  - Remove key from bucket
  - No error if key doesn't exist (idempotent)

Service Operations:

Create Service:
  - Store service with ID as key
  - Includes replicas, image, deploy strategy
  - Links to secrets, volumes, networks

Get Service:
  - Direct key lookup by service ID
  - Returns complete service definition

Get Service By Name:
  - Cursor scan to find matching name
  - Returns first match (names should be unique)
  - Error if not found

List Services:
  - Full bucket scan and deserialization
  - Used by scheduler and reconciler
  - Typically < 1000 services per cluster

Update Service:
  - Upsert with updated fields
  - UpdatedAt timestamp managed by caller
  - Triggers reconciliation loop

Delete Service:
  - Remove service entry
  - Tasks must be deleted separately
  - Consider cascade delete (future)

Task Operations:

Create Task:
  - Store task with ID as key
  - Includes service ID, node ID, state
  - Resource requirements and mounts

List Tasks:
  - Full scan for all tasks
  - Used by reconciler for global state

List Tasks By Service:
  - Filter tasks by service ID
  - Returns tasks for specific service
  - Used during rolling updates

List Tasks By Node:
  - Filter tasks by node ID
  - Returns tasks assigned to node
  - Used by worker agent on startup

Update Task:
  - Update state, timestamps, health
  - Called frequently (state transitions)
  - High write throughput operation

Delete Task:
  - Remove completed/failed tasks
  - Cleanup during scaling down
  - Idempotent operation

# Usage

Creating a Store:

	store, err := storage.NewBoltStore("/var/lib/warren/manager-1")
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

Node Operations:

	// Create node
	node := &types.Node{
		ID:       "node-abc123",
		Hostname: "worker-01",
		Role:     types.RoleWorker,
		Status:   types.NodeStatusReady,
		Resources: &types.NodeResources{
			CPUCores:    4,
			MemoryBytes: 8 * 1024 * 1024 * 1024, // 8GB
		},
	}
	err := store.CreateNode(node)

	// Get node
	node, err := store.GetNode("node-abc123")

	// List all nodes
	nodes, err := store.ListNodes()

	// Update node
	node.Status = types.NodeStatusDown
	err = store.UpdateNode(node)

	// Delete node
	err = store.DeleteNode("node-abc123")

Service Operations:

	// Create service
	service := &types.Service{
		ID:       "service-xyz789",
		Name:     "web",
		Image:    "nginx:latest",
		Replicas: 3,
		DeployStrategy: types.DeployStrategyRolling,
	}
	err := store.CreateService(service)

	// Get by ID
	service, err := store.GetService("service-xyz789")

	// Get by name
	service, err := store.GetServiceByName("web")

	// List all services
	services, err := store.ListServices()

	// Update service
	service.Replicas = 5
	err = store.UpdateService(service)

	// Delete service
	err = store.DeleteService("service-xyz789")

Task Operations:

	// Create task
	task := &types.Task{
		ID:          "task-def456",
		ServiceID:   "service-xyz789",
		NodeID:      "node-abc123",
		DesiredState: types.TaskStateRunning,
		ActualState:  types.TaskStatePending,
	}
	err := store.CreateTask(task)

	// List tasks by service
	tasks, err := store.ListTasksByService("service-xyz789")

	// List tasks by node
	tasks, err := store.ListTasksByNode("node-abc123")

	// Update task state
	task.ActualState = types.TaskStateRunning
	err = store.UpdateTask(task)

Secret Operations:

	// Create secret (already encrypted)
	secret := &types.Secret{
		ID:   "secret-ghi789",
		Name: "db-password",
		Data: encryptedData, // AES-256-GCM encrypted
	}
	err := store.CreateSecret(secret)

	// Get secret
	secret, err := store.GetSecret("secret-ghi789")

	// Get by name
	secret, err := store.GetSecretByName("db-password")

	// List secrets
	secrets, err := store.ListSecrets()

	// Delete secret
	err = store.DeleteSecret("secret-ghi789")

Certificate Authority:

	// Save CA certificate and key
	caData := []byte("PEM-encoded CA cert and key")
	err := store.SaveCA(caData)

	// Get CA data
	caData, err := store.GetCA()

# Integration Points

This package integrates with:

  - pkg/manager: Raft FSM reads/writes cluster state
  - pkg/scheduler: Reads nodes and services for placement
  - pkg/reconciler: Reads tasks and services for reconciliation
  - pkg/security: Stores encrypted secrets and CA data
  - pkg/types: All entity definitions

# Design Patterns

Upsert Pattern:
  - Create and Update use same method (db.Put)
  - No separate "exists" check needed
  - Simplifies API and caller code
  - Atomic replacement

Idempotent Deletes:
  - Delete returns no error if key doesn't exist
  - Safe to call multiple times
  - Simplifies cleanup code

Cursor Iteration:
  - ForEach pattern for full bucket scans
  - Memory efficient (streaming)
  - Consistent snapshot during iteration

Error Wrapping:
  - All errors wrapped with context: fmt.Errorf("op failed: %w", err)
  - Preserves original error for inspection
  - Provides operation context in logs

Filter Pattern:
  - List all, filter in memory (ListTasksByService)
  - Simple implementation for small datasets
  - Future: Secondary indexes for performance

# Performance Characteristics

Read Operations:
  - Get by key: O(log n) via B+tree, typically < 1ms
  - List all: O(n) full scan, ~1ms per 1000 entries
  - Filter by field: O(n) scan with predicate, same as List
  - Concurrent reads: Supported via MVCC snapshots

Write Operations:
  - Insert/Update: O(log n) for key, ~1-5ms with fsync
  - Delete: O(log n) for key, ~1-5ms with fsync
  - Batch writes: Single transaction, amortized cost
  - Serialized: Only one writer at a time (BoltDB limitation)

Database File Size:
  - Empty: 32KB (header + initial pages)
  - Small cluster (10 nodes, 20 services): ~1MB
  - Medium cluster (100 nodes, 200 services): ~10MB
  - Large cluster (500 nodes, 1000 services): ~50MB
  - Growth: Linear with entity count + history

Memory Usage:
  - mmap: Database file mapped to memory
  - Read-only pages: Shared across processes
  - Write buffer: ~4MB per transaction
  - Page cache: OS manages (warm frequently accessed pages)

Transaction Latency:
  - Read transaction: < 100µs (memory access)
  - Write transaction: 1-5ms (fsync to disk)
  - Under load: May queue (single writer)

# Troubleshooting

Common Issues:

Database Locked:
  - Symptom: "database is locked" error
  - Cause: Another process has exclusive lock
  - Solution: Ensure only one manager accesses database
  - Check: No dangling processes holding file

Database Corruption:
  - Symptom: "invalid database" or checksum errors
  - Cause: Unclean shutdown, disk failure, bug
  - Solution: Restore from Raft snapshot backup
  - Prevention: Use fsync (enabled by default)

Slow Writes:
  - Symptom: High latency on Create/Update operations
  - Cause: Slow disk, large database, fragmentation
  - Check: fsync latency, disk I/O wait
  - Solution: Use SSD, compact database (future)

Memory Growth:
  - Symptom: Manager memory usage grows over time
  - Cause: mmap keeps pages in cache
  - Check: OS page cache usage
  - Solution: Normal behavior, OS manages eviction

Large Database File:
  - Symptom: Database file grows large over time
  - Cause: No compaction, deleted keys leave space
  - Check: Compare file size to expected data size
  - Solution: Manual compact (future) or backup/restore

# Monitoring

Key metrics to monitor:

Database Operations:
  - storage_read_duration: Time for read transactions
  - storage_write_duration: Time for write transactions
  - storage_operations_total: Count by operation type
  - storage_errors_total: Failed operations

Database Health:
  - storage_db_size_bytes: Database file size
  - storage_db_open: Database connection status (1=open)
  - storage_tx_duration: Transaction latency (p50, p95, p99)

Entity Counts:
  - storage_nodes_total: Number of nodes stored
  - storage_services_total: Number of services
  - storage_tasks_total: Number of tasks
  - storage_secrets_total: Number of secrets

# Data Integrity

Transaction Guarantees:
  - Atomicity: All-or-nothing commits
  - Consistency: JSON validation before commit
  - Isolation: Snapshot reads, serialized writes
  - Durability: fsync ensures crash recovery

Backup and Restore:
  - Database is single file (easy to copy)
  - Backup: Copy file while database is closed OR use db.View()
  - Restore: Replace file and restart manager
  - Raft handles replication across managers

Data Migration:
  - Schema changes handled via JSON flexibility
  - New fields: Add with omitempty tag (backward compatible)
  - Remove fields: Ignored during unmarshal
  - Major changes: Implement migration in NewBoltStore

# Security

Encryption at Rest:
  - Database file not encrypted by default
  - Recommendation: Use disk encryption (LUKS, dm-crypt)
  - Secrets already encrypted before storage (AES-256-GCM)
  - Future: Full database encryption option

File Permissions:
  - Database file: 0600 (owner read/write only)
  - Directory: 0700 (owner full access only)
  - Prevents unprivileged access to cluster state
  - Root or warren user only

Access Control:
  - No authentication within database
  - Rely on OS file permissions
  - Manager API provides authorization layer
  - Direct database access only for recovery

# See Also

  - pkg/manager for Raft FSM integration
  - pkg/types for all entity definitions
  - pkg/scheduler for read-heavy workloads
  - pkg/reconciler for state reconciliation
  - BoltDB documentation: https://github.com/etcd-io/bbolt
  - ACID properties: https://en.wikipedia.org/wiki/ACID
*/
package storage
