# Warren Testing with Lima

**Status:** Active
**Last Updated:** 2025-10-10
**Owner:** Warren Team
**Related:** [Milestone 2](../../tasks/todo.md#milestone-2-high-availability) | [Test Scripts](../../test/lima/)

## Overview

Warren uses [Lima](https://lima-vm.io/) for multi-VM integration testing. Lima provides lightweight Linux VMs with native containerd support, enabling realistic testing of Warren's multi-manager Raft cluster and container orchestration without requiring cloud infrastructure.

## Why Lima?

### Problem

Warren's Milestone 2 requires testing:
- Multi-manager Raft cluster (3+ managers)
- Leader failover and re-election
- Real containerd integration (not simulated)
- VM-to-VM networking for Raft consensus
- Worker nodes with actual container execution

### Solution: Lima

Lima is perfect for Warren testing because:

| Requirement | Lima Solution |
|-------------|--------------|
| **Real Containerd** | Native containerd in each VM, full socket access |
| **Multi-VM Cluster** | Easy creation of 3+ VMs with networking |
| **Cross-Platform** | Works on macOS (dev) and Linux (CI) |
| **Lightweight** | Micro-VMs, faster than traditional VMs |
| **VM-to-VM Networking** | user-v2 network with mDNS (.internal hostnames) |
| **File Sharing** | Auto-mount Warren workspace in all VMs |
| **Reproducible** | YAML templates ensure consistent environments |

### Alternatives Considered

| Alternative | Why Not? |
|-------------|----------|
| **Docker Compose** | Can't access real containerd, nested container issues |
| **Kind/k3d** | Kubernetes-focused, overkill for Warren |
| **Multipass** | Heavier than Lima, no built-in containerd |
| **Vagrant** | Traditional/heavy, slower iteration |
| **Cloud VMs** | Cost, slower, requires network access |

## Architecture

### Test Environment

```
┌─────────────────────────────────────────────────────────────┐
│                      Host Machine                            │
│                   (macOS or Linux)                           │
│                                                              │
│  Lima VMs (user-v2 network: 192.168.104.0/24)               │
│                                                              │
│  ┌──────────────────┐  ┌──────────────────┐  ┌─────────────┐│
│  │ warren-manager-1 │  │ warren-manager-2 │  │warren-mgr-3 ││
│  │                  │  │                  │  │             ││
│  │ Ubuntu 22.04     │  │ Ubuntu 22.04     │  │Ubuntu 22.04 ││
│  │ Containerd       │  │ Containerd       │  │Containerd   ││
│  │ Warren Manager   │  │ Warren Manager   │  │Warren Mgr   ││
│  │                  │  │                  │  │             ││
│  │ API: :8080       │  │ API: :8081       │  │API: :8082   ││
│  │ Raft: :7946      │  │ Raft: :7947      │  │Raft: :7948  ││
│  │                  │  │                  │  │             ││
│  │ lima-warren-     │  │ lima-warren-     │  │lima-warren- ││
│  │ manager-1        │  │ manager-2        │  │manager-3    ││
│  │ .internal        │  │ .internal        │  │.internal    ││
│  └────────┬─────────┘  └────────┬─────────┘  └──────┬──────┘│
│           │                     │                    │       │
│           └─────────────────────┴────────────────────┘       │
│                     Raft Cluster (Quorum: 2/3)               │
│                                                              │
│  ┌──────────────────┐  ┌──────────────────┐                 │
│  │ warren-worker-1  │  │ warren-worker-2  │                 │
│  │                  │  │                  │                 │
│  │ Warren Worker    │  │ Warren Worker    │                 │
│  │ Containerd       │  │ Containerd       │                 │
│  │ (Runs containers)│  │ (Runs containers)│                 │
│  └──────────────────┘  └──────────────────┘                 │
│                                                              │
│  Warren Workspace: ~/warren (shared from host)              │
└─────────────────────────────────────────────────────────────┘
```

### Networking

**Mode:** user-v2 (Lima's VM-to-VM networking)

- **Subnet:** 192.168.104.0/24
- **Gateway:** 192.168.104.1
- **DNS:** mDNS resolution for `.internal` hostnames
- **Connectivity:** All VMs can reach each other via `lima-<VM_NAME>.internal`

**Example:**
- Manager-1 → Manager-2: `http://lima-warren-manager-2.internal:8081`
- Worker-1 → Manager-1: `http://lima-warren-manager-1.internal:8080`

## Test Scenarios

### Scenario 1: Multi-Manager Cluster Formation

**Test:** `test/lima/test-cluster.sh`

**Steps:**
1. Bootstrap manager-1 (Raft leader)
2. Generate secure join token
3. Join manager-2 with token
4. Join manager-3 with token
5. Verify Raft quorum (3 voters)
6. Start 2 worker nodes
7. Deploy nginx service (2 replicas)
8. Verify containers running

**Validates:**
- ✅ Token-based secure joining
- ✅ Raft cluster formation
- ✅ Leader election
- ✅ Worker registration
- ✅ Service deployment
- ✅ Real containerd execution

### Scenario 2: Leader Failover

**Test:** `test/lima/test-failover.sh`

**Steps:**
1. Identify current Raft leader
2. Kill leader process (simulate crash)
3. Measure time to new leader election
4. Verify cluster still operational
5. Create new service (test write operation)
6. Verify reads work from followers
7. Restart killed leader

**Validates:**
- ✅ Leader re-election < 10s
- ✅ Cluster availability during failover
- ✅ Write operations after failover
- ✅ Follower-to-leader promotion
- ✅ Previous leader can rejoin

### Scenario 3: End-to-End

**Test:** `test/lima/test-e2e.sh`

Runs both scenarios sequentially, measures total time, reports results.

## Usage

### Quick Start

```bash
# 1. Setup (one-time)
./test/lima/setup.sh

# 2. Run all tests
./test/lima/test-e2e.sh

# 3. Cleanup
./test/lima/cleanup.sh
```

### Individual Tests

```bash
# Just cluster formation
./test/lima/test-cluster.sh

# Just leader failover (requires cluster running)
./test/lima/test-failover.sh
```

### Development Workflow

```bash
# Setup VMs once
./test/lima/setup.sh

# Iterate on code
make build

# Re-run tests (skip setup)
./test/lima/test-e2e.sh --skip-setup

# Keep VMs for debugging
./test/lima/test-e2e.sh --skip-cleanup
```

## Prerequisites

### Install Lima

**macOS:**
```bash
brew install lima
```

**Linux (Ubuntu/Debian):**
```bash
curl -fsSL https://lima-vm.io/install.sh | sh
```

**Other Linux:**
See [Lima installation guide](https://lima-vm.io/docs/installation/)

### Build Warren

```bash
cd /path/to/warren
make build
```

## Troubleshooting

### Common Issues

**Issue:** VMs won't start
```bash
# Check Lima version (need 0.16.0+)
limactl --version

# View VM logs
limactl shell warren-manager-1 -- journalctl -xe

# Delete and recreate
./test/lima/cleanup.sh
./test/lima/setup.sh
```

**Issue:** Can't reach VMs via .internal hostnames
```bash
# Inside VM, check mDNS
limactl shell warren-manager-1
systemctl status systemd-resolved

# Test resolution
nslookup lima-warren-manager-2.internal

# Restart if needed
sudo systemctl restart systemd-resolved
```

**Issue:** Containerd not working
```bash
limactl shell warren-manager-1
sudo systemctl status containerd
sudo journalctl -u containerd -f
```

**Issue:** Tests timeout
```bash
# View Warren logs
limactl shell warren-manager-1
tail -f /tmp/warren-manager-1.log

# Check processes
ps aux | grep warren

# Kill stuck processes
sudo pkill -9 warren
```

## CI/CD Integration

### GitHub Actions

Lima works in GitHub Actions (macOS runners):

```yaml
name: Warren Integration Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: macos-latest
    steps:
      - uses: actions/checkout@v3

      - name: Install Lima
        run: brew install lima

      - name: Build Warren
        run: make build

      - name: Run Tests
        run: ./test/lima/test-e2e.sh

      - name: Cleanup
        if: always()
        run: ./test/lima/cleanup.sh --force
```

### Linux CI

Lima also works on Linux (Ubuntu runners):

```yaml
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Install Lima
        run: |
          curl -fsSL https://lima-vm.io/install.sh | sh
          echo "$HOME/.local/bin" >> $GITHUB_PATH

      - name: Build Warren
        run: make build

      - name: Run Tests
        run: ./test/lima/test-e2e.sh
```

## Performance

### VM Resource Usage

Per VM:
- **CPU:** 2 cores (configurable in `warren.yaml`)
- **Memory:** 2GB (configurable)
- **Disk:** 20GB (thin-provisioned)

Total for 5 VMs:
- **CPU:** ~10 cores (shared)
- **Memory:** ~10GB
- **Disk:** ~100GB (thin-provisioned, actual usage ~10-20GB)

### Test Duration

Typical timings on 2021 M1 MacBook Pro:

| Operation | Duration |
|-----------|----------|
| VM Creation (5 VMs) | ~3-5 minutes |
| Cluster Formation Test | ~30-60 seconds |
| Leader Failover Test | ~20-30 seconds |
| Full E2E Suite | ~2-3 minutes (excl. setup) |

## Best Practices

1. **Reuse VMs:** Use `--skip-setup` for faster iteration
2. **Debug Locally:** Use `--skip-cleanup` to inspect VM state
3. **View Logs:** Check `/tmp/warren-*.log` inside VMs
4. **Resource Tuning:** Adjust `warren.yaml` for your hardware
5. **Network Debugging:** Use `limactl shell` + `curl` to test connectivity
6. **Clean State:** Run `./test/lima/setup.sh --clean` for fresh environment

## Future Enhancements

Potential improvements for testing infrastructure:

- [ ] Chaos testing (random node failures)
- [ ] Network partition simulation (iptables rules)
- [ ] Performance benchmarking (throughput, latency)
- [ ] Multi-datacenter simulation (network delay)
- [ ] Automated test result reporting
- [ ] Integration with GitHub Actions
- [ ] Test coverage metrics
- [ ] Load testing (100+ services, 1000+ tasks)

## References

- [Lima Documentation](https://lima-vm.io/docs/)
- [Lima GitHub](https://github.com/lima-vm/lima)
- [Warren Test Scripts](../../test/lima/)
- [Test README](../../test/lima/README.md)
- [Milestone 2 Tasks](../../tasks/todo.md#milestone-2-high-availability)

## Questions?

Check the [test/lima/README.md](../../test/lima/README.md) for detailed usage instructions and troubleshooting.
