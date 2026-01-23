package core

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/inovacc/clonr/internal/model"
)

// RepoStats contains commit statistics for a repository
type RepoStats struct {
	TotalCommits   int       `json:"total_commits"`
	RecentCommits  int       `json:"recent_commits"` // Last 30 days
	LastCommitDate time.Time `json:"last_commit_date"`
	LastCommitMsg  string    `json:"last_commit_msg"`
	Additions      int       `json:"additions"`
	Deletions      int       `json:"deletions"`
}

// RepoWithStats combines a repository with its stats
type RepoWithStats struct {
	model.Repository

	Stats *RepoStats `json:"stats,omitempty"`
}

// GetRepoStats returns commit statistics for a repository
func GetRepoStats(repoPath string) (*RepoStats, error) {
	stats := &RepoStats{}

	// Get total commit count
	cmd := exec.Command("git", "-C", repoPath, "rev-list", "--count", "HEAD")

	output, err := cmd.Output()
	if err == nil {
		count, _ := strconv.Atoi(strings.TrimSpace(string(output)))
		stats.TotalCommits = count
	}

	// Get recent commits (last 30 days)
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	cmd = exec.Command("git", "-C", repoPath, "rev-list", "--count", "--since="+thirtyDaysAgo, "HEAD")

	output, err = cmd.Output()
	if err == nil {
		count, _ := strconv.Atoi(strings.TrimSpace(string(output)))
		stats.RecentCommits = count
	}

	// Get last commit info
	cmd = exec.Command("git", "-C", repoPath, "log", "-1", "--format=%H|%s|%ai")

	output, err = cmd.Output()
	if err == nil {
		parts := strings.SplitN(strings.TrimSpace(string(output)), "|", 3)
		if len(parts) >= 3 {
			stats.LastCommitMsg = truncateString(parts[1], 60)

			if t, err := time.Parse("2006-01-02 15:04:05 -0700", parts[2]); err == nil {
				stats.LastCommitDate = t
			}
		}
	}

	// Get total additions/deletions (this can be slow for large repos)
	// Only get stats for the last 100 commits to avoid performance issues
	cmd = exec.Command("git", "-C", repoPath, "log", "--shortstat", "-100", "--format=")

	output, err = cmd.Output()
	if err == nil {
		stats.Additions, stats.Deletions = parseShortstat(string(output))
	}

	return stats, nil
}

// parseShortstat extracts additions and deletions from git shortstat output
func parseShortstat(output string) (additions, deletions int) {
	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse lines like: " 3 files changed, 45 insertions(+), 12 deletions(-)"
		if strings.Contains(line, "insertion") || strings.Contains(line, "deletion") {
			for part := range strings.SplitSeq(line, ",") {
				part = strings.TrimSpace(part)
				if strings.Contains(part, "insertion") {
					if n, err := strconv.Atoi(strings.Fields(part)[0]); err == nil {
						additions += n
					}
				} else if strings.Contains(part, "deletion") {
					if n, err := strconv.Atoi(strings.Fields(part)[0]); err == nil {
						deletions += n
					}
				}
			}
		}
	}

	return additions, deletions
}

// SortBy defines how repositories should be sorted
type SortBy string

const (
	SortByName          SortBy = "name"
	SortByClonedAt      SortBy = "cloned"
	SortByUpdatedAt     SortBy = "updated"
	SortByCommits       SortBy = "commits"
	SortByRecentCommits SortBy = "recent"
	SortByChanges       SortBy = "changes"
)

// ListReposWithStats returns repos with optional stats and sorting
func ListReposWithStats(favoritesOnly bool, sortBy SortBy, withStats bool) ([]RepoWithStats, error) {
	repos, err := ListReposFiltered(favoritesOnly)
	if err != nil {
		return nil, err
	}

	result := make([]RepoWithStats, len(repos))
	for i, repo := range repos {
		result[i] = RepoWithStats{Repository: repo}

		if withStats {
			stats, err := GetRepoStats(repo.Path)
			if err == nil {
				result[i].Stats = stats
			}
		}
	}

	// Sort based on sortBy
	sortRepos(result, sortBy)

	return result, nil
}

// sortRepos sorts the repos based on the given criteria
func sortRepos(repos []RepoWithStats, sortBy SortBy) {
	switch sortBy {
	case SortByName:
		sortByName(repos)
	case SortByClonedAt:
		sortByCloned(repos)
	case SortByUpdatedAt:
		sortByUpdated(repos)
	case SortByCommits:
		sortByCommits(repos)
	case SortByRecentCommits:
		sortByRecentCommits(repos)
	case SortByChanges:
		sortByChanges(repos)
	}
}

func sortByName(repos []RepoWithStats) {
	for i := 0; i < len(repos)-1; i++ {
		for j := i + 1; j < len(repos); j++ {
			if repos[i].URL > repos[j].URL {
				repos[i], repos[j] = repos[j], repos[i]
			}
		}
	}
}

func sortByCloned(repos []RepoWithStats) {
	for i := 0; i < len(repos)-1; i++ {
		for j := i + 1; j < len(repos); j++ {
			if repos[i].ClonedAt.Before(repos[j].ClonedAt) {
				repos[i], repos[j] = repos[j], repos[i]
			}
		}
	}
}

func sortByUpdated(repos []RepoWithStats) {
	for i := 0; i < len(repos)-1; i++ {
		for j := i + 1; j < len(repos); j++ {
			if repos[i].UpdatedAt.Before(repos[j].UpdatedAt) {
				repos[i], repos[j] = repos[j], repos[i]
			}
		}
	}
}

func sortByCommits(repos []RepoWithStats) {
	for i := 0; i < len(repos)-1; i++ {
		for j := i + 1; j < len(repos); j++ {
			ci, cj := 0, 0
			if repos[i].Stats != nil {
				ci = repos[i].Stats.TotalCommits
			}

			if repos[j].Stats != nil {
				cj = repos[j].Stats.TotalCommits
			}

			if ci < cj {
				repos[i], repos[j] = repos[j], repos[i]
			}
		}
	}
}

func sortByRecentCommits(repos []RepoWithStats) {
	for i := 0; i < len(repos)-1; i++ {
		for j := i + 1; j < len(repos); j++ {
			ci, cj := 0, 0
			if repos[i].Stats != nil {
				ci = repos[i].Stats.RecentCommits
			}

			if repos[j].Stats != nil {
				cj = repos[j].Stats.RecentCommits
			}

			if ci < cj {
				repos[i], repos[j] = repos[j], repos[i]
			}
		}
	}
}

func sortByChanges(repos []RepoWithStats) {
	for i := 0; i < len(repos)-1; i++ {
		for j := i + 1; j < len(repos); j++ {
			ci, cj := 0, 0
			if repos[i].Stats != nil {
				ci = repos[i].Stats.Additions + repos[i].Stats.Deletions
			}

			if repos[j].Stats != nil {
				cj = repos[j].Stats.Additions + repos[j].Stats.Deletions
			}

			if ci < cj {
				repos[i], repos[j] = repos[j], repos[i]
			}
		}
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen-3] + "..."
	}

	return s
}

// FormatRepoStats returns a formatted string of repo stats
func FormatRepoStats(stats *RepoStats) string {
	if stats == nil {
		return ""
	}

	return fmt.Sprintf("%d commits (%d recent) | +%d -%d",
		stats.TotalCommits, stats.RecentCommits,
		stats.Additions, stats.Deletions)
}
