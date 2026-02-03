package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/inovacc/clonr/internal/actionsdb"
	"github.com/spf13/cobra"
)

var (
	monitorLimit int
	monitorRepo  string
)

var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "View GitHub Actions status for pushed commits",
	Long: `View GitHub Actions workflow status for commits pushed via clonr.

When you push using 'clonr push', the commit is automatically queued for
GitHub Actions monitoring. This command shows the status of those workflows.

Examples:
  clonr monitor                    # Show recent push statuses
  clonr monitor --limit 20         # Show last 20 pushes
  clonr monitor --repo owner/repo  # Filter by repository`,
	RunE: runMonitor,
}

func init() {
	rootCmd.AddCommand(monitorCmd)
	monitorCmd.Flags().IntVarP(&monitorLimit, "limit", "n", 10, "Number of pushes to show")
	monitorCmd.Flags().StringVarP(&monitorRepo, "repo", "r", "", "Filter by repository (owner/repo)")
}

func runMonitor(_ *cobra.Command, _ []string) error {
	dbPath, err := actionsdb.DefaultDBPath()
	if err != nil {
		return fmt.Errorf("failed to get database path: %w", err)
	}

	db, err := actionsdb.Open(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open actions database: %w", err)
	}
	defer func() { _ = db.Close() }()

	// Parse repo filter
	var owner, repo string
	if monitorRepo != "" {
		parts := strings.Split(monitorRepo, "/")
		if len(parts) == 2 {
			owner, repo = parts[0], parts[1]
		} else {
			owner = monitorRepo
		}
	}

	// Get push records
	records, err := db.ListPushRecords(owner, repo, monitorLimit)
	if err != nil {
		return fmt.Errorf("failed to list push records: %w", err)
	}

	if len(records) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, "No push records found.")
		_, _ = fmt.Fprintln(os.Stdout, dimStyle.Render("\nPush commits using 'clonr push' to enable actions monitoring."))
		return nil
	}

	// Get queue stats
	pending, checking, completed, failed, _ := db.GetQueueStats()

	// Header
	_, _ = fmt.Fprintln(os.Stdout, "")
	_, _ = fmt.Fprintln(os.Stdout, lipgloss.NewStyle().Bold(true).Render("GitHub Actions Monitor"))
	_, _ = fmt.Fprintln(os.Stdout, strings.Repeat("─", 60))

	// Queue stats
	_, _ = fmt.Fprintf(os.Stdout, "Queue: %d pending, %d checking, %d completed, %d failed\n\n",
		pending, checking, completed, failed)

	// Display each push
	for _, record := range records {
		displayPushRecord(db, &record)
	}

	return nil
}

func displayPushRecord(db *actionsdb.DB, record *actionsdb.PushRecord) {
	// Get workflow runs for this push
	runs, _ := db.ListWorkflowRunsByPush(record.ID)

	// Status indicator
	statusIcon := "○" // Pending
	statusStyle := dimStyle

	if len(runs) > 0 {
		allComplete := true
		hasFailure := false
		for _, run := range runs {
			if run.Status != "completed" {
				allComplete = false
			}
			if run.Conclusion == "failure" {
				hasFailure = true
			}
		}

		if allComplete {
			if hasFailure {
				statusIcon = "✗"
				statusStyle = errStyle
			} else {
				statusIcon = "✓"
				statusStyle = okStyle
			}
		} else {
			statusIcon = "●"
			statusStyle = warnStyle
		}
	}

	// Repository name
	repoName := fmt.Sprintf("%s/%s", record.RepoOwner, record.RepoName)

	// Time ago
	timeAgo := formatAge(record.PushedAt)

	_, _ = fmt.Fprintf(os.Stdout, "%s %s %s %s\n",
		statusStyle.Render(statusIcon),
		lipgloss.NewStyle().Bold(true).Render(repoName),
		dimStyle.Render("·"),
		dimStyle.Render(timeAgo))

	_, _ = fmt.Fprintf(os.Stdout, "  %s → %s\n",
		dimStyle.Render(record.Branch),
		dimStyle.Render(record.CommitSHA[:8]))

	// Show workflow runs
	if len(runs) > 0 {
		for _, run := range runs {
			runIcon := "○"
			runStyle := dimStyle

			switch run.Status {
			case "completed":
				switch run.Conclusion {
				case "success":
					runIcon = "✓"
					runStyle = okStyle
				case "failure":
					runIcon = "✗"
					runStyle = errStyle
				case "cancelled":
					runIcon = "⊘"
					runStyle = warnStyle
				default:
					runIcon = "?"
				}
			case "in_progress":
				runIcon = "●"
				runStyle = warnStyle
			case "queued":
				runIcon = "○"
			}

			duration := ""
			if !run.StartedAt.IsZero() && !run.CompletedAt.IsZero() {
				d := run.CompletedAt.Sub(run.StartedAt)
				duration = fmt.Sprintf(" (%s)", formatShortDuration(d))
			}

			_, _ = fmt.Fprintf(os.Stdout, "    %s %s%s\n",
				runStyle.Render(runIcon),
				run.WorkflowName,
				dimStyle.Render(duration))
		}
	} else if record.Monitored {
		_, _ = fmt.Fprintln(os.Stdout, dimStyle.Render("    No workflows detected"))
	} else {
		_, _ = fmt.Fprintln(os.Stdout, dimStyle.Render("    Waiting for workflows..."))
	}

	_, _ = fmt.Fprintln(os.Stdout, "")
}
