package cmd

//
// import (
// 	"fmt"
// 	"log"
// 	"os"
// 	"path/filepath"
//
// 	"github.com/charmbracelet/bubbles/list"
// 	"github.com/charmbracelet/bubbles/textinput"
// 	tea "github.com/charmbracelet/bubbletea"
// 	"github.com/charmbracelet/huh"
// 	"github.com/charmbracelet/lipgloss"
// 	"github.com/dyammarcano/clonr/internal/core"
// 	"github.com/spf13/cobra"
// )
//
// // A MenuItem holds the text and its corresponding submenu choices.
// type MenuItem struct {
// 	Text   string
// 	Action string
// }
//
// // Menu structure mapping main menu options to their submenu items.
// var menuItems = []MenuItem{
// 	{"List Repositories", "list"},
// 	{"Add Repository", "add"},
// 	{"Map Repositories", "map"},
// 	{"Remove Repositories", "remove"},
// 	{"Initialize", "init"},
// }
//
// // AppState State represents the current view of the application.
// type AppState int
//
// const (
// 	MainMenu AppState = iota
// 	ListingRepos
// 	AddingRepo
// 	MappingRepos
// 	RemovingRepos
// 	Initializing
// )
//
// // AppModel Main application model
// type AppModel struct {
// 	state     AppState
// 	menuList  list.Model
// 	textInput textinput.Model
// 	form      *huh.Form
// 	err       error
// }
//
// // NewMenuModel creates an initial model for the main menu.
// func NewMenuModel() AppModel {
// 	items := make([]list.Item, len(menuItems))
// 	for i, item := range menuItems {
// 		items[i] = list.Item(textinput.New()) // Using textinput as a generic item placeholder
// 	}
//
// 	// Initialize the list model for the main menu
// 	mList := list.New(items, list.NewDefaultDelegate(), 0, 0)
// 	mList.Title = "Clonr - A Git Wrapper"
// 	mList.SetShowStatusBar(false)
//
// 	// Set up the menu items
// 	menuItemsAsListItems := make([]list.Item, len(menuItems))
// 	for i, mi := range menuItems {
// 		menuItemsAsListItems[i] = list.Item(menuListItem(mi.Text))
// 	}
// 	mList.SetItems(menuItemsAsListItems)
//
// 	return AppModel{
// 		state:     MainMenu,
// 		menuList:  mList,
// 		textInput: textinput.New(), // Placeholder for the 'add' view
// 	}
// }
//
// // Custom list item to display menu options
// type menuListItem string
//
// func (i menuListItem) Title() string       { return string(i) }
// func (i menuListItem) Description() string { return "" }
// func (i menuListItem) FilterValue() string { return string(i) }
//
// // Init initializes the main model.
// func (m AppModel) Init() tea.Cmd {
// 	return nil
// }
//
// // Update handles key presses and updates the model's state.
// func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
// 	var cmd tea.Cmd
// 	switch msg := msg.(type) {
// 	case tea.WindowSizeMsg:
// 		h, v := lipgloss.NewStyle().GetFrameSize()
// 		m.menuList.SetSize(msg.Width-h, msg.Height-v)
// 		// You'll need to update sub-model sizes here as well
//
// 	case tea.KeyMsg:
// 		switch msg.String() {
// 		case "ctrl+c", "q":
// 			return m, tea.Quit
// 		case "enter":
// 			if m.state == MainMenu {
// 				selectedItem, ok := m.menuList.SelectedItem().(menuListItem)
// 				if !ok {
// 					return m, nil
// 				}
// 				switch selectedItem {
// 				case "List Repositories":
// 					m.state = ListingRepos
// 					repos, err := core.ListRepos()
// 					if err != nil {
// 						m.err = err
// 					}
// 					// This is where you would build your list model for the list view
// 					// ...
// 					return m, nil
// 				case "Add Repository":
// 					m.state = AddingRepo
// 					m.textInput.Placeholder = "Enter directory path..."
// 					m.textInput.Focus()
// 					return m, textinput.Blink
// 				case "Remove Repositories":
// 					// Transition to the remove view
// 					m.state = RemovingRepos
// 					return m, nil
// 				case "Initialize":
// 					// Create a form using huh for initialization
// 					var dir, editor, term string
// 					m.form = huh.NewForm(
// 						huh.NewGroup(
// 							huh.NewInput().
// 								Key("dir").
// 								Title("Default Directory").
// 								Value(&dir),
// 							huh.NewInput().
// 								Key("editor").
// 								Title("Default Editor").
// 								Value(&editor),
// 							huh.NewInput().
// 								Key("terminal").
// 								Title("Default Terminal").
// 								Value(&term),
// 						),
// 					).WithTheme(huh.ThemeDracula())
// 					return m, m.form.Init()
// 				}
// 			} else if m.state == AddingRepo {
// 				path := m.textInput.Value()
// 				fullPath, err := getFullPath(path)
// 				if err != nil {
// 					// Handle error, maybe show it in a status bar
// 					m.err = err
// 					return m, nil
// 				}
// 				log.Printf("Adding directory: %s", fullPath)
// 				m.state = MainMenu // Return to main menu
// 				return m, nil
// 			}
// 		case "esc":
// 			if m.state != MainMenu {
// 				m.state = MainMenu
// 				m.err = nil // Clear any errors
// 				m.textInput.Reset()
// 				return m, nil
// 			}
// 		}
// 	}
//
// 	// Delegate updates to the active component
// 	switch m.state {
// 	case MainMenu:
// 		m.menuList, cmd = m.menuList.Update(msg)
// 	case AddingRepo:
// 		m.textInput, cmd = m.textInput.Update(msg)
// 	case Initializing:
// 		var formCmd tea.Cmd
// 		m.form, formCmd = m.form.Update(msg)
// 		if m.form.State == huh.StateCompleted {
// 			// Handle form submission
// 			log.Printf("Default Directory: %s", m.form.GetString("dir"))
// 			log.Printf("Default Editor: %s", m.form.GetString("editor"))
// 			log.Printf("Default Terminal: %s", m.form.GetString("terminal"))
// 			m.state = MainMenu
// 			return m, nil
// 		}
// 		cmd = formCmd
// 		// ... add cases for other views
// 	}
//
// 	return m, cmd
// }
//
// // View renders the CLI's interface.
// func (m AppModel) View() string {
// 	if m.err != nil {
// 		return fmt.Sprintf("Error: %v\n", m.err)
// 	}
//
// 	switch m.state {
// 	case MainMenu:
// 		return m.menuList.View()
// 	case AddingRepo:
// 		return lipgloss.JoinVertical(
// 			lipgloss.Center,
// 			"Enter path to add:",
// 			m.textInput.View(),
// 		)
// 	case Initializing:
// 		return m.form.View()
// 	// ... add cases for other views
// 	case ListingRepos:
// 		// You would render the list of repos here.
// 		return "Listing repos..."
// 	case RemovingRepos:
// 		// You would render the remove menu here.
// 		return "Removing repos..."
// 	default:
// 		return ""
// 	}
// }
//
// // getFullPath resolves a given path to its absolute path.
// func getFullPath(path string) (string, error) {
// 	absPath, err := filepath.Abs(path)
// 	if err != nil {
// 		return "", fmt.Errorf("could not get absolute path for %s: %w", path, err)
// 	}
//
// 	fileInfo, err := os.Stat(absPath)
// 	if err != nil {
// 		return "", fmt.Errorf("path does not exist or is not a directory: %w", err)
// 	}
//
// 	if !fileInfo.IsDir() {
// 		return "", fmt.Errorf("path is not a directory: %s", absPath)
// 	}
//
// 	return absPath, nil
// }
