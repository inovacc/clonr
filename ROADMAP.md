# Clonr Roadmap

This roadmap focuses on the features requested in todo.md and aligns them with the current architecture. It is scoped to the next two minor releases. Dates are tentative and based on effort, with
sequence reflecting dependencies.

## Vision

Clonr simplifies cloning and managing many Git repositories from the CLI or a local API server. The next step is to improve repository onboarding (add), quality-of-life (favorites and
open-after-clone), and private repository support.

## Milestones

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

## Feature Breakdowns

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
- Minimal surprise: defaults don’t break current flows; flags are opt-in
