# Phase 6.4: mTLS Security - Implementation Plan

**Date**: 2025-10-11
**Priority**: [REQUIRED]
**Status**: Planning
**Related**: M6 - Production Hardening

---

## Overview

Implement mutual TLS (mTLS) authentication for all gRPC communications in Warren cluster. This ensures:
- All manager-to-manager communication is authenticated and encrypted
- All worker-to-manager communication is authenticated and encrypted
- All CLI-to-manager communication is authenticated and encrypted
- Certificate rotation with 90-day expiry
- Self-signed root CA managed by cluster

## Philosophy Alignment

Following Warren's simplicity principle:
- **Simple**: Self-signed CA, no external PKI infrastructure
- **Self-contained**: CA generated on cluster init, stored in Raft
- **Automatic**: Certificate issuance during node join, auto-rotation
- **Incremental**: Add mTLS without breaking existing functionality

## Current State

### Existing Security
- ✅ **Join tokens**: 64-char hex tokens with 24h expiry (pkg/manager/token.go)
- ✅ **Secrets encryption**: AES-256-GCM for secrets at rest (pkg/security/secrets.go)
- ❌ **gRPC security**: Currently unencrypted TCP (port 8080)
- ❌ **Node authentication**: Token-based but no mutual auth after join

### Existing gRPC Setup
- **Manager**: gRPC server on port 8080 (pkg/api/server.go)
- **Worker**: gRPC client connects to manager (pkg/worker/worker.go)
- **CLI**: gRPC client for commands (pkg/client/client.go)
- **No TLS**: Using `grpc.NewServer()` without credentials

## Technical Design

### Certificate Hierarchy

```
Warren Root CA (10 years)
├── Manager Certificates (90 days)
│   ├── manager-<node-id>.crt
│   └── Auto-rotation at 60 days
├── Worker Certificates (90 days)
│   ├── worker-<node-id>.crt
│   └── Auto-rotation at 60 days
└── CLI Certificates (90 days)
    ├── cli-<user>-<machine>.crt
    └── Manual rotation or auto on cluster access
```

### Certificate Storage

**Root CA**:
- Generated on `warren cluster init`
- Stored in BoltDB under `ca` bucket
- Private key encrypted with cluster encryption key
- Replicated via Raft to all managers

**Node Certificates**:
- Issued during `warren worker start` or `warren manager join`
- Stored locally in `~/.warren/certs/<node-id>/`
  - `node.crt` - Certificate
  - `node.key` - Private key
  - `ca.crt` - Root CA cert for verification

**CLI Certificates**:
- Issued on first `warren` command to cluster
- Stored in `~/.warren/certs/cli/`
  - `client.crt` - Certificate
  - `client.key` - Private key
  - `ca.crt` - Root CA cert

### mTLS Flow

**1. Cluster Initialization**:
```
warren cluster init
  ↓
Generate Root CA
  ↓
Store CA in BoltDB (encrypted)
  ↓
Issue manager certificate for self
  ↓
Store cert in ~/.warren/certs/manager-<id>/
  ↓
Start gRPC server with TLS
```

**2. Worker Join**:
```
warren worker start --manager <addr> --token <token>
  ↓
Connect to manager (TLS, token auth)
  ↓
Request certificate (gRPC: RequestCertificate)
  ↓
Manager issues worker cert
  ↓
Worker saves cert locally
  ↓
Worker reconnects with mTLS
```

**3. CLI Command**:
```
warren service list --manager <addr>
  ↓
Check for local cert (~/.warren/certs/cli/)
  ↓
If missing: Request cert (token or interactive auth)
  ↓
Connect with mTLS
  ↓
Execute command
```

## Implementation Plan

### Phase A: Certificate Authority (pkg/security/ca.go)

**File**: `pkg/security/ca.go` (~300 lines)

**Types**:
```go
type CertAuthority struct {
    rootCert    *x509.Certificate
    rootKey     *rsa.PrivateKey
    store       storage.Store
    certCache   map[string]*CachedCert
    mu          sync.RWMutex
}

type CachedCert struct {
    Cert      *x509.Certificate
    Key       *rsa.PrivateKey
    IssuedAt  time.Time
    ExpiresAt time.Time
}
```

**Functions**:
- `NewCertAuthority(store storage.Store) (*CertAuthority, error)`
- `Initialize() error` - Generate root CA
- `LoadFromStore() error` - Load existing CA
- `SaveToStore() error` - Save CA to BoltDB
- `IssueNodeCertificate(nodeID, role string) (*tls.Certificate, error)`
- `IssueClientCertificate(clientID string) (*tls.Certificate, error)`
- `VerifyCertificate(cert *x509.Certificate) error`
- `GetRootCACert() []byte` - PEM-encoded CA cert for distribution

**Implementation Details**:
- RSA 4096-bit for root CA (long-lived, high security)
- RSA 2048-bit for node certs (shorter-lived, faster)
- Root CA: 10-year validity
- Node certs: 90-day validity
- Encrypt private key with cluster encryption key before storage

**Testing**:
- TestInitializeCA
- TestIssueNodeCertificate
- TestVerifyCertificate
- TestSaveLoadCA

---

### Phase B: Certificate Management (pkg/security/certs.go)

**File**: `pkg/security/certs.go` (~200 lines)

**Functions**:
- `SaveCertToFile(cert *tls.Certificate, path string) error`
- `LoadCertFromFile(path string) (*tls.Certificate, error)`
- `SaveCACertToFile(caCert []byte, path string) error`
- `LoadCACertFromFile(path string) (*x509.Certificate, error)`
- `CertNeedsRotation(cert *x509.Certificate) bool` - True if <30 days remaining
- `GetCertExpiry(cert *x509.Certificate) time.Time`
- `ValidateCertChain(cert, ca *x509.Certificate) error`

**Storage Paths**:
```
~/.warren/certs/
├── manager-<node-id>/
│   ├── node.crt
│   ├── node.key
│   └── ca.crt
├── worker-<node-id>/
│   ├── node.crt
│   ├── node.key
│   └── ca.crt
└── cli/
    ├── client.crt
    ├── client.key
    └── ca.crt
```

**Testing**:
- TestSaveLoadCert
- TestCertNeedsRotation
- TestValidateCertChain

---

### Phase C: Manager mTLS (pkg/api/server.go)

**Changes to existing code**:

**1. Add TLS credentials to Server struct**:
```go
type Server struct {
    proto.UnimplementedWarrenAPIServer
    manager *manager.Manager
    grpc    *grpc.Server
    ca      *security.CertAuthority  // NEW
}
```

**2. Update NewServer() to configure mTLS**:
```go
func NewServer(mgr *manager.Manager, ca *security.CertAuthority) *Server {
    // Load node certificate
    cert, err := security.LoadCertFromFile(certPath)
    if err != nil {
        log.Fatal().Err(err).Msg("Failed to load node certificate")
    }

    // Load CA cert for client verification
    caCert, err := security.LoadCACertFromFile(caPath)
    if err != nil {
        log.Fatal().Err(err).Msg("Failed to load CA certificate")
    }

    // Create cert pool for client verification
    certPool := x509.NewCertPool()
    certPool.AddCert(caCert)

    // Configure TLS
    tlsConfig := &tls.Config{
        ClientAuth:   tls.RequireAndVerifyClientCert,
        Certificates: []tls.Certificate{*cert},
        ClientCAs:    certPool,
        MinVersion:   tls.VersionTLS13,
    }

    // Create gRPC server with TLS
    creds := credentials.NewTLS(tlsConfig)
    grpcServer := grpc.NewServer(grpc.Creds(creds))

    return &Server{
        manager: mgr,
        grpc:    grpcServer,
        ca:      ca,
    }
}
```

**3. Add RequestCertificate RPC handler**:
```go
func (s *Server) RequestCertificate(ctx context.Context, req *proto.RequestCertificateRequest) (*proto.RequestCertificateResponse, error) {
    // Verify join token
    role, err := s.manager.ValidateToken(req.Token)
    if err != nil {
        return nil, fmt.Errorf("invalid token: %w", err)
    }

    // Issue certificate
    cert, err := s.ca.IssueNodeCertificate(req.NodeId, role)
    if err != nil {
        return nil, fmt.Errorf("failed to issue certificate: %w", err)
    }

    // Convert to PEM for transmission
    certPEM := pem.EncodeToMemory(&pem.Block{
        Type:  "CERTIFICATE",
        Bytes: cert.Certificate[0],
    })

    keyPEM := pem.EncodeToMemory(&pem.Block{
        Type:  "RSA PRIVATE KEY",
        Bytes: x509.MarshalPKCS1PrivateKey(cert.PrivateKey.(*rsa.PrivateKey)),
    })

    caCertPEM := s.ca.GetRootCACert()

    return &proto.RequestCertificateResponse{
        Certificate: certPEM,
        PrivateKey:  keyPEM,
        CaCert:      caCertPEM,
    }, nil
}
```

**4. Update protobuf** (api/proto/warren.proto):
```protobuf
message RequestCertificateRequest {
    string node_id = 1;
    string token = 2;
}

message RequestCertificateResponse {
    bytes certificate = 1;
    bytes private_key = 2;
    bytes ca_cert = 3;
}

service WarrenAPI {
    // ... existing methods
    rpc RequestCertificate(RequestCertificateRequest) returns (RequestCertificateResponse);
}
```

**Testing**:
- TestServerMTLS
- TestRequestCertificate
- TestUnauthorizedAccess

---

### Phase D: Worker mTLS (pkg/worker/worker.go)

**Changes**:

**1. Add certificate request on first connection**:
```go
func (w *Worker) connectToManager() error {
    // Check for existing certificate
    certPath := filepath.Join(w.certDir, "node.crt")
    if _, err := os.Stat(certPath); os.IsNotExist(err) {
        // Request certificate
        if err := w.requestCertificate(); err != nil {
            return fmt.Errorf("failed to request certificate: %w", err)
        }
    }

    // Load certificate
    cert, err := security.LoadCertFromFile(certPath)
    if err != nil {
        return fmt.Errorf("failed to load certificate: %w", err)
    }

    // Load CA cert
    caCert, err := security.LoadCACertFromFile(filepath.Join(w.certDir, "ca.crt"))
    if err != nil {
        return fmt.Errorf("failed to load CA certificate: %w", err)
    }

    // Configure TLS
    certPool := x509.NewCertPool()
    certPool.AddCert(caCert)

    tlsConfig := &tls.Config{
        Certificates: []tls.Certificate{*cert},
        RootCAs:      certPool,
        MinVersion:   tls.VersionTLS13,
    }

    // Connect with mTLS
    creds := credentials.NewTLS(tlsConfig)
    conn, err := grpc.Dial(w.managerAddr, grpc.WithTransportCredentials(creds))
    if err != nil {
        return fmt.Errorf("failed to dial manager: %w", err)
    }

    w.client = proto.NewWarrenAPIClient(conn)
    return nil
}
```

**2. Add requestCertificate() method**:
```go
func (w *Worker) requestCertificate() error {
    // First connection without TLS (just for cert request)
    // Use InsecureSkipVerify temporarily
    tlsConfig := &tls.Config{InsecureSkipVerify: true}
    creds := credentials.NewTLS(tlsConfig)
    conn, err := grpc.Dial(w.managerAddr, grpc.WithTransportCredentials(creds))
    if err != nil {
        return err
    }
    defer conn.Close()

    client := proto.NewWarrenAPIClient(conn)

    // Request certificate
    resp, err := client.RequestCertificate(context.Background(), &proto.RequestCertificateRequest{
        NodeId: w.id,
        Token:  w.joinToken,
    })
    if err != nil {
        return err
    }

    // Save certificate files
    os.MkdirAll(w.certDir, 0700)

    if err := ioutil.WriteFile(filepath.Join(w.certDir, "node.crt"), resp.Certificate, 0600); err != nil {
        return err
    }
    if err := ioutil.WriteFile(filepath.Join(w.certDir, "node.key"), resp.PrivateKey, 0600); err != nil {
        return err
    }
    if err := ioutil.WriteFile(filepath.Join(w.certDir, "ca.crt"), resp.CaCert, 0600); err != nil {
        return err
    }

    return nil
}
```

**3. Add certificate rotation check**:
```go
func (w *Worker) checkCertificateRotation() {
    ticker := time.NewTicker(24 * time.Hour)
    for range ticker.C {
        cert, err := security.LoadCertFromFile(filepath.Join(w.certDir, "node.crt"))
        if err != nil {
            continue
        }

        if security.CertNeedsRotation(cert.Leaf) {
            log.Info().Msg("Certificate expiring soon, requesting rotation")
            if err := w.rotateCertificate(); err != nil {
                log.Error().Err(err).Msg("Failed to rotate certificate")
            }
        }
    }
}
```

**Testing**:
- TestWorkerMTLS
- TestCertificateRequest
- TestCertificateRotation

---

### Phase E: CLI mTLS (pkg/client/client.go)

**Changes**:

Similar to worker, but:
- Store certs in `~/.warren/certs/cli/`
- Request certificate on first command
- Reuse certificate for subsequent commands
- Manual rotation via `warren cert rotate`

**Testing**:
- TestCLIMTLS
- TestCLICertRequest

---

### Phase F: CLI Commands (cmd/warren/cert.go)

**New commands**:

```bash
# Rotate current node certificate
warren cert rotate

# List certificates (for managers)
warren cert list

# Show certificate expiry
warren cert info

# Revoke a certificate (for managers)
warren cert revoke <node-id>
```

**Implementation**:
- `cmd/warren/cert.go` - Cobra command definitions
- Wire to CA methods

---

### Phase G: Testing & Documentation

**Unit Tests**:
- `pkg/security/ca_test.go` - CA operations
- `pkg/security/certs_test.go` - Cert file operations
- `pkg/api/server_test.go` - mTLS connection tests

**Integration Tests**:
- `test/integration/mtls_test.go` - End-to-end mTLS flow

**Documentation**:
- `docs/security/mtls.md` - mTLS setup and troubleshooting
- Update `docs/getting-started.md` - Mention mTLS automatic setup
- Update `docs/troubleshooting.md` - Add mTLS issues

---

## Implementation Order

1. **Phase A**: Certificate Authority (pkg/security/ca.go) - Foundation
2. **Phase B**: Certificate Management (pkg/security/certs.go) - File I/O
3. **Phase C**: Manager mTLS (pkg/api/server.go) - Server-side
4. **Phase D**: Worker mTLS (pkg/worker/worker.go) - Client-side
5. **Phase E**: CLI mTLS (pkg/client/client.go) - CLI client
6. **Phase F**: CLI Commands (cmd/warren/cert.go) - Management tools
7. **Phase G**: Testing & Documentation - Verification

## Acceptance Criteria

- [ ] Root CA generated on cluster init
- [ ] CA stored encrypted in BoltDB
- [ ] Manager certificates issued and used
- [ ] Worker certificates issued and used
- [ ] CLI certificates issued and used
- [ ] All gRPC connections use mTLS
- [ ] Unauthorized connections rejected
- [ ] Certificate rotation works (90-day expiry)
- [ ] CLI commands functional (`warren cert rotate`, `warren cert list`)
- [ ] Unit tests passing (>80% coverage)
- [ ] Integration tests passing
- [ ] Documentation complete

## Risks & Mitigations

**Risk 1**: Breaking existing clusters
- **Mitigation**: Feature flag `--enable-mtls` (default: true)
- **Mitigation**: Graceful upgrade path for existing deployments

**Risk 2**: Certificate loss (node cert deleted)
- **Mitigation**: CA can reissue certificates
- **Mitigation**: Document recovery process

**Risk 3**: CA private key compromise
- **Mitigation**: Stored encrypted with cluster key
- **Mitigation**: Requires access to BoltDB and cluster key

**Risk 4**: Certificate expiry not noticed
- **Mitigation**: Auto-rotation at 60 days (30 days before expiry)
- **Mitigation**: Warning logs when <30 days remaining

## Out of Scope (Future)

- [ ] Integration with external PKI (e.g., HashiCorp Vault)
- [ ] Certificate revocation lists (CRLs)
- [ ] OCSP (Online Certificate Status Protocol)
- [ ] Hardware security module (HSM) integration
- [ ] Let's Encrypt integration for public-facing services

These can be added in future milestones if needed.

---

## Success Metrics

- **Security**: All gRPC traffic authenticated and encrypted
- **Simplicity**: Zero external dependencies (self-signed CA)
- **Automation**: Certificates issued automatically on join
- **Reliability**: Auto-rotation prevents expiry issues
- **Performance**: < 5ms overhead for mTLS handshake

---

## References

- [Tech Spec - Security Model](../specs/tech.md#security-model)
- [Go crypto/tls](https://pkg.go.dev/crypto/tls)
- [Go crypto/x509](https://pkg.go.dev/crypto/x509)
- [gRPC Security](https://grpc.io/docs/guides/auth/)

---

**Plan Status**: ✅ Complete - Ready for Review
**Estimated Effort**: 2-3 days
**Next Step**: Review with user, then begin Phase A implementation
