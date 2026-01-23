# Clonr Branding Guidelines

This document defines the branding standards and naming conventions for the Clonr project.

## Project Name

| Usage | Format | Example |
|-------|--------|---------|
| **Official Name** | Clonr | "Clonr is a Git repository manager" |
| **Command Line** | `clonr` (lowercase) | `clonr clone`, `clonr list` |
| **Code References** | clonr (lowercase) | `github.com/inovacc/clonr` |
| **Documentation Title** | Clonr | "Clonr Documentation" |

### Name Origin

**Clonr** = Clone + r (repository)

A tool for cloning and managing Git repositories.

## Terminology

### Core Concepts

| Term | Definition | Usage |
|------|------------|-------|
| **Repository** | A Git repository tracked by Clonr | "Add a repository to Clonr" |
| **Clone** | Download a remote repository | "Clone a repository" |
| **Managed Repository** | A repository registered in Clonr's database | "List all managed repositories" |
| **Favorite** | A starred/bookmarked repository | "Mark as favorite" |

### Commands

| Command | Action Verb | Description |
|---------|-------------|-------------|
| `clone` | Clone | Download and register a repository |
| `list` | List | Display managed repositories |
| `add` | Add | Register an existing local repository |
| `remove` | Remove | Unregister a repository (not delete) |
| `branches` | List/Switch | View and checkout branches |
| `update` | Update/Pull | Fetch latest changes |
| `map` | Map/Scan | Discover repositories in a directory |
| `configure` | Configure | Adjust settings |

### GitHub Integration (`gh` subcommand)

| Command | Description |
|---------|-------------|
| `gh issues` | Manage GitHub issues |
| `gh pr` | View pull request status |
| `gh actions` | Check workflow runs |
| `gh release` | Manage releases |

## Taglines

**Primary:**
> "A Git repository manager"

**Alternative:**
> "Manage your Git repositories efficiently"
> "Clone, organize, and manage Git repos"

## Color Palette (TUI)

| Element | Color Code | Usage |
|---------|------------|-------|
| **Primary/Selected** | `#d75faf` (170) | Selected menu items |
| **Success** | `#00d787` (42) | Success messages, current branch |
| **Error** | `#ff0000` (196) | Error messages |
| **URL/Highlight** | `#5fd7d7` (86) | Repository URLs, local branches |
| **Muted/Secondary** | `#808080` (244) | Paths, remote branches, timestamps |
| **Spinner** | `#ff87af` (205) | Loading indicators |

## Voice & Tone

### Documentation

- **Clear**: Use simple, direct language
- **Concise**: Avoid unnecessary words
- **Technical**: Appropriate for developers
- **Helpful**: Include examples

### CLI Output

- **Brief**: Short, actionable messages
- **Consistent**: Use same terminology throughout
- **Informative**: Show what happened and what to do next

**Examples:**

```
✓ Successfully cloned to /path/to/repo
✗ Clone failed: repository not found
Switched to branch 'feature/new-feature'
```

### Error Messages

Format: `[Context]: [Problem] - [Solution if applicable]`

```
failed to list branches: not a git repository
repository already exists: use --force to re-clone
```

## File Naming

| Type | Convention | Example |
|------|------------|---------|
| Commands | `cmd/<name>.go` | `cmd/branches.go` |
| Core Logic | `internal/core/<name>.go` | `internal/core/branches.go` |
| TUI Components | `internal/cli/<name>.go` | `internal/cli/branches.go` |
| GitHub Features | `gh_<feature>.go` | `gh_release.go` |
| Tests | `<name>_test.go` | `branches_test.go` |

## Version Naming

Format: `vMAJOR.MINOR.PATCH`

- **MAJOR**: Breaking changes
- **MINOR**: New features (backward compatible)
- **PATCH**: Bug fixes

Example: `v0.1.0`, `v1.0.0`, `v1.2.3`

## Repository Structure

```
clonr/
├── cmd/           # CLI commands
├── internal/      # Private packages
│   ├── cli/       # TUI components
│   ├── core/      # Business logic
│   ├── database/  # Data persistence
│   └── model/     # Data structures
├── docs/          # Documentation
├── api/           # API definitions (protobuf)
└── pkg/           # Public packages
```

## Usage Examples in Documentation

Always include practical examples:

```bash
# Clone a repository
clonr clone https://github.com/user/repo

# List all repositories
clonr list

# Show branches interactively
clonr branches

# Output as JSON for scripting
clonr branches --json /path/to/repo
```

## Attribution

When referencing Clonr in external documentation:

> **Clonr** - A Git repository manager
> https://github.com/inovacc/clonr

## Logo (Future)

*Reserved for future logo specifications*

- Format: SVG (primary), PNG (fallback)
- Minimum size: 32x32px
- Clear space: Equal to height of "C" in logo
