/*
Package volume provides volume orchestration and lifecycle management for Warren clusters.

This package implements a pluggable volume driver system for managing persistent
storage across the cluster. The initial implementation includes a local volume
driver that creates directories on worker nodes for persistent data, with support
for node affinity to ensure stateful workloads remain co-located with their data.

# Architecture

Warren's volume system uses a driver-based architecture similar to Docker volumes:

	┌─────────────────────────────────────────────────────────────┐
	│                    Volume Architecture                      │
	└─────┬───────────────────────────────────────────────────────┘
	      │
	      ▼
	┌──────────────────────────────────────────────────────────────┐
	│                      VolumeManager                           │
	│  • Coordinates volume operations across drivers              │
	│  • Routes requests to appropriate driver                     │
	│  • Manages driver registry                                   │
	└────────┬─────────────────────────────────────────────────────┘
	         │
	         ▼
	┌────────────────────┐
	│  VolumeDriver API  │  (Interface)
	│  • Create          │
	│  • Delete          │
	│  • Mount           │
	│  • Unmount         │
	└────────┬───────────┘
	         │
	    ┌────┴────┐
	    ▼         ▼
	┌──────┐  ┌──────┐
	│Local │  │ NFS  │  (Future drivers)
	│Driver│  │Driver│
	└──────┘  └──────┘

## Local Volume Driver

The local driver creates directories on the host filesystem:

	Volume: postgres-data
	├── Driver: local
	├── NodeID: worker-1 (pinned to specific node)
	├── Path: /var/lib/warren/volumes/{volume-id}
	└── Mount: Bind mount into container

This provides simple, high-performance local storage with node affinity.

# Volume Lifecycle

Volumes follow a clear lifecycle from creation to deletion:

 1. CREATE
    ├── User: warren volume create postgres-data
    ├── Manager: Creates volume record in DB
    ├── Scheduler: Assigns volume to node (if not specified)
    └── Worker: Creates directory on host

 2. MOUNT
    ├── Task starts on node with volume
    ├── Worker: Calls driver.Mount(volume)
    ├── Driver: Returns host path
    └── Worker: Bind mounts into container

 3. UNMOUNT
    ├── Task stops
    ├── Worker: Calls driver.Unmount(volume)
    └── Driver: Cleanup (no-op for local driver)

 4. DELETE
    ├── User: warren volume rm postgres-data
    ├── Manager: Checks no tasks using volume
    ├── Worker: Calls driver.Delete(volume)
    └── Driver: Removes directory and data

# Core Components

## VolumeDriver Interface

All volume drivers implement this interface:

	type VolumeDriver interface {
		Create(volume *types.Volume) error
		Delete(volume *types.Volume) error
		Mount(volume *types.Volume) (string, error)
		Unmount(volume *types.Volume) error
		GetPath(volume *types.Volume) string
	}

This abstraction allows plugging in different storage backends (local, NFS,
Ceph, etc.) without changing the core orchestration logic.

## LocalDriver

The local driver implements simple directory-based storage:

	/var/lib/warren/volumes/
	├── vol-abc123/  (Volume 1)
	│   └── data files...
	├── vol-def456/  (Volume 2)
	│   └── data files...
	└── vol-ghi789/  (Volume 3)
	    └── data files...

Each volume gets a unique directory based on its volume ID.

## VolumeManager

The VolumeManager coordinates operations across multiple drivers:

	manager := NewVolumeManager()
	manager.CreateVolume(volume)  // Routes to appropriate driver
	path := manager.MountVolume(volume)  // Returns mount path
	manager.DeleteVolume(volume)  // Cleanup

# Node Affinity

Local volumes are pinned to specific nodes for data locality:

	Volume {
		ID:       "vol-abc123"
		Name:     "postgres-data"
		Driver:   "local"
		NodeID:   "worker-1"  // Pinned to this node
		MountPath: "/var/lib/warren/volumes/vol-abc123"
	}

Once a volume is created on a node:
  - All tasks using that volume must run on the same node
  - Scheduler enforces this constraint automatically
  - Volume cannot be moved (delete and recreate required)

This is critical for stateful workloads - data stays where it's created.

# Usage Examples

## Creating a Volume Manager

	import "github.com/cuemby/warren/pkg/volume"

	// Create volume manager with default local driver
	vm, err := volume.NewVolumeManager()
	if err != nil {
		panic(err)
	}

	// Volume manager is ready for use
	fmt.Println("Volume manager initialized")

## Creating a Local Volume

	// Define volume
	vol := &types.Volume{
		ID:        uuid.New().String(),
		Name:      "postgres-data",
		Driver:    "local",
		NodeID:    "worker-1",  // Assign to specific node
		Labels:    map[string]string{"app": "database"},
		CreatedAt: time.Now(),
	}

	// Create volume (driver creates directory)
	err := vm.CreateVolume(vol)
	if err != nil {
		panic(err)
	}

	fmt.Println("Volume created at:", vol.MountPath)
	// Output: Volume created at: /var/lib/warren/volumes/vol-abc123

## Mounting a Volume into Container

	// Mount volume to get host path
	hostPath, err := vm.MountVolume(vol)
	if err != nil {
		panic(err)
	}

	// Use host path for container bind mount
	containerSpec := &types.ContainerSpec{
		Image: "postgres:15",
		Mounts: []types.Mount{
			{
				Type:   "bind",
				Source: hostPath,  // /var/lib/warren/volumes/vol-abc123
				Target: "/var/lib/postgresql/data",
			},
		},
	}

	// Start container with mounted volume...

## Service with Volume

	// Service definition with volume
	service := &types.Service{
		ID:    "svc-postgres",
		Name:  "postgres",
		Image: "postgres:15",
		Replicas: 1,  // Stateful: single replica
		Volumes: []*types.VolumeMount{
			{
				Source: "postgres-data",  // Volume name
				Target: "/var/lib/postgresql/data",
				Driver: "local",
			},
		},
	}

	// Scheduler will:
	// 1. Check if volume "postgres-data" exists
	// 2. If yes, schedule task on volume's node
	// 3. If no, create volume on selected node
	// 4. Task always runs on same node as volume

## Deleting a Volume

	// Check no tasks using volume first
	tasks, err := manager.ListTasks()
	for _, task := range tasks {
		for _, mount := range task.Mounts {
			if mount.Source == vol.Name {
				panic("Volume in use by task: " + task.ID)
			}
		}
	}

	// Safe to delete
	err = vm.DeleteVolume(vol)
	if err != nil {
		panic(err)
	}

	fmt.Println("Volume deleted")

## Custom Volume Driver (Future)

	// Implement VolumeDriver interface
	type NFSDriver struct {
		nfsServer string
		basePath  string
	}

	func (d *NFSDriver) Create(vol *types.Volume) error {
		// Create NFS export
		return createNFSExport(d.nfsServer, vol.Name)
	}

	func (d *NFSDriver) Mount(vol *types.Volume) (string, error) {
		// Mount NFS share
		mountPoint := "/mnt/nfs/" + vol.Name
		err := mountNFS(d.nfsServer, vol.Name, mountPoint)
		return mountPoint, err
	}

	// Register with volume manager
	vm.RegisterDriver("nfs", NewNFSDriver("nfs.example.com"))

	// Use NFS volumes
	vol := &types.Volume{
		Driver: "nfs",  // Uses custom driver
		Name:   "shared-data",
	}

# Integration Points

## Scheduler Integration

The scheduler enforces volume affinity constraints:

 1. Service has volume requirement
 2. Scheduler calls manager.GetVolumeByName(volumeName)
 3. If volume exists with NodeID:
    • Schedule task on that node ONLY
 4. If volume doesn't exist:
    • Use normal scheduling
    • Worker creates volume on assigned node

This ensures stateful workloads stay pinned to their data.

## Worker Integration

Workers create and mount volumes:

 1. Task assigned with volume requirement
 2. Worker checks if volume exists locally:
    • If yes, mount existing volume
    • If no, create volume first
 3. Worker calls volumeManager.MountVolume(volume)
 4. Worker binds mount into container
 5. Container starts with data access

## Storage Integration

Volume metadata is persisted to BoltDB:

	Bucket: "volumes"
	Key: volume.ID
	Value: {
		ID:        "vol-abc123",
		Name:      "postgres-data",
		Driver:    "local",
		NodeID:    "worker-1",
		MountPath: "/var/lib/warren/volumes/vol-abc123",
		Labels:    {...},
		CreatedAt: "2024-01-15T10:00:00Z",
	}

The actual volume data lives on the node's filesystem, not in the database.

## Container Runtime Integration

Volumes are mounted using containerd's mount API:

	// Create containerd mount spec
	mount := specs.Mount{
		Source:      hostPath,  // From volume driver
		Destination: "/data",   // In container
		Type:        "bind",
		Options:     []string{"rbind", "rw"},
	}

	// Add to container spec
	containerSpec.Mounts = append(containerSpec.Mounts, mount)

# Design Patterns

## Plugin Architecture

The driver interface allows pluggable storage backends:

	Manager → Interface → [Local | NFS | Ceph | ...]

Benefits:
  - Core logic independent of storage backend
  - Easy to add new drivers
  - Drivers can be swapped without code changes

## Lazy Creation

Volumes are created on-demand:

 1. Service references volume "postgres-data"
 2. Scheduler checks if volume exists
 3. If not, worker creates it when task starts
 4. Volume persists after task stops

This simplifies workflow - no need to pre-create volumes.

## Path Abstraction

Drivers abstract the storage path:

  - Local: /var/lib/warren/volumes/{id}
  - NFS: /mnt/nfs/{name}
  - Ceph: /mnt/ceph/{id}

The manager doesn't care about paths - drivers handle all details.

## Immutable Volume IDs

Volume IDs are UUIDs generated at creation and never change:

	ID: "550e8400-e29b-41d4-a716-446655440000"

This ensures:
  - Unique identification across cluster
  - No naming conflicts
  - Stable references even if name changes

# Performance Characteristics

## Local Driver Performance

Local volumes use bind mounts, providing native filesystem performance:

  - Read/Write: Same as underlying filesystem
  - Latency: ~0μs (no overhead)
  - IOPS: Limited by disk, not Warren
  - Throughput: Full disk bandwidth

For SSDs:
  - Read: 500-3000 MB/s
  - Write: 200-2000 MB/s
  - IOPS: 50K-500K

## Volume Operations

Operation latencies:

  - Create: 1-10ms (mkdir + database write)
  - Mount: 1-5ms (path lookup)
  - Unmount: <1ms (no-op for local driver)
  - Delete: 10-100ms (recursive delete + database)

For large volumes (10GB+):
  - Delete can take seconds (many files)
  - Consider async deletion for large datasets

## Memory Usage

Volume operations are very memory-efficient:

  - LocalDriver: ~1KB (just base path)
  - VolumeManager: ~5KB (driver registry)
  - Per-volume metadata: ~500 bytes

Total: ~1MB for 1000 volumes.

# Troubleshooting

## Volume Creation Failures

If volume creation fails:

1. Check permissions:
  - Warren needs write access to /var/lib/warren/volumes
  - Ensure directory exists and is writable
  - Check SELinux/AppArmor policies

2. Check disk space:
  - Verify sufficient space on volume path
  - Use: df -h /var/lib/warren/volumes

3. Check driver errors:
  - Look for driver-specific error messages
  - Verify driver is registered
  - Check driver configuration

## Task Won't Start (Volume Issue)

If task fails to start with volume errors:

1. Verify volume exists:
  - Run: warren volume ls
  - Check volume is on same node as task
  - Verify volume.NodeID matches task.NodeID

2. Check mount path:
  - Verify path exists on node
  - Check path permissions
  - Ensure no other process using path

3. Check container spec:
  - Verify mount source path is correct
  - Check mount target path in container
  - Verify mount options (rw vs ro)

## Volume Deletion Blocked

If volume deletion fails:

1. Check for active tasks:
  - Run: warren service ps --all
  - Look for tasks using this volume
  - Stop services before deleting volume

2. Check for orphaned mounts:
  - Run: mount | grep warren
  - Unmount manually if needed
  - Restart worker to clear stale mounts

3. Force deletion (last resort):
  - Delete volume record from database
  - Manually remove directory on node
  - Risk of data loss!

## Data Not Persisting

If data isn't surviving container restarts:

1. Verify volume is mounted:
  - Check container inspect output
  - Verify mount appears in /proc/mounts
  - Ensure mount type is "bind" not "tmpfs"

2. Check write permissions:
  - Verify container user can write to mount
  - Check directory ownership/permissions
  - May need to chown volume directory

3. Verify volume driver:
  - Ensure using "local" driver (persistent)
  - Check volume wasn't recreated (would be empty)
  - Verify volume.ID hasn't changed

# Monitoring Metrics

Key volume metrics to monitor:

  - Volumes created/deleted per minute
  - Active volumes per node
  - Disk space used by volumes
  - Mount/unmount operations per second
  - Volume operation errors

# Best Practices

1. Volume Naming
  - Use descriptive names: "postgres-data" not "vol1"
  - Include app name: "myapp-logs"
  - Avoid special characters (use hyphens)

2. Stateful Service Design
  - Use replicas=1 for stateful services
  - Pin service to specific node (via volume affinity)
  - Consider backup strategy before deploying
  - Test recovery procedures

3. Volume Lifecycle Management
  - Delete unused volumes to free space
  - Monitor disk usage on volume nodes
  - Implement volume backup policies
  - Document volume contents/purpose

4. Node Planning
  - Dedicate nodes for stateful workloads
  - Use faster disks (SSD) for database volumes
  - Plan for volume growth over time
  - Consider RAID for data redundancy

5. Performance Tuning
  - Use local volumes for best performance
  - Avoid network volumes for latency-sensitive apps
  - Consider volume driver overhead
  - Monitor filesystem performance

# Future Enhancements

Planned volume features:

  - NFS driver (network storage)
  - Volume replication (multi-node)
  - Volume snapshots (backup/restore)
  - Volume resize (grow volumes)
  - Volume migration (move between nodes)
  - Storage quotas (limit volume size)
  - Volume encryption (at-rest encryption)

# See Also

  - pkg/scheduler - Volume affinity scheduling
  - pkg/worker - Volume mounting and management
  - pkg/storage - Volume metadata persistence
  - pkg/types - Volume data structures
  - docs/concepts/storage.md - Storage architecture guide
*/
package volume
