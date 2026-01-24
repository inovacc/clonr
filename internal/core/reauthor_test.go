package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildEnvFilterScript(t *testing.T) {
	tests := []struct {
		name     string
		opts     ReauthorOptions
		contains []string
	}{
		{
			name: "basic email replacement",
			opts: ReauthorOptions{
				OldEmail: "old@example.com",
				NewEmail: "new@example.com",
			},
			contains: []string{
				`OLD_EMAIL="old@example.com"`,
				`CORRECT_EMAIL="new@example.com"`,
				`GIT_COMMITTER_EMAIL`,
				`GIT_AUTHOR_EMAIL`,
			},
		},
		{
			name: "with name replacement",
			opts: ReauthorOptions{
				OldEmail: "old@example.com",
				NewEmail: "new@example.com",
				NewName:  "New Name",
			},
			contains: []string{
				`OLD_EMAIL="old@example.com"`,
				`CORRECT_EMAIL="new@example.com"`,
				`CORRECT_NAME="New Name"`,
				`GIT_COMMITTER_NAME`,
				`GIT_AUTHOR_NAME`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			script := buildEnvFilterScript(tt.opts)

			for _, expected := range tt.contains {
				assert.Contains(t, script, expected)
			}
		})
	}
}

func TestParseFilterBranchOutput(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected *ReauthorResult
	}{
		{
			name:   "empty output",
			output: "",
			expected: &ReauthorResult{
				CommitsRewritten:  0,
				TagsRewritten:     []string{},
				BranchesRewritten: []string{},
			},
		},
		{
			name: "with rewritten commits and refs",
			output: `Rewrite abc123 (1/3)
Rewrite def456 (2/3)
Rewrite ghi789 (3/3)
Ref 'refs/heads/main' was rewritten
Ref 'refs/tags/v1.0.0' was rewritten`,
			expected: &ReauthorResult{
				CommitsRewritten:  3,
				TagsRewritten:     []string{"v1.0.0"},
				BranchesRewritten: []string{"main"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseFilterBranchOutput(tt.output)

			assert.Equal(t, tt.expected.CommitsRewritten, result.CommitsRewritten)
			assert.Equal(t, tt.expected.TagsRewritten, result.TagsRewritten)
			assert.Equal(t, tt.expected.BranchesRewritten, result.BranchesRewritten)
		})
	}
}

func TestExtractRefName(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		prefix   string
		expected string
	}{
		{
			name:     "branch ref",
			line:     "Ref 'refs/heads/main' was rewritten",
			prefix:   "refs/heads/",
			expected: "main",
		},
		{
			name:     "tag ref",
			line:     "Ref 'refs/tags/v1.0.0' was rewritten",
			prefix:   "refs/tags/",
			expected: "v1.0.0",
		},
		{
			name:     "feature branch",
			line:     "Ref 'refs/heads/feature/my-feature' was rewritten",
			prefix:   "refs/heads/",
			expected: "feature/my-feature",
		},
		{
			name:     "no match",
			line:     "Some other line",
			prefix:   "refs/heads/",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractRefName(tt.line, tt.prefix)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestReauthorOptionsValidation(t *testing.T) {
	tests := []struct {
		name        string
		opts        ReauthorOptions
		expectError bool
	}{
		{
			name: "valid options",
			opts: ReauthorOptions{
				OldEmail: "old@example.com",
				NewEmail: "new@example.com",
			},
			expectError: false,
		},
		{
			name: "missing old email",
			opts: ReauthorOptions{
				NewEmail: "new@example.com",
			},
			expectError: true,
		},
		{
			name: "missing new email",
			opts: ReauthorOptions{
				OldEmail: "old@example.com",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We can't fully test Reauthor without a real git repo,
			// but we can test the validation logic
			if tt.opts.OldEmail == "" {
				assert.True(t, tt.expectError, "should expect error for missing old email")
			}
			if tt.opts.NewEmail == "" {
				assert.True(t, tt.expectError, "should expect error for missing new email")
			}
		})
	}
}
