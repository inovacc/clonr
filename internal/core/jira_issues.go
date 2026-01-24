package core

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	jira "github.com/andygrunwald/go-jira/v2/cloud"
)

// JiraIssue represents a Jira issue with essential fields
type JiraIssue struct {
	Key         string     `json:"key"`
	Summary     string     `json:"summary"`
	Description string     `json:"description,omitempty"`
	Status      string     `json:"status"`
	Priority    string     `json:"priority,omitempty"`
	Assignee    string     `json:"assignee,omitempty"`
	Reporter    string     `json:"reporter"`
	IssueType   string     `json:"issue_type"`
	Labels      []string   `json:"labels,omitempty"`
	SprintName  string     `json:"sprint,omitempty"`
	StoryPoints *float64   `json:"story_points,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
	URL         string     `json:"url"`
}

// JiraIssuesData contains all issues for a project
type JiraIssuesData struct {
	Project     string       `json:"project"`
	BaseURL     string       `json:"base_url"`
	FetchedAt   time.Time    `json:"fetched_at"`
	TotalCount  int          `json:"total_count"`
	Issues      []JiraIssue  `json:"issues"`
}

// ListJiraIssuesOptions configures the issue listing behavior
type ListJiraIssuesOptions struct {
	Status    []string // Filter by status (e.g., "To Do", "In Progress")
	Assignee  string   // Filter by assignee email or account ID
	Reporter  string   // Filter by reporter
	Labels    []string // Filter by labels
	IssueType []string // Filter by issue type (Bug, Story, Task)
	Sprint    string   // Filter by sprint name or ID
	JQL       string   // Custom JQL query (overrides other filters)
	Sort      string   // Sort field (created, updated, priority)
	Order     string   // Sort order (asc, desc)
	Limit     int      // Max issues to return (default: 50)
	Logger    *slog.Logger
}

// ListJiraIssues fetches issues from a Jira project
func ListJiraIssues(client *jira.Client, projectKey string, opts ListJiraIssuesOptions) (*JiraIssuesData, error) {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	// Build JQL query
	jql := opts.JQL
	if jql == "" {
		jql = buildJQL(projectKey, opts)
	}

	logger.Debug("fetching Jira issues",
		slog.String("project", projectKey),
		slog.String("jql", jql),
	)

	// Set defaults
	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	searchOpts := &jira.SearchOptions{
		MaxResults: limit,
		StartAt:    0,
	}

	issues, resp, err := client.Issue.Search(ctx, jql, searchOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to search issues: %w", err)
	}

	logger.Debug("received issues",
		slog.Int("count", len(issues)),
		slog.Int("total", resp.Total),
	)

	// Convert to our format
	baseURL := client.BaseURL.String()
	jiraIssues := make([]JiraIssue, 0, len(issues))
	for _, issue := range issues {
		jiraIssues = append(jiraIssues, convertJiraIssue(issue, baseURL))
	}

	return &JiraIssuesData{
		Project:    projectKey,
		BaseURL:    baseURL,
		FetchedAt:  time.Now(),
		TotalCount: resp.Total,
		Issues:     jiraIssues,
	}, nil
}

// buildJQL constructs a JQL query from options
func buildJQL(projectKey string, opts ListJiraIssuesOptions) string {
	var conditions []string

	// Project is always required
	conditions = append(conditions, fmt.Sprintf("project = %s", projectKey))

	// Status filter
	if len(opts.Status) > 0 {
		statuses := make([]string, len(opts.Status))
		for i, s := range opts.Status {
			statuses[i] = fmt.Sprintf("\"%s\"", s)
		}
		conditions = append(conditions, fmt.Sprintf("status IN (%s)", strings.Join(statuses, ", ")))
	}

	// Assignee filter
	if opts.Assignee != "" {
		if opts.Assignee == "@me" || opts.Assignee == "currentUser()" {
			conditions = append(conditions, "assignee = currentUser()")
		} else {
			conditions = append(conditions, fmt.Sprintf("assignee = \"%s\"", opts.Assignee))
		}
	}

	// Reporter filter
	if opts.Reporter != "" {
		conditions = append(conditions, fmt.Sprintf("reporter = \"%s\"", opts.Reporter))
	}

	// Labels filter
	if len(opts.Labels) > 0 {
		for _, label := range opts.Labels {
			conditions = append(conditions, fmt.Sprintf("labels = \"%s\"", label))
		}
	}

	// Issue type filter
	if len(opts.IssueType) > 0 {
		types := make([]string, len(opts.IssueType))
		for i, t := range opts.IssueType {
			types[i] = fmt.Sprintf("\"%s\"", t)
		}
		conditions = append(conditions, fmt.Sprintf("issuetype IN (%s)", strings.Join(types, ", ")))
	}

	// Sprint filter
	if opts.Sprint != "" {
		conditions = append(conditions, fmt.Sprintf("sprint = \"%s\"", opts.Sprint))
	}

	jql := strings.Join(conditions, " AND ")

	// Add sorting
	sortField := opts.Sort
	if sortField == "" {
		sortField = "created"
	}

	sortOrder := opts.Order
	if sortOrder == "" {
		sortOrder = "DESC"
	} else {
		sortOrder = strings.ToUpper(sortOrder)
	}

	jql += fmt.Sprintf(" ORDER BY %s %s", sortField, sortOrder)

	return jql
}

// convertJiraIssue converts a Jira API issue to our format
func convertJiraIssue(issue jira.Issue, baseURL string) JiraIssue {
	ji := JiraIssue{
		Key:         issue.Key,
		Summary:     issue.Fields.Summary,
		Description: issue.Fields.Description,
		URL:         fmt.Sprintf("%s/browse/%s", strings.TrimSuffix(baseURL, "/"), issue.Key),
	}

	if issue.Fields.Status != nil {
		ji.Status = issue.Fields.Status.Name
	}

	if issue.Fields.Priority != nil {
		ji.Priority = issue.Fields.Priority.Name
	}

	if issue.Fields.Assignee != nil {
		ji.Assignee = issue.Fields.Assignee.DisplayName
	}

	if issue.Fields.Reporter != nil {
		ji.Reporter = issue.Fields.Reporter.DisplayName
	}

	if issue.Fields.Type.Name != "" {
		ji.IssueType = issue.Fields.Type.Name
	}

	if issue.Fields.Labels != nil {
		ji.Labels = issue.Fields.Labels
	}

	// Convert timestamps (jira.Time wraps time.Time)
	ji.CreatedAt = time.Time(issue.Fields.Created)
	ji.UpdatedAt = time.Time(issue.Fields.Updated)

	if !time.Time(issue.Fields.Resolutiondate).IsZero() {
		t := time.Time(issue.Fields.Resolutiondate)
		ji.ResolvedAt = &t
	}

	return ji
}

// GetJiraIssueOptions configures getting a single issue
type GetJiraIssueOptions struct {
	Logger *slog.Logger
}

// GetJiraIssue fetches a single issue by key
func GetJiraIssue(client *jira.Client, issueKey string, opts GetJiraIssueOptions) (*JiraIssue, error) {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	logger.Debug("fetching Jira issue",
		slog.String("key", issueKey),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	issue, _, err := client.Issue.Get(ctx, issueKey, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get issue %s: %w", issueKey, err)
	}

	ji := convertJiraIssue(*issue, client.BaseURL.String())

	return &ji, nil
}

// CreateJiraIssueOptions configures issue creation
type CreateJiraIssueOptions struct {
	Summary     string
	Description string
	IssueType   string   // Bug, Story, Task, etc.
	Priority    string   // Highest, High, Medium, Low, Lowest
	Assignee    string   // Account ID or email
	Labels      []string
	Sprint      string // Sprint ID
	Logger      *slog.Logger
}

// CreatedJiraIssue represents the result of creating an issue
type CreatedJiraIssue struct {
	Key       string    `json:"key"`
	Summary   string    `json:"summary"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateJiraIssue creates a new issue in the specified project
func CreateJiraIssue(client *jira.Client, projectKey string, opts CreateJiraIssueOptions) (*CreatedJiraIssue, error) {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	if opts.Summary == "" {
		return nil, fmt.Errorf("issue summary is required")
	}

	// Default issue type
	issueType := opts.IssueType
	if issueType == "" {
		issueType = "Task"
	}

	logger.Debug("creating Jira issue",
		slog.String("project", projectKey),
		slog.String("summary", opts.Summary),
		slog.String("type", issueType),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	issueFields := &jira.IssueFields{
		Project: jira.Project{
			Key: projectKey,
		},
		Summary: opts.Summary,
		Type: jira.IssueType{
			Name: issueType,
		},
	}

	if opts.Description != "" {
		issueFields.Description = opts.Description
	}

	if opts.Priority != "" {
		issueFields.Priority = &jira.Priority{Name: opts.Priority}
	}

	if len(opts.Labels) > 0 {
		issueFields.Labels = opts.Labels
	}

	issue := &jira.Issue{
		Fields: issueFields,
	}

	created, _, err := client.Issue.Create(ctx, issue)
	if err != nil {
		return nil, fmt.Errorf("failed to create issue: %w", err)
	}

	return &CreatedJiraIssue{
		Key:       created.Key,
		Summary:   opts.Summary,
		URL:       fmt.Sprintf("%s/browse/%s", strings.TrimSuffix(client.BaseURL.String(), "/"), created.Key),
		CreatedAt: time.Now(),
	}, nil
}

// TransitionJiraIssueOptions configures issue transitions
type TransitionJiraIssueOptions struct {
	Comment string // Optional comment to add with transition
	Logger  *slog.Logger
}

// JiraTransition represents an available transition
type JiraTransition struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	To   string `json:"to"` // Target status name
}

// GetJiraTransitions returns available transitions for an issue
func GetJiraTransitions(client *jira.Client, issueKey string, opts TransitionJiraIssueOptions) ([]JiraTransition, error) {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	logger.Debug("fetching transitions for issue",
		slog.String("key", issueKey),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	transitions, _, err := client.Issue.GetTransitions(ctx, issueKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get transitions for %s: %w", issueKey, err)
	}

	result := make([]JiraTransition, 0, len(transitions))
	for _, t := range transitions {
		result = append(result, JiraTransition{
			ID:   t.ID,
			Name: t.Name,
			To:   t.To.Name,
		})
	}

	return result, nil
}

// TransitionedJiraIssue represents the result of transitioning an issue
type TransitionedJiraIssue struct {
	Key       string `json:"key"`
	FromState string `json:"from_state"`
	ToState   string `json:"to_state"`
	URL       string `json:"url"`
}

// TransitionJiraIssue moves an issue to a new status
func TransitionJiraIssue(client *jira.Client, issueKey, targetStatus string, opts TransitionJiraIssueOptions) (*TransitionedJiraIssue, error) {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	logger.Debug("transitioning issue",
		slog.String("key", issueKey),
		slog.String("target", targetStatus),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get current issue state
	issue, _, err := client.Issue.Get(ctx, issueKey, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get issue %s: %w", issueKey, err)
	}

	fromState := ""
	if issue.Fields.Status != nil {
		fromState = issue.Fields.Status.Name
	}

	// Get available transitions
	transitions, _, err := client.Issue.GetTransitions(ctx, issueKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get transitions: %w", err)
	}

	// Find matching transition
	var transitionID string
	var toState string
	targetLower := strings.ToLower(targetStatus)

	for _, t := range transitions {
		if strings.ToLower(t.Name) == targetLower || strings.ToLower(t.To.Name) == targetLower {
			transitionID = t.ID
			toState = t.To.Name
			break
		}
	}

	if transitionID == "" {
		availableNames := make([]string, 0, len(transitions))
		for _, t := range transitions {
			availableNames = append(availableNames, fmt.Sprintf("%s -> %s", t.Name, t.To.Name))
		}

		return nil, fmt.Errorf("no transition found for status '%s'\n\nAvailable transitions:\n  %s",
			targetStatus, strings.Join(availableNames, "\n  "))
	}

	// Perform transition
	_, err = client.Issue.DoTransition(ctx, issueKey, transitionID)
	if err != nil {
		return nil, fmt.Errorf("failed to transition issue: %w", err)
	}

	// Add comment if provided
	if opts.Comment != "" {
		commentBody := &jira.Comment{
			Body: opts.Comment,
		}

		_, _, err = client.Issue.AddComment(ctx, issueKey, commentBody)
		if err != nil {
			logger.Warn("failed to add comment after transition",
				slog.String("error", err.Error()),
			)
		}
	}

	return &TransitionedJiraIssue{
		Key:       issueKey,
		FromState: fromState,
		ToState:   toState,
		URL:       fmt.Sprintf("%s/browse/%s", strings.TrimSuffix(client.BaseURL.String(), "/"), issueKey),
	}, nil
}
