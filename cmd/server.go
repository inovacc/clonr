package cmd

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/google/gops/goprocess"
	"github.com/inovacc/clonr/internal/grpcserver"
	"github.com/inovacc/clonr/internal/store"
	"github.com/spf13/cobra"
)

var (
	serverPort        int
	serverIdleTimeout time.Duration
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Server management commands",
	Long:  `Manage the Clonr gRPC server. Use 'clonr server start' to start the server.`,
}

var serverStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the gRPC server",
	Long:  `Start the Clonr gRPC server on the configured port. The server will continue running until interrupted with Ctrl+C.`,
	RunE:  runServerStart,
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

	serverStopCmd.Flags().DurationVar(&stopTimeout, "timeout", 30*time.Second, "Timeout waiting for server to stop")

	serverRestartCmd.Flags().IntVarP(&serverPort, "port", "p", 50051, "Port to listen on")
	serverRestartCmd.Flags().DurationVar(&serverIdleTimeout, "idle-timeout", 5*time.Minute, "Shutdown after being idle for this duration (0 to disable)")
	serverRestartCmd.Flags().DurationVar(&restartTimeout, "timeout", 30*time.Second, "Timeout waiting for server to stop before restart")
}

func runServerStart(cmd *cobra.Command, args []string) error {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Check if server is already running - silent abort if so
	if grpcserver.IsServerRunning() != nil {
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

	// Write server info file for client discovery
	if err := grpcserver.WriteServerInfo(serverPort); err != nil {
		log.Printf("Warning: failed to write server info file: %v", err)
	} else {
		log.Printf("Server info written to local data directory")
	}

	srvWithHealth := grpcserver.NewServer(db, serverIdleTimeout)

	// Start idle tracker if enabled
	if srvWithHealth.IdleTracker.IsEnabled() {
		go srvWithHealth.IdleTracker.Start()

		log.Printf("Idle timeout enabled: server will shutdown after %v of inactivity", serverIdleTimeout)
	}

	go func() {
		log.Printf("Starting Clonr gRPC server on %s", addr)

		if err := srvWithHealth.GRPCServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	// Wait for shutdown signal (OS signal or idle timeout)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	select {
	case <-quit:
		log.Println("Received shutdown signal...")
	case <-srvWithHealth.IdleTracker.ShutdownChan():
		log.Printf("Server idle for %v, shutting down...", serverIdleTimeout)
	}

	// Stop idle tracker
	srvWithHealth.IdleTracker.Stop()

	log.Println("Shutting down server...")

	// Set health status to NOT_SERVING before shutdown (per guide)
	srvWithHealth.HealthServer.SetServingStatus("", 2) // 2 = NOT_SERVING

	// Start graceful stop with timeout (per guide)
	stopChan := make(chan struct{})

	go func() {
		srvWithHealth.GRPCServer.GracefulStop()
		close(stopChan)
	}()

	// Wait for graceful stop or force stop after 30 seconds
	select {
	case <-stopChan:
		log.Println("Server stopped gracefully")
	case <-time.After(30 * time.Second):
		log.Println("Timeout waiting for graceful shutdown, forcing stop")
		srvWithHealth.GRPCServer.Stop()
	}

	// Clean up server info file
	grpcserver.RemoveServerInfo()
	log.Println("Server info file removed")

	return nil
}

func runServerStop(_ *cobra.Command, _ []string) error {
	info := grpcserver.IsServerRunning()
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
	info := grpcserver.IsServerRunning()

	// If server is running, stop it first
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
	info := grpcserver.IsServerRunning()
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
	deadline := time.Now().Add(timeout)
	checkInterval := 100 * time.Millisecond

	for time.Now().Before(deadline) {
		// Use goprocess to check if the process is still running
		if !isProcessRunning(pid) {
			return nil
		}

		time.Sleep(checkInterval)
	}

	return fmt.Errorf("process %d still running after %v", pid, timeout)
}

// isProcessRunning checks if a process with the given PID is still running
func isProcessRunning(pid int) bool {
	processes := goprocess.FindAll()

	for _, proc := range processes {
		if proc.PID == pid {
			return true
		}
	}

	return false
}
