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

	// Containers
	CreateContainer(container *types.Container) error
	GetContainer(id string) (*types.Container, error)
	ListContainers() ([]*types.Container, error)
	ListContainersByService(serviceID string) ([]*types.Container, error)
	ListContainersByNode(nodeID string) ([]*types.Container, error)
	UpdateContainer(container *types.Container) error
	DeleteContainer(id string) error

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

	// Certificate Authority
	SaveCA(data []byte) error
	GetCA() ([]byte, error)

	// Ingresses
	CreateIngress(ingress *types.Ingress) error
	GetIngress(id string) (*types.Ingress, error)
	GetIngressByName(name string) (*types.Ingress, error)
	ListIngresses() ([]*types.Ingress, error)
	UpdateIngress(ingress *types.Ingress) error
	DeleteIngress(id string) error

	// TLS Certificates
	CreateTLSCertificate(cert *types.TLSCertificate) error
	GetTLSCertificate(id string) (*types.TLSCertificate, error)
	GetTLSCertificateByName(name string) (*types.TLSCertificate, error)
	GetTLSCertificatesByHost(host string) ([]*types.TLSCertificate, error)
	ListTLSCertificates() ([]*types.TLSCertificate, error)
	UpdateTLSCertificate(cert *types.TLSCertificate) error
	DeleteTLSCertificate(id string) error

	// Utility
	Close() error
}
