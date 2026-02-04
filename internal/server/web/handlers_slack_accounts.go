package web

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/inovacc/clonr/internal/crypto/tpm"
	"github.com/inovacc/clonr/internal/model"
	"github.com/inovacc/clonr/internal/slack"
)

// SlackAccountsPageData holds data for the Slack accounts page
type SlackAccountsPageData struct {
	Accounts      []*model.SlackAccount
	ActiveAccount *model.SlackAccount
}

// handleSlackAccountsPage renders the Slack accounts list page
func (s *Server) handleSlackAccountsPage(w http.ResponseWriter, _ *http.Request) {
	accounts, err := s.slackAccountService.ListAccounts()
	if err != nil {
		log.Printf("Failed to list Slack accounts: %v", err)
		accounts = []*model.SlackAccount{}
	}

	activeAccount, _ := s.slackAccountService.GetActiveAccount()

	data := PageData{
		Title:        "Slack Accounts",
		ActivePage:   "slack",
		TPMAvailable: tpm.IsTPMAvailable(),
		Data: SlackAccountsPageData{
			Accounts:      accounts,
			ActiveAccount: activeAccount,
		},
	}

	s.render(w, "slack_accounts.html", data)
}

// handleSlackAccountAddPage renders the add Slack account page
func (s *Server) handleSlackAccountAddPage(w http.ResponseWriter, _ *http.Request) {
	data := PageData{
		Title:        "Add Slack Account",
		ActivePage:   "slack",
		TPMAvailable: tpm.IsTPMAvailable(),
	}

	s.render(w, "slack_account_add.html", data)
}

// handleListSlackAccounts returns all Slack accounts as JSON or HTML partial
func (s *Server) handleListSlackAccounts(w http.ResponseWriter, r *http.Request) {
	accounts, err := s.slackAccountService.ListAccounts()
	if err != nil {
		s.jsonError(w, "Failed to list Slack accounts", http.StatusInternalServerError)
		return
	}

	// Check if HTMX request
	if r.Header.Get("HX-Request") == "true" {
		s.renderPartial(w, "slack_account_list.html", accounts)
		return
	}

	s.jsonResponse(w, accounts)
}

// handleCreateSlackAccount creates a new Slack account
func (s *Server) handleCreateSlackAccount(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.jsonError(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	token := strings.TrimSpace(r.FormValue("token"))

	if name == "" {
		s.jsonError(w, "Account name is required", http.StatusBadRequest)
		return
	}

	if token == "" {
		s.jsonError(w, "Bot token is required", http.StatusBadRequest)
		return
	}

	// Validate token format
	if !strings.HasPrefix(token, "xoxb-") {
		s.jsonError(w, "Invalid token format. Bot tokens should start with 'xoxb-'", http.StatusBadRequest)
		return
	}

	// Check if account already exists
	exists, err := s.slackAccountService.AccountExists(name)
	if err != nil {
		log.Printf("Failed to check Slack account existence for %q: %v", name, err)
		s.jsonError(w, fmt.Sprintf("Failed to check account existence: %v", err), http.StatusInternalServerError)
		return
	}
	if exists {
		s.jsonError(w, "Account with this name already exists", http.StatusConflict)
		return
	}

	// Validate the token by calling Slack API
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	testClient := slack.NewClient(token, slack.ClientOptions{})
	authInfo, err := testClient.AuthTest(ctx)
	if err != nil {
		log.Printf("Slack token validation failed: %v", err)
		s.jsonError(w, fmt.Sprintf("Token validation failed: %v", err), http.StatusBadRequest)
		return
	}

	log.Printf("Slack token validated for workspace %q (user: %s)", authInfo.Team, authInfo.User)

	// Create account with workspace info from auth test
	account, err := s.slackAccountService.CreateAccountWithInfo(
		name,
		token,
		authInfo.TeamID,
		authInfo.Team,
		authInfo.UserID,
		authInfo.TeamID,
	)
	if err != nil {
		log.Printf("Failed to create Slack account: %v", err)
		s.jsonError(w, fmt.Sprintf("Failed to create account: %v", err), http.StatusInternalServerError)
		return
	}

	// Broadcast SSE event
	s.BroadcastEvent(EventSlackAccountCreated, "Slack account created", map[string]any{
		"name":      account.Name,
		"workspace": account.WorkspaceName,
	})

	// Return success
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/slack/accounts")
		w.WriteHeader(http.StatusOK)
		return
	}

	s.jsonResponse(w, APIResponse{
		Success: true,
		Message: "Slack account created successfully",
		Data:    account,
	})
}

// handleDeleteSlackAccount deletes a Slack account
func (s *Server) handleDeleteSlackAccount(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		s.jsonError(w, "Account name required", http.StatusBadRequest)
		return
	}

	if err := s.slackAccountService.DeleteAccount(name); err != nil {
		log.Printf("Failed to delete Slack account %q: %v", name, err)
		s.jsonError(w, "Failed to delete account", http.StatusInternalServerError)
		return
	}

	// Broadcast SSE event
	s.BroadcastEvent(EventSlackAccountDeleted, "Slack account deleted", map[string]any{
		"name": name,
	})

	// Check if HTMX request
	if r.Header.Get("HX-Request") == "true" {
		// Return empty response to remove the row
		w.WriteHeader(http.StatusOK)
		return
	}

	s.jsonResponse(w, APIResponse{
		Success: true,
		Message: "Slack account deleted",
	})
}

// handleSetActiveSlackAccount sets a Slack account as active
func (s *Server) handleSetActiveSlackAccount(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		s.jsonError(w, "Account name required", http.StatusBadRequest)
		return
	}

	if err := s.slackAccountService.SetActiveAccount(name); err != nil {
		log.Printf("Failed to set active Slack account %q: %v", name, err)
		s.jsonError(w, "Failed to set active account", http.StatusInternalServerError)
		return
	}

	// Broadcast SSE event
	s.BroadcastEvent(EventSlackAccountActivated, "Slack account activated", map[string]any{
		"name": name,
	})

	// Check if HTMX request - refresh the account list
	if r.Header.Get("HX-Request") == "true" {
		accounts, _ := s.slackAccountService.ListAccounts()
		s.renderPartial(w, "slack_account_list.html", accounts)
		return
	}

	s.jsonResponse(w, APIResponse{
		Success: true,
		Message: "Active Slack account set",
	})
}
