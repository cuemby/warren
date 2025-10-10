# Warren Lima Testing Infrastructure

Lima-based multi-VM testing environment for Warren's multi-manager Raft cluster and containerd integration.

## Overview

This testing infrastructure uses [Lima](https://lima-vm.io/) to create lightweight Linux VMs on macOS (or Linux) for comprehensive Warren testing. Lima provides:

- ✅ Native containerd support (no Docker Desktop)
- ✅ Multi-VM orchestration (perfect for Raft cluster)
- ✅ Cross-platform (macOS Intel/ARM + Linux)
- ✅ VM-to-VM networking via user-v2
- ✅ Automatic file sharing (Warren workspace mounted)

## Prerequisites

### Install Lima

**macOS:**
```bash
brew install lima
```

**Linux:**
See [Lima installation docs](https://lima-vm.io/docs/installation/)

### Build Warren

```bash
cd /path/to/warren
make build
```

## Quick Start

### 1. Setup Test Environment

Create 3 manager VMs and 2 worker VMs:

```bash
./test/lima/setup.sh
```

This will:
- Create `warren-manager-1`, `warren-manager-2`, `warren-manager-3`
- Create `warren-worker-1`, `warren-worker-2`
- Configure user-v2 networking (VM-to-VM communication)
- Install containerd and dependencies in each VM
- Share Warren workspace at `~/warren` in all VMs

### 2. Run Tests

**Option A: Run all tests end-to-end**
```bash
./test/lima/test-e2e.sh
```

**Option B: Run individual tests**
```bash
# Test cluster formation
./test/lima/test-cluster.sh

# Test leader failover
./test/lima/test-failover.sh
```

### 3. Cleanup

```bash
# Delete all VMs
./test/lima/cleanup.sh

# Or just stop Warren processes (keep VMs)
./test/lima/cleanup.sh --keep-vms
```

## Test Scenarios

### Test 1: Cluster Formation (`test-cluster.sh`)

Tests 3-manager Raft cluster formation:

1. Bootstrap first manager (manager-1)
2. Generate manager join token
3. Join second manager (manager-2) with token
4. Join third manager (manager-3) with token
5. Verify Raft quorum (3 voters)
6. Start 2 worker nodes
7. Deploy test service (nginx-test)
8. Verify service is running

**Expected Results:**
- ✓ 3 managers in Raft cluster
- ✓ Leader elected
- ✓ 2 workers registered
- ✓ Service deployed with 2 replicas

### Test 2: Leader Failover (`test-failover.sh`)

Tests Raft leader failover and recovery:

1. Identify current leader
2. Kill leader process
3. Verify new leader elected **within 10 seconds**
4. Verify cluster continues operating
5. Test read/write operations after failover
6. Restart killed leader

**Expected Results:**
- ✓ New leader elected < 10s
- ✓ Cluster operational after failover
- ✓ Can create services after failover
- ✓ Killed leader can rejoin

### Test 3: End-to-End (`test-e2e.sh`)

Runs complete test suite:

1. Setup: Create VMs (or skip with `--skip-setup`)
2. Run cluster formation test
3. Run leader failover test
4. Cleanup (or skip with `--skip-cleanup`)

```bash
# Full e2e
./test/lima/test-e2e.sh

# Skip setup (VMs already exist)
./test/lima/test-e2e.sh --skip-setup

# Keep VMs after tests
./test/lima/test-e2e.sh --keep-vms
```

## VM Architecture

### Networking (user-v2)

```
Host Machine
├── warren-manager-1 (lima-warren-manager-1.internal)
│   ├── IP: 192.168.104.x
│   ├── API: :8080
│   └── Raft: :7946
├── warren-manager-2 (lima-warren-manager-2.internal)
│   ├── IP: 192.168.104.x
│   ├── API: :8081
│   └── Raft: :7947
├── warren-manager-3 (lima-warren-manager-3.internal)
│   ├── IP: 192.168.104.x
│   ├── API: :8082
│   └── Raft: :7948
├── warren-worker-1 (lima-warren-worker-1.internal)
│   └── IP: 192.168.104.x
└── warren-worker-2 (lima-warren-worker-2.internal)
    └── IP: 192.168.104.x
```

VMs communicate via `.internal` hostnames (mDNS resolution).

### Resources per VM

- **CPUs:** 2 cores
- **Memory:** 2GB
- **Disk:** 20GB
- **OS:** Ubuntu 22.04 LTS
- **Containerd:** System containerd (/run/containerd/containerd.sock)

## Manual Testing

### Access a VM

```bash
# Shell into manager-1
limactl shell warren-manager-1

# Inside VM
cd ~/warren  # Warren workspace is mounted here
sudo ./bin/warren cluster info --manager=lima-warren-manager-1.internal:8080
```

### View Logs

```bash
# Manager logs
limactl shell warren-manager-1
tail -f /tmp/warren-manager-1.log

# Worker logs
limactl shell warren-worker-1
tail -f /tmp/warren-worker-1.log
```

### Check Containerd

```bash
limactl shell warren-manager-1
sudo ctr containers list  # List containers
sudo ctr namespaces list  # List namespaces
```

### Manually Start Components

**Start manager (bootstrap):**
```bash
limactl shell warren-manager-1
cd ~/warren
sudo ./bin/warren cluster init \
  --node-id=manager-1 \
  --bind-addr=lima-warren-manager-1.internal:7946 \
  --api-addr=lima-warren-manager-1.internal:8080 \
  --data-dir=/tmp/warren-data-1
```

**Generate join token:**
```bash
sudo ./bin/warren cluster join-token manager \
  --manager=lima-warren-manager-1.internal:8080
```

**Join another manager:**
```bash
limactl shell warren-manager-2
cd ~/warren
sudo ./bin/warren manager join \
  --node-id=manager-2 \
  --bind-addr=lima-warren-manager-2.internal:7947 \
  --api-addr=lima-warren-manager-2.internal:8081 \
  --data-dir=/tmp/warren-data-2 \
  --leader=lima-warren-manager-1.internal:8080 \
  --token=<TOKEN>
```

**Start worker:**
```bash
limactl shell warren-worker-1
cd ~/warren
sudo ./bin/warren worker start \
  --node-id=worker-1 \
  --manager=lima-warren-manager-1.internal:8080 \
  --data-dir=/tmp/warren-worker-1 \
  --cpu=2 \
  --memory=2
```

## Troubleshooting

### VMs not starting

```bash
# Check Lima status
limactl list

# View VM logs
limactl shell warren-manager-1 -- journalctl -xe

# Delete and recreate
./test/lima/cleanup.sh
./test/lima/setup.sh
```

### Networking issues

```bash
# Inside VM, test connectivity
limactl shell warren-manager-1

# Ping gateway
ping 192.168.104.1

# Resolve other VMs
nslookup lima-warren-manager-2.internal

# Test Raft connectivity
curl http://lima-warren-manager-2.internal:8081/health
```

### Containerd not working

```bash
limactl shell warren-manager-1

# Check containerd status
sudo systemctl status containerd

# View containerd logs
sudo journalctl -u containerd -f

# Test containerd
sudo ctr version
```

### Warren processes stuck

```bash
# Kill all Warren processes in all VMs
./test/lima/cleanup.sh --keep-vms

# Or manually in a VM
limactl shell warren-manager-1
sudo pkill -9 warren
```

## Files Reference

### Scripts

| File | Description |
|------|-------------|
| `setup.sh` | Create and configure all Lima VMs |
| `test-cluster.sh` | Test 3-manager cluster formation |
| `test-failover.sh` | Test Raft leader failover |
| `test-e2e.sh` | Run complete test suite |
| `cleanup.sh` | Stop processes and delete VMs |

### Configuration

| File | Description |
|------|-------------|
| `warren.yaml` | Lima VM template (OS, resources, provisioning) |

## Advanced Usage

### Custom VM Configuration

```bash
# Create more managers
./test/lima/setup.sh --managers 5 --workers 3

# Clean before setup
./test/lima/setup.sh --clean
```

### Keep VMs for Debugging

```bash
# Run tests but keep VMs alive
./test/lima/test-e2e.sh --skip-cleanup

# Later, manually cleanup
./test/lima/cleanup.sh
```

### CI/CD Integration

Lima works in GitHub Actions:

```yaml
- name: Install Lima
  run: brew install lima

- name: Run Warren Tests
  run: ./test/lima/test-e2e.sh
```

## Performance Tips

1. **Pre-pull Images:** Lima VMs cache container images between runs
2. **Reuse VMs:** Use `--skip-setup` when VMs already exist
3. **Parallel Tests:** Lima supports multiple VM instances simultaneously
4. **Resource Tuning:** Edit `warren.yaml` to adjust CPU/memory per VM

## See Also

- [Lima Documentation](https://lima-vm.io/docs/)
- [Warren Architecture](../../.agent/System/project-architecture.md)
- [Warren PRD](../../specs/prd.md)
- [Milestone 2 Tasks](../../tasks/todo.md#milestone-2-high-availability)

## Questions?

- Check Lima logs: `limactl shell <VM> -- journalctl -xe`
- Inspect Warren logs: `/tmp/warren-*.log` inside VMs
- View network status: `limactl list`
- Debug VM: `limactl shell <VM> bash`
