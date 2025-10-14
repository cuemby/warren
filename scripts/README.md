# Warren Deployment Scripts

Automated production deployment system for Warren v1.3.1. Creates a fully functional Warren cluster using Lima VMs with complete E2E validation.

## 📋 Overview

The deployment automation provides:

- ✅ **Cross-Platform**: macOS, Linux, Windows (WSL2)
- ✅ **Architecture Agnostic**: Intel (amd64) and Apple Silicon (arm64)
- ✅ **Fully Automated**: From VM creation to validated cluster
- ✅ **Production-Ready**: Includes monitoring setup and E2E validation
- ✅ **Idempotent**: Safe to re-run
- ✅ **Modular Design**: Reusable helper libraries

## 🚀 Quick Start

### Prerequisites

- **Lima**: VM manager for container operations
  - macOS: `brew install lima`
  - Linux: See [Lima installation](https://lima-vm.io/docs/installation/)
  - Windows: Install via WSL2 + Linux method

- **jq** (recommended): JSON parsing
  - macOS: `brew install jq`
  - Linux: `apt-get install jq` or `yum install jq`

### Basic Usage

```bash
# Deploy with defaults (3 managers + 3 workers)
./scripts/deploy-production.sh

# Deploy minimal cluster (1 manager + 2 workers)
./scripts/deploy-production.sh --managers 1 --workers 2

# Deploy with custom resources
./scripts/deploy-production.sh --cpus 4 --memory 4

# Clean deployment (remove existing VMs first)
./scripts/deploy-production.sh --clean

# Dry run to see what would happen
./scripts/deploy-production.sh --dry-run --verbose
```

## 📖 Command Line Options

### Cluster Configuration

- `--managers N` - Number of manager nodes (default: 3, min: 1)
  - 1 manager: Development/testing
  - 3 managers: Production HA (recommended)
  - 5+ managers: Large-scale production

- `--workers N` - Number of worker nodes (default: 3, min: 1)
  - Scale based on workload requirements

- `--version V` - Warren version to deploy (default: v1.3.1)

### VM Resources

- `--cpus N` - CPUs per VM (default: 2)
- `--memory N` - Memory per VM in GB (default: 2)

### Deployment Options

- `--clean` - Delete existing Warren VMs before deployment
- `--skip-monitoring` - Skip Prometheus monitoring setup
- `--skip-validation` - Skip E2E validation tests
- `--keep-on-failure` - Keep VMs running if deployment fails (for debugging)

### Execution Control

- `--dry-run` - Show what would be done without executing
- `--verbose` - Enable verbose output
- `--help` - Show help message

## 🏗️ Architecture

### Main Script

**`deploy-production.sh`** - Orchestrates the complete deployment

Deployment phases:
1. **VM Creation** - Creates Lima VMs with Ubuntu 22.04, containerd
2. **Warren Installation** - Downloads and installs Warren binary
3. **Cluster Initialization** - Initializes managers and joins workers
4. **Monitoring Setup** - Configures Prometheus metrics collection
5. **E2E Validation** - Runs comprehensive validation tests
6. **Post-Deployment** - Generates cluster information and access details

### Helper Libraries

**`lib/lima-utils.sh`** - Lima VM management
- VM lifecycle (create, delete, start, stop)
- Dynamic Lima YAML generation
- Command execution in VMs
- File transfer utilities
- Network configuration (user-v2 networking for VM-to-VM communication)

**`lib/warren-utils.sh`** - Warren cluster operations
- Binary download and installation
- Manager initialization
- Token generation and distribution
- Worker joining
- Cluster health verification
- Service deployment and management

**`lib/validation-utils.sh`** - E2E validation automation
- Cluster health validation (8 phases)
- Service deployment testing
- Scaling operations validation
- Leader failover testing
- Secrets management validation
- Volume management validation
- Metrics endpoint verification
- Performance benchmarking

## 📊 Deployment Phases

### Phase 1: VM Creation

Creates Lima VMs with:
- Ubuntu 22.04 LTS (amd64/arm64)
- Containerd runtime (system-managed)
- User-v2 networking (VM-to-VM communication)
- Warren workspace mounted (read-only)
- Required system packages

Example:
```bash
# Creates VMs with names: warren-manager-1, warren-manager-2, warren-manager-3,
#                         warren-worker-1, warren-worker-2, warren-worker-3
```

### Phase 2: Warren Installation

Downloads Warren binary (if not found locally) and installs on all VMs:
- Detects host OS and architecture
- Downloads from GitHub releases or uses local binary
- Installs to `/usr/local/bin/warren`
- Verifies installation

### Phase 3: Cluster Initialization

Initializes Warren cluster:
1. Start manager-1 as leader
2. Generate manager and worker join tokens
3. Join additional managers (if HA cluster)
4. Join all workers
5. Verify cluster health

### Phase 4: Monitoring Setup

Configures monitoring:
- Prometheus metrics available at `:9090/metrics`
- Health check endpoints (`:8080/health`, `:8080/ready`, `:8080/live`)
- Provides Prometheus scrape configuration guidance

### Phase 5: E2E Validation

Runs comprehensive validation suite (8 phases):
1. **Cluster Health** - Health endpoints, Raft leadership
2. **Service Deployment** - Create and verify nginx service
3. **Scaling** - Scale up (1→5) and down (5→2)
4. **Leader Failover** - Stop leader, verify new election (HA only)
5. **Secrets** - Create, list, delete secrets
6. **Volumes** - Create, list, delete volumes
7. **Metrics** - Verify all critical metrics exist
8. **Performance** - Service creation latency, API response time

### Phase 6: Post-Deployment

Generates cluster information:
- Cluster configuration summary
- Node details (names, IPs)
- Access information
- Quick command reference
- Documentation links

## 🔧 Helper Library Reference

### Lima Utilities (`lib/lima-utils.sh`)

```bash
# Create VM
lima_create_vm "warren-manager-1" 2 4  # name, cpus, memory_gb

# Execute command in VM
lima_exec "warren-manager-1" "uptime"

# Execute as root
lima_exec_root "warren-manager-1" "systemctl status containerd"

# Get VM IP
ip=$(lima_get_ip "warren-manager-1")

# Wait for VM to be ready
lima_wait_ready "warren-manager-1" 120  # timeout in seconds

# Copy file to VM
lima_copy_to_vm "warren-manager-1" "/local/file" "/remote/path"

# Create all cluster VMs
lima_create_cluster_vms 3 3 2 2  # managers, workers, cpus, memory
```

### Warren Utilities (`lib/warren-utils.sh`)

```bash
# Download Warren binary
warren_download_binary "v1.3.1" "darwin" "arm64" "/tmp/bin"

# Install on VM
warren_install_on_vm "warren-manager-1" "/local/warren-darwin-arm64"

# Initialize manager
warren_init_manager "warren-manager-1"

# Get join tokens
manager_token=$(warren_get_manager_token "warren-manager-1")
worker_token=$(warren_get_worker_token "warren-manager-1")

# Join additional manager
warren_join_manager "warren-manager-2" "192.168.104.2:8080" "$manager_token"

# Start worker
warren_start_worker "warren-worker-1" "192.168.104.2:8080" "$worker_token"

# Verify cluster health
warren_verify_cluster_health "warren-manager-1" 3 3  # expected managers, workers

# Deploy test service
warren_deploy_test_service "warren-manager-1" "nginx" "nginx:alpine" 3

# Initialize complete cluster
warren_initialize_cluster 3 3  # managers, workers
```

### Validation Utilities (`lib/validation-utils.sh`)

```bash
# Individual validation phases
validate_cluster_health "warren-manager-1"
validate_service_deployment "warren-manager-1"
validate_scaling "warren-manager-1"
validate_leader_failover 3  # num_managers
validate_secrets "warren-manager-1"
validate_volumes "warren-manager-1"
validate_metrics "warren-manager-1"
validate_performance "warren-manager-1"

# Run all validation phases
validate_all "warren-manager-1" 3 3  # leader_vm, managers, workers
```

## 📝 Output & Logs

### Deployment Log

All deployment activity is logged to:
```
/tmp/warren-prod-YYYYMMDD-HHMMSS/deployment.log
```

Contains:
- Timestamped operations
- Command execution details
- Error messages
- Validation results

### Cluster Information

Generated at:
```
/tmp/warren-prod-YYYYMMDD-HHMMSS/cluster-info.txt
```

Contains:
- Deployment configuration
- Node details (names, IPs)
- Access information
- Quick commands
- Documentation links

## 🎯 Usage Examples

### Development Cluster

Single manager, 2 workers, minimal resources:

```bash
./scripts/deploy-production.sh \
  --managers 1 \
  --workers 2 \
  --cpus 2 \
  --memory 2 \
  --skip-validation
```

Deployment time: ~5 minutes

### Production HA Cluster

3 managers, 5 workers, more resources:

```bash
./scripts/deploy-production.sh \
  --managers 3 \
  --workers 5 \
  --cpus 4 \
  --memory 4
```

Deployment time: ~15 minutes (with full E2E validation)

### Testing & Debugging

Clean deployment with verbose output:

```bash
./scripts/deploy-production.sh \
  --clean \
  --verbose \
  --keep-on-failure
```

If deployment fails, VMs remain running for inspection.

### Custom Version

Deploy specific Warren version:

```bash
./scripts/deploy-production.sh \
  --version v1.2.0 \
  --managers 3 \
  --workers 3
```

## 🔍 Accessing the Cluster

### After Successful Deployment

```bash
# Access leader manager
limactl shell warren-manager-1

# Check cluster status
limactl shell warren-manager-1 sudo warren node list --manager localhost:8080

# Deploy a service
limactl shell warren-manager-1 sudo warren service create myapp \
  --image nginx:alpine \
  --replicas 3 \
  --manager localhost:8080

# View metrics
curl http://localhost:9090/metrics

# Check health
curl http://localhost:8080/health
```

### List All VMs

```bash
# List Warren VMs
limactl list | grep warren

# Output:
# warren-manager-1  Running
# warren-manager-2  Running
# warren-manager-3  Running
# warren-worker-1   Running
# warren-worker-2   Running
# warren-worker-3   Running
```

## 🧹 Cleanup

### Stop All VMs

```bash
for vm in $(limactl list | grep ^warren | awk '{print $1}'); do
  limactl stop "$vm"
done
```

### Delete All VMs

```bash
for vm in $(limactl list | grep ^warren | awk '{print $1}'); do
  limactl delete -f "$vm"
done
```

Or use the clean flag:

```bash
./scripts/deploy-production.sh --clean --dry-run  # Preview what will be deleted
./scripts/deploy-production.sh --clean            # Actually delete and redeploy
```

## 🐛 Troubleshooting

### Lima Not Starting VMs

**Problem**: VMs fail to start with QEMU errors

**Solution**:
```bash
# Check Lima installation
limactl --version

# Check Lima status
limactl list

# View Lima logs
limactl shell warren-manager-1 --debug
```

### Warren Binary Not Found

**Problem**: Warren binary not available locally or on GitHub

**Solution**:
```bash
# Build Warren locally first
cd /Users/ar4mirez/Developer/Work/cuemby/warren
make build

# Then run deployment
./scripts/deploy-production.sh
```

### Validation Failures

**Problem**: E2E validation tests fail

**Solution**:
```bash
# Skip validation initially
./scripts/deploy-production.sh --skip-validation

# Then manually run validation
limactl shell warren-manager-1
sudo warren node list --manager localhost:8080
sudo warren service create test --image nginx:alpine --replicas 1 --manager localhost:8080
```

### VMs Running Out of Resources

**Problem**: VMs slow or unresponsive

**Solution**:
```bash
# Increase resources
./scripts/deploy-production.sh \
  --clean \
  --cpus 4 \
  --memory 8
```

### Network Connectivity Issues

**Problem**: VMs cannot communicate

**Solution**:
```bash
# Check Lima networking
limactl shell warren-manager-1 ip addr show lima0
limactl shell warren-manager-1 ping -c 3 192.168.104.1

# Verify user-v2 network is enabled (automatic in script)
```

## 📚 References

- [Warren Documentation](../docs/)
- [DEPLOYMENT-CHECKLIST.md](../DEPLOYMENT-CHECKLIST.md) - Manual deployment steps
- [E2E Validation Guide](../docs/e2e-validation.md) - Detailed validation procedures
- [Operational Runbooks](../docs/operational-runbooks.md) - Day-2 operations
- [Lima Documentation](https://lima-vm.io/docs/)

## 🤝 Contributing

Improvements to the deployment automation are welcome! Please:

1. Test changes thoroughly
2. Update this README
3. Document new functions in helper libraries
4. Add validation for new features

## 📄 License

Apache 2.0 - See [LICENSE](../LICENSE) for details.

---

**Maintained by**: Cuemby Inc.
**Version**: 1.0.0
**Last Updated**: 2025-10-14
