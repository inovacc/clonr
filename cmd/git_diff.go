package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/inovacc/clonr/internal/git"
	"github.com/spf13/cobra"
)

var gitDiffCmd = &cobra.Command{
	Use:   "diff [commit] [-- path]",
	Short: "Show changes between commits, working tree, etc.",
	Long: `Show changes between the working tree and the index or a tree,
changes between the index and a tree, changes between two trees,
or changes between two files on disk.

Examples:
  clonr git diff                    # Show unstaged changes
  clonr git diff --staged           # Show staged changes
  clonr git diff --stat             # Show diffstat
  clonr git diff HEAD~1             # Compare with previous commit
  clonr git diff --name-only        # Show only file names
  clonr git diff -- path/to/file    # Diff specific path`,
	RunE: runGitDiff,
}

func init() {
	gitCmd.AddCommand(gitDiffCmd)
	gitDiffCmd.Flags().Bool("staged", false, "Show staged changes (same as --cached)")
	gitDiffCmd.Flags().Bool("cached", false, "Show staged changes")
	gitDiffCmd.Flags().Bool("stat", false, "Show diffstat instead of full diff")
	gitDiffCmd.Flags().Bool("name-only", false, "Show only names of changed files")
	gitDiffCmd.Flags().Bool("name-status", false, "Show names and status of changed files")
}

func runGitDiff(cmd *cobra.Command, args []string) error {
	client := git.NewClient()
	ctx := context.Background()

	if !client.IsRepository(ctx) {
		return fmt.Errorf("not a git repository")
	}

	staged, _ := cmd.Flags().GetBool("staged")
	cached, _ := cmd.Flags().GetBool("cached")
	stat, _ := cmd.Flags().GetBool("stat")
	nameOnly, _ := cmd.Flags().GetBool("name-only")
	nameStatus, _ := cmd.Flags().GetBool("name-status")

	opts := git.DiffOptions{
		Staged:     staged || cached,
		Stat:       stat,
		NameOnly:   nameOnly,
		NameStatus: nameStatus,
	}

	// Parse args for commit and path
	for i, arg := range args {
		if arg == "--" {
			if i+1 < len(args) {
				opts.Path = args[i+1]
			}
			break
		}
		if opts.Commit == "" {
			opts.Commit = arg
		}
	}

	output, err := client.Diff(ctx, opts)
	if err != nil {
		return err
	}

	if output == "" {
		if opts.Staged {
			_, _ = fmt.Fprintln(os.Stdout, "No staged changes")
		} else {
			_, _ = fmt.Fprintln(os.Stdout, "No changes")
		}
		return nil
	}

	_, _ = fmt.Fprint(os.Stdout, output)

	return nil
}
