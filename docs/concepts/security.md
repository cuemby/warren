# Security Model

Warren implements a tiered security model that balances simplicity with security. This document explains how Warren's security works, when to use each authentication method, and best practices.

## Overview

Warren v1.4.0+ uses a **tiered security model** inspired by Docker Swarm's proven approach:

1. **Tier 1: Unix Socket** - Local, instant access (read-only)
2. **Tier 2: Local TCP with Auto-Bootstrap** - Future enhancement (not yet implemented)
3. **Tier 3: Remote TCP with mTLS** - Full remote access (all operations)

This model provides Docker Swarm-level simplicity for local operations while maintaining strong security for cluster communication and remote access.

---

## Tier 1: Unix Socket (Local Read-Only Access)

### What Is It?

The Unix socket provides zero-configuration local access to Warren CLI commands for read operations.

### How It Works

```
┌─────────────────┐
│  warren CLI     │
│  (same machine) │
└────────┬────────┘
         │
         │ Unix Socket
         │ /var/run/warren.sock
         │ No encryption (local IPC)
         │
┌────────▼────────┐
│  Warren Manager │
│  Read-Only API  │
└─────────────────┘
```

### Features

- **Path**: `/var/run/warren.sock`
- **Permissions**: 0660 (rw-rw----)
- **Owner**: root:warren
- **Authentication**: OS-level file permissions
- **Operations**: Read-only (list, inspect, get, watch)
- **Encryption**: None (local IPC, not needed)

### Allowed Operations

Read operations that work via Unix socket:
- `warren node list`
- `warren service list`
- `warren service inspect <name>`
- `warren secret list`
- `warren volume list`
- `warren cluster info`
- Any command with `List*`, `Get*`, `Inspect*`, `Watch*`, `Describe*`, `Show*`

### Blocked Operations

Write operations blocked with helpful error message:
```bash
$ warren service create nginx --image nginx:latest

Error: rpc error: code = PermissionDenied desc = write operations not allowed on Unix socket - use TCP connection with mTLS (warren init --manager <addr> --token <token>)
```

### Security Properties

**Strengths**:
- OS-level access control via file permissions
- No network exposure (local IPC only)
- Proven pattern (used by Docker, containerd, etc.)
- Fast (no TLS handshake)

**Limitations**:
- Read-only (write operations blocked)
- Local machine only (no remote access)
- Requires user to be in `warren` group or root

### Setup

Unix socket is created automatically when Warren manager starts. No configuration required.

To allow non-root users:
```bash
# Add user to warren group
sudo usermod -a -G warren username

# Reload group membership
newgrp warren

# Verify access
ls -la /var/run/warren.sock
```

---

## Tier 3: Remote TCP with mTLS

### What Is It?

Mutual TLS (mTLS) provides secure remote access to Warren clusters with full read/write capabilities.

### How It Works

```
┌─────────────────┐
│  warren CLI     │
│  (any machine)  │
└────────┬────────┘
         │
         │ TCP Port 8080
         │ mTLS (X.509 certificates)
         │ Both client + server auth
         │
┌────────▼────────┐
│  Warren Manager │
│  Full API       │
└─────────────────┘
```

### Features

- **Port**: TCP 8080 (configurable)
- **Protocol**: gRPC over TLS 1.3
- **Authentication**: Mutual TLS (client + server certificates)
- **Operations**: All (read + write)
- **Encryption**: AES-256-GCM
- **Certificate Rotation**: 90-day expiry, auto-renewal planned

### When To Use mTLS

| Scenario | Use mTLS? |
|----------|-----------|
| Local read operations | ❌ No - Unix socket sufficient |
| Local write operations | ✅ Yes - required for security |
| Remote access from laptop | ✅ Yes - required |
| CI/CD automation | ✅ Yes - required for write ops |
| Production deployments | ✅ Yes - recommended |

### Setup for Local Write Operations

If you're on the manager node and need write access:

```bash
# Option 1: Use --manager flag (quick, per-command)
warren service create nginx --image nginx:latest --manager 127.0.0.1:8080

# Option 2: Initialize CLI with mTLS (permanent setup)
# Step 1: Get the manager token
sudo cat /var/lib/warren/cluster/manager_token.txt

# Step 2: Initialize CLI
warren init --manager 127.0.0.1:8080 --token <token-from-above>

# Step 3: Verify
warren service create nginx --image nginx:latest  # Works without --manager!
```

### Setup for Remote Access

**On the manager node:**
```bash
# Generate join token for CLI access
warren cluster join-token manager --manager 127.0.0.1:8080

# Output:
# SWMTKN-1-3x7h9f2k1p6v8w4q0n5m7j9l2b8d6f4g1h3k5m7n9p2r4t6
```

**On your local machine:**
```bash
# Initialize CLI with manager address and token
warren init \
  --manager 192.168.1.10:8080 \
  --token SWMTKN-1-3x7h9f2k1p6v8w4q0n5m7j9l2b8d6f4g1h3k5m7n9p2r4t6

# All commands now work
warren service create web --image nginx:latest
```

### Certificate Storage

Certificates are stored in:
- **Client**: `~/.warren/certs/cli/`
  - `client.crt` - Client certificate
  - `client.key` - Client private key (0600 permissions)
  - `ca.crt` - CA certificate
- **Server**: `/var/lib/warren/certs/`
  - `server.crt` - Server certificate
  - `server.key` - Server private key (0600 permissions)
  - `ca.crt` - CA certificate

### Security Properties

**Strengths**:
- Industry-standard mTLS (proven security)
- Both client and server authenticated
- Encrypted channel (TLS 1.3)
- Certificate-based access control
- Works over untrusted networks

**Limitations**:
- Requires initial setup (`warren init`)
- Certificate management overhead
- Token must be kept secure

---

## Cluster Communication Security

All Warren cluster communication uses mTLS, regardless of CLI access method.

### Manager ↔ Manager

```
Manager 1 (Leader)  ←──mTLS──→  Manager 2 (Follower)
                    ←──mTLS──→  Manager 3 (Follower)
```

- **Protocol**: Raft over gRPC with mTLS
- **Port**: 7946 (Raft consensus)
- **Auth**: X.509 certificates
- **Encryption**: TLS 1.3

### Manager ↔ Worker

```
Manager (Scheduler)  ←──mTLS──→  Worker (Executor)
```

- **Protocol**: gRPC with mTLS
- **Port**: 8080 (API)
- **Auth**: X.509 certificates
- **Encryption**: TLS 1.3

### Join Tokens

Tokens are used to authorize new nodes joining the cluster:

- **Format**: `SWMTKN-{version}-{secret}`
- **Expiry**: 24 hours (configurable)
- **Usage**: One-time use for certificate issuance
- **Storage**: `/var/lib/warren/cluster/{manager|worker}_token.txt`

---

## Secrets Management

Warren encrypts secrets at rest and in transit.

### Encryption at Rest

- **Algorithm**: AES-256-GCM
- **Key Derivation**: PBKDF2 with cluster-specific salt
- **Storage**: BoltDB with encrypted values
- **Key Rotation**: Manual (planned for future)

### Encryption in Transit

- **Manager ↔ Manager**: mTLS (secrets replicated via Raft)
- **Manager ↔ Worker**: mTLS (secrets pushed to workers)
- **Worker → Container**: Tmpfs mount (never hits disk)

### Secret Access

```bash
# Create secret (write operation - requires mTLS)
warren secret create db-password \
  --from-literal password=secretValue \
  --manager 127.0.0.1:8080

# List secrets (read operation - works via Unix socket!)
warren secret list

# Use in service
warren service create webapp \
  --image myapp:latest \
  --secret db-password \
  --manager 127.0.0.1:8080

# Secret is mounted at /run/secrets/db-password in container (read-only tmpfs)
```

---

## Security Best Practices

### 1. Principle of Least Privilege

**Use Unix socket for monitoring/read-only access**:
```bash
# Good: Monitoring scripts that only need to read
#!/bin/bash
warren node list | grep ready  # Works via Unix socket
warren service list            # Works via Unix socket
```

**Use mTLS only when write access is needed**:
```bash
# Good: Deployment scripts that need write access
warren init --manager 127.0.0.1:8080 --token $MANAGER_TOKEN
warren service create app --image myapp:latest
```

### 2. Protect Join Tokens

**Don't**:
```bash
# Bad: Token in command history
warren init --manager 192.168.1.10:8080 --token SWMTKN-1-abc123...

# Bad: Token in script (committed to git)
TOKEN="SWMTKN-1-abc123..."
warren init --manager $MANAGER --token $TOKEN
```

**Do**:
```bash
# Good: Read token from file
TOKEN=$(sudo cat /var/lib/warren/cluster/manager_token.txt)
warren init --manager 192.168.1.10:8080 --token $TOKEN

# Good: Use environment variable (not committed)
export WARREN_JOIN_TOKEN=$(cat ~/.warren/token)
warren init --manager $MANAGER --token $WARREN_JOIN_TOKEN
```

### 3. Rotate Certificates

```bash
# Check certificate expiry
openssl x509 -in ~/.warren/certs/cli/client.crt -noout -dates

# Renew before expiry (planned: auto-renewal)
warren init --manager 127.0.0.1:8080 --token $NEW_TOKEN --force
```

### 4. Secure Manager Tokens

```bash
# Good: Restrict token file permissions
sudo chmod 600 /var/lib/warren/cluster/manager_token.txt
sudo chown root:root /var/lib/warren/cluster/manager_token.txt

# Good: Generate new tokens periodically
warren cluster join-token manager --manager 127.0.0.1:8080 --rotate
```

### 5. Network Security

**Firewall Rules**:
```bash
# Manager nodes: Allow 8080 (API) and 7946 (Raft) from cluster IPs only
sudo ufw allow from 192.168.1.0/24 to any port 8080 proto tcp
sudo ufw allow from 192.168.1.0/24 to any port 7946 proto tcp

# Worker nodes: Allow 8080 from manager IPs only
sudo ufw allow from 192.168.1.10 to any port 8080 proto tcp
```

**VPN/Private Network**:
- Run Warren cluster on private network (VPC, VPN)
- Don't expose ports 8080/7946 to public internet
- Use bastion host for remote access

### 6. Audit Logging

```bash
# Enable audit logging (future feature)
warren cluster config set audit.enabled=true
warren cluster config set audit.log-path=/var/log/warren/audit.log

# Monitor who's accessing the cluster
tail -f /var/log/warren/audit.log
```

---

## Security Comparison: Docker Swarm vs Warren

| Feature | Docker Swarm | Warren v1.4.0 |
|---------|--------------|---------------|
| Local CLI (read) | Unix socket | ✅ Unix socket |
| Local CLI (write) | Unix socket | mTLS or --manager flag |
| Remote CLI | mTLS | ✅ mTLS |
| Cluster communication | mTLS | ✅ mTLS |
| Secrets encryption | AES-256 | ✅ AES-256-GCM |
| Certificate rotation | Auto | Planned (manual for now) |
| Join tokens | 24h expiry | ✅ 24h expiry (configurable) |

Warren's security model closely follows Docker Swarm's proven approach, with the main difference being write operations require explicit mTLS setup for local access (providing clearer security boundaries).

---

## Troubleshooting

### Permission Denied: /var/run/warren.sock

**Problem**:
```bash
$ warren node list
Error: dial unix /var/run/warren.sock: connect: permission denied
```

**Solution**:
```bash
# Add user to warren group
sudo usermod -a -G warren $(whoami)

# Reload group membership
newgrp warren

# Verify
warren node list  # Should work now
```

### Write Operations Not Allowed on Unix Socket

**Problem**:
```bash
$ warren service create nginx --image nginx:latest
Error: rpc error: code = PermissionDenied desc = write operations not allowed on Unix socket
```

**Solution**:
```bash
# Option 1: Use --manager flag
warren service create nginx --image nginx:latest --manager 127.0.0.1:8080

# Option 2: Set up mTLS
warren init --manager 127.0.0.1:8080 --token <manager-token>
warren service create nginx --image nginx:latest  # Works now
```

### CLI Certificate Not Found

**Problem**:
```bash
$ warren service list
Error: CLI certificate not found at /home/user/.warren/certs/cli
```

**Solution**:
```bash
# Initialize CLI with mTLS
warren init --manager 192.168.1.10:8080 --token <join-token>
```

### Certificate Expired

**Problem**:
```bash
$ warren service list
Error: x509: certificate has expired
```

**Solution**:
```bash
# Renew certificate
TOKEN=$(sudo cat /var/lib/warren/cluster/manager_token.txt)
warren init --manager 127.0.0.1:8080 --token $TOKEN --force
```

---

## Future Enhancements

### Tier 2: Auto-Bootstrap (Planned)

Future versions may implement automatic certificate provisioning for local write operations:

```bash
# Planned: Write operations trigger auto-bootstrap
warren service create nginx --image nginx:latest
# → Auto-detects local manager
# → Reads bootstrap token from /var/lib/warren/bootstrap-token
# → Requests certificate automatically
# → Completes operation
```

### Certificate Auto-Renewal

Planned automatic certificate renewal before expiry.

### Hardware Security Module (HSM) Support

Planned support for storing private keys in HSM for high-security environments.

---

## Summary

Warren's tiered security model provides:

1. **Simplicity**: Unix socket for local read operations (zero config)
2. **Security**: mTLS for write operations and remote access
3. **Flexibility**: Choose the right tool for the job
4. **Proven**: Based on Docker Swarm's battle-tested approach

**Key Principle**: Use the simplest authentication method that meets your security requirements.

- **Monitoring/read-only**: Unix socket ✅
- **Local admin**: mTLS with `--manager` flag or `warren init` ✅
- **Remote access**: mTLS with `warren init` ✅
- **Production**: mTLS + VPN + firewall rules ✅

---

**Related Documentation**:
- [Getting Started](../getting-started.md) - Initial setup
- [Troubleshooting](../troubleshooting.md) - Common issues
- [High Availability](high-availability.md) - Multi-manager clusters
