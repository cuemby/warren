# ADR-003: Use WireGuard for Overlay Networking

**Status**: Accepted
**Date**: 2025-10-09

## Decision

**Use WireGuard for Warren's encrypted overlay network**, with kernel module on Linux and userspace fallback (wireguard-go) for macOS/Windows.

## Context

Warren needs secure overlay networking for:
- Container-to-container communication across nodes
- Service VIPs and load balancing
- DNS resolution

Options:
1. **WireGuard** - Modern VPN protocol
2. **VXLAN** - Virtual extensible LAN
3. **Flannel** - Kubernetes overlay network
4. **Weave** - Container network

## Rationale

**Chose WireGuard because**:

✅ **Kernel-native** (Linux 5.6+): Excellent performance
✅ **Encrypted by default**: ChaCha20-Poly1305, modern cryptography
✅ **Simple**: Minimal configuration, easy to understand
✅ **Performant**: > 90% of native network speed
✅ **Cross-platform**: Kernel (Linux) + userspace (macOS/Windows)
✅ **Audited**: Security audits, widely trusted
✅ **Go library**: `wgctrl` for configuration

## Alternatives Rejected

### VXLAN
❌ No encryption (requires separate IPsec)
❌ More complex configuration
✓ Slightly better performance (but marginal)

### Flannel
❌ Designed for Kubernetes (Warren is independent)
❌ Additional dependency/daemon
❌ More complex than needed

### Weave
❌ Heavier (separate network daemon)
❌ More features than Warren needs
❌ Less performant than WireGuard

## Implementation

**Linux** (kernel module):
```go
// Use netlink to create interface
link := &netlink.Wireguard{Name: "wg0"}
netlink.LinkAdd(link)

// Configure via wgctrl
client, _ := wgctrl.New()
client.ConfigureDevice("wg0", wgtypes.Config{
    PrivateKey: privateKey,
    ListenPort: 51820,
    Peers: []wgtypes.PeerConfig{...},
})
```

**macOS/Windows** (userspace):
```go
// Use wireguard-go library
// Fallback when kernel module unavailable
```

## Consequences

✅ Encrypted by default (zero-config security)
✅ High performance (kernel fast path on Linux)
✅ Simple configuration (just keys + endpoints)
✅ Works across platforms
⚠️ Requires kernel 5.6+ on Linux (2020+, widely available)
⚠️ Userspace mode slower on macOS (acceptable trade-off)
⚠️ Requires root/CAP_NET_ADMIN for interface creation

## Validation

See [poc/wireguard/](../../poc/wireguard/) - 3-node mesh, throughput > 90% native.

**Status**: Accepted - POC confirms performance target
