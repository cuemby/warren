package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/cuemby/warren/pkg/api"
	"github.com/cuemby/warren/pkg/client"
	"github.com/cuemby/warren/pkg/manager"
	"github.com/cuemby/warren/pkg/reconciler"
	"github.com/cuemby/warren/pkg/scheduler"
	"github.com/cuemby/warren/pkg/types"
	"github.com/cuemby/warren/pkg/worker"
	"github.com/spf13/cobra"
)

var (
	// Version information (set via ldflags during build)
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "warren",
	Short: "Warren - Simple yet powerful container orchestrator",
	Long: `Warren is a container orchestration platform that combines
the simplicity of Docker Swarm with the features of Kubernetes,
delivered as a single binary with zero external dependencies.

Built for edge computing with telco-grade reliability.`,
	Version: Version,
}

func init() {
	// Set version template
	rootCmd.SetVersionTemplate(fmt.Sprintf(
		"Warren version %s\nCommit: %s\nBuilt: %s\n",
		Version, Commit, BuildTime,
	))

	// Add subcommands
	rootCmd.AddCommand(clusterCmd)
	rootCmd.AddCommand(workerCmd)
	rootCmd.AddCommand(serviceCmd)
	rootCmd.AddCommand(nodeCmd)
	rootCmd.AddCommand(secretCmd)
	rootCmd.AddCommand(volumeCmd)
}

// Cluster commands
var clusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Manage Warren cluster",
}

var clusterInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new Warren cluster",
	Long: `Initialize a new Warren cluster with this node as the first manager.

This command starts the Warren manager in single-node mode, which will
automatically form a Raft quorum once additional managers join.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		nodeID, _ := cmd.Flags().GetString("node-id")
		bindAddr, _ := cmd.Flags().GetString("bind-addr")
		apiAddr, _ := cmd.Flags().GetString("api-addr")
		dataDir, _ := cmd.Flags().GetString("data-dir")

		fmt.Println("Initializing Warren cluster...")
		fmt.Printf("  Node ID: %s\n", nodeID)
		fmt.Printf("  Raft Address: %s\n", bindAddr)
		fmt.Printf("  API Address: %s\n", apiAddr)
		fmt.Printf("  Data Directory: %s\n", dataDir)
		fmt.Println()

		// Create manager
		mgr, err := manager.NewManager(&manager.Config{
			NodeID:   nodeID,
			BindAddr: bindAddr,
			DataDir:  dataDir,
		})
		if err != nil {
			return fmt.Errorf("failed to create manager: %v", err)
		}

		// Bootstrap cluster
		if err := mgr.Bootstrap(); err != nil {
			return fmt.Errorf("failed to bootstrap cluster: %v", err)
		}

		fmt.Println("✓ Cluster initialized successfully")

		// Start scheduler
		sched := scheduler.NewScheduler(mgr)
		sched.Start()
		fmt.Println("✓ Scheduler started")

		// Start reconciler
		recon := reconciler.NewReconciler(mgr)
		recon.Start()
		fmt.Println("✓ Reconciler started")

		// Start API server in background
		apiServer := api.NewServer(mgr)
		errCh := make(chan error, 1)
		go func() {
			if err := apiServer.Start(apiAddr); err != nil {
				errCh <- fmt.Errorf("API server error: %v", err)
			}
		}()

		fmt.Println()
		fmt.Println("Manager is running. Press Ctrl+C to stop.")

		// Wait for interrupt signal or API server error
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

		select {
		case <-sigCh:
			fmt.Println("\nShutting down...")
		case err := <-errCh:
			fmt.Fprintf(os.Stderr, "\nError: %v\n", err)
		}

		// Shutdown
		sched.Stop()
		recon.Stop()
		apiServer.Stop()
		if err := mgr.Shutdown(); err != nil {
			return fmt.Errorf("failed to shutdown: %v", err)
		}

		fmt.Println("✓ Shutdown complete")
		return nil
	},
}

var clusterJoinTokenCmd = &cobra.Command{
	Use:   "join-token [worker|manager]",
	Short: "Generate a join token for workers or managers",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		role := args[0]
		if role != "worker" && role != "manager" {
			return fmt.Errorf("role must be 'worker' or 'manager'")
		}

		fmt.Printf("Generating join token for %s...\n", role)
		fmt.Println("Implementation coming in Milestone 1!")
		return nil
	},
}

var clusterJoinCmd = &cobra.Command{
	Use:   "join --token TOKEN",
	Short: "Join this node to an existing cluster",
	RunE: func(cmd *cobra.Command, args []string) error {
		token, _ := cmd.Flags().GetString("token")
		if token == "" {
			return fmt.Errorf("--token is required")
		}

		fmt.Println("Joining cluster...")
		fmt.Println("Implementation coming in Milestone 1!")
		return nil
	},
}

func init() {
	clusterCmd.AddCommand(clusterInitCmd)
	clusterCmd.AddCommand(clusterJoinTokenCmd)
	clusterCmd.AddCommand(clusterJoinCmd)

	// Flags for init command
	clusterInitCmd.Flags().String("node-id", "manager-1", "Unique node ID")
	clusterInitCmd.Flags().String("bind-addr", "127.0.0.1:7946", "Address for Raft communication")
	clusterInitCmd.Flags().String("api-addr", "127.0.0.1:8080", "Address for gRPC API")
	clusterInitCmd.Flags().String("data-dir", "./warren-data", "Data directory for cluster state")

	// Flags for join command
	clusterJoinCmd.Flags().String("token", "", "Join token from manager")
}

// Worker commands
var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Worker node operations",
}

var workerStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a worker node",
	Long:  `Start a Warren worker node and connect to the manager.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		nodeID, _ := cmd.Flags().GetString("node-id")
		managerAddr, _ := cmd.Flags().GetString("manager")
		dataDir, _ := cmd.Flags().GetString("data-dir")
		cpuCores, _ := cmd.Flags().GetInt("cpu")
		memoryGB, _ := cmd.Flags().GetInt("memory")

		fmt.Println("Starting Warren worker...")
		fmt.Printf("  Node ID: %s\n", nodeID)
		fmt.Printf("  Manager: %s\n", managerAddr)
		fmt.Printf("  Data Directory: %s\n", dataDir)
		fmt.Printf("  Resources: %d cores, %d GB memory\n", cpuCores, memoryGB)
		fmt.Println()

		// Create worker
		w, err := worker.NewWorker(&worker.Config{
			NodeID:      nodeID,
			ManagerAddr: managerAddr,
			DataDir:     dataDir,
		})
		if err != nil {
			return fmt.Errorf("failed to create worker: %v", err)
		}

		// Start worker
		resources := &types.NodeResources{
			CPUCores:    cpuCores,
			MemoryBytes: int64(memoryGB) * 1024 * 1024 * 1024,
			DiskBytes:   100 * 1024 * 1024 * 1024, // 100GB default
		}

		if err := w.Start(resources); err != nil {
			return fmt.Errorf("failed to start worker: %v", err)
		}

		fmt.Println()
		fmt.Println("Worker is running. Press Ctrl+C to stop.")

		// Wait for interrupt
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh

		fmt.Println("\nShutting down...")
		if err := w.Stop(); err != nil {
			return fmt.Errorf("failed to stop worker: %v", err)
		}

		fmt.Println("✓ Shutdown complete")
		return nil
	},
}

func init() {
	workerCmd.AddCommand(workerStartCmd)

	workerStartCmd.Flags().String("node-id", "worker-1", "Unique node ID")
	workerStartCmd.Flags().String("manager", "127.0.0.1:8080", "Manager gRPC address")
	workerStartCmd.Flags().String("data-dir", "./warren-worker-data", "Data directory")
	workerStartCmd.Flags().Int("cpu", 4, "CPU cores")
	workerStartCmd.Flags().Int("memory", 8, "Memory in GB")
}

// Service commands
var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Manage services",
}

var serviceCreateCmd = &cobra.Command{
	Use:   "create NAME",
	Short: "Create a new service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		image, _ := cmd.Flags().GetString("image")
		replicas, _ := cmd.Flags().GetInt("replicas")
		manager, _ := cmd.Flags().GetString("manager")
		envVars, _ := cmd.Flags().GetStringSlice("env")

		// Parse env vars
		env := make(map[string]string)
		for _, e := range envVars {
			// Split on first = only
			parts := splitEnv(e)
			if len(parts) == 2 {
				env[parts[0]] = parts[1]
			}
		}

		// Connect to manager
		c, err := client.NewClient(manager)
		if err != nil {
			return fmt.Errorf("failed to connect to manager: %v", err)
		}
		defer c.Close()

		// Create service
		service, err := c.CreateService(name, image, int32(replicas), env)
		if err != nil {
			return fmt.Errorf("failed to create service: %v", err)
		}

		fmt.Printf("✓ Service created: %s\n", service.Name)
		fmt.Printf("  ID: %s\n", service.Id)
		fmt.Printf("  Image: %s\n", service.Image)
		fmt.Printf("  Replicas: %d\n", service.Replicas)
		return nil
	},
}

var serviceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List services",
	RunE: func(cmd *cobra.Command, args []string) error {
		manager, _ := cmd.Flags().GetString("manager")

		// Connect to manager
		c, err := client.NewClient(manager)
		if err != nil {
			return fmt.Errorf("failed to connect to manager: %v", err)
		}
		defer c.Close()

		// List services
		services, err := c.ListServices()
		if err != nil {
			return fmt.Errorf("failed to list services: %v", err)
		}

		if len(services) == 0 {
			fmt.Println("No services found")
			return nil
		}

		fmt.Printf("%-20s %-12s %-30s %s\n", "NAME", "REPLICAS", "IMAGE", "ID")
		for _, svc := range services {
			fmt.Printf("%-20s %-12d %-30s %s\n",
				truncate(svc.Name, 20),
				svc.Replicas,
				truncate(svc.Image, 30),
				svc.Id[:12])
		}
		return nil
	},
}

var serviceInspectCmd = &cobra.Command{
	Use:   "inspect NAME",
	Short: "Inspect a service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		manager, _ := cmd.Flags().GetString("manager")

		// Connect to manager
		c, err := client.NewClient(manager)
		if err != nil {
			return fmt.Errorf("failed to connect to manager: %v", err)
		}
		defer c.Close()

		// Get service
		service, err := c.GetService(name)
		if err != nil {
			return fmt.Errorf("failed to get service: %v", err)
		}

		fmt.Printf("Service: %s\n", service.Name)
		fmt.Printf("  ID: %s\n", service.Id)
		fmt.Printf("  Image: %s\n", service.Image)
		fmt.Printf("  Replicas: %d\n", service.Replicas)
		fmt.Printf("  Mode: %s\n", service.Mode)
		if len(service.Env) > 0 {
			fmt.Println("  Environment:")
			for k, v := range service.Env {
				fmt.Printf("    %s=%s\n", k, v)
			}
		}
		return nil
	},
}

var serviceDeleteCmd = &cobra.Command{
	Use:   "delete NAME",
	Short: "Delete a service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		manager, _ := cmd.Flags().GetString("manager")

		// Connect to manager
		c, err := client.NewClient(manager)
		if err != nil {
			return fmt.Errorf("failed to connect to manager: %v", err)
		}
		defer c.Close()

		// Get service to get ID
		service, err := c.GetService(name)
		if err != nil {
			return fmt.Errorf("failed to find service: %v", err)
		}

		// Delete service
		if err := c.DeleteService(service.Id); err != nil {
			return fmt.Errorf("failed to delete service: %v", err)
		}

		fmt.Printf("✓ Service deleted: %s\n", name)
		return nil
	},
}

var serviceScaleCmd = &cobra.Command{
	Use:   "scale NAME --replicas N",
	Short: "Scale a service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		replicas, _ := cmd.Flags().GetInt("replicas")
		manager, _ := cmd.Flags().GetString("manager")

		// Connect to manager
		c, err := client.NewClient(manager)
		if err != nil {
			return fmt.Errorf("failed to connect to manager: %v", err)
		}
		defer c.Close()

		// Get service to get ID
		service, err := c.GetService(name)
		if err != nil {
			return fmt.Errorf("failed to find service: %v", err)
		}

		// Update replicas
		updated, err := c.UpdateService(service.Id, int32(replicas))
		if err != nil {
			return fmt.Errorf("failed to scale service: %v", err)
		}

		fmt.Printf("✓ Service scaled: %s\n", name)
		fmt.Printf("  Replicas: %d → %d\n", service.Replicas, updated.Replicas)
		return nil
	},
}

func init() {
	serviceCmd.AddCommand(serviceCreateCmd)
	serviceCmd.AddCommand(serviceListCmd)
	serviceCmd.AddCommand(serviceInspectCmd)
	serviceCmd.AddCommand(serviceDeleteCmd)
	serviceCmd.AddCommand(serviceScaleCmd)

	// Common flag
	for _, cmd := range []*cobra.Command{serviceCreateCmd, serviceListCmd, serviceInspectCmd, serviceDeleteCmd, serviceScaleCmd} {
		cmd.Flags().String("manager", "127.0.0.1:8080", "Manager address")
	}

	serviceCreateCmd.Flags().String("image", "", "Container image")
	serviceCreateCmd.Flags().Int("replicas", 1, "Number of replicas")
	serviceCreateCmd.Flags().StringSlice("env", []string{}, "Environment variables (KEY=VALUE)")
	serviceCreateCmd.MarkFlagRequired("image")

	serviceScaleCmd.Flags().Int("replicas", 0, "Number of replicas")
	serviceScaleCmd.MarkFlagRequired("replicas")
}

// Node commands
var nodeCmd = &cobra.Command{
	Use:   "node",
	Short: "Manage nodes",
}

var nodeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List nodes in the cluster",
	RunE: func(cmd *cobra.Command, args []string) error {
		manager, _ := cmd.Flags().GetString("manager")

		// Connect to manager
		c, err := client.NewClient(manager)
		if err != nil {
			return fmt.Errorf("failed to connect to manager: %v", err)
		}
		defer c.Close()

		// List nodes
		nodes, err := c.ListNodes()
		if err != nil {
			return fmt.Errorf("failed to list nodes: %v", err)
		}

		if len(nodes) == 0 {
			fmt.Println("No nodes found")
			return nil
		}

		fmt.Printf("%-15s %-10s %-15s %-10s\n", "ID", "ROLE", "STATUS", "CPU")
		for _, node := range nodes {
			fmt.Printf("%-15s %-10s %-15s %-10d\n",
				truncate(node.Id, 15),
				node.Role,
				node.Status,
				node.Resources.CpuCores)
		}
		return nil
	},
}

func init() {
	nodeCmd.AddCommand(nodeListCmd)

	nodeListCmd.Flags().String("manager", "127.0.0.1:8080", "Manager address")
}

// Secret commands
var secretCmd = &cobra.Command{
	Use:   "secret",
	Short: "Manage secrets",
}

var secretListCmd = &cobra.Command{
	Use:   "list",
	Short: "List secrets",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Listing secrets...")
		fmt.Println("Implementation coming in Milestone 1!")
		return nil
	},
}

func init() {
	secretCmd.AddCommand(secretListCmd)
}

// Volume commands
var volumeCmd = &cobra.Command{
	Use:   "volume",
	Short: "Manage volumes",
}

var volumeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List volumes",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Listing volumes...")
		fmt.Println("Implementation coming in Milestone 1!")
		return nil
	},
}

func init() {
	volumeCmd.AddCommand(volumeListCmd)
}

// Helper functions

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func splitEnv(s string) []string {
	idx := strings.Index(s, "=")
	if idx == -1 {
		return []string{s}
	}
	return []string{s[:idx], s[idx+1:]}
}
