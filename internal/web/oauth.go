package web

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/go-github/v82/github"
	"github.com/inovacc/clonr/internal/core"
	"github.com/inovacc/clonr/internal/crypto/tpm"
	"github.com/inovacc/clonr/internal/model"
	"github.com/inovacc/clonr/internal/slack"
)

// OAuthSession tracks an in-progress OAuth flow
type OAuthSession struct {
	ProfileName string
	Workspace   string
	Host        string
	DeviceCode  string
	UserCode    string
	VerifyURL   string
	ExpiresAt   time.Time
	Completed   bool
	Error       string
	Result      *OAuthResult
}

// OAuthResult holds the result of a completed OAuth flow
type OAuthResult struct {
	Token    string
	Username string
	Scopes   []string
}

// oauthSessions stores active OAuth sessions
var (
	oauthSessions     = make(map[string]*OAuthSession)
	oauthSessionMutex sync.RWMutex
)

// handleGitHubOAuthStart initiates a GitHub OAuth device flow
func (s *Server) handleGitHubOAuthStart(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	workspace := strings.TrimSpace(r.URL.Query().Get("workspace"))
	host := strings.TrimSpace(r.URL.Query().Get("host"))

	if name == "" {
		s.jsonError(w, "Profile name required", http.StatusBadRequest)
		return
	}

	if workspace == "" {
		s.jsonError(w, "Workspace required", http.StatusBadRequest)
		return
	}

	if host == "" {
		host = "github.com"
	}

	// Check if profile exists
	exists, err := s.grpcClient.ProfileExists(name) //nolint:contextcheck // client manages its own timeout
	if err != nil {
		s.jsonError(w, "Failed to check profile existence", http.StatusInternalServerError)
		return
	}
	if exists {
		s.jsonError(w, "Profile already exists", http.StatusConflict)
		return
	}

	// Check workspace exists
	wsExists, err := s.grpcClient.WorkspaceExists(workspace) //nolint:contextcheck // client manages its own timeout
	if err != nil {
		s.jsonError(w, "Failed to check workspace existence", http.StatusInternalServerError)
		return
	}
	if !wsExists {
		s.jsonError(w, "Workspace not found", http.StatusBadRequest)
		return
	}

	// Create OAuth session
	session := &OAuthSession{
		ProfileName: name,
		Workspace:   workspace,
		Host:        host,
		ExpiresAt:   time.Now().Add(10 * time.Minute),
	}

	// Start OAuth flow in background
	go func() { //nolint:contextcheck // background goroutine creates its own context
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		flow := core.NewOAuthFlow(host, model.DefaultScopes())

		flow.OnDeviceCode(func(code, url string) {
			oauthSessionMutex.Lock()
			session.UserCode = code
			session.VerifyURL = url
			oauthSessions[name] = session
			oauthSessionMutex.Unlock()
		})

		result, err := flow.Run(ctx)
		if err != nil {
			oauthSessionMutex.Lock()
			session.Error = err.Error()
			oauthSessionMutex.Unlock()
			return
		}

		oauthSessionMutex.Lock()
		session.Completed = true
		session.Result = &OAuthResult{
			Token:    result.Token,
			Username: result.Username,
			Scopes:   result.Scopes,
		}
		oauthSessionMutex.Unlock()
	}()

	// Wait a bit for device code to be available
	time.Sleep(500 * time.Millisecond)

	oauthSessionMutex.RLock()
	sess, exists := oauthSessions[name]
	oauthSessionMutex.RUnlock()

	if !exists || sess.UserCode == "" {
		// Still waiting, return polling response
		s.jsonResponse(w, map[string]any{
			"status":  "pending",
			"message": "Starting OAuth flow...",
		})
		return
	}

	s.jsonResponse(w, map[string]any{
		"status":    "waiting",
		"user_code": sess.UserCode,
		"verify_url": sess.VerifyURL,
		"message":   "Enter the code in your browser",
	})
}

// handleGitHubOAuthStatus checks the status of an OAuth flow
func (s *Server) handleGitHubOAuthStatus(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name == "" {
		s.jsonError(w, "Profile name required", http.StatusBadRequest)
		return
	}

	oauthSessionMutex.RLock()
	session, exists := oauthSessions[name]
	oauthSessionMutex.RUnlock()

	if !exists {
		s.jsonError(w, "OAuth session not found", http.StatusNotFound)
		return
	}

	if session.Error != "" {
		s.jsonResponse(w, map[string]any{
			"status": "error",
			"error":  session.Error,
		})
		// Clean up session
		oauthSessionMutex.Lock()
		delete(oauthSessions, name)
		oauthSessionMutex.Unlock()
		return
	}

	if session.Completed && session.Result != nil {
		// Create the profile
		result := session.Result

		// Encrypt token
		encryptedToken, err := tpm.EncryptToken(result.Token, name, session.Host)
		if err != nil {
			s.jsonError(w, "Failed to encrypt token", http.StatusInternalServerError)
			return
		}

		tokenStorage := model.TokenStorageEncrypted
		if tpm.IsDataOpen(encryptedToken) {
			tokenStorage = model.TokenStorageOpen
		}

		// Check if first profile
		profiles, _ := s.pm.ListProfiles()
		isFirst := len(profiles) == 0

		// Create profile
		profile := &model.Profile{
			Name:           name,
			Host:           session.Host,
			User:           result.Username,
			TokenStorage:   tokenStorage,
			Scopes:         result.Scopes,
			Default:        isFirst,
			EncryptedToken: encryptedToken,
			CreatedAt:      time.Now(),
			LastUsedAt:     time.Now(),
			Workspace:      session.Workspace,
		}

		if err := s.grpcClient.SaveProfile(profile); err != nil { //nolint:contextcheck // client manages its own timeout
			s.jsonError(w, "Failed to save profile", http.StatusInternalServerError)
			return
		}

		// Clean up session
		oauthSessionMutex.Lock()
		delete(oauthSessions, name)
		oauthSessionMutex.Unlock()

		s.jsonResponse(w, map[string]any{
			"status":   "completed",
			"username": result.Username,
			"profile":  profile.Name,
		})
		return
	}

	// Still waiting
	response := map[string]any{
		"status": "waiting",
	}

	if session.UserCode != "" {
		response["user_code"] = session.UserCode
		response["verify_url"] = session.VerifyURL
	}

	s.jsonResponse(w, response)
}

// Slack OAuth handling

// SlackOAuthSession tracks an in-progress Slack OAuth flow
type SlackOAuthSession struct {
	ProfileName string
	State       string
	ExpiresAt   time.Time
	Completed   bool
	Error       string
	Result      *slack.OAuthResult
}

var (
	slackOAuthSessions     = make(map[string]*SlackOAuthSession)
	slackOAuthSessionMutex sync.RWMutex
)

// handleSlackOAuthStart initiates a Slack OAuth flow
func (s *Server) handleSlackOAuthStart(w http.ResponseWriter, r *http.Request) {
	clientID := strings.TrimSpace(r.URL.Query().Get("client_id"))
	clientSecret := strings.TrimSpace(r.URL.Query().Get("client_secret"))

	if clientID == "" || clientSecret == "" {
		s.jsonError(w, "Slack client_id and client_secret required", http.StatusBadRequest)
		return
	}

	activeProfile, err := s.pm.GetActiveProfile()
	if err != nil || activeProfile == nil {
		s.jsonError(w, "No active profile", http.StatusBadRequest)
		return
	}

	// Create OAuth handler
	port := s.config.Port
	config := slack.OAuthConfig{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Port:         port,
		RedirectURI:  fmt.Sprintf("http://localhost:%d/oauth/slack/callback", port),
	}

	handler, err := slack.NewOAuthHandler(config)
	if err != nil {
		s.jsonError(w, "Failed to create OAuth handler", http.StatusInternalServerError)
		return
	}

	// Get authorization URL
	authURL := handler.GetAuthorizationURL()

	// Store session (the state is internal to the handler)
	session := &SlackOAuthSession{
		ProfileName: activeProfile.Name,
		ExpiresAt:   time.Now().Add(10 * time.Minute),
	}

	slackOAuthSessionMutex.Lock()
	slackOAuthSessions[activeProfile.Name] = session
	slackOAuthSessionMutex.Unlock()

	// Return URL for browser redirect
	s.jsonResponse(w, map[string]any{
		"auth_url": authURL,
	})
}

// handleSlackOAuthCallback handles the Slack OAuth callback
func (s *Server) handleSlackOAuthCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	errMsg := r.URL.Query().Get("error")

	if errMsg != "" {
		s.renderSlackCallbackError(w, "Authorization Failed", errMsg)
		return
	}

	if code == "" {
		s.renderSlackCallbackError(w, "Authorization Failed", "No authorization code received")
		return
	}

	// For now, show success and store the code
	// The actual token exchange would need the client secret
	s.renderSlackCallbackSuccess(w)
}

// renderSlackCallbackSuccess renders success page for Slack OAuth callback
func (s *Server) renderSlackCallbackSuccess(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprint(w, `<!DOCTYPE html>
<html>
<head>
    <title>Slack Authorization Successful</title>
    <script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="bg-gradient-to-br from-purple-600 to-blue-600 min-h-screen flex items-center justify-center">
    <div class="bg-white p-10 rounded-xl shadow-2xl text-center">
        <div class="text-6xl mb-4">&#10003;</div>
        <h1 class="text-2xl font-bold text-green-600 mb-2">Authorization Successful!</h1>
        <p class="text-gray-600">You can close this window and return to clonr.</p>
    </div>
    <script>
        // Close window after delay or notify parent
        setTimeout(() => {
            if (window.opener) {
                window.opener.postMessage({type: 'slack-oauth-success'}, '*');
                window.close();
            }
        }, 2000);
    </script>
</body>
</html>`)
}

// renderSlackCallbackError renders error page for Slack OAuth callback
func (s *Server) renderSlackCallbackError(w http.ResponseWriter, title, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusBadRequest)
	_, _ = fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
    <title>Slack Authorization Failed</title>
    <script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="bg-gradient-to-br from-purple-600 to-blue-600 min-h-screen flex items-center justify-center">
    <div class="bg-white p-10 rounded-xl shadow-2xl text-center">
        <div class="text-6xl mb-4">&#10007;</div>
        <h1 class="text-2xl font-bold text-red-600 mb-2">%s</h1>
        <p class="text-gray-600">%s</p>
    </div>
</body>
</html>`, title, message)
}

// handleSlackStatus returns Slack integration status for active profile
func (s *Server) handleSlackStatus(w http.ResponseWriter, r *http.Request) {
	activeProfile, err := s.pm.GetActiveProfile()
	if err != nil || activeProfile == nil {
		s.jsonResponse(w, map[string]any{
			"connected": false,
			"error":     "No active profile",
		})
		return
	}

	// Check if Slack channel exists
	channel, err := s.pm.GetNotifyChannelByType(activeProfile.Name, model.ChannelSlack)
	if err != nil || channel == nil {
		s.jsonResponse(w, map[string]any{
			"connected": false,
			"profile":   activeProfile.Name,
		})
		return
	}

	s.jsonResponse(w, map[string]any{
		"connected": true,
		"profile":   activeProfile.Name,
		"channel":   channel.Name,
	})
}

// handleSlackAdd adds Slack integration with a token
func (s *Server) handleSlackAdd(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.jsonError(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	token := strings.TrimSpace(r.FormValue("token"))
	if token == "" {
		s.jsonError(w, "Token required", http.StatusBadRequest)
		return
	}

	activeProfile, err := s.pm.GetActiveProfile()
	if err != nil || activeProfile == nil {
		s.jsonError(w, "No active profile", http.StatusBadRequest)
		return
	}

	// Create Slack notify channel
	channel := &model.NotifyChannel{
		ID:      "slack-" + activeProfile.Name,
		Name:    "Slack",
		Type:    model.ChannelSlack,
		Enabled: true,
		Config: map[string]string{
			"bot_token": token,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.pm.AddNotifyChannel(activeProfile.Name, channel); err != nil {
		s.jsonError(w, "Failed to add Slack channel", http.StatusInternalServerError)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		s.renderPartial(w, "slack_status.html", map[string]any{
			"connected": true,
			"profile":   activeProfile.Name,
			"channel":   channel.Name,
		})
		return
	}

	s.jsonResponse(w, APIResponse{
		Success: true,
		Message: "Slack integration added",
	})
}

// handleSlackRemove removes Slack integration
func (s *Server) handleSlackRemove(w http.ResponseWriter, r *http.Request) {
	activeProfile, err := s.pm.GetActiveProfile()
	if err != nil || activeProfile == nil {
		s.jsonError(w, "No active profile", http.StatusBadRequest)
		return
	}

	channelID := "slack-" + activeProfile.Name
	if err := s.pm.RemoveNotifyChannel(activeProfile.Name, channelID); err != nil {
		log.Printf("Failed to remove Slack channel: %v", err)
	}

	if r.Header.Get("HX-Request") == "true" {
		s.renderPartial(w, "slack_status.html", map[string]any{
			"connected": false,
			"profile":   activeProfile.Name,
		})
		return
	}

	s.jsonResponse(w, APIResponse{
		Success: true,
		Message: "Slack integration removed",
	})
}

// validateGitHubToken validates a GitHub token and returns username
func validateGitHubToken(ctx context.Context, token, host string) (bool, string, error) {
	client := github.NewClient(nil).WithAuthToken(token)

	// Handle enterprise GitHub
	if host != "" && host != "github.com" {
		baseURL := fmt.Sprintf("https://%s/api/v3/", host)
		uploadURL := fmt.Sprintf("https://%s/api/uploads/", host)

		var err error
		client, err = client.WithEnterpriseURLs(baseURL, uploadURL)
		if err != nil {
			return false, "", fmt.Errorf("failed to configure enterprise client: %w", err)
		}
	}

	user, resp, err := client.Users.Get(ctx, "")
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusUnauthorized {
			return false, "", nil
		}
		return false, "", err
	}

	return true, user.GetLogin(), nil
}
