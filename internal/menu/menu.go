package menu

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func NewMenuModel(args []string) error {
	return nil
}

// A MenuItem holds the text and its corresponding submenu choices.
type MenuItem struct {
	Text    string
	Choices []string
}

// Menu structure mapping main menu options to their submenu items.
var menuItems = map[string]MenuItem{
	"list":    {"List", []string{"ls", "ls -a", "tree"}},
	"nerds":   {"Nerd Fonts", []string{"nerd-font", "powerline", "zsh"}},
	"monitor": {"Monitor", []string{"htop", "glances", "gtop"}},
	"move":    {"Move", []string{"cd ..", "cd /home", "cd /var"}},
	"map":     {"Map", []string{"map-a", "map-b"}},
	"remove":  {"Remove", []string{"rm -rf", "delete"}},
}

// Model represents the state of the CLI application.
type Model struct {
	mainMenuChoices []string
	currentChoices  []string
	cursor          int
	isSubMenu       bool
	selectedMenu    string
}

// NewModel creates an initial model for the main menu.
func NewModel() Model {
	mainChoices := make([]string, 0, len(menuItems))
	for key := range menuItems {
		mainChoices = append(mainChoices, key)
	}

	return Model{
		mainMenuChoices: mainChoices,
		currentChoices:  mainChoices,
	}
}

// Init initializes the model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles key presses and updates the model's state.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.currentChoices)-1 {
				m.cursor++
			}
		case "enter":
			if m.isSubMenu {
				_, _ = fmt.Fprintf(os.Stdout, "You selected: %s from %s\n", m.currentChoices[m.cursor], m.selectedMenu)

				return m, tea.Quit
			}

			m.selectedMenu = m.currentChoices[m.cursor]

			item, ok := menuItems[m.selectedMenu]
			if !ok {
				return m, tea.Quit
			}

			m.currentChoices = item.Choices
			m.isSubMenu = true
			m.cursor = 0
		case "esc":
			if m.isSubMenu {
				m.isSubMenu = false
				m.currentChoices = m.mainMenuChoices
				m.cursor = 0
			}
		}
	}

	return m, nil
}

// View renders the CLI's interface.
func (m Model) View() string {
	var s string

	if m.isSubMenu {
		s = fmt.Sprintf("Submenu: %s\n\n", menuItems[m.selectedMenu].Text)
	} else {
		s = "Select an option:\n\n"
	}

	for i, choice := range m.currentChoices {
		cursor := " "
		if i == m.cursor {
			cursor = ">"
		}

		s += fmt.Sprintf("%s [%s]\n", cursor, choice)
	}

	s += "\nPress 'q' to quit or 'esc' to go back.\n"

	return s
}
