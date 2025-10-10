package storage

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/cuemby/warren/pkg/types"
	bolt "go.etcd.io/bbolt"
)

var (
	// Bucket names
	bucketNodes    = []byte("nodes")
	bucketServices = []byte("services")
	bucketTasks    = []byte("tasks")
	bucketSecrets  = []byte("secrets")
	bucketVolumes  = []byte("volumes")
	bucketNetworks = []byte("networks")
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
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	// Create buckets
	err = db.Update(func(tx *bolt.Tx) error {
		buckets := [][]byte{
			bucketNodes,
			bucketServices,
			bucketTasks,
			bucketSecrets,
			bucketVolumes,
			bucketNetworks,
		}

		for _, bucket := range buckets {
			if _, err := tx.CreateBucketIfNotExists(bucket); err != nil {
				return fmt.Errorf("failed to create bucket %s: %v", bucket, err)
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

// Task operations
func (s *BoltStore) CreateTask(task *types.Task) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketTasks)
		data, err := json.Marshal(task)
		if err != nil {
			return err
		}
		return b.Put([]byte(task.ID), data)
	})
}

func (s *BoltStore) GetTask(id string) (*types.Task, error) {
	var task types.Task
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketTasks)
		data := b.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("task not found: %s", id)
		}
		return json.Unmarshal(data, &task)
	})
	return &task, err
}

func (s *BoltStore) ListTasks() ([]*types.Task, error) {
	var tasks []*types.Task
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketTasks)
		return b.ForEach(func(k, v []byte) error {
			var task types.Task
			if err := json.Unmarshal(v, &task); err != nil {
				return err
			}
			tasks = append(tasks, &task)
			return nil
		})
	})
	return tasks, err
}

func (s *BoltStore) ListTasksByService(serviceID string) ([]*types.Task, error) {
	tasks, err := s.ListTasks()
	if err != nil {
		return nil, err
	}

	var filtered []*types.Task
	for _, task := range tasks {
		if task.ServiceID == serviceID {
			filtered = append(filtered, task)
		}
	}
	return filtered, nil
}

func (s *BoltStore) ListTasksByNode(nodeID string) ([]*types.Task, error) {
	tasks, err := s.ListTasks()
	if err != nil {
		return nil, err
	}

	var filtered []*types.Task
	for _, task := range tasks {
		if task.NodeID == nodeID {
			filtered = append(filtered, task)
		}
	}
	return filtered, nil
}

func (s *BoltStore) UpdateTask(task *types.Task) error {
	return s.CreateTask(task)
}

func (s *BoltStore) DeleteTask(id string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketTasks)
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
