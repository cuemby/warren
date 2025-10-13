/*
Package security provides cryptographic services for Warren clusters.

This package implements three core security capabilities: secrets encryption using
AES-256-GCM, a Certificate Authority (CA) for mutual TLS (mTLS), and certificate
lifecycle management. Together, these components provide end-to-end encryption
for sensitive data and secure authentication for all cluster communications.

# Architecture

Warren's security architecture is built on three pillars:

	┌─────────────────────────────────────────────────────────────┐
	│                    Security Architecture                    │
	└─────┬───────────────────────┬──────────────────┬────────────┘
	      │                       │                  │
	      ▼                       ▼                  ▼
	┌─────────────┐      ┌────────────────┐   ┌──────────────┐
	│   Secrets   │      │       CA       │   │ Certificate  │
	│ Encryption  │      │  (Root + Sub)  │   │  Management  │
	└─────┬───────┘      └────────┬───────┘   └──────┬───────┘
	      │                       │                   │
	      ▼                       ▼                   ▼
	  AES-256-GCM         RSA 4096-bit          90-day rotation
	  User secrets        10-year validity      Automatic renewal

## Cluster Encryption Key

All security is rooted in the cluster encryption key, a 32-byte key derived
from the cluster ID during initialization:

	clusterKey = SHA-256(clusterID)  // 32 bytes for AES-256

This key encrypts:
  - User secrets (via SecretsManager)
  - CA private key (in storage)
  - Any sensitive cluster data

The key is stored only in memory on manager nodes and must be provided when
joining the cluster or recovering from backups.

# Secrets Encryption

## SecretsManager

The SecretsManager encrypts and decrypts user secrets (API keys, passwords, etc.)
using AES-256 in Galois/Counter Mode (GCM), providing authenticated encryption:

	Plaintext → AES-256-GCM → Ciphertext + Authentication Tag
	                ↑
	            32-byte key

Key features:
  - Authenticated encryption (integrity + confidentiality)
  - Random nonce per encryption (no nonce reuse)
  - Fast performance (~100MB/s on modern CPUs)

## Encryption Process

 1. Generate random 12-byte nonce
 2. Encrypt plaintext with AES-256-GCM
 3. Prepend nonce to ciphertext
 4. Store combined bytes: [nonce || ciphertext || tag]

This ensures each secret has a unique nonce, preventing cryptographic attacks.

## Secret Storage Format

Secrets are stored encrypted in BoltDB:

	Secret {
		ID:   "secret-abc123"
		Name: "database-password"
		Data: [nonce || ciphertext || tag]  // Binary data
	}

Decryption reverses the process:

 1. Extract nonce (first 12 bytes)
 2. Extract ciphertext + tag (remaining bytes)
 3. Decrypt and verify authentication tag
 4. Return plaintext or error if tampered

# Certificate Authority

## Root CA

Warren's CA uses a hierarchical structure with a long-lived root certificate:

	Root CA (self-signed)
	├── 10-year validity
	├── RSA 4096-bit key (high security)
	├── KeyUsage: CertSign, CRLSign
	└── Subject: CN=Warren Root CA, O=Warren Cluster

The root CA is created during cluster initialization and stored encrypted:

	Root Certificate: Stored in BoltDB (plaintext, public)
	Root Private Key: Stored in BoltDB (encrypted with cluster key)

## Node Certificates

The CA issues certificates for all cluster nodes (managers and workers):

	Node Certificate
	├── 90-day validity
	├── RSA 2048-bit key (faster operations)
	├── KeyUsage: DigitalSignature, KeyEncipherment
	├── ExtKeyUsage: ServerAuth, ClientAuth
	├── Subject: CN={role}-{nodeID}, O=Warren Cluster
	├── DNS Names: [node hostname]
	└── IP Addresses: [node IP]

Each node receives a unique certificate for mutual TLS authentication:

	Manager Node ←→ mTLS ←→ Worker Node
	     ↓                       ↓
	CA verifies             CA verifies
	worker cert             manager cert

## Client Certificates

CLI clients also receive certificates for authentication:

	CLI Certificate
	├── 90-day validity
	├── KeyUsage: DigitalSignature, KeyEncipherment
	├── ExtKeyUsage: ClientAuth
	└── Subject: CN=cli-{clientID}, O=Warren Cluster

This allows secure CLI → Manager communication without passwords.

# Usage Examples

## Creating a Secrets Manager

	import "github.com/cuemby/warren/pkg/security"

	// Method 1: From raw key (32 bytes)
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		panic(err)
	}

	sm, err := security.NewSecretsManager(key)
	if err != nil {
		panic(err)
	}

	// Method 2: From password (key derived via SHA-256)
	sm, err := security.NewSecretsManagerFromPassword("my-cluster-secret")
	if err != nil {
		panic(err)
	}

## Encrypting and Decrypting Secrets

	// Encrypt a database password
	plaintext := []byte("super-secret-password")
	ciphertext, err := sm.EncryptSecret(plaintext)
	if err != nil {
		panic(err)
	}

	// Store ciphertext in database...

	// Later, decrypt the secret
	decrypted, err := sm.DecryptSecret(ciphertext)
	if err != nil {
		panic(err)  // Tampering detected or wrong key
	}

	fmt.Println(string(decrypted))  // "super-secret-password"

## Creating User Secrets

	// High-level API for creating secrets
	secret, err := sm.CreateSecret("db-password", []byte("my-password"))
	if err != nil {
		panic(err)
	}

	// Secret is ready to store
	fmt.Println("Secret ID:", secret.ID)    // "secret-..."
	fmt.Println("Secret Name:", secret.Name)  // "db-password"
	fmt.Println("Data encrypted:", len(secret.Data) > 0)  // true

	// Retrieve plaintext later
	plaintext, err := sm.GetSecretData(secret)
	if err != nil {
		panic(err)
	}

## Setting Up Certificate Authority

	import (
		"github.com/cuemby/warren/pkg/security"
		"github.com/cuemby/warren/pkg/storage"
	)

	// Create storage backend
	store, err := storage.NewBoltStore("/var/lib/warren/cluster.db")
	if err != nil {
		panic(err)
	}

	// Set cluster encryption key (required for CA)
	clusterKey := security.DeriveKeyFromClusterID(clusterID)
	err = security.SetClusterEncryptionKey(clusterKey)
	if err != nil {
		panic(err)
	}

	// Create and initialize CA
	ca := security.NewCertAuthority(store)
	err = ca.Initialize()  // Generates root CA
	if err != nil {
		panic(err)
	}

	// Save CA to storage (encrypted)
	err = ca.SaveToStore()
	if err != nil {
		panic(err)
	}

## Issuing Node Certificates

	// Issue certificate for a manager node
	nodeID := "manager-1"
	role := "manager"
	dnsNames := []string{"manager1.cluster.local", "localhost"}
	ipAddresses := []net.IP{
		net.ParseIP("192.168.1.10"),
		net.ParseIP("127.0.0.1"),
	}

	tlsCert, err := ca.IssueNodeCertificate(nodeID, role, dnsNames, ipAddresses)
	if err != nil {
		panic(err)
	}

	// Certificate ready to use for TLS
	fmt.Println("Certificate issued for:", nodeID)
	fmt.Println("Valid until:", tlsCert.Leaf.NotAfter)

## Verifying Certificates

	// Load certificate from file or network
	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		panic(err)
	}

	// Verify against CA
	err = ca.VerifyCertificate(cert)
	if err != nil {
		// Certificate invalid or not issued by this CA
		panic(err)
	}

	fmt.Println("Certificate verified successfully")

## Certificate Rotation

	// Check if certificate needs rotation (< 30 days remaining)
	needsRotation := security.CertNeedsRotation(cert)

	if needsRotation {
		// Request new certificate from CA
		newTLSCert, err := ca.IssueNodeCertificate(nodeID, role, dnsNames, ipAddresses)
		if err != nil {
			panic(err)
		}

		// Save new certificate
		certDir, _ := security.GetCertDir(role, nodeID)
		err = security.SaveCertToFile(newTLSCert, certDir)
		if err != nil {
			panic(err)
		}

		fmt.Println("Certificate rotated successfully")
	}

# Integration Points

## Storage Integration

All security artifacts are persisted to BoltDB:

	Bucket: "ca"
	Key: "root-ca"
	Value: {RootCertDER: [...], RootKeyDER: [...encrypted...]}

	Bucket: "secrets"
	Key: "secret-{id}"
	Value: {ID, Name, Data: [...encrypted...], CreatedAt, UpdatedAt}

The CA and secrets are always encrypted at rest.

## Manager Integration

The manager coordinates security operations:

  - CreateSecret(name, data) → Encrypts and stores
  - GetSecret(id) → Retrieves and decrypts
  - RequestCertificate(nodeID) → Issues certificate via CA
  - VerifyClientCert(cert) → Validates CLI certificates

## gRPC TLS Integration

All gRPC communication uses mTLS with CA-issued certificates:

	// Server-side (Manager)
	creds := credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{managerCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    certPool,  // Contains root CA
	})

	// Client-side (Worker/CLI)
	creds := credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{workerCert},
		RootCAs:      certPool,  // Contains root CA
	})

This ensures:
  - All connections encrypted (TLS 1.2+)
  - Mutual authentication (both parties verified)
  - No unauthorized access (CA-signed certs required)

## Container Integration

Secrets are injected into containers as files or environment variables:

	// File mount: /run/secrets/{secret-name}
	task.Secrets = []*types.SecretReference{
		{SecretName: "db-password", Target: "/run/secrets/db-password"},
	}

	// Environment variable: DB_PASSWORD=...
	task.Secrets = []*types.SecretReference{
		{SecretName: "db-password", Target: "env:DB_PASSWORD"},
	}

Workers decrypt secrets before injection, ensuring they're never stored
unencrypted on disk.

# Design Patterns

## Authenticated Encryption

GCM mode provides both confidentiality and integrity:

	Encryption:  plaintext + key + nonce → ciphertext + tag
	Decryption:  ciphertext + tag + key + nonce → plaintext (or error)

The authentication tag prevents tampering:
  - Modified ciphertext → decryption fails
  - Wrong key → decryption fails
  - Wrong nonce → decryption fails

This is critical for secrets - we must detect tampering.

## Hierarchical PKI

The CA uses a standard hierarchical structure:

	Root CA (trust anchor)
	└── Node/Client Certificates (issued by root)

Benefits:
  - Root key rarely used (only for issuing certs)
  - Root can be offline for additional security
  - Revocation via CRL/OCSP (future enhancement)

## Key Derivation

The cluster encryption key is derived deterministically:

	clusterKey = SHA-256(clusterID)

This means:
  - Same cluster ID → same key (important for replicas)
  - Key can be recomputed without storage
  - Backup = cluster ID (must be kept secret!)

## Certificate Caching

The CA caches issued certificates in memory:

	certCache[nodeID] = {Cert, Key, IssuedAt, ExpiresAt}

This reduces cryptographic operations and improves performance:
  - First request: Generate new cert (~100ms)
  - Subsequent requests: Return cached cert (~1μs)

# Performance Characteristics

## Encryption Performance

AES-256-GCM is hardware-accelerated on modern CPUs (AES-NI):

  - Encryption: ~100-200 MB/s per core
  - Decryption: ~100-200 MB/s per core
  - Small secrets (< 1KB): ~1-2μs per operation

For Warren's use case (secrets typically < 1KB):
  - 1000 secrets/second easily achievable
  - Negligible CPU overhead

## Certificate Issuance Performance

Certificate generation is more expensive:

  - Root CA generation (RSA 4096): ~500ms (one-time)
  - Node cert generation (RSA 2048): ~50-100ms
  - Certificate verification: ~1-2ms

Recommendations:
  - Cache certificates (reduces load)
  - Issue certificates asynchronously (don't block)
  - Pre-generate certificates when possible

## Memory Usage

Security operations are memory-efficient:

  - SecretsManager: ~1KB (just the key)
  - CA: ~100KB (root cert + cache)
  - Per-node certificate: ~2KB

Total: ~5-10MB for typical cluster (100 nodes).

# Security Considerations

## Key Management

The cluster encryption key is critical:

  - Compromise = all secrets exposed
  - Loss = cluster unrecoverable
  - Must be backed up securely
  - Consider key rotation (future enhancement)

Best practices:
  - Store cluster ID in encrypted vault (HashiCorp Vault, etc.)
  - Use hardware security modules (HSM) for production
  - Rotate key periodically (requires re-encryption)

## Certificate Rotation

Certificates expire after 90 days (nodes) or 10 years (root CA):

  - Automatic rotation: Not yet implemented
  - Manual rotation: warren node update-cert
  - Grace period: 30 days before expiry

Plan for rotation:
  - Monitor certificate expiry dates
  - Implement automated renewal (future)
  - Test rotation in staging

## Threat Model

Warren's security protects against:

	✓ Network eavesdropping (TLS encryption)
	✓ Unauthorized access (mTLS authentication)
	✓ Secret tampering (authenticated encryption)
	✓ Impersonation (CA-signed certificates)

Warren does NOT protect against:

	✗ Compromised cluster encryption key (all secrets exposed)
	✗ Compromised CA private key (issue fake certificates)
	✗ Compromised manager node (full cluster access)
	✗ Physical access to storage (encrypted, but key in memory)

Defense in depth:
  - Encrypt storage volumes (LUKS, etc.)
  - Use secure boot and TPM
  - Implement RBAC (future enhancement)
  - Audit all security operations

## Cryptographic Agility

Warren uses modern, proven cryptography:

  - AES-256-GCM (NIST approved, widely used)
  - RSA 2048/4096 (NIST approved, secure until ~2030)
  - SHA-256 (NIST approved, no known attacks)
  - TLS 1.2+ (industry standard)

Future considerations:
  - Ed25519 for certificates (faster, smaller)
  - ChaCha20-Poly1305 for secrets (software-friendly)
  - Post-quantum cryptography (long-term)

# Troubleshooting

## Secret Decryption Failures

If decryption fails:

1. Check encryption key:
  - Ensure cluster key is correct
  - Verify key derivation from cluster ID
  - Check for key rotation events

2. Check for data corruption:
  - Verify ciphertext length (>= 28 bytes: 12 nonce + 16 tag)
  - Check storage backend integrity
  - Look for bit flips or disk errors

3. Check for tampering:
  - GCM will detect any modification
  - Check logs for unauthorized access
  - Review audit trails

## Certificate Verification Failures

If certificate verification fails:

1. Check CA consistency:
  - Ensure CA is loaded correctly
  - Verify root certificate matches
  - Check for CA rotation

2. Check certificate validity:
  - Verify not expired (NotAfter > now)
  - Verify not used too early (NotBefore < now)
  - Check certificate chain

3. Check certificate content:
  - Verify DNS names match
  - Verify IP addresses match
  - Check key usage flags

## Performance Issues

If security operations are slow:

1. Check CPU features:
  - Verify AES-NI is enabled (lscpu | grep aes)
  - Check for CPU throttling
  - Monitor CPU usage during encryption

2. Check certificate caching:
  - Verify cache is being used
  - Check cache hit rate
  - Monitor cert generation frequency

3. Check key size:
  - Consider RSA 2048 instead of 4096 for nodes
  - Balance security vs. performance
  - Profile cryptographic operations

# Monitoring Metrics

Key security metrics to monitor:

  - Secrets encrypted/decrypted per second
  - Certificate issuance rate
  - Certificate verification failures
  - Certificate expiry dates
  - CA operations (rare, should be low)

# See Also

  - pkg/storage - Encrypted storage backend
  - pkg/manager - Security operations coordinator
  - pkg/worker - Secret injection into containers
  - docs/security.md - Security architecture overview
*/
package security
