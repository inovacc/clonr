package cmd

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/inovacc/clonr/internal/database"
	"github.com/inovacc/clonr/internal/grpcserver"
	"github.com/spf13/cobra"
)

var serverPort int

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

func init() {
	rootCmd.AddCommand(serverCmd)
	serverCmd.AddCommand(serverStartCmd)
	serverStartCmd.Flags().IntVarP(&serverPort, "port", "p", 50051, "Port to listen on")
}

func runServerStart(cmd *cobra.Command, args []string) error {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	db := database.GetDB()

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

	srvWithHealth := grpcserver.NewServer(db)
	go func() {
		log.Printf("Starting Clonr gRPC server on %s", addr)
		if err := srvWithHealth.GRPCServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

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
