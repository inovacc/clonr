package grpcserver

import (
	"time"

	"github.com/inovacc/clonr/internal/database"
	v1 "github.com/inovacc/clonr/pkg/api/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

// NewServer creates a new gRPC server with all interceptors and registered services
func NewServer(db database.Store) *grpc.Server {
	// Server options
	opts := []grpc.ServerOption{
		// Chain interceptors in order: recovery -> logging -> timeout
		grpc.ChainUnaryInterceptor(
			recoveryInterceptor(),
			loggingInterceptor(),
			timeoutInterceptor(30*time.Second),
		),
		// Keepalive settings
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle: 15 * time.Minute,
			Time:              5 * time.Minute,
			Timeout:           20 * time.Second,
		}),
		// Message size limits (4MB)
		grpc.MaxRecvMsgSize(4 * 1024 * 1024),
		grpc.MaxSendMsgSize(4 * 1024 * 1024),
	}

	// Create gRPC server
	srv := grpc.NewServer(opts...)

	// Register service implementation
	svc := NewService(db)
	v1.RegisterClonrServiceServer(srv, svc)

	return srv
}
