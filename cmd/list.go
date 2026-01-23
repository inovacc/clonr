package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/inovacc/clonr/internal/cli"
	"github.com/inovacc/clonr/internal/core"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Interactively list all repositories",
	Long: `Display all managed repositories in an interactive list.

Use arrow keys to navigate and Enter to select actions.

Sorting Options:
  --sort name     Sort alphabetically by URL
  --sort cloned   Sort by clone date (newest first)
  --sort updated  Sort by last update date (newest first)
  --sort commits  Sort by total commit count (highest first)
  --sort recent   Sort by recent commits in last 30 days (highest first)
  --sort changes  Sort by total changes (additions + deletions)

Examples:
  clonr list                          # Interactive list
  clonr list --sort commits --stats   # Sort by commits with stats
  clonr list --json --stats           # JSON output with stats`,
	RunE: runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().Bool("favorites", false, "Show only favorite repositories")
	listCmd.Flags().String("sort", "", "Sort by: name, cloned, updated, commits, recent, changes")
	listCmd.Flags().Bool("stats", false, "Include commit statistics (slower)")
	listCmd.Flags().Bool("json", false, "Output as JSON")
}

func runList(cmd *cobra.Command, args []string) error {
	favoritesOnly, _ := cmd.Flags().GetBool("favorites")
	sortBy, _ := cmd.Flags().GetString("sort")
	withStats, _ := cmd.Flags().GetBool("stats")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	// If sorting by commits/recent/changes, we need stats
	if sortBy == "commits" || sortBy == "recent" || sortBy == "changes" {
		withStats = true
	}

	// Non-interactive mode with JSON or sort
	if jsonOutput || sortBy != "" {
		return listReposNonInteractive(favoritesOnly, sortBy, withStats, jsonOutput)
	}

	// Interactive mode
	m, err := cli.NewRepoList(favoritesOnly)
	if err != nil {
		return err
	}

	p := tea.NewProgram(m)
	_, err = p.Run()

	return err
}

func listReposNonInteractive(favoritesOnly bool, sortBy string, withStats, jsonOutput bool) error {
	var sort core.SortBy

	switch sortBy {
	case "name":
		sort = core.SortByName
	case "cloned":
		sort = core.SortByClonedAt
	case "updated":
		sort = core.SortByUpdatedAt
	case "commits":
		sort = core.SortByCommits
	case "recent":
		sort = core.SortByRecentCommits
	case "changes":
		sort = core.SortByChanges
	default:
		sort = core.SortByName
	}

	if !jsonOutput {
		_, _ = fmt.Fprintf(os.Stderr, "Fetching repositories")
		if withStats {
			_, _ = fmt.Fprintf(os.Stderr, " with stats")
		}

		_, _ = fmt.Fprintf(os.Stderr, "...\n")
	}

	repos, err := core.ListReposWithStats(favoritesOnly, sort, withStats)
	if err != nil {
		return fmt.Errorf("failed to list repos: %w", err)
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")

		return enc.Encode(repos)
	}

	// Text output
	if len(repos) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "No repositories found")
		return nil
	}

	_, _ = fmt.Fprintf(os.Stdout, "\nRepositories (%d)\n\n", len(repos))

	for _, r := range repos {
		fav := ""
		if r.Favorite {
			fav = " *"
		}

		_, _ = fmt.Fprintf(os.Stdout, "%s%s\n", r.URL, fav)
		_, _ = fmt.Fprintf(os.Stdout, "  Path: %s\n", r.Path)

		if r.Stats != nil {
			_, _ = fmt.Fprintf(os.Stdout, "  Stats: %s\n", core.FormatRepoStats(r.Stats))

			if !r.Stats.LastCommitDate.IsZero() {
				_, _ = fmt.Fprintf(os.Stdout, "  Last commit: %s - %s\n",
					r.Stats.LastCommitDate.Format("Jan 2, 2006"),
					r.Stats.LastCommitMsg)
			}
		}

		_, _ = fmt.Fprintln(os.Stdout)
	}

	return nil
}
