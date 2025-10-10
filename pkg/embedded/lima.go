// +build darwin

package embedded

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/lima-vm/lima/pkg/instance"
	"github.com/lima-vm/lima/pkg/limayaml"
	"github.com/lima-vm/lima/pkg/store"
	"github.com/rs/zerolog"
)

const (
	// WarrenLimaInstanceName is the name of the Lima VM instance
	WarrenLimaInstanceName = "warren"

	// LimaContainerdSocket is the path to containerd socket inside Lima VM
	LimaContainerdSocket = "/run/containerd/containerd.sock"
)

// LimaManager manages the Lima VM for Warren on macOS
type LimaManager struct {
	instanceName string
	instance     *store.Instance
	dataDir      string
	logger       zerolog.Logger
}

// NewLimaManager creates a new Lima VM manager
func NewLimaManager(dataDir string) (*LimaManager, error) {
	logger := zerolog.New(os.Stdout).With().
		Str("component", "lima-vm").
		Timestamp().
		Logger()

	return &LimaManager{
		instanceName: WarrenLimaInstanceName,
		dataDir:      dataDir,
		logger:       logger,
	}, nil
}

// Start starts the Lima VM with containerd
func (lm *LimaManager) Start(ctx context.Context) error {
	lm.logger.Info().Msg("Starting Lima VM for Warren")

	// Check if Lima is installed
	if !lm.isLimaInstalled() {
		return fmt.Errorf("Lima is not installed. Install with: brew install lima")
	}

	// Check if instance already exists
	inst, err := store.Inspect(lm.instanceName)
	if err == nil {
		// Instance exists
		lm.instance = inst
		lm.logger.Info().Msgf("Lima instance '%s' already exists", lm.instanceName)

		// Check if it's running
		if inst.Status == store.StatusRunning {
			lm.logger.Info().Msg("Lima VM already running")
			return nil
		}

		// Start existing instance
		lm.logger.Info().Msg("Starting existing Lima instance")
		if err := instance.Start(ctx, inst, "", false); err != nil {
			return fmt.Errorf("failed to start Lima instance: %w", err)
		}

		// Wait for it to be ready
		return lm.waitForReady(ctx)
	}

	// Instance doesn't exist, create it
	lm.logger.Info().Msg("Creating new Lima instance for Warren")
	if err := lm.createInstance(ctx); err != nil {
		return fmt.Errorf("failed to create Lima instance: %w", err)
	}

	// Get the created instance
	inst, err = store.Inspect(lm.instanceName)
	if err != nil {
		return fmt.Errorf("failed to inspect created instance: %w", err)
	}
	lm.instance = inst

	// Start the instance
	lm.logger.Info().Msg("Starting Lima instance")
	if err := instance.Start(ctx, inst, "", false); err != nil {
		return fmt.Errorf("failed to start Lima instance: %w", err)
	}

	// Wait for it to be ready
	if err := lm.waitForReady(ctx); err != nil {
		return fmt.Errorf("Lima VM failed to become ready: %w", err)
	}

	lm.logger.Info().Msg("Lima VM started successfully")
	return nil
}

// Stop stops the Lima VM
func (lm *LimaManager) Stop(ctx context.Context) error {
	if lm.instance == nil {
		return nil
	}

	lm.logger.Info().Msg("Stopping Lima VM")

	// Stop gracefully
	if err := instance.StopGracefully(ctx, lm.instance, false); err != nil {
		lm.logger.Warn().Msgf("Graceful stop failed: %v, forcing stop", err)
		instance.StopForcibly(lm.instance)
	}

	lm.logger.Info().Msg("Lima VM stopped")
	return nil
}

// GetSocketPath returns the path to containerd socket
// For Lima, we need to use lima command to access the socket
func (lm *LimaManager) GetSocketPath() string {
	// Lima exposes the socket via a Unix socket on the host
	// The actual path is in the instance directory
	if lm.instance == nil {
		return ""
	}

	// Lima creates a socket at: $LIMA_HOME/<instance>/sock/containerd.sock
	limaHome := os.Getenv("LIMA_HOME")
	if limaHome == "" {
		home, _ := os.UserHomeDir()
		limaHome = filepath.Join(home, ".lima")
	}

	socketPath := filepath.Join(limaHome, lm.instanceName, "sock", "containerd.sock")
	return socketPath
}

// createInstance creates a new Lima instance with Warren configuration
func (lm *LimaManager) createInstance(ctx context.Context) error {
	// Create Lima configuration
	config := lm.createLimaConfig()

	// Marshal config to YAML (stream=false for single document)
	configYAML, err := limayaml.Marshal(&config, false)
	if err != nil {
		return fmt.Errorf("failed to marshal Lima config: %w", err)
	}

	// Create the instance
	_, err = instance.Create(ctx, lm.instanceName, configYAML, false)
	if err != nil {
		return fmt.Errorf("failed to create instance: %w", err)
	}

	return nil
}

// createLimaConfig creates a Lima configuration optimized for Warren
func (lm *LimaManager) createLimaConfig() limayaml.LimaYAML {
	// Determine architecture
	arch := limayaml.AARCH64
	if runtime.GOARCH == "amd64" {
		arch = limayaml.X8664
	}

	// Memory and CPU configuration
	cpus := 2
	memory := "2GiB"
	disk := "20GiB"

	// Create configuration
	config := limayaml.LimaYAML{
		Arch: &arch,
		CPUs: &cpus,
		Memory: &memory,
		Disk: &disk,

		// Use Alpine Linux for minimal footprint
		Images: []limayaml.Image{
			{
				File: limayaml.File{
					Location: "https://dl-cdn.alpinelinux.org/alpine/v3.19/releases/cloud/alpine-virt-3.19.0-aarch64.iso",
					Arch:     limayaml.AARCH64,
				},
			},
			{
				File: limayaml.File{
					Location: "https://dl-cdn.alpinelinux.org/alpine/v3.19/releases/cloud/alpine-virt-3.19.0-x86_64.iso",
					Arch:     limayaml.X8664,
				},
			},
		},

		// Containerd configuration
		Containerd: limayaml.Containerd{
			System: ptrBool(true), // Install containerd as system service
		},

		// Mount Warren data directory
		Mounts: []limayaml.Mount{
			{
				Location: lm.dataDir,
				Writable: ptrBool(true),
			},
		},

		// Provision scripts to setup containerd
		Provision: []limayaml.Provision{
			{
				Mode:   limayaml.ProvisionModeSystem,
				Script: "#!/bin/sh\nset -eux -o pipefail\n# Install containerd if not present\nif ! command -v containerd > /dev/null; then\n  apk add containerd\nfi\n# Start containerd\nrc-update add containerd default\nrc-service containerd start || true",
			},
		},

		// Message to show when instance starts
		Message: "Warren Lima VM - Ready to run containers",
	}

	return config
}

// waitForReady waits for Lima VM to be ready
func (lm *LimaManager) waitForReady(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for Lima VM to be ready")
		case <-ticker.C:
			// Check instance status
			inst, err := store.Inspect(lm.instanceName)
			if err != nil {
				lm.logger.Debug().Msgf("Failed to inspect instance: %v", err)
				continue
			}

			if inst.Status == store.StatusRunning {
				lm.logger.Info().Msg("Lima VM is running")
				// Check if containerd socket is available
				socketPath := lm.GetSocketPath()
				if _, err := os.Stat(socketPath); err == nil {
					lm.logger.Info().Msgf("Containerd socket ready at %s", socketPath)
					return nil
				}
				lm.logger.Debug().Msgf("Waiting for containerd socket at %s", socketPath)
			}
		}
	}
}

// isLimaInstalled checks if Lima is installed on the system
func (lm *LimaManager) isLimaInstalled() bool {
	_, err := exec.LookPath("limactl")
	return err == nil
}

// ptrBool returns a pointer to a bool value
func ptrBool(b bool) *bool {
	return &b
}

// EnsureLima starts Lima VM and returns the containerd socket path
func EnsureLima(ctx context.Context, dataDir string) (*LimaManager, error) {
	manager, err := NewLimaManager(dataDir)
	if err != nil {
		return nil, err
	}

	if err := manager.Start(ctx); err != nil {
		return nil, err
	}

	return manager, nil
}
