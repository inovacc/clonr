package cmd

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/inovacc/clonr/internal/core"
	"github.com/spf13/cobra"
)

var issuesCmd = &cobra.Command{
	Use:   "issues",
	Short: "Manage GitHub issues",
	Long: `List, create, and close GitHub issues for a repository.

Repository Detection:
  Commands auto-detect the repository from the current directory,
  or you can specify it explicitly as owner/repo.

Examples:
  clonr gh issues list                    # List issues in current repo
  clonr gh issues list owner/repo         # List issues in specified repo
  clonr gh issues create --title "Bug"    # Create an issue
  clonr gh issues close 123               # Close issue #123`,
}

var issuesListCmd = &cobra.Command{
	Use:   "list [owner/repo]",
	Short: "List issues for a repository",
	Long: `List GitHub issues for a repository.

By default, lists open issues. Use --state to filter by state.

Examples:
  clonr gh issues list                           # List open issues
  clonr gh issues list --state all               # List all issues
  clonr gh issues list --state closed            # List closed issues
  clonr gh issues list --labels bug,urgent       # Filter by labels
  clonr gh issues list --assignee @me            # Filter by assignee
  clonr gh issues list --limit 10                # Limit results`,
	RunE: runIssuesList,
}

var issuesCreateCmd = &cobra.Command{
	Use:   "create [owner/repo]",
	Short: "Create a new issue",
	Long: `Create a new GitHub issue in a repository.

The --title flag is required. Body can be provided via --body flag.

Examples:
  clonr gh issues create --title "Bug report"
  clonr gh issues create --title "Feature" --body "Description here"
  clonr gh issues create --title "Bug" --labels bug,critical
  clonr gh issues create owner/repo --title "Issue"`,
	RunE: runIssuesCreate,
}

var issuesCloseCmd = &cobra.Command{
	Use:   "close <issue-number> [owner/repo]",
	Short: "Close an issue",
	Long: `Close a GitHub issue by its number.

The issue number is required as the first argument.

Examples:
  clonr gh issues close 123                # Close issue #123 in current repo
  clonr gh issues close 123 owner/repo     # Close issue in specified repo
  clonr gh issues close 123 --comment "Fixed in v1.0"  # Close with comment`,
	RunE: runIssuesClose,
}

func init() {
	ghCmd.AddCommand(issuesCmd)
	issuesCmd.AddCommand(issuesListCmd)
	issuesCmd.AddCommand(issuesCreateCmd)
	issuesCmd.AddCommand(issuesCloseCmd)

	// List flags
	addGHCommonFlags(issuesListCmd)
	issuesListCmd.Flags().String("state", "open", "Filter by state: open, closed, all")
	issuesListCmd.Flags().StringSlice("labels", nil, "Filter by labels (comma-separated)")
	issuesListCmd.Flags().String("assignee", "", "Filter by assignee (@me for yourself)")
	issuesListCmd.Flags().String("creator", "", "Filter by creator")
	issuesListCmd.Flags().String("sort", "created", "Sort by: created, updated, comments")
	issuesListCmd.Flags().String("order", "desc", "Sort order: asc, desc")
	issuesListCmd.Flags().Int("limit", 30, "Maximum number of issues to list (0 = unlimited)")

	// Create flags
	addGHCommonFlags(issuesCreateCmd)
	issuesCreateCmd.Flags().String("title", "", "Issue title (required)")
	issuesCreateCmd.Flags().String("body", "", "Issue body")
	issuesCreateCmd.Flags().StringSlice("labels", nil, "Labels to add (comma-separated)")
	issuesCreateCmd.Flags().StringSlice("assignees", nil, "Assignees (comma-separated)")

	// Close flags
	addGHCommonFlags(issuesCloseCmd)
	issuesCloseCmd.Flags().String("comment", "", "Add a comment when closing")
	issuesCloseCmd.Flags().String("reason", "completed", "Close reason: completed, not_planned")
}

func runIssuesList(cmd *cobra.Command, args []string) error {
	// Get flags
	tokenFlag, _ := cmd.Flags().GetString("token")
	profileFlag, _ := cmd.Flags().GetString("profile")
	repoFlag, _ := cmd.Flags().GetString("repo")
	outputJson, _ := cmd.Flags().GetBool("json")
	state, _ := cmd.Flags().GetString("state")
	labels, _ := cmd.Flags().GetStringSlice("labels")
	assignee, _ := cmd.Flags().GetString("assignee")
	creator, _ := cmd.Flags().GetString("creator")
	sortBy, _ := cmd.Flags().GetString("sort")
	order, _ := cmd.Flags().GetString("order")
	limit, _ := cmd.Flags().GetInt("limit")

	// Get repo argument if provided
	var repoArg string
	if len(args) > 0 {
		repoArg = args[0]
	}

	// Resolve token
	token, _, err := core.ResolveGitHubToken(tokenFlag, profileFlag)
	if err != nil {
		return err
	}

	// Detect repository
	owner, repo, err := core.DetectRepository(repoArg, repoFlag)
	if err != nil {
		return fmt.Errorf("could not determine repository: %w\n\nSpecify a repository with: clonr gh issues list owner/repo", err)
	}

	// Setup logger
	var logger *slog.Logger
	if outputJson {
		logger = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	} else {
		logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	}

	// Fetch issues
	opts := core.ListIssuesOptions{
		State:    state,
		Labels:   labels,
		Assignee: assignee,
		Creator:  creator,
		Sort:     sortBy,
		Order:    order,
		Limit:    limit,
		Logger:   logger,
	}

	if !outputJson {
		_, _ = fmt.Fprintf(os.Stderr, "Fetching issues for %s/%s...\n", owner, repo)
	}

	issues, err := core.ListIssuesFromAPI(token, owner, repo, opts)
	if err != nil {
		return fmt.Errorf("failed to fetch issues: %w", err)
	}

	// Output results
	if outputJson {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")

		return enc.Encode(issues)
	}

	// Text output
	if len(issues.Issues) == 0 {
		_, _ = fmt.Fprintf(os.Stdout, "No %s issues found in %s/%s\n", state, owner, repo)
		return nil
	}

	_, _ = fmt.Fprintf(os.Stdout, "\nIssues for %s/%s (%d total, %d open, %d closed)\n\n",
		owner, repo, issues.TotalCount, issues.OpenCount, issues.ClosedCount)

	for _, issue := range issues.Issues {
		stateIcon := "🟢"
		if issue.State == "closed" {
			stateIcon = "🟣"
		}

		// Format labels
		labelStr := ""
		if len(issue.Labels) > 0 {
			labelStr = fmt.Sprintf(" [%s]", strings.Join(issue.Labels, ", "))
		}

		// Format age
		age := formatAge(issue.CreatedAt)

		_, _ = fmt.Fprintf(os.Stdout, "%s #%-5d %s%s\n", stateIcon, issue.Number, issue.Title, labelStr)
		_, _ = fmt.Fprintf(os.Stdout, "         opened %s by @%s", age, issue.Author)

		if issue.Comments > 0 {
			_, _ = fmt.Fprintf(os.Stdout, " · %d comments", issue.Comments)
		}

		_, _ = fmt.Fprintln(os.Stdout)
	}

	return nil
}

func runIssuesCreate(cmd *cobra.Command, args []string) error {
	// Get flags
	tokenFlag, _ := cmd.Flags().GetString("token")
	profileFlag, _ := cmd.Flags().GetString("profile")
	repoFlag, _ := cmd.Flags().GetString("repo")
	outputJson, _ := cmd.Flags().GetBool("json")
	title, _ := cmd.Flags().GetString("title")
	body, _ := cmd.Flags().GetString("body")
	labels, _ := cmd.Flags().GetStringSlice("labels")
	assignees, _ := cmd.Flags().GetStringSlice("assignees")

	// Get repo argument if provided
	var repoArg string
	if len(args) > 0 {
		repoArg = args[0]
	}

	// Validate title
	if title == "" {
		return fmt.Errorf("--title is required\n\nUsage: clonr gh issues create --title \"Issue title\"")
	}

	// Resolve token
	token, _, err := core.ResolveGitHubToken(tokenFlag, profileFlag)
	if err != nil {
		return err
	}

	// Detect repository
	owner, repo, err := core.DetectRepository(repoArg, repoFlag)
	if err != nil {
		return fmt.Errorf("could not determine repository: %w\n\nSpecify a repository with: clonr gh issues create owner/repo --title \"title\"", err)
	}

	// Setup logger
	var logger *slog.Logger
	if outputJson {
		logger = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	} else {
		logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	}

	// Create issue
	opts := core.CreateIssueOptions{
		Title:     title,
		Body:      body,
		Labels:    labels,
		Assignees: assignees,
		Logger:    logger,
	}

	if !outputJson {
		_, _ = fmt.Fprintf(os.Stderr, "Creating issue in %s/%s...\n", owner, repo)
	}

	created, err := core.CreateIssue(token, owner, repo, opts)
	if err != nil {
		return fmt.Errorf("failed to create issue: %w", err)
	}

	// Output results
	if outputJson {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")

		return enc.Encode(created)
	}

	_, _ = fmt.Fprintf(os.Stdout, "\n✓ Created issue #%d: %s\n", created.Number, created.Title)
	_, _ = fmt.Fprintf(os.Stdout, "  %s\n", created.URL)

	return nil
}

func runIssuesClose(cmd *cobra.Command, args []string) error {
	// Get flags
	tokenFlag, _ := cmd.Flags().GetString("token")
	profileFlag, _ := cmd.Flags().GetString("profile")
	repoFlag, _ := cmd.Flags().GetString("repo")
	outputJson, _ := cmd.Flags().GetBool("json")
	comment, _ := cmd.Flags().GetString("comment")
	reason, _ := cmd.Flags().GetString("reason")

	// Parse arguments - first should be issue number, optional second is owner/repo
	if len(args) == 0 {
		return fmt.Errorf("issue number is required\n\nUsage: clonr gh issues close <issue-number>")
	}

	issueNumber, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid issue number: %s", args[0])
	}

	var repoArg string
	if len(args) > 1 {
		repoArg = args[1]
	}

	// Resolve token
	token, _, err := core.ResolveGitHubToken(tokenFlag, profileFlag)
	if err != nil {
		return err
	}

	// Detect repository
	owner, repo, err := core.DetectRepository(repoArg, repoFlag)
	if err != nil {
		return fmt.Errorf("could not determine repository: %w\n\nSpecify a repository with: clonr gh issues close <number> owner/repo", err)
	}

	// Setup logger
	var logger *slog.Logger
	if outputJson {
		logger = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	} else {
		logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	}

	// Close issue
	opts := core.CloseIssueOptions{
		Comment: comment,
		Reason:  reason,
		Logger:  logger,
	}

	if !outputJson {
		_, _ = fmt.Fprintf(os.Stderr, "Closing issue #%d in %s/%s...\n", issueNumber, owner, repo)
	}

	closed, err := core.CloseIssue(token, owner, repo, issueNumber, opts)
	if err != nil {
		return fmt.Errorf("failed to close issue: %w", err)
	}

	// Output results
	if outputJson {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")

		return enc.Encode(closed)
	}

	_, _ = fmt.Fprintf(os.Stdout, "\n✓ Closed issue #%d: %s\n", closed.Number, closed.Title)
	_, _ = fmt.Fprintf(os.Stdout, "  %s\n", closed.URL)

	return nil
}

// formatAge formats a time as a human-readable age string
func formatAge(t time.Time) string {
	d := time.Since(t)

	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		mins := int(d.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}

		return fmt.Sprintf("%d minutes ago", mins)
	case d < 24*time.Hour:
		hours := int(d.Hours())
		if hours == 1 {
			return "1 hour ago"
		}

		return fmt.Sprintf("%d hours ago", hours)
	case d < 30*24*time.Hour:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}

		return fmt.Sprintf("%d days ago", days)
	case d < 365*24*time.Hour:
		months := int(d.Hours() / 24 / 30)
		if months == 1 {
			return "1 month ago"
		}

		return fmt.Sprintf("%d months ago", months)
	default:
		years := int(d.Hours() / 24 / 365)
		if years == 1 {
			return "1 year ago"
		}

		return fmt.Sprintf("%d years ago", years)
	}
}
