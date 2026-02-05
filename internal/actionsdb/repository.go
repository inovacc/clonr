package actionsdb

import (
	"database/sql"
	"fmt"
	"time"
)

// SavePushRecord saves a push record to the database
func (db *DB) SavePushRecord(record *PushRecord) error {
	result, err := db.Exec(`
		INSERT INTO push_records (repo_owner, repo_name, branch, commit_sha, remote, pushed_at, monitored, last_check)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(repo_owner, repo_name, commit_sha) DO UPDATE SET
			branch = excluded.branch,
			remote = excluded.remote,
			pushed_at = excluded.pushed_at,
			monitored = excluded.monitored,
			last_check = excluded.last_check
	`, record.RepoOwner, record.RepoName, record.Branch, record.CommitSHA,
		record.Remote, record.PushedAt, record.Monitored, record.LastCheck)
	if err != nil {
		return fmt.Errorf("failed to save push record: %w", err)
	}

	if record.ID == 0 {
		id, err := result.LastInsertId()
		if err == nil {
			record.ID = id
		}
	}

	return nil
}

// GetPushRecord retrieves a push record by ID
func (db *DB) GetPushRecord(id int64) (*PushRecord, error) {
	record := &PushRecord{}

	err := db.QueryRow(`
		SELECT id, repo_owner, repo_name, branch, commit_sha, remote, pushed_at, monitored, last_check
		FROM push_records WHERE id = ?
	`, id).Scan(&record.ID, &record.RepoOwner, &record.RepoName, &record.Branch,
		&record.CommitSHA, &record.Remote, &record.PushedAt, &record.Monitored, &record.LastCheck)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get push record: %w", err)
	}

	return record, nil
}

// GetPushRecordBySHA retrieves a push record by commit SHA
func (db *DB) GetPushRecordBySHA(owner, repo, sha string) (*PushRecord, error) {
	record := &PushRecord{}

	err := db.QueryRow(`
		SELECT id, repo_owner, repo_name, branch, commit_sha, remote, pushed_at, monitored, last_check
		FROM push_records WHERE repo_owner = ? AND repo_name = ? AND commit_sha = ?
	`, owner, repo, sha).Scan(&record.ID, &record.RepoOwner, &record.RepoName, &record.Branch,
		&record.CommitSHA, &record.Remote, &record.PushedAt, &record.Monitored, &record.LastCheck)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get push record: %w", err)
	}

	return record, nil
}

// ListPushRecords lists push records with optional filtering
func (db *DB) ListPushRecords(owner, repo string, limit int) ([]PushRecord, error) {
	query := `
		SELECT id, repo_owner, repo_name, branch, commit_sha, remote, pushed_at, monitored, last_check
		FROM push_records
	`
	args := []any{}

	if owner != "" && repo != "" {
		query += " WHERE repo_owner = ? AND repo_name = ?"

		args = append(args, owner, repo)
	} else if owner != "" {
		query += " WHERE repo_owner = ?"

		args = append(args, owner)
	}

	query += " ORDER BY pushed_at DESC"

	if limit > 0 {
		query += " LIMIT ?"

		args = append(args, limit)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list push records: %w", err)
	}

	defer func() { _ = rows.Close() }()

	var records []PushRecord

	for rows.Next() {
		var r PushRecord
		if err := rows.Scan(&r.ID, &r.RepoOwner, &r.RepoName, &r.Branch,
			&r.CommitSHA, &r.Remote, &r.PushedAt, &r.Monitored, &r.LastCheck); err != nil {
			return nil, fmt.Errorf("failed to scan push record: %w", err)
		}

		records = append(records, r)
	}

	return records, nil
}

// SaveWorkflowRun saves a workflow run to the database
func (db *DB) SaveWorkflowRun(run *WorkflowRun) error {
	result, err := db.Exec(`
		INSERT INTO workflow_runs (run_id, repo_owner, repo_name, workflow_id, workflow_name,
			head_branch, head_sha, event, status, conclusion, html_url, created_at, updated_at,
			started_at, completed_at, push_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(run_id) DO UPDATE SET
			status = excluded.status,
			conclusion = excluded.conclusion,
			updated_at = excluded.updated_at,
			started_at = excluded.started_at,
			completed_at = excluded.completed_at
	`, run.RunID, run.RepoOwner, run.RepoName, run.WorkflowID, run.WorkflowName,
		run.HeadBranch, run.HeadSHA, run.Event, run.Status, run.Conclusion, run.HTMLURL,
		run.CreatedAt, run.UpdatedAt, run.StartedAt, run.CompletedAt, run.PushID)
	if err != nil {
		return fmt.Errorf("failed to save workflow run: %w", err)
	}

	if run.ID == 0 {
		id, err := result.LastInsertId()
		if err == nil {
			run.ID = id
		}
	}

	return nil
}

// GetWorkflowRun retrieves a workflow run by GitHub run ID
func (db *DB) GetWorkflowRun(runID int64) (*WorkflowRun, error) {
	run := &WorkflowRun{}

	var startedAt, completedAt sql.NullTime

	err := db.QueryRow(`
		SELECT id, run_id, repo_owner, repo_name, workflow_id, workflow_name,
			head_branch, head_sha, event, status, conclusion, html_url, created_at, updated_at,
			started_at, completed_at, push_id
		FROM workflow_runs WHERE run_id = ?
	`, runID).Scan(&run.ID, &run.RunID, &run.RepoOwner, &run.RepoName, &run.WorkflowID,
		&run.WorkflowName, &run.HeadBranch, &run.HeadSHA, &run.Event, &run.Status,
		&run.Conclusion, &run.HTMLURL, &run.CreatedAt, &run.UpdatedAt,
		&startedAt, &completedAt, &run.PushID)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get workflow run: %w", err)
	}

	if startedAt.Valid {
		run.StartedAt = startedAt.Time
	}

	if completedAt.Valid {
		run.CompletedAt = completedAt.Time
	}

	return run, nil
}

// ListWorkflowRunsByPush lists workflow runs for a specific push
func (db *DB) ListWorkflowRunsByPush(pushID int64) ([]WorkflowRun, error) {
	rows, err := db.Query(`
		SELECT id, run_id, repo_owner, repo_name, workflow_id, workflow_name,
			head_branch, head_sha, event, status, conclusion, html_url, created_at, updated_at,
			started_at, completed_at, push_id
		FROM workflow_runs WHERE push_id = ?
		ORDER BY created_at DESC
	`, pushID)
	if err != nil {
		return nil, fmt.Errorf("failed to list workflow runs: %w", err)
	}

	defer func() { _ = rows.Close() }()

	var runs []WorkflowRun

	for rows.Next() {
		var (
			r                      WorkflowRun
			startedAt, completedAt sql.NullTime
		)

		if err := rows.Scan(&r.ID, &r.RunID, &r.RepoOwner, &r.RepoName, &r.WorkflowID,
			&r.WorkflowName, &r.HeadBranch, &r.HeadSHA, &r.Event, &r.Status,
			&r.Conclusion, &r.HTMLURL, &r.CreatedAt, &r.UpdatedAt,
			&startedAt, &completedAt, &r.PushID); err != nil {
			return nil, fmt.Errorf("failed to scan workflow run: %w", err)
		}

		if startedAt.Valid {
			r.StartedAt = startedAt.Time
		}

		if completedAt.Valid {
			r.CompletedAt = completedAt.Time
		}

		runs = append(runs, r)
	}

	return runs, nil
}

// ListWorkflowRunsBySHA lists workflow runs for a specific commit SHA
func (db *DB) ListWorkflowRunsBySHA(owner, repo, sha string) ([]WorkflowRun, error) {
	rows, err := db.Query(`
		SELECT id, run_id, repo_owner, repo_name, workflow_id, workflow_name,
			head_branch, head_sha, event, status, conclusion, html_url, created_at, updated_at,
			started_at, completed_at, push_id
		FROM workflow_runs WHERE repo_owner = ? AND repo_name = ? AND head_sha = ?
		ORDER BY created_at DESC
	`, owner, repo, sha)
	if err != nil {
		return nil, fmt.Errorf("failed to list workflow runs: %w", err)
	}

	defer func() { _ = rows.Close() }()

	var runs []WorkflowRun

	for rows.Next() {
		var (
			r                      WorkflowRun
			startedAt, completedAt sql.NullTime
		)

		if err := rows.Scan(&r.ID, &r.RunID, &r.RepoOwner, &r.RepoName, &r.WorkflowID,
			&r.WorkflowName, &r.HeadBranch, &r.HeadSHA, &r.Event, &r.Status,
			&r.Conclusion, &r.HTMLURL, &r.CreatedAt, &r.UpdatedAt,
			&startedAt, &completedAt, &r.PushID); err != nil {
			return nil, fmt.Errorf("failed to scan workflow run: %w", err)
		}

		if startedAt.Valid {
			r.StartedAt = startedAt.Time
		}

		if completedAt.Valid {
			r.CompletedAt = completedAt.Time
		}

		runs = append(runs, r)
	}

	return runs, nil
}

// SaveWorkflowJob saves a workflow job to the database
func (db *DB) SaveWorkflowJob(job *WorkflowJob) error {
	result, err := db.Exec(`
		INSERT INTO workflow_jobs (job_id, run_id, name, status, conclusion,
			started_at, completed_at, steps, steps_passed, steps_failed)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(job_id) DO UPDATE SET
			status = excluded.status,
			conclusion = excluded.conclusion,
			completed_at = excluded.completed_at,
			steps = excluded.steps,
			steps_passed = excluded.steps_passed,
			steps_failed = excluded.steps_failed
	`, job.JobID, job.RunID, job.Name, job.Status, job.Conclusion,
		job.StartedAt, job.CompletedAt, job.Steps, job.StepsPassed, job.StepsFailed)
	if err != nil {
		return fmt.Errorf("failed to save workflow job: %w", err)
	}

	if job.ID == 0 {
		id, err := result.LastInsertId()
		if err == nil {
			job.ID = id
		}
	}

	return nil
}

// ListJobsByRun lists jobs for a workflow run
func (db *DB) ListJobsByRun(runID int64) ([]WorkflowJob, error) {
	rows, err := db.Query(`
		SELECT id, job_id, run_id, name, status, conclusion, started_at, completed_at,
			steps, steps_passed, steps_failed
		FROM workflow_jobs WHERE run_id = ?
		ORDER BY started_at
	`, runID)
	if err != nil {
		return nil, fmt.Errorf("failed to list workflow jobs: %w", err)
	}

	defer func() { _ = rows.Close() }()

	var jobs []WorkflowJob

	for rows.Next() {
		var (
			j                      WorkflowJob
			startedAt, completedAt sql.NullTime
		)

		if err := rows.Scan(&j.ID, &j.JobID, &j.RunID, &j.Name, &j.Status, &j.Conclusion,
			&startedAt, &completedAt, &j.Steps, &j.StepsPassed, &j.StepsFailed); err != nil {
			return nil, fmt.Errorf("failed to scan workflow job: %w", err)
		}

		if startedAt.Valid {
			j.StartedAt = startedAt.Time
		}

		if completedAt.Valid {
			j.CompletedAt = completedAt.Time
		}

		jobs = append(jobs, j)
	}

	return jobs, nil
}

// EnqueueItem adds an item to the monitoring queue
func (db *DB) EnqueueItem(item *QueueItem) error {
	now := time.Now()
	if item.CreatedAt.IsZero() {
		item.CreatedAt = now
	}

	item.UpdatedAt = now

	result, err := db.Exec(`
		INSERT INTO queue_items (push_id, repo_owner, repo_name, commit_sha, status,
			retry_count, next_check, created_at, updated_at, error)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, item.PushID, item.RepoOwner, item.RepoName, item.CommitSHA, item.Status,
		item.RetryCount, item.NextCheck, item.CreatedAt, item.UpdatedAt, item.Error)
	if err != nil {
		return fmt.Errorf("failed to enqueue item: %w", err)
	}

	id, err := result.LastInsertId()
	if err == nil {
		item.ID = id
	}

	return nil
}

// DequeueItems retrieves items ready for processing
func (db *DB) DequeueItems(limit int) ([]QueueItem, error) {
	now := time.Now()

	rows, err := db.Query(`
		SELECT id, push_id, repo_owner, repo_name, commit_sha, status,
			retry_count, next_check, created_at, updated_at, error
		FROM queue_items
		WHERE status IN ('pending', 'checking') AND next_check <= ?
		ORDER BY next_check ASC
		LIMIT ?
	`, now, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to dequeue items: %w", err)
	}

	defer func() { _ = rows.Close() }()

	var items []QueueItem

	for rows.Next() {
		var (
			i      QueueItem
			errStr sql.NullString
		)

		if err := rows.Scan(&i.ID, &i.PushID, &i.RepoOwner, &i.RepoName, &i.CommitSHA,
			&i.Status, &i.RetryCount, &i.NextCheck, &i.CreatedAt, &i.UpdatedAt, &errStr); err != nil {
			return nil, fmt.Errorf("failed to scan queue item: %w", err)
		}

		if errStr.Valid {
			i.Error = errStr.String
		}

		items = append(items, i)
	}

	return items, nil
}

// UpdateQueueItem updates a queue item
func (db *DB) UpdateQueueItem(item *QueueItem) error {
	item.UpdatedAt = time.Now()

	_, err := db.Exec(`
		UPDATE queue_items SET
			status = ?,
			retry_count = ?,
			next_check = ?,
			updated_at = ?,
			error = ?
		WHERE id = ?
	`, item.Status, item.RetryCount, item.NextCheck, item.UpdatedAt, item.Error, item.ID)
	if err != nil {
		return fmt.Errorf("failed to update queue item: %w", err)
	}

	return nil
}

// DeleteQueueItem removes a queue item
func (db *DB) DeleteQueueItem(id int64) error {
	_, err := db.Exec("DELETE FROM queue_items WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete queue item: %w", err)
	}

	return nil
}

// GetQueueStats returns queue statistics
func (db *DB) GetQueueStats() (pending, checking, completed, failed int, err error) {
	rows, err := db.Query(`
		SELECT status, COUNT(*) FROM queue_items GROUP BY status
	`)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("failed to get queue stats: %w", err)
	}

	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			status string
			count  int
		)

		if err := rows.Scan(&status, &count); err != nil {
			return 0, 0, 0, 0, err
		}

		switch status {
		case "pending":
			pending = count
		case "checking":
			checking = count
		case "completed":
			completed = count
		case "failed":
			failed = count
		}
	}

	return pending, checking, completed, failed, nil
}

// CleanupOldRecords removes records older than the specified duration
func (db *DB) CleanupOldRecords(olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)

	// Delete old queue items first
	result, err := db.Exec(`
		DELETE FROM queue_items WHERE status = 'completed' AND updated_at < ?
	`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup queue items: %w", err)
	}

	deleted, _ := result.RowsAffected()

	// We keep push records and workflow runs for history
	// but could add cleanup here if needed

	return deleted, nil
}
