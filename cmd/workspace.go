package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/inovacc/clonr/internal/cli"
	"github.com/inovacc/clonr/internal/client/grpc"
	"github.com/inovacc/clonr/internal/core"
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
	Long: `List all configured workspaces.

Examples:
  clonr workspace list
  clonr workspace list --json`,
	Aliases: []string{"ls"},
	RunE:    runWorkspaceList,
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

var workspaceCloneCmd = &cobra.Command{
	Use:   "clone <source> <new-name>",
	Short: "Clone a workspace with a new name",
	Long: `Clone an existing workspace configuration with a new name.

By default, creates a new directory alongside the source workspace.
Use --path to specify a custom directory.

Examples:
  clonr workspace clone personal work
  clonr workspace clone personal work --path ~/projects/work
  clonr workspace clone work corp --description "Corporate projects"`,
	Args: cobra.ExactArgs(2),
	RunE: runWorkspaceClone,
}

var workspaceEditCmd = &cobra.Command{
	Use:   "edit <name>",
	Short: "Edit a workspace",
	Long: `Edit an existing workspace's properties.

You can modify the workspace name, path, or description.
At least one flag must be provided.

Examples:
  clonr workspace edit personal --name private
  clonr workspace edit work --path ~/new/work/path
  clonr workspace edit work --description "Updated description"
  clonr workspace edit personal --name private --description "Private projects"`,
	Args: cobra.ExactArgs(1),
	RunE: runWorkspaceEdit,
}

var workspaceInfoCmd = &cobra.Command{
	Use:   "info <name>",
	Short: "Show workspace information",
	Long: `Show detailed information about a workspace.

Examples:
  clonr workspace info personal
  clonr workspace info work --json`,
	Args: cobra.ExactArgs(1),
	RunE: runWorkspaceInfo,
}

var workspaceMapCmd = &cobra.Command{
	Use:   "map <name>",
	Short: "Scan workspace directory for Git repositories",
	Long: `Scan the workspace's directory for existing Git repositories and register them.

This command scans the workspace's configured path for Git repositories and
registers them with this workspace.

Examples:
  clonr workspace map work                  # Scan and register repos
  clonr workspace map work --dry-run        # Preview without adding
  clonr workspace map work --depth 3        # Limit scan depth
  clonr workspace map work --json           # Output as JSON`,
	Args: cobra.ExactArgs(1),
	RunE: runWorkspaceMap,
}

var (
	workspaceAddPath        string
	workspaceAddDescription string
	workspaceClonePath      string
	workspaceCloneDesc      string
	workspaceListJSON       bool
	workspaceEditName       string
	workspaceEditPath       string
	workspaceEditDesc       string
	workspaceInfoJSON       bool
	workspaceMapDryRun      bool
	workspaceMapDepth       int
	workspaceMapJSON        bool
	workspaceMapVerbose     bool
)

func init() {
	rootCmd.AddCommand(workspaceCmd)

	// Add subcommands
	workspaceCmd.AddCommand(workspaceAddCmd)
	workspaceCmd.AddCommand(workspaceListCmd)
	workspaceCmd.AddCommand(workspaceRemoveCmd)
	workspaceCmd.AddCommand(workspaceMoveCmd)
	workspaceCmd.AddCommand(workspaceSelectCmd)
	workspaceCmd.AddCommand(workspaceCloneCmd)
	workspaceCmd.AddCommand(workspaceEditCmd)
	workspaceCmd.AddCommand(workspaceInfoCmd)
	workspaceCmd.AddCommand(workspaceMapCmd)

	// Add flags
	workspaceAddCmd.Flags().StringVar(&workspaceAddPath, "path", "", "Base directory for this workspace (required)")
	workspaceAddCmd.Flags().StringVar(&workspaceAddDescription, "description", "", "Description of the workspace")

	workspaceListCmd.Flags().BoolVar(&workspaceListJSON, "json", false, "Output as JSON")

	workspaceCloneCmd.Flags().StringVar(&workspaceClonePath, "path", "", "Directory for the new workspace (default: alongside source)")
	workspaceCloneCmd.Flags().StringVar(&workspaceCloneDesc, "description", "", "Description for the new workspace")

	workspaceEditCmd.Flags().StringVar(&workspaceEditName, "name", "", "New name for the workspace")
	workspaceEditCmd.Flags().StringVar(&workspaceEditPath, "path", "", "New path for the workspace")
	workspaceEditCmd.Flags().StringVar(&workspaceEditDesc, "description", "", "New description for the workspace")

	workspaceInfoCmd.Flags().BoolVar(&workspaceInfoJSON, "json", false, "Output as JSON")

	workspaceMapCmd.Flags().BoolVar(&workspaceMapDryRun, "dry-run", false, "Show what would be added without actually adding")
	workspaceMapCmd.Flags().IntVar(&workspaceMapDepth, "depth", 0, "Maximum directory depth to scan (0 = unlimited)")
	workspaceMapCmd.Flags().BoolVar(&workspaceMapJSON, "json", false, "Output results as JSON")
	workspaceMapCmd.Flags().BoolVarP(&workspaceMapVerbose, "verbose", "v", false, "Show verbose output")

	_ = workspaceAddCmd.MarkFlagRequired("path")
}

func runWorkspaceAdd(_ *cobra.Command, args []string) error {
	name := args[0]

	grpcClient, err := grpc.GetClient()
	if err != nil {
		return err
	}

	// Check if workspace already exists
	exists, err := grpcClient.WorkspaceExists(name)
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

	// Create workspace
	workspace := &model.Workspace{
		Name:        name,
		Description: workspaceAddDescription,
		Path:        absPath,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := grpcClient.SaveWorkspace(workspace); err != nil {
		return fmt.Errorf("failed to save workspace: %w", err)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Workspace '%s' created\n", name)
	_, _ = fmt.Fprintf(os.Stdout, "Path: %s\n", absPath)
	_, _ = fmt.Fprintf(os.Stdout, "Associate a profile with: clonr profile add <name> --workspace %s\n", name)

	return nil
}

// WorkspaceListItem represents a workspace in JSON output
type WorkspaceListItem struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Description string `json:"description,omitempty"`
	RepoCount   int    `json:"repo_count"`
	Profiles    int    `json:"profiles"`
}

func runWorkspaceList(_ *cobra.Command, _ []string) error {
	client, err := grpc.GetClient()
	if err != nil {
		return err
	}

	workspaces, err := client.ListWorkspaces()
	if err != nil {
		return fmt.Errorf("failed to list workspaces: %w", err)
	}

	if len(workspaces) == 0 {
		if workspaceListJSON {
			_, _ = fmt.Fprintln(os.Stdout, "[]")
			return nil
		}

		_, _ = fmt.Fprintln(os.Stdout, "No workspaces configured.")
		_, _ = fmt.Fprintln(os.Stdout, "Create one with: clonr workspace add <name> --path <directory>")

		return nil
	}

	// Get all repos once for counting
	allRepos, _ := client.GetAllRepos()

	// Get all profiles to count per workspace
	allProfiles, _ := client.ListProfiles()

	// JSON output
	if workspaceListJSON {
		items := make([]WorkspaceListItem, 0, len(workspaces))

		for _, w := range workspaces {
			repoCount := countReposInWorkspace(client, w.Name, w.Path, allRepos)
			profileCount := countProfilesInWorkspace(w.Name, allProfiles)

			items = append(items, WorkspaceListItem{
				Name:        w.Name,
				Path:        w.Path,
				Description: w.Description,
				RepoCount:   repoCount,
				Profiles:    profileCount,
			})
		}

		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")

		return enc.Encode(items)
	}

	// Text output
	_, _ = fmt.Fprintf(os.Stdout, "Workspaces (%d):\n\n", len(workspaces))

	for _, w := range workspaces {
		_, _ = fmt.Fprintf(os.Stdout, "  %s\n", w.Name)
		_, _ = fmt.Fprintf(os.Stdout, "    Path: %s\n", w.Path)

		if w.Description != "" {
			_, _ = fmt.Fprintf(os.Stdout, "    Description: %s\n", w.Description)
		}

		// Count repos in workspace
		repoCount := countReposInWorkspace(client, w.Name, w.Path, allRepos)
		if repoCount > 0 {
			_, _ = fmt.Fprintf(os.Stdout, "    Repositories: %d\n", repoCount)
		}

		// Count profiles in workspace
		profileCount := countProfilesInWorkspace(w.Name, allProfiles)
		if profileCount > 0 {
			_, _ = fmt.Fprintf(os.Stdout, "    Profiles: %d\n", profileCount)
		}

		_, _ = fmt.Fprintln(os.Stdout)
	}

	return nil
}

// countReposInWorkspace counts repos by workspace name and by path
func countReposInWorkspace(client *grpc.Client, workspaceName, workspacePath string, allRepos []model.Repository) int {
	// Get repos by workspace field
	repos, err := client.GetReposByWorkspace(workspaceName)
	if err != nil {
		repos = []string{}
	}

	repoSet := make(map[string]bool)
	for _, r := range repos {
		repoSet[r] = true
	}

	// Also count repos whose path is within the workspace path
	for _, r := range allRepos {
		if !repoSet[r.URL] && r.Path != "" && isPathWithin(r.Path, workspacePath) {
			repoSet[r.URL] = true
		}
	}

	return len(repoSet)
}

// countProfilesInWorkspace counts profiles associated with a workspace
func countProfilesInWorkspace(workspaceName string, profiles []model.Profile) int {
	count := 0

	for _, p := range profiles {
		if p.Workspace == workspaceName {
			count++
		}
	}

	return count
}

func runWorkspaceRemove(_ *cobra.Command, args []string) error {
	name := args[0]

	client, err := grpc.GetClient()
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

	client, err := grpc.GetClient()
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

func runWorkspaceClone(_ *cobra.Command, args []string) error {
	sourceName := args[0]
	newName := args[1]

	client, err := grpc.GetClient()
	if err != nil {
		return err
	}

	// Check if source workspace exists
	source, err := client.GetWorkspace(sourceName)
	if err != nil {
		return fmt.Errorf("failed to get source workspace: %w", err)
	}

	if source == nil {
		return fmt.Errorf("workspace '%s' not found", sourceName)
	}

	// Check if new workspace already exists
	exists, err := client.WorkspaceExists(newName)
	if err != nil {
		return fmt.Errorf("failed to check workspace existence: %w", err)
	}

	if exists {
		return fmt.Errorf("workspace '%s' already exists", newName)
	}

	// Determine the path for the new workspace
	var newPath string
	if workspaceClonePath != "" {
		newPath = workspaceClonePath

		// Expand ~ to home directory
		if newPath[0] == '~' {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("failed to get home directory: %w", err)
			}

			newPath = filepath.Join(home, newPath[1:])
		}
	} else {
		// Default: create alongside source workspace
		parentDir := filepath.Dir(source.Path)
		newPath = filepath.Join(parentDir, newName)
	}

	// Make path absolute
	absPath, err := filepath.Abs(newPath)
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

	// Determine description
	description := workspaceCloneDesc
	if description == "" && source.Description != "" {
		description = fmt.Sprintf("Cloned from %s", sourceName)
	}

	// Create new workspace
	workspace := &model.Workspace{
		Name:        newName,
		Description: description,
		Path:        absPath,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := client.SaveWorkspace(workspace); err != nil {
		return fmt.Errorf("failed to save workspace: %w", err)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Workspace '%s' cloned from '%s'\n", newName, sourceName)
	_, _ = fmt.Fprintf(os.Stdout, "Path: %s\n", absPath)
	_, _ = fmt.Fprintf(os.Stdout, "To use this workspace: clonr workspace use %s\n", newName)

	return nil
}

func runWorkspaceEdit(_ *cobra.Command, args []string) error {
	name := args[0]

	// Check if at least one flag is provided
	if workspaceEditName == "" && workspaceEditPath == "" && workspaceEditDesc == "" {
		return fmt.Errorf("at least one of --name, --path, or --description must be provided")
	}

	client, err := grpc.GetClient()
	if err != nil {
		return err
	}

	// Get the existing workspace
	workspace, err := client.GetWorkspace(name)
	if err != nil {
		return fmt.Errorf("failed to get workspace: %w", err)
	}

	if workspace == nil {
		return fmt.Errorf("workspace '%s' not found", name)
	}

	// Track changes for output
	var changes []string

	// Update name if provided
	newName := name
	if workspaceEditName != "" && workspaceEditName != name {
		// Check if new name already exists
		exists, err := client.WorkspaceExists(workspaceEditName)
		if err != nil {
			return fmt.Errorf("failed to check workspace existence: %w", err)
		}

		if exists {
			return fmt.Errorf("workspace '%s' already exists", workspaceEditName)
		}

		newName = workspaceEditName
		changes = append(changes, fmt.Sprintf("name: %s -> %s", name, newName))
	}

	// Update path if provided
	if workspaceEditPath != "" {
		newPath := workspaceEditPath

		// Expand ~ to home directory
		if len(newPath) > 0 && newPath[0] == '~' {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("failed to get home directory: %w", err)
			}

			newPath = filepath.Join(home, newPath[1:])
		}

		// Make path absolute
		absPath, err := filepath.Abs(newPath)
		if err != nil {
			return fmt.Errorf("failed to resolve path: %w", err)
		}

		if absPath != workspace.Path {
			// Create directory if it doesn't exist
			if _, err := os.Stat(absPath); os.IsNotExist(err) {
				if err := os.MkdirAll(absPath, 0755); err != nil {
					return fmt.Errorf("failed to create directory: %w", err)
				}

				_, _ = fmt.Fprintf(os.Stdout, "Created directory: %s\n", absPath)
			}

			changes = append(changes, fmt.Sprintf("path: %s -> %s", workspace.Path, absPath))
			workspace.Path = absPath
		}
	}

	// Update description if provided
	if workspaceEditDesc != "" && workspaceEditDesc != workspace.Description {
		oldDesc := workspace.Description
		if oldDesc == "" {
			oldDesc = "(empty)"
		}

		changes = append(changes, fmt.Sprintf("description: %s -> %s", oldDesc, workspaceEditDesc))
		workspace.Description = workspaceEditDesc
	}

	// No actual changes
	if len(changes) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "No changes to apply.")
		return nil
	}

	// Update timestamp
	workspace.UpdatedAt = time.Now()

	// If name changed, we need to delete old and create new
	if newName != name {
		workspace.Name = newName

		// Delete old workspace
		if err := client.DeleteWorkspace(name); err != nil {
			return fmt.Errorf("failed to delete old workspace: %w", err)
		}

		// Update profile references if any profiles use this workspace
		profiles, err := client.ListProfiles()
		if err == nil {
			for i := range profiles {
				if profiles[i].Workspace == name {
					profiles[i].Workspace = newName
					_ = client.SaveProfile(&profiles[i])
				}
			}
		}

		// Update repository references
		repos, err := client.GetReposByWorkspace(name)
		if err == nil {
			for _, repoURL := range repos {
				_ = client.UpdateRepoWorkspace(repoURL, newName)
			}
		}
	}

	// Save workspace
	if err := client.SaveWorkspace(workspace); err != nil {
		return fmt.Errorf("failed to save workspace: %w", err)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Workspace '%s' updated:\n", newName)

	for _, change := range changes {
		_, _ = fmt.Fprintf(os.Stdout, "  - %s\n", change)
	}

	return nil
}

// WorkspaceInfoItem represents detailed workspace info for JSON output
type WorkspaceInfoItem struct {
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	Description string    `json:"description,omitempty"`
	RepoCount   int       `json:"repo_count"`
	Repos       []string  `json:"repos,omitempty"`
	Profiles    []string  `json:"profiles,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	DiskUsage   string    `json:"disk_usage,omitempty"`
	PathExists  bool      `json:"path_exists"`
}

func runWorkspaceInfo(_ *cobra.Command, args []string) error {
	client, err := grpc.GetClient()
	if err != nil {
		return err
	}

	name := args[0]

	workspace, err := client.GetWorkspace(name)
	if err != nil {
		return fmt.Errorf("failed to get workspace: %w", err)
	}

	if workspace == nil {
		return fmt.Errorf("workspace '%s' not found", name)
	}

	// Get repos in workspace (by workspace field)
	repos, err := client.GetReposByWorkspace(workspace.Name)
	if err != nil {
		repos = []string{}
	}

	// Also check repos by path (repos within workspace directory)
	allRepos, err := client.GetAllRepos()
	if err == nil {
		repoSet := make(map[string]bool)
		for _, r := range repos {
			repoSet[r] = true
		}

		// Add repos whose path is within the workspace path
		for _, r := range allRepos {
			if !repoSet[r.URL] && r.Path != "" && isPathWithin(r.Path, workspace.Path) {
				repos = append(repos, r.URL)
				repoSet[r.URL] = true
			}
		}
	}

	// Find profiles associated with this workspace
	var profileNames []string

	profiles, err := client.ListProfiles()
	if err == nil {
		for _, p := range profiles {
			if p.Workspace == workspace.Name {
				name := p.Name
				if p.Default {
					name += " (default)"
				}

				profileNames = append(profileNames, name)
			}
		}
	}

	// Check if path exists
	pathExists := true
	if _, err := os.Stat(workspace.Path); os.IsNotExist(err) {
		pathExists = false
	}

	// Calculate disk usage if path exists
	var diskUsage string
	if pathExists {
		diskUsage = calculateDirSize(workspace.Path)
	}

	// JSON output
	if workspaceInfoJSON {
		info := WorkspaceInfoItem{
			Name:        workspace.Name,
			Path:        workspace.Path,
			Description: workspace.Description,
			RepoCount:   len(repos),
			Repos:       repos,
			Profiles:    profileNames,
			CreatedAt:   workspace.CreatedAt,
			UpdatedAt:   workspace.UpdatedAt,
			DiskUsage:   diskUsage,
			PathExists:  pathExists,
		}

		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")

		return enc.Encode(info)
	}

	// Text output
	_, _ = fmt.Fprintf(os.Stdout, "Workspace: %s\n", workspace.Name)
	_, _ = fmt.Fprintf(os.Stdout, "Path: %s\n", workspace.Path)

	if !pathExists {
		_, _ = fmt.Fprintln(os.Stdout, "  ⚠ Path does not exist")
	}

	if workspace.Description != "" {
		_, _ = fmt.Fprintf(os.Stdout, "Description: %s\n", workspace.Description)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Repositories: %d\n", len(repos))

	if len(profileNames) > 0 {
		_, _ = fmt.Fprintf(os.Stdout, "Profiles: %s\n", strings.Join(profileNames, ", "))
	}

	if diskUsage != "" {
		_, _ = fmt.Fprintf(os.Stdout, "Disk Usage: %s\n", diskUsage)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Created: %s\n", workspace.CreatedAt.Format(time.RFC3339))

	if !workspace.UpdatedAt.IsZero() && workspace.UpdatedAt != workspace.CreatedAt {
		_, _ = fmt.Fprintf(os.Stdout, "Updated: %s\n", workspace.UpdatedAt.Format(time.RFC3339))
	}

	// List repos if any
	if len(repos) > 0 {
		_, _ = fmt.Fprintln(os.Stdout, "\nRepositories:")

		for _, repo := range repos {
			_, _ = fmt.Fprintf(os.Stdout, "  - %s\n", repo)
		}
	}

	return nil
}

// calculateDirSize returns a human-readable size of a directory
func calculateDirSize(path string) string {
	var size int64

	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return filepath.SkipDir // Skip inaccessible directories
		}

		if !info.IsDir() {
			size += info.Size()
		}

		return nil
	})
	if err != nil {
		return ""
	}

	return formatBytes(size)
}

// formatBytes formats bytes into human-readable format
func formatBytes(bytes int64) string {
	const unit = 1024

	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// isPathWithin checks if childPath is within parentPath
func isPathWithin(childPath, parentPath string) bool {
	// Clean and normalize paths
	child, err := filepath.Abs(childPath)
	if err != nil {
		return false
	}

	parent, err := filepath.Abs(parentPath)
	if err != nil {
		return false
	}

	// On Windows, paths are case-insensitive
	if runtime.GOOS == "windows" {
		child = strings.ToLower(child)
		parent = strings.ToLower(parent)
	}

	// Ensure parent ends with separator for accurate prefix matching
	if !strings.HasSuffix(parent, string(filepath.Separator)) {
		parent += string(filepath.Separator)
	}

	return strings.HasPrefix(child, parent)
}

func runWorkspaceMap(_ *cobra.Command, args []string) error {
	name := args[0]

	client, err := grpc.GetClient()
	if err != nil {
		return err
	}

	// Get the workspace
	workspace, err := client.GetWorkspace(name)
	if err != nil {
		return fmt.Errorf("failed to get workspace: %w", err)
	}

	if workspace == nil {
		return fmt.Errorf("workspace '%s' not found", name)
	}

	// Check if path exists
	if _, err := os.Stat(workspace.Path); os.IsNotExist(err) {
		return fmt.Errorf("workspace path does not exist: %s", workspace.Path)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Scanning workspace '%s' at %s...\n\n", name, workspace.Path)

	// Build map options
	opts := core.MapOptions{
		DryRun:    workspaceMapDryRun,
		MaxDepth:  workspaceMapDepth,
		Exclude:   core.DefaultExcludeDirs,
		JSON:      workspaceMapJSON,
		Verbose:   workspaceMapVerbose,
		Workspace: name,
	}

	return core.MapReposWithOptions([]string{workspace.Path}, opts)
}
