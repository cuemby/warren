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
	// Get all tasks for the service
	tasks, err := lb.getServiceTasks(ctx, serviceName)
	if err != nil {
		return "", fmt.Errorf("failed to get service tasks: %v", err)
	}

	if len(tasks) == 0 {
		return "", fmt.Errorf("no tasks found for service: %s", serviceName)
	}

	// Filter healthy tasks
	healthyTasks := make([]*types.Task, 0)
	for _, task := range tasks {
		// Only include running and healthy tasks
		if task.ActualState == "running" {
			// If health check is configured, check health status
			if task.HealthCheck != nil {
				if task.HealthStatus == "healthy" || task.HealthStatus == "" {
					healthyTasks = append(healthyTasks, task)
				}
			} else {
				// No health check, consider running tasks as healthy
				healthyTasks = append(healthyTasks, task)
			}
		}
	}

	if len(healthyTasks) == 0 {
		return "", fmt.Errorf("no healthy tasks found for service: %s", serviceName)
	}

	// Round-robin selection
	lb.mu.Lock()
	index := lb.indexes[serviceName] % len(healthyTasks)
	lb.indexes[serviceName] = (index + 1) % len(healthyTasks)
	lb.mu.Unlock()

	selectedTask := healthyTasks[index]

	// Get task IP from node
	// For now, use the node address (this works for host network mode)
	// TODO: In Phase 7.3, support overlay networking with container IPs
	node, err := lb.getNode(ctx, selectedTask.NodeID)
	if err != nil {
		log.Warnf("Failed to get node %s: %v, using localhost", selectedTask.NodeID, err)
		// Fallback to localhost for development
		return fmt.Sprintf("127.0.0.1:%d", port), nil
	}

	// Return node IP with the service port
	return fmt.Sprintf("%s:%d", node.Address, port), nil
}

// getServiceTasks queries the manager API for service tasks
func (lb *LoadBalancer) getServiceTasks(ctx context.Context, serviceName string) ([]*types.Task, error) {
	// Query manager gRPC API
	// For now, we'll implement a simple version
	// TODO: Implement proper gRPC client call

	// This is a placeholder - we'll need to import the proto package
	// and make a proper ListTasks call filtered by service

	log.Debugf("LoadBalancer: Getting tasks for service %s", serviceName)

	// Return empty for now - this will be implemented when we integrate with manager
	return nil, fmt.Errorf("not yet implemented: getServiceTasks")
}

// getNode queries the manager API for node information
func (lb *LoadBalancer) getNode(ctx context.Context, nodeID string) (*types.Node, error) {
	// Query manager gRPC API
	// TODO: Implement proper gRPC client call

	log.Debugf("LoadBalancer: Getting node %s", nodeID)

	// Return nil for now - this will be implemented when we integrate with manager
	return nil, fmt.Errorf("not yet implemented: getNode")
}
