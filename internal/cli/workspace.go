package cli

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/inovacc/clonr/internal/client/grpc"
	"github.com/inovacc/clonr/internal/model"
)

var (
	workspaceNameStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("205")).
				Bold(true)

	workspacePathStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240"))

	workspaceActiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("42"))
)

// WorkspaceItem implements list.Item for workspace selection
type WorkspaceItem struct {
	workspace model.Workspace
	isNew     bool
}

func (i WorkspaceItem) Title() string {
	if i.isNew {
		return "+ Create new workspace..."
	}

	active := ""
	if i.workspace.Active {
		active = workspaceActiveStyle.Render(" (active)")
	}

	return workspaceNameStyle.Render(i.workspace.Name) + active
}

func (i WorkspaceItem) Description() string {
	if i.isNew {
		return "Create a new workspace for organizing repositories"
	}

	return workspacePathStyle.Render(i.workspace.Path)
}

func (i WorkspaceItem) FilterValue() string {
	if i.isNew {
		return "create new"
	}

	return i.workspace.Name
}

// WorkspaceSelectorModel is the TUI model for workspace selection
type WorkspaceSelectorModel struct {
	list            list.Model
	selected        *model.Workspace
	creating        bool
	nameInput       textinput.Model
	pathInput       textinput.Model
	focusIndex      int
	quitting        bool
	err             error
	allowCreate     bool
	returnNewOnQuit bool
}

// NewWorkspaceSelector creates a new workspace selector TUI
func NewWorkspaceSelector(allowCreate bool) (WorkspaceSelectorModel, error) {
	client, err := grpc.GetClient()
	if err != nil {
		return WorkspaceSelectorModel{err: err}, err
	}

	workspaces, err := client.ListWorkspaces()
	if err != nil {
		return WorkspaceSelectorModel{err: err}, err
	}

	items := make([]list.Item, 0, len(workspaces)+1)
	for _, w := range workspaces {
		items = append(items, WorkspaceItem{workspace: w})
	}

	// Add the "Create new workspace" option if allowed
	if allowCreate {
		items = append(items, WorkspaceItem{isNew: true})
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Select Workspace"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)

	// Create text inputs for a new workspace
	nameInput := textinput.New()
	nameInput.Placeholder = "workspace-name"
	nameInput.Focus()
	nameInput.CharLimit = 50
	nameInput.Width = 40

	pathInput := textinput.New()
	pathInput.Placeholder = "~/clonr/workspace-name"
	pathInput.CharLimit = 200
	pathInput.Width = 60

	return WorkspaceSelectorModel{
		list:        l,
		nameInput:   nameInput,
		pathInput:   pathInput,
		allowCreate: allowCreate,
	}, nil
}

// NewWorkspaceSelectorForClone creates a workspace selector that returns the
// active workspace if the user quits without selecting
func NewWorkspaceSelectorForClone() (WorkspaceSelectorModel, error) {
	m, err := NewWorkspaceSelector(true)
	if err != nil {
		return m, err
	}

	m.returnNewOnQuit = false

	return m, nil
}

func (m WorkspaceSelectorModel) Init() tea.Cmd {
	return nil
}

func (m WorkspaceSelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle creation mode separately
	if m.creating {
		return m.updateCreating(msg)
	}

	switch keyMsg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(keyMsg.Width-h, keyMsg.Height-v)

		return m, nil

	case tea.KeyMsg:
		switch keyMsg.String() {
		case "ctrl+c", "q", "esc":
			m.quitting = true

			return m, tea.Quit

		case "enter":
			i, ok := m.list.SelectedItem().(WorkspaceItem)
			if ok {
				if i.isNew {
					// Switch to creation mode
					m.creating = true
					m.focusIndex = 0
					m.nameInput.Focus()

					return m, textinput.Blink
				}

				m.selected = &i.workspace

				return m, tea.Quit
			}
		}
	}

	var cmd tea.Cmd

	m.list, cmd = m.list.Update(msg)

	return m, cmd
}

func (m WorkspaceSelectorModel) updateCreating(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		var cmd tea.Cmd

		if m.focusIndex == 0 {
			m.nameInput, cmd = m.nameInput.Update(msg)
		} else {
			m.pathInput, cmd = m.pathInput.Update(msg)
		}

		return m, cmd
	}

	switch keyMsg.String() {
	case "ctrl+c", "esc":
		// Go back to the list
		m.creating = false
		m.nameInput.Reset()
		m.pathInput.Reset()

		return m, nil

	case "tab", "shift+tab":
		// Toggle focus
		if m.focusIndex == 0 {
			m.focusIndex = 1
			m.nameInput.Blur()
			m.pathInput.Focus()
		} else {
			m.focusIndex = 0
			m.pathInput.Blur()
			m.nameInput.Focus()
		}

		return m, textinput.Blink

	case "enter":
		if m.focusIndex == 0 && m.nameInput.Value() != "" {
			// Move to path input
			m.focusIndex = 1
			m.nameInput.Blur()
			m.pathInput.Focus()

			// Suggest a path based on name
			if m.pathInput.Value() == "" {
				m.pathInput.SetValue(fmt.Sprintf("~/clonr/%s", m.nameInput.Value()))
			}

			return m, textinput.Blink
		}

		if m.focusIndex == 1 && m.nameInput.Value() != "" && m.pathInput.Value() != "" {
			// Create the workspace
			m.selected = &model.Workspace{
				Name: m.nameInput.Value(),
				Path: m.pathInput.Value(),
			}

			return m, tea.Quit
		}
	}

	var cmd tea.Cmd

	if m.focusIndex == 0 {
		m.nameInput, cmd = m.nameInput.Update(msg)
	} else {
		m.pathInput, cmd = m.pathInput.Update(msg)
	}

	return m, cmd
}

func (m WorkspaceSelectorModel) View() string {
	if m.quitting {
		return ""
	}

	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	if m.creating {
		return m.viewCreating()
	}

	return docStyle.Render(m.list.View())
}

func (m WorkspaceSelectorModel) viewCreating() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		Render("Create New Workspace")

	instructions := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render("Press Tab to switch fields, Enter to submit, Esc to cancel")

	nameLabel := "Name:"
	pathLabel := "Path:"

	if m.focusIndex == 0 {
		nameLabel = lipgloss.NewStyle().Bold(true).Render("Name:")
	} else {
		pathLabel = lipgloss.NewStyle().Bold(true).Render("Path:")
	}

	return docStyle.Render(fmt.Sprintf(
		"%s\n\n%s\n\n%s\n%s\n\n%s\n%s",
		title,
		instructions,
		nameLabel,
		m.nameInput.View(),
		pathLabel,
		m.pathInput.View(),
	))
}

// GetSelected returns the selected workspace, or nil if none was selected
func (m WorkspaceSelectorModel) GetSelected() *model.Workspace {
	return m.selected
}

// IsNewWorkspace returns true if a new workspace should be created
func (m WorkspaceSelectorModel) IsNewWorkspace() bool {
	return m.selected != nil && m.creating
}
