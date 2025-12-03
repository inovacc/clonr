package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/inovacc/clonr/internal/cli"
	"github.com/inovacc/clonr/internal/core"
	"github.com/inovacc/clonr/internal/database"
	"github.com/inovacc/clonr/internal/server"
)

var (
	// Global flags
	helpFlag = flag.Bool("help", false, "Show help message")
	version  = flag.Bool("version", false, "Show version")

	// Add command flags
	addYes  = flag.Bool("y", false, "Skip confirmation prompt (add command)")
	addName = flag.String("name", "", "Optional display name (add command)")

	// List command flags
	favoritesOnly = flag.Bool("favorites", false, "Show only favorite repositories (list command)")

	versionStr = "0.2.0"
)

func main() {
	flag.Usage = printUsage

	// Parse flags
	flag.Parse()

	if *helpFlag {
		printUsage()

		os.Exit(0)
	}

	if *version {
		fmt.Println("clonr version " + versionStr)

		os.Exit(0)
	}

	args := flag.Args()

	// If no command specified, show an interactive menu
	if len(args) == 0 {
		runInteractiveMenu()
		return
	}

	command := args[0]
	commandArgs := args[1:]

	// If the first arg looks like a URL, treat it as a clone command
	if isURL(command) {
		if err := cmdClone(args); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)

			os.Exit(1)
		}

		return
	}

	// Execute command
	if err := executeCommand(command, commandArgs); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)

		os.Exit(1)
	}
}

func runInteractiveMenu() {
	for {
		m := cli.NewMainMenu()
		p := tea.NewProgram(m)

		finalModel, err := p.Run()
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)

			os.Exit(1)
		}

		menuModel := finalModel.(cli.MainMenuModel)
		choice := menuModel.GetChoice()

		if choice == "" || choice == "exit" {
			fmt.Println("Goodbye!")

			return
		}

		// Execute the chosen command
		if err := executeCommand(choice, []string{}); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			fmt.Println("\nPress Enter to continue...")

			_, _ = fmt.Scanln()
		} else if choice != "remove" && choice != "list" && choice != "open" {
			fmt.Println("\nPress Enter to continue...")

			_, _ = fmt.Scanln()
		}
	}
}

func isURL(str string) bool {
	// Check for common URL patterns
	return strings.HasPrefix(str, "http://") ||
		strings.HasPrefix(str, "https://") ||
		strings.HasPrefix(str, "git://") ||
		strings.HasPrefix(str, "ssh://") ||
		strings.HasPrefix(str, "ftp://") ||
		strings.HasPrefix(str, "sftp://") ||
		strings.HasPrefix(str, "git@")
}

func executeCommand(command string, args []string) error {
	switch command {
	case "add":
		return cmdAdd(args)
	case "list":
		return cmdList(args)
	case "favorite":
		return cmdFavorite(args)
	case "unfavorite":
		return cmdUnfavorite(args)
	case "remove", "rm":
		return cmdRemove(args)
	case "map":
		return cmdMap(args)
	case "open":
		return cmdOpen(args)
	case "update":
		return cmdUpdate(args)
	case "status":
		return cmdStatus(args)
	case "nerds":
		return cmdNerds(args)
	case "configure":
		return cmdConfigure(args)
	case "server":
		return cmdServer(args)
	case "clone":
		return cmdClone(args)
	default:
		return fmt.Errorf("unknown command: %s\nRun 'clonr --help' for usage", command)
	}
}

func cmdAdd(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("add command requires a path argument\nUsage: clonr add <path> [-y] [--name <name>]")
	}

	path := args[0]

	if !*addYes {
		fmt.Printf("Add '%s' to repositories? [y/N]: ", path)

		var response string

		_, _ = fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Cancelled.")

			return nil
		}
	}

	id, err := core.AddRepo(path, core.AddOptions{Yes: *addYes, Name: *addName})
	if err != nil {
		return err
	}

	fmt.Printf("Added: %s\n", id)

	return nil
}

func cmdList(_ []string) error {
	// Interactive mode - show a list with actions
	m, err := cli.NewRepoList(*favoritesOnly)
	if err != nil {
		return err
	}

	p := tea.NewProgram(m)
	_, err = p.Run()

	return err
}

func cmdFavorite(_ []string) error {
	// Show interactive list to select repository to favorite
	m, err := cli.NewRepoList(false)
	if err != nil {
		return err
	}

	p := tea.NewProgram(m)

	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	repoModel := finalModel.(cli.RepoListModel)
	selected := repoModel.GetSelectedRepo()

	if selected != nil {
		if err := core.SetFavoriteByURL(selected.URL, true); err != nil {
			return err
		}
		fmt.Printf("✓ Marked %s as favorite\n", selected.URL)
	}

	return nil
}

func cmdUnfavorite(_ []string) error {
	// Show interactive list of favorites to unfavorite
	m, err := cli.NewRepoList(true)
	if err != nil {
		return err
	}

	p := tea.NewProgram(m)

	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	repoModel := finalModel.(cli.RepoListModel)
	selected := repoModel.GetSelectedRepo()

	if selected != nil {
		if err := core.SetFavoriteByURL(selected.URL, false); err != nil {
			return err
		}
		fmt.Printf("✓ Removed favorite from %s\n", selected.URL)
	}

	return nil
}

func cmdRemove(_ []string) error {
	// Use interactive bubbles UI
	m, err := cli.NewRepoList(false)
	if err != nil {
		return err
	}

	p := tea.NewProgram(m)

	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	repoModel := finalModel.(cli.RepoListModel)
	selected := repoModel.GetSelectedRepo()

	if selected != nil {
		fmt.Printf("Removing repository: %s\n", selected.URL)

		return core.RemoveRepo(selected.URL)
	}

	return nil
}

func cmdMap(args []string) error {
	return core.MapRepos(args)
}

func cmdOpen(_ []string) error {
	// Show interactive list to select repository to open
	m, err := cli.NewRepoList(false)
	if err != nil {
		return err
	}

	p := tea.NewProgram(m)

	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	repoModel := finalModel.(cli.RepoListModel)
	selected := repoModel.GetSelectedRepo()

	if selected == nil {
		return nil
	}

	// Get editor from config
	db := database.GetDB()

	cfg, err := db.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	if cfg.Editor == "" {
		return fmt.Errorf("no editor configured. Run 'clonr configure' to set an editor")
	}

	// Open the repository in the configured editor
	fmt.Printf("Opening %s in %s...\n", selected.Path, cfg.Editor)
	cmd := exec.Command(cfg.Editor, selected.Path)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to open editor: %w", err)
	}

	fmt.Printf("✓ Opened %s\n", selected.URL)

	return nil
}

func cmdUpdate(_ []string) error {
	fmt.Println("Update command - to be implemented")

	return nil
}

func cmdStatus(_ []string) error {
	fmt.Println("Status command - to be implemented")

	return nil
}

func cmdNerds(_ []string) error {
	fmt.Println("Nerds command - to be implemented")

	return nil
}

func cmdConfigure(args []string) error {
	// Handle flags
	if len(args) > 0 {
		switch args[0] {
		case "--show", "-s":
			return core.ShowConfig()
		case "--reset", "-r":
			return core.ResetConfig()
		}
	}

	// Show current config before interactive form
	if err := core.ShowConfig(); err != nil {
		// If there's an error showing config, continue anyway with defaults
		fmt.Println("No configuration found, using defaults.")
	}
	fmt.Println("\nStarting interactive configuration...\n")

	m, err := cli.NewConfigureModel()
	if err != nil {
		return err
	}

	p := tea.NewProgram(&m)

	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	// Check for errors in the final model
	configModel := finalModel.(cli.ConfigureModel)
	if configModel.Err != nil {
		return configModel.Err
	}

	return nil
}

func cmdServer(args []string) error {
	return server.StartServer(args)
}

func cmdClone(args []string) error {
	// Prepare a clone operation
	repoURL, targetPath, err := core.PrepareClonePath(args)
	if err != nil {
		return err
	}

	// Run clone with progress UI
	m := cli.NewCloneModel(repoURL.String(), targetPath)
	p := tea.NewProgram(m)

	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	cloneModel := finalModel.(cli.CloneModel)
	if cloneModel.Error() != nil {
		return cloneModel.Error()
	}

	// Save to a database after a successful clone
	return core.SaveClonedRepo(repoURL, targetPath)
}

func printUsage() {
	fmt.Println(`clonr - A Git Repository Manager

Usage:
  clonr [command] [arguments] [flags]
  clonr <url> [destination]    Clone a repository directly

  Run 'clonr' without arguments to enter interactive mode.

Available Commands:
  <url> [dest]         Clone a repository (supports https, http, git, ssh, ftp, sftp, git@)
  add <path>           Register an existing local Git repository
  list                 Interactively list all repositories
  favorite             Mark a repository as favorite (interactive)
  unfavorite           Remove favorite mark from a repository (interactive)
  remove               Remove repository from management (interactive)
  open                 Open a favorite repository in your configured editor
  map [directory]      Scan directory for existing Git repositories
  update [name]        Pull latest changes for all or specific repository
  status [name]        Show git status of repositories
  nerds [name]         Display repository statistics
  configure            Configure clonr settings (interactive)
  server               Start the API server

Global Flags:
  --help               Show this help message
  --version            Show version information

Command-Specific Flags:
  add:
    -y, --yes          Skip confirmation prompt
    --name <name>      Optional display name

  list:
    --favorites        Show only favorite repositories

  configure:
    --show, -s         Show current configuration
    --reset, -r        Reset configuration to defaults

Examples:
  clonr                                      # Start interactive menu
  clonr https://github.com/user/repo         # Clone directly
  clonr https://github.com/user/repo ./code  # Clone to specific directory
  clonr configure                            # Configure settings interactively
  clonr configure --show                     # Show current configuration
  clonr configure --reset                    # Reset to default configuration
  clonr add /path/to/repo -y
  clonr list --favorites
  clonr favorite                             # Interactive selection to mark as favorite
  clonr unfavorite                           # Interactive selection to unmark favorite
  clonr remove                               # Interactive selection to remove

For more information, visit: https://github.com/inovacc/clonr`)
}
