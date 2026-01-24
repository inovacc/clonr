package core

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/go-github/v67/github"
	"golang.org/x/oauth2"
)

// ZenHubIssueData combines GitHub issue data with ZenHub-specific fields
type ZenHubIssueData struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	State     string    `json:"state"`
	Author    string    `json:"author"`
	Assignees []string  `json:"assignees,omitempty"`
	Labels    []string  `json:"labels,omitempty"`
	Pipeline  string    `json:"pipeline"`
	Estimate  *int      `json:"estimate,omitempty"`
	IsEpic    bool      `json:"is_epic"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	URL       string    `json:"url"`
}

// ZenHubBoardData contains the full board state with issues
type ZenHubBoardData struct {
	Repository string               `json:"repository"`
	RepoID     int64                `json:"repo_id"`
	FetchedAt  time.Time            `json:"fetched_at"`
	Pipelines  []ZenHubPipelineData `json:"pipelines"`
}

// ZenHubPipelineData represents a pipeline with its issues
type ZenHubPipelineData struct {
	Name        string            `json:"name"`
	IssueCount  int               `json:"issue_count"`
	TotalPoints int               `json:"total_points"`
	Issues      []ZenHubIssueData `json:"issues,omitempty"`
}

// ZenHubEpicData represents an epic with its details
type ZenHubEpicData struct {
	IssueNumber int    `json:"issue_number"`
	Title       string `json:"title,omitempty"`
	Pipeline    string `json:"pipeline,omitempty"`
	Estimate    *int   `json:"estimate,omitempty"`
	ChildCount  int    `json:"child_count"`
}

// ZenHubEpicsData contains all epics for a repository
type ZenHubEpicsData struct {
	Repository string           `json:"repository"`
	RepoID     int64            `json:"repo_id"`
	FetchedAt  time.Time        `json:"fetched_at"`
	TotalCount int              `json:"total_count"`
	Epics      []ZenHubEpicData `json:"epics"`
}

// GetZenHubBoardOptions configures board fetching
type GetZenHubBoardOptions struct {
	IncludeIssueDetails bool // Whether to fetch full issue details
	Logger              *slog.Logger
}

// GetZenHubBoard fetches the board state for a repository
func GetZenHubBoard(client *ZenHubClient, repoID int64, repoName string, opts GetZenHubBoardOptions) (*ZenHubBoardData, error) {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	logger.Debug("fetching ZenHub board",
		slog.Int64("repo_id", repoID),
		slog.String("repo", repoName),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	board, err := client.GetBoard(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch board: %w", err)
	}

	pipelines := make([]ZenHubPipelineData, 0, len(board.Pipelines))
	for _, p := range board.Pipelines {
		pipelineData := ZenHubPipelineData{
			Name:       p.Name,
			IssueCount: len(p.Issues),
		}

		// Calculate total points
		totalPoints := 0

		for _, issue := range p.Issues {
			if issue.Estimate != nil {
				totalPoints += issue.Estimate.Value
			}
		}

		pipelineData.TotalPoints = totalPoints

		// Optionally include issue details
		if opts.IncludeIssueDetails {
			issues := make([]ZenHubIssueData, 0, len(p.Issues))
			for _, issue := range p.Issues {
				issueData := ZenHubIssueData{
					Number:   issue.IssueNumber,
					Pipeline: p.Name,
					IsEpic:   issue.IsEpic,
				}
				if issue.Estimate != nil {
					est := issue.Estimate.Value
					issueData.Estimate = &est
				}

				issues = append(issues, issueData)
			}

			pipelineData.Issues = issues
		}

		pipelines = append(pipelines, pipelineData)
	}

	return &ZenHubBoardData{
		Repository: repoName,
		RepoID:     repoID,
		FetchedAt:  time.Now(),
		Pipelines:  pipelines,
	}, nil
}

// GetZenHubEpicsOptions configures epics fetching
type GetZenHubEpicsOptions struct {
	Logger *slog.Logger
}

// GetZenHubEpics fetches all epics for a repository
func GetZenHubEpics(client *ZenHubClient, repoID int64, repoName string, opts GetZenHubEpicsOptions) (*ZenHubEpicsData, error) {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	logger.Debug("fetching ZenHub epics",
		slog.Int64("repo_id", repoID),
		slog.String("repo", repoName),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	epics, err := client.GetEpics(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch epics: %w", err)
	}

	epicsData := make([]ZenHubEpicData, 0, len(epics))
	for _, e := range epics {
		epicsData = append(epicsData, ZenHubEpicData{
			IssueNumber: e.IssueNumber,
		})
	}

	return &ZenHubEpicsData{
		Repository: repoName,
		RepoID:     repoID,
		FetchedAt:  time.Now(),
		TotalCount: len(epicsData),
		Epics:      epicsData,
	}, nil
}

// GetZenHubIssueOptions configures issue fetching
type GetZenHubIssueOptions struct {
	Logger *slog.Logger
}

// GetZenHubIssue fetches ZenHub data for a specific issue
func GetZenHubIssue(client *ZenHubClient, repoID int64, issueNumber int, opts GetZenHubIssueOptions) (*ZenHubIssue, error) {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	logger.Debug("fetching ZenHub issue data",
		slog.Int64("repo_id", repoID),
		slog.Int("issue", issueNumber),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	issue, err := client.GetIssueData(ctx, repoID, issueNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch issue: %w", err)
	}

	return issue, nil
}

// ZenHubWorkspacesData contains all workspaces for a repository
type ZenHubWorkspacesData struct {
	Repository string                `json:"repository"`
	RepoID     int64                 `json:"repo_id"`
	FetchedAt  time.Time             `json:"fetched_at"`
	TotalCount int                   `json:"total_count"`
	Workspaces []ZenHubWorkspaceFull `json:"workspaces"`
}

// GetZenHubWorkspacesOptions configures workspace fetching
type GetZenHubWorkspacesOptions struct {
	Logger *slog.Logger
}

// GetZenHubWorkspaces fetches all workspaces for a repository
func GetZenHubWorkspaces(client *ZenHubClient, repoID int64, repoName string, opts GetZenHubWorkspacesOptions) (*ZenHubWorkspacesData, error) {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	logger.Debug("fetching ZenHub workspaces",
		slog.Int64("repo_id", repoID),
		slog.String("repo", repoName),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	workspaces, err := client.GetWorkspacesForRepo(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch workspaces: %w", err)
	}

	return &ZenHubWorkspacesData{
		Repository: repoName,
		RepoID:     repoID,
		FetchedAt:  time.Now(),
		TotalCount: len(workspaces),
		Workspaces: workspaces,
	}, nil
}

// GetGitHubRepoID attempts to get a GitHub repository ID
// This requires a GitHub API call
func GetGitHubRepoID(token, owner, repo string, logger *slog.Logger) (int64, error) {
	if logger == nil {
		logger = slog.Default()
	}

	logger.Debug("fetching GitHub repo ID",
		slog.String("owner", owner),
		slog.String("repo", repo),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Use go-github client
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	repository, _, err := client.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return 0, fmt.Errorf("failed to get repository: %w", err)
	}

	return repository.GetID(), nil
}
