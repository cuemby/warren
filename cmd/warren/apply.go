package main

import (
	"fmt"
	"os"

	"github.com/cuemby/warren/pkg/client"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply a configuration file",
	Long: `Apply a Warren configuration from a YAML file.

Examples:
  # Apply a service definition
  warren apply -f service.yaml

  # Apply multiple resources
  warren apply -f cluster-config.yaml`,
	RunE: runApply,
}

func init() {
	applyCmd.Flags().StringP("file", "f", "", "YAML file to apply (required)")
	applyCmd.Flags().String("manager", "localhost:8080", "Manager address")
	_ = applyCmd.MarkFlagRequired("file")

	rootCmd.AddCommand(applyCmd)
}

// WarrenResource represents a generic Warren resource
type WarrenResource struct {
	APIVersion string                 `yaml:"apiVersion"`
	Kind       string                 `yaml:"kind"`
	Metadata   ResourceMetadata       `yaml:"metadata"`
	Spec       map[string]interface{} `yaml:"spec"`
}

type ResourceMetadata struct {
	Name   string            `yaml:"name"`
	Labels map[string]string `yaml:"labels,omitempty"`
}

func runApply(cmd *cobra.Command, args []string) error {
	filename, _ := cmd.Flags().GetString("file")
	managerAddr, _ := cmd.Flags().GetString("manager")

	// Read YAML file
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file: %v", err)
	}

	// Parse YAML
	var resource WarrenResource
	if err := yaml.Unmarshal(data, &resource); err != nil {
		return fmt.Errorf("failed to parse YAML: %v", err)
	}

	// Connect to manager
	c, err := client.NewClient(managerAddr)
	if err != nil {
		return fmt.Errorf("failed to connect to manager: %v", err)
	}
	defer c.Close()

	// Apply resource based on kind
	switch resource.Kind {
	case "Service":
		return applyService(c, &resource)
	case "Secret":
		return applySecret(c, &resource)
	case "Volume":
		return applyVolume(c, &resource)
	default:
		return fmt.Errorf("unsupported resource kind: %s", resource.Kind)
	}
}

func applyService(c *client.Client, resource *WarrenResource) error {
	name := resource.Metadata.Name
	image := getString(resource.Spec, "image", "")
	replicas := getInt(resource.Spec, "replicas", 1)

	if image == "" {
		return fmt.Errorf("service image is required")
	}

	// Try to get existing service
	existing, err := c.GetService(name)
	if err == nil && existing != nil {
		// Service exists, update it (scale)
		fmt.Printf("Updating service: %s\n", name)
		if _, err := c.UpdateService(name, int32(replicas)); err != nil {
			return fmt.Errorf("failed to update service: %v", err)
		}
		fmt.Printf("✓ Service updated: %s (replicas=%d)\n", name, replicas)
	} else {
		// Service doesn't exist, create it
		fmt.Printf("Creating service: %s\n", name)

		// Get environment variables if specified
		env := make(map[string]string)
		if envSpec, ok := resource.Spec["env"].(map[string]interface{}); ok {
			for k, v := range envSpec {
				env[k] = fmt.Sprintf("%v", v)
			}
		}

		service, err := c.CreateService(name, image, int32(replicas), env)
		if err != nil {
			return fmt.Errorf("failed to create service: %v", err)
		}
		fmt.Printf("✓ Service created: %s (ID: %s)\n", name, service.Id)
	}

	return nil
}

func applySecret(c *client.Client, resource *WarrenResource) error {
	name := resource.Metadata.Name
	data := getString(resource.Spec, "data", "")

	if data == "" {
		return fmt.Errorf("secret data is required")
	}

	// Check if secret exists
	existing, _ := c.GetSecretByName(name)
	if existing != nil {
		fmt.Printf("Secret already exists: %s (skipping)\n", name)
		return nil
	}

	// Create secret
	fmt.Printf("Creating secret: %s\n", name)
	secret, err := c.CreateSecret(name, []byte(data))
	if err != nil {
		return fmt.Errorf("failed to create secret: %v", err)
	}

	fmt.Printf("✓ Secret created: %s (ID: %s)\n", name, secret.Id)
	return nil
}

func applyVolume(c *client.Client, resource *WarrenResource) error {
	name := resource.Metadata.Name
	driver := getString(resource.Spec, "driver", "local")

	// Get driver options if specified
	opts := make(map[string]string)
	if optsSpec, ok := resource.Spec["driverOpts"].(map[string]interface{}); ok {
		for k, v := range optsSpec {
			opts[k] = fmt.Sprintf("%v", v)
		}
	}

	// Check if volume exists
	existing, _ := c.GetVolumeByName(name)
	if existing != nil {
		fmt.Printf("Volume already exists: %s (skipping)\n", name)
		return nil
	}

	// Create volume
	fmt.Printf("Creating volume: %s\n", name)
	volume, err := c.CreateVolume(name, driver, opts)
	if err != nil {
		return fmt.Errorf("failed to create volume: %v", err)
	}

	fmt.Printf("✓ Volume created: %s (ID: %s)\n", name, volume.Id)
	return nil
}

// Helper functions
func getString(m map[string]interface{}, key, defaultValue string) string {
	if v, ok := m[key]; ok {
		return fmt.Sprintf("%v", v)
	}
	return defaultValue
}

func getInt(m map[string]interface{}, key string, defaultValue int) int {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case int:
			return val
		case float64:
			return int(val)
		}
	}
	return defaultValue
}
