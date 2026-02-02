package core

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/go-github/v67/github"
)

// WorkflowRun represents a GitHub Actions workflow run
type WorkflowRun struct {
	ID           int64      `json:"id"`
	Name         string     `json:"name"`
	WorkflowName string     `json:"workflow_name"`
	Status       string     `json:"status"`     // queued, in_progress, completed, waiting
	Conclusion   string     `json:"conclusion"` // success, failure, neutral, cancelled, skipped, timed_out, action_required, stale
	Branch       string     `json:"branch"`
	Event        string     `json:"event"` // push, pull_request, schedule, workflow_dispatch, etc.
	HeadSHA      string     `json:"head_sha"`
	HeadCommit   string     `json:"head_commit,omitempty"`
	Actor        string     `json:"actor"`
	RunNumber    int        `json:"run_number"`
	RunAttempt   int        `json:"run_attempt"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	URL          string     `json:"url"`
	Duration     string     `json:"duration,omitempty"`
}

// WorkflowJob represents a job within a workflow run
type WorkflowJob struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Status      string     `json:"status"`
	Conclusion  string     `json:"conclusion"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Steps       []JobStep  `json:"steps,omitempty"`
	RunnerName  string     `json:"runner_name,omitempty"`
	URL         string     `json:"url"`
}

// JobStep represents a step within a workflow job
type JobStep struct {
	Name        string     `json:"name"`
	Status      string     `json:"status"`
	Conclusion  string     `json:"conclusion"`
	Number      int64      `json:"number"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// WorkflowRunsData contains workflow runs for a repository
type WorkflowRunsData struct {
	Repository string        `json:"repository"`
	FetchedAt  time.Time     `json:"fetched_at"`
	TotalCount int           `json:"total_count"`
	Runs       []WorkflowRun `json:"workflow_runs"`
}

// WorkflowRunDetail contains detailed information about a specific workflow run
type WorkflowRunDetail struct {
	Run  WorkflowRun   `json:"run"`
	Jobs []WorkflowJob `json:"jobs,omitempty"`
}

// ListWorkflowRunsOptions configures workflow run listing
type ListWorkflowRunsOptions struct {
	Branch     string // Filter by branch
	Event      string // Filter by event type (push, pull_request, etc.)
	Status     string // Filter by status (queued, in_progress, completed)
	Actor      string // Filter by actor (username)
	WorkflowID int64  // Filter by specific workflow ID
	Limit      int    // Max runs to return (0 = unlimited)
	Logger     *slog.Logger
}

// GetWorkflowRunOptions configures getting a specific workflow run
type GetWorkflowRunOptions struct {
	IncludeJobs bool // Include job details
	Logger      *slog.Logger
}

// ListWorkflowRuns lists workflow runs for a repository
func ListWorkflowRuns(token, owner, repo string, opts ListWorkflowRunsOptions) (*WorkflowRunsData, error) {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	client := NewGitHubClient(ctx, token)

	listOpts := &github.ListWorkflowRunsOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	if opts.Branch != "" {
		listOpts.Branch = opts.Branch
	}

	if opts.Event != "" {
		listOpts.Event = opts.Event
	}

	if opts.Status != "" {
		listOpts.Status = opts.Status
	}

	if opts.Actor != "" {
		listOpts.Actor = opts.Actor
	}

	var allRuns []*github.WorkflowRun

	collected := 0

	for {
		var runs *github.WorkflowRuns

		var resp *github.Response

		var err error

		if opts.WorkflowID > 0 {
			runs, resp, err = client.Actions.ListWorkflowRunsByID(ctx, owner, repo, opts.WorkflowID, listOpts)
		} else {
			runs, resp, err = client.Actions.ListRepositoryWorkflowRuns(ctx, owner, repo, listOpts)
		}

		if err != nil {
			var rateLimitErr *github.RateLimitError
			if errors.As(err, &rateLimitErr) {
				resetTime := rateLimitErr.Rate.Reset.Time
				waitDuration := time.Until(resetTime) + time.Second

				logger.Warn("rate limited, waiting",
					slog.Duration("wait", waitDuration),
				)

				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(waitDuration):
					continue
				}
			}

			return nil, fmt.Errorf("failed to list workflow runs: %w", err)
		}

		allRuns = append(allRuns, runs.WorkflowRuns...)
		collected += len(runs.WorkflowRuns)

		if opts.Limit > 0 && collected >= opts.Limit {
			if len(allRuns) > opts.Limit {
				allRuns = allRuns[:opts.Limit]
			}

			break
		}

		if resp.NextPage == 0 {
			break
		}

		listOpts.Page = resp.NextPage
	}

	return convertWorkflowRuns(owner, repo, allRuns), nil
}

// GetWorkflowRunStatus retrieves the status of a specific workflow run
func GetWorkflowRunStatus(token, owner, repo string, runID int64, opts GetWorkflowRunOptions) (*WorkflowRunDetail, error) {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := NewGitHubClient(ctx, token)

	// Get the workflow run
	run, _, err := client.Actions.GetWorkflowRunByID(ctx, owner, repo, runID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow run %d: %w", runID, err)
	}

	detail := &WorkflowRunDetail{
		Run: *convertWorkflowRun(run),
	}

	// Get jobs if requested
	if opts.IncludeJobs {
		jobs, err := getWorkflowJobs(ctx, client, owner, repo, runID, logger)
		if err != nil {
			logger.Warn("failed to get workflow jobs", slog.Int64("run_id", runID), slog.String("error", err.Error()))
		} else {
			detail.Jobs = jobs
		}
	}

	return detail, nil
}

func getWorkflowJobs(ctx context.Context, client *github.Client, owner, repo string, runID int64, _ *slog.Logger) ([]WorkflowJob, error) {
	jobs, _, err := client.Actions.ListWorkflowJobs(ctx, owner, repo, runID, &github.ListWorkflowJobsOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list workflow jobs: %w", err)
	}

	result := make([]WorkflowJob, 0, len(jobs.Jobs))
	for _, job := range jobs.Jobs {
		wj := WorkflowJob{
			ID:         job.GetID(),
			Name:       job.GetName(),
			Status:     job.GetStatus(),
			Conclusion: job.GetConclusion(),
			RunnerName: job.GetRunnerName(),
			URL:        job.GetHTMLURL(),
		}

		if !job.GetStartedAt().IsZero() {
			t := job.GetStartedAt().Time
			wj.StartedAt = &t
		}

		if !job.GetCompletedAt().IsZero() {
			t := job.GetCompletedAt().Time
			wj.CompletedAt = &t
		}

		// Convert steps
		for _, step := range job.Steps {
			js := JobStep{
				Name:       step.GetName(),
				Status:     step.GetStatus(),
				Conclusion: step.GetConclusion(),
				Number:     step.GetNumber(),
			}

			if !step.GetStartedAt().IsZero() {
				t := step.GetStartedAt().Time
				js.StartedAt = &t
			}

			if !step.GetCompletedAt().IsZero() {
				t := step.GetCompletedAt().Time
				js.CompletedAt = &t
			}

			wj.Steps = append(wj.Steps, js)
		}

		result = append(result, wj)
	}

	return result, nil
}

func convertWorkflowRuns(owner, repo string, runs []*github.WorkflowRun) *WorkflowRunsData {
	data := &WorkflowRunsData{
		Repository: fmt.Sprintf("%s/%s", owner, repo),
		FetchedAt:  time.Now(),
		TotalCount: len(runs),
		Runs:       make([]WorkflowRun, 0, len(runs)),
	}

	for _, run := range runs {
		data.Runs = append(data.Runs, *convertWorkflowRun(run))
	}

	return data
}

func convertWorkflowRun(run *github.WorkflowRun) *WorkflowRun {
	// Get short SHA safely
	sha := run.GetHeadSHA()
	if len(sha) > 7 {
		sha = sha[:7]
	}

	wr := &WorkflowRun{
		HeadSHA:    sha,
		ID:         run.GetID(),
		Name:       run.GetName(),
		Status:     run.GetStatus(),
		Conclusion: run.GetConclusion(),
		Branch:     run.GetHeadBranch(),
		Event:      run.GetEvent(),
		Actor:      run.GetActor().GetLogin(),
		RunNumber:  run.GetRunNumber(),
		RunAttempt: run.GetRunAttempt(),
		CreatedAt:  run.GetCreatedAt().Time,
		UpdatedAt:  run.GetUpdatedAt().Time,
		URL:        run.GetHTMLURL(),
	}

	// Get display title if available
	if run.DisplayTitle != nil && *run.DisplayTitle != "" {
		wr.Name = *run.DisplayTitle
	} else if run.Name != nil && *run.Name != "" {
		wr.Name = *run.Name
	}

	// Get head commit message if available
	if run.HeadCommit != nil && run.HeadCommit.Message != nil {
		msg := *run.HeadCommit.Message
		// Truncate to the first line
		if idx := findNewline(msg); idx > 0 {
			msg = msg[:idx]
		}

		if len(msg) > 50 {
			msg = msg[:47] + "..."
		}

		wr.HeadCommit = msg
	}

	if !run.GetRunStartedAt().IsZero() {
		t := run.GetRunStartedAt().Time
		wr.StartedAt = &t
	}

	// Calculate duration if completed
	if run.GetStatus() == "completed" && wr.StartedAt != nil {
		var end time.Time

		if !run.GetUpdatedAt().IsZero() {
			end = run.GetUpdatedAt().Time
		} else {
			end = time.Now()
		}

		duration := end.Sub(*wr.StartedAt)
		wr.Duration = formatDuration(duration)
		wr.CompletedAt = &end
	}

	return wr
}

func findNewline(s string) int {
	for i, c := range s {
		if c == '\n' || c == '\r' {
			return i
		}
	}

	return -1
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}

	if d < time.Hour {
		mins := int(d.Minutes())
		secs := int(d.Seconds()) % 60

		return fmt.Sprintf("%dm %ds", mins, secs)
	}

	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60

	return fmt.Sprintf("%dh %dm", hours, mins)
}
