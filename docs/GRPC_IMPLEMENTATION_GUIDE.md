# gRPC Client-Server Implementation Guide

**Project:** WimGuard Backup System
**Language:** Go 1.21+
**gRPC Version:** v1.78.0
**Pattern:** Service-Oriented Architecture with Health Checks

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Proto Definitions](#proto-definitions)
3. [Server Implementation](#server-implementation)
4. [Client Implementation](#client-implementation)
5. [Health Service Integration](#health-service-integration)
6. [Configuration Management](#configuration-management)
7. [Integration Patterns](#integration-patterns)
8. [Best Practices](#best-practices)
9. [Common Pitfalls](#common-pitfalls)
10. [Complete Examples](#complete-examples)

---

## Architecture Overview

### Design Philosophy

**Separation of Concerns:**
- **Server:** Single daemon process with exclusive database access
- **Client:** Lightweight CLI tools that communicate via gRPC
- **Config:** Shared YAML file for service discovery

**Key Benefits:**
- Centralized state management (database access)
- Multiple clients can connect simultaneously
- Service can run as background daemon
- Health checking for reliability
- Streaming support for long-running operations

### Component Diagram

```
┌─────────────────────────────────────────┐
│           Client (CLI)                  │
│  - Reads config for server address      │
│  - Creates gRPC connection              │
│  - Performs health check                │
│  - Calls RPC methods                    │
└──────────────┬──────────────────────────┘
               │ gRPC (port 52730)
               │ Protocol Buffers
               ▼
┌─────────────────────────────────────────┐
│        Server (Daemon)                  │
│  - Listens on configurable port         │
│  - Health service registration          │
│  - Business logic implementation        │
│  - Exclusive database access            │
│  - Configuration management             │
└─────────────────────────────────────────┘
               │
               ▼
         ┌──────────┐
         │  SQLite  │
         │ Database │
         └──────────┘
```

---

## Proto Definitions

### Directory Structure

```
proto/v1/
├── backup.proto          # Backup service
├── restore.proto         # Restore service
├── snapshot.proto        # Snapshot service
├── schedule.proto        # Schedule service
├── registry.proto        # Registry service
├── env.proto            # Environment service
├── statistics.proto     # Statistics service
└── common.proto         # Shared types
```

### Example: Backup Service Proto

**File:** `proto/v1/backup.proto`

```protobuf
syntax = "proto3";

package wimguard.v1;

option go_package = "github.com/dyammarcano/wimguard/internal/api/v1;apiv1";

// BackupService handles backup operations
service BackupService {
  // CreateBackup performs a backup with streaming progress
  rpc CreateBackup(CreateBackupRequest) returns (stream BackupProgress);

  // ListBackups returns all backups
  rpc ListBackups(ListBackupsRequest) returns (ListBackupsResponse);

  // GetBackup returns a single backup by UUID
  rpc GetBackup(GetBackupRequest) returns (GetBackupResponse);

  // DeleteBackup removes a backup
  rpc DeleteBackup(DeleteBackupRequest) returns (DeleteBackupResponse);
}

// Request/Response messages
message CreateBackupRequest {
  string source = 1;
  string destination = 2;
  string compression = 3;
  string description = 4;
}

message BackupProgress {
  string uuid = 1;
  string status = 2;  // "started", "in_progress", "completed", "failed"
  int64 bytes_processed = 3;
  int64 total_bytes = 4;
  string message = 5;
  string error = 6;
}

message ListBackupsRequest {
  int32 limit = 1;
  int32 offset = 2;
}

message ListBackupsResponse {
  repeated Backup backups = 1;
  int32 total = 2;
}

message Backup {
  string uuid = 1;
  string name = 2;
  string source = 3;
  string destination = 4;
  string type = 5;
  string status = 6;
  int64 size = 7;
  string created_at = 8;
}

message GetBackupRequest {
  string uuid = 1;
}

message GetBackupResponse {
  Backup backup = 1;
}

message DeleteBackupRequest {
  string uuid = 1;
}

message DeleteBackupResponse {
  bool success = 1;
}
```

### Code Generation

**Makefile:**
```makefile
.PHONY: proto
proto:
	protoc --go_out=. --go_opt=paths=source_relative \
	       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
	       proto/v1/*.proto
```

**Command:**
```bash
make proto
```

**Generated Files:**
- `internal/api/v1/backup.pb.go` - Message types
- `internal/api/v1/backup_grpc.pb.go` - Service interfaces

---

## Server Implementation

### Main Server Structure

**File:** `cmd/server.go`

```go
package cmd

import (
    "context"
    "fmt"
    "log"
    "net"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/spf13/cobra"
    "google.golang.org/grpc"
    "google.golang.org/grpc/health"
    healthpb "google.golang.org/grpc/health/grpc_health_v1"

    "github.com/dyammarcano/wimguard/internal/config"
    "github.com/dyammarcano/wimguard/internal/database"
    "github.com/dyammarcano/wimguard/internal/grpcserver"
)

var serverCmd = &cobra.Command{
    Use:   "server",
    Short: "Run gRPC server daemon",
    RunE:  runServer,
}

func init() {
    rootCmd.AddCommand(serverCmd)
    serverCmd.Flags().IntVarP(&serverPort, "port", "p", 52730, "gRPC server port")
}

func runServer(ctx context.Context) error {
    // 1. Setup signal handling for graceful shutdown
    ctx, cancel := context.WithCancel(ctx)
    defer cancel()

    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

    go func() {
        <-sigChan
        fmt.Println("\nShutting down server...")
        cancel()
    }()

    // 2. Load/create configuration
    appDir, err := config.GetApplicationDirectory()
    if err != nil {
        return err
    }

    if err := config.PrepareConfig(appDir, config.DefaultConfig()); err != nil {
        return fmt.Errorf("failed to prepare config: %w", err)
    }

    // 3. Update config with server settings
    cfg := config.GetConfig()
    cfg.Server.Port = serverPort
    cfg.Client.ServerAddress = fmt.Sprintf("localhost:%d", serverPort)

    if err := config.SaveConfig(appDir); err != nil {
        return fmt.Errorf("failed to save config: %w", err)
    }

    // 4. Open database
    dbPath := filepath.Join(appDir, cfg.Server.Database)
    db, err := database.NewDatabase(dbPath)
    if err != nil {
        return fmt.Errorf("failed to open database: %w", err)
    }
    defer db.Close()

    // 5. Create gRPC server
    grpcServer := grpc.NewServer()

    // 6. Register health service
    healthServer := health.NewServer()
    healthpb.RegisterHealthServer(grpcServer, healthServer)
    healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

    // 7. Create and register business services
    wimguardServer := grpcserver.NewServer(db, appDir)
    wimguardServer.RegisterServices(grpcServer)

    // 8. Start scheduler
    if err := wimguardServer.StartScheduler(ctx); err != nil {
        return fmt.Errorf("failed to start scheduler: %w", err)
    }

    // 9. Create listener
    addr := fmt.Sprintf(":%d", serverPort)
    listener, err := net.Listen("tcp", addr)
    if err != nil {
        return fmt.Errorf("failed to listen: %w", err)
    }

    log.Printf("gRPC server listening on %s", addr)

    // 10. Serve in goroutine
    errChan := make(chan error, 1)
    go func() {
        if err := grpcServer.Serve(listener); err != nil {
            errChan <- err
        }
    }()

    // 11. Wait for shutdown signal or error
    select {
    case <-ctx.Done():
        log.Println("Shutting down gracefully...")
        healthServer.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
        grpcServer.GracefulStop()
        wimguardServer.Shutdown()
        return nil
    case err := <-errChan:
        return err
    }
}
```

### Service Implementation

**File:** `internal/grpcserver/server.go`

```go
package grpcserver

import (
    "context"
    "sync"

    "google.golang.org/grpc"

    "github.com/dyammarcano/wimguard/internal/database"
    pb "github.com/dyammarcano/wimguard/internal/api/v1"
)

type Server struct {
    db       *database.Database
    appDir   string
    mu       sync.RWMutex
    // Add any shared state here
}

func NewServer(db *database.Database, appDir string) *Server {
    return &Server{
        db:     db,
        appDir: appDir,
    }
}

// RegisterServices registers all gRPC services
func (s *Server) RegisterServices(grpcServer *grpc.Server) {
    pb.RegisterBackupServiceServer(grpcServer, s)
    pb.RegisterRestoreServiceServer(grpcServer, s)
    pb.RegisterSnapshotServiceServer(grpcServer, s)
    pb.RegisterScheduleServiceServer(grpcServer, s)
    pb.RegisterRegistryServiceServer(grpcServer, s)
    pb.RegisterEnvServiceServer(grpcServer, s)
    pb.RegisterStatisticsServiceServer(grpcServer, s)
}

func (s *Server) Shutdown() {
    // Cleanup logic
}
```

### Backup Service Implementation

**File:** `internal/grpcserver/backup_service.go`

```go
package grpcserver

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/google/uuid"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"

    "github.com/dyammarcano/wimguard/internal/backup"
    pb "github.com/dyammarcano/wimguard/internal/api/v1"
)

// CreateBackup implements streaming backup with progress updates
func (s *Server) CreateBackup(
    req *pb.CreateBackupRequest,
    stream pb.BackupService_CreateBackupServer,
) error {
    // 1. Validate request
    if req.Source == "" || req.Destination == "" {
        return status.Error(codes.InvalidArgument, "source and destination required")
    }

    // 2. Generate UUID for this backup
    backupUUID := uuid.New().String()

    // 3. Send initial progress
    if err := stream.Send(&pb.BackupProgress{
        Uuid:   backupUUID,
        Status: "started",
        Message: fmt.Sprintf("Starting backup of %s", req.Source),
    }); err != nil {
        return err
    }

    // 4. Create backup configuration
    cfg := backup.Config{
        Source:      req.Source,
        Destination: req.Destination,
        Compression: req.Compression,
        Description: req.Description,
    }

    // 5. Create backup manager with progress callback
    mgr := backup.NewManager(cfg, s.db)
    mgr.SetProgressCallback(func(bytesProcessed, totalBytes int64, msg string) {
        // Send progress updates to client
        stream.Send(&pb.BackupProgress{
            Uuid:           backupUUID,
            Status:         "in_progress",
            BytesProcessed: bytesProcessed,
            TotalBytes:     totalBytes,
            Message:        msg,
        })
    })

    // 6. Execute backup
    ctx := stream.Context()
    result, err := mgr.CreateBackup(ctx)
    if err != nil {
        // Send failure
        stream.Send(&pb.BackupProgress{
            Uuid:   backupUUID,
            Status: "failed",
            Error:  err.Error(),
        })
        return status.Errorf(codes.Internal, "backup failed: %v", err)
    }

    // 7. Send completion
    return stream.Send(&pb.BackupProgress{
        Uuid:           backupUUID,
        Status:         "completed",
        BytesProcessed: result.Size,
        TotalBytes:     result.Size,
        Message:        "Backup completed successfully",
    })
}

// ListBackups implements pagination
func (s *Server) ListBackups(
    ctx context.Context,
    req *pb.ListBackupsRequest,
) (*pb.ListBackupsResponse, error) {
    // Default pagination
    limit := req.Limit
    if limit == 0 {
        limit = 20
    }

    offset := req.Offset

    // Query database
    backups, err := s.db.Queries.ListBackups(ctx, db.ListBackupsParams{
        Limit:  int64(limit),
        Offset: int64(offset),
    })
    if err != nil {
        return nil, status.Errorf(codes.Internal, "failed to list backups: %v", err)
    }

    // Convert to protobuf messages
    pbBackups := make([]*pb.Backup, len(backups))
    for i, b := range backups {
        pbBackups[i] = &pb.Backup{
            Uuid:        b.Uuid,
            Name:        b.Name,
            Source:      b.Source,
            Destination: b.Destination,
            Type:        b.Type,
            Status:      b.Status,
            Size:        b.Size,
            CreatedAt:   b.CreatedAt.Format(time.RFC3339),
        }
    }

    return &pb.ListBackupsResponse{
        Backups: pbBackups,
        Total:   int32(len(backups)),
    }, nil
}

// GetBackup retrieves a single backup by UUID
func (s *Server) GetBackup(
    ctx context.Context,
    req *pb.GetBackupRequest,
) (*pb.GetBackupResponse, error) {
    if req.Uuid == "" {
        return nil, status.Error(codes.InvalidArgument, "uuid required")
    }

    backup, err := s.db.Queries.GetBackup(ctx, req.Uuid)
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, status.Error(codes.NotFound, "backup not found")
        }
        return nil, status.Errorf(codes.Internal, "failed to get backup: %v", err)
    }

    return &pb.GetBackupResponse{
        Backup: &pb.Backup{
            Uuid:        backup.Uuid,
            Name:        backup.Name,
            Source:      backup.Source,
            Destination: backup.Destination,
            Type:        backup.Type,
            Status:      backup.Status,
            Size:        backup.Size,
            CreatedAt:   backup.CreatedAt.Format(time.RFC3339),
        },
    }, nil
}
```

---

## Client Implementation

### Client Library Structure

**File:** `internal/grpcclient/client.go`

```go
package grpcclient

import (
    "context"
    "fmt"
    "time"

    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
    healthpb "google.golang.org/grpc/health/grpc_health_v1"

    pb "github.com/dyammarcano/wimguard/internal/api/v1"
)

// Client wraps the gRPC connection and service clients
type Client struct {
    conn       *grpc.ClientConn
    Backup     pb.BackupServiceClient
    Restore    pb.RestoreServiceClient
    Snapshot   pb.SnapshotServiceClient
    Schedule   pb.ScheduleServiceClient
    Registry   pb.RegistryServiceClient
    Env        pb.EnvServiceClient
    Statistics pb.StatisticsServiceClient
}

// NewClient creates a client with default timeout (30s)
func NewClient(addr string) (*Client, error) {
    return NewClientWithTimeout(addr, 30*time.Second)
}

// NewClientWithTimeout creates a client with custom timeout
func NewClientWithTimeout(addr string, timeout time.Duration) (*Client, error) {
    // 1. Create gRPC connection
    // IMPORTANT: gRPC v1.78.0+ uses lazy connection establishment
    // The actual connection happens on first RPC call
    conn, err := grpc.NewClient(
        addr,
        grpc.WithTransportCredentials(insecure.NewCredentials()),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create grpc client: %w", err)
    }

    // 2. Perform health check to verify server is reachable
    // This triggers the lazy connection establishment
    health := healthpb.NewHealthClient(conn)
    healthCtx, healthCancel := context.WithTimeout(context.Background(), timeout)
    defer healthCancel()

    resp, err := health.Check(healthCtx, &healthpb.HealthCheckRequest{})
    if err != nil {
        _ = conn.Close()
        return nil, fmt.Errorf("server not reachable: %w", err)
    }

    // 3. Verify server is healthy
    if resp.GetStatus() != healthpb.HealthCheckResponse_SERVING {
        _ = conn.Close()
        return nil, fmt.Errorf("server not healthy: status=%v", resp.GetStatus())
    }

    // 4. Create service clients
    return &Client{
        conn:       conn,
        Backup:     pb.NewBackupServiceClient(conn),
        Restore:    pb.NewRestoreServiceClient(conn),
        Snapshot:   pb.NewSnapshotServiceClient(conn),
        Schedule:   pb.NewScheduleServiceClient(conn),
        Registry:   pb.NewRegistryServiceClient(conn),
        Env:        pb.NewEnvServiceClient(conn),
        Statistics: pb.NewStatisticsServiceClient(conn),
    }, nil
}

// Close closes the gRPC connection
func (c *Client) Close() error {
    return c.conn.Close()
}
```

### CLI Command Using Client

**File:** `cmd/backup.go`

```go
package cmd

import (
    "context"
    "fmt"
    "io"
    "os"
    "time"

    "github.com/spf13/cobra"

    "github.com/dyammarcano/wimguard/internal/config"
    "github.com/dyammarcano/wimguard/internal/grpcclient"
    pb "github.com/dyammarcano/wimguard/internal/api/v1"
)

var backupCmd = &cobra.Command{
    Use:   "backup",
    Short: "Create a backup",
    RunE:  runBackup,
}

var (
    backupSource      string
    backupDestination string
    backupCompression string
    backupDescription string
)

func init() {
    rootCmd.AddCommand(backupCmd)
    backupCmd.Flags().StringVarP(&backupSource, "source", "s", "", "Source directory")
    backupCmd.Flags().StringVarP(&backupDestination, "destination", "d", "", "Destination file")
    backupCmd.Flags().StringVarP(&backupCompression, "compression", "c", "LZX", "Compression type")
    backupCmd.Flags().StringVarP(&backupDescription, "description", "D", "", "Backup description")
    backupCmd.MarkFlagRequired("source")
    backupCmd.MarkFlagRequired("destination")
}

func runBackup(cmd *cobra.Command, args []string) error {
    // 1. Load configuration to get server address
    appDir, err := config.GetApplicationDirectory()
    if err != nil {
        return err
    }

    if err := config.LoadConfig(appDir); err != nil {
        return fmt.Errorf("failed to load config: %w", err)
    }

    cfg := config.GetConfig()
    serverAddr := cfg.Client.ServerAddress

    // 2. Create gRPC client
    fmt.Printf("Connecting to server at %s...\n", serverAddr)
    client, err := grpc.NewClient(serverAddr)
    if err != nil {
        return fmt.Errorf("failed to connect to server: %w", err)
    }
    defer client.Close()

    // 3. Create backup request
    req := &pb.CreateBackupRequest{
        Source:      backupSource,
        Destination: backupDestination,
        Compression: backupCompression,
        Description: backupDescription,
    }

    // 4. Call streaming RPC
    ctx, cancel := context.WithTimeout(context.Background(), 1*time.Hour)
    defer cancel()

    stream, err := client.Backup.CreateBackup(ctx, req)
    if err != nil {
        return fmt.Errorf("failed to start backup: %w", err)
    }

    fmt.Printf("\nCreating backup of %s via gRPC server...\n", backupSource)

    // 5. Receive streaming progress updates
    var backupUUID string
    for {
        progress, err := stream.Recv()
        if err == io.EOF {
            // Stream completed
            break
        }
        if err != nil {
            return fmt.Errorf("backup failed: %w", err)
        }

        // Store UUID from first message
        if backupUUID == "" {
            backupUUID = progress.Uuid
        }

        // Display progress based on status
        switch progress.Status {
        case "started":
            fmt.Printf("Backup started (UUID: %s)\n", progress.Uuid)
        case "in_progress":
            if progress.TotalBytes > 0 {
                percent := float64(progress.BytesProcessed) / float64(progress.TotalBytes) * 100
                fmt.Printf("\rProgress: %.1f%% (%s)", percent, progress.Message)
            } else {
                fmt.Printf("\r%s", progress.Message)
            }
        case "completed":
            fmt.Printf("\n\n✓ Backup completed successfully\n")
            fmt.Printf("  UUID: %s\n", progress.Uuid)
            fmt.Printf("  Size: %s\n", formatBytes(progress.BytesProcessed))
        case "failed":
            return fmt.Errorf("backup failed: %s", progress.Error)
        }
    }

    return nil
}

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
    return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
```

---

## Health Service Integration

### Why Health Checks?

1. **Service Discovery:** Verify server is reachable before operations
2. **Load Balancing:** Route traffic only to healthy instances
3. **Monitoring:** External tools can check service health
4. **Graceful Shutdown:** Signal clients before shutting down

### Server-Side Health Registration

```go
import (
    "google.golang.org/grpc/health"
    healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// In server startup
healthServer := health.NewServer()
healthpb.RegisterHealthServer(grpcServer, healthServer)

// Mark as serving
healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

// During graceful shutdown
healthServer.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
```

### Client-Side Health Check

```go
import healthpb "google.golang.org/grpc/health/grpc_health_v1"

func verifyServerHealth(conn *grpc.ClientConn, timeout time.Duration) error {
    health := healthpb.NewHealthClient(conn)

    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()

    resp, err := health.Check(ctx, &healthpb.HealthCheckRequest{})
    if err != nil {
        return fmt.Errorf("health check failed: %w", err)
    }

    if resp.GetStatus() != healthpb.HealthCheckResponse_SERVING {
        return fmt.Errorf("server not healthy: %v", resp.GetStatus())
    }

    return nil
}
```

---

## Configuration Management

### Shared Config Pattern

**Purpose:** Server writes config, client reads for service discovery

**File:** `internal/config/config.go`

```go
package config

import (
    "fmt"
    "os"
    "path/filepath"

    "gopkg.in/yaml.v3"
)

type Config struct {
    Environment string         `yaml:"environment"`
    AppID       string         `yaml:"appID"`
    AppSecret   string         `yaml:"appSecret"`
    Service     ServiceConfig  `yaml:"service"`
}

type ServiceConfig struct {
    Server ServerConfig `yaml:"server"`
    Client ClientConfig `yaml:"client"`
}

type ServerConfig struct {
    Port     int    `yaml:"port"`
    Enabled  bool   `yaml:"enabled"`
    Database string `yaml:"database"`
}

type ClientConfig struct {
    ServerAddress string `yaml:"serverAddress"`
    Timeout       string `yaml:"timeout"`
}

var moduleCfg *Config

// GetConfig returns the current configuration
func GetConfig() *Config {
    return moduleCfg
}

// LoadConfig loads configuration from file
func LoadConfig(appDir string) error {
    configPath := filepath.Join(appDir, "config.yaml")

    data, err := os.ReadFile(configPath)
    if err != nil {
        return fmt.Errorf("failed to read config: %w", err)
    }

    var cfg Config
    if err := yaml.Unmarshal(data, &cfg); err != nil {
        return fmt.Errorf("failed to parse config: %w", err)
    }

    moduleCfg = &cfg
    return nil
}

// SaveConfig saves configuration to file
func SaveConfig(appDir string) error {
    configPath := filepath.Join(appDir, "config.yaml")

    data, err := yaml.Marshal(moduleCfg)
    if err != nil {
        return fmt.Errorf("failed to marshal config: %w", err)
    }

    if err := os.WriteFile(configPath, data, 0644); err != nil {
        return fmt.Errorf("failed to write config: %w", err)
    }

    return nil
}

// PrepareConfig initializes config with defaults
func PrepareConfig(appDir string, defaults *Config) error {
    configPath := filepath.Join(appDir, "config.yaml")

    // Create directory if not exists
    if err := os.MkdirAll(appDir, 0755); err != nil {
        return err
    }

    // Use existing config or create from defaults
    if _, err := os.Stat(configPath); os.IsNotExist(err) {
        moduleCfg = defaults
        return SaveConfig(appDir)
    }

    return LoadConfig(appDir)
}

// GetApplicationDirectory returns the app data directory
func GetApplicationDirectory() (string, error) {
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return "", err
    }
    return filepath.Join(homeDir, "AppData", "Local", "wimbackup"), nil
}
```

### Example Config File

**File:** `%LOCALAPPDATA%\wimbackup\config.yaml`

```yaml
environment: production
appID: 550e8400-e29b-41d4-a716-446655440000
appSecret: secret-key-here
service:
  server:
    port: 52730
    enabled: true
    database: wimbackup.db
  client:
    serverAddress: localhost:52730
    timeout: 30s
```

---

## Integration Patterns

### Pattern 1: Request-Response (Unary RPC)

**Use Case:** Simple queries, CRUD operations

**Proto:**
```protobuf
rpc GetBackup(GetBackupRequest) returns (GetBackupResponse);
```

**Server:**
```go
func (s *Server) GetBackup(ctx context.Context, req *pb.GetBackupRequest) (*pb.GetBackupResponse, error) {
    backup, err := s.db.GetBackup(ctx, req.Uuid)
    if err != nil {
        return nil, status.Error(codes.NotFound, "not found")
    }
    return &pb.GetBackupResponse{Backup: backup}, nil
}
```

**Client:**
```go
resp, err := client.Backup.GetBackup(ctx, &pb.GetBackupRequest{Uuid: uuid})
```

### Pattern 2: Server Streaming

**Use Case:** Long-running operations with progress updates

**Proto:**
```protobuf
rpc CreateBackup(CreateBackupRequest) returns (stream BackupProgress);
```

**Server:**
```go
func (s *Server) CreateBackup(req *pb.CreateBackupRequest, stream pb.BackupService_CreateBackupServer) error {
    // Send multiple progress updates
    for progress := range progressChannel {
        if err := stream.Send(&pb.BackupProgress{...}); err != nil {
            return err
        }
    }
    return nil
}
```

**Client:**
```go
stream, err := client.Backup.CreateBackup(ctx, req)
for {
    progress, err := stream.Recv()
    if err == io.EOF {
        break
    }
    if err != nil {
        return err
    }
    // Handle progress
}
```

### Pattern 3: Client Streaming

**Use Case:** Uploading data in chunks

**Proto:**
```protobuf
rpc UploadFile(stream FileChunk) returns (UploadResponse);
```

**Server:**
```go
func (s *Server) UploadFile(stream pb.FileService_UploadFileServer) error {
    for {
        chunk, err := stream.Recv()
        if err == io.EOF {
            return stream.SendAndClose(&pb.UploadResponse{Success: true})
        }
        if err != nil {
            return err
        }
        // Process chunk
    }
}
```

**Client:**
```go
stream, err := client.File.UploadFile(ctx)
for _, chunk := range chunks {
    stream.Send(&pb.FileChunk{Data: chunk})
}
resp, err := stream.CloseAndRecv()
```

### Pattern 4: Bidirectional Streaming

**Use Case:** Real-time communication, chat

**Proto:**
```protobuf
rpc Chat(stream Message) returns (stream Message);
```

**Implementation:** Both sides send/receive in any order

---

## Best Practices

### 1. Error Handling

**Use gRPC Status Codes:**
```go
import (
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
)

// Server
func (s *Server) GetBackup(ctx context.Context, req *pb.GetBackupRequest) (*pb.GetBackupResponse, error) {
    if req.Uuid == "" {
        return nil, status.Error(codes.InvalidArgument, "uuid required")
    }

    backup, err := s.db.GetBackup(ctx, req.Uuid)
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, status.Error(codes.NotFound, "backup not found")
        }
        return nil, status.Errorf(codes.Internal, "database error: %v", err)
    }

    return &pb.GetBackupResponse{Backup: backup}, nil
}

// Client
resp, err := client.Backup.GetBackup(ctx, req)
if err != nil {
    st, ok := status.FromError(err)
    if ok {
        switch st.Code() {
        case codes.NotFound:
            fmt.Println("Backup not found")
        case codes.InvalidArgument:
            fmt.Println("Invalid request:", st.Message())
        default:
            fmt.Println("Error:", st.Message())
        }
    }
}
```

### 2. Context Management

**Always respect context cancellation:**
```go
func (s *Server) CreateBackup(req *pb.CreateBackupRequest, stream pb.BackupService_CreateBackupServer) error {
    ctx := stream.Context()

    for {
        select {
        case <-ctx.Done():
            return status.Error(codes.Canceled, "client disconnected")
        case progress := <-progressChan:
            if err := stream.Send(progress); err != nil {
                return err
            }
        }
    }
}
```

### 3. Timeouts

**Client-side:**
```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

resp, err := client.Backup.GetBackup(ctx, req)
```

**Server-side:**
```go
grpcServer := grpc.NewServer(
    grpc.ConnectionTimeout(10 * time.Second),
    grpc.MaxRecvMsgSize(10 * 1024 * 1024), // 10MB
)
```

### 4. Graceful Shutdown

**Server:**
```go
// Set health to NOT_SERVING before shutdown
healthServer.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)

// Wait for in-flight RPCs
grpcServer.GracefulStop()

// Force shutdown after timeout
go func() {
    <-time.After(30 * time.Second)
    grpcServer.Stop()
}()
```

### 5. Connection Pooling

**gRPC connections are heavy - reuse them:**
```go
// BAD: Creating new connection per request
func doRequest() {
    conn, _ := grpc.Dial(addr)
    defer conn.Close()
    client := pb.NewBackupServiceClient(conn)
    // ...
}

// GOOD: Reuse connection
var globalClient *grpc.Client

func init() {
    globalClient, _ = grpc.NewClient(addr)
}

func doRequest() {
    resp, err := globalClient.Backup.GetBackup(ctx, req)
    // ...
}
```

### 6. Interceptors for Cross-Cutting Concerns

**Logging Interceptor:**
```go
func loggingInterceptor(
    ctx context.Context,
    req interface{},
    info *grpc.UnaryServerInfo,
    handler grpc.UnaryHandler,
) (interface{}, error) {
    start := time.Now()

    resp, err := handler(ctx, req)

    log.Printf("[%s] %s - %v",
        info.FullMethod,
        time.Since(start),
        err,
    )

    return resp, err
}

// Register
grpcServer := grpc.NewServer(
    grpc.UnaryInterceptor(loggingInterceptor),
)
```

---

## Common Pitfalls

### 1. gRPC v1.78.0+ Lazy Connection

**Problem:**
```go
// This doesn't work in gRPC v1.78.0+
conn, _ := grpc.Dial(addr)
conn.WaitForStateChange(ctx, connectivity.Ready) // ❌ Never completes
```

**Solution:**
```go
// Use health check to trigger connection
conn, _ := grpc.NewClient(addr)
health := healthpb.NewHealthClient(conn)
health.Check(ctx, &healthpb.HealthCheckRequest{}) // ✅ Triggers connection
```

### 2. Forgetting to Close Streams

**Problem:**
```go
stream, _ := client.CreateBackup(ctx, req)
for {
    msg, err := stream.Recv()
    if err != nil {
        return err // ❌ Stream leak
    }
}
```

**Solution:**
```go
stream, _ := client.CreateBackup(ctx, req)
defer stream.CloseSend() // ✅ Always close

for {
    msg, err := stream.Recv()
    if err == io.EOF {
        break
    }
    if err != nil {
        return err
    }
}
```

### 3. Not Handling io.EOF in Streaming

**Problem:**
```go
for {
    msg, err := stream.Recv()
    if err != nil {
        return err // ❌ io.EOF is normal completion
    }
}
```

**Solution:**
```go
for {
    msg, err := stream.Recv()
    if err == io.EOF {
        break // ✅ Normal stream end
    }
    if err != nil {
        return err
    }
}
```

### 4. Large Message Sizes

**Problem:**
```go
// Default limit is 4MB
resp, err := client.GetLargeData(ctx, req) // ❌ May fail
```

**Solution:**
```go
// Server
grpcServer := grpc.NewServer(
    grpc.MaxRecvMsgSize(50 * 1024 * 1024), // 50MB
    grpc.MaxSendMsgSize(50 * 1024 * 1024),
)

// Client
conn, _ := grpc.Dial(addr,
    grpc.WithDefaultCallOptions(
        grpc.MaxCallRecvMsgSize(50 * 1024 * 1024),
        grpc.MaxCallSendMsgSize(50 * 1024 * 1024),
    ),
)
```

### 5. Blocking in Stream Send/Recv

**Problem:**
```go
// Blocking forever if context is cancelled
stream.Send(msg) // ❌ No context check
```

**Solution:**
```go
select {
case <-ctx.Done():
    return ctx.Err()
default:
    if err := stream.Send(msg); err != nil {
        return err
    }
}
```

---

## Complete Examples

### Example 1: Simple List Operation

**Proto:**
```protobuf
service BackupService {
  rpc ListBackups(ListBackupsRequest) returns (ListBackupsResponse);
}

message ListBackupsRequest {
  int32 limit = 1;
  int32 offset = 2;
}

message ListBackupsResponse {
  repeated Backup backups = 1;
}
```

**Server:**
```go
func (s *Server) ListBackups(ctx context.Context, req *pb.ListBackupsRequest) (*pb.ListBackupsResponse, error) {
    backups, err := s.db.ListBackups(ctx, req.Limit, req.Offset)
    if err != nil {
        return nil, status.Errorf(codes.Internal, "query failed: %v", err)
    }

    return &pb.ListBackupsResponse{Backups: backups}, nil
}
```

**Client:**
```go
resp, err := client.Backup.ListBackups(ctx, &pb.ListBackupsRequest{
    Limit:  20,
    Offset: 0,
})
if err != nil {
    log.Fatal(err)
}

for _, backup := range resp.Backups {
    fmt.Printf("Backup: %s\n", backup.Name)
}
```

### Example 2: Streaming with Progress

**Complete flow from server to client:**

**Server:**
```go
func (s *Server) CreateBackup(req *pb.CreateBackupRequest, stream pb.BackupService_CreateBackupServer) error {
    ctx := stream.Context()

    // Send start
    stream.Send(&pb.BackupProgress{Status: "started"})

    // Simulate work with progress
    total := int64(1000)
    for i := int64(0); i <= total; i += 100 {
        select {
        case <-ctx.Done():
            return status.Error(codes.Canceled, "cancelled")
        default:
            stream.Send(&pb.BackupProgress{
                Status:         "in_progress",
                BytesProcessed: i,
                TotalBytes:     total,
            })
            time.Sleep(100 * time.Millisecond)
        }
    }

    // Send completion
    return stream.Send(&pb.BackupProgress{Status: "completed"})
}
```

**Client:**
```go
stream, err := client.Backup.CreateBackup(ctx, &pb.CreateBackupRequest{
    Source:      "/data",
    Destination: "/backup.wim",
})
if err != nil {
    log.Fatal(err)
}

for {
    progress, err := stream.Recv()
    if err == io.EOF {
        break
    }
    if err != nil {
        log.Fatal(err)
    }

    switch progress.Status {
    case "started":
        fmt.Println("Backup started")
    case "in_progress":
        pct := float64(progress.BytesProcessed) / float64(progress.TotalBytes) * 100
        fmt.Printf("\rProgress: %.1f%%", pct)
    case "completed":
        fmt.Println("\nBackup completed")
    }
}
```

---

## Summary

### Key Takeaways

1. **Architecture:**
   - Single server daemon with exclusive database access
   - Lightweight clients communicate via gRPC
   - Health checks for reliability

2. **Implementation:**
   - Define services in proto files
   - Generate code with protoc
   - Implement server interfaces
   - Create client wrapper library

3. **Best Practices:**
   - Use health checks
   - Handle errors with status codes
   - Respect context cancellation
   - Implement graceful shutdown
   - Reuse connections

4. **Common Patterns:**
   - Unary RPC for simple operations
   - Server streaming for progress updates
   - Client streaming for uploads
   - Bidirectional for real-time

5. **Avoid Pitfalls:**
   - Test with gRPC v1.78.0+ lazy connections
   - Always handle io.EOF in streams
   - Close streams properly
   - Configure message size limits
   - Check context in long operations

### Benefits Achieved

- ✅ Clean separation of concerns
- ✅ Type-safe communication
- ✅ Streaming support
- ✅ Health monitoring
- ✅ Service discovery via config
- ✅ Multiple concurrent clients
- ✅ Graceful degradation

This implementation pattern can be adapted to any Go project requiring client-server architecture with gRPC.
