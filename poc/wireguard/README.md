# WireGuard POC - Encrypted Mesh Network

## Overview

This POC validates WireGuard for Warren's overlay networking:
- Create WireGuard interfaces
- Configure full mesh (3 nodes)
- Test encrypted connectivity
- Measure throughput (target: > 90% native)

## Prerequisites

### Linux
```bash
# WireGuard kernel module (Linux 5.6+)
# Usually built-in, check with:
modprobe wireguard
lsmod | grep wireguard

# Install tools
sudo apt install wireguard-tools  # Ubuntu/Debian
# or
sudo yum install wireguard-tools  # RHEL/CentOS
```

### macOS
```bash
# Note: macOS doesn't have kernel WireGuard
# This POC demonstrates the control API
# Warren will use wireguard-go (userspace) for macOS

# Install WireGuard tools for testing
brew install wireguard-tools
```

## Running the POC

```bash
cd poc/wireguard
go mod download

# Run (requires sudo)
sudo go run .
```

## Manual 3-Node Mesh Setup

### Node 1 (10.0.0.1)

```bash
# Generate keys
wg genkey | tee privatekey1 | wg pubkey > publickey1

# Create interface
sudo ip link add wg0 type wireguard
sudo ip addr add 10.0.0.1/24 dev wg0

# Configure WireGuard
sudo wg set wg0 private-key ./privatekey1
sudo wg set wg0 listen-port 51820

# Add peers (Node 2 and 3)
# Replace <publickey2>, <node2-ip>, etc. with actual values
sudo wg set wg0 peer <publickey2> endpoint <node2-ip>:51820 allowed-ips 10.0.0.2/32
sudo wg set wg0 peer <publickey3> endpoint <node3-ip>:51820 allowed-ips 10.0.0.3/32

# Bring up interface
sudo ip link set wg0 up
```

### Node 2 (10.0.0.2)

```bash
# Generate keys
wg genkey | tee privatekey2 | wg pubkey > publickey2

# Create interface
sudo ip link add wg0 type wireguard
sudo ip addr add 10.0.0.2/24 dev wg0

# Configure WireGuard
sudo wg set wg0 private-key ./privatekey2
sudo wg set wg0 listen-port 51820

# Add peers (Node 1 and 3)
sudo wg set wg0 peer <publickey1> endpoint <node1-ip>:51820 allowed-ips 10.0.0.1/32
sudo wg set wg0 peer <publickey3> endpoint <node3-ip>:51820 allowed-ips 10.0.0.3/32

# Bring up interface
sudo ip link set wg0 up
```

### Node 3 (10.0.0.3)

```bash
# Similar to Node 2, with IP 10.0.0.3
```

## Test Scenarios

### 1. Interface Creation

**Expected**: Successfully create wg0 interface
**Result**: ✅ PASS / ❌ FAIL

```bash
# Verify interface
ip link show wg0
```

**Observed**:
```
# Output from ip link
```

### 2. Mesh Connectivity

**Expected**: Each node can ping others via overlay IPs

**Test**:
```bash
# From Node 1
ping -c 5 10.0.0.2
ping -c 5 10.0.0.3

# From Node 2
ping -c 5 10.0.0.1
ping -c 5 10.0.0.3
```

**Result**: ✅ PASS / ❌ FAIL

**Observed**:
```
# Ping results
```

### 3. Encryption Verification

**Expected**: Traffic is encrypted (ChaCha20-Poly1305)

```bash
# Check WireGuard status
sudo wg show wg0

# Capture traffic (should be encrypted)
sudo tcpdump -i wg0 -c 10
```

**Result**: ✅ PASS / ❌ FAIL

**Observed**:
```
# wg show output
```

### 4. Throughput Test

**Expected**: > 90% of native network speed

**Setup**:
```bash
# Install iperf3
sudo apt install iperf3  # Linux
brew install iperf3      # macOS
```

**Test**:
```bash
# Node 1: Start server
iperf3 -s

# Node 2: Test to Node 1
# Native (direct IP)
iperf3 -c <node1-real-ip>

# Over WireGuard
iperf3 -c 10.0.0.1
```

**Result**: ✅ PASS / ❌ FAIL

**Measurements**:
- Native throughput: ___ Gbps
- WireGuard throughput: ___ Gbps
- Percentage: ___%

**Observed**:
```
# iperf3 results
```

### 5. Latency Test

**Expected**: Minimal latency overhead (< 1ms)

```bash
# Native ping
ping -c 100 <node2-real-ip> | tail -1

# WireGuard ping
ping -c 100 10.0.0.2 | tail -1
```

**Result**: ✅ PASS / ❌ FAIL

**Measurements**:
- Native latency: ___ ms
- WireGuard latency: ___ ms
- Overhead: ___ ms

### 6. Peer Handshake

**Expected**: Peers establish connection automatically

```bash
# Check handshake status
sudo wg show wg0 latest-handshakes

# Should show recent timestamp for each peer
```

**Result**: ✅ PASS / ❌ FAIL

**Observed**:
```
# Handshake times
```

## Performance Measurements

### Throughput (1 Gbps Network)

| Test | Bandwidth | CPU Usage |
|------|-----------|-----------|
| Native | ___ Gbps | ___% |
| WireGuard | ___ Gbps | ___% |
| Overhead | ___% | ___% |

### Latency

| Test | Min | Avg | Max | Mdev |
|------|-----|-----|-----|------|
| Native | ___ ms | ___ ms | ___ ms | ___ ms |
| WireGuard | ___ ms | ___ ms | ___ ms | ___ ms |

## Conclusions

### Success Criteria

- [ ] WireGuard interface created successfully
- [ ] 3-node mesh connectivity working
- [ ] All nodes can reach each other
- [ ] Traffic is encrypted
- [ ] Throughput > 90% of native
- [ ] Latency overhead < 1ms
- [ ] Peer handshakes successful

### Go/No-Go Decision

**Decision**: ✅ GO / ❌ NO-GO

**Rationale**:
```
# Why WireGuard meets (or doesn't meet) Warren's requirements
```

### Issues Discovered

```
# Problems:
# - Platform-specific setup (Linux kernel vs userspace)
# - Requires root/sudo
# - Manual peer configuration complexity

# Mitigations:
# - Warren will abstract platform differences
# - Run as privileged container or system service
# - Automate peer config via Raft state distribution
```

### Recommendations for Warren Implementation

```
# Architecture:
# - Use wgctrl library for configuration
# - Use netlink (Linux) or wireguard-go (macOS/Windows)
# - Distribute peer keys via Raft
# - Auto-configure peers when nodes join

# Configuration:
# - Persistent keepalive: 25 seconds (for NAT)
# - Listen port: 51820 (default)
# - Allowed IPs: Individual /32 for each peer

# Security:
# - Rotate keys periodically (e.g., 90 days)
# - Store private keys encrypted in Raft
```

## Next Steps

If GO:
- [ ] Proceed to binary size POC
- [ ] Design Warren's network manager component
- [ ] Document WireGuard integration strategy

If NO-GO:
- [ ] Investigate alternatives (VXLAN, Flannel, Weave)
- [ ] Document blockers and performance issues
