# Networking in Warren

Warren provides built-in networking for secure communication between containers and services. This guide explains how networking works in Warren.

## Overview

Warren's networking is built on four key components:

1. **WireGuard Overlay** - Encrypted mesh network between all nodes
2. **DNS Service Discovery** - Built-in DNS for service name resolution ✅ NEW
3. **Service VIPs** - Virtual IPs for load-balanced service access (planned)
4. **Container Networking** - Per-container IP assignment

## Network Architecture

```
┌─────────────────────────────────────────────────────────┐
│                   Warren Cluster                         │
│                                                           │
│  ┌──────────────┐         ┌──────────────┐              │
│  │  Manager 1   │         │  Worker 1    │              │
│  │  10.0.0.1    │◄───────►│  10.0.0.2    │              │
│  │              │ WireGuard│              │              │
│  │  [warren]    │         │  [nginx-1]   │ 10.0.1.5     │
│  └──────────────┘         │  [nginx-2]   │ 10.0.1.6     │
│         ▲                 └──────────────┘              │
│         │                                                 │
│         │       WireGuard Mesh                           │
│         │                                                 │
│         └─────────────────┐                              │
│                            │                              │
│                     ┌──────▼──────┐                      │
│                     │  Worker 2   │                      │
│                     │  10.0.0.3   │                      │
│                     │             │                      │
│                     │  [nginx-3]  │ 10.0.1.7            │
│                     │  [redis-1]  │ 10.0.1.8            │
│                     └─────────────┘                      │
│                                                           │
│  Service VIP:                                            │
│    nginx → 10.1.0.5 (load-balanced to nginx-1,2,3)      │
│    redis → 10.1.0.6 (routes to redis-1)                 │
└─────────────────────────────────────────────────────────┘
```

## WireGuard Overlay Network

### What is WireGuard?

[WireGuard](https://www.wireguard.com/) is a modern VPN protocol that creates secure tunnels between nodes.

**Benefits:**
- **Encryption** - All traffic encrypted (ChaCha20-Poly1305)
- **Performance** - Minimal overhead, line-rate speeds
- **Simplicity** - Easy configuration, small codebase
- **NAT Traversal** - Works across NAT boundaries
- **Battery Friendly** - Low power consumption (edge devices)

### How Warren Uses WireGuard

Warren creates a **full mesh network** where every node can communicate with every other node:

```
Manager-1 ←→ Manager-2
    ↓  ╲        ↓
    ↓    ╲      ↓
Worker-1 ←→ Worker-2
```

**IP Allocation:**
- **Managers**: 10.0.0.x
- **Workers**: 10.0.0.x (continued)
- **Containers**: 10.0.1.x

### WireGuard Configuration (Automatic)

Warren handles WireGuard configuration automatically:

```bash
# Initialize cluster (generates WireGuard keys)
warren cluster init

# Join as manager (exchanges keys automatically)
warren manager join --token <token> --manager <leader>

# Join as worker (exchanges keys automatically)
warren worker start --manager <manager-ip>
```

**What happens:**
1. Node generates WireGuard keypair (private, public)
2. Node registers with manager (sends public key)
3. Manager distributes peer configs to all nodes
4. Each node configures local WireGuard interface
5. Mesh network established

### Manual WireGuard Inspection

```bash
# View WireGuard interface (Linux)
sudo wg show warren0

# Output:
# interface: warren0
#   public key: ABC123...
#   private key: (hidden)
#   listening port: 51820
#
# peer: XYZ789...
#   endpoint: 192.168.1.10:51820
#   allowed ips: 10.0.0.1/32
#   latest handshake: 30 seconds ago
#   transfer: 1.5 MiB received, 2.3 MiB sent
```

### WireGuard Troubleshooting

**Check interface exists:**
```bash
ip link show warren0
```

**Check routing:**
```bash
ip route show table main | grep warren0
```

**Check handshake:**
```bash
sudo wg show warren0 latest-handshakes
```

**Common Issues:**
- **Firewall blocking UDP 51820** - Ensure UDP port 51820 is open
- **NAT traversal issues** - Use keepalive packets (automatic)
- **Key mismatch** - Re-join cluster to regenerate keys

## Service VIPs (Virtual IPs)

### What is a VIP?

A **Virtual IP** is a single IP address that load-balances traffic across all replicas of a service.

**Example:**
```bash
# Create service with 3 replicas
warren service create nginx --image nginx:latest --replicas 3

# Warren assigns VIP: 10.1.0.5
# Traffic to 10.1.0.5 → nginx-1 (10.0.1.10) OR
#                     → nginx-2 (10.0.1.11) OR
#                     → nginx-3 (10.0.1.12)
```

### VIP Allocation

VIPs are allocated from a dedicated subnet:

- **VIP Subnet**: 10.1.0.0/16
- **Allocation**: Sequential (10.1.0.1, 10.1.0.2, ...)
- **Persistence**: VIP remains constant for service lifetime

### Load Balancing

Warren uses **iptables NAT** for load balancing:

```bash
# Example iptables rules (automatic)
-A PREROUTING -d 10.1.0.5/32 -p tcp -m tcp --dport 80 \
  -m statistic --mode random --probability 0.33 -j DNAT --to-destination 10.0.1.10:80

-A PREROUTING -d 10.1.0.5/32 -p tcp -m tcp --dport 80 \
  -m statistic --mode random --probability 0.50 -j DNAT --to-destination 10.0.1.11:80

-A PREROUTING -d 10.1.0.5/32 -p tcp -m tcp --dport 80 \
  -j DNAT --to-destination 10.0.1.12:80
```

**Algorithm**: Random (equal probability)

**Characteristics:**
- **Stateless** - No connection tracking overhead
- **Simple** - Minimal CPU usage
- **Fair** - Equal distribution over time

### Accessing Services via VIP

**From within cluster:**
```bash
# Any container can access nginx service
curl http://10.1.0.5
```

**From manager node:**
```bash
# Manager can route to VIP
curl http://10.1.0.5
```

**Future: DNS-based discovery** (M6)
```bash
# Access by service name
curl http://nginx.warren.local
```

## Container Networking

### Container IP Assignment

Each task gets a unique IP from the container subnet:

- **Subnet**: 10.0.1.0/16
- **Allocation**: Sequential per worker
- **Isolation**: containerd namespace isolation

**Example:**
```
Worker 1:
  - nginx-task-1 → 10.0.1.5
  - nginx-task-2 → 10.0.1.6

Worker 2:
  - nginx-task-3 → 10.0.1.7
  - redis-task-1 → 10.0.1.8
```

### Container-to-Container Communication

**Same worker:**
- Direct communication via bridge network
- No encryption needed (localhost)

**Different workers:**
- Traffic routed through WireGuard tunnel
- Encrypted automatically
- Transparent to containers

**Example:**
```bash
# Container on Worker 1 accessing container on Worker 2
# nginx-task-1 (10.0.1.5) → redis-task-1 (10.0.1.8)
# Traffic encrypted via WireGuard automatically
```

### Network Isolation

Warren uses **containerd namespaces** for isolation:

```bash
# Warren namespace
ctr -n warren containers list

# Prevents interference with other containerd users
# (e.g., Docker, Kubernetes, etc.)
```

## Port Mapping (Future)

In M6, Warren will support publishing ports to host:

```bash
# Future: Publish container port to host
warren service create nginx \
  --image nginx:latest \
  --publish 80:80 \
  --replicas 3
```

This will make services accessible from outside the cluster.

## Network Policies (Future)

In M7, Warren will support network policies for fine-grained access control:

```yaml
# Future: Network policy
apiVersion: warren.io/v1
kind: NetworkPolicy
metadata:
  name: api-policy
spec:
  podSelector:
    matchLabels:
      app: api
  ingress:
    - from:
      - podSelector:
          matchLabels:
            app: frontend
      ports:
        - protocol: TCP
          port: 8080
```

## Service Discovery

### DNS-based Service Discovery ✅ NEW

Warren includes a built-in DNS server for service discovery. Containers can discover services by name instead of hardcoded IPs.

**How it works:**

1. Manager runs DNS server on port 53 (127.0.0.11)
2. Workers auto-configure containers with Warren DNS
3. Containers query DNS → Manager resolves → Service IPs returned
4. External DNS fallback (8.8.8.8, 1.1.1.1) for internet domains

**DNS Resolution Patterns:**

```bash
# From within a container:

# Service name → Round-robin instance IPs
ping nginx
# Resolves to: 10.0.1.5, 10.0.1.6, 10.0.1.7 (all instances)

# With domain suffix
ping nginx.warren
# Same as above

# Specific instance (instance-1, instance-2, etc.)
ping nginx-1
# Resolves to: 10.0.1.5 (first instance by creation time)

ping nginx-2
# Resolves to: 10.0.1.6 (second instance)

# External domains (forwarded to 8.8.8.8)
ping google.com
# Works! Forwarded to upstream DNS
```

**Container DNS Configuration:**

Warren automatically configures containers with custom resolv.conf:

```
# /etc/resolv.conf inside containers
nameserver <manager-ip>     # Primary: Warren DNS
nameserver 8.8.8.8          # Fallback: Google DNS
nameserver 1.1.1.1          # Fallback: Cloudflare DNS
search warren               # Search domain
options ndots:0             # Use search immediately
```

**Benefits:**

- ✅ **No hardcoded IPs** - Services discoverable by name
- ✅ **Load balancing** - DNS returns all healthy instances
- ✅ **Instance-specific** - Access specific replicas (nginx-1, nginx-2)
- ✅ **Automatic** - Zero configuration required
- ✅ **Fallback** - External DNS for internet domains
- ✅ **Docker-compatible** - Uses 127.0.0.11:53 convention

**Example Usage:**

```bash
# Deploy a service
warren service create nginx --image nginx --replicas 3

# From another container:
# Access the service by name (load-balanced)
curl http://nginx

# Access specific instance
curl http://nginx-1

# Works with domain suffix too
curl http://nginx.warren
```

**DNS Server Details:**

- **Location**: Runs on all manager nodes
- **Port**: 127.0.0.11:53 (Docker-compatible)
- **Protocol**: UDP
- **Resolution**: Service name → Healthy instance IPs
- **Load Balancing**: Round-robin via multiple A records
- **TTL**: 10 seconds (dynamic services)
- **Upstream**: 8.8.8.8 (Google), 1.1.1.1 (Cloudflare)

**Testing DNS:**

```bash
# From within a container
nslookup nginx
# Output:
# Server:    <manager-ip>
# Address:   <manager-ip>:53
#
# Name:      nginx
# Address:   10.0.1.5
# Address:   10.0.1.6
# Address:   10.0.1.7

# Query specific instance
nslookup nginx-1
# Output:
# Name:      nginx-1
# Address:   10.0.1.5

# External domain
nslookup google.com
# Works! Forwarded to upstream DNS
```

### VIP-based (Future - Phase 6.2)

VIP infrastructure is planned for Phase 6.2 (Published Ports):

- Each service will get a Virtual IP (VIP) from 10.1.0.0/16
- DNS will return VIPs instead of instance IPs
- iptables DNAT will load-balance VIP → instances
- Enables ingress mode port publishing

**Current:** DNS returns instance IPs directly (round-robin)
**Future:** DNS returns VIP → iptables routes to instances

## Multi-Cluster Networking (Future)

In M8, Warren will support federation across clusters:

```
Cluster A (US-East)     Cluster B (EU-West)
       ↓                        ↓
   Global Load Balancer
```

**Features:**
- Cross-cluster service discovery
- Global load balancing
- Federated secrets and configs

## Performance Characteristics

### Throughput

- **WireGuard Overhead**: < 5% CPU at 1 Gbps
- **VIP NAT Overhead**: < 1% CPU
- **Latency**: +1-2ms (WireGuard encryption)

### Scalability

- **Nodes**: 100+ nodes in mesh
- **Services**: 1000+ VIPs
- **Connections**: Limited by kernel (100k+ concurrent)

### Optimization Tips

1. **Use VIPs for load balancing** - More efficient than external LB
2. **Collocate communicating services** - Reduce cross-worker traffic
3. **Monitor WireGuard handshakes** - Ensure mesh is healthy
4. **Use keepalive** - Maintain NAT traversal (automatic)

## Network Troubleshooting

### Check WireGuard Status

```bash
# View WireGuard interface
sudo wg show warren0

# Check if peers are connected
sudo wg show warren0 latest-handshakes

# Should show handshakes within last 2 minutes
```

### Test Connectivity

```bash
# Ping manager from worker (via WireGuard)
ping 10.0.0.1

# Ping container from manager
ping 10.0.1.5
```

### Check VIP Routing

```bash
# View iptables NAT rules
sudo iptables -t nat -L -n -v | grep 10.1.0.5

# Verify DNAT rules exist for service VIP
```

### Debug Container Networking

```bash
# List containers in Warren namespace
sudo ctr -n warren containers list

# Get container task info
sudo ctr -n warren tasks list

# Exec into container for debugging
sudo ctr -n warren tasks exec -t --exec-id debug <container-id> sh

# Inside container:
# - Check IP: ip addr show
# - Check routes: ip route show
# - Test connectivity: ping 10.0.1.x
```

### Common Issues

**DNS not resolving service names:**

```bash
# 1. Check DNS server is running on manager
# On manager node:
sudo netstat -tulpn | grep 53
# Should show warren process listening on 127.0.0.11:53

# 2. Check container DNS configuration
# From inside container:
cat /etc/resolv.conf
# Should show: nameserver <manager-ip>

# 3. Test DNS resolution manually
nslookup nginx
# Should resolve to instance IPs

# 4. Check if service has running instances
warren service inspect nginx
# Verify ActualState: running

# 5. Test external DNS (fallback)
nslookup google.com
# Should work (forwarded to 8.8.8.8)

# Common fixes:
# - Restart manager if DNS server not running
# - Check manager IP is reachable from worker
# - Verify service has healthy instances
```

**Cannot ping containers:**
- Check WireGuard is up: `ip link show warren0`
- Check routes: `ip route show | grep warren0`
- Check firewall: `sudo iptables -L -n`

**Service VIP not working (future feature):**
- Check iptables NAT rules exist
- Verify service has running tasks
- Check task IPs are reachable

**Cross-worker traffic fails:**
- Check WireGuard handshakes
- Verify UDP 51820 is open on firewall
- Check allowed-ips in WireGuard config

## Security Considerations

### WireGuard Encryption

All inter-node traffic is encrypted:

- **Algorithm**: ChaCha20-Poly1305 (AEAD)
- **Key Exchange**: Curve25519
- **MAC**: Poly1305
- **Perfect Forward Secrecy**: Yes (rotating keys)

### Network Isolation

- **Namespace Isolation**: Warren containers isolated from other containerd users
- **Overlay Traffic**: Encrypted via WireGuard (no plaintext)
- **VIP Access**: Only from within cluster (not externally routable)

### Future: mTLS (M6)

Warren will support mutual TLS for API communication:

```bash
warren cluster init --enable-mtls
```

## Best Practices

### 1. Use DNS Service Names ✅

```bash
# Bad: Hardcoded IP (breaks on container restart)
curl http://10.0.1.5

# Good: Service name (load-balanced, discoverable)
curl http://nginx

# Also good: With domain suffix
curl http://nginx.warren

# Advanced: Specific instance
curl http://nginx-1
```

**Benefits:**
- Load-balanced automatically (round-robin)
- Resilient to container restarts (IPs may change)
- Readable and maintainable
- Works across all environments

### 2. Use Search Domain

With `search warren` in resolv.conf, you can use short names:

```bash
# Both work:
ping nginx              # ✅ Tries nginx.warren automatically
ping nginx.warren       # ✅ Explicit

# External domains still work:
ping google.com         # ✅ Forwarded to 8.8.8.8
```

### 3. Collocate Related Services

Deploy services that communicate frequently on same worker:

```bash
# Good: App + cache on same node (reduces network hops)
warren service create app --replicas 3 --constraint "node.id==worker-1"
warren service create cache --replicas 1 --constraint "node.id==worker-1"
```

(Constraint support coming in M6)

### 4. Monitor WireGuard Health

```bash
# Check handshakes regularly
sudo wg show warren0 latest-handshakes

# Alert if no handshake > 5 minutes
```

### 5. Test DNS Resolution

```bash
# From within container:
nslookup nginx        # Should resolve to instance IPs
ping nginx-1          # Should resolve to specific instance
nslookup google.com   # Should resolve via external DNS
```

### 6. Plan IP Ranges

Default IP ranges:

- **WireGuard**: 10.0.0.0/16 (65k nodes)
- **Containers**: 10.0.1.0/16 (65k containers)
- **VIPs**: 10.1.0.0/16 (65k services)

Ensure no overlap with existing networks.

## Network Configuration Reference

### Default Settings

```go
// WireGuard
ListenPort:    51820           // UDP port for WireGuard
KeepaliveInterval: 25s         // NAT traversal keepalive

// IP Allocation
WireGuardSubnet:  "10.0.0.0/16"
ContainerSubnet:  "10.0.1.0/16"
VIPSubnet:        "10.1.0.0/16"

// VIP Load Balancing
Algorithm:        "random"      // Random distribution
Persistence:      false         // No session affinity
```

### Future: Custom Configuration (M6)

```bash
warren cluster init \
  --wireguard-subnet 172.16.0.0/16 \
  --container-subnet 172.17.0.0/16 \
  --vip-subnet 172.18.0.0/16
```

## Next Steps

- **[Storage](storage.md)** - Volumes and persistent data
- **[High Availability](high-availability.md)** - Multi-manager resilience
- **[Architecture](architecture.md)** - Overall system architecture

---

**Questions?** See [Troubleshooting](../troubleshooting.md) or ask in [GitHub Discussions](https://github.com/cuemby/warren/discussions).
