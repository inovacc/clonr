package cli

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/inovacc/clonr/internal/core"
)

var (
	warningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	infoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	dimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	boldStyle    = lipgloss.NewStyle().Bold(true)
)

// MirrorModel represents the state of the mirror TUI
type MirrorModel struct {
	plan    *core.MirrorPlan
	results []core.MirrorResult

	// Progress tracking
	total   int
	cloned  int
	updated int
	skipped int
	failed  int
	current int

	// Active operations (limited by parallel count)
	active map[string]activeOperation
	mu     sync.Mutex

	// Recent activity log (last N completed operations)
	activity []activityItem

	// UI components
	spinner  spinner.Model
	progress progress.Model

	// State
	done   bool
	paused bool
	err    error

	// Channels for worker coordination
	workQueue chan core.MirrorRepo
	resultCh  chan core.MirrorResult
	doneCh    chan struct{}
}

type activeOperation struct {
	repo      core.MirrorRepo
	startTime time.Time
}

type activityItem struct {
	repo       string
	action     string
	status     string // "success", "skip", or "error"
	duration   time.Duration
	message    string
	retryCount int
}

// Message types
type mirrorResultMsg struct {
	result core.MirrorResult
}

type mirrorStartMsg struct {
	repo core.MirrorRepo
}

type mirrorDoneMsg struct{}

// NewMirrorModel creates a new mirror TUI model
func NewMirrorModel(plan *core.MirrorPlan) *MirrorModel {
	m := &MirrorModel{
		plan:      plan,
		total:     len(plan.Repos),
		workQueue: make(chan core.MirrorRepo, len(plan.Repos)),
		resultCh:  make(chan core.MirrorResult, len(plan.Repos)),
		doneCh:    make(chan struct{}),
		activity:  make([]activityItem, 0, 10),
		active:    make(map[string]activeOperation),
		results:   make([]core.MirrorResult, 0, len(plan.Repos)),
	}

	// Initialize UI components
	m.spinner = spinner.New()
	m.spinner.Spinner = spinner.Dot
	m.spinner.Style = spinnerStyle

	m.progress = progress.New(progress.WithDefaultGradient())

	return m
}

func (m *MirrorModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.startWorkers(),
		m.queueWork(),
		m.waitForResults(),
	)
}

func (m *MirrorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch keyMsg := msg.(type) {
	case tea.KeyMsg:
		switch keyMsg.String() {
		case "q", "ctrl+c":
			// Close channels to stop workers
			close(m.workQueue)
			close(m.doneCh)

			return m, tea.Quit
		case "p":
			m.paused = !m.paused
			return m, nil
		}

	case mirrorStartMsg:
		m.mu.Lock()
		m.active[keyMsg.repo.Name] = activeOperation{
			repo:      keyMsg.repo,
			startTime: time.Now(),
		}
		m.mu.Unlock()

		return m, nil

	case mirrorResultMsg:
		// Remove from active
		m.mu.Lock()
		delete(m.active, keyMsg.result.Repo.Name)
		m.mu.Unlock()

		// Update counters based on a result
		m.results = append(m.results, keyMsg.result)
		m.current++

		switch {
		case keyMsg.result.Repo.Action == "skip":
			m.skipped++
		case keyMsg.result.Success:
			switch keyMsg.result.Repo.Action {
			case "clone":
				m.cloned++
			case "update":
				m.updated++
			}
		default:
			m.failed++
		}

		// Add to the activity log
		m.addActivity(keyMsg.result)

		// Check if done
		if m.current >= m.total {
			m.done = true
			return m, tea.Quit
		}

		return m, m.waitForResults()

	case mirrorDoneMsg:
		m.done = true
		return m, tea.Quit

	case spinner.TickMsg:
		var cmd tea.Cmd

		m.spinner, cmd = m.spinner.Update(keyMsg)

		return m, cmd
	}

	return m, nil
}

func (m *MirrorModel) View() string {
	if m.done {
		return m.renderComplete()
	}

	var b strings.Builder

	// Header
	b.WriteString("\n")
	b.WriteString(boldStyle.Render(fmt.Sprintf("Mirroring organization: %s", m.plan.OrgName)))
	b.WriteString(dimStyle.Render(fmt.Sprintf(" (%d repositories)", m.total)))
	b.WriteString("\n\n")

	// Status counters
	b.WriteString(boldStyle.Render("Status:"))
	b.WriteString("\n")
	b.WriteString(successStyle.Render(fmt.Sprintf("  Cloned:  %d\n", m.cloned)))
	b.WriteString(infoStyle.Render(fmt.Sprintf("  Updated: %d\n", m.updated)))
	b.WriteString(warningStyle.Render(fmt.Sprintf("  Skipped: %d\n", m.skipped)))
	b.WriteString(errorStyle.Render(fmt.Sprintf("  Failed:  %d\n", m.failed)))
	b.WriteString("\n")

	// Progress bar
	pct := float64(m.current) / float64(m.total)
	b.WriteString(m.progress.ViewAs(pct))
	b.WriteString(dimStyle.Render(fmt.Sprintf(" %d/%d\n\n", m.current, m.total)))

	// Active operations
	m.mu.Lock()

	activeCount := len(m.active)
	if activeCount > 0 {
		b.WriteString(boldStyle.Render(fmt.Sprintf("Currently processing (%d):", activeCount)))
		b.WriteString("\n")

		for _, op := range m.active {
			icon := m.spinner.View()

			b.WriteString(infoStyle.Render(fmt.Sprintf("  [%s] %s - %sing...\n", icon, op.repo.Name, op.repo.Action)))
		}

		b.WriteString("\n")
	}

	m.mu.Unlock()

	// Recent activity log
	if len(m.activity) > 0 {
		b.WriteString(boldStyle.Render("Recent activity:"))
		b.WriteString("\n")

		start := max(len(m.activity)-5, 0)

		for _, item := range m.activity[start:] {
			var (
				statusIcon string
				style      lipgloss.Style
			)

			switch item.status {
			case "success":
				statusIcon = "[OK]"
				style = successStyle
			case "skip":
				statusIcon = "[SKIP]"
				style = warningStyle
			default:
				statusIcon = "[FAIL]"
				style = errorStyle
			}

			message := item.message
			if len(message) > 60 {
				message = message[:57] + "..."
			}

			// Show retry count if any retries occurred
			retryInfo := ""
			if item.retryCount > 0 {
				retryInfo = fmt.Sprintf(" (retries: %d)", item.retryCount)
			}

			b.WriteString(style.Render(fmt.Sprintf("  %s %s", statusIcon, item.repo)))
			b.WriteString(dimStyle.Render(fmt.Sprintf(" - %s%s\n", message, retryInfo)))
		}

		b.WriteString("\n")
	}

	// Footer
	b.WriteString(dimStyle.Render("Press 'q' to cancel"))
	b.WriteString("\n")

	return b.String()
}

func (m *MirrorModel) renderComplete() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(successStyle.Render("Mirror operation complete!"))
	b.WriteString("\n\n")

	return b.String()
}

// Worker goroutines
func (m *MirrorModel) startWorkers() tea.Cmd {
	return func() tea.Msg {
		var wg sync.WaitGroup

		// Spawn N workers (based on a parallel flag)
		for i := 0; i < m.plan.Parallel; i++ {
			wg.Go(func() {
				for {
					select {
					case repo, ok := <-m.workQueue:
						if !ok {
							return
						}

						result := m.processRepo(repo)
						m.resultCh <- result
					case <-m.doneCh:
						return
					}
				}
			})
		}

		// Wait for all workers to finish in a separate goroutine
		go func() {
			wg.Wait()
			close(m.resultCh)
		}()

		return nil
	}
}

func (m *MirrorModel) queueWork() tea.Cmd {
	return func() tea.Msg {
		for _, repo := range m.plan.Repos {
			m.workQueue <- repo
		}

		close(m.workQueue)

		return nil
	}
}

func (m *MirrorModel) waitForResults() tea.Cmd {
	return func() tea.Msg {
		result, ok := <-m.resultCh
		if !ok {
			return mirrorDoneMsg{}
		}

		return mirrorResultMsg{result: result}
	}
}

func (m *MirrorModel) processRepo(repo core.MirrorRepo) core.MirrorResult {
	start := time.Now()

	var err error

	retryCount := 0

	switch repo.Action {
	case "clone":
		err = m.executeWithNetworkRetry(func() error {
			return core.MirrorCloneRepo(repo.URL, repo.Path, m.plan.Shallow)
		}, &retryCount)
		if err == nil {
			err = core.SaveMirroredRepo(repo.URL, repo.Path)
		}

	case "update":
		// Get logger from plan, use default if nil
		logger := m.plan.Logger
		if logger == nil {
			logger = slog.Default()
		}

		err = m.executeWithNetworkRetry(func() error {
			return core.MirrorUpdateRepo(repo.URL, repo.Path, m.plan.DirtyStrategy, logger)
		}, &retryCount)
		if err == nil {
			err = core.SaveMirroredRepo(repo.URL, repo.Path)
		}

	case "skip":
		// Already skipped, just record it
		err = nil
	}

	duration := time.Since(start)

	return core.MirrorResult{
		Repo:       repo,
		Success:    err == nil || repo.Action == "skip",
		Error:      err,
		Duration:   duration.Milliseconds(),
		RetryCount: retryCount,
	}
}

// executeWithNetworkRetry wraps an operation with network retry logic
func (m *MirrorModel) executeWithNetworkRetry(op func() error, retryCount *int) error {
	maxRetries := m.plan.NetworkRetries
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
		if !core.IsNetworkError(err) {
			return err // Non-retryable error
		}

		*retryCount++
		lastErr = err

		// Exponential backoff: 1s, 2s, 4s...
		backoff := min(time.Duration(1<<attempt)*time.Second, 30*time.Second)

		time.Sleep(backoff)
	}

	return &core.NetworkError{
		Operation: "git operation",
		Err:       lastErr,
		Attempts:  maxRetries,
	}
}

func (m *MirrorModel) addActivity(result core.MirrorResult) {
	var message string

	switch {
	case result.Repo.Action == "skip":
		message = result.Repo.Reason
	case result.Success:
		message = fmt.Sprintf("%sed in %.1fs", result.Repo.Action, float64(result.Duration)/1000.0)
	default:
		message = result.Error.Error()
	}

	var status string

	switch {
	case result.Repo.Action == "skip":
		status = "skip"
	case !result.Success:
		status = "error"
	default:
		status = "success"
	}

	item := activityItem{
		repo:       result.Repo.Name,
		action:     result.Repo.Action,
		status:     status,
		duration:   time.Duration(result.Duration) * time.Millisecond,
		message:    message,
		retryCount: result.RetryCount,
	}

	m.activity = append(m.activity, item)
}

// Error returns the error if the mirror failed
func (m *MirrorModel) Error() error {
	return m.err
}

// Results returns all mirror results
func (m *MirrorModel) Results() []core.MirrorResult {
	return m.results
}
