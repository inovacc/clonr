# Clonr [![Test](https://github.com/inovacc/clonr/actions/workflows/test.yml/badge.svg)](https://github.com/inovacc/clonr/actions/workflows/test.yml)

Clonr is a client-server tool for managing Git repositories efficiently. It uses a gRPC architecture where a persistent server manages repository metadata via BoltDB/SQLite, while the CLI client provides an interactive interface for cloning, organizing, and working with multiple repositories.

## Features

- **Clone**: Clone Git repositories from various protocols (HTTPS, HTTP, Git, SSH, FTP, SFTP).
- **Add**: Register existing local Git repositories for management.
- **List**: Interactively list all managed repositories with actions (open, remove, info, stats).
- **Favorites**: Mark repositories as favorites and filter lists by favorites.
- **Remove**: Remove repositories from management via interactive menu selection.
- **Update**: Pull the latest changes for all or specific repositories.
- **Open**: Quickly open favorited repositories in your configured editor.
- **Configure**: Set monitor interval, port, default repository directory, and editor preferences.
- **Map**: Map local directories to search for existing repositories.
- **Status**: Show git status of all managed repositories.
- **Nerds**: Display statistics and metrics for all repositories.
- **Reauthor**: Rewrite git history to change author/committer email and name.
- **GitHub CLI Integration**: Interact with GitHub issues, PRs, actions, and releases directly from clonr.
- **Profile Management**: Manage multiple GitHub authentication profiles with OAuth device flow and secure token storage.
- **Client-Server Architecture**: Persistent gRPC server with centralized database, lightweight CLI client performs git operations locally.

## Installation

Install directly using Go (requires Go 1.24+):

```sh
go install github.com/inovacc/clonr@latest
```

This will place the `clonr` binary in your `$GOPATH/bin` or `$HOME/go/bin` directory. Make sure this directory is in your `PATH`.

## Running Clonr

Clonr uses a client-server architecture. You need to start the server before using the client.

### Option 1: Run Server Directly

```sh
clonr server start
```

The server runs on port 50051 by default and manages the repository database. You can configure a different port:

```sh
clonr server start --port 50052
```

The server will continue running until you stop it with Ctrl+C or `clonr server stop`.

### Option 2: Run Server as a Service (Recommended)

For production use, install the clonr server as a system service:

```sh
# Install the service
clonr service --install

# Start the service
clonr service --start

# Check service status
clonr service --status

# Stop the service
clonr service --stop

# Uninstall the service
clonr service --uninstall
```

**Platform Support:**
- **Windows**: Creates a Windows Service
- **Linux**: Creates a systemd service
- **macOS**: Creates a launchd service

The service will automatically start on system boot and run in the background.

### 2. Use the Client

Once the server is running, use the `clonr` client commands:

```sh
clonr list
clonr clone https://github.com/user/repo
```

All client commands automatically connect to the server.

### Automatic Server Discovery

The client **automatically discovers** running servers without any configuration needed!

Discovery process:
1. **Environment variable**: `CLONR_SERVER` (e.g., `export CLONR_SERVER=localhost:50052`)
2. **Server info file**: Reads `AppData\Local\clonr\server.json` (written by running server)
   - Windows: `C:\Users\<user>\AppData\Local\clonr\server.json`
   - Linux: `~/.local/share/clonr/server.json`
   - macOS: `~/Library/Application Support/clonr/server.json`
3. **Auto-probe**: Searches common ports (50051-50055) for a running server
4. **Config file**: Checks `~/.config/clonr/client.json` if configured
5. **Default fallback**: `localhost:50051`

**No configuration needed** - the server automatically writes its connection info when it starts!

## Usage

### Command Line

Clone repositories directly or use commands:

```sh
clonr <url> [destination]     # Clone directly
clonr [command] [flags]       # Run specific command
clonr                          # Interactive menu
```

#### Available Commands

- `clonr <url> [destination]`: Clone a repository directly (supports https, http, git, ssh, ftp, sftp, git@).
- `clonr add [path]`: Register an existing local Git repository for management.
- `clonr list`: Interactively list all repositories with options to open, remove, view info, or show stats.
- `clonr list --favorites`: Show only favorited repositories.
- `clonr remove` or `clonr rm`: Interactive menu to select and remove repositories.
- `clonr favorite <name>`: Mark a repository as favorite.
- `clonr open`: List favorited repositories and open the selected one in your configured editor.
- `clonr update [repo-name]`: Pull latest changes for all or a specific repository.
- `clonr configure`: Interactive configuration wizard for all settings.
- `clonr configure --show` or `-s`: Display current configuration.
- `clonr configure --reset` or `-r`: Reset configuration to default values.
- `clonr map`: Map a local directory to search and register existing Git repositories.
- `clonr status`: Show the Git status of all managed repositories.
- `clonr nerds`: Display nerd statistics and metrics for all repositories.
- `clonr reauthor`: Rewrite git history to change author/committer identity.
- `clonr reauthor --list`: List all unique author emails in the repository.
- `clonr server start`: Start the gRPC server.
- `clonr server stop`: Stop the running gRPC server.
- `clonr server restart`: Restart the gRPC server.
- `clonr server status`: Show server status (PID, uptime, address).
- `clonr service`: Manage the server as a system service (install, uninstall, start, stop, status).
- `clonr profile`: Manage GitHub authentication profiles (see below).
- `clonr workspace`: Manage workspaces for organizing repositories (see below).
- `clonr data export`: Export all data encrypted with password to base58.
- `clonr data import`: Import data from encrypted export.
- `clonr gh`: GitHub CLI integration (see below).
- `clonr help`: Display help information.

Use `clonr [command] --help` for more details on each command.

### Profile Management

Clonr supports multiple GitHub authentication profiles with secure token storage:

```sh
# Add a new profile (OAuth device flow)
clonr profile add work                  # Opens browser for GitHub OAuth
clonr profile add personal              # Create another profile

# List all profiles
clonr profile list                      # Shows all profiles with default marker

# Switch default profile
clonr profile use work                  # Set 'work' as default profile

# View profile details
clonr profile status                    # Show current profile info
clonr profile status --validate         # Also validate token with GitHub API

# Remove a profile
clonr profile remove old-profile        # Delete a profile
```

**Features:**
- **Default Profile**: When no `--profile` flag is provided, the default profile is used
- **OAuth Device Flow**: Browser-based GitHub login (like `gh auth login`)
- **KeePass Storage**: Tokens stored in encrypted KeePass database (`.kdbx` format)
- **TPM 2.0 Support**: Hardware-backed encryption on Linux - no password required
- **Fallback Options**: System keyring or AES-256-GCM encryption when KeePass unavailable
- **Multiple Profiles**: Switch between work/personal GitHub accounts
- **Auto-detection**: Active profile token used automatically for `gh` commands

### TPM 2.0 & KeePass Storage (Linux)

Clonr uses KeePass database format (`.kdbx`) for secure token storage, with optional TPM 2.0 hardware-backed encryption on Linux. TPM integration is handled automatically when available.

**Token Storage Priority:**
1. **KeePass** (requires TPM) - Encrypted `.kdbx` database with TPM-derived password
2. **System Keyring** - macOS Keychain, Windows Credential Manager, Linux Secret Service
3. **Encrypted File** - AES-256-GCM fallback

**TPM Features (Linux, via [sealbox](https://github.com/inovacc/sealbox)):**
- **Hardware-Bound Keys**: Encryption keys sealed to TPM cannot be extracted
- **Passwordless**: No password required - authentication is automatic via TPM
- **Offline Attack Resistant**: Keys only accessible on the original machine
- **KeePass Integration**: TPM-derived password protects KeePass database
- **PCR Policy Binding**: Optional binding to specific platform configuration registers
- **Versioned Format**: Forward-compatible sealed key format

**Security Note:** TPM-sealed keys cannot be backed up. If you lose access to the TPM (hardware failure, BIOS update), encrypted data will be inaccessible. This is a security feature, not a bug.

**Requirements (for TPM):**
- Linux with TPM 2.0 device (`/dev/tpmrm0`)
- TPM 2.0 enabled in BIOS
- User must be in the `tss` group for TPM access:
  ```sh
  # Add yourself to the tss group
  sudo usermod -aG tss $USER
  # Then log out and back in, or use:
  newgrp tss
  ```

**Without TPM:** KeePass storage is not available. Use system keyring or encrypted file storage instead.

### Workspace Management

Workspaces allow you to organize repositories into logical groups (e.g., work, personal, projects):

```sh
# Create a new workspace
clonr workspace add work --path ~/clonr/work
clonr workspace add personal --path ~/clonr/personal --description "Personal projects"

# List all workspaces
clonr workspace list                    # Shows workspaces with repo counts
clonr workspace list --json             # JSON output

# View workspace details
clonr workspace info work               # Shows repos, profiles, disk usage
clonr workspace info work --json        # JSON output

# Clone a workspace configuration
clonr workspace clone work corp --path ~/clonr/corp

# Edit a workspace
clonr workspace edit work --description "Work projects"
clonr workspace edit work --name company  # Rename workspace

# Move a repository between workspaces
clonr workspace move https://github.com/user/repo personal

# Remove a workspace (must be empty)
clonr workspace remove old-workspace
```

**Features:**
- **Profile Association**: Each profile points to one workspace
- **Repository Counting**: Counts repos by workspace field and by path
- **Disk Usage**: Shows total size of workspace directory
- **JSON Output**: All list commands support `--json` flag

### Data Export/Import

Export and import all clonr data (profiles, workspaces, repositories, config) with password encryption:

```sh
# Export all data encrypted with password (outputs base58 to stdout)
clonr data export > backup.txt
clonr data export --no-tokens > backup.txt  # Exclude authentication tokens

# Import from encrypted backup
clonr data import < backup.txt
clonr data import --file backup.txt
clonr data import --file backup.txt --merge  # Merge with existing data
```

**Security:**
- AES-256-GCM authenticated encryption
- PBKDF2 key derivation (100,000 iterations)
- Base58 encoding for safe copy/paste
- Password minimum 8 characters with confirmation

### GitHub CLI Integration

Clonr includes GitHub CLI-like functionality for managing GitHub resources:

```sh
# Issues
clonr gh issues list                    # List open issues in current repo
clonr gh issues list owner/repo         # List issues in specified repo
clonr gh issues list --state all        # List all issues (open + closed)
clonr gh issues create --title "Bug"    # Create a new issue

# Pull Requests
clonr gh pr status                      # List open PRs in current repo
clonr gh pr status 123                  # Detailed status of PR #123
clonr gh pr status --base main          # Filter by base branch

# Actions (Workflow Runs)
clonr gh actions status                 # List recent workflow runs
clonr gh actions status 123456789       # Detailed status of specific run
clonr gh actions status --branch main   # Filter by branch

# Releases
clonr gh release list                   # List releases
clonr gh release create --tag v1.0.0    # Create a new release
clonr gh release download --tag latest  # Download release assets
```

**Features:**
- **Auto-detection**: Commands auto-detect repository from current directory's git config
- **Profile support**: Use `--profile` flag to select a specific authentication profile
- **Token resolution**: Finds GitHub token from (in order): `--token` flag, `--profile` flag, `GITHUB_TOKEN`, `GH_TOKEN`, active clonr profile, gh CLI config
- **JSON output**: All commands support `--json` flag for scripting
- **Filtering**: Rich filtering options for each command

### Reauthor (Git History Rewriting)

Clonr includes functionality to rewrite git history and change author/committer identity:

```sh
# List all unique author emails in the current repository
clonr reauthor --list

# Replace an email address in git history
clonr reauthor --old-email="old@company.com" --new-email="new@personal.com"

# Also change the author/committer name
clonr reauthor --old-email="old@company.com" --new-email="new@personal.com" --new-name="John Doe"

# Run in a specific repository
clonr reauthor --old-email="old@email.com" --new-email="new@email.com" --repo /path/to/repo

# Skip confirmation prompt
clonr reauthor --old-email="old@email.com" --new-email="new@email.com" --force
```

**Features:**
- **List authors**: View all unique author emails with commit counts
- **Preview**: Shows number of commits that will be affected before rewriting
- **Confirmation**: Requires confirmation before rewriting (unless `--force` is used)
- **Next steps**: Provides guidance for force pushing after rewrite

**Warning**: This operation rewrites git history. After reauthoring:
1. Review changes: `git log --oneline`
2. Force push: `git push --force --all && git push --force --tags`
3. All collaborators will need to re-clone or rebase their work

### Interactive Features

Clonr uses [Bubbletea](https://github.com/charmbracelet/bubbletea) for beautiful terminal UIs:

- **Main Menu**: Run `clonr` without arguments for an interactive menu of all commands
- **Configure**: Interactive form with tab navigation and live validation
- **List**: Filterable, searchable list of repositories with ⭐ for favorites
- **Remove**: Interactive selection of repositories to remove
- **Open**: Select from favorite repositories to open in your editor

All interactive UIs support:

- ↑/↓ or j/k for navigation
- Tab/Shift+Tab for form navigation
- Enter to select/submit
- Esc or Ctrl+C to quit
- Type to search/filter (where applicable)

## Configuration

Clonr stores configuration in a database (BoltDB or SQLite) with an interactive setup interface.

### Interactive Configuration

Run the interactive configuration wizard:

```sh
clonr configure
```

This opens a beautiful terminal UI where you can configure:

- **Default Clone Directory**: Where repositories are cloned by default (default: `~/clonr`)
- **Editor**: Your preferred editor for opening repositories (default: `code`)
- **Terminal**: Terminal application (optional)
- **Monitor Interval**: Seconds between repository status checks (default: 300 seconds)
- **Server Port**: Port for the API server (default: 4000)

### View Current Configuration

```sh
clonr configure --show
```

Output example:

```
Current Configuration:
=====================
Default Clone Directory: ~/clonr
Editor:                  code
Terminal:
Monitor Interval:        300 seconds
Server Port:             4000
```

### Reset Configuration to Defaults

```sh
clonr configure --reset
```

This will reset all configuration values to their defaults:

```
✓ Configuration reset to defaults:
==================================
Default Clone Directory: ~/clonr
Editor:                  code
Terminal:
Monitor Interval:        300 seconds
Server Port:             4000
```

### How Configuration is Used

- **Clone Command**: Uses configured default directory unless overridden
  ```sh
  clonr https://github.com/user/repo  # Clones to configured directory
  clonr https://github.com/user/repo ./custom  # Overrides with ./custom
  ```
- **Open Command**: Opens repositories in the configured editor
- **Server**: Runs on a configured port
- **Monitor**: Uses a configured interval for status checks

## Development

### Requirements

- Go 1.24 or newer
- Git (for cloning operations)
- Protocol Buffers compiler (protoc)

### Building

Clonr provides multiple build methods:

#### Using Make (recommended)

```sh
make build         # Build clonr binary
make proto         # Generate protobuf code
make clean         # Clean generated files
make test          # Run tests
make install       # Install to GOPATH/bin
```

#### Using Build Scripts

**Windows (Batch):**
```batch
build.bat          # Build clonr binary
build.bat clean    # Clean generated files
```

**Windows (PowerShell):**
```powershell
.\build.ps1 -Target all     # Build clonr binary
.\build.ps1 -Target clean   # Clean generated files
```

#### Manual Build

```sh
# Generate protobuf code first
go run scripts/proto/generate.go

# Build with BoltDB (default)
go build -o bin/clonr.exe .

# Build with SQLite instead
go build -tags sqlite -o bin/clonr.exe .
```

### Dependencies

- [Bubbletea](https://github.com/charmbracelet/bubbletea) - Terminal UI framework
- [Bubbles](https://github.com/charmbracelet/bubbles) - TUI components
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Styling for terminal UIs
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [gRPC](https://grpc.io) - RPC framework for client-server communication
- [Protocol Buffers](https://protobuf.dev) - Data serialization
- [BoltDB](https://github.com/etcd-io/bbolt) - Embedded key-value database
- [GORM](https://gorm.io) - ORM for SQLite support
- [sealbox](https://github.com/inovacc/sealbox) - TPM 2.0 hardware-backed encryption (wraps go-tpm)
- [gokeepasslib](https://github.com/tobischo/gokeepasslib) - KeePass database format for secure token storage

## Examples

### Quick Start

```sh
# 1. Install and start the server as a service (recommended)
clonr service --install
clonr service --start

# Or run server directly in a separate terminal
# clonr server start

# 2. Use the client (all commands below)
# Start with interactive menu
clonr

# Configure your preferences first
clonr configure

# Clone a repository
clonr https://github.com/user/awesome-project

# Add an existing local repo
clonr add ~/projects/my-app

# Scan a directory for repos
clonr map ~/code

# List all repos interactively
clonr list

# Mark a repo as favorite
clonr favorite https://github.com/user/awesome-project

# List only favorites
clonr list --favorites

# Remove a repo (interactive)
clonr remove
```

### Workflow Example

```sh
# 1. Install and start the server as a service
clonr service --install
clonr service --start
# Server is now running in the background

# 2. Configure clonr for your environment (optional, sensible defaults provided)
clonr configure
# Defaults: directory ~/clonr, editor code, port 50051

# 2. Clone repositories (they go to ~/clonr by default)
clonr https://github.com/golang/go
clonr https://github.com/torvalds/linux

# 3. Add existing projects
clonr add ~/old-projects/website

# 4. Scan for more repos
clonr map ~/workspace

# 5. Organize with favorites
clonr favorite https://github.com/golang/go
clonr favorite https://github.com/torvalds/linux

# 6. Quick access to favorites
clonr list --favorites
clonr open  # Opens interactive menu of favorites

# 7. Check current config
clonr configure --show

# 8. Reset config if needed
clonr configure --reset
```

## Project Structure

```
clonr/
├── main.go                           # CLI entry point
├── cmd/                              # Commands (Cobra)
│   ├── root.go                       # Root command
│   ├── clone.go, list.go, reauthor.go, etc.  # Client commands
│   ├── server.go                     # Server commands
│   ├── service.go                    # Service management commands
│   ├── profile.go                    # Profile parent command
│   ├── profile_add.go                # Add profile with OAuth
│   ├── profile_list.go               # List profiles
│   ├── profile_use.go                # Set active profile
│   ├── profile_remove.go             # Remove profile
│   ├── profile_status.go             # Show profile info
│   ├── gh.go                         # GitHub CLI parent command
│   ├── gh_issues.go                  # GitHub issues commands
│   ├── gh_pr.go                      # GitHub PR status command
│   ├── gh_actions.go                 # GitHub Actions commands
│   └── gh_release.go                 # GitHub release commands
├── internal/
│   ├── cli/                          # Bubbletea UI components
│   │   └── profile_login.go          # OAuth TUI component
│   ├── core/                         # Core business logic (uses gRPC client)
│   │   ├── auth.go                   # Token resolution with profile support
│   │   ├── profile.go                # Profile management logic
│   │   ├── oauth.go                  # GitHub OAuth device flow
│   │   ├── keyring.go                # Secure keyring storage
│   │   ├── keepass.go                # KeePass database manager for tokens
│   │   ├── encrypt.go                # AES-256-GCM encryption + KeePass password derivation
│   │   ├── tpm.go                    # TPM 2.0 key management (Linux)
│   │   ├── tpm_stub.go               # TPM stub (non-Linux platforms)
│   │   └── tpm_keystore.go           # TPM sealed key storage
│   │   ├── issues.go                 # Issues logic (list, create)
│   │   ├── gh_pr.go                  # PR status logic
│   │   ├── gh_actions.go             # Actions workflow logic
│   │   ├── gh_release.go             # Release management logic
│   │   ├── reauthor.go               # Git history rewriting logic
│   │   └── common.go                 # Shared utilities (DetectRepository)
│   ├── database/                     # Database abstraction (BoltDB/SQLite)
│   ├── model/                        # Data models (Repository, Config)
│   ├── grpcserver/                   # gRPC server implementation
│   │   ├── server.go                 # Server setup
│   │   ├── service.go                # RPC method implementations
│   │   ├── mapper.go                 # Proto ↔ Model conversions
│   │   └── interceptors.go           # Logging, recovery, timeout
│   ├── grpcclient/                   # gRPC client wrapper
│   │   ├── client.go                 # Client methods (mirrors Store interface)
│   │   └── discovery.go              # Server address discovery
│   └── monitor/                      # Repository monitoring
├── api/proto/v1/                     # Protocol Buffer definitions
│   ├── common.proto                  # Common messages
│   ├── repository.proto              # Repository messages
│   ├── config.proto                  # Config messages
│   └── clonr.proto                   # Service definition
├── pkg/api/v1/                       # Generated protobuf code
├── scripts/proto/                    # Proto generation scripts
└── build scripts (Makefile, build.bat, build.ps1)
```

### Architecture Highlights

- **Client-Server Architecture**: Persistent gRPC server manages database, CLI client performs git operations locally
- **gRPC Communication**: Unary RPCs with 30-second timeouts for all operations
- **Server Discovery**: Environment variable → config file → default (localhost:50051)
- **Database Singleton**: Server uses `database.GetDB()` for BoltDB/SQLite access
- **Client Singleton**: Client uses `grpc.GetClient()` to connect to server
- **Bubbletea UIs**: Beautiful, interactive terminal interfaces
- **Cobra CLI**: Modern command-line interface with subcommands
- **Dual Database Support**: Choose BoltDB (default) or SQLite at build time
- **Protocol Buffers**: Type-safe API contracts between client and server

## Roadmap

See the roadmap for planned features and milestones: [ROADMAP.md](ROADMAP.md)

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for details.

## Contributing

Contributions are welcome! Please open issues or submit pull requests.
