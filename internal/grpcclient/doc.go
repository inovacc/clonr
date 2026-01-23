// Package grpcclient provides the gRPC client for connecting to the Clonr server.
//
// The client implements all repository and configuration operations by forwarding
// requests to the gRPC server. It uses a singleton pattern to ensure a single
// connection is shared across the application.
//
// # Getting the Client
//
// Use [GetClient] to obtain the singleton client instance:
//
//	client, err := grpcclient.GetClient()
//	if err != nil {
//	    // Handle connection error
//	}
//	repos, err := client.GetAllRepos()
//
// # Server Discovery
//
// The client automatically discovers the server address using [discoverServerAddress]
// with the following priority:
//
//  1. CLONR_SERVER environment variable
//  2. Server info file (~/.cache/clonr/server.json) with PID verification
//  3. Port probing (50051-50055) with gRPC health check
//  4. Client config file (~/.config/clonr/client.json)
//  5. Default fallback: localhost:50051
//
// # PID Verification
//
// When reading the server info file, the client verifies the PID is actually
// a running clonr process using goprocess before attempting to connect. This
// prevents connecting to unrelated processes that may have reused the PID.
//
// # Health Checking
//
// The client performs a gRPC health check during initialization to verify
// the server is responsive. If the server is not healthy, an error is returned
// with instructions on how to start the server.
//
// # Timeout
//
// All gRPC requests have a 30-second timeout by default.
package grpcclient
