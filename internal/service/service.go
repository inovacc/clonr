package service

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/inovacc/clonr/internal/database"
	"github.com/inovacc/clonr/internal/grpcserver"
	"github.com/spf13/cobra"
)

func Service(cmd *cobra.Command, args []string, port int) error {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Initialize database
	db := database.GetDB()

	// If port not specified via flag, try to get from config
	if port == 50051 {
		cfg, err := db.GetConfig()
		if err == nil && cfg.ServerPort > 0 && cfg.ServerPort != 4000 {
			port = cfg.ServerPort
		}
	}

	// Create listener
	addr := fmt.Sprintf(":%d", port)

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	// Create gRPC server (no idle timeout for service mode - run forever)
	srv := grpcserver.NewServer(db, 0)

	// Start server in background
	go func() {
		log.Printf("Starting Clonr gRPC server on %s", addr)

		if err := srv.GRPCServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	// Graceful shutdown
	log.Println("Shutting down server...")
	srv.GRPCServer.GracefulStop()
	log.Println("Server stopped")

	return nil
}
