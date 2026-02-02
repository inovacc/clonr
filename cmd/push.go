package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/inovacc/clonr/internal/actionsdb"
	"github.com/inovacc/clonr/internal/git"
	"github.com/inovacc/clonr/internal/security"
	"github.com/spf13/cobra"
)

var (
	pushTags       bool
	pushSetUp      bool
	pushForce      bool
	pushCheckLeaks bool
	pushSkipLeaks  bool
)

var pushCmd = &cobra.Command{
	Use:   "push [remote] [branch]",
	Short: "Push changes to remote repository",
	Long: `Push changes to remote repository using profile authentication.

By default, scans for secrets before pushing. Use --skip-leaks to bypass.

Examples:
  clonr push
  clonr push --tags
  clonr push -u origin main
  clonr push --skip-leaks         # Skip secret scanning`,
	RunE: runPush,
}

func init() {
	rootCmd.AddCommand(pushCmd)
	pushCmd.Flags().BoolVar(&pushTags, "tags", false, "Push all tags")
	pushCmd.Flags().BoolVarP(&pushSetUp, "set-upstream", "u", false, "Set upstream for the current branch")
	pushCmd.Flags().BoolVarP(&pushForce, "force", "f", false, "Force push")
	pushCmd.Flags().BoolVar(&pushCheckLeaks, "check-leaks", true, "Check for secrets before pushing (default: true)")
	pushCmd.Flags().BoolVar(&pushSkipLeaks, "skip-leaks", false, "Skip secret scanning")
}

// Styles for output
var (
	spinStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	okStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	warnStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	errStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	dimStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
)

type scanModel struct {
	spinner  spinner.Model
	scanning bool
	done     bool
	result   *security.ScanResult
	err      error
}

type scanDoneMsg struct {
	result *security.ScanResult
	err    error
}

func newScanModel() scanModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = spinStyle

	return scanModel{spinner: s, scanning: true}
}

func (m scanModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.runScan)
}

func (m scanModel) runScan() tea.Msg {
	scanner, err := security.NewLeakScanner()
	if err != nil {
		return scanDoneMsg{err: err}
	}

	repoPath, _ := os.Getwd()
	_ = scanner.LoadGitleaksIgnore(repoPath)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	result, err := scanner.ScanUnpushedCommits(ctx, repoPath)

	return scanDoneMsg{result: result, err: err}
}

func (m scanModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case scanDoneMsg:
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

func (m scanModel) View() string {
	if m.done {
		if m.err != nil {
			return warnStyle.Render("  ⚠ Scan warning: ") + dimStyle.Render(m.err.Error()) + "\n"
		}

		if m.result != nil && m.result.HasLeaks {
			return errStyle.Render(fmt.Sprintf("  ✗ Found %d secret(s)\n", len(m.result.Findings)))
		}

		return okStyle.Render("  ✓ No secrets detected\n")
	}

	if m.scanning {
		return fmt.Sprintf("  %s Scanning for secrets...\n", m.spinner.View())
	}

	return ""
}

func runPush(_ *cobra.Command, args []string) error {
	client := git.NewClient()
	ctx := context.Background()

	if !client.IsRepository(ctx) {
		return fmt.Errorf("not a git repository")
	}

	// Check for leaks before pushing (unless skipped)
	if pushCheckLeaks && !pushSkipLeaks {
		_, _ = fmt.Fprintln(os.Stdout, "\n"+dimStyle.Render("Pre-push security check"))
		_, _ = fmt.Fprintln(os.Stdout, dimStyle.Render("───────────────────────"))

		m := newScanModel()
		p := tea.NewProgram(m)

		finalModel, err := p.Run()
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, warnStyle.Render("  ⚠ Scan failed: %v\n"), err)
		} else {
			scanM := finalModel.(scanModel)
			if scanM.result != nil && scanM.result.HasLeaks {
				_, _ = fmt.Fprint(os.Stderr, security.FormatFindings(scanM.result.Findings))
				_, _ = fmt.Fprintln(os.Stderr, errStyle.Render("\n❌ Push aborted: secrets detected!"))
				_, _ = fmt.Fprintln(os.Stderr, dimStyle.Render("   Use --skip-leaks to push anyway (not recommended)"))

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
		SetUpstream: pushSetUp,
		Force:       pushForce,
		Tags:        pushTags,
	}

	// Use authenticated command via credential helper
	if err := client.Push(ctx, remote, branch, opts); err != nil {
		return err
	}

	_, _ = fmt.Fprintln(os.Stdout, okStyle.Render("Push completed successfully!"))

	// Enqueue for GitHub Actions monitoring
	if err := enqueueForActionsMonitoring(ctx, remote); err != nil {
		// Non-fatal error - just log it
		_, _ = fmt.Fprintf(os.Stderr, dimStyle.Render("  Note: Could not enqueue for actions monitoring: %v\n"), err)
	}

	return nil
}

// enqueueForActionsMonitoring adds the push to the GitHub Actions monitoring queue
func enqueueForActionsMonitoring(ctx context.Context, remote string) error {
	// Get current repo info
	repoPath, err := os.Getwd()
	if err != nil {
		return err
	}

	// Get the remote URL
	remoteURL, err := getRemoteURL(repoPath, remote)
	if err != nil {
		return err
	}

	// Parse owner and repo from URL
	owner, repo, err := parseGitHubURL(remoteURL)
	if err != nil {
		return err // Not a GitHub repo, skip monitoring
	}

	// Get current branch
	branch, err := getCurrentBranch(repoPath)
	if err != nil {
		return err
	}

	// Get HEAD commit SHA
	commitSHA, err := getHeadCommitSHA(repoPath)
	if err != nil {
		return err
	}

	// Open actions database
	dbPath, err := actionsdb.DefaultDBPath()
	if err != nil {
		return err
	}

	db, err := actionsdb.Open(dbPath)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	// Create push record and enqueue
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

	// Enqueue for monitoring
	item := &actionsdb.QueueItem{
		PushID:    record.ID,
		RepoOwner: owner,
		RepoName:  repo,
		CommitSHA: commitSHA,
		Status:    "pending",
		NextCheck: time.Now().Add(10 * time.Second), // Wait a bit before first check
	}

	if err := db.EnqueueItem(item); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(os.Stdout, dimStyle.Render("  📊 Enqueued for GitHub Actions monitoring\n"))
	return nil
}

// getRemoteURL gets the URL for a remote
func getRemoteURL(repoPath, remote string) (string, error) {
	if remote == "" {
		remote = "origin"
	}
	cmd := exec.Command("git", "-C", repoPath, "remote", "get-url", remote)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// parseGitHubURL parses owner and repo from a GitHub URL
func parseGitHubURL(url string) (owner, repo string, err error) {
	// Handle SSH URLs: git@github.com:owner/repo.git
	if strings.HasPrefix(url, "git@github.com:") {
		path := strings.TrimPrefix(url, "git@github.com:")
		path = strings.TrimSuffix(path, ".git")
		parts := strings.Split(path, "/")
		if len(parts) >= 2 {
			return parts[0], parts[1], nil
		}
	}

	// Handle HTTPS URLs: https://github.com/owner/repo.git
	if strings.Contains(url, "github.com/") {
		idx := strings.Index(url, "github.com/")
		path := url[idx+len("github.com/"):]
		path = strings.TrimSuffix(path, ".git")
		parts := strings.Split(path, "/")
		if len(parts) >= 2 {
			return parts[0], parts[1], nil
		}
	}

	return "", "", fmt.Errorf("not a GitHub URL: %s", url)
}

// getCurrentBranch gets the current branch name
func getCurrentBranch(repoPath string) (string, error) {
	cmd := exec.Command("git", "-C", repoPath, "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// getHeadCommitSHA gets the HEAD commit SHA
func getHeadCommitSHA(repoPath string) (string, error) {
	cmd := exec.Command("git", "-C", repoPath, "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}
