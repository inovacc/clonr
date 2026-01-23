package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/go-github/v67/github"
	"golang.org/x/oauth2"
)

// Issue represents a GitHub issue with essential fields
type Issue struct {
	Number    int        `json:"number"`
	Title     string     `json:"title"`
	State     string     `json:"state"`
	Body      string     `json:"body,omitempty"`
	Labels    []string   `json:"labels,omitempty"`
	Assignees []string   `json:"assignees,omitempty"`
	Author    string     `json:"author"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	ClosedAt  *time.Time `json:"closed_at,omitempty"`
	Comments  int        `json:"comments"`
	URL       string     `json:"url"`
	IsPR      bool       `json:"is_pull_request"`
}

// IssuesData contains all issues for a repository
type IssuesData struct {
	Repository  string    `json:"repository"`
	FetchedAt   time.Time `json:"fetched_at"`
	TotalCount  int       `json:"total_count"`
	OpenCount   int       `json:"open_count"`
	ClosedCount int       `json:"closed_count"`
	Issues      []Issue   `json:"issues"`
}

// FetchIssuesOptions configures the issue fetching behavior
type FetchIssuesOptions struct {
	Token      string
	Logger     *slog.Logger
	IncludePRs bool // Whether to include pull requests (default: false)
}

// FetchAndSaveIssues fetches issues from a GitHub repository and saves them
func FetchAndSaveIssues(repoURL, repoPath string, opts FetchIssuesOptions) error {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	// Parse owner and repo from URL
	owner, repo, err := parseGitHubURL(repoURL)
	if err != nil {
		logger.Debug("not a GitHub URL, skipping issues fetch",
			slog.String("url", repoURL),
		)

		return nil // Not a GitHub repo, silently skip
	}

	// If no token provided, try to get from environment
	token := opts.Token
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
		if token == "" {
			token = os.Getenv("GH_TOKEN")
		}
	}

	if token == "" {
		logger.Debug("no GitHub token available, skipping issues fetch",
			slog.String("repo", repo),
		)

		return nil // No token, silently skip
	}

	logger.Info("fetching issues from GitHub",
		slog.String("owner", owner),
		slog.String("repo", repo),
	)

	// Create GitHub client
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	// Fetch all issues (open and closed)
	issues, err := fetchAllIssues(ctx, client, owner, repo, opts.IncludePRs, logger)
	if err != nil {
		logger.Warn("failed to fetch issues",
			slog.String("repo", repo),
			slog.String("error", err.Error()),
		)

		return nil // Don't fail the clone/mirror operation due to issues fetch
	}

	// Convert to our format
	issuesData := convertIssues(fmt.Sprintf("%s/%s", owner, repo), issues)

	// Save to file
	if err := saveIssues(repoPath, issuesData); err != nil {
		logger.Warn("failed to save issues",
			slog.String("repo", repo),
			slog.String("error", err.Error()),
		)

		return nil // Don't fail the clone/mirror operation
	}

	logger.Info("saved issues",
		slog.String("repo", repo),
		slog.Int("total", issuesData.TotalCount),
		slog.Int("open", issuesData.OpenCount),
		slog.Int("closed", issuesData.ClosedCount),
	)

	return nil
}

// parseGitHubURL extracts owner and repo from various GitHub URL formats
func parseGitHubURL(repoURL string) (owner, repo string, err error) {
	// Normalize URL
	url := strings.TrimSuffix(repoURL, ".git")
	url = strings.Replace(url, "git@github.com:", "https://github.com/", 1)

	// Must be a GitHub URL
	if !strings.Contains(url, "github.com") {
		return "", "", fmt.Errorf("not a GitHub URL")
	}

	// Extract path after github.com
	parts := strings.Split(url, "github.com/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid GitHub URL format")
	}

	pathParts := strings.Split(strings.Trim(parts[1], "/"), "/")
	if len(pathParts) < 2 {
		return "", "", fmt.Errorf("invalid GitHub URL: missing owner/repo")
	}

	return pathParts[0], pathParts[1], nil
}

// fetchAllIssues fetches all issues with pagination
func fetchAllIssues(ctx context.Context, client *github.Client, owner, repo string, includePRs bool, logger *slog.Logger) ([]*github.Issue, error) {
	opt := &github.IssueListByRepoOptions{
		State:       "all",
		Sort:        "created",
		Direction:   "desc",
		ListOptions: github.ListOptions{PerPage: 100},
	}

	var allIssues []*github.Issue

	for {
		issues, resp, err := client.Issues.ListByRepo(ctx, owner, repo, opt)
		if err != nil {
			// Handle rate limiting
			var rateLimitErr *github.RateLimitError
			if errors.As(err, &rateLimitErr) {
				resetTime := rateLimitErr.Rate.Reset.Time
				waitDuration := time.Until(resetTime) + time.Second

				logger.Warn("rate limited, waiting",
					slog.Duration("wait", waitDuration),
				)

				select {
				case <-ctx.Done():
					return allIssues, ctx.Err()
				case <-time.After(waitDuration):
					continue
				}
			}

			return nil, fmt.Errorf("failed to list issues: %w", err)
		}

		// Filter out PRs if not included
		for _, issue := range issues {
			if !includePRs && issue.IsPullRequest() {
				continue
			}

			allIssues = append(allIssues, issue)
		}

		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	return allIssues, nil
}

// convertIssues converts GitHub issues to our format
func convertIssues(repoName string, ghIssues []*github.Issue) *IssuesData {
	issues := make([]Issue, 0, len(ghIssues))
	openCount := 0
	closedCount := 0

	for _, gi := range ghIssues {
		issue := Issue{
			Number:    gi.GetNumber(),
			Title:     gi.GetTitle(),
			State:     gi.GetState(),
			Body:      gi.GetBody(),
			Author:    gi.GetUser().GetLogin(),
			CreatedAt: gi.GetCreatedAt().Time,
			UpdatedAt: gi.GetUpdatedAt().Time,
			Comments:  gi.GetComments(),
			URL:       gi.GetHTMLURL(),
			IsPR:      gi.IsPullRequest(),
		}

		if !gi.GetClosedAt().IsZero() {
			t := gi.GetClosedAt().Time
			issue.ClosedAt = &t
		}

		// Extract labels
		for _, label := range gi.Labels {
			issue.Labels = append(issue.Labels, label.GetName())
		}

		// Extract assignees
		for _, assignee := range gi.Assignees {
			issue.Assignees = append(issue.Assignees, assignee.GetLogin())
		}

		issues = append(issues, issue)

		if gi.GetState() == "open" {
			openCount++
		} else {
			closedCount++
		}
	}

	return &IssuesData{
		Repository:  repoName,
		FetchedAt:   time.Now(),
		TotalCount:  len(issues),
		OpenCount:   openCount,
		ClosedCount: closedCount,
		Issues:      issues,
	}
}

// saveIssues saves issues to a JSON file in the repository
func saveIssues(repoPath string, data *IssuesData) error {
	// Create .clonr directory in the repo
	clonrDir := filepath.Join(repoPath, ".clonr")
	if err := os.MkdirAll(clonrDir, 0755); err != nil {
		return fmt.Errorf("failed to create .clonr directory: %w", err)
	}

	// Write issues to JSON file
	issuesPath := filepath.Join(clonrDir, "issues.json")

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal issues: %w", err)
	}

	if err := os.WriteFile(issuesPath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write issues file: %w", err)
	}

	return nil
}

// GetGitHubToken retrieves GitHub token from environment or gh CLI config
func GetGitHubToken() string {
	// Check environment variables first
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		return token
	}

	if token := os.Getenv("GH_TOKEN"); token != "" {
		return token
	}

	// Try to get token from gh CLI config
	return getGHCLIToken()
}

// getGHCLIToken attempts to read token from gh CLI configuration
func getGHCLIToken() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	// gh stores hosts.yml in different locations by OS
	configPaths := []string{
		filepath.Join(homeDir, ".config", "gh", "hosts.yml"),
		filepath.Join(homeDir, "AppData", "Roaming", "GitHub CLI", "hosts.yml"),
	}

	for _, configPath := range configPaths {
		data, err := os.ReadFile(configPath)
		if err != nil {
			continue
		}

		// Simple parsing - look for oauth_token line under github.com
		lines := strings.Split(string(data), "\n")
		inGitHub := false

		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "github.com:") {
				inGitHub = true
				continue
			}

			if inGitHub && strings.HasPrefix(trimmed, "oauth_token:") {
				token := strings.TrimPrefix(trimmed, "oauth_token:")
				token = strings.TrimSpace(token)

				return token
			}

			// Reset if we hit another host
			if inGitHub && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") && strings.Contains(line, ":") {
				inGitHub = false
			}
		}
	}

	return ""
}
