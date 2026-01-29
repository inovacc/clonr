package core

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/cli/oauth"
	"github.com/google/go-github/v67/github"
)

// OAuthConfig holds OAuth configuration
type OAuthConfig struct {
	Host     string
	Scopes   []string
	ClientID string
}

// OAuthResult contains the result of an OAuth flow
type OAuthResult struct {
	Token    string
	Username string
	Scopes   []string
}

// OAuthFlow handles the OAuth device flow
type OAuthFlow struct {
	config       OAuthConfig
	onDeviceCode func(code, verificationURL string)
}

// DeviceCode represents the device code response from GitHub
type DeviceCode struct {
	UserCode        string
	VerificationURL string
}

// DefaultClientID is the OAuth client ID for clonr
// This uses the gh CLI's OAuth App credentials since cli/oauth expects them
// The cli/oauth package handles the client ID internally when using its default flow
const DefaultClientID = ""

var (
	// ErrOAuthCanceled is returned when the OAuth flow is canceled
	ErrOAuthCanceled = errors.New("OAuth flow canceled")

	// ErrOAuthExpired is returned when the device code expires
	ErrOAuthExpired = errors.New("device code expired")

	// ErrOAuthDenied is returned when the user denies authorization
	ErrOAuthDenied = errors.New("authorization denied by user")
)

// NewOAuthFlow creates a new OAuth flow
func NewOAuthFlow(host string, scopes []string) *OAuthFlow {
	return &OAuthFlow{
		config: OAuthConfig{
			Host:   host,
			Scopes: scopes,
		},
	}
}

// OnDeviceCode sets the callback for when a device code is received
func (f *OAuthFlow) OnDeviceCode(callback func(code, verificationURL string)) {
	f.onDeviceCode = callback
}

// Run executes the OAuth device flow and returns the result
func (f *OAuthFlow) Run(ctx context.Context) (*OAuthResult, error) {
	// Create the oauth host
	host, err := oauth.NewGitHubHost(f.getGitHubHost())
	if err != nil {
		return nil, fmt.Errorf("invalid GitHub host: %w", err)
	}

	// Create the oauth flow using cli/oauth
	flow := &oauth.Flow{
		Host:   host,
		Scopes: f.config.Scopes,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Set device callback if provided
	if f.onDeviceCode != nil {
		flow.DisplayCode = func(code, verificationURL string) error {
			f.onDeviceCode(code, verificationURL)

			return nil
		}
	}

	// Run the OAuth flow
	accessToken, err := flow.DetectFlow()
	if err != nil {
		return nil, fmt.Errorf("OAuth flow failed: %w", err)
	}

	// Get user info
	username, err := f.getUsername(ctx, accessToken.Token)
	if err != nil {
		return nil, fmt.Errorf("failed to get username: %w", err)
	}

	return &OAuthResult{
		Token:    accessToken.Token,
		Username: username,
		Scopes:   f.config.Scopes,
	}, nil
}

// getGitHubHost returns the host string for oauth
func (f *OAuthFlow) getGitHubHost() string {
	if f.config.Host == "" || f.config.Host == "github.com" {
		return "github.com"
	}

	return f.config.Host
}

// getUsername fetches the authenticated user's username
func (f *OAuthFlow) getUsername(ctx context.Context, token string) (string, error) {
	client := github.NewClient(nil).WithAuthToken(token)

	// Handle enterprise GitHub
	if f.config.Host != "" && f.config.Host != "github.com" {
		baseURL := fmt.Sprintf("https://%s/api/v3/", f.config.Host)
		uploadURL := fmt.Sprintf("https://%s/api/uploads/", f.config.Host)

		var err error

		client, err = client.WithEnterpriseURLs(baseURL, uploadURL)
		if err != nil {
			return "", fmt.Errorf("failed to configure enterprise client: %w", err)
		}
	}

	user, _, err := client.Users.Get(ctx, "")
	if err != nil {
		return "", fmt.Errorf("failed to get user: %w", err)
	}

	return user.GetLogin(), nil
}

// ValidateToken checks if a token is still valid by making an API call
func ValidateToken(ctx context.Context, token, host string) (bool, string, error) {
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

		return false, "", fmt.Errorf("token validation failed: %w", err)
	}

	return true, user.GetLogin(), nil
}
