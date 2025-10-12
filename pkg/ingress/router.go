package ingress

import (
	"strings"

	"github.com/cuemby/warren/pkg/types"
)

// Router handles request routing based on host and path
type Router struct {
	ingresses []*types.Ingress
}

// NewRouter creates a new router with the given ingresses
func NewRouter(ingresses []*types.Ingress) *Router {
	return &Router{
		ingresses: ingresses,
	}
}

// Route finds the matching backend for the given host and path
// Returns the matched backend or nil if no match
func (r *Router) Route(host, path string) *types.IngressBackend {
	// Try each ingress
	for _, ingress := range r.ingresses {
		if backend := r.matchIngress(ingress, host, path); backend != nil {
			return backend
		}
	}
	return nil
}

// matchIngress tries to match a single ingress against host and path
func (r *Router) matchIngress(ingress *types.Ingress, host, path string) *types.IngressBackend {
	// Try each rule in the ingress
	for _, rule := range ingress.Rules {
		if !r.matchHost(rule.Host, host) {
			continue
		}

		// Host matched, now try paths
		// Find longest matching path
		var bestMatch *types.IngressPath
		var bestMatchLen int

		for _, ingressPath := range rule.Paths {
			if r.matchPath(ingressPath, path) {
				pathLen := len(ingressPath.Path)
				if pathLen > bestMatchLen {
					bestMatch = ingressPath
					bestMatchLen = pathLen
				}
			}
		}

		if bestMatch != nil {
			return bestMatch.Backend
		}
	}

	return nil
}

// matchHost checks if the request host matches the ingress host pattern
func (r *Router) matchHost(pattern, host string) bool {
	// Empty pattern matches all hosts
	if pattern == "" {
		return true
	}

	// Remove port from host if present
	if idx := strings.IndexByte(host, ':'); idx != -1 {
		host = host[:idx]
	}

	// Exact match
	if pattern == host {
		return true
	}

	// Wildcard match (*.example.com)
	if strings.HasPrefix(pattern, "*.") {
		suffix := pattern[1:] // Remove "*"
		return strings.HasSuffix(host, suffix)
	}

	return false
}

// matchPath checks if the request path matches the ingress path
func (r *Router) matchPath(ingressPath *types.IngressPath, requestPath string) bool {
	pattern := ingressPath.Path
	pathType := ingressPath.PathType

	// Default to Prefix if not specified
	if pathType == "" {
		pathType = types.PathTypePrefix
	}

	switch pathType {
	case types.PathTypeExact:
		return pattern == requestPath

	case types.PathTypePrefix:
		// Match if request path starts with pattern
		// "/" matches everything
		if pattern == "/" {
			return true
		}
		// Ensure pattern ends with / or request path has / after pattern
		if strings.HasPrefix(requestPath, pattern) {
			// Exact match
			if len(requestPath) == len(pattern) {
				return true
			}
			// Pattern ends with / or next char is /
			if pattern[len(pattern)-1] == '/' {
				return true
			}
			if len(requestPath) > len(pattern) && requestPath[len(pattern)] == '/' {
				return true
			}
		}
		return false

	default:
		return false
	}
}

// UpdateIngresses updates the router with a new list of ingresses
func (r *Router) UpdateIngresses(ingresses []*types.Ingress) {
	r.ingresses = ingresses
}
