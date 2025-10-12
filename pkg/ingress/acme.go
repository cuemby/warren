package ingress

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"sync"
	"time"

	"github.com/cuemby/warren/pkg/log"
	"github.com/cuemby/warren/pkg/storage"
	"github.com/cuemby/warren/pkg/types"
	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"
)

// ACMEClient manages Let's Encrypt certificate issuance and renewal
type ACMEClient struct {
	store         storage.Store
	proxy         *Proxy
	client        *lego.Client
	user          *ACMEUser
	challengeProvider *HTTP01Provider
	mu            sync.RWMutex
}

// ACMEUser implements the required user interface for ACME registration
type ACMEUser struct {
	Email        string
	Registration *registration.Resource
	key          crypto.PrivateKey
}

func (u *ACMEUser) GetEmail() string {
	return u.Email
}

func (u *ACMEUser) GetRegistration() *registration.Resource {
	return u.Registration
}

func (u *ACMEUser) GetPrivateKey() crypto.PrivateKey {
	return u.key
}

// HTTP01Provider implements the lego HTTP-01 challenge provider interface
type HTTP01Provider struct {
	proxy *Proxy
	mu    sync.RWMutex
	// Map of domain -> (token -> keyAuth)
	challenges map[string]map[string]string
}

// NewHTTP01Provider creates a new HTTP-01 challenge provider
func NewHTTP01Provider(proxy *Proxy) *HTTP01Provider {
	return &HTTP01Provider{
		proxy:      proxy,
		challenges: make(map[string]map[string]string),
	}
}

// Present presents the HTTP-01 challenge by storing it for the proxy to serve
func (p *HTTP01Provider) Present(domain, token, keyAuth string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.challenges[domain] == nil {
		p.challenges[domain] = make(map[string]string)
	}
	p.challenges[domain][token] = keyAuth

	log.Info(fmt.Sprintf("ACME: Presenting challenge for domain %s, token %s", domain, token))
	return nil
}

// CleanUp removes the HTTP-01 challenge after verification
func (p *HTTP01Provider) CleanUp(domain, token, keyAuth string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if domainChallenges, exists := p.challenges[domain]; exists {
		delete(domainChallenges, token)
		if len(domainChallenges) == 0 {
			delete(p.challenges, domain)
		}
	}

	log.Info(fmt.Sprintf("ACME: Cleaned up challenge for domain %s, token %s", domain, token))
	return nil
}

// GetKeyAuth retrieves the key authorization for a given domain and token
func (p *HTTP01Provider) GetKeyAuth(domain, token string) (string, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if domainChallenges, exists := p.challenges[domain]; exists {
		keyAuth, ok := domainChallenges[token]
		return keyAuth, ok
	}
	return "", false
}

// NewACMEClient creates a new ACME client
func NewACMEClient(store storage.Store, proxy *Proxy, email string) (*ACMEClient, error) {
	// Generate private key for ACME account
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %v", err)
	}

	// Create ACME user
	user := &ACMEUser{
		Email: email,
		key:   privateKey,
	}

	// Create lego configuration
	config := lego.NewConfig(user)

	// Use Let's Encrypt staging for development
	// Production: https://acme-v02.api.letsencrypt.org/directory
	// Staging: https://acme-staging-v02.api.letsencrypt.org/directory
	config.CADirURL = "https://acme-staging-v02.api.letsencrypt.org/directory"
	config.Certificate.KeyType = certcrypto.RSA2048

	// Create lego client
	client, err := lego.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create lego client: %v", err)
	}

	// Create HTTP-01 challenge provider
	challengeProvider := NewHTTP01Provider(proxy)

	// Set HTTP-01 provider
	err = client.Challenge.SetHTTP01Provider(challengeProvider)
	if err != nil {
		return nil, fmt.Errorf("failed to set HTTP-01 provider: %v", err)
	}

	// Register user with ACME server
	reg, err := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
	if err != nil {
		return nil, fmt.Errorf("failed to register with ACME server: %v", err)
	}
	user.Registration = reg

	log.Info(fmt.Sprintf("ACME client registered with email: %s", email))

	return &ACMEClient{
		store:             store,
		proxy:             proxy,
		client:            client,
		user:              user,
		challengeProvider: challengeProvider,
	}, nil
}

// ObtainCertificate requests a new certificate from Let's Encrypt
func (a *ACMEClient) ObtainCertificate(domains []string) (*types.TLSCertificate, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	log.Info(fmt.Sprintf("ACME: Requesting certificate for domains: %v", domains))

	// Request certificate
	request := certificate.ObtainRequest{
		Domains: domains,
		Bundle:  true,
	}

	certificates, err := a.client.Certificate.Obtain(request)
	if err != nil {
		return nil, fmt.Errorf("failed to obtain certificate: %v", err)
	}

	// Parse certificate to extract metadata
	block, _ := pem.Decode(certificates.Certificate)
	if block == nil {
		return nil, fmt.Errorf("failed to decode certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %v", err)
	}

	// Create TLS certificate
	tlsCert := &types.TLSCertificate{
		ID:        fmt.Sprintf("acme-%s", time.Now().Format("20060102-150405")),
		Name:      fmt.Sprintf("acme-%s", domains[0]),
		Hosts:     domains,
		CertPEM:   certificates.Certificate,
		KeyPEM:    certificates.PrivateKey,
		Issuer:    cert.Issuer.CommonName,
		NotBefore: cert.NotBefore,
		NotAfter:  cert.NotAfter,
		AutoRenew: true,
		Labels:    map[string]string{"acme": "true"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	log.Info(fmt.Sprintf("ACME: Certificate obtained for %v, valid until %s", domains, cert.NotAfter))

	return tlsCert, nil
}

// RenewCertificate renews an existing certificate
func (a *ACMEClient) RenewCertificate(cert *types.TLSCertificate) (*types.TLSCertificate, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	log.Info(fmt.Sprintf("ACME: Renewing certificate for %v", cert.Hosts))

	// Parse existing certificate and key
	certResource := &certificate.Resource{
		Certificate: cert.CertPEM,
		PrivateKey:  cert.KeyPEM,
	}

	// Renew certificate
	renewed, err := a.client.Certificate.Renew(*certResource, true, false, "")
	if err != nil {
		return nil, fmt.Errorf("failed to renew certificate: %v", err)
	}

	// Parse renewed certificate to extract metadata
	block, _ := pem.Decode(renewed.Certificate)
	if block == nil {
		return nil, fmt.Errorf("failed to decode renewed certificate PEM")
	}

	renewedCert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse renewed certificate: %v", err)
	}

	// Update certificate
	cert.CertPEM = renewed.Certificate
	cert.KeyPEM = renewed.PrivateKey
	cert.Issuer = renewedCert.Issuer.CommonName
	cert.NotBefore = renewedCert.NotBefore
	cert.NotAfter = renewedCert.NotAfter
	cert.UpdatedAt = time.Now()

	log.Info(fmt.Sprintf("ACME: Certificate renewed for %v, valid until %s", cert.Hosts, renewedCert.NotAfter))

	return cert, nil
}

// CheckAndRenewCertificates checks all auto-renewable certificates and renews if needed
func (a *ACMEClient) CheckAndRenewCertificates() error {
	// Get all certificates
	certs, err := a.store.ListTLSCertificates()
	if err != nil {
		return fmt.Errorf("failed to list certificates: %v", err)
	}

	now := time.Now()
	renewalThreshold := 30 * 24 * time.Hour // 30 days before expiry

	for _, cert := range certs {
		// Skip certificates that don't have auto-renewal enabled
		if !cert.AutoRenew {
			continue
		}

		// Check if certificate needs renewal (30 days before expiry)
		if cert.NotAfter.Sub(now) > renewalThreshold {
			continue
		}

		log.Info(fmt.Sprintf("ACME: Certificate %s expires in %s, renewing...", cert.Name, cert.NotAfter.Sub(now)))

		// Renew certificate
		renewed, err := a.RenewCertificate(cert)
		if err != nil {
			log.Error(fmt.Sprintf("ACME: Failed to renew certificate %s: %v", cert.Name, err))
			continue
		}

		// Update certificate in storage
		if err := a.store.UpdateTLSCertificate(renewed); err != nil {
			log.Error(fmt.Sprintf("ACME: Failed to update renewed certificate %s: %v", cert.Name, err))
			continue
		}

		// Reload certificates in proxy
		if err := a.proxy.ReloadTLSCertificates(); err != nil {
			log.Warn(fmt.Sprintf("ACME: Failed to reload certificates: %v", err))
		}

		log.Info(fmt.Sprintf("ACME: Successfully renewed certificate %s", cert.Name))
	}

	return nil
}

// StartRenewalJob starts a background job that checks for certificate renewal
func (a *ACMEClient) StartRenewalJob() {
	ticker := time.NewTicker(24 * time.Hour) // Check daily
	go func() {
		for range ticker.C {
			if err := a.CheckAndRenewCertificates(); err != nil {
				log.Error(fmt.Sprintf("ACME: Renewal job error: %v", err))
			}
		}
	}()

	log.Info("ACME: Certificate renewal job started (checking daily)")
}

// SaveACMEAccount saves the ACME account to storage as a secret
func (a *ACMEClient) SaveACMEAccount() error {
	// Marshal account data
	accountData := map[string]interface{}{
		"email":        a.user.Email,
		"registration": a.user.Registration,
	}

	data, err := json.Marshal(accountData)
	if err != nil {
		return fmt.Errorf("failed to marshal account data: %v", err)
	}

	// Store as secret (for future: save private key securely)
	secret := &types.Secret{
		ID:        "acme-account",
		Name:      "acme-account",
		Data:      data,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return a.store.CreateSecret(secret)
}
