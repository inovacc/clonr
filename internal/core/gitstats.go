package core

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/inovacc/clonr/internal/common"
	gitnerds "github.com/inovacc/git-nerds"
)

// GitStats contains comprehensive repository statistics from git-nerds
type GitStats struct {
	Repository   string    `json:"repository"`
	Path         string    `json:"path"`
	FetchedAt    time.Time `json:"fetched_at"`
	TotalCommits int       `json:"total_commits"`
	TotalAuthors int       `json:"total_authors"`
	LinesAdded   int       `json:"lines_added"`
	LinesDeleted int       `json:"lines_deleted"`
	LinesChanged int       `json:"lines_changed"`

	// Temporal boundaries
	FirstCommitAt time.Time `json:"first_commit_at,omitzero"`
	LastCommitAt  time.Time `json:"last_commit_at,omitzero"`

	// Detailed statistics
	Authors      []AuthorStats      `json:"authors,omitempty"`
	Contributors []ContributorStats `json:"contributors,omitempty"`
	Branches     []BranchStats      `json:"branches,omitempty"`

	// Temporal analysis
	CommitsByDay      map[string]int `json:"commits_by_day,omitempty"`
	CommitsByMonth    map[string]int `json:"commits_by_month,omitempty"`
	CommitsByYear     map[string]int `json:"commits_by_year,omitempty"`
	CommitsByWeekday  map[string]int `json:"commits_by_weekday,omitempty"`
	CommitsByHour     map[int]int    `json:"commits_by_hour,omitempty"`
	CommitsByTimezone map[string]int `json:"commits_by_timezone,omitempty"`
}

// AuthorStats contains statistics for a single author
type AuthorStats struct {
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	Commits      int       `json:"commits"`
	LinesAdded   int       `json:"lines_added"`
	LinesDeleted int       `json:"lines_deleted"`
	LinesChanged int       `json:"lines_changed"`
	FilesChanged int       `json:"files_changed"`
	FirstCommit  time.Time `json:"first_commit"`
	LastCommit   time.Time `json:"last_commit"`
	ActiveDays   int       `json:"active_days"`
}

// ContributorStats contains simplified contributor information
type ContributorStats struct {
	Name    string    `json:"name"`
	Email   string    `json:"email"`
	Commits int       `json:"commits"`
	Since   time.Time `json:"since"`
}

// BranchStats contains branch information
type BranchStats struct {
	Name      string        `json:"name"`
	Hash      string        `json:"hash"`
	UpdatedAt time.Time     `json:"updated_at"`
	Age       time.Duration `json:"age"`
	IsActive  bool          `json:"is_active"`
}

// FetchGitStatsOptions configures the git stats fetching behavior
type FetchGitStatsOptions struct {
	Logger          *slog.Logger
	IncludeTemporal bool // Include temporal analysis (commits by day/month/etc)
	IncludeBranches bool // Include branch information
	Since           time.Time
	Until           time.Time
}

// FetchAndSaveGitStats gathers repository statistics using git-nerds and saves them
func FetchAndSaveGitStats(repoURL, repoPath string, opts FetchGitStatsOptions) error {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	logger.Info("gathering git statistics", slog.String("path", repoPath))

	// Open the repository with git-nerds
	gnOpts := gitnerds.DefaultOptions()
	if !opts.Since.IsZero() {
		gnOpts.Since = opts.Since
	}

	if !opts.Until.IsZero() {
		gnOpts.Until = opts.Until
	}

	repo, err := gitnerds.Open(repoPath, gnOpts)
	if err != nil {
		logger.Warn("failed to open repository with git-nerds", slog.String("path", repoPath), slog.String("error", err.Error()))

		return nil // Don't fail the mirror operation
	}

	stats := &GitStats{
		Repository: common.SanitizeGitURL(repoURL),
		Path:       repoPath,
		FetchedAt:  time.Now(),
	}

	// Get detailed stats
	detailedStats, err := repo.DetailedStats()
	if err != nil {
		logger.Warn("failed to get detailed stats", slog.String("error", err.Error()))
	} else {
		stats.TotalCommits = detailedStats.TotalCommits
		stats.TotalAuthors = detailedStats.TotalAuthors
		stats.LinesAdded = detailedStats.LinesAdded
		stats.LinesDeleted = detailedStats.LinesDeleted
		stats.LinesChanged = detailedStats.LinesChanged
		stats.FirstCommitAt = detailedStats.FirstCommitAt
		stats.LastCommitAt = detailedStats.LastCommitAt

		// Convert authors
		stats.Authors = make([]AuthorStats, len(detailedStats.Authors))
		for i, a := range detailedStats.Authors {
			stats.Authors[i] = AuthorStats{
				Name:         a.Name,
				Email:        a.Email,
				Commits:      a.Commits,
				LinesAdded:   a.LinesAdded,
				LinesDeleted: a.LinesDeleted,
				LinesChanged: a.LinesChanged,
				FilesChanged: a.FilesChanged,
				FirstCommit:  a.FirstCommit,
				LastCommit:   a.LastCommit,
				ActiveDays:   a.ActiveDays,
			}
		}

		// Convert branches if available
		if opts.IncludeBranches && len(detailedStats.Branches) > 0 {
			stats.Branches = make([]BranchStats, len(detailedStats.Branches))
			for i, b := range detailedStats.Branches {
				stats.Branches[i] = BranchStats{
					Name:      b.Name,
					Hash:      b.Hash,
					UpdatedAt: b.UpdatedAt,
					Age:       b.Age,
					IsActive:  b.IsActive,
				}
			}
		}
	}

	// Get contributors
	contributors, err := repo.Contributors()
	if err != nil {
		logger.Debug("failed to get contributors", slog.String("error", err.Error()))
	} else {
		stats.Contributors = make([]ContributorStats, len(contributors))
		for i, c := range contributors {
			stats.Contributors[i] = ContributorStats{
				Name:    c.Name,
				Email:   c.Email,
				Commits: c.Commits,
				Since:   c.Since,
			}
		}
	}

	// Get temporal analysis if requested
	if opts.IncludeTemporal {
		if data, err := repo.CommitsByDay(); err == nil {
			stats.CommitsByDay = data
		}

		if data, err := repo.CommitsByMonth(); err == nil {
			stats.CommitsByMonth = data
		}

		if data, err := repo.CommitsByYear(); err == nil {
			stats.CommitsByYear = data
		}

		if data, err := repo.CommitsByWeekday(); err == nil {
			stats.CommitsByWeekday = data
		}

		if data, err := repo.CommitsByHour(); err == nil {
			stats.CommitsByHour = data
		}

		if data, err := repo.CommitsByTimezone(); err == nil {
			stats.CommitsByTimezone = data
		}
	}

	// Get branches by date if requested
	if opts.IncludeBranches && len(stats.Branches) == 0 {
		branches, err := repo.BranchesByDate()
		if err == nil {
			stats.Branches = make([]BranchStats, len(branches))
			for i, b := range branches {
				stats.Branches[i] = BranchStats{
					Name:      b.Name,
					Hash:      b.Hash,
					UpdatedAt: b.UpdatedAt,
					Age:       b.Age,
					IsActive:  b.IsActive,
				}
			}
		}
	}

	// Save to file
	if err := saveGitStats(repoPath, stats); err != nil {
		logger.Warn("failed to save git stats", slog.String("path", repoPath), slog.String("error", err.Error()))

		return nil // Don't fail the mirror operation
	}

	logger.Info("saved git statistics",
		slog.String("path", repoPath), slog.Int("commits", stats.TotalCommits),
		slog.Int("authors", stats.TotalAuthors), slog.Int("contributors", len(stats.Contributors)))

	return nil
}

// saveGitStats saves statistics to a JSON file in the repository
func saveGitStats(repoPath string, stats *GitStats) error {
	// Create .clonr directory in the repo
	clonrDir := filepath.Join(repoPath, ".clonr")
	if err := os.MkdirAll(clonrDir, 0755); err != nil {
		return fmt.Errorf("failed to create .clonr directory: %w", err)
	}

	// Write stats to the JSON file
	statsPath := filepath.Join(clonrDir, "stats.json")

	jsonData, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal stats: %w", err)
	}

	if err := os.WriteFile(statsPath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write stats file: %w", err)
	}

	return nil
}

// LoadGitStats loads previously saved git statistics from a repository
func LoadGitStats(repoPath string) (*GitStats, error) {
	statsPath := filepath.Join(repoPath, ".clonr", "stats.json")

	data, err := os.ReadFile(statsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read stats file: %w", err)
	}

	var stats GitStats
	if err := json.Unmarshal(data, &stats); err != nil {
		return nil, fmt.Errorf("failed to unmarshal stats: %w", err)
	}

	return &stats, nil
}

// GitStatsExists checks if git stats have been gathered for a repository
func GitStatsExists(repoPath string) bool {
	statsPath := filepath.Join(repoPath, ".clonr", "stats.json")
	_, err := os.Stat(statsPath)

	return err == nil
}

// RefreshGitStats updates the git statistics for a repository
func RefreshGitStats(repoURL, repoPath string, opts FetchGitStatsOptions) error {
	return FetchAndSaveGitStats(repoURL, repoPath, opts)
}

// GetGitStatsSummary returns a brief summary of the git statistics
func GetGitStatsSummary(repoPath string) (string, error) {
	stats, err := LoadGitStats(repoPath)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%d commits by %d authors | +%d -%d lines", stats.TotalCommits, stats.TotalAuthors, stats.LinesAdded, stats.LinesDeleted), nil
}
