package framework

import (
	"github.com/cuemby/warren/api/proto"
	"github.com/cuemby/warren/pkg/client"
)

// Client wraps the Warren client with test-friendly methods
type Client struct {
	*client.Client
}

// NewClient creates a new test client wrapper
func NewClient(c *client.Client) *Client {
	return &Client{Client: c}
}

// CreateService creates a service with default environment
func (c *Client) CreateService(name, image string, replicas int) error {
	_, err := c.Client.CreateService(name, image, int32(replicas), nil)
	return err
}

// CreateServiceWithEnv creates a service with custom environment variables
func (c *Client) CreateServiceWithEnv(name, image string, replicas int, env map[string]string) error {
	_, err := c.Client.CreateService(name, image, int32(replicas), env)
	return err
}

// CreateIngress creates an ingress rule
func (c *Client) CreateIngress(name string, spec *IngressSpec) error {
	req := &proto.CreateIngressRequest{
		Name: name,
		Spec: &proto.IngressSpec{
			Rules: []*proto.IngressRule{
				{
					Host: spec.Host,
					Paths: []*proto.IngressPath{
						{
							Path:     spec.Path,
							PathType: spec.PathType,
							Backend: &proto.IngressBackend{
								ServiceName: spec.Backend.Service,
								ServicePort: int32(spec.Backend.Port),
							},
						},
					},
				},
			},
		},
	}

	if spec.TLS != nil && spec.TLS.Enabled {
		req.Spec.Tls = []*proto.IngressTLS{
			{
				Hosts:      []string{spec.Host},
				SecretName: spec.TLS.SecretName,
			},
		}
	}

	_, err := c.Client.CreateIngress(req)
	return err
}

// DeleteIngress deletes an ingress rule
func (c *Client) DeleteIngress(name string) error {
	return c.Client.DeleteIngress(&proto.DeleteIngressRequest{Name: name})
}

// ListIngresses lists all ingress rules
func (c *Client) ListIngresses() ([]*proto.Ingress, error) {
	return c.Client.ListIngresses()
}
