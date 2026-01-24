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

var jiraSprintsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List sprints for a board",
	Long: `List Jira sprints for a board.

You must specify a board ID or a project key (which will find the default board).

Examples:
  clonr pm jira sprints list --board 123            # List sprints for board
  clonr pm jira sprints list --project PROJ         # List sprints for project's board
  clonr pm jira sprints list --board 123 --state active    # Only active sprints
  clonr pm jira sprints list --board 123 --state closed    # Only closed sprints`,
	RunE: runJiraSprintsList,
}

var jiraSprintsCurrentCmd = &cobra.Command{
	Use:   "current",
	Short: "Show the current active sprint",
	Long: `Show details about the current active sprint.

Displays sprint information including:
  - Sprint name, goal, and dates
  - Progress (completed vs total issues)
  - Issues by status
  - Days remaining

Examples:
  clonr pm jira sprints current --board 123
  clonr pm jira sprints current --project PROJ
  clonr pm jira sprints current --board 123 --json`,
	RunE: runJiraSprintsCurrent,
}

var jiraBoardsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List Jira boards",
	Long: `List Jira boards.

Optionally filter by project or board type.

Examples:
  clonr pm jira boards list                          # List all boards
  clonr pm jira boards list --project PROJ           # Filter by project
  clonr pm jira boards list --type scrum             # Filter by type
  clonr pm jira boards list --name "Sprint Board"    # Filter by name`,
	RunE: runJiraBoardsList,
}

func init() {
	jiraSprintsCmd.AddCommand(jiraSprintsListCmd)
	jiraSprintsCmd.AddCommand(jiraSprintsCurrentCmd)
	jiraBoardsCmd.AddCommand(jiraBoardsListCmd)

	// Sprints list flags
	addJiraCommonFlags(jiraSprintsListCmd)
	jiraSprintsListCmd.Flags().Int("board", 0, "Board ID")
	jiraSprintsListCmd.Flags().String("project", "", "Project key (finds default board)")
	jiraSprintsListCmd.Flags().String("state", "", "Filter by state: active, closed, future")

	// Sprints current flags
	addJiraCommonFlags(jiraSprintsCurrentCmd)
	jiraSprintsCurrentCmd.Flags().Int("board", 0, "Board ID")
	jiraSprintsCurrentCmd.Flags().String("project", "", "Project key (finds default board)")

	// Boards list flags
	addJiraCommonFlags(jiraBoardsListCmd)
	jiraBoardsListCmd.Flags().String("project", "", "Filter by project key")
	jiraBoardsListCmd.Flags().String("type", "", "Filter by type: scrum, kanban")
	jiraBoardsListCmd.Flags().String("name", "", "Filter by name (contains)")
}

func runJiraSprintsList(cmd *cobra.Command, args []string) error {
	// Get flags
	tokenFlag, _ := cmd.Flags().GetString("token")
	urlFlag, _ := cmd.Flags().GetString("url")
	emailFlag, _ := cmd.Flags().GetString("email")
	outputJson, _ := cmd.Flags().GetBool("json")
	boardID, _ := cmd.Flags().GetInt("board")
	projectKey, _ := cmd.Flags().GetString("project")
	state, _ := cmd.Flags().GetString("state")

	// Validate input
	if boardID == 0 && projectKey == "" {
		return fmt.Errorf("either --board or --project is required\n\nUsage: clonr pm jira sprints list --board 123\n       clonr pm jira sprints list --project PROJ")
	}

	// Resolve credentials
	creds, err := core.ResolveJiraCredentials(tokenFlag, emailFlag, urlFlag)
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
	client, err := core.CreateJiraClient(creds, core.JiraClientOptions{Logger: logger})
	if err != nil {
		return fmt.Errorf("failed to create Jira client: %w", err)
	}

	// If project key provided, find the board
	if boardID == 0 && projectKey != "" {
		if !outputJson {
			_, _ = fmt.Fprintf(os.Stderr, "Finding board for project %s...\n", projectKey)
		}

		boardID, err = core.GetBoardIDForProject(client, projectKey, logger)
		if err != nil {
			return err
		}
	}

	if !outputJson {
		_, _ = fmt.Fprintf(os.Stderr, "Fetching sprints for board %d...\n", boardID)
	}

	// Fetch sprints
	opts := core.ListJiraSprintsOptions{
		State:  state,
		Logger: logger,
	}

	sprints, err := core.ListJiraSprints(client, boardID, opts)
	if err != nil {
		return fmt.Errorf("failed to fetch sprints: %w", err)
	}

	// Output results
	if outputJson {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")

		return enc.Encode(sprints)
	}

	// Text output
	if len(sprints.Sprints) == 0 {
		_, _ = fmt.Fprintf(os.Stdout, "No sprints found for board %d\n", boardID)
		return nil
	}

	_, _ = fmt.Fprintf(os.Stdout, "\nSprints for board %d (%d total)\n\n", boardID, sprints.TotalCount)

	for _, sprint := range sprints.Sprints {
		stateIcon := getSprintStateIcon(sprint.State)

		_, _ = fmt.Fprintf(os.Stdout, "%s %-12s %s\n", stateIcon, sprint.State, sprint.Name)

		// Show dates
		if sprint.StartDate != nil || sprint.EndDate != nil {
			dateStr := "  "
			if sprint.StartDate != nil {
				dateStr += sprint.StartDate.Format("Jan 2")
			}

			dateStr += " - "
			if sprint.EndDate != nil {
				dateStr += sprint.EndDate.Format("Jan 2, 2006")
			}

			_, _ = fmt.Fprintln(os.Stdout, dateStr)
		}

		if sprint.Goal != "" {
			_, _ = fmt.Fprintf(os.Stdout, "  Goal: %s\n", core.TruncateString(sprint.Goal, 60))
		}

		_, _ = fmt.Fprintln(os.Stdout)
	}

	return nil
}

func runJiraSprintsCurrent(cmd *cobra.Command, args []string) error {
	// Get flags
	tokenFlag, _ := cmd.Flags().GetString("token")
	urlFlag, _ := cmd.Flags().GetString("url")
	emailFlag, _ := cmd.Flags().GetString("email")
	outputJson, _ := cmd.Flags().GetBool("json")
	boardID, _ := cmd.Flags().GetInt("board")
	projectKey, _ := cmd.Flags().GetString("project")

	// Validate input
	if boardID == 0 && projectKey == "" {
		return fmt.Errorf("either --board or --project is required\n\nUsage: clonr pm jira sprints current --board 123")
	}

	// Resolve credentials
	creds, err := core.ResolveJiraCredentials(tokenFlag, emailFlag, urlFlag)
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
	client, err := core.CreateJiraClient(creds, core.JiraClientOptions{Logger: logger})
	if err != nil {
		return fmt.Errorf("failed to create Jira client: %w", err)
	}

	// If project key provided, find the board
	if boardID == 0 && projectKey != "" {
		if !outputJson {
			_, _ = fmt.Fprintf(os.Stderr, "Finding board for project %s...\n", projectKey)
		}

		boardID, err = core.GetBoardIDForProject(client, projectKey, logger)
		if err != nil {
			return err
		}
	}

	if !outputJson {
		_, _ = fmt.Fprintf(os.Stderr, "Fetching current sprint for board %d...\n", boardID)
	}

	// Fetch current sprint
	opts := core.GetCurrentSprintOptions{
		Logger: logger,
	}

	current, err := core.GetCurrentSprint(client, boardID, opts)
	if err != nil {
		return fmt.Errorf("failed to fetch current sprint: %w", err)
	}

	// Output results
	if outputJson {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")

		return enc.Encode(current)
	}

	// Text output
	_, _ = fmt.Fprintf(os.Stdout, "\n🏃 Current Sprint: %s\n", current.Sprint.Name)
	_, _ = fmt.Fprintf(os.Stdout, "   State: %s\n", current.Sprint.State)

	// Show dates and days left
	if current.Sprint.StartDate != nil && current.Sprint.EndDate != nil {
		_, _ = fmt.Fprintf(os.Stdout, "   Dates: %s - %s",
			current.Sprint.StartDate.Format("Jan 2"),
			current.Sprint.EndDate.Format("Jan 2, 2006"))

		if current.DaysLeft > 0 {
			_, _ = fmt.Fprintf(os.Stdout, " (%d days left)", current.DaysLeft)
		} else if current.DaysLeft == 0 {
			_, _ = fmt.Fprint(os.Stdout, " (ends today)")
		}

		_, _ = fmt.Fprintln(os.Stdout)
	}

	if current.Sprint.Goal != "" {
		_, _ = fmt.Fprintf(os.Stdout, "   Goal: %s\n", current.Sprint.Goal)
	}

	// Progress
	_, _ = fmt.Fprintf(os.Stdout, "\n📊 Progress: %.0f%% complete (%d issues)\n", current.Progress, current.IssueCount)

	// Issues by status
	if len(current.ByStatus) > 0 {
		_, _ = fmt.Fprintln(os.Stdout, "\n   Issues by status:")

		for status, count := range current.ByStatus {
			icon := getJiraStatusIcon(status)
			_, _ = fmt.Fprintf(os.Stdout, "   %s %s: %d\n", icon, status, count)
		}
	}

	return nil
}

func runJiraBoardsList(cmd *cobra.Command, args []string) error {
	// Get flags
	tokenFlag, _ := cmd.Flags().GetString("token")
	urlFlag, _ := cmd.Flags().GetString("url")
	emailFlag, _ := cmd.Flags().GetString("email")
	outputJson, _ := cmd.Flags().GetBool("json")
	projectKey, _ := cmd.Flags().GetString("project")
	boardType, _ := cmd.Flags().GetString("type")
	name, _ := cmd.Flags().GetString("name")

	// Resolve credentials
	creds, err := core.ResolveJiraCredentials(tokenFlag, emailFlag, urlFlag)
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
	client, err := core.CreateJiraClient(creds, core.JiraClientOptions{Logger: logger})
	if err != nil {
		return fmt.Errorf("failed to create Jira client: %w", err)
	}

	if !outputJson {
		_, _ = fmt.Fprintf(os.Stderr, "Fetching boards...\n")
	}

	// Fetch boards
	opts := core.ListJiraBoardsOptions{
		ProjectKey: projectKey,
		Type:       boardType,
		Name:       name,
		Logger:     logger,
	}

	boards, err := core.ListJiraBoards(client, opts)
	if err != nil {
		return fmt.Errorf("failed to fetch boards: %w", err)
	}

	// Output results
	if outputJson {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")

		return enc.Encode(boards)
	}

	// Text output
	if len(boards.Boards) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "No boards found")
		return nil
	}

	_, _ = fmt.Fprintf(os.Stdout, "\nJira Boards (%d total)\n\n", boards.TotalCount)

	for _, board := range boards.Boards {
		typeIcon := getBoardTypeIcon(board.Type)

		_, _ = fmt.Fprintf(os.Stdout, "%s %-8d %-10s %s", typeIcon, board.ID, board.Type, board.Name)

		if board.ProjectKey != "" {
			_, _ = fmt.Fprintf(os.Stdout, " [%s]", board.ProjectKey)
		}

		_, _ = fmt.Fprintln(os.Stdout)
	}

	return nil
}

// getSprintStateIcon returns an icon based on sprint state
func getSprintStateIcon(state string) string {
	switch strings.ToLower(state) {
	case "active":
		return "🏃"
	case "closed":
		return "✅"
	case "future":
		return "📅"
	default:
		return "⚫"
	}
}

// getBoardTypeIcon returns an icon based on board type
func getBoardTypeIcon(boardType string) string {
	switch strings.ToLower(boardType) {
	case "scrum":
		return "🏃"
	case "kanban":
		return "📋"
	default:
		return "📊"
	}
}
