package cli

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/inovacc/clonr/internal/core"
	"github.com/inovacc/clonr/internal/grpcclient"
	"github.com/inovacc/clonr/internal/model"
)

var (
	workspaceTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("205")).
				MarginBottom(1)

	activeWorkspaceBadge = lipgloss.NewStyle().
				Background(lipgloss.Color("42")).
				Foreground(lipgloss.Color("0")).
				Padding(0, 1).
				MarginLeft(1)

	inactiveWorkspaceBadge = lipgloss.NewStyle().
				Background(lipgloss.Color("240")).
				Foreground(lipgloss.Color("255")).
				Padding(0, 1).
				MarginLeft(1)

	workspaceHelpStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("241")).
				MarginTop(1)
)

// WorkspaceRepoItem wraps a repository for display in workspace view
type WorkspaceRepoItem struct {
	repo model.Repository
}

func (i WorkspaceRepoItem) Title() string {
	fav := ""
	if i.repo.Favorite {
		fav = " *"
	}

	return fmt.Sprintf("%s%s", i.repo.URL, fav)
}

func (i WorkspaceRepoItem) Description() string {
	return i.repo.Path
}

func (i WorkspaceRepoItem) FilterValue() string {
	return i.repo.URL
}

// WorkspaceReposModel is the TUI model for browsing repos by workspace
type WorkspaceReposModel struct {
	workspaces       []model.Workspace
	currentWorkspace int
	repoList         list.Model
	repos            map[string][]model.Repository
	selectedRepo     *model.Repository
	quitting         bool
	err              error
	width            int
	height           int
}

// NewWorkspaceReposModel creates a new workspace-based repo browser
func NewWorkspaceReposModel() (WorkspaceReposModel, error) {
	client, err := grpcclient.GetClient()
	if err != nil {
		return WorkspaceReposModel{err: err}, err
	}

	workspaces, err := client.ListWorkspaces()
	if err != nil {
		return WorkspaceReposModel{err: err}, err
	}

	if len(workspaces) == 0 {
		return WorkspaceReposModel{
			err: fmt.Errorf("no workspaces configured\nCreate one with: clonr workspace add <name> --path <directory>"),
		}, nil
	}

	// Find active workspace index
	activeIdx := 0

	for i, ws := range workspaces {
		if ws.Active {
			activeIdx = i

			break
		}
	}

	// Load all repos and group by workspace
	allRepos, err := core.ListRepos()
	if err != nil {
		return WorkspaceReposModel{err: err}, err
	}

	reposByWorkspace := make(map[string][]model.Repository)

	// Initialize empty slices for all workspaces
	for _, ws := range workspaces {
		reposByWorkspace[ws.Name] = []model.Repository{}
	}

	// Add "unassigned" category for repos without workspace
	reposByWorkspace[""] = []model.Repository{}

	// Group repos by workspace
	for _, repo := range allRepos {
		wsName := repo.Workspace
		reposByWorkspace[wsName] = append(reposByWorkspace[wsName], repo)
	}

	// Create initial list for active workspace
	m := WorkspaceReposModel{
		workspaces:       workspaces,
		currentWorkspace: activeIdx,
		repos:            reposByWorkspace,
	}

	m = m.withUpdatedRepoList()

	return m, nil
}

func (m WorkspaceReposModel) withUpdatedRepoList() WorkspaceReposModel {
	wsName := ""
	if m.currentWorkspace < len(m.workspaces) {
		wsName = m.workspaces[m.currentWorkspace].Name
	}

	repos := m.repos[wsName]
	items := make([]list.Item, len(repos))

	for i, repo := range repos {
		items[i] = WorkspaceRepoItem{repo: repo}
	}

	delegate := list.NewDefaultDelegate()
	l := list.New(items, delegate, 0, 0)
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.SetShowTitle(false)

	if m.width > 0 && m.height > 0 {
		h, v := docStyle.GetFrameSize()
		// Account for header height (workspace tabs + help)
		headerHeight := 4
		l.SetSize(m.width-h, m.height-v-headerHeight)
	}

	m.repoList = l

	return m
}

func (m WorkspaceReposModel) Init() tea.Cmd {
	return nil
}

func (m WorkspaceReposModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m = m.withUpdatedRepoList()

		return m, nil

	case tea.KeyMsg:
		// Don't handle workspace switching when filtering
		if m.repoList.FilterState() == list.Filtering {
			var cmd tea.Cmd

			m.repoList, cmd = m.repoList.Update(msg)

			return m, cmd
		}

		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true

			return m, tea.Quit

		case "esc":
			// If filtering, let list handle it; otherwise quit
			if m.repoList.FilterState() != list.Unfiltered {
				var cmd tea.Cmd

				m.repoList, cmd = m.repoList.Update(msg)

				return m, cmd
			}

			m.quitting = true

			return m, tea.Quit

		case "tab", "right", "l":
			// Next workspace
			if len(m.workspaces) > 0 {
				m.currentWorkspace = (m.currentWorkspace + 1) % len(m.workspaces)
				m = m.withUpdatedRepoList()
			}

			return m, nil

		case "shift+tab", "left", "h":
			// Previous workspace
			if len(m.workspaces) > 0 {
				m.currentWorkspace--
				if m.currentWorkspace < 0 {
					m.currentWorkspace = len(m.workspaces) - 1
				}

				m = m.withUpdatedRepoList()
			}

			return m, nil

		case "enter":
			i, ok := m.repoList.SelectedItem().(WorkspaceRepoItem)
			if ok {
				m.selectedRepo = &i.repo
			}

			return m, tea.Quit

		case "s":
			// Set current workspace as active
			if len(m.workspaces) > 0 {
				client, err := grpcclient.GetClient()
				if err == nil {
					wsName := m.workspaces[m.currentWorkspace].Name

					if err := client.SetActiveWorkspace(wsName); err == nil {
						// Update local state
						for i := range m.workspaces {
							m.workspaces[i].Active = (i == m.currentWorkspace)
						}
					}
				}
			}

			return m, nil
		}
	}

	var cmd tea.Cmd

	m.repoList, cmd = m.repoList.Update(msg)

	return m, cmd
}

func (m WorkspaceReposModel) View() string {
	if m.quitting {
		return ""
	}

	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	// Build workspace tabs
	var tabs strings.Builder

	for i, ws := range m.workspaces {
		repoCount := len(m.repos[ws.Name])
		label := fmt.Sprintf("%s (%d)", ws.Name, repoCount)

		if i == m.currentWorkspace {
			tabs.WriteString(activeWorkspaceBadge.Render(label))
		} else {
			tabs.WriteString(inactiveWorkspaceBadge.Render(label))
		}
	}

	// Show unassigned count if any
	if unassigned := m.repos[""]; len(unassigned) > 0 {
		label := fmt.Sprintf("unassigned (%d)", len(unassigned))
		tabs.WriteString(inactiveWorkspaceBadge.Render(label))
	}

	// Current workspace info
	currentWs := m.workspaces[m.currentWorkspace]
	title := workspaceTitleStyle.Render(fmt.Sprintf("Workspace: %s", currentWs.Name))

	activeMarker := ""
	if currentWs.Active {
		activeMarker = " (active)"
	}

	pathInfo := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render(fmt.Sprintf("Path: %s%s", currentWs.Path, activeMarker))

	// Help text
	help := workspaceHelpStyle.Render("tab/←→: switch workspace • s: set active • enter: select • /: filter • q: quit")

	// Combine all parts
	header := fmt.Sprintf("%s\n%s\n%s", tabs.String(), title, pathInfo)

	return docStyle.Render(fmt.Sprintf("%s\n%s\n%s", header, m.repoList.View(), help))
}

// GetSelectedRepo returns the selected repository
func (m WorkspaceReposModel) GetSelectedRepo() *model.Repository {
	return m.selectedRepo
}

// GetCurrentWorkspace returns the currently viewed workspace
func (m WorkspaceReposModel) GetCurrentWorkspace() *model.Workspace {
	if m.currentWorkspace < len(m.workspaces) {
		return &m.workspaces[m.currentWorkspace]
	}

	return nil
}
