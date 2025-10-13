package volume

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cuemby/warren/pkg/types"
)

func TestNewLocalDriver(t *testing.T) {
	tmpDir := t.TempDir()

	driver, err := NewLocalDriver(tmpDir)
	if err != nil {
		t.Fatalf("NewLocalDriver() error = %v", err)
	}

	if driver == nil {
		t.Fatal("NewLocalDriver() returned nil driver")
	}

	if driver.basePath != tmpDir {
		t.Errorf("basePath = %v, want %v", driver.basePath, tmpDir)
	}

	// Verify base directory was created
	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		t.Error("Base directory was not created")
	}
}

func TestLocalDriver_Create(t *testing.T) {
	tmpDir := t.TempDir()
	driver, _ := NewLocalDriver(tmpDir)

	volume := &types.Volume{
		ID:     "test-volume",
		Name:   "test",
		Driver: "local",
	}

	err := driver.Create(volume)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Verify volume directory was created
	volumePath := driver.GetPath(volume)
	if _, err := os.Stat(volumePath); os.IsNotExist(err) {
		t.Errorf("Volume directory was not created at %s", volumePath)
	}

	// Verify MountPath was set
	if volume.MountPath != volumePath {
		t.Errorf("MountPath = %v, want %v", volume.MountPath, volumePath)
	}
}

func TestLocalDriver_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	driver, _ := NewLocalDriver(tmpDir)

	volume := &types.Volume{
		ID:     "test-volume",
		Name:   "test",
		Driver: "local",
	}

	// Create volume first
	if err := driver.Create(volume); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	volumePath := driver.GetPath(volume)

	// Create a file in the volume
	testFile := filepath.Join(volumePath, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Delete the volume
	err := driver.Delete(volume)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify volume directory was deleted
	if _, err := os.Stat(volumePath); !os.IsNotExist(err) {
		t.Error("Volume directory still exists after delete")
	}
}

func TestLocalDriver_Delete_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	driver, _ := NewLocalDriver(tmpDir)

	volume := &types.Volume{
		ID:     "nonexistent",
		Name:   "test",
		Driver: "local",
	}

	// Delete non-existent volume should not error
	err := driver.Delete(volume)
	if err != nil {
		t.Errorf("Delete() on non-existent volume error = %v, want nil", err)
	}
}

func TestLocalDriver_Mount(t *testing.T) {
	tmpDir := t.TempDir()
	driver, _ := NewLocalDriver(tmpDir)

	volume := &types.Volume{
		ID:     "test-volume",
		Name:   "test",
		Driver: "local",
	}

	// Create volume first
	if err := driver.Create(volume); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Mount the volume
	mountPath, err := driver.Mount(volume)
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	expectedPath := driver.GetPath(volume)
	if mountPath != expectedPath {
		t.Errorf("Mount() path = %v, want %v", mountPath, expectedPath)
	}

	// Verify path exists
	if _, err := os.Stat(mountPath); os.IsNotExist(err) {
		t.Errorf("Mount path does not exist: %s", mountPath)
	}
}

func TestLocalDriver_Mount_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	driver, _ := NewLocalDriver(tmpDir)

	volume := &types.Volume{
		ID:     "nonexistent",
		Name:   "test",
		Driver: "local",
	}

	// Mount non-existent volume should error
	_, err := driver.Mount(volume)
	if err == nil {
		t.Error("Mount() on non-existent volume should return error")
	}
}

func TestLocalDriver_Unmount(t *testing.T) {
	tmpDir := t.TempDir()
	driver, _ := NewLocalDriver(tmpDir)

	volume := &types.Volume{
		ID:     "test-volume",
		Name:   "test",
		Driver: "local",
	}

	// Unmount should not error (no-op for local driver)
	err := driver.Unmount(volume)
	if err != nil {
		t.Errorf("Unmount() error = %v, want nil", err)
	}
}

func TestVolumeManager_CreateAndDelete(t *testing.T) {
	tmpDir := t.TempDir()
	localDriver, _ := NewLocalDriver(tmpDir)

	vm := &VolumeManager{
		drivers: map[string]VolumeDriver{
			"local": localDriver,
		},
	}

	volume := &types.Volume{
		ID:     "test-volume",
		Name:   "test",
		Driver: "local",
	}

	// Create volume
	if err := vm.CreateVolume(volume); err != nil {
		t.Fatalf("CreateVolume() error = %v", err)
	}

	// Verify volume was created
	volumePath := localDriver.GetPath(volume)
	if _, err := os.Stat(volumePath); os.IsNotExist(err) {
		t.Error("Volume was not created")
	}

	// Delete volume
	if err := vm.DeleteVolume(volume); err != nil {
		t.Fatalf("DeleteVolume() error = %v", err)
	}

	// Verify volume was deleted
	if _, err := os.Stat(volumePath); !os.IsNotExist(err) {
		t.Error("Volume was not deleted")
	}
}

func TestVolumeManager_UnknownDriver(t *testing.T) {
	tmpDir := t.TempDir()
	localDriver, _ := NewLocalDriver(tmpDir)

	vm := &VolumeManager{
		drivers: map[string]VolumeDriver{
			"local": localDriver,
		},
	}

	volume := &types.Volume{
		ID:     "test-volume",
		Name:   "test",
		Driver: "unknown-driver",
	}

	err := vm.CreateVolume(volume)
	if err == nil {
		t.Error("CreateVolume() with unknown driver should return error")
	}
}

func TestVolumeManager_MountVolume(t *testing.T) {
	tmpDir := t.TempDir()
	localDriver, _ := NewLocalDriver(tmpDir)

	vm := &VolumeManager{
		drivers: map[string]VolumeDriver{
			"local": localDriver,
		},
	}

	volume := &types.Volume{
		ID:     "test-volume",
		Name:   "test",
		Driver: "local",
	}

	// Create volume
	if err := vm.CreateVolume(volume); err != nil {
		t.Fatalf("CreateVolume() error = %v", err)
	}

	// Mount volume
	mountPath, err := vm.MountVolume(volume)
	if err != nil {
		t.Fatalf("MountVolume() error = %v", err)
	}

	if mountPath == "" {
		t.Error("MountVolume() returned empty path")
	}

	// Verify mount path exists
	if _, err := os.Stat(mountPath); os.IsNotExist(err) {
		t.Errorf("Mount path does not exist: %s", mountPath)
	}

	// Cleanup
	vm.DeleteVolume(volume)
}
