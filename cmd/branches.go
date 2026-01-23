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

var branchesCmd = &cobra.Command{
	Use:   "branches [path]",
	Short: "List and manage git branches",
	Long: `List git branches for a repository interactively.

If no path is provided, shows a repository selector first.
Use arrow keys to navigate, Enter to checkout a branch, and / to filter.

Examples:
  clonr branches                    # Select repo then list branches
  clonr branches /path/to/repo      # List branches for specific repo
  clonr branches --all              # Include remote branches
  clonr branches --json             # Output as JSON`,
	RunE: runBranches,
}

func init() {
	rootCmd.AddCommand(branchesCmd)
	branchesCmd.Flags().BoolP("all", "a", false, "Show all branches (local and remote)")
	branchesCmd.Flags().Bool("json", false, "Output branches as JSON")
	branchesCmd.Flags().Bool("current", false, "Show only current branch")
}

func runBranches(cmd *cobra.Command, args []string) error {
	showAll, _ := cmd.Flags().GetBool("all")
	jsonOutput, _ := cmd.Flags().GetBool("json")
	showCurrent, _ := cmd.Flags().GetBool("current")

	var repoPath, repoURL string

	// If path provided, use it directly
	if len(args) > 0 {
		repoPath = args[0]
	} else {
		// Show repository selector
		repoModel, err := cli.NewRepoList(false)
		if err != nil {
			return fmt.Errorf("failed to load repositories: %w", err)
		}

		p := tea.NewProgram(repoModel)

		finalModel, err := p.Run()
		if err != nil {
			return err
		}

		selectedRepo := finalModel.(cli.RepoListModel).GetSelectedRepo()
		if selectedRepo == nil {
			return nil // User cancelled
		}

		repoPath = selectedRepo.Path
		repoURL = selectedRepo.URL
	}

	// Handle --current flag
	if showCurrent {
		current, err := core.GetCurrentBranch(repoPath)
		if err != nil {
			return err
		}

		if jsonOutput {
			result := map[string]string{"current_branch": current}

			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")

			return enc.Encode(result)
		}

		_, _ = fmt.Fprintln(os.Stdout, current)

		return nil
	}

	// Handle --json flag
	if jsonOutput {
		opts := core.BranchListOptions{All: showAll}

		branches, err := core.ListBranches(repoPath, opts)
		if err != nil {
			return err
		}

		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")

		return enc.Encode(branches)
	}

	// Interactive mode
	branchModel, err := cli.NewBranchList(repoPath, repoURL, showAll)
	if err != nil {
		return err
	}

	p := tea.NewProgram(branchModel)

	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	result := finalModel.(cli.BranchListModel)

	// Handle checkout action
	if result.GetAction() == "checkout" && result.GetSelectedBranch() != nil {
		branch := result.GetSelectedBranch()

		// Don't checkout if already on this branch
		if branch.IsCurrent {
			_, _ = fmt.Fprintf(os.Stdout, "Already on branch '%s'\n", branch.Name)

			return nil
		}

		// Don't checkout detached HEAD
		if branch.Name == "(detached HEAD)" {
			_, _ = fmt.Fprintln(os.Stderr, "Cannot checkout detached HEAD state")

			return nil
		}

		// For remote branches, checkout without the remote prefix
		branchName := branch.Name
		if branch.IsRemote {
			// Extract branch name from origin/branch-name format
			if idx := findLastSlash(branchName); idx != -1 {
				branchName = branchName[idx+1:]
			}
		}

		if err := core.CheckoutBranch(result.GetRepoPath(), branchName); err != nil {
			return fmt.Errorf("failed to checkout branch: %w", err)
		}

		_, _ = fmt.Fprintf(os.Stdout, "Switched to branch '%s'\n", branchName)
	}

	return nil
}

func findLastSlash(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '/' {
			return i
		}
	}

	return -1
}
