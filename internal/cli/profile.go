package cli

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/inovacc/clonr/internal/grpcclient"
	"github.com/inovacc/clonr/internal/model"
)

var (
	profileNameStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("205")).
				Bold(true)

	profileHostStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240"))

	profileActiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("42"))

	profileWorkspaceStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("33"))
)

// ProfileItem implements list.Item for profile selection
type ProfileItem struct {
	profile model.Profile
}

func (i ProfileItem) Title() string {
	active := ""
	if i.profile.Active {
		active = profileActiveStyle.Render(" (active)")
	}

	return profileNameStyle.Render(i.profile.Name) + active
}

func (i ProfileItem) Description() string {
	workspace := ""
	if i.profile.Workspace != "" {
		workspace = profileWorkspaceStyle.Render(fmt.Sprintf(" → %s", i.profile.Workspace))
	}

	return fmt.Sprintf("%s@%s%s",
		profileHostStyle.Render(i.profile.User),
		profileHostStyle.Render(i.profile.Host),
		workspace,
	)
}

func (i ProfileItem) FilterValue() string {
	return i.profile.Name
}

// ProfileSelectorModel is the TUI model for profile selection
type ProfileSelectorModel struct {
	list     list.Model
	selected *model.Profile
	quitting bool
	err      error
}

// NewProfileSelector creates a new profile selector TUI
func NewProfileSelector() (ProfileSelectorModel, error) {
	client, err := grpcclient.GetClient()
	if err != nil {
		return ProfileSelectorModel{err: err}, err
	}

	profiles, err := client.ListProfiles()
	if err != nil {
		return ProfileSelectorModel{err: err}, err
	}

	if len(profiles) == 0 {
		return ProfileSelectorModel{
			err: fmt.Errorf("no profiles configured\nCreate one with: clonr profile add <name> --workspace <workspace>"),
		}, nil
	}

	items := make([]list.Item, len(profiles))
	for i, p := range profiles {
		items[i] = ProfileItem{profile: p}
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Select Profile"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)

	return ProfileSelectorModel{
		list: l,
	}, nil
}

func (m ProfileSelectorModel) Init() tea.Cmd {
	return nil
}

func (m ProfileSelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			i, ok := m.list.SelectedItem().(ProfileItem)
			if ok {
				m.selected = &i.profile

				return m, tea.Quit
			}
		}
	}

	var cmd tea.Cmd

	m.list, cmd = m.list.Update(msg)

	return m, cmd
}

func (m ProfileSelectorModel) View() string {
	if m.quitting {
		return ""
	}

	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	return docStyle.Render(m.list.View())
}

// GetSelected returns the selected profile, or nil if none was selected
func (m ProfileSelectorModel) GetSelected() *model.Profile {
	return m.selected
}

// HasProfiles returns true if there are profiles to select from
func (m ProfileSelectorModel) HasProfiles() bool {
	return m.err == nil && len(m.list.Items()) > 0
}
