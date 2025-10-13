package worker

import (
	"context"
	"fmt"
	"time"

	specs "github.com/opencontainers/runtime-spec/specs-go"

	"github.com/cuemby/warren/api/proto"
	"github.com/cuemby/warren/pkg/types"
	"github.com/cuemby/warren/pkg/volume"
)

// VolumesHandler manages volume mounting for tasks
type VolumesHandler struct {
	worker        *Worker
	volumeManager *volume.VolumeManager
}

// NewVolumesHandler creates a new volumes handler
func NewVolumesHandler(worker *Worker) (*VolumesHandler, error) {
	vm, err := volume.NewVolumeManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create volume manager: %w", err)
	}

	return &VolumesHandler{
		worker:        worker,
		volumeManager: vm,
	}, nil
}

// PrepareVolumesForTask prepares all volumes for a task and returns mount specs
func (vh *VolumesHandler) PrepareVolumesForTask(task *types.Container) ([]specs.Mount, error) {
	if len(task.Mounts) == 0 {
		return nil, nil
	}

	mounts := make([]specs.Mount, 0, len(task.Mounts))

	for _, mount := range task.Mounts {
		// Fetch volume metadata from manager
		vol, err := vh.fetchVolume(mount.Source)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch volume %s: %w", mount.Source, err)
		}

		// Ensure volume exists on this node
		if err := vh.ensureVolumeExists(vol); err != nil {
			return nil, fmt.Errorf("failed to ensure volume %s: %w", vol.Name, err)
		}

		// Get mount path for volume
		mountPath, err := vh.volumeManager.MountVolume(vol)
		if err != nil {
			return nil, fmt.Errorf("failed to mount volume %s: %w", vol.Name, err)
		}

		// Create mount spec for container
		mountSpec := specs.Mount{
			Source:      mountPath,
			Destination: mount.Target,
			Type:        "bind",
			Options:     []string{"rbind"},
		}

		// Add read-only option if specified
		if mount.ReadOnly {
			mountSpec.Options = append(mountSpec.Options, "ro")
		} else {
			mountSpec.Options = append(mountSpec.Options, "rw")
		}

		mounts = append(mounts, mountSpec)
	}

	return mounts, nil
}

// fetchVolume retrieves volume metadata from the manager
func (vh *VolumesHandler) fetchVolume(volumeName string) (*types.Volume, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := vh.worker.client.GetVolumeByName(ctx, &proto.GetVolumeByNameRequest{
		Name: volumeName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch volume from manager: %w", err)
	}

	// Convert proto volume to types.Volume
	vol := &types.Volume{
		ID:        resp.Volume.Id,
		Name:      resp.Volume.Name,
		Driver:    resp.Volume.Driver,
		NodeID:    resp.Volume.NodeId,
		MountPath: resp.Volume.MountPath,
		Options:   resp.Volume.DriverOpts,
	}

	return vol, nil
}

// ensureVolumeExists creates the volume on this worker if it doesn't exist
func (vh *VolumesHandler) ensureVolumeExists(vol *types.Volume) error {
	// Try to mount - if it fails, create it first
	_, err := vh.volumeManager.MountVolume(vol)
	if err != nil {
		// Volume doesn't exist, create it
		if err := vh.volumeManager.CreateVolume(vol); err != nil {
			return fmt.Errorf("failed to create volume: %w", err)
		}
	}

	return nil
}

// CleanupVolumesForTask unmounts volumes for a task (no-op for local driver)
func (vh *VolumesHandler) CleanupVolumesForTask(task *types.Container) error {
	if len(task.Mounts) == 0 {
		return nil
	}

	for _, mount := range task.Mounts {
		vol := &types.Volume{
			Name:   mount.Source,
			Driver: "local", // Assume local for now
		}

		if err := vh.volumeManager.UnmountVolume(vol); err != nil {
			// Log but don't fail - unmount is best-effort for local driver
			fmt.Printf("Warning: failed to unmount volume %s: %v\n", vol.Name, err)
		}
	}

	return nil
}
