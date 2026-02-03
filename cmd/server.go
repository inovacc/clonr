package cmd

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/inovacc/clonr/internal/actionsdb"
	"github.com/inovacc/clonr/internal/core"
	"github.com/inovacc/clonr/internal/process"
	"github.com/inovacc/clonr/internal/server/grpc"
	"github.com/inovacc/clonr/internal/store"
	"github.com/spf13/cobra"
)

var actionsWorker *actionsdb.Worker
var rotationScheduler *grpc.RotationScheduler

var (
	serverPort        int
	serverIdleTimeout time.Duration
	serverMaxRuntime  time.Duration
	procs             = process.NewProcess()
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Server management commands",
	Long:  `Manage the Clonr gRPC server. Use 'clonr server start' to start the server.`,
}

var serverStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the gRPC server",
	Long: `Start the Clonr gRPC server on the configured port.

The server will shutdown when any of these conditions are met:
- Interrupted with Ctrl+C or SIGTERM
- Idle timeout reached (default: 5 minutes of no requests)
- Max runtime reached (default: 1 hour)

Use --idle-timeout=0 and --max-runtime=0 to run indefinitely.`,
	RunE: runServerStart,
}

var serverStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the running gRPC server",
	Long:  `Stop the Clonr gRPC server by sending a termination signal to the running process.`,
	RunE:  runServerStop,
}

var serverRestartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart the gRPC server",
	Long:  `Restart the Clonr gRPC server by stopping the current instance and starting a new one.`,
	RunE:  runServerRestart,
}

var serverStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show server status",
	Long:  `Show the current status of the Clonr gRPC server.`,
	RunE:  runServerStatus,
}

var (
	stopTimeout    time.Duration
	restartTimeout time.Duration
)

func init() {
	rootCmd.AddCommand(serverCmd)
	serverCmd.AddCommand(serverStartCmd)
	serverCmd.AddCommand(serverStopCmd)
	serverCmd.AddCommand(serverRestartCmd)
	serverCmd.AddCommand(serverStatusCmd)

	serverStartCmd.Flags().IntVarP(&serverPort, "port", "p", 50051, "Port to listen on")
	serverStartCmd.Flags().DurationVar(&serverIdleTimeout, "idle-timeout", 5*time.Minute, "Shutdown after being idle for this duration (0 to disable)")
	serverStartCmd.Flags().DurationVar(&serverMaxRuntime, "max-runtime", 1*time.Hour, "Maximum server runtime before auto-shutdown (0 to disable)")

	serverStopCmd.Flags().DurationVar(&stopTimeout, "timeout", 30*time.Second, "Timeout waiting for server to stop")

	serverRestartCmd.Flags().IntVarP(&serverPort, "port", "p", 50051, "Port to listen on")
	serverRestartCmd.Flags().DurationVar(&serverIdleTimeout, "idle-timeout", 5*time.Minute, "Shutdown after being idle for this duration (0 to disable)")
	serverRestartCmd.Flags().DurationVar(&serverMaxRuntime, "max-runtime", 1*time.Hour, "Maximum server runtime before auto-shutdown (0 to disable)")
	serverRestartCmd.Flags().DurationVar(&restartTimeout, "timeout", 30*time.Second, "Timeout waiting for server to stop before restart")
}

func runServerStart(cmd *cobra.Command, args []string) error {
	_, _ = cmd, args

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Check if the server is already running - silent abort if so
	if grpc.IsServerRunning() != nil {
		return nil
	}

	db := store.GetDB()

	// Use configured port if default not overridden
	if serverPort == 50051 {
		cfg, err := db.GetConfig()
		if err == nil && cfg.ServerPort > 0 && cfg.ServerPort != 4000 {
			serverPort = cfg.ServerPort
		}
	}

	addr := fmt.Sprintf(":%d", serverPort)

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	// Write a server info file for client discovery
	if err := grpc.WriteServerInfo(serverPort); err != nil {
		log.Printf("Warning: failed to write server info file: %v", err)
	} else {
		log.Printf("Server info written to local data directory")
	}

	srvWithHealth := grpc.NewServer(db, serverIdleTimeout)

	// Start idle tracker if enabled
	if srvWithHealth.IdleTracker.IsEnabled() {
		go srvWithHealth.IdleTracker.Start()

		log.Printf("Idle timeout enabled: server will shutdown after %v of inactivity", serverIdleTimeout)
	}

	// Setup max runtime timer if enabled
	var maxRuntimeTimer *time.Timer
	maxRuntimeChan := make(chan struct{})
	if serverMaxRuntime > 0 {
		maxRuntimeTimer = time.AfterFunc(serverMaxRuntime, func() {
			close(maxRuntimeChan)
		})
		log.Printf("Max runtime enabled: server will shutdown after %v", serverMaxRuntime)
	}

	go func() {
		log.Printf("Starting Clonr gRPC server on %s", addr)

		if err := srvWithHealth.GRPCServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	// Start GitHub Actions monitoring worker
	if err := startActionsWorker(); err != nil {
		log.Printf("Warning: failed to start actions worker: %v", err)
	}

	// Start key rotation scheduler
	startRotationScheduler(db)

	// Wait for a shutdown signal (OS signal, idle timeout, or max runtime)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	select {
	case <-quit:
		log.Println("Received shutdown signal...")
	case <-srvWithHealth.IdleTracker.ShutdownChan():
		log.Printf("Server idle for %v, shutting down...", serverIdleTimeout)
	case <-maxRuntimeChan:
		log.Printf("Server reached max runtime of %v, shutting down...", serverMaxRuntime)
	}

	// Stop max runtime timer if still running
	if maxRuntimeTimer != nil {
		maxRuntimeTimer.Stop()
	}

	// Stop idle tracker
	srvWithHealth.IdleTracker.Stop()

	// Stop rotation scheduler
	stopRotationScheduler()

	// Stop actions worker
	stopActionsWorker()

	log.Println("Shutting down server...")

	// Set health status to NOT_SERVING before shutdown (per guide)
	srvWithHealth.HealthServer.SetServingStatus("", 2) // 2 = NOT_SERVING

	// Start graceful stop with timeout (per guide)
	stopChan := make(chan struct{})

	go func() {
		srvWithHealth.GRPCServer.GracefulStop()
		close(stopChan)
	}()

	// Wait for a graceful stop or force stop after 30 seconds
	select {
	case <-stopChan:
		log.Println("Server stopped gracefully")
	case <-time.After(30 * time.Second):
		log.Println("Timeout waiting for graceful shutdown, forcing stop")
		srvWithHealth.GRPCServer.Stop()
	}

	// Clean up server info file
	grpc.RemoveServerInfo()
	log.Println("Server info file removed")

	return nil
}

func runServerStop(_ *cobra.Command, _ []string) error {
	info := grpc.IsServerRunning()
	if info == nil {
		_, _ = fmt.Fprintln(os.Stdout, "Server is not running")
		return nil
	}

	_, _ = fmt.Fprintf(os.Stdout, "Stopping server (PID: %d)...\n", info.PID)

	// Send termination signal to the process
	if err := terminateProcess(info.PID); err != nil {
		return fmt.Errorf("failed to stop server: %w", err)
	}

	// Wait for the process to exit
	if err := waitForProcessExit(info.PID, stopTimeout); err != nil {
		return fmt.Errorf("server did not stop within timeout: %w", err)
	}

	_, _ = fmt.Fprintln(os.Stdout, "Server stopped successfully")

	return nil
}

func runServerRestart(cmd *cobra.Command, args []string) error {
	info := grpc.IsServerRunning()

	// If the server is running, stop it first
	if info != nil {
		_, _ = fmt.Fprintf(os.Stdout, "Stopping server (PID: %d)...\n", info.PID)

		if err := terminateProcess(info.PID); err != nil {
			return fmt.Errorf("failed to stop server: %w", err)
		}

		// Wait for the process to exit
		if err := waitForProcessExit(info.PID, restartTimeout); err != nil {
			return fmt.Errorf("server did not stop within timeout: %w", err)
		}

		_, _ = fmt.Fprintln(os.Stdout, "Server stopped")
	}

	_, _ = fmt.Fprintln(os.Stdout, "Starting server...")

	// Start the server
	return runServerStart(cmd, args)
}

func runServerStatus(_ *cobra.Command, _ []string) error {
	info := grpc.IsServerRunning()
	if info == nil {
		_, _ = fmt.Fprintln(os.Stdout, "Server status: stopped")
		return nil
	}

	_, _ = fmt.Fprintln(os.Stdout, "Server status: running")
	_, _ = fmt.Fprintf(os.Stdout, "  Address: %s\n", info.Address)
	_, _ = fmt.Fprintf(os.Stdout, "  PID: %d\n", info.PID)
	_, _ = fmt.Fprintf(os.Stdout, "  Started: %s\n", info.StartedAt.Format(time.RFC3339))
	_, _ = fmt.Fprintf(os.Stdout, "  Uptime: %s\n", time.Since(info.StartedAt).Round(time.Second))

	return nil
}

// terminateProcess sends a termination signal to the process with the given PID
func terminateProcess(pid int) error {
	if runtime.GOOS == "windows" {
		// On Windows, use taskkill command
		cmd := exec.Command("taskkill", "/PID", fmt.Sprintf("%d", pid), "/F")
		return cmd.Run()
	}

	// On Unix-like systems, send SIGTERM
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process: %w", err)
	}

	return process.Signal(syscall.SIGTERM)
}

// waitForProcessExit waits for a process to exit using gops to monitor
func waitForProcessExit(pid int, timeout time.Duration) error {
	if err := procs.ListProcesses(); err != nil {
		return fmt.Errorf("failed to list processes: %w", err)
	}

	deadline := time.Now().Add(timeout)
	checkInterval := 100 * time.Millisecond

	for time.Now().Before(deadline) {
		// Use goprocess to check if the process is still running
		if !procs.IsProcessRunning(pid) {
			return nil
		}

		time.Sleep(checkInterval)
	}

	return fmt.Errorf("process %d still running after %v", pid, timeout)
}

// startActionsWorker initializes and starts the GitHub Actions monitoring worker
func startActionsWorker() error {
	// Open actions database
	dbPath, err := actionsdb.DefaultDBPath()
	if err != nil {
		return fmt.Errorf("failed to get actions db path: %w", err)
	}

	actionsDB, err := actionsdb.Open(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open actions database: %w", err)
	}

	// Token function that gets the active profile token
	tokenFunc := func() string {
		pm, err := core.NewProfileManager()
		if err != nil {
			slog.Debug("actions worker: failed to create profile manager", "error", err)
			return ""
		}

		token, err := pm.GetActiveProfileToken()
		if err != nil {
			slog.Debug("actions worker: failed to get active profile token", "error", err)
			return ""
		}

		return token
	}

	// Create worker with default config
	config := actionsdb.DefaultWorkerConfig()
	actionsWorker = actionsdb.NewWorker(actionsDB, tokenFunc, config)
	actionsWorker.WithLogger(slog.Default())

	// Start the worker
	ctx := context.Background()
	if err := actionsWorker.Start(ctx); err != nil {
		return fmt.Errorf("failed to start actions worker: %w", err)
	}

	log.Println("GitHub Actions monitoring worker started")
	return nil
}

// stopActionsWorker stops the GitHub Actions monitoring worker
func stopActionsWorker() {
	if actionsWorker != nil && actionsWorker.IsRunning() {
		actionsWorker.Stop()
		log.Println("GitHub Actions monitoring worker stopped")
	}
}

// startRotationScheduler initializes and starts the key rotation scheduler
func startRotationScheduler(db store.Store) {
	cfg, err := db.GetConfig()
	if err != nil {
		log.Printf("Warning: failed to get config for rotation scheduler: %v", err)
		return
	}

	// Convert days to duration
	if cfg.KeyRotationDays <= 0 {
		log.Println("Key rotation scheduler disabled (key_rotation_days = 0)")
		return
	}

	maxAge := time.Duration(cfg.KeyRotationDays) * 24 * time.Hour
	checkInterval := 1 * time.Hour // Check every hour

	rotationScheduler = grpc.NewRotationScheduler(db, checkInterval, maxAge)
	rotationScheduler.Start()
}

// stopRotationScheduler stops the key rotation scheduler
func stopRotationScheduler() {
	if rotationScheduler != nil {
		rotationScheduler.Stop()
	}
}
