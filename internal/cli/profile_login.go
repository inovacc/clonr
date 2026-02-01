package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/inovacc/clonr/internal/core"
	"github.com/inovacc/clonr/internal/crypto/tpm"
	"github.com/inovacc/clonr/internal/model"
)

// ProfileLoginModel is the Bubbletea model for OAuth login
type ProfileLoginModel struct {
	profileName     string
	host            string
	scopes          []string
	spinner         spinner.Model
	deviceCode      string
	verificationURL string
	state           loginState
	profile         *model.Profile
	err             error
}

type loginState int

const (
	stateInitializing loginState = iota
	stateWaitingForAuth
	stateComplete
	stateError
)

// OAuth result message
type oauthResultMsg struct {
	profile *model.Profile
	err     error
}

// Device code message
type deviceCodeMsg struct {
	code string
	url  string
}

// NewProfileLoginModel creates a new profile login model
func NewProfileLoginModel(name, host string, scopes []string) *ProfileLoginModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	if host == "" {
		host = model.DefaultHost()
	}

	if len(scopes) == 0 {
		scopes = model.DefaultScopes()
	}

	return &ProfileLoginModel{
		profileName: name,
		host:        host,
		scopes:      scopes,
		spinner:     s,
		state:       stateInitializing,
	}
}

// Init initializes the model
func (m *ProfileLoginModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.startOAuth,
	)
}

// startOAuth initiates the OAuth flow
func (m *ProfileLoginModel) startOAuth() tea.Msg {
	pm, err := core.NewProfileManager()
	if err != nil {
		return oauthResultMsg{err: err}
	}

	ctx := context.Background()

	// Run OAuth flow with device code callback
	flow := core.NewOAuthFlow(m.host, m.scopes)

	// Channel to receive device code
	deviceCodeCh := make(chan deviceCodeMsg, 1)

	flow.OnDeviceCode(func(code, url string) {
		deviceCodeCh <- deviceCodeMsg{code: code, url: url}
	})

	// Run OAuth in goroutine
	go func() {
		result, err := flow.Run(ctx)
		if err != nil {
			// Send error through device code channel with empty values to signal error
		} else {
			// Store the token
			var tokenStorage model.TokenStorage

			var encryptedToken []byte

			if keyErr := core.SetToken(m.profileName, m.host, result.Token); keyErr != nil {
				// Keyring not available, use encrypted storage
				encryptedToken, err = tpm.EncryptToken(result.Token, m.profileName, m.host)
				if err != nil {
					return
				}

				tokenStorage = model.TokenStorageInsecure
			} else {
				tokenStorage = model.TokenStorageKeyring
			}

			// Get client to check if first profile
			client, clientErr := pm.ListProfiles() //nolint:contextcheck // client manages its own timeout
			isFirstProfile := clientErr == nil && len(client) == 0

			profile := &model.Profile{
				Name:           m.profileName,
				Host:           m.host,
				User:           result.Username,
				TokenStorage:   tokenStorage,
				Scopes:         result.Scopes,
				Active:         isFirstProfile,
				EncryptedToken: encryptedToken,
			}

			// Use the result channel to notify completion
			_ = profile
		}
	}()

	// Wait for device code
	select {
	case dc := <-deviceCodeCh:
		return dc
	case <-ctx.Done():
		return oauthResultMsg{err: ctx.Err()}
	}
}

// Update handles messages
func (m *ProfileLoginModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	case spinner.TickMsg:
		var cmd tea.Cmd

		m.spinner, cmd = m.spinner.Update(msg)

		return m, cmd
	case deviceCodeMsg:
		m.deviceCode = msg.code
		m.verificationURL = msg.url
		m.state = stateWaitingForAuth

		return m, m.spinner.Tick
	case oauthResultMsg:
		if msg.err != nil {
			m.err = msg.err
			m.state = stateError

			return m, tea.Quit
		}

		m.profile = msg.profile
		m.state = stateComplete

		return m, tea.Quit
	}

	return m, nil
}

// View renders the UI
func (m *ProfileLoginModel) View() string {
	var sb strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205"))

	codeStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		Background(lipgloss.Color("236")).
		Padding(0, 1)

	urlStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("39")).
		Underline(true)

	successStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("42"))

	errorStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("196"))

	switch m.state {
	case stateInitializing:
		sb.WriteString(titleStyle.Render("Creating profile: "+m.profileName) + "\n\n")
		sb.WriteString(m.spinner.View() + " Initializing OAuth flow...\n")
	case stateWaitingForAuth:
		sb.WriteString(titleStyle.Render("GitHub OAuth Authentication") + "\n\n")
		sb.WriteString("1. Copy this code: " + codeStyle.Render(m.deviceCode) + "\n\n")
		sb.WriteString("2. Open: " + urlStyle.Render(m.verificationURL) + "\n\n")
		sb.WriteString("3. Paste the code and authorize clonr\n\n")
		sb.WriteString(m.spinner.View() + " Waiting for authorization...\n\n")
		sb.WriteString("Press q to cancel")
	case stateComplete:
		sb.WriteString(successStyle.Render("Success!") + "\n\n")

		if m.profile != nil {
			sb.WriteString(fmt.Sprintf("Profile: %s\n", m.profile.Name))
			sb.WriteString(fmt.Sprintf("User: %s\n", m.profile.User))
			sb.WriteString(fmt.Sprintf("Host: %s\n", m.profile.Host))
			sb.WriteString(fmt.Sprintf("Storage: %s\n", m.profile.TokenStorage))

			if m.profile.Active {
				sb.WriteString("\nThis profile is now active.\n")
			}
		}
	case stateError:
		sb.WriteString(errorStyle.Render("Error") + "\n\n")
		sb.WriteString(m.err.Error() + "\n")
	}

	return sb.String()
}

// Profile returns the created profile (after completion)
func (m *ProfileLoginModel) Profile() *model.Profile {
	return m.profile
}

// Error returns any error that occurred
func (m *ProfileLoginModel) Error() error {
	return m.err
}
