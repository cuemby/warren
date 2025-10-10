package client

import (
	"context"
	"fmt"
	"time"

	"github.com/cuemby/warren/api/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client wraps the Warren gRPC client for easy CLI usage
type Client struct {
	conn   *grpc.ClientConn
	client proto.WarrenAPIClient
}

// NewClient creates a new Warren client
func NewClient(addr string) (*Client, error) {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %v", err)
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

// ListSecrets lists all secrets
func (c *Client) ListSecrets() ([]*proto.Secret, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := c.client.ListSecrets(ctx, &proto.ListSecretsRequest{})
	if err != nil {
		return nil, err
	}

	return resp.Secrets, nil
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
