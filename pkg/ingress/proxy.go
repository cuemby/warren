package ingress

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/cuemby/warren/pkg/log"
	"github.com/cuemby/warren/pkg/storage"
	"github.com/cuemby/warren/pkg/types"
	"google.golang.org/grpc"
)

// Proxy is the main HTTP reverse proxy
type Proxy struct {
	store        storage.Store
	router       *Router
	lb           *LoadBalancer
	httpServer   *http.Server
	httpsServer  *http.Server
	tlsConfig    *tls.Config
	managerAddr  string
	grpcClient   *grpc.ClientConn
}

// NewProxy creates a new ingress proxy
func NewProxy(store storage.Store, managerAddr string, grpcClient *grpc.ClientConn) *Proxy {
	p := &Proxy{
		store:       store,
		managerAddr: managerAddr,
		grpcClient:  grpcClient,
	}

	// Initialize router with current ingresses
	ingresses, err := store.ListIngresses()
	if err != nil {
		log.Warn(fmt.Sprintf("Failed to load ingresses: %v", err))
		ingresses = []*types.Ingress{}
	}

	p.router = NewRouter(ingresses)
	p.lb = NewLoadBalancer(managerAddr, grpcClient)

	return p
}

// Start starts the HTTP and HTTPS proxy servers
func (p *Proxy) Start(ctx context.Context) error {
	// Load TLS certificates
	if err := p.loadTLSCertificates(); err != nil {
		log.Warn(fmt.Sprintf("Failed to load TLS certificates: %v", err))
	}

	// Create HTTP server on port 8000
	p.httpServer = &http.Server{
		Addr:         ":8000",
		Handler:      http.HandlerFunc(p.handleRequest),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start HTTP server
	httpListener, err := net.Listen("tcp", p.httpServer.Addr)
	if err != nil {
		return fmt.Errorf("failed to listen on :8000: %v", err)
	}

	log.Info(fmt.Sprintf("Ingress proxy listening on :8000 (HTTP)"))

	// Serve HTTP in goroutine
	go func() {
		if err := p.httpServer.Serve(httpListener); err != nil && err != http.ErrServerClosed {
			log.Error(fmt.Sprintf("HTTP server error: %v", err))
		}
	}()

	// Create HTTPS server on port 8443 if TLS is configured
	if p.tlsConfig != nil && len(p.tlsConfig.Certificates) > 0 {
		p.httpsServer = &http.Server{
			Addr:         ":8443",
			Handler:      http.HandlerFunc(p.handleRequest),
			TLSConfig:    p.tlsConfig,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  120 * time.Second,
		}

		// Start HTTPS server
		httpsListener, err := net.Listen("tcp", p.httpsServer.Addr)
		if err != nil {
			log.Warn(fmt.Sprintf("Failed to listen on :8443: %v", err))
		} else {
			log.Info(fmt.Sprintf("Ingress proxy listening on :8443 (HTTPS)"))

			// Serve HTTPS in goroutine
			go func() {
				tlsListener := tls.NewListener(httpsListener, p.tlsConfig)
				if err := p.httpsServer.Serve(tlsListener); err != nil && err != http.ErrServerClosed {
					log.Error(fmt.Sprintf("HTTPS server error: %v", err))
				}
			}()
		}
	} else {
		log.Info("No TLS certificates loaded, HTTPS disabled")
	}

	// Wait for context cancellation
	<-ctx.Done()
	log.Info("Shutting down ingress proxy")

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := p.httpServer.Shutdown(shutdownCtx); err != nil {
		log.Error(fmt.Sprintf("Failed to shutdown HTTP server: %v", err))
	}

	// Shutdown HTTPS server if running
	if p.httpsServer != nil {
		if err := p.httpsServer.Shutdown(shutdownCtx); err != nil {
			log.Error(fmt.Sprintf("Failed to shutdown HTTPS server: %v", err))
		}
	}

	return nil
}

// handleRequest handles incoming HTTP requests
func (p *Proxy) handleRequest(w http.ResponseWriter, r *http.Request) {
	host := r.Host
	path := r.URL.Path

	log.Debug(fmt.Sprintf("Ingress request: %s %s%s", r.Method, host, path))

	// Route the request
	backend := p.router.Route(host, path)
	if backend == nil {
		log.Warn(fmt.Sprintf("No backend found for %s%s", host, path))
		http.Error(w, "Service not found", http.StatusNotFound)
		return
	}

	log.Debug(fmt.Sprintf("Matched backend: service=%s, port=%d", backend.ServiceName, backend.Port))

	// Select backend instance via load balancer
	backendAddr, err := p.lb.SelectBackend(r.Context(), backend.ServiceName, backend.Port)
	if err != nil {
		log.Error(fmt.Sprintf("Failed to select backend: %v", err))
		http.Error(w, "Service temporarily unavailable", http.StatusServiceUnavailable)
		return
	}

	log.Debug(fmt.Sprintf("Selected backend address: %s", backendAddr))

	// Proxy the request
	if err := p.proxyRequest(w, r, backendAddr); err != nil {
		log.Error(fmt.Sprintf("Proxy error: %v", err))
		http.Error(w, "Bad gateway", http.StatusBadGateway)
		return
	}
}

// proxyRequest proxies the request to the backend
func (p *Proxy) proxyRequest(w http.ResponseWriter, r *http.Request, backendAddr string) error {
	// Parse backend URL
	targetURL, err := url.Parse(fmt.Sprintf("http://%s", backendAddr))
	if err != nil {
		return fmt.Errorf("invalid backend address: %v", err)
	}

	// Create reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// Customize the director to preserve the original request path
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		// Preserve original Host header for virtual hosting
		req.Host = r.Host
		// Add X-Forwarded headers
		req.Header.Set("X-Forwarded-For", r.RemoteAddr)
		req.Header.Set("X-Forwarded-Proto", "http")
		req.Header.Set("X-Forwarded-Host", r.Host)
	}

	// Custom error handler
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Error(fmt.Sprintf("Proxy error for %s: %v", backendAddr, err))
		http.Error(w, "Bad gateway", http.StatusBadGateway)
	}

	// Proxy the request
	proxy.ServeHTTP(w, r)

	return nil
}

// ReloadIngresses reloads the ingress rules from storage
func (p *Proxy) ReloadIngresses() error {
	ingresses, err := p.store.ListIngresses()
	if err != nil {
		return fmt.Errorf("failed to load ingresses: %v", err)
	}

	p.router.UpdateIngresses(ingresses)
	log.Info(fmt.Sprintf("Reloaded %d ingress rules", len(ingresses)))

	return nil
}

// loadTLSCertificates loads TLS certificates from storage and configures TLS
func (p *Proxy) loadTLSCertificates() error {
	// Get all TLS certificates from storage
	certs, err := p.store.ListTLSCertificates()
	if err != nil {
		return fmt.Errorf("failed to list certificates: %v", err)
	}

	if len(certs) == 0 {
		log.Debug("No TLS certificates found in storage")
		return nil
	}

	// Create TLS config
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		},
		PreferServerCipherSuites: true,
	}

	// Load all certificates
	var loadedCount int
	for _, cert := range certs {
		// Parse certificate and key
		tlsCert, err := tls.X509KeyPair(cert.CertPEM, cert.KeyPEM)
		if err != nil {
			log.Warn(fmt.Sprintf("Failed to load certificate %s: %v", cert.Name, err))
			continue
		}

		tlsConfig.Certificates = append(tlsConfig.Certificates, tlsCert)
		loadedCount++
		log.Debug(fmt.Sprintf("Loaded TLS certificate: %s (hosts: %v)", cert.Name, cert.Hosts))
	}

	if loadedCount > 0 {
		p.tlsConfig = tlsConfig
		log.Info(fmt.Sprintf("Loaded %d TLS certificate(s)", loadedCount))
	}

	return nil
}

// ReloadTLSCertificates reloads TLS certificates from storage
func (p *Proxy) ReloadTLSCertificates() error {
	if err := p.loadTLSCertificates(); err != nil {
		return err
	}

	// If HTTPS server is running, update its TLS config
	if p.httpsServer != nil && p.tlsConfig != nil {
		p.httpsServer.TLSConfig = p.tlsConfig
		log.Info("Reloaded TLS certificates for HTTPS server")
		return nil
	}

	// If HTTPS server is not running but we now have certificates, start it
	if p.httpsServer == nil && p.tlsConfig != nil && len(p.tlsConfig.Certificates) > 0 {
		p.httpsServer = &http.Server{
			Addr:         ":8443",
			Handler:      http.HandlerFunc(p.handleRequest),
			TLSConfig:    p.tlsConfig,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  120 * time.Second,
		}

		// Start HTTPS server
		httpsListener, err := net.Listen("tcp", p.httpsServer.Addr)
		if err != nil {
			log.Warn(fmt.Sprintf("Failed to listen on :8443: %v", err))
			return err
		}

		log.Info(fmt.Sprintf("Starting HTTPS server on :8443"))

		// Serve HTTPS in goroutine
		go func() {
			tlsListener := tls.NewListener(httpsListener, p.tlsConfig)
			if err := p.httpsServer.Serve(tlsListener); err != nil && err != http.ErrServerClosed {
				log.Error(fmt.Sprintf("HTTPS server error: %v", err))
			}
		}()
	}

	return nil
}
