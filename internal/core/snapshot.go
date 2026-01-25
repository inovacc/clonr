package core

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/inovacc/clonr/internal/grpcclient"
	"github.com/inovacc/clonr/internal/model"
)

// SnapshotVersion is the current snapshot format version
const SnapshotVersion = "1.0"

// Snapshot represents a complete database export
type Snapshot struct {
	Version      string               `json:"version"`
	CreatedAt    time.Time            `json:"created_at"`
	Hostname     string               `json:"hostname,omitempty"`
	Repositories []RepositorySnapshot `json:"repositories"`
	Config       *model.Config        `json:"config,omitempty"`
}

// RepositorySnapshot extends Repository with branch info
type RepositorySnapshot struct {
	model.Repository

	CurrentBranch string `json:"current_branch,omitempty"`
	BranchError   string `json:"branch_error,omitempty"`
}

// CreateSnapshotOptions configures snapshot creation
type CreateSnapshotOptions struct {
	IncludeBranches bool // Fetch current branch for each repo
	IncludeConfig   bool // Include configuration
}

// DefaultSnapshotOptions returns sensible defaults for snapshot creation
func DefaultSnapshotOptions() CreateSnapshotOptions {
	return CreateSnapshotOptions{
		IncludeBranches: true,
		IncludeConfig:   true,
	}
}

// CreateSnapshot creates a database snapshot with optional branch info
func CreateSnapshot(opts CreateSnapshotOptions) (*Snapshot, error) {
	client, err := grpcclient.GetClient()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}

	// Get all repositories
	repos, err := client.GetAllRepos()
	if err != nil {
		return nil, fmt.Errorf("failed to get repositories: %w", err)
	}

	// Build repository snapshots
	repoSnapshots := make([]RepositorySnapshot, 0, len(repos))
	for _, repo := range repos {
		snapshot := RepositorySnapshot{
			Repository: repo,
		}

		// Get current branch if requested and path exists
		if opts.IncludeBranches && repo.Path != "" {
			if _, err := os.Stat(repo.Path); err == nil {
				branch, branchErr := GetCurrentBranch(repo.Path)
				if branchErr != nil {
					snapshot.BranchError = branchErr.Error()
				} else {
					snapshot.CurrentBranch = branch
				}
			} else {
				snapshot.BranchError = "path does not exist"
			}
		}

		repoSnapshots = append(repoSnapshots, snapshot)
	}

	// Build snapshot
	snapshot := &Snapshot{
		Version:      SnapshotVersion,
		CreatedAt:    time.Now().UTC(),
		Repositories: repoSnapshots,
	}

	// Get hostname
	if hostname, err := os.Hostname(); err == nil {
		snapshot.Hostname = hostname
	}

	// Get config if requested
	if opts.IncludeConfig {
		cfg, err := client.GetConfig()
		if err == nil {
			snapshot.Config = cfg
		}
		// Silently ignore config errors - it's optional
	}

	return snapshot, nil
}

// WriteSnapshot writes a snapshot to a writer as JSON
func WriteSnapshot(w io.Writer, snapshot *Snapshot, pretty bool) error {
	enc := json.NewEncoder(w)
	if pretty {
		enc.SetIndent("", "  ")
	}

	if err := enc.Encode(snapshot); err != nil {
		return fmt.Errorf("failed to encode snapshot: %w", err)
	}

	return nil
}

// WriteSnapshotToFile writes a snapshot to a file
func WriteSnapshotToFile(path string, snapshot *Snapshot, pretty bool) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	defer func() {
		if err := file.Close(); err != nil {
			// Log error but don't fail - the write may have succeeded
			_, _ = fmt.Fprintf(os.Stderr, "warning: failed to close file: %v\n", err)
		}
	}()

	return WriteSnapshot(file, snapshot, pretty)
}
