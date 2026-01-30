package cmd

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/inovacc/clonr/internal/cli"
	"github.com/inovacc/clonr/internal/core"
	"github.com/spf13/cobra"
)

var cloneCmd = &cobra.Command{
	DisableFlagsInUseLine: true,
	Use:                   "clone <repository> [<directory>] [-- <gitflags>...]",
	Short:                 "Clone a Git repository",
	Long: `Clone a Git repository and register it with Clonr.

Supports multiple repository formats (like gh repo clone):
  - owner/repo           Clone from GitHub using owner/repo format
  - repo                 Clone from your GitHub account (requires gh auth)
  - https://...          Clone using HTTPS URL
  - git@host:owner/repo  Clone using SSH URL

If the OWNER/ portion is omitted, it defaults to the authenticated GitHub user.

You can clone from any GitHub URL, including:
  - github.com/owner/repo/blob/main/file.go#L10  (strips extra path)
  - github.com/owner/repo/pull/123               (extracts repo)

Pass additional git clone flags after '--':
  clonr clone owner/repo -- --depth=1 --single-branch

If no destination is specified, the repository is cloned to the default clone
directory configured in settings (default: ~/clonr).`,
	Example: `  # Clone using owner/repo format
  clonr clone btcsuite/btcd

  # Clone from your own GitHub account
  clonr clone myrepo

  # Clone to a specific directory
  clonr clone cli/cli workspace/cli

  # Clone with git flags (shallow clone)
  clonr clone owner/repo -- --depth=1

  # Clone using SSH
  clonr clone git@github.com:owner/repo.git

  # Clone from a GitHub URL (extra path is stripped)
  clonr clone https://github.com/owner/repo/blob/main/README.md`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		force, _ := cmd.Flags().GetBool("force")
		noTUI, _ := cmd.Flags().GetBool("no-tui")

		opts := core.CloneOptions{
			Force: force,
		}

		if noTUI {
			return core.CloneRepoWithOptions(args, opts)
		}

		result, err := core.PrepareClone(args, opts)
		if err != nil {
			return err
		}

		// Authentication is handled via credential helper (clonr auth git-credential)
		m := cli.NewCloneModel(result.CloneURL, result.TargetPath)
		p := tea.NewProgram(m)

		finalModel, err := p.Run()
		if err != nil {
			return err
		}

		cloneModel := finalModel.(cli.CloneModel)
		if cloneModel.Error() != nil {
			return cloneModel.Error()
		}

		return core.SaveClonedRepoFromResult(result)
	},
}

func init() {
	rootCmd.AddCommand(cloneCmd)
	cloneCmd.Flags().BoolP("force", "f", false, "Force clone even if repository/directory already exists (removes existing)")
	cloneCmd.Flags().Bool("no-tui", false, "Non-interactive mode (no TUI, useful for scripts)")
}
