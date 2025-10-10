package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/cuemby/warren/pkg/api"
	"github.com/cuemby/warren/pkg/manager"
	"github.com/cuemby/warren/pkg/reconciler"
	"github.com/cuemby/warren/pkg/scheduler"
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

		fmt.Printf("Creating service '%s'\n", name)
		fmt.Printf("  Image: %s\n", image)
		fmt.Printf("  Replicas: %d\n", replicas)
		fmt.Println()
		fmt.Println("Implementation coming in Milestone 1!")
		return nil
	},
}

var serviceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List services",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Listing services...")
		fmt.Println("Implementation coming in Milestone 1!")
		return nil
	},
}

func init() {
	serviceCmd.AddCommand(serviceCreateCmd)
	serviceCmd.AddCommand(serviceListCmd)

	serviceCreateCmd.Flags().String("image", "", "Container image")
	serviceCreateCmd.Flags().Int("replicas", 1, "Number of replicas")
	serviceCreateCmd.MarkFlagRequired("image")
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
		fmt.Println("Listing nodes...")
		fmt.Println("Implementation coming in Milestone 1!")
		return nil
	},
}

func init() {
	nodeCmd.AddCommand(nodeListCmd)
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
