package ingress

import (
	"context"
	"fmt"
	"sync"

	"github.com/cuemby/warren/pkg/log"
	"github.com/cuemby/warren/pkg/types"
	"google.golang.org/grpc"
)

// LoadBalancer handles backend selection and load balancing
type LoadBalancer struct {
	managerAddr string
	grpcClient  *grpc.ClientConn

	// Round-robin state
	mu      sync.Mutex
	indexes map[string]int // service name -> current index
}

// NewLoadBalancer creates a new load balancer
func NewLoadBalancer(managerAddr string, grpcClient *grpc.ClientConn) *LoadBalancer {
	return &LoadBalancer{
		managerAddr: managerAddr,
		grpcClient:  grpcClient,
		indexes:     make(map[string]int),
	}
}

// Backend represents a backend endpoint
type Backend struct {
	ServiceName string
	IP          string
	Port        int
	Healthy     bool
}

// SelectBackend selects a backend for the given service
// Returns the backend IP:port or error
func (lb *LoadBalancer) SelectBackend(ctx context.Context, serviceName string, port int) (string, error) {
	// Get all containers for the service
	containers, err := lb.getServiceContainers(ctx, serviceName)
	if err != nil {
		return "", fmt.Errorf("failed to get service containers: %w", err)
	}

	// M7.1 MVP: If no containers found (because getServiceContainers returns empty list),
	// use localhost fallback for basic testing
	if len(containers) == 0 {
		log.Debug(fmt.Sprintf("No containers found for service %s, using localhost fallback", serviceName))
		return fmt.Sprintf("127.0.0.1:%d", port), nil
	}

	// Filter healthy containers
	healthyContainers := make([]*types.Container, 0)
	for _, container := range containers {
		// Only include running and healthy containers
		if container.ActualState == "running" {
			// If health check is configured, check health status
			if container.HealthCheck != nil {
				if container.HealthStatus != nil && container.HealthStatus.Healthy {
					healthyContainers = append(healthyContainers, container)
				} else if container.HealthStatus == nil {
					// Health check configured but not yet checked, include container
					healthyContainers = append(healthyContainers, container)
				}
			} else {
				// No health check, consider running containers as healthy
				healthyContainers = append(healthyContainers, container)
			}
		}
	}

	if len(healthyContainers) == 0 {
		return "", fmt.Errorf("no healthy containers found for service: %s", serviceName)
	}

	// Round-robin selection
	lb.mu.Lock()
	index := lb.indexes[serviceName] % len(healthyContainers)
	lb.indexes[serviceName] = (index + 1) % len(healthyContainers)
	lb.mu.Unlock()

	selectedContainer := healthyContainers[index]

	// Get container IP from node
	// For now, use the node address (this works for host network mode)
	// TODO: In Phase 7.3, support overlay networking with container IPs
	node, err := lb.getNode(ctx, selectedContainer.NodeID)
	if err != nil {
		log.Warn(fmt.Sprintf("Failed to get node %s: %v, using localhost", selectedContainer.NodeID, err))
		// Fallback to localhost for development
		return fmt.Sprintf("127.0.0.1:%d", port), nil
	}

	// Return node IP with the service port
	return fmt.Sprintf("%s:%d", node.Address, port), nil
}

// getServiceContainers queries the manager API for service containers
func (lb *LoadBalancer) getServiceContainers(ctx context.Context, serviceName string) ([]*types.Container, error) {
	log.Debug(fmt.Sprintf("LoadBalancer: Getting containers for service %s", serviceName))

	// TODO (M7 Phase 7.2): Implement full gRPC query to manager
	// For M7 Phase 7.1 MVP, return empty list to allow basic functionality
	// The proxy will fall back to localhost:port for development testing

	return []*types.Container{}, nil
}

// getNode queries the manager API for node information
func (lb *LoadBalancer) getNode(ctx context.Context, nodeID string) (*types.Node, error) {
	log.Debug(fmt.Sprintf("LoadBalancer: Getting node %s", nodeID))

	// TODO (M7 Phase 7.2): Implement full gRPC query to manager
	// For M7 Phase 7.1 MVP, return error to trigger localhost fallback
	// This allows basic testing without full cluster integration

	return nil, fmt.Errorf("using localhost fallback for M7 Phase 7.1")
}
