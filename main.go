package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dyammarcano/clonr/internal/cli"
	"github.com/dyammarcano/clonr/internal/core"
	"github.com/dyammarcano/clonr/internal/server"
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
		fmt.Println("clonr version 0.2.0")
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
			_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Execute command
	if err := executeCommand(command, commandArgs); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
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
		} else {
			if choice != "remove" && choice != "list" && choice != "open" {
				fmt.Println("\nPress Enter to continue...")
				_, _ = fmt.Scanln()
			}
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

func cmdList(args []string) error {
	// Interactive mode - show list with actions
	m, err := cli.NewRepoList(*favoritesOnly)
	if err != nil {
		return err
	}

	p := tea.NewProgram(m)
	_, err = p.Run()
	return err
}

func cmdFavorite(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("favorite command requires a URL argument\nUsage: clonr favorite <url>")
	}
	return core.SetFavoriteByURL(args[0], true)
}

func cmdUnfavorite(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("unfavorite command requires a URL argument\nUsage: clonr unfavorite <url>")
	}
	return core.SetFavoriteByURL(args[0], false)
}

func cmdRemove(args []string) error {
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

func cmdOpen(args []string) error {
	fmt.Println("Open command - to be implemented with favorite selection")
	return nil
}

func cmdUpdate(args []string) error {
	fmt.Println("Update command - to be implemented")
	return nil
}

func cmdStatus(args []string) error {
	fmt.Println("Status command - to be implemented")
	return nil
}

func cmdNerds(args []string) error {
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

	p := tea.NewProgram(m)
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
	return core.CloneRepo(args)
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
  favorite <url>       Mark a repository as favorite
  unfavorite <url>     Remove favorite mark from a repository
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
  clonr favorite https://github.com/user/repo
  clonr remove                               # Interactive selection

For more information, visit: https://github.com/dyammarcano/clonr`)
}
