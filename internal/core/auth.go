package core

import (
	"fmt"
	"os"

	"github.com/cli/go-gh/v2/pkg/auth"
)

// TokenSource indicates where the token was found
type TokenSource string

const (
	TokenSourceFlag      TokenSource = "flag"
	TokenSourceEnvGitHub TokenSource = "GITHUB_TOKEN"
	TokenSourceEnvGH     TokenSource = "GH_TOKEN"
	TokenSourceGHCLI     TokenSource = "gh-cli"
	TokenSourceNone      TokenSource = "none"
)

// ResolveGitHubToken attempts to find a GitHub token from multiple sources.
// Priority order:
//  1. flagToken (explicit --token flag)
//  2. GITHUB_TOKEN environment variable
//  3. GH_TOKEN environment variable
//  4. gh CLI auth (keyring + config file)
func ResolveGitHubToken(flagToken string) (token string, source TokenSource, err error) {
	// 1. Flag has highest priority
	if flagToken != "" {
		return flagToken, TokenSourceFlag, nil
	}

	// 2. Check GITHUB_TOKEN env var
	if token = os.Getenv("GITHUB_TOKEN"); token != "" {
		return token, TokenSourceEnvGitHub, nil
	}

	// 3. Check GH_TOKEN env var
	if token = os.Getenv("GH_TOKEN"); token != "" {
		return token, TokenSourceEnvGH, nil
	}

	// 4. Try gh CLI auth (keyring + config file)
	if token, _ = auth.TokenForHost("github.com"); token != "" {
		return token, TokenSourceGHCLI, nil
	}

	// 5. No token found
	return "", TokenSourceNone, fmt.Errorf(`GitHub token required

Provide a token via one of:
  * gh auth login          (recommended - auto-detected)
  * GITHUB_TOKEN env var
  * --token flag

Create a token at: https://github.com/settings/tokens`)
}

// ResolveGitHubTokenForHost resolves token for a specific host (enterprise support).
// Priority order:
//  1. flagToken (explicit --token flag)
//  2. GITHUB_TOKEN environment variable
//  3. GH_TOKEN environment variable
//  4. gh CLI auth for the specific host
func ResolveGitHubTokenForHost(flagToken, host string) (token string, source TokenSource, err error) {
	// 1. Flag has highest priority
	if flagToken != "" {
		return flagToken, TokenSourceFlag, nil
	}

	// 2. Check environment variables
	if token = os.Getenv("GITHUB_TOKEN"); token != "" {
		return token, TokenSourceEnvGitHub, nil
	}
	if token = os.Getenv("GH_TOKEN"); token != "" {
		return token, TokenSourceEnvGH, nil
	}

	// 3. Try gh CLI auth for specific host
	if token, _ = auth.TokenForHost(host); token != "" {
		return token, TokenSourceGHCLI, nil
	}

	return "", TokenSourceNone, fmt.Errorf("no GitHub token found for host: %s", host)
}
