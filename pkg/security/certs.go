package security

import (
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	// Certificate rotation threshold: rotate when less than 30 days remaining
	certRotationThreshold = 30 * 24 * time.Hour

	// Default certificate directory
	defaultCertDir = ".warren/certs"
)

// GetCertDir returns the certificate directory for the given node type
func GetCertDir(nodeType, nodeID string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	certDir := filepath.Join(homeDir, defaultCertDir, fmt.Sprintf("%s-%s", nodeType, nodeID))
	return certDir, nil
}

// GetCLICertDir returns the certificate directory for CLI
func GetCLICertDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	certDir := filepath.Join(homeDir, defaultCertDir, "cli")
	return certDir, nil
}

// SaveCertToFile saves a TLS certificate to files (cert and key)
func SaveCertToFile(cert *tls.Certificate, certDir string) error {
	// Create directory
	if err := os.MkdirAll(certDir, 0700); err != nil {
		return fmt.Errorf("failed to create cert directory: %w", err)
	}

	// Save certificate
	certPath := filepath.Join(certDir, "node.crt")
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Certificate[0],
	})
	if err := os.WriteFile(certPath, certPEM, 0600); err != nil {
		return fmt.Errorf("failed to write certificate: %w", err)
	}

	// Save private key
	keyPath := filepath.Join(certDir, "node.key")
	privateKey, ok := cert.PrivateKey.(*rsa.PrivateKey)
	if !ok {
		return fmt.Errorf("private key is not RSA")
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	if err := os.WriteFile(keyPath, keyPEM, 0600); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	return nil
}

// LoadCertFromFile loads a TLS certificate from files
func LoadCertFromFile(certDir string) (*tls.Certificate, error) {
	certPath := filepath.Join(certDir, "node.crt")
	keyPath := filepath.Join(certDir, "node.key")

	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load certificate: %w", err)
	}

	// Parse certificate to populate Leaf field
	if cert.Leaf == nil {
		x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
		if err != nil {
			return nil, fmt.Errorf("failed to parse certificate: %w", err)
		}
		cert.Leaf = x509Cert
	}

	return &cert, nil
}

// SaveCACertToFile saves the CA certificate to a file
func SaveCACertToFile(caCert []byte, certDir string) error {
	// Create directory
	if err := os.MkdirAll(certDir, 0700); err != nil {
		return fmt.Errorf("failed to create cert directory: %w", err)
	}

	// Save CA certificate
	caPath := filepath.Join(certDir, "ca.crt")
	caPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caCert,
	})
	if err := os.WriteFile(caPath, caPEM, 0644); err != nil {
		return fmt.Errorf("failed to write CA certificate: %w", err)
	}

	return nil
}

// LoadCACertFromFile loads the CA certificate from a file
func LoadCACertFromFile(certDir string) (*x509.Certificate, error) {
	caPath := filepath.Join(certDir, "ca.crt")
	caPEM, err := os.ReadFile(caPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %w", err)
	}

	// Decode PEM
	block, _ := pem.Decode(caPEM)
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("failed to decode CA certificate PEM")
	}

	// Parse certificate
	caCert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CA certificate: %w", err)
	}

	return caCert, nil
}

// CertExists checks if a certificate exists in the given directory
func CertExists(certDir string) bool {
	certPath := filepath.Join(certDir, "node.crt")
	keyPath := filepath.Join(certDir, "node.key")
	caPath := filepath.Join(certDir, "ca.crt")

	_, err1 := os.Stat(certPath)
	_, err2 := os.Stat(keyPath)
	_, err3 := os.Stat(caPath)

	return err1 == nil && err2 == nil && err3 == nil
}

// CertNeedsRotation returns true if the certificate should be rotated
// This happens when less than 30 days remain until expiry
func CertNeedsRotation(cert *x509.Certificate) bool {
	if cert == nil {
		return true
	}

	timeUntilExpiry := time.Until(cert.NotAfter)
	return timeUntilExpiry < certRotationThreshold
}

// GetCertExpiry returns the expiry time of the certificate
func GetCertExpiry(cert *x509.Certificate) time.Time {
	if cert == nil {
		return time.Time{}
	}
	return cert.NotAfter
}

// GetCertTimeRemaining returns the time remaining until certificate expiry
func GetCertTimeRemaining(cert *x509.Certificate) time.Duration {
	if cert == nil {
		return 0
	}
	return time.Until(cert.NotAfter)
}

// ValidateCertChain validates that a certificate is signed by the CA
func ValidateCertChain(cert, ca *x509.Certificate) error {
	if cert == nil {
		return fmt.Errorf("certificate is nil")
	}
	if ca == nil {
		return fmt.Errorf("CA certificate is nil")
	}

	// Create cert pool with CA
	roots := x509.NewCertPool()
	roots.AddCert(ca)

	// Verify certificate
	opts := x509.VerifyOptions{
		Roots:     roots,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
	}

	if _, err := cert.Verify(opts); err != nil {
		return fmt.Errorf("certificate verification failed: %w", err)
	}

	return nil
}

// GetCertInfo returns human-readable information about a certificate
func GetCertInfo(cert *x509.Certificate) map[string]interface{} {
	if cert == nil {
		return map[string]interface{}{"error": "certificate is nil"}
	}

	return map[string]interface{}{
		"subject":       cert.Subject.CommonName,
		"issuer":        cert.Issuer.CommonName,
		"serial_number": cert.SerialNumber.String(),
		"not_before":    cert.NotBefore.Format(time.RFC3339),
		"not_after":     cert.NotAfter.Format(time.RFC3339),
		"is_ca":         cert.IsCA,
		"key_usage":     describeKeyUsage(cert.KeyUsage),
		"ext_key_usage": describeExtKeyUsage(cert.ExtKeyUsage),
	}
}

// describeKeyUsage converts x509.KeyUsage to human-readable string
func describeKeyUsage(usage x509.KeyUsage) []string {
	var usages []string
	if usage&x509.KeyUsageDigitalSignature != 0 {
		usages = append(usages, "DigitalSignature")
	}
	if usage&x509.KeyUsageKeyEncipherment != 0 {
		usages = append(usages, "KeyEncipherment")
	}
	if usage&x509.KeyUsageCertSign != 0 {
		usages = append(usages, "CertSign")
	}
	if usage&x509.KeyUsageCRLSign != 0 {
		usages = append(usages, "CRLSign")
	}
	return usages
}

// describeExtKeyUsage converts []x509.ExtKeyUsage to human-readable strings
func describeExtKeyUsage(usages []x509.ExtKeyUsage) []string {
	var result []string
	for _, usage := range usages {
		switch usage {
		case x509.ExtKeyUsageClientAuth:
			result = append(result, "ClientAuth")
		case x509.ExtKeyUsageServerAuth:
			result = append(result, "ServerAuth")
		}
	}
	return result
}

// RemoveCerts removes all certificates from a directory
func RemoveCerts(certDir string) error {
	return os.RemoveAll(certDir)
}
