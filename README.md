# Clonr [![Test](https://github.com/dyammarcano/clonr/actions/workflows/test.yml/badge.svg)](https://github.com/dyammarcano/clonr/actions/workflows/test.yml)

Clonr is a command-line tool and server for managing Git repositories efficiently. It provides commands to clone, list, update, and remove repositories, as well as a server component for API-based management.

## Features

- **Clone**: Clone Git repositories to a local directory.
- **List**: List all managed repositories.
- **Update**: Pull the latest changes for all or specific repositories.
- **Remove**: Remove repositories from management and disk.
- **Monitor**: Monitor repository status (if implemented).
- **Server**: Run as a server to expose repository management via an API.

## Installation

Install directly using Go (requires Go 1.24+):

```sh
go install github.com/dyammarcano/clonr@latest
```

This will place the `clonr` binary in your `$GOPATH/bin` or `$HOME/go/bin` directory. Make sure this directory is in your `PATH`.

## Usage

### Command Line

Run the tool with various commands:

```sh
clonr [command] [flags]
```

#### Available Commands

- `clone <repo-url>`: Clone a new repository.
- `list`: List all managed repositories.
- `update [repo-name]`: Update all or a specific repository.
- `remove <repo-name>`: Remove a repository.
- `monitor`: Monitor repository status.
- `server`: Start the API server.

Use `clonr [command] --help` for more details on each command.

### Server Mode

Start the server:

```sh
clonr server
```

The server exposes an API for repository management (see API documentation if available).

## Configuration

Configuration options can be set via command-line flags or environment variables (see `params/params.go`).

## Development

- Requires Go 1.18 or newer.
- Task automation is available via [Taskfile](https://taskfile.dev/):
  ```sh
  task build
  task test
  ```

## Project Structure

- `cmd/`: CLI command implementations.
- `internal/db/`: Database logic.
- `internal/git/`: Git operations (clone, list, remove, update).
- `internal/model/`: Data models.
- `internal/params/`: Parameter and configuration handling.
- `internal/server/`: Server logic.

## Roadmap
See the roadmap for planned features and milestones: [ROADMAP.md](ROADMAP.md)

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for details.

## Contributing

Contributions are welcome! Please open issues or submit pull requests.
