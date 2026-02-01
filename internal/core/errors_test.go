package core

import (
	"errors"
	"testing"
)

func TestDirtyRepoError(t *testing.T) {
	err := &DirtyRepoError{Path: "/home/user/repo"}

	expected := "repository has uncommitted changes: /home/user/repo"
	if err.Error() != expected {
		t.Errorf("DirtyRepoError.Error() = %q, want %q", err.Error(), expected)
	}
}

func TestPathCollisionError(t *testing.T) {
	err := &PathCollisionError{
		Path:        "/home/user/repo",
		ExpectedURL: "https://github.com/user/repo",
		ActualURL:   "https://github.com/other/repo",
	}

	expected := "path collision: /home/user/repo contains https://github.com/other/repo, expected https://github.com/user/repo"
	if err.Error() != expected {
		t.Errorf("PathCollisionError.Error() = %q, want %q", err.Error(), expected)
	}
}

func TestNetworkError(t *testing.T) {
	innerErr := errors.New("connection refused")
	err := &NetworkError{
		Operation: "clone",
		Err:       innerErr,
		Attempts:  3,
	}

	expected := "clone failed after 3 attempts: connection refused"
	if err.Error() != expected {
		t.Errorf("NetworkError.Error() = %q, want %q", err.Error(), expected)
	}

	// Test Unwrap
	if !errors.Is(innerErr, err.Unwrap()) {
		t.Error("NetworkError.Unwrap() should return the inner error")
	}
}

func TestNetworkError_Unwrap(t *testing.T) {
	innerErr := errors.New("timeout")
	err := &NetworkError{
		Operation: "fetch",
		Err:       innerErr,
		Attempts:  5,
	}

	// Should work with errors.Is/errors.As
	if !errors.Is(err, innerErr) {
		t.Error("errors.Is should find the inner error")
	}
}

func TestSkipReason_String(t *testing.T) {
	tests := []struct {
		reason   SkipReason
		expected string
	}{
		{SkipReasonNone, ""},
		{SkipReasonDirty, "dirty repository"},
		{SkipReasonPathCollision, "path collision"},
		{SkipReasonArchived, "archived"},
		{SkipReasonFiltered, "filtered out"},
		{SkipReasonNotGitRepo, "not a git repository"},
		{SkipReason(99), ""}, // Unknown reason
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.reason.String(); got != tt.expected {
				t.Errorf("SkipReason(%d).String() = %q, want %q", tt.reason, got, tt.expected)
			}
		})
	}
}

func TestDirtyRepoStrategy_String(t *testing.T) {
	tests := []struct {
		strategy DirtyRepoStrategy
		expected string
	}{
		{DirtyStrategySkip, "skip"},
		{DirtyStrategyStash, "stash"},
		{DirtyStrategyReset, "reset"},
		{DirtyRepoStrategy(99), "skip"}, // Unknown defaults to skip
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.strategy.String(); got != tt.expected {
				t.Errorf("DirtyRepoStrategy(%d).String() = %q, want %q", tt.strategy, got, tt.expected)
			}
		})
	}
}

func TestParseDirtyStrategy(t *testing.T) {
	tests := []struct {
		input    string
		expected DirtyRepoStrategy
	}{
		{"skip", DirtyStrategySkip},
		{"stash", DirtyStrategyStash},
		{"reset", DirtyStrategyReset},
		{"unknown", DirtyStrategySkip},
		{"", DirtyStrategySkip},
		{"STASH", DirtyStrategySkip}, // Case sensitive, defaults to skip
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := ParseDirtyStrategy(tt.input); got != tt.expected {
				t.Errorf("ParseDirtyStrategy(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestSkipReasonConstants(t *testing.T) {
	// Verify constants have expected values
	if SkipReasonNone != 0 {
		t.Errorf("SkipReasonNone = %d, want 0", SkipReasonNone)
	}

	if SkipReasonDirty != 1 {
		t.Errorf("SkipReasonDirty = %d, want 1", SkipReasonDirty)
	}

	if SkipReasonPathCollision != 2 {
		t.Errorf("SkipReasonPathCollision = %d, want 2", SkipReasonPathCollision)
	}

	if SkipReasonArchived != 3 {
		t.Errorf("SkipReasonArchived = %d, want 3", SkipReasonArchived)
	}

	if SkipReasonFiltered != 4 {
		t.Errorf("SkipReasonFiltered = %d, want 4", SkipReasonFiltered)
	}

	if SkipReasonNotGitRepo != 5 {
		t.Errorf("SkipReasonNotGitRepo = %d, want 5", SkipReasonNotGitRepo)
	}
}

func TestDirtyStrategyConstants(t *testing.T) {
	// Verify constants have expected values
	if DirtyStrategySkip != 0 {
		t.Errorf("DirtyStrategySkip = %d, want 0", DirtyStrategySkip)
	}

	if DirtyStrategyStash != 1 {
		t.Errorf("DirtyStrategyStash = %d, want 1", DirtyStrategyStash)
	}

	if DirtyStrategyReset != 2 {
		t.Errorf("DirtyStrategyReset = %d, want 2", DirtyStrategyReset)
	}
}
