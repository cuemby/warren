//go:build darwin
// +build darwin

package embedded

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

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

	// Check if instance exists
	exists, err := lm.instanceExists()
	if err != nil {
		return fmt.Errorf("failed to check Lima instance: %w", err)
	}

	if !exists {
		// Create new Lima instance
		lm.logger.Info().Msg("Creating new Lima instance for Warren")
		if err := lm.createInstance(ctx); err != nil {
			return fmt.Errorf("failed to create Lima instance: %w", err)
		}
	}

	// Start the instance if not running
	running, err := lm.isRunning()
	if err != nil {
		return fmt.Errorf("failed to check instance status: %w", err)
	}

	if !running {
		lm.logger.Info().Msg("Starting Lima instance")
		if err := lm.startInstance(ctx); err != nil {
			return fmt.Errorf("failed to start Lima instance: %w", err)
		}
	} else {
		lm.logger.Info().Msg("Lima instance already running")
	}

	// Wait for containerd to be ready
	return lm.waitForReady(ctx)
}

// Stop stops the Lima VM gracefully
func (lm *LimaManager) Stop(ctx context.Context) error {
	lm.logger.Info().Msg("Stopping Lima VM")

	// Check if instance exists and is running
	running, err := lm.isRunning()
	if err != nil || !running {
		lm.logger.Info().Msg("Lima instance not running, nothing to stop")
		return nil
	}

	// Stop the instance
	cmd := exec.CommandContext(ctx, "limactl", "stop", lm.instanceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		lm.logger.Error().Msgf("Failed to stop Lima instance: %s", string(output))
		return fmt.Errorf("failed to stop Lima instance: %w", err)
	}

	lm.logger.Info().Msg("Lima VM stopped successfully")
	return nil
}

// GetSocketPath returns the path to the containerd socket
func (lm *LimaManager) GetSocketPath() string {
	// Lima exposes the socket via a Unix socket on the host
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	// Default Lima socket path: ~/.lima/<instance>/sock/containerd.sock
	return filepath.Join(home, ".lima", lm.instanceName, "sock", "containerd.sock")
}

// isLimaInstalled checks if limactl is available
func (lm *LimaManager) isLimaInstalled() bool {
	_, err := exec.LookPath("limactl")
	return err == nil
}

// instanceExists checks if the Lima instance exists
func (lm *LimaManager) instanceExists() (bool, error) {
	cmd := exec.Command("limactl", "list", "--quiet")
	output, err := cmd.Output()
	if err != nil {
		// If list fails, assume no instances exist
		return false, nil
	}

	instances := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, inst := range instances {
		if strings.TrimSpace(inst) == lm.instanceName {
			return true, nil
		}
	}

	return false, nil
}

// isRunning checks if the Lima instance is running
func (lm *LimaManager) isRunning() (bool, error) {
	cmd := exec.Command("limactl", "list", "--format", "{{.Status}}", lm.instanceName)
	output, err := cmd.Output()
	if err != nil {
		return false, nil
	}

	status := strings.TrimSpace(string(output))
	return status == "Running", nil
}

// createInstance creates a new Lima instance with containerd
func (lm *LimaManager) createInstance(ctx context.Context) error {
	// Create a Lima YAML configuration
	config := lm.createLimaConfig()

	// Write config to temp file
	tmpFile, err := os.CreateTemp("", "warren-lima-*.yaml")
	if err != nil {
		return fmt.Errorf("failed to create temp config file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(config); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	tmpFile.Close()

	// Create instance with limactl
	cmd := exec.CommandContext(ctx, "limactl", "create", "--name", lm.instanceName, tmpFile.Name())
	output, err := cmd.CombinedOutput()
	if err != nil {
		lm.logger.Error().Msgf("Failed to create Lima instance: %s", string(output))
		return fmt.Errorf("failed to create Lima instance: %w", err)
	}

	return nil
}

// startInstance starts the Lima instance
func (lm *LimaManager) startInstance(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "limactl", "start", lm.instanceName)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		lm.logger.Error().Msgf("Failed to start Lima instance: stdout=%s stderr=%s", stdout.String(), stderr.String())
		return fmt.Errorf("failed to start Lima instance: %w", err)
	}

	return nil
}

// waitForReady waits for containerd to be ready inside the VM
func (lm *LimaManager) waitForReady(ctx context.Context) error {
	lm.logger.Info().Msg("Waiting for containerd to be ready")

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	timeout := time.After(60 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("timeout waiting for containerd to be ready")
		case <-ticker.C:
			// Check if containerd socket exists
			socketPath := lm.GetSocketPath()
			if _, err := os.Stat(socketPath); err == nil {
				lm.logger.Info().Msg("Containerd is ready")
				return nil
			}
		}
	}
}

// createLimaConfig creates a Lima YAML configuration for Warren
func (lm *LimaManager) createLimaConfig() string {
	return `# Warren Lima VM Configuration
# Auto-generated - DO NOT EDIT

# VM Resources
cpus: 2
memory: "2GiB"
disk: "20GiB"

# Use Docker template's Ubuntu base for maximum compatibility
images:
  # Use Lima's built-in Ubuntu template
  - location: "https://cloud-images.ubuntu.com/releases/22.04/release/ubuntu-22.04-server-cloudimg-amd64.img"
    arch: "x86_64"
  - location: "https://cloud-images.ubuntu.com/releases/22.04/release/ubuntu-22.04-server-cloudimg-arm64.img"
    arch: "aarch64"

# Mount containerd socket to host
mounts:
  - location: "~"
    writable: false
  - location: "/tmp/lima"
    writable: true

# Install and start containerd
containerd:
  system: true
  user: false

# Provision script to ensure containerd is running
provision:
  - mode: system
    script: |
      #!/bin/bash
      set -eux -o pipefail

      # Install containerd if not already installed
      if ! command -v containerd > /dev/null 2>&1; then
        apt-get update
        apt-get install -y containerd
      fi

      # Ensure containerd is running
      systemctl enable containerd || true
      systemctl start containerd || true
      systemctl status containerd || true

      # Wait for socket to be available
      for i in {1..30}; do
        if [ -S /run/containerd/containerd.sock ]; then
          echo "Containerd is ready"
          exit 0
        fi
        sleep 1
      done

      echo "Containerd socket not found"
      exit 1

# Expose containerd socket to host
portForwards:
  - guestSocket: "/run/containerd/containerd.sock"
    hostSocket: "{{.Dir}}/sock/containerd.sock"

# SSH settings
ssh:
  localPort: 0
  loadDotSSHPubKeys: false
`
}

// EnsureLima ensures the Lima VM is running
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
