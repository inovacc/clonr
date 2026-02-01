// Package grpcserver provides the gRPC server implementation for Clonr.
//
// The server exposes repository management operations via gRPC, allowing
// remote clients to interact with the repository database. It implements
// the ClonrService defined in the protobuf definitions.
//
// # Server Lifecycle
//
// The server is started via [NewServer] which creates a gRPC server with
// health checking, logging, recovery, and timeout interceptors:
//
//	srv := grpcserver.NewServer(db)
//	srv.GRPCServer.Serve(listener)
//
// # Server Discovery
//
// When the server starts, it writes a server.json file containing connection
// information (address, port, PID) to the user's cache directory. This allows
// clients to automatically discover running servers without configuration.
//
// The server info file locations are platform-specific:
//   - Windows: %LOCALAPPDATA%\clonr\server.json
//   - Linux: ~/.cache/clonr/server.json
//   - macOS: ~/Library/Caches/clonr/server.json
//
// # Duplicate Prevention
//
// Before starting, the server checks if another instance is already running
// using [IsServerRunning]. This function reads the server.json file and
// verifies the PID is a running clonr process using goprocess.
//
// # Interceptors
//
// The server includes three interceptors:
//   - Logging: Logs all RPC calls with method name, status, and duration
//   - Recovery: Catches panics and converts them to gRPC errors
//   - Timeout: Enforces a 30-second timeout on all requests
package grpc
