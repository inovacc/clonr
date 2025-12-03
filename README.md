# Clonr [![Test](https://github.com/inovacc/clonr/actions/workflows/test.yml/badge.svg)](https://github.com/inovacc/clonr/actions/workflows/test.yml)

Clonr is a command-line tool and server for managing Git repositories efficiently. It provides an interactive interface for cloning, organizing, and working with multiple repositories from the CLI or
via API.

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
- **Server**: Run as a server to expose repository management via an API.

## Installation

Install directly using Go (requires Go 1.24+):

```sh
go install github.com/inovacc/clonr@latest
```

This will place the `clonr` binary in your `$GOPATH/bin` or `$HOME/go/bin` directory. Make sure this directory is in your `PATH`.

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
- `clonr server`: Start the API server for remote management.
- `clonr help`: Display help information.

Use `clonr [command] --help` for more details on each command.

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

### Server Mode

Start the server:

```sh
clonr server
```

The server exposes an API for repository management (see API documentation if available).

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

### Building

```sh
# Build with BoltDB (default)
go build -o clonr .

# Build with SQLite
go build -tags sqlite -o clonr .
```

### Task Automation

Task automation is available via [Taskfile](https://taskfile.dev/):

```sh
task build    # Build the binary
task test     # Run tests
```

### Dependencies

- [Bubbletea](https://github.com/charmbracelet/bubbletea) - Terminal UI framework
- [Bubbles](https://github.com/charmbracelet/bubbles) - TUI components
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Styling for terminal UIs
- [BoltDB](https://github.com/etcd-io/bbolt) - Embedded key-value database
- [GORM](https://gorm.io) - ORM for SQLite support
- [Gin](https://github.com/gin-gonic/gin) - HTTP framework for API server

## Examples

### Quick Start

```sh
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
# 1. Configure clonr for your environment (optional, sensible defaults provided)
clonr configure
# Defaults: directory ~/clonr, editor code, port 4000

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

- `main.go`: CLI entry point with flag parsing and command routing
- `internal/cli/`: Bubbletea UI components (menu, configure, repository list)
- `internal/core/`: Core business logic (clone, add, map, config, etc.)
- `internal/database/`: Database abstraction with Bolt and SQLite implementations
- `internal/model/`: Data models (Repository, Config)
- `internal/server/`: Gin-based API server
- `internal/monitor/`: Repository monitoring functionality

### Architecture Highlights

- **Database Singleton**: Uses `database.GetDB()` for consistent database access
- **Bubbletea UIs**: Beautiful, interactive terminal interfaces
- **Flag-based CLI**: Standard library `flag` package (no external CLI framework)
- **Dual Database Support**: Choose BoltDB (default) or SQLite at build time

## Roadmap

See the roadmap for planned features and milestones: [ROADMAP.md](ROADMAP.md)

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for details.

## Contributing

Contributions are welcome! Please open issues or submit pull requests.
