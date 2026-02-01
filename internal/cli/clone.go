package cli

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/inovacc/clonr/internal/git"
)

var (
	spinnerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	urlStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true)
	pathStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
)

type CloneModel struct {
	spinner spinner.Model
	url     string
	path    string
	cloning bool
	done    bool
	err     error
}

type cloneCompleteMsg struct {
	err error
}

// NewCloneModel creates a new clone model
// Authentication is handled via clonr's credential helper
func NewCloneModel(url, path string) CloneModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = spinnerStyle

	return CloneModel{
		spinner: s,
		url:     url,
		path:    path,
		cloning: true,
	}
}

func (m CloneModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.cloneRepo)
}

func (m CloneModel) cloneRepo() tea.Msg {
	// Use git client with a credential helper for authentication
	client := git.NewClient()

	err := client.Clone(context.Background(), m.url, m.path)
	if err != nil {
		return cloneCompleteMsg{err: err}
	}

	return cloneCompleteMsg{err: nil}
}

func (m CloneModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch keyMsg := msg.(type) {
	case tea.KeyMsg:
		if keyMsg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		if m.done {
			return m, tea.Quit
		}

	case cloneCompleteMsg:
		m.cloning = false
		m.done = true
		m.err = keyMsg.err

		return m, tea.Quit

	case spinner.TickMsg:
		var cmd tea.Cmd

		m.spinner, cmd = m.spinner.Update(keyMsg)

		return m, cmd
	}

	return m, nil
}

func (m CloneModel) View() string {
	if m.done {
		if m.err != nil {
			return errorStyle.Render(fmt.Sprintf("\n  ✗ Clone failed: %v\n\n", m.err))
		}

		return successStyle.Render(fmt.Sprintf("\n  ✓ Successfully cloned to %s\n\n", m.path))
	}

	if m.cloning {
		return fmt.Sprintf("\n  %s Cloning %s\n  %s\n\n", m.spinner.View(), urlStyle.Render(m.url), pathStyle.Render("→ "+m.path))
	}

	return ""
}

func (m CloneModel) Error() error {
	return m.err
}
