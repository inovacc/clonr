package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/inovacc/clonr/internal/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSanitizeGitURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "URL with credentials",
			input:    "https://user:token123@github.com/owner/repo.git",
			expected: "https://github.com/owner/repo.git",
		},
		{
			name:     "URL with token only (OAuth style)",
			input:    "https://ghp_xxxxx:x-oauth-basic@github.com/owner/repo.git",
			expected: "https://github.com/owner/repo.git",
		},
		{
			name:     "URL without credentials",
			input:    "https://github.com/owner/repo.git",
			expected: "https://github.com/owner/repo.git",
		},
		{
			name:     "SSH URL (unchanged)",
			input:    "git@github.com:owner/repo.git",
			expected: "git@github.com:owner/repo.git",
		},
		{
			name:     "Empty URL",
			input:    "",
			expected: "",
		},
		{
			name:     "URL with port",
			input:    "https://user:pass@gitlab.example.com:8443/group/project.git",
			expected: "https://gitlab.example.com:8443/group/project.git",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := common.SanitizeGitURL(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSaveAndLoadGitStats(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	// Create test stats
	stats := &GitStats{
		Repository:   "https://github.com/test/repo",
		Path:         tmpDir,
		FetchedAt:    time.Now(),
		TotalCommits: 100,
		TotalAuthors: 5,
		LinesAdded:   1000,
		LinesDeleted: 500,
		LinesChanged: 1500,
		Contributors: []ContributorStats{
			{
				Name:    "Test User",
				Email:   "test@example.com",
				Commits: 50,
				Since:   time.Now().AddDate(-1, 0, 0),
			},
		},
	}

	// Save stats
	err := saveGitStats(tmpDir, stats)
	require.NoError(t, err)

	// Verify file exists
	statsPath := filepath.Join(tmpDir, ".clonr", "stats.json")
	assert.FileExists(t, statsPath)

	// Load stats
	loaded, err := LoadGitStats(tmpDir)
	require.NoError(t, err)

	// Verify loaded stats
	assert.Equal(t, stats.Repository, loaded.Repository)
	assert.Equal(t, stats.TotalCommits, loaded.TotalCommits)
	assert.Equal(t, stats.TotalAuthors, loaded.TotalAuthors)
	assert.Equal(t, stats.LinesAdded, loaded.LinesAdded)
	assert.Len(t, loaded.Contributors, 1)
	assert.Equal(t, "Test User", loaded.Contributors[0].Name)
}

func TestGitStatsExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Should not exist initially
	assert.False(t, GitStatsExists(tmpDir))

	// Create the stats file
	clonrDir := filepath.Join(tmpDir, ".clonr")
	require.NoError(t, os.MkdirAll(clonrDir, 0755))

	statsPath := filepath.Join(clonrDir, "stats.json")
	require.NoError(t, os.WriteFile(statsPath, []byte("{}"), 0644))

	// Should exist now
	assert.True(t, GitStatsExists(tmpDir))
}

func TestLoadGitStats_FileNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := LoadGitStats(tmpDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read stats file")
}

func TestLoadGitStats_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()

	// Create invalid JSON file
	clonrDir := filepath.Join(tmpDir, ".clonr")
	require.NoError(t, os.MkdirAll(clonrDir, 0755))

	statsPath := filepath.Join(clonrDir, "stats.json")
	require.NoError(t, os.WriteFile(statsPath, []byte("not valid json"), 0644))

	_, err := LoadGitStats(tmpDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal stats")
}

func TestGetGitStatsSummary(t *testing.T) {
	tmpDir := t.TempDir()

	// Create and save test stats
	stats := &GitStats{
		TotalCommits: 42,
		TotalAuthors: 3,
		LinesAdded:   1500,
		LinesDeleted: 200,
	}

	err := saveGitStats(tmpDir, stats)
	require.NoError(t, err)

	// Get summary
	summary, err := GetGitStatsSummary(tmpDir)
	require.NoError(t, err)

	assert.Contains(t, summary, "42 commits")
	assert.Contains(t, summary, "3 authors")
	assert.Contains(t, summary, "+1500")
	assert.Contains(t, summary, "-200")
}

func TestGitStatsJSONSerialization(t *testing.T) {
	stats := &GitStats{
		Repository:   "https://github.com/test/repo",
		Path:         "/test/path",
		FetchedAt:    time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC),
		TotalCommits: 100,
		TotalAuthors: 5,
		LinesAdded:   1000,
		LinesDeleted: 500,
		LinesChanged: 1500,
		Authors: []AuthorStats{
			{
				Name:         "Author One",
				Email:        "one@example.com",
				Commits:      60,
				LinesAdded:   600,
				LinesDeleted: 300,
				ActiveDays:   10,
			},
		},
		Branches: []BranchStats{
			{
				Name:      "main",
				Hash:      "abc123",
				UpdatedAt: time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC),
				Age:       24 * time.Hour,
				IsActive:  true,
			},
		},
		CommitsByWeekday: map[string]int{
			"Monday":  10,
			"Tuesday": 15,
		},
	}

	// Serialize to JSON
	data, err := json.MarshalIndent(stats, "", "  ")
	require.NoError(t, err)

	// Deserialize back
	var loaded GitStats
	require.NoError(t, json.Unmarshal(data, &loaded))

	// Verify
	assert.Equal(t, stats.Repository, loaded.Repository)
	assert.Equal(t, stats.TotalCommits, loaded.TotalCommits)
	assert.Len(t, loaded.Authors, 1)
	assert.Equal(t, "Author One", loaded.Authors[0].Name)
	assert.Len(t, loaded.Branches, 1)
	assert.Equal(t, "main", loaded.Branches[0].Name)
	assert.True(t, loaded.Branches[0].IsActive)
	assert.Equal(t, 10, loaded.CommitsByWeekday["Monday"])
}

func TestBranchStats_AgeDuration(t *testing.T) {
	branch := BranchStats{
		Name:      "feature-branch",
		Hash:      "def456",
		UpdatedAt: time.Now().Add(-48 * time.Hour),
		Age:       48 * time.Hour,
		IsActive:  false,
	}

	// Verify age is stored as duration
	assert.Equal(t, 48*time.Hour, branch.Age)

	// Serialize and deserialize
	data, err := json.Marshal(branch)
	require.NoError(t, err)

	var loaded BranchStats
	require.NoError(t, json.Unmarshal(data, &loaded))

	assert.Equal(t, branch.Age, loaded.Age)
}

func TestAuthorStats(t *testing.T) {
	author := AuthorStats{
		Name:         "Test Author",
		Email:        "test@example.com",
		Commits:      100,
		LinesAdded:   5000,
		LinesDeleted: 1000,
		LinesChanged: 6000,
		FilesChanged: 50,
		FirstCommit:  time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		LastCommit:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		ActiveDays:   30,
	}

	// Serialize
	data, err := json.Marshal(author)
	require.NoError(t, err)

	// Deserialize
	var loaded AuthorStats
	require.NoError(t, json.Unmarshal(data, &loaded))

	assert.Equal(t, author.Name, loaded.Name)
	assert.Equal(t, author.Email, loaded.Email)
	assert.Equal(t, author.Commits, loaded.Commits)
	assert.Equal(t, author.LinesAdded, loaded.LinesAdded)
	assert.Equal(t, author.FilesChanged, loaded.FilesChanged)
	assert.Equal(t, author.ActiveDays, loaded.ActiveDays)
}

func TestContributorStats(t *testing.T) {
	contributor := ContributorStats{
		Name:    "Contributor",
		Email:   "contrib@example.com",
		Commits: 25,
		Since:   time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC),
	}

	data, err := json.Marshal(contributor)
	require.NoError(t, err)

	var loaded ContributorStats
	require.NoError(t, json.Unmarshal(data, &loaded))

	assert.Equal(t, contributor.Name, loaded.Name)
	assert.Equal(t, contributor.Commits, loaded.Commits)
}
