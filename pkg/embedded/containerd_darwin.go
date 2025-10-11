// +build darwin

package embedded

import (
	"context"
	"fmt"
	"os"

	"github.com/rs/zerolog"
)

// EnsureContainerdMacOS starts Lima VM with containerd on macOS
func EnsureContainerdMacOS(ctx context.Context, dataDir string) (*ContainerdManager, error) {
	logger := zerolog.New(os.Stdout).With().
		Str("component", "lima-containerd").
		Timestamp().
		Logger()

	logger.Info().Msg("Starting Lima VM for containerd on macOS")

	// Start Lima VM with containerd
	limaManager, err := EnsureLima(ctx, dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to start Lima VM: %w", err)
	}

	// Get containerd socket path from Lima VM
	socketPath := limaManager.GetSocketPath()
	if socketPath == "" {
		return nil, fmt.Errorf("failed to get containerd socket path from Lima VM")
	}

	logger.Info().Msgf("Using containerd socket at %s", socketPath)

	// Create containerd manager using Lima socket
	manager := &ContainerdManager{
		dataDir:     dataDir,
		socketPath:  socketPath,
		useExternal: false, // We're managing the VM, so it's not "external"
		limaManager: limaManager, // Store reference for lifecycle management
		logger:      logger,
	}

	// Note: We don't call manager.Start() here because Lima already started containerd
	// The VM is running and containerd is ready

	return manager, nil
}
