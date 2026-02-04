package slack

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// SlackOAuthAuthorizeURL is the Slack OAuth authorization endpoint.
	SlackOAuthAuthorizeURL = "https://slack.com/oauth/v2/authorize"
	// SlackOAuthTokenURL is the Slack OAuth token endpoint.
	SlackOAuthTokenURL = "https://slack.com/api/oauth.v2.access"

	// DefaultOAuthPort is the default port for the local callback server.
	DefaultOAuthPort = 8338
	// OAuthCallbackPath is the callback path for OAuth redirect.
	OAuthCallbackPath = "/slack/callback"

	// DefaultScopes are the bot token scopes needed for pm slack commands.
	DefaultScopes = "channels:read,channels:history,groups:read,groups:history,im:history,mpim:history,search:read,users:read"
)

// OAuthConfig holds the OAuth configuration.
type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	Scopes       string
	Port         int
}

// OAuthResult contains the result of the OAuth flow.
type OAuthResult struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	Scope        string    `json:"scope"`
	BotUserID    string    `json:"bot_user_id"`
	AppID        string    `json:"app_id"`
	Team         TeamInfo  `json:"team"`
	AuthedUser   UserInfo  `json:"authed_user"`
	Enterprise   *EntInfo  `json:"enterprise,omitempty"`
	IsEnterprise bool      `json:"is_enterprise_install"`
	ExpiresIn    int       `json:"expires_in,omitempty"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	ObtainedAt   time.Time `json:"-"`
}

// TeamInfo contains team information from OAuth.
type TeamInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// UserInfo contains user information from OAuth.
type UserInfo struct {
	ID          string `json:"id"`
	Scope       string `json:"scope"`
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

// EntInfo contains enterprise information.
type EntInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// OAuthHandler manages the OAuth flow.
type OAuthHandler struct {
	config     OAuthConfig
	state      string
	resultChan chan *oauthCallbackResult
	server     *http.Server
}

type oauthCallbackResult struct {
	code  string
	state string
	err   error
}

// NewOAuthHandler creates a new OAuth handler.
func NewOAuthHandler(config OAuthConfig) (*OAuthHandler, error) {
	if config.ClientID == "" {
		return nil, fmt.Errorf("client ID is required")
	}

	if config.ClientSecret == "" {
		return nil, fmt.Errorf("client secret is required")
	}

	if config.Port == 0 {
		config.Port = DefaultOAuthPort
	}

	if config.Scopes == "" {
		config.Scopes = DefaultScopes
	}

	if config.RedirectURI == "" {
		config.RedirectURI = fmt.Sprintf("http://localhost:%d%s", config.Port, OAuthCallbackPath)
	}

	// Generate random state
	state, err := generateState()
	if err != nil {
		return nil, fmt.Errorf("failed to generate state: %w", err)
	}

	return &OAuthHandler{
		config:     config,
		state:      state,
		resultChan: make(chan *oauthCallbackResult, 1),
	}, nil
}

// GetAuthorizationURL returns the URL to redirect the user for authorization.
func (h *OAuthHandler) GetAuthorizationURL() string {
	params := url.Values{}
	params.Set("client_id", h.config.ClientID)
	params.Set("scope", h.config.Scopes)
	params.Set("redirect_uri", h.config.RedirectURI)
	params.Set("state", h.state)

	return fmt.Sprintf("%s?%s", SlackOAuthAuthorizeURL, params.Encode())
}

// StartCallbackServer starts the local HTTP server to receive the OAuth callback.
func (h *OAuthHandler) StartCallbackServer(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc(OAuthCallbackPath, h.handleCallback)

	// Also handle root path in case of misconfiguration
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			_, _ = fmt.Fprintf(w, "Waiting for Slack OAuth callback...")
			return
		}

		http.NotFound(w, r)
	})

	addr := fmt.Sprintf("127.0.0.1:%d", h.config.Port)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to start callback server: %w", err)
	}

	h.server = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		if err := h.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			h.resultChan <- &oauthCallbackResult{err: err}
		}
	}()

	return nil
}

// WaitForCallback waits for the OAuth callback and returns the authorization code.
func (h *OAuthHandler) WaitForCallback(ctx context.Context, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	select {
	case result := <-h.resultChan:
		if result.err != nil {
			return "", result.err
		}

		if result.state != h.state {
			return "", fmt.Errorf("state mismatch: possible CSRF attack")
		}

		return result.code, nil
	case <-ctx.Done():
		return "", fmt.Errorf("timeout waiting for OAuth callback")
	}
}

// ExchangeCode exchanges the authorization code for an access token.
func (h *OAuthHandler) ExchangeCode(ctx context.Context, code string) (*OAuthResult, error) {
	data := url.Values{}
	data.Set("client_id", h.config.ClientID)
	data.Set("client_secret", h.config.ClientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", h.config.RedirectURI)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, SlackOAuthTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var tokenResp struct {
		OAuthResult

		OK    bool   `json:"ok"`
		Error string `json:"error,omitempty"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !tokenResp.OK {
		return nil, fmt.Errorf("oauth error: %s", tokenResp.Error)
	}

	tokenResp.ObtainedAt = time.Now()

	return &tokenResp.OAuthResult, nil
}

// Shutdown gracefully shuts down the callback server.
func (h *OAuthHandler) Shutdown(ctx context.Context) error {
	if h.server != nil {
		return h.server.Shutdown(ctx)
	}

	return nil
}

// handleCallback handles the OAuth callback from Slack.
func (h *OAuthHandler) handleCallback(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	// Check for error
	if errMsg := query.Get("error"); errMsg != "" {
		h.resultChan <- &oauthCallbackResult{
			err: fmt.Errorf("oauth error: %s", errMsg),
		}

		h.renderError(w, "Authorization Failed", errMsg)

		return
	}

	code := query.Get("code")
	state := query.Get("state")

	if code == "" {
		h.resultChan <- &oauthCallbackResult{
			err: fmt.Errorf("no authorization code received"),
		}

		h.renderError(w, "Authorization Failed", "No authorization code received")

		return
	}

	h.resultChan <- &oauthCallbackResult{
		code:  code,
		state: state,
	}

	h.renderSuccess(w)
}

// renderSuccess renders the success page.
func (h *OAuthHandler) renderSuccess(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprint(w, `<!DOCTYPE html>
<html>
<head>
    <title>Slack Authorization Successful</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
            display: flex;
            justify-content: center;
            align-items: center;
            height: 100vh;
            margin: 0;
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
        }
        .container {
            background: white;
            padding: 40px 60px;
            border-radius: 12px;
            box-shadow: 0 10px 40px rgba(0,0,0,0.2);
            text-align: center;
        }
        .success-icon {
            font-size: 64px;
            margin-bottom: 20px;
        }
        h1 {
            color: #2eb67d;
            margin: 0 0 10px 0;
        }
        p {
            color: #666;
            margin: 0;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="success-icon">✅</div>
        <h1>Authorization Successful!</h1>
        <p>You can close this window and return to the terminal.</p>
    </div>
</body>
</html>`)
}

// renderError renders the error page.
func (h *OAuthHandler) renderError(w http.ResponseWriter, title, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusBadRequest)
	_, _ = fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
    <title>Slack Authorization Failed</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
            display: flex;
            justify-content: center;
            align-items: center;
            height: 100vh;
            margin: 0;
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
        }
        .container {
            background: white;
            padding: 40px 60px;
            border-radius: 12px;
            box-shadow: 0 10px 40px rgba(0,0,0,0.2);
            text-align: center;
        }
        .error-icon {
            font-size: 64px;
            margin-bottom: 20px;
        }
        h1 {
            color: #e01e5a;
            margin: 0 0 10px 0;
        }
        p {
            color: #666;
            margin: 0;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="error-icon">❌</div>
        <h1>%s</h1>
        <p>%s</p>
    </div>
</body>
</html>`, title, message)
}

// generateState generates a random state string for CSRF protection.
func generateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(b), nil
}

// RunOAuthFlow runs the complete OAuth flow.
func RunOAuthFlow(ctx context.Context, config OAuthConfig, openBrowser func(string) error) (*OAuthResult, error) {
	handler, err := NewOAuthHandler(config)
	if err != nil {
		return nil, err
	}

	// Start callback server
	if err := handler.StartCallbackServer(ctx); err != nil {
		return nil, err
	}

	defer func() { //nolint:contextcheck // Use Background for shutdown since parent ctx may be cancelled
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_ = handler.Shutdown(shutdownCtx)
	}()

	// Open browser
	authURL := handler.GetAuthorizationURL()
	if err := openBrowser(authURL); err != nil {
		return nil, fmt.Errorf("failed to open browser: %w\n\nPlease open this URL manually:\n%s", err, authURL)
	}

	// Wait for callback
	code, err := handler.WaitForCallback(ctx, 5*time.Minute)
	if err != nil {
		return nil, err
	}

	// Exchange code for token
	result, err := handler.ExchangeCode(ctx, code)
	if err != nil {
		return nil, err
	}

	return result, nil
}
