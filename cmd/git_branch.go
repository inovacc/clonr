package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/inovacc/clonr/internal/git"
	"github.com/spf13/cobra"
)

var gitBranchCmd = &cobra.Command{
	Use:   "branch [name]",
	Short: "List, create, or delete branches",
	Long: `List, create, or delete branches.

If called without arguments, lists existing branches.
The current branch is highlighted with an asterisk.

Examples:
  clonr git branch                  # List local branches
  clonr git branch --all            # List all branches (local and remote)
  clonr git branch feature          # Create new branch
  clonr git branch -d feature       # Delete branch
  clonr git branch -D feature       # Force delete branch
  clonr git branch --json           # Output as JSON`,
	RunE: runGitBranch,
}

func init() {
	gitCmd.AddCommand(gitBranchCmd)
	gitBranchCmd.Flags().BoolP("all", "a", false, "List both local and remote branches")
	gitBranchCmd.Flags().BoolP("delete", "d", false, "Delete the branch")
	gitBranchCmd.Flags().BoolP("force-delete", "D", false, "Force delete the branch")
	gitBranchCmd.Flags().Bool("json", false, "Output as JSON")
}

func runGitBranch(cmd *cobra.Command, args []string) error {
	client := git.NewClient()
	ctx := context.Background()

	if !client.IsRepository(ctx) {
		return fmt.Errorf("not a git repository")
	}

	all, _ := cmd.Flags().GetBool("all")
	del, _ := cmd.Flags().GetBool("delete")
	forceDel, _ := cmd.Flags().GetBool("force-delete")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	// Delete branch
	if del || forceDel {
		if len(args) == 0 {
			return fmt.Errorf("branch name required for deletion")
		}
		name := args[0]
		if err := client.DeleteBranch(ctx, name, forceDel); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(os.Stdout, okStyle.Render("Deleted branch '%s'\n"), name)
		return nil
	}

	// Create branch
	if len(args) > 0 {
		name := args[0]
		if err := client.Checkout(ctx, name, git.CheckoutOptions{Create: true}); err != nil {
			if git.IsAlreadyExists(err) {
				return fmt.Errorf("branch '%s' already exists", name)
			}
			return err
		}
		_, _ = fmt.Fprintf(os.Stdout, okStyle.Render("Created and switched to branch '%s'\n"), name)
		return nil
	}

	// List branches
	branches, err := client.ListBranchesDetailed(ctx, all)
	if err != nil {
		return err
	}

	if len(branches) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "No branches found")
		return nil
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(branches)
	}

	for _, b := range branches {
		var prefix string
		if b.Current {
			prefix = okStyle.Render("* ")
		} else {
			prefix = "  "
		}

		name := b.Name
		if b.Current {
			name = okStyle.Render(name)
		}

		var info string
		if b.Upstream != "" {
			if b.Gone {
				info = errStyle.Render(" [gone]")
			} else {
				info = dimStyle.Render(fmt.Sprintf(" -> %s", b.Upstream))
			}
		}

		_, _ = fmt.Fprintf(os.Stdout, "%s%s%s\n", prefix, name, info)
	}

	return nil
}
