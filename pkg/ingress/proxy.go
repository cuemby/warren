package ingress

import (
	"context"
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
	store       storage.Store
	router      *Router
	lb          *LoadBalancer
	httpServer  *http.Server
	managerAddr string
	grpcClient  *grpc.ClientConn
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

// Start starts the HTTP proxy server on port 8000 (M7.1 MVP - port 80 in M7.2)
func (p *Proxy) Start(ctx context.Context) error {
	// Create HTTP server
	// TODO(M7.2): Change to port 80 with proper capabilities/permissions
	p.httpServer = &http.Server{
		Addr:         ":8000",
		Handler:      http.HandlerFunc(p.handleRequest),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start HTTP server
	listener, err := net.Listen("tcp", p.httpServer.Addr)
	if err != nil {
		return fmt.Errorf("failed to listen on :8000: %v", err)
	}

	log.Info(fmt.Sprintf("Ingress proxy listening on :8000"))

	// Serve in goroutine
	go func() {
		if err := p.httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Error(fmt.Sprintf("HTTP server error: %v", err))
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()
	log.Info("Shutting down ingress proxy")

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := p.httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("failed to shutdown HTTP server: %v", err)
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
