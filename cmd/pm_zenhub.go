package cmd

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/inovacc/clonr/internal/core"
	"github.com/spf13/cobra"
)

var zenhubCmd = &cobra.Command{
	Use:   "zenhub",
	Short: "ZenHub operations for project management",
	Long: `Interact with ZenHub boards, issues, epics, and workspaces.

Available Commands:
  issues        List issues with ZenHub enrichment (pipeline, estimate)
  epic          View epic with children and progress
  move          Move issue to a different pipeline
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
  clonr pm zenhub issues owner/repo
  clonr pm zenhub issues owner/repo --pipeline "In Progress"
  clonr pm zenhub epic owner/repo 42
  clonr pm zenhub move owner/repo 123 --pipeline "In Review"
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

var zenhubIssuesCmd = &cobra.Command{
	Use:   "issues [owner/repo]",
	Short: "List issues with ZenHub enrichment",
	Long: `List GitHub issues enriched with ZenHub data (pipeline, estimate, epic status).

Shows:
  - Issue number, title, state
  - ZenHub pipeline (board column)
  - Story points estimate
  - Labels and assignees

Filters:
  --pipeline    Filter by ZenHub pipeline name
  --estimate    Filter by estimate range (e.g., "3-8" or "5+")
  --epic        Filter by parent epic number
  --state       Filter by GitHub state (open, closed, all)
  --labels      Filter by GitHub labels

Examples:
  clonr pm zenhub issues owner/repo
  clonr pm zenhub issues owner/repo --pipeline "In Progress"
  clonr pm zenhub issues owner/repo --estimate "3-8"
  clonr pm zenhub issues owner/repo --epic 42
  clonr pm zenhub issues owner/repo --state all --json`,
	RunE: runZenHubIssues,
}

var zenhubEpicDetailCmd = &cobra.Command{
	Use:   "epic [owner/repo] <issue-number>",
	Short: "Show epic with children and progress",
	Long: `View a ZenHub epic with all its child issues and progress tracking.

Shows:
  - Epic title, state, pipeline, estimate
  - Progress bar (completed points / total points)
  - Child issues with their pipeline, estimate, and state
  - Open/closed counts

Examples:
  clonr pm zenhub epic owner/repo 42
  clonr pm zenhub epic 42 --repo owner/repo
  clonr pm zenhub epic owner/repo 42 --include-closed
  clonr pm zenhub epic owner/repo 42 --json`,
	RunE: runZenHubEpicDetail,
}

var zenhubMoveCmd = &cobra.Command{
	Use:   "move [owner/repo] <issue-number>",
	Short: "Move issue to a different pipeline",
	Long: `Move a GitHub issue to a different ZenHub pipeline (board column).

The issue will be moved to the specified pipeline at the given position.
Position can be "top", "bottom", or a numeric index (0-based).

Examples:
  clonr pm zenhub move owner/repo 123 --pipeline "In Review"
  clonr pm zenhub move 123 --repo owner/repo --pipeline "Done"
  clonr pm zenhub move owner/repo 123 --pipeline "In Progress" --position bottom
  clonr pm zenhub move owner/repo 123 --pipeline "Backlog" --position 0`,
	RunE: runZenHubMove,
}

func init() {
	pmCmd.AddCommand(zenhubCmd)
	zenhubCmd.AddCommand(zenhubBoardCmd)
	zenhubCmd.AddCommand(zenhubEpicsCmd)
	zenhubCmd.AddCommand(zenhubIssueCmd)
	zenhubCmd.AddCommand(zenhubWorkspacesCmd)
	zenhubCmd.AddCommand(zenhubAuthCmd)
	zenhubCmd.AddCommand(zenhubIssuesCmd)
	zenhubCmd.AddCommand(zenhubEpicDetailCmd)
	zenhubCmd.AddCommand(zenhubMoveCmd)

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

	// Issues (enriched) flags
	addPMCommonFlags(zenhubIssuesCmd)
	zenhubIssuesCmd.Flags().String("repo", "", "Repository (owner/repo)")
	zenhubIssuesCmd.Flags().String("gh-token", "", "GitHub token (for repo ID lookup)")
	zenhubIssuesCmd.Flags().String("pipeline", "", "Filter by ZenHub pipeline name")
	zenhubIssuesCmd.Flags().String("estimate", "", "Filter by estimate range (e.g., '3-8' or '5+')")
	zenhubIssuesCmd.Flags().Int("epic", 0, "Filter by parent epic number")
	zenhubIssuesCmd.Flags().String("state", "open", "GitHub state filter (open, closed, all)")
	zenhubIssuesCmd.Flags().StringSlice("labels", nil, "Filter by GitHub labels")
	zenhubIssuesCmd.Flags().Int("limit", 0, "Max issues to return (0 = unlimited)")

	// Epic detail flags
	addPMCommonFlags(zenhubEpicDetailCmd)
	zenhubEpicDetailCmd.Flags().String("repo", "", "Repository (owner/repo)")
	zenhubEpicDetailCmd.Flags().String("gh-token", "", "GitHub token (for repo ID lookup)")
	zenhubEpicDetailCmd.Flags().Bool("include-closed", true, "Include closed child issues")

	// Move flags
	addPMCommonFlags(zenhubMoveCmd)
	zenhubMoveCmd.Flags().String("repo", "", "Repository (owner/repo)")
	zenhubMoveCmd.Flags().String("gh-token", "", "GitHub token (for repo ID lookup)")
	zenhubMoveCmd.Flags().String("pipeline", "", "Target pipeline name (required)")
	zenhubMoveCmd.Flags().String("position", "top", "Position in pipeline (top, bottom, or numeric index)")
	_ = zenhubMoveCmd.MarkFlagRequired("pipeline")
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
	ghToken, _, err := core.ResolveGitHubToken(ghTokenFlag, "")
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
	ghToken, _, err := core.ResolveGitHubToken(ghTokenFlag, "")
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
	ghToken, _, err := core.ResolveGitHubToken(ghTokenFlag, "")
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
		_, _ = fmt.Fprintf(os.Stdout, "Estimate:  %d points\n", issue.Estimate.Value)
	} else {
		_, _ = fmt.Fprintf(os.Stdout, "Estimate:  Not set\n")
	}

	if issue.Pipeline != nil {
		_, _ = fmt.Fprintf(os.Stdout, "Pipeline:  %s\n", issue.Pipeline.Name)
	} else if len(issue.Pipelines) > 0 {
		_, _ = fmt.Fprintf(os.Stdout, "Pipelines:\n")
		for _, p := range issue.Pipelines {
			_, _ = fmt.Fprintf(os.Stdout, "  - %s\n", p.Name)
		}
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
	ghToken, _, err := core.ResolveGitHubToken(ghTokenFlag, "")
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

//nolint:maintidx // CLI command handlers are inherently complex with flag handling
func runZenHubIssues(cmd *cobra.Command, args []string) error {
	// Get flags
	tokenFlag, _ := cmd.Flags().GetString("token")
	ghTokenFlag, _ := cmd.Flags().GetString("gh-token")
	repoFlag, _ := cmd.Flags().GetString("repo")
	outputJson, _ := cmd.Flags().GetBool("json")
	pipelineFilter, _ := cmd.Flags().GetString("pipeline")
	estimateFilter, _ := cmd.Flags().GetString("estimate")
	epicFilter, _ := cmd.Flags().GetInt("epic")
	stateFilter, _ := cmd.Flags().GetString("state")
	labelsFilter, _ := cmd.Flags().GetStringSlice("labels")
	limitFlag, _ := cmd.Flags().GetInt("limit")

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
	ghToken, _, err := core.ResolveGitHubToken(ghTokenFlag, "")
	if err != nil {
		return fmt.Errorf("GitHub token required: %w", err)
	}

	// Detect repository
	owner, repo, err := core.DetectRepository(repoArg, repoFlag)
	if err != nil {
		return fmt.Errorf("could not determine repository: %w\n\nSpecify a repository with: clonr pm zenhub issues owner/repo", err)
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
		_, _ = fmt.Fprintf(os.Stderr, "Fetching enriched issues...\n")
	}

	// Parse estimate filter
	var minEst, maxEst *int
	if estimateFilter != "" {
		minEst, maxEst, err = parseEstimateFilter(estimateFilter)
		if err != nil {
			return fmt.Errorf("invalid estimate filter: %w", err)
		}
	}

	// Build epic filter
	var epicNum *int
	if epicFilter > 0 {
		epicNum = &epicFilter
	}

	// Fetch enriched issues
	opts := core.GetEnrichedIssuesOptions{
		State:       stateFilter,
		Labels:      labelsFilter,
		Pipeline:    pipelineFilter,
		MinEstimate: minEst,
		MaxEstimate: maxEst,
		EpicNumber:  epicNum,
		Limit:       limitFlag,
		Logger:      logger,
	}

	issues, err := core.GetEnrichedIssues(zhClient, ghToken, owner, repo, repoID, opts)
	if err != nil {
		return fmt.Errorf("failed to fetch enriched issues: %w", err)
	}

	// Output results
	if outputJson {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")

		return enc.Encode(issues)
	}

	// Text output - group by pipeline
	if issues.TotalCount == 0 {
		_, _ = fmt.Fprintf(os.Stdout, "No issues found matching filters\n")
		return nil
	}

	_, _ = fmt.Fprintf(os.Stdout, "\nZenHub Issues: %s/%s\n", owner, repo)

	// Group by pipeline
	pipelineIssues := make(map[string][]core.EnrichedIssue)
	pipelineOrder := make([]string, 0)
	totalPoints := 0

	for _, issue := range issues.Issues {
		pipeline := issue.Pipeline
		if pipeline == "" {
			pipeline = "(No Pipeline)"
		}

		if _, exists := pipelineIssues[pipeline]; !exists {
			pipelineOrder = append(pipelineOrder, pipeline)
		}

		pipelineIssues[pipeline] = append(pipelineIssues[pipeline], issue)

		if issue.Estimate != nil {
			totalPoints += *issue.Estimate
		}
	}

	// Print each pipeline
	for _, pipeline := range pipelineOrder {
		pipelineItems := pipelineIssues[pipeline]
		pipelinePoints := 0

		for _, issue := range pipelineItems {
			if issue.Estimate != nil {
				pipelinePoints += *issue.Estimate
			}
		}

		pointsStr := ""
		if pipelinePoints > 0 {
			pointsStr = fmt.Sprintf(", %d pts", pipelinePoints)
		}

		_, _ = fmt.Fprintf(os.Stdout, "\nPipeline: %s (%d issues%s)\n\n", pipeline, len(pipelineItems), pointsStr)
		_, _ = fmt.Fprintf(os.Stdout, "  %-5s │ %-40s │ %-6s │ %-12s │ %s\n", "#", "Title", "Points", "Assignee", "Labels")
		_, _ = fmt.Fprintf(os.Stdout, "  ──────┼──────────────────────────────────────────┼────────┼──────────────┼────────────\n")

		for _, issue := range pipelineItems {
			title := issue.Title
			if len(title) > 40 {
				title = title[:37] + "..."
			}

			estStr := "-"
			if issue.Estimate != nil {
				estStr = fmt.Sprintf("%d", *issue.Estimate)
			}

			assignee := ""
			if len(issue.Assignees) > 0 {
				assignee = "@" + issue.Assignees[0]
				if len(issue.Assignees) > 1 {
					assignee += fmt.Sprintf("+%d", len(issue.Assignees)-1)
				}
			}

			if len(assignee) > 12 {
				assignee = assignee[:11] + "…"
			}

			labels := ""
			if len(issue.Labels) > 0 {
				labels = issue.Labels[0]
				if len(issue.Labels) > 1 {
					labels += fmt.Sprintf("+%d", len(issue.Labels)-1)
				}
			}

			_, _ = fmt.Fprintf(os.Stdout, "  %-5d │ %-40s │ %-6s │ %-12s │ %s\n",
				issue.Number, title, estStr, assignee, labels)
		}
	}

	_, _ = fmt.Fprintf(os.Stdout, "\nTotal: %d issues, %d points\n", issues.TotalCount, totalPoints)

	return nil
}

// parseEstimateFilter parses estimate filter like "3-8", "5+", or "3"
func parseEstimateFilter(filter string) (minEst, maxEst *int, err error) {
	// Check for range format "min-max"
	if idx := len(filter) - 1; idx > 0 && filter[idx] == '+' {
		// Format: "N+" means N or more
		var n int
		if _, scanErr := fmt.Sscanf(filter[:idx+1], "%d+", &n); scanErr != nil {
			return nil, nil, fmt.Errorf("invalid format: %s", filter)
		}

		return &n, nil, nil
	}

	// Check for range format
	var minVal, maxVal int
	if n, _ := fmt.Sscanf(filter, "%d-%d", &minVal, &maxVal); n == 2 {
		return &minVal, &maxVal, nil
	}

	// Single value
	var val int
	if _, scanErr := fmt.Sscanf(filter, "%d", &val); scanErr != nil {
		return nil, nil, fmt.Errorf("invalid format: %s", filter)
	}

	return &val, &val, nil
}

func runZenHubEpicDetail(cmd *cobra.Command, args []string) error {
	// Get flags
	tokenFlag, _ := cmd.Flags().GetString("token")
	ghTokenFlag, _ := cmd.Flags().GetString("gh-token")
	repoFlag, _ := cmd.Flags().GetString("repo")
	outputJson, _ := cmd.Flags().GetBool("json")
	includeClosed, _ := cmd.Flags().GetBool("include-closed")

	// Parse arguments - can be "owner/repo 42" or "42" with --repo flag
	var repoArg string

	var epicNumber int

	switch {
	case len(args) >= 2:
		repoArg = args[0]
		if _, err := fmt.Sscanf(args[1], "%d", &epicNumber); err != nil {
			return fmt.Errorf("invalid epic number: %s", args[1])
		}
	case len(args) == 1:
		if _, err := fmt.Sscanf(args[0], "%d", &epicNumber); err != nil {
			// First arg is not a number, might be repo
			return fmt.Errorf("epic number is required\n\nUsage: clonr pm zenhub epic owner/repo <epic-number>")
		}
	default:
		return fmt.Errorf("epic number is required\n\nUsage: clonr pm zenhub epic owner/repo <epic-number>")
	}

	// Resolve ZenHub token
	zhToken, _, err := core.ResolveZenHubToken(tokenFlag)
	if err != nil {
		return err
	}

	// Resolve GitHub token
	ghToken, _, err := core.ResolveGitHubToken(ghTokenFlag, "")
	if err != nil {
		return fmt.Errorf("GitHub token required: %w", err)
	}

	// Detect repository
	owner, repo, err := core.DetectRepository(repoArg, repoFlag)
	if err != nil {
		return fmt.Errorf("could not determine repository: %w\n\nSpecify a repository with: clonr pm zenhub epic owner/repo <epic-number>", err)
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
		_, _ = fmt.Fprintf(os.Stderr, "Fetching epic #%d with children...\n", epicNumber)
	}

	// Fetch epic with children
	opts := core.GetEpicWithChildrenOptions{
		IncludeClosedChildren: includeClosed,
		Logger:                logger,
	}

	epic, err := core.GetEpicWithChildren(zhClient, ghToken, owner, repo, repoID, epicNumber, opts)
	if err != nil {
		return fmt.Errorf("failed to fetch epic: %w", err)
	}

	// Output results
	if outputJson {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")

		return enc.Encode(epic)
	}

	// Text output
	_, _ = fmt.Fprintf(os.Stdout, "\nEpic #%d: %s\n\n", epic.Number, epic.Title)

	if epic.Pipeline != "" {
		_, _ = fmt.Fprintf(os.Stdout, "Pipeline:   %s\n", epic.Pipeline)
	}

	if epic.Estimate != nil {
		_, _ = fmt.Fprintf(os.Stdout, "Estimate:   %d points\n", *epic.Estimate)
	}

	_, _ = fmt.Fprintf(os.Stdout, "State:      %s\n", epic.State)
	_, _ = fmt.Fprintf(os.Stdout, "URL:        %s\n", epic.URL)

	// Progress bar
	if epic.TotalPoints > 0 {
		_, _ = fmt.Fprintf(os.Stdout, "\nProgress: %d/%d points (%d%%)\n",
			epic.CompletedPoints, epic.TotalPoints,
			epic.CompletedPoints*100/epic.TotalPoints)

		// Draw progress bar
		barWidth := 20
		filled := epic.CompletedPoints * barWidth / epic.TotalPoints

		var barBuilder strings.Builder
		barBuilder.Grow(barWidth * 3) // Each character is multi-byte UTF-8

		for i := range barWidth {
			if i < filled {
				barBuilder.WriteString("█")
			} else {
				barBuilder.WriteString("░")
			}
		}

		_, _ = fmt.Fprintf(os.Stdout, "          %s\n", barBuilder.String())
	}

	// Children table
	_, _ = fmt.Fprintf(os.Stdout, "\nChildren (%d issues):\n\n", epic.ChildCount)

	if epic.ChildCount > 0 {
		_, _ = fmt.Fprintf(os.Stdout, "  %-5s │ %-35s │ %-12s │ %-6s │ %s\n", "#", "Title", "Pipeline", "Points", "State")
		_, _ = fmt.Fprintf(os.Stdout, "  ──────┼─────────────────────────────────────┼──────────────┼────────┼───────\n")

		for _, child := range epic.Children {
			title := child.Title
			if len(title) > 35 {
				title = title[:32] + "..."
			}

			pipeline := child.Pipeline
			if len(pipeline) > 12 {
				pipeline = pipeline[:11] + "…"
			}

			estStr := "-"
			if child.Estimate != nil {
				estStr = fmt.Sprintf("%d", *child.Estimate)
			}

			_, _ = fmt.Fprintf(os.Stdout, "  %-5d │ %-35s │ %-12s │ %-6s │ %s\n",
				child.Number, title, pipeline, estStr, child.State)
		}
	}

	return nil
}

func runZenHubMove(cmd *cobra.Command, args []string) error {
	// Get flags
	tokenFlag, _ := cmd.Flags().GetString("token")
	ghTokenFlag, _ := cmd.Flags().GetString("gh-token")
	repoFlag, _ := cmd.Flags().GetString("repo")
	outputJson, _ := cmd.Flags().GetBool("json")
	pipelineFlag, _ := cmd.Flags().GetString("pipeline")
	positionFlag, _ := cmd.Flags().GetString("position")

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
			return fmt.Errorf("issue number is required\n\nUsage: clonr pm zenhub move owner/repo <issue-number> --pipeline <name>")
		}
	default:
		return fmt.Errorf("issue number is required\n\nUsage: clonr pm zenhub move owner/repo <issue-number> --pipeline <name>")
	}

	// Resolve ZenHub token
	zhToken, _, err := core.ResolveZenHubToken(tokenFlag)
	if err != nil {
		return err
	}

	// Resolve GitHub token
	ghToken, _, err := core.ResolveGitHubToken(ghTokenFlag, "")
	if err != nil {
		return fmt.Errorf("GitHub token required: %w", err)
	}

	// Detect repository
	owner, repo, err := core.DetectRepository(repoArg, repoFlag)
	if err != nil {
		return fmt.Errorf("could not determine repository: %w\n\nSpecify a repository with: clonr pm zenhub move owner/repo <issue-number> --pipeline <name>", err)
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
		_, _ = fmt.Fprintf(os.Stderr, "Moving issue #%d to pipeline \"%s\"...\n", issueNumber, pipelineFlag)
	}

	// Move issue
	opts := core.MoveIssueOptions{
		Position: positionFlag,
		Logger:   logger,
	}

	result, err := core.MoveIssue(zhClient, repoID, issueNumber, pipelineFlag, opts)
	if err != nil {
		return fmt.Errorf("failed to move issue: %w", err)
	}

	// Output results
	if outputJson {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")

		return enc.Encode(result)
	}

	// Text output
	_, _ = fmt.Fprintf(os.Stdout, "\nMoved issue #%d\n\n", result.IssueNumber)

	if result.FromPipeline != "" {
		_, _ = fmt.Fprintf(os.Stdout, "From: %s\n", result.FromPipeline)
	}

	_, _ = fmt.Fprintf(os.Stdout, "To:   %s\n", result.ToPipeline)
	_, _ = fmt.Fprintf(os.Stdout, "Position: %s\n", result.Position)

	return nil
}
