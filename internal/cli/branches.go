package cli

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/inovacc/clonr/internal/core"
)

var (
	branchCurrentStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	branchRemoteStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	branchLocalStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
)

type branchItem struct {
	branch core.Branch
}

func (i branchItem) Title() string {
	prefix := "  "
	style := branchLocalStyle

	if i.branch.IsCurrent {
		prefix = "* "
		style = branchCurrentStyle
	} else if i.branch.IsRemote {
		style = branchRemoteStyle
	}

	return style.Render(prefix + i.branch.Name)
}

func (i branchItem) Description() string {
	if i.branch.IsCurrent {
		return "current branch"
	}

	if i.branch.IsRemote {
		return "remote branch"
	}

	return "local branch"
}

func (i branchItem) FilterValue() string {
	return i.branch.Name
}

// BranchListModel is the Bubbletea model for branch listing
type BranchListModel struct {
	list           list.Model
	repoPath       string
	repoURL        string
	selectedBranch *core.Branch
	action         string
	err            error
	quitting       bool
	showHelp       bool
}

func (m BranchListModel) Init() tea.Cmd {
	return nil
}

func (m BranchListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)

		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.quitting = true

			return m, tea.Quit

		case "enter":
			i, ok := m.list.SelectedItem().(branchItem)
			if ok {
				m.selectedBranch = &i.branch
				m.action = "checkout"
			}

			return m, tea.Quit

		case "?":
			m.showHelp = !m.showHelp

			return m, nil
		}
	}

	var cmd tea.Cmd

	m.list, cmd = m.list.Update(msg)

	return m, cmd
}

func (m BranchListModel) View() string {
	if m.quitting {
		return ""
	}

	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	view := docStyle.Render(m.list.View())

	if m.showHelp {
		helpText := lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Render("\n  enter: checkout branch • q/esc: quit • /: filter • ?: toggle help")
		view += helpText
	}

	return view
}

// GetSelectedBranch returns the selected branch
func (m BranchListModel) GetSelectedBranch() *core.Branch {
	return m.selectedBranch
}

// GetAction returns the action to perform
func (m BranchListModel) GetAction() string {
	return m.action
}

// GetRepoPath returns the repository path
func (m BranchListModel) GetRepoPath() string {
	return m.repoPath
}

// NewBranchList creates a new branch list model for the given repository
func NewBranchList(repoPath, repoURL string, showAll bool) (BranchListModel, error) {
	opts := core.BranchListOptions{
		All: showAll,
	}

	branches, err := core.ListBranches(repoPath, opts)
	if err != nil {
		return BranchListModel{err: err}, err
	}

	items := make([]list.Item, len(branches))
	for i, branch := range branches {
		items[i] = branchItem{branch: branch}
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)

	title := "Branches"
	if repoURL != "" {
		title = fmt.Sprintf("Branches - %s", repoURL)
	}

	l.Title = title
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)

	return BranchListModel{
		list:     l,
		repoPath: repoPath,
		repoURL:  repoURL,
	}, nil
}
