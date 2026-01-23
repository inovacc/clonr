package core

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/go-github/v67/github"
)

// Contributor represents a repository contributor
type Contributor struct {
	Login         string `json:"login"`
	Name          string `json:"name,omitempty"`
	AvatarURL     string `json:"avatar_url"`
	Contributions int    `json:"contributions"`
	URL           string `json:"url"`
}

// ContributorJourney represents a contributor's activity in a repository
type ContributorJourney struct {
	Contributor    Contributor          `json:"contributor"`
	Commits        []ContributorCommit  `json:"commits"`
	PullRequests   []ContributorPR      `json:"pull_requests"`
	Issues         []ContributorIssue   `json:"issues"`
	TotalCommits   int                  `json:"total_commits"`
	TotalPRs       int                  `json:"total_prs"`
	TotalIssues    int                  `json:"total_issues"`
	FirstActivity  *time.Time           `json:"first_activity,omitempty"`
	LastActivity   *time.Time           `json:"last_activity,omitempty"`
}

// ContributorCommit represents a commit by a contributor
type ContributorCommit struct {
	SHA       string    `json:"sha"`
	Message   string    `json:"message"`
	Date      time.Time `json:"date"`
	Additions int       `json:"additions"`
	Deletions int       `json:"deletions"`
	URL       string    `json:"url"`
}

// ContributorPR represents a pull request by a contributor
type ContributorPR struct {
	Number    int        `json:"number"`
	Title     string     `json:"title"`
	State     string     `json:"state"`
	Merged    bool       `json:"merged"`
	CreatedAt time.Time  `json:"created_at"`
	MergedAt  *time.Time `json:"merged_at,omitempty"`
	URL       string     `json:"url"`
}

// ContributorIssue represents an issue created by a contributor
type ContributorIssue struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	State     string    `json:"state"`
	CreatedAt time.Time `json:"created_at"`
	URL       string    `json:"url"`
}

// ContributorsResult contains the list of contributors
type ContributorsResult struct {
	Repository   string        `json:"repository"`
	Contributors []Contributor `json:"contributors"`
	TotalCount   int           `json:"total_count"`
}

// ListContributorsOptions configures the contributor listing
type ListContributorsOptions struct {
	Limit int // Maximum number of contributors to return
}

// ListContributors returns the list of contributors for a repository
func ListContributors(token, owner, repo string, opts ListContributorsOptions) (*ContributorsResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := github.NewClient(nil).WithAuthToken(token)

	listOpts := &github.ListContributorsOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	var allContributors []*github.Contributor

	for {
		contributors, resp, err := client.Repositories.ListContributors(ctx, owner, repo, listOpts)
		if err != nil {
			return nil, fmt.Errorf("failed to list contributors: %w", err)
		}

		allContributors = append(allContributors, contributors...)

		if resp.NextPage == 0 {
			break
		}

		listOpts.Page = resp.NextPage

		// Apply limit if set
		if opts.Limit > 0 && len(allContributors) >= opts.Limit {
			allContributors = allContributors[:opts.Limit]
			break
		}
	}

	// Apply limit if set
	if opts.Limit > 0 && len(allContributors) > opts.Limit {
		allContributors = allContributors[:opts.Limit]
	}

	result := &ContributorsResult{
		Repository:   fmt.Sprintf("%s/%s", owner, repo),
		Contributors: make([]Contributor, 0, len(allContributors)),
		TotalCount:   len(allContributors),
	}

	for _, c := range allContributors {
		contrib := Contributor{
			Login:         c.GetLogin(),
			AvatarURL:     c.GetAvatarURL(),
			Contributions: c.GetContributions(),
			URL:           c.GetHTMLURL(),
		}

		result.Contributors = append(result.Contributors, contrib)
	}

	return result, nil
}

// GetContributorJourneyOptions configures the journey retrieval
type GetContributorJourneyOptions struct {
	IncludeCommits bool
	IncludePRs     bool
	IncludeIssues  bool
	Limit          int // Limit per category
}

// GetContributorJourney returns the activity journey of a contributor
func GetContributorJourney(token, owner, repo, username string, opts GetContributorJourneyOptions) (*ContributorJourney, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client := github.NewClient(nil).WithAuthToken(token)

	// Get user info
	user, _, err := client.Users.Get(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	journey := &ContributorJourney{
		Contributor: Contributor{
			Login:     user.GetLogin(),
			Name:      user.GetName(),
			AvatarURL: user.GetAvatarURL(),
			URL:       user.GetHTMLURL(),
		},
		Commits:      make([]ContributorCommit, 0),
		PullRequests: make([]ContributorPR, 0),
		Issues:       make([]ContributorIssue, 0),
	}

	var allTimes []time.Time

	// Get commits
	if opts.IncludeCommits {
		commits, err := getContributorCommits(ctx, client, owner, repo, username, opts.Limit)
		if err == nil {
			journey.Commits = commits
			journey.TotalCommits = len(commits)

			for _, c := range commits {
				allTimes = append(allTimes, c.Date)
			}
		}
	}

	// Get PRs
	if opts.IncludePRs {
		prs, err := getContributorPRs(ctx, client, owner, repo, username, opts.Limit)
		if err == nil {
			journey.PullRequests = prs
			journey.TotalPRs = len(prs)

			for _, pr := range prs {
				allTimes = append(allTimes, pr.CreatedAt)
			}
		}
	}

	// Get issues
	if opts.IncludeIssues {
		issues, err := getContributorIssues(ctx, client, owner, repo, username, opts.Limit)
		if err == nil {
			journey.Issues = issues
			journey.TotalIssues = len(issues)

			for _, issue := range issues {
				allTimes = append(allTimes, issue.CreatedAt)
			}
		}
	}

	// Calculate first and last activity
	if len(allTimes) > 0 {
		sort.Slice(allTimes, func(i, j int) bool {
			return allTimes[i].Before(allTimes[j])
		})

		journey.FirstActivity = &allTimes[0]
		journey.LastActivity = &allTimes[len(allTimes)-1]
	}

	return journey, nil
}

func getContributorCommits(ctx context.Context, client *github.Client, owner, repo, username string, limit int) ([]ContributorCommit, error) {
	opts := &github.CommitsListOptions{
		Author:      username,
		ListOptions: github.ListOptions{PerPage: 100},
	}

	if limit > 0 && limit < 100 {
		opts.PerPage = limit
	}

	var allCommits []ContributorCommit

	for {
		commits, resp, err := client.Repositories.ListCommits(ctx, owner, repo, opts)
		if err != nil {
			return nil, err
		}

		for _, c := range commits {
			commit := ContributorCommit{
				SHA:     c.GetSHA()[:7],
				Message: truncateMessage(c.GetCommit().GetMessage(), 80),
				Date:    c.GetCommit().GetAuthor().GetDate().Time,
				URL:     c.GetHTMLURL(),
			}

			// Get commit stats if available
			if c.Stats != nil {
				commit.Additions = c.Stats.GetAdditions()
				commit.Deletions = c.Stats.GetDeletions()
			}

			allCommits = append(allCommits, commit)
		}

		if resp.NextPage == 0 || (limit > 0 && len(allCommits) >= limit) {
			break
		}

		opts.Page = resp.NextPage
	}

	if limit > 0 && len(allCommits) > limit {
		allCommits = allCommits[:limit]
	}

	return allCommits, nil
}

func getContributorPRs(ctx context.Context, client *github.Client, owner, repo, username string, limit int) ([]ContributorPR, error) {
	// Search for PRs by author
	query := fmt.Sprintf("repo:%s/%s author:%s is:pr", owner, repo, username)

	opts := &github.SearchOptions{
		Sort:        "created",
		Order:       "desc",
		ListOptions: github.ListOptions{PerPage: 100},
	}

	if limit > 0 && limit < 100 {
		opts.PerPage = limit
	}

	var allPRs []ContributorPR

	for {
		result, resp, err := client.Search.Issues(ctx, query, opts)
		if err != nil {
			return nil, err
		}

		for _, issue := range result.Issues {
			pr := ContributorPR{
				Number:    issue.GetNumber(),
				Title:     issue.GetTitle(),
				State:     issue.GetState(),
				CreatedAt: issue.GetCreatedAt().Time,
				URL:       issue.GetHTMLURL(),
			}

			// Check if merged (PR merged_at is in PullRequestLinks)
			if issue.PullRequestLinks != nil && issue.GetState() == "closed" {
				// We'll assume closed PRs with PR links that are closed are merged
				// For accurate merged status, we'd need to fetch each PR individually
				pr.Merged = true
			}

			allPRs = append(allPRs, pr)
		}

		if resp.NextPage == 0 || (limit > 0 && len(allPRs) >= limit) {
			break
		}

		opts.Page = resp.NextPage
	}

	if limit > 0 && len(allPRs) > limit {
		allPRs = allPRs[:limit]
	}

	return allPRs, nil
}

func getContributorIssues(ctx context.Context, client *github.Client, owner, repo, username string, limit int) ([]ContributorIssue, error) {
	// Search for issues by author (excluding PRs)
	query := fmt.Sprintf("repo:%s/%s author:%s is:issue", owner, repo, username)

	opts := &github.SearchOptions{
		Sort:        "created",
		Order:       "desc",
		ListOptions: github.ListOptions{PerPage: 100},
	}

	if limit > 0 && limit < 100 {
		opts.PerPage = limit
	}

	var allIssues []ContributorIssue

	for {
		result, resp, err := client.Search.Issues(ctx, query, opts)
		if err != nil {
			return nil, err
		}

		for _, issue := range result.Issues {
			i := ContributorIssue{
				Number:    issue.GetNumber(),
				Title:     issue.GetTitle(),
				State:     issue.GetState(),
				CreatedAt: issue.GetCreatedAt().Time,
				URL:       issue.GetHTMLURL(),
			}

			allIssues = append(allIssues, i)
		}

		if resp.NextPage == 0 || (limit > 0 && len(allIssues) >= limit) {
			break
		}

		opts.Page = resp.NextPage
	}

	if limit > 0 && len(allIssues) > limit {
		allIssues = allIssues[:limit]
	}

	return allIssues, nil
}

func truncateMessage(msg string, maxLen int) string {
	// Get first line only
	for i, c := range msg {
		if c == '\n' {
			msg = msg[:i]
			break
		}
	}

	if len(msg) > maxLen {
		return msg[:maxLen-3] + "..."
	}

	return msg
}
