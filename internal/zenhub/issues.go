package zenhub

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/go-github/v67/github"
	"github.com/inovacc/clonr/internal/core"
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

	// Use go-GitHub client
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	repository, _, err := client.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return 0, fmt.Errorf("failed to get repository: %w", err)
	}

	return repository.GetID(), nil
}

// EnrichedIssue combines GitHub issue with ZenHub metadata
type EnrichedIssue struct {
	// GitHub fields
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	State     string    `json:"state"`
	Author    string    `json:"author"`
	Assignees []string  `json:"assignees,omitempty"`
	Labels    []string  `json:"labels,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	URL       string    `json:"url"`

	// ZenHub fields
	Pipeline string `json:"pipeline,omitempty"`
	Estimate *int   `json:"estimate,omitempty"`
	IsEpic   bool   `json:"is_epic"`
	EpicRef  *int   `json:"epic,omitempty"` // Parent epic number
}

// EnrichedIssuesData contains enriched issues list
type EnrichedIssuesData struct {
	Repository string          `json:"repository"`
	RepoID     int64           `json:"repo_id"`
	FetchedAt  time.Time       `json:"fetched_at"`
	TotalCount int             `json:"total_count"`
	Issues     []EnrichedIssue `json:"issues"`
}

// EpicWithChildren represents an epic and its child issues
type EpicWithChildren struct {
	// Epic info
	Number   int    `json:"number"`
	Title    string `json:"title"`
	State    string `json:"state"`
	Pipeline string `json:"pipeline,omitempty"`
	Estimate *int   `json:"estimate,omitempty"`
	URL      string `json:"url"`

	// Children
	ChildCount int             `json:"child_count"`
	Children   []EnrichedIssue `json:"children"`

	// Progress
	TotalPoints     int `json:"total_points"`
	CompletedPoints int `json:"completed_points"`
	OpenCount       int `json:"open_count"`
	ClosedCount     int `json:"closed_count"`
}

// MoveIssueResult contains the result of a pipeline move
type MoveIssueResult struct {
	IssueNumber  int    `json:"issue_number"`
	FromPipeline string `json:"from_pipeline"`
	ToPipeline   string `json:"to_pipeline"`
	Position     string `json:"position"`
}

// GetEnrichedIssuesOptions configures enriched issue fetching
type GetEnrichedIssuesOptions struct {
	State       string   // GitHub state filter (open, closed, all)
	Labels      []string // GitHub label filter
	Pipeline    string   // ZenHub pipeline filter
	MinEstimate *int     // Minimum estimate filter
	MaxEstimate *int     // Maximum estimate filter
	EpicNumber  *int     // Filter by parent epic
	Limit       int      // Max issues to return
	Logger      *slog.Logger
}

// GetEnrichedIssues fetches GitHub issues with ZenHub metadata
func GetEnrichedIssues(
	zhClient *ZenHubClient,
	ghToken string,
	owner, repo string,
	repoID int64,
	opts GetEnrichedIssuesOptions,
) (*EnrichedIssuesData, error) {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// 1. Fetch ZenHub board to get pipeline and estimate data for all issues
	logger.Debug("fetching ZenHub board for enrichment",
		slog.Int64("repo_id", repoID),
	)

	board, err := zhClient.GetBoard(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ZenHub board: %w", err)
	}

	// Build issue number -> ZenHub data map
	type zhData struct {
		Pipeline string
		Estimate *int
		IsEpic   bool
	}

	zhMap := make(map[int]zhData)

	for _, pipeline := range board.Pipelines {
		for _, issue := range pipeline.Issues {
			data := zhData{
				Pipeline: pipeline.Name,
				IsEpic:   issue.IsEpic,
			}

			if issue.Estimate != nil {
				est := issue.Estimate.Value
				data.Estimate = &est
			}

			zhMap[issue.IssueNumber] = data
		}
	}

	// 2. Fetch epic children if filtering by epic
	var epicChildren map[int]bool

	if opts.EpicNumber != nil {
		epicDetail, err := zhClient.GetEpicData(ctx, repoID, *opts.EpicNumber)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch epic data: %w", err)
		}

		epicChildren = make(map[int]bool)

		for _, child := range epicDetail.Issues {
			if child.RepoID == repoID {
				epicChildren[child.IssueNumber] = true
			}
		}
	}

	// 3. Fetch GitHub issues
	logger.Debug("fetching GitHub issues",
		slog.String("owner", owner),
		slog.String("repo", repo),
	)

	state := opts.State
	if state == "" {
		state = "open"
	}

	ghOpts := core.ListIssuesOptions{
		State:  state,
		Labels: opts.Labels,
		Limit:  0, // Fetch all, we'll filter
		Logger: logger,
	}

	ghIssues, err := core.ListIssuesFromAPI(ghToken, owner, repo, ghOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch GitHub issues: %w", err)
	}

	// 4. Enrich and filter issues
	enrichedIssues := make([]EnrichedIssue, 0, len(ghIssues.Issues))

	for _, ghIssue := range ghIssues.Issues {
		// Skip PRs (shouldn't be in issues but just in case)
		if ghIssue.IsPR {
			continue
		}

		// Get ZenHub data
		zh, hasZH := zhMap[ghIssue.Number]

		enriched := EnrichedIssue{
			Number:    ghIssue.Number,
			Title:     ghIssue.Title,
			State:     ghIssue.State,
			Author:    ghIssue.Author,
			Assignees: ghIssue.Assignees,
			Labels:    ghIssue.Labels,
			CreatedAt: ghIssue.CreatedAt,
			UpdatedAt: ghIssue.UpdatedAt,
			URL:       ghIssue.URL,
		}

		if hasZH {
			enriched.Pipeline = zh.Pipeline
			enriched.Estimate = zh.Estimate
			enriched.IsEpic = zh.IsEpic
		}

		// Apply filters
		// Pipeline filter
		if opts.Pipeline != "" && enriched.Pipeline != opts.Pipeline {
			continue
		}

		// Estimate filters
		if opts.MinEstimate != nil {
			if enriched.Estimate == nil || *enriched.Estimate < *opts.MinEstimate {
				continue
			}
		}

		if opts.MaxEstimate != nil {
			if enriched.Estimate == nil || *enriched.Estimate > *opts.MaxEstimate {
				continue
			}
		}

		// Epic children filter
		if opts.EpicNumber != nil {
			if !epicChildren[ghIssue.Number] {
				continue
			}
		}

		enrichedIssues = append(enrichedIssues, enriched)

		// Check limit
		if opts.Limit > 0 && len(enrichedIssues) >= opts.Limit {
			break
		}
	}

	return &EnrichedIssuesData{
		Repository: fmt.Sprintf("%s/%s", owner, repo),
		RepoID:     repoID,
		FetchedAt:  time.Now(),
		TotalCount: len(enrichedIssues),
		Issues:     enrichedIssues,
	}, nil
}

// GetEpicWithChildrenOptions configures epic children fetching
type GetEpicWithChildrenOptions struct {
	IncludeClosedChildren bool
	Logger                *slog.Logger
}

// GetEpicWithChildren fetches an epic with all its child issues
func GetEpicWithChildren(
	zhClient *ZenHubClient,
	ghToken string,
	owner, repo string,
	repoID int64,
	epicNumber int,
	opts GetEpicWithChildrenOptions,
) (*EpicWithChildren, error) {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// 1. Fetch epic detail from ZenHub
	logger.Debug("fetching epic detail",
		slog.Int64("repo_id", repoID),
		slog.Int("epic", epicNumber),
	)

	epicDetail, err := zhClient.GetEpicData(ctx, repoID, epicNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch epic data: %w", err)
	}

	// 2. Fetch the epic's own ZenHub data
	epicZH, err := zhClient.GetIssueData(ctx, repoID, epicNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch epic issue data: %w", err)
	}

	// 3. Fetch ZenHub board for pipeline/estimate data of children
	board, err := zhClient.GetBoard(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ZenHub board: %w", err)
	}

	// Build issue -> ZenHub data map
	type zhData struct {
		Pipeline string
		Estimate *int
		IsEpic   bool
	}

	zhMap := make(map[int]zhData)

	for _, pipeline := range board.Pipelines {
		for _, issue := range pipeline.Issues {
			data := zhData{
				Pipeline: pipeline.Name,
				IsEpic:   issue.IsEpic,
			}

			if issue.Estimate != nil {
				est := issue.Estimate.Value
				data.Estimate = &est
			}

			zhMap[issue.IssueNumber] = data
		}
	}

	// 4. Fetch GitHub issue data for epic
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: ghToken})
	tc := oauth2.NewClient(ctx, ts)
	ghClient := github.NewClient(tc)

	ghEpic, _, err := ghClient.Issues.Get(ctx, owner, repo, epicNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch epic from GitHub: %w", err)
	}

	// 5. Collect child issue numbers (same repo only)
	childNumbers := make([]int, 0, len(epicDetail.Issues))
	for _, child := range epicDetail.Issues {
		if child.RepoID == repoID {
			childNumbers = append(childNumbers, child.IssueNumber)
		}
	}

	// 6. Fetch GitHub data for children
	state := "all"
	if !opts.IncludeClosedChildren {
		state = "open"
	}

	ghOpts := core.ListIssuesOptions{
		State:  state,
		Limit:  0,
		Logger: logger,
	}

	ghIssues, err := core.ListIssuesFromAPI(ghToken, owner, repo, ghOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch GitHub issues: %w", err)
	}

	// Build issue number -> GitHub data map
	ghMap := make(map[int]core.Issue)
	for _, issue := range ghIssues.Issues {
		ghMap[issue.Number] = issue
	}

	// 7. Build enriched children list
	children := make([]EnrichedIssue, 0, len(childNumbers))
	totalPoints := 0
	completedPoints := 0
	openCount := 0
	closedCount := 0

	for _, num := range childNumbers {
		ghIssue, hasGH := ghMap[num]
		zh, hasZH := zhMap[num]

		// Skip closed children if not included
		if !opts.IncludeClosedChildren && hasGH && ghIssue.State == "closed" {
			continue
		}

		enriched := EnrichedIssue{
			Number:  num,
			EpicRef: &epicNumber,
		}

		if hasGH {
			enriched.Title = ghIssue.Title
			enriched.State = ghIssue.State
			enriched.Author = ghIssue.Author
			enriched.Assignees = ghIssue.Assignees
			enriched.Labels = ghIssue.Labels
			enriched.CreatedAt = ghIssue.CreatedAt
			enriched.UpdatedAt = ghIssue.UpdatedAt
			enriched.URL = ghIssue.URL

			if ghIssue.State == "open" {
				openCount++
			} else {
				closedCount++
			}
		}

		if hasZH {
			enriched.Pipeline = zh.Pipeline
			enriched.Estimate = zh.Estimate
			enriched.IsEpic = zh.IsEpic

			if zh.Estimate != nil {
				totalPoints += *zh.Estimate
				// Count as completed if closed
				if hasGH && ghIssue.State == "closed" {
					completedPoints += *zh.Estimate
				}
			}
		}

		children = append(children, enriched)
	}

	// Build result
	result := &EpicWithChildren{
		Number:          epicNumber,
		Title:           ghEpic.GetTitle(),
		State:           ghEpic.GetState(),
		URL:             ghEpic.GetHTMLURL(),
		ChildCount:      len(children),
		Children:        children,
		TotalPoints:     totalPoints,
		CompletedPoints: completedPoints,
		OpenCount:       openCount,
		ClosedCount:     closedCount,
	}

	// Add ZenHub data for epic itself
	if epicZH.Pipeline != nil {
		result.Pipeline = epicZH.Pipeline.Name
	}

	if epicZH.Estimate != nil {
		est := epicZH.Estimate.Value
		result.Estimate = &est
	}

	return result, nil
}

// MoveIssueOptions configures issue movement
type MoveIssueOptions struct {
	Position string // "top", "bottom", or numeric position
	Logger   *slog.Logger
}

// MoveIssue moves an issue to a different pipeline
func MoveIssue(
	zhClient *ZenHubClient,
	repoID int64,
	issueNumber int,
	pipelineName string,
	opts MoveIssueOptions,
) (*MoveIssueResult, error) {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1. Get current board to find pipeline IDs and current position
	logger.Debug("fetching board to find pipeline",
		slog.Int64("repo_id", repoID),
		slog.String("target_pipeline", pipelineName),
	)

	board, err := zhClient.GetBoard(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch board: %w", err)
	}

	// Find target pipeline by name
	var (
		targetPipelineID string
		fromPipeline     string
	)

	for _, pipeline := range board.Pipelines {
		// Check if this is the target pipeline
		if pipeline.Name == pipelineName {
			targetPipelineID = pipeline.ID
		}

		// Check if issue is currently in this pipeline
		for _, issue := range pipeline.Issues {
			if issue.IssueNumber == issueNumber {
				fromPipeline = pipeline.Name
			}
		}
	}

	if targetPipelineID == "" {
		return nil, fmt.Errorf("pipeline not found: %s", pipelineName)
	}

	// 2. Determine position
	position := opts.Position
	if position == "" {
		position = "top"
	}

	// 3. Move the issue
	logger.Debug("moving issue",
		slog.Int("issue", issueNumber),
		slog.String("from", fromPipeline),
		slog.String("to", pipelineName),
		slog.String("position", position),
	)

	if err := zhClient.MoveIssueToPipeline(ctx, repoID, issueNumber, targetPipelineID, position); err != nil {
		return nil, fmt.Errorf("failed to move issue: %w", err)
	}

	return &MoveIssueResult{
		IssueNumber:  issueNumber,
		FromPipeline: fromPipeline,
		ToPipeline:   pipelineName,
		Position:     position,
	}, nil
}
