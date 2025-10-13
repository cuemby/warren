/*
Package client provides a Go client library for the Warren gRPC API.

The client package wraps the Warren gRPC API with a convenient, idiomatic Go
interface. It handles connection management, mTLS authentication, certificate
requests, error handling, and provides type-safe methods for all cluster
operations.

# Architecture

The client provides a high-level interface to Warren's API:

	┌──────────────────── APPLICATION CODE ──────────────────────┐
	│                                                              │
	│  import "github.com/cuemby/warren/pkg/client"               │
	│                                                              │
	│  client, err := client.NewClient("manager:8080")            │
	│  svc, err := client.CreateService(...)                      │
	│                                                              │
	└──────────────────┬───────────────────────────────────────┘
	                   │
	┌──────────────────▼──── pkg/client ─────────────────────────┐
	│                                                              │
	│  ┌──────────────────────────────────────────────┐          │
	│  │           Client Wrapper                      │          │
	│  │  - High-level methods                         │          │
	│  │  - Error handling                             │          │
	│  │  - Type conversions                           │          │
	│  │  - Connection pooling                         │          │
	│  └──────────────────┬───────────────────────────┘          │
	│                     │                                        │
	│  ┌──────────────────▼───────────────────────────┐          │
	│  │         gRPC Client (mTLS)                    │          │
	│  │  - Certificate authentication                 │          │
	│  │  - TLS 1.3 encryption                         │          │
	│  │  - Protocol Buffer serialization              │          │
	│  └──────────────────┬───────────────────────────┘          │
	└─────────────────────┼────────────────────────────────────┘
	                      │ gRPC (port 8080)
	                      ▼
	              Manager API Server

# Core Features

Connection Management:
  - Automatic mTLS certificate handling
  - Connection pooling and reuse
  - Graceful connection shutdown
  - Exponential backoff on failures

Certificate Management:
  - Auto-request certificate with join token
  - Load existing certificate from disk
  - Verify certificate validity
  - Helpful error messages

Type Safety:
  - Strong typing for all operations
  - Go structs instead of proto messages
  - Compile-time safety
  - IDE autocomplete support

Error Handling:
  - Wrapped gRPC errors
  - Friendly error messages
  - Structured error types
  - Context for debugging

# Usage

Creating a Client (with existing certificate):

	import (
		"log"
		"github.com/cuemby/warren/pkg/client"
	)

	// Expects certificate at ~/.warren/cli/
	client, err := client.NewClient("192.168.1.10:8080")
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

Creating a Client (with join token):

	// Automatically requests and saves certificate
	client, err := client.NewClientWithToken(
		"192.168.1.10:8080",
		"worker-join-token-xyz789",
	)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

# Service Operations

Creating a Service:

	service, err := client.CreateService(
		"nginx",                    // name
		"nginx:latest",             // image
		3,                          // replicas
		map[string]string{          // environment
			"ENV": "production",
		},
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Service created: %s\n", service.Id)

Listing Services:

	services, err := client.ListServices()
	if err != nil {
		log.Fatal(err)
	}
	for _, svc := range services {
		fmt.Printf("- %s (%d replicas)\n", svc.Name, svc.Replicas)
	}

Getting a Service:

	service, err := client.GetService("nginx")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Service: %s\n", service.Name)
	fmt.Printf("Image: %s\n", service.Image)
	fmt.Printf("Replicas: %d\n", service.Replicas)

Updating a Service:

	// Triggers rolling update
	err := client.UpdateService("nginx", &UpdateServiceRequest{
		Replicas: proto.Int32(5),
		Image:    proto.String("nginx:1.25"),
	})
	if err != nil {
		log.Fatal(err)
	}

Scaling a Service:

	err := client.ScaleService("nginx", 10)
	if err != nil {
		log.Fatal(err)
	}

Deleting a Service:

	err := client.DeleteService("nginx")
	if err != nil {
		log.Fatal(err)
	}

# Task Operations

Listing Tasks:

	tasks, err := client.ListTasks()
	if err != nil {
		log.Fatal(err)
	}
	for _, task := range tasks {
		fmt.Printf("- %s: %s (%s)\n",
			task.Id, task.ServiceName, task.State)
	}

Getting a Task:

	task, err := client.GetTask("task-123")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Task %s on node %s: %s\n",
		task.Id, task.NodeId, task.State)

# Node Operations

Listing Nodes:

	nodes, err := client.ListNodes()
	if err != nil {
		log.Fatal(err)
	}
	for _, node := range nodes {
		fmt.Printf("- %s (%s): %s\n",
			node.Id, node.Role, node.Status)
	}

Getting a Node:

	node, err := client.GetNode("worker-1")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Node: %s\n", node.Id)
	fmt.Printf("Resources: %d CPUs, %d MB RAM\n",
		node.Resources.CpuCores,
		node.Resources.MemoryBytes/(1024*1024))

# Secret Operations

Creating a Secret:

	secret, err := client.CreateSecret(
		"db-password",
		[]byte("super-secret-password"),
	)
	if err != nil {
		log.Fatal(err)
	}

Listing Secrets:

	secrets, err := client.ListSecrets()
	if err != nil {
		log.Fatal(err)
	}
	for _, secret := range secrets {
		fmt.Printf("- %s (created: %s)\n",
			secret.Name, secret.CreatedAt)
	}

Deleting a Secret:

	err := client.DeleteSecret("db-password")
	if err != nil {
		log.Fatal(err)
	}

# Volume Operations

Creating a Volume:

	volume, err := client.CreateVolume(
		"db-data",
		"local",
		map[string]string{
			"path": "/var/lib/warren/volumes/db-data",
		},
	)
	if err != nil {
		log.Fatal(err)
	}

Listing Volumes:

	volumes, err := client.ListVolumes()
	if err != nil {
		log.Fatal(err)
	}
	for _, vol := range volumes {
		fmt.Printf("- %s (%s driver)\n", vol.Name, vol.Driver)
	}

Deleting a Volume:

	err := client.DeleteVolume("db-data")
	if err != nil {
		log.Fatal(err)
	}

# Ingress Operations

Creating an Ingress:

	ingress, err := client.CreateIngress(&IngressRequest{
		Name: "my-ingress",
		Rules: []*IngressRule{
			{
				Host: "api.example.com",
				Paths: []*IngressPath{
					{
						Path:        "/",
						PathType:    "Prefix",
						ServiceName: "api-service",
						ServicePort: 80,
					},
				},
			},
		},
		TLS: &IngressTLS{
			Domains:   []string{"api.example.com"},
			AutoIssue: true,
			ACMEEmail: "admin@example.com",
		},
	})
	if err != nil {
		log.Fatal(err)
	}

Listing Ingresses:

	ingresses, err := client.ListIngresses()
	if err != nil {
		log.Fatal(err)
	}
	for _, ing := range ingresses {
		fmt.Printf("- %s: %d rules\n", ing.Name, len(ing.Rules))
	}

# Certificate Operations

Creating a Certificate:

	cert, err := client.CreateCertificate(
		"api.example.com",
		certPEM,
		keyPEM,
	)
	if err != nil {
		log.Fatal(err)
	}

Listing Certificates:

	certs, err := client.ListCertificates()
	if err != nil {
		log.Fatal(err)
	}
	for _, cert := range certs {
		fmt.Printf("- %s (expires: %s)\n",
			cert.Domain, cert.ExpiresAt)
	}

# Cluster Operations

Getting Cluster Info:

	info, err := client.GetClusterInfo()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Cluster ID: %s\n", info.Id)
	fmt.Printf("Managers: %d\n", len(info.Managers))
	fmt.Printf("Workers: %d\n", len(info.Workers))

Generating Join Tokens:

	// Worker token
	workerToken, err := client.GenerateWorkerToken()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Worker token: %s\n", workerToken)

	// Manager token
	managerToken, err := client.GenerateManagerToken()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Manager token: %s\n", managerToken)

# Error Handling

The client provides structured error handling:

Common Errors:

	_, err := client.CreateService(...)
	if err != nil {
		switch {
		case strings.Contains(err.Error(), "not the leader"):
			// Retry on leader
			leaderAddr := extractLeaderAddr(err)
			newClient, _ := client.NewClient(leaderAddr)
			// Retry...

		case strings.Contains(err.Error(), "already exists"):
			// Service already exists, update instead
			client.UpdateService(...)

		case strings.Contains(err.Error(), "not found"):
			// Resource doesn't exist
			fmt.Println("Resource not found")

		default:
			// Unknown error
			log.Fatal(err)
		}
	}

Connection Errors:

	client, err := client.NewClient("192.168.1.10:8080")
	if err != nil {
		if strings.Contains(err.Error(), "certificate not found") {
			// Need to request certificate first
			fmt.Println("Please run 'warren init' first")
		} else if strings.Contains(err.Error(), "connection refused") {
			// Manager not running
			fmt.Println("Manager is not reachable")
		} else {
			log.Fatal(err)
		}
	}

# Performance Considerations

Connection Pooling:
  - Reuse client across multiple operations
  - Don't create new client per request
  - Connection overhead: ~10ms
  - Reuse overhead: ~0.1ms

Timeouts:
  - Default timeout: 10 seconds
  - Adjust for slow operations (list large datasets)
  - Use context.WithTimeout for custom timeouts

Batching:
  - Batch creates when possible
  - Use UpdateService for multiple changes
  - List operations return all items (paginate if needed)

# Integration Points

This package integrates with:

  - pkg/api: Consumes gRPC API
  - pkg/security: Uses mTLS certificates
  - api/proto: Protocol Buffer messages
  - pkg/types: Core data types (indirect)

# Design Patterns

Client Pattern:
  - Single client instance per connection
  - Wraps low-level gRPC client
  - Provides high-level convenience methods

Builder Pattern:
  - Request builders for complex operations
  - Fluent API for readability
  - Type-safe construction

Resource Pattern:
  - CRUD methods for each resource type
  - Consistent naming (Create, List, Get, Update, Delete)
  - Standard error handling

# Certificate Management

Certificate Locations:

	CLI certificates:    ~/.warren/cli/
	Worker certificates: /etc/warren/certs/worker-<id>/
	Manager certificates: /etc/warren/certs/manager-<id>/

Certificate Files:

	cert.pem  - Client certificate
	key.pem   - Private key
	ca.pem    - CA certificate

Certificate Request Flow:

	1. NewClientWithToken("manager:8080", "token")
	2. Check if certificate exists
	3. If not, call RequestCertificate gRPC
	4. Save certificate to disk
	5. Load certificate and connect with mTLS

# Thread Safety

The client is safe for concurrent use:

	client, _ := client.NewClient("manager:8080")

	// Goroutine 1
	go func() {
		services, _ := client.ListServices()
		// Use services...
	}()

	// Goroutine 2
	go func() {
		nodes, _ := client.ListNodes()
		// Use nodes...
	}()

gRPC connections are thread-safe by design, and the client wrapper
maintains no mutable state.

# Troubleshooting

Common Issues:

Certificate Not Found:
  - Error: "CLI certificate not found"
  - Solution: Use NewClientWithToken or run 'warren init'

Connection Refused:
  - Error: "connection refused"
  - Solution: Check manager is running and address is correct

Not the Leader:
  - Error: "not the leader, current leader is at: X"
  - Solution: Retry automatically or connect to leader

Timeout:
  - Error: "context deadline exceeded"
  - Solution: Increase timeout or check network/manager health

# Testing

The client can be mocked for testing:

	type MockClient struct {
		CreateServiceFunc func(...) (*proto.Service, error)
	}

	func (m *MockClient) CreateService(...) (*proto.Service, error) {
		return m.CreateServiceFunc(...)
	}

	// In tests:
	mock := &MockClient{
		CreateServiceFunc: func(...) (*proto.Service, error) {
			return &proto.Service{Id: "test-id"}, nil
		},
	}

# See Also

  - pkg/api for server-side implementation
  - api/proto for Protocol Buffer definitions
  - pkg/security for certificate management
  - cmd/warren for CLI usage examples
  - docs/api-reference.md for complete API documentation
*/
package client
