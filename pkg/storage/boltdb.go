package storage

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/cuemby/warren/pkg/types"
	bolt "go.etcd.io/bbolt"
)

var (
	// Bucket names
	bucketNodes           = []byte("nodes")
	bucketServices        = []byte("services")
	bucketContainers      = []byte("containers")
	bucketSecrets         = []byte("secrets")
	bucketVolumes         = []byte("volumes")
	bucketNetworks        = []byte("networks")
	bucketCA              = []byte("ca")
	bucketIngresses       = []byte("ingresses")
	bucketTLSCertificates = []byte("tls_certificates")
)

// BoltStore implements Store interface using BoltDB
type BoltStore struct {
	db *bolt.DB
}

// NewBoltStore creates a new BoltDB-backed store
func NewBoltStore(dataDir string) (*BoltStore, error) {
	dbPath := filepath.Join(dataDir, "warren.db")

	db, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Create buckets
	err = db.Update(func(tx *bolt.Tx) error {
		buckets := [][]byte{
			bucketNodes,
			bucketServices,
			bucketContainers,
			bucketSecrets,
			bucketVolumes,
			bucketNetworks,
			bucketCA,
			bucketIngresses,
			bucketTLSCertificates,
		}

		for _, bucket := range buckets {
			if _, err := tx.CreateBucketIfNotExists(bucket); err != nil {
				return fmt.Errorf("failed to create bucket %s: %w", bucket, err)
			}
		}
		return nil
	})

	if err != nil {
		db.Close()
		return nil, err
	}

	return &BoltStore{db: db}, nil
}

// Close closes the database
func (s *BoltStore) Close() error {
	return s.db.Close()
}

// Node operations
func (s *BoltStore) CreateNode(node *types.Node) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketNodes)
		data, err := json.Marshal(node)
		if err != nil {
			return err
		}
		return b.Put([]byte(node.ID), data)
	})
}

func (s *BoltStore) GetNode(id string) (*types.Node, error) {
	var node types.Node
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketNodes)
		data := b.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("node not found: %s", id)
		}
		return json.Unmarshal(data, &node)
	})
	return &node, err
}

func (s *BoltStore) ListNodes() ([]*types.Node, error) {
	var nodes []*types.Node
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketNodes)
		return b.ForEach(func(k, v []byte) error {
			var node types.Node
			if err := json.Unmarshal(v, &node); err != nil {
				return err
			}
			nodes = append(nodes, &node)
			return nil
		})
	})
	return nodes, err
}

func (s *BoltStore) UpdateNode(node *types.Node) error {
	return s.CreateNode(node) // Same as create (upsert)
}

func (s *BoltStore) DeleteNode(id string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketNodes)
		return b.Delete([]byte(id))
	})
}

// Service operations
func (s *BoltStore) CreateService(service *types.Service) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketServices)
		data, err := json.Marshal(service)
		if err != nil {
			return err
		}
		return b.Put([]byte(service.ID), data)
	})
}

func (s *BoltStore) GetService(id string) (*types.Service, error) {
	var service types.Service
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketServices)
		data := b.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("service not found: %s", id)
		}
		return json.Unmarshal(data, &service)
	})
	return &service, err
}

func (s *BoltStore) GetServiceByName(name string) (*types.Service, error) {
	var found *types.Service
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketServices)
		return b.ForEach(func(k, v []byte) error {
			var service types.Service
			if err := json.Unmarshal(v, &service); err != nil {
				return err
			}
			if service.Name == name {
				found = &service
				return nil
			}
			return nil
		})
	})
	if found == nil {
		return nil, fmt.Errorf("service not found: %s", name)
	}
	return found, err
}

func (s *BoltStore) ListServices() ([]*types.Service, error) {
	var services []*types.Service
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketServices)
		return b.ForEach(func(k, v []byte) error {
			var service types.Service
			if err := json.Unmarshal(v, &service); err != nil {
				return err
			}
			services = append(services, &service)
			return nil
		})
	})
	return services, err
}

func (s *BoltStore) UpdateService(service *types.Service) error {
	return s.CreateService(service)
}

func (s *BoltStore) DeleteService(id string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketServices)
		return b.Delete([]byte(id))
	})
}

// Container operations
func (s *BoltStore) CreateContainer(container *types.Container) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketContainers)
		data, err := json.Marshal(container)
		if err != nil {
			return err
		}
		return b.Put([]byte(container.ID), data)
	})
}

func (s *BoltStore) GetContainer(id string) (*types.Container, error) {
	var container types.Container
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketContainers)
		data := b.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("container not found: %s", id)
		}
		return json.Unmarshal(data, &container)
	})
	return &container, err
}

func (s *BoltStore) ListContainers() ([]*types.Container, error) {
	var containers []*types.Container
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketContainers)
		return b.ForEach(func(k, v []byte) error {
			var container types.Container
			if err := json.Unmarshal(v, &container); err != nil {
				return err
			}
			containers = append(containers, &container)
			return nil
		})
	})
	return containers, err
}

func (s *BoltStore) ListContainersByService(serviceID string) ([]*types.Container, error) {
	containers, err := s.ListContainers()
	if err != nil {
		return nil, err
	}

	var filtered []*types.Container
	for _, container := range containers {
		if container.ServiceID == serviceID {
			filtered = append(filtered, container)
		}
	}
	return filtered, nil
}

func (s *BoltStore) ListContainersByNode(nodeID string) ([]*types.Container, error) {
	containers, err := s.ListContainers()
	if err != nil {
		return nil, err
	}

	var filtered []*types.Container
	for _, container := range containers {
		if container.NodeID == nodeID {
			filtered = append(filtered, container)
		}
	}
	return filtered, nil
}

func (s *BoltStore) UpdateContainer(container *types.Container) error {
	return s.CreateContainer(container)
}

func (s *BoltStore) DeleteContainer(id string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketContainers)
		return b.Delete([]byte(id))
	})
}

// Secret operations
func (s *BoltStore) CreateSecret(secret *types.Secret) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketSecrets)
		data, err := json.Marshal(secret)
		if err != nil {
			return err
		}
		return b.Put([]byte(secret.ID), data)
	})
}

func (s *BoltStore) GetSecret(id string) (*types.Secret, error) {
	var secret types.Secret
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketSecrets)
		data := b.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("secret not found: %s", id)
		}
		return json.Unmarshal(data, &secret)
	})
	return &secret, err
}

func (s *BoltStore) GetSecretByName(name string) (*types.Secret, error) {
	var found *types.Secret
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketSecrets)
		return b.ForEach(func(k, v []byte) error {
			var secret types.Secret
			if err := json.Unmarshal(v, &secret); err != nil {
				return err
			}
			if secret.Name == name {
				found = &secret
				return nil
			}
			return nil
		})
	})
	if found == nil {
		return nil, fmt.Errorf("secret not found: %s", name)
	}
	return found, err
}

func (s *BoltStore) ListSecrets() ([]*types.Secret, error) {
	var secrets []*types.Secret
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketSecrets)
		return b.ForEach(func(k, v []byte) error {
			var secret types.Secret
			if err := json.Unmarshal(v, &secret); err != nil {
				return err
			}
			secrets = append(secrets, &secret)
			return nil
		})
	})
	return secrets, err
}

func (s *BoltStore) DeleteSecret(id string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketSecrets)
		return b.Delete([]byte(id))
	})
}

// Volume operations
func (s *BoltStore) CreateVolume(volume *types.Volume) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketVolumes)
		data, err := json.Marshal(volume)
		if err != nil {
			return err
		}
		return b.Put([]byte(volume.ID), data)
	})
}

func (s *BoltStore) GetVolume(id string) (*types.Volume, error) {
	var volume types.Volume
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketVolumes)
		data := b.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("volume not found: %s", id)
		}
		return json.Unmarshal(data, &volume)
	})
	return &volume, err
}

func (s *BoltStore) GetVolumeByName(name string) (*types.Volume, error) {
	var found *types.Volume
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketVolumes)
		return b.ForEach(func(k, v []byte) error {
			var volume types.Volume
			if err := json.Unmarshal(v, &volume); err != nil {
				return err
			}
			if volume.Name == name {
				found = &volume
				return nil
			}
			return nil
		})
	})
	if found == nil {
		return nil, fmt.Errorf("volume not found: %s", name)
	}
	return found, err
}

func (s *BoltStore) ListVolumes() ([]*types.Volume, error) {
	var volumes []*types.Volume
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketVolumes)
		return b.ForEach(func(k, v []byte) error {
			var volume types.Volume
			if err := json.Unmarshal(v, &volume); err != nil {
				return err
			}
			volumes = append(volumes, &volume)
			return nil
		})
	})
	return volumes, err
}

func (s *BoltStore) DeleteVolume(id string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketVolumes)
		return b.Delete([]byte(id))
	})
}

// Network operations
func (s *BoltStore) CreateNetwork(network *types.Network) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketNetworks)
		data, err := json.Marshal(network)
		if err != nil {
			return err
		}
		return b.Put([]byte(network.ID), data)
	})
}

func (s *BoltStore) GetNetwork(id string) (*types.Network, error) {
	var network types.Network
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketNetworks)
		data := b.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("network not found: %s", id)
		}
		return json.Unmarshal(data, &network)
	})
	return &network, err
}

func (s *BoltStore) ListNetworks() ([]*types.Network, error) {
	var networks []*types.Network
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketNetworks)
		return b.ForEach(func(k, v []byte) error {
			var network types.Network
			if err := json.Unmarshal(v, &network); err != nil {
				return err
			}
			networks = append(networks, &network)
			return nil
		})
	})
	return networks, err
}

func (s *BoltStore) DeleteNetwork(id string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketNetworks)
		return b.Delete([]byte(id))
	})
}

// Certificate Authority operations
func (s *BoltStore) SaveCA(data []byte) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketCA)
		// Use fixed key "ca" for the CA data
		return b.Put([]byte("ca"), data)
	})
}

func (s *BoltStore) GetCA() ([]byte, error) {
	var data []byte
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketCA)
		data = b.Get([]byte("ca"))
		if data == nil {
			return fmt.Errorf("CA not found")
		}
		// Make a copy since BoltDB data is only valid during the transaction
		dataCopy := make([]byte, len(data))
		copy(dataCopy, data)
		data = dataCopy
		return nil
	})
	return data, err
}

// --- Ingress Operations ---

// CreateIngress creates a new ingress
func (s *BoltStore) CreateIngress(ingress *types.Ingress) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketIngresses)
		data, err := json.Marshal(ingress)
		if err != nil {
			return err
		}
		return b.Put([]byte(ingress.ID), data)
	})
}

// GetIngress retrieves an ingress by ID
func (s *BoltStore) GetIngress(id string) (*types.Ingress, error) {
	var ingress *types.Ingress
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketIngresses)
		data := b.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("ingress not found: %s", id)
		}
		return json.Unmarshal(data, &ingress)
	})
	return ingress, err
}

// GetIngressByName retrieves an ingress by name
func (s *BoltStore) GetIngressByName(name string) (*types.Ingress, error) {
	var result *types.Ingress
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketIngresses)
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var ingress types.Ingress
			if err := json.Unmarshal(v, &ingress); err != nil {
				continue
			}
			if ingress.Name == name {
				result = &ingress
				return nil
			}
		}
		return fmt.Errorf("ingress not found: %s", name)
	})
	return result, err
}

// ListIngresses returns all ingresses
func (s *BoltStore) ListIngresses() ([]*types.Ingress, error) {
	var ingresses []*types.Ingress
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketIngresses)
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var ingress types.Ingress
			if err := json.Unmarshal(v, &ingress); err != nil {
				return err
			}
			ingresses = append(ingresses, &ingress)
		}
		return nil
	})
	return ingresses, err
}

// UpdateIngress updates an existing ingress
func (s *BoltStore) UpdateIngress(ingress *types.Ingress) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketIngresses)
		data, err := json.Marshal(ingress)
		if err != nil {
			return err
		}
		return b.Put([]byte(ingress.ID), data)
	})
}

// DeleteIngress deletes an ingress
func (s *BoltStore) DeleteIngress(id string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketIngresses)
		return b.Delete([]byte(id))
	})
}

// --- TLS Certificates ---

// CreateTLSCertificate creates a new TLS certificate
func (s *BoltStore) CreateTLSCertificate(cert *types.TLSCertificate) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketTLSCertificates)
		data, err := json.Marshal(cert)
		if err != nil {
			return err
		}
		return b.Put([]byte(cert.ID), data)
	})
}

// GetTLSCertificate retrieves a TLS certificate by ID
func (s *BoltStore) GetTLSCertificate(id string) (*types.TLSCertificate, error) {
	var cert types.TLSCertificate
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketTLSCertificates)
		data := b.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("certificate not found")
		}
		return json.Unmarshal(data, &cert)
	})
	if err != nil {
		return nil, err
	}
	return &cert, nil
}

// GetTLSCertificateByName retrieves a TLS certificate by name
func (s *BoltStore) GetTLSCertificateByName(name string) (*types.TLSCertificate, error) {
	var cert *types.TLSCertificate
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketTLSCertificates)
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var c types.TLSCertificate
			if err := json.Unmarshal(v, &c); err != nil {
				continue
			}
			if c.Name == name {
				cert = &c
				return nil
			}
		}
		return fmt.Errorf("certificate not found")
	})
	return cert, err
}

// GetTLSCertificatesByHost retrieves all TLS certificates that cover a specific host
func (s *BoltStore) GetTLSCertificatesByHost(host string) ([]*types.TLSCertificate, error) {
	var certs []*types.TLSCertificate
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketTLSCertificates)
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var cert types.TLSCertificate
			if err := json.Unmarshal(v, &cert); err != nil {
				continue
			}
			// Check if this cert covers the requested host
			for _, h := range cert.Hosts {
				if h == host || matchWildcard(h, host) {
					certs = append(certs, &cert)
					break
				}
			}
		}
		return nil
	})
	return certs, err
}

// matchWildcard checks if a wildcard pattern matches a host
func matchWildcard(pattern, host string) bool {
	// Simple wildcard matching: *.example.com matches foo.example.com
	if !strings.HasPrefix(pattern, "*.") {
		return false
	}
	suffix := pattern[1:] // Remove * to get .example.com
	return strings.HasSuffix(host, suffix)
}

// ListTLSCertificates lists all TLS certificates
func (s *BoltStore) ListTLSCertificates() ([]*types.TLSCertificate, error) {
	var certs []*types.TLSCertificate
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketTLSCertificates)
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var cert types.TLSCertificate
			if err := json.Unmarshal(v, &cert); err != nil {
				continue
			}
			certs = append(certs, &cert)
		}
		return nil
	})
	return certs, err
}

// UpdateTLSCertificate updates an existing TLS certificate
func (s *BoltStore) UpdateTLSCertificate(cert *types.TLSCertificate) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketTLSCertificates)
		data, err := json.Marshal(cert)
		if err != nil {
			return err
		}
		return b.Put([]byte(cert.ID), data)
	})
}

// DeleteTLSCertificate deletes a TLS certificate
func (s *BoltStore) DeleteTLSCertificate(id string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketTLSCertificates)
		return b.Delete([]byte(id))
	})
}
