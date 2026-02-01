package cli

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/inovacc/clonr/internal/core"
)

// EditorItem represents an editor option for selection.
type EditorItem struct {
	Name    string // Display name: "VS Code"
	Command string // Executable: "code"
	Icon    string // Optional icon
}

func (e EditorItem) Title() string {
	if e.Icon != "" {
		return fmt.Sprintf("%s %s", e.Icon, e.Name)
	}

	return e.Name
}

func (e EditorItem) Description() string {
	return e.Command
}

func (e EditorItem) FilterValue() string {
	return e.Name
}

// EditorListModel is a Bubbletea model for selecting an editor.
type EditorListModel struct {
	list           list.Model
	selectedEditor *EditorItem
	quitting       bool
	err            error
}

// Init initializes the model.
func (m EditorListModel) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model.
func (m EditorListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			i, ok := m.list.SelectedItem().(EditorItem)
			if ok {
				m.selectedEditor = &i
			}

			return m, tea.Quit
		}
	}

	var cmd tea.Cmd

	m.list, cmd = m.list.Update(msg)

	return m, cmd
}

// View renders the model.
func (m EditorListModel) View() string {
	if m.quitting {
		return ""
	}

	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	return docStyle.Render(m.list.View())
}

// GetSelectedEditor returns the selected editor or nil if none was selected.
func (m EditorListModel) GetSelectedEditor() *EditorItem {
	return m.selectedEditor
}

// NewEditorList creates a new editor selection model with only installed editors.
// This includes both default editors and custom editors from the configuration.
func NewEditorList() (EditorListModel, error) {
	installedEditors, err := core.GetInstalledEditors()
	if err != nil {
		// Fallback to default editors only if we can't get config
		installedEditors = getDefaultInstalledEditors()
	}

	if len(installedEditors) == 0 {
		return EditorListModel{}, fmt.Errorf("no editors found installed on this system")
	}

	items := make([]list.Item, len(installedEditors))
	for i, editor := range installedEditors {
		items[i] = EditorItem{
			Name:    editor.Name,
			Command: editor.Command,
			Icon:    editor.Icon,
		}
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Select Editor"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)

	return EditorListModel{list: l}, nil
}

// getDefaultInstalledEditors returns only the default editors that are installed.
// Used as a fallback when the server is not available.
func getDefaultInstalledEditors() []core.EditorInfo {
	var available []core.EditorInfo

	for _, editor := range core.DefaultEditors {
		if core.IsEditorInstalled(editor.Command) {
			available = append(available, editor)
		}
	}

	return available
}
