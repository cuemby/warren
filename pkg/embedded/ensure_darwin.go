// +build darwin

package embedded

import (
	"context"
)

// EnsureContainerd ensures containerd is available (starts embedded if not using external)
// On macOS: Uses Lima VM with containerd
func EnsureContainerd(ctx context.Context, dataDir string, useExternal bool) (*ContainerdManager, error) {
	// On macOS, we need to use Lima VM (unless user wants external containerd)
	if !useExternal {
		return EnsureContainerdMacOS(ctx, dataDir)
	}

	// Using external containerd
	manager, err := NewContainerdManager(dataDir, useExternal)
	if err != nil {
		return nil, err
	}

	if err := manager.Start(ctx); err != nil {
		return nil, err
	}

	return manager, nil
}
