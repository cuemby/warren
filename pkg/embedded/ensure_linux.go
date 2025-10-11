// +build linux

package embedded

import (
	"context"
)

// EnsureContainerd ensures containerd is available (starts embedded if not using external)
// On Linux: Uses embedded containerd binary or external containerd
func EnsureContainerd(ctx context.Context, dataDir string, useExternal bool) (*ContainerdManager, error) {
	manager, err := NewContainerdManager(dataDir, useExternal)
	if err != nil {
		return nil, err
	}

	if err := manager.Start(ctx); err != nil {
		return nil, err
	}

	return manager, nil
}
