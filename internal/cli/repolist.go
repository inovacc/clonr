package cli

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/inovacc/clonr/internal/core"
	"github.com/inovacc/clonr/internal/model"
)

var (
	docStyle = lipgloss.NewStyle().Margin(1, 2)
)

type repoItem struct {
	repo model.Repository
}

func (i repoItem) Title() string {
	fav := ""
	if i.repo.Favorite {
		fav = "⭐ "
	}

	return fmt.Sprintf("%s%s", fav, i.repo.URL)
}

func (i repoItem) Description() string {
	desc := i.repo.Path

	if !i.repo.ClonedAt.IsZero() {
		clonedAt := i.repo.ClonedAt.Format("2006-01-02 15:04")
		desc = fmt.Sprintf("%s | Cloned: %s", desc, clonedAt)

		if !i.repo.UpdatedAt.IsZero() {
			updatedAt := i.repo.UpdatedAt.Format("2006-01-02 15:04")
			desc = fmt.Sprintf("%s | Updated: %s", desc, updatedAt)
		}
	}

	return desc
}

func (i repoItem) FilterValue() string {
	return i.repo.URL
}

type RepoListModel struct {
	list         list.Model
	selectedRepo *model.Repository
	action       string
	err          error
	quitting     bool
}

func (m RepoListModel) Init() tea.Cmd {
	return nil
}

func (m RepoListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			i, ok := m.list.SelectedItem().(repoItem)
			if ok {
				m.selectedRepo = &i.repo
				m.action = "selected"
			}

			return m, tea.Quit
		}
	}

	var cmd tea.Cmd

	m.list, cmd = m.list.Update(msg)

	return m, cmd
}

func (m RepoListModel) View() string {
	if m.quitting {
		return ""
	}

	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	return docStyle.Render(m.list.View())
}

func (m RepoListModel) GetSelectedRepo() *model.Repository {
	return m.selectedRepo
}

func NewRepoList(favoritesOnly bool) (RepoListModel, error) {
	repos, err := core.ListReposFiltered(favoritesOnly)
	if err != nil {
		return RepoListModel{err: err}, err
	}

	items := make([]list.Item, len(repos))
	for i, repo := range repos {
		items[i] = repoItem{repo: repo}
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	if favoritesOnly {
		l.Title = "Favorite Repositories"
	} else {
		l.Title = "All Repositories"
	}

	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)

	return RepoListModel{list: l}, nil
}
