package security

import (
	"crypto/x509"
	"net"
	"os"
	"testing"
	"time"

	"github.com/cuemby/warren/pkg/storage"
)

func TestInitializeCA(t *testing.T) {
	// Set cluster encryption key
	key := DeriveKeyFromClusterID("test-cluster")
	if err := SetClusterEncryptionKey(key); err != nil {
		t.Fatalf("Failed to set cluster encryption key: %v", err)
	}

	// Create temporary BoltDB
	tmpDir, err := os.MkdirTemp("", "warren-ca-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := storage.NewBoltStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Create CA
	ca := NewCertAuthority(store)

	// Initialize CA
	if err := ca.Initialize(); err != nil {
		t.Fatalf("Failed to initialize CA: %v", err)
	}

	// Verify CA is initialized
	if !ca.IsInitialized() {
		t.Error("CA should be initialized")
	}

	// Verify root cert exists
	if ca.rootCert == nil {
		t.Error("Root certificate should not be nil")
	}

	// Verify root key exists
	if ca.rootKey == nil {
		t.Error("Root key should not be nil")
	}

	// Verify root cert is a CA
	if !ca.rootCert.IsCA {
		t.Error("Root certificate should be a CA")
	}

	// Verify validity period
	expectedExpiry := time.Now().Add(rootCAValidity)
	if ca.rootCert.NotAfter.Before(expectedExpiry.Add(-time.Hour)) {
		t.Errorf("Root cert expiry too early: %v, expected around %v", ca.rootCert.NotAfter, expectedExpiry)
	}
}

func TestSaveLoadCA(t *testing.T) {
	// Set cluster encryption key
	key := DeriveKeyFromClusterID("test-cluster")
	if err := SetClusterEncryptionKey(key); err != nil {
		t.Fatalf("Failed to set cluster encryption key: %v", err)
	}

	// Create temporary BoltDB
	tmpDir, err := os.MkdirTemp("", "warren-ca-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := storage.NewBoltStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Create and initialize CA
	ca1 := NewCertAuthority(store)
	if err := ca1.Initialize(); err != nil {
		t.Fatalf("Failed to initialize CA: %v", err)
	}

	// Save CA
	if err := ca1.SaveToStore(); err != nil {
		t.Fatalf("Failed to save CA: %v", err)
	}

	// Create new CA instance and load
	ca2 := NewCertAuthority(store)
	if err := ca2.LoadFromStore(); err != nil {
		t.Fatalf("Failed to load CA: %v", err)
	}

	// Verify loaded CA matches original
	if !ca2.IsInitialized() {
		t.Error("Loaded CA should be initialized")
	}

	if !ca1.rootCert.Equal(ca2.rootCert) {
		t.Error("Loaded root cert should match original")
	}

	if ca1.rootKey.N.Cmp(ca2.rootKey.N) != 0 {
		t.Error("Loaded root key should match original")
	}
}

func TestIssueNodeCertificate(t *testing.T) {
	// Set cluster encryption key
	key := DeriveKeyFromClusterID("test-cluster")
	if err := SetClusterEncryptionKey(key); err != nil {
		t.Fatalf("Failed to set cluster encryption key: %v", err)
	}

	// Create temporary BoltDB
	tmpDir, err := os.MkdirTemp("", "warren-ca-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := storage.NewBoltStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Create and initialize CA
	ca := NewCertAuthority(store)
	if err := ca.Initialize(); err != nil {
		t.Fatalf("Failed to initialize CA: %v", err)
	}

	tests := []struct {
		name   string
		nodeID string
		role   string
	}{
		{"Manager certificate", "node1", "manager"},
		{"Worker certificate", "node2", "worker"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Issue certificate
			cert, err := ca.IssueNodeCertificate(tt.nodeID, tt.role, []string{}, []net.IP{})
			if err != nil {
				t.Fatalf("Failed to issue certificate: %v", err)
			}

			// Verify certificate
			if cert.Leaf == nil {
				t.Error("Certificate Leaf should not be nil")
			}

			// Verify subject
			expectedCN := tt.role + "-" + tt.nodeID
			if cert.Leaf.Subject.CommonName != expectedCN {
				t.Errorf("Expected CN %s, got %s", expectedCN, cert.Leaf.Subject.CommonName)
			}

			// Verify validity period
			expectedExpiry := time.Now().Add(nodeCertValidity)
			if cert.Leaf.NotAfter.Before(expectedExpiry.Add(-time.Hour)) {
				t.Errorf("Cert expiry too early: %v, expected around %v", cert.Leaf.NotAfter, expectedExpiry)
			}

			// Verify key usages
			if cert.Leaf.KeyUsage&x509.KeyUsageDigitalSignature == 0 {
				t.Error("Certificate should have DigitalSignature key usage")
			}

			// Verify extended key usages
			hasClientAuth := false
			hasServerAuth := false
			for _, usage := range cert.Leaf.ExtKeyUsage {
				if usage == x509.ExtKeyUsageClientAuth {
					hasClientAuth = true
				}
				if usage == x509.ExtKeyUsageServerAuth {
					hasServerAuth = true
				}
			}
			if !hasClientAuth {
				t.Error("Certificate should have ClientAuth extended key usage")
			}
			if !hasServerAuth {
				t.Error("Certificate should have ServerAuth extended key usage")
			}
		})
	}
}

func TestIssueClientCertificate(t *testing.T) {
	// Set cluster encryption key
	key := DeriveKeyFromClusterID("test-cluster")
	if err := SetClusterEncryptionKey(key); err != nil {
		t.Fatalf("Failed to set cluster encryption key: %v", err)
	}

	// Create temporary BoltDB
	tmpDir, err := os.MkdirTemp("", "warren-ca-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := storage.NewBoltStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Create and initialize CA
	ca := NewCertAuthority(store)
	if err := ca.Initialize(); err != nil {
		t.Fatalf("Failed to initialize CA: %v", err)
	}

	// Issue client certificate
	clientID := "user@machine"
	cert, err := ca.IssueClientCertificate(clientID)
	if err != nil {
		t.Fatalf("Failed to issue client certificate: %v", err)
	}

	// Verify certificate
	if cert.Leaf == nil {
		t.Error("Certificate Leaf should not be nil")
	}

	// Verify subject
	expectedCN := "cli-" + clientID
	if cert.Leaf.Subject.CommonName != expectedCN {
		t.Errorf("Expected CN %s, got %s", expectedCN, cert.Leaf.Subject.CommonName)
	}

	// Verify only ClientAuth usage (not ServerAuth)
	hasClientAuth := false
	hasServerAuth := false
	for _, usage := range cert.Leaf.ExtKeyUsage {
		if usage == x509.ExtKeyUsageClientAuth {
			hasClientAuth = true
		}
		if usage == x509.ExtKeyUsageServerAuth {
			hasServerAuth = true
		}
	}
	if !hasClientAuth {
		t.Error("Client certificate should have ClientAuth extended key usage")
	}
	if hasServerAuth {
		t.Error("Client certificate should not have ServerAuth extended key usage")
	}
}

func TestVerifyCertificate(t *testing.T) {
	// Set cluster encryption key
	key := DeriveKeyFromClusterID("test-cluster")
	if err := SetClusterEncryptionKey(key); err != nil {
		t.Fatalf("Failed to set cluster encryption key: %v", err)
	}

	// Create temporary BoltDB
	tmpDir, err := os.MkdirTemp("", "warren-ca-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := storage.NewBoltStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Create and initialize CA
	ca := NewCertAuthority(store)
	if err := ca.Initialize(); err != nil {
		t.Fatalf("Failed to initialize CA: %v", err)
	}

	// Issue a certificate
	cert, err := ca.IssueNodeCertificate("test-node", "worker", []string{}, []net.IP{})
	if err != nil {
		t.Fatalf("Failed to issue certificate: %v", err)
	}

	// Verify certificate
	if err := ca.VerifyCertificate(cert.Leaf); err != nil {
		t.Errorf("Certificate verification failed: %v", err)
	}
}

func TestGetRootCACert(t *testing.T) {
	// Set cluster encryption key
	key := DeriveKeyFromClusterID("test-cluster")
	if err := SetClusterEncryptionKey(key); err != nil {
		t.Fatalf("Failed to set cluster encryption key: %v", err)
	}

	// Create temporary BoltDB
	tmpDir, err := os.MkdirTemp("", "warren-ca-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := storage.NewBoltStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Create and initialize CA
	ca := NewCertAuthority(store)
	if err := ca.Initialize(); err != nil {
		t.Fatalf("Failed to initialize CA: %v", err)
	}

	// Get root CA cert
	rootCertDER := ca.GetRootCACert()
	if rootCertDER == nil {
		t.Fatal("Root CA cert should not be nil")
	}

	// Parse and verify it's the same cert
	parsedCert, err := x509.ParseCertificate(rootCertDER)
	if err != nil {
		t.Fatalf("Failed to parse root CA cert: %v", err)
	}

	if !parsedCert.Equal(ca.rootCert) {
		t.Error("Returned root CA cert should match internal cert")
	}
}

func TestCertCache(t *testing.T) {
	// Set cluster encryption key
	key := DeriveKeyFromClusterID("test-cluster")
	if err := SetClusterEncryptionKey(key); err != nil {
		t.Fatalf("Failed to set cluster encryption key: %v", err)
	}

	// Create temporary BoltDB
	tmpDir, err := os.MkdirTemp("", "warren-ca-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := storage.NewBoltStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Create and initialize CA
	ca := NewCertAuthority(store)
	if err := ca.Initialize(); err != nil {
		t.Fatalf("Failed to initialize CA: %v", err)
	}

	// Issue certificate (should be cached)
	nodeID := "test-node"
	_, err = ca.IssueNodeCertificate(nodeID, "worker", []string{}, []net.IP{})
	if err != nil {
		t.Fatalf("Failed to issue certificate: %v", err)
	}

	// Retrieve from cache
	cached, exists := ca.GetCachedCert(nodeID)
	if !exists {
		t.Error("Certificate should be in cache")
	}

	if cached == nil {
		t.Error("Cached certificate should not be nil")
	}

	if cached.Cert.Subject.CommonName != "worker-"+nodeID {
		t.Errorf("Cached cert CN mismatch: %s", cached.Cert.Subject.CommonName)
	}
}
