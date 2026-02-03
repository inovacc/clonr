package git

import (
	"errors"
	"os/exec"
	"strings"
)

// Common error messages from git
const (
	errMsgNotRepository    = "not a git repository"
	errMsgNoUpstream       = "no upstream branch"
	errMsgAuthFailed       = "Authentication failed"
	errMsgPermissionDenied = "Permission denied"
	errMsgRefNotFound      = "couldn't find remote ref"
	errMsgConflict         = "CONFLICT"
	errMsgAlreadyExists    = "already exists"
	errMsgNothingToCommit  = "nothing to commit"
	errMsgDetachedHead     = "HEAD detached"
)

// IsNotRepository checks if the error indicates not a git repository
func IsNotRepository(err error) bool {
	return containsError(err, errMsgNotRepository)
}

// IsAuthRequired checks if the error indicates authentication is required
func IsAuthRequired(err error) bool {
	return containsError(err, errMsgAuthFailed) || containsError(err, errMsgPermissionDenied)
}

// IsNoUpstream checks if the error indicates no upstream branch configured
func IsNoUpstream(err error) bool {
	return containsError(err, errMsgNoUpstream)
}

// IsRefNotFound checks if the error indicates a ref was not found
func IsRefNotFound(err error) bool {
	return containsError(err, errMsgRefNotFound)
}

// IsConflict checks if the error indicates a merge conflict
func IsConflict(err error) bool {
	return containsError(err, errMsgConflict)
}

// IsAlreadyExists checks if the error indicates something already exists
func IsAlreadyExists(err error) bool {
	return containsError(err, errMsgAlreadyExists)
}

// IsNothingToCommit checks if the error indicates nothing to commit
func IsNothingToCommit(err error) bool {
	return containsError(err, errMsgNothingToCommit)
}

// IsDetachedHead checks if the error indicates HEAD is detached
func IsDetachedHead(err error) bool {
	return containsError(err, errMsgDetachedHead)
}

// containsError checks if the error contains a specific message
func containsError(err error, msg string) bool {
	if err == nil {
		return false
	}

	var gitErr *GitError
	if errors.As(err, &gitErr) {
		return strings.Contains(strings.ToLower(gitErr.Stderr), strings.ToLower(msg))
	}

	return strings.Contains(strings.ToLower(err.Error()), strings.ToLower(msg))
}

// GetExitCode returns the exit code from a git error, or -1 if not available
func GetExitCode(err error) int {
	if err == nil {
		return 0
	}

	var gitErr *GitError
	if errors.As(err, &gitErr) {
		return gitErr.ExitCode
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}

	return -1
}

// NewGitError creates a GitError from command output and error
func NewGitError(args []string, stderr string, err error) *GitError {
	exitCode := -1

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		exitCode = exitErr.ExitCode()
	}

	return &GitError{
		ExitCode: exitCode,
		Stderr:   stderr,
		Args:     args,
		err:      err,
	}
}
