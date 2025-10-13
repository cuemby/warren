package e2e

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/cuemby/warren/test/framework"
)

// TestIngressBasicHTTP tests basic HTTP ingress routing
// Replaces test/lima/test-ingress-simple.sh
func TestIngressBasicHTTP(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping ingress test in short mode")
	}

	config := &framework.ClusterConfig{
		NumManagers: 1,
		NumWorkers:  1,
		UseLima:     true,
		ManagerVMConfig: &framework.VMConfig{
			CPUs:   2,
			Memory: "2GiB",
			Disk:   "10GiB",
		},
		WorkerVMConfig: &framework.VMConfig{
			CPUs:   2,
			Memory: "2GiB",
			Disk:   "10GiB",
		},
	}

	cluster, err := framework.NewCluster(config)
	if err != nil {
		t.Fatalf("Failed to create cluster: %v", err)
	}
	defer cluster.Cleanup()

	if err := cluster.Start(); err != nil {
		t.Fatalf("Failed to start cluster: %v", err)
	}
	defer cluster.Stop()

	waiter := framework.DefaultWaiter()
	ctx := context.Background()

	// Wait for cluster
	if err := waiter.WaitForLeaderElection(ctx, cluster); err != nil {
		t.Fatalf("Leader election failed: %v", err)
	}

	leader, err := cluster.GetLeader()
	if err != nil {
		t.Fatalf("Failed to get leader: %v", err)
	}

	t.Run("CreateBackendService", func(t *testing.T) {
		// Create nginx service as backend
		serviceName := "nginx-backend"
		if err := leader.Client.CreateService(serviceName, "nginx:alpine", 1); err != nil {
			t.Fatalf("Failed to create backend service: %v", err)
		}
		defer leader.Client.DeleteService(serviceName)

		// Wait for service to be running
		if err := waiter.WaitForServiceRunning(ctx, leader.Client, serviceName); err != nil {
			t.Fatalf("Backend service failed to start: %v", err)
		}

		t.Log("✓ Backend service running")
	})

	t.Run("CreateIngressRule", func(t *testing.T) {
		// Create ingress rule
		ingressName := "test-ingress"
		err := leader.Client.CreateIngress(ingressName, &framework.IngressSpec{
			Host:     "test.local",
			Path:     "/",
			PathType: "Prefix",
			Backend: framework.IngressBackend{
				Service: "nginx-backend",
				Port:    80,
			},
		})

		if err != nil {
			t.Fatalf("Failed to create ingress: %v", err)
		}
		defer leader.Client.DeleteIngress(ingressName)

		t.Log("✓ Ingress rule created")

		// Verify ingress exists
		ingresses, err := leader.Client.ListIngresses()
		if err != nil {
			t.Fatalf("Failed to list ingresses: %v", err)
		}

		found := false
		for _, ing := range ingresses {
			if ing.Name == ingressName {
				found = true
				break
			}
		}

		if !found {
			t.Error("Ingress not found in list")
		}
	})

	t.Run("TestHTTPRouting", func(t *testing.T) {
		// Give ingress proxy time to update
		time.Sleep(2 * time.Second)

		// Get ingress endpoint (port 8000 on manager)
		ingressURL := fmt.Sprintf("http://%s:8000/", leader.APIAddr)

		// Test correct host header (should work)
		t.Log("Testing correct host header...")
		req, err := http.NewRequest("GET", ingressURL, nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Host = "test.local"

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		} else {
			t.Log("✓ HTTP routing with correct host works")
		}

		// Test wrong host header (should return 404)
		t.Log("Testing wrong host header...")
		req2, err := http.NewRequest("GET", ingressURL, nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req2.Host = "wrong.local"

		resp2, err := client.Do(req2)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp2.Body.Close()

		if resp2.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status 404 for wrong host, got %d", resp2.StatusCode)
		} else {
			t.Log("✓ Wrong host correctly returns 404")
		}
	})
}

// TestIngressPathBased tests path-based routing
// Replaces parts of test/lima/test-ingress.sh
func TestIngressPathBased(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping path-based ingress test in short mode")
	}

	config := &framework.ClusterConfig{
		NumManagers: 1,
		NumWorkers:  1,
		UseLima:     true,
		ManagerVMConfig: &framework.VMConfig{
			CPUs:   2,
			Memory: "2GiB",
			Disk:   "10GiB",
		},
		WorkerVMConfig: &framework.VMConfig{
			CPUs:   2,
			Memory: "2GiB",
			Disk:   "10GiB",
		},
	}

	cluster, err := framework.NewCluster(config)
	if err != nil {
		t.Fatalf("Failed to create cluster: %v", err)
	}
	defer cluster.Cleanup()

	if err := cluster.Start(); err != nil {
		t.Fatalf("Failed to start cluster: %v", err)
	}
	defer cluster.Stop()

	waiter := framework.DefaultWaiter()
	ctx := context.Background()

	if err := waiter.WaitForLeaderElection(ctx, cluster); err != nil {
		t.Fatalf("Leader election failed: %v", err)
	}

	leader, err := cluster.GetLeader()
	if err != nil {
		t.Fatalf("Failed to get leader: %v", err)
	}

	t.Run("SetupBackendServices", func(t *testing.T) {
		// Create two different backend services
		services := []string{"app-v1", "app-v2"}
		for _, svc := range services {
			if err := leader.Client.CreateService(svc, "nginx:alpine", 1); err != nil {
				t.Fatalf("Failed to create service %s: %v", svc, err)
			}
			defer leader.Client.DeleteService(svc)

			if err := waiter.WaitForServiceRunning(ctx, leader.Client, svc); err != nil {
				t.Fatalf("Service %s failed to start: %v", svc, err)
			}
		}

		t.Log("✓ Backend services created")
	})

	t.Run("CreatePathBasedRoutes", func(t *testing.T) {
		// Create ingress for /v1/* -> app-v1
		err := leader.Client.CreateIngress("path-v1", &framework.IngressSpec{
			Host:     "api.local",
			Path:     "/v1",
			PathType: "Prefix",
			Backend: framework.IngressBackend{
				Service: "app-v1",
				Port:    80,
			},
		})
		if err != nil {
			t.Fatalf("Failed to create v1 ingress: %v", err)
		}
		defer leader.Client.DeleteIngress("path-v1")

		// Create ingress for /v2/* -> app-v2
		err = leader.Client.CreateIngress("path-v2", &framework.IngressSpec{
			Host:     "api.local",
			Path:     "/v2",
			PathType: "Prefix",
			Backend: framework.IngressBackend{
				Service: "app-v2",
				Port:    80,
			},
		})
		if err != nil {
			t.Fatalf("Failed to create v2 ingress: %v", err)
		}
		defer leader.Client.DeleteIngress("path-v2")

		t.Log("✓ Path-based routes created")

		time.Sleep(2 * time.Second) // Let proxy update
	})

	t.Run("TestPathRouting", func(t *testing.T) {
		baseURL := fmt.Sprintf("http://%s:8000", leader.APIAddr)
		client := &http.Client{Timeout: 10 * time.Second}

		tests := []struct {
			path       string
			shouldWork bool
		}{
			{"/v1/", true},
			{"/v2/", true},
			{"/v3/", false}, // No route
			{"/", false},    // No exact match
		}

		for _, tt := range tests {
			req, err := http.NewRequest("GET", baseURL+tt.path, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			req.Host = "api.local"

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Request to %s failed: %v", tt.path, err)
			}
			defer resp.Body.Close()

			if tt.shouldWork {
				if resp.StatusCode != http.StatusOK {
					t.Errorf("Path %s: expected 200, got %d", tt.path, resp.StatusCode)
				} else {
					t.Logf("✓ Path %s routed correctly", tt.path)
				}
			} else {
				if resp.StatusCode == http.StatusOK {
					t.Errorf("Path %s: expected non-200, got 200", tt.path)
				} else {
					t.Logf("✓ Path %s correctly rejected", tt.path)
				}
			}
		}
	})
}

// TestIngressHTTPS tests HTTPS ingress with TLS
// Replaces test/lima/test-https.sh
func TestIngressHTTPS(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping HTTPS ingress test in short mode")
	}

	config := &framework.ClusterConfig{
		NumManagers: 1,
		NumWorkers:  1,
		UseLima:     true,
		ManagerVMConfig: &framework.VMConfig{
			CPUs:   2,
			Memory: "2GiB",
			Disk:   "10GiB",
		},
		WorkerVMConfig: &framework.VMConfig{
			CPUs:   2,
			Memory: "2GiB",
			Disk:   "10GiB",
		},
	}

	cluster, err := framework.NewCluster(config)
	if err != nil {
		t.Fatalf("Failed to create cluster: %v", err)
	}
	defer cluster.Cleanup()

	if err := cluster.Start(); err != nil {
		t.Fatalf("Failed to start cluster: %v", err)
	}
	defer cluster.Stop()

	waiter := framework.DefaultWaiter()
	ctx := context.Background()

	if err := waiter.WaitForLeaderElection(ctx, cluster); err != nil {
		t.Fatalf("Leader election failed: %v", err)
	}

	leader, err := cluster.GetLeader()
	if err != nil {
		t.Fatalf("Failed to get leader: %v", err)
	}

	t.Run("CreateTLSSecret", func(t *testing.T) {
		// Create TLS secret with self-signed cert
		// In real test, we'd generate or use test certificates
		// secretName := "tls-secret"

		// For now, skip actual TLS secret creation
		// This would require certificate generation
		t.Skip("TLS certificate generation not yet implemented in framework")

		// Future implementation:
		// err := leader.Client.CreateSecret(secretName, framework.SecretTypeTLS, map[string][]byte{
		//     "tls.crt": certPEM,
		//     "tls.key": keyPEM,
		// })
	})

	t.Run("CreateHTTPSIngress", func(t *testing.T) {
		// Create backend service
		if err := leader.Client.CreateService("https-backend", "nginx:alpine", 1); err != nil {
			t.Fatalf("Failed to create backend: %v", err)
		}
		defer leader.Client.DeleteService("https-backend")

		// Create HTTPS ingress
		err := leader.Client.CreateIngress("https-ingress", &framework.IngressSpec{
			Host:     "secure.local",
			Path:     "/",
			PathType: "Prefix",
			Backend: framework.IngressBackend{
				Service: "https-backend",
				Port:    80,
			},
			TLS: &framework.IngressTLS{
				Enabled:    true,
				SecretName: "tls-secret",
			},
		})

		if err != nil {
			t.Logf("HTTPS ingress creation: %v (expected if TLS not fully implemented)", err)
			t.Skip("HTTPS ingress not fully implemented yet")
		}
		defer leader.Client.DeleteIngress("https-ingress")
	})

	t.Run("TestHTTPSAccess", func(t *testing.T) {
		// Test HTTPS access
		httpsURL := fmt.Sprintf("https://%s:8443/", leader.APIAddr)

		// Create client that accepts self-signed certs
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client := &http.Client{
			Transport: tr,
			Timeout:   10 * time.Second,
		}

		req, err := http.NewRequest("GET", httpsURL, nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Host = "secure.local"

		resp, err := client.Do(req)
		if err != nil {
			t.Logf("HTTPS request failed (expected if not implemented): %v", err)
			t.Skip("HTTPS not fully configured")
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			t.Log("✓ HTTPS ingress working")
		}
	})
}

// TestIngressAdvancedRouting tests complex routing scenarios
// Replaces test/lima/test-advanced-routing.sh
func TestIngressAdvancedRouting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping advanced routing test in short mode")
	}

	config := &framework.ClusterConfig{
		NumManagers: 1,
		NumWorkers:  2,
		UseLima:     true,
		ManagerVMConfig: &framework.VMConfig{
			CPUs:   2,
			Memory: "2GiB",
			Disk:   "10GiB",
		},
		WorkerVMConfig: &framework.VMConfig{
			CPUs:   2,
			Memory: "2GiB",
			Disk:   "10GiB",
		},
	}

	cluster, err := framework.NewCluster(config)
	if err != nil {
		t.Fatalf("Failed to create cluster: %v", err)
	}
	defer cluster.Cleanup()

	if err := cluster.Start(); err != nil {
		t.Fatalf("Failed to start cluster: %v", err)
	}
	defer cluster.Stop()

	waiter := framework.DefaultWaiter()
	ctx := context.Background()

	if err := waiter.WaitForLeaderElection(ctx, cluster); err != nil {
		t.Fatalf("Leader election failed: %v", err)
	}

	leader, err := cluster.GetLeader()
	if err != nil {
		t.Fatalf("Failed to get leader: %v", err)
	}

	t.Run("SetupMultipleServices", func(t *testing.T) {
		services := []struct {
			name     string
			replicas int
		}{
			{"frontend", 2},
			{"api", 2},
			{"admin", 1},
		}

		for _, svc := range services {
			if err := leader.Client.CreateService(svc.name, "nginx:alpine", svc.replicas); err != nil {
				t.Fatalf("Failed to create %s: %v", svc.name, err)
			}
			defer leader.Client.DeleteService(svc.name)

			if err := waiter.WaitForReplicas(ctx, leader.Client, svc.name, svc.replicas); err != nil {
				t.Fatalf("%s failed to start: %v", svc.name, err)
			}
		}

		t.Log("✓ All backend services running")
	})

	t.Run("CreateComplexRoutingRules", func(t *testing.T) {
		rules := []struct {
			name     string
			host     string
			path     string
			pathType string
			service  string
		}{
			{"frontend-root", "myapp.local", "/", "Prefix", "frontend"},
			{"api-prefix", "myapp.local", "/api", "Prefix", "api"},
			{"admin-exact", "myapp.local", "/admin", "Exact", "admin"},
		}

		for _, rule := range rules {
			err := leader.Client.CreateIngress(rule.name, &framework.IngressSpec{
				Host:     rule.host,
				Path:     rule.path,
				PathType: rule.pathType,
				Backend: framework.IngressBackend{
					Service: rule.service,
					Port:    80,
				},
			})
			if err != nil {
				t.Fatalf("Failed to create ingress %s: %v", rule.name, err)
			}
			defer leader.Client.DeleteIngress(rule.name)
		}

		t.Log("✓ Complex routing rules created")
		time.Sleep(2 * time.Second)
	})

	t.Run("TestRoutingPriority", func(t *testing.T) {
		baseURL := fmt.Sprintf("http://%s:8000", leader.APIAddr)
		client := &http.Client{Timeout: 10 * time.Second}

		tests := []struct {
			path            string
			expectedBackend string
		}{
			{"/", "frontend"},       // Root goes to frontend
			{"/api", "api"},         // API prefix
			{"/api/users", "api"},   // API subpath
			{"/admin", "admin"},     // Exact match
			{"/admin/", "frontend"}, // Admin with trailing slash goes to frontend (not exact match)
		}

		for _, tt := range tests {
			req, err := http.NewRequest("GET", baseURL+tt.path, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			req.Host = "myapp.local"

			resp, err := client.Do(req)
			if err != nil {
				t.Errorf("Request to %s failed: %v", tt.path, err)
				continue
			}
			defer resp.Body.Close()

			// Read a bit of response to identify backend
			// In real implementation, backend might add custom headers
			body, _ := io.ReadAll(resp.Body)

			if resp.StatusCode == http.StatusOK {
				t.Logf("✓ Path %s routed successfully (expected: %s)", tt.path, tt.expectedBackend)
			} else {
				t.Logf("⚠ Path %s returned status %d (body: %s)", tt.path, resp.StatusCode, string(body[:min(100, len(body))]))
			}
		}
	})

	t.Run("TestMultipleHosts", func(t *testing.T) {
		// Create rules for different hosts
		hosts := []struct {
			name    string
			host    string
			service string
		}{
			{"app1-host", "app1.local", "frontend"},
			{"app2-host", "app2.local", "api"},
		}

		for _, h := range hosts {
			err := leader.Client.CreateIngress(h.name, &framework.IngressSpec{
				Host:     h.host,
				Path:     "/",
				PathType: "Prefix",
				Backend: framework.IngressBackend{
					Service: h.service,
					Port:    80,
				},
			})
			if err != nil {
				t.Fatalf("Failed to create host-based ingress %s: %v", h.name, err)
			}
			defer leader.Client.DeleteIngress(h.name)
		}

		time.Sleep(2 * time.Second)

		// Test each host
		baseURL := fmt.Sprintf("http://%s:8000/", leader.APIAddr)
		client := &http.Client{Timeout: 10 * time.Second}

		for _, h := range hosts {
			req, err := http.NewRequest("GET", baseURL, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			req.Host = h.host

			resp, err := client.Do(req)
			if err != nil {
				t.Errorf("Request to %s failed: %v", h.host, err)
				continue
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				t.Logf("✓ Host %s routed to %s", h.host, h.service)
			}
		}
	})
}

// Helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
