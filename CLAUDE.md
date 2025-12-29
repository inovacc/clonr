# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Clonr is a CLI tool and server for managing Git repositories. It provides an interactive TUI for cloning, organizing, and working with multiple repositories via command line or API.

## Build Commands

### Standard Build
```sh
# Build with BoltDB (default embedded database)
go build -o clonr .

# Build with SQLite instead
go build -tags sqlite -o clonr .
```

### Using Task (Taskfile.dev)
```sh
task test         # Run golangci-lint fmt, golangci-lint run, tests with race detector, and benchmarks
task upgrade      # Update dependencies and tidy go.mod
task build-dev    # Build using goreleaser with snapshot and clean
task build-prod   # Build production snapshot with goreleaser
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

### Database Abstraction Pattern
The project uses a **singleton database pattern** with build-time database selection:

- `internal/database/database.go` defines the `Store` interface and exposes `GetDB()` singleton
- `internal/database/bolt.go` and `internal/database/sqlite.go` provide implementations
- Database is selected at **build time** via build tags (`-tags sqlite` for SQLite, default is BoltDB)
- Always use `database.GetDB()` to access the database throughout the codebase
- The database is initialized once via `init()` and shared globally

### CLI Architecture
The CLI uses **standard library `flag` package** (not cobra or similar frameworks):

- `main.go` handles all command routing and flag parsing
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
internal/
├── cli/         - Bubbletea TUI components
├── core/        - Business logic (clone, add, map, config, etc.)
├── database/    - Database abstraction and implementations
├── model/       - Data models (Repository, Config)
├── server/      - Gin-based HTTP API server
├── monitor/     - Repository monitoring functionality
├── params/      - Parameter handling
└── menu/        - Menu utilities
```

### Repository Model
The `model.Repository` struct tracks:
- `UID` - Unique identifier (UUID)
- `URL` - Remote repository URL
- `Path` - Local filesystem path
- `Favorite` - Boolean favorite flag
- `ClonedAt`, `UpdatedAt`, `LastChecked` - Timestamps

### Configuration System
Configuration is stored in the database (not files):
- Default clone directory (default: `~/clonr`)
- Editor (default: `code`)
- Terminal (optional)
- Monitor interval in seconds (default: 300)
- Server port (default: 4000)

Access via `database.GetDB().GetConfig()` and `database.GetDB().SaveConfig()`

## Development Patterns

### Adding a New Command
1. Add command case in `main.go:executeCommand()`
2. Implement command function (e.g., `cmdFoo()`) in `main.go`
3. Add corresponding business logic in `internal/core/foo.go`
4. If interactive UI needed, create Bubbletea model in `internal/cli/foo.go`
5. Update help text in `main.go:printUsage()`

### Database Operations
Always follow this pattern:
```go
db := database.GetDB()
// Use db.SaveRepo(), db.GetAllRepos(), etc.
```

Never instantiate database implementations directly - always use the singleton.

### Error Handling
- Core functions return errors; they don't print
- Main command functions (cmdXxx) print errors and return them
- Use `fmt.Errorf()` with `%w` for error wrapping

### Testing
- Limited test coverage currently (only `internal/core/common_test.go`)
- Tests run with race detector enabled (`-race`)
- Benchmark tests included in the test suite

## Server Mode
The HTTP API server (Gin framework) exposes:
- `GET /repos` - List all repositories
- `POST /repos/update-all` - Pull updates for all repos

Server runs on configured port with graceful shutdown support (SIGINT handling).

## Requirements
- Go 1.25+ (note: go.mod specifies 1.25, README mentions 1.24+)
- Git must be installed for clone operations
- Optional: [Task](https://taskfile.dev/) for task automation
