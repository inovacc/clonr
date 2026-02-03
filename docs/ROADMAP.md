# Clonr Roadmap

This roadmap focuses on the features requested in todo.md and aligns them with the current architecture. It is scoped to the next two minor releases. Dates are tentative and based on effort, with
sequence reflecting dependencies.

## Vision

Clonr simplifies cloning and managing many Git repositories from the CLI or a local API server. The next step is to improve repository onboarding (add), quality-of-life (favorites and
open-after-clone), and private repository support.

## Milestones

### v0.1.1 – GitHub CLI Integration ✅ (Completed)

- GitHub Issues (`clonr gh issues list`, `clonr gh issues create`)
- Pull Request Status (`clonr gh pr status`)
- GitHub Actions (`clonr gh actions status`)
- GitHub Releases (`clonr gh release list`, `clonr gh release create`, `clonr gh release download`)

### v0.2.0 – Onboarding and QoL

- Add Command (Manual registration of existing local repos)
- Map Command (Scan local directories to discover and register existing repos)
- Favorites (mark/star repositories; filter and list favorites)
- Open Command (Quick access to favorited repos in configured editor)
- Configure Command (Set editor, default directory, monitor interval, server port)
- Post-Clone Prompt to open project or folder

### v0.3.0 – Insights and Private Repositories

- Status Command (Show git status across all managed repositories)
- Nerds Command (Display detailed statistics and metrics for repositories)
- Private repository auth support (SSH and HTTPS with token)
- Server API endpoints for favorites, add, map, and configure operations
- Optional secure storage integration for credentials (system keychain when available)
- JSON output for all list commands (`--json` flag): ✅
  - `clonr workspace list --json` ✅
  - `clonr profile list --json` ✅
  - `clonr list --json` (repositories)
  - `clonr org list --json` ✅
  - `clonr workspace info --json` ✅
- Workspace management enhancements: ✅
  - `clonr workspace clone` - Clone workspace with new name
  - `clonr workspace edit` - Edit workspace properties
  - `clonr workspace info` - Show detailed workspace information
- Default profile concept: ✅
  - Renamed "active" to "default" for profiles
  - When no `--profile` flag provided, use default profile
- Server lifecycle commands: ✅
  - `clonr server stop` - Stop running server via signal
  - `clonr server restart` - Restart server with PID monitoring (gops)
  - `clonr server status` - Show server status, PID, uptime
- Data export/import: ✅
  - `clonr data export` - Export encrypted data to base58
  - `clonr data import` - Import from encrypted backup
  - AES-256-GCM + PBKDF2 encryption

## Feature Breakdowns

### 0) GitHub CLI Integration ✅

CLI: `clonr gh <subcommand>`

The GitHub CLI integration provides gh-like functionality for managing GitHub resources directly from clonr.

#### Issues

- `clonr gh issues list [owner/repo]` - List issues for a repository
  - Flags: `--state`, `--labels`, `--assignee`, `--creator`, `--sort`, `--order`, `--limit`, `--json`
- `clonr gh issues create [owner/repo]` - Create a new issue
  - Flags: `--title` (required), `--body`, `--labels`, `--assignees`, `--json`

#### Pull Requests

- `clonr gh pr status [pr-number | owner/repo]` - Check PR status
  - Without PR number: lists all open PRs
  - With PR number: shows detailed status (reviews, CI checks, mergeable state)
  - Flags: `--state`, `--base`, `--head`, `--sort`, `--order`, `--limit`, `--json`

#### Actions

- `clonr gh actions status [run-id | owner/repo]` - Check workflow runs
  - Without run ID: lists recent workflow runs
  - With run ID: shows detailed run status including jobs
  - Flags: `--branch`, `--event`, `--status`, `--actor`, `--limit`, `--jobs`, `--json`

#### Releases

- `clonr gh release list [owner/repo]` - List releases
  - Flags: `--limit`, `--include-drafts`, `--json`
- `clonr gh release create [owner/repo]` - Create a new release
  - Flags: `--tag` (required), `--name`, `--notes`, `--target`, `--draft`, `--prerelease`, `--latest`, `--assets`, `--json`
- `clonr gh release download [owner/repo]` - Download release assets
  - Flags: `--tag`, `--patterns`, `--dir`, `--json`

#### Common Features

- **Auto-detection**: All commands auto-detect repository from current directory's git config
- **Token resolution**: Uses go-gh for automatic token discovery (GITHUB_TOKEN, GH_TOKEN, gh CLI config)
- **JSON output**: All commands support `--json` flag for scripting
- **Pagination**: Automatic pagination with configurable limits
- **Rate limiting**: Handles GitHub API rate limits with automatic backoff

#### Implementation Files

- `cmd/gh.go` - Parent command with common flags
- `cmd/gh_issues.go` - Issues list and create commands
- `cmd/gh_pr.go` - PR status command
- `cmd/gh_actions.go` - Actions status command
- `cmd/gh_release.go` - Release list, create, download commands
- `internal/core/issues.go` - Issues business logic (extended)
- `internal/core/gh_pr.go` - PR status logic
- `internal/core/gh_actions.go` - Actions workflow logic
- `internal/core/gh_release.go` - Release management logic
- `internal/core/common.go` - DetectRepository helper

### 1) Add Command

CLI: `clonr add [path]`

- Description: Register an existing local directory as a managed repository in the database.
- Flow:
  1. Resolve and validate path; verify it’s a Git repo (has .git).
  2. Derive repo name (folder name) and remote URL (if available) via `git config --get remote.origin.url`.
  3. Prompt for confirmation; if no TTY or `--yes`, proceed non-interactively.
  4. Persist into DB if not already present.
- Flags:
  - `--yes, -y` skip confirmation
  - `--name <string>` override inferred name
- Modules to implement:
  - cmd/add.go: wire Cobra command to core logic
  - internal/core: core.AddRepo(path, opts)
  - internal/database: DB method `InsertRepoIfNotExists(repo)`
- Acceptance Criteria:
  - Adding a valid path inserts one record.
  - Adding the same path is idempotent (no duplicate).
  - Non-git directories error out with a clear message.

### 2) Favorites

- Description: Allow users to mark repos as favorites and filter lists.
- CLI:
  - `clonr list --favorites` show only favorites
  - `clonr favorite <name>` mark as favorite
  - `clonr unfavorite <name>` remove favorite flag
- DB Schema:
  - Add nullable/boolean `favorite` column (default false) to repos table
  - Migration approach: if using sqlite, run `ALTER TABLE` guarded by PRAGMA checks on startup
- Modules:
  - internal/database: migration + getters/setters
  - cmd/list.go: add `--favorites` flag
  - new cmds: cmd/favorite.go, cmd/unfavorite.go (or a single `favorite` with subcommands)
- Acceptance Criteria:
  - Favorite flag persists and is reflected in list output
  - Filtering works and is consistent with other flags

### 3) Post-Clone Open Prompt

- Description: After a successful clone, prompt to open the project in an editor or reveal folder.
- CLI:
  - Global flag `--open-after-clone[=editor]` or env `CLONR_OPEN_AFTER_CLONE`
  - Editor detection order: explicit flag > $CLONR_EDITOR > common defaults (code, goland, idea, sublime, vim)
  - Cross-platform folder reveal if no editor: `open` (macOS) / `xdg-open` (Linux) / `start` (Windows)
- Modules:
  - internal/core/clone.go: call helper when clone succeeds
  - internal/core/common.go: helpers for editor/folder opening; platform detection
  - cmd/root.go: global flag wiring
- Acceptance Criteria:
  - When enabled, repo opens in the selected editor after clone
  - Fallback to folder reveal if editor not found

### 4) Map Command

CLI: `clonr map [directory]`

- Description: Scan a local directory tree to discover existing Git repositories and register them.
- Flow:
  1. Accept a directory path (defaults to current directory or configured base path).
  2. Recursively walk the directory tree looking for `.git` folders.
  3. For each discovered repo, extract name and remote URL.
  4. Present list of found repositories with option to select which to add.
  5. Register selected repositories in the database.
- Flags:
  - `--all, -a` automatically add all discovered repos without prompting
  - `--depth <n>` limit recursion depth (default: unlimited)
  - `--exclude <pattern>` skip directories matching pattern (e.g., `node_modules`)
- Modules:
  - cmd/map.go: CLI command
  - internal/core: core.MapDirectory(path, opts) -> []Repository
  - internal/database: reuse InsertRepoIfNotExists
- Acceptance Criteria:
  - Successfully discovers Git repositories in nested directory structures
  - Avoids duplicates with existing managed repos
  - Respects depth limits and exclusion patterns

### 5) Configure Command

CLI: `clonr configure`

- Description: Interactive configuration of application settings.
- Settings:
  - Default editor (code, goland, vim, etc.)
  - Base directory for cloned repositories
  - Monitor interval for repository status checks
  - Server port for API
- Storage:
  - Configuration stored in `~/.config/clonr/config.yaml` or `%APPDATA%\clonr\config.yaml` (Windows)
  - Environment variables override config file
- Modules:
  - cmd/configure.go: interactive prompts
  - internal/config: config file read/write with validation
- Acceptance Criteria:
  - Configuration persists across sessions
  - Invalid values are rejected with helpful messages
  - Environment variables take precedence

### 6) Status Command

CLI: `clonr status [repo-name]`

- Description: Show git status for all or specific managed repositories.
- Flow:
  1. Query database for repositories (all or filtered by name)
  2. For each repo, execute `git status --porcelain` and parse output
  3. Display summary: clean, modified files, untracked files, ahead/behind remote
  4. Optionally show detailed status with `--verbose`
- Flags:
  - `--verbose, -v` show detailed git status output
  - `--dirty-only` show only repositories with uncommitted changes
- Modules:
  - cmd/status.go: CLI command
  - internal/core: core.GetRepoStatus(repoPath) -> StatusInfo
- Acceptance Criteria:
  - Accurately reflects git status
  - Handles repositories in detached HEAD or merge states
  - Clear visual indicators for clean vs dirty repos

### 7) Nerds Command

CLI: `clonr nerds [repo-name]`

- Description: Display detailed statistics and metrics for repositories.
- Metrics:
  - Total commits, contributors, branches, tags
  - Repository size (disk usage)
  - Last commit date and author
  - Language breakdown (if available)
  - Lines of code statistics
- Flags:
  - `--format <json|table>` output format
  - `--sort <commits|size|date>` sort repositories by metric
- Modules:
  - cmd/nerds.go: CLI command
  - internal/core: core.GetRepoStats(repoPath) -> RepoStats
  - Use git commands: `git rev-list --count HEAD`, `git shortlog -sn`, `git count-objects -vH`
- Acceptance Criteria:
  - Statistics are accurate and useful
  - Performance is acceptable even with many repos
  - Gracefully handles repos with no commits or missing data

### 8) Private Repository Support

- Description: Support cloning and updating private repos using SSH or HTTPS tokens.
- CLI / Config:
  - Global flags/env:
    - `--auth-method [ssh|https]` (default auto)
    - `--token-env VAR_NAME` for HTTPS tokens
  - Prefer SSH if `~/.ssh` keys are configured; otherwise HTTPS with token
- Implementation:
  - For clone/update, rely on Git and environment configuration; detect common auth errors and surface helpful guidance
  - Optional: store token reference (env var name only) in DB, not the token itself
  - Future: integrate with OS keychain via a thin wrapper
- Server:
  - Expose endpoints to register HTTPS token aliases (names only) and to create repos with method
- Acceptance Criteria:
  - Private repos can be cloned and updated using configured method
  - No secrets are persisted in plaintext

## API/Server Additions (v0.3.0)

- POST /repos/add { path, name? }
- POST /repos/map { directory, depth?, exclude? }
- POST /repos/{name}/favorite { favorite: true|false }
- GET /repos?favorites=true
- GET /repos/{name}/status
- GET /repos/status (bulk status)
- GET /repos/{name}/stats (nerds data)
- PUT /config { editor?, baseDir?, monitorInterval?, serverPort? }
- GET /config
- Security: local-only by default; add `--listen` to bind non-local; optional basic auth for non-local

## Database

- Current: internal/database/database.go (inspect for existing schema: name, path, remote, etc.)
- Add column `favorite` BOOLEAN DEFAULT 0
- Migrations:
  - On DB init, ensure table exists and add missing columns using PRAGMA table_info

## Telemetry & Metrics

- Track counts for: add, favorite, clone success/failure, open-after-clone
- Implement counters in internal/metrics and log them; optional Prometheus endpoint in server

## Testing Strategy

- Unit tests:
  - Path resolution and git-dir checks for Add and Map
  - Directory traversal and Git repo discovery for Map
  - Favorites flag behavior and DB accessors
  - Editor selection and OS open commands (with exec command abstraction and fakes)
  - Configuration file parsing and validation
  - Git status parsing and error handling
  - Statistics calculation for Nerds command
- Integration tests:
  - SQLite and Bolt DB migrations
  - Clone+open flow with a temp repo (use testdata)
  - Map command with mock directory structures
  - Status command with test repositories in various states
- E2E (optional):
  - run `clonr` commands in a temp workspace using Taskfile
  - Full workflow: map -> favorite -> open -> status -> nerds

## Risks & Mitigations

- Git and editor binaries may not exist → add robust detection and helpful messages
- DB migration failures → idempotent ALTERs with backup/restore guidance
- Cross-platform open behavior → isolate in helpers and test per OS with build tags if needed

## Deliverables

- v0.2.0: Add, Map, Favorites, Open, Configure commands (CLI only), Post-clone open prompt
- v0.3.0: Status, Nerds, Private repos (SSH/HTTPS), full server API endpoints, secure credential handling

## Acceptance

- All features documented in README with examples
- Minimal surprise: defaults don't break current flows; flags are opt-in

---

## Future Roadmap

### v0.4.0 – Git History & Advanced Analytics

#### Auto-Update Feature ✅ (Completed)
- [x] `clonr update` - Check for and install updates
- [ ] Automatic update check on startup (configurable)
- [x] GitHub Releases integration via [autoupdater](https://github.com/inovacc/autoupdater)
- [x] GoReleaser detection and parsing
- [x] Semantic version comparison
- [x] Platform-specific asset selection (OS/arch)
- [x] SHA256 checksum verification
- [x] Archive extraction (.tar.gz, .tgz, .zip)
- [x] Download progress display
- [x] Graceful shutdown with state preservation
- [x] `--check` flag to only check without installing
- [x] `--force` flag to reinstall current version
- [x] `--pre` flag to include pre-release versions

#### Reauthor Command ✅ (Completed)
- `clonr reauthor --list` - List all unique author emails
- `clonr reauthor --old-email --new-email` - Rewrite git history
- `clonr reauthor --new-name` - Also change author name
- Integration with git-nerds library

#### Branch Management
- [ ] `clonr branches` - List and manage branches across repositories
- [ ] Branch cleanup suggestions (stale branches)
- [ ] Default branch detection and switching
- [ ] Branch comparison across repos

#### Enhanced Statistics
- [ ] Commit activity heatmaps
- [ ] Contributor leaderboards
- [ ] Code churn analysis
- [ ] File hotspot detection

#### Security: Gitleaks Integration
- [ ] `clonr pull --check-leaks` - Scan for secrets after pulling changes
- [ ] `clonr push --check-leaks` - Scan for secrets before pushing commits
- [ ] Pre-commit hook integration for leak detection
- [ ] Configurable secret patterns and rules
- [ ] `.gitleaks.toml` support for custom configurations
- [ ] Interactive prompt to abort push if leaks detected
- [ ] `clonr security scan [repo]` - Manual scan for secrets in repository
- [ ] Report generation for found secrets (JSON/table output)
- [ ] Integration with [gitleaks](https://github.com/gitleaks/gitleaks) library

### v0.5.0 – Organization & Team Features

#### GitHub Organization Support ✅ (In Progress)
- `clonr org list` - List organization repositories
- `clonr org mirror` - Mirror entire organizations
- Batch operations on org repos

#### Contributors Command
- [ ] `clonr gh contributors` - Analyze repository contributors
- [ ] Contribution statistics and rankings
- [ ] New contributor detection

#### Team Workflows
- [ ] Repository grouping/tagging
- [ ] Bulk operations on tagged repos
- [ ] Team activity dashboards

#### Project Management Integrations ✅ (Completed)
- `clonr pm jira issues list` - List Jira issues
- `clonr pm jira issues create` - Create Jira issues
- `clonr pm jira issues view` - View issue details
- `clonr pm jira issues transition` - Move issues through workflow
- `clonr pm jira sprints list` - List sprints
- `clonr pm jira sprints current` - Show current sprint progress
- `clonr pm jira boards list` - List Jira boards
- `clonr pm zenhub board` - View ZenHub board state
- `clonr pm zenhub epics` - List ZenHub epics
- `clonr pm zenhub issue` - View ZenHub issue details
- Multi-source authentication (flags, env vars, config files)
- JSON output support for all commands

### v0.6.0 – Instance Synchronization & Backup

This version introduces **standalone mode** for secure synchronization between clonr instances across machines.

#### Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        CLONR INSTANCE SYNC FLOW                             │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  SOURCE INSTANCE                         DESTINATION INSTANCE               │
│  ─────────────────                       ─────────────────────              │
│                                                                             │
│  ┌─────────────────┐                     ┌─────────────────────┐            │
│  │ clonr standalone│                     │ clonr standalone    │            │
│  │ init            │──── Standalone ────▶│ connect <key>       │            │
│  └────────┬────────┘     Key (JSON)      └──────────┬──────────┘            │
│           │                                         │                       │
│           ▼                                         ▼                       │
│  ┌─────────────────┐                     ┌─────────────────────┐            │
│  │ Generates:      │                     │ Prompts for:        │            │
│  │ • API Key       │                     │ • Decryption Key    │            │
│  │ • Refresh Token │                     │ (local password)    │            │
│  │ • Host/IP Info  │                     └──────────┬──────────┘            │
│  └────────┬────────┘                                │                       │
│           │                                         │                       │
│           ▼                                         ▼                       │
│  ┌─────────────────┐                     ┌─────────────────────┐            │
│  │ Exposes gRPC    │◀═══ Encrypted ═════▶│ Syncs data:         │            │
│  │ Sync Endpoint   │     Channel         │ • Profiles          │            │
│  │ (port 50052)    │                     │ • Workspaces        │            │
│  └─────────────────┘                     │ • Repos (metadata)  │            │
│                                          │ • Config            │            │
│                                          └─────────────────────┘            │
│                                                                             │
│  Data encrypted with standalone key, decrypted locally with user password  │
└─────────────────────────────────────────────────────────────────────────────┘
```

#### Standalone Mode Commands

##### Source Instance (Server)

```bash
# Initialize standalone mode - generates sync key
clonr standalone init
# Output: JSON with { ip, port, api_key, refresh_token, host?, expires_at }

# Show current standalone status
clonr standalone status

# Regenerate API key (invalidates old connections)
clonr standalone rotate

# List connected clients
clonr standalone clients

# Revoke a specific client
clonr standalone revoke <client_id>

# Disable standalone mode
clonr standalone disable
```

##### Destination Instance (Client)

```bash
# Connect to a standalone instance
clonr standalone connect <standalone_key_json>
# Prompts for: decryption password (stored locally, never transmitted)

# Or connect with key file
clonr standalone connect --file standalone-key.json

# List standalone connections
clonr standalone list

# Check connection status
clonr standalone status <connection_name>

# Sync data from standalone instance
clonr standalone sync <connection_name>
# Options: --profiles, --workspaces, --repos, --config, --all

# Disconnect from standalone instance
clonr standalone disconnect <connection_name>
```

##### Standalone Profiles Management

```bash
# List profiles from standalone instances
clonr profile standalone list
clonr profile standalone list --connection <name>

# Show standalone profile details
clonr profile standalone status <profile_name>

# Import a standalone profile locally (decrypts with local password)
clonr profile standalone import <profile_name>

# Delete synced standalone profile
clonr profile standalone delete <profile_name>

# Update decryption password for standalone data
clonr profile standalone update-password
```

#### Data Structures

##### Standalone Key (Generated by Source)

```json
{
  "version": 1,
  "instance_id": "uuid-of-source-instance",
  "host": "192.168.1.100",
  "port": 50052,
  "api_key": "base58-encoded-api-key",
  "refresh_token": "base58-encoded-refresh-token",
  "encryption_key_hint": "first-4-chars-of-key-hash",
  "expires_at": "2024-12-31T23:59:59Z",
  "created_at": "2024-01-01T00:00:00Z",
  "capabilities": ["profiles", "workspaces", "repos", "config"]
}
```

##### Standalone Connection (Stored at Destination)

```json
{
  "name": "home-server",
  "instance_id": "uuid-of-source-instance",
  "host": "192.168.1.100",
  "port": 50052,
  "api_key_encrypted": "locally-encrypted-api-key",
  "refresh_token_encrypted": "locally-encrypted-refresh-token",
  "local_password_hash": "argon2-hash-for-verification",
  "last_sync": "2024-06-15T10:30:00Z",
  "sync_status": "connected",
  "synced_items": {
    "profiles": 3,
    "workspaces": 2,
    "repos": 45,
    "config": true
  }
}
```

##### Synced Profile (Encrypted at Rest)

```json
{
  "source_instance": "uuid-of-source-instance",
  "source_profile_name": "github-work",
  "encrypted_data": "base64-encrypted-profile-blob",
  "encryption_method": "AES-256-GCM",
  "synced_at": "2024-06-15T10:30:00Z",
  "decrypted": false
}
```

#### Security Architecture

##### Encryption Layers

1. **Transport Layer**: gRPC with TLS (mTLS optional)
2. **API Authentication**: API key + refresh token (JWT-like rotation)
3. **Data Encryption**: AES-256-GCM with key derived from standalone key
4. **Local Decryption**: User password + Argon2 key derivation

##### Key Derivation

```
Source Instance:
  standalone_key = random(32 bytes)
  api_key = HKDF(standalone_key, "api-auth", instance_id)
  encryption_key = HKDF(standalone_key, "data-encryption", instance_id)

Destination Instance:
  local_key = Argon2id(user_password, salt, params)
  storage_key = HKDF(local_key, "local-storage", connection_id)

Data Flow:
  source_data → encrypt(encryption_key) → transmit → store_encrypted
  retrieve → decrypt(local_key + encryption_key) → use
```

##### Security Properties

- **Zero-knowledge source**: Source doesn't know destination's local password
- **Forward secrecy**: Key rotation doesn't expose historical data
- **Offline capable**: Synced data usable without network (if decrypted)
- **Revocable access**: Source can revoke any client instantly
- **Audit trail**: All sync operations logged on both sides

#### Implementation Files

```
internal/
├── standalone/
│   ├── server.go          # Standalone mode server (source instance)
│   ├── client.go          # Standalone mode client (destination)
│   ├── key.go             # Key generation and management
│   ├── crypto.go          # Encryption/decryption utilities
│   ├── sync.go            # Data synchronization logic
│   ├── connection.go      # Connection management
│   └── types.go           # Data structures
├── server/grpc/
│   ├── standalone_service.go  # gRPC service for sync
│   └── standalone.proto       # Protocol definitions
cmd/
├── standalone.go          # Parent command
├── standalone_init.go     # Initialize standalone mode
├── standalone_connect.go  # Connect to standalone instance
├── standalone_sync.go     # Sync data
├── standalone_list.go     # List connections
└── profile_standalone.go  # Standalone profile management
api/proto/v1/
└── standalone.proto       # Sync protocol definitions
```

#### Protocol Buffer Definitions

```protobuf
// api/proto/v1/standalone.proto

service StandaloneService {
  // Authentication
  rpc Authenticate(AuthRequest) returns (AuthResponse);
  rpc RefreshToken(RefreshRequest) returns (RefreshResponse);

  // Sync operations
  rpc SyncProfiles(SyncRequest) returns (stream EncryptedProfile);
  rpc SyncWorkspaces(SyncRequest) returns (stream EncryptedWorkspace);
  rpc SyncRepos(SyncRequest) returns (stream EncryptedRepo);
  rpc SyncConfig(SyncRequest) returns (EncryptedConfig);

  // Full sync
  rpc FullSync(SyncRequest) returns (stream SyncChunk);

  // Status
  rpc GetSyncStatus(StatusRequest) returns (SyncStatus);
  rpc Ping(Empty) returns (PingResponse);
}

message AuthRequest {
  string api_key = 1;
  string client_id = 2;
  string client_name = 3;
}

message SyncRequest {
  string auth_token = 1;
  int64 since_timestamp = 2;  // Incremental sync
  repeated string item_types = 3;  // ["profiles", "workspaces", ...]
}

message EncryptedProfile {
  string id = 1;
  bytes encrypted_data = 2;
  bytes nonce = 3;
  int64 updated_at = 4;
}

message SyncChunk {
  string type = 1;  // "profile", "workspace", "repo", "config"
  bytes encrypted_data = 2;
  bytes nonce = 3;
  int32 sequence = 4;
  int32 total = 5;
}
```

#### Phase 1: Core Infrastructure (v0.6.0-alpha) ✅ Completed

- [x] `internal/standalone/types.go` - Data structures for keys, connections, sync
- [x] `internal/standalone/key.go` - Key generation and serialization
- [x] `internal/standalone/crypto.go` - AES-256-GCM encryption with PBKDF2/Argon2
- [x] `internal/standalone/archive.go` - Encrypted repository archiving (zip + AES-256-GCM)
- [x] `internal/standalone/sync.go` - Sync logic with encrypted storage states
- [x] `internal/standalone/handshake.go` - Client-server handshake with per-client encryption
- [x] `proto/v1/standalone.proto` - Protocol definitions for StandaloneService
- [x] `cmd/standalone.go` - Parent command with help text
- [x] `cmd/standalone_init.go` - Generate standalone key (`clonr standalone init`)
- [x] `cmd/standalone_status.go` - Show standalone status (`clonr standalone status`)
- [x] `cmd/standalone_archive.go` - Create encrypted repo archive (`clonr standalone archive`)
- [x] `cmd/standalone_extract.go` - Extract encrypted archive (`clonr standalone extract`)
- [x] `cmd/standalone_encrypt.go` - Setup server encryption (`clonr standalone encrypt setup`)
- [x] `cmd/standalone_decrypt.go` - Decrypt synced data (`clonr standalone decrypt`)
- [x] `internal/store/bolt.go` - BoltDB storage for standalone config and connections
- [x] `internal/standalone/*_test.go` - Unit tests for crypto, key generation, archiving, and handshake
- [ ] Basic gRPC sync endpoint (StandaloneService implementation)

#### Phase 2: Connection Management (v0.6.0-beta) 🚧 In Progress

- [x] `clonr standalone connect` - Connect to standalone instance with key display
- [x] `clonr standalone accept` - Accept pending client connections (server-side)
- [x] `clonr standalone clients` - List registered clients (server-side)
- [x] `internal/standalone/handshake.go` - Per-client encryption key management
- [x] Local password encryption for stored credentials
- [x] Client registration persistence in BoltDB
- [ ] `clonr standalone list` - List connections (client-side view)
- [ ] `clonr standalone disconnect` - Remove connection
- [ ] Connection health checking

##### Client-Server Handshake Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                     CLIENT-SERVER HANDSHAKE PROTOCOL                        │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  CLIENT                                SERVER                               │
│  ──────                               ──────                                │
│                                                                             │
│  clonr standalone connect <key>       clonr standalone init                 │
│       │                                    │                                │
│       │  1. Parse & validate key           │                                │
│       │  2. Generate encryption key        │                                │
│       │  3. Display key to user            │                                │
│       │                                    │                                │
│       ▼                                    ▼                                │
│  ┌─────────────────────────────┐    ┌─────────────────────────────┐         │
│  │   ENCRYPTION KEY            │    │   clonr standalone accept   │         │
│  │   xxxx-xxxx-xxxx-xxxx       │───▶│   (enter displayed key)     │         │
│  │   xxxx-xxxx-xxxx-xxxx       │    │                             │         │
│  └─────────────────────────────┘    └──────────────┬──────────────┘         │
│                                                    │                        │
│       │                                            ▼                        │
│       │                               ┌─────────────────────────────┐       │
│       │                               │   RegisteredClient stored   │       │
│       │                               │   - Key hash (Argon2)       │       │
│       │                               │   - Key hint                │       │
│       │                               │   - Machine info            │       │
│       ▼                               └─────────────────────────────┘       │
│  ┌─────────────────────────────┐                                            │
│  │   Connection stored locally │                                            │
│  │   - Encrypted with local pw │                                            │
│  │   - Client key preserved    │                                            │
│  └─────────────────────────────┘                                            │
│                                                                             │
│  Data Classification:                                                       │
│  • Sensitive (tokens, credentials) → Encrypted with per-client key          │
│  • Public (repos, workspaces) → Stored normally                             │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

#### Phase 3: Data Synchronization (v0.6.0-rc)

- [ ] `clonr standalone sync` - Sync profiles, workspaces, config
- [ ] Incremental sync (only changed items)
- [ ] Conflict detection and resolution
- [ ] `clonr profile standalone list/import/delete`

#### Phase 4: Production Ready (v0.6.0)

- [ ] Key rotation (`clonr standalone rotate`)
- [ ] Client revocation (`clonr standalone revoke`)
- [ ] Audit logging
- [ ] Rate limiting
- [ ] Documentation and examples

#### Repository Archive Feature ✅ (Completed)

Encrypted repository archiving allows secure backup and transfer of repositories.

**Commands:**
```bash
# Create encrypted archive from specific paths
clonr standalone archive /path/to/repo1 /path/to/repo2 -o backup.clonr

# Archive all managed repositories
clonr standalone archive --all -o all-repos.clonr

# Archive only favorites
clonr standalone archive --favorites -o favorites.clonr

# Archive by workspace
clonr standalone archive --workspace work -o work-repos.clonr

# Archive without .git (smaller, no history)
clonr standalone archive /path/to/repo --no-git -o backup.clonr

# Extract archive
clonr standalone extract backup.clonr -o /path/to/output

# List archive contents without extracting
clonr standalone extract backup.clonr --list
```

**Features:**
- AES-256-GCM encryption with PBKDF2 key derivation
- DEFLATE compression (configurable level 0-9)
- Manifest with repository metadata (URL, last commit, file count)
- Exclusion patterns (node_modules, vendor, __pycache__, etc.)
- Optional .git directory exclusion
- Integrity protection via GCM authentication tag

**Archive Format (.clonr):**
- Magic header: `CLONR-REPO`
- Version byte
- Encrypted payload (salt + nonce + ciphertext)
- Contains: ZIP archive with manifest.json + repository files

#### Future: Repository Sync (v0.6.1+)

- [ ] Sync repository metadata via gRPC
- [ ] Sync encrypted archives between instances
- [ ] Bandwidth-aware sync scheduling
- [ ] Selective sync filters

#### Backup & Restore (Parallel Track)

- [x] `clonr standalone archive` - Create encrypted repository archives
- [x] `clonr standalone extract` - Extract from encrypted archive
- [ ] Incremental backups (delta archives)
- [ ] Cloud storage integration (S3, GCS)

### v0.7.0 – Cross-Platform TPM & Security Hardening

#### Sealbox Integration ✅ (Completed)

TPM functionality has been extracted to the external `github.com/inovacc/sealbox` package, providing a reusable TPM key management library.

**Completed:**
- [x] Integrated sealbox library via `go mod replace`
- [x] Migrated from custom `internal/core/tpm.go` to sealbox wrappers
- [x] Platform-agnostic abstractions (`KeyManager`, `KeyStore`, `SealedData`)
- [x] Linux TPM 2.0 support via `/dev/tpmrm0`
- [x] Sealed key storage with `WithAppConfig("clonr", ".clonr_sealed_key")`
- [x] KeePass integration with TPM-derived password

**Sealbox Features:**
- PCR policy binding for hardware attestation
- Optional password protection layer
- Versioned SealedData format for forward compatibility
- Key hierarchy support (primary/derived keys)
- Automatic retry logic for transient TPM errors

**Pending Windows Support:**
- [ ] Complete Windows TBS implementation in sealbox
- [ ] Test on Windows 10/11 with TPM 2.0
- [ ] Update clonr `internal/core/tpm_windows.go` wrapper

**Pending macOS Support (Stretch Goal):**
- [ ] Research Secure Enclave integration in sealbox
- [ ] Evaluate keychain with biometric protection alternative

#### Sealbox API (Current)

```go
// High-level API
sealbox.Initialize(opts...)           // Create and seal new key
sealbox.GetSealedMasterKey(opts...)   // Retrieve unsealed key
sealbox.HasKey(opts...)               // Check if key exists
sealbox.Reset(opts...)                // Remove sealed key

// Options
sealbox.WithAppConfig(app, filename)  // App-specific config dir
sealbox.WithStorePath(path)           // Custom storage path

// Low-level API
km, _ := sealbox.NewKeyManager()      // Direct TPM access
km.SealKey(key)                       // Seal arbitrary data
km.UnsealKey(sealed)                  // Unseal data
km.GenerateAndSealKey()               // Generate random key

store, _ := sealbox.NewKeyStore(opts...)
store.Save(sealed)                    // Persist sealed data
store.Load()                          // Load sealed data
store.Exists()                        // Check existence
```

#### Clonr TPM Wrapper Files

```
internal/core/
├── tpm.go           # Linux: wraps sealbox.NewKeyManager()
├── tpm_stub.go      # Non-Linux: returns ErrTPMNotSupported
└── tpm_keystore.go  # All platforms: wraps sealbox high-level API
```

#### Testing Strategy

- [ ] Unit tests with TPM simulator (swtpm)
- [ ] Integration tests on real hardware (Linux CI)
- [ ] Windows CI pipeline with TPM simulator
- [ ] Cross-platform build verification
- [ ] Benchmark sealing/unsealing operations

#### Documentation

- [x] Updated CLAUDE.md with sealbox integration details
- [ ] Usage examples for each platform
- [ ] Security considerations and threat model
- [ ] Migration guide for existing users

---

### v0.8.0 – Multi-Database Support

Add support for multiple database backends with runtime selection.

#### Overview

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      DATABASE BACKEND SELECTION                          │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  clonr init                          clonr init --db sqlite              │
│  (default: BoltDB)                   (explicit: SQLite)                  │
│       │                                    │                             │
│       ▼                                    ▼                             │
│  ┌──────────┐                       ┌──────────────┐                     │
│  │  BoltDB  │                       │    SQLite    │                     │
│  │ (embed)  │                       │   (file)     │                     │
│  └──────────┘                       └──────────────┘                     │
│                                                                          │
│  clonr init --db postgres --db-url "postgres://..."                     │
│       │                                                                  │
│       ▼                                                                  │
│  ┌──────────────┐                                                        │
│  │  PostgreSQL  │  ← For multi-user / server deployments                │
│  │   (remote)   │                                                        │
│  └──────────────┘                                                        │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

#### Commands

```bash
# Initialize with default BoltDB (embedded, single-user)
clonr init

# Initialize with SQLite (file-based, portable)
clonr init --db sqlite
clonr init --db sqlite --db-path /path/to/clonr.db

# Initialize with PostgreSQL (multi-user, server)
clonr init --db postgres --db-url "postgres://user:pass@host:5432/clonr"

# Check current database backend
clonr config db

# Migrate data between backends
clonr data migrate --from bolt --to sqlite
clonr data migrate --from sqlite --to postgres --db-url "..."
```

#### Database Comparison

| Feature | BoltDB | SQLite | PostgreSQL |
|---------|--------|--------|------------|
| Default | ✅ Yes | No | No |
| Single file | ✅ | ✅ | ❌ |
| Embedded | ✅ | ✅ | ❌ |
| Multi-user | ❌ | ⚠️ Limited | ✅ |
| Remote access | ❌ | ❌ | ✅ |
| Transactions | ✅ | ✅ | ✅ |
| Full-text search | ❌ | ✅ | ✅ |
| Backup | File copy | File copy | pg_dump |
| Use case | CLI tool | Portable | Server/Team |

#### Implementation Files

```
internal/store/
├── store.go           # Store interface (existing)
├── bolt.go            # BoltDB implementation (existing, default)
├── sqlite.go          # SQLite implementation (new)
├── postgres.go        # PostgreSQL implementation (new)
├── factory.go         # Database factory with backend selection
├── migrate.go         # Cross-backend migration utilities
└── config.go          # Database configuration management
```

#### Configuration

Database selection stored in `~/.config/clonr/db.json`:

```json
{
  "backend": "bolt",
  "bolt": {
    "path": "~/.local/share/clonr/clonr.bolt"
  },
  "sqlite": {
    "path": "~/.local/share/clonr/clonr.db",
    "journal_mode": "WAL"
  },
  "postgres": {
    "url": "postgres://localhost:5432/clonr",
    "max_connections": 10,
    "ssl_mode": "prefer"
  }
}
```

#### Phase 1: SQLite Support

- [ ] `internal/store/sqlite.go` - SQLite Store implementation
- [ ] Schema migration system (golang-migrate or custom)
- [ ] `clonr init --db sqlite` flag
- [ ] Database configuration file
- [ ] Unit tests with in-memory SQLite

#### Phase 2: PostgreSQL Support

- [ ] `internal/store/postgres.go` - PostgreSQL Store implementation
- [ ] Connection pooling (pgxpool)
- [ ] `clonr init --db postgres` flag
- [ ] SSL/TLS connection support
- [ ] Environment variable support for credentials

#### Phase 3: Migration & Management

- [ ] `internal/store/migrate.go` - Cross-backend migration
- [ ] `clonr data migrate` command
- [ ] Backup/restore per backend
- [ ] `clonr config db` status command

#### Build Tags

```go
// internal/store/bolt.go
//go:build !sqlite && !postgres

// internal/store/sqlite.go
//go:build sqlite || all

// internal/store/postgres.go
//go:build postgres || all
```

Default build includes only BoltDB. Use build tags to include other backends:

```bash
# Build with all backends
go build -tags "sqlite,postgres" ./...

# Build with only SQLite
go build -tags sqlite ./...
```

#### Dependencies

| Backend | Package | Notes |
|---------|---------|-------|
| BoltDB | `go.etcd.io/bbolt` | Already included |
| SQLite | `modernc.org/sqlite` | Pure Go, no CGO |
| PostgreSQL | `github.com/jackc/pgx/v5` | Modern PostgreSQL driver |

### v0.9.0 – Git/GitHub Subcommand Reorganization ✅ (Completed)

Consolidated git operations under `clonr gh git` subcommand with enhanced error handling and gh CLI-inspired patterns.

#### Design Decision

After analyzing the GitHub CLI (`github.com/cli/cli`), we discovered that gh uses `exec.Command` to invoke the system git binary, NOT go-git. This approach was adopted for clonr because:

- **Compatibility**: System git handles edge cases, hooks, and configurations
- **Credential helper pattern**: Secure token injection without exposing secrets in process args
- **Maintenance**: Leverages battle-tested git implementation

#### Git Commands (exec.Command with Credential Helper)

| Command | Description | Status |
|---------|-------------|--------|
| `clonr gh git clone <url>` | Clone with profile/workspace selection | ✅ |
| `clonr gh git status` | Show working tree status (short/porcelain) | ✅ |
| `clonr gh git commit -m "msg"` | Create commit with -a flag support | ✅ |
| `clonr gh git push` | Push with pre-push security scan | ✅ |
| `clonr gh git pull` | Pull with profile authentication | ✅ |
| `clonr gh git log` | Show commit log with filtering | ✅ |
| `clonr gh git diff` | Show changes (staged/stat/name-only) | ✅ |
| `clonr gh git branch` | List/create/delete branches | ✅ |

#### GitHub API Integration (Existing)

| Command | Description | Status |
|---------|-------------|--------|
| `clonr gh issues` | List/manage issues | ✅ exists |
| `clonr gh prs` | List/manage pull requests | ✅ exists |
| `clonr gh actions` | View workflow runs | ✅ exists |
| `clonr gh releases` | Manage releases | ✅ exists |
| `clonr gh repo create` | Create new repository | Planned |
| `clonr gh repo fork` | Fork repository | Planned |

#### Examples

```bash
# Git operations under gh git subcommand
clonr gh git status
clonr gh git status --porcelain
clonr gh git log --limit 10
clonr gh git log --oneline --author "John"
clonr gh git log --json
clonr gh git diff --staged
clonr gh git diff --stat HEAD~1
clonr gh git branch
clonr gh git branch --json
clonr gh git branch -d old-branch
clonr gh git commit -a -m "feat: add feature"
clonr gh git push -u origin main
clonr gh git pull origin main
clonr gh git clone owner/repo --profile work

# Existing top-level commands still work (backward compatible)
clonr clone owner/repo
clonr commit -m "message"
clonr push
clonr pull
```

#### Git Client Enhancements

**CommandModifier Pattern** (inspired by gh CLI):
```go
gc := client.NewGitCommand(ctx, "log", "--oneline")
gc.WithEnv("GIT_TERMINAL_PROMPT=0").WithDir("/path/to/repo")
output, err := gc.Output()
```

**Error Helper Functions**:
- `IsNotRepository(err)` - Check if not a git repo
- `IsAuthRequired(err)` - Authentication needed
- `IsNoUpstream(err)` - No upstream configured
- `IsConflict(err)` - Merge conflict detected
- `IsNothingToCommit(err)` - Working tree clean
- `GetExitCode(err)` - Extract exit code from error

**New Git Operations**:
- `Log(ctx, LogOptions)` - Structured commit log
- `LogOneline(ctx, limit)` - One-line log output
- `Diff(ctx, DiffOptions)` - Diff with options
- `ListBranchesDetailed(ctx, all)` - Branch info with upstream
- `DeleteBranch(ctx, name, force)` - Delete branches
- `ListRemotes(ctx)` - List configured remotes
- `StatusPorcelain(ctx)` - Machine-readable status
- `GetHead(ctx)` / `GetShortHead(ctx)` - Current HEAD ref

#### Implementation Files

```
internal/git/
├── client.go             # Enhanced with CommandModifier, new operations
├── errors.go             # Error helper functions (IsNotRepository, etc.)

cmd/
├── gh.go                 # Updated with ghGitCmd parent command
├── gh_git_clone.go       # Clone with profile/workspace selection
├── gh_git_status.go      # Status with short/porcelain modes
├── gh_git_commit.go      # Commit with -a/-m flags
├── gh_git_push.go        # Push with pre-push security scan
├── gh_git_pull.go        # Pull with auth
├── gh_git_log.go         # Log with filtering and JSON output
├── gh_git_diff.go        # Diff with staged/stat options
├── gh_git_branch.go      # Branch management with JSON output
```

#### Backward Compatibility

All existing top-level commands continue to work:
- `clonr clone` → unchanged
- `clonr commit` → unchanged
- `clonr push` → unchanged
- `clonr pull` → unchanged
- `clonr checkout` → unchanged
- `clonr stash` → unchanged
- `clonr merge` → unchanged

#### Future Enhancements (v0.9.1+)

- [ ] `clonr gh git stash` - Stash under gh git
- [ ] `clonr gh git checkout` - Checkout under gh git
- [ ] `clonr gh git merge` - Merge under gh git
- [ ] `clonr gh git tag` - Tag management
- [ ] `clonr gh git remote` - Remote management
- [ ] `clonr gh repo create/fork` - Repository creation via API

### v1.0.0 – Production Ready

#### Plugin System
- [ ] Custom command plugins
- [ ] Hook system for pre/post operations
- [ ] Plugin marketplace

#### Advanced UI
- [ ] Web dashboard (optional)
- [ ] Rich TUI improvements
- [ ] Notifications and alerts

#### Enterprise Features
- [ ] Multi-user support
- [ ] Role-based access control
- [ ] Audit logging
- [ ] LDAP/SSO integration

---

## Ideas & Future Exploration

### Integrations
- [ ] GitLab support
- [ ] Bitbucket support
- [ ] Azure DevOps support
- [ ] Gitea/Forgejo support

### Container Registry Profiles

Store Docker/container registry credentials securely using the same encryption as GitHub profiles.

#### Commands

```bash
# Add Docker Hub credentials
clonr profile docker add dockerhub --username myuser
# Prompts for password/token, encrypts with keystore

# Add other registries
clonr profile docker add ghcr --registry ghcr.io --username myuser
clonr profile docker add ecr --registry 123456789.dkr.ecr.us-east-1.amazonaws.com
clonr profile docker add gcr --registry gcr.io --service-account key.json

# List docker profiles
clonr profile docker list

# Login to registry (uses stored credentials)
clonr profile docker login dockerhub
clonr profile docker login --all  # Login to all registries

# Remove docker profile
clonr profile docker remove dockerhub

# Show docker profile status
clonr profile docker status dockerhub --json
```

#### Features
- [ ] Docker Hub authentication
- [ ] GitHub Container Registry (ghcr.io)
- [ ] AWS ECR authentication
- [ ] Google Container Registry (gcr.io)
- [ ] Azure Container Registry
- [ ] Generic registry support (any Docker v2 registry)
- [ ] Credential rotation with keystore
- [ ] Auto-login on `docker pull/push` (via credential helper)

#### Data Model

```go
type DockerProfile struct {
    Name           string    `json:"name"`
    Registry       string    `json:"registry"`       // e.g., "docker.io", "ghcr.io"
    Username       string    `json:"username"`
    EncryptedToken []byte    `json:"encrypted_token"` // Password/token encrypted with keystore
    CreatedAt      time.Time `json:"created_at"`
    LastUsedAt     time.Time `json:"last_used_at"`
}
```

#### Security
- Credentials encrypted with profile's keystore (same as GitHub tokens)
- TPM-backed when available
- Auto-rotation support
- No plaintext credentials on disk

### Messaging & Notification Platforms

Integration with team collaboration tools for notifications, alerts, and workflow automation. **Profile-aware** for proper separation of concerns.

#### Architecture: Profile-Based Notifications

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    PROFILE-BASED NOTIFICATION ARCHITECTURE                   │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  PROFILES                     NOTIFICATION CHANNELS                         │
│  ────────                     ──────────────────────                        │
│                                                                             │
│  ┌─────────────────┐          ┌─────────────────────────────────┐          │
│  │  work           │─────────▶│  slack: company-workspace       │          │
│  │  (github-work)  │          │  teams: devops-channel          │          │
│  │                 │          │  email: team@company.com        │          │
│  └─────────────────┘          │  webhook: ci.company.com        │          │
│                               └─────────────────────────────────┘          │
│                                                                             │
│  ┌─────────────────┐          ┌─────────────────────────────────┐          │
│  │  personal       │─────────▶│  discord: dev-server            │          │
│  │  (github-user)  │          │  email: me@gmail.com            │          │
│  └─────────────────┘          └─────────────────────────────────┘          │
│                                                                             │
│  ┌─────────────────┐          ┌─────────────────────────────────┐          │
│  │  opensource     │─────────▶│  discord: oss-community         │          │
│  │  (github-oss)   │          │  slack: oss-maintainers         │          │
│  └─────────────────┘          └─────────────────────────────────┘          │
│                                                                             │
│  Events from profile repos → routed to profile's notification channels     │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

#### Profile Notification Management

```bash
# Add notification channel to a profile
clonr profile notify add <profile> slack --webhook "https://hooks.slack.com/..."
clonr profile notify add <profile> slack --token "xoxb-..." --channel "#dev-ops"
clonr profile notify add <profile> teams --webhook "https://outlook.office.com/..."
clonr profile notify add <profile> discord --webhook "https://discord.com/api/webhooks/..."
clonr profile notify add <profile> email --smtp "smtp.gmail.com:587" --user "..." --password "..."
clonr profile notify add <profile> webhook --url "https://ci.example.com/notify"

# List notification channels for a profile
clonr profile notify list <profile>
clonr profile notify list  # uses default profile

# Remove notification channel
clonr profile notify remove <profile> slack
clonr profile notify remove <profile> discord

# Test notification channel
clonr profile notify test <profile> slack "Test message from clonr"
clonr profile notify test <profile> --all  # test all channels
```

#### Event Configuration Per Profile

```bash
# Configure which events trigger notifications (per profile)
clonr profile notify config <profile> --on push --channel slack --target "#commits"
clonr profile notify config <profile> --on pr-merge --channel teams
clonr profile notify config <profile> --on ci-fail --channel slack --target "#alerts" --priority high
clonr profile notify config <profile> --on release --channel discord,email

# Filter events by branch/repo patterns
clonr profile notify config <profile> --on push --channel slack --filter "branch:main"
clonr profile notify config <profile> --on push --channel slack --filter "repo:critical-*"

# Disable specific events
clonr profile notify config <profile> --on clone --disable

# List event configuration
clonr profile notify config <profile> --list
clonr profile notify config <profile> --list --event push
```

#### Global Notifications (Cross-Profile)

```bash
# Configure global notifications (all profiles)
clonr notify global add slack --webhook "..." --name "admin-alerts"
clonr notify global config --on error --channel admin-alerts
clonr notify global config --on sync --channel admin-alerts

# Global notifications for specific events across all profiles
clonr notify global list
```

#### Notification Channel Types

**Slack:**
```bash
clonr profile notify add work slack \
  --webhook "https://hooks.slack.com/services/..." \
  --default-channel "#dev-notifications"

# Or with bot token (richer features)
clonr profile notify add work slack \
  --token "xoxb-..." \
  --default-channel "#dev-notifications"
```

Features:
- [ ] Webhook support (simple setup)
- [ ] Bot token support (threads, reactions, file uploads)
- [ ] Interactive messages with buttons
- [ ] Thread replies for related events
- [ ] Per-repo channel overrides

**Microsoft Teams:**
```bash
clonr profile notify add work teams \
  --webhook "https://outlook.office.com/webhook/..."

# Or with Power Automate
clonr profile notify add work teams \
  --connector "..." \
  --flow-url "..."
```

Features:
- [ ] Incoming webhook support
- [ ] Adaptive cards for rich formatting
- [ ] Power Automate connector
- [ ] @mention support

**Discord:**
```bash
clonr profile notify add personal discord \
  --webhook "https://discord.com/api/webhooks/..."

# Or with bot token
clonr profile notify add personal discord \
  --token "..." \
  --default-channel "123456789"
```

Features:
- [ ] Webhook support
- [ ] Rich embeds with colors
- [ ] Role mentions
- [ ] Bot token for DMs and advanced features

**Email:**
```bash
# SMTP
clonr profile notify add work email \
  --provider smtp \
  --host "smtp.gmail.com" --port 587 \
  --user "..." --password "..." \
  --from "notifications@company.com" \
  --to "team@company.com"

# SendGrid
clonr profile notify add work email \
  --provider sendgrid \
  --api-key "..." \
  --from "noreply@company.com" \
  --to "team@company.com"

# AWS SES
clonr profile notify add work email \
  --provider ses \
  --region us-east-1 \
  --from "noreply@company.com"
```

Features:
- [ ] SMTP, SendGrid, AWS SES support
- [ ] HTML email templates
- [ ] Digest mode (batch notifications)
- [ ] CC/BCC support

**Generic Webhook:**
```bash
clonr profile notify add work webhook \
  --url "https://ci.example.com/notify" \
  --method POST \
  --header "X-API-Key: ..." \
  --template custom-payload.json

# With HMAC signing
clonr profile notify add work webhook \
  --url "..." \
  --hmac-secret "..." \
  --hmac-header "X-Signature"
```

Features:
- [ ] Custom HTTP headers
- [ ] HMAC signature verification
- [ ] Customizable payload templates (Go templates)
- [ ] Retry with exponential backoff
- [ ] Built-in templates: PagerDuty, Opsgenie, Jira, ServiceNow

#### Supported Events

| Event | Description | Context |
|-------|-------------|---------|
| `clone` | Repository cloned | repo name, profile |
| `push` | Commits pushed | repo, branch, commits |
| `pull` | Changes pulled | repo, branch, changes |
| `commit` | Local commit created | repo, message, files |
| `pr-create` | Pull request created | repo, PR number, title |
| `pr-merge` | Pull request merged | repo, PR number, author |
| `ci-pass` | CI workflow passed | repo, workflow, duration |
| `ci-fail` | CI workflow failed | repo, workflow, error |
| `release` | New release published | repo, version, notes |
| `sync` | Standalone sync completed | source, items synced |
| `error` | Any error occurred | command, error message |

#### Data Model

```go
// Profile contains notification channels (stored inside encrypted profile blob)
type Profile struct {
    // ... existing fields (name, token, ssh_key, etc.)

    // Notification channels - encrypted with profile
    NotifyChannels []NotifyChannel `json:"notify_channels,omitempty"`
}

// NotifyChannel represents a notification channel (stored inside Profile, not separate)
type NotifyChannel struct {
    ID        string            `json:"id"`
    Type      ChannelType       `json:"type"`        // slack, teams, discord, email, webhook
    Name      string            `json:"name"`        // User-friendly name
    Config    map[string]string `json:"config"`      // Webhook URLs, tokens, API keys (all encrypted with profile)
    Events    []EventConfig     `json:"events"`      // Which events trigger this channel
    Enabled   bool              `json:"enabled"`
    CreatedAt time.Time         `json:"created_at"`
    UpdatedAt time.Time         `json:"updated_at"`
}

type EventConfig struct {
    Event    string   `json:"event"`    // push, pr-merge, etc.
    Filters  []string `json:"filters"`  // branch:main, repo:critical-*
    Target   string   `json:"target"`   // #channel, email address, etc.
    Priority string   `json:"priority"` // low, normal, high
    Template string   `json:"template"` // Custom template name
}

type ChannelType string
const (
    ChannelSlack   ChannelType = "slack"
    ChannelTeams   ChannelType = "teams"
    ChannelDiscord ChannelType = "discord"
    ChannelEmail   ChannelType = "email"
    ChannelWebhook ChannelType = "webhook"
)

// Note: No foreign key needed - channels live inside the Profile struct
// When profile is serialized → encrypted → stored
// When profile is loaded → decrypted → channels available in memory
```

#### Implementation Files

```
internal/notify/
├── notify.go           # Notification dispatcher (profile-aware)
├── channel.go          # NotifyChannel model (no separate store - lives in Profile)
├── router.go           # Event → channel routing logic
├── sender.go           # Interface for sending notifications
├── slack/
│   ├── client.go       # Slack API client
│   └── templates.go    # Slack message templates
├── teams/
│   ├── client.go       # Teams webhook/connector client
│   └── cards.go        # Adaptive card builders
├── discord/
│   ├── client.go       # Discord webhook/bot client
│   └── embeds.go       # Embed builders
├── email/
│   ├── client.go       # Email client interface
│   ├── smtp.go         # SMTP implementation
│   ├── sendgrid.go     # SendGrid implementation
│   └── ses.go          # AWS SES implementation
├── webhook/
│   ├── client.go       # Generic webhook client
│   └── templates/      # Built-in payload templates
└── templates/
    └── *.tmpl          # Go templates for messages

cmd/
├── profile_notify.go        # clonr profile notify add/remove/list/test
├── profile_notify_config.go # clonr profile notify config
└── notify_global.go         # clonr notify global (uses dedicated encrypted store)

internal/core/
└── profile.go              # Profile struct extended with NotifyChannels field

# Note: No separate notify store needed
# NotifyChannels are part of Profile → encrypted together → stored in profile bucket
# Global notifications use a separate encrypted store (not tied to any profile)
```

#### Security Considerations

**Profile encryption applies to all notification data:**
- Notification channels are stored **inside the profile's encrypted blob**
- Webhook URLs, tokens, API keys, SMTP credentials - all encrypted at rest
- No separate storage - leverages existing profile encryption (AES-256-GCM)
- Decryption only happens in memory when profile is unlocked

```
┌─────────────────────────────────────────────────────────────┐
│  PROFILE (encrypted at rest)                                │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────────────────────────────────────────┐   │
│  │  Git Credentials (token, SSH key)                    │   │
│  ├─────────────────────────────────────────────────────┤   │
│  │  Notification Channels                               │   │
│  │  ├── Slack: webhook URL, bot token                  │   │
│  │  ├── Teams: webhook URL, connector token            │   │
│  │  ├── Discord: webhook URL, bot token                │   │
│  │  ├── Email: SMTP password, API keys                 │   │
│  │  └── Webhook: auth headers, HMAC secrets            │   │
│  ├─────────────────────────────────────────────────────┤   │
│  │  Event Configuration                                 │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  Encrypted with: AES-256-GCM + PBKDF2/Argon2               │
│  Key source: user password or TPM-sealed key               │
└─────────────────────────────────────────────────────────────┘
```

Additional security measures:
- [ ] HMAC verification for incoming webhooks (if bidirectional)
- [ ] Rate limiting to prevent notification spam
- [ ] Audit log for sent notifications (non-sensitive metadata only)
- [ ] Token rotation support for long-lived credentials
- [ ] Memory wiping after use (sensitive fields)

### Cloud Storage Integrations (PM Context)

Fetch complementary documentation from cloud drives when working with project management tools (Jira, ZenHub). Enables linking PRDs, design docs, meeting notes, and specifications to issues/epics.

**Planned Providers:**
- [ ] Google Drive - OAuth2 + Drive API v3
- [ ] OneDrive / SharePoint - Microsoft Graph API
- [ ] Dropbox - Dropbox API v2
- [ ] Box - Box API
- [ ] Notion - Notion API (pages as documents)
- [ ] Confluence - Atlassian API (pairs with Jira)

**Commands:**
```bash
# Configure cloud storage connection
clonr pm drive add google --name "work-drive"
clonr pm drive add onedrive --name "team-docs"
clonr pm drive list

# Link documents to Jira/ZenHub issues
clonr pm jira issues view PROJ-123 --with-docs
clonr pm zenhub issue 456 --with-docs

# Search for related documents
clonr pm docs search "authentication redesign"
clonr pm docs search --issue PROJ-123

# Fetch document content for context
clonr pm docs fetch <doc-id>
clonr pm docs fetch --linked PROJ-123
```

**Features:**
- Auto-detect linked documents from issue descriptions (Drive/OneDrive URLs)
- Full-text search across connected drives
- Cache frequently accessed documents locally
- Offline mode with cached content
- JSON output for AI/automation context
- Document summarization for issue context

**Use Cases:**
- Pull PRD context when viewing Jira epics
- Find design docs related to ZenHub issues
- Aggregate meeting notes for sprint planning
- Build AI context from linked documentation

### Project Management
- [x] Jira Cloud integration (v0.5.0)
- [x] ZenHub integration (v0.5.0)
- [ ] Linear integration
- [ ] Trello integration
- [ ] Asana integration
- [ ] Monday.com integration
- [ ] ClickUp integration

### Developer Experience
- [ ] Shell completions (bash, zsh, fish, powershell)
- [ ] VS Code extension
- [ ] JetBrains plugin
- [ ] Alfred/Raycast workflows

### Security
- [x] TPM 2.0 hardware-backed encryption (Linux) - `clonr tpm init/status/reset/migrate/migrate-profiles`
- [x] KeePass database integration for secure token storage
- [x] Extracted TPM to reusable `sealbox` package (v0.7.0) - `github.com/inovacc/sealbox`
- [ ] TPM 2.0 support for Windows (v0.7.0) - pending sealbox Windows implementation
- [ ] macOS Secure Enclave support (v0.7.0 stretch goal)
- [ ] Gitleaks integration for secret detection (v0.4.0)
- [ ] Pre-push/pre-pull secret scanning
- [ ] Automated security audits
- [ ] Dependency vulnerability scanning
- [ ] License compliance checking

### Advanced Features
- [ ] Repository templates
- [ ] Automated PR creation
- [ ] CI/CD status monitoring
- [ ] Dependency tracking across repos

---

## Version History

| Version | Status | Highlights |
|---------|--------|------------|
| v0.1.1  | ✅ Done | GitHub CLI integration (issues, PRs, actions, releases) |
| v0.2.0  | ✅ Done | Add, Map, Favorites, Open, Configure, gRPC architecture |
| v0.3.0  | 🚧 WIP | Status, Nerds, Reauthor, Organization support |
| v0.4.0  | Planned | Branch management, enhanced statistics |
| v0.5.0  | 🚧 WIP | Team features, PM integrations (Jira, ZenHub) |
| v0.6.0  | 🚧 WIP | Instance synchronization (standalone mode), encrypted sync, repo archives |
| v0.7.0  | 🚧 WIP | Cross-platform TPM, sealbox integration (done), Windows support (pending) |
| v0.7.1  | ✅ Done | Code refactoring: shared packages, reduced duplication |
| v0.8.0  | Planned | Multi-database support (BoltDB default, SQLite, PostgreSQL) |
| v0.9.0  | ✅ Done | Git subcommand reorganization (`gh git`), enhanced git client, error helpers |
| v0.9.1  | Planned | Messaging integrations (Slack, Teams, Discord, webhooks, email) |
| v1.0.0  | Planned | Production ready with plugins and enterprise features |

---

## Code Quality Improvements

### v0.7.1 – Code Refactoring ✅ (Completed)

Consolidated duplicate code patterns and created shared utility packages:

#### New Shared Packages

| Package | Purpose | Lines Saved |
|---------|---------|-------------|
| `internal/mapper` | Proto ↔ Model conversions shared by server/client | ~150 lines |
| `internal/auth` | Generic token resolver framework | Reduces future duplication |
| `internal/encoding` | JSON/file utilities with generics | ~50+ lines |

#### New Helper Files

| File | Purpose |
|------|---------|
| `internal/core/gh_client.go` | GitHub OAuth client creation (consolidated 16+ patterns) |
| `internal/core/context.go` | Context timeout helpers with predefined durations |
| `internal/server/grpc/validation.go` | gRPC validation helpers (RequiredString, etc.) |

#### Improvements

- **Mapper Package**: Centralized proto-model conversions eliminate duplication between server and client
- **GitHub Client Helper**: `NewGitHubClient(ctx, token)` replaces scattered OAuth2 boilerplate
- **Context Check Interceptor**: Fast-fail for already-canceled gRPC requests
- **AppName Constants**: Standardized `application.AppName`, `AppExeName`, `AppExeNameWindows`
- **Validation Helpers**: Reusable gRPC validation with consistent error messages
- **Token Resolver**: Builder pattern for flexible multi-source token resolution

#### Files Updated

- `cmd/root.go`, `cmd/service.go`, `cmd/aicontext.go` - Use `application.AppName` constant
- `internal/core/*.go` - Use `NewGitHubClient` helper, removed oauth2 imports
- `internal/client/grpc/client.go` - Use shared mapper package
- `internal/server/grpc/server.go` - Added context check interceptor
- `internal/zenhub/issues.go` - Use `core.NewGitHubClient`
