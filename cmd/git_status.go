package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/inovacc/clonr/internal/git"
	"github.com/spf13/cobra"
)

var gitStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the working tree status",
	Long: `Displays paths that have differences between the index file and the
current HEAD commit, paths that have differences between the working
tree and the index file, and paths in the working tree that are not
tracked by Git.

Examples:
  clonr git status
  clonr git status --short
  clonr git status --porcelain`,
	RunE: runGitStatus,
}

func init() {
	gitCmd.AddCommand(gitStatusCmd)
	gitStatusCmd.Flags().BoolP("short", "s", false, "Give output in short format")
	gitStatusCmd.Flags().Bool("porcelain", false, "Give output in machine-readable format")
	gitStatusCmd.Flags().BoolP("branch", "b", false, "Show branch info in short format")
}

func runGitStatus(cmd *cobra.Command, _ []string) error {
	client := git.NewClient()
	ctx := context.Background()

	if !client.IsRepository(ctx) {
		return fmt.Errorf("not a git repository")
	}

	short, _ := cmd.Flags().GetBool("short")
	porcelain, _ := cmd.Flags().GetBool("porcelain")
	branch, _ := cmd.Flags().GetBool("branch")

	if porcelain {
		output, err := client.StatusPorcelain(ctx)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprint(os.Stdout, output)
		return nil
	}

	if short {
		output, err := client.Status(ctx)
		if err != nil {
			return err
		}
		if output == "" {
			_, _ = fmt.Fprintln(os.Stdout, "nothing to commit, working tree clean")
			return nil
		}
		_, _ = fmt.Fprint(os.Stdout, output)
		return nil
	}

	// Full status with branch info
	currentBranch, err := client.CurrentBranch(ctx)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(os.Stdout, "On branch %s\n", currentBranch)

	if branch {
		// Get detailed branch info
		branches, err := client.ListBranchesDetailed(ctx, false)
		if err == nil {
			for _, b := range branches {
				if b.Current && b.Upstream != "" {
					_, _ = fmt.Fprintf(os.Stdout, "Your branch is tracking '%s'\n", b.Upstream)
					break
				}
			}
		}
	}

	_, _ = fmt.Fprintln(os.Stdout, "")

	output, err := client.Status(ctx)
	if err != nil {
		return err
	}

	if output == "" {
		_, _ = fmt.Fprintln(os.Stdout, "nothing to commit, working tree clean")
		return nil
	}

	// Parse and display status in a more readable format
	lines := strings.Split(strings.TrimSpace(output), "\n")

	var staged, unstaged, untracked []string
	for _, line := range lines {
		if len(line) < 3 {
			continue
		}
		indexStatus := line[0]
		workTreeStatus := line[1]
		path := strings.TrimSpace(line[3:])

		switch {
		case indexStatus != ' ' && indexStatus != '?':
			staged = append(staged, fmt.Sprintf("  %c  %s", indexStatus, path))
		case workTreeStatus != ' ' && workTreeStatus != '?':
			unstaged = append(unstaged, fmt.Sprintf("  %c  %s", workTreeStatus, path))
		case indexStatus == '?' && workTreeStatus == '?':
			untracked = append(untracked, fmt.Sprintf("      %s", path))
		}
	}

	if len(staged) > 0 {
		_, _ = fmt.Fprintln(os.Stdout, "Changes to be committed:")
		_, _ = fmt.Fprintln(os.Stdout, "  (use \"git restore --staged <file>...\" to unstage)")
		for _, s := range staged {
			_, _ = fmt.Fprintln(os.Stdout, okStyle.Render(s))
		}
		_, _ = fmt.Fprintln(os.Stdout, "")
	}

	if len(unstaged) > 0 {
		_, _ = fmt.Fprintln(os.Stdout, "Changes not staged for commit:")
		_, _ = fmt.Fprintln(os.Stdout, "  (use \"git add <file>...\" to update what will be committed)")
		for _, s := range unstaged {
			_, _ = fmt.Fprintln(os.Stdout, errStyle.Render(s))
		}
		_, _ = fmt.Fprintln(os.Stdout, "")
	}

	if len(untracked) > 0 {
		_, _ = fmt.Fprintln(os.Stdout, "Untracked files:")
		_, _ = fmt.Fprintln(os.Stdout, "  (use \"git add <file>...\" to include in what will be committed)")
		for _, s := range untracked {
			_, _ = fmt.Fprintln(os.Stdout, errStyle.Render(s))
		}
		_, _ = fmt.Fprintln(os.Stdout, "")
	}

	return nil
}
