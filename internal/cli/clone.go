package cli

import (
	"fmt"
	"os/exec"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	spinnerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	urlStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true)
	pathStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
)

type CloneModel struct {
	spinner  spinner.Model
	url      string
	path     string
	token    string
	cloning  bool
	done     bool
	err      error
}

type cloneCompleteMsg struct {
	err error
}

func NewCloneModel(url, path, token string) CloneModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = spinnerStyle

	return CloneModel{
		spinner: s,
		url:     url,
		path:    path,
		token:   token,
		cloning: true,
	}
}

func (m CloneModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.cloneRepo)
}

func (m CloneModel) cloneRepo() tea.Msg {
	cloneURL := m.url

	// Inject token into URL for authentication if provided
	if m.token != "" {
		cloneURL = injectTokenIntoURL(m.url, m.token)
	}

	cmd := exec.Command("git", "clone", cloneURL, m.path)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return cloneCompleteMsg{err: fmt.Errorf("git clone failed: %v - %s", err, string(output))}
	}

	return cloneCompleteMsg{err: nil}
}

// injectTokenIntoURL adds authentication token to HTTPS URLs
func injectTokenIntoURL(rawURL, token string) string {
	var scheme, rest string

	if len(rawURL) > 8 && rawURL[:8] == "https://" {
		scheme = "https://"
		rest = rawURL[8:]
	} else if len(rawURL) > 7 && rawURL[:7] == "http://" {
		scheme = "http://"
		rest = rawURL[7:]
	} else {
		return rawURL
	}

	// Remove existing credentials if present (user:pass@host or user@host)
	for i := 0; i < len(rest); i++ {
		if rest[i] == '@' {
			// Check if @ appears before the first /
			slashFound := false
			for j := 0; j < i; j++ {
				if rest[j] == '/' {
					slashFound = true
					break
				}
			}
			if !slashFound {
				rest = rest[i+1:]
				break
			}
		}
		if rest[i] == '/' {
			break
		}
	}

	return scheme + token + "@" + rest
}

func (m CloneModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		if m.done {
			return m, tea.Quit
		}

	case cloneCompleteMsg:
		m.cloning = false
		m.done = true
		m.err = msg.err

		return m, tea.Quit

	case spinner.TickMsg:
		var cmd tea.Cmd

		m.spinner, cmd = m.spinner.Update(msg)

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
