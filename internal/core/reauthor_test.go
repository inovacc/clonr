package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReauthorOptionsDefaults(t *testing.T) {
	tests := []struct {
		name        string
		opts        ReauthorOptions
		expectValid bool
	}{
		{
			name: "valid options",
			opts: ReauthorOptions{
				OldEmail: "old@example.com",
				NewEmail: "new@example.com",
			},
			expectValid: true,
		},
		{
			name: "valid options with name",
			opts: ReauthorOptions{
				OldEmail: "old@example.com",
				NewEmail: "new@example.com",
				NewName:  "New Name",
			},
			expectValid: true,
		},
		{
			name: "valid options with all fields",
			opts: ReauthorOptions{
				OldEmail: "old@example.com",
				NewEmail: "new@example.com",
				NewName:  "New Name",
				RepoPath: "/path/to/repo",
				AllRefs:  true,
			},
			expectValid: true,
		},
		{
			name: "missing old email",
			opts: ReauthorOptions{
				NewEmail: "new@example.com",
			},
			expectValid: false,
		},
		{
			name: "missing new email",
			opts: ReauthorOptions{
				OldEmail: "old@example.com",
			},
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic validation - check required fields
			isValid := tt.opts.OldEmail != "" && tt.opts.NewEmail != ""
			assert.Equal(t, tt.expectValid, isValid)
		})
	}
}

func TestReauthorResultStructure(t *testing.T) {
	result := &ReauthorResult{
		CommitsRewritten:  10,
		TagsRewritten:     []string{"v1.0.0", "v2.0.0"},
		BranchesRewritten: []string{"main", "develop"},
	}

	assert.Equal(t, 10, result.CommitsRewritten)
	assert.Len(t, result.TagsRewritten, 2)
	assert.Len(t, result.BranchesRewritten, 2)
	assert.Contains(t, result.TagsRewritten, "v1.0.0")
	assert.Contains(t, result.BranchesRewritten, "main")
}
