# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Clonr is a **unified client-server tool** for managing Git repositories using **gRPC** architecture. A single `clonr` binary provides both client commands and server functionality. The persistent server manages repository metadata via BoltDB, while client commands provide an interactive TUI and execute git operations locally.

## Build Commands

### Using Taskfile (Recommended)

```sh
task build         # Build clonr binary (includes proto generation)
task proto         # Generate protobuf code only
task install       # Install to GOPATH/bin
task clean         # Clean build artifacts
task clean:all     # Clean everything (build + proto + coverage)
task test          # Run tests with coverage
task lint          # Run golangci-lint
task check         # Run all quality checks (fmt, vet, lint, test)
task --list        # List all available tasks
```

### Manual Build

```sh
go run scripts/proto/generate.go  # Generate protobuf code
go build -o bin/clonr .           # Build binary
```

### Testing

```sh
# Run all tests with race detector
go test -race -p=1 ./...

# Run tests with benchmarks
go test -race -v -bench=. -benchmem ./...

# Linting
golangci-lint fmt
golangci-lint run
```

## Code Architecture

### Client-Server Architecture

Clonr uses a **unified binary** with **gRPC client-server architecture**:

#### Server Side (`clonr server`)
- **Persistent process** that manages the repository database
- Exposes 12 RPC methods via gRPC (mirrors `Store` interface exactly)
- Uses `database.GetDB()` singleton to access BoltDB
- Runs on port 50051 by default (configurable)
- Started with `clonr server start`
- Located in `cmd/server.go` and `internal/grpcserver/`

#### Client Side (`clonr <command>`)
- **Lightweight commands** that connect to server via gRPC for database operations
- Executes git operations (clone, pull) **locally** on the client machine
- Uses `grpcclient.GetClient()` singleton to connect to server
- Cobra-based CLI with subcommands (located in `cmd/`)
- Bubbletea TUI components (located in `internal/cli/`)

### Database Abstraction Pattern

The **server** uses a singleton database pattern:

- `internal/database/database.go` defines the `Store` interface and exposes `GetDB()` singleton
- `internal/database/bolt.go` provides the BoltDB implementation
- **Server code** uses `database.GetDB()` to access the database
- The database is initialized once via `init()` and shared globally on the server

### gRPC Client Abstraction

The **client** uses a singleton gRPC client pattern:

- `internal/grpcclient/client.go` provides `GetClient()` singleton
- Client methods mirror the `Store` interface exactly (all 12 methods)
- **Core business logic** (`internal/core/*.go`) uses `grpcclient.GetClient()` instead of `database.GetDB()`
- Server discovery priority: `CLONR_SERVER` env var → server.json (with PID check) → port probe → `localhost:50051`
- Uses `goprocess` to verify PIDs are actually running clonr processes
- 30-second timeout on all gRPC requests

### CLI Architecture

The CLI uses **Cobra framework** for command structure in a **unified binary**:

- `main.go` initializes the root command
- `cmd/*.go` contains command implementations:
  - Client commands: `clone.go`, `list.go`, `add.go`, etc.
  - Server commands: `server.go` (contains `clonr server start`)
  - Service management: `service.go` (contains `clonr service --install/--start/--stop`)
- Commands can be invoked directly or through an interactive menu
- URL detection: If first arg matches URL patterns (http://, https://, git@, etc.), it's treated as a clone operation
- Interactive mode is triggered when no arguments are provided

### Bubbletea UI Components

The project heavily uses [Bubbletea](https://github.com/charmbracelet/bubbletea) for interactive TUIs:

- `internal/cli/menu.go` - Main interactive menu
- `internal/cli/configure.go` - Configuration wizard with form navigation
- `internal/cli/repolist.go` - Repository list with filtering and actions
- `internal/cli/clone.go` - Clone progress UI

All Bubbletea models follow the standard Init/Update/View pattern. When implementing new TUIs:

1. Create a model struct with state
2. Implement `Init() tea.Cmd`, `Update(tea.Msg) (tea.Model, tea.Cmd)`, and `View() string`
3. Use Bubbles components (list, textinput, etc.) for common UI patterns
4. Style with Lipgloss for consistent appearance

### Core Business Logic Layer

`internal/core/` contains all business logic, separated from UI:

- **Separation of concerns**: Core functions handle logic, CLI handles presentation
- Clone operations are split into `PrepareClonePath()` (validation/setup) and `SaveClonedRepo()` (persistence)
- This split allows the Bubbletea UI to handle the git clone process while core handles validation
- Core functions should not print to stdout/stderr directly - return errors instead

### Module Structure

```
clonr/
├── main.go                           # CLI entry point
├── cmd/                              # Commands (Cobra)
│   ├── root.go                       # Root command
│   ├── clone.go, list.go, etc.      # Client commands
│   ├── server.go                     # Server commands (clonr server start)
│   └── service.go                    # Service management (clonr service --install)
├── docs/                             # Project documentation
│   ├── GRPC_IMPLEMENTATION_GUIDE.md  # gRPC implementation details
│   ├── ROADMAP.md                    # Project roadmap
│   └── ...                           # Other documentation
├── internal/
│   ├── cli/                          # Bubbletea TUI components
│   ├── core/                         # Business logic (uses gRPC client)
│   ├── database/                     # Database abstraction (server-side)
│   ├── model/                        # Data models (Repository, Config)
│   ├── grpcserver/                   # gRPC server implementation
│   │   ├── server.go                 # Server setup
│   │   ├── serverinfo.go             # Server info file & PID checking
│   │   ├── service.go                # RPC implementations (12 methods)
│   │   ├── mapper.go                 # Proto ↔ Model conversions
│   │   └── interceptors.go           # Logging, recovery, timeout
│   ├── grpcclient/                   # gRPC client wrapper
│   │   ├── client.go                 # Client methods (mirrors Store)
│   │   └── discovery.go              # Server address discovery + PID check
│   └── monitor/                      # Repository monitoring
├── api/proto/v1/                     # Protocol Buffer definitions
│   ├── common.proto                  # Common messages (Empty)
│   ├── repository.proto              # Repository messages + 9 RPCs
│   ├── config.proto                  # Config messages + 2 RPCs
│   └── clonr.proto                   # Service definition (12 RPCs)
├── pkg/api/v1/                       # Generated protobuf code
│   ├── *.pb.go                       # Generated Go code
│   └── *_grpc.pb.go                  # Generated gRPC code
└── scripts/proto/                    # Proto generation
    └── generate.go                   # Cross-platform proto gen
```

### Repository Model

The `model.Repository` struct tracks:

- `UID` - Unique identifier (UUID)
- `URL` - Remote repository URL
- `Path` - Local filesystem path
- `Favorite` - Boolean favorite flag
- `ClonedAt`, `UpdatedAt`, `LastChecked` - Timestamps

### Configuration System

Configuration is stored in the **server's database** (not files):

- Default clone directory (default: `~/clonr`)
- Editor (default: `code`)
- Terminal (optional)
- Monitor interval in seconds (default: 300)
- Server port (default: 50051)

**Client access:** `grpcclient.GetClient().GetConfig()` and `.SaveConfig()`
**Server access:** `database.GetDB().GetConfig()` and `.SaveConfig()`

### Automatic Server Discovery

Client **automatically discovers** running servers without configuration:

**Discovery Priority:**
1. `CLONR_SERVER` environment variable (explicit override)
2. **Server info file** - `~/.cache/clonr/server.json` (written by server at startup)
   - Windows: `C:\Users\<user>\AppData\Local\clonr\server.json`
   - Linux: `~/.cache/clonr/server.json`
   - macOS: `~/Library/Caches/clonr/server.json`
   - Contains: address, port, PID, started_at timestamp
   - **PID verified using goprocess** before connecting
   - Automatically cleaned up when server stops gracefully
3. **Auto-probe** common ports (50051-50055) - verifies with gRPC health check
4. `~/.config/clonr/client.json` config file (legacy, still supported)
5. Default fallback: `localhost:50051`

**Implementation Details:**
- **Server writes** `server.json` on startup (`internal/grpcserver/serverinfo.go`)
- **Server checks** for duplicate instances using `IsClonrProcessRunning()` with goprocess
- **Client reads** `server.json` first (`internal/grpcclient/discovery.go`)
- **Client verifies** PID is a running clonr process using goprocess before network check
- Stale files (server not running or wrong process) are auto-cleaned
- Quick TCP port check (500ms timeout per port) for probing
- gRPC health check verification (500ms timeout)
- Uses `os.UserCacheDir()` for cross-platform local data directory

**Benefits:**
- **Instant discovery** - no port probing needed if server.json exists
- **Reliable PID checking** - goprocess verifies it's actually a clonr Go process
- **Silent duplicate prevention** - second server start exits silently if already running
- Zero configuration for standard setups
- Automatic cleanup of stale discovery info
- Cross-platform compatible

## Development Patterns

### Adding a New Command

1. Create new command file in `cmd/foo.go` using Cobra pattern
2. Add corresponding business logic in `internal/core/foo.go`
3. Core logic should use `grpcclient.GetClient()` for database operations
4. If interactive UI needed, create Bubbletea model in `internal/cli/foo.go`
5. Register command in `cmd/root.go` init function

Example:
```go
// cmd/foo.go
var fooCmd = &cobra.Command{
    Use:   "foo",
    Short: "Do something",
    RunE: func(cmd *cobra.Command, args []string) error {
        return core.DoFoo(args)
    },
}

func init() {
    rootCmd.AddCommand(fooCmd)
}

// internal/core/foo.go
func DoFoo(args []string) error {
    client, err := grpcclient.GetClient()
    if err != nil {
        return fmt.Errorf("failed to connect to server: %w", err)
    }
    // Use client methods...
    return nil
}
```

### gRPC Client Operations (Client-Side)

**Core business logic** uses the gRPC client singleton:

```go
client, err := grpcclient.GetClient()
if err != nil {
    return fmt.Errorf("failed to connect to server: %w", err)
}
// Use client.SaveRepo(), client.GetAllRepos(), etc.
```

Never use `database.GetDB()` in client code - only in server code.

### Database Operations (Server-Side)

**Server code** uses the database singleton:

```go
db := database.GetDB()
// Use db.SaveRepo(), db.GetAllRepos(), etc.
```

Never instantiate database implementations directly - always use the singleton.

### Adding a New RPC Method

If you need to add a new database operation:

1. Add method to `Store` interface in `internal/database/database.go`
2. Implement in `internal/database/bolt.go`
3. Define request/response messages in `api/proto/v1/*.proto`
4. Add RPC to service definition in `api/proto/v1/clonr.proto`
5. Run `make proto` to regenerate code
6. Implement RPC in `internal/grpcserver/service.go`
7. Add client wrapper method in `internal/grpcclient/client.go`
8. Use in core logic via `grpcclient.GetClient()`

### Error Handling

- Core functions return errors; they don't print
- Main command functions (cmdXxx) print errors and return them
- Use `fmt.Errorf()` with `%w` for error wrapping

### Testing

- Limited test coverage currently (only `internal/core/common_test.go`)
- Tests run with race detector enabled (`-race`)
- Benchmark tests included in the test suite

## Running the Application

### Start the Server

The server must be running before using client commands. You can run it directly or as a service.

#### Option 1: Run Directly

```sh
# Start on default port (50051)
clonr server start

# Start on custom port
clonr server start --port 50052
```

#### Option 2: Run as a System Service (Recommended)

```sh
# Install the service (runs 'clonr server start' as a service)
clonr service --install

# Start the service
clonr service --start

# Check status
clonr service --status

# Stop the service
clonr service --stop

# Uninstall the service
clonr service --uninstall
```

**Service Features:**
- Cross-platform: Windows Service, systemd (Linux), launchd (macOS)
- Auto-start on system boot
- Runs in background
- Uses `github.com/kardianos/service` library (v1.2.4)
- Service executes `clonr server start --port <port>` internally
- Automatically finds clonr executable using `findClonrExecutable()` in `cmd/service.go`

**Server Features:**
- Graceful shutdown (SIGINT/SIGTERM handling)
- Logging interceptor (logs all RPCs with duration)
- Recovery interceptor (catches panics)
- Timeout interceptor (30s per request)
- Uses configured port from database or flag
- Writes `server.json` for client discovery on startup
- Cleans up `server.json` on shutdown

### Use Client Commands

Once server is running:

```sh
clonr list
clonr clone https://github.com/user/repo
clonr configure
```

If server is not running, client commands will fail with a helpful error:
```
Error: Server not running
Start the server with: clonr server start
```

### Non-Interactive Mode

Some commands support non-interactive mode for scripting and CI workflows:

```sh
# Remove repository by URL (non-interactive)
clonr remove https://github.com/user/repo
clonr remove --url https://github.com/user/repo

# Mirror organization repositories without TUI
clonr org mirror <org_name> --no-tui
```

## gRPC Service Definition

The service exposes 12 RPC methods (defined in `api/proto/v1/clonr.proto`):

### Repository Operations
- `SaveRepo(url, path) -> success`
- `RepoExistsByURL(url) -> exists`
- `RepoExistsByPath(path) -> exists`
- `InsertRepoIfNotExists(url, path) -> inserted`
- `GetAllRepos() -> repositories[]`
- `GetRepos(favoritesOnly) -> repositories[]`
- `SetFavoriteByURL(url, favorite) -> success`
- `UpdateRepoTimestamp(url) -> success`
- `RemoveRepoByURL(url) -> success`

### Configuration Operations
- `GetConfig() -> config`
- `SaveConfig(config) -> success`

### Health Check
- `Ping() -> Empty`

All RPCs use unary (request-response) pattern with 30-second timeouts.

## Requirements

- Go 1.24+ (go.mod specifies 1.25)
- Git must be installed for clone operations
- Protocol Buffers compiler (protoc) for proto generation
- Both client and server must use compatible protobuf versions
