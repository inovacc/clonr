package core

import (
	"fmt"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// MirrorBatchOptions configures batch (non-TUI) mirror execution
type MirrorBatchOptions struct {
	Plan   *MirrorPlan
	Logger *slog.Logger
}

// MirrorBatchResult contains the results of a batch mirror operation
type MirrorBatchResult struct {
	Results  []MirrorResult
	Cloned   int
	Updated  int
	Skipped  int
	Failed   int
	Duration time.Duration
}

// ExecuteMirrorBatch runs the mirror operation without TUI
func ExecuteMirrorBatch(opts MirrorBatchOptions) (*MirrorBatchResult, error) {
	plan := opts.Plan

	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	start := time.Now()

	// Counters
	var (
		cloned, updated, skipped, failed atomic.Int32
		current                          atomic.Int32
	)

	total := len(plan.Repos)
	results := make([]MirrorResult, 0, total)

	var resultsMu sync.Mutex

	// Work queue
	workQueue := make(chan MirrorRepo, total)

	var wg sync.WaitGroup

	// Progress printer
	printProgress := func(repo string, action string, success bool, err error, retryCount int) {
		curr := current.Add(1)
		pct := float64(curr) / float64(total) * 100

		var (
			status string
			detail string
		)

		switch {
		case action == "skip":
			status = "SKIP"

			skipped.Add(1)
		case success:
			status = "OK"

			if action == "clone" {
				cloned.Add(1)
			} else {
				updated.Add(1)
			}
		default:
			status = "FAIL"

			failed.Add(1)

			if err != nil {
				detail = fmt.Sprintf(" - %s", err.Error())
				if len(detail) > 60 {
					detail = detail[:57] + "..."
				}
			}
		}

		retryInfo := ""
		if retryCount > 0 {
			retryInfo = fmt.Sprintf(" (retries: %d)", retryCount)
		}

		_, _ = fmt.Fprintf(os.Stdout, "[%3.0f%%] [%-5s] %-40s%s%s\n", pct, status, repo, detail, retryInfo)
	}

	// Spawn workers
	for i := 0; i < plan.Parallel; i++ {
		wg.Go(func() {
			for repo := range workQueue {
				result := processRepoBatch(repo, plan, logger)
				printProgress(repo.Name, repo.Action, result.Success, result.Error, result.RetryCount)

				resultsMu.Lock()

				results = append(results, result)

				resultsMu.Unlock()
			}
		})
	}

	// Queue work
	for _, repo := range plan.Repos {
		workQueue <- repo
	}

	close(workQueue)

	// Wait for completion
	wg.Wait()

	duration := time.Since(start)

	return &MirrorBatchResult{
		Results:  results,
		Cloned:   int(cloned.Load()),
		Updated:  int(updated.Load()),
		Skipped:  int(skipped.Load()),
		Failed:   int(failed.Load()),
		Duration: duration,
	}, nil
}

// processRepoBatch processes a single repo in batch mode
func processRepoBatch(repo MirrorRepo, plan *MirrorPlan, logger *slog.Logger) MirrorResult {
	start := time.Now()

	var err error

	retryCount := 0

	switch repo.Action {
	case "clone":
		err = executeWithNetworkRetryBatch(func() error {
			return MirrorCloneRepo(repo.URL, repo.Path, plan.Shallow)
		}, plan.NetworkRetries, &retryCount)
		if err == nil {
			err = SaveMirroredRepo(repo.URL, repo.Path)
		}

	case "update":
		err = executeWithNetworkRetryBatch(func() error {
			return MirrorUpdateRepo(repo.URL, repo.Path, plan.DirtyStrategy, logger)
		}, plan.NetworkRetries, &retryCount)
		if err == nil {
			err = SaveMirroredRepo(repo.URL, repo.Path)
		}

	case "skip":
		err = nil
	}

	duration := time.Since(start)

	return MirrorResult{
		Repo:       repo,
		Success:    err == nil || repo.Action == "skip",
		Error:      err,
		Duration:   duration.Milliseconds(),
		RetryCount: retryCount,
	}
}

// executeWithNetworkRetryBatch wraps an operation with network retry logic
func executeWithNetworkRetryBatch(op func() error, maxRetries int, retryCount *int) error {
	if maxRetries == 0 {
		maxRetries = 3
	}

	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		err := op()
		if err == nil {
			return nil
		}

		// Check if it's a network error
		if !IsNetworkError(err) {
			return err // Non-retryable error
		}

		*retryCount++
		lastErr = err

		// Exponential backoff: 1s, 2s, 4s...
		backoff := min(time.Duration(1<<attempt)*time.Second, 30*time.Second)

		time.Sleep(backoff)
	}

	return &NetworkError{
		Operation: "git operation",
		Err:       lastErr,
		Attempts:  maxRetries,
	}
}

// PrintBatchSummary prints a summary of the batch mirror results
func PrintBatchSummary(result *MirrorBatchResult) {
	_, _ = fmt.Fprintln(os.Stdout)
	_, _ = fmt.Fprintln(os.Stdout, "═══════════════════════════════════════════════════════════")
	_, _ = fmt.Fprintln(os.Stdout, "                    Mirror Complete")
	_, _ = fmt.Fprintln(os.Stdout, "═══════════════════════════════════════════════════════════")
	_, _ = fmt.Fprintf(os.Stdout, "  Cloned:   %d\n", result.Cloned)
	_, _ = fmt.Fprintf(os.Stdout, "  Updated:  %d\n", result.Updated)
	_, _ = fmt.Fprintf(os.Stdout, "  Skipped:  %d\n", result.Skipped)
	_, _ = fmt.Fprintf(os.Stdout, "  Failed:   %d\n", result.Failed)
	_, _ = fmt.Fprintln(os.Stdout, "───────────────────────────────────────────────────────────")
	_, _ = fmt.Fprintf(os.Stdout, "  Total:    %d repositories in %s\n",
		result.Cloned+result.Updated+result.Skipped+result.Failed,
		result.Duration.Round(time.Millisecond))
	_, _ = fmt.Fprintln(os.Stdout, "═══════════════════════════════════════════════════════════")

	// Show failed repos if any
	if result.Failed > 0 {
		_, _ = fmt.Fprintln(os.Stdout, "\nFailed repositories:")

		for _, r := range result.Results {
			if !r.Success && r.Repo.Action != "skip" {
				errMsg := "unknown error"
				if r.Error != nil {
					errMsg = r.Error.Error()
					if len(errMsg) > 70 {
						errMsg = errMsg[:67] + "..."
					}
				}

				_, _ = fmt.Fprintf(os.Stdout, "  - %s: %s\n", r.Repo.Name, errMsg)
			}
		}
	}
}
