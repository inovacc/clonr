package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/inovacc/clonr/internal/git"
	"github.com/spf13/cobra"
)

var gitLogCmd = &cobra.Command{
	Use:   "log",
	Short: "Show commit logs",
	Long: `Show the commit logs with optional filtering and formatting.

Examples:
  clonr git log
  clonr git log --limit 10
  clonr git log --oneline
  clonr git log --author "John"
  clonr git log --since "2024-01-01"
  clonr git log --json`,
	RunE: runGitLog,
}

func init() {
	gitCmd.AddCommand(gitLogCmd)
	gitLogCmd.Flags().IntP("limit", "n", 10, "Limit the number of commits")
	gitLogCmd.Flags().Bool("oneline", false, "Show each commit on a single line")
	gitLogCmd.Flags().Bool("all", false, "Show commits from all branches")
	gitLogCmd.Flags().String("author", "", "Filter by author")
	gitLogCmd.Flags().String("since", "", "Show commits more recent than date")
	gitLogCmd.Flags().String("until", "", "Show commits older than date")
	gitLogCmd.Flags().String("grep", "", "Filter by commit message")
	gitLogCmd.Flags().Bool("json", false, "Output as JSON")
}

func runGitLog(cmd *cobra.Command, _ []string) error {
	client := git.NewClient()
	ctx := context.Background()

	if !client.IsRepository(ctx) {
		return fmt.Errorf("not a git repository")
	}

	limit, _ := cmd.Flags().GetInt("limit")
	oneline, _ := cmd.Flags().GetBool("oneline")
	all, _ := cmd.Flags().GetBool("all")
	author, _ := cmd.Flags().GetString("author")
	since, _ := cmd.Flags().GetString("since")
	until, _ := cmd.Flags().GetString("until")
	grep, _ := cmd.Flags().GetString("grep")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	if oneline && !jsonOutput {
		output, err := client.LogOneline(ctx, limit)
		if err != nil {
			return err
		}

		if output == "" {
			_, _ = fmt.Fprintln(os.Stdout, "No commits found")
			return nil
		}

		_, _ = fmt.Fprintln(os.Stdout, output)

		return nil
	}

	opts := git.LogOptions{
		Limit:  limit,
		All:    all,
		Author: author,
		Since:  since,
		Until:  until,
		Grep:   grep,
	}

	commits, err := client.Log(ctx, opts)
	if err != nil {
		return err
	}

	if len(commits) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "No commits found")
		return nil
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")

		return enc.Encode(commits)
	}

	// Display commits in a readable format
	for i, commit := range commits {
		_, _ = fmt.Fprintf(os.Stdout, "%s %s\n",
			warnStyle.Render(commit.ShortSHA),
			commit.Subject,
		)
		_, _ = fmt.Fprintf(os.Stdout, "  %s <%s>\n",
			dimStyle.Render(commit.Author),
			dimStyle.Render(commit.Email),
		)

		_, _ = fmt.Fprintf(os.Stdout, "  %s\n",
			dimStyle.Render(commit.Date),
		)
		if i < len(commits)-1 {
			_, _ = fmt.Fprintln(os.Stdout, "")
		}
	}

	return nil
}
