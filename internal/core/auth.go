package core

import (
	"fmt"
	"os"

	"github.com/cli/go-gh/v2/pkg/auth"
	"github.com/inovacc/clonr/internal/client/grpc"
	"github.com/inovacc/clonr/internal/crypto/tpm"
	"github.com/inovacc/clonr/internal/model"
)

// TokenSource indicates where the token was found
type TokenSource string

const (
	TokenSourceFlag      TokenSource = "flag"
	TokenSourceProfile   TokenSource = "profile"
	TokenSourceEnvGitHub TokenSource = "GITHUB_TOKEN"
	TokenSourceEnvGH     TokenSource = "GH_TOKEN"
	TokenSourceGHCLI     TokenSource = "gh-cli"
	TokenSourceNone      TokenSource = "none"
)

// ResolveGitHubToken attempts to find a GitHub token from multiple sources.
// Priority order:
//  1. flagToken (explicit --token flag)
//  2. profileName (explicit --profile flag)
//  3. GITHUB_TOKEN environment variable
//  4. GH_TOKEN environment variable
//  5. Active clonr profile token
//  6. gh CLI auth (config file)
func ResolveGitHubToken(flagToken, profileName string) (token string, source TokenSource, err error) {
	return ResolveGitHubTokenForHost(flagToken, profileName, "github.com")
}

// ResolveGitHubTokenForHost resolves token for a specific host (enterprise support).
// Priority order:
//  1. flagToken (explicit --token flag)
//  2. profileName (explicit --profile flag)
//  3. GITHUB_TOKEN environment variable
//  4. GH_TOKEN environment variable
//  5. Active clonr profile token
//  6. gh CLI auth for the specific host
func ResolveGitHubTokenForHost(flagToken, profileName, host string) (token string, source TokenSource, err error) {
	// 1. Flag has highest priority
	if flagToken != "" {
		return flagToken, TokenSourceFlag, nil
	}

	// 2. Explicit profile flag
	if profileName != "" {
		token, err = getProfileToken(profileName, host)
		if err == nil && token != "" {
			return token, TokenSourceProfile, nil
		}
		// If profile specified but token retrieval failed, return error
		if err != nil {
			return "", TokenSourceNone, fmt.Errorf("failed to get token from profile '%s': %w", profileName, err)
		}
	}

	// 3. Check GITHUB_TOKEN env var
	if token = os.Getenv("GITHUB_TOKEN"); token != "" {
		return token, TokenSourceEnvGitHub, nil
	}

	// 4. Check GH_TOKEN env var
	if token = os.Getenv("GH_TOKEN"); token != "" {
		return token, TokenSourceEnvGH, nil
	}

	// 5. Try active clonr profile
	token, err = getActiveProfileToken(host)
	if err == nil && token != "" {
		return token, TokenSourceProfile, nil
	}

	// 6. Try gh CLI auth (keyring + config file)
	if token, _ = auth.TokenForHost(host); token != "" {
		return token, TokenSourceGHCLI, nil
	}

	// No token found
	return "", TokenSourceNone, fmt.Errorf(`GitHub token required

Provide a token via one of:
  * clonr profile add <name>  (recommended - creates a profile with OAuth)
  * gh auth login             (auto-detected from gh CLI)
  * GITHUB_TOKEN env var
  * --token flag

Create a token at: https://github.com/settings/tokens`)
}

// getProfileToken retrieves a token from a specific profile
func getProfileToken(profileName, host string) (string, error) {
	client, err := grpc.GetClient()
	if err != nil {
		return "", err
	}

	profile, err := client.GetProfile(profileName)
	if err != nil {
		return "", err
	}

	if profile == nil {
		return "", ErrProfileNotFound
	}

	// Check host matches
	if host != "" && host != "github.com" && profile.Host != host {
		return "", fmt.Errorf("profile '%s' is for host '%s', not '%s'", profileName, profile.Host, host)
	}

	return tokenFromProfile(profile)
}

// getActiveProfileToken retrieves the token from the active profile
func getActiveProfileToken(host string) (string, error) {
	client, err := grpc.GetClient()
	if err != nil {
		// Server not running, skip profile token
		return "", nil //nolint:nilerr // intentionally ignore error when server not running
	}

	profile, err := client.GetActiveProfile()
	if err != nil {
		return "", nil //nolint:nilerr // intentionally ignore error to fallback to other auth methods
	}

	if profile == nil {
		return "", nil
	}

	// Check host matches (for enterprise support)
	if host != "" && host != "github.com" && profile.Host != host {
		return "", nil
	}

	return tokenFromProfile(profile)
}

// tokenFromProfile decrypts and retrieves token from profile
func tokenFromProfile(profile *model.Profile) (string, error) {
	if profile == nil {
		return "", ErrTokenNotFound
	}

	if len(profile.EncryptedToken) == 0 {
		return "", ErrTokenNotFound
	}

	return tpm.DecryptToken(profile.EncryptedToken, profile.Name, profile.Host)
}
