package ingress

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/cuemby/warren/pkg/log"
	"github.com/cuemby/warren/pkg/types"
	"golang.org/x/time/rate"
)

// Middleware handles request modification, rate limiting, and access control
type Middleware struct {
	rateLimiters map[string]*rate.Limiter
	mu           sync.RWMutex
}

// NewMiddleware creates a new middleware handler
func NewMiddleware() *Middleware {
	return &Middleware{
		rateLimiters: make(map[string]*rate.Limiter),
	}
}

// ApplyHeaderManipulation applies header manipulation rules to the request
func (m *Middleware) ApplyHeaderManipulation(r *http.Request, config *types.HeaderManipulation) {
	if config == nil {
		return
	}

	// Add headers (only if not already present)
	for key, value := range config.Add {
		if r.Header.Get(key) == "" {
			r.Header.Set(key, value)
			log.Debug(fmt.Sprintf("Added header %s: %s", key, value))
		}
	}

	// Set headers (overwrite if present)
	for key, value := range config.Set {
		r.Header.Set(key, value)
		log.Debug(fmt.Sprintf("Set header %s: %s", key, value))
	}

	// Remove headers
	for _, key := range config.Remove {
		r.Header.Del(key)
		log.Debug(fmt.Sprintf("Removed header %s", key))
	}
}

// AddProxyHeaders adds standard proxy headers (X-Forwarded-For, X-Real-IP, etc.)
func (m *Middleware) AddProxyHeaders(r *http.Request) {
	// Get client IP
	clientIP := getClientIP(r)

	// X-Real-IP: The original client IP
	if r.Header.Get("X-Real-IP") == "" {
		r.Header.Set("X-Real-IP", clientIP)
	}

	// X-Forwarded-For: Chain of proxies
	if prior := r.Header.Get("X-Forwarded-For"); prior != "" {
		r.Header.Set("X-Forwarded-For", prior+", "+clientIP)
	} else {
		r.Header.Set("X-Forwarded-For", clientIP)
	}

	// X-Forwarded-Proto: Original protocol (http or https)
	if r.Header.Get("X-Forwarded-Proto") == "" {
		proto := "http"
		if r.TLS != nil {
			proto = "https"
		}
		r.Header.Set("X-Forwarded-Proto", proto)
	}

	// X-Forwarded-Host: Original host header
	if r.Header.Get("X-Forwarded-Host") == "" {
		r.Header.Set("X-Forwarded-Host", r.Host)
	}

	log.Debug(fmt.Sprintf("Added proxy headers for client %s", clientIP))
}

// ApplyPathRewrite applies path rewriting rules to the request
func (m *Middleware) ApplyPathRewrite(r *http.Request, config *types.PathRewrite) {
	if config == nil {
		return
	}

	originalPath := r.URL.Path
	originalRawQuery := r.URL.RawQuery

	// ReplacePath takes precedence over StripPrefix
	if config.ReplacePath != "" {
		r.URL.Path = config.ReplacePath
		log.Debug(fmt.Sprintf("Replaced path %s → %s", originalPath, r.URL.Path))
	} else if config.StripPrefix != "" {
		// Strip prefix if present
		if strings.HasPrefix(r.URL.Path, config.StripPrefix) {
			r.URL.Path = strings.TrimPrefix(r.URL.Path, config.StripPrefix)
			// Ensure path starts with /
			if !strings.HasPrefix(r.URL.Path, "/") {
				r.URL.Path = "/" + r.URL.Path
			}
			log.Debug(fmt.Sprintf("Stripped prefix %s: %s → %s", config.StripPrefix, originalPath, r.URL.Path))
		}
	}

	// Preserve query parameters
	r.URL.RawQuery = originalRawQuery
}

// CheckRateLimit checks if the request should be rate limited
func (m *Middleware) CheckRateLimit(r *http.Request, config *types.RateLimit) bool {
	if config == nil {
		return true // No rate limit configured, allow request
	}

	clientIP := getClientIP(r)

	m.mu.Lock()
	limiter, exists := m.rateLimiters[clientIP]
	if !exists {
		// Create new rate limiter for this IP
		limiter = rate.NewLimiter(rate.Limit(config.RequestsPerSecond), config.Burst)
		m.rateLimiters[clientIP] = limiter
		log.Debug(fmt.Sprintf("Created rate limiter for %s: %.2f req/s, burst %d", clientIP, config.RequestsPerSecond, config.Burst))
	}
	m.mu.Unlock()

	// Check if request is allowed
	allowed := limiter.Allow()
	if !allowed {
		log.Warn(fmt.Sprintf("Rate limit exceeded for %s", clientIP))
	}

	return allowed
}

// CheckAccessControl checks if the request is allowed based on IP access control
func (m *Middleware) CheckAccessControl(r *http.Request, config *types.AccessControl) (bool, string) {
	if config == nil {
		return true, "" // No access control configured, allow request
	}

	clientIP := getClientIP(r)
	ip := net.ParseIP(clientIP)
	if ip == nil {
		log.Warn(fmt.Sprintf("Invalid client IP: %s", clientIP))
		return false, "Invalid client IP"
	}

	// Check deny list first (deny takes precedence)
	for _, cidr := range config.DeniedIPs {
		if matchCIDR(ip, cidr) {
			log.Warn(fmt.Sprintf("Access denied for %s (matched deny rule: %s)", clientIP, cidr))
			return false, "Access denied by IP filter"
		}
	}

	// If allow list is specified, client must match at least one entry
	if len(config.AllowedIPs) > 0 {
		for _, cidr := range config.AllowedIPs {
			if matchCIDR(ip, cidr) {
				log.Debug(fmt.Sprintf("Access allowed for %s (matched allow rule: %s)", clientIP, cidr))
				return true, ""
			}
		}
		// Client didn't match any allow rule
		log.Warn(fmt.Sprintf("Access denied for %s (not in allow list)", clientIP))
		return false, "Access denied by IP filter"
	}

	// No deny match and no allow list = allow
	return true, ""
}

// CleanupRateLimiters removes old rate limiters (call periodically)
func (m *Middleware) CleanupRateLimiters() {
	// This is a simple implementation - in production, you'd track last access time
	// and remove limiters that haven't been used in a while
	m.mu.Lock()
	defer m.mu.Unlock()

	// For now, just clear all if we have too many
	if len(m.rateLimiters) > 10000 {
		log.Info(fmt.Sprintf("Clearing rate limiters (count: %d)", len(m.rateLimiters)))
		m.rateLimiters = make(map[string]*rate.Limiter)
	}
}

// StartCleanupJob starts a background job to clean up old rate limiters
func (m *Middleware) StartCleanupJob() {
	ticker := time.NewTicker(1 * time.Hour)
	go func() {
		for range ticker.C {
			m.CleanupRateLimiters()
		}
	}()
	log.Info("Middleware cleanup job started (running hourly)")
}

// Helper functions

// getClientIP extracts the client IP from the request
func getClientIP(r *http.Request) string {
	// Try X-Forwarded-For first
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the chain
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}

	// Try X-Real-IP
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// matchCIDR checks if an IP matches a CIDR range
func matchCIDR(ip net.IP, cidr string) bool {
	// Handle single IP addresses (convert to /32 or /128)
	if !strings.Contains(cidr, "/") {
		parsedIP := net.ParseIP(cidr)
		if parsedIP == nil {
			return false
		}
		return ip.Equal(parsedIP)
	}

	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		log.Warn(fmt.Sprintf("Invalid CIDR: %s", cidr))
		return false
	}

	return ipNet.Contains(ip)
}
