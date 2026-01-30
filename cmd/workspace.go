package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/inovacc/clonr/internal/cli"
	"github.com/inovacc/clonr/internal/grpcclient"
	"github.com/inovacc/clonr/internal/model"
	"github.com/spf13/cobra"
)

var workspaceCmd = &cobra.Command{
	Use:   "workspace",
	Short: "Manage workspaces",
	Long: `Manage workspaces for organizing repositories.

Workspaces allow you to logically separate repositories (e.g., work, personal, corporate).
Each workspace has its own base clone directory.`,
}

var workspaceAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Create a new workspace",
	Long: `Create a new workspace for organizing repositories.

Examples:
  clonr workspace add personal --path ~/clonr/personal
  clonr workspace add work --path ~/clonr/work --description "Work projects"`,
	Args: cobra.ExactArgs(1),
	RunE: runWorkspaceAdd,
}

var workspaceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all workspaces",
	Long:  `List all configured workspaces.`,
	RunE:  runWorkspaceList,
}

var workspaceUseCmd = &cobra.Command{
	Use:   "use <name>",
	Short: "Set the active workspace",
	Long: `Set a workspace as the active workspace.

The active workspace is used by default when cloning repositories.

Example:
  clonr workspace use personal`,
	Args: cobra.ExactArgs(1),
	RunE: runWorkspaceUse,
}

var workspaceRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a workspace",
	Long: `Remove a workspace.

A workspace can only be removed if it has no repositories.
Move repositories to another workspace first using 'clonr workspace move'.

Example:
  clonr workspace remove old-workspace`,
	Args: cobra.ExactArgs(1),
	RunE: runWorkspaceRemove,
}

var workspaceMoveCmd = &cobra.Command{
	Use:   "move <repo-url> <target-workspace>",
	Short: "Move a repository to a different workspace",
	Long: `Move a repository to a different workspace.

This updates the workspace association in the database.
Note: This does not move the files on disk.

Example:
  clonr workspace move https://github.com/owner/repo work`,
	Args: cobra.ExactArgs(2),
	RunE: runWorkspaceMove,
}

var workspaceSelectCmd = &cobra.Command{
	Use:   "select",
	Short: "Interactively select a workspace",
	Long:  `Open an interactive TUI to select a workspace.`,
	RunE:  runWorkspaceSelect,
}

var (
	workspaceAddPath        string
	workspaceAddDescription string
)

func init() {
	rootCmd.AddCommand(workspaceCmd)

	// Add subcommands
	workspaceCmd.AddCommand(workspaceAddCmd)
	workspaceCmd.AddCommand(workspaceListCmd)
	workspaceCmd.AddCommand(workspaceUseCmd)
	workspaceCmd.AddCommand(workspaceRemoveCmd)
	workspaceCmd.AddCommand(workspaceMoveCmd)
	workspaceCmd.AddCommand(workspaceSelectCmd)

	// Add flags
	workspaceAddCmd.Flags().StringVar(&workspaceAddPath, "path", "", "Base directory for this workspace (required)")
	workspaceAddCmd.Flags().StringVar(&workspaceAddDescription, "description", "", "Description of the workspace")

	_ = workspaceAddCmd.MarkFlagRequired("path")
}

func runWorkspaceAdd(_ *cobra.Command, args []string) error {
	name := args[0]

	client, err := grpcclient.GetClient()
	if err != nil {
		return err
	}

	// Check if workspace already exists
	exists, err := client.WorkspaceExists(name)
	if err != nil {
		return fmt.Errorf("failed to check workspace existence: %w", err)
	}

	if exists {
		return fmt.Errorf("workspace '%s' already exists", name)
	}

	// Expand and validate path
	path := workspaceAddPath
	if path == "" {
		return fmt.Errorf("--path is required")
	}

	// Expand ~ to home directory
	if path[0] == '~' {
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

		_, _ = fmt.Fprintf(os.Stdout, "Created directory: %s\n", absPath)
	}

	// Check if this is the first workspace (make it active)
	workspaces, err := client.ListWorkspaces()
	if err != nil {
		return fmt.Errorf("failed to list workspaces: %w", err)
	}

	isFirstWorkspace := len(workspaces) == 0

	// Create workspace
	workspace := &model.Workspace{
		Name:        name,
		Description: workspaceAddDescription,
		Path:        absPath,
		Active:      isFirstWorkspace,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := client.SaveWorkspace(workspace); err != nil {
		return fmt.Errorf("failed to save workspace: %w", err)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Workspace '%s' created\n", name)
	_, _ = fmt.Fprintf(os.Stdout, "Path: %s\n", absPath)

	if isFirstWorkspace {
		_, _ = fmt.Fprintln(os.Stdout, "This workspace is now active.")
	} else {
		_, _ = fmt.Fprintf(os.Stdout, "To use this workspace: clonr workspace use %s\n", name)
	}

	return nil
}

func runWorkspaceList(_ *cobra.Command, _ []string) error {
	client, err := grpcclient.GetClient()
	if err != nil {
		return err
	}

	workspaces, err := client.ListWorkspaces()
	if err != nil {
		return fmt.Errorf("failed to list workspaces: %w", err)
	}

	if len(workspaces) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "No workspaces configured.")
		_, _ = fmt.Fprintln(os.Stdout, "Create one with: clonr workspace add <name> --path <directory>")

		return nil
	}

	_, _ = fmt.Fprintf(os.Stdout, "Workspaces (%d):\n\n", len(workspaces))

	for _, w := range workspaces {
		active := ""
		if w.Active {
			active = " (active)"
		}

		_, _ = fmt.Fprintf(os.Stdout, "  %s%s\n", w.Name, active)
		_, _ = fmt.Fprintf(os.Stdout, "    Path: %s\n", w.Path)

		if w.Description != "" {
			_, _ = fmt.Fprintf(os.Stdout, "    Description: %s\n", w.Description)
		}

		// Count repos in workspace
		urls, err := client.GetReposByWorkspace(w.Name)
		if err == nil && len(urls) > 0 {
			_, _ = fmt.Fprintf(os.Stdout, "    Repositories: %d\n", len(urls))
		}

		_, _ = fmt.Fprintln(os.Stdout)
	}

	return nil
}

func runWorkspaceUse(_ *cobra.Command, args []string) error {
	name := args[0]

	client, err := grpcclient.GetClient()
	if err != nil {
		return err
	}

	// Check if workspace exists
	exists, err := client.WorkspaceExists(name)
	if err != nil {
		return fmt.Errorf("failed to check workspace existence: %w", err)
	}

	if !exists {
		return fmt.Errorf("workspace '%s' not found", name)
	}

	if err := client.SetActiveWorkspace(name); err != nil {
		return fmt.Errorf("failed to set active workspace: %w", err)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Workspace '%s' is now active\n", name)

	return nil
}

func runWorkspaceRemove(_ *cobra.Command, args []string) error {
	name := args[0]

	client, err := grpcclient.GetClient()
	if err != nil {
		return err
	}

	// Check if workspace exists
	exists, err := client.WorkspaceExists(name)
	if err != nil {
		return fmt.Errorf("failed to check workspace existence: %w", err)
	}

	if !exists {
		return fmt.Errorf("workspace '%s' not found", name)
	}

	// Check if workspace has repositories
	urls, err := client.GetReposByWorkspace(name)
	if err != nil {
		return fmt.Errorf("failed to get repositories: %w", err)
	}

	if len(urls) > 0 {
		return fmt.Errorf("workspace '%s' has %d repositories\nMove them first: clonr workspace move <repo-url> <target-workspace>", name, len(urls))
	}

	if err := client.DeleteWorkspace(name); err != nil {
		return fmt.Errorf("failed to delete workspace: %w", err)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Workspace '%s' removed\n", name)

	return nil
}

func runWorkspaceMove(_ *cobra.Command, args []string) error {
	repoURL := args[0]
	targetWorkspace := args[1]

	client, err := grpcclient.GetClient()
	if err != nil {
		return err
	}

	// Check if target workspace exists
	exists, err := client.WorkspaceExists(targetWorkspace)
	if err != nil {
		return fmt.Errorf("failed to check workspace existence: %w", err)
	}

	if !exists {
		return fmt.Errorf("workspace '%s' not found", targetWorkspace)
	}

	if err := client.UpdateRepoWorkspace(repoURL, targetWorkspace); err != nil {
		return fmt.Errorf("failed to move repository: %w", err)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Repository moved to workspace '%s'\n", targetWorkspace)

	return nil
}

func runWorkspaceSelect(_ *cobra.Command, _ []string) error {
	m, err := cli.NewWorkspaceSelector(false)
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

	if selected == nil {
		return nil
	}

	_, _ = fmt.Fprintf(os.Stdout, "Selected workspace: %s\n", selected.Name)

	return nil
}
