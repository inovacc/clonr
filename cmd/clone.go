package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/inovacc/clonr/internal/cli"
	"github.com/inovacc/clonr/internal/core"
	"github.com/inovacc/clonr/internal/grpcclient"
	"github.com/inovacc/clonr/internal/model"
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

If no destination is specified, the repository is cloned to the active workspace's
directory, or the default clone directory if no workspace is configured.

Use --workspace to specify which workspace to clone into. If workspaces exist
but none is specified in interactive mode, you'll be prompted to select one.`,
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
  clonr clone https://github.com/owner/repo/blob/main/README.md

  # Clone into a specific workspace
  clonr clone owner/repo --workspace personal

  # Clone non-interactively (uses active workspace)
  clonr clone owner/repo --no-tui`,
	Args: cobra.MinimumNArgs(1),
	RunE: runClone,
}

func init() {
	rootCmd.AddCommand(cloneCmd)
	cloneCmd.Flags().BoolP("force", "f", false, "Force clone even if repository/directory already exists (removes existing)")
	cloneCmd.Flags().Bool("no-tui", false, "Non-interactive mode (no TUI, useful for scripts)")
	cloneCmd.Flags().StringP("workspace", "w", "", "Workspace to clone into")
}

func runClone(cmd *cobra.Command, args []string) error {
	force, _ := cmd.Flags().GetBool("force")
	noTUI, _ := cmd.Flags().GetBool("no-tui")
	workspace, _ := cmd.Flags().GetString("workspace")

	opts := core.CloneOptions{
		Force:     force,
		Workspace: workspace,
	}

	// Get client to check workspaces
	client, err := grpcclient.GetClient()
	if err != nil {
		return err
	}

	// If no workspace specified and TUI mode, check if we need workspace selection
	if workspace == "" && !noTUI {
		workspaces, err := client.ListWorkspaces()
		if err != nil {
			return fmt.Errorf("failed to list workspaces: %w", err)
		}

		// If no workspaces exist, create a default one
		if len(workspaces) == 0 {
			if err := createDefaultWorkspace(client); err != nil {
				return err
			}
		} else if len(workspaces) > 1 {
			// Multiple workspaces exist - show selection TUI
			m, err := cli.NewWorkspaceSelectorForClone()
			if err != nil {
				return err
			}

			p := tea.NewProgram(m)

			finalModel, err := p.Run()
			if err != nil {
				return err
			}

			selector := finalModel.(cli.WorkspaceSelectorModel)
			selected := selector.GetSelected()

			switch {
			case selected == nil:
				// User cancelled - use active workspace
				active, err := client.GetActiveWorkspace()
				if err == nil && active != nil {
					opts.Workspace = active.Name
				}
			case selector.IsNewWorkspace():
				// User wants to create a new workspace
				if err := createWorkspaceFromSelection(client, selected); err != nil {
					return err
				}

				opts.Workspace = selected.Name
			default:
				opts.Workspace = selected.Name
			}
		}
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
}

func createDefaultWorkspace(client *grpcclient.Client) error {
	// Get config to use as default path
	cfg, err := client.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	workspace := &model.Workspace{
		Name:        model.DefaultWorkspaceName(),
		Description: "Default workspace",
		Path:        cfg.DefaultCloneDir,
		Active:      true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := client.SaveWorkspace(workspace); err != nil {
		return fmt.Errorf("failed to create default workspace: %w", err)
	}

	_, _ = fmt.Fprintln(os.Stdout, "Created default workspace")

	return nil
}

func createWorkspaceFromSelection(client *grpcclient.Client, ws *model.Workspace) error {
	// Expand ~ to home directory
	path := ws.Path
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}

		path = filepath.Join(home, path[1:])
	}

	// Make path absolute
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Create directory if it doesn't exist
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		if err := os.MkdirAll(absPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	workspace := &model.Workspace{
		Name:        ws.Name,
		Description: ws.Description,
		Path:        absPath,
		Active:      true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := client.SaveWorkspace(workspace); err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Created workspace '%s'\n", ws.Name)

	return nil
}
