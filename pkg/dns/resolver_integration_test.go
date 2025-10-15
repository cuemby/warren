package dns

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/cuemby/warren/pkg/types"
	"github.com/miekg/dns"
)

// mockStore is a simple in-memory store for testing
type mockStore struct {
	services   map[string]*types.Service
	containers map[string]*types.Container
}

func newMockStore() *mockStore {
	return &mockStore{
		services:   make(map[string]*types.Service),
		containers: make(map[string]*types.Container),
	}
}

func (m *mockStore) GetService(id string) (*types.Service, error) {
	if svc, ok := m.services[id]; ok {
		return svc, nil
	}
	return nil, fmt.Errorf("service not found: %s", id)
}

func (m *mockStore) GetServiceByName(name string) (*types.Service, error) {
	for _, svc := range m.services {
		if svc.Name == name {
			return svc, nil
		}
	}
	return nil, fmt.Errorf("service not found: %s", name)
}

func (m *mockStore) ListServices() ([]*types.Service, error) {
	services := make([]*types.Service, 0, len(m.services))
	for _, svc := range m.services {
		services = append(services, svc)
	}
	return services, nil
}

func (m *mockStore) ListContainers() ([]*types.Container, error) {
	containers := make([]*types.Container, 0, len(m.containers))
	for _, container := range m.containers {
		containers = append(containers, container)
	}
	return containers, nil
}

// Stub methods for interface compliance
func (m *mockStore) GetContainer(id string) (*types.Container, error) { return nil, nil }
func (m *mockStore) CreateService(svc *types.Service) error           { return nil }
func (m *mockStore) UpdateService(svc *types.Service) error           { return nil }
func (m *mockStore) DeleteService(id string) error                    { return nil }
func (m *mockStore) CreateContainer(c *types.Container) error         { return nil }
func (m *mockStore) UpdateContainer(c *types.Container) error         { return nil }
func (m *mockStore) DeleteContainer(id string) error                  { return nil }
func (m *mockStore) ListContainersByService(id string) ([]*types.Container, error) {
	return nil, nil
}
func (m *mockStore) ListContainersByNode(id string) ([]*types.Container, error) {
	return nil, nil
}
func (m *mockStore) GetNode(id string) (*types.Node, error)               { return nil, nil }
func (m *mockStore) ListNodes() ([]*types.Node, error)                    { return nil, nil }
func (m *mockStore) CreateNode(n *types.Node) error                       { return nil }
func (m *mockStore) UpdateNode(n *types.Node) error                       { return nil }
func (m *mockStore) DeleteNode(id string) error                           { return nil }
func (m *mockStore) CreateSecret(s *types.Secret) error                   { return nil }
func (m *mockStore) GetSecret(id string) (*types.Secret, error)           { return nil, nil }
func (m *mockStore) GetSecretByName(name string) (*types.Secret, error)   { return nil, nil }
func (m *mockStore) ListSecrets() ([]*types.Secret, error)                { return nil, nil }
func (m *mockStore) DeleteSecret(id string) error                         { return nil }
func (m *mockStore) CreateVolume(v *types.Volume) error                   { return nil }
func (m *mockStore) GetVolume(id string) (*types.Volume, error)           { return nil, nil }
func (m *mockStore) GetVolumeByName(name string) (*types.Volume, error)   { return nil, nil }
func (m *mockStore) ListVolumes() ([]*types.Volume, error)                { return nil, nil }
func (m *mockStore) DeleteVolume(id string) error                         { return nil }
func (m *mockStore) CreateNetwork(n *types.Network) error                 { return nil }
func (m *mockStore) GetNetwork(id string) (*types.Network, error)         { return nil, nil }
func (m *mockStore) GetNetworkByName(name string) (*types.Network, error) { return nil, nil }
func (m *mockStore) ListNetworks() ([]*types.Network, error)              { return nil, nil }
func (m *mockStore) DeleteNetwork(id string) error                        { return nil }
func (m *mockStore) CreateIngress(i *types.Ingress) error                 { return nil }
func (m *mockStore) GetIngress(id string) (*types.Ingress, error)         { return nil, nil }
func (m *mockStore) GetIngressByName(name string) (*types.Ingress, error) { return nil, nil }
func (m *mockStore) ListIngresses() ([]*types.Ingress, error)             { return nil, nil }
func (m *mockStore) UpdateIngress(i *types.Ingress) error                 { return nil }
func (m *mockStore) DeleteIngress(id string) error                        { return nil }
func (m *mockStore) CreateTLSCertificate(c *types.TLSCertificate) error   { return nil }
func (m *mockStore) GetTLSCertificate(id string) (*types.TLSCertificate, error) {
	return nil, nil
}
func (m *mockStore) GetTLSCertificateByName(name string) (*types.TLSCertificate, error) {
	return nil, nil
}
func (m *mockStore) ListTLSCertificates() ([]*types.TLSCertificate, error) { return nil, nil }
func (m *mockStore) UpdateTLSCertificate(c *types.TLSCertificate) error    { return nil }
func (m *mockStore) DeleteTLSCertificate(id string) error                  { return nil }
func (m *mockStore) ListTLSCertificatesByHost(host string) ([]*types.TLSCertificate, error) {
	return nil, nil
}
func (m *mockStore) GetTLSCertificatesByHost(host string) ([]*types.TLSCertificate, error) {
	return nil, nil
}
func (m *mockStore) SaveCA(data []byte) error { return nil }
func (m *mockStore) GetCA() ([]byte, error)   { return nil, nil }
func (m *mockStore) Close() error             { return nil }

// TestResolverServiceResolutionWithMockStore tests service name resolution with mock data
func TestResolverServiceResolutionWithMockStore(t *testing.T) {
	store := newMockStore()
	r := NewResolver(store, "warren", []string{"8.8.8.8:53"})

	// Create test service
	service := &types.Service{
		ID:   "svc-1",
		Name: "nginx",
	}
	store.services[service.ID] = service

	// Create running containers for the service
	now := time.Now()
	containers := []*types.Container{
		{
			ID:          "container-1",
			ServiceID:   service.ID,
			ActualState: types.ContainerStateRunning,
			CreatedAt:   now,
		},
		{
			ID:          "container-2",
			ServiceID:   service.ID,
			ActualState: types.ContainerStateRunning,
			CreatedAt:   now.Add(1 * time.Second),
		},
		{
			ID:          "container-3",
			ServiceID:   service.ID,
			ActualState: types.ContainerStateRunning,
			CreatedAt:   now.Add(2 * time.Second),
		},
	}

	for _, container := range containers {
		store.containers[container.ID] = container
	}

	tests := []struct {
		name        string
		queryName   string
		wantRecords int
		wantErr     bool
	}{
		{
			name:        "service name without domain",
			queryName:   "nginx",
			wantRecords: 3,
			wantErr:     false,
		},
		{
			name:        "service name with domain",
			queryName:   "nginx.warren",
			wantRecords: 3,
			wantErr:     false,
		},
		{
			name:        "non-existent service",
			queryName:   "notfound",
			wantRecords: 0,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			records, err := r.Resolve(tt.queryName)

			if (err != nil) != tt.wantErr {
				t.Errorf("Resolve() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(records) != tt.wantRecords {
					t.Errorf("Resolve() got %d records, want %d", len(records), tt.wantRecords)
				}

				// Verify all records are A records
				for _, rr := range records {
					if _, ok := rr.(*dns.A); !ok {
						t.Errorf("Resolve() record is not A record: %T", rr)
					}
				}
			}
		})
	}
}

// TestResolverInstanceResolutionWithMockStore tests instance-specific resolution
func TestResolverInstanceResolutionWithMockStore(t *testing.T) {
	store := newMockStore()
	r := NewResolver(store, "warren", []string{"8.8.8.8:53"})

	// Create test service
	service := &types.Service{
		ID:   "svc-1",
		Name: "nginx",
	}
	store.services[service.ID] = service

	// Create 3 running containers with staggered creation times
	now := time.Now()
	containers := []*types.Container{
		{
			ID:          "container-1",
			ServiceID:   service.ID,
			ActualState: types.ContainerStateRunning,
			CreatedAt:   now, // This will be instance 1
		},
		{
			ID:          "container-2",
			ServiceID:   service.ID,
			ActualState: types.ContainerStateRunning,
			CreatedAt:   now.Add(1 * time.Second), // Instance 2
		},
		{
			ID:          "container-3",
			ServiceID:   service.ID,
			ActualState: types.ContainerStateRunning,
			CreatedAt:   now.Add(2 * time.Second), // Instance 3
		},
	}

	for _, container := range containers {
		store.containers[container.ID] = container
	}

	tests := []struct {
		name      string
		queryName string
		wantErr   bool
	}{
		{
			name:      "first instance",
			queryName: "nginx-1",
			wantErr:   false,
		},
		{
			name:      "second instance",
			queryName: "nginx-2",
			wantErr:   false,
		},
		{
			name:      "third instance",
			queryName: "nginx-3",
			wantErr:   false,
		},
		{
			name:      "instance with domain",
			queryName: "nginx-1.warren",
			wantErr:   false,
		},
		{
			name:      "non-existent instance number",
			queryName: "nginx-10",
			wantErr:   true,
		},
		{
			name:      "instance zero",
			queryName: "nginx-0",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			records, err := r.Resolve(tt.queryName)

			if (err != nil) != tt.wantErr {
				t.Errorf("Resolve() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(records) != 1 {
					t.Errorf("Resolve() got %d records, want 1", len(records))
				}

				// Verify it's an A record
				aRecord, ok := records[0].(*dns.A)
				if !ok {
					t.Errorf("Resolve() record is not A record: %T", records[0])
				}

				// Verify IP is not nil
				if aRecord.A == nil {
					t.Error("Resolve() A record has nil IP")
				}
			}
		})
	}
}

// TestResolverEdgeCases tests various edge cases
func TestResolverEdgeCases(t *testing.T) {
	store := newMockStore()
	r := NewResolver(store, "warren", []string{"8.8.8.8:53"})

	t.Run("empty query name", func(t *testing.T) {
		_, err := r.Resolve("")
		if err == nil {
			t.Error("Resolve() expected error for empty query, got nil")
		}
	})

	t.Run("service with no running containers", func(t *testing.T) {
		service := &types.Service{
			ID:   "svc-empty",
			Name: "empty-service",
		}
		store.services[service.ID] = service

		// Add stopped containers
		store.containers["c1"] = &types.Container{
			ID:          "c1",
			ServiceID:   service.ID,
			ActualState: types.ContainerStateShutdown,
		}

		_, err := r.Resolve("empty-service")
		if err == nil {
			t.Error("Resolve() expected error for service with no running containers")
		}
	})

	t.Run("service with unhealthy containers", func(t *testing.T) {
		service := &types.Service{
			ID:   "svc-unhealthy",
			Name: "unhealthy-service",
		}
		store.services[service.ID] = service

		// Add unhealthy container
		store.containers["c-unhealthy"] = &types.Container{
			ID:           "c-unhealthy",
			ServiceID:    service.ID,
			ActualState:  types.ContainerStateRunning,
			HealthStatus: &types.HealthStatus{Healthy: false},
		}

		_, err := r.Resolve("unhealthy-service")
		if err == nil {
			t.Error("Resolve() expected error for service with only unhealthy containers")
		}
	})

	t.Run("service with mixed healthy and unhealthy containers", func(t *testing.T) {
		service := &types.Service{
			ID:   "svc-mixed",
			Name: "mixed-service",
		}
		store.services[service.ID] = service

		// Add healthy container
		store.containers["c-healthy"] = &types.Container{
			ID:           "c-healthy",
			ServiceID:    service.ID,
			ActualState:  types.ContainerStateRunning,
			HealthStatus: &types.HealthStatus{Healthy: true},
			CreatedAt:    time.Now(),
		}

		// Add unhealthy container (should be filtered out)
		store.containers["c-unhealthy-2"] = &types.Container{
			ID:           "c-unhealthy-2",
			ServiceID:    service.ID,
			ActualState:  types.ContainerStateRunning,
			HealthStatus: &types.HealthStatus{Healthy: false},
			CreatedAt:    time.Now(),
		}

		records, err := r.Resolve("mixed-service")
		if err != nil {
			t.Errorf("Resolve() unexpected error: %v", err)
		}

		// Should only return healthy container
		if len(records) != 1 {
			t.Errorf("Resolve() got %d records, want 1 (only healthy)", len(records))
		}
	})

	t.Run("query with trailing dot", func(t *testing.T) {
		service := &types.Service{
			ID:   "svc-dot",
			Name: "dotted",
		}
		store.services[service.ID] = service

		store.containers["c-dot"] = &types.Container{
			ID:          "c-dot",
			ServiceID:   service.ID,
			ActualState: types.ContainerStateRunning,
			CreatedAt:   time.Now(),
		}

		_, err := r.Resolve("dotted.")
		if err != nil {
			t.Errorf("Resolve() should handle trailing dot: %v", err)
		}
	})
}

// TestResolverConcurrency tests resolver under concurrent access
func TestResolverConcurrency(t *testing.T) {
	store := newMockStore()
	r := NewResolver(store, "warren", []string{"8.8.8.8:53"})

	// Create test data
	service := &types.Service{
		ID:   "svc-concurrent",
		Name: "concurrent",
	}
	store.services[service.ID] = service

	for i := 0; i < 5; i++ {
		container := &types.Container{
			ID:          fmt.Sprintf("c-%d", i),
			ServiceID:   service.ID,
			ActualState: types.ContainerStateRunning,
			CreatedAt:   time.Now().Add(time.Duration(i) * time.Second),
		}
		store.containers[container.ID] = container
	}

	// Run concurrent queries
	const concurrency = 50
	done := make(chan bool, concurrency)
	errors := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(n int) {
			var queryName string
			if n%2 == 0 {
				queryName = "concurrent"
			} else {
				queryName = fmt.Sprintf("concurrent-%d", (n%5)+1)
			}

			_, err := r.Resolve(queryName)
			if err != nil {
				errors <- err
			}
			done <- true
		}(i)
	}

	// Wait for all to complete
	for i := 0; i < concurrency; i++ {
		<-done
	}
	close(errors)

	// Check for errors
	var errorCount int
	for err := range errors {
		t.Errorf("Concurrent query failed: %v", err)
		errorCount++
	}

	if errorCount > 0 {
		t.Errorf("Got %d errors in concurrent test", errorCount)
	}
}

// TestResolverIPConsistency tests that same container always gets same IP
func TestResolverIPConsistency(t *testing.T) {
	r := NewResolver(nil, "warren", nil)

	container := &types.Container{
		ID: "test-container-123",
	}

	// Get IP multiple times
	ips := make([]net.IP, 10)
	for i := 0; i < 10; i++ {
		ips[i] = r.getContainerIP(container)
	}

	// Verify all IPs are identical
	firstIP := ips[0]
	for i := 1; i < len(ips); i++ {
		if !ips[i].Equal(firstIP) {
			t.Errorf("IP not consistent: ips[0]=%v, ips[%d]=%v", firstIP, i, ips[i])
		}
	}
}

// TestResolverMultipleServicesIsolation tests that services are properly isolated
func TestResolverMultipleServicesIsolation(t *testing.T) {
	store := newMockStore()
	r := NewResolver(store, "warren", []string{"8.8.8.8:53"})

	// Create two services
	svc1 := &types.Service{ID: "svc-1", Name: "web"}
	svc2 := &types.Service{ID: "svc-2", Name: "api"}
	store.services[svc1.ID] = svc1
	store.services[svc2.ID] = svc2

	// Add containers for svc1
	store.containers["c1-1"] = &types.Container{
		ID:          "c1-1",
		ServiceID:   svc1.ID,
		ActualState: types.ContainerStateRunning,
		CreatedAt:   time.Now(),
	}

	// Add containers for svc2
	store.containers["c2-1"] = &types.Container{
		ID:          "c2-1",
		ServiceID:   svc2.ID,
		ActualState: types.ContainerStateRunning,
		CreatedAt:   time.Now(),
	}

	// Query web service - should only return web containers
	webRecords, err := r.Resolve("web")
	if err != nil {
		t.Fatalf("Resolve(web) error: %v", err)
	}
	if len(webRecords) != 1 {
		t.Errorf("Resolve(web) got %d records, want 1", len(webRecords))
	}

	// Query api service - should only return api containers
	apiRecords, err := r.Resolve("api")
	if err != nil {
		t.Fatalf("Resolve(api) error: %v", err)
	}
	if len(apiRecords) != 1 {
		t.Errorf("Resolve(api) got %d records, want 1", len(apiRecords))
	}

	// Verify IPs are different (services isolated)
	webIP := webRecords[0].(*dns.A).A
	apiIP := apiRecords[0].(*dns.A).A
	if webIP.Equal(apiIP) {
		t.Error("Different services should have different IPs")
	}
}
