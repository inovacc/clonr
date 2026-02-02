package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/inovacc/clonr/internal/core"
	"github.com/spf13/cobra"
)

var prCmd = &cobra.Command{
	Use:   "pr",
	Short: "Check pull request status",
	Long: `Check the status of pull requests in a repository.

Repository Detection:
  Commands auto-detect the repository from the current directory,
  or you can specify it explicitly as owner/repo.

Examples:
  clonr gh pr status                    # List open PRs in current repo
  clonr gh pr status 123                # Check specific PR status
  clonr gh pr status owner/repo         # List PRs in specified repo`,
}

var prStatusCmd = &cobra.Command{
	Use:   "status [pr-number | owner/repo]",
	Short: "Check pull request status",
	Long: `Check the status of pull requests.

Without a PR number, lists all open pull requests.
With a PR number, shows detailed status of that PR including:
  - Review state (approved, changes requested, pending)
  - CI check status
  - Merge status

Examples:
  clonr gh pr status                    # List open PRs
  clonr gh pr status 123                # Detailed status of PR #123
  clonr gh pr status --state all        # List all PRs (open + closed)
  clonr gh pr status --base main        # Filter by base branch`,
	RunE: runPRStatus,
}

func init() {
	ghCmd.AddCommand(prCmd)
	prCmd.AddCommand(prStatusCmd)

	// Status flags
	addGHCommonFlags(prStatusCmd)
	prStatusCmd.Flags().String("state", "open", "Filter by state: open, closed, all")
	prStatusCmd.Flags().String("base", "", "Filter by base branch")
	prStatusCmd.Flags().String("head", "", "Filter by head branch (user:branch)")
	prStatusCmd.Flags().String("sort", "created", "Sort by: created, updated, popularity, long-running")
	prStatusCmd.Flags().String("order", "desc", "Sort order: asc, desc")
	prStatusCmd.Flags().Int("limit", 30, "Maximum number of PRs to list (0 = unlimited)")
}

func runPRStatus(cmd *cobra.Command, args []string) error {
	// Get common and command-specific flags
	flags := extractGHFlags(cmd)
	state, _ := cmd.Flags().GetString("state")
	base, _ := cmd.Flags().GetString("base")
	head, _ := cmd.Flags().GetString("head")
	sortBy, _ := cmd.Flags().GetString("sort")
	order, _ := cmd.Flags().GetString("order")
	limit, _ := cmd.Flags().GetInt("limit")

	// Resolve token
	token, _, err := core.ResolveGitHubToken(flags.Token, flags.Profile)
	if err != nil {
		return err
	}

	// Parse arguments - could be PR number or owner/repo
	var (
		prNumber int
		repoArg  string
	)

	for _, arg := range args {
		if n, err := strconv.Atoi(arg); err == nil {
			prNumber = n
		} else {
			repoArg = arg
		}
	}

	// Detect repository
	owner, repo, err := detectRepo([]string{repoArg}, flags.Repo, "Specify a repository with: clonr gh pr status owner/repo")
	if err != nil {
		return err
	}

	// If specific PR number, show detailed status
	if prNumber > 0 {
		return showPRDetail(token, owner, repo, prNumber, flags.JSON)
	}

	// Otherwise, list PRs
	return listPRs(token, owner, repo, flags.JSON, state, base, head, sortBy, order, limit)
}

func showPRDetail(token, owner, repo string, prNumber int, jsonOutput bool) error {
	if !jsonOutput {
		_, _ = fmt.Fprintf(os.Stderr, "Fetching PR #%d from %s/%s...\n", prNumber, owner, repo)
	}

	status, err := core.GetPRStatus(token, owner, repo, prNumber, core.PRStatusOptions{})
	if err != nil {
		return fmt.Errorf("failed to get PR status: %w", err)
	}

	if jsonOutput {
		return outputJSON(status)
	}

	// Text output
	printPRDetail(status)

	return nil
}

func listPRs(token, owner, repo string, jsonOutput bool, state, base, head, sortBy, order string, limit int) error {
	if !jsonOutput {
		_, _ = fmt.Fprintf(os.Stderr, "Fetching PRs for %s/%s...\n", owner, repo)
	}

	opts := core.ListPRsOptions{
		State: state,
		Sort:  sortBy,
		Order: order,
		Base:  base,
		Head:  head,
		Limit: limit,
	}

	data, err := core.ListOpenPRs(token, owner, repo, opts)
	if err != nil {
		return fmt.Errorf("failed to list PRs: %w", err)
	}

	if jsonOutput {
		return outputJSON(data)
	}

	// Text output
	if len(data.PRs) == 0 {
		_, _ = fmt.Fprintf(os.Stdout, "No %s pull requests found in %s/%s\n", state, owner, repo)
		return nil
	}

	_, _ = fmt.Fprintf(os.Stdout, "\nPull Requests for %s/%s (%d total)\n\n",
		owner, repo, data.TotalCount)

	for _, pr := range data.PRs {
		printPRSummary(&pr)
	}

	return nil
}

func printPRSummary(pr *core.PRStatus) {
	// State icon
	stateIcon := "🟢" // open
	if pr.Merged {
		stateIcon = "🟣" // merged
	} else if pr.State == "closed" {
		stateIcon = "🔴" // closed
	}

	// Draft indicator
	draftStr := ""
	if pr.Draft {
		draftStr = " [Draft]"
	}

	// Labels
	labelStr := ""
	if len(pr.Labels) > 0 {
		labelStr = fmt.Sprintf(" [%s]", strings.Join(pr.Labels, ", "))
	}

	_, _ = fmt.Fprintf(os.Stdout, "%s #%-5d %s%s%s\n", stateIcon, pr.Number, pr.Title, draftStr, labelStr)
	_, _ = fmt.Fprintf(os.Stdout, "         %s → %s · opened %s by @%s\n",
		pr.Branch, pr.BaseBranch, formatAge(pr.CreatedAt), pr.Author)
}

func printPRDetail(pr *core.PRStatus) {
	// Header
	stateStr := pr.State
	if pr.Merged {
		stateStr = "merged"
	}

	_, _ = fmt.Fprintf(os.Stdout, "\n#%d %s\n", pr.Number, pr.Title)
	_, _ = fmt.Fprintf(os.Stdout, "State: %s", stateStr)

	if pr.Draft {
		_, _ = fmt.Fprint(os.Stdout, " (draft)")
	}

	_, _ = fmt.Fprintln(os.Stdout)

	// Branch info
	_, _ = fmt.Fprintf(os.Stdout, "Branch: %s → %s\n", pr.Branch, pr.BaseBranch)

	// Author and timing
	_, _ = fmt.Fprintf(os.Stdout, "Author: @%s\n", pr.Author)
	_, _ = fmt.Fprintf(os.Stdout, "Created: %s\n", formatAge(pr.CreatedAt))

	if pr.MergedAt != nil {
		_, _ = fmt.Fprintf(os.Stdout, "Merged: %s\n", formatAge(*pr.MergedAt))
	} else if pr.ClosedAt != nil {
		_, _ = fmt.Fprintf(os.Stdout, "Closed: %s\n", formatAge(*pr.ClosedAt))
	}

	// Changes
	_, _ = fmt.Fprintf(os.Stdout, "Changes: +%d -%d across %d files\n",
		pr.Additions, pr.Deletions, pr.ChangedFiles)

	// Review status
	reviewIcon := "⏳"

	switch pr.ReviewState {
	case "approved":
		reviewIcon = "✅"
	case "changes_requested":
		reviewIcon = "🔄"
	case "commented":
		reviewIcon = "💬"
	}

	_, _ = fmt.Fprintf(os.Stdout, "Reviews: %s %s", reviewIcon, pr.ReviewState)

	if pr.ReviewCount > 0 {
		_, _ = fmt.Fprintf(os.Stdout, " (%d reviews)", pr.ReviewCount)
	}

	_, _ = fmt.Fprintln(os.Stdout)

	// Reviewers
	if len(pr.Reviewers) > 0 {
		_, _ = fmt.Fprintf(os.Stdout, "Reviewers: %s\n", strings.Join(formatUsers(pr.Reviewers), ", "))
	}

	// CI Status
	checksIcon := "⏳"

	switch pr.ChecksStatus {
	case "success":
		checksIcon = "✅"
	case "failure":
		checksIcon = "❌"
	case "none":
		checksIcon = "➖"
	}

	_, _ = fmt.Fprintf(os.Stdout, "Checks: %s %s", checksIcon, pr.ChecksStatus)

	if len(pr.Checks) > 0 {
		_, _ = fmt.Fprintf(os.Stdout, " (%d checks)", len(pr.Checks))
	}

	_, _ = fmt.Fprintln(os.Stdout)

	// Show individual checks if there are failures
	if pr.ChecksStatus == "failure" || pr.ChecksStatus == "pending" {
		for _, check := range pr.Checks {
			icon := "⏳"

			if check.Status == "completed" {
				switch check.Conclusion {
				case "success":
					icon = "✅"
				case "failure":
					icon = "❌"
				case "skipped":
					icon = "⏭️"
				case "cancelled":
					icon = "🚫"
				}
			}

			_, _ = fmt.Fprintf(os.Stdout, "  %s %s\n", icon, check.Name)
		}
	}

	// Mergeable status
	if pr.State == "open" && pr.Mergeable != nil {
		mergeIcon := "✅"
		mergeStr := "Mergeable"

		if !*pr.Mergeable {
			mergeIcon = "❌"
			mergeStr = "Not mergeable (conflicts)"
		}

		_, _ = fmt.Fprintf(os.Stdout, "Merge: %s %s\n", mergeIcon, mergeStr)
	}

	// Labels
	if len(pr.Labels) > 0 {
		_, _ = fmt.Fprintf(os.Stdout, "Labels: %s\n", strings.Join(pr.Labels, ", "))
	}

	// Assignees
	if len(pr.Assignees) > 0 {
		_, _ = fmt.Fprintf(os.Stdout, "Assignees: %s\n", strings.Join(formatUsers(pr.Assignees), ", "))
	}

	// URL
	_, _ = fmt.Fprintf(os.Stdout, "URL: %s\n", pr.URL)
}

func formatUsers(users []string) []string {
	result := make([]string, len(users))

	for i, u := range users {
		result[i] = "@" + u
	}

	return result
}
