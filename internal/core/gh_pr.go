package core

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/go-github/v67/github"
	"golang.org/x/oauth2"
)

// PRStatus represents the status of a pull request
type PRStatus struct {
	Number       int        `json:"number"`
	Title        string     `json:"title"`
	State        string     `json:"state"` // open, closed
	Merged       bool       `json:"merged"`
	Draft        bool       `json:"draft"`
	Mergeable    *bool      `json:"mergeable,omitempty"` // nil if unknown
	Author       string     `json:"author"`
	Branch       string     `json:"head_branch"`
	BaseBranch   string     `json:"base_branch"`
	ReviewState  string     `json:"review_state"` // approved, changes_requested, pending, commented
	Checks       []CheckRun `json:"checks,omitempty"`
	ChecksStatus string     `json:"checks_status"` // success, failure, pending, none
	Additions    int        `json:"additions"`
	Deletions    int        `json:"deletions"`
	ChangedFiles int        `json:"changed_files"`
	Comments     int        `json:"comments"`
	ReviewCount  int        `json:"review_count"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	MergedAt     *time.Time `json:"merged_at,omitempty"`
	ClosedAt     *time.Time `json:"closed_at,omitempty"`
	URL          string     `json:"url"`
	Labels       []string   `json:"labels,omitempty"`
	Assignees    []string   `json:"assignees,omitempty"`
	Reviewers    []string   `json:"reviewers,omitempty"`
}

// CheckRun represents a CI check run status
type CheckRun struct {
	Name        string     `json:"name"`
	Status      string     `json:"status"`     // queued, in_progress, completed
	Conclusion  string     `json:"conclusion"` // success, failure, neutral, cancelled, skipped, timed_out, action_required
	URL         string     `json:"url,omitempty"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// PRStatusOptions configures PR status retrieval
type PRStatusOptions struct {
	Logger *slog.Logger
}

// ListPRsOptions configures PR listing
type ListPRsOptions struct {
	State  string // open, closed, all (default: open)
	Sort   string // created, updated, popularity, long-running (default: created)
	Order  string // asc, desc (default: desc)
	Base   string // Filter by base branch
	Head   string // Filter by head branch (user:branch or org:branch)
	Limit  int    // Max PRs to return (0 = unlimited)
	Logger *slog.Logger
}

// PRsData contains all PRs for a repository
type PRsData struct {
	Repository  string     `json:"repository"`
	FetchedAt   time.Time  `json:"fetched_at"`
	TotalCount  int        `json:"total_count"`
	OpenCount   int        `json:"open_count"`
	ClosedCount int        `json:"closed_count"`
	MergedCount int        `json:"merged_count"`
	PRs         []PRStatus `json:"pull_requests"`
}

// GetPRStatus retrieves the status of a specific pull request
func GetPRStatus(token, owner, repo string, prNumber int, opts PRStatusOptions) (*PRStatus, error) {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	// Get PR details
	pr, _, err := client.PullRequests.Get(ctx, owner, repo, prNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get PR #%d: %w", prNumber, err)
	}

	status := convertPRToStatus(pr)

	// Get reviews to determine the review state
	reviews, _, err := client.PullRequests.ListReviews(ctx, owner, repo, prNumber, &github.ListOptions{PerPage: 100})
	if err != nil {
		logger.Warn("failed to get PR reviews",
			slog.Int("pr", prNumber),
			slog.String("error", err.Error()),
		)
	} else {
		status.ReviewState = determineReviewState(reviews)
		status.ReviewCount = len(reviews)

		// Extract reviewers
		seen := make(map[string]bool)

		for _, review := range reviews {
			if review.User != nil && review.User.Login != nil {
				login := *review.User.Login
				if !seen[login] {
					status.Reviewers = append(status.Reviewers, login)
					seen[login] = true
				}
			}
		}
	}

	// Get check runs
	checkRuns, checksStatus, err := getCheckRuns(ctx, client, owner, repo, pr.GetHead().GetSHA())
	if err != nil {
		logger.Warn("failed to get check runs",
			slog.Int("pr", prNumber),
			slog.String("error", err.Error()),
		)

		status.ChecksStatus = "unknown"
	} else {
		status.Checks = checkRuns
		status.ChecksStatus = checksStatus
	}

	return status, nil
}

// ListOpenPRs lists open pull requests for a repository
func ListOpenPRs(token, owner, repo string, opts ListPRsOptions) (*PRsData, error) {
	return listPRs(token, owner, repo, opts)
}

func listPRs(token, owner, repo string, opts ListPRsOptions) (*PRsData, error) {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	// Set defaults
	if opts.State == "" {
		opts.State = "open"
	}

	if opts.Sort == "" {
		opts.Sort = "created"
	}

	if opts.Order == "" {
		opts.Order = "desc"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	listOpts := &github.PullRequestListOptions{
		State:       opts.State,
		Sort:        opts.Sort,
		Direction:   opts.Order,
		ListOptions: github.ListOptions{PerPage: 100},
	}

	if opts.Base != "" {
		listOpts.Base = opts.Base
	}

	if opts.Head != "" {
		listOpts.Head = opts.Head
	}

	var allPRs []*github.PullRequest

	collected := 0

	for {
		prs, resp, err := client.PullRequests.List(ctx, owner, repo, listOpts)
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

			return nil, fmt.Errorf("failed to list pull requests: %w", err)
		}

		allPRs = append(allPRs, prs...)
		collected += len(prs)

		if opts.Limit > 0 && collected >= opts.Limit {
			// Trim to limit
			if len(allPRs) > opts.Limit {
				allPRs = allPRs[:opts.Limit]
			}

			break
		}

		if resp.NextPage == 0 {
			break
		}

		listOpts.Page = resp.NextPage
	}

	// Convert to our format
	return convertPRsData(owner, repo, allPRs), nil
}

func convertPRToStatus(pr *github.PullRequest) *PRStatus {
	status := &PRStatus{
		Number:       pr.GetNumber(),
		Title:        pr.GetTitle(),
		State:        pr.GetState(),
		Merged:       pr.GetMerged(),
		Draft:        pr.GetDraft(),
		Author:       pr.GetUser().GetLogin(),
		Branch:       pr.GetHead().GetRef(),
		BaseBranch:   pr.GetBase().GetRef(),
		Additions:    pr.GetAdditions(),
		Deletions:    pr.GetDeletions(),
		ChangedFiles: pr.GetChangedFiles(),
		Comments:     pr.GetComments() + pr.GetReviewComments(),
		CreatedAt:    pr.GetCreatedAt().Time,
		UpdatedAt:    pr.GetUpdatedAt().Time,
		URL:          pr.GetHTMLURL(),
	}

	// Set mergeable if known
	if pr.Mergeable != nil {
		status.Mergeable = pr.Mergeable
	}

	// Set merged/closed times
	if !pr.GetMergedAt().IsZero() {
		t := pr.GetMergedAt().Time
		status.MergedAt = &t
	}

	if !pr.GetClosedAt().IsZero() {
		t := pr.GetClosedAt().Time
		status.ClosedAt = &t
	}

	// Extract labels
	for _, label := range pr.Labels {
		status.Labels = append(status.Labels, label.GetName())
	}

	// Extract assignees
	for _, assignee := range pr.Assignees {
		status.Assignees = append(status.Assignees, assignee.GetLogin())
	}

	return status
}

func convertPRsData(owner, repo string, prs []*github.PullRequest) *PRsData {
	data := &PRsData{
		Repository: fmt.Sprintf("%s/%s", owner, repo),
		FetchedAt:  time.Now(),
		PRs:        make([]PRStatus, 0, len(prs)),
	}

	for _, pr := range prs {
		status := convertPRToStatus(pr)
		data.PRs = append(data.PRs, *status)

		switch {
		case pr.GetMerged():
			data.MergedCount++
		case pr.GetState() == "closed":
			data.ClosedCount++
		default:
			data.OpenCount++
		}
	}

	data.TotalCount = len(prs)

	return data
}

func determineReviewState(reviews []*github.PullRequestReview) string {
	// Track the latest review state per reviewer
	latestStates := make(map[string]string)

	for _, review := range reviews {
		if review.User == nil || review.User.Login == nil {
			continue
		}

		login := *review.User.Login
		state := review.GetState()

		// Update to the latest state for this reviewer
		latestStates[login] = state
	}

	// Determine overall state
	hasApproved := false
	hasChangesRequested := false
	hasCommented := false

	for _, state := range latestStates {
		switch state {
		case "APPROVED":
			hasApproved = true
		case "CHANGES_REQUESTED":
			hasChangesRequested = true
		case "COMMENTED":
			hasCommented = true
		}
	}

	// Changes requested takes precedence
	if hasChangesRequested {
		return "changes_requested"
	}

	if hasApproved {
		return "approved"
	}

	if hasCommented {
		return "commented"
	}

	return "pending"
}

func getCheckRuns(ctx context.Context, client *github.Client, owner, repo, ref string) ([]CheckRun, string, error) {
	result, _, err := client.Checks.ListCheckRunsForRef(ctx, owner, repo, ref, &github.ListCheckRunsOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	})
	if err != nil {
		return nil, "", fmt.Errorf("failed to list check runs: %w", err)
	}

	checks := make([]CheckRun, 0, result.GetTotal())
	allSuccess := true
	hasFailure := false
	hasPending := false

	for _, run := range result.CheckRuns {
		check := CheckRun{
			Name:       run.GetName(),
			Status:     run.GetStatus(),
			Conclusion: run.GetConclusion(),
			URL:        run.GetHTMLURL(),
		}

		if !run.GetStartedAt().IsZero() {
			t := run.GetStartedAt().Time
			check.StartedAt = &t
		}

		if !run.GetCompletedAt().IsZero() {
			t := run.GetCompletedAt().Time
			check.CompletedAt = &t
		}

		checks = append(checks, check)

		// Track overall status
		if run.GetStatus() != "completed" {
			hasPending = true
			allSuccess = false
		} else {
			conclusion := run.GetConclusion()
			if conclusion != "success" && conclusion != "skipped" && conclusion != "neutral" {
				hasFailure = true
				allSuccess = false
			}
		}
	}

	// Determine overall checks status
	var checksStatus string

	switch {
	case len(checks) == 0:
		checksStatus = "none"
	case hasFailure:
		checksStatus = "failure"
	case hasPending:
		checksStatus = "pending"
	case allSuccess:
		checksStatus = "success"
	default:
		checksStatus = "unknown"
	}

	return checks, checksStatus, nil
}
