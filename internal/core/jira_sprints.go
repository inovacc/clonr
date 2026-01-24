package core

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	jira "github.com/andygrunwald/go-jira/v2/cloud"
)

// JiraSprint represents a Jira sprint with essential fields
type JiraSprint struct {
	ID            int        `json:"id"`
	Name          string     `json:"name"`
	State         string     `json:"state"` // active, closed, future
	StartDate     *time.Time `json:"start_date,omitempty"`
	EndDate       *time.Time `json:"end_date,omitempty"`
	CompleteDate  *time.Time `json:"complete_date,omitempty"`
	Goal          string     `json:"goal,omitempty"`
	BoardID       int        `json:"board_id"`
}

// JiraSprintsData contains all sprints for a board
type JiraSprintsData struct {
	BoardID    int           `json:"board_id"`
	BoardName  string        `json:"board_name,omitempty"`
	FetchedAt  time.Time     `json:"fetched_at"`
	TotalCount int           `json:"total_count"`
	Sprints    []JiraSprint  `json:"sprints"`
}

// JiraBoard represents a Jira board
type JiraBoard struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"` // scrum, kanban
	ProjectKey string `json:"project_key,omitempty"`
}

// JiraBoardsData contains all boards
type JiraBoardsData struct {
	FetchedAt  time.Time    `json:"fetched_at"`
	TotalCount int          `json:"total_count"`
	Boards     []JiraBoard  `json:"boards"`
}

// ListJiraSprintsOptions configures sprint listing
type ListJiraSprintsOptions struct {
	State  string // active, closed, future (empty = all)
	Logger *slog.Logger
}

// ListJiraSprints fetches sprints for a board
func ListJiraSprints(client *jira.Client, boardID int, opts ListJiraSprintsOptions) (*JiraSprintsData, error) {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	logger.Debug("fetching sprints",
		slog.Int("board_id", boardID),
		slog.String("state", opts.State),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Build options
	sprintOpts := &jira.GetAllSprintsOptions{
		State: opts.State,
	}

	sprints, _, err := client.Board.GetAllSprints(ctx, int64(boardID), sprintOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch sprints for board %d: %w", boardID, err)
	}

	// Convert sprints
	jiraSprints := make([]JiraSprint, 0, len(sprints.Values))
	for _, s := range sprints.Values {
		jiraSprints = append(jiraSprints, convertJiraSprint(s, boardID))
	}

	return &JiraSprintsData{
		BoardID:    boardID,
		FetchedAt:  time.Now(),
		TotalCount: len(jiraSprints),
		Sprints:    jiraSprints,
	}, nil
}

// convertJiraSprint converts a Jira API sprint to our format
func convertJiraSprint(sprint jira.Sprint, boardID int) JiraSprint {
	js := JiraSprint{
		ID:      sprint.ID,
		Name:    sprint.Name,
		State:   sprint.State,
		Goal:    sprint.Goal,
		BoardID: boardID,
	}

	// Parse dates if they exist
	if sprint.StartDate != nil {
		t := time.Time(*sprint.StartDate)
		if !t.IsZero() {
			js.StartDate = &t
		}
	}

	if sprint.EndDate != nil {
		t := time.Time(*sprint.EndDate)
		if !t.IsZero() {
			js.EndDate = &t
		}
	}

	if sprint.CompleteDate != nil {
		t := time.Time(*sprint.CompleteDate)
		if !t.IsZero() {
			js.CompleteDate = &t
		}
	}

	return js
}

// GetCurrentSprintOptions configures getting the current sprint
type GetCurrentSprintOptions struct {
	Logger *slog.Logger
}

// CurrentSprintData contains the current sprint and its issues summary
type CurrentSprintData struct {
	Sprint       JiraSprint `json:"sprint"`
	IssueCount   int        `json:"issue_count"`
	DaysLeft     int        `json:"days_left,omitempty"`
	Progress     float64    `json:"progress"` // Percentage of completed issues
	ByStatus     map[string]int `json:"by_status"`
}

// GetCurrentSprint fetches the active sprint for a board
func GetCurrentSprint(client *jira.Client, boardID int, opts GetCurrentSprintOptions) (*CurrentSprintData, error) {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	logger.Debug("fetching current sprint",
		slog.Int("board_id", boardID),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Get active sprints
	sprintOpts := &jira.GetAllSprintsOptions{
		State: "active",
	}

	sprints, _, err := client.Board.GetAllSprints(ctx, int64(boardID), sprintOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch active sprints: %w", err)
	}

	if len(sprints.Values) == 0 {
		return nil, fmt.Errorf("no active sprint found for board %d", boardID)
	}

	// Use the first active sprint
	activeSprint := sprints.Values[0]
	jiraSprint := convertJiraSprint(activeSprint, boardID)

	// Calculate days left
	var daysLeft int
	if jiraSprint.EndDate != nil {
		daysLeft = int(time.Until(*jiraSprint.EndDate).Hours() / 24)
		if daysLeft < 0 {
			daysLeft = 0
		}
	}

	// Get issues in sprint
	jql := fmt.Sprintf("sprint = %d", activeSprint.ID)
	searchOpts := &jira.SearchOptions{
		MaxResults: 200,
	}

	issues, _, err := client.Issue.Search(ctx, jql, searchOpts)
	if err != nil {
		logger.Warn("failed to fetch sprint issues",
			slog.String("error", err.Error()),
		)
	}

	// Count issues by status
	byStatus := make(map[string]int)
	completedCount := 0
	for _, issue := range issues {
		if issue.Fields.Status != nil {
			statusName := issue.Fields.Status.Name
			byStatus[statusName]++

			// Check if done (common done category)
			statusLower := strings.ToLower(statusName)
			if strings.Contains(statusLower, "done") ||
				strings.Contains(statusLower, "closed") ||
				strings.Contains(statusLower, "resolved") {
				completedCount++
			}
		}
	}

	// Calculate progress
	var progress float64
	if len(issues) > 0 {
		progress = float64(completedCount) / float64(len(issues)) * 100
	}

	return &CurrentSprintData{
		Sprint:     jiraSprint,
		IssueCount: len(issues),
		DaysLeft:   daysLeft,
		Progress:   progress,
		ByStatus:   byStatus,
	}, nil
}

// ListJiraBoardsOptions configures board listing
type ListJiraBoardsOptions struct {
	ProjectKey string // Filter by project
	Type       string // scrum, kanban (empty = all)
	Name       string // Filter by name contains
	Logger     *slog.Logger
}

// ListJiraBoards fetches all boards
func ListJiraBoards(client *jira.Client, opts ListJiraBoardsOptions) (*JiraBoardsData, error) {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	logger.Debug("fetching boards",
		slog.String("project", opts.ProjectKey),
		slog.String("type", opts.Type),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	boardOpts := &jira.BoardListOptions{
		ProjectKeyOrID: opts.ProjectKey,
		BoardType:      opts.Type,
		Name:           opts.Name,
	}

	boards, _, err := client.Board.GetAllBoards(ctx, boardOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch boards: %w", err)
	}

	// Convert boards
	jiraBoards := make([]JiraBoard, 0, len(boards.Values))
	for _, b := range boards.Values {
		jb := JiraBoard{
			ID:         b.ID,
			Name:       b.Name,
			Type:       b.Type,
			ProjectKey: b.Location.ProjectKey,
		}

		jiraBoards = append(jiraBoards, jb)
	}

	return &JiraBoardsData{
		FetchedAt:  time.Now(),
		TotalCount: len(jiraBoards),
		Boards:     jiraBoards,
	}, nil
}

// GetBoardIDForProject finds a board ID for a given project
func GetBoardIDForProject(client *jira.Client, projectKey string, logger *slog.Logger) (int, error) {
	if logger == nil {
		logger = slog.Default()
	}

	boards, err := ListJiraBoards(client, ListJiraBoardsOptions{
		ProjectKey: projectKey,
		Logger:     logger,
	})
	if err != nil {
		return 0, err
	}

	if len(boards.Boards) == 0 {
		return 0, fmt.Errorf("no boards found for project %s", projectKey)
	}

	// Prefer scrum boards over kanban for sprint operations
	for _, b := range boards.Boards {
		if b.Type == "scrum" {
			return b.ID, nil
		}
	}

	// Return first board if no scrum board found
	return boards.Boards[0].ID, nil
}
