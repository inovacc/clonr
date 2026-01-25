package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

const (
	zenHubAPIBaseURL = "https://api.zenhub.com"
)

// ZenHubClient is a client for the ZenHub API
type ZenHubClient struct {
	httpClient *http.Client
	token      string
	baseURL    string
	logger     *slog.Logger
}

// ZenHubClientOptions configures the ZenHub client
type ZenHubClientOptions struct {
	Logger *slog.Logger
}

// CreateZenHubClient creates a new ZenHub API client
func CreateZenHubClient(token string, opts ZenHubClientOptions) (*ZenHubClient, error) {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	if token == "" {
		return nil, fmt.Errorf("API token is required")
	}

	logger.Debug("creating ZenHub client")

	return &ZenHubClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		token:   token,
		baseURL: zenHubAPIBaseURL,
		logger:  logger,
	}, nil
}

// doRequest performs an HTTP request to the ZenHub API
//
//nolint:unparam // method is always GET for now, but kept for future extensibility
func (c *ZenHubClient) doRequest(ctx context.Context, method, path string, result any) error {
	url := c.baseURL + path

	c.logger.Debug("making ZenHub API request",
		slog.String("method", method),
		slog.String("path", path),
	)

	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Authentication-Token", c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// doRequestWithBody performs an HTTP request with a JSON body to the ZenHub API
func (c *ZenHubClient) doRequestWithBody(ctx context.Context, method, path string, body any, result any) error {
	url := c.baseURL + path

	c.logger.Debug("making ZenHub API request with body",
		slog.String("method", method),
		slog.String("path", path),
	)

	var bodyReader io.Reader

	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}

		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Authentication-Token", c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// ZenHubIssuePipeline represents the pipeline info for an issue
type ZenHubIssuePipeline struct {
	Name        string `json:"name"`
	PipelineID  string `json:"pipeline_id,omitempty"`
	WorkspaceID string `json:"workspace_id,omitempty"`
}

// ZenHubIssueEstimate represents the estimate for an issue
type ZenHubIssueEstimate struct {
	Value int `json:"value"`
}

// ZenHubIssue represents an issue with ZenHub-specific data
type ZenHubIssue struct {
	IssueNumber int                   `json:"issue_number"`
	RepoID      int64                 `json:"repo_id"`
	Estimate    *ZenHubIssueEstimate  `json:"estimate,omitempty"`
	Pipeline    *ZenHubIssuePipeline  `json:"pipeline,omitempty"`
	Pipelines   []ZenHubIssuePipeline `json:"pipelines,omitempty"`
	IsEpic      bool                  `json:"is_epic"`
	Position    int                   `json:"position,omitempty"`
}

// ZenHubPipeline represents a pipeline (column) in a ZenHub board
type ZenHubPipeline struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ZenHubWorkspace represents a ZenHub workspace
type ZenHubWorkspace struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// ZenHubEpic represents a ZenHub epic
type ZenHubEpic struct {
	IssueNumber int    `json:"issue_number"`
	RepoID      int64  `json:"repo_id"`
	IssueURL    string `json:"issue_url"`
}

// ZenHubBoard represents a ZenHub board state
type ZenHubBoard struct {
	Pipelines []ZenHubBoardPipeline `json:"pipelines"`
}

// ZenHubBoardPipeline represents a pipeline with issues
type ZenHubBoardPipeline struct {
	ID     string             `json:"id"`
	Name   string             `json:"name"`
	Issues []ZenHubBoardIssue `json:"issues"`
}

// ZenHubBoardIssue represents an issue in a board pipeline
type ZenHubBoardIssue struct {
	IssueNumber int    `json:"issue_number"`
	Estimate    *Value `json:"estimate,omitempty"`
	Position    int    `json:"position"`
	IsEpic      bool   `json:"is_epic"`
}

// Value wraps a numeric value from ZenHub API
type Value struct {
	Value int `json:"value"`
}

// ZenHubSprint represents a ZenHub sprint
type ZenHubSprint struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	StartDate   *time.Time `json:"start_at,omitempty"`
	EndDate     *time.Time `json:"end_at,omitempty"`
	State       string     `json:"state"` // open, closed
	Description string     `json:"description,omitempty"`
}

// GetIssueData returns ZenHub data for a specific issue
func (c *ZenHubClient) GetIssueData(ctx context.Context, repoID int64, issueNumber int) (*ZenHubIssue, error) {
	path := fmt.Sprintf("/p1/repositories/%d/issues/%d", repoID, issueNumber)

	var result ZenHubIssue
	if err := c.doRequest(ctx, "GET", path, &result); err != nil {
		return nil, err
	}

	result.RepoID = repoID
	result.IssueNumber = issueNumber

	return &result, nil
}

// GetBoard returns the board for a repository
func (c *ZenHubClient) GetBoard(ctx context.Context, repoID int64) (*ZenHubBoard, error) {
	path := fmt.Sprintf("/p1/repositories/%d/board", repoID)

	var result ZenHubBoard
	if err := c.doRequest(ctx, "GET", path, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetEpics returns all epics for a repository
func (c *ZenHubClient) GetEpics(ctx context.Context, repoID int64) ([]ZenHubEpic, error) {
	path := fmt.Sprintf("/p1/repositories/%d/epics", repoID)

	var result struct {
		Epics []ZenHubEpic `json:"epic_issues"`
	}
	if err := c.doRequest(ctx, "GET", path, &result); err != nil {
		return nil, err
	}

	return result.Epics, nil
}

// ZenHubWorkspaceFull represents a workspace with full details
type ZenHubWorkspaceFull struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Description  string  `json:"description,omitempty"`
	Repositories []int64 `json:"repositories,omitempty"`
}

// GetWorkspacesForRepo returns all workspaces that include a specific repository
func (c *ZenHubClient) GetWorkspacesForRepo(ctx context.Context, repoID int64) ([]ZenHubWorkspaceFull, error) {
	path := fmt.Sprintf("/p2/repositories/%d/workspaces", repoID)

	var result []ZenHubWorkspaceFull
	if err := c.doRequest(ctx, "GET", path, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// ZenHubEpicIssue represents a child issue in an epic
type ZenHubEpicIssue struct {
	IssueNumber int    `json:"issue_number"`
	RepoID      int64  `json:"repo_id"`
	IssueURL    string `json:"issue_url,omitempty"`
}

// ZenHubEpicDetail represents detailed epic data including children
type ZenHubEpicDetail struct {
	TotalEpicEstimates struct {
		Value int `json:"value"`
	} `json:"total_epic_estimates"`
	Estimate *Value               `json:"estimate,omitempty"`
	Pipeline *ZenHubIssuePipeline `json:"pipeline,omitempty"`
	Issues   []ZenHubEpicIssue    `json:"issues"`
}

// GetEpicData returns detailed epic info including child issues
func (c *ZenHubClient) GetEpicData(ctx context.Context, repoID int64, epicNumber int) (*ZenHubEpicDetail, error) {
	path := fmt.Sprintf("/p1/repositories/%d/epics/%d", repoID, epicNumber)

	var result ZenHubEpicDetail
	if err := c.doRequest(ctx, "GET", path, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// MoveIssueRequest represents a request to move an issue to a pipeline
type MoveIssueRequest struct {
	PipelineID string `json:"pipeline_id"`
	Position   string `json:"position"` // "top", "bottom", or numeric index
}

// MoveIssueToPipeline moves an issue to a different pipeline
func (c *ZenHubClient) MoveIssueToPipeline(ctx context.Context, repoID int64, issueNumber int, pipelineID string, position string) error {
	path := fmt.Sprintf("/p1/repositories/%d/issues/%d/moves", repoID, issueNumber)

	reqBody := MoveIssueRequest{
		PipelineID: pipelineID,
		Position:   position,
	}

	if err := c.doRequestWithBody(ctx, "POST", path, reqBody, nil); err != nil {
		return err
	}

	return nil
}

// SetIssueEstimateRequest represents a request to set an issue estimate
type SetIssueEstimateRequest struct {
	Estimate int `json:"estimate"`
}

// SetIssueEstimate sets the estimate (story points) for an issue
func (c *ZenHubClient) SetIssueEstimate(ctx context.Context, repoID int64, issueNumber int, estimate int) error {
	path := fmt.Sprintf("/p1/repositories/%d/issues/%d/estimate", repoID, issueNumber)

	reqBody := SetIssueEstimateRequest{
		Estimate: estimate,
	}

	if err := c.doRequestWithBody(ctx, "PUT", path, reqBody, nil); err != nil {
		return err
	}

	return nil
}
