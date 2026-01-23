package core

import "fmt"

// DirtyRepoError indicates a repository has uncommitted changes
type DirtyRepoError struct {
	Path string
}

func (e *DirtyRepoError) Error() string {
	return fmt.Sprintf("repository has uncommitted changes: %s", e.Path)
}

// PathCollisionError indicates the path exists with different content
type PathCollisionError struct {
	Path        string
	ExpectedURL string
	ActualURL   string
}

func (e *PathCollisionError) Error() string {
	return fmt.Sprintf("path collision: %s contains %s, expected %s",
		e.Path, e.ActualURL, e.ExpectedURL)
}

// NetworkError wraps transient network failures
type NetworkError struct {
	Operation string
	Err       error
	Attempts  int
}

func (e *NetworkError) Error() string {
	return fmt.Sprintf("%s failed after %d attempts: %v",
		e.Operation, e.Attempts, e.Err)
}

func (e *NetworkError) Unwrap() error {
	return e.Err
}

// SkipReason categorizes why a repo was skipped
type SkipReason int

const (
	SkipReasonNone SkipReason = iota
	SkipReasonDirty
	SkipReasonPathCollision
	SkipReasonArchived
	SkipReasonFiltered
	SkipReasonNotGitRepo
)

func (r SkipReason) String() string {
	switch r {
	case SkipReasonNone:
		return ""
	case SkipReasonDirty:
		return "dirty repository"
	case SkipReasonPathCollision:
		return "path collision"
	case SkipReasonArchived:
		return "archived"
	case SkipReasonFiltered:
		return "filtered out"
	case SkipReasonNotGitRepo:
		return "not a git repository"
	}
	return ""
}

// DirtyRepoStrategy defines how to handle repos with uncommitted changes
type DirtyRepoStrategy int

const (
	DirtyStrategySkip  DirtyRepoStrategy = iota // Skip with warning (default)
	DirtyStrategyStash                          // Stash changes, pull, unstash
	DirtyStrategyReset                          // Reset to clean state (destructive)
)

func (s DirtyRepoStrategy) String() string {
	switch s {
	case DirtyStrategySkip:
		return "skip"
	case DirtyStrategyStash:
		return "stash"
	case DirtyStrategyReset:
		return "reset"
	default:
		return "skip"
	}
}

// ParseDirtyStrategy converts a string to DirtyRepoStrategy
func ParseDirtyStrategy(s string) DirtyRepoStrategy {
	switch s {
	case "stash":
		return DirtyStrategyStash
	case "reset":
		return DirtyStrategyReset
	default:
		return DirtyStrategySkip
	}
}
