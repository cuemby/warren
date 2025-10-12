# Warren Ingress Controller

Warren includes a built-in ingress controller that provides HTTP/HTTPS routing to services without requiring external load balancers.

## Overview

The Warren ingress controller provides:

- **HTTP reverse proxy** on port 8000
- **HTTPS termination** on port 8443
- **Host-based routing** (api.example.com, web.example.com)
- **Path-based routing** (/api, /web, etc.)
- **TLS certificate management** (manual and Let's Encrypt)
- **Advanced routing** (headers, path rewriting, rate limiting, access control)
- **Load balancing** across service replicas

## Quick Start

### 1. Deploy a Service

First, deploy a service that you want to expose:

```bash
warren service create my-app \
  --image nginx:latest \
  --replicas 3 \
  --port 80 \
  --manager localhost:2377
```

### 2. Create Basic Ingress (HTTP)

Create an ingress to route HTTP traffic to your service:

```bash
warren ingress create my-ingress \
  --host app.example.com \
  --service my-app \
  --port 80 \
  --manager localhost:2377
```

Now requests to `http://app.example.com:8000` will be routed to your service.

### 3. Create Ingress with HTTPS (Let's Encrypt)

For automatic HTTPS with Let's Encrypt:

```bash
warren ingress create my-ingress \
  --host app.example.com \
  --service my-app \
  --port 80 \
  --tls \
  --tls-email admin@example.com \
  --manager localhost:2377
```

Warren will:
1. Create the ingress
2. Request a certificate from Let's Encrypt
3. Handle the HTTP-01 challenge automatically
4. Store the certificate securely
5. Start the HTTPS server on port 8443
6. Auto-renew the certificate (30 days before expiry)

## Architecture

### Components

1. **Router** - Matches requests to ingress rules (host + path)
2. **Load Balancer** - Selects healthy backend instances
3. **Middleware** - Handles headers, rate limiting, access control
4. **Proxy** - Forwards requests to backend services
5. **ACME Client** - Manages Let's Encrypt certificates

### Request Flow

```
Client Request
    ↓
[Port 8000 (HTTP) or 8443 (HTTPS)]
    ↓
[ACME Challenge Check] ← HTTP-01 challenges
    ↓
[Route Matching] ← Host + Path
    ↓
[Access Control] ← IP whitelist/blacklist
    ↓
[Rate Limiting] ← Per-IP limits
    ↓
[Header Manipulation] ← Add/Set/Remove headers
    ↓
[Path Rewriting] ← Strip prefix or replace
    ↓
[Load Balancer] ← Select healthy backend
    ↓
[Proxy to Backend]
    ↓
Backend Service
```

## Features

### Host-Based Routing

Route different hostnames to different services:

```bash
# API service
warren ingress create api-ingress \
  --host api.example.com \
  --service api-service \
  --port 8080

# Web service
warren ingress create web-ingress \
  --host web.example.com \
  --service web-service \
  --port 80
```

Wildcard hosts are supported:

```bash
warren ingress create wildcard-ingress \
  --host "*.example.com" \
  --service catch-all \
  --port 80
```

### Path-Based Routing

Route different paths to different services (currently via API):

```bash
# API handles /api/*
# Web handles /web/*
# App handles /*
```

Path matching supports:
- **Prefix matching**: `/api` matches `/api/users`, `/api/posts`
- **Exact matching**: `/api` matches only `/api`

Longest prefix wins for overlapping paths.

### TLS/HTTPS

#### Manual Certificates

Upload your own certificates:

```bash
# Create certificate
warren certificate create my-cert \
  --cert /path/to/cert.pem \
  --key /path/to/key.pem \
  --hosts app.example.com,www.app.example.com \
  --manager localhost:2377

# List certificates
warren certificate list --manager localhost:2377

# Inspect certificate
warren certificate inspect my-cert --manager localhost:2377

# Delete certificate
warren certificate delete my-cert --manager localhost:2377
```

#### Let's Encrypt (Automatic)

For automatic certificate issuance:

```bash
warren ingress create my-app \
  --host app.example.com \
  --service my-service \
  --port 80 \
  --tls \
  --tls-email admin@example.com
```

**Requirements**:
- Public domain name
- DNS pointing to Warren manager
- Port 80 accessible (for HTTP-01 challenge)

**Features**:
- Automatic certificate issuance
- HTTP-01 challenge handling
- Certificate renewal (30 days before expiry)
- Zero-downtime certificate updates
- Uses Let's Encrypt staging by default (see Production section)

### Proxy Headers

Warren automatically adds standard proxy headers to all requests:

- `X-Real-IP`: Original client IP
- `X-Forwarded-For`: Chain of proxy IPs
- `X-Forwarded-Proto`: Original protocol (http/https)
- `X-Forwarded-Host`: Original host header

These headers are useful for:
- Logging the real client IP in backend services
- Generating correct URLs in responses
- Security and rate limiting

### Advanced Features

The following features are implemented in the middleware but require API/protobuf configuration (future enhancement):

#### Header Manipulation

```go
// Add headers (only if not present)
Add: {"X-Custom-Header": "value"}

// Set headers (overwrite if present)
Set: {"X-Frame-Options": "DENY"}

// Remove headers
Remove: ["X-Powered-By", "Server"]
```

#### Path Rewriting

```go
// Strip prefix: /api/v1/users → /users
StripPrefix: "/api/v1"

// Replace path: /old → /new
ReplacePath: "/new"
```

#### Rate Limiting

```go
// Per-IP rate limiting with token bucket
RequestsPerSecond: 10
Burst: 20
// Returns HTTP 429 when exceeded
```

#### Access Control

```go
// IP whitelist/blacklist with CIDR support
AllowedIPs: ["192.168.1.0/24", "10.0.0.1"]
DeniedIPs: ["203.0.113.0/24"]
// Returns HTTP 403 when denied
```

## Load Balancing

Warren automatically load balances traffic across service replicas:

- **Algorithm**: Round-robin
- **Health checks**: Only routes to healthy tasks
- **Service discovery**: Uses Warren's built-in DNS

If a service has 3 replicas, traffic is distributed evenly across all healthy instances.

## Operations

### List Ingresses

```bash
warren ingress list --manager localhost:2377
```

### Inspect Ingress

```bash
warren ingress inspect my-ingress --manager localhost:2377
```

### Update Ingress

```bash
warren ingress update my-ingress \
  --service new-service \
  --port 8080 \
  --manager localhost:2377
```

### Delete Ingress

```bash
warren ingress delete my-ingress --manager localhost:2377
```

### Certificate Management

```bash
# List all certificates
warren certificate list

# View certificate details
warren certificate inspect my-cert

# Delete expired certificate
warren certificate delete old-cert
```

## Production Setup

### Let's Encrypt Production

By default, Warren uses Let's Encrypt **staging** environment for safety. To switch to production:

1. Edit `pkg/ingress/acme.go` line 135:

```go
// Change from:
config.CADirURL = "https://acme-staging-v02.api.letsencrypt.org/directory"

// To:
config.CADirURL = "https://acme-v02.api.letsencrypt.org/directory"
```

2. Rebuild Warren
3. Restart the manager

**Note**: Let's Encrypt has rate limits. Use staging for testing!

### DNS Configuration

For Let's Encrypt to work:

1. **Domain**: Must own a public domain
2. **DNS A Record**: Point to Warren manager's public IP
3. **Firewall**: Allow ports 80 (HTTP-01 challenge) and 443 (HTTPS)

Example:
```
app.example.com  →  A  →  203.0.113.10 (Warren manager IP)
```

### High Availability

In a multi-manager cluster:
- Certificates are replicated via Raft
- All managers can serve ingress traffic
- Use external load balancer or DNS round-robin to distribute to managers

### Monitoring

Warren provides Prometheus metrics on port 9090:
- `warren_ingress_requests_total` - Total requests processed
- `warren_ingress_request_duration_seconds` - Request latency
- `warren_ingress_backend_errors_total` - Backend errors

### Logging

Set log level for ingress debugging:

```bash
warren cluster init --log-level debug
```

Ingress logs include:
- Route matching
- Backend selection
- Proxy headers added
- Rate limiting events
- Access control decisions

## Examples

### Example 1: Simple Web App

```bash
# Deploy web app
warren service create web \
  --image nginx:latest \
  --replicas 2 \
  --port 80

# Create ingress
warren ingress create web-ingress \
  --host www.example.com \
  --service web \
  --port 80

# Test
curl -H "Host: www.example.com" http://manager-ip:8000/
```

### Example 2: API with HTTPS

```bash
# Deploy API
warren service create api \
  --image my-api:latest \
  --replicas 3 \
  --port 8080

# Create ingress with Let's Encrypt
warren ingress create api-ingress \
  --host api.example.com \
  --service api \
  --port 8080 \
  --tls \
  --tls-email ops@example.com

# Wait for certificate issuance (check logs or certificate list)
warren certificate list

# Test HTTPS
curl https://api.example.com:8443/health
```

### Example 3: Multi-Service Setup

```bash
# Deploy services
warren service create frontend --image frontend:latest --replicas 2 --port 80
warren service create backend --image backend:latest --replicas 3 --port 8080
warren service create admin --image admin:latest --replicas 1 --port 3000

# Create ingresses
warren ingress create frontend-ingress \
  --host app.example.com \
  --service frontend \
  --port 80 \
  --tls \
  --tls-email ops@example.com

warren ingress create backend-ingress \
  --host api.example.com \
  --service backend \
  --port 8080 \
  --tls \
  --tls-email ops@example.com

warren ingress create admin-ingress \
  --host admin.example.com \
  --service admin \
  --port 3000 \
  --tls \
  --tls-email ops@example.com
```

## Troubleshooting

### Ingress Not Routing

1. Check ingress exists: `warren ingress list`
2. Check service is running: `warren service list`
3. Check service has healthy tasks: `warren task list`
4. Check logs: `warren cluster init --log-level debug`

### HTTPS Not Working

1. Check certificate exists: `warren certificate list`
2. Check port 8443 is listening: `netstat -tlnp | grep 8443`
3. Check certificate validity: `warren certificate inspect my-cert`
4. For Let's Encrypt:
   - Verify DNS points to correct IP
   - Check port 80 is accessible
   - Check manager logs for ACME errors

### 429 Too Many Requests

Rate limiting is active. This is expected if:
- Middleware has rate limiting configured
- Making too many requests from same IP

To check: Review ingress path configuration for `RateLimit` settings.

### 403 Forbidden

Access control is blocking the request. Check:
- Client IP against allowed/denied lists
- Ingress path configuration for `AccessControl` settings

## Security

### TLS Configuration

Warren uses secure TLS settings:
- **Minimum version**: TLS 1.2
- **Cipher suites**: ECDHE-RSA/ECDSA with AES-128/256-GCM
- **Certificate storage**: Encrypted in Warren storage
- **Private keys**: Excluded from list operations

### Headers

Remove sensitive headers:
```go
Remove: ["Server", "X-Powered-By", "X-AspNet-Version"]
```

Add security headers:
```go
Set: {
  "X-Frame-Options": "DENY",
  "X-Content-Type-Options": "nosniff",
  "Strict-Transport-Security": "max-age=31536000"
}
```

### Access Control

Restrict access by IP:
```go
// Only allow office network
AllowedIPs: ["203.0.113.0/24"]

// Block known bad actors
DeniedIPs: ["198.51.100.0/24"]
```

### Rate Limiting

Prevent abuse:
```go
// Limit to 100 requests per second per IP
RequestsPerSecond: 100
Burst: 200
```

## Performance

### Benchmarks

Expected performance (single manager):
- **Throughput**: ~10,000 requests/second
- **Latency**: < 1ms overhead (proxy + routing)
- **Memory**: ~50MB for ingress proxy
- **CPU**: Minimal (< 5% at 10K req/s)

### Optimization

1. **HTTP Keep-Alive**: Enabled by default
2. **Connection Pooling**: Reuses connections to backends
3. **Rate Limiter Cleanup**: Hourly cleanup prevents memory leaks
4. **Efficient Routing**: O(n) route matching where n = number of ingresses

## API Reference

See [API Documentation](api.md) for:
- `CreateIngress` - Create new ingress
- `GetIngress` - Get ingress details
- `ListIngresses` - List all ingresses
- `UpdateIngress` - Update ingress configuration
- `DeleteIngress` - Delete ingress

Certificate APIs:
- `CreateTLSCertificate` - Upload certificate
- `GetTLSCertificate` - Get certificate details
- `ListTLSCertificates` - List all certificates
- `DeleteTLSCertificate` - Delete certificate

## Limitations

Current limitations (may be addressed in future versions):

1. **No HTTP→HTTPS redirect** - Clients must use HTTPS directly
2. **Advanced routing via API only** - CLI doesn't expose header/path/rate-limit configuration
3. **No WebSocket support** - Only HTTP/HTTPS
4. **No gRPC proxying** - Only HTTP/1.1
5. **Single ingress per host/path** - Last one wins if overlapping

## Roadmap

Future enhancements:
- HTTP→HTTPS automatic redirect
- CLI support for advanced routing features
- WebSocket proxying
- gRPC proxying
- Custom error pages
- Request/response transformation
- Circuit breaking
- Distributed tracing integration

## Related Documentation

- [Warren Architecture](.agent/System/project-architecture.md)
- [Services Guide](services.md)
- [TLS Certificates Guide](certificates.md)
- [API Reference](api.md)
