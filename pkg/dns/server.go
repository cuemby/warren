package dns

import (
	"context"
	"fmt"
	"sync"

	"github.com/cuemby/warren/pkg/log"
	"github.com/cuemby/warren/pkg/storage"
	"github.com/miekg/dns"
)

const (
	// DefaultListenAddr is the Docker-compatible DNS address
	DefaultListenAddr = "127.0.0.11:53"

	// DefaultDomain is the default search domain for Warren services
	DefaultDomain = "warren"

	// DefaultUpstream is the fallback DNS server for external queries
	DefaultUpstream = "8.8.8.8:53"
)

// Server is the Warren DNS server for service discovery
type Server struct {
	store      storage.Store
	resolver   *Resolver
	dnsServer  *dns.Server
	listenAddr string
	upstream   []string // External DNS servers for forwarding
	mu         sync.RWMutex
	running    bool
}

// Config holds DNS server configuration
type Config struct {
	ListenAddr string   // Address to listen on (default: 127.0.0.11:53)
	Domain     string   // Search domain (default: "warren")
	Upstream   []string // Upstream DNS servers (default: [8.8.8.8:53])
}

// NewServer creates a new DNS server
func NewServer(store storage.Store, config *Config) *Server {
	if config == nil {
		config = &Config{
			ListenAddr: DefaultListenAddr,
			Domain:     DefaultDomain,
			Upstream:   []string{DefaultUpstream},
		}
	}

	if config.ListenAddr == "" {
		config.ListenAddr = DefaultListenAddr
	}
	if config.Domain == "" {
		config.Domain = DefaultDomain
	}
	if len(config.Upstream) == 0 {
		config.Upstream = []string{DefaultUpstream}
	}

	s := &Server{
		store:      store,
		listenAddr: config.ListenAddr,
		upstream:   config.Upstream,
	}

	// Create resolver
	s.resolver = NewResolver(store, config.Domain, config.Upstream)

	return s
}

// Start starts the DNS server
func (s *Server) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("DNS server already running")
	}
	s.running = true
	s.mu.Unlock()

	log.Logger.Info().
		Str("component", "dns").
		Str("address", s.listenAddr).
		Msg("starting DNS server")

	// Create DNS handler
	mux := dns.NewServeMux()
	mux.HandleFunc(".", s.handleDNSQuery)

	// Create DNS server
	s.dnsServer = &dns.Server{
		Addr:    s.listenAddr,
		Net:     "udp",
		Handler: mux,
	}

	// Start server in goroutine
	errCh := make(chan error, 1)
	go func() {
		if err := s.dnsServer.ListenAndServe(); err != nil {
			log.Logger.Error().
				Err(err).
				Str("component", "dns").
				Msg("DNS server error")
			errCh <- err
		}
	}()

	// Wait for server to start or error
	select {
	case err := <-errCh:
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
		return err
	case <-ctx.Done():
		return s.Stop()
	default:
		log.Logger.Info().
			Str("component", "dns").
			Str("address", s.listenAddr).
			Msg("DNS server started successfully")
		return nil
	}
}

// Stop stops the DNS server
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	log.Logger.Info().
		Str("component", "dns").
		Msg("stopping DNS server")

	if s.dnsServer != nil {
		if err := s.dnsServer.Shutdown(); err != nil {
			log.Logger.Error().
				Err(err).
				Str("component", "dns").
				Msg("error stopping DNS server")
			return err
		}
	}

	s.running = false

	log.Logger.Info().
		Str("component", "dns").
		Msg("DNS server stopped")

	return nil
}

// handleDNSQuery handles incoming DNS queries
func (s *Server) handleDNSQuery(w dns.ResponseWriter, r *dns.Msg) {
	msg := &dns.Msg{}
	msg.SetReply(r)
	msg.Authoritative = true

	// Log query
	if len(r.Question) > 0 {
		q := r.Question[0]
		log.Logger.Debug().
			Str("component", "dns").
			Str("query", q.Name).
			Uint16("type", q.Qtype).
			Msg("DNS query received")
	}

	// Process each question
	for _, q := range r.Question {
		// We only handle A records for now
		if q.Qtype != dns.TypeA {
			log.Logger.Debug().
				Str("component", "dns").
				Str("query", q.Name).
				Uint16("type", q.Qtype).
				Msg("unsupported query type, forwarding to upstream")

			// Forward unsupported types to upstream
			s.forwardQuery(w, r)
			return
		}

		// Try to resolve the query
		answers, err := s.resolver.Resolve(q.Name)
		if err != nil {
			log.Logger.Debug().
				Err(err).
				Str("component", "dns").
				Str("query", q.Name).
				Msg("failed to resolve query, forwarding to upstream")

			// Forward to upstream DNS
			s.forwardQuery(w, r)
			return
		}

		// Add answers to response
		msg.Answer = append(msg.Answer, answers...)
	}

	// Send response
	if err := w.WriteMsg(msg); err != nil {
		log.Logger.Error().
			Err(err).
			Str("component", "dns").
			Msg("failed to write DNS response")
	}
}

// forwardQuery forwards a DNS query to upstream DNS servers
func (s *Server) forwardQuery(w dns.ResponseWriter, r *dns.Msg) {
	client := &dns.Client{Net: "udp"}

	// Try each upstream server
	for _, upstream := range s.upstream {
		resp, _, err := client.Exchange(r, upstream)
		if err != nil {
			log.Logger.Debug().
				Err(err).
				Str("component", "dns").
				Str("upstream", upstream).
				Msg("failed to forward query to upstream")
			continue
		}

		// Successfully forwarded, send response
		if err := w.WriteMsg(resp); err != nil {
			log.Logger.Error().
				Err(err).
				Str("component", "dns").
				Msg("failed to write forwarded DNS response")
		}
		return
	}

	// All upstreams failed, return SERVFAIL
	msg := &dns.Msg{}
	msg.SetReply(r)
	msg.Rcode = dns.RcodeServerFailure

	if err := w.WriteMsg(msg); err != nil {
		log.Logger.Error().
			Err(err).
			Str("component", "dns").
			Msg("failed to write DNS error response")
	}
}

// IsRunning returns true if the DNS server is running
func (s *Server) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}
