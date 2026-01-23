package cli

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(2)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
)

type menuItem struct {
	title       string
	description string
	action      string
}

func (i menuItem) FilterValue() string { return i.title }

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(menuItem)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s", index+1, i.title)

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + s[0])
		}
	}

	_, _ = fmt.Fprint(w, fn(str))
}

type MainMenuModel struct {
	list         list.Model
	choice       string
	quitting     bool
	selectedItem menuItem
}

func (m MainMenuModel) Init() tea.Cmd {
	return nil
}

func (m MainMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)

		return m, nil

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "ctrl+c", "q":
			m.quitting = true

			return m, tea.Quit

		case "enter":
			i, ok := m.list.SelectedItem().(menuItem)
			if ok {
				m.selectedItem = i
				m.choice = i.action
			}

			return m, tea.Quit
		}
	}

	var cmd tea.Cmd

	m.list, cmd = m.list.Update(msg)

	return m, cmd
}

func (m MainMenuModel) View() string {
	if m.choice != "" {
		return ""
	}

	if m.quitting {
		return "Goodbye!\n"
	}

	return "\n" + m.list.View()
}

func (m MainMenuModel) GetChoice() string {
	return m.choice
}

func NewMainMenu() MainMenuModel {
	items := []list.Item{
		menuItem{title: "Clone Repository", description: "Clone a Git repository", action: "clone"},
		menuItem{title: "List Repositories", description: "List all managed repositories", action: "list"},
		menuItem{title: "List Branches", description: "List and switch branches", action: "branches"},
		menuItem{title: "Add Repository", description: "Add an existing local repository", action: "add"},
		menuItem{title: "Map Repositories", description: "Scan directory for Git repositories", action: "map"},
		menuItem{title: "Favorite Repository", description: "Mark/unmark repository as favorite", action: "favorite"},
		menuItem{title: "Open Repository", description: "Open a favorite repository", action: "open"},
		menuItem{title: "Remove Repository", description: "Remove repository from management", action: "remove"},
		menuItem{title: "Update Repositories", description: "Pull latest changes", action: "update"},
		menuItem{title: "Repository Status", description: "Show git status", action: "status"},
		menuItem{title: "Repository Stats (Nerds)", description: "Show detailed statistics", action: "nerds"},
		menuItem{title: "Configure", description: "Configure clonr settings", action: "configure"},
		menuItem{title: "Start Server", description: "Start API server", action: "server"},
		menuItem{title: "Exit", description: "Exit clonr", action: "exit"},
	}

	const defaultWidth = 20

	l := list.New(items, itemDelegate{}, defaultWidth, 15)
	l.Title = "Clonr - Git Repository Manager"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	return MainMenuModel{list: l}
}
