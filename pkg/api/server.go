package api

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/cuemby/warren/api/proto"
	"github.com/cuemby/warren/pkg/manager"
	"github.com/cuemby/warren/pkg/security"
	"github.com/cuemby/warren/pkg/types"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Server implements the WarrenAPI gRPC service
type Server struct {
	proto.UnimplementedWarrenAPIServer
	manager *manager.Manager
	grpc    *grpc.Server
}

// NewServer creates a new API server with mTLS
func NewServer(mgr *manager.Manager) (*Server, error) {
	// Get certificate directory for this manager
	certDir, err := security.GetCertDir("manager", mgr.NodeID())
	if err != nil {
		return nil, fmt.Errorf("failed to get cert directory: %w", err)
	}

	// Check if certificates exist
	if !security.CertExists(certDir) {
		return nil, fmt.Errorf("manager certificate not found at %s - ensure cluster is initialized", certDir)
	}

	// Load manager certificate
	cert, err := security.LoadCertFromFile(certDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load manager certificate: %w", err)
	}

	// Load CA certificate for client verification
	caCert, err := security.LoadCACertFromFile(certDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load CA certificate: %w", err)
	}

	// Create cert pool for client verification
	certPool := x509.NewCertPool()
	certPool.AddCert(caCert)

	// Configure TLS with client certificate verification
	// Use RequestClientCert to allow initial connections without certificates (for RequestCertificate RPC)
	// Individual RPCs will verify client certs as needed
	tlsConfig := &tls.Config{
		ClientAuth:   tls.RequestClientCert, // Request but don't require - verify per-RPC
		Certificates: []tls.Certificate{*cert},
		ClientCAs:    certPool,
		MinVersion:   tls.VersionTLS13,
	}

	// Create gRPC server with TLS credentials
	creds := credentials.NewTLS(tlsConfig)
	grpcServer := grpc.NewServer(grpc.Creds(creds))

	return &Server{
		manager: mgr,
		grpc:    grpcServer,
	}, nil
}

// ensureLeader checks if this node is the leader and returns an error if not
// This should be called for all write operations
func (s *Server) ensureLeader() error {
	if !s.manager.IsLeader() {
		leaderAddr := s.manager.LeaderAddr()
		if leaderAddr == "" {
			return fmt.Errorf("no leader elected yet")
		}
		return fmt.Errorf("not the leader, current leader is at: %s", leaderAddr)
	}
	return nil
}

// Start starts the gRPC server
func (s *Server) Start(addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	proto.RegisterWarrenAPIServer(s.grpc, s)

	fmt.Printf("gRPC API listening on %s\n", addr)
	return s.grpc.Serve(lis)
}

// Stop gracefully stops the gRPC server
func (s *Server) Stop() {
	if s.grpc != nil {
		s.grpc.GracefulStop()
	}
}

// RegisterNode registers a new node (worker or manager) with the cluster
func (s *Server) RegisterNode(ctx context.Context, req *proto.RegisterNodeRequest) (*proto.RegisterNodeResponse, error) {
	// Ensure we're the leader for write operations
	if err := s.ensureLeader(); err != nil {
		return nil, err
	}

	node := &types.Node{
		ID:      req.Id,
		Role:    types.NodeRole(req.Role),
		Address: req.Address,
		Resources: &types.NodeResources{
			CPUCores:    int(req.Resources.CpuCores),
			MemoryBytes: req.Resources.MemoryBytes,
			DiskBytes:   req.Resources.DiskBytes,
		},
		Status:        types.NodeStatusReady,
		LastHeartbeat: time.Now(),
		CreatedAt:     time.Now(),
		Labels:        req.Labels,
	}

	// TODO: Allocate overlay IP from IP pool
	node.OverlayIP = net.ParseIP("10.0.0.1")

	if err := s.manager.CreateNode(node); err != nil {
		return nil, fmt.Errorf("failed to create node: %w", err)
	}

	return &proto.RegisterNodeResponse{
		Node:      nodeToProto(node),
		OverlayIp: node.OverlayIP.String(),
	}, nil
}

// Heartbeat processes heartbeat from a worker node
func (s *Server) Heartbeat(ctx context.Context, req *proto.HeartbeatRequest) (*proto.HeartbeatResponse, error) {
	node, err := s.manager.GetNode(req.NodeId)
	if err != nil {
		return nil, fmt.Errorf("node not found: %w", err)
	}

	// Update node heartbeat and available resources
	node.LastHeartbeat = time.Now()
	node.Status = types.NodeStatusReady
	if req.AvailableResources != nil {
		node.Resources.CPUCores = int(req.AvailableResources.CpuCores)
		node.Resources.MemoryBytes = req.AvailableResources.MemoryBytes
		node.Resources.DiskBytes = req.AvailableResources.DiskBytes
	}

	if err := s.manager.UpdateNode(node); err != nil {
		return nil, fmt.Errorf("failed to update node: %w", err)
	}

	// Process task status updates
	for _, ts := range req.TaskStatuses {
		task, err := s.manager.GetTask(ts.TaskId)
		if err != nil {
			continue // Skip tasks that don't exist
		}

		task.ActualState = types.TaskState(ts.ActualState)
		task.ContainerID = ts.ContainerId

		if err := s.manager.UpdateTask(task); err != nil {
			// Log error but don't fail heartbeat
			continue
		}
	}

	return &proto.HeartbeatResponse{
		Status: "ok",
	}, nil
}

// ListNodes returns all nodes in the cluster
func (s *Server) ListNodes(ctx context.Context, req *proto.ListNodesRequest) (*proto.ListNodesResponse, error) {
	nodes, err := s.manager.ListNodes()
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	// Filter by role if specified
	var filtered []*types.Node
	if req.RoleFilter != "" {
		for _, n := range nodes {
			if string(n.Role) == req.RoleFilter {
				filtered = append(filtered, n)
			}
		}
	} else {
		filtered = nodes
	}

	protoNodes := make([]*proto.Node, len(filtered))
	for i, n := range filtered {
		protoNodes[i] = nodeToProto(n)
	}

	return &proto.ListNodesResponse{
		Nodes: protoNodes,
	}, nil
}

// GetNode returns a specific node by ID
func (s *Server) GetNode(ctx context.Context, req *proto.GetNodeRequest) (*proto.GetNodeResponse, error) {
	node, err := s.manager.GetNode(req.Id)
	if err != nil {
		return nil, fmt.Errorf("node not found: %w", err)
	}

	return &proto.GetNodeResponse{
		Node: nodeToProto(node),
	}, nil
}

// RemoveNode removes a node from the cluster
func (s *Server) RemoveNode(ctx context.Context, req *proto.RemoveNodeRequest) (*proto.RemoveNodeResponse, error) {
	// Ensure we're the leader for write operations
	if err := s.ensureLeader(); err != nil {
		return nil, err
	}

	if err := s.manager.DeleteNode(req.Id); err != nil {
		return nil, fmt.Errorf("failed to remove node: %w", err)
	}

	return &proto.RemoveNodeResponse{
		Status: "ok",
	}, nil
}

// CreateService creates a new service
func (s *Server) CreateService(ctx context.Context, req *proto.CreateServiceRequest) (*proto.CreateServiceResponse, error) {
	// Ensure we're the leader for write operations
	if err := s.ensureLeader(); err != nil {
		return nil, err
	}

	// Convert env map to slice
	var envSlice []string
	for k, v := range req.Env {
		envSlice = append(envSlice, fmt.Sprintf("%s=%s", k, v))
	}

	service := &types.Service{
		ID:             uuid.New().String(),
		Name:           req.Name,
		Image:          req.Image,
		Replicas:       int(req.Replicas),
		Mode:           types.ServiceMode(req.Mode),
		DeployStrategy: types.DeployStrategy(req.DeployStrategy),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		Env:            envSlice,
		Networks:       req.Networks,
	}

	if req.UpdateConfig != nil {
		service.UpdateConfig = &types.UpdateConfig{
			Parallelism:   int(req.UpdateConfig.Parallelism),
			Delay:         time.Duration(req.UpdateConfig.DelaySeconds) * time.Second,
			FailureAction: req.UpdateConfig.FailureAction,
		}
	}

	if req.HealthCheck != nil {
		service.HealthCheck = protoToHealthCheck(req.HealthCheck)
	}

	if req.RestartPolicy != nil {
		service.RestartPolicy = &types.RestartPolicy{
			Condition:   types.RestartCondition(req.RestartPolicy.Condition),
			MaxAttempts: int(req.RestartPolicy.MaxAttempts),
			Delay:       time.Duration(req.RestartPolicy.DelaySeconds) * time.Second,
		}
	}

	if req.Resources != nil {
		service.Resources = &types.ResourceRequirements{
			CPULimit:          float64(req.Resources.CpuShares) / 1024.0, // Convert shares to cores
			MemoryLimit:       req.Resources.MemoryBytes,
			MemoryReservation: req.Resources.MemoryReservationBytes,
		}
	}

	// Convert port mappings from proto to types
	if len(req.Ports) > 0 {
		service.Ports = make([]*types.PortMapping, 0, len(req.Ports))
		for _, protoPort := range req.Ports {
			publishMode := types.PublishModeHost
			if protoPort.PublishMode == proto.PortMapping_INGRESS {
				publishMode = types.PublishModeIngress
			}

			service.Ports = append(service.Ports, &types.PortMapping{
				Name:          protoPort.Name,
				ContainerPort: int(protoPort.ContainerPort),
				HostPort:      int(protoPort.HostPort),
				Protocol:      protoPort.Protocol,
				PublishMode:   publishMode,
			})
		}
	}

	if err := s.manager.CreateService(service); err != nil {
		return nil, fmt.Errorf("failed to create service: %w", err)
	}

	return &proto.CreateServiceResponse{
		Service: serviceToProto(service),
	}, nil
}

// UpdateService updates an existing service
func (s *Server) UpdateService(ctx context.Context, req *proto.UpdateServiceRequest) (*proto.UpdateServiceResponse, error) {
	// Ensure we're the leader for write operations
	if err := s.ensureLeader(); err != nil {
		return nil, err
	}

	service, err := s.manager.GetService(req.Id)
	if err != nil {
		return nil, fmt.Errorf("service not found: %w", err)
	}

	// Update fields
	if req.Replicas > 0 {
		service.Replicas = int(req.Replicas)
	}
	if req.Image != "" {
		service.Image = req.Image
	}
	if req.Env != nil {
		var envSlice []string
		for k, v := range req.Env {
			envSlice = append(envSlice, fmt.Sprintf("%s=%s", k, v))
		}
		service.Env = envSlice
	}
	service.UpdatedAt = time.Now()

	if err := s.manager.UpdateService(service); err != nil {
		return nil, fmt.Errorf("failed to update service: %w", err)
	}

	return &proto.UpdateServiceResponse{
		Service: serviceToProto(service),
	}, nil
}

// DeleteService deletes a service
func (s *Server) DeleteService(ctx context.Context, req *proto.DeleteServiceRequest) (*proto.DeleteServiceResponse, error) {
	// Ensure we're the leader for write operations
	if err := s.ensureLeader(); err != nil {
		return nil, err
	}

	if err := s.manager.DeleteService(req.Id); err != nil {
		return nil, fmt.Errorf("failed to delete service: %w", err)
	}

	return &proto.DeleteServiceResponse{
		Status: "ok",
	}, nil
}

// GetService returns a specific service
func (s *Server) GetService(ctx context.Context, req *proto.GetServiceRequest) (*proto.GetServiceResponse, error) {
	var service *types.Service
	var err error

	if req.Id != "" {
		service, err = s.manager.GetService(req.Id)
	} else if req.Name != "" {
		service, err = s.manager.GetServiceByName(req.Name)
	} else {
		return nil, fmt.Errorf("either id or name must be specified")
	}

	if err != nil {
		return nil, fmt.Errorf("service not found: %w", err)
	}

	return &proto.GetServiceResponse{
		Service: serviceToProto(service),
	}, nil
}

// ListServices returns all services
func (s *Server) ListServices(ctx context.Context, req *proto.ListServicesRequest) (*proto.ListServicesResponse, error) {
	services, err := s.manager.ListServices()
	if err != nil {
		return nil, fmt.Errorf("failed to list services: %w", err)
	}

	protoServices := make([]*proto.Service, len(services))
	for i, svc := range services {
		protoServices[i] = serviceToProto(svc)
	}

	return &proto.ListServicesResponse{
		Services: protoServices,
	}, nil
}

// UpdateTaskStatus updates the status of a task
func (s *Server) UpdateTaskStatus(ctx context.Context, req *proto.UpdateTaskStatusRequest) (*proto.UpdateTaskStatusResponse, error) {
	task, err := s.manager.GetTask(req.TaskId)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

	task.ActualState = types.TaskState(req.ActualState)
	task.ContainerID = req.ContainerId

	if err := s.manager.UpdateTask(task); err != nil {
		return nil, fmt.Errorf("failed to update task: %w", err)
	}

	return &proto.UpdateTaskStatusResponse{
		Status: "ok",
	}, nil
}

// ReportTaskHealth reports the health status of a task
func (s *Server) ReportTaskHealth(ctx context.Context, req *proto.ReportTaskHealthRequest) (*proto.ReportTaskHealthResponse, error) {
	task, err := s.manager.GetTask(req.TaskId)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

	// Update task health status
	if task.HealthStatus == nil {
		task.HealthStatus = &types.HealthStatus{}
	}

	task.HealthStatus.Healthy = req.Healthy
	task.HealthStatus.Message = req.Message
	task.HealthStatus.CheckedAt = req.CheckedAt.AsTime()
	task.HealthStatus.ConsecutiveFailures = int(req.ConsecutiveFailures)
	task.HealthStatus.ConsecutiveSuccesses = int(req.ConsecutiveSuccesses)

	// Update task in storage
	if err := s.manager.UpdateTask(task); err != nil {
		return nil, fmt.Errorf("failed to update task health: %w", err)
	}

	return &proto.ReportTaskHealthResponse{
		Status: "ok",
	}, nil
}

// ListTasks returns tasks, optionally filtered by service or node
func (s *Server) ListTasks(ctx context.Context, req *proto.ListTasksRequest) (*proto.ListTasksResponse, error) {
	var tasks []*types.Task
	var err error

	if req.ServiceId != "" {
		tasks, err = s.manager.ListTasksByService(req.ServiceId)
	} else if req.NodeId != "" {
		tasks, err = s.manager.ListTasksByNode(req.NodeId)
	} else {
		tasks, err = s.manager.ListTasks()
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	protoTasks := make([]*proto.Task, len(tasks))
	for i, task := range tasks {
		protoTasks[i] = taskToProto(task)
	}

	return &proto.ListTasksResponse{
		Tasks: protoTasks,
	}, nil
}

// GetTask returns a specific task
func (s *Server) GetTask(ctx context.Context, req *proto.GetTaskRequest) (*proto.GetTaskResponse, error) {
	task, err := s.manager.GetTask(req.Id)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

	return &proto.GetTaskResponse{
		Task: taskToProto(task),
	}, nil
}

// WatchTasks streams task events to a worker node
func (s *Server) WatchTasks(req *proto.WatchTasksRequest, stream proto.WarrenAPI_WatchTasksServer) error {
	// TODO: Implement task watch stream
	// For now, return unimplemented
	return fmt.Errorf("WatchTasks not yet implemented")
}

// CreateSecret creates a new secret
func (s *Server) CreateSecret(ctx context.Context, req *proto.CreateSecretRequest) (*proto.CreateSecretResponse, error) {
	// Ensure we're the leader for write operations
	if err := s.ensureLeader(); err != nil {
		return nil, err
	}

	// Encrypt the secret data before storing
	encryptedData, err := s.manager.EncryptSecret(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt secret: %w", err)
	}

	secret := &types.Secret{
		ID:        uuid.New().String(),
		Name:      req.Name,
		Data:      encryptedData,
		CreatedAt: time.Now(),
	}

	if err := s.manager.CreateSecret(secret); err != nil {
		return nil, fmt.Errorf("failed to create secret: %w", err)
	}

	return &proto.CreateSecretResponse{
		Secret: secretToProto(secret),
	}, nil
}

// GetSecretByName retrieves a secret by name (includes encrypted data for workers)
func (s *Server) GetSecretByName(ctx context.Context, req *proto.GetSecretByNameRequest) (*proto.GetSecretByNameResponse, error) {
	secret, err := s.manager.GetSecretByName(req.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get secret: %w", err)
	}

	// Include encrypted data for workers to decrypt
	return &proto.GetSecretByNameResponse{
		Secret: secretToProtoWithData(secret),
	}, nil
}

// DeleteSecret deletes a secret
func (s *Server) DeleteSecret(ctx context.Context, req *proto.DeleteSecretRequest) (*proto.DeleteSecretResponse, error) {
	// Ensure we're the leader for write operations
	if err := s.ensureLeader(); err != nil {
		return nil, err
	}

	if err := s.manager.DeleteSecret(req.Id); err != nil {
		return nil, fmt.Errorf("failed to delete secret: %w", err)
	}

	return &proto.DeleteSecretResponse{
		Status: "ok",
	}, nil
}

// ListSecrets returns all secrets (without data)
func (s *Server) ListSecrets(ctx context.Context, req *proto.ListSecretsRequest) (*proto.ListSecretsResponse, error) {
	secrets, err := s.manager.ListSecrets()
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets: %w", err)
	}

	protoSecrets := make([]*proto.Secret, len(secrets))
	for i, secret := range secrets {
		protoSecrets[i] = secretToProto(secret)
	}

	return &proto.ListSecretsResponse{
		Secrets: protoSecrets,
	}, nil
}

// CreateVolume creates a new volume
func (s *Server) CreateVolume(ctx context.Context, req *proto.CreateVolumeRequest) (*proto.CreateVolumeResponse, error) {
	// Ensure we're the leader for write operations
	if err := s.ensureLeader(); err != nil {
		return nil, err
	}

	volume := &types.Volume{
		ID:        uuid.New().String(),
		Name:      req.Name,
		Driver:    req.Driver,
		Options:   req.DriverOpts,
		CreatedAt: time.Now(),
	}

	if err := s.manager.CreateVolume(volume); err != nil {
		return nil, fmt.Errorf("failed to create volume: %w", err)
	}

	return &proto.CreateVolumeResponse{
		Volume: volumeToProto(volume),
	}, nil
}

// GetVolumeByName retrieves a volume by name
func (s *Server) GetVolumeByName(ctx context.Context, req *proto.GetVolumeByNameRequest) (*proto.GetVolumeByNameResponse, error) {
	volume, err := s.manager.GetVolumeByName(req.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get volume: %w", err)
	}

	return &proto.GetVolumeByNameResponse{
		Volume: volumeToProto(volume),
	}, nil
}

// DeleteVolume deletes a volume
func (s *Server) DeleteVolume(ctx context.Context, req *proto.DeleteVolumeRequest) (*proto.DeleteVolumeResponse, error) {
	// Ensure we're the leader for write operations
	if err := s.ensureLeader(); err != nil {
		return nil, err
	}

	if err := s.manager.DeleteVolume(req.Id); err != nil {
		return nil, fmt.Errorf("failed to delete volume: %w", err)
	}

	return &proto.DeleteVolumeResponse{
		Status: "ok",
	}, nil
}

// ListVolumes returns all volumes
func (s *Server) ListVolumes(ctx context.Context, req *proto.ListVolumesRequest) (*proto.ListVolumesResponse, error) {
	volumes, err := s.manager.ListVolumes()
	if err != nil {
		return nil, fmt.Errorf("failed to list volumes: %w", err)
	}

	protoVolumes := make([]*proto.Volume, len(volumes))
	for i, vol := range volumes {
		protoVolumes[i] = volumeToProto(vol)
	}

	return &proto.ListVolumesResponse{
		Volumes: protoVolumes,
	}, nil
}

// GenerateJoinToken generates a join token for adding nodes
func (s *Server) GenerateJoinToken(ctx context.Context, req *proto.GenerateJoinTokenRequest) (*proto.GenerateJoinTokenResponse, error) {
	// Ensure we're the leader (only leader can generate tokens)
	if err := s.ensureLeader(); err != nil {
		return nil, err
	}

	// Generate token
	token, err := s.manager.GenerateJoinToken(req.Role)
	if err != nil {
		return nil, fmt.Errorf("failed to generate join token: %w", err)
	}

	return &proto.GenerateJoinTokenResponse{
		Token:     token.Token,
		Role:      token.Role,
		ExpiresAt: timestamppb.New(token.ExpiresAt),
	}, nil
}

// JoinCluster handles a manager join request
func (s *Server) JoinCluster(ctx context.Context, req *proto.JoinClusterRequest) (*proto.JoinClusterResponse, error) {
	// Ensure we're the leader (only leader can add voters)
	if err := s.ensureLeader(); err != nil {
		return nil, err
	}

	// Validate the join token
	role, err := s.manager.ValidateJoinToken(req.Token)
	if err != nil {
		return nil, fmt.Errorf("invalid join token: %w", err)
	}

	// Only managers can join via this endpoint
	if role != "manager" {
		return nil, fmt.Errorf("invalid token role: expected manager, got %s", role)
	}

	// Add the manager as a voter in Raft
	if err := s.manager.AddVoter(req.NodeId, req.BindAddr); err != nil {
		return nil, fmt.Errorf("failed to add voter: %w", err)
	}

	return &proto.JoinClusterResponse{
		Status:     "success",
		LeaderAddr: s.manager.LeaderAddr(),
	}, nil
}

// GetClusterInfo returns information about the Raft cluster
func (s *Server) GetClusterInfo(ctx context.Context, req *proto.GetClusterInfoRequest) (*proto.GetClusterInfoResponse, error) {
	servers, err := s.manager.GetClusterServers()
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster servers: %w", err)
	}

	protoServers := make([]*proto.ClusterServer, len(servers))
	for i, srv := range servers {
		protoServers[i] = &proto.ClusterServer{
			Id:       string(srv.ID),
			Address:  string(srv.Address),
			Suffrage: srv.Suffrage.String(),
		}
	}

	leaderAddr := s.manager.LeaderAddr()
	leaderID := ""
	// Try to find leader ID from servers
	for _, srv := range servers {
		if string(srv.Address) == leaderAddr {
			leaderID = string(srv.ID)
			break
		}
	}

	return &proto.GetClusterInfoResponse{
		LeaderId:   leaderID,
		LeaderAddr: leaderAddr,
		Servers:    protoServers,
	}, nil
}

// Helper functions to convert between internal types and protobuf

func nodeToProto(n *types.Node) *proto.Node {
	return &proto.Node{
		Id:      n.ID,
		Role:    string(n.Role),
		Address: n.Address,
		OverlayIp: func() string {
			if n.OverlayIP != nil {
				return n.OverlayIP.String()
			}
			return ""
		}(),
		Resources: &proto.NodeResources{
			CpuCores:    int64(n.Resources.CPUCores),
			MemoryBytes: n.Resources.MemoryBytes,
			DiskBytes:   n.Resources.DiskBytes,
		},
		Status:        string(n.Status),
		LastHeartbeat: timestamppb.New(n.LastHeartbeat),
		CreatedAt:     timestamppb.New(n.CreatedAt),
		Labels:        n.Labels,
	}
}

func serviceToProto(s *types.Service) *proto.Service {
	// Convert env slice to map
	envMap := make(map[string]string)
	for _, e := range s.Env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	ps := &proto.Service{
		Id:             s.ID,
		Name:           s.Name,
		Image:          s.Image,
		Replicas:       int32(s.Replicas),
		Mode:           string(s.Mode),
		DeployStrategy: string(s.DeployStrategy),
		Env:            envMap,
		Networks:       s.Networks,
		CreatedAt:      timestamppb.New(s.CreatedAt),
		UpdatedAt:      timestamppb.New(s.UpdatedAt),
	}

	if s.UpdateConfig != nil {
		ps.UpdateConfig = &proto.UpdateConfig{
			Parallelism:   int32(s.UpdateConfig.Parallelism),
			DelaySeconds:  int32(s.UpdateConfig.Delay / time.Second),
			FailureAction: s.UpdateConfig.FailureAction,
		}
	}

	if s.HealthCheck != nil {
		ps.HealthCheck = healthCheckToProto(s.HealthCheck)
	}

	if s.RestartPolicy != nil {
		ps.RestartPolicy = &proto.RestartPolicy{
			Condition:    string(s.RestartPolicy.Condition),
			MaxAttempts:  int32(s.RestartPolicy.MaxAttempts),
			DelaySeconds: int32(s.RestartPolicy.Delay / time.Second),
		}
	}

	if s.Resources != nil {
		ps.Resources = &proto.ResourceRequirements{
			CpuShares:              int64(s.Resources.CPULimit * 1024), // Convert cores to shares
			MemoryBytes:            s.Resources.MemoryLimit,
			MemoryReservationBytes: s.Resources.MemoryReservation,
		}
	}

	// Convert port mappings from types to proto
	if len(s.Ports) > 0 {
		ps.Ports = make([]*proto.PortMapping, 0, len(s.Ports))
		for _, typePort := range s.Ports {
			publishMode := proto.PortMapping_HOST
			if typePort.PublishMode == types.PublishModeIngress {
				publishMode = proto.PortMapping_INGRESS
			}

			ps.Ports = append(ps.Ports, &proto.PortMapping{
				Name:          typePort.Name,
				ContainerPort: int32(typePort.ContainerPort),
				HostPort:      int32(typePort.HostPort),
				Protocol:      typePort.Protocol,
				PublishMode:   publishMode,
			})
		}
	}

	return ps
}

func taskToProto(t *types.Task) *proto.Task {
	// Convert env slice to map
	envMap := make(map[string]string)
	for _, e := range t.Env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	pt := &proto.Task{
		Id:           t.ID,
		ServiceId:    t.ServiceID,
		ServiceName:  t.ServiceName,
		NodeId:       t.NodeID,
		ContainerId:  t.ContainerID,
		DesiredState: string(t.DesiredState),
		ActualState:  string(t.ActualState),
		Image:        t.Image,
		Env:          envMap,
		CreatedAt:    timestamppb.New(t.CreatedAt),
		Error:        t.Error,
	}

	// Use StartedAt for UpdatedAt if available, otherwise CreatedAt
	if !t.StartedAt.IsZero() {
		pt.UpdatedAt = timestamppb.New(t.StartedAt)
	} else {
		pt.UpdatedAt = timestamppb.New(t.CreatedAt)
	}

	if t.Resources != nil {
		pt.Resources = &proto.ResourceRequirements{
			CpuShares:              int64(t.Resources.CPULimit * 1024),
			MemoryBytes:            t.Resources.MemoryLimit,
			MemoryReservationBytes: t.Resources.MemoryReservation,
		}
	}

	if t.HealthCheck != nil {
		pt.HealthCheck = healthCheckToProto(t.HealthCheck)
	}

	if t.RestartPolicy != nil {
		pt.RestartPolicy = &proto.RestartPolicy{
			Condition:    string(t.RestartPolicy.Condition),
			MaxAttempts:  int32(t.RestartPolicy.MaxAttempts),
			DelaySeconds: int32(t.RestartPolicy.Delay / time.Second),
		}
	}

	return pt
}

func secretToProto(s *types.Secret) *proto.Secret {
	return &proto.Secret{
		Id:        s.ID,
		Name:      s.Name,
		CreatedAt: timestamppb.New(s.CreatedAt),
		// Note: Data is not included for security (CLI listing)
	}
}

func secretToProtoWithData(s *types.Secret) *proto.Secret {
	return &proto.Secret{
		Id:        s.ID,
		Name:      s.Name,
		CreatedAt: timestamppb.New(s.CreatedAt),
		Data:      s.Data, // Include encrypted data for workers
	}
}

func volumeToProto(v *types.Volume) *proto.Volume {
	return &proto.Volume{
		Id:         v.ID,
		Name:       v.Name,
		Driver:     v.Driver,
		DriverOpts: v.Options,
		NodeId:     v.NodeID,
		MountPath:  v.MountPath,
		Labels:     make(map[string]string), // Volume doesn't have labels in types
		CreatedAt:  timestamppb.New(v.CreatedAt),
	}
}

// protoToHealthCheck converts proto HealthCheck to types.HealthCheck
func protoToHealthCheck(ph *proto.HealthCheck) *types.HealthCheck {
	if ph == nil {
		return nil
	}

	hc := &types.HealthCheck{
		Interval: time.Duration(ph.IntervalSeconds) * time.Second,
		Timeout:  time.Duration(ph.TimeoutSeconds) * time.Second,
		Retries:  int(ph.Retries),
	}

	switch ph.Type {
	case proto.HealthCheck_HTTP:
		hc.Type = types.HealthCheckHTTP
		if ph.Http != nil {
			// Construct endpoint from HTTP config
			scheme := ph.Http.Scheme
			if scheme == "" {
				scheme = "http"
			}
			hc.Endpoint = fmt.Sprintf("%s://:%d%s", scheme, ph.Http.Port, ph.Http.Path)
		}
	case proto.HealthCheck_TCP:
		hc.Type = types.HealthCheckTCP
		if ph.Tcp != nil {
			hc.Endpoint = fmt.Sprintf(":%d", ph.Tcp.Port)
		}
	case proto.HealthCheck_EXEC:
		hc.Type = types.HealthCheckExec
		if ph.Exec != nil {
			hc.Command = ph.Exec.Command
		}
	}

	return hc
}

// healthCheckToProto converts types.HealthCheck to proto HealthCheck
func healthCheckToProto(hc *types.HealthCheck) *proto.HealthCheck {
	if hc == nil {
		return nil
	}

	ph := &proto.HealthCheck{
		IntervalSeconds: int32(hc.Interval / time.Second),
		TimeoutSeconds:  int32(hc.Timeout / time.Second),
		Retries:         int32(hc.Retries),
	}

	switch hc.Type {
	case types.HealthCheckHTTP:
		ph.Type = proto.HealthCheck_HTTP
		// Parse endpoint to extract path and port
		// For now, keep it simple - just set basic HTTP config
		ph.Http = &proto.HTTPHealthCheck{
			Path:          "/health",
			Port:          8080,
			Scheme:        "http",
			StatusCodeMin: 200,
			StatusCodeMax: 399,
		}
	case types.HealthCheckTCP:
		ph.Type = proto.HealthCheck_TCP
		// Parse endpoint to extract port
		ph.Tcp = &proto.TCPHealthCheck{
			Port: 8080,
		}
	case types.HealthCheckExec:
		ph.Type = proto.HealthCheck_EXEC
		ph.Exec = &proto.ExecHealthCheck{
			Command: hc.Command,
		}
	}

	return ph
}

// StreamEvents streams cluster events to the client
// TODO: Complete implementation
func (s *Server) StreamEvents(req *proto.StreamEventsRequest, stream proto.WarrenAPI_StreamEventsServer) error {
	return fmt.Errorf("event streaming not yet implemented")
}

// RequestCertificate issues a certificate for a node joining the cluster
func (s *Server) RequestCertificate(ctx context.Context, req *proto.RequestCertificateRequest) (*proto.RequestCertificateResponse, error) {
	// Validate token
	role, err := s.manager.ValidateToken(req.Token)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	// Issue certificate
	cert, err := s.manager.IssueCertificate(req.NodeId, role)
	if err != nil {
		return nil, fmt.Errorf("failed to issue certificate: %w", err)
	}

	// Convert to PEM format for transmission
	certPEM, keyPEM, err := s.manager.CertToPEM(cert)
	if err != nil {
		return nil, fmt.Errorf("failed to convert certificate to PEM: %w", err)
	}

	// Get CA certificate
	caCertPEM := s.manager.GetCACertPEM()

	return &proto.RequestCertificateResponse{
		Certificate: certPEM,
		PrivateKey:  keyPEM,
		CaCert:      caCertPEM,
	}, nil
}

// --- Ingress Operations ---

// CreateIngress creates a new ingress
func (s *Server) CreateIngress(ctx context.Context, req *proto.CreateIngressRequest) (*proto.CreateIngressResponse, error) {
	// Forward to leader if not leader
	if !s.manager.IsLeader() {
		return nil, fmt.Errorf("not leader, forward to leader")
	}

	// Validate request
	if req.Name == "" {
		return nil, fmt.Errorf("ingress name is required")
	}

	// Check if ingress already exists
	existing, _ := s.manager.GetIngressByName(req.Name)
	if existing != nil {
		return nil, fmt.Errorf("ingress %s already exists", req.Name)
	}

	// Convert proto to internal type
	ingress := &types.Ingress{
		ID:        uuid.New().String(),
		Name:      req.Name,
		Rules:     convertProtoIngressRules(req.Rules),
		TLS:       convertProtoIngressTLS(req.Tls),
		Labels:    req.Labels,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Create ingress in storage via Raft
	if err := s.manager.CreateIngress(ingress); err != nil {
		return nil, fmt.Errorf("failed to create ingress: %w", err)
	}

	// Reload ingress proxy to pick up the new ingress
	if err := s.manager.ReloadIngress(); err != nil {
		// Log but don't fail the request
		fmt.Printf("Warning: Failed to reload ingress proxy: %v\n", err)
	}

	// If AutoTLS is enabled, request Let's Encrypt certificate
	if ingress.TLS != nil && ingress.TLS.AutoTLS && ingress.TLS.Email != "" {
		// Enable ACME if not already enabled
		if err := s.manager.EnableACME(ingress.TLS.Email); err != nil {
			fmt.Printf("Warning: Failed to enable ACME: %v\n", err)
		} else {
			// Issue certificate for all hosts in the ingress
			var domains []string
			for _, rule := range ingress.Rules {
				if rule.Host != "" {
					domains = append(domains, rule.Host)
				}
			}
			if len(domains) > 0 {
				// Issue certificate asynchronously to avoid blocking ingress creation
				go func() {
					if err := s.manager.IssueACMECertificate(domains); err != nil {
						fmt.Printf("Warning: Failed to issue ACME certificate: %v\n", err)
					}
				}()
			}
		}
	}

	// Convert back to proto
	protoIngress := convertIngressToProto(ingress)

	return &proto.CreateIngressResponse{
		Ingress: protoIngress,
	}, nil
}

// UpdateIngress updates an existing ingress
func (s *Server) UpdateIngress(ctx context.Context, req *proto.UpdateIngressRequest) (*proto.UpdateIngressResponse, error) {
	// Forward to leader if not leader
	if !s.manager.IsLeader() {
		return nil, fmt.Errorf("not leader, forward to leader")
	}

	// Validate request
	if req.Id == "" && req.Name == "" {
		return nil, fmt.Errorf("ingress ID or name is required")
	}

	// Get existing ingress
	var existing *types.Ingress
	var err error
	if req.Id != "" {
		existing, err = s.manager.GetIngress(req.Id)
	} else {
		existing, err = s.manager.GetIngressByName(req.Name)
	}
	if err != nil {
		return nil, fmt.Errorf("ingress not found: %w", err)
	}

	// Update fields
	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.Rules != nil {
		existing.Rules = convertProtoIngressRules(req.Rules)
	}
	if req.Tls != nil {
		existing.TLS = convertProtoIngressTLS(req.Tls)
	}
	if req.Labels != nil {
		existing.Labels = req.Labels
	}
	existing.UpdatedAt = time.Now()

	// Update ingress in storage via Raft
	if err := s.manager.UpdateIngress(existing); err != nil {
		return nil, fmt.Errorf("failed to update ingress: %w", err)
	}

	// Reload ingress proxy to pick up the changes
	if err := s.manager.ReloadIngress(); err != nil {
		// Log but don't fail the request
		fmt.Printf("Warning: Failed to reload ingress proxy: %v\n", err)
	}

	// Convert back to proto
	protoIngress := convertIngressToProto(existing)

	return &proto.UpdateIngressResponse{
		Ingress: protoIngress,
	}, nil
}

// DeleteIngress deletes an ingress
func (s *Server) DeleteIngress(ctx context.Context, req *proto.DeleteIngressRequest) (*proto.DeleteIngressResponse, error) {
	// Forward to leader if not leader
	if !s.manager.IsLeader() {
		return nil, fmt.Errorf("not leader, forward to leader")
	}

	// Validate request
	if req.Id == "" && req.Name == "" {
		return nil, fmt.Errorf("ingress ID or name is required")
	}

	// Get ingress to delete
	var ingress *types.Ingress
	var err error
	if req.Id != "" {
		ingress, err = s.manager.GetIngress(req.Id)
	} else {
		ingress, err = s.manager.GetIngressByName(req.Name)
	}
	if err != nil {
		return nil, fmt.Errorf("ingress not found: %w", err)
	}

	// Delete ingress via Raft
	if err := s.manager.DeleteIngress(ingress.ID); err != nil {
		return nil, fmt.Errorf("failed to delete ingress: %w", err)
	}

	// Reload ingress proxy to remove the deleted ingress
	if err := s.manager.ReloadIngress(); err != nil {
		// Log but don't fail the request
		fmt.Printf("Warning: Failed to reload ingress proxy: %v\n", err)
	}

	return &proto.DeleteIngressResponse{
		Status: "deleted",
	}, nil
}

// GetIngress retrieves an ingress
func (s *Server) GetIngress(ctx context.Context, req *proto.GetIngressRequest) (*proto.GetIngressResponse, error) {
	// Validate request
	if req.Id == "" && req.Name == "" {
		return nil, fmt.Errorf("ingress ID or name is required")
	}

	// Get ingress from storage
	var ingress *types.Ingress
	var err error
	if req.Id != "" {
		ingress, err = s.manager.GetIngress(req.Id)
	} else {
		ingress, err = s.manager.GetIngressByName(req.Name)
	}
	if err != nil {
		return nil, fmt.Errorf("ingress not found: %w", err)
	}

	// Convert to proto
	protoIngress := convertIngressToProto(ingress)

	return &proto.GetIngressResponse{
		Ingress: protoIngress,
	}, nil
}

// ListIngresses lists all ingresses
func (s *Server) ListIngresses(ctx context.Context, req *proto.ListIngressesRequest) (*proto.ListIngressesResponse, error) {
	// Get all ingresses from storage
	ingresses, err := s.manager.ListIngresses()
	if err != nil {
		return nil, fmt.Errorf("failed to list ingresses: %w", err)
	}

	// Convert to proto
	protoIngresses := make([]*proto.Ingress, len(ingresses))
	for i, ingress := range ingresses {
		protoIngresses[i] = convertIngressToProto(ingress)
	}

	return &proto.ListIngressesResponse{
		Ingresses: protoIngresses,
	}, nil
}

// Helper functions for Ingress conversion

func convertIngressToProto(ingress *types.Ingress) *proto.Ingress {
	if ingress == nil {
		return nil
	}

	protoRules := make([]*proto.IngressRule, len(ingress.Rules))
	for i, rule := range ingress.Rules {
		protoPaths := make([]*proto.IngressPath, len(rule.Paths))
		for j, path := range rule.Paths {
			protoPaths[j] = &proto.IngressPath{
				Path:     path.Path,
				PathType: string(path.PathType),
				Backend: &proto.IngressBackend{
					ServiceName: path.Backend.ServiceName,
					Port:        int32(path.Backend.Port),
				},
			}
		}
		protoRules[i] = &proto.IngressRule{
			Host:  rule.Host,
			Paths: protoPaths,
		}
	}

	var protoTLS *proto.IngressTLS
	if ingress.TLS != nil {
		protoTLS = &proto.IngressTLS{
			Enabled:    ingress.TLS.Enabled,
			SecretName: ingress.TLS.SecretName,
			Hosts:      ingress.TLS.Hosts,
			AutoTls:    ingress.TLS.AutoTLS,
			Email:      ingress.TLS.Email,
		}
	}

	return &proto.Ingress{
		Id:        ingress.ID,
		Name:      ingress.Name,
		Rules:     protoRules,
		Tls:       protoTLS,
		Labels:    ingress.Labels,
		CreatedAt: timestamppb.New(ingress.CreatedAt),
		UpdatedAt: timestamppb.New(ingress.UpdatedAt),
	}
}

func convertProtoIngressRules(protoRules []*proto.IngressRule) []*types.IngressRule {
	rules := make([]*types.IngressRule, len(protoRules))
	for i, protoRule := range protoRules {
		paths := make([]*types.IngressPath, len(protoRule.Paths))
		for j, protoPath := range protoRule.Paths {
			paths[j] = &types.IngressPath{
				Path:     protoPath.Path,
				PathType: types.PathType(protoPath.PathType),
				Backend: &types.IngressBackend{
					ServiceName: protoPath.Backend.ServiceName,
					Port:        int(protoPath.Backend.Port),
				},
			}
		}
		rules[i] = &types.IngressRule{
			Host:  protoRule.Host,
			Paths: paths,
		}
	}
	return rules
}

func convertProtoIngressTLS(protoTLS *proto.IngressTLS) *types.IngressTLS {
	if protoTLS == nil {
		return nil
	}
	return &types.IngressTLS{
		Enabled:    protoTLS.Enabled,
		SecretName: protoTLS.SecretName,
		Hosts:      protoTLS.Hosts,
		AutoTLS:    protoTLS.AutoTls,
		Email:      protoTLS.Email,
	}
}

// --- TLS Certificate Handlers ---

// CreateTLSCertificate creates a new TLS certificate
func (s *Server) CreateTLSCertificate(ctx context.Context, req *proto.CreateTLSCertificateRequest) (*proto.CreateTLSCertificateResponse, error) {
	// Forward to leader if not leader
	if !s.manager.IsLeader() {
		return nil, fmt.Errorf("not leader, forward to leader")
	}

	// Validate request
	if req.Name == "" {
		return nil, fmt.Errorf("certificate name is required")
	}
	if len(req.Hosts) == 0 {
		return nil, fmt.Errorf("at least one host is required")
	}
	if len(req.CertPem) == 0 || len(req.KeyPem) == 0 {
		return nil, fmt.Errorf("certificate and key are required")
	}

	// Check if certificate already exists
	existing, _ := s.manager.GetTLSCertificateByName(req.Name)
	if existing != nil {
		return nil, fmt.Errorf("certificate %s already exists", req.Name)
	}

	// Parse certificate to extract metadata
	certPEM := req.CertPem
	cert, err := parseCertificate(certPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Create TLS certificate
	tlsCert := &types.TLSCertificate{
		ID:        uuid.New().String(),
		Name:      req.Name,
		Hosts:     req.Hosts,
		CertPEM:   certPEM,
		KeyPEM:    req.KeyPem,
		Issuer:    cert.Issuer.CommonName,
		NotBefore: cert.NotBefore,
		NotAfter:  cert.NotAfter,
		AutoRenew: false, // M7.3 feature
		Labels:    req.Labels,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Create certificate in storage via Raft
	if err := s.manager.CreateTLSCertificate(tlsCert); err != nil {
		return nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	// Convert back to proto
	protoCert := convertTLSCertificateToProto(tlsCert)

	return &proto.CreateTLSCertificateResponse{
		Certificate: protoCert,
	}, nil
}

// GetTLSCertificate retrieves a TLS certificate
func (s *Server) GetTLSCertificate(ctx context.Context, req *proto.GetTLSCertificateRequest) (*proto.GetTLSCertificateResponse, error) {
	var cert *types.TLSCertificate
	var err error

	if req.Id != "" {
		cert, err = s.manager.GetTLSCertificate(req.Id)
	} else if req.Name != "" {
		cert, err = s.manager.GetTLSCertificateByName(req.Name)
	} else {
		return nil, fmt.Errorf("id or name is required")
	}

	if err != nil {
		return nil, fmt.Errorf("certificate not found: %w", err)
	}

	return &proto.GetTLSCertificateResponse{
		Certificate: convertTLSCertificateToProto(cert),
	}, nil
}

// ListTLSCertificates lists all TLS certificates
func (s *Server) ListTLSCertificates(ctx context.Context, req *proto.ListTLSCertificatesRequest) (*proto.ListTLSCertificatesResponse, error) {
	// Get all certificates from storage
	certs, err := s.manager.ListTLSCertificates()
	if err != nil {
		return nil, fmt.Errorf("failed to list certificates: %w", err)
	}

	// Convert to proto (but exclude private keys for security)
	protoCerts := make([]*proto.TLSCertificate, len(certs))
	for i, cert := range certs {
		protoCert := convertTLSCertificateToProto(cert)
		// Clear private key from response for security
		protoCert.KeyPem = nil
		protoCerts[i] = protoCert
	}

	return &proto.ListTLSCertificatesResponse{
		Certificates: protoCerts,
	}, nil
}

// DeleteTLSCertificate deletes a TLS certificate
func (s *Server) DeleteTLSCertificate(ctx context.Context, req *proto.DeleteTLSCertificateRequest) (*proto.DeleteTLSCertificateResponse, error) {
	// Forward to leader if not leader
	if !s.manager.IsLeader() {
		return nil, fmt.Errorf("not leader, forward to leader")
	}

	// Get certificate to delete
	var cert *types.TLSCertificate
	var err error

	if req.Id != "" {
		cert, err = s.manager.GetTLSCertificate(req.Id)
	} else if req.Name != "" {
		cert, err = s.manager.GetTLSCertificateByName(req.Name)
	} else {
		return nil, fmt.Errorf("id or name is required")
	}

	if err != nil {
		return nil, fmt.Errorf("certificate not found: %w", err)
	}

	// Delete certificate via Raft
	if err := s.manager.DeleteTLSCertificate(cert.ID); err != nil {
		return nil, fmt.Errorf("failed to delete certificate: %w", err)
	}

	return &proto.DeleteTLSCertificateResponse{
		Status: "deleted",
	}, nil
}

// Helper functions for TLS Certificate conversion

func convertTLSCertificateToProto(cert *types.TLSCertificate) *proto.TLSCertificate {
	return &proto.TLSCertificate{
		Id:        cert.ID,
		Name:      cert.Name,
		Hosts:     cert.Hosts,
		CertPem:   cert.CertPEM,
		KeyPem:    cert.KeyPEM,
		Issuer:    cert.Issuer,
		NotBefore: timestamppb.New(cert.NotBefore),
		NotAfter:  timestamppb.New(cert.NotAfter),
		AutoRenew: cert.AutoRenew,
		Labels:    cert.Labels,
		CreatedAt: timestamppb.New(cert.CreatedAt),
		UpdatedAt: timestamppb.New(cert.UpdatedAt),
	}
}

func parseCertificate(certPEM []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}
	return x509.ParseCertificate(block.Bytes)
}
