package microsoft

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// Microsoft identity platform endpoints
	authURLTemplate  = "https://login.microsoftonline.com/%s/oauth2/v2.0/authorize"
	tokenURLTemplate = "https://login.microsoftonline.com/%s/oauth2/v2.0/token"
	graphAPIBaseURL  = "https://graph.microsoft.com/v1.0"
)

// DefaultTeamsScopes are the default Microsoft Teams scopes.
var DefaultTeamsScopes = []string{
	"offline_access",
	"Team.ReadBasic.All",
	"Channel.ReadBasic.All",
	"ChannelMessage.Read.All",
	"Chat.Read",
	"User.Read",
}

// DefaultOutlookScopes are the default Outlook/Mail scopes.
var DefaultOutlookScopes = []string{
	"offline_access",
	"Mail.Read",
	"Mail.ReadBasic",
	"User.Read",
}

// OAuthConfig configures the Microsoft OAuth flow.
type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	TenantID     string // "common", "organizations", "consumers", or specific tenant ID
	RedirectURI  string
	Port         int
	Scopes       []string
}

// OAuthResult contains the result of OAuth authentication.
type OAuthResult struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
	IDToken      string `json:"id_token"`
}

// UserProfile contains Microsoft user profile information.
type UserProfile struct {
	ID                string `json:"id"`
	DisplayName       string `json:"displayName"`
	Mail              string `json:"mail"`
	UserPrincipalName string `json:"userPrincipalName"`
	JobTitle          string `json:"jobTitle"`
	OfficeLocation    string `json:"officeLocation"`
}

// RunOAuthFlow runs the OAuth2 authorization flow for Microsoft services.
func RunOAuthFlow(ctx context.Context, config OAuthConfig, openBrowser func(string) error) (*OAuthResult, error) {
	if config.Port == 0 {
		config.Port = 8340
	}

	if config.TenantID == "" {
		config.TenantID = "common"
	}

	if config.RedirectURI == "" {
		config.RedirectURI = fmt.Sprintf("http://localhost:%d/microsoft/callback", config.Port)
	}

	// Build authorization URL
	authURL := buildAuthURL(config)

	// Start local server to receive callback
	resultChan := make(chan *OAuthResult, 1)
	errChan := make(chan error, 1)

	mux := http.NewServeMux()
	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", config.Port),
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	mux.HandleFunc("/microsoft/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			errMsg := r.URL.Query().Get("error_description")
			if errMsg == "" {
				errMsg = r.URL.Query().Get("error")
			}

			if errMsg == "" {
				errMsg = "no authorization code received"
			}

			errChan <- fmt.Errorf("OAuth error: %s", errMsg)

			w.Header().Set("Content-Type", "text/html")
			_, _ = fmt.Fprintf(w, `<html><body><h1>Authorization Failed</h1><p>%s</p></body></html>`, errMsg)

			return
		}

		// Exchange code for token
		result, err := exchangeCode(ctx, config, code)
		if err != nil {
			errChan <- err

			w.Header().Set("Content-Type", "text/html")
			_, _ = fmt.Fprintf(w, `<html><body><h1>Token Exchange Failed</h1><p>%v</p></body></html>`, err)

			return
		}

		resultChan <- result

		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprint(w, `<html><body>
			<h1>Authorization Successful!</h1>
			<p>You can close this window and return to the terminal.</p>
			<script>window.close();</script>
		</body></html>`)
	})

	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			errChan <- fmt.Errorf("server error: %w", err)
		}
	}()

	defer func() { //nolint:contextcheck // fresh context for shutdown is intentional
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_ = server.Shutdown(shutdownCtx)
	}()

	// Open browser
	if err := openBrowser(authURL); err != nil {
		return nil, fmt.Errorf("failed to open browser: %w", err)
	}

	// Wait for result or timeout
	select {
	case result := <-resultChan:
		return result, nil
	case err := <-errChan:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(5 * time.Minute):
		return nil, fmt.Errorf("OAuth flow timed out")
	}
}

// buildAuthURL constructs the Microsoft OAuth authorization URL.
func buildAuthURL(config OAuthConfig) string {
	baseURL := fmt.Sprintf(authURLTemplate, config.TenantID)

	params := url.Values{}
	params.Set("client_id", config.ClientID)
	params.Set("redirect_uri", config.RedirectURI)
	params.Set("response_type", "code")
	params.Set("scope", strings.Join(config.Scopes, " "))
	params.Set("response_mode", "query")

	return baseURL + "?" + params.Encode()
}

// exchangeCode exchanges the authorization code for tokens.
func exchangeCode(ctx context.Context, config OAuthConfig, code string) (*OAuthResult, error) {
	tokenURL := fmt.Sprintf(tokenURLTemplate, config.TenantID)

	data := url.Values{}
	data.Set("client_id", config.ClientID)
	data.Set("client_secret", config.ClientSecret)
	data.Set("code", code)
	data.Set("grant_type", "authorization_code")
	data.Set("redirect_uri", config.RedirectURI)
	data.Set("scope", strings.Join(config.Scopes, " "))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		return nil, fmt.Errorf("token exchange error %d: %s", resp.StatusCode, string(body))
	}

	var result OAuthResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	return &result, nil
}

// RefreshAccessToken refreshes an access token using a refresh token.
func RefreshAccessToken(ctx context.Context, clientID, clientSecret, tenantID, refreshToken string, scopes []string) (*OAuthResult, error) {
	if tenantID == "" {
		tenantID = "common"
	}

	tokenURL := fmt.Sprintf(tokenURLTemplate, tenantID)

	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("refresh_token", refreshToken)
	data.Set("grant_type", "refresh_token")
	data.Set("scope", strings.Join(scopes, " "))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token refresh failed: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		return nil, fmt.Errorf("token refresh error %d: %s", resp.StatusCode, string(body))
	}

	var result OAuthResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	return &result, nil
}

// GetUserProfile retrieves the current user's profile from Microsoft Graph.
func GetUserProfile(ctx context.Context, accessToken string) (*UserProfile, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, graphAPIBaseURL+"/me", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 30 * time.Second}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var profile UserProfile
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &profile, nil
}
