# GitHub CLI Integration

Clonr includes built-in GitHub CLI functionality for managing GitHub resources directly from the command line. This provides a subset of `gh` CLI features integrated into clonr.

## Overview

All GitHub commands are accessed through the `clonr gh` subcommand:

```sh
clonr gh <resource> <action> [options]
```

## Common Features

All `gh` commands share these features:

### Repository Detection

Commands automatically detect the repository from:
1. Explicit argument: `clonr gh issues list owner/repo`
2. `--repo` flag: `clonr gh issues list --repo owner/repo`
3. Current directory's git config (parses remote origin URL)

### Authentication

Token resolution priority:
1. `--token` flag (explicit token)
2. `--profile` flag (use token from specified clonr profile)
3. `GITHUB_TOKEN` environment variable
4. `GH_TOKEN` environment variable
5. Active clonr profile token (if set via `clonr profile use`)
6. `gh` CLI configuration file (`~/.config/gh/hosts.yml`)

#### Using Profiles

Clonr supports multiple GitHub authentication profiles for switching between accounts:

```sh
# Create profiles
clonr profile add work       # OAuth flow for work account
clonr profile add personal   # OAuth flow for personal account

# Set active profile (used automatically)
clonr profile use work

# Or specify per-command
clonr gh issues list --profile personal
clonr gh pr status --profile work
```

See `clonr profile --help` for more details.

### Output Formats

- **Text output** (default): Human-readable formatted output
- **JSON output** (`--json`): Machine-readable JSON for scripting

### Common Flags

| Flag | Description |
|------|-------------|
| `--token` | GitHub token (default: auto-detect) |
| `--profile` | Use token from specified clonr profile |
| `--repo` | Repository in owner/repo format |
| `--json` | Output as JSON |

---

## Issues

### List Issues

```sh
clonr gh issues list [owner/repo] [flags]
```

List GitHub issues for a repository.

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--state` | `open` | Filter by state: open, closed, all |
| `--labels` | | Filter by labels (comma-separated) |
| `--assignee` | | Filter by assignee (@me for yourself) |
| `--creator` | | Filter by creator |
| `--sort` | `created` | Sort by: created, updated, comments |
| `--order` | `desc` | Sort order: asc, desc |
| `--limit` | `30` | Maximum number of issues (0 = unlimited) |

**Examples:**

```sh
# List open issues in current repo
clonr gh issues list

# List all issues
clonr gh issues list --state all

# List issues with specific labels
clonr gh issues list --labels bug,urgent

# List issues assigned to you
clonr gh issues list --assignee @me

# List issues in JSON format
clonr gh issues list --json
```

### Create Issue

```sh
clonr gh issues create [owner/repo] [flags]
```

Create a new GitHub issue.

**Flags:**

| Flag | Required | Description |
|------|----------|-------------|
| `--title` | Yes | Issue title |
| `--body` | No | Issue body/description |
| `--labels` | No | Labels to add (comma-separated) |
| `--assignees` | No | Assignees (comma-separated) |

**Examples:**

```sh
# Create a simple issue
clonr gh issues create --title "Bug report"

# Create issue with body and labels
clonr gh issues create --title "Feature request" --body "Description here" --labels enhancement

# Create issue in specific repo
clonr gh issues create owner/repo --title "Issue title"
```

---

## Pull Requests

### PR Status

```sh
clonr gh pr status [pr-number | owner/repo] [flags]
```

Check pull request status. Without a PR number, lists all open PRs. With a PR number, shows detailed status.

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--state` | `open` | Filter by state: open, closed, all |
| `--base` | | Filter by base branch |
| `--head` | | Filter by head branch (user:branch) |
| `--sort` | `created` | Sort by: created, updated, popularity, long-running |
| `--order` | `desc` | Sort order: asc, desc |
| `--limit` | `30` | Maximum number of PRs (0 = unlimited) |

**Examples:**

```sh
# List open PRs
clonr gh pr status

# Check specific PR
clonr gh pr status 123

# List all PRs
clonr gh pr status --state all

# Filter by base branch
clonr gh pr status --base main
```

**Detailed PR Status includes:**
- Review state (approved, changes requested, pending)
- CI check status (success, failure, pending)
- Merge status (mergeable, conflicts)
- Labels, assignees, reviewers
- Changes (+additions, -deletions, files changed)

---

## Actions

### Workflow Status

```sh
clonr gh actions status [run-id | owner/repo] [flags]
```

Check GitHub Actions workflow run status. Without a run ID, lists recent runs. With a run ID, shows detailed status.

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--branch` | | Filter by branch name |
| `--event` | | Filter by event type (push, pull_request, schedule) |
| `--status` | | Filter by status (queued, in_progress, completed) |
| `--actor` | | Filter by actor (username) |
| `--limit` | `20` | Maximum number of runs (0 = unlimited) |
| `--jobs` | `false` | Include job details (for specific run) |

**Examples:**

```sh
# List recent workflow runs
clonr gh actions status

# Check specific run
clonr gh actions status 123456789

# Filter by branch
clonr gh actions status --branch main

# Filter by event
clonr gh actions status --event push

# Include job details
clonr gh actions status 123456789 --jobs
```

**Status Icons:**
- ✅ Success
- ❌ Failure
- 🔄 In progress
- 🕐 Queued/Waiting
- 🚫 Cancelled
- ⏭️ Skipped
- ⏰ Timed out

---

## Releases

### List Releases

```sh
clonr gh release list [owner/repo] [flags]
```

List releases for a repository.

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--limit` | `30` | Maximum number of releases (0 = unlimited) |
| `--include-drafts` | `false` | Include draft releases |

**Examples:**

```sh
# List releases
clonr gh release list

# Include drafts
clonr gh release list --include-drafts

# Limit results
clonr gh release list --limit 10
```

### Create Release

```sh
clonr gh release create [owner/repo] [flags]
```

Create a new GitHub release.

**Flags:**

| Flag | Required | Description |
|------|----------|-------------|
| `--tag` | Yes | Tag name for the release |
| `--name` | No | Release name (default: tag name) |
| `--notes` | No | Release notes |
| `--target` | No | Target commitish (branch/commit, default: main branch) |
| `--draft` | No | Create as draft release |
| `--prerelease` | No | Mark as pre-release |
| `--latest` | No | Mark as latest release |
| `--assets` | No | Files to upload (comma-separated paths) |

**Examples:**

```sh
# Create a simple release
clonr gh release create --tag v1.0.0

# Create with notes
clonr gh release create --tag v1.0.0 --name "Version 1.0.0" --notes "Release notes here"

# Create draft pre-release
clonr gh release create --tag v2.0.0-beta --prerelease --draft

# Create with assets
clonr gh release create --tag v1.0.0 --assets ./dist/app.zip,./dist/app.tar.gz
```

### Download Release

```sh
clonr gh release download [owner/repo] [flags]
```

Download release assets.

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--tag` | `latest` | Tag to download (or "latest") |
| `--patterns` | | Asset name patterns to download (comma-separated) |
| `--dir` | `.` | Destination directory |

**Examples:**

```sh
# Download latest release assets
clonr gh release download

# Download specific tag
clonr gh release download --tag v1.0.0

# Download matching patterns
clonr gh release download --patterns "*.zip,*.tar.gz"

# Download to specific directory
clonr gh release download --dir ./downloads
```

---

## JSON Output

All commands support `--json` flag for machine-readable output:

```sh
# Get issues as JSON
clonr gh issues list --json | jq '.issues[].title'

# Get PR status as JSON
clonr gh pr status 123 --json | jq '.review_state'

# Get workflow runs as JSON
clonr gh actions status --json | jq '.runs[] | {name, status, conclusion}'
```

---

## Error Handling

### Rate Limiting

Commands automatically handle GitHub API rate limits with exponential backoff. If rate limited, the command will wait and retry.

### Authentication Errors

If authentication fails, ensure you have a valid GitHub token configured:

```sh
# Option 1: Create a clonr profile (recommended)
clonr profile add myprofile   # Opens browser for OAuth

# Option 2: Set via environment variable
export GITHUB_TOKEN=ghp_xxxxxxxxxxxx

# Option 3: Use gh CLI to authenticate
gh auth login
```

### Repository Not Found

If the repository cannot be detected:

```sh
# Specify explicitly
clonr gh issues list owner/repo

# Or use --repo flag
clonr gh issues list --repo owner/repo
```

---

## Implementation Details

The GitHub CLI integration uses:

- [go-github](https://github.com/google/go-github) v67 for GitHub API interactions
- [go-gh](https://github.com/cli/go-gh) for token resolution from gh CLI config
- OAuth2 for authentication
- Automatic pagination with configurable limits
- Context timeouts (30 seconds default, 2 minutes for long operations)

### Files

| File | Purpose |
|------|---------|
| `cmd/gh.go` | Parent command with common flags |
| `cmd/gh_issues.go` | Issues list and create commands |
| `cmd/gh_pr.go` | PR status command |
| `cmd/gh_actions.go` | Actions status command |
| `cmd/gh_release.go` | Release list, create, download commands |
| `cmd/profile.go` | Profile parent command |
| `cmd/profile_add.go` | Add profile with OAuth |
| `cmd/profile_list.go` | List all profiles |
| `cmd/profile_use.go` | Set active profile |
| `cmd/profile_remove.go` | Remove a profile |
| `cmd/profile_status.go` | Show profile info |
| `internal/core/auth.go` | Token resolution with profile support |
| `internal/core/profile.go` | Profile management logic |
| `internal/core/oauth.go` | GitHub OAuth device flow |
| `internal/core/keyring.go` | Secure keyring storage |
| `internal/core/encrypt.go` | AES-256-GCM encryption fallback |
| `internal/core/issues.go` | Issues business logic |
| `internal/core/gh_pr.go` | PR status logic |
| `internal/core/gh_actions.go` | Actions workflow logic |
| `internal/core/gh_release.go` | Release management logic |
| `internal/core/common.go` | DetectRepository and parseGitHubURL helpers |
| `internal/cli/profile_login.go` | OAuth TUI component |
