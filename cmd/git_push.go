package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/inovacc/clonr/internal/actionsdb"
	"github.com/inovacc/clonr/internal/core"
	"github.com/inovacc/clonr/internal/git"
	"github.com/inovacc/clonr/internal/security"
	"github.com/spf13/cobra"
)

var gitPushCmd = &cobra.Command{
	Use:   "push [remote] [branch]",
	Short: "Push changes to remote repository",
	Long: `Update remote refs along with associated objects.

Uses clonr profile authentication automatically. By default, scans for
secrets before pushing to prevent accidental credential leaks.

Examples:
  clonr git push
  clonr git push origin main
  clonr git push -u origin feature-branch
  clonr git push --tags
  clonr git push --skip-leaks  # Skip security scan (not recommended)`,
	RunE: runGitPush,
}

func init() {
	gitCmd.AddCommand(gitPushCmd)
	gitPushCmd.Flags().BoolP("set-upstream", "u", false, "Set upstream for the current branch")
	gitPushCmd.Flags().BoolP("force", "f", false, "Force push (use with caution)")
	gitPushCmd.Flags().Bool("tags", false, "Push all tags")
	gitPushCmd.Flags().Bool("skip-leaks", false, "Skip pre-push secret scanning")
}

type gitPushScanModel struct {
	spinner  spinner.Model
	scanning bool
	done     bool
	result   *security.ScanResult
	err      error
}

type gitPushScanDoneMsg struct {
	result *security.ScanResult
	err    error
}

func newGitPushScanModel() gitPushScanModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = spinStyle

	return gitPushScanModel{spinner: s, scanning: true}
}

func (m gitPushScanModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.runScan)
}

func (m gitPushScanModel) runScan() tea.Msg {
	scanner, err := security.NewLeakScanner()
	if err != nil {
		return gitPushScanDoneMsg{err: err}
	}

	repoPath, _ := os.Getwd()
	_ = scanner.LoadGitleaksIgnore(repoPath)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	result, err := scanner.ScanUnpushedCommits(ctx, repoPath)

	return gitPushScanDoneMsg{result: result, err: err}
}

func (m gitPushScanModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case gitPushScanDoneMsg:
		m.scanning = false
		m.done = true
		m.result = msg.result
		m.err = msg.err

		return m, tea.Quit
	case spinner.TickMsg:
		var cmd tea.Cmd

		m.spinner, cmd = m.spinner.Update(msg)

		return m, cmd
	}

	return m, nil
}

func (m gitPushScanModel) View() string {
	if m.done {
		if m.err != nil {
			return warnStyle.Render("  Warning: ") + dimStyle.Render(m.err.Error()) + "\n"
		}

		if m.result != nil && m.result.HasLeaks {
			return errStyle.Render(fmt.Sprintf("  Found %d secret(s)\n", len(m.result.Findings)))
		}

		return okStyle.Render("  No secrets detected\n")
	}

	if m.scanning {
		return fmt.Sprintf("  %s Scanning for secrets...\n", m.spinner.View())
	}

	return ""
}

func runGitPush(cmd *cobra.Command, args []string) error {
	client := git.NewClient()
	ctx := context.Background()

	if !client.IsRepository(ctx) {
		return fmt.Errorf("not a git repository")
	}

	skipLeaks, _ := cmd.Flags().GetBool("skip-leaks")
	setUpstream, _ := cmd.Flags().GetBool("set-upstream")
	force, _ := cmd.Flags().GetBool("force")
	tags, _ := cmd.Flags().GetBool("tags")

	// Pre-push security scan
	if !skipLeaks {
		_, _ = fmt.Fprintln(os.Stdout, "\n"+dimStyle.Render("Pre-push security check"))
		_, _ = fmt.Fprintln(os.Stdout, dimStyle.Render("───────────────────────"))

		m := newGitPushScanModel()
		p := tea.NewProgram(m, tea.WithInput(nil))

		finalModel, err := p.Run()
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, warnStyle.Render("  Warning: scan failed: %v\n"), err)
		} else {
			scanM := finalModel.(gitPushScanModel)
			if scanM.result != nil && scanM.result.HasLeaks {
				_, _ = fmt.Fprint(os.Stderr, security.FormatFindings(scanM.result.Findings))
				_, _ = fmt.Fprintln(os.Stderr, errStyle.Render("\nPush aborted: secrets detected!"))
				_, _ = fmt.Fprintln(os.Stderr, dimStyle.Render("Use --skip-leaks to push anyway (not recommended)"))

				return fmt.Errorf("secrets detected in commits")
			}
		}

		_, _ = fmt.Fprintln(os.Stdout, "")
	}

	var remote, branch string
	if len(args) >= 1 {
		remote = args[0]
	}

	if len(args) >= 2 {
		branch = args[1]
	}

	opts := git.PushOptions{
		SetUpstream: setUpstream,
		Force:       force,
		Tags:        tags,
	}

	if err := client.Push(ctx, remote, branch, opts); err != nil {
		if git.IsAuthRequired(err) {
			return fmt.Errorf("authentication failed - check your profile token")
		}

		if git.IsNoUpstream(err) {
			return fmt.Errorf("no upstream branch configured - use -u to set upstream")
		}

		return err
	}

	_, _ = fmt.Fprintln(os.Stdout, okStyle.Render("Push completed successfully!"))

	// Enqueue for GitHub Actions monitoring
	if err := enqueueGitPushForMonitoring(ctx, remote); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, dimStyle.Render("  Note: Could not enqueue for actions monitoring: %v\n"), err)
	}

	// Send push notification (async, non-blocking)
	repoPath, _ := os.Getwd()

	actualBranch := branch
	if actualBranch == "" {
		if b, err := client.CurrentBranch(ctx); err == nil {
			actualBranch = b
		}
	}

	go core.NotifyPush(ctx, repoPath, remote, actualBranch)

	return nil
}

func enqueueGitPushForMonitoring(ctx context.Context, remote string) error {
	repoPath, err := os.Getwd()
	if err != nil {
		return err
	}

	remoteURL, err := getRemoteURL(repoPath, remote)
	if err != nil {
		return err
	}

	owner, repo, err := parseGitHubURL(remoteURL)
	if err != nil {
		return err
	}

	branch, err := getCurrentBranch(repoPath)
	if err != nil {
		return err
	}

	commitSHA, err := getHeadCommitSHA(repoPath)
	if err != nil {
		return err
	}

	dbPath, err := actionsdb.DefaultDBPath()
	if err != nil {
		return err
	}

	db, err := actionsdb.Open(dbPath)
	if err != nil {
		return err
	}

	defer func() { _ = db.Close() }()

	record := &actionsdb.PushRecord{
		RepoOwner: owner,
		RepoName:  repo,
		Branch:    branch,
		CommitSHA: commitSHA,
		Remote:    remoteURL,
		PushedAt:  time.Now(),
	}

	if err := db.SavePushRecord(record); err != nil {
		return err
	}

	item := &actionsdb.QueueItem{
		PushID:    record.ID,
		RepoOwner: owner,
		RepoName:  repo,
		CommitSHA: commitSHA,
		Status:    "pending",
		NextCheck: time.Now().Add(10 * time.Second),
	}

	if err := db.EnqueueItem(item); err != nil {
		return err
	}

	_, _ = fmt.Fprintln(os.Stdout, dimStyle.Render("  Enqueued for GitHub Actions monitoring"))

	return nil
}
