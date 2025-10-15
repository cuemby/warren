package security

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"sync"
	"time"

	"github.com/cuemby/warren/pkg/storage"
)

// CertAuthority manages the cluster's certificate authority
type CertAuthority struct {
	rootCert  *x509.Certificate
	rootKey   *rsa.PrivateKey
	store     storage.Store
	certCache map[string]*CachedCert
	mu        sync.RWMutex
}

// CachedCert represents a cached certificate
type CachedCert struct {
	Cert      *x509.Certificate
	Key       *rsa.PrivateKey
	IssuedAt  time.Time
	ExpiresAt time.Time
}

// CAData represents the serialized CA data for storage
type CAData struct {
	RootCertDER []byte
	RootKeyDER  []byte
}

const (
	// Root CA validity: 10 years
	rootCAValidity = 10 * 365 * 24 * time.Hour
	// Node certificate validity: 90 days
	nodeCertValidity = 90 * 24 * time.Hour
	// Root CA key size: 4096 bits (long-lived, high security)
	rootKeySize = 4096
	// Node key size: 2048 bits (shorter-lived, faster)
	nodeKeySize = 2048
)

// NewCertAuthority creates a new certificate authority
func NewCertAuthority(store storage.Store) *CertAuthority {
	return &CertAuthority{
		store:     store,
		certCache: make(map[string]*CachedCert),
	}
}

// Initialize generates a new root CA certificate
func (ca *CertAuthority) Initialize() error {
	ca.mu.Lock()
	defer ca.mu.Unlock()

	// Generate root key
	rootKey, err := rsa.GenerateKey(rand.Reader, rootKeySize)
	if err != nil {
		return fmt.Errorf("failed to generate root key: %w", err)
	}

	// Create root CA certificate template
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return fmt.Errorf("failed to generate serial number: %w", err)
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Warren Cluster"},
			CommonName:   "Warren Root CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(rootCAValidity),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		IsCA:                  true,
		BasicConstraintsValid: true,
		MaxPathLen:            1,
		MaxPathLenZero:        false,
	}

	// Create self-signed certificate
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &rootKey.PublicKey, rootKey)
	if err != nil {
		return fmt.Errorf("failed to create root certificate: %w", err)
	}

	// Parse certificate
	rootCert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return fmt.Errorf("failed to parse root certificate: %w", err)
	}

	ca.rootCert = rootCert
	ca.rootKey = rootKey

	return nil
}

// LoadFromStore loads the CA from storage
func (ca *CertAuthority) LoadFromStore() error {
	ca.mu.Lock()
	defer ca.mu.Unlock()

	// Get CA data from storage
	data, err := ca.store.GetCA()
	if err != nil {
		return fmt.Errorf("failed to get CA from storage: %w", err)
	}

	var caData CAData
	if err := json.Unmarshal(data, &caData); err != nil {
		return fmt.Errorf("failed to unmarshal CA data: %w", err)
	}

	// Decrypt root key
	decryptedKey, err := Decrypt(caData.RootKeyDER)
	if err != nil {
		return fmt.Errorf("failed to decrypt root key: %w", err)
	}

	// Parse certificate
	rootCert, err := x509.ParseCertificate(caData.RootCertDER)
	if err != nil {
		return fmt.Errorf("failed to parse root certificate: %w", err)
	}

	// Parse private key
	rootKey, err := x509.ParsePKCS1PrivateKey(decryptedKey)
	if err != nil {
		return fmt.Errorf("failed to parse root key: %w", err)
	}

	ca.rootCert = rootCert
	ca.rootKey = rootKey

	return nil
}

// SaveToStore saves the CA to storage
func (ca *CertAuthority) SaveToStore() error {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	if ca.rootCert == nil || ca.rootKey == nil {
		return fmt.Errorf("CA not initialized")
	}

	// Encrypt root key
	rootKeyDER := x509.MarshalPKCS1PrivateKey(ca.rootKey)
	encryptedKey, err := Encrypt(rootKeyDER)
	if err != nil {
		return fmt.Errorf("failed to encrypt root key: %w", err)
	}

	// Serialize CA data
	caData := CAData{
		RootCertDER: ca.rootCert.Raw,
		RootKeyDER:  encryptedKey,
	}

	data, err := json.Marshal(caData)
	if err != nil {
		return fmt.Errorf("failed to marshal CA data: %w", err)
	}

	// Save to storage
	if err := ca.store.SaveCA(data); err != nil {
		return fmt.Errorf("failed to save CA to storage: %w", err)
	}

	return nil
}

// IssueNodeCertificate issues a certificate for a node (manager or worker)
func (ca *CertAuthority) IssueNodeCertificate(nodeID, role string, dnsNames []string, ipAddresses []net.IP) (*tls.Certificate, error) {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	if ca.rootCert == nil || ca.rootKey == nil {
		return nil, fmt.Errorf("CA not initialized")
	}

	// Generate node key
	nodeKey, err := rsa.GenerateKey(rand.Reader, nodeKeySize)
	if err != nil {
		return nil, fmt.Errorf("failed to generate node key: %w", err)
	}

	// Create certificate template
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Warren Cluster"},
			CommonName:   fmt.Sprintf("%s-%s", role, nodeID),
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(nodeCertValidity),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		DNSNames:    dnsNames,
		IPAddresses: ipAddresses,
	}

	// Create certificate signed by root CA
	certDER, err := x509.CreateCertificate(rand.Reader, template, ca.rootCert, &nodeKey.PublicKey, ca.rootKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create node certificate: %w", err)
	}

	// Parse certificate
	nodeCert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, fmt.Errorf("failed to parse node certificate: %w", err)
	}

	// Create TLS certificate
	tlsCert := &tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  nodeKey,
		Leaf:        nodeCert,
	}

	// Cache certificate
	ca.cacheCertificate(nodeID, nodeCert, nodeKey)

	return tlsCert, nil
}

// IssueClientCertificate issues a certificate for a CLI client
func (ca *CertAuthority) IssueClientCertificate(clientID string) (*tls.Certificate, error) {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	if ca.rootCert == nil || ca.rootKey == nil {
		return nil, fmt.Errorf("CA not initialized")
	}

	// Generate client key
	clientKey, err := rsa.GenerateKey(rand.Reader, nodeKeySize)
	if err != nil {
		return nil, fmt.Errorf("failed to generate client key: %w", err)
	}

	// Create certificate template
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Warren Cluster"},
			CommonName:   fmt.Sprintf("cli-%s", clientID),
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(nodeCertValidity),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	// Create certificate signed by root CA
	certDER, err := x509.CreateCertificate(rand.Reader, template, ca.rootCert, &clientKey.PublicKey, ca.rootKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create client certificate: %w", err)
	}

	// Parse certificate
	clientCert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, fmt.Errorf("failed to parse client certificate: %w", err)
	}

	// Create TLS certificate
	tlsCert := &tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  clientKey,
		Leaf:        clientCert,
	}

	// Cache certificate
	ca.cacheCertificate(clientID, clientCert, clientKey)

	return tlsCert, nil
}

// VerifyCertificate verifies a certificate against the root CA
func (ca *CertAuthority) VerifyCertificate(cert *x509.Certificate) error {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	if ca.rootCert == nil {
		return fmt.Errorf("CA not initialized")
	}

	// Create cert pool with root CA
	roots := x509.NewCertPool()
	roots.AddCert(ca.rootCert)

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

// GetRootCACert returns the root CA certificate in DER format
func (ca *CertAuthority) GetRootCACert() []byte {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	if ca.rootCert == nil {
		return nil
	}

	return ca.rootCert.Raw
}

// IsInitialized returns true if the CA is initialized
func (ca *CertAuthority) IsInitialized() bool {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	return ca.rootCert != nil && ca.rootKey != nil
}

// cacheCertificate adds a certificate to the cache
func (ca *CertAuthority) cacheCertificate(id string, cert *x509.Certificate, key *rsa.PrivateKey) {
	ca.certCache[id] = &CachedCert{
		Cert:      cert,
		Key:       key,
		IssuedAt:  cert.NotBefore,
		ExpiresAt: cert.NotAfter,
	}
}

// GetCachedCert retrieves a cached certificate
func (ca *CertAuthority) GetCachedCert(id string) (*CachedCert, bool) {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	cert, exists := ca.certCache[id]
	return cert, exists
}
