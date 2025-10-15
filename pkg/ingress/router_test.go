package ingress

import (
	"testing"

	"github.com/cuemby/warren/pkg/types"
)

// TestRouterHostMatching tests host pattern matching
func TestRouterHostMatching(t *testing.T) {
	r := &Router{}

	tests := []struct {
		name     string
		pattern  string
		host     string
		expected bool
	}{
		// Exact matches
		{
			name:     "exact match",
			pattern:  "example.com",
			host:     "example.com",
			expected: true,
		},
		{
			name:     "exact match with port",
			pattern:  "example.com",
			host:     "example.com:8080",
			expected: true,
		},
		{
			name:     "exact mismatch",
			pattern:  "example.com",
			host:     "other.com",
			expected: false,
		},
		// Wildcard matches
		{
			name:     "wildcard match subdomain",
			pattern:  "*.example.com",
			host:     "api.example.com",
			expected: true,
		},
		{
			name:     "wildcard match nested subdomain",
			pattern:  "*.example.com",
			host:     "api.v1.example.com",
			expected: true,
		},
		{
			name:     "wildcard no match root",
			pattern:  "*.example.com",
			host:     "example.com",
			expected: false,
		},
		{
			name:     "wildcard no match different domain",
			pattern:  "*.example.com",
			host:     "other.com",
			expected: false,
		},
		// Empty pattern (catch-all)
		{
			name:     "empty pattern matches all",
			pattern:  "",
			host:     "any-host.com",
			expected: true,
		},
		// Edge cases
		{
			name:     "case sensitive match",
			pattern:  "Example.com",
			host:     "example.com",
			expected: false,
		},
		{
			name:     "IPv4 address",
			pattern:  "192.168.1.1",
			host:     "192.168.1.1",
			expected: true,
		},
		{
			name:     "localhost",
			pattern:  "localhost",
			host:     "localhost",
			expected: true,
		},
		{
			name:     "localhost with port",
			pattern:  "localhost",
			host:     "localhost:8080",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := r.matchHost(tt.pattern, tt.host)
			if result != tt.expected {
				t.Errorf("matchHost(%q, %q) = %v, want %v", tt.pattern, tt.host, result, tt.expected)
			}
		})
	}
}

// TestRouterPathMatching tests path matching logic
func TestRouterPathMatching(t *testing.T) {
	r := &Router{}

	tests := []struct {
		name        string
		ingressPath *types.IngressPath
		requestPath string
		expected    bool
	}{
		// Prefix matching
		{
			name: "prefix match root",
			ingressPath: &types.IngressPath{
				Path:     "/",
				PathType: types.PathTypePrefix,
			},
			requestPath: "/anything",
			expected:    true,
		},
		{
			name: "prefix match specific path",
			ingressPath: &types.IngressPath{
				Path:     "/api",
				PathType: types.PathTypePrefix,
			},
			requestPath: "/api/users",
			expected:    true,
		},
		{
			name: "prefix no match",
			ingressPath: &types.IngressPath{
				Path:     "/api",
				PathType: types.PathTypePrefix,
			},
			requestPath: "/web",
			expected:    false,
		},
		{
			name: "prefix match exact",
			ingressPath: &types.IngressPath{
				Path:     "/api",
				PathType: types.PathTypePrefix,
			},
			requestPath: "/api",
			expected:    true,
		},
		// Exact matching
		{
			name: "exact match",
			ingressPath: &types.IngressPath{
				Path:     "/api/users",
				PathType: types.PathTypeExact,
			},
			requestPath: "/api/users",
			expected:    true,
		},
		{
			name: "exact no match with subpath",
			ingressPath: &types.IngressPath{
				Path:     "/api/users",
				PathType: types.PathTypeExact,
			},
			requestPath: "/api/users/123",
			expected:    false,
		},
		{
			name: "exact no match different path",
			ingressPath: &types.IngressPath{
				Path:     "/api/users",
				PathType: types.PathTypeExact,
			},
			requestPath: "/api/posts",
			expected:    false,
		},
		// Default to Prefix
		{
			name: "no pathType defaults to Prefix",
			ingressPath: &types.IngressPath{
				Path:     "/api",
				PathType: "",
			},
			requestPath: "/api/users",
			expected:    true,
		},
		// Edge cases
		{
			name: "empty path pattern",
			ingressPath: &types.IngressPath{
				Path:     "",
				PathType: types.PathTypePrefix,
			},
			requestPath: "/anything",
			expected:    true,
		},
		{
			name: "trailing slash in pattern",
			ingressPath: &types.IngressPath{
				Path:     "/api/",
				PathType: types.PathTypePrefix,
			},
			requestPath: "/api/users",
			expected:    true,
		},
		{
			name: "no trailing slash in request",
			ingressPath: &types.IngressPath{
				Path:     "/api",
				PathType: types.PathTypePrefix,
			},
			requestPath: "/api",
			expected:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := r.matchPath(tt.ingressPath, tt.requestPath)
			if result != tt.expected {
				t.Errorf("matchPath(%v, %q) = %v, want %v", tt.ingressPath.Path, tt.requestPath, result, tt.expected)
			}
		})
	}
}

// TestRouterRoute tests full routing with ingresses
func TestRouterRoute(t *testing.T) {
	// Create test ingresses
	ingresses := []*types.Ingress{
		{
			ID:   "ing-1",
			Name: "api-ingress",
			Rules: []*types.IngressRule{
				{
					Host: "api.example.com",
					Paths: []*types.IngressPath{
						{
							Path:     "/v1",
							PathType: types.PathTypePrefix,
							Backend: &types.IngressBackend{
								ServiceName: "api-v1",
								Port:        8080,
							},
						},
						{
							Path:     "/v2",
							PathType: types.PathTypePrefix,
							Backend: &types.IngressBackend{
								ServiceName: "api-v2",
								Port:        8080,
							},
						},
					},
				},
			},
		},
		{
			ID:   "ing-2",
			Name: "web-ingress",
			Rules: []*types.IngressRule{
				{
					Host: "example.com",
					Paths: []*types.IngressPath{
						{
							Path:     "/",
							PathType: types.PathTypePrefix,
							Backend: &types.IngressBackend{
								ServiceName: "web",
								Port:        80,
							},
						},
					},
				},
			},
		},
	}

	router := NewRouter(ingresses)

	tests := []struct {
		name            string
		host            string
		path            string
		wantServiceName string
		wantPort        int
		wantNil         bool
	}{
		{
			name:            "route to api-v1",
			host:            "api.example.com",
			path:            "/v1/users",
			wantServiceName: "api-v1",
			wantPort:        8080,
			wantNil:         false,
		},
		{
			name:            "route to api-v2",
			host:            "api.example.com",
			path:            "/v2/posts",
			wantServiceName: "api-v2",
			wantPort:        8080,
			wantNil:         false,
		},
		{
			name:            "route to web",
			host:            "example.com",
			path:            "/",
			wantServiceName: "web",
			wantPort:        80,
			wantNil:         false,
		},
		{
			name:    "no match - wrong host",
			host:    "other.com",
			path:    "/",
			wantNil: true,
		},
		{
			name:    "no match - wrong path",
			host:    "api.example.com",
			path:    "/v3/data",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := router.Route(tt.host, tt.path)

			if tt.wantNil {
				if result != nil {
					t.Errorf("Route(%q, %q) expected nil, got %+v", tt.host, tt.path, result)
				}
				return
			}

			if result == nil {
				t.Fatalf("Route(%q, %q) returned nil, want match", tt.host, tt.path)
			}

			if result.Backend.ServiceName != tt.wantServiceName {
				t.Errorf("Route() ServiceName = %q, want %q", result.Backend.ServiceName, tt.wantServiceName)
			}

			if result.Backend.Port != tt.wantPort {
				t.Errorf("Route() Port = %d, want %d", result.Backend.Port, tt.wantPort)
			}
		})
	}
}

// TestRouterLongestPrefixMatch tests that longest prefix wins
func TestRouterLongestPrefixMatch(t *testing.T) {
	ingresses := []*types.Ingress{
		{
			ID:   "ing-test",
			Name: "test-ingress",
			Rules: []*types.IngressRule{
				{
					Host: "example.com",
					Paths: []*types.IngressPath{
						{
							Path:     "/",
							PathType: types.PathTypePrefix,
							Backend: &types.IngressBackend{
								ServiceName: "root",
								Port:        80,
							},
						},
						{
							Path:     "/api",
							PathType: types.PathTypePrefix,
							Backend: &types.IngressBackend{
								ServiceName: "api",
								Port:        8080,
							},
						},
						{
							Path:     "/api/admin",
							PathType: types.PathTypePrefix,
							Backend: &types.IngressBackend{
								ServiceName: "admin",
								Port:        9090,
							},
						},
					},
				},
			},
		},
	}

	router := NewRouter(ingresses)

	tests := []struct {
		name            string
		path            string
		wantServiceName string
	}{
		{
			name:            "match root",
			path:            "/home",
			wantServiceName: "root",
		},
		{
			name:            "match /api",
			path:            "/api/users",
			wantServiceName: "api",
		},
		{
			name:            "match /api/admin (longest)",
			path:            "/api/admin/settings",
			wantServiceName: "admin",
		},
		{
			name:            "match /api/admin exactly",
			path:            "/api/admin",
			wantServiceName: "admin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := router.Route("example.com", tt.path)

			if result == nil {
				t.Fatalf("Route() returned nil for path %q", tt.path)
			}

			if result.Backend.ServiceName != tt.wantServiceName {
				t.Errorf("Route() ServiceName = %q, want %q", result.Backend.ServiceName, tt.wantServiceName)
			}
		})
	}
}

// TestRouterEmptyIngresses tests router with no ingresses
func TestRouterEmptyIngresses(t *testing.T) {
	router := NewRouter([]*types.Ingress{})

	result := router.Route("any-host.com", "/any-path")
	if result != nil {
		t.Errorf("Route() with empty ingresses should return nil, got %+v", result)
	}
}

// TestRouterWildcardHost tests wildcard host matching
func TestRouterWildcardHost(t *testing.T) {
	ingresses := []*types.Ingress{
		{
			ID:   "ing-wildcard",
			Name: "wildcard-ingress",
			Rules: []*types.IngressRule{
				{
					Host: "*.apps.example.com",
					Paths: []*types.IngressPath{
						{
							Path:     "/",
							PathType: types.PathTypePrefix,
							Backend: &types.IngressBackend{
								ServiceName: "app-proxy",
								Port:        8080,
							},
						},
					},
				},
			},
		},
	}

	router := NewRouter(ingresses)

	tests := []struct {
		name    string
		host    string
		wantNil bool
	}{
		{
			name:    "match subdomain",
			host:    "myapp.apps.example.com",
			wantNil: false,
		},
		{
			name:    "match another subdomain",
			host:    "test.apps.example.com",
			wantNil: false,
		},
		{
			name:    "no match root domain",
			host:    "apps.example.com",
			wantNil: true,
		},
		{
			name:    "no match different domain",
			host:    "example.com",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := router.Route(tt.host, "/")

			if tt.wantNil && result != nil {
				t.Errorf("Route(%q) expected nil, got %+v", tt.host, result)
			}

			if !tt.wantNil && result == nil {
				t.Errorf("Route(%q) expected match, got nil", tt.host)
			}
		})
	}
}
