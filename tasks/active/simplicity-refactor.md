# Warren Simplicity Refactoring

**Initiative**: Architectural refactoring to achieve true "Docker Swarm simplicity"
**Status**: 🚧 In Progress
**Priority**: [CRITICAL]
**Started**: 2025-10-14
**Target**: Warren v1.4.0

---

## Context

Our deployment automation struggle revealed a fundamental conflict between Warren's stated goal and its implementation. The PRD promises "Docker Swarm simplicity + Kubernetes features", but the current mTLS-for-everything approach creates complexity that contradicts this goal.

**The Wake-Up Call**: Spent 2+ hours debugging deployment scripts because `warren node list` requires `warren init` first, which requires manager to be fully started, which creates timing issues and connection refused errors.

**The Insight**: Docker Swarm got this right - local CLI works immediately, cluster operations use mTLS. We need to adopt this proven pattern.

---

## Goals

### Primary
- ✅ **Simplicity**: `warren node list` works immediately after `warren cluster init`
- ✅ **Zero-config**: No certificate setup required for local operations
- ✅ **Automation-friendly**: Deployment scripts work without mTLS wrestling

### Secondary
- 🔒 **Maintain security**: Cluster mTLS unchanged, remote access still secure
- 📚 **Better UX**: Clear error messages, Docker-like experience
- 🎯 **Meet PRD target**: Deploy production cluster in < 5 minutes with 3 commands

---

## Architecture Changes

### Current (v1.3.1)
```
┌─────────────────────┐
│   CLI Command       │
│                     │
│ warren node list    │
└──────────┬──────────┘
           │
           │ gRPC over TCP :8080
           │ mTLS REQUIRED ❌
           ▼
┌─────────────────────┐
│   Manager API       │
│                     │
│ Checks for cert...  │
│ ERROR: "Run init"   │
└─────────────────────┘
```

### Proposed (v1.4.0)
```
┌─────────────────────────────────────────┐
│            CLI Command                  │
│                                         │
│       warren node list                  │
└────────────┬────────────────────────────┘
             │
             ├─ If local ──────────────────┐
             │                             │
             │  Unix Socket                │
             │  NO mTLS ✅                 │
             │  Read-only                  │
             ▼                             │
        ┌────────────┐                     │
        │  Manager   │                     │
        │  (local)   │                     │
        └────────────┘                     │
                                          │
             ├─ If remote ────────────────┤
             │                             │
             │  TCP :2377                  │
             │  mTLS REQUIRED 🔒           │
             │  All operations             │
             ▼                             │
        ┌────────────┐                     │
        │  Manager   │                     │
        │  (remote)  │                     │
        └────────────┘                     │
                                          │
```

---

## Implementation Plan

### Phase 1: Unix Socket API (PRIORITY 1)

**Goal**: Enable local CLI to work without certificates

#### Tasks

- [ ] **1.1 Create Unix Socket Listener**
  - File: `pkg/api/server.go`
  - Add field: `unixServer *grpc.Server`
  - Add method: `StartUnixListener(socketPath string)`
  - Default path: `/var/run/warren.sock`
  - Permissions: 0660, owner: root, group: warren
  - Auto-create /var/run if missing

- [ ] **1.2 Implement Read-Only Interceptor**
  - File: `pkg/api/interceptor.go` (NEW)
  - Intercept all gRPC calls on Unix socket
  - Allow: List*, Get*, Inspect*, Watch* methods
  - Block: Create*, Update*, Delete*, Join* methods
  - Return: `codes.PermissionDenied` with message: "Write operations require TCP connection with mTLS"

- [ ] **1.3 Update Client Auto-Detection**
  - File: `pkg/client/client.go`
  - Add: `NewClientAuto()` method
  - Logic:
    1. Try Unix socket `/var/run/warren.sock` first
    2. If not exists or fails, try TCP with mTLS
    3. If no certs, return helpful error
  - Preserve: `NewClient(addr)` for explicit connections

- [ ] **1.4 Update All CLI Commands**
  - Files: `cmd/warren/*.go`
  - Change: Use `client.NewClientAuto()` instead of `client.NewClient()`
  - Commands affected: node, service, secret, volume, cluster info

#### Validation
```bash
# Should work immediately after cluster init:
warren cluster init
warren node list        # ✅ Works via Unix socket
warren service list     # ✅ Works via Unix socket
warren service create   # ❌ "Write operations require TCP + mTLS"
```

---

### Phase 2: Certificate Auto-Bootstrap (PRIORITY 2)

**Goal**: Auto-request certificates for write operations without user intervention

#### Tasks

- [ ] **2.1 Create Bootstrap Token System**
  - File: `pkg/manager/bootstrap.go` (NEW)
  - Generate secure random token on cluster init
  - Store in: `/var/lib/warren/bootstrap-token` (mode 0600, root-only)
  - Token format: `WARREN-BOOTSTRAP-<64-char-hex>`
  - Auto-delete after first successful cert issuance

- [ ] **2.2 Add Bootstrap Endpoint**
  - File: `pkg/api/server.go`
  - Add RPC: `RequestCLICertificate(token, certRequest) -> cert`
  - Validate bootstrap token matches stored token
  - Issue certificate with 90-day expiry
  - Mark token as used (prevent replay)

- [ ] **2.3 Client Auto-Bootstrap Flow**
  - File: `pkg/client/client.go`
  - Add: `NewClientWithAutoBootstrap(addr)` method
  - Logic for write operations:
    1. Detect "mTLS required" error
    2. Check for bootstrap token in `/var/lib/warren/bootstrap-token`
    3. If exists, auto-request certificate
    4. Save to `~/.warren/certs/cli/`
    5. Retry original operation
    6. If no token, return: "Run `warren init --manager <addr> --token <token>` for remote access"

- [ ] **2.4 Update Manager CLI Command**
  - File: `cmd/warren/cluster.go`
  - On `warren cluster init`, print:
    ```
    ✓ Cluster initialized successfully
    ✓ Manager listening on Unix socket: /var/run/warren.sock
    ✓ Manager listening on TCP: 0.0.0.0:2377 (mTLS)

    Local CLI is ready to use immediately!
    Remote access: warren init --manager <this-node-ip>:2377 --token <token>
    ```

#### Validation
```bash
warren cluster init
warren service create web --image nginx  # ✅ Auto-bootstraps cert, then creates service
warren service list                      # ✅ Uses cert for subsequent calls
```

---

### Phase 3: Port Architecture Cleanup (PRIORITY 3)

**Goal**: Standardize ports, eliminate confusion, match tech spec

#### Tasks

- [ ] **3.1 Update Port Definitions**
  - File: `pkg/api/server.go`
  - Constants:
    ```go
    const (
        DefaultUnixSocket = "/var/run/warren.sock"
        DefaultTCPPort    = 2377  // Cluster API (was 8080)
        DefaultMetricsPort = 9090 // Prometheus
        DefaultIngressHTTP = 8000 // Ingress HTTP
        DefaultIngressHTTPS = 8443 // Ingress HTTPS
    )
    ```
  - Remove all references to port 8080

- [ ] **3.2 Update Deployment Scripts**
  - Files: `scripts/lib/warren-utils.sh`, `scripts/deploy-production.sh`
  - Change all `8080` to `2377`
  - Update examples to use Unix socket for local ops

- [ ] **3.3 Update Technical Specification**
  - File: `specs/tech.md`
  - Section: "Authentication" (line 228)
  - Add tiered security model documentation
  - Update port table with Unix socket

- [ ] **3.4 Update All Examples**
  - Files: `examples/*.yaml`, `docs/**/*.md`
  - Replace `localhost:8080` with `localhost:2377`
  - Add Unix socket examples where appropriate

#### Validation
```bash
# Verify no references to port 8080 remain:
grep -r "8080" . --exclude-dir=node_modules --exclude-dir=.git
# Should only find this task file!
```

---

### Phase 4: CLI Experience Polish (PRIORITY 4)

**Goal**: Remove `warren init` from normal workflow, improve error messages

#### Tasks

- [ ] **4.1 Make warren init Optional**
  - File: `cmd/warren/init.go`
  - Update help text: "Only required for remote CLI access"
  - Add check: Warn if running on manager node ("not necessary - Unix socket available")

- [ ] **4.2 Improve Error Messages**
  - File: `pkg/client/client.go`
  - Remote access error:
    ```
    ❌ Cannot connect to remote manager at 10.0.1.5:2377

    For remote access, run:
      warren init --manager 10.0.1.5:2377 --token <join-token>

    Get token from manager:
      ssh user@10.0.1.5 'warren cluster join-token manager'
    ```
  - Unix socket permission error:
    ```
    ❌ Permission denied: /var/run/warren.sock

    Add your user to the 'warren' group:
      sudo usermod -a -G warren $(whoami)
      newgrp warren
    ```

- [ ] **4.3 Update Getting Started Guide**
  - File: `docs/getting-started.md`
  - Remove `warren init` from initial flow
  - Add "Remote CLI Setup" section (optional)
  - Show Unix socket usage examples

- [ ] **4.4 Add Security Concepts Doc**
  - File: `docs/concepts/security.md` (NEW)
  - Explain tiered security model
  - When to use Unix socket vs. TCP
  - Certificate management (when needed)
  - Best practices

#### Validation
- [ ] New user can follow getting-started.md without getting stuck
- [ ] Remote access setup is clear and documented
- [ ] Error messages are actionable

---

## Testing Strategy

### Unit Tests

- [ ] **Unix Socket Server**
  - File: `pkg/api/server_test.go`
  - Test: Socket creation with correct permissions
  - Test: Read-only enforcement on socket
  - Test: Dual listener (Unix + TCP)

- [ ] **Read-Only Interceptor**
  - File: `pkg/api/interceptor_test.go`
  - Test: Allow List*/Get*/Inspect*
  - Test: Block Create*/Update*/Delete*
  - Test: Correct error codes returned

- [ ] **Client Auto-Detection**
  - File: `pkg/client/client_test.go`
  - Test: Prefers Unix socket when available
  - Test: Falls back to TCP
  - Test: Clear errors when neither works

### Integration Tests

- [ ] **Unix Socket End-to-End**
  - File: `test/integration/unix_socket_test.go` (NEW)
  - Setup: Start manager with Unix socket
  - Test: `warren node list` works without certs
  - Test: `warren service create` fails appropriately
  - Test: Permissions enforcement

- [ ] **Auto-Bootstrap Flow**
  - File: `test/integration/auto_bootstrap_test.go` (NEW)
  - Setup: Fresh cluster, no CLI certs
  - Test: Write operation triggers bootstrap
  - Test: Certificate saved correctly
  - Test: Subsequent operations use cert

- [ ] **Lima Cluster Test**
  - File: `test/lima/test-local-cli.sh` (NEW)
  - Setup: 3-manager + 2-worker Lima cluster
  - Test: Local CLI works immediately (no init)
  - Test: Remote CLI requires init + token
  - Test: Deployment automation works

### Deployment Automation Validation

- [ ] **Update Deployment Script**
  - File: `scripts/lib/warren-utils.sh`
  - Remove `warren init` workarounds
  - Use Unix socket for verification
  - Test: Full deployment succeeds without cert setup

- [ ] **End-to-End Deployment**
  ```bash
  ./scripts/deploy-production.sh --managers 3 --workers 3
  # Should complete in < 5 minutes with ZERO certificate errors
  ```

---

## Documentation Updates

### Must Update

- [ ] **specs/prd.md**
  - Update "Goals and Objectives" - validate < 5 min target met
  - Update "Current Solution" limitations - remove mTLS friction

- [ ] **specs/tech.md**
  - Section 1.2 "API Server" - add Unix socket
  - Section "Authentication" - tiered security model
  - Update port table

- [ ] **docs/getting-started.md**
  - Remove `warren init` from quick start
  - Add "Remote Access Setup" section
  - Show Unix socket examples

- [ ] **docs/troubleshooting.md**
  - Add "Unix Socket Issues" section
  - Add "Certificate Problems" section (for remote)
  - Update permission error solutions

### New Docs

- [ ] **docs/concepts/security.md** (NEW)
  - Tiered security model explained
  - Unix socket vs. TCP comparison
  - When to use each approach
  - Certificate lifecycle (for remote)

- [ ] **docs/cli-reference.md**
  - Update all command examples (port 2377)
  - Add `--socket` and `--tcp` flags documentation
  - Add remote access examples

---

## Rollout Plan

### Stage 1: Implementation (Week 1)
- Implement Phase 1 (Unix socket)
- Write unit tests
- Internal validation

### Stage 2: Integration (Week 2)
- Implement Phase 2 (auto-bootstrap)
- Integration tests
- Lima cluster testing

### Stage 3: Polish (Week 3)
- Phase 3 (port cleanup)
- Phase 4 (UX polish)
- Documentation updates

### Stage 4: Validation (Week 3)
- Full deployment automation testing
- Performance regression check
- User acceptance testing (internal)

### Stage 5: Release (Week 4)
- Tag v1.4.0
- Release notes highlighting simplicity improvements
- Blog post: "Warren v1.4: True Docker Swarm Simplicity"

---

## Success Criteria

### Quantitative
- ✅ Deployment automation completes in < 5 minutes (PRD target)
- ✅ Zero certificate-related errors in deployment logs
- ✅ `warren node list` succeeds < 1 second after `warren cluster init`
- ✅ No performance regression (API latency, throughput)

### Qualitative
- ✅ New users can follow docs without getting stuck
- ✅ Error messages are actionable (no "run warren init" dead ends)
- ✅ Remote access setup is clear and documented
- ✅ Automation scripts are simpler (fewer workarounds)

### User Feedback (post-release)
- ✅ "Warren is as simple as Docker Swarm was!"
- ✅ "Setup took 3 minutes, not 30"
- ✅ "Finally, an orchestrator that just works"

---

## Risks & Mitigations

### Risk: Unix Socket Security Concerns
**Mitigation**:
- Document security model clearly
- Use file permissions (proven pattern from Docker)
- Feature flag `--no-unix-socket` for paranoid users

### Risk: Breaking Changes for Existing Users
**Mitigation**:
- Keep TCP + mTLS working (backward compatible)
- Add clear migration guide
- Release notes with upgrade instructions

### Risk: Port Change (8080 → 2377) Breaks Existing Deployments
**Mitigation**:
- Search codebase for hardcoded 8080 references
- Update all examples and docs
- Add upgrade note in CHANGELOG

### Risk: Performance Regression
**Mitigation**:
- Run load tests before/after
- Monitor API latency metrics
- Unix socket should be FASTER (no TLS overhead)

---

## Related Issues

- Deployment automation struggles (this session)
- PRD target: "< 5 minutes with 3 commands" not met (v1.3.1)
- User feedback: "Too complex compared to Docker Swarm" (hypothetical, but validated by our experience)

---

## Follow-Up Tasks

After this refactoring:
- [ ] Update demo videos (YouTube)
- [ ] Update marketing copy ("Docker Swarm simplicity")
- [ ] Community feedback collection
- [ ] Consider: Unix socket support for Windows (named pipes?)

---

**Owner**: Cuemby Engineering
**Last Updated**: 2025-10-14
**Next Review**: After Phase 1 completion
