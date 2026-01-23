package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/inovacc/clonr/internal/core"
	"github.com/spf13/cobra"
)

var statsCmd = &cobra.Command{
	Use:   "stats [path]",
	Short: "Show git statistics for a repository",
	Long: `Display git statistics gathered by git-nerds for a repository.

Statistics include:
  - Commit counts and temporal distribution
  - Author/contributor information
  - Lines added/deleted
  - Branch information

If no path is provided, uses the current directory.

Examples:
  clonr stats                    # Stats for current directory
  clonr stats /path/to/repo      # Stats for specific repo
  clonr stats --json             # Output as JSON
  clonr stats --refresh          # Refresh statistics before displaying`,
	RunE: runStats,
}

func init() {
	rootCmd.AddCommand(statsCmd)
	statsCmd.Flags().Bool("json", false, "Output as JSON")
	statsCmd.Flags().Bool("refresh", false, "Refresh statistics before displaying")
	statsCmd.Flags().Bool("temporal", false, "Include temporal analysis in output")
}

func runStats(cmd *cobra.Command, args []string) error {
	jsonOutput, _ := cmd.Flags().GetBool("json")
	refresh, _ := cmd.Flags().GetBool("refresh")
	showTemporal, _ := cmd.Flags().GetBool("temporal")

	// Determine repo path
	repoPath := "."
	if len(args) > 0 {
		repoPath = args[0]
	}

	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Check if it's a git repository
	gitDir := filepath.Join(absPath, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return fmt.Errorf("not a git repository: %s", absPath)
	}

	// Refresh stats if requested or if they don't exist
	if refresh || !core.GitStatsExists(absPath) {
		// Try to determine the repo URL from git config
		repoURL := getRepoURLFromPath(absPath)

		_, _ = fmt.Fprintf(os.Stderr, "Gathering statistics for %s...\n", absPath)

		if err := core.FetchAndSaveGitStats(repoURL, absPath, core.FetchGitStatsOptions{
			IncludeTemporal: true,
			IncludeBranches: true,
		}); err != nil {
			return fmt.Errorf("failed to gather statistics: %w", err)
		}
	}

	// Load stats
	stats, err := core.LoadGitStats(absPath)
	if err != nil {
		return fmt.Errorf("failed to load statistics: %w", err)
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")

		return enc.Encode(stats)
	}

	// Text output
	printStatsText(stats, showTemporal)

	return nil
}

func printStatsText(stats *core.GitStats, showTemporal bool) {
	_, _ = fmt.Fprintln(os.Stdout)
	_, _ = fmt.Fprintln(os.Stdout, "Repository Statistics")
	_, _ = fmt.Fprintln(os.Stdout, "=====================")
	_, _ = fmt.Fprintln(os.Stdout)

	_, _ = fmt.Fprintf(os.Stdout, "Path: %s\n", stats.Path)
	_, _ = fmt.Fprintf(os.Stdout, "URL:  %s\n", stats.Repository)
	_, _ = fmt.Fprintf(os.Stdout, "Gathered: %s\n\n", stats.FetchedAt.Format("Jan 2, 2006 15:04:05"))

	_, _ = fmt.Fprintln(os.Stdout, "Overview")
	_, _ = fmt.Fprintln(os.Stdout, "--------")
	_, _ = fmt.Fprintf(os.Stdout, "  Total Commits:  %d\n", stats.TotalCommits)
	_, _ = fmt.Fprintf(os.Stdout, "  Total Authors:  %d\n", stats.TotalAuthors)
	_, _ = fmt.Fprintf(os.Stdout, "  Lines Added:    %d\n", stats.LinesAdded)
	_, _ = fmt.Fprintf(os.Stdout, "  Lines Deleted:  %d\n", stats.LinesDeleted)
	_, _ = fmt.Fprintf(os.Stdout, "  Lines Changed:  %d\n", stats.LinesChanged)

	if !stats.FirstCommitAt.IsZero() {
		_, _ = fmt.Fprintf(os.Stdout, "  First Commit:   %s\n", stats.FirstCommitAt.Format("Jan 2, 2006"))
	}

	if !stats.LastCommitAt.IsZero() {
		_, _ = fmt.Fprintf(os.Stdout, "  Last Commit:    %s\n", stats.LastCommitAt.Format("Jan 2, 2006"))
	}

	_, _ = fmt.Fprintln(os.Stdout)

	// Contributors summary
	if len(stats.Contributors) > 0 {
		_, _ = fmt.Fprintln(os.Stdout, "Top Contributors")
		_, _ = fmt.Fprintln(os.Stdout, "----------------")

		limit := min(len(stats.Contributors), 10)

		for i := range limit {
			c := stats.Contributors[i]
			_, _ = fmt.Fprintf(os.Stdout, "  %d. %s <%s> - %d commits\n", i+1, c.Name, c.Email, c.Commits)
		}

		if len(stats.Contributors) > 10 {
			_, _ = fmt.Fprintf(os.Stdout, "  ... and %d more\n", len(stats.Contributors)-10)
		}

		_, _ = fmt.Fprintln(os.Stdout)
	}

	// Branches summary
	if len(stats.Branches) > 0 {
		_, _ = fmt.Fprintf(os.Stdout, "Branches (%d total)\n", len(stats.Branches))
		_, _ = fmt.Fprintln(os.Stdout, "------------------")

		limit := min(len(stats.Branches), 5)

		for i := range limit {
			b := stats.Branches[i]

			status := ""
			if b.IsActive {
				status = " [active]"
			}

			_, _ = fmt.Fprintf(os.Stdout, "  - %s%s\n", b.Name, status)
		}

		if len(stats.Branches) > 5 {
			_, _ = fmt.Fprintf(os.Stdout, "  ... and %d more\n", len(stats.Branches)-5)
		}

		_, _ = fmt.Fprintln(os.Stdout)
	}

	// Temporal analysis
	if showTemporal {
		if len(stats.CommitsByWeekday) > 0 {
			_, _ = fmt.Fprintln(os.Stdout, "Commits by Weekday")
			_, _ = fmt.Fprintln(os.Stdout, "------------------")

			weekdays := []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
			for _, day := range weekdays {
				if count, ok := stats.CommitsByWeekday[day]; ok {
					_, _ = fmt.Fprintf(os.Stdout, "  %-10s %d\n", day, count)
				}
			}

			_, _ = fmt.Fprintln(os.Stdout)
		}

		if len(stats.CommitsByHour) > 0 {
			_, _ = fmt.Fprintln(os.Stdout, "Commits by Hour (Top 5)")
			_, _ = fmt.Fprintln(os.Stdout, "-----------------------")

			// Find top 5 hours
			type hourCount struct {
				hour  int
				count int
			}

			hours := make([]hourCount, 0, len(stats.CommitsByHour))
			for h, c := range stats.CommitsByHour {
				hours = append(hours, hourCount{h, c})
			}

			// Sort by count descending (simple bubble sort for small data)
			for i := 0; i < len(hours)-1; i++ {
				for j := i + 1; j < len(hours); j++ {
					if hours[i].count < hours[j].count {
						hours[i], hours[j] = hours[j], hours[i]
					}
				}
			}

			limit := min(len(hours), 5)

			for i := range limit {
				_, _ = fmt.Fprintf(os.Stdout, "  %02d:00  %d commits\n", hours[i].hour, hours[i].count)
			}

			_, _ = fmt.Fprintln(os.Stdout)
		}
	}
}

// getRepoURLFromPath attempts to get the repository URL from git config
func getRepoURLFromPath(path string) string {
	configPath := filepath.Join(path, ".git", "config")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return ""
	}

	lines := strings.Split(string(data), "\n")
	inRemoteOrigin := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "[remote \"origin\"]" {
			inRemoteOrigin = true
			continue
		}

		if inRemoteOrigin {
			if rawURL, found := strings.CutPrefix(trimmed, "url = "); found {
				return sanitizeGitURL(rawURL)
			}

			if strings.HasPrefix(trimmed, "[") {
				break
			}
		}
	}

	return ""
}

// sanitizeGitURL removes credentials from a git URL
func sanitizeGitURL(rawURL string) string {
	// Handle URLs with embedded credentials like:
	// https://user:token@github.com/owner/repo.git
	if strings.Contains(rawURL, "@") && strings.Contains(rawURL, "://") {
		// Split at ://
		parts := strings.SplitN(rawURL, "://", 2)
		if len(parts) == 2 {
			// Find @ and remove everything before it
			atIdx := strings.Index(parts[1], "@")
			if atIdx != -1 {
				return parts[0] + "://" + parts[1][atIdx+1:]
			}
		}
	}

	return rawURL
}
