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
- Uses `grpc.GetClient()` singleton to connect to server
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
- **Core business logic** (`internal/core/*.go`) uses `grpc.GetClient()` instead of `database.GetDB()`
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
- `internal/cli/profile_login.go` - OAuth device flow TUI

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

### KeePass Token Storage

`internal/core/keepass.go` provides KeePass database management for secure profile token storage:

- **Library**: `github.com/tobischo/gokeepasslib/v3`
- **Format**: Standard KeePass `.kdbx` database format
- **Location**: `~/.config/clonr/clonr.kdbx`

**Key Functions:**
```go
// Create or open KeePass database
kpm, err := core.NewKeePassManager(password)

// Store a profile token
kpm.SetProfileToken(profileName, host, token)

// Retrieve a profile token
token, err := kpm.GetProfileToken(profileName, host)

// Delete a profile token
kpm.DeleteProfileToken(profileName, host)

// List all profiles
profiles := kpm.ListProfiles()

// Change database password
kpm.ChangePassword(newPassword)

// Check if KeePass database exists
core.KeePassDBExists() bool

// Get database path
path, err := core.GetKeePassDBPath()
```

**Token Storage Priority:**
```
Profile Token Storage:
┌─────────────────────────────────────────────────┐
│  1. KeePass (if database exists)                │
│     └─ Password from TPM (required)             │
│  2. System Keyring                              │
│  3. Encrypted file (AES-256-GCM)                │
└─────────────────────────────────────────────────┘
```

### TPM 2.0 Integration (Hardware-Backed Encryption)

`internal/core/tpm*.go` provides TPM 2.0 key management for hardware-backed encryption using the **sealbox** library:

- **Platform Support**: Linux (uses `/dev/tpmrm0`), Windows support planned (via TBS)
- **Library**: `github.com/inovacc/sealbox` (wraps `github.com/google/go-tpm`)
- **Development**: Uses `go mod replace` to link local sealbox at `/home/dyam/shared/personal/GolandProjects/keystore`
- **Fallback**: Non-Linux platforms use password-based KeePass or file-based encryption

**Files:**
- `tpm.go` - TPM 2.0 key sealing/unsealing via sealbox (Linux build tag)
- `tpm_stub.go` - Stub for non-Linux platforms (returns `ErrTPMNotSupported`)
- `tpm_keystore.go` - Sealed key storage using sealbox's FileKeyStore

**Key Functions:**
```go
// Check TPM availability
core.IsTPMAvailable() bool

// Initialize new TPM-sealed key
core.InitializeTPMKey() error

// Get master key from TPM
core.GetTPMSealedMasterKey() ([]byte, error)

// Check if TPM key exists
core.HasTPMKey() bool

// Remove TPM-sealed key
core.ResetTPMKey() error

// Get TPM key store path
core.GetTPMKeyStorePath() (string, error)
```

**Sealbox Features:**
- **PCR Policy Binding**: Keys can be bound to specific PCR values
- **Password Protection**: Optional additional password layer
- **Versioned SealedData**: Forward-compatible sealed key format
- **Key Hierarchy**: Support for primary/derived key patterns
- **Retry Logic**: Automatic retry on transient TPM errors

**TPM + KeePass Flow:**
```
Token Storage with TPM:
┌─────────────────────────────────────────────────┐
│  1. TPM unseals master key (via sealbox)        │
│  2. Derive KeePass password from master key     │
│  3. Open KeePass database (no user password)    │
│  4. Store/retrieve tokens from KeePass          │
└─────────────────────────────────────────────────┘
```

**Storage Locations:**
| File | Platform | Path |
|------|----------|------|
| TPM Sealed Key | Linux | `~/.config/clonr/.clonr_sealed_key` |
| TPM Sealed Key | macOS | `~/Library/Application Support/clonr/.clonr_sealed_key` |
| TPM Sealed Key | Windows | `%LOCALAPPDATA%\clonr\.clonr_sealed_key` |
| KeePass DB | All | `~/.config/clonr/clonr.kdbx` |

**Security Benefits (TPM used internally when available):**
- Key material bound to hardware (cannot be extracted)
- No password required - authentication automatic via TPM
- Resistant to offline attacks
- KeePass database portable but protected
- Keys cannot be backed up (by design)

### Git Client (Credential Helper Pattern)

`internal/git/client.go` provides a centralized git client inspired by [GitHub CLI](https://github.com/cli/cli):

- **Credential Helper Pattern**: Uses `git -c credential.helper=` to inject authentication
- Clonr registers itself as a git credential helper: `clonr auth git-credential`
- Tokens never appear in process arguments (more secure than URL injection)
- Supports host-specific or all-matching credential patterns

```go
client := git.NewClient()
// Clone with automatic authentication
err := client.Clone(ctx, "https://github.com/user/repo", "/path/to/clone")

// Push with authentication
err = client.Push(ctx, "origin", "main", git.PushOptions{SetUpstream: true})
```

**Available Operations:**
- `Clone`, `Pull`, `Push`, `Fetch` - Remote operations with credential helper
- `Commit`, `Tag` - Local operations
- `Stash`, `StashPop`, `StashList`, `StashDrop` - Stash management
- `Checkout`, `Merge`, `ListBranches` - Branch operations
- `Status`, `CurrentBranch`, `IsRepository` - Repository info

### Security Scanning (Gitleaks Integration)

`internal/security/leaks.go` integrates [gitleaks](https://github.com/zricethezav/gitleaks) for secret detection:

- **Pre-push scanning**: Automatically scans unpushed commits before `clonr push`
- **Manual scanning**: Use `clonr scan [path]` to scan directories or git history
- **Gitleaks rules**: Detects API keys, tokens, passwords, private keys, cloud credentials
- **.gitleaksignore support**: Respects ignore patterns in repository

```go
scanner, _ := security.NewLeakScanner()
_ = scanner.LoadGitleaksIgnore(repoPath)

// Scan unpushed commits
result, err := scanner.ScanUnpushedCommits(ctx, repoPath)

// Scan directory files
result, err := scanner.ScanDirectory(ctx, path)

// Scan full git history
result, err := scanner.ScanGitRepo(ctx, path)
```

**Push Workflow:**
1. User runs `clonr push`
2. Pre-push security check runs with TUI spinner
3. If secrets detected: push aborted, findings displayed
4. If clean: push proceeds with authentication
5. Use `--skip-leaks` to bypass (not recommended)

### Module Structure

```
clonr/
├── main.go                           # CLI entry point
├── cmd/                              # Commands (Cobra)
│   ├── root.go                       # Root command
│   ├── helpers.go                    # Shared CLI utilities (prompts, formatting, boxes)
│   ├── gh.go                         # GitHub command helpers (time/size formatting)
│   ├── clone.go, list.go, etc.      # Client commands
│   ├── server.go                     # Server commands (clonr server start)
│   ├── service.go                    # Service management (clonr service --install)
│   ├── profile.go                    # Profile parent command
│   ├── profile_add.go                # Add profile with OAuth or PAT (--token flag)
│   ├── profile_list.go               # List profiles
│   ├── profile_use.go                # Set active profile
│   ├── profile_remove.go             # Remove profile
│   ├── profile_status.go             # Show profile info
│   ├── auth_git_credential.go        # Git credential helper (internal)
│   ├── commit.go                     # Git commit with profile auth
│   ├── tag.go                        # Git tag creation
│   ├── pull.go                       # Git pull with profile auth
│   ├── push.go                       # Git push with pre-push leak scan
│   ├── stash.go                      # Git stash operations
│   ├── checkout.go                   # Git checkout branches
│   ├── merge.go                      # Git merge branches
│   ├── scan.go                       # Manual secret scanning
├── docs/                             # Project documentation
│   ├── GRPC_IMPLEMENTATION_GUIDE.md  # gRPC implementation details
│   ├── ROADMAP.md                    # Project roadmap
│   └── ...                           # Other documentation
├── internal/
│   ├── application/                  # Application constants and paths
│   │   └── application.go            # AppName, AppExeName constants, GetApplicationDirectory()
│   ├── auth/                         # Generic authentication utilities
│   │   └── resolver.go               # Token resolver framework (builder pattern)
│   ├── cli/                          # Bubbletea TUI components
│   │   └── profile_login.go          # OAuth device flow TUI
│   ├── core/                         # Business logic (uses gRPC client)
│   │   ├── auth.go                   # Token resolution with profile support
│   │   ├── context.go                # Context timeout helpers (WithShortTimeout, etc.)
│   │   ├── gh_client.go              # GitHub OAuth client creation helper
│   │   ├── profile.go                # Profile management logic
│   │   ├── oauth.go                  # GitHub OAuth device flow
│   │   ├── keyring.go                # Secure keyring storage
│   │   ├── keepass.go                # KeePass database manager for tokens
│   │   ├── encrypt.go                # AES-256-GCM encryption + KeePass password
│   │   ├── tpm.go                    # TPM 2.0 key sealing (Linux only)
│   │   ├── tpm_stub.go               # TPM stub for non-Linux platforms
│   │   └── tpm_keystore.go           # TPM sealed key storage
│   ├── encoding/                     # Encoding and file utilities
│   │   ├── json.go                   # Generic LoadJSON[T], SaveJSON[T], ParseJSON[T]
│   │   └── file.go                   # FileExists, EnsureDir, WriteFileSecure
│   ├── git/                          # Git client with credential helper
│   │   └── client.go                 # Centralized git operations (gh CLI pattern)
│   ├── mapper/                       # Shared type conversions
│   │   └── grpc.go                   # Proto ↔ Model conversions (shared by server/client)
│   ├── security/                     # Security scanning (gitleaks)
│   │   └── leaks.go                  # Secret detection in commits/files
│   ├── database/                     # Database abstraction (server-side)
│   ├── model/                        # Data models (Repository, Config, Profile)
│   ├── server/grpc/                  # gRPC server implementation
│   │   ├── server.go                 # Server setup with interceptor chain
│   │   ├── serverinfo.go             # Server info file & PID checking
│   │   ├── service.go                # RPC implementations (12 methods)
│   │   ├── mapper.go                 # Delegates to internal/mapper
│   │   ├── validation.go             # gRPC validation helpers (RequiredString, etc.)
│   │   └── interceptors.go           # Logging, recovery, timeout, context check
│   ├── client/grpc/                  # gRPC client wrapper
│   │   ├── client.go                 # Client methods (mirrors Store)
│   │   └── discovery.go              # Server address discovery + PID check
│   └── monitor/                      # Repository monitoring
├── api/proto/v1/                     # Protocol Buffer definitions
│   ├── common.proto                  # Common messages (Empty)
│   ├── repository.proto              # Repository messages + 9 RPCs
│   ├── config.proto                  # Config messages + 2 RPCs
│   └── clonr.proto                   # Service definition (12 RPCs)
├── internal/api/v1/                       # Generated protobuf code
│   ├── *.pb.go                       # Generated Go code
│   └── *_grpc.pb.go                  # Generated gRPC code
└── scripts/proto/                    # Proto generation
    └── generate.go                   # Cross-platform proto gen
```

### Shared Utility Packages

The codebase uses several shared utility packages to avoid code duplication:

#### `internal/application` - Application Constants

Centralized application name and path constants:

```go
const (
    AppName           = "clonr"      // Used for directories, process identification
    AppExeName        = "clonr"      // Executable name (no extension)
    AppExeNameWindows = "clonr.exe"  // Windows executable name
)

// Get application config directory (cross-platform)
dir, err := application.GetApplicationDirectory()
```

#### `internal/mapper` - Proto ↔ Model Conversions

Shared conversion functions used by both server and client:

```go
import "github.com/inovacc/clonr/internal/mapper"

// Convert between proto and model types
protoRepo := mapper.ModelToProtoRepository(repo)
modelRepo := mapper.ProtoToModelRepository(protoRepo)

// Available: Repository, Config, Profile, Workspace conversions
```

#### `internal/core` - Business Logic Helpers

**GitHub Client Helper** (`gh_client.go`):
```go
// Create authenticated GitHub client (replaces 16+ duplicate patterns)
client := core.NewGitHubClient(ctx, token)

// Or with fresh background context
client := core.NewGitHubClientWithContext(token)

// For direct HTTP requests (e.g., asset downloads)
httpClient := core.NewOAuth2HTTPClient(ctx, token)
```

**Context Helpers** (`context.go`):
```go
// Predefined timeouts
ctx, cancel := core.WithShortTimeout()   // 30 seconds
ctx, cancel := core.WithMediumTimeout()  // 2 minutes
ctx, cancel := core.WithLongTimeout()    // 5 minutes
ctx, cancel := core.WithXLongTimeout()   // 10 minutes

// Custom timeout
ctx, cancel := core.WithTimeout(45 * time.Second)
```

#### `internal/auth` - Token Resolution Framework

Generic token resolver with builder pattern:

```go
import "github.com/inovacc/clonr/internal/auth"

result, err := auth.NewResolver("GitHub").
    WithFlagValue(flagToken).
    WithEnvs("GITHUB_TOKEN", "GH_TOKEN").
    WithProvider(customProvider).
    WithHelpMessage("Get token at: https://github.com/settings/tokens").
    Resolve()

// result.Token, result.Source, result.Name
```

#### `internal/encoding` - File and JSON Utilities

Generic file and JSON operations:

```go
import "github.com/inovacc/clonr/internal/encoding"

// Load/save JSON with generics
config, err := encoding.LoadJSON[Config](path)  // Returns nil, nil if not exists
err := encoding.SaveJSON(path, config)

// File utilities
if encoding.FileExists(path) { ... }
if encoding.DirExists(path) { ... }
err := encoding.EnsureDir(path)
err := encoding.WriteFileSecure(path, data)  // 0600 permissions
```

#### `internal/server/grpc` - gRPC Validation Helpers

Reusable validation functions for service methods:

```go
// In service method implementations
if err := RequiredString(req.GetName(), "name"); err != nil {
    return nil, err
}
if err := RequiredURL(req.GetUrl()); err != nil {
    return nil, err
}
if err := RequiredOneOf(map[string]string{"url": url, "path": path}); err != nil {
    return nil, err
}

// Error helpers
return nil, NotFoundError("repository")
return nil, InternalErrorf("failed to save: %v", err)
```

#### `cmd/helpers.go` - CLI Shared Utilities

Centralized helper functions for CLI commands to avoid duplication:

```go
// Token storage formatting
formatTokenStorage(ts model.TokenStorage) string  // Returns "encrypted (TPM)" or "plain text"

// User interaction
promptConfirm(prompt string) bool  // Y/N confirmation prompt

// Path handling
expandPath(path string) (string, error)  // Expands ~ and returns absolute path

// Empty result messaging
printEmptyResult(resourceType, createCmd string)  // "No X configured. Create one with: Y"

// String utilities
centerString(s string, width int) string      // Center text in field
truncateString(s string, maxLen int) string   // Truncate with ellipsis

// Box drawing for terminal UI
const boxWidth = 64
printBoxHeader(title string)                           // ╔═══ TITLE ═══╗
printBoxLine(label, value string)                      // ║  Label: Value  ║
printBoxFooter()                                       // ╚══════════════════╝
printInfoBox(title string, items map[string]string, order []string)  // Complete box
```

#### `cmd/gh.go` - GitHub Command Helpers

Shared helpers for GitHub CLI commands (issues, PRs, actions, releases):

```go
// Time formatting
formatAge(t time.Time) string           // "2h ago", "3d ago", "2w ago"
formatShortDuration(d time.Duration) string  // "1m 30s", "2h 15m"

// Size formatting
formatFileSize(bytes int64) string      // "1.5 MB", "256 KB"

// String utilities
truncateStr(s string, maxLen int) string  // Truncate for table display

// Flag extraction and JSON output
extractGHFlags(cmd *cobra.Command) (repo, token, profile string, jsonOutput bool, err error)
outputJSON(v any) error  // Marshal and print JSON
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

**Client access:** `grpc.GetClient().GetConfig()` and `.SaveConfig()`
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
3. Core logic should use `grpc.GetClient()` for database operations
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
    client, err := grpc.GetClient()
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
client, err := grpc.GetClient()
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
8. Use in core logic via `grpc.GetClient()`

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

### Git Commands with Profile Authentication

Clonr provides git commands that automatically use profile authentication:

```sh
# Commit changes (uses git client)
clonr commit -m "feat: add feature"
clonr commit -a -m "fix: bug fix"      # Stage all modified files

# Create tags
clonr tag v1.0.0
clonr tag v1.0.0 -m "Release 1.0.0"    # Annotated tag

# Pull changes (with profile auth)
clonr pull
clonr pull origin main

# Push changes (with pre-push security scan)
clonr push
clonr push -u origin main              # Set upstream
clonr push --tags                      # Push all tags
clonr push --skip-leaks                # Skip security scan (not recommended)

# Stash operations
clonr stash                            # Stash changes
clonr stash push -m "WIP"              # Stash with message
clonr stash list                       # List stashes
clonr stash pop                        # Pop latest stash
clonr stash drop                       # Drop latest stash

# Branch operations
clonr checkout feature-branch
clonr checkout -b new-branch           # Create and checkout
clonr merge feature-branch
clonr merge --squash feature-branch

# Security scanning
clonr scan                             # Scan current directory
clonr scan /path/to/repo               # Scan specific path
clonr scan --git                       # Scan git history
```

### Profile Management with PAT Support

Profiles support both OAuth device flow and direct Personal Access Token (PAT) input:

```sh
# Add profile with OAuth device flow (interactive)
clonr profile add github

# Add profile with Personal Access Token (non-interactive)
clonr profile add github --token ghp_xxxxxxxxxxxx

# List profiles
clonr profile list

# Switch active profile
clonr profile use work

# Show current profile status
clonr profile status

# Remove profile
clonr profile remove github
```

**PAT Token Benefits:**
- Skip OAuth flow for CI/CD environments
- Direct token validation with GitHub API
- Same secure storage (keyring or encrypted file)

### TPM 2.0 Key Management (Automatic)

Hardware-backed encryption using TPM 2.0 is handled automatically:

- **Auto-initialization**: When storage is needed and TPM is available, a sealed key is created silently
- **Transparent**: No CLI commands needed - TPM is used internally by the storage layer
- **Fallback**: If TPM unavailable, falls back to file-based encryption

**TPM Features:**
- Hardware-bound encryption keys (cannot be extracted)
- No password required - automatic authentication
- Keys cannot be backed up (security by design)

**TPM Permissions (Linux):**
The TPM device `/dev/tpmrm0` is owned by `tss:tss`. Users must be in the `tss` group:
```sh
# Add user to tss group
sudo usermod -aG tss $USER

# Apply immediately (or log out/in)
newgrp tss
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

## Key Dependencies

- **gRPC/Protobuf**: `google.golang.org/grpc`, `google.golang.org/protobuf`
- **CLI Framework**: `github.com/spf13/cobra`
- **TUI Components**: `github.com/charmbracelet/bubbletea`, `github.com/charmbracelet/lipgloss`
- **Database**: `go.etcd.io/bbolt` (BoltDB)
- **Security**: `github.com/zricethezav/gitleaks/v8` (secret scanning)
- **Keyring**: `github.com/zalando/go-keyring`
- **TPM 2.0**: `github.com/inovacc/sealbox` (hardware-backed encryption, wraps go-tpm)
- **KeePass**: `github.com/tobischo/gokeepasslib/v3` (secure token storage)
- **Process Management**: `github.com/shirou/gopsutil/v4/process`
- **Encryption**: `golang.org/x/crypto/argon2` (password hashing), `crypto/aes` (AES-256-GCM)
- **Encoding**: `github.com/btcsuite/btcutil/base58` (key encoding)

## Standalone Mode (Instance Synchronization)

### Overview

Standalone mode enables secure synchronization between clonr instances across machines. A server instance can accept connections from client instances, allowing data synchronization with per-client encryption for sensitive data.

### Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                     STANDALONE MODE ARCHITECTURE                            │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  SERVER INSTANCE                      CLIENT INSTANCES                      │
│  ───────────────                      ────────────────                      │
│                                                                             │
│  ┌─────────────────┐                  ┌─────────────────┐                   │
│  │ clonr standalone│                  │ clonr standalone│                   │
│  │ init            │                  │ connect <key>   │                   │
│  └────────┬────────┘                  └────────┬────────┘                   │
│           │                                    │                            │
│           ▼                                    ▼                            │
│  ┌─────────────────┐                  ┌─────────────────┐                   │
│  │ Generates Key   │                  │ Generates       │                   │
│  │ • InstanceID    │◀── Shared ──────▶│ Encryption Key  │                   │
│  │ • API Key       │    via file/     │ (displayed to   │                   │
│  │ • Refresh Token │    clipboard     │  user)          │                   │
│  └────────┬────────┘                  └────────┬────────┘                   │
│           │                                    │                            │
│           ▼                                    ▼                            │
│  ┌─────────────────┐                  ┌─────────────────┐                   │
│  │ clonr standalone│◀── Key Entry ───│ User enters key │                   │
│  │ accept          │    (manual)      │ on server       │                   │
│  └────────┬────────┘                  └─────────────────┘                   │
│           │                                                                 │
│           ▼                                                                 │
│  ┌─────────────────────────────────────────────────────┐                    │
│  │              DATA CLASSIFICATION                     │                    │
│  │  Sensitive (tokens) → Encrypted with client key      │                    │
│  │  Public (repos)     → Stored normally                │                    │
│  └─────────────────────────────────────────────────────┘                    │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Standalone Commands

**Server-side (source instance):**
```bash
clonr standalone init            # Initialize standalone mode, generate key
clonr standalone status          # Show standalone status
clonr standalone accept          # Accept pending client connections
clonr standalone clients         # List registered clients
clonr standalone encrypt setup   # Setup server encryption
```

**Client-side (destination instance):**
```bash
clonr standalone connect <key>   # Connect to a server instance
clonr standalone decrypt --list  # List encrypted synced data
clonr standalone decrypt         # Decrypt synced data
```

**Archive operations:**
```bash
clonr standalone archive <paths> # Create encrypted repo archive
clonr standalone extract <file>  # Extract encrypted archive
```

### Implementation Files

```
internal/standalone/
├── types.go           # StandaloneKey, StandaloneConfig, StandaloneConnection
├── key.go             # Key generation, encoding (Base58), validation
├── crypto.go          # AES-256-GCM encryption, PBKDF2, Argon2 key derivation
├── sync.go            # Sync logic, SyncedData, EncryptionKeyManager
├── handshake.go       # Client-server handshake, per-client encryption
├── archive.go         # Encrypted ZIP archives for repos
└── *_test.go          # Comprehensive test coverage

cmd/
├── standalone.go          # Parent command
├── standalone_init.go     # Initialize standalone mode
├── standalone_status.go   # Show status
├── standalone_connect.go  # Client connection with key display
├── standalone_accept.go   # Accept pending clients (server)
├── standalone_clients.go  # List registered clients
├── standalone_encrypt.go  # Server encryption setup
├── standalone_decrypt.go  # Decrypt synced data
├── standalone_archive.go  # Create encrypted archives
└── standalone_extract.go  # Extract archives
```

### Key Types

```go
// StandaloneKey - shared with clients to establish connections
type StandaloneKey struct {
    Version           int       `json:"version"`
    InstanceID        string    `json:"instance_id"`
    Host              string    `json:"host"`
    Port              int       `json:"port"`
    APIKey            string    `json:"api_key"`       // Base58-encoded
    RefreshToken      string    `json:"refresh_token"` // Base58-encoded
    EncryptionKeyHint string    `json:"encryption_key_hint"`
    ExpiresAt         time.Time `json:"expires_at"`
    Capabilities      []string  `json:"capabilities"`
}

// StandaloneConfig - stored on server instance
type StandaloneConfig struct {
    Enabled      bool      `json:"enabled"`
    IsServer     bool      `json:"is_server"`     // Server accepts connections
    InstanceID   string    `json:"instance_id"`
    Port         int       `json:"port"`
    APIKeyHash   []byte    `json:"api_key_hash"`  // Argon2 hash
    // ...
}

// RegisteredClient - client registered on server
type RegisteredClient struct {
    ClientID          string      `json:"client_id"`
    ClientName        string      `json:"client_name"`
    MachineInfo       MachineInfo `json:"machine_info"`
    EncryptionKeyHash []byte      `json:"encryption_key_hash"` // Argon2
    EncryptionSalt    []byte      `json:"encryption_salt"`
    KeyHint           string      `json:"key_hint"`
    Status            string      `json:"status"` // "active", "suspended"
    // ...
}
```

### Encryption Architecture

**Key Derivation:**
```
Master Key (32 bytes) ─┬─ HKDF("api-auth") ──────▶ API Key
                       └─ HKDF("data-encryption") ▶ Encryption Key

Client Key (32 bytes) ◀── Argon2(display_key, salt)
                          (derived from 32-char hex display key)

Local Storage Key ◀────── Argon2(user_password, local_salt)
```

**Data Classification:**
- **Sensitive** (tokens, credentials): Encrypted with per-client key
- **Public** (repos, workspaces, config): Stored normally

### Handshake Flow

1. **Server** runs `clonr standalone init` → generates standalone key
2. **Client** runs `clonr standalone connect <key>`
   - Parses and validates server key
   - Generates 32-byte encryption key
   - Displays formatted key to user (e.g., `xxxx-xxxx-xxxx-xxxx-xxxx-xxxx-xxxx-xxxx`)
   - Prompts for local password to store connection
3. **User** enters displayed key on server
4. **Server** runs `clonr standalone accept`
   - Lists pending registrations
   - Accepts by entering the displayed key
   - Stores client with hashed encryption key
5. **Sync** can now occur with per-client encryption for sensitive data

### Store Interface Methods

```go
// Pending registrations (server-side)
SavePendingRegistration(reg *standalone.ClientRegistration) error
GetPendingRegistration(clientID string) (*standalone.ClientRegistration, error)
ListPendingRegistrations() ([]*standalone.ClientRegistration, error)
RemovePendingRegistration(clientID string) error

// Registered clients (server-side)
SaveRegisteredClient(client *standalone.RegisteredClient) error
GetRegisteredClient(clientID string) (*standalone.RegisteredClient, error)
ListRegisteredClients() ([]*standalone.RegisteredClient, error)
DeleteRegisteredClient(clientID string) error

// Synced data (encrypted storage)
GetSyncedData(connectionName, dataType, name string) (*standalone.SyncedData, error)
ListSyncedDataByState(state standalone.SyncState) ([]standalone.SyncedData, error)
SaveSyncedData(data *standalone.SyncedData) error
```
