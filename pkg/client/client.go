package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	"github.com/cuemby/warren/api/proto"
	"github.com/cuemby/warren/pkg/security"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// Client wraps the Warren gRPC client for easy CLI usage
type Client struct {
	conn   *grpc.ClientConn
	client proto.WarrenAPIClient
}

// NewClient creates a new Warren client with mTLS
func NewClient(addr string) (*Client, error) {
	// Check for CLI certificate
	certDir, err := security.GetCertDir("cli", "")
	if err != nil {
		return nil, fmt.Errorf("failed to get cert directory: %w", err)
	}

	var conn *grpc.ClientConn
	if security.CertExists(certDir) {
		// Use mTLS with existing certificate
		conn, err = connectWithMTLS(addr, certDir)
		if err != nil {
			return nil, fmt.Errorf("failed to connect with mTLS: %w", err)
		}
	} else {
		// For CLI, we don't auto-request certificates like workers do
		// CLI commands that need certificates should fail with helpful error
		return nil, fmt.Errorf("CLI certificate not found at %s. Please run 'warren init' to request a certificate from the manager", certDir)
	}

	return &Client{
		conn:   conn,
		client: proto.NewWarrenAPIClient(conn),
	}, nil
}

// NewClientWithToken creates a new Warren client and requests a certificate using a join token
func NewClientWithToken(addr, token string) (*Client, error) {
	certDir, err := security.GetCertDir("cli", "")
	if err != nil {
		return nil, fmt.Errorf("failed to get cert directory: %w", err)
	}

	// Request certificate if not exists
	if !security.CertExists(certDir) {
		fmt.Println("CLI certificate not found, requesting from manager...")
		if err := requestCertificate(addr, token, certDir); err != nil {
			return nil, fmt.Errorf("failed to request certificate: %w", err)
		}
		fmt.Printf("✓ Certificate obtained and saved to %s\n", certDir)
	} else {
		fmt.Printf("✓ Using existing certificate from %s\n", certDir)
	}

	// Connect with mTLS
	conn, err := connectWithMTLS(addr, certDir)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to manager: %w", err)
	}

	return &Client{
		conn:   conn,
		client: proto.NewWarrenAPIClient(conn),
	}, nil
}

// Close closes the client connection
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// CreateService creates a new service
func (c *Client) CreateService(name, image string, replicas int32, env map[string]string) (*proto.Service, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req := &proto.CreateServiceRequest{
		Name:     name,
		Image:    image,
		Replicas: replicas,
		Mode:     "replicated",
		Env:      env,
	}

	resp, err := c.client.CreateService(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp.Service, nil
}

// CreateServiceWithOptions creates a new service with full options including health checks
func (c *Client) CreateServiceWithOptions(req *proto.CreateServiceRequest) (*proto.Service, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := c.client.CreateService(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp.Service, nil
}

// ListServices lists all services
func (c *Client) ListServices() ([]*proto.Service, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := c.client.ListServices(ctx, &proto.ListServicesRequest{})
	if err != nil {
		return nil, err
	}

	return resp.Services, nil
}

// GetService gets a service by name or ID
func (c *Client) GetService(nameOrID string) (*proto.Service, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := c.client.GetService(ctx, &proto.GetServiceRequest{
		Name: nameOrID,
	})
	if err != nil {
		return nil, err
	}

	return resp.Service, nil
}

// UpdateService updates a service (scale replicas)
func (c *Client) UpdateService(id string, replicas int32) (*proto.Service, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := c.client.UpdateService(ctx, &proto.UpdateServiceRequest{
		Id:       id,
		Replicas: replicas,
	})
	if err != nil {
		return nil, err
	}

	return resp.Service, nil
}

// DeleteService deletes a service
func (c *Client) DeleteService(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := c.client.DeleteService(ctx, &proto.DeleteServiceRequest{
		Id: id,
	})

	return err
}

// ListNodes lists all nodes
func (c *Client) ListNodes() ([]*proto.Node, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := c.client.ListNodes(ctx, &proto.ListNodesRequest{})
	if err != nil {
		return nil, err
	}

	return resp.Nodes, nil
}

// GetNode gets a node by ID
func (c *Client) GetNode(id string) (*proto.Node, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := c.client.GetNode(ctx, &proto.GetNodeRequest{
		Id: id,
	})
	if err != nil {
		return nil, err
	}

	return resp.Node, nil
}

// ListTasks lists all tasks (optionally filtered)
func (c *Client) ListTasks(serviceID, nodeID string) ([]*proto.Task, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := c.client.ListTasks(ctx, &proto.ListTasksRequest{
		ServiceId: serviceID,
		NodeId:    nodeID,
	})
	if err != nil {
		return nil, err
	}

	return resp.Tasks, nil
}

// CreateSecret creates a new secret
func (c *Client) CreateSecret(name string, data []byte) (*proto.Secret, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := c.client.CreateSecret(ctx, &proto.CreateSecretRequest{
		Name: name,
		Data: data,
	})
	if err != nil {
		return nil, err
	}

	return resp.Secret, nil
}

// GetSecretByName retrieves a secret by name (returns metadata only, no data)
func (c *Client) GetSecretByName(name string) (*proto.Secret, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := c.client.GetSecretByName(ctx, &proto.GetSecretByNameRequest{
		Name: name,
	})
	if err != nil {
		return nil, err
	}

	return resp.Secret, nil
}

// DeleteSecret deletes a secret
func (c *Client) DeleteSecret(name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := c.client.DeleteSecret(ctx, &proto.DeleteSecretRequest{
		Id: name,
	})

	return err
}

// ListSecrets lists all secrets (returns metadata only, no data)
func (c *Client) ListSecrets() ([]*proto.Secret, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := c.client.ListSecrets(ctx, &proto.ListSecretsRequest{})
	if err != nil {
		return nil, err
	}

	return resp.Secrets, nil
}

// CreateVolume creates a new volume
func (c *Client) CreateVolume(name, driver string, options map[string]string) (*proto.Volume, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := c.client.CreateVolume(ctx, &proto.CreateVolumeRequest{
		Name:       name,
		Driver:     driver,
		DriverOpts: options,
	})
	if err != nil {
		return nil, err
	}

	return resp.Volume, nil
}

// GetVolumeByName retrieves a volume by name
func (c *Client) GetVolumeByName(name string) (*proto.Volume, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := c.client.GetVolumeByName(ctx, &proto.GetVolumeByNameRequest{
		Name: name,
	})
	if err != nil {
		return nil, err
	}

	return resp.Volume, nil
}

// DeleteVolume deletes a volume
func (c *Client) DeleteVolume(name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := c.client.DeleteVolume(ctx, &proto.DeleteVolumeRequest{
		Id: name,
	})

	return err
}

// ListVolumes lists all volumes
func (c *Client) ListVolumes() ([]*proto.Volume, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := c.client.ListVolumes(ctx, &proto.ListVolumesRequest{})
	if err != nil {
		return nil, err
	}

	return resp.Volumes, nil
}

// GenerateJoinToken generates a join token for a worker or manager
func (c *Client) GenerateJoinToken(role string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := c.client.GenerateJoinToken(ctx, &proto.GenerateJoinTokenRequest{
		Role: role,
	})
	if err != nil {
		return "", err
	}

	return resp.Token, nil
}

// GetClusterInfo returns information about the cluster
func (c *Client) GetClusterInfo() (*proto.GetClusterInfoResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := c.client.GetClusterInfo(ctx, &proto.GetClusterInfoRequest{})
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// JoinCluster joins a node to the cluster
func (c *Client) JoinCluster(nodeID, bindAddr, token string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := c.client.JoinCluster(ctx, &proto.JoinClusterRequest{
		NodeId:   nodeID,
		BindAddr: bindAddr,
		Token:    token,
	})

	return err
}

// requestCertificate requests a certificate from the manager using a join token
func requestCertificate(addr, token, certDir string) error {
	// Connect without TLS for certificate request (token provides authentication)
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to manager: %w", err)
	}
	defer conn.Close()

	client := proto.NewWarrenAPIClient(conn)

	// Request certificate with "cli" as node ID
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.RequestCertificate(ctx, &proto.RequestCertificateRequest{
		NodeId: "cli",
		Token:  token,
	})
	if err != nil {
		return fmt.Errorf("failed to request certificate: %w", err)
	}

	// Create directory and save certificate files
	if err := os.MkdirAll(certDir, 0700); err != nil {
		return fmt.Errorf("failed to create cert directory: %w", err)
	}

	certPath := certDir + "/node.crt"
	keyPath := certDir + "/node.key"
	caPath := certDir + "/ca.crt"

	if err := os.WriteFile(certPath, resp.Certificate, 0600); err != nil {
		return fmt.Errorf("failed to write certificate: %w", err)
	}
	if err := os.WriteFile(keyPath, resp.PrivateKey, 0600); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}
	if err := os.WriteFile(caPath, resp.CaCert, 0644); err != nil {
		return fmt.Errorf("failed to write CA certificate: %w", err)
	}

	return nil
}

// connectWithMTLS establishes a gRPC connection with mTLS
func connectWithMTLS(addr, certDir string) (*grpc.ClientConn, error) {
	// Load CLI certificate
	cert, err := security.LoadCertFromFile(certDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load CLI certificate: %w", err)
	}

	// Load CA certificate for server verification
	caCert, err := security.LoadCACertFromFile(certDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load CA certificate: %w", err)
	}

	// Create cert pool for server verification
	certPool := x509.NewCertPool()
	certPool.AddCert(caCert)

	// Configure TLS
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{*cert},
		RootCAs:      certPool,
		MinVersion:   tls.VersionTLS13,
	}

	// Create gRPC connection with TLS
	creds := credentials.NewTLS(tlsConfig)
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("failed to dial manager: %w", err)
	}

	return conn, nil
}
