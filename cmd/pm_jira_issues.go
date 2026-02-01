package cmd

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/inovacc/clonr/internal/core"
	"github.com/inovacc/clonr/internal/jira"
	"github.com/spf13/cobra"
)

var jiraIssuesListCmd = &cobra.Command{
	Use:   "list [project-key]",
	Short: "List issues for a Jira project",
	Long: `List Jira issues for a project.

By default, lists all issues. Use filters to narrow results.

Examples:
  clonr pm jira issues list PROJ                           # List issues in project
  clonr pm jira issues list PROJ --status "In Progress"    # Filter by status
  clonr pm jira issues list PROJ --assignee @me            # Your issues
  clonr pm jira issues list PROJ --labels bug,critical     # Filter by labels
  clonr pm jira issues list PROJ --type Bug,Story          # Filter by type
  clonr pm jira issues list PROJ --jql "priority = High"   # Custom JQL`,
	RunE: runJiraIssuesList,
}

var jiraIssuesCreateCmd = &cobra.Command{
	Use:   "create [project-key]",
	Short: "Create a new Jira issue",
	Long: `Create a new Jira issue in a project.

The --summary flag is required. Other fields are optional.

Examples:
  clonr pm jira issues create PROJ --summary "Bug report"
  clonr pm jira issues create PROJ --summary "Feature" --type Story
  clonr pm jira issues create PROJ --summary "Task" --labels backend,urgent
  clonr pm jira issues create PROJ --summary "High priority" --priority High`,
	RunE: runJiraIssuesCreate,
}

var jiraIssuesViewCmd = &cobra.Command{
	Use:   "view <issue-key>",
	Short: "View details of a Jira issue",
	Long: `View detailed information about a Jira issue.

The issue key (e.g., PROJ-123) is required.

Examples:
  clonr pm jira issues view PROJ-123
  clonr pm jira issues view PROJ-123 --json`,
	RunE: runJiraIssuesView,
}

var jiraIssuesTransitionCmd = &cobra.Command{
	Use:   "transition <issue-key> [status]",
	Short: "Move an issue to a new status",
	Long: `Transition a Jira issue to a new status.

If status is not provided, lists available transitions.

Examples:
  clonr pm jira issues transition PROJ-123                  # Show available transitions
  clonr pm jira issues transition PROJ-123 "In Progress"    # Move to In Progress
  clonr pm jira issues transition PROJ-123 Done             # Move to Done
  clonr pm jira issues transition PROJ-123 Done --comment "Fixed in v1.0"`,
	RunE: runJiraIssuesTransition,
}

func init() {
	jiraIssuesCmd.AddCommand(jiraIssuesListCmd)
	jiraIssuesCmd.AddCommand(jiraIssuesCreateCmd)
	jiraIssuesCmd.AddCommand(jiraIssuesViewCmd)
	jiraIssuesCmd.AddCommand(jiraIssuesTransitionCmd)

	// List flags
	addJiraCommonFlags(jiraIssuesListCmd)
	jiraIssuesListCmd.Flags().StringSlice("status", nil, "Filter by status (comma-separated)")
	jiraIssuesListCmd.Flags().String("assignee", "", "Filter by assignee (@me for yourself)")
	jiraIssuesListCmd.Flags().String("reporter", "", "Filter by reporter")
	jiraIssuesListCmd.Flags().StringSlice("labels", nil, "Filter by labels (comma-separated)")
	jiraIssuesListCmd.Flags().StringSlice("type", nil, "Filter by issue type (Bug, Story, Task)")
	jiraIssuesListCmd.Flags().String("sprint", "", "Filter by sprint name or ID")
	jiraIssuesListCmd.Flags().String("jql", "", "Custom JQL query (overrides other filters)")
	jiraIssuesListCmd.Flags().String("sort", "created", "Sort by: created, updated, priority")
	jiraIssuesListCmd.Flags().String("order", "desc", "Sort order: asc, desc")
	jiraIssuesListCmd.Flags().Int("limit", 50, "Maximum number of issues to list")
	jiraIssuesListCmd.Flags().String("project", "", "Jira project key (alternative to positional arg)")

	// Create flags
	addJiraCommonFlags(jiraIssuesCreateCmd)
	jiraIssuesCreateCmd.Flags().String("summary", "", "Issue summary (required)")
	jiraIssuesCreateCmd.Flags().String("description", "", "Issue description")
	jiraIssuesCreateCmd.Flags().String("type", "Task", "Issue type (Bug, Story, Task, Epic)")
	jiraIssuesCreateCmd.Flags().String("priority", "", "Priority (Highest, High, Medium, Low, Lowest)")
	jiraIssuesCreateCmd.Flags().String("assignee", "", "Assignee account ID or email")
	jiraIssuesCreateCmd.Flags().StringSlice("labels", nil, "Labels to add (comma-separated)")
	jiraIssuesCreateCmd.Flags().String("project", "", "Jira project key (alternative to positional arg)")

	// View flags
	addJiraCommonFlags(jiraIssuesViewCmd)

	// Transition flags
	addJiraCommonFlags(jiraIssuesTransitionCmd)
	jiraIssuesTransitionCmd.Flags().String("comment", "", "Add a comment with the transition")
}

func runJiraIssuesList(cmd *cobra.Command, args []string) error {
	// Get flags
	tokenFlag, _ := cmd.Flags().GetString("token")
	urlFlag, _ := cmd.Flags().GetString("url")
	emailFlag, _ := cmd.Flags().GetString("email")
	outputJson, _ := cmd.Flags().GetBool("json")
	projectFlag, _ := cmd.Flags().GetString("project")
	status, _ := cmd.Flags().GetStringSlice("status")
	assignee, _ := cmd.Flags().GetString("assignee")
	reporter, _ := cmd.Flags().GetString("reporter")
	labels, _ := cmd.Flags().GetStringSlice("labels")
	issueTypes, _ := cmd.Flags().GetStringSlice("type")
	sprint, _ := cmd.Flags().GetString("sprint")
	jql, _ := cmd.Flags().GetString("jql")
	sortBy, _ := cmd.Flags().GetString("sort")
	order, _ := cmd.Flags().GetString("order")
	limit, _ := cmd.Flags().GetInt("limit")

	// Get project argument
	var projectArg string
	if len(args) > 0 {
		projectArg = args[0]
	}

	// Resolve credentials
	creds, err := jira.ResolveJiraCredentials(tokenFlag, emailFlag, urlFlag)
	if err != nil {
		return err
	}

	// Detect project
	projectKey, err := core.DetectJiraProject(projectArg, projectFlag)
	if err != nil {
		return err
	}

	// Setup logger
	var logger *slog.Logger
	if outputJson {
		logger = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	} else {
		logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	}

	// Create client
	client, err := jira.CreateJiraClient(creds, jira.JiraClientOptions{Logger: logger})
	if err != nil {
		return fmt.Errorf("failed to create Jira client: %w", err)
	}

	if !outputJson {
		_, _ = fmt.Fprintf(os.Stderr, "Fetching issues for project %s...\n", projectKey)
	}

	// Fetch issues
	opts := jira.ListJiraIssuesOptions{
		Status:    status,
		Assignee:  assignee,
		Reporter:  reporter,
		Labels:    labels,
		IssueType: issueTypes,
		Sprint:    sprint,
		JQL:       jql,
		Sort:      sortBy,
		Order:     order,
		Limit:     limit,
		Logger:    logger,
	}

	issues, err := jira.ListJiraIssues(client, projectKey, opts)
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
		_, _ = fmt.Fprintf(os.Stdout, "No issues found in project %s\n", projectKey)
		return nil
	}

	_, _ = fmt.Fprintf(os.Stdout, "\nIssues for %s (%d total)\n\n", projectKey, issues.TotalCount)

	for _, issue := range issues.Issues {
		// Status icon based on common statuses
		statusIcon := getJiraStatusIcon(issue.Status)

		// Format labels
		labelStr := ""
		if len(issue.Labels) > 0 {
			labelStr = fmt.Sprintf(" [%s]", strings.Join(issue.Labels, ", "))
		}

		// Format age
		age := core.FormatAge(issue.CreatedAt)

		_, _ = fmt.Fprintf(os.Stdout, "%s %-12s %s%s\n", statusIcon, issue.Key, issue.Summary, labelStr)
		_, _ = fmt.Fprintf(os.Stdout, "              %s · %s · created %s",
			issue.Status, issue.IssueType, age)

		if issue.Assignee != "" {
			_, _ = fmt.Fprintf(os.Stdout, " · @%s", issue.Assignee)
		}

		_, _ = fmt.Fprintln(os.Stdout)
	}

	return nil
}

func runJiraIssuesCreate(cmd *cobra.Command, args []string) error {
	// Get flags
	tokenFlag, _ := cmd.Flags().GetString("token")
	urlFlag, _ := cmd.Flags().GetString("url")
	emailFlag, _ := cmd.Flags().GetString("email")
	outputJson, _ := cmd.Flags().GetBool("json")
	projectFlag, _ := cmd.Flags().GetString("project")
	summary, _ := cmd.Flags().GetString("summary")
	description, _ := cmd.Flags().GetString("description")
	issueType, _ := cmd.Flags().GetString("type")
	priority, _ := cmd.Flags().GetString("priority")
	assignee, _ := cmd.Flags().GetString("assignee")
	labels, _ := cmd.Flags().GetStringSlice("labels")

	// Get project argument
	var projectArg string
	if len(args) > 0 {
		projectArg = args[0]
	}

	// Validate summary
	if summary == "" {
		return fmt.Errorf("--summary is required\n\nUsage: clonr pm jira issues create PROJ --summary \"Issue title\"")
	}

	// Resolve credentials
	creds, err := jira.ResolveJiraCredentials(tokenFlag, emailFlag, urlFlag)
	if err != nil {
		return err
	}

	// Detect project
	projectKey, err := core.DetectJiraProject(projectArg, projectFlag)
	if err != nil {
		return err
	}

	// Setup logger
	var logger *slog.Logger
	if outputJson {
		logger = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	} else {
		logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	}

	// Create client
	client, err := jira.CreateJiraClient(creds, jira.JiraClientOptions{Logger: logger})
	if err != nil {
		return fmt.Errorf("failed to create Jira client: %w", err)
	}

	if !outputJson {
		_, _ = fmt.Fprintf(os.Stderr, "Creating issue in %s...\n", projectKey)
	}

	// Create issue
	opts := jira.CreateJiraIssueOptions{
		Summary:     summary,
		Description: description,
		IssueType:   issueType,
		Priority:    priority,
		Assignee:    assignee,
		Labels:      labels,
		Logger:      logger,
	}

	created, err := jira.CreateJiraIssue(client, projectKey, opts)
	if err != nil {
		return fmt.Errorf("failed to create issue: %w", err)
	}

	// Output results
	if outputJson {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")

		return enc.Encode(created)
	}

	_, _ = fmt.Fprintf(os.Stdout, "\n✓ Created issue %s: %s\n", created.Key, created.Summary)
	_, _ = fmt.Fprintf(os.Stdout, "  %s\n", created.URL)

	return nil
}

func runJiraIssuesView(cmd *cobra.Command, args []string) error {
	// Get flags
	tokenFlag, _ := cmd.Flags().GetString("token")
	urlFlag, _ := cmd.Flags().GetString("url")
	emailFlag, _ := cmd.Flags().GetString("email")
	outputJson, _ := cmd.Flags().GetBool("json")

	// Validate issue key
	if len(args) == 0 {
		return fmt.Errorf("issue key is required\n\nUsage: clonr pm jira issues view PROJ-123")
	}

	issueKey, err := core.ExtractJiraIssueKey(args[0])
	if err != nil {
		return err
	}

	// Resolve credentials
	creds, err := jira.ResolveJiraCredentials(tokenFlag, emailFlag, urlFlag)
	if err != nil {
		return err
	}

	// Setup logger
	var logger *slog.Logger
	if outputJson {
		logger = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	} else {
		logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	}

	// Create client
	client, err := jira.CreateJiraClient(creds, jira.JiraClientOptions{Logger: logger})
	if err != nil {
		return fmt.Errorf("failed to create Jira client: %w", err)
	}

	// Fetch issue
	issue, err := jira.GetJiraIssue(client, issueKey, jira.GetJiraIssueOptions{Logger: logger})
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
	statusIcon := getJiraStatusIcon(issue.Status)
	_, _ = fmt.Fprintf(os.Stdout, "\n%s %s: %s\n\n", statusIcon, issue.Key, issue.Summary)
	_, _ = fmt.Fprintf(os.Stdout, "Status:     %s\n", issue.Status)
	_, _ = fmt.Fprintf(os.Stdout, "Type:       %s\n", issue.IssueType)

	if issue.Priority != "" {
		_, _ = fmt.Fprintf(os.Stdout, "Priority:   %s\n", issue.Priority)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Reporter:   %s\n", issue.Reporter)

	if issue.Assignee != "" {
		_, _ = fmt.Fprintf(os.Stdout, "Assignee:   %s\n", issue.Assignee)
	}

	if len(issue.Labels) > 0 {
		_, _ = fmt.Fprintf(os.Stdout, "Labels:     %s\n", strings.Join(issue.Labels, ", "))
	}

	_, _ = fmt.Fprintf(os.Stdout, "Created:    %s\n", core.FormatAge(issue.CreatedAt))
	_, _ = fmt.Fprintf(os.Stdout, "Updated:    %s\n", core.FormatAge(issue.UpdatedAt))

	if issue.Description != "" {
		_, _ = fmt.Fprintf(os.Stdout, "\nDescription:\n%s\n", issue.Description)
	}

	_, _ = fmt.Fprintf(os.Stdout, "\nURL: %s\n", issue.URL)

	return nil
}

func runJiraIssuesTransition(cmd *cobra.Command, args []string) error {
	// Get flags
	tokenFlag, _ := cmd.Flags().GetString("token")
	urlFlag, _ := cmd.Flags().GetString("url")
	emailFlag, _ := cmd.Flags().GetString("email")
	outputJson, _ := cmd.Flags().GetBool("json")
	comment, _ := cmd.Flags().GetString("comment")

	// Validate issue key
	if len(args) == 0 {
		return fmt.Errorf("issue key is required\n\nUsage: clonr pm jira issues transition PROJ-123 [status]")
	}

	issueKey, err := core.ExtractJiraIssueKey(args[0])
	if err != nil {
		return err
	}

	// Get target status if provided
	var targetStatus string
	if len(args) > 1 {
		targetStatus = args[1]
	}

	// Resolve credentials
	creds, err := jira.ResolveJiraCredentials(tokenFlag, emailFlag, urlFlag)
	if err != nil {
		return err
	}

	// Setup logger
	var logger *slog.Logger
	if outputJson {
		logger = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	} else {
		logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	}

	// Create client
	client, err := jira.CreateJiraClient(creds, jira.JiraClientOptions{Logger: logger})
	if err != nil {
		return fmt.Errorf("failed to create Jira client: %w", err)
	}

	opts := jira.TransitionJiraIssueOptions{
		Comment: comment,
		Logger:  logger,
	}

	// If no target status, show available transitions
	if targetStatus == "" {
		transitions, err := jira.GetJiraTransitions(client, issueKey, opts)
		if err != nil {
			return fmt.Errorf("failed to get transitions: %w", err)
		}

		if outputJson {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")

			return enc.Encode(transitions)
		}

		_, _ = fmt.Fprintf(os.Stdout, "\nAvailable transitions for %s:\n\n", issueKey)

		for _, t := range transitions {
			_, _ = fmt.Fprintf(os.Stdout, "  %s -> %s\n", t.Name, t.To)
		}

		_, _ = fmt.Fprintf(os.Stdout, "\nUsage: clonr pm jira issues transition %s \"<status>\"\n", issueKey)

		return nil
	}

	// Perform transition
	if !outputJson {
		_, _ = fmt.Fprintf(os.Stderr, "Transitioning %s to %s...\n", issueKey, targetStatus)
	}

	result, err := jira.TransitionJiraIssue(client, issueKey, targetStatus, opts)
	if err != nil {
		return err
	}

	// Output results
	if outputJson {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")

		return enc.Encode(result)
	}

	_, _ = fmt.Fprintf(os.Stdout, "\n✓ Transitioned %s: %s -> %s\n", result.Key, result.FromState, result.ToState)
	_, _ = fmt.Fprintf(os.Stdout, "  %s\n", result.URL)

	return nil
}

// getJiraStatusIcon returns an icon based on the status name
func getJiraStatusIcon(status string) string {
	statusLower := strings.ToLower(status)

	switch {
	case strings.Contains(statusLower, "done") || strings.Contains(statusLower, "closed") || strings.Contains(statusLower, "resolved"):
		return "✅"
	case strings.Contains(statusLower, "progress") || strings.Contains(statusLower, "review"):
		return "🔵"
	case strings.Contains(statusLower, "todo") || strings.Contains(statusLower, "open") || strings.Contains(statusLower, "backlog"):
		return "⚪"
	case strings.Contains(statusLower, "blocked") || strings.Contains(statusLower, "impediment"):
		return "🔴"
	default:
		return "⚫"
	}
}
