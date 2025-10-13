package dns

import (
	"fmt"
	"math/rand"
	"net"
	"strings"
	"time"

	"github.com/cuemby/warren/pkg/log"
	"github.com/cuemby/warren/pkg/storage"
	"github.com/cuemby/warren/pkg/types"
	"github.com/miekg/dns"
)

// Resolver handles DNS resolution for Warren services and instances
type Resolver struct {
	store    storage.Store
	domain   string   // Search domain (e.g., "warren")
	upstream []string // Upstream DNS servers for external queries
	rnd      *rand.Rand
}

// NewResolver creates a new DNS resolver
func NewResolver(store storage.Store, domain string, upstream []string) *Resolver {
	return &Resolver{
		store:    store,
		domain:   domain,
		upstream: upstream,
		rnd:      rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Resolve resolves a DNS query name to DNS resource records
func (r *Resolver) Resolve(queryName string) ([]dns.RR, error) {
	// Normalize query name (remove trailing dot)
	name := strings.TrimSuffix(queryName, ".")

	log.Logger.Debug().
		Str("component", "dns.resolver").
		Str("query", name).
		Msg("resolving DNS query")

	// Try to resolve as service name
	if records, err := r.resolveService(name); err == nil {
		return records, nil
	}

	// Try to resolve as instance name (e.g., nginx-1, nginx-2)
	if record, err := r.resolveInstance(name); err == nil {
		return []dns.RR{record}, nil
	}

	// Not a Warren service or instance
	return nil, fmt.Errorf("query not resolvable by Warren DNS: %s", name)
}

// resolveService resolves a service name to A records for all healthy instances
// Supports:
//   - nginx
//   - nginx.warren
func (r *Resolver) resolveService(name string) ([]dns.RR, error) {
	// Strip domain suffix if present
	serviceName := r.stripDomain(name)

	// Get service by name
	service, err := r.store.GetServiceByName(serviceName)
	if err != nil {
		return nil, fmt.Errorf("service not found: %s", serviceName)
	}

	// Get all containers for this service
	containers, err := r.store.ListContainers()
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	// Filter containers for this service that are running and healthy
	var healthyIPs []net.IP
	for _, container := range containers {
		if container.ServiceID == service.ID &&
			container.ActualState == types.ContainerStateRunning &&
			(container.HealthStatus == nil || container.HealthStatus.Healthy) {

			// Get container IP (for now, we'll use a placeholder)
			// TODO: Real container IPs will come from containerd networking
			if ip := r.getContainerIP(container); ip != nil {
				healthyIPs = append(healthyIPs, ip)
			}
		}
	}

	if len(healthyIPs) == 0 {
		return nil, fmt.Errorf("no healthy instances for service: %s", serviceName)
	}

	log.Logger.Debug().
		Str("component", "dns.resolver").
		Str("service", serviceName).
		Int("instances", len(healthyIPs)).
		Msg("resolved service to instances")

	// Create A records for all healthy instances (round-robin via multiple records)
	var records []dns.RR
	fqdn := r.makeFQDN(name)

	// Shuffle IPs for round-robin load balancing
	r.shuffleIPs(healthyIPs)

	for _, ip := range healthyIPs {
		records = append(records, &dns.A{
			Hdr: dns.RR_Header{
				Name:   fqdn,
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    10, // Short TTL for dynamic services
			},
			A: ip,
		})
	}

	return records, nil
}

// resolveInstance resolves an instance-specific name to a single A record
// Supports:
//   - nginx-1 (first instance)
//   - nginx-2 (second instance)
//   - nginx-1.warren
func (r *Resolver) resolveInstance(name string) (*dns.A, error) {
	// Strip domain suffix if present
	name = r.stripDomain(name)

	// Parse instance name (e.g., "nginx-1" -> service="nginx", instance=1)
	serviceName, instanceNum, err := parseInstanceName(name)
	if err != nil {
		return nil, err
	}

	// Get service
	service, err := r.store.GetServiceByName(serviceName)
	if err != nil {
		return nil, fmt.Errorf("service not found: %s", serviceName)
	}

	// Get all tasks for this service
	tasks, err := r.store.ListContainers()
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	// Filter running tasks for this service and sort by creation time
	var serviceTasks []*types.Container
	for _, task := range tasks {
		if task.ServiceID == service.ID && task.ActualState == types.ContainerStateRunning {
			serviceTasks = append(serviceTasks, task)
		}
	}

	if len(serviceTasks) == 0 {
		return nil, fmt.Errorf("no running instances for service: %s", serviceName)
	}

	// Sort containers by creation time (oldest first = instance 1)
	// This gives us consistent instance numbering
	sortContainersByCreationTime(serviceTasks)

	// Check if instance number is valid (1-indexed)
	if instanceNum < 1 || instanceNum > len(serviceTasks) {
		return nil, fmt.Errorf("instance %d not found (service has %d instances)", instanceNum, len(serviceTasks))
	}

	// Get the specific instance (convert from 1-indexed to 0-indexed)
	task := serviceTasks[instanceNum-1]

	// Get task IP
	ip := r.getContainerIP(task)
	if ip == nil {
		return nil, fmt.Errorf("no IP for instance %s-%d", serviceName, instanceNum)
	}

	log.Logger.Debug().
		Str("component", "dns.resolver").
		Str("service", serviceName).
		Int("instance", instanceNum).
		Str("ip", ip.String()).
		Msg("resolved instance to IP")

	// Create A record
	fqdn := r.makeFQDN(name)
	return &dns.A{
		Hdr: dns.RR_Header{
			Name:   fqdn,
			Rrtype: dns.TypeA,
			Class:  dns.ClassINET,
			Ttl:    10,
		},
		A: ip,
	}, nil
}

// stripDomain removes the Warren domain suffix from a name
// nginx.warren -> nginx
// nginx -> nginx
func (r *Resolver) stripDomain(name string) string {
	suffix := "." + r.domain
	return strings.TrimSuffix(name, suffix)
}

// makeFQDN ensures a name ends with a dot (fully qualified)
func (r *Resolver) makeFQDN(name string) string {
	if !strings.HasSuffix(name, ".") {
		return name + "."
	}
	return name
}

// getContainerIP returns the IP address for a task
// TODO: In Phase C (VIP), this will return container IPs from containerd
// For now, we return a placeholder IP based on task ID hash
func (r *Resolver) getContainerIP(task *types.Container) net.IP {
	// Placeholder implementation
	// In a real implementation, we would:
	// 1. Query containerd for the container's network namespace
	// 2. Get the actual IP from the container's network interface
	// 3. Return that IP

	// For now, generate a consistent IP based on task ID
	// This is just for development/testing purposes
	hash := hashString(task.ID)
	return net.IPv4(10, 0, byte((hash>>8)&0xFF), byte(hash&0xFF))
}

// shuffleIPs randomly shuffles a slice of IPs (for round-robin)
func (r *Resolver) shuffleIPs(ips []net.IP) {
	r.rnd.Shuffle(len(ips), func(i, j int) {
		ips[i], ips[j] = ips[j], ips[i]
	})
}

// sortContainersByCreationTime sorts containers by creation time (oldest first)
func sortContainersByCreationTime(containers []*types.Container) {
	// Simple bubble sort by creation time
	for i := 0; i < len(containers)-1; i++ {
		for j := 0; j < len(containers)-i-1; j++ {
			if containers[j].CreatedAt.After(containers[j+1].CreatedAt) {
				containers[j], containers[j+1] = containers[j+1], containers[j]
			}
		}
	}
}

// hashString returns a simple hash of a string
func hashString(s string) uint32 {
	var hash uint32
	for i := 0; i < len(s); i++ {
		hash = hash*31 + uint32(s[i])
	}
	return hash
}
