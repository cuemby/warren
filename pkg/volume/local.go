package volume

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cuemby/warren/pkg/types"
)

const (
	// DefaultVolumesPath is the base directory for local volumes
	DefaultVolumesPath = "/var/lib/warren/volumes"
)

// VolumeDriver defines the interface for volume drivers
type VolumeDriver interface {
	// Create creates a new volume
	Create(volume *types.Volume) error

	// Delete removes a volume
	Delete(volume *types.Volume) error

	// Mount returns the host path for mounting to containers
	Mount(volume *types.Volume) (string, error)

	// Unmount performs cleanup after unmounting
	Unmount(volume *types.Volume) error

	// GetPath returns the host path for a volume
	GetPath(volume *types.Volume) string
}

// LocalDriver implements a simple local volume driver
type LocalDriver struct {
	basePath string
}

// NewLocalDriver creates a new local volume driver
func NewLocalDriver(basePath string) (*LocalDriver, error) {
	if basePath == "" {
		basePath = DefaultVolumesPath
	}

	// Ensure base directory exists
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create volumes directory: %w", err)
	}

	return &LocalDriver{
		basePath: basePath,
	}, nil
}

// Create creates a new local volume directory
func (d *LocalDriver) Create(volume *types.Volume) error {
	volumePath := d.GetPath(volume)

	// Create the volume directory
	if err := os.MkdirAll(volumePath, 0755); err != nil {
		return fmt.Errorf("failed to create volume directory: %w", err)
	}

	// Update volume with actual mount path
	volume.MountPath = volumePath

	return nil
}

// Delete removes a local volume directory
func (d *LocalDriver) Delete(volume *types.Volume) error {
	volumePath := d.GetPath(volume)

	// Check if volume exists
	if _, err := os.Stat(volumePath); os.IsNotExist(err) {
		return nil // Already deleted
	}

	// Remove the volume directory and all contents
	if err := os.RemoveAll(volumePath); err != nil {
		return fmt.Errorf("failed to delete volume directory: %w", err)
	}

	return nil
}

// Mount returns the host path for bind mounting to containers
func (d *LocalDriver) Mount(volume *types.Volume) (string, error) {
	volumePath := d.GetPath(volume)

	// Verify volume exists
	if _, err := os.Stat(volumePath); os.IsNotExist(err) {
		return "", fmt.Errorf("volume directory does not exist: %s", volumePath)
	}

	return volumePath, nil
}

// Unmount performs cleanup (no-op for local driver)
func (d *LocalDriver) Unmount(volume *types.Volume) error {
	// Local driver doesn't need to do anything on unmount
	// The directory stays on disk
	return nil
}

// GetPath returns the host path for a volume
func (d *LocalDriver) GetPath(volume *types.Volume) string {
	return filepath.Join(d.basePath, volume.ID)
}

// VolumeManager manages volume operations
type VolumeManager struct {
	drivers map[string]VolumeDriver
}

// NewVolumeManager creates a new volume manager
func NewVolumeManager() (*VolumeManager, error) {
	// Initialize with local driver
	localDriver, err := NewLocalDriver("")
	if err != nil {
		return nil, fmt.Errorf("failed to create local driver: %w", err)
	}

	return &VolumeManager{
		drivers: map[string]VolumeDriver{
			"local": localDriver,
		},
	}, nil
}

// GetDriver returns the driver for a volume
func (vm *VolumeManager) GetDriver(driverName string) (VolumeDriver, error) {
	driver, ok := vm.drivers[driverName]
	if !ok {
		return nil, fmt.Errorf("unknown volume driver: %s", driverName)
	}
	return driver, nil
}

// CreateVolume creates a volume using the appropriate driver
func (vm *VolumeManager) CreateVolume(volume *types.Volume) error {
	driver, err := vm.GetDriver(volume.Driver)
	if err != nil {
		return err
	}

	return driver.Create(volume)
}

// DeleteVolume deletes a volume using the appropriate driver
func (vm *VolumeManager) DeleteVolume(volume *types.Volume) error {
	driver, err := vm.GetDriver(volume.Driver)
	if err != nil {
		return err
	}

	return driver.Delete(volume)
}

// MountVolume returns the mount path for a volume
func (vm *VolumeManager) MountVolume(volume *types.Volume) (string, error) {
	driver, err := vm.GetDriver(volume.Driver)
	if err != nil {
		return "", err
	}

	return driver.Mount(volume)
}

// UnmountVolume performs cleanup after unmounting
func (vm *VolumeManager) UnmountVolume(volume *types.Volume) error {
	driver, err := vm.GetDriver(volume.Driver)
	if err != nil {
		return err
	}

	return driver.Unmount(volume)
}
