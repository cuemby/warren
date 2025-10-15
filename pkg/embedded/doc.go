/*
Package embedded provides containerd binary management for Warren across platforms.

The embedded package handles platform-specific containerd integration, embedding
containerd binaries on Linux and managing Lima VM with containerd on macOS. It
provides automatic lifecycle management, socket detection, and graceful shutdown
for zero-dependency container runtime deployment.

# Architecture

Warren provides embedded containerd with platform-specific implementations:

	┌────────────── EMBEDDED CONTAINERD MANAGEMENT ────────────┐
	│                                                            │
	│  ┌────────────────────────────────────────────┐          │
	│  │      ContainerdManager (Cross-Platform)     │          │
	│  │  - Common interface for all platforms       │          │
	│  │  - Start/Stop lifecycle management          │          │
	│  │  - Socket path detection                    │          │
	│  └──────────────────┬─────────────────────────┘          │
	│                     │                                      │
	│        ┌────────────┴────────────┐                        │
	│        │                          │                        │
	│        ▼                          ▼                        │
	│  ┌──────────┐              ┌──────────┐                  │
	│  │  Linux   │              │  macOS   │                  │
	│  └─────┬────┘              └────┬─────┘                  │
	│        │                         │                        │
	│        ▼                         ▼                        │
	│  ┌──────────────────┐    ┌────────────────────┐         │
	│  │ Embedded Binary  │    │   Lima VM Manager  │         │
	│  │                  │    │                     │         │
	│  │ - containerd     │    │ - Ubuntu 22.04     │         │
	│  │ - Extracted from │    │ - containerd       │         │
	│  │   go:embed       │    │ - 2 CPU, 2GB RAM   │         │
	│  │ - Direct exec    │    │ - Auto-start       │         │
	│  └──────┬───────────┘    └────┬───────────────┘         │
	│         │                      │                          │
	│         ▼                      ▼                          │
	│  ┌──────────────────┐    ┌────────────────────┐         │
	│  │ Linux Process    │    │   Lima VM          │         │
	│  │                  │    │                     │         │
	│  │ containerd       │    │ limactl start      │         │
	│  │   --socket       │    │   warren           │         │
	│  │   /run/warren/   │    │                     │         │
	│  │   containerd.sock│    │ Socket forwarded:  │         │
	│  │                  │    │ ~/.lima/warren/    │         │
	│  │                  │    │   sock/            │         │
	│  │                  │    │   containerd.sock  │         │
	│  └──────────────────┘    └────────────────────┘         │
	└────────────────────────────────────────────────────────┘

# Core Components

ContainerdManager (Linux):
  - Extracts containerd binary from embedded FS
  - Launches containerd as child process
  - Creates minimal configuration file
  - Monitors process and restarts if crashes
  - Graceful shutdown with SIGTERM → SIGKILL

LimaManager (macOS):
  - Creates and manages Lima VM instance
  - Provisions Ubuntu with containerd
  - Forwards containerd socket to host
  - Auto-starts VM on Warren launch
  - Graceful VM shutdown

Socket Detection:
  - Linux: /run/warren-containerd/containerd.sock (embedded)
  - Linux: /run/containerd/containerd.sock (system)
  - macOS: ~/.lima/warren/sock/containerd.sock (Lima)
  - Auto-detect and prefer system containerd if available

Binary Embedding:
  - Go embed directive: //go:embed binaries/*
  - Binaries extracted to /var/lib/warren/bin/
  - Platform-specific: containerd-linux-amd64, containerd-linux-arm64
  - Download via scripts/download-containerd.sh during build

# Platform Implementations

Linux Implementation:

Architecture:
  - Embedded containerd binary (amd64, arm64)
  - Direct process spawn with custom config
  - Socket: /run/warren-containerd/containerd.sock
  - Config: /etc/warren-containerd/config.toml
  - Data: /var/lib/warren/containerd

Binary Extraction:
 1. Check if binary exists and is recent
 2. Read from embedded FS (binaries/containerd-linux-amd64)
 3. Write to /var/lib/warren/bin/containerd
 4. Set executable permissions (0755)

Process Management:
  - Start: exec.Command(containerd, --config, --address, --root, --state)
  - Monitor: Goroutine watches process exit
  - Restart: Automatic on unexpected exit (future)
  - Stop: SIGTERM → wait 10s → SIGKILL

macOS Implementation:

Architecture:
  - Lima VM with Ubuntu 22.04
  - Containerd installed via apt
  - Socket forwarded via Lima port forwarding
  - VM lifecycle managed by limactl

Lima Configuration:
  - CPUs: 2 cores
  - Memory: 2GB
  - Disk: 20GB
  - OS: Ubuntu 22.04 (x86_64/aarch64)
  - Containerd: System package (containerd.io)

VM Provisioning:
 1. Create Lima YAML config
 2. limactl create --name warren config.yaml
 3. limactl start warren
 4. Wait for containerd socket

Socket Forwarding:
  - Lima maps /run/containerd/containerd.sock (guest)
  - To ~/.lima/warren/sock/containerd.sock (host)
  - Warren connects to host socket path
  - Transparent to Warren runtime client

VM Lifecycle:
  - Create: limactl create (one-time setup)
  - Start: limactl start warren (automatic on Warren start)
  - Stop: limactl stop warren (graceful on Warren stop)
  - Delete: Manual cleanup (limactl delete warren)

# Usage

Linux - Embedded Containerd:

	import "github.com/cuemby/warren/pkg/embedded"

	// Create manager
	mgr, err := embedded.NewContainerdManager("/var/lib/warren", false)
	if err != nil {
		log.Fatal(err)
	}

	// Start containerd
	ctx := context.Background()
	if err := mgr.Start(ctx); err != nil {
		log.Fatal(err)
	}
	defer mgr.Stop()

	// Get socket path
	socketPath := mgr.GetSocketPath()
	fmt.Printf("Containerd socket: %s\n", socketPath)

Linux - External Containerd:

	// Use system containerd instead of embedded
	mgr, err := embedded.NewContainerdManager("/var/lib/warren", true)
	if err != nil {
		log.Fatal(err)
	}

	// Start is no-op for external containerd
	ctx := context.Background()
	if err := mgr.Start(ctx); err != nil {
		log.Fatal(err)
	}

	socketPath := mgr.GetSocketPath()
	// Returns: /run/containerd/containerd.sock

macOS - Lima VM:

	import "github.com/cuemby/warren/pkg/embedded"

	// EnsureLima creates and starts Lima VM
	ctx := context.Background()
	limaManager, err := embedded.EnsureLima(ctx, "/var/lib/warren")
	if err != nil {
		log.Fatal(err)
	}
	defer limaManager.Stop(ctx)

	// Get socket path
	socketPath := limaManager.GetSocketPath()
	fmt.Printf("Containerd socket: %s\n", socketPath)
	// Output: /Users/username/.lima/warren/sock/containerd.sock

Complete Example:

	package main

	import (
		"context"
		"fmt"
		"runtime"
		"github.com/cuemby/warren/pkg/embedded"
		"github.com/cuemby/warren/pkg/runtime"
	)

	func main() {
		ctx := context.Background()

		// Platform-specific containerd setup
		var socketPath string
		if runtime.GOOS == "darwin" {
			// macOS: Use Lima VM
			limaManager, err := embedded.EnsureLima(ctx, "/tmp/warren")
			if err != nil {
				panic(err)
			}
			defer limaManager.Stop(ctx)
			socketPath = limaManager.GetSocketPath()
		} else {
			// Linux: Use embedded containerd
			mgr, err := embedded.NewContainerdManager("/var/lib/warren", false)
			if err != nil {
				panic(err)
			}
			if err := mgr.Start(ctx); err != nil {
				panic(err)
			}
			defer mgr.Stop()
			socketPath = mgr.GetSocketPath()
		}

		fmt.Printf("Containerd socket: %s\n", socketPath)

		// Connect to containerd
		rt, err := runtime.NewContainerdRuntime(socketPath)
		if err != nil {
			panic(err)
		}
		defer rt.Close()

		fmt.Println("Successfully connected to containerd")
	}

# Integration Points

This package integrates with:

  - pkg/runtime: Provides containerd socket path
  - pkg/worker: Uses embedded containerd for task execution
  - pkg/manager: May use for single-node deployments
  - Lima: macOS VM management (limactl)

# Containerd Configuration

Minimal Config (Linux):

	version = 2

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

Configuration Details:
  - Version 2: Latest containerd config format
  - CRI plugin: Required for container management
  - Snapshotter: overlayfs for layer management
  - Runtime: runc v2 (OCI-compatible)
  - SystemdCgroup: Use systemd for cgroup management
  - Registry: Docker Hub mirror configuration

# Design Patterns

Platform Abstraction:
  - Build tags separate Linux/macOS implementations
  - Common ContainerdManager interface
  - GetSocketPath() returns platform-specific path
  - Transparent to calling code

Graceful Degradation:
  - Try embedded first, fall back to system containerd
  - Check socket existence before starting
  - Re-use existing VM if already running
  - Non-fatal errors on shutdown

Idempotent Operations:
  - Start checks if already running
  - Stop gracefully handles not running
  - Extract binary skips if recent
  - Safe to call multiple times

Binary Caching:
  - Check modification time before extracting
  - Skip extraction if binary < 24 hours old
  - Reduces startup time on repeated runs
  - Invalidates on Warren binary update

# Performance Characteristics

Linux - Embedded Containerd:
  - Binary extraction: ~500ms (first run)
  - Binary extraction: ~1ms (cached, skip)
  - Process start: ~2-5s (containerd initialization)
  - Memory: ~50MB (containerd base)
  - Disk: ~30MB (binary + data)

macOS - Lima VM:
  - VM creation: ~2-5 minutes (one-time)
  - VM start: ~20-40s (if stopped)
  - VM start: ~1s (if already running)
  - Memory: ~2GB (VM allocation)
  - Disk: ~2GB (VM image + data)

Socket Detection:
  - Check file existence: ~1ms
  - Socket validation: ~50ms (attempt connection)
  - Total startup: 2-5s (Linux), 20-40s (macOS first run)

# Troubleshooting

Common Issues:

Binary Not Found (Linux):
  - Symptom: "failed to read embedded binary" error
  - Cause: Binary not embedded during build
  - Solution: Run make build to download and embed containerd
  - Check: ls pkg/embedded/binaries/

Permission Denied (Linux):
  - Symptom: "permission denied" when starting containerd
  - Cause: Insufficient privileges (need root or containerd group)
  - Solution: Run Warren with sudo or add user to containerd group
  - Check: ls -l /run/containerd/containerd.sock

Lima Not Installed (macOS):
  - Symptom: "Lima is not installed"
  - Cause: limactl not in PATH
  - Solution: brew install lima
  - Check: which limactl

VM Fails to Start (macOS):
  - Symptom: Lima VM won't start or timeout
  - Check: limactl list (show VM status)
  - Check: limactl logs warren (VM boot logs)
  - Solution: Delete and recreate (limactl delete warren)

Socket Not Found:
  - Symptom: "socket not found" when connecting
  - Check: Socket path from GetSocketPath()
  - Check: ls -l <socket_path>
  - Check: Containerd/Lima is running
  - Solution: Restart containerd or Warren

# Monitoring

Key metrics to monitor:

Containerd Health:
  - embedded_containerd_running: Containerd process status (1=running)
  - embedded_containerd_restarts: Process restart count
  - embedded_containerd_uptime: Time since last start

Lima Health (macOS):
  - embedded_lima_running: Lima VM status (1=running)
  - embedded_lima_restarts: VM restart count
  - embedded_lima_memory: VM memory usage

Startup Performance:
  - embedded_startup_duration: Time to start containerd/Lima
  - embedded_socket_ready_duration: Time until socket available
  - embedded_extraction_duration: Binary extraction time

# Security

Embedded Binary Integrity:
  - Binaries verified during download (checksum)
  - Embedded at build time (no runtime download)
  - Read-only in Warren binary (go:embed)
  - Extracted with restrictive permissions (0755)

Lima VM Security:
  - VM isolated from host filesystem (controlled mounts)
  - Socket-only access to containerd (no SSH by default)
  - Ubuntu receives security updates (system containerd)
  - VM network isolated (no external access by default)

Privilege Requirements:
  - Linux: Root or containerd group membership
  - macOS: User-level (Lima runs as user)
  - Socket permissions: 0660 (owner + group)

# Limitations

Current Limitations:
  - Windows: Not supported (WSL2 future)
  - Binary size: Adds ~30MB to Warren binary
  - Lima dependency: Requires Lima on macOS
  - Platform detection: Based on GOOS/GOARCH only
  - No automatic updates: Containerd version fixed at build

Future Enhancements:
  - Windows containerd support
  - Automatic containerd updates
  - Multiple containerd versions (version selector)
  - Podman integration as alternative
  - ARM v7 support (32-bit)

# Platform Support

Linux:
  - Distributions: Ubuntu, Debian, CentOS, Alpine, etc.
  - Architectures: amd64 (x86_64), arm64 (aarch64)
  - Kernel: 3.10+ (4.x+ recommended)
  - Cgroups: v1 or v2

macOS:
  - Versions: macOS 12+ (Monterey and later)
  - Architectures: Intel (x86_64), Apple Silicon (arm64)
  - Dependencies: Lima (installed via Homebrew)
  - VM: Ubuntu 22.04 in Lima

Windows:
  - Status: Not currently supported
  - Future: WSL2 with embedded containerd
  - Alternative: Docker Desktop (external containerd)

# See Also

  - pkg/runtime for containerd client usage
  - pkg/worker for task execution with containerd
  - containerd documentation: https://containerd.io/
  - Lima documentation: https://lima-vm.io/
  - runc documentation: https://github.com/opencontainers/runc
*/
package embedded
