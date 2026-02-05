package cmd

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/inovacc/clonr/internal/core"
	"github.com/spf13/cobra"
)

var ghCmd = &cobra.Command{
	Use:   "gh",
	Short: "GitHub operations for repositories",
	Long: `Interact with GitHub issues, PRs, actions, releases, and contributors.

Available Commands:
  issues        Manage GitHub issues (list, create, close)
  pr            Check pull request status
  actions       Check GitHub Actions workflow status
  release       Manage GitHub releases (create, download)
  contributors  View contributors and their activity journey

Repository Detection:
  Commands auto-detect the repository from the current directory,
  or you can specify it explicitly as owner/repo.

Authentication:
  Uses GitHub token from (in priority order):
  1. --token flag
  2. --profile flag (clonr profile token)
  3. GITHUB_TOKEN environment variable
  4. GH_TOKEN environment variable
  5. Active clonr profile token
  6. gh CLI authentication`,
}

func init() {
	rootCmd.AddCommand(ghCmd)
}

// addGHCommonFlags adds flags common to all gh subcommands
func addGHCommonFlags(cmd *cobra.Command) {
	cmd.Flags().String("token", "", "GitHub token (default: auto-detect)")
	cmd.Flags().String("profile", "", "Use token from specified profile")
	cmd.Flags().String("repo", "", "Repository (owner/repo)")
	cmd.Flags().Bool("json", false, "Output as JSON")
}

// GHFlags holds common flags for all gh subcommands
type GHFlags struct {
	Token   string
	Profile string
	Repo    string
	JSON    bool
}

// extractGHFlags extracts common flags from a cobra command
func extractGHFlags(cmd *cobra.Command) GHFlags {
	token, _ := cmd.Flags().GetString("token")
	profile, _ := cmd.Flags().GetString("profile")
	repo, _ := cmd.Flags().GetString("repo")
	jsonOut, _ := cmd.Flags().GetBool("json")

	return GHFlags{
		Token:   token,
		Profile: profile,
		Repo:    repo,
		JSON:    jsonOut,
	}
}

// outputJSON encodes data as indented JSON to stdout
func outputJSON(data any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")

	return enc.Encode(data)
}

// newGHLogger creates a logger appropriate for gh commands
// Uses JSON handler when JSON output is enabled, text otherwise
func newGHLogger(jsonOutput bool) *slog.Logger {
	opts := &slog.HandlerOptions{Level: slog.LevelWarn}
	if jsonOutput {
		return slog.New(slog.NewJSONHandler(os.Stderr, opts))
	}

	return slog.New(slog.NewTextHandler(os.Stderr, opts))
}

// detectRepo detects repository from args and flags
// Returns owner, repo, or error with usage hint
func detectRepo(args []string, repoFlag, usageHint string) (owner, repo string, err error) {
	var repoArg string
	if len(args) > 0 {
		repoArg = args[0]
	}

	owner, repo, err = core.DetectRepository(repoArg, repoFlag)
	if err != nil {
		return "", "", fmt.Errorf("could not determine repository: %w\n\n%s", err, usageHint)
	}

	return owner, repo, nil
}

// formatAge formats a time as a human-readable age string (e.g., "2 hours ago")
func formatAge(t time.Time) string {
	d := time.Since(t)

	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		mins := int(d.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}

		return fmt.Sprintf("%d minutes ago", mins)
	case d < 24*time.Hour:
		hours := int(d.Hours())
		if hours == 1 {
			return "1 hour ago"
		}

		return fmt.Sprintf("%d hours ago", hours)
	case d < 30*24*time.Hour:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}

		return fmt.Sprintf("%d days ago", days)
	case d < 365*24*time.Hour:
		months := int(d.Hours() / 24 / 30)
		if months == 1 {
			return "1 month ago"
		}

		return fmt.Sprintf("%d months ago", months)
	default:
		years := int(d.Hours() / 24 / 365)
		if years == 1 {
			return "1 year ago"
		}

		return fmt.Sprintf("%d years ago", years)
	}
}

// formatShortDuration formats a duration as a compact string (e.g., "2m 30s")
// Use for short durations like workflow/job execution times
func formatShortDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}

	if d < time.Hour {
		mins := int(d.Minutes())
		secs := int(d.Seconds()) % 60

		return fmt.Sprintf("%dm %ds", mins, secs)
	}

	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60

	return fmt.Sprintf("%dh %dm", hours, mins)
}

// formatFileSize formats bytes as a human-readable size (e.g., "1.5 MB")
func formatFileSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// truncateStr truncates a string to maxLen, padding or adding ellipsis as needed
func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s + strings.Repeat(" ", maxLen-len(s))
	}

	return s[:maxLen-3] + "..."
}
