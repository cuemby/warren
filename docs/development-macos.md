# Warren Development on macOS

Warren requires Linux (containerd is Linux-only). This guide shows how to develop and test Warren on macOS using Lima VM.

## Prerequisites

- macOS with Homebrew installed
- Lima VM manager

## Setup

### 1. Install Lima

```bash
brew install lima
```

### 2. Create Warren Development VM

```bash
# Create VM with default template
limactl create --name=warren template://default

# Start the VM
limactl start warren
```

### 3. Build Warren for Linux

```bash
# Build Linux ARM64 binary (for Apple Silicon Macs)
make build-linux-arm64

# OR build Linux AMD64 binary (for Intel Macs or testing AMD64)
make build-linux-amd64
```

### 4. Install Warren in Lima VM

```bash
# Copy binary to VM (adjust for your architecture)
limactl copy bin/warren-linux-arm64 warren:/tmp/warren

# Move to system location (inside VM)
limactl shell warren sudo mv /tmp/warren /usr/local/bin/warren
```

### 5. Initialize Warren Cluster

```bash
# Enter the VM
limactl shell warren

# Initialize cluster from writable directory with sudo
cd /tmp
sudo warren cluster init --data-dir /tmp/warren-data
```

**Output (expected)**:
```
✓ Embedded containerd started (socket: /run/warren-containerd/containerd.sock)
✓ Certificate Authority initialized
✓ Cluster initialized successfully
✓ Scheduler started
✓ Reconciler started
✓ Metrics collector started
✓ DNS server started successfully (address=127.0.0.11:53)
✓ API server listening on Unix socket: /var/run/warren.sock (local, read-only)
```

### 6. Verify Installation

```bash
# List nodes (should work via Unix socket)
sudo warren node list

# Check cluster info
sudo warren cluster info

# List services (should be empty initially)
sudo warren service list
```

## Common Issues

### Error: "read-only file system"

**Problem**: Lima mounts your macOS home directory as read-only.

**Solution**: Use a writable directory like `/tmp`:
```bash
cd /tmp
sudo warren cluster init --data-dir /tmp/warren-data
```

### Error: "permission denied"

**Problem**: Warren needs root permissions to create system directories (`/etc/warren-containerd`, `/var/run/warren.sock`).

**Solution**: Use `sudo`:
```bash
sudo warren cluster init --data-dir /tmp/warren-data
```

## Development Workflow

### Edit-Build-Test Cycle

```bash
# 1. Edit code on macOS (your preferred editor)
vim cmd/warren/main.go

# 2. Build for Linux
make build-linux-arm64

# 3. Copy to Lima VM
limactl copy bin/warren-linux-arm64 warren:/tmp/warren-new

# 4. Replace binary in VM
limactl shell warren sudo mv /tmp/warren-new /usr/local/bin/warren

# 5. Restart Warren (if running)
limactl shell warren sudo pkill warren
limactl shell warren sudo warren cluster init --data-dir /tmp/warren-data
```

### Quick Commands

```bash
# Enter VM
limactl shell warren

# Check Warren logs (if running in background)
limactl shell warren sudo journalctl -u warren -f

# Clean up test data
limactl shell warren sudo rm -rf /tmp/warren-data

# Stop VM
limactl stop warren

# Delete VM (careful - destroys all VM data!)
limactl delete warren
```

## Testing

### Manual Testing

```bash
# Enter VM
limactl shell warren

# Deploy a test service
sudo warren service create nginx --image nginx:alpine --replicas 2

# Check service status
sudo warren service list
sudo warren service inspect nginx

# Scale service
sudo warren service scale nginx --replicas 5

# Delete service
sudo warren service delete nginx
```

### Automated Testing

```bash
# Run test script in VM
limactl copy test/manual/test-unix-socket-lima.sh warren:/tmp/
limactl shell warren sudo bash /tmp/test-unix-socket-lima.sh
```

## Why Lima?

**Warren requires Linux** because:
- containerd only runs on Linux (uses kernel features like cgroups, namespaces)
- Container networking requires Linux kernel features
- Lima provides a lightweight Linux VM on macOS

**Benefits of Lima**:
- Fast startup (~5 seconds)
- Automatic file sharing from macOS
- Native feel (like Docker Desktop, but lighter)
- Full Linux environment for testing

## Persistent vs. Temporary Data

### Temporary Testing (Current Setup)

Using `/tmp/warren-data` means data is **lost on VM restart**:

```bash
cd /tmp
sudo warren cluster init --data-dir /tmp/warren-data
```

**Use for**: Quick tests, development, CI/CD

### Persistent Testing

For data that survives VM restarts, use VM home directory:

```bash
# Create persistent data directory
limactl shell warren mkdir -p ~/warren-data

# Initialize with persistent storage
limactl shell warren bash -c "cd ~ && sudo warren cluster init --data-dir ~/warren-data"
```

**Use for**: Long-running tests, demo environments

## Comparison to Docker Desktop

| Feature | Lima + Warren | Docker Desktop |
|---------|---------------|----------------|
| **VM Size** | ~2GB | ~4GB |
| **Startup** | ~5 seconds | ~15 seconds |
| **Resource Usage** | Minimal | Higher |
| **Cost** | Free | Free (personal) |
| **Purpose** | Linux dev environment | Container runtime |

## Advanced: Multi-Node Cluster in Lima

To test multi-node scenarios:

```bash
# Create 3 VMs for a cluster
limactl create --name=warren-manager-1 template://default
limactl create --name=warren-worker-1 template://default
limactl create --name=warren-worker-2 template://default

# Start all VMs
limactl start warren-manager-1
limactl start warren-worker-1
limactl start warren-worker-2

# Install Warren on all VMs (repeat for each)
limactl copy bin/warren-linux-arm64 warren-manager-1:/tmp/warren
limactl shell warren-manager-1 sudo mv /tmp/warren /usr/local/bin/warren

# Initialize cluster on manager
limactl shell warren-manager-1 sudo warren cluster init --api-addr 0.0.0.0:8080

# Get join token
JOIN_TOKEN=$(limactl shell warren-manager-1 sudo warren cluster join-token)

# Join workers
limactl shell warren-worker-1 sudo warren cluster join <manager-ip>:8080 --token $JOIN_TOKEN
limactl shell warren-worker-2 sudo warren cluster join <manager-ip>:8080 --token $JOIN_TOKEN
```

## Troubleshooting

### Lima VM Won't Start

```bash
# Check Lima status
limactl list

# View VM logs
limactl shell warren dmesg

# Restart Lima
limactl stop warren
limactl start warren
```

### Warren Won't Start

```bash
# Check if containerd is running
limactl shell warren sudo ps aux | grep containerd

# Check Warren logs (if running as service)
limactl shell warren sudo journalctl -u warren -n 100

# Check for port conflicts
limactl shell warren sudo netstat -tulpn | grep 8080
```

### Can't Connect to Warren

```bash
# Check Unix socket exists
limactl shell warren sudo ls -la /var/run/warren.sock

# Check if Warren API is listening
limactl shell warren sudo netstat -tulpn | grep warren

# Try with explicit manager address
limactl shell warren sudo warren node list --manager localhost:8080
```

## Resources

- [Lima Documentation](https://lima-vm.io/docs/)
- [Warren Documentation](../README.md)
- [Containerd Documentation](https://containerd.io/docs/)

## Getting Help

If you encounter issues:
1. Check this troubleshooting guide
2. Check Lima logs: `limactl shell warren dmesg`
3. Check Warren logs in the VM
4. Open an issue: https://github.com/cuemby/warren/issues
