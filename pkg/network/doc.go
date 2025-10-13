/*
Package network provides host port publishing for Warren services using iptables.

The network package implements host mode port publishing, allowing services to
expose ports on the host's network interface. It uses iptables NAT rules to
forward incoming traffic from host ports to container IP addresses, providing
direct connectivity without overlay network overhead.

# Architecture

Warren's network package handles host port publishing through iptables DNAT
and MASQUERADE rules:

	┌────────────────── HOST PORT PUBLISHING ──────────────────┐
	│                                                            │
	│  ┌────────────────────────────────────────────┐          │
	│  │      HostPortPublisher                      │          │
	│  │  - Tracks published ports per task          │          │
	│  │  - Manages iptables rule lifecycle          │          │
	│  │  - Prevents port conflicts                  │          │
	│  └──────────────────┬─────────────────────────┘          │
	│                     │                                      │
	│  ┌──────────────────▼─────────────────────────┐          │
	│  │         Port Publishing Flow                │          │
	│  │                                              │          │
	│  │  1. Client → Host Port (e.g., :80)         │          │
	│  │  2. PREROUTING: DNAT rule intercepts        │          │
	│  │  3. Rewrite dest: Container IP:Port         │          │
	│  │  4. FORWARD: Allow packet                   │          │
	│  │  5. POSTROUTING: MASQUERADE for return      │          │
	│  │  6. Container receives packet               │          │
	│  └──────────────────┬─────────────────────────┘          │
	│                     │                                      │
	│  ┌──────────────────▼─────────────────────────┐          │
	│  │          iptables Rules                     │          │
	│  │                                              │          │
	│  │  PREROUTING (nat):                          │          │
	│  │    -p tcp --dport 80 → 10.88.0.5:8080      │          │
	│  │    -p tcp --dport 443 → 10.88.0.5:8443     │          │
	│  │                                              │          │
	│  │  POSTROUTING (nat):                         │          │
	│  │    -d 10.88.0.5 --dport 8080 → MASQUERADE  │          │
	│  │    -d 10.88.0.5 --dport 8443 → MASQUERADE  │          │
	│  │                                              │          │
	│  │  FORWARD (filter):                          │          │
	│  │    -d 10.88.0.5 --dport 8080 → ACCEPT      │          │
	│  │    -d 10.88.0.5 --dport 8443 → ACCEPT      │          │
	│  └────────────────────────────────────────────┘           │
	│                                                            │
	│  ┌────────────────────────────────────────────┐          │
	│  │         Traffic Path                        │          │
	│  │                                              │          │
	│  │  Internet → Host:80 (DNAT) →               │          │
	│  │  Container 10.88.0.5:8080 (FORWARD) →      │          │
	│  │  Response (MASQUERADE) → Client             │          │
	│  └────────────────────────────────────────────┘           │
	└────────────────────────────────────────────────────────┘

# Core Components

HostPortPublisher:
  - Manages lifecycle of iptables rules for port publishing
  - Tracks published ports per task for cleanup
  - Implements rollback on partial failure
  - Thread-safe via internal state management

Port Mapping:
  - HostPort: Port on host network (e.g., 80, 443)
  - ContainerPort: Port inside container (e.g., 8080, 8443)
  - Protocol: tcp or udp
  - PublishMode: "host" for host port publishing

iptables Operations:
  - PREROUTING: Destination NAT (host port → container IP:port)
  - POSTROUTING: Masquerade for return traffic
  - FORWARD: Allow forwarded packets through filter table
  - Cleanup: Delete rules on task removal

# Port Publishing

Publish Modes:

Host Mode (PublishMode: "host"):
  - Ports published on node running the task
  - Direct connection without load balancing
  - Best for single-replica services or sticky sessions
  - Example: Database with primary-replica setup

Ingress Mode (PublishMode: "ingress" - future):
  - Ports published on all manager nodes
  - Load balanced across replicas via VIP
  - Best for stateless web services
  - Example: HTTP/HTTPS services

Port Conflict Detection:
  - Check if host port already in use (future)
  - Return error on conflict
  - Caller must handle port allocation
  - Recommendation: Use high ports (> 1024) when possible

# Usage

Creating a Publisher:

	publisher := network.NewHostPortPublisher()

Publishing Ports:

	ports := []types.PortMapping{
		{
			HostPort:      80,
			ContainerPort: 8080,
			Protocol:      "tcp",
			PublishMode:   types.PublishModeHost,
		},
		{
			HostPort:      443,
			ContainerPort: 8443,
			Protocol:      "tcp",
			PublishMode:   types.PublishModeHost,
		},
	}

	err := publisher.PublishPorts("task-abc123", "10.88.0.5", ports)
	if err != nil {
		log.Fatal(err)
	}

Unpublishing Ports:

	err := publisher.UnpublishPorts("task-abc123")
	if err != nil {
		log.Warn("Failed to unpublish ports: %v", err)
	}

Getting Published Ports:

	ports := publisher.GetPublishedPorts("task-abc123")
	for _, port := range ports {
		fmt.Printf("Host:%d → Container:%d (%s)\n",
			port.HostPort, port.ContainerPort, port.Protocol)
	}

Complete Example:

	package main

	import (
		"fmt"
		"github.com/cuemby/warren/pkg/network"
		"github.com/cuemby/warren/pkg/types"
	)

	func main() {
		publisher := network.NewHostPortPublisher()

		// Publish ports for nginx task
		ports := []types.PortMapping{
			{HostPort: 80, ContainerPort: 80, Protocol: "tcp", PublishMode: types.PublishModeHost},
		}

		containerIP := "10.88.0.5"
		taskID := "task-nginx-1"

		if err := publisher.PublishPorts(taskID, containerIP, ports); err != nil {
			fmt.Printf("Failed to publish ports: %v\n", err)
			return
		}

		fmt.Printf("Published port 80 for task %s\n", taskID)

		// ... task runs ...

		// Cleanup on task removal
		if err := publisher.UnpublishPorts(taskID); err != nil {
			fmt.Printf("Warning: Failed to cleanup ports: %v\n", err)
		}
	}

# Integration Points

This package integrates with:

  - pkg/worker: Publishes ports when starting tasks
  - pkg/types: PortMapping definitions
  - pkg/runtime: Gets container IP addresses
  - iptables: Linux netfilter for packet manipulation

# iptables Rules Explained

PREROUTING Chain (nat table):
  - Purpose: Rewrite destination before routing decision
  - Rule: Match host port, change dest to container IP:port
  - Example: -p tcp --dport 80 -j DNAT --to-destination 10.88.0.5:8080
  - Result: Packet now destined for container

POSTROUTING Chain (nat table):
  - Purpose: Masquerade source for return traffic
  - Rule: Match packets to container, rewrite source to host
  - Example: -p tcp -d 10.88.0.5 --dport 8080 -j MASQUERADE
  - Result: Container sees request from host, not original client

FORWARD Chain (filter table):
  - Purpose: Allow forwarded packets through firewall
  - Rule: Match packets to container port, accept
  - Example: -p tcp -d 10.88.0.5 --dport 8080 -j ACCEPT
  - Result: Packet not dropped by firewall

# Design Patterns

Rule Lifecycle:
  - Create: Three rules per port (PREROUTING, POSTROUTING, FORWARD)
  - Track: Store ports per task for cleanup
  - Delete: Remove all three rules on unpublish
  - Idempotent: Delete ignores missing rules

Rollback on Failure:
  - If any rule fails, remove previously created rules
  - Ensures consistent state (all or nothing)
  - Prevents partial port publishing
  - Caller can retry safely

Error Handling:
  - Rule creation errors returned immediately
  - Cleanup errors logged but not fatal
  - Missing rules during delete ignored
  - Prevents cascading failures

Protocol Support:
  - TCP: Fully supported (default if not specified)
  - UDP: Supported via protocol parameter
  - SCTP: Not supported (iptables limitation)

# Performance Characteristics

Rule Creation:
  - Per port: 3 iptables commands (~10-30ms each)
  - Total: ~30-100ms for typical service (1-3 ports)
  - Bottleneck: iptables lock (serialized updates)

Rule Lookup:
  - Linear scan of iptables rules per packet
  - Performance: O(n) where n = number of rules
  - Recommendation: Keep total rules < 1000 per node
  - Impact: Negligible for < 100 rules

Packet Processing:
  - NAT overhead: ~5-10µs per packet (negligible)
  - Connection tracking: First packet slow, rest fast
  - Throughput: Line rate (no significant overhead)
  - Latency: < 1ms additional latency

Cleanup:
  - Per task: 3 iptables commands per port
  - Async: Can run in background
  - Failures: Non-critical (rules cleaned on node restart)

# Troubleshooting

Common Issues:

Port Already in Use:
  - Symptom: "Address already in use" or silent failure
  - Check: netstat -tlnp | grep :80
  - Check: Existing iptables rules (iptables -t nat -L)
  - Solution: Use different host port or stop conflicting service

Rules Not Working:
  - Symptom: Connection refused or timeout
  - Check: iptables -t nat -L -n (PREROUTING rules)
  - Check: iptables -L FORWARD -n (FORWARD rules)
  - Check: Container is running and has correct IP
  - Solution: Verify rules exist and container IP is correct

Cannot Delete Rules:
  - Symptom: "No chain/target/match by that name"
  - Cause: Rules already deleted or never created
  - Impact: None (idempotent delete)
  - Solution: Ignore error or check rule existence first

iptables Command Fails:
  - Symptom: "Permission denied" or "command not found"
  - Check: Run as root or with CAP_NET_ADMIN
  - Check: iptables binary exists in PATH
  - Solution: Run Warren worker with appropriate privileges

Firewall Conflicts:
  - Symptom: Rules exist but traffic blocked
  - Check: firewalld or ufw may override rules
  - Check: Chain policy (should be ACCEPT or rules at top)
  - Solution: Configure firewall to work with iptables

# Monitoring

Key metrics to monitor:

Port Publishing:
  - network_ports_published_total: Number of ports currently published
  - network_publish_duration: Time to publish ports
  - network_publish_errors_total: Failed publish operations
  - network_unpublish_errors_total: Failed cleanup operations

iptables Health:
  - network_iptables_rules_total: Total rules managed by Warren
  - network_iptables_commands_total: iptables commands executed
  - network_iptables_errors_total: Failed iptables commands

Traffic:
  - network_published_port_connections: Connections per published port
  - network_published_port_bytes: Traffic through published ports
  - Note: Requires iptables byte counters or custom instrumentation

# Security

Port Exposure:
  - Host ports expose containers directly to network
  - No authentication or authorization at network layer
  - Recommendation: Use firewall to restrict source IPs
  - Best practice: Publish only necessary ports

Privilege Requirements:
  - iptables requires root or CAP_NET_ADMIN
  - Warren worker must run with appropriate permissions
  - Risk: Misconfigured rules can affect all network traffic
  - Mitigation: Validate rules before applying

Port Conflicts:
  - Multiple tasks cannot publish same host port
  - Current: No automated conflict detection
  - Risk: Silent failure or service disruption
  - Future: Port allocation and conflict prevention

DoS Protection:
  - No rate limiting at iptables layer
  - Recommendation: Use connection tracking limits
  - Example: -m connlimit --connlimit-above 100
  - Consider: External firewall or DDoS protection

# Limitations

Current Limitations:
  - No port conflict detection (caller responsible)
  - No port allocation (caller provides ports)
  - Cleanup requires container IP (must be tracked separately)
  - No IPv6 support (only IPv4)
  - No ingress mode (only host mode)

Future Enhancements:
  - Port pool allocation
  - Automatic conflict detection
  - IPv6 support via ip6tables
  - Ingress mode with VIP and load balancing
  - Connection draining for graceful shutdown

# Platform Support

Linux:
  - Fully supported (iptables required)
  - Kernel 3.10+ recommended
  - iptables 1.4.7+ required

macOS:
  - Not supported (no iptables)
  - Alternative: Use Lima VM with port forwarding
  - Future: Implement using pfctl

Windows:
  - Not supported (no iptables)
  - Alternative: Use WSL2 with iptables
  - Future: Implement using netsh or WinAPI

# See Also

  - pkg/worker for task lifecycle and port publishing
  - pkg/types for PortMapping definitions
  - pkg/runtime for container IP retrieval
  - iptables documentation: https://netfilter.org/documentation/
  - NAT tutorial: https://www.karlrupp.net/en/computer/nat_tutorial
*/
package network
