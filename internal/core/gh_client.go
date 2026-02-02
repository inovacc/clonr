package core

import (
	"context"
	"net/http"

	"github.com/google/go-github/v67/github"
	"golang.org/x/oauth2"
)

// NewGitHubClient creates a new authenticated GitHub client using the provided token.
// This is the standard way to create GitHub API clients throughout the codebase.
func NewGitHubClient(ctx context.Context, token string) *github.Client {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	return github.NewClient(tc)
}

// NewGitHubClientWithContext creates a new authenticated GitHub client using background context.
// This is a convenience function for cases where a fresh context is preferred.
func NewGitHubClientWithContext(token string) *github.Client {
	return NewGitHubClient(context.Background(), token)
}

// NewOAuth2HTTPClient creates an authenticated HTTP client for direct HTTP requests.
// Use this when you need to make authenticated requests outside of the GitHub API,
// such as downloading release assets directly.
func NewOAuth2HTTPClient(ctx context.Context, token string) *http.Client {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	return oauth2.NewClient(ctx, ts)
}
