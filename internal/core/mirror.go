package core

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"math/rand/v2"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/go-github/v67/github"
	"github.com/inovacc/clonr/internal/grpcclient"
	"golang.org/x/oauth2"
)

// RateLimitConfig contains settings for GitHub API rate limiting
type RateLimitConfig struct {
	MaxRetries        int           // Maximum retry attempts (default: 5)
	InitialBackoff    time.Duration // Initial backoff duration (default: 1s)
	MaxBackoff        time.Duration // Maximum backoff duration (default: 2min)
	BackoffMultiplier float64       // Multiplier for exponential backoff (default: 2.0)
}

// DefaultRateLimitConfig returns sensible defaults
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		MaxRetries:        5,
		InitialBackoff:    1 * time.Second,
		MaxBackoff:        2 * time.Minute,
		BackoffMultiplier: 2.0,
	}
}

// MirrorOptions contains configuration for mirror operations
type MirrorOptions struct {
	SkipArchived    bool
	PublicOnly      bool
	Filter          *regexp.Regexp
	Parallel        int
	DirtyStrategy   DirtyRepoStrategy
	RateLimitConfig RateLimitConfig
	NetworkRetries  int // default: 3
	Shallow         bool
	Logger          *slog.Logger
}

// MirrorPlan represents the prepared mirror operation
type MirrorPlan struct {
	OrgName        string
	Repos          []MirrorRepo
	BaseDir        string
	Token          string
	Parallel       int
	SkipArchived   bool
	Filter         *regexp.Regexp
	DirtyStrategy  DirtyRepoStrategy
	NetworkRetries int
	Shallow        bool
	Logger         *slog.Logger
}

// MirrorRepo represents a single repository to mirror
type MirrorRepo struct {
	Name       string
	URL        string
	Path       string
	Action     string // "clone", "update", or "skip"
	Reason     string // reason for skip
	SkipReason SkipReason
	IsArchived bool
	IsFork     bool
	Size       int64
}

// MirrorResult captures the result of mirroring one repo
type MirrorResult struct {
	Repo       MirrorRepo
	Success    bool
	Error      error
	Duration   int64 // duration in milliseconds
	RetryCount int   // number of retries performed
}

// GitHubClientWrapper wraps the GitHub client with rate limit awareness
type GitHubClientWrapper struct {
	client  *github.Client
	rateCfg RateLimitConfig
	logger  *slog.Logger
}

// NewGitHubClientWrapper creates a rate-limit-aware GitHub client
func NewGitHubClientWrapper(token string, cfg RateLimitConfig, logger *slog.Logger) *GitHubClientWrapper {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)

	return &GitHubClientWrapper{
		client:  github.NewClient(tc),
		rateCfg: cfg,
		logger:  logger,
	}
}

// calculateBackoff computes exponential backoff with jitter
func (w *GitHubClientWrapper) calculateBackoff(attempt int) time.Duration {
	backoff := float64(w.rateCfg.InitialBackoff) * math.Pow(w.rateCfg.BackoffMultiplier, float64(attempt))

	// Apply max cap
	if backoff > float64(w.rateCfg.MaxBackoff) {
		backoff = float64(w.rateCfg.MaxBackoff)
	}

	// Add jitter (10%)
	jitter := backoff * 0.1 * (rand.Float64()*2 - 1)
	backoff += jitter

	return time.Duration(backoff)
}

// isTransientError checks if an error is transient and retryable
func isTransientError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())
	transientIndicators := []string{
		"timeout",
		"connection refused",
		"connection reset",
		"temporary failure",
		"network is unreachable",
		"no such host",
		"503",
		"502",
		"504",
	}

	for _, indicator := range transientIndicators {
		if strings.Contains(errStr, indicator) {
			return true
		}
	}

	return false
}

// fetchOrgReposWithRetry fetches all repos from a GitHub org with pagination and rate limit handling
func (w *GitHubClientWrapper) fetchOrgReposWithRetry(ctx context.Context, orgName string) ([]*github.Repository, error) {
	opt := &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	var allRepos []*github.Repository

	for {
		var (
			repos   []*github.Repository
			resp    *github.Response
			lastErr error
		)

		// Retry loop for this page

		for attempt := 0; attempt <= w.rateCfg.MaxRetries; attempt++ {
			var err error

			repos, resp, err = w.client.Repositories.ListByOrg(ctx, orgName, opt)
			if err == nil {
				break
			}

			// Check if rate limited
			var rateLimitErr *github.RateLimitError
			if errors.As(err, &rateLimitErr) {
				resetTime := rateLimitErr.Rate.Reset.Time
				waitDuration := time.Until(resetTime) + time.Second // add 1s buffer

				w.logger.Warn("rate limited by GitHub API",
					slog.Int("attempt", attempt+1),
					slog.Duration("wait_duration", waitDuration),
					slog.Time("reset_at", resetTime),
				)

				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(waitDuration):
					continue
				}
			}

			// Check for abuse rate limit
			var abuseErr *github.AbuseRateLimitError
			if errors.As(err, &abuseErr) {
				retryAfter := abuseErr.GetRetryAfter()
				w.logger.Warn("abuse rate limit hit",
					slog.Int("attempt", attempt+1),
					slog.Duration("retry_after", retryAfter),
				)

				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(retryAfter):
					continue
				}
			}

			// Check for transient errors
			if isTransientError(err) {
				backoff := w.calculateBackoff(attempt)
				w.logger.Warn("transient error, retrying",
					slog.Int("attempt", attempt+1),
					slog.Duration("backoff", backoff),
					slog.String("error", err.Error()),
				)

				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(backoff):
					lastErr = err
					continue
				}
			}

			// Non-retryable error
			return nil, fmt.Errorf("failed to fetch repos: %w", err)
		}

		if repos == nil && lastErr != nil {
			return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
		}

		allRepos = append(allRepos, repos...)

		if resp == nil || resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	return allRepos, nil
}

// fetchUserReposWithRetry fetches all repos from a GitHub user with pagination and rate limit handling
func (w *GitHubClientWrapper) fetchUserReposWithRetry(ctx context.Context, username string) ([]*github.Repository, error) {
	opt := &github.RepositoryListByUserOptions{
		Type:        "owner", // only repos owned by the user, not forks or collaborations
		ListOptions: github.ListOptions{PerPage: 100},
	}

	var allRepos []*github.Repository

	for {
		var (
			repos   []*github.Repository
			resp    *github.Response
			lastErr error
		)

		// Retry loop for this page

		for attempt := 0; attempt <= w.rateCfg.MaxRetries; attempt++ {
			var err error

			repos, resp, err = w.client.Repositories.ListByUser(ctx, username, opt)
			if err == nil {
				break
			}

			// Check if rate limited
			var rateLimitErr *github.RateLimitError
			if errors.As(err, &rateLimitErr) {
				resetTime := rateLimitErr.Rate.Reset.Time
				waitDuration := time.Until(resetTime) + time.Second // add 1s buffer

				w.logger.Warn("rate limited by GitHub API",
					slog.Int("attempt", attempt+1),
					slog.Duration("wait_duration", waitDuration),
					slog.Time("reset_at", resetTime),
				)

				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(waitDuration):
					continue
				}
			}

			// Check for abuse rate limit
			var abuseErr *github.AbuseRateLimitError
			if errors.As(err, &abuseErr) {
				retryAfter := abuseErr.GetRetryAfter()
				w.logger.Warn("abuse rate limit hit",
					slog.Int("attempt", attempt+1),
					slog.Duration("retry_after", retryAfter),
				)

				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(retryAfter):
					continue
				}
			}

			// Check for transient errors
			if isTransientError(err) {
				backoff := w.calculateBackoff(attempt)
				w.logger.Warn("transient error, retrying",
					slog.Int("attempt", attempt+1),
					slog.Duration("backoff", backoff),
					slog.String("error", err.Error()),
				)

				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(backoff):
					lastErr = err
					continue
				}
			}

			// Non-retryable error
			return nil, fmt.Errorf("failed to fetch repos: %w", err)
		}

		if repos == nil && lastErr != nil {
			return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
		}

		allRepos = append(allRepos, repos...)

		if resp == nil || resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	return allRepos, nil
}

// fetchReposWithRetry tries to fetch repos as org first, then falls back to user
func (w *GitHubClientWrapper) fetchReposWithRetry(ctx context.Context, name string) ([]*github.Repository, bool, error) {
	// Try as organization first
	repos, err := w.fetchOrgReposWithRetry(ctx, name)
	if err == nil {
		return repos, false, nil // isUser = false
	}

	// Check if it's a 404 (not found as org), then try as user
	if strings.Contains(err.Error(), "404") {
		w.logger.Info("not found as organization, trying as user",
			slog.String("name", name),
		)

		repos, err = w.fetchUserReposWithRetry(ctx, name)
		if err != nil {
			return nil, false, fmt.Errorf("failed to fetch repos (tried org and user): %w", err)
		}

		return repos, true, nil // isUser = true
	}

	return nil, false, err
}

// PrepareMirror fetches repos from GitHub and determines actions
func PrepareMirror(orgName, token string, opts MirrorOptions) (*MirrorPlan, error) {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	logger.Info("preparing mirror operation",
		slog.String("org", orgName),
		slog.Int("parallel", opts.Parallel),
		slog.String("dirty_strategy", opts.DirtyStrategy.String()),
	)

	// Create rate-limit-aware GitHub client
	rateCfg := opts.RateLimitConfig
	if rateCfg.MaxRetries == 0 {
		rateCfg = DefaultRateLimitConfig()
	}

	clientWrapper := NewGitHubClientWrapper(token, rateCfg, logger)

	// Fetch all repositories (tries org first, then user)
	ctx := context.Background()

	repos, isUser, err := clientWrapper.fetchReposWithRetry(ctx, orgName)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repositories: %w", err)
	}

	entityType := "org"
	if isUser {
		entityType = "user"
	}

	logger.Info("fetched repositories from GitHub",
		slog.String(entityType, orgName),
		slog.Int("count", len(repos)),
	)

	// Apply filters (archived, regex, public-only)
	filteredRepos := applyFilters(repos, opts)

	logger.Info("filtered repositories",
		slog.Int("before", len(repos)),
		slog.Int("after", len(filteredRepos)),
	)

	// Get config to determine base directory
	grpcClient, err := grpcclient.GetClient()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}

	cfg, err := grpcClient.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	baseDir := filepath.Join(cfg.DefaultCloneDir, orgName)

	// For each repo, determine action (clone/update/skip)
	mirrorRepos := make([]MirrorRepo, len(filteredRepos))
	for i, repo := range filteredRepos {
		path := filepath.Join(baseDir, repo.GetName())
		action, reason, skipReason := determineAction(repo, path, logger)

		mirrorRepos[i] = MirrorRepo{
			Name:       repo.GetName(),
			URL:        repo.GetCloneURL(),
			Path:       path,
			Action:     action,
			Reason:     reason,
			SkipReason: skipReason,
			IsArchived: repo.GetArchived(),
			IsFork:     repo.GetFork(),
			Size:       int64(repo.GetSize()),
		}
	}

	networkRetries := opts.NetworkRetries
	if networkRetries == 0 {
		networkRetries = 3
	}

	return &MirrorPlan{
		OrgName:        orgName,
		Repos:          mirrorRepos,
		BaseDir:        baseDir,
		Token:          token,
		Parallel:       opts.Parallel,
		SkipArchived:   opts.SkipArchived,
		Filter:         opts.Filter,
		DirtyStrategy:  opts.DirtyStrategy,
		NetworkRetries: networkRetries,
		Shallow:        opts.Shallow,
		Logger:         logger,
	}, nil
}

// determineAction decides whether to clone, update, or skip
func determineAction(repo *github.Repository, path string, logger *slog.Logger) (action, reason string, skipReason SkipReason) {
	// Check if directory exists on disk
	_, err := os.Stat(path)
	existsOnDisk := !os.IsNotExist(err)

	if existsOnDisk {
		// Verify it's a valid git repo
		if !isGitRepo(path) {
			return "skip", "path exists but is not a git repository", SkipReasonNotGitRepo
		}

		// Check for URL collision
		existingURL, err := getRepoRemoteURL(path)
		if err != nil {
			logger.Warn("could not determine remote URL",
				slog.String("path", path),
				slog.String("error", err.Error()),
			)

			return "skip", "could not verify remote URL", SkipReasonPathCollision
		}

		if !urlsMatch(existingURL, repo.GetCloneURL()) {
			logger.Warn("path collision detected",
				slog.String("path", path),
				slog.String("expected", repo.GetCloneURL()),
				slog.String("actual", existingURL),
			)

			return "skip", fmt.Sprintf("path contains different repo: %s", existingURL), SkipReasonPathCollision
		}

		return "update", "", SkipReasonNone
	}

	return "clone", "", SkipReasonNone
}

// getRepoRemoteURL gets the origin remote URL from a git repository
func getRepoRemoteURL(path string) (string, error) {
	cmd := exec.Command("git", "-C", path, "remote", "get-url", "origin")

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git remote get-url failed: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// urlsMatch compares two git URLs accounting for different formats
func urlsMatch(url1, url2 string) bool {
	normalize := func(u string) string {
		u = strings.TrimSuffix(u, ".git")
		u = strings.Replace(u, "git@github.com:", "https://github.com/", 1)

		return strings.ToLower(u)
	}

	return normalize(url1) == normalize(url2)
}

// isGitRepo checks if a directory contains a valid git repository
func isGitRepo(path string) bool {
	gitDir := filepath.Join(path, ".git")
	info, err := os.Stat(gitDir)

	return err == nil && info.IsDir()
}

// isRepoDirty checks if repo has uncommitted changes
func isRepoDirty(path string) bool {
	cmd := exec.Command("git", "-C", path, "status", "--porcelain")

	output, err := cmd.Output()
	if err != nil {
		return true // assume dirty on error
	}

	return len(output) > 0
}

// stashChanges stashes uncommitted changes in a repository
func stashChanges(path string) error {
	cmd := exec.Command("git", "-C", path, "stash", "push", "-m", "clonr-mirror-autostash")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git stash failed: %v - %s", err, string(output))
	}

	return nil
}

// unstashChanges pops stashed changes in a repository
func unstashChanges(path string, logger *slog.Logger) {
	cmd := exec.Command("git", "-C", path, "stash", "pop")

	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error("failed to unstash changes",
			slog.String("path", path),
			slog.String("error", err.Error()),
			slog.String("output", string(output)),
		)
	}
}

// resetRepo resets a repository to clean state
func resetRepo(path string) error {
	// Reset staged and unstaged changes
	cmd := exec.Command("git", "-C", path, "reset", "--hard", "HEAD")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git reset failed: %v - %s", err, string(output))
	}

	// Clean untracked files
	cmd = exec.Command("git", "-C", path, "clean", "-fd")

	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clean failed: %v - %s", err, string(output))
	}

	return nil
}

// MirrorCloneRepo clones a single repository for mirroring
func MirrorCloneRepo(repoURL, path string, shallow bool) error {
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	args := []string{"clone"}
	if shallow {
		args = append(args, "--depth", "1")
	}

	args = append(args, repoURL, path)

	cmd := exec.Command("git", args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone failed: %v - %s", err, string(output))
	}

	return nil
}

// MirrorUpdateRepo pulls latest changes for mirroring with dirty repo strategy support
func MirrorUpdateRepo(repoURL, path string, strategy DirtyRepoStrategy, logger *slog.Logger) error {
	// Check for uncommitted changes
	if isRepoDirty(path) {
		switch strategy {
		case DirtyStrategySkip:
			logger.Warn("skipping dirty repository",
				slog.String("path", path),
				slog.String("url", repoURL),
			)

			return &DirtyRepoError{Path: path}

		case DirtyStrategyStash:
			logger.Info("stashing changes before update",
				slog.String("path", path),
			)

			if err := stashChanges(path); err != nil {
				return fmt.Errorf("failed to stash changes: %w", err)
			}
			defer unstashChanges(path, logger)

		case DirtyStrategyReset:
			logger.Warn("resetting repository to clean state",
				slog.String("path", path),
			)

			if err := resetRepo(path); err != nil {
				return fmt.Errorf("failed to reset repository: %w", err)
			}
		}
	}

	cmd := exec.Command("git", "-C", path, "pull", "--ff-only")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git pull failed: %v - %s", err, string(output))
	}

	// Update timestamp in database
	client, err := grpcclient.GetClient()
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	if err := client.UpdateRepoTimestamp(repoURL); err != nil {
		return fmt.Errorf("failed to update timestamp: %w", err)
	}

	return nil
}

// SaveMirroredRepo saves the repo to database
func SaveMirroredRepo(repoURL, path string) error {
	client, err := grpcclient.GetClient()
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	u, err := url.Parse(repoURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Try to insert the repo
	err = client.InsertRepoIfNotExists(u, path)
	if err != nil {
		// If it already exists, that's okay - just update the timestamp
		if strings.Contains(err.Error(), "already exists") {
			if err := client.UpdateRepoTimestamp(repoURL); err != nil {
				return fmt.Errorf("failed to update timestamp: %w", err)
			}

			return nil
		}

		return fmt.Errorf("failed to save repo: %w", err)
	}

	// Fetch and save GitHub issues (non-blocking, errors are logged but don't fail mirror)
	token := GetGitHubToken()
	if token != "" {
		_ = FetchAndSaveIssues(repoURL, path, FetchIssuesOptions{
			Token: token,
		})
	}

	return nil
}

// applyFilters applies user-specified filters to repo list
func applyFilters(repos []*github.Repository, opts MirrorOptions) []*github.Repository {
	filtered := make([]*github.Repository, 0, len(repos))

	for _, repo := range repos {
		// Skip archived if requested
		if opts.SkipArchived && repo.GetArchived() {
			continue
		}

		// Skip non-public if public-only
		if opts.PublicOnly && repo.GetPrivate() {
			continue
		}

		// Apply regex filter if provided
		if opts.Filter != nil && !opts.Filter.MatchString(repo.GetName()) {
			continue
		}

		filtered = append(filtered, repo)
	}

	return filtered
}

// LogDryRunPlan logs what would be done without executing
func LogDryRunPlan(plan *MirrorPlan, logger *slog.Logger) {
	cloneCount := 0
	updateCount := 0
	skipCount := 0

	for _, repo := range plan.Repos {
		switch repo.Action {
		case "clone":
			cloneCount++
		case "update":
			updateCount++
		case "skip":
			skipCount++
		}
	}

	logger.Info("dry run plan summary",
		slog.String("org", plan.OrgName),
		slog.String("base_dir", plan.BaseDir),
		slog.Int("total_repos", len(plan.Repos)),
		slog.Int("to_clone", cloneCount),
		slog.Int("to_update", updateCount),
		slog.Int("to_skip", skipCount),
	)

	for _, repo := range plan.Repos {
		logger.Info("planned action",
			slog.String("repo", repo.Name),
			slog.String("action", repo.Action),
			slog.String("reason", repo.Reason),
			slog.Bool("archived", repo.IsArchived),
			slog.Bool("fork", repo.IsFork),
		)
	}
}

// LogMirrorSummary logs the final summary after mirroring
func LogMirrorSummary(results []MirrorResult, logger *slog.Logger) {
	cloned := 0
	updated := 0
	skipped := 0
	failed := 0

	for _, result := range results {
		if result.Success {
			switch result.Repo.Action {
			case "clone":
				cloned++
			case "update":
				updated++
			case "skip":
				skipped++
			}
		} else {
			failed++
		}
	}

	logger.Info("mirror operation complete",
		slog.Int("cloned", cloned),
		slog.Int("updated", updated),
		slog.Int("skipped", skipped),
		slog.Int("failed", failed),
	)

	for _, result := range results {
		if !result.Success {
			logger.Error("repository failed",
				slog.String("repo", result.Repo.Name),
				slog.String("action", result.Repo.Action),
				slog.String("error", result.Error.Error()),
				slog.Int("retry_count", result.RetryCount),
			)
		}
	}
}

// PrintDryRunPlan prints what would be done without executing (for TUI display)
func PrintDryRunPlan(plan *MirrorPlan) {
	_, _ = fmt.Fprintf(os.Stdout, "\nDry run: Mirroring organization '%s'\n", plan.OrgName)
	_, _ = fmt.Fprintf(os.Stdout, "Base directory: %s\n", plan.BaseDir)
	_, _ = fmt.Fprintf(os.Stdout, "Total repositories: %d\n\n", len(plan.Repos))

	cloneCount := 0
	updateCount := 0
	skipCount := 0

	for _, repo := range plan.Repos {
		switch repo.Action {
		case "clone":
			cloneCount++
		case "update":
			updateCount++
		case "skip":
			skipCount++
		}
	}

	_, _ = fmt.Fprintf(os.Stdout, "Actions:\n")
	_, _ = fmt.Fprintf(os.Stdout, "  Clone: %d repositories\n", cloneCount)
	_, _ = fmt.Fprintf(os.Stdout, "  Update: %d repositories\n", updateCount)
	_, _ = fmt.Fprintf(os.Stdout, "  Skip: %d repositories\n\n", skipCount)

	if cloneCount > 0 {
		_, _ = fmt.Fprintln(os.Stdout, "Repositories to clone:")

		for _, repo := range plan.Repos {
			if repo.Action == "clone" {
				archivedStr := ""
				if repo.IsArchived {
					archivedStr = " [archived]"
				}

				forkStr := ""
				if repo.IsFork {
					forkStr = " [fork]"
				}

				_, _ = fmt.Fprintf(os.Stdout, "  * %s%s%s\n", repo.Name, archivedStr, forkStr)
			}
		}

		_, _ = fmt.Fprintln(os.Stdout)
	}

	if updateCount > 0 {
		_, _ = fmt.Fprintln(os.Stdout, "Repositories to update:")

		for _, repo := range plan.Repos {
			if repo.Action == "update" {
				_, _ = fmt.Fprintf(os.Stdout, "  * %s\n", repo.Name)
			}
		}

		_, _ = fmt.Fprintln(os.Stdout)
	}

	if skipCount > 0 {
		_, _ = fmt.Fprintln(os.Stdout, "Repositories to skip:")

		for _, repo := range plan.Repos {
			if repo.Action == "skip" {
				_, _ = fmt.Fprintf(os.Stdout, "  * %s - %s\n", repo.Name, repo.Reason)
			}
		}

		_, _ = fmt.Fprintln(os.Stdout)
	}
}

// PrintMirrorSummary prints the final summary after mirroring (for TUI display)
func PrintMirrorSummary(results []MirrorResult) {
	_, _ = fmt.Fprintln(os.Stdout, "\nMirror operation complete!")

	cloned := 0
	updated := 0
	skipped := 0
	failed := 0

	for _, result := range results {
		if result.Success {
			switch result.Repo.Action {
			case "clone":
				cloned++
			case "update":
				updated++
			case "skip":
				skipped++
			}
		} else {
			failed++
		}
	}

	_, _ = fmt.Fprintln(os.Stdout, "Results:")
	_, _ = fmt.Fprintf(os.Stdout, "  Cloned:  %d\n", cloned)
	_, _ = fmt.Fprintf(os.Stdout, "  Updated: %d\n", updated)
	_, _ = fmt.Fprintf(os.Stdout, "  Skipped: %d\n", skipped)
	_, _ = fmt.Fprintf(os.Stdout, "  Failed:  %d\n\n", failed)

	if failed > 0 {
		_, _ = fmt.Fprintln(os.Stdout, "Failed repositories:")

		for _, result := range results {
			if !result.Success {
				errMsg := result.Error.Error()
				// Truncate long error messages
				if len(errMsg) > 100 {
					errMsg = errMsg[:100] + "..."
				}

				_, _ = fmt.Fprintf(os.Stdout, "  * %s: %s\n", result.Repo.Name, errMsg)
			}
		}

		_, _ = fmt.Fprintln(os.Stdout)
	}
}

// ValidateOrgName validates the organization name
func ValidateOrgName(orgName string) error {
	if orgName == "" {
		return fmt.Errorf("organization name cannot be empty")
	}

	// Check for path traversal attempts
	if strings.Contains(orgName, "..") || strings.Contains(orgName, "/") || strings.Contains(orgName, "\\") {
		return fmt.Errorf("invalid organization name: contains illegal characters")
	}

	return nil
}

// IsNetworkError checks if an error is a transient network error
func IsNetworkError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())
	networkIndicators := []string{
		"connection refused",
		"connection reset",
		"timeout",
		"temporary failure",
		"network is unreachable",
		"no such host",
		"i/o timeout",
		"tls handshake timeout",
	}

	for _, indicator := range networkIndicators {
		if strings.Contains(errStr, indicator) {
			return true
		}
	}

	return false
}
