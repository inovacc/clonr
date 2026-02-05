package actionsdb

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/go-github/v82/github"
	"golang.org/x/oauth2"
)

// WorkerConfig contains configuration for the actions monitoring worker
type WorkerConfig struct {
	PollInterval     time.Duration // How often to check the queue
	CheckInterval    time.Duration // How often to check GitHub for updates
	MaxRetries       int           // Maximum retries for failed checks
	RetryBackoff     time.Duration // Initial backoff for retries
	BatchSize        int           // Number of items to process at once
	CompletionWindow time.Duration // Wait this long after workflow starts before first check
}

// DefaultWorkerConfig returns sensible defaults for the worker
func DefaultWorkerConfig() WorkerConfig {
	return WorkerConfig{
		PollInterval:     30 * time.Second,
		CheckInterval:    60 * time.Second,
		MaxRetries:       10,
		RetryBackoff:     30 * time.Second,
		BatchSize:        10,
		CompletionWindow: 5 * time.Second,
	}
}

// Worker monitors GitHub Actions for pushed commits
type Worker struct {
	db         *DB
	config     WorkerConfig
	tokenFunc  func() string // Function to get current GitHub token
	logger     *slog.Logger
	mu         sync.RWMutex
	running    bool
	stopCh     chan struct{}
	stoppedCh  chan struct{}
	onComplete func(*WorkflowRun) // Callback when a workflow completes
}

// NewWorker creates a new actions monitoring worker
func NewWorker(db *DB, tokenFunc func() string, config WorkerConfig) *Worker {
	return &Worker{
		db:        db,
		config:    config,
		tokenFunc: tokenFunc,
		logger:    slog.Default(),
		stopCh:    make(chan struct{}),
		stoppedCh: make(chan struct{}),
	}
}

// WithLogger sets the logger for the worker
func (w *Worker) WithLogger(logger *slog.Logger) *Worker {
	w.logger = logger
	return w
}

// OnComplete sets a callback for when a workflow run completes
func (w *Worker) OnComplete(fn func(*WorkflowRun)) *Worker {
	w.onComplete = fn
	return w
}

// Start begins the monitoring worker
func (w *Worker) Start(ctx context.Context) error {
	w.mu.Lock()

	if w.running {
		w.mu.Unlock()
		return fmt.Errorf("worker already running")
	}

	w.running = true
	w.stopCh = make(chan struct{})
	w.stoppedCh = make(chan struct{})
	w.mu.Unlock()

	w.logger.Info("starting actions monitoring worker",
		"poll_interval", w.config.PollInterval,
		"check_interval", w.config.CheckInterval)

	go w.run(ctx)

	return nil
}

// Stop stops the monitoring worker
func (w *Worker) Stop() {
	w.mu.Lock()

	if !w.running {
		w.mu.Unlock()
		return
	}

	w.running = false
	close(w.stopCh)
	w.mu.Unlock()

	// Wait for worker to stop
	<-w.stoppedCh
	w.logger.Info("actions monitoring worker stopped")
}

// IsRunning returns whether the worker is currently running
func (w *Worker) IsRunning() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return w.running
}

// run is the main worker loop
func (w *Worker) run(ctx context.Context) {
	defer close(w.stoppedCh)

	ticker := time.NewTicker(w.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		case <-ticker.C:
			if err := w.processQueue(ctx); err != nil {
				w.logger.Error("error processing queue", "error", err)
			}
		}
	}
}

// processQueue processes pending items in the queue
func (w *Worker) processQueue(ctx context.Context) error {
	items, err := w.db.DequeueItems(w.config.BatchSize)
	if err != nil {
		return fmt.Errorf("failed to dequeue items: %w", err)
	}

	if len(items) == 0 {
		return nil
	}

	w.logger.Debug("processing queue items", "count", len(items))

	for _, item := range items {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-w.stopCh:
			return nil
		default:
		}

		if err := w.processItem(ctx, &item); err != nil {
			w.logger.Error("error processing item",
				"push_id", item.PushID,
				"repo", fmt.Sprintf("%s/%s", item.RepoOwner, item.RepoName),
				"error", err)
		}
	}

	return nil
}

// processItem processes a single queue item
func (w *Worker) processItem(ctx context.Context, item *QueueItem) error {
	// Update status to checking
	item.Status = "checking"
	if err := w.db.UpdateQueueItem(item); err != nil {
		return fmt.Errorf("failed to update item status: %w", err)
	}

	// Get GitHub client
	token := w.tokenFunc()
	if token == "" {
		item.Status = "failed"
		item.Error = "no GitHub token available"

		return w.db.UpdateQueueItem(item)
	}

	client := w.createGitHubClient(ctx, token)

	// Check workflow runs for this commit
	runs, err := w.fetchWorkflowRuns(ctx, client, item.RepoOwner, item.RepoName, item.CommitSHA)
	if err != nil {
		return w.handleCheckError(item, err)
	}

	if len(runs) == 0 {
		// No workflows found yet, retry later
		return w.scheduleRetry(item, nil)
	}

	// Save workflow runs
	allComplete := true

	for _, run := range runs {
		workflowRun := w.convertRun(run, item.PushID, item.RepoOwner, item.RepoName)
		if err := w.db.SaveWorkflowRun(workflowRun); err != nil {
			w.logger.Error("failed to save workflow run", "run_id", run.GetID(), "error", err)
		}

		if run.GetStatus() != "completed" {
			allComplete = false
		} else if w.onComplete != nil {
			w.onComplete(workflowRun)
		}

		// Fetch and save jobs
		if err := w.fetchAndSaveJobs(ctx, client, item.RepoOwner, item.RepoName, run.GetID()); err != nil {
			w.logger.Error("failed to save jobs", "run_id", run.GetID(), "error", err)
		}
	}

	// Update push record
	if err := w.updatePushMonitored(item.PushID); err != nil {
		w.logger.Error("failed to update push record", "error", err)
	}

	if allComplete {
		// All workflows complete
		item.Status = "completed"
		item.Error = ""

		return w.db.UpdateQueueItem(item)
	}

	// Schedule next check
	return w.scheduleRetry(item, nil)
}

// fetchWorkflowRuns fetches workflow runs from GitHub
func (w *Worker) fetchWorkflowRuns(ctx context.Context, client *github.Client, owner, repo, sha string) ([]*github.WorkflowRun, error) {
	opts := &github.ListWorkflowRunsOptions{
		HeadSHA: sha,
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	runs, _, err := client.Actions.ListRepositoryWorkflowRuns(ctx, owner, repo, opts)
	if err != nil {
		return nil, err
	}

	return runs.WorkflowRuns, nil
}

// fetchAndSaveJobs fetches and saves jobs for a workflow run
func (w *Worker) fetchAndSaveJobs(ctx context.Context, client *github.Client, owner, repo string, runID int64) error {
	jobs, _, err := client.Actions.ListWorkflowJobs(ctx, owner, repo, runID, &github.ListWorkflowJobsOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	})
	if err != nil {
		return err
	}

	for _, job := range jobs.Jobs {
		wj := &WorkflowJob{
			JobID:      job.GetID(),
			RunID:      runID,
			Name:       job.GetName(),
			Status:     job.GetStatus(),
			Conclusion: job.GetConclusion(),
			Steps:      len(job.Steps),
		}

		if job.StartedAt != nil {
			wj.StartedAt = job.StartedAt.Time
		}

		if job.CompletedAt != nil {
			wj.CompletedAt = job.CompletedAt.Time
		}

		// Count steps
		for _, step := range job.Steps {
			switch step.GetConclusion() {
			case "success":
				wj.StepsPassed++
			case "failure":
				wj.StepsFailed++
			}
		}

		if err := w.db.SaveWorkflowJob(wj); err != nil {
			return err
		}
	}

	return nil
}

// convertRun converts a GitHub workflow run to our model
func (w *Worker) convertRun(run *github.WorkflowRun, pushID int64, owner, repo string) *WorkflowRun {
	wr := &WorkflowRun{
		RunID:        run.GetID(),
		RepoOwner:    owner,
		RepoName:     repo,
		WorkflowID:   run.GetWorkflowID(),
		WorkflowName: run.GetName(),
		HeadBranch:   run.GetHeadBranch(),
		HeadSHA:      run.GetHeadSHA(),
		Event:        run.GetEvent(),
		Status:       run.GetStatus(),
		Conclusion:   run.GetConclusion(),
		HTMLURL:      run.GetHTMLURL(),
		PushID:       pushID,
	}

	if run.CreatedAt != nil {
		wr.CreatedAt = run.CreatedAt.Time
	}

	if run.UpdatedAt != nil {
		wr.UpdatedAt = run.UpdatedAt.Time
	}

	if run.RunStartedAt != nil {
		wr.StartedAt = run.RunStartedAt.Time
	}
	// CompletedAt is derived from UpdatedAt when status is completed

	return wr
}

// handleCheckError handles an error during workflow check
func (w *Worker) handleCheckError(item *QueueItem, err error) error {
	item.RetryCount++
	item.Error = err.Error()

	if item.RetryCount >= w.config.MaxRetries {
		item.Status = "failed"
		w.logger.Warn("max retries reached",
			"push_id", item.PushID,
			"repo", fmt.Sprintf("%s/%s", item.RepoOwner, item.RepoName),
			"retries", item.RetryCount)
	} else {
		// Exponential backoff
		backoff := w.config.RetryBackoff * time.Duration(1<<uint(item.RetryCount-1))
		item.NextCheck = time.Now().Add(backoff)
		item.Status = "pending"
	}

	return w.db.UpdateQueueItem(item)
}

// scheduleRetry schedules a retry for the item
func (w *Worker) scheduleRetry(item *QueueItem, err error) error {
	if err != nil {
		item.Error = err.Error()
		item.RetryCount++
	}

	item.Status = "pending"
	item.NextCheck = time.Now().Add(w.config.CheckInterval)

	return w.db.UpdateQueueItem(item)
}

// updatePushMonitored marks a push as being monitored
func (w *Worker) updatePushMonitored(pushID int64) error {
	record, err := w.db.GetPushRecord(pushID)
	if err != nil {
		return err
	}

	if record == nil {
		return nil
	}

	record.Monitored = true
	record.LastCheck = time.Now()

	return w.db.SavePushRecord(record)
}

// createGitHubClient creates a GitHub client with the given token
func (w *Worker) createGitHubClient(ctx context.Context, token string) *github.Client {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)

	return github.NewClient(tc)
}

// EnqueuePush adds a push to the monitoring queue
func (w *Worker) EnqueuePush(owner, repo, branch, commitSHA, remote string) (*PushRecord, error) {
	now := time.Now()

	// Create push record
	record := &PushRecord{
		RepoOwner: owner,
		RepoName:  repo,
		Branch:    branch,
		CommitSHA: commitSHA,
		Remote:    remote,
		PushedAt:  now,
		Monitored: false,
	}

	if err := w.db.SavePushRecord(record); err != nil {
		return nil, fmt.Errorf("failed to save push record: %w", err)
	}

	// Create queue item
	item := &QueueItem{
		PushID:    record.ID,
		RepoOwner: owner,
		RepoName:  repo,
		CommitSHA: commitSHA,
		Status:    "pending",
		NextCheck: now.Add(w.config.CompletionWindow), // Wait a bit before first check
	}

	if err := w.db.EnqueueItem(item); err != nil {
		return nil, fmt.Errorf("failed to enqueue item: %w", err)
	}

	w.logger.Info("push enqueued for monitoring",
		"repo", fmt.Sprintf("%s/%s", owner, repo),
		"sha", commitSHA[:8])

	return record, nil
}

// GetPushStatus returns the status of a pushed commit
func (w *Worker) GetPushStatus(pushID int64) (*PushStatus, error) {
	record, err := w.db.GetPushRecord(pushID)
	if err != nil {
		return nil, err
	}

	if record == nil {
		return nil, fmt.Errorf("push record not found")
	}

	runs, err := w.db.ListWorkflowRunsByPush(pushID)
	if err != nil {
		return nil, err
	}

	status := &PushStatus{
		Push: *record,
		Runs: runs,
	}

	// Calculate overall status
	if len(runs) == 0 {
		status.OverallStatus = "pending"
	} else {
		status.OverallStatus = "success"

		for _, run := range runs {
			if run.Status != "completed" {
				status.OverallStatus = "in_progress"
				break
			}

			if run.Conclusion == "failure" {
				status.OverallStatus = "failure"
			} else if run.Conclusion == "cancelled" && status.OverallStatus == "success" {
				status.OverallStatus = "cancelled"
			}
		}
	}

	return status, nil
}

// PushStatus represents the overall status of a push and its workflows
type PushStatus struct {
	Push          PushRecord    `json:"push"`
	Runs          []WorkflowRun `json:"runs"`
	OverallStatus string        `json:"overall_status"` // "pending", "in_progress", "success", "failure", "cancelled"
}
