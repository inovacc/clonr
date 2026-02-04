package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/inovacc/clonr/internal/cli"
	"github.com/inovacc/clonr/internal/core"
	"github.com/inovacc/clonr/internal/model"
	"github.com/spf13/cobra"
)

var gitCloneCmd = &cobra.Command{
	Use:   "clone <repository> [<directory>]",
	Short: "Clone a repository with profile authentication",
	Long: `Clone a Git repository using clonr profile authentication.

Supports multiple repository formats:
  - owner/repo           Clone from GitHub using owner/repo format
  - https://...          Clone using HTTPS URL
  - git@host:owner/repo  Clone using SSH URL

Uses the active clonr profile for authentication with private repositories.

Examples:
  clonr git clone owner/repo
  clonr git clone owner/repo ~/projects/myrepo
  clonr git clone https://github.com/owner/repo.git
  clonr git clone --profile work owner/repo
  clonr git clone --no-tui owner/repo`,
	Args: cobra.MinimumNArgs(1),
	RunE: runGitClone,
}

func init() {
	gitCmd.AddCommand(gitCloneCmd)
	gitCloneCmd.Flags().BoolP("force", "f", false, "Force clone even if directory exists (removes existing)")
	gitCloneCmd.Flags().Bool("no-tui", false, "Non-interactive mode")
	gitCloneCmd.Flags().StringP("workspace", "w", "", "Workspace to clone into")
	gitCloneCmd.Flags().StringP("profile", "p", "", "Profile to use for authentication")
}

func runGitClone(cmd *cobra.Command, args []string) error {
	force, _ := cmd.Flags().GetBool("force")
	noTUI, _ := cmd.Flags().GetBool("no-tui")
	workspace, _ := cmd.Flags().GetString("workspace")
	profile, _ := cmd.Flags().GetString("profile")

	opts := core.CloneOptions{
		Force:     force,
		Workspace: workspace,
	}

	client, err := getClient()
	if err != nil {
		return err
	}

	// Set profile if specified
	if profile != "" {
		if err := client.SetActiveProfile(profile); err != nil {
			return fmt.Errorf("failed to set active profile '%s': %w", profile, err)
		}

		p, err := client.GetProfile(profile)
		if err != nil {
			return fmt.Errorf("failed to get profile '%s': %w", profile, err)
		}

		if p != nil && workspace == "" && p.Workspace != "" {
			opts.Workspace = p.Workspace
		}

		_, _ = fmt.Fprintf(os.Stdout, dimStyle.Render("Using profile '%s'\n"), profile)
	}

	// Interactive profile selection
	if profile == "" && !noTUI {
		profiles, err := client.ListProfiles()
		if err != nil {
			return fmt.Errorf("failed to list profiles: %w", err)
		}

		if len(profiles) > 0 {
			m, err := cli.NewProfileSelector()
			if err != nil {
				return err
			}

			p := tea.NewProgram(m)
			finalModel, err := p.Run()
			if err != nil {
				return err
			}

			selector := finalModel.(cli.ProfileSelectorModel)
			selected := selector.GetSelected()

			if selected != nil {
				profile = selected.Name
				if err := client.SetActiveProfile(profile); err != nil {
					return fmt.Errorf("failed to set active profile: %w", err)
				}

				if workspace == "" && selected.Workspace != "" {
					opts.Workspace = selected.Workspace
					_, _ = fmt.Fprintf(os.Stdout, dimStyle.Render("Using profile '%s' (workspace: %s)\n"), selected.Name, selected.Workspace)
				} else {
					_, _ = fmt.Fprintf(os.Stdout, dimStyle.Render("Using profile '%s'\n"), selected.Name)
				}
			}
		}
	}

	// Interactive workspace selection
	if opts.Workspace == "" && workspace == "" && !noTUI {
		workspaces, err := client.ListWorkspaces()
		if err != nil {
			return fmt.Errorf("failed to list workspaces: %w", err)
		}

		if len(workspaces) == 0 {
			if err := createGitDefaultWorkspace(client); err != nil {
				return err
			}
		} else if len(workspaces) > 1 {
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
				active, err := client.GetActiveWorkspace()
				if err == nil && active != nil {
					opts.Workspace = active.Name
				}
			case selector.IsNewWorkspace():
				if err := createGitWorkspaceFromSelection(client, selected); err != nil {
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

	// Clone with TUI
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

	_, _ = fmt.Fprintln(os.Stdout, okStyle.Render("Clone completed successfully!"))

	return core.SaveClonedRepoFromResult(result)
}

func createGitDefaultWorkspace(client ClientInterface) error {
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

	_, _ = fmt.Fprintln(os.Stdout, dimStyle.Render("Created default workspace"))

	return nil
}

func createGitWorkspaceFromSelection(client ClientInterface, ws *model.Workspace) error {
	path := ws.Path
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		path = filepath.Join(home, path[1:])
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

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

	_, _ = fmt.Fprintf(os.Stdout, dimStyle.Render("Created workspace '%s'\n"), ws.Name)

	return nil
}
