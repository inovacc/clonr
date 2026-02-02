package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/inovacc/clonr/internal/core"
	"github.com/spf13/cobra"
)

var actionsCmd = &cobra.Command{
	Use:   "actions",
	Short: "Check GitHub Actions workflow status",
	Long: `Check the status of GitHub Actions workflow runs.

Repository Detection:
  Commands auto-detect the repository from the current directory,
  or you can specify it explicitly as owner/repo.

Examples:
  clonr gh actions status                    # Recent workflow runs
  clonr gh actions status --branch main      # Filter by branch
  clonr gh actions status 123456789          # Specific run details`,
}

var actionsStatusCmd = &cobra.Command{
	Use:   "status [run-id | owner/repo]",
	Short: "Check workflow run status",
	Long: `Check the status of GitHub Actions workflow runs.

Without a run ID, lists recent workflow runs.
With a run ID, shows detailed status of that specific run including jobs.

Examples:
  clonr gh actions status                    # List recent runs
  clonr gh actions status 123456789          # Detailed status of run
  clonr gh actions status --branch main      # Filter by branch
  clonr gh actions status --event push       # Filter by event type
  clonr gh actions status --status completed # Filter by status`,
	RunE: runActionsStatus,
}

func init() {
	ghCmd.AddCommand(actionsCmd)
	actionsCmd.AddCommand(actionsStatusCmd)

	// Status flags
	addGHCommonFlags(actionsStatusCmd)
	actionsStatusCmd.Flags().String("branch", "", "Filter by branch name")
	actionsStatusCmd.Flags().String("event", "", "Filter by event type (push, pull_request, schedule, etc.)")
	actionsStatusCmd.Flags().String("status", "", "Filter by status (queued, in_progress, completed)")
	actionsStatusCmd.Flags().String("actor", "", "Filter by actor (username)")
	actionsStatusCmd.Flags().Int("limit", 20, "Maximum number of runs to list (0 = unlimited)")
	actionsStatusCmd.Flags().Bool("jobs", false, "Include job details (for specific run)")
}

func runActionsStatus(cmd *cobra.Command, args []string) error {
	// Get common and command-specific flags
	flags := extractGHFlags(cmd)
	branch, _ := cmd.Flags().GetString("branch")
	event, _ := cmd.Flags().GetString("event")
	status, _ := cmd.Flags().GetString("status")
	actor, _ := cmd.Flags().GetString("actor")
	limit, _ := cmd.Flags().GetInt("limit")
	includeJobs, _ := cmd.Flags().GetBool("jobs")

	// Resolve token
	token, _, err := core.ResolveGitHubToken(flags.Token, flags.Profile)
	if err != nil {
		return err
	}

	// Parse arguments - could be run ID or owner/repo
	var (
		runID   int64
		repoArg string
	)

	for _, arg := range args {
		if n, err := strconv.ParseInt(arg, 10, 64); err == nil {
			runID = n
		} else {
			repoArg = arg
		}
	}

	// Detect repository
	owner, repo, err := detectRepo([]string{repoArg}, flags.Repo, "Specify a repository with: clonr gh actions status owner/repo")
	if err != nil {
		return err
	}

	// If specific run ID, show detailed status
	if runID > 0 {
		return showRunDetail(token, owner, repo, runID, includeJobs, flags.JSON)
	}

	// Otherwise, list runs
	return listRuns(token, owner, repo, flags.JSON, branch, event, status, actor, limit)
}

func showRunDetail(token, owner, repo string, runID int64, includeJobs, jsonOutput bool) error {
	if !jsonOutput {
		_, _ = fmt.Fprintf(os.Stderr, "Fetching workflow run %d from %s/%s...\n", runID, owner, repo)
	}

	detail, err := core.GetWorkflowRunStatus(token, owner, repo, runID, core.GetWorkflowRunOptions{
		IncludeJobs: includeJobs,
	})
	if err != nil {
		return fmt.Errorf("failed to get workflow run: %w", err)
	}

	if jsonOutput {
		return outputJSON(detail)
	}

	// Text output
	printRunDetail(detail)

	return nil
}

func listRuns(token, owner, repo string, jsonOutput bool, branch, event, status, actor string, limit int) error {
	if !jsonOutput {
		_, _ = fmt.Fprintf(os.Stderr, "Fetching workflow runs for %s/%s...\n", owner, repo)
	}

	opts := core.ListWorkflowRunsOptions{
		Branch: branch,
		Event:  event,
		Status: status,
		Actor:  actor,
		Limit:  limit,
	}

	data, err := core.ListWorkflowRuns(token, owner, repo, opts)
	if err != nil {
		return fmt.Errorf("failed to list workflow runs: %w", err)
	}

	if jsonOutput {
		return outputJSON(data)
	}

	// Text output
	if len(data.Runs) == 0 {
		_, _ = fmt.Fprintf(os.Stdout, "No workflow runs found in %s/%s\n", owner, repo)
		return nil
	}

	_, _ = fmt.Fprintf(os.Stdout, "\nWorkflow Runs for %s/%s (%d shown)\n\n",
		owner, repo, len(data.Runs))

	for _, run := range data.Runs {
		printRunSummary(&run)
	}

	return nil
}

func printRunSummary(run *core.WorkflowRun) {
	// Status icon
	statusIcon := getStatusIcon(run.Status, run.Conclusion)

	// Duration or status
	var durationStr string

	switch {
	case run.Duration != "":
		durationStr = fmt.Sprintf(" (%s)", run.Duration)
	case run.Status == "in_progress":
		durationStr = " (running)"
	case run.Status == "queued":
		durationStr = " (queued)"
	}

	// Workflow name
	name := run.Name
	if name == "" {
		name = fmt.Sprintf("Run #%d", run.RunNumber)
	}

	_, _ = fmt.Fprintf(os.Stdout, "%s %-40s %s%s\n",
		statusIcon,
		truncate(name, 40),
		run.Branch,
		durationStr)

	// Second line with details
	_, _ = fmt.Fprintf(os.Stdout, "   #%-8d %s · %s · %s by @%s\n",
		run.RunNumber,
		run.HeadSHA,
		run.Event,
		formatAge(run.CreatedAt),
		run.Actor)
}

func printRunDetail(detail *core.WorkflowRunDetail) {
	run := &detail.Run

	// Header
	_, _ = fmt.Fprintf(os.Stdout, "\n%s %s\n", getStatusIcon(run.Status, run.Conclusion), run.Name)
	_, _ = fmt.Fprintf(os.Stdout, "Run #%d · Attempt %d\n", run.RunNumber, run.RunAttempt)

	// Status
	statusStr := run.Status
	if run.Conclusion != "" {
		statusStr = run.Conclusion
	}

	_, _ = fmt.Fprintf(os.Stdout, "Status: %s\n", statusStr)

	// Branch and commit
	_, _ = fmt.Fprintf(os.Stdout, "Branch: %s (%s)\n", run.Branch, run.HeadSHA)
	if run.HeadCommit != "" {
		_, _ = fmt.Fprintf(os.Stdout, "Commit: %s\n", run.HeadCommit)
	}

	// Event and actor
	_, _ = fmt.Fprintf(os.Stdout, "Event: %s\n", run.Event)
	_, _ = fmt.Fprintf(os.Stdout, "Actor: @%s\n", run.Actor)

	// Timing
	_, _ = fmt.Fprintf(os.Stdout, "Started: %s\n", formatAge(run.CreatedAt))
	if run.Duration != "" {
		_, _ = fmt.Fprintf(os.Stdout, "Duration: %s\n", run.Duration)
	}

	// URL
	_, _ = fmt.Fprintf(os.Stdout, "URL: %s\n", run.URL)

	// Jobs
	if len(detail.Jobs) > 0 {
		_, _ = fmt.Fprintln(os.Stdout, "\nJobs:")

		for _, job := range detail.Jobs {
			printJobDetail(&job)
		}
	}
}

func printJobDetail(job *core.WorkflowJob) {
	statusIcon := getStatusIcon(job.Status, job.Conclusion)
	durationStr := ""

	// Calculate duration
	if job.StartedAt != nil && job.CompletedAt != nil {
		d := job.CompletedAt.Sub(*job.StartedAt)
		durationStr = fmt.Sprintf(" (%s)", formatJobDuration(d))
	} else if job.Status == "in_progress" {
		durationStr = " (running)"
	}

	_, _ = fmt.Fprintf(os.Stdout, "  %s %s%s\n", statusIcon, job.Name, durationStr)

	// Show steps for failed or in-progress jobs
	if (job.Conclusion == "failure" || job.Status == "in_progress") && len(job.Steps) > 0 {
		for _, step := range job.Steps {
			stepIcon := getStatusIcon(step.Status, step.Conclusion)
			if step.Status != "completed" || step.Conclusion != "success" {
				_, _ = fmt.Fprintf(os.Stdout, "    %s %s\n", stepIcon, step.Name)
			}
		}
	}
}

func getStatusIcon(status, conclusion string) string {
	if status == "completed" {
		switch conclusion {
		case "success":
			return "✅"
		case "failure":
			return "❌"
		case "cancelled":
			return "🚫"
		case "skipped":
			return "⏭️"
		case "timed_out":
			return "⏰"
		case "action_required":
			return "⚠️"
		default:
			return "❓"
		}
	}

	switch status {
	case "queued", "waiting":
		return "🕐"
	case "in_progress":
		return "🔄"
	default:
		return "❓"
	}
}

func formatJobDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}

	mins := int(d.Minutes())
	secs := int(d.Seconds()) % 60

	return fmt.Sprintf("%dm %ds", mins, secs)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s + strings.Repeat(" ", maxLen-len(s))
	}

	return s[:maxLen-3] + "..."
}
