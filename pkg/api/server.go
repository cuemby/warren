package api

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/cuemby/warren/api/proto"
	"github.com/cuemby/warren/pkg/manager"
	"github.com/cuemby/warren/pkg/types"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Server implements the WarrenAPI gRPC service
type Server struct {
	proto.UnimplementedWarrenAPIServer
	manager *manager.Manager
	grpc    *grpc.Server
}

// NewServer creates a new API server
func NewServer(mgr *manager.Manager) *Server {
	return &Server{
		manager: mgr,
		grpc:    grpc.NewServer(),
	}
}

// Start starts the gRPC server
func (s *Server) Start(addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
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
		return nil, fmt.Errorf("failed to create node: %v", err)
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
		return nil, fmt.Errorf("node not found: %v", err)
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
		return nil, fmt.Errorf("failed to update node: %v", err)
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
		return nil, fmt.Errorf("failed to list nodes: %v", err)
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
		return nil, fmt.Errorf("node not found: %v", err)
	}

	return &proto.GetNodeResponse{
		Node: nodeToProto(node),
	}, nil
}

// RemoveNode removes a node from the cluster
func (s *Server) RemoveNode(ctx context.Context, req *proto.RemoveNodeRequest) (*proto.RemoveNodeResponse, error) {
	if err := s.manager.DeleteNode(req.Id); err != nil {
		return nil, fmt.Errorf("failed to remove node: %v", err)
	}

	return &proto.RemoveNodeResponse{
		Status: "ok",
	}, nil
}

// CreateService creates a new service
func (s *Server) CreateService(ctx context.Context, req *proto.CreateServiceRequest) (*proto.CreateServiceResponse, error) {
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
		service.HealthCheck = &types.HealthCheck{
			Type:     types.HealthCheckType(req.HealthCheck.Type),
			Endpoint: req.HealthCheck.Endpoint,
			Interval: time.Duration(req.HealthCheck.IntervalSeconds) * time.Second,
			Timeout:  time.Duration(req.HealthCheck.TimeoutSeconds) * time.Second,
			Retries:  int(req.HealthCheck.Retries),
		}
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

	if err := s.manager.CreateService(service); err != nil {
		return nil, fmt.Errorf("failed to create service: %v", err)
	}

	return &proto.CreateServiceResponse{
		Service: serviceToProto(service),
	}, nil
}

// UpdateService updates an existing service
func (s *Server) UpdateService(ctx context.Context, req *proto.UpdateServiceRequest) (*proto.UpdateServiceResponse, error) {
	service, err := s.manager.GetService(req.Id)
	if err != nil {
		return nil, fmt.Errorf("service not found: %v", err)
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
		return nil, fmt.Errorf("failed to update service: %v", err)
	}

	return &proto.UpdateServiceResponse{
		Service: serviceToProto(service),
	}, nil
}

// DeleteService deletes a service
func (s *Server) DeleteService(ctx context.Context, req *proto.DeleteServiceRequest) (*proto.DeleteServiceResponse, error) {
	if err := s.manager.DeleteService(req.Id); err != nil {
		return nil, fmt.Errorf("failed to delete service: %v", err)
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
		return nil, fmt.Errorf("service not found: %v", err)
	}

	return &proto.GetServiceResponse{
		Service: serviceToProto(service),
	}, nil
}

// ListServices returns all services
func (s *Server) ListServices(ctx context.Context, req *proto.ListServicesRequest) (*proto.ListServicesResponse, error) {
	services, err := s.manager.ListServices()
	if err != nil {
		return nil, fmt.Errorf("failed to list services: %v", err)
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
		return nil, fmt.Errorf("task not found: %v", err)
	}

	task.ActualState = types.TaskState(req.ActualState)
	task.ContainerID = req.ContainerId

	if err := s.manager.UpdateTask(task); err != nil {
		return nil, fmt.Errorf("failed to update task: %v", err)
	}

	return &proto.UpdateTaskStatusResponse{
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
		return nil, fmt.Errorf("failed to list tasks: %v", err)
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
		return nil, fmt.Errorf("task not found: %v", err)
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
	secret := &types.Secret{
		ID:        uuid.New().String(),
		Name:      req.Name,
		Data:      req.Data,
		CreatedAt: time.Now(),
	}

	if err := s.manager.CreateSecret(secret); err != nil {
		return nil, fmt.Errorf("failed to create secret: %v", err)
	}

	return &proto.CreateSecretResponse{
		Secret: secretToProto(secret),
	}, nil
}

// DeleteSecret deletes a secret
func (s *Server) DeleteSecret(ctx context.Context, req *proto.DeleteSecretRequest) (*proto.DeleteSecretResponse, error) {
	if err := s.manager.DeleteSecret(req.Id); err != nil {
		return nil, fmt.Errorf("failed to delete secret: %v", err)
	}

	return &proto.DeleteSecretResponse{
		Status: "ok",
	}, nil
}

// ListSecrets returns all secrets (without data)
func (s *Server) ListSecrets(ctx context.Context, req *proto.ListSecretsRequest) (*proto.ListSecretsResponse, error) {
	secrets, err := s.manager.ListSecrets()
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets: %v", err)
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
	volume := &types.Volume{
		ID:        uuid.New().String(),
		Name:      req.Name,
		Driver:    req.Driver,
		Options:   req.DriverOpts,
		CreatedAt: time.Now(),
	}

	if err := s.manager.CreateVolume(volume); err != nil {
		return nil, fmt.Errorf("failed to create volume: %v", err)
	}

	return &proto.CreateVolumeResponse{
		Volume: volumeToProto(volume),
	}, nil
}

// DeleteVolume deletes a volume
func (s *Server) DeleteVolume(ctx context.Context, req *proto.DeleteVolumeRequest) (*proto.DeleteVolumeResponse, error) {
	if err := s.manager.DeleteVolume(req.Id); err != nil {
		return nil, fmt.Errorf("failed to delete volume: %v", err)
	}

	return &proto.DeleteVolumeResponse{
		Status: "ok",
	}, nil
}

// ListVolumes returns all volumes
func (s *Server) ListVolumes(ctx context.Context, req *proto.ListVolumesRequest) (*proto.ListVolumesResponse, error) {
	volumes, err := s.manager.ListVolumes()
	if err != nil {
		return nil, fmt.Errorf("failed to list volumes: %v", err)
	}

	protoVolumes := make([]*proto.Volume, len(volumes))
	for i, vol := range volumes {
		protoVolumes[i] = volumeToProto(vol)
	}

	return &proto.ListVolumesResponse{
		Volumes: protoVolumes,
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
		ps.HealthCheck = &proto.HealthCheck{
			Type:            string(s.HealthCheck.Type),
			Endpoint:        s.HealthCheck.Endpoint,
			IntervalSeconds: int32(s.HealthCheck.Interval / time.Second),
			TimeoutSeconds:  int32(s.HealthCheck.Timeout / time.Second),
			Retries:         int32(s.HealthCheck.Retries),
		}
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
		pt.HealthCheck = &proto.HealthCheck{
			Type:            string(t.HealthCheck.Type),
			Endpoint:        t.HealthCheck.Endpoint,
			IntervalSeconds: int32(t.HealthCheck.Interval / time.Second),
			TimeoutSeconds:  int32(t.HealthCheck.Timeout / time.Second),
			Retries:         int32(t.HealthCheck.Retries),
		}
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
		// Note: Data is never exposed via API
	}
}

func volumeToProto(v *types.Volume) *proto.Volume {
	return &proto.Volume{
		Id:         v.ID,
		Name:       v.Name,
		Driver:     v.Driver,
		DriverOpts: v.Options,
		Labels:     make(map[string]string), // Volume doesn't have labels in types
		CreatedAt:  timestamppb.New(v.CreatedAt),
	}
}
