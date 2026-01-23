package core

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestFetchIssuesWithData(t *testing.T) {
	token := GetGitHubToken()
	if token == "" {
		t.Skip("No GitHub token available, skipping test")
	}

	testPath := filepath.Join(os.TempDir(), "test-issues-gops")
	testURL := "https://github.com/google/gops"

	// Clean up
	defer func() {
		_ = os.RemoveAll(testPath)
	}()

	// Clone manually with git
	if _, err := os.Stat(testPath); os.IsNotExist(err) {
		cmd := exec.Command("git", "clone", "--depth", "1", testURL, testPath)
		if err := cmd.Run(); err != nil {
			t.Fatalf("Clone failed: %v", err)
		}
	}

	// Fetch issues
	err := FetchAndSaveIssues(testURL, testPath, FetchIssuesOptions{
		Token: token,
	})
	if err != nil {
		t.Fatalf("Failed to fetch issues: %v", err)
	}

	// Check if issues file was created
	issuesPath := filepath.Join(testPath, ".clonr", "issues.json")
	info, err := os.Stat(issuesPath)
	if err != nil {
		t.Fatalf("Issues file not created: %v", err)
	}

	t.Logf("Issues file created: %s (%d bytes)", issuesPath, info.Size())

	// Read and show stats
	data, _ := os.ReadFile(issuesPath)
	previewLen := 800
	if len(data) < previewLen {
		previewLen = len(data)
	}
	t.Logf("Content preview (first %d chars):\n%s", previewLen, string(data[:previewLen]))
}
