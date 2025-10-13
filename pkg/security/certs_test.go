package security

import (
	"crypto/x509"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cuemby/warren/pkg/storage"
)

func TestSaveLoadCertToFile(t *testing.T) {
	// Set cluster encryption key
	key := DeriveKeyFromClusterID("test-cluster")
	if err := SetClusterEncryptionKey(key); err != nil {
		t.Fatalf("Failed to set cluster encryption key: %v", err)
	}

	// Create temporary directories
	tmpStoreDir, err := os.MkdirTemp("", "warren-store-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp store dir: %v", err)
	}
	defer os.RemoveAll(tmpStoreDir)

	tmpCertDir, err := os.MkdirTemp("", "warren-cert-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp cert dir: %v", err)
	}
	defer os.RemoveAll(tmpCertDir)

	// Create CA and issue certificate
	store, err := storage.NewBoltStore(tmpStoreDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ca := NewCertAuthority(store)
	if err := ca.Initialize(); err != nil {
		t.Fatalf("Failed to initialize CA: %v", err)
	}

	cert, err := ca.IssueNodeCertificate("test-node", "worker", []string{}, []net.IP{})
	if err != nil {
		t.Fatalf("Failed to issue certificate: %v", err)
	}

	// Save certificate to file
	if err := SaveCertToFile(cert, tmpCertDir); err != nil {
		t.Fatalf("Failed to save certificate: %v", err)
	}

	// Verify files exist
	certPath := filepath.Join(tmpCertDir, "node.crt")
	keyPath := filepath.Join(tmpCertDir, "node.key")

	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		t.Error("Certificate file should exist")
	}
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		t.Error("Key file should exist")
	}

	// Load certificate from file
	loadedCert, err := LoadCertFromFile(tmpCertDir)
	if err != nil {
		t.Fatalf("Failed to load certificate: %v", err)
	}

	// Verify loaded certificate matches original
	if loadedCert.Leaf.Subject.CommonName != cert.Leaf.Subject.CommonName {
		t.Errorf("Loaded cert CN mismatch: expected %s, got %s",
			cert.Leaf.Subject.CommonName, loadedCert.Leaf.Subject.CommonName)
	}
}

func TestSaveLoadCACertToFile(t *testing.T) {
	// Set cluster encryption key
	key := DeriveKeyFromClusterID("test-cluster")
	if err := SetClusterEncryptionKey(key); err != nil {
		t.Fatalf("Failed to set cluster encryption key: %v", err)
	}

	// Create temporary directories
	tmpStoreDir, err := os.MkdirTemp("", "warren-store-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp store dir: %v", err)
	}
	defer os.RemoveAll(tmpStoreDir)

	tmpCertDir, err := os.MkdirTemp("", "warren-cert-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp cert dir: %v", err)
	}
	defer os.RemoveAll(tmpCertDir)

	// Create CA
	store, err := storage.NewBoltStore(tmpStoreDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ca := NewCertAuthority(store)
	if err := ca.Initialize(); err != nil {
		t.Fatalf("Failed to initialize CA: %v", err)
	}

	// Get CA cert
	caCertDER := ca.GetRootCACert()

	// Save CA cert to file
	if err := SaveCACertToFile(caCertDER, tmpCertDir); err != nil {
		t.Fatalf("Failed to save CA certificate: %v", err)
	}

	// Verify file exists
	caPath := filepath.Join(tmpCertDir, "ca.crt")
	if _, err := os.Stat(caPath); os.IsNotExist(err) {
		t.Error("CA certificate file should exist")
	}

	// Load CA cert from file
	loadedCACert, err := LoadCACertFromFile(tmpCertDir)
	if err != nil {
		t.Fatalf("Failed to load CA certificate: %v", err)
	}

	// Verify loaded CA cert matches original
	if !loadedCACert.Equal(ca.rootCert) {
		t.Error("Loaded CA cert should match original")
	}
}

func TestCertExists(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "warren-cert-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Initially should not exist
	if CertExists(tmpDir) {
		t.Error("Certificate should not exist initially")
	}

	// Create files
	certPath := filepath.Join(tmpDir, "node.crt")
	keyPath := filepath.Join(tmpDir, "node.key")
	caPath := filepath.Join(tmpDir, "ca.crt")

	_ = os.WriteFile(certPath, []byte("cert"), 0600)
	_ = os.WriteFile(keyPath, []byte("key"), 0600)
	_ = os.WriteFile(caPath, []byte("ca"), 0600)

	// Now should exist
	if !CertExists(tmpDir) {
		t.Error("Certificate should exist after creating files")
	}

	// Remove one file
	os.Remove(keyPath)

	// Should not exist (incomplete)
	if CertExists(tmpDir) {
		t.Error("Certificate should not exist with missing key file")
	}
}

func TestCertNeedsRotation(t *testing.T) {
	tests := []struct {
		name       string
		notAfter   time.Time
		needsRot   bool
	}{
		{
			name:     "Cert expiring in 1 day - needs rotation",
			notAfter: time.Now().Add(24 * time.Hour),
			needsRot: true,
		},
		{
			name:     "Cert expiring in 29 days - needs rotation",
			notAfter: time.Now().Add(29 * 24 * time.Hour),
			needsRot: true,
		},
		{
			name:     "Cert expiring in 31 days - no rotation needed",
			notAfter: time.Now().Add(31 * 24 * time.Hour),
			needsRot: false,
		},
		{
			name:     "Cert expiring in 60 days - no rotation needed",
			notAfter: time.Now().Add(60 * 24 * time.Hour),
			needsRot: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cert := &x509.Certificate{
				NotAfter: tt.notAfter,
			}

			needsRot := CertNeedsRotation(cert)
			if needsRot != tt.needsRot {
				t.Errorf("Expected needsRotation=%v, got %v", tt.needsRot, needsRot)
			}
		})
	}

	// Test nil certificate
	if !CertNeedsRotation(nil) {
		t.Error("Nil certificate should need rotation")
	}
}

func TestGetCertExpiry(t *testing.T) {
	expectedExpiry := time.Now().Add(90 * 24 * time.Hour)
	cert := &x509.Certificate{
		NotAfter: expectedExpiry,
	}

	expiry := GetCertExpiry(cert)
	if !expiry.Equal(expectedExpiry) {
		t.Errorf("Expected expiry %v, got %v", expectedExpiry, expiry)
	}

	// Test nil certificate
	nilExpiry := GetCertExpiry(nil)
	if !nilExpiry.IsZero() {
		t.Error("Nil certificate should return zero time")
	}
}

func TestGetCertTimeRemaining(t *testing.T) {
	expectedRemaining := 45 * 24 * time.Hour
	cert := &x509.Certificate{
		NotAfter: time.Now().Add(expectedRemaining),
	}

	remaining := GetCertTimeRemaining(cert)

	// Allow 1 second tolerance for test execution time
	diff := remaining - expectedRemaining
	if diff < -time.Second || diff > time.Second {
		t.Errorf("Expected remaining ~%v, got %v (diff: %v)", expectedRemaining, remaining, diff)
	}

	// Test nil certificate
	nilRemaining := GetCertTimeRemaining(nil)
	if nilRemaining != 0 {
		t.Error("Nil certificate should return zero duration")
	}
}

func TestValidateCertChain(t *testing.T) {
	// Set cluster encryption key
	key := DeriveKeyFromClusterID("test-cluster")
	if err := SetClusterEncryptionKey(key); err != nil {
		t.Fatalf("Failed to set cluster encryption key: %v", err)
	}

	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "warren-ca-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create CA and issue certificate
	store, err := storage.NewBoltStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ca := NewCertAuthority(store)
	if err := ca.Initialize(); err != nil {
		t.Fatalf("Failed to initialize CA: %v", err)
	}

	cert, err := ca.IssueNodeCertificate("test-node", "worker", []string{}, []net.IP{})
	if err != nil {
		t.Fatalf("Failed to issue certificate: %v", err)
	}

	// Validate cert chain
	if err := ValidateCertChain(cert.Leaf, ca.rootCert); err != nil {
		t.Errorf("Certificate chain validation failed: %v", err)
	}

	// Test with nil certificate
	if err := ValidateCertChain(nil, ca.rootCert); err == nil {
		t.Error("Validation should fail with nil certificate")
	}

	// Test with nil CA
	if err := ValidateCertChain(cert.Leaf, nil); err == nil {
		t.Error("Validation should fail with nil CA")
	}
}

func TestGetCertInfo(t *testing.T) {
	// Set cluster encryption key
	key := DeriveKeyFromClusterID("test-cluster")
	if err := SetClusterEncryptionKey(key); err != nil {
		t.Fatalf("Failed to set cluster encryption key: %v", err)
	}

	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "warren-ca-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create CA and issue certificate
	store, err := storage.NewBoltStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ca := NewCertAuthority(store)
	if err := ca.Initialize(); err != nil {
		t.Fatalf("Failed to initialize CA: %v", err)
	}

	cert, err := ca.IssueNodeCertificate("test-node", "worker", []string{}, []net.IP{})
	if err != nil {
		t.Fatalf("Failed to issue certificate: %v", err)
	}

	// Get cert info
	info := GetCertInfo(cert.Leaf)

	// Verify info contains expected fields
	if info["subject"] != "worker-test-node" {
		t.Errorf("Expected subject 'worker-test-node', got %v", info["subject"])
	}

	if info["issuer"] != "Warren Root CA" {
		t.Errorf("Expected issuer 'Warren Root CA', got %v", info["issuer"])
	}

	if info["is_ca"] != false {
		t.Error("Node certificate should not be a CA")
	}

	// Test with nil certificate
	nilInfo := GetCertInfo(nil)
	if _, hasError := nilInfo["error"]; !hasError {
		t.Error("Info for nil certificate should contain error")
	}
}

func TestGetCertDir(t *testing.T) {
	tests := []struct {
		nodeType string
		nodeID   string
	}{
		{"manager", "node1"},
		{"worker", "node2"},
	}

	for _, tt := range tests {
		t.Run(tt.nodeType+"-"+tt.nodeID, func(t *testing.T) {
			certDir, err := GetCertDir(tt.nodeType, tt.nodeID)
			if err != nil {
				t.Fatalf("Failed to get cert dir: %v", err)
			}

			// Verify path contains expected components
			expected := tt.nodeType + "-" + tt.nodeID
			if filepath.Base(certDir) != expected {
				t.Errorf("Expected cert dir to end with %s, got %s", expected, certDir)
			}
		})
	}
}

func TestGetCLICertDir(t *testing.T) {
	certDir, err := GetCLICertDir()
	if err != nil {
		t.Fatalf("Failed to get CLI cert dir: %v", err)
	}

	// Verify path ends with "cli"
	if filepath.Base(certDir) != "cli" {
		t.Errorf("Expected cert dir to end with 'cli', got %s", filepath.Base(certDir))
	}
}

func TestRemoveCerts(t *testing.T) {
	// Create temporary directory with files
	tmpDir, err := os.MkdirTemp("", "warren-cert-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create some files
	_ = os.WriteFile(filepath.Join(tmpDir, "node.crt"), []byte("cert"), 0600)
	_ = os.WriteFile(filepath.Join(tmpDir, "node.key"), []byte("key"), 0600)

	// Remove certificates
	if err := RemoveCerts(tmpDir); err != nil {
		t.Fatalf("Failed to remove certificates: %v", err)
	}

	// Verify directory no longer exists
	if _, err := os.Stat(tmpDir); !os.IsNotExist(err) {
		t.Error("Certificate directory should not exist after removal")
	}
}
