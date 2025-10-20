package cmd

// import (
// 	"fmt"
// 	"log"
//
// 	"github.com/charmbracelet/bubbles/list"
// 	"github.com/charmbracelet/bubbles/spinner"
// 	"github.com/charmbracelet/bubbles/textinput"
// 	tea "github.com/charmbracelet/bubbletea"
// 	"github.com/charmbracelet/lipgloss"
// 	"github.com/dyammarcano/clonr/internal/core"
// 	"github.com/spf13/cobra"
// )
//
// var removeCmd = &cobra.Command{
// 	Use:   "remove",
// 	Short: "Remove repositories from the registry (does not delete files)",
// 	Long: `Interactively select one or more repositories to remove from the clonr registry. This does not delete any
// files from disk, only removes the selected repositories from clonr's management.`,
// 	RunE: func(cmd *cobra.Command, args []string) error {
// 		p := tea.NewProgram(initialModel())
// 		if _, err := p.Run(); err != nil {
// 			return fmt.Errorf("error running program: %w", err)
// 		}
// 		return nil
// 	},
// }
//
// // Styles
// var (
// 	appStyle   = lipgloss.NewStyle().Margin(1, 2)
// 	titleStyle = lipgloss.NewStyle().
// 			Foreground(lipgloss.Color("#FFF87F")).
// 			Background(lipgloss.Color("#565656")).
// 			Padding(0, 1)
//
// 	// A simple spinner
// 	spinStyle = lipgloss.NewStyle().
// 			Foreground(lipgloss.Color("#F88000"))
// )
//
// // The main application model
// type model struct {
// 	list     list.Model
// 	removing bool
// 	err      error
// }
//
// // Custom item for the list
// type item struct {
// 	title, desc string
// }
//
// func (i item) Title() string       { return i.title }
// func (i item) Description() string { return i.desc }
// func (i item) FilterValue() string { return i.title }
//
// // Initial state of the model
// func initialModel() model {
// 	// Get the list of repos from the core logic
// 	repos, err := core.ListRepos()
// 	if err != nil {
// 		return model{err: err}
// 	}
//
// 	var items []list.Item
// 	for _, repo := range repos {
// 		items = append(items, item{
// 			title: repo.URL,
// 			desc:  repo.Path,
// 		})
// 	}
//
// 	// Initialize the list model
// 	mList := list.New(items, list.NewDefaultDelegate(), 0, 0)
// 	mList.Title = "Selecciona los repositorios a eliminar del registro"
// 	mList.SetShowStatusBar(true)
// 	mList.SetFilteringEnabled(true)
// 	mList.SetShowFilter(true)
// 	mList.SetSize(60, len(items)+5) // Adjust size based on content
//
// 	return model{
// 		list: mList,
// 	}
// }
//
// // Update The main update loop
// func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
// 	var cmd tea.Cmd
// 	switch msg := msg.(type) {
// 	case tea.WindowSizeMsg:
// 		h, v := appStyle.GetFrameSize()
// 		m.list.SetSize(msg.Width-h, msg.Height-v)
//
// 	case tea.KeyMsg:
// 		switch msg.String() {
// 		case "ctrl+c", "q":
// 			return m, tea.Quit
// 		case "enter":
// 			// Get selected items and start removal process
// 			if m.removing {
// 				return m, nil
// 			}
// 			m.removing = true
// 			return m, m.removeSelected
// 		}
// 	}
//
// 	// Pass messages to the list component
// 	m.list, cmd = m.list.Update(msg)
// 	return m, cmd
// }
//
// // Function to handle removal of selected items
// func (m model) removeSelected() tea.Msg {
// 	selectedItems := m.list.SelectedItems()
// 	for _, sel := range selectedItems {
// 		if i, ok := sel.(item); ok {
// 			if err := core.RemoveRepo(i.title); err != nil {
// 				log.Printf("Error removing %s: %v\n", i.title, err)
// 			} else {
// 				log.Printf("Repositorio removido: %s\n", i.title)
// 			}
// 		}
// 	}
// 	return tea.Quit
// }
//
// // View The main view function
// func (m model) View() string {
// 	if m.err != nil {
// 		return fmt.Sprintf("Error: %v\n", m.err)
// 	}
//
// 	if m.removing {
// 		return "Removiendo repositorios..."
// 	}
//
// 	return appStyle.Render(m.list.View())
// }
//
// func init() {
// 	rootCmd.AddCommand(removeCmd)
// }
