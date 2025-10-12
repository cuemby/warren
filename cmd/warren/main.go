package main

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof" // Import pprof for profiling endpoints
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/cuemby/warren/api/proto"
	"github.com/cuemby/warren/pkg/api"
	"github.com/cuemby/warren/pkg/client"
	"github.com/cuemby/warren/pkg/embedded"
	"github.com/cuemby/warren/pkg/log"
	"github.com/cuemby/warren/pkg/manager"
	"github.com/cuemby/warren/pkg/metrics"
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

	// Global flags
	rootCmd.PersistentFlags().String("log-level", "info", "Log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().Bool("log-json", false, "Output logs in JSON format")
	rootCmd.PersistentFlags().Bool("external-containerd", false, "Use external containerd instead of embedded (requires containerd daemon running)")
	rootCmd.PersistentFlags().String("containerd-socket", "", "Custom containerd socket path (auto-detected if not specified)")

	// Initialize logging before command execution
	cobra.OnInitialize(initLogging)

	// Add subcommands
	rootCmd.AddCommand(clusterCmd)
	rootCmd.AddCommand(managerCmd)
	rootCmd.AddCommand(workerCmd)
	rootCmd.AddCommand(serviceCmd)
	rootCmd.AddCommand(nodeCmd)
	rootCmd.AddCommand(secretCmd)
	rootCmd.AddCommand(volumeCmd)
	rootCmd.AddCommand(ingressCmd)
	rootCmd.AddCommand(certificateCmd)
}

func initLogging() {
	logLevel, _ := rootCmd.PersistentFlags().GetString("log-level")
	logJSON, _ := rootCmd.PersistentFlags().GetBool("log-json")

	log.Init(log.Config{
		Level:      log.Level(logLevel),
		JSONOutput: logJSON,
	})
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
		useExternal, _ := cmd.Flags().GetBool("external-containerd")

		fmt.Println("Initializing Warren cluster...")
		fmt.Printf("  Node ID: %s\n", nodeID)
		fmt.Printf("  Raft Address: %s\n", bindAddr)
		fmt.Printf("  API Address: %s\n", apiAddr)
		fmt.Printf("  Data Directory: %s\n", dataDir)
		if useExternal {
			fmt.Println("  Containerd: External (system containerd)")
		} else {
			fmt.Println("  Containerd: Embedded")
		}
		fmt.Println()

		// Start embedded containerd if needed
		ctx := context.Background()
		containerdMgr, err := embedded.EnsureContainerd(ctx, dataDir, useExternal)
		if err != nil {
			return fmt.Errorf("failed to start containerd: %v", err)
		}
		defer containerdMgr.Stop()

		if !useExternal {
			socketPath := containerdMgr.GetSocketPath()
			if runtime.GOOS == "darwin" {
				fmt.Printf("✓ Lima VM started with containerd (socket: %s)\n", socketPath)
			} else {
				fmt.Printf("✓ Embedded containerd started (socket: %s)\n", socketPath)
			}
		}

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

		// Start metrics collector
		metricsCollector := metrics.NewCollector(mgr)
		metricsCollector.Start()
		fmt.Println("✓ Metrics collector started")

		// Set version for health checks
		metrics.SetVersion("1.1.0")

		// Register initial health status
		metrics.RegisterComponent("raft", true, "bootstrapped")
		metrics.RegisterComponent("containerd", false, "initializing")
		metrics.RegisterComponent("api", false, "initializing")

		// Start metrics HTTP server in background
		metricsAddr := "127.0.0.1:9090"
		pprofEnabled, _ := cmd.Flags().GetBool("enable-pprof")

		go func() {
			http.Handle("/metrics", metrics.Handler())
			http.Handle("/health", metrics.HealthHandler())
			http.Handle("/ready", metrics.ReadyHandler())
			http.Handle("/live", metrics.LivenessHandler())
			if err := http.ListenAndServe(metricsAddr, nil); err != nil {
				fmt.Printf("Metrics server error: %v\n", err)
			}
		}()
		fmt.Printf("✓ Metrics endpoint: http://%s/metrics\n", metricsAddr)
		fmt.Printf("✓ Health endpoints:\n")
		fmt.Printf("  - Health check: http://%s/health\n", metricsAddr)
		fmt.Printf("  - Readiness:    http://%s/ready\n", metricsAddr)
		fmt.Printf("  - Liveness:     http://%s/live\n", metricsAddr)

		if pprofEnabled {
			fmt.Printf("✓ Profiling endpoints enabled at http://%s/debug/pprof/\n", metricsAddr)
			fmt.Printf("  - Heap profile: http://%s/debug/pprof/heap\n", metricsAddr)
			fmt.Printf("  - CPU profile: http://%s/debug/pprof/profile\n", metricsAddr)
			fmt.Printf("  - Goroutines: http://%s/debug/pprof/goroutine\n", metricsAddr)
		}

		// Start API server in background
		apiServer, err := api.NewServer(mgr)
		if err != nil {
			return fmt.Errorf("failed to create API server: %v", err)
		}
		errCh := make(chan error, 1)
		go func() {
			if err := apiServer.Start(apiAddr); err != nil {
				errCh <- fmt.Errorf("API server error: %v", err)
			}
		}()

		// Wait for API server to start
		time.Sleep(500 * time.Millisecond)

		// Update health status - API is now ready
		metrics.RegisterComponent("api", true, "ready")
		metrics.RegisterComponent("containerd", true, "ready")

		// Start ingress proxy
		if err := mgr.StartIngress(); err != nil {
			fmt.Printf("Warning: Failed to start ingress proxy: %v\n", err)
		} else {
			fmt.Println("✓ Ingress proxy started on ports 8000 (HTTP) and 8443 (HTTPS)")
		}

		// Generate and display join tokens for initial setup
		fmt.Println()
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("  Join Tokens (valid for 24 hours)")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println()

		// Generate worker token
		workerToken, _ := mgr.GenerateJoinToken("worker")
		fmt.Println("Worker Token:")
		fmt.Printf("  %s\n", workerToken.Token)
		fmt.Println()
		fmt.Println("To add a worker node:")
		fmt.Printf("  warren worker start --manager %s --token %s\n", apiAddr, workerToken.Token)
		fmt.Println()

		// Generate manager token
		managerToken, _ := mgr.GenerateJoinToken("manager")
		fmt.Println("Manager Token:")
		fmt.Printf("  %s\n", managerToken.Token)
		fmt.Println()
		fmt.Println("To add a manager node:")
		fmt.Printf("  warren manager join --leader %s --token %s\n", apiAddr, managerToken.Token)
		fmt.Println()

		// Generate CLI token
		cliToken, _ := mgr.GenerateJoinToken("worker") // CLI can use worker token
		fmt.Println("CLI Token (for remote CLI access):")
		fmt.Printf("  %s\n", cliToken.Token)
		fmt.Println()
		fmt.Println("To initialize CLI:")
		fmt.Printf("  warren init --manager %s --token %s\n", apiAddr, cliToken.Token)
		fmt.Println()
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println()
		fmt.Println("Manager is running. Press Ctrl+C to stop.")
		fmt.Printf("gRPC API listening on %s\n", apiAddr)
		fmt.Println()

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
		metricsCollector.Stop()
		mgr.StopIngress()
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

		manager, _ := cmd.Flags().GetString("manager")

		// Connect to manager
		c, err := client.NewClient(manager)
		if err != nil {
			return fmt.Errorf("failed to connect to manager: %v", err)
		}
		defer c.Close()

		// Generate token
		token, err := c.GenerateJoinToken(role)
		if err != nil {
			return fmt.Errorf("failed to generate token: %v", err)
		}

		fmt.Printf("Join token for %s:\n\n", role)
		fmt.Printf("    %s\n\n", token)
		fmt.Println("This token expires in 24 hours.")
		fmt.Printf("\nTo join a %s to the cluster, run:\n", role)
		if role == "manager" {
			fmt.Printf("    warren manager join --token %s --leader %s\n", token, manager)
		} else {
			fmt.Printf("    warren worker start --manager %s --token %s\n", manager, token)
		}
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

var clusterInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Display cluster information",
	RunE: func(cmd *cobra.Command, args []string) error {
		manager, _ := cmd.Flags().GetString("manager")

		// Connect to manager
		c, err := client.NewClient(manager)
		if err != nil {
			return fmt.Errorf("failed to connect to manager: %v", err)
		}
		defer c.Close()

		// Get cluster info
		info, err := c.GetClusterInfo()
		if err != nil {
			return fmt.Errorf("failed to get cluster info: %v", err)
		}

		fmt.Println("Cluster Information:")
		fmt.Printf("  Leader ID: %s\n", info.LeaderId)
		fmt.Printf("  Leader Address: %s\n", info.LeaderAddr)
		fmt.Printf("  Servers: %d\n", len(info.Servers))
		fmt.Println()
		fmt.Println("Raft Servers:")
		for _, server := range info.Servers {
			fmt.Printf("  - ID: %s\n", server.Id)
			fmt.Printf("    Address: %s\n", server.Address)
			fmt.Printf("    Suffrage: %s\n", server.Suffrage)
			fmt.Println()
		}
		return nil
	},
}

var cliInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize CLI certificate for secure communication with manager",
	Long: `Request a certificate from the manager to enable mTLS authentication.
This command must be run before using other CLI commands.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		managerAddr, _ := cmd.Flags().GetString("manager")
		token, _ := cmd.Flags().GetString("token")

		if token == "" {
			return fmt.Errorf("--token is required (get token from 'warren cluster join-token --manager <addr>')")
		}

		fmt.Println("Initializing CLI certificate...")
		fmt.Printf("  Manager: %s\n", managerAddr)

		// Create client with token to request certificate
		c, err := client.NewClientWithToken(managerAddr, token)
		if err != nil {
			return fmt.Errorf("failed to initialize CLI: %v", err)
		}
		defer c.Close()

		fmt.Println("\n✓ CLI initialized successfully")
		fmt.Println("You can now use other Warren CLI commands")

		return nil
	},
}

func init() {
	// Add init command to root
	rootCmd.AddCommand(cliInitCmd)
	cliInitCmd.Flags().String("manager", "127.0.0.1:8080", "Manager address")
	cliInitCmd.Flags().String("token", "", "Join token from manager (required)")

	clusterCmd.AddCommand(clusterInitCmd)
	clusterCmd.AddCommand(clusterJoinTokenCmd)
	clusterCmd.AddCommand(clusterJoinCmd)
	clusterCmd.AddCommand(clusterInfoCmd)

	// Flags for init command
	clusterInitCmd.Flags().String("node-id", "manager-1", "Unique node ID")
	clusterInitCmd.Flags().String("bind-addr", "127.0.0.1:7946", "Address for Raft communication")
	clusterInitCmd.Flags().String("api-addr", "127.0.0.1:8080", "Address for gRPC API")
	clusterInitCmd.Flags().String("data-dir", "./warren-data", "Data directory for cluster state")
	clusterInitCmd.Flags().Bool("enable-pprof", false, "Enable pprof profiling endpoints on metrics server")

	// Flags for join-token and info commands
	clusterJoinTokenCmd.Flags().String("manager", "127.0.0.1:8080", "Manager address")
	clusterInfoCmd.Flags().String("manager", "127.0.0.1:8080", "Manager address")

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
		useExternal, _ := cmd.Flags().GetBool("external-containerd")
		token, _ := cmd.Flags().GetString("token")

		fmt.Println("Starting Warren worker...")
		fmt.Printf("  Node ID: %s\n", nodeID)
		fmt.Printf("  Manager: %s\n", managerAddr)
		fmt.Printf("  Data Directory: %s\n", dataDir)
		fmt.Printf("  Resources: %d cores, %d GB memory\n", cpuCores, memoryGB)
		if useExternal {
			fmt.Println("  Containerd: External (system containerd)")
		} else {
			fmt.Println("  Containerd: Embedded")
		}
		fmt.Println()

		// Start embedded containerd if needed
		ctx := context.Background()
		containerdMgr, err := embedded.EnsureContainerd(ctx, dataDir, useExternal)
		if err != nil {
			return fmt.Errorf("failed to start containerd: %v", err)
		}
		defer containerdMgr.Stop()

		if !useExternal {
			socketPath := containerdMgr.GetSocketPath()
			if runtime.GOOS == "darwin" {
				fmt.Printf("✓ Lima VM started with containerd (socket: %s)\n", socketPath)
			} else {
				fmt.Printf("✓ Embedded containerd started (socket: %s)\n", socketPath)
			}
		}

		// Create worker
		w, err := worker.NewWorker(&worker.Config{
			NodeID:           nodeID,
			ManagerAddr:      managerAddr,
			DataDir:          dataDir,
			ContainerdSocket: containerdMgr.GetSocketPath(),
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

		if err := w.Start(resources, token); err != nil {
			return fmt.Errorf("failed to start worker: %v", err)
		}

		// Start pprof server if enabled
		pprofEnabled, _ := cmd.Flags().GetBool("enable-pprof")
		if pprofEnabled {
			pprofAddr := "127.0.0.1:6060"
			go func() {
				if err := http.ListenAndServe(pprofAddr, nil); err != nil {
					fmt.Printf("Profiling server error: %v\n", err)
				}
			}()
			fmt.Printf("✓ Profiling endpoints enabled at http://%s/debug/pprof/\n", pprofAddr)
			fmt.Printf("  - Heap profile: http://%s/debug/pprof/heap\n", pprofAddr)
			fmt.Printf("  - CPU profile: http://%s/debug/pprof/profile\n", pprofAddr)
			fmt.Printf("  - Goroutines: http://%s/debug/pprof/goroutine\n", pprofAddr)
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
	workerStartCmd.Flags().String("token", "", "Join token from manager (required for first connection)")
	workerStartCmd.Flags().Bool("enable-pprof", false, "Enable pprof profiling endpoints")
}

// Manager commands
var managerCmd = &cobra.Command{
	Use:   "manager",
	Short: "Manager node operations",
}

var managerJoinCmd = &cobra.Command{
	Use:   "join",
	Short: "Join this manager to an existing cluster",
	RunE: func(cmd *cobra.Command, args []string) error {
		nodeID, _ := cmd.Flags().GetString("node-id")
		bindAddr, _ := cmd.Flags().GetString("bind-addr")
		apiAddr, _ := cmd.Flags().GetString("api-addr")
		dataDir, _ := cmd.Flags().GetString("data-dir")
		leader, _ := cmd.Flags().GetString("leader")
		token, _ := cmd.Flags().GetString("token")

		if token == "" {
			return fmt.Errorf("--token is required")
		}
		if leader == "" {
			return fmt.Errorf("--leader is required")
		}

		fmt.Println("Joining cluster as manager...")
		fmt.Printf("  Node ID: %s\n", nodeID)
		fmt.Printf("  Bind Address: %s\n", bindAddr)
		fmt.Printf("  API Address: %s\n", apiAddr)
		fmt.Printf("  Leader: %s\n", leader)
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

		// Join the cluster
		if err := mgr.Join(leader, token); err != nil {
			return fmt.Errorf("failed to join cluster: %v", err)
		}

		fmt.Println("✓ Successfully joined cluster")

		// Start scheduler
		sched := scheduler.NewScheduler(mgr)
		sched.Start()
		fmt.Println("✓ Scheduler started")

		// Start reconciler
		recon := reconciler.NewReconciler(mgr)
		recon.Start()
		fmt.Println("✓ Reconciler started")

		// Start metrics collector
		metricsCollector := metrics.NewCollector(mgr)
		metricsCollector.Start()
		fmt.Println("✓ Metrics collector started")

		// Set version for health checks
		metrics.SetVersion("1.1.0")

		// Register initial health status
		metrics.RegisterComponent("raft", true, "bootstrapped")
		metrics.RegisterComponent("containerd", false, "initializing")
		metrics.RegisterComponent("api", false, "initializing")

		// Start metrics HTTP server in background
		metricsAddr := "127.0.0.1:9090"
		pprofEnabled, _ := cmd.Flags().GetBool("enable-pprof")

		go func() {
			http.Handle("/metrics", metrics.Handler())
			http.Handle("/health", metrics.HealthHandler())
			http.Handle("/ready", metrics.ReadyHandler())
			http.Handle("/live", metrics.LivenessHandler())
			if err := http.ListenAndServe(metricsAddr, nil); err != nil {
				fmt.Printf("Metrics server error: %v\n", err)
			}
		}()
		fmt.Printf("✓ Metrics endpoint: http://%s/metrics\n", metricsAddr)
		fmt.Printf("✓ Health endpoints:\n")
		fmt.Printf("  - Health check: http://%s/health\n", metricsAddr)
		fmt.Printf("  - Readiness:    http://%s/ready\n", metricsAddr)
		fmt.Printf("  - Liveness:     http://%s/live\n", metricsAddr)

		if pprofEnabled {
			fmt.Printf("✓ Profiling endpoints enabled at http://%s/debug/pprof/\n", metricsAddr)
			fmt.Printf("  - Heap profile: http://%s/debug/pprof/heap\n", metricsAddr)
			fmt.Printf("  - CPU profile: http://%s/debug/pprof/profile\n", metricsAddr)
			fmt.Printf("  - Goroutines: http://%s/debug/pprof/goroutine\n", metricsAddr)
		}

		// Start API server in background
		apiServer, err := api.NewServer(mgr)
		if err != nil {
			return fmt.Errorf("failed to create API server: %v", err)
		}
		errCh := make(chan error, 1)
		go func() {
			if err := apiServer.Start(apiAddr); err != nil {
				errCh <- fmt.Errorf("API server error: %v", err)
			}
		}()

		// Wait for API server to start
		time.Sleep(500 * time.Millisecond)

		// Update health status - API is now ready
		metrics.RegisterComponent("api", true, "ready")
		metrics.RegisterComponent("containerd", true, "ready")

		// Start ingress proxy
		if err := mgr.StartIngress(); err != nil {
			fmt.Printf("Warning: Failed to start ingress proxy: %v\n", err)
		} else {
			fmt.Println("✓ Ingress proxy started on ports 8000 (HTTP) and 8443 (HTTPS)")
		}

		// Generate and display join tokens for initial setup
		fmt.Println()
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("  Join Tokens (valid for 24 hours)")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println()

		// Generate worker token
		workerToken, _ := mgr.GenerateJoinToken("worker")
		fmt.Println("Worker Token:")
		fmt.Printf("  %s\n", workerToken.Token)
		fmt.Println()
		fmt.Println("To add a worker node:")
		fmt.Printf("  warren worker start --manager %s --token %s\n", apiAddr, workerToken.Token)
		fmt.Println()

		// Generate manager token
		managerToken, _ := mgr.GenerateJoinToken("manager")
		fmt.Println("Manager Token:")
		fmt.Printf("  %s\n", managerToken.Token)
		fmt.Println()
		fmt.Println("To add a manager node:")
		fmt.Printf("  warren manager join --leader %s --token %s\n", apiAddr, managerToken.Token)
		fmt.Println()

		// Generate CLI token
		cliToken, _ := mgr.GenerateJoinToken("worker") // CLI can use worker token
		fmt.Println("CLI Token (for remote CLI access):")
		fmt.Printf("  %s\n", cliToken.Token)
		fmt.Println()
		fmt.Println("To initialize CLI:")
		fmt.Printf("  warren init --manager %s --token %s\n", apiAddr, cliToken.Token)
		fmt.Println()
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println()
		fmt.Println("Manager is running. Press Ctrl+C to stop.")
		fmt.Printf("gRPC API listening on %s\n", apiAddr)
		fmt.Println()

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
		metricsCollector.Stop()
		mgr.StopIngress()
		apiServer.Stop()
		if err := mgr.Shutdown(); err != nil {
			return fmt.Errorf("failed to shutdown: %v", err)
		}

		fmt.Println("✓ Shutdown complete")
		return nil
	},
}

func init() {
	managerCmd.AddCommand(managerJoinCmd)

	managerJoinCmd.Flags().String("node-id", "manager-2", "Unique node ID")
	managerJoinCmd.Flags().String("bind-addr", "127.0.0.1:7947", "Address for Raft communication")
	managerJoinCmd.Flags().String("api-addr", "127.0.0.1:8081", "Address for gRPC API")
	managerJoinCmd.Flags().String("data-dir", "./warren-data-2", "Data directory for cluster state")
	managerJoinCmd.Flags().String("leader", "", "Leader manager address")
	managerJoinCmd.Flags().String("token", "", "Join token from leader")
	managerJoinCmd.Flags().Bool("enable-pprof", false, "Enable pprof profiling endpoints on metrics server")
	managerJoinCmd.MarkFlagRequired("token")
	managerJoinCmd.MarkFlagRequired("leader")
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

		// Port publishing flags
		publishPorts, _ := cmd.Flags().GetStringSlice("publish")
		publishMode, _ := cmd.Flags().GetString("publish-mode")

		// Health check flags
		healthHTTP, _ := cmd.Flags().GetString("health-http")
		healthTCP, _ := cmd.Flags().GetString("health-tcp")
		healthCmd, _ := cmd.Flags().GetStringSlice("health-cmd")
		healthInterval, _ := cmd.Flags().GetInt("health-interval")
		healthTimeout, _ := cmd.Flags().GetInt("health-timeout")
		healthRetries, _ := cmd.Flags().GetInt("health-retries")

		// Resource limit flags
		cpus, _ := cmd.Flags().GetFloat64("cpus")
		memory, _ := cmd.Flags().GetString("memory")

		// Graceful shutdown flags
		stopTimeout, _ := cmd.Flags().GetInt("stop-timeout")

		// Parse env vars
		env := make(map[string]string)
		for _, e := range envVars {
			// Split on first = only
			parts := splitEnv(e)
			if len(parts) == 2 {
				env[parts[0]] = parts[1]
			}
		}

		// Parse port mappings
		ports, err := parsePortMappings(publishPorts, publishMode)
		if err != nil {
			return fmt.Errorf("failed to parse port mappings: %v", err)
		}

		// Connect to manager
		c, err := client.NewClient(manager)
		if err != nil {
			return fmt.Errorf("failed to connect to manager: %v", err)
		}
		defer c.Close()

		// Build service request
		req := &proto.CreateServiceRequest{
			Name:     name,
			Image:    image,
			Replicas: int32(replicas),
			Mode:     "replicated",
			Env:      env,
			Ports:    ports,
		}

		// Add health check if specified
		if healthHTTP != "" || healthTCP != "" || len(healthCmd) > 0 {
			req.HealthCheck = buildHealthCheck(healthHTTP, healthTCP, healthCmd, healthInterval, healthTimeout, healthRetries)
		}

		// Add resource limits if specified
		if cpus > 0 || memory != "" {
			resources := &proto.ResourceRequirements{}

			if cpus > 0 {
				resources.CpuShares = int64(cpus * 1024) // Convert cores to shares
			}

			if memory != "" {
				memBytes, err := parseMemory(memory)
				if err != nil {
					return fmt.Errorf("invalid memory format: %v", err)
				}
				resources.MemoryBytes = memBytes
			}

			req.Resources = resources
		}

		// Add stop timeout if specified
		if stopTimeout > 0 {
			req.StopTimeout = int32(stopTimeout)
		}

		// Create service
		service, err := c.CreateServiceWithOptions(req)
		if err != nil {
			return fmt.Errorf("failed to create service: %v", err)
		}

		fmt.Printf("✓ Service created: %s\n", service.Name)
		fmt.Printf("  ID: %s\n", service.Id)
		fmt.Printf("  Image: %s\n", service.Image)
		fmt.Printf("  Replicas: %d\n", service.Replicas)
		if len(service.Ports) > 0 {
			fmt.Printf("  Published Ports:\n")
			for _, port := range service.Ports {
				mode := "host"
				if port.PublishMode == proto.PortMapping_INGRESS {
					mode = "ingress"
				}
				fmt.Printf("    %d:%d/%s (%s)\n", port.HostPort, port.ContainerPort, port.Protocol, mode)
			}
		}
		if service.HealthCheck != nil {
			fmt.Printf("  Health Check: %s\n", service.HealthCheck.Type)
		}
		if service.Resources != nil {
			if service.Resources.CpuShares > 0 {
				cpus := float64(service.Resources.CpuShares) / 1024.0
				fmt.Printf("  CPU Limit: %.2f cores\n", cpus)
			}
			if service.Resources.MemoryBytes > 0 {
				fmt.Printf("  Memory Limit: %s\n", formatBytes(service.Resources.MemoryBytes))
			}
		}
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

	// Port publishing flags
	serviceCreateCmd.Flags().StringSliceP("publish", "p", []string{}, "Publish ports (e.g., 8080:80, 443:443/tcp)")
	serviceCreateCmd.Flags().String("publish-mode", "host", "Port publish mode: 'host' or 'ingress'")

	// Health check flags
	serviceCreateCmd.Flags().String("health-http", "", "HTTP health check path (e.g., /health)")
	serviceCreateCmd.Flags().String("health-tcp", "", "TCP health check port (e.g., 8080 or :8080)")
	serviceCreateCmd.Flags().StringSlice("health-cmd", []string{}, "Exec health check command (e.g., pg_isready)")
	serviceCreateCmd.Flags().Int("health-interval", 30, "Health check interval in seconds")
	serviceCreateCmd.Flags().Int("health-timeout", 10, "Health check timeout in seconds")
	serviceCreateCmd.Flags().Int("health-retries", 3, "Health check retries before marking unhealthy")

	// Resource limit flags
	serviceCreateCmd.Flags().Float64("cpus", 0, "CPU limit in cores (e.g., 0.5, 1.0, 2.0)")
	serviceCreateCmd.Flags().String("memory", "", "Memory limit (e.g., 512m, 1g, 2g)")

	// Graceful shutdown flags
	serviceCreateCmd.Flags().Int("stop-timeout", 10, "Seconds to wait before force-killing container (default: 10)")

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

var secretCreateCmd = &cobra.Command{
	Use:   "create NAME",
	Short: "Create a new secret",
	Long: `Create a new secret from a file or literal value.

Examples:
  # Create secret from file
  warren secret create db-password --from-file ./password.txt

  # Create secret from literal value
  warren secret create api-key --from-literal "my-secret-key"

  # Create secret from stdin
  echo "my-secret" | warren secret create db-pass --from-stdin`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		managerAddr, _ := cmd.Flags().GetString("manager")
		fromFile, _ := cmd.Flags().GetString("from-file")
		fromLiteral, _ := cmd.Flags().GetString("from-literal")
		fromStdin, _ := cmd.Flags().GetBool("from-stdin")

		// Determine data source
		var data []byte
		var err error

		if fromFile != "" {
			data, err = os.ReadFile(fromFile)
			if err != nil {
				return fmt.Errorf("failed to read file: %w", err)
			}
		} else if fromLiteral != "" {
			data = []byte(fromLiteral)
		} else if fromStdin {
			data, err = os.ReadFile("/dev/stdin")
			if err != nil {
				return fmt.Errorf("failed to read stdin: %w", err)
			}
		} else {
			return fmt.Errorf("must specify one of: --from-file, --from-literal, or --from-stdin")
		}

		// Create client and secret
		c, err := client.NewClient(managerAddr)
		if err != nil {
			return fmt.Errorf("failed to connect to manager: %w", err)
		}
		defer c.Close()

		secret, err := c.CreateSecret(name, data)
		if err != nil {
			return fmt.Errorf("failed to create secret: %w", err)
		}

		fmt.Printf("Secret created: %s\n", secret.Name)
		fmt.Printf("  ID: %s\n", secret.Id)
		return nil
	},
}

var secretListCmd = &cobra.Command{
	Use:   "list",
	Short: "List secrets",
	RunE: func(cmd *cobra.Command, args []string) error {
		managerAddr, _ := cmd.Flags().GetString("manager")

		c, err := client.NewClient(managerAddr)
		if err != nil {
			return fmt.Errorf("failed to connect to manager: %w", err)
		}
		defer c.Close()

		secrets, err := c.ListSecrets()
		if err != nil {
			return fmt.Errorf("failed to list secrets: %w", err)
		}

		if len(secrets) == 0 {
			fmt.Println("No secrets found")
			return nil
		}

		fmt.Printf("%-20s %-40s %s\n", "NAME", "ID", "CREATED")
		fmt.Println(strings.Repeat("-", 80))
		for _, secret := range secrets {
			createdAt := secret.CreatedAt.AsTime().Format("2006-01-02 15:04:05")
			fmt.Printf("%-20s %-40s %s\n",
				truncate(secret.Name, 20),
				truncate(secret.Id, 40),
				createdAt,
			)
		}

		return nil
	},
}

var secretInspectCmd = &cobra.Command{
	Use:   "inspect NAME",
	Short: "Display detailed information about a secret",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		managerAddr, _ := cmd.Flags().GetString("manager")

		c, err := client.NewClient(managerAddr)
		if err != nil {
			return fmt.Errorf("failed to connect to manager: %w", err)
		}
		defer c.Close()

		secret, err := c.GetSecretByName(name)
		if err != nil {
			return fmt.Errorf("failed to get secret: %w", err)
		}

		fmt.Printf("Name: %s\n", secret.Name)
		fmt.Printf("ID: %s\n", secret.Id)
		fmt.Printf("Created: %s\n", secret.CreatedAt.AsTime().Format("2006-01-02 15:04:05"))
		fmt.Printf("\nNote: Secret data is encrypted and not displayed for security.\n")

		return nil
	},
}

var secretDeleteCmd = &cobra.Command{
	Use:   "delete NAME",
	Short: "Delete a secret",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		managerAddr, _ := cmd.Flags().GetString("manager")

		c, err := client.NewClient(managerAddr)
		if err != nil {
			return fmt.Errorf("failed to connect to manager: %w", err)
		}
		defer c.Close()

		if err := c.DeleteSecret(name); err != nil {
			return fmt.Errorf("failed to delete secret: %w", err)
		}

		fmt.Printf("Secret deleted: %s\n", name)
		return nil
	},
}

func init() {
	// secret create flags
	secretCreateCmd.Flags().String("manager", "localhost:7946", "Manager address")
	secretCreateCmd.Flags().String("from-file", "", "Read secret data from file")
	secretCreateCmd.Flags().String("from-literal", "", "Use literal string as secret data")
	secretCreateCmd.Flags().Bool("from-stdin", false, "Read secret data from stdin")

	// secret list flags
	secretListCmd.Flags().String("manager", "localhost:7946", "Manager address")

	// secret inspect flags
	secretInspectCmd.Flags().String("manager", "localhost:7946", "Manager address")

	// secret delete flags
	secretDeleteCmd.Flags().String("manager", "localhost:7946", "Manager address")

	secretCmd.AddCommand(secretCreateCmd)
	secretCmd.AddCommand(secretListCmd)
	secretCmd.AddCommand(secretInspectCmd)
	secretCmd.AddCommand(secretDeleteCmd)
}

// Volume commands
var volumeCmd = &cobra.Command{
	Use:   "volume",
	Short: "Manage volumes",
}

var volumeCreateCmd = &cobra.Command{
	Use:   "create NAME",
	Short: "Create a new volume",
	Long: `Create a new volume with specified driver.

Examples:
  # Create local volume
  warren volume create my-data --driver local

  # Create volume with options
  warren volume create my-data --driver local --opt size=10GB`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		managerAddr, _ := cmd.Flags().GetString("manager")
		driver, _ := cmd.Flags().GetString("driver")
		opts, _ := cmd.Flags().GetStringToString("opt")

		// Create client and volume
		c, err := client.NewClient(managerAddr)
		if err != nil {
			return fmt.Errorf("failed to connect to manager: %w", err)
		}
		defer c.Close()

		volume, err := c.CreateVolume(name, driver, opts)
		if err != nil {
			return fmt.Errorf("failed to create volume: %w", err)
		}

		fmt.Printf("Volume created: %s\n", volume.Name)
		fmt.Printf("  ID: %s\n", volume.Id)
		fmt.Printf("  Driver: %s\n", volume.Driver)
		return nil
	},
}

var volumeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List volumes",
	RunE: func(cmd *cobra.Command, args []string) error {
		managerAddr, _ := cmd.Flags().GetString("manager")

		c, err := client.NewClient(managerAddr)
		if err != nil {
			return fmt.Errorf("failed to connect to manager: %w", err)
		}
		defer c.Close()

		volumes, err := c.ListVolumes()
		if err != nil {
			return fmt.Errorf("failed to list volumes: %w", err)
		}

		if len(volumes) == 0 {
			fmt.Println("No volumes found")
			return nil
		}

		fmt.Printf("%-20s %-40s %-10s %-15s %s\n", "NAME", "ID", "DRIVER", "NODE", "CREATED")
		fmt.Println(strings.Repeat("-", 100))
		for _, volume := range volumes {
			createdAt := volume.CreatedAt.AsTime().Format("2006-01-02 15:04:05")
			nodeID := volume.NodeId
			if nodeID == "" {
				nodeID = "<none>"
			}
			fmt.Printf("%-20s %-40s %-10s %-15s %s\n",
				truncate(volume.Name, 20),
				truncate(volume.Id, 40),
				volume.Driver,
				truncate(nodeID, 15),
				createdAt,
			)
		}

		return nil
	},
}

var volumeInspectCmd = &cobra.Command{
	Use:   "inspect NAME",
	Short: "Display detailed information about a volume",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		managerAddr, _ := cmd.Flags().GetString("manager")

		c, err := client.NewClient(managerAddr)
		if err != nil {
			return fmt.Errorf("failed to connect to manager: %w", err)
		}
		defer c.Close()

		volume, err := c.GetVolumeByName(name)
		if err != nil {
			return fmt.Errorf("failed to get volume: %w", err)
		}

		fmt.Printf("Name: %s\n", volume.Name)
		fmt.Printf("ID: %s\n", volume.Id)
		fmt.Printf("Driver: %s\n", volume.Driver)
		if volume.NodeId != "" {
			fmt.Printf("Node: %s\n", volume.NodeId)
		}
		if volume.MountPath != "" {
			fmt.Printf("Mount Path: %s\n", volume.MountPath)
		}
		fmt.Printf("Created: %s\n", volume.CreatedAt.AsTime().Format("2006-01-02 15:04:05"))

		if len(volume.DriverOpts) > 0 {
			fmt.Println("\nDriver Options:")
			for k, v := range volume.DriverOpts {
				fmt.Printf("  %s: %s\n", k, v)
			}
		}

		if len(volume.Labels) > 0 {
			fmt.Println("\nLabels:")
			for k, v := range volume.Labels {
				fmt.Printf("  %s: %s\n", k, v)
			}
		}

		return nil
	},
}

var volumeDeleteCmd = &cobra.Command{
	Use:   "delete NAME",
	Short: "Delete a volume",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		managerAddr, _ := cmd.Flags().GetString("manager")

		c, err := client.NewClient(managerAddr)
		if err != nil {
			return fmt.Errorf("failed to connect to manager: %w", err)
		}
		defer c.Close()

		if err := c.DeleteVolume(name); err != nil {
			return fmt.Errorf("failed to delete volume: %w", err)
		}

		fmt.Printf("Volume deleted: %s\n", name)
		return nil
	},
}

func init() {
	// volume create flags
	volumeCreateCmd.Flags().String("manager", "localhost:7946", "Manager address")
	volumeCreateCmd.Flags().String("driver", "local", "Volume driver (local)")
	volumeCreateCmd.Flags().StringToString("opt", map[string]string{}, "Driver-specific options")

	// volume list flags
	volumeListCmd.Flags().String("manager", "localhost:7946", "Manager address")

	// volume inspect flags
	volumeInspectCmd.Flags().String("manager", "localhost:7946", "Manager address")

	// volume delete flags
	volumeDeleteCmd.Flags().String("manager", "localhost:7946", "Manager address")

	volumeCmd.AddCommand(volumeCreateCmd)
	volumeCmd.AddCommand(volumeListCmd)
	volumeCmd.AddCommand(volumeInspectCmd)
	volumeCmd.AddCommand(volumeDeleteCmd)
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

// buildHealthCheck creates a HealthCheck proto message from CLI flags
func buildHealthCheck(httpPath, tcpPort string, execCmd []string, interval, timeout, retries int) *proto.HealthCheck {
	// Set defaults if not specified
	if interval == 0 {
		interval = 30
	}
	if timeout == 0 {
		timeout = 10
	}
	if retries == 0 {
		retries = 3
	}

	hc := &proto.HealthCheck{
		IntervalSeconds: int32(interval),
		TimeoutSeconds:  int32(timeout),
		Retries:         int32(retries),
	}

	// Determine type and build type-specific config
	if httpPath != "" {
		// Parse HTTP path (format: [scheme://]host:port/path)
		// For simplicity, assume format is :port/path or just /path (default port 80)
		hc.Type = proto.HealthCheck_HTTP
		hc.Http = &proto.HTTPHealthCheck{
			Path:          httpPath,
			Port:          80, // Default, will be overridden if port specified
			Scheme:        "http",
			StatusCodeMin: 200,
			StatusCodeMax: 399,
		}
	} else if tcpPort != "" {
		// Parse TCP port (format: :port or port)
		hc.Type = proto.HealthCheck_TCP
		port := int32(80) // Default
		if tcpPort != "" {
			fmt.Sscanf(tcpPort, ":%d", &port)
			if port == 80 { // If didn't match :port format, try just port
				fmt.Sscanf(tcpPort, "%d", &port)
			}
		}
		hc.Tcp = &proto.TCPHealthCheck{
			Port: port,
		}
	} else if len(execCmd) > 0 {
		hc.Type = proto.HealthCheck_EXEC
		hc.Exec = &proto.ExecHealthCheck{
			Command: execCmd,
		}
	}

	return hc
}

// parsePortMappings parses port mapping strings like "8080:80" or "443:443/tcp"
func parsePortMappings(portSpecs []string, defaultMode string) ([]*proto.PortMapping, error) {
	if len(portSpecs) == 0 {
		return nil, nil
	}

	var ports []*proto.PortMapping

	// Determine publish mode
	publishMode := proto.PortMapping_HOST
	if defaultMode == "ingress" {
		publishMode = proto.PortMapping_INGRESS
	}

	for _, spec := range portSpecs {
		port, err := parsePortSpec(spec, publishMode)
		if err != nil {
			return nil, fmt.Errorf("invalid port spec '%s': %v", spec, err)
		}
		ports = append(ports, port)
	}

	return ports, nil
}

// parsePortSpec parses a single port specification
// Formats supported:
//   - "8080:80"       -> host:container, tcp
//   - "8080:80/tcp"   -> host:container/protocol
//   - "8080:80/udp"   -> host:container/protocol
func parsePortSpec(spec string, publishMode proto.PortMapping_PublishMode) (*proto.PortMapping, error) {
	// Default protocol
	protocol := "tcp"

	// Check for protocol suffix
	parts := strings.Split(spec, "/")
	if len(parts) == 2 {
		spec = parts[0]
		protocol = strings.ToLower(parts[1])
		if protocol != "tcp" && protocol != "udp" {
			return nil, fmt.Errorf("protocol must be 'tcp' or 'udp', got '%s'", protocol)
		}
	} else if len(parts) > 2 {
		return nil, fmt.Errorf("invalid format, too many '/' separators")
	}

	// Parse host:container ports
	portParts := strings.Split(spec, ":")
	if len(portParts) != 2 {
		return nil, fmt.Errorf("format must be 'hostPort:containerPort'")
	}

	hostPort := 0
	containerPort := 0

	// Parse host port
	_, err := fmt.Sscanf(portParts[0], "%d", &hostPort)
	if err != nil || hostPort <= 0 || hostPort > 65535 {
		return nil, fmt.Errorf("invalid host port: must be 1-65535")
	}

	// Parse container port
	_, err = fmt.Sscanf(portParts[1], "%d", &containerPort)
	if err != nil || containerPort <= 0 || containerPort > 65535 {
		return nil, fmt.Errorf("invalid container port: must be 1-65535")
	}

	return &proto.PortMapping{
		ContainerPort: int32(containerPort),
		HostPort:      int32(hostPort),
		Protocol:      protocol,
		PublishMode:   publishMode,
	}, nil
}

// parseMemory parses a memory string (e.g., "512m", "1g", "2048k") into bytes
func parseMemory(mem string) (int64, error) {
	if mem == "" {
		return 0, nil
	}

	mem = strings.ToLower(strings.TrimSpace(mem))

	// Find the numeric part and unit
	var value float64
	var unit string

	// Try to parse with unit
	_, err := fmt.Sscanf(mem, "%f%s", &value, &unit)
	if err != nil {
		// Try parsing as just a number (bytes)
		_, err = fmt.Sscanf(mem, "%f", &value)
		if err != nil {
			return 0, fmt.Errorf("invalid memory format: %s (use format like '512m', '1g', '2048k')", mem)
		}
		return int64(value), nil
	}

	// Convert to bytes based on unit
	var bytes int64
	switch unit {
	case "b", "":
		bytes = int64(value)
	case "k", "kb":
		bytes = int64(value * 1024)
	case "m", "mb":
		bytes = int64(value * 1024 * 1024)
	case "g", "gb":
		bytes = int64(value * 1024 * 1024 * 1024)
	default:
		return 0, fmt.Errorf("invalid memory unit: %s (use b, k/kb, m/mb, g/gb)", unit)
	}

	if bytes <= 0 {
		return 0, fmt.Errorf("memory must be positive")
	}

	return bytes, nil
}

// formatBytes formats a byte count into human-readable format
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGT"[exp])
}

// Ingress commands
var ingressCmd = &cobra.Command{
	Use:   "ingress",
	Short: "Manage ingress rules",
}

var ingressCreateCmd = &cobra.Command{
	Use:   "create NAME",
	Short: "Create a new ingress",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		host, _ := cmd.Flags().GetString("host")
		path, _ := cmd.Flags().GetString("path")
		pathType, _ := cmd.Flags().GetString("path-type")
		serviceName, _ := cmd.Flags().GetString("service")
		servicePort, _ := cmd.Flags().GetInt("port")
		manager, _ := cmd.Flags().GetString("manager")
		tls, _ := cmd.Flags().GetBool("tls")
		tlsEmail, _ := cmd.Flags().GetString("tls-email")

		// Validate required flags
		if serviceName == "" {
			return fmt.Errorf("--service is required")
		}
		if servicePort == 0 {
			return fmt.Errorf("--port is required")
		}
		if tls && tlsEmail == "" {
			return fmt.Errorf("--tls-email is required when --tls is enabled")
		}
		if tls && host == "" {
			return fmt.Errorf("--host is required when --tls is enabled (Let's Encrypt requires a specific domain)")
		}

		// Default path
		if path == "" {
			path = "/"
		}
		// Default path type
		if pathType == "" {
			pathType = "Prefix"
		}

		// Connect to manager
		c, err := client.NewClient(manager)
		if err != nil {
			return fmt.Errorf("failed to connect to manager: %v", err)
		}
		defer c.Close()

		// Build ingress request
		req := &proto.CreateIngressRequest{
			Name: name,
			Rules: []*proto.IngressRule{
				{
					Host: host,
					Paths: []*proto.IngressPath{
						{
							Path:     path,
							PathType: pathType,
							Backend: &proto.IngressBackend{
								ServiceName: serviceName,
								Port:        int32(servicePort),
							},
						},
					},
				},
			},
		}

		// Add TLS configuration if enabled
		if tls {
			req.Tls = &proto.IngressTLS{
				Enabled: true,
				AutoTls: true,
				Email:   tlsEmail,
				Hosts:   []string{host},
			}
		}

		// Create ingress
		ingress, err := c.CreateIngress(req)
		if err != nil {
			return fmt.Errorf("failed to create ingress: %v", err)
		}

		fmt.Printf("Ingress %s created successfully\n", ingress.Name)
		fmt.Printf("ID: %s\n", ingress.Id)
		if host != "" {
			fmt.Printf("Host: %s\n", host)
		}
		fmt.Printf("Path: %s (type: %s)\n", path, pathType)
		fmt.Printf("Backend: %s:%d\n", serviceName, servicePort)
		if tls {
			fmt.Printf("TLS: Enabled (Let's Encrypt)\n")
			fmt.Printf("Email: %s\n", tlsEmail)
			fmt.Printf("Note: Certificate issuance is in progress. Check certificate list for status.\n")
		}

		return nil
	},
}

var ingressListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all ingresses",
	Aliases: []string{"ls"},
	RunE: func(cmd *cobra.Command, args []string) error {
		manager, _ := cmd.Flags().GetString("manager")

		// Connect to manager
		c, err := client.NewClient(manager)
		if err != nil {
			return fmt.Errorf("failed to connect to manager: %v", err)
		}
		defer c.Close()

		// List ingresses
		ingresses, err := c.ListIngresses()
		if err != nil {
			return fmt.Errorf("failed to list ingresses: %v", err)
		}

		if len(ingresses) == 0 {
			fmt.Println("No ingresses found")
			return nil
		}

		// Print table header
		fmt.Printf("%-20s %-30s %-15s %-30s\n", "NAME", "HOST", "PATH", "BACKEND")
		fmt.Println(strings.Repeat("-", 95))

		// Print each ingress
		for _, ingress := range ingresses {
			for _, rule := range ingress.Rules {
				host := rule.Host
				if host == "" {
					host = "*"
				}
				for _, path := range rule.Paths {
					backend := fmt.Sprintf("%s:%d", path.Backend.ServiceName, path.Backend.Port)
					fmt.Printf("%-20s %-30s %-15s %-30s\n",
						ingress.Name,
						host,
						path.Path,
						backend,
					)
				}
			}
		}

		return nil
	},
}

var ingressInspectCmd = &cobra.Command{
	Use:   "inspect NAME",
	Short: "Inspect an ingress",
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

		// Get ingress
		ingress, err := c.GetIngress(&proto.GetIngressRequest{
			Name: name,
		})
		if err != nil {
			return fmt.Errorf("failed to get ingress: %v", err)
		}

		// Print ingress details
		fmt.Printf("Name: %s\n", ingress.Name)
		fmt.Printf("ID: %s\n", ingress.Id)
		fmt.Printf("Created: %s\n", ingress.CreatedAt.AsTime().Format(time.RFC3339))
		fmt.Printf("Updated: %s\n", ingress.UpdatedAt.AsTime().Format(time.RFC3339))
		fmt.Println()

		fmt.Println("Rules:")
		for i, rule := range ingress.Rules {
			fmt.Printf("  Rule %d:\n", i+1)
			if rule.Host != "" {
				fmt.Printf("    Host: %s\n", rule.Host)
			} else {
				fmt.Printf("    Host: * (all hosts)\n")
			}
			fmt.Println("    Paths:")
			for j, path := range rule.Paths {
				fmt.Printf("      Path %d:\n", j+1)
				fmt.Printf("        Path: %s\n", path.Path)
				fmt.Printf("        Type: %s\n", path.PathType)
				fmt.Printf("        Backend: %s:%d\n", path.Backend.ServiceName, path.Backend.Port)
			}
		}

		return nil
	},
}

var ingressDeleteCmd = &cobra.Command{
	Use:   "delete NAME",
	Short: "Delete an ingress",
	Aliases: []string{"rm"},
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

		// Delete ingress
		err = c.DeleteIngress(&proto.DeleteIngressRequest{
			Name: name,
		})
		if err != nil {
			return fmt.Errorf("failed to delete ingress: %v", err)
		}

		fmt.Printf("Ingress %s deleted successfully\n", name)
		return nil
	},
}

// Certificate commands

var certificateCmd = &cobra.Command{
	Use:     "certificate",
	Aliases: []string{"cert", "certs"},
	Short:   "Manage TLS certificates",
}

var certificateCreateCmd = &cobra.Command{
	Use:   "create NAME",
	Short: "Create a new TLS certificate",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		certFile, _ := cmd.Flags().GetString("cert")
		keyFile, _ := cmd.Flags().GetString("key")
		hosts, _ := cmd.Flags().GetStringSlice("hosts")
		managerAddr, _ := cmd.Flags().GetString("manager")

		// Validate flags
		if certFile == "" || keyFile == "" {
			return fmt.Errorf("both --cert and --key are required")
		}
		if len(hosts) == 0 {
			return fmt.Errorf("at least one host is required (--hosts)")
		}

		// Read certificate and key files
		certPEM, err := os.ReadFile(certFile)
		if err != nil {
			return fmt.Errorf("failed to read certificate file: %v", err)
		}

		keyPEM, err := os.ReadFile(keyFile)
		if err != nil {
			return fmt.Errorf("failed to read key file: %v", err)
		}

		// Connect to manager
		c, err := client.NewClient(managerAddr)
		if err != nil {
			return fmt.Errorf("failed to connect to manager: %v", err)
		}
		defer c.Close()

		// Create certificate
		req := &proto.CreateTLSCertificateRequest{
			Name:    name,
			Hosts:   hosts,
			CertPem: certPEM,
			KeyPem:  keyPEM,
		}

		cert, err := c.CreateTLSCertificate(req)
		if err != nil {
			return fmt.Errorf("failed to create certificate: %v", err)
		}

		fmt.Printf("Certificate %s created successfully\n", cert.Name)
		fmt.Printf("  ID: %s\n", cert.Id)
		fmt.Printf("  Hosts: %v\n", cert.Hosts)
		fmt.Printf("  Issuer: %s\n", cert.Issuer)
		fmt.Printf("  Valid from: %s\n", cert.NotBefore.AsTime().Format(time.RFC3339))
		fmt.Printf("  Valid until: %s\n", cert.NotAfter.AsTime().Format(time.RFC3339))

		return nil
	},
}

var certificateListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all TLS certificates",
	RunE: func(cmd *cobra.Command, args []string) error {
		managerAddr, _ := cmd.Flags().GetString("manager")

		// Connect to manager
		c, err := client.NewClient(managerAddr)
		if err != nil {
			return fmt.Errorf("failed to connect to manager: %v", err)
		}
		defer c.Close()

		// List certificates
		certs, err := c.ListTLSCertificates()
		if err != nil {
			return fmt.Errorf("failed to list certificates: %v", err)
		}

		if len(certs) == 0 {
			fmt.Println("No certificates found")
			return nil
		}

		// Print table header
		fmt.Printf("%-20s %-40s %-25s %-25s\n", "NAME", "HOSTS", "VALID UNTIL", "ISSUER")
		fmt.Println(strings.Repeat("-", 110))

		// Print each certificate
		for _, cert := range certs {
			hostsStr := strings.Join(cert.Hosts, ", ")
			if len(hostsStr) > 38 {
				hostsStr = hostsStr[:35] + "..."
			}

			issuer := cert.Issuer
			if len(issuer) > 23 {
				issuer = issuer[:20] + "..."
			}

			validUntil := cert.NotAfter.AsTime().Format("2006-01-02 15:04")

			fmt.Printf("%-20s %-40s %-25s %-25s\n", cert.Name, hostsStr, validUntil, issuer)
		}

		return nil
	},
}

var certificateInspectCmd = &cobra.Command{
	Use:   "inspect NAME",
	Short: "Inspect a TLS certificate",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		managerAddr, _ := cmd.Flags().GetString("manager")

		// Connect to manager
		c, err := client.NewClient(managerAddr)
		if err != nil {
			return fmt.Errorf("failed to connect to manager: %v", err)
		}
		defer c.Close()

		// Get certificate
		cert, err := c.GetTLSCertificate(&proto.GetTLSCertificateRequest{
			Name: name,
		})
		if err != nil {
			return fmt.Errorf("failed to get certificate: %v", err)
		}

		// Print certificate details
		fmt.Printf("Name: %s\n", cert.Name)
		fmt.Printf("ID: %s\n", cert.Id)
		fmt.Printf("Issuer: %s\n", cert.Issuer)
		fmt.Printf("Hosts:\n")
		for _, host := range cert.Hosts {
			fmt.Printf("  - %s\n", host)
		}
		fmt.Printf("Valid from: %s\n", cert.NotBefore.AsTime().Format(time.RFC3339))
		fmt.Printf("Valid until: %s\n", cert.NotAfter.AsTime().Format(time.RFC3339))
		fmt.Printf("Created: %s\n", cert.CreatedAt.AsTime().Format(time.RFC3339))
		fmt.Printf("Updated: %s\n", cert.UpdatedAt.AsTime().Format(time.RFC3339))

		return nil
	},
}

var certificateDeleteCmd = &cobra.Command{
	Use:     "delete NAME",
	Aliases: []string{"rm"},
	Short:   "Delete a TLS certificate",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		managerAddr, _ := cmd.Flags().GetString("manager")

		// Connect to manager
		c, err := client.NewClient(managerAddr)
		if err != nil {
			return fmt.Errorf("failed to connect to manager: %v", err)
		}
		defer c.Close()

		// Delete certificate
		err = c.DeleteTLSCertificate(&proto.DeleteTLSCertificateRequest{
			Name: name,
		})
		if err != nil {
			return fmt.Errorf("failed to delete certificate: %v", err)
		}

		fmt.Printf("Certificate %s deleted successfully\n", name)
		return nil
	},
}

func init() {
	// Ingress create command
	ingressCreateCmd.Flags().String("manager", "localhost:2377", "Manager address")
	ingressCreateCmd.Flags().String("host", "", "Host to match (e.g., api.example.com, leave empty for all hosts)")
	ingressCreateCmd.Flags().String("path", "/", "Path to match (default: /)")
	ingressCreateCmd.Flags().String("path-type", "Prefix", "Path type: Prefix or Exact (default: Prefix)")
	ingressCreateCmd.Flags().String("service", "", "Backend service name (required)")
	ingressCreateCmd.Flags().Int("port", 0, "Backend service port (required)")
	ingressCreateCmd.Flags().Bool("tls", false, "Enable TLS with Let's Encrypt")
	ingressCreateCmd.Flags().String("tls-email", "", "Email for Let's Encrypt notifications (required with --tls)")
	ingressCreateCmd.MarkFlagRequired("service")
	ingressCreateCmd.MarkFlagRequired("port")

	// Ingress list command
	ingressListCmd.Flags().String("manager", "localhost:2377", "Manager address")

	// Ingress inspect command
	ingressInspectCmd.Flags().String("manager", "localhost:2377", "Manager address")

	// Ingress delete command
	ingressDeleteCmd.Flags().String("manager", "localhost:2377", "Manager address")

	// Add ingress subcommands
	ingressCmd.AddCommand(ingressCreateCmd)
	ingressCmd.AddCommand(ingressListCmd)
	ingressCmd.AddCommand(ingressInspectCmd)
	ingressCmd.AddCommand(ingressDeleteCmd)

	// Certificate create command
	certificateCreateCmd.Flags().String("manager", "localhost:2377", "Manager address")
	certificateCreateCmd.Flags().String("cert", "", "Path to certificate file (PEM format) (required)")
	certificateCreateCmd.Flags().String("key", "", "Path to private key file (PEM format) (required)")
	certificateCreateCmd.Flags().StringSlice("hosts", []string{}, "Hostnames covered by this certificate (required)")
	certificateCreateCmd.MarkFlagRequired("cert")
	certificateCreateCmd.MarkFlagRequired("key")
	certificateCreateCmd.MarkFlagRequired("hosts")

	// Certificate list command
	certificateListCmd.Flags().String("manager", "localhost:2377", "Manager address")

	// Certificate inspect command
	certificateInspectCmd.Flags().String("manager", "localhost:2377", "Manager address")

	// Certificate delete command
	certificateDeleteCmd.Flags().String("manager", "localhost:2377", "Manager address")

	// Add certificate subcommands
	certificateCmd.AddCommand(certificateCreateCmd)
	certificateCmd.AddCommand(certificateListCmd)
	certificateCmd.AddCommand(certificateInspectCmd)
	certificateCmd.AddCommand(certificateDeleteCmd)
}
