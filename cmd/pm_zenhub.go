package cmd

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/inovacc/clonr/internal/core"
	"github.com/spf13/cobra"
)

var zenhubCmd = &cobra.Command{
	Use:   "zenhub",
	Short: "ZenHub operations for project management",
	Long: `Interact with ZenHub boards, issues, epics, and workspaces.

Available Commands:
  board         View ZenHub board state
  epics         List ZenHub epics
  issue         View ZenHub issue details (estimate, pipeline)
  workspaces    List ZenHub workspaces for a repository

ZenHub works with GitHub repositories. You need both:
  - ZenHub API token (for ZenHub data)
  - GitHub token (for repository ID lookup)

Authentication:
  ZenHub token from (in priority order):
  1. --token flag
  2. ZENHUB_TOKEN environment variable
  3. ~/.config/clonr/zenhub.json config file

  GitHub token is used for repository ID lookup (auto-detected).

Examples:
  clonr pm zenhub board owner/repo
  clonr pm zenhub epics owner/repo
  clonr pm zenhub issue owner/repo 123
  clonr pm zenhub workspaces owner/repo`,
}

var zenhubBoardCmd = &cobra.Command{
	Use:   "board [owner/repo]",
	Short: "View ZenHub board state",
	Long: `Display the ZenHub board showing all pipelines and issue counts.

Shows:
  - All pipelines (columns) with issue counts
  - Total story points per pipeline
  - Optionally full issue details with --details flag

Examples:
  clonr pm zenhub board owner/repo
  clonr pm zenhub board owner/repo --details
  clonr pm zenhub board --repo owner/repo --json`,
	RunE: runZenHubBoard,
}

var zenhubEpicsCmd = &cobra.Command{
	Use:   "epics [owner/repo]",
	Short: "List ZenHub epics",
	Long: `List all epics in a repository.

Examples:
  clonr pm zenhub epics owner/repo
  clonr pm zenhub epics --repo owner/repo --json`,
	RunE: runZenHubEpics,
}

var zenhubIssueCmd = &cobra.Command{
	Use:   "issue [owner/repo] <issue-number>",
	Short: "View ZenHub issue details",
	Long: `View ZenHub-specific data for a GitHub issue.

Shows:
  - Estimate (story points)
  - Pipeline (board column)
  - Whether it's an epic

Examples:
  clonr pm zenhub issue owner/repo 123
  clonr pm zenhub issue 123 --repo owner/repo
  clonr pm zenhub issue owner/repo 123 --json`,
	RunE: runZenHubIssue,
}

var zenhubWorkspacesCmd = &cobra.Command{
	Use:   "workspaces [owner/repo]",
	Short: "List ZenHub workspaces",
	Long: `List all ZenHub workspaces that include a repository.

Shows:
  - Workspace ID
  - Workspace name
  - Description (if available)

Examples:
  clonr pm zenhub workspaces owner/repo
  clonr pm zenhub workspaces --repo owner/repo --json`,
	RunE: runZenHubWorkspaces,
}

var zenhubAuthCmd = &cobra.Command{
	Use:   "auth",
	Short: "Open ZenHub and GitHub token pages in browser",
	Long: `Open the token settings pages for ZenHub and GitHub in your default browser.

This command helps you quickly access the token generation pages:
  - ZenHub: https://app.zenhub.com/settings/tokens
  - GitHub: https://github.com/settings/tokens

Use --zenhub or --github to open only one page.

Examples:
  clonr pm zenhub auth           # Opens both ZenHub and GitHub token pages
  clonr pm zenhub auth --zenhub  # Opens only ZenHub token page
  clonr pm zenhub auth --github  # Opens only GitHub token page`,
	RunE: runZenHubAuth,
}

func init() {
	pmCmd.AddCommand(zenhubCmd)
	zenhubCmd.AddCommand(zenhubBoardCmd)
	zenhubCmd.AddCommand(zenhubEpicsCmd)
	zenhubCmd.AddCommand(zenhubIssueCmd)
	zenhubCmd.AddCommand(zenhubWorkspacesCmd)
	zenhubCmd.AddCommand(zenhubAuthCmd)

	// Board flags
	addPMCommonFlags(zenhubBoardCmd)
	zenhubBoardCmd.Flags().String("repo", "", "Repository (owner/repo)")
	zenhubBoardCmd.Flags().String("gh-token", "", "GitHub token (for repo ID lookup)")
	zenhubBoardCmd.Flags().Bool("details", false, "Include issue details in output")

	// Epics flags
	addPMCommonFlags(zenhubEpicsCmd)
	zenhubEpicsCmd.Flags().String("repo", "", "Repository (owner/repo)")
	zenhubEpicsCmd.Flags().String("gh-token", "", "GitHub token (for repo ID lookup)")

	// Issue flags
	addPMCommonFlags(zenhubIssueCmd)
	zenhubIssueCmd.Flags().String("repo", "", "Repository (owner/repo)")
	zenhubIssueCmd.Flags().String("gh-token", "", "GitHub token (for repo ID lookup)")

	// Workspaces flags
	addPMCommonFlags(zenhubWorkspacesCmd)
	zenhubWorkspacesCmd.Flags().String("repo", "", "Repository (owner/repo)")
	zenhubWorkspacesCmd.Flags().String("gh-token", "", "GitHub token (for repo ID lookup)")

	// Auth flags
	zenhubAuthCmd.Flags().Bool("zenhub", false, "Open only ZenHub token page")
	zenhubAuthCmd.Flags().Bool("github", false, "Open only GitHub token page")
}

func runZenHubBoard(cmd *cobra.Command, args []string) error {
	// Get flags
	tokenFlag, _ := cmd.Flags().GetString("token")
	ghTokenFlag, _ := cmd.Flags().GetString("gh-token")
	repoFlag, _ := cmd.Flags().GetString("repo")
	outputJson, _ := cmd.Flags().GetBool("json")
	showDetails, _ := cmd.Flags().GetBool("details")

	// Get repo argument
	var repoArg string
	if len(args) > 0 {
		repoArg = args[0]
	}

	// Resolve ZenHub token
	zhToken, _, err := core.ResolveZenHubToken(tokenFlag)
	if err != nil {
		return err
	}

	// Resolve GitHub token for repo ID lookup
	ghToken, _, err := core.ResolveGitHubToken(ghTokenFlag)
	if err != nil {
		return fmt.Errorf("GitHub token required for repository ID lookup: %w", err)
	}

	// Detect repository
	owner, repo, err := core.DetectRepository(repoArg, repoFlag)
	if err != nil {
		return fmt.Errorf("could not determine repository: %w\n\nSpecify a repository with: clonr pm zenhub board owner/repo", err)
	}

	// Setup logger
	var logger *slog.Logger
	if outputJson {
		logger = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	} else {
		logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	}

	// Get GitHub repo ID
	if !outputJson {
		_, _ = fmt.Fprintf(os.Stderr, "Fetching repository ID for %s/%s...\n", owner, repo)
	}

	repoID, err := core.GetGitHubRepoID(ghToken, owner, repo, logger)
	if err != nil {
		return fmt.Errorf("failed to get repository ID: %w", err)
	}

	// Create ZenHub client
	zhClient, err := core.CreateZenHubClient(zhToken, core.ZenHubClientOptions{Logger: logger})
	if err != nil {
		return fmt.Errorf("failed to create ZenHub client: %w", err)
	}

	if !outputJson {
		_, _ = fmt.Fprintf(os.Stderr, "Fetching ZenHub board...\n")
	}

	// Fetch board
	opts := core.GetZenHubBoardOptions{
		IncludeIssueDetails: showDetails,
		Logger:              logger,
	}

	board, err := core.GetZenHubBoard(zhClient, repoID, fmt.Sprintf("%s/%s", owner, repo), opts)
	if err != nil {
		return fmt.Errorf("failed to fetch board: %w", err)
	}

	// Output results
	if outputJson {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")

		return enc.Encode(board)
	}

	// Text output
	_, _ = fmt.Fprintf(os.Stdout, "\nZenHub Board: %s/%s\n\n", owner, repo)

	for _, pipeline := range board.Pipelines {
		pointsStr := ""
		if pipeline.TotalPoints > 0 {
			pointsStr = fmt.Sprintf(" (%d pts)", pipeline.TotalPoints)
		}

		_, _ = fmt.Fprintf(os.Stdout, "📋 %s: %d issues%s\n", pipeline.Name, pipeline.IssueCount, pointsStr)

		// Show issue details if requested
		if showDetails && len(pipeline.Issues) > 0 {
			for _, issue := range pipeline.Issues {
				estStr := ""
				if issue.Estimate != nil {
					estStr = fmt.Sprintf(" [%d pts]", *issue.Estimate)
				}

				epicStr := ""
				if issue.IsEpic {
					epicStr = " (Epic)"
				}

				_, _ = fmt.Fprintf(os.Stdout, "   #%-5d%s%s\n", issue.Number, estStr, epicStr)
			}
		}
	}

	return nil
}

func runZenHubEpics(cmd *cobra.Command, args []string) error {
	// Get flags
	tokenFlag, _ := cmd.Flags().GetString("token")
	ghTokenFlag, _ := cmd.Flags().GetString("gh-token")
	repoFlag, _ := cmd.Flags().GetString("repo")
	outputJson, _ := cmd.Flags().GetBool("json")

	// Get repo argument
	var repoArg string
	if len(args) > 0 {
		repoArg = args[0]
	}

	// Resolve ZenHub token
	zhToken, _, err := core.ResolveZenHubToken(tokenFlag)
	if err != nil {
		return err
	}

	// Resolve GitHub token
	ghToken, _, err := core.ResolveGitHubToken(ghTokenFlag)
	if err != nil {
		return fmt.Errorf("GitHub token required for repository ID lookup: %w", err)
	}

	// Detect repository
	owner, repo, err := core.DetectRepository(repoArg, repoFlag)
	if err != nil {
		return fmt.Errorf("could not determine repository: %w\n\nSpecify a repository with: clonr pm zenhub epics owner/repo", err)
	}

	// Setup logger
	var logger *slog.Logger
	if outputJson {
		logger = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	} else {
		logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	}

	// Get GitHub repo ID
	if !outputJson {
		_, _ = fmt.Fprintf(os.Stderr, "Fetching repository ID for %s/%s...\n", owner, repo)
	}

	repoID, err := core.GetGitHubRepoID(ghToken, owner, repo, logger)
	if err != nil {
		return fmt.Errorf("failed to get repository ID: %w", err)
	}

	// Create ZenHub client
	zhClient, err := core.CreateZenHubClient(zhToken, core.ZenHubClientOptions{Logger: logger})
	if err != nil {
		return fmt.Errorf("failed to create ZenHub client: %w", err)
	}

	if !outputJson {
		_, _ = fmt.Fprintf(os.Stderr, "Fetching ZenHub epics...\n")
	}

	// Fetch epics
	opts := core.GetZenHubEpicsOptions{
		Logger: logger,
	}

	epics, err := core.GetZenHubEpics(zhClient, repoID, fmt.Sprintf("%s/%s", owner, repo), opts)
	if err != nil {
		return fmt.Errorf("failed to fetch epics: %w", err)
	}

	// Output results
	if outputJson {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")

		return enc.Encode(epics)
	}

	// Text output
	if epics.TotalCount == 0 {
		_, _ = fmt.Fprintf(os.Stdout, "No epics found in %s/%s\n", owner, repo)
		return nil
	}

	_, _ = fmt.Fprintf(os.Stdout, "\nZenHub Epics: %s/%s (%d total)\n\n", owner, repo, epics.TotalCount)

	for _, epic := range epics.Epics {
		_, _ = fmt.Fprintf(os.Stdout, "📚 #%d\n", epic.IssueNumber)
	}

	return nil
}

func runZenHubIssue(cmd *cobra.Command, args []string) error {
	// Get flags
	tokenFlag, _ := cmd.Flags().GetString("token")
	ghTokenFlag, _ := cmd.Flags().GetString("gh-token")
	repoFlag, _ := cmd.Flags().GetString("repo")
	outputJson, _ := cmd.Flags().GetBool("json")

	// Parse arguments - can be "owner/repo 123" or "123" with --repo flag
	var repoArg string

	var issueNumber int

	switch {
	case len(args) >= 2:
		repoArg = args[0]
		if _, err := fmt.Sscanf(args[1], "%d", &issueNumber); err != nil {
			return fmt.Errorf("invalid issue number: %s", args[1])
		}
	case len(args) == 1:
		if _, err := fmt.Sscanf(args[0], "%d", &issueNumber); err != nil {
			// First arg is not a number, assume it's a repo without issue number
			return fmt.Errorf("issue number is required\n\nUsage: clonr pm zenhub issue owner/repo <issue-number>")
		}
	default:
		return fmt.Errorf("issue number is required\n\nUsage: clonr pm zenhub issue owner/repo <issue-number>")
	}

	// Resolve ZenHub token
	zhToken, _, err := core.ResolveZenHubToken(tokenFlag)
	if err != nil {
		return err
	}

	// Resolve GitHub token
	ghToken, _, err := core.ResolveGitHubToken(ghTokenFlag)
	if err != nil {
		return fmt.Errorf("GitHub token required for repository ID lookup: %w", err)
	}

	// Detect repository
	owner, repo, err := core.DetectRepository(repoArg, repoFlag)
	if err != nil {
		return fmt.Errorf("could not determine repository: %w\n\nSpecify a repository with: clonr pm zenhub issue owner/repo <issue-number>", err)
	}

	// Setup logger
	var logger *slog.Logger
	if outputJson {
		logger = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	} else {
		logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	}

	// Get GitHub repo ID
	if !outputJson {
		_, _ = fmt.Fprintf(os.Stderr, "Fetching repository ID for %s/%s...\n", owner, repo)
	}

	repoID, err := core.GetGitHubRepoID(ghToken, owner, repo, logger)
	if err != nil {
		return fmt.Errorf("failed to get repository ID: %w", err)
	}

	// Create ZenHub client
	zhClient, err := core.CreateZenHubClient(zhToken, core.ZenHubClientOptions{Logger: logger})
	if err != nil {
		return fmt.Errorf("failed to create ZenHub client: %w", err)
	}

	if !outputJson {
		_, _ = fmt.Fprintf(os.Stderr, "Fetching ZenHub issue data...\n")
	}

	// Fetch issue
	opts := core.GetZenHubIssueOptions{
		Logger: logger,
	}

	issue, err := core.GetZenHubIssue(zhClient, repoID, issueNumber, opts)
	if err != nil {
		return fmt.Errorf("failed to fetch issue: %w", err)
	}

	// Output results
	if outputJson {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")

		return enc.Encode(issue)
	}

	// Text output
	_, _ = fmt.Fprintf(os.Stdout, "\nZenHub Issue: %s/%s#%d\n\n", owner, repo, issueNumber)

	if issue.Estimate != nil {
		_, _ = fmt.Fprintf(os.Stdout, "Estimate:  %d points\n", *issue.Estimate)
	} else {
		_, _ = fmt.Fprintf(os.Stdout, "Estimate:  Not set\n")
	}

	if issue.Pipeline != "" {
		_, _ = fmt.Fprintf(os.Stdout, "Pipeline:  %s\n", issue.Pipeline)
	}

	if issue.IsEpic {
		_, _ = fmt.Fprintf(os.Stdout, "Type:      Epic\n")
	}

	return nil
}

func runZenHubAuth(cmd *cobra.Command, args []string) error {
	zenhubOnly, _ := cmd.Flags().GetBool("zenhub")
	githubOnly, _ := cmd.Flags().GetBool("github")

	// If neither flag is set, open both
	openBoth := !zenhubOnly && !githubOnly

	if zenhubOnly || openBoth {
		_, _ = fmt.Fprintf(os.Stdout, "Opening ZenHub token page: %s\n", core.ZenHubTokenURL)
		if err := core.OpenZenHubTokenPage(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Failed to open browser: %v\n", err)
			_, _ = fmt.Fprintf(os.Stdout, "Please visit: %s\n", core.ZenHubTokenURL)
		}
	}

	if githubOnly || openBoth {
		_, _ = fmt.Fprintf(os.Stdout, "Opening GitHub token page: %s\n", core.GitHubTokenURL)
		if err := core.OpenGitHubTokenPage(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Failed to open browser: %v\n", err)
			_, _ = fmt.Fprintf(os.Stdout, "Please visit: %s\n", core.GitHubTokenURL)
		}
	}

	_, _ = fmt.Fprintf(os.Stdout, "\nAfter creating tokens, configure them:\n")
	_, _ = fmt.Fprintf(os.Stdout, "  ZenHub: export ZENHUB_TOKEN=<token>\n")
	_, _ = fmt.Fprintf(os.Stdout, "      or: echo '{\"token\": \"<token>\"}' > ~/.config/clonr/zenhub.json\n")
	_, _ = fmt.Fprintf(os.Stdout, "  GitHub: export GITHUB_TOKEN=<token>\n")
	_, _ = fmt.Fprintf(os.Stdout, "      or: gh auth login\n")

	return nil
}

func runZenHubWorkspaces(cmd *cobra.Command, args []string) error {
	// Get flags
	tokenFlag, _ := cmd.Flags().GetString("token")
	ghTokenFlag, _ := cmd.Flags().GetString("gh-token")
	repoFlag, _ := cmd.Flags().GetString("repo")
	outputJson, _ := cmd.Flags().GetBool("json")

	// Get repo argument
	var repoArg string
	if len(args) > 0 {
		repoArg = args[0]
	}

	// Resolve ZenHub token
	zhToken, _, err := core.ResolveZenHubToken(tokenFlag)
	if err != nil {
		return err
	}

	// Resolve GitHub token
	ghToken, _, err := core.ResolveGitHubToken(ghTokenFlag)
	if err != nil {
		return fmt.Errorf("GitHub token required for repository ID lookup: %w", err)
	}

	// Detect repository
	owner, repo, err := core.DetectRepository(repoArg, repoFlag)
	if err != nil {
		return fmt.Errorf("could not determine repository: %w\n\nSpecify a repository with: clonr pm zenhub workspaces owner/repo", err)
	}

	// Setup logger
	var logger *slog.Logger
	if outputJson {
		logger = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	} else {
		logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	}

	// Get GitHub repo ID
	if !outputJson {
		_, _ = fmt.Fprintf(os.Stderr, "Fetching repository ID for %s/%s...\n", owner, repo)
	}

	repoID, err := core.GetGitHubRepoID(ghToken, owner, repo, logger)
	if err != nil {
		return fmt.Errorf("failed to get repository ID: %w", err)
	}

	// Create ZenHub client
	zhClient, err := core.CreateZenHubClient(zhToken, core.ZenHubClientOptions{Logger: logger})
	if err != nil {
		return fmt.Errorf("failed to create ZenHub client: %w", err)
	}

	if !outputJson {
		_, _ = fmt.Fprintf(os.Stderr, "Fetching ZenHub workspaces...\n")
	}

	// Fetch workspaces
	opts := core.GetZenHubWorkspacesOptions{
		Logger: logger,
	}

	workspaces, err := core.GetZenHubWorkspaces(zhClient, repoID, fmt.Sprintf("%s/%s", owner, repo), opts)
	if err != nil {
		return fmt.Errorf("failed to fetch workspaces: %w", err)
	}

	// Output results
	if outputJson {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")

		return enc.Encode(workspaces)
	}

	// Text output
	if workspaces.TotalCount == 0 {
		_, _ = fmt.Fprintf(os.Stdout, "No workspaces found for %s/%s\n", owner, repo)
		return nil
	}

	_, _ = fmt.Fprintf(os.Stdout, "\nZenHub Workspaces for %s/%s (%d total)\n\n", owner, repo, workspaces.TotalCount)

	for _, ws := range workspaces.Workspaces {
		_, _ = fmt.Fprintf(os.Stdout, "🗂️  %s\n", ws.Name)
		_, _ = fmt.Fprintf(os.Stdout, "   ID: %s\n", ws.ID)

		if ws.Description != "" {
			_, _ = fmt.Fprintf(os.Stdout, "   Description: %s\n", ws.Description)
		}

		if len(ws.Repositories) > 0 {
			_, _ = fmt.Fprintf(os.Stdout, "   Repositories: %d\n", len(ws.Repositories))
		}

		_, _ = fmt.Fprintf(os.Stdout, "\n")
	}

	return nil
}
