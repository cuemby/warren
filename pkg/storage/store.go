package storage

import (
	"github.com/cuemby/warren/pkg/types"
)

// Store defines the interface for cluster state storage
// This will be implemented by BoltDB-backed storage
type Store interface {
	// Nodes
	CreateNode(node *types.Node) error
	GetNode(id string) (*types.Node, error)
	ListNodes() ([]*types.Node, error)
	UpdateNode(node *types.Node) error
	DeleteNode(id string) error

	// Services
	CreateService(service *types.Service) error
	GetService(id string) (*types.Service, error)
	GetServiceByName(name string) (*types.Service, error)
	ListServices() ([]*types.Service, error)
	UpdateService(service *types.Service) error
	DeleteService(id string) error

	// Tasks
	CreateTask(task *types.Task) error
	GetTask(id string) (*types.Task, error)
	ListTasks() ([]*types.Task, error)
	ListTasksByService(serviceID string) ([]*types.Task, error)
	ListTasksByNode(nodeID string) ([]*types.Task, error)
	UpdateTask(task *types.Task) error
	DeleteTask(id string) error

	// Secrets
	CreateSecret(secret *types.Secret) error
	GetSecret(id string) (*types.Secret, error)
	GetSecretByName(name string) (*types.Secret, error)
	ListSecrets() ([]*types.Secret, error)
	DeleteSecret(id string) error

	// Volumes
	CreateVolume(volume *types.Volume) error
	GetVolume(id string) (*types.Volume, error)
	GetVolumeByName(name string) (*types.Volume, error)
	ListVolumes() ([]*types.Volume, error)
	DeleteVolume(id string) error

	// Networks
	CreateNetwork(network *types.Network) error
	GetNetwork(id string) (*types.Network, error)
	ListNetworks() ([]*types.Network, error)
	DeleteNetwork(id string) error

	// Utility
	Close() error
}
