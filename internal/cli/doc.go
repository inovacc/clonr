// Package cli provides the terminal user interface components for Clonr.
//
// The package uses [Bubbletea] for building interactive terminal UIs and
// [Lipgloss] for styling. All UI components follow the standard Bubbletea
// Model-View-Update (MVU) architecture.
//
// # Components
//
// The package provides several UI components:
//   - Menu: Main interactive menu for selecting operations
//   - RepoList: Filterable list of repositories with actions
//   - Clone: Progress display for git clone operations
//   - Configure: Configuration wizard with form navigation
//
// # Creating New Components
//
// To create a new Bubbletea component:
//
//  1. Define a model struct containing component state
//  2. Implement Init() tea.Cmd for initialization
//  3. Implement Update(tea.Msg) (tea.Model, tea.Cmd) for state updates
//  4. Implement View() string for rendering
//
// Example:
//
//	type myModel struct {
//	    items []string
//	    cursor int
//	}
//
//	func (m myModel) Init() tea.Cmd { return nil }
//	func (m myModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) { ... }
//	func (m myModel) View() string { ... }
//
// # Styling
//
// Use Lipgloss for consistent styling across components. Common styles
// are defined as package-level variables for reuse.
//
// [Bubbletea]: https://github.com/charmbracelet/bubbletea
// [Lipgloss]: https://github.com/charmbracelet/lipgloss
package cli
