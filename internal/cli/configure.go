package cli

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/inovacc/clonr/internal/database"
	"github.com/inovacc/clonr/internal/model"
)

const fmtV1 = " %s\n %s\n\n"

var (
	focusedStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	blurredStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	cursorStyle        = focusedStyle
	noStyle            = lipgloss.NewStyle()
	helpStyleConfigure = blurredStyle

	focusedButton = focusedStyle.Render("[ Submit ]")
	blurredButton = fmt.Sprintf("[ %s ]", blurredStyle.Render("Submit"))
)

type ConfigureModel struct {
	focusIndex int
	inputs     []textinput.Model
	db         database.Store
	Saved      bool
	Err        error
}

func NewConfigureModel() (ConfigureModel, error) {
	db := database.GetDB()

	// Load existing config or defaults
	cfg, err := db.GetConfig()
	if err != nil {
		return ConfigureModel{}, err
	}

	m := ConfigureModel{
		inputs: make([]textinput.Model, 5),
		db:     db,
	}

	var t textinput.Model
	for i := range m.inputs {
		t = textinput.New()
		t.Cursor.Style = cursorStyle
		t.CharLimit = 256

		switch i {
		case 0:
			t.Placeholder = "~/clonr"
			t.SetValue(cfg.DefaultCloneDir)
			t.Focus()
			t.PromptStyle = focusedStyle
			t.TextStyle = focusedStyle
		case 1:
			t.Placeholder = "code, vim, etc."
			t.SetValue(cfg.Editor)
		case 2:
			t.Placeholder = "terminal application (optional)"
			t.SetValue(cfg.Terminal)
		case 3:
			t.Placeholder = "300"
			t.CharLimit = 10
			t.SetValue(strconv.Itoa(cfg.MonitorInterval))
		case 4:
			t.Placeholder = "4000"
			t.CharLimit = 5
			t.SetValue(strconv.Itoa(cfg.ServerPort))
		}

		m.inputs[i] = t
	}

	return m, nil
}

func (m *ConfigureModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *ConfigureModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case successMsg:
		m.Saved = true
		return m, tea.Quit
	case errMsg:
		m.Err = msg.err
		return m, tea.Quit
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit

		case "tab", "shift+tab", "enter", "up", "down":
			s := msg.String()

			// Submit on enter when on submitted button
			if s == "enter" && m.focusIndex == len(m.inputs) {
				return m, m.saveConfig
			}

			// Cycle indexes
			if s == "up" || s == "shift+tab" {
				m.focusIndex--
			} else {
				m.focusIndex++
			}

			if m.focusIndex > len(m.inputs) {
				m.focusIndex = 0
			} else if m.focusIndex < 0 {
				m.focusIndex = len(m.inputs)
			}

			cmds := make([]tea.Cmd, len(m.inputs))
			for i := 0; i <= len(m.inputs)-1; i++ {
				if i == m.focusIndex {
					// Set focused state
					cmds[i] = m.inputs[i].Focus()
					m.inputs[i].PromptStyle = focusedStyle
					m.inputs[i].TextStyle = focusedStyle

					continue
				}
				// Remove the focused state
				m.inputs[i].Blur()
				m.inputs[i].PromptStyle = noStyle
				m.inputs[i].TextStyle = noStyle
			}

			return m, tea.Batch(cmds...)
		}
	}

	// Handle character input and blinking
	cmd := m.updateInputs(msg)

	return m, cmd
}

func (m *ConfigureModel) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))

	// Only text inputs with Focus() set will respond, so it's safe to simply
	// update all of them here without any further logic.
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}

	return tea.Batch(cmds...)
}

func (m *ConfigureModel) View() string {
	if m.Saved {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")).
			Render("\n  ✓ Configuration saved successfully!\n\n")
	}

	if m.Err != nil {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Render(fmt.Sprintf("\n  ✗ Error: %v\n\n", m.Err))
	}

	// Show header and current values info
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205"))

	s := headerStyle.Render("Configure Clonr Settings") + "\n"
	s += blurredStyle.Render("Edit the fields below and press Tab to navigate") + "\n\n"
	s += fmt.Sprintf(fmtV1, blurredStyle.Render("Default Clone Directory:"), m.inputs[0].View())
	s += fmt.Sprintf(fmtV1, blurredStyle.Render("Default Editor:"), m.inputs[1].View())
	s += fmt.Sprintf(fmtV1, blurredStyle.Render("Default Terminal:"), m.inputs[2].View())
	s += fmt.Sprintf(fmtV1, blurredStyle.Render("Monitor Interval (seconds):"), m.inputs[3].View())
	s += fmt.Sprintf(fmtV1, blurredStyle.Render("Server Port:"), m.inputs[4].View())

	button := &blurredButton
	if m.focusIndex == len(m.inputs) {
		button = &focusedButton
	}

	s += fmt.Sprintf("\n\n %s\n\n", *button)
	s += helpStyleConfigure.Render(" tab/shift+tab: navigate • enter: submit • esc: quit")

	return s
}

func (m *ConfigureModel) saveConfig() tea.Msg {
	// Parse monitor interval
	monitorInterval, err := strconv.Atoi(m.inputs[3].Value())
	if err != nil {
		monitorInterval = 300
	}

	// Parse server port
	serverPort, err := strconv.Atoi(m.inputs[4].Value())
	if err != nil {
		serverPort = 4000
	}

	cfg := &model.Config{
		DefaultCloneDir: m.inputs[0].Value(),
		Editor:          m.inputs[1].Value(),
		Terminal:        m.inputs[2].Value(),
		MonitorInterval: monitorInterval,
		ServerPort:      serverPort,
	}

	if err := m.db.SaveConfig(cfg); err != nil {
		return errMsg{err}
	}

	return successMsg{}
}

type successMsg struct{}
type errMsg struct{ err error }
