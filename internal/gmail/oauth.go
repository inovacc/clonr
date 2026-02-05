package gmail

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
	googleAuthURL  = "https://accounts.google.com/o/oauth2/v2/auth"
	googleTokenURL = "https://oauth2.googleapis.com/token"
)

// DefaultScopes are the default Gmail API scopes.
var DefaultScopes = []string{
	"https://www.googleapis.com/auth/gmail.readonly",
	"https://www.googleapis.com/auth/userinfo.email",
}

// OAuthConfig configures the OAuth flow.
type OAuthConfig struct {
	ClientID     string
	ClientSecret string
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

// TokenInfo contains information about the user from the token.
type TokenInfo struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

// RunOAuthFlow runs the OAuth2 authorization flow for Gmail.
func RunOAuthFlow(ctx context.Context, config OAuthConfig, openBrowser func(string) error) (*OAuthResult, error) {
	if config.Port == 0 {
		config.Port = 8339
	}

	if len(config.Scopes) == 0 {
		config.Scopes = DefaultScopes
	}

	if config.RedirectURI == "" {
		config.RedirectURI = fmt.Sprintf("http://localhost:%d/gmail/callback", config.Port)
	}

	// Build authorization URL
	authURL := buildAuthURL(config)

	// Start local server to receive callback
	resultChan := make(chan *OAuthResult, 1)
	errChan := make(chan error, 1)

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", config.Port),
		ReadHeaderTimeout: 10 * time.Second,
	}

	http.HandleFunc("/gmail/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			errMsg := r.URL.Query().Get("error")
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

// buildAuthURL constructs the Google OAuth authorization URL.
func buildAuthURL(config OAuthConfig) string {
	params := url.Values{}
	params.Set("client_id", config.ClientID)
	params.Set("redirect_uri", config.RedirectURI)
	params.Set("response_type", "code")
	params.Set("scope", strings.Join(config.Scopes, " "))
	params.Set("access_type", "offline")
	params.Set("prompt", "consent")

	return googleAuthURL + "?" + params.Encode()
}

// exchangeCode exchanges the authorization code for tokens.
func exchangeCode(ctx context.Context, config OAuthConfig, code string) (*OAuthResult, error) {
	data := url.Values{}
	data.Set("client_id", config.ClientID)
	data.Set("client_secret", config.ClientSecret)
	data.Set("code", code)
	data.Set("grant_type", "authorization_code")
	data.Set("redirect_uri", config.RedirectURI)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, googleTokenURL, strings.NewReader(data.Encode()))
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
func RefreshAccessToken(ctx context.Context, clientID, clientSecret, refreshToken string) (*OAuthResult, error) {
	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("refresh_token", refreshToken)
	data.Set("grant_type", "refresh_token")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, googleTokenURL, strings.NewReader(data.Encode()))
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

// ValidateToken validates an access token by calling the tokeninfo endpoint.
func ValidateToken(ctx context.Context, accessToken string) (*TokenInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://www.googleapis.com/oauth2/v3/userinfo", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 30 * time.Second}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("validation request failed: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		return nil, fmt.Errorf("token validation error %d: %s", resp.StatusCode, string(body))
	}

	var info TokenInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to decode userinfo response: %w", err)
	}

	return &info, nil
}
