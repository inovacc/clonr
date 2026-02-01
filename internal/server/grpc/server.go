package grpc

import (
	"time"

	"github.com/inovacc/clonr/internal/store"
	v1 "github.com/inovacc/clonr/pkg/api/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
)

// ServerWithHealth wraps gRPC server and health service for lifecycle management
type ServerWithHealth struct {
	GRPCServer   *grpc.Server
	HealthServer *health.Server
	IdleTracker  *IdleTracker
}

// NewServer creates a new gRPC server with all interceptors, health service, and registered services.
// If idleTimeout is > 0, the server will track activity and signal shutdown after being idle.
func NewServer(db store.Store, idleTimeout time.Duration) *ServerWithHealth {
	// Create idle tracker
	idleTracker := NewIdleTracker(idleTimeout)

	// Build interceptor chain
	interceptors := []grpc.UnaryServerInterceptor{
		recoveryInterceptor(),
		loggingInterceptor(),
		timeoutInterceptor(30 * time.Second),
	}

	// Add activity interceptor if idle timeout is enabled
	if idleTracker.IsEnabled() {
		interceptors = append([]grpc.UnaryServerInterceptor{activityInterceptor(idleTracker)}, interceptors...)
	}

	// Server options
	opts := []grpc.ServerOption{
		// Chain interceptors in order: activity -> recovery -> logging -> timeout
		grpc.ChainUnaryInterceptor(interceptors...),
		// Connection timeout (per guide)
		grpc.ConnectionTimeout(10 * time.Second),
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

	// Register health service (per guide)
	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(srv, healthServer)
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

	// Register service implementation
	svc := NewService(db)
	v1.RegisterClonrServiceServer(srv, svc)

	return &ServerWithHealth{
		GRPCServer:   srv,
		HealthServer: healthServer,
		IdleTracker:  idleTracker,
	}
}
