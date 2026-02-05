package actionsdb

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// Database schema version for migrations
const SchemaVersion = 1

// DB represents the GitHub Actions monitoring database
type DB struct {
	*sql.DB
	path string
}

// WorkflowRun represents a GitHub Actions workflow run
type WorkflowRun struct {
	ID           int64     `json:"id"`
	RunID        int64     `json:"run_id"` // GitHub run ID
	RepoOwner    string    `json:"repo_owner"`
	RepoName     string    `json:"repo_name"`
	WorkflowID   int64     `json:"workflow_id"`
	WorkflowName string    `json:"workflow_name"`
	HeadBranch   string    `json:"head_branch"`
	HeadSHA      string    `json:"head_sha"`
	Event        string    `json:"event"`      // "push", "pull_request", etc.
	Status       string    `json:"status"`     // "queued", "in_progress", "completed"
	Conclusion   string    `json:"conclusion"` // "success", "failure", "cancelled", etc.
	HTMLURL      string    `json:"html_url"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	StartedAt    time.Time `json:"started_at,omitempty"`
	CompletedAt  time.Time `json:"completed_at,omitempty"`
	PushID       int64     `json:"push_id"` // Local reference to the push that triggered this
}

// WorkflowJob represents a job within a workflow run
type WorkflowJob struct {
	ID          int64     `json:"id"`
	JobID       int64     `json:"job_id"` // GitHub job ID
	RunID       int64     `json:"run_id"` // Parent workflow run ID
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	Conclusion  string    `json:"conclusion"`
	StartedAt   time.Time `json:"started_at,omitempty"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
	Steps       int       `json:"steps"` // Number of steps
	StepsPassed int       `json:"steps_passed"`
	StepsFailed int       `json:"steps_failed"`
}

// PushRecord represents a push made via clonr
type PushRecord struct {
	ID        int64     `json:"id"`
	RepoOwner string    `json:"repo_owner"`
	RepoName  string    `json:"repo_name"`
	Branch    string    `json:"branch"`
	CommitSHA string    `json:"commit_sha"`
	Remote    string    `json:"remote"`
	PushedAt  time.Time `json:"pushed_at"`
	Monitored bool      `json:"monitored"` // Whether actions are being monitored
	LastCheck time.Time `json:"last_check"`
}

// QueueItem represents an item in the monitoring queue
type QueueItem struct {
	ID         int64     `json:"id"`
	PushID     int64     `json:"push_id"`
	RepoOwner  string    `json:"repo_owner"`
	RepoName   string    `json:"repo_name"`
	CommitSHA  string    `json:"commit_sha"`
	Status     string    `json:"status"` // "pending", "checking", "completed", "failed"
	RetryCount int       `json:"retry_count"`
	NextCheck  time.Time `json:"next_check"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	Error      string    `json:"error,omitempty"`
}

// DefaultDBPath returns the default path for the actions database
func DefaultDBPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get config directory: %w", err)
	}
	return filepath.Join(configDir, "clonr", "actions.db"), nil
}

// Open opens or creates the actions database
func Open(path string) (*DB, error) {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open SQLite database
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable WAL mode for better concurrency
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	adb := &DB{DB: db, path: path}

	// Run migrations
	if err := adb.migrate(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return adb, nil
}

// migrate runs database migrations
func (db *DB) migrate() error {
	// Create migrations table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get current version
	var currentVersion int
	err = db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&currentVersion)
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	// Apply migrations
	migrations := []struct {
		version int
		sql     string
	}{
		{1, `
			-- Push records table
			CREATE TABLE IF NOT EXISTS push_records (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				repo_owner TEXT NOT NULL,
				repo_name TEXT NOT NULL,
				branch TEXT NOT NULL,
				commit_sha TEXT NOT NULL,
				remote TEXT NOT NULL,
				pushed_at DATETIME NOT NULL,
				monitored BOOLEAN DEFAULT FALSE,
				last_check DATETIME,
				UNIQUE(repo_owner, repo_name, commit_sha)
			);

			CREATE INDEX IF NOT EXISTS idx_push_records_repo ON push_records(repo_owner, repo_name);
			CREATE INDEX IF NOT EXISTS idx_push_records_sha ON push_records(commit_sha);

			-- Workflow runs table
			CREATE TABLE IF NOT EXISTS workflow_runs (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				run_id INTEGER UNIQUE NOT NULL,
				repo_owner TEXT NOT NULL,
				repo_name TEXT NOT NULL,
				workflow_id INTEGER NOT NULL,
				workflow_name TEXT NOT NULL,
				head_branch TEXT NOT NULL,
				head_sha TEXT NOT NULL,
				event TEXT NOT NULL,
				status TEXT NOT NULL,
				conclusion TEXT,
				html_url TEXT,
				created_at DATETIME NOT NULL,
				updated_at DATETIME NOT NULL,
				started_at DATETIME,
				completed_at DATETIME,
				push_id INTEGER REFERENCES push_records(id)
			);

			CREATE INDEX IF NOT EXISTS idx_workflow_runs_repo ON workflow_runs(repo_owner, repo_name);
			CREATE INDEX IF NOT EXISTS idx_workflow_runs_sha ON workflow_runs(head_sha);
			CREATE INDEX IF NOT EXISTS idx_workflow_runs_status ON workflow_runs(status);
			CREATE INDEX IF NOT EXISTS idx_workflow_runs_push ON workflow_runs(push_id);

			-- Workflow jobs table
			CREATE TABLE IF NOT EXISTS workflow_jobs (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				job_id INTEGER UNIQUE NOT NULL,
				run_id INTEGER NOT NULL REFERENCES workflow_runs(run_id),
				name TEXT NOT NULL,
				status TEXT NOT NULL,
				conclusion TEXT,
				started_at DATETIME,
				completed_at DATETIME,
				steps INTEGER DEFAULT 0,
				steps_passed INTEGER DEFAULT 0,
				steps_failed INTEGER DEFAULT 0
			);

			CREATE INDEX IF NOT EXISTS idx_workflow_jobs_run ON workflow_jobs(run_id);

			-- Monitoring queue table
			CREATE TABLE IF NOT EXISTS queue_items (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				push_id INTEGER NOT NULL REFERENCES push_records(id),
				repo_owner TEXT NOT NULL,
				repo_name TEXT NOT NULL,
				commit_sha TEXT NOT NULL,
				status TEXT NOT NULL DEFAULT 'pending',
				retry_count INTEGER DEFAULT 0,
				next_check DATETIME NOT NULL,
				created_at DATETIME NOT NULL,
				updated_at DATETIME NOT NULL,
				error TEXT
			);

			CREATE INDEX IF NOT EXISTS idx_queue_items_status ON queue_items(status, next_check);
			CREATE INDEX IF NOT EXISTS idx_queue_items_push ON queue_items(push_id);
		`},
	}

	for _, m := range migrations {
		if m.version > currentVersion {
			if _, err := db.Exec(m.sql); err != nil {
				return fmt.Errorf("failed to apply migration %d: %w", m.version, err)
			}
			if _, err := db.Exec("INSERT INTO schema_migrations (version) VALUES (?)", m.version); err != nil {
				return fmt.Errorf("failed to record migration %d: %w", m.version, err)
			}
		}
	}

	return nil
}

// Path returns the database file path
func (db *DB) Path() string {
	return db.path
}
