package embedded

import (
	"context"
	"embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/cuemby/warren/pkg/log"
)

//go:embed binaries/*
var binaries embed.FS

const (
	// DefaultDataDir is where Warren stores extracted binaries and data
	DefaultDataDir = "/var/lib/warren"

	// ContainerdSocketPath is the socket path for embedded containerd
	ContainerdSocketPath = "/run/warren-containerd/containerd.sock"

	// ContainerdConfigPath is the config file path
	ContainerdConfigPath = "/etc/warren-containerd/config.toml"
)

// ContainerdManager manages the embedded containerd daemon
type ContainerdManager struct {
	dataDir      string
	socketPath   string
	configPath   string
	binaryPath   string
	cmd          *exec.Cmd
	useExternal  bool
	logger       *log.Logger
}

// NewContainerdManager creates a new containerd manager
func NewContainerdManager(dataDir string, useExternal bool) (*ContainerdManager, error) {
	if dataDir == "" {
		dataDir = DefaultDataDir
	}

	logger := log.NewLogger("embedded-containerd", "info")

	return &ContainerdManager{
		dataDir:     dataDir,
		socketPath:  ContainerdSocketPath,
		configPath:  ContainerdConfigPath,
		useExternal: useExternal,
		logger:      logger,
	}, nil
}

// Start starts the embedded containerd daemon
func (cm *ContainerdManager) Start(ctx context.Context) error {
	if cm.useExternal {
		cm.logger.Info("Using external containerd, skipping embedded start")
		return nil
	}

	// Extract containerd binary if needed
	if err := cm.extractBinary(); err != nil {
		return fmt.Errorf("failed to extract containerd binary: %w", err)
	}

	// Create config file
	if err := cm.createConfig(); err != nil {
		return fmt.Errorf("failed to create containerd config: %w", err)
	}

	// Create socket directory
	socketDir := filepath.Dir(cm.socketPath)
	if err := os.MkdirAll(socketDir, 0755); err != nil {
		return fmt.Errorf("failed to create socket directory: %w", err)
	}

	// Start containerd process
	cm.logger.Info(fmt.Sprintf("Starting embedded containerd at %s", cm.socketPath))

	cm.cmd = exec.CommandContext(ctx, cm.binaryPath,
		"--config", cm.configPath,
		"--address", cm.socketPath,
		"--root", filepath.Join(cm.dataDir, "containerd"),
		"--state", filepath.Join(cm.dataDir, "containerd-state"),
	)

	// Set up logging
	cm.cmd.Stdout = &logWriter{logger: cm.logger, level: "info"}
	cm.cmd.Stderr = &logWriter{logger: cm.logger, level: "error"}

	// Start the process
	if err := cm.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start containerd: %w", err)
	}

	// Wait for containerd to be ready
	if err := cm.waitForReady(ctx, 30*time.Second); err != nil {
		cm.Stop()
		return fmt.Errorf("containerd failed to become ready: %w", err)
	}

	cm.logger.Info("Embedded containerd started successfully")

	// Monitor containerd in background
	go cm.monitor(ctx)

	return nil
}

// Stop stops the embedded containerd daemon
func (cm *ContainerdManager) Stop() error {
	if cm.useExternal || cm.cmd == nil || cm.cmd.Process == nil {
		return nil
	}

	cm.logger.Info("Stopping embedded containerd")

	// Send SIGTERM for graceful shutdown
	if err := cm.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		cm.logger.Error(fmt.Sprintf("Failed to send SIGTERM: %v", err))
	}

	// Wait for up to 10 seconds for graceful shutdown
	done := make(chan error, 1)
	go func() {
		done <- cm.cmd.Wait()
	}()

	select {
	case <-time.After(10 * time.Second):
		// Force kill if graceful shutdown fails
		cm.logger.Warn("Containerd did not stop gracefully, force killing")
		if err := cm.cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill containerd: %w", err)
		}
		<-done // Wait for process to exit
	case err := <-done:
		if err != nil && err.Error() != "signal: terminated" {
			cm.logger.Error(fmt.Sprintf("Containerd exited with error: %v", err))
		}
	}

	cm.logger.Info("Embedded containerd stopped")
	return nil
}

// GetSocketPath returns the containerd socket path
func (cm *ContainerdManager) GetSocketPath() string {
	if cm.useExternal {
		return "/run/containerd/containerd.sock" // System default
	}
	return cm.socketPath
}

// extractBinary extracts the containerd binary from embedded FS
func (cm *ContainerdManager) extractBinary() error {
	// Determine binary name based on OS and architecture
	binaryName := fmt.Sprintf("containerd-%s-%s", runtime.GOOS, runtime.GOARCH)
	embeddedPath := fmt.Sprintf("binaries/%s", binaryName)

	// Check if binary already exists and is up-to-date
	binDir := filepath.Join(cm.dataDir, "bin")
	cm.binaryPath = filepath.Join(binDir, "containerd")

	if info, err := os.Stat(cm.binaryPath); err == nil {
		// Binary exists, check if it's recent enough (skip re-extraction for now)
		if time.Since(info.ModTime()) < 24*time.Hour {
			cm.logger.Info("Using existing containerd binary")
			return nil
		}
	}

	cm.logger.Info("Extracting containerd binary")

	// Create bin directory
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}

	// Read embedded binary
	data, err := binaries.ReadFile(embeddedPath)
	if err != nil {
		return fmt.Errorf("failed to read embedded binary %s: %w (this binary may not have containerd bundled - run 'make build' to bundle it)", embeddedPath, err)
	}

	// Write to disk
	if err := os.WriteFile(cm.binaryPath, data, 0755); err != nil {
		return fmt.Errorf("failed to write binary: %w", err)
	}

	cm.logger.Info(fmt.Sprintf("Extracted containerd binary to %s", cm.binaryPath))
	return nil
}

// createConfig creates a minimal containerd config
func (cm *ContainerdManager) createConfig() error {
	configDir := filepath.Dir(cm.configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	config := `version = 2

[plugins]
  [plugins."io.containerd.grpc.v1.cri"]
    sandbox_image = "registry.k8s.io/pause:3.9"

    [plugins."io.containerd.grpc.v1.cri".containerd]
      snapshotter = "overlayfs"

      [plugins."io.containerd.grpc.v1.cri".containerd.runtimes]
        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc]
          runtime_type = "io.containerd.runc.v2"

          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
            SystemdCgroup = true

[plugins."io.containerd.grpc.v1.cri".registry]
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors]
    [plugins."io.containerd.grpc.v1.cri".registry.mirrors."docker.io"]
      endpoint = ["https://registry-1.docker.io"]
`

	if err := os.WriteFile(cm.configPath, []byte(config), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// waitForReady waits for containerd to be ready
func (cm *ContainerdManager) waitForReady(ctx context.Context, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for containerd to be ready")
		case <-ticker.C:
			// Check if socket exists
			if _, err := os.Stat(cm.socketPath); err == nil {
				// Socket exists, try to connect
				// TODO: Could add actual health check via gRPC if needed
				return nil
			}
		}
	}
}

// monitor watches the containerd process and restarts if it crashes
func (cm *ContainerdManager) monitor(ctx context.Context) {
	if cm.cmd == nil || cm.cmd.Process == nil {
		return
	}

	// Wait for process to exit
	err := cm.cmd.Wait()

	// Check if context was cancelled (intentional shutdown)
	select {
	case <-ctx.Done():
		cm.logger.Info("Containerd monitor exiting (context cancelled)")
		return
	default:
	}

	// Unexpected exit - log error
	if err != nil {
		cm.logger.Error(fmt.Sprintf("Containerd process exited unexpectedly: %v", err))
	} else {
		cm.logger.Warn("Containerd process exited unexpectedly with no error")
	}

	// TODO: Implement automatic restart logic if needed
	// For now, just log the failure - Warren should handle this at a higher level
}

// logWriter adapts containerd output to Warren's logger
type logWriter struct {
	logger *log.Logger
	level  string
}

func (lw *logWriter) Write(p []byte) (n int, err error) {
	msg := string(p)
	if lw.level == "error" {
		lw.logger.Error(msg)
	} else {
		lw.logger.Info(msg)
	}
	return len(p), nil
}

// EnsureContainerd ensures containerd is available (starts embedded if not using external)
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
