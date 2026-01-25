package core

import (
	"testing"
)

func TestTokenSource_String(t *testing.T) {
	tests := []struct {
		source TokenSource
		want   string
	}{
		{TokenSourceFlag, "flag"},
		{TokenSourceEnvGitHub, "GITHUB_TOKEN"},
		{TokenSourceEnvGH, "GH_TOKEN"},
		{TokenSourceGHCLI, "gh-cli"},
		{TokenSourceNone, "none"},
	}

	for _, tt := range tests {
		t.Run(string(tt.source), func(t *testing.T) {
			if string(tt.source) != tt.want {
				t.Errorf("TokenSource = %q, want %q", tt.source, tt.want)
			}
		})
	}
}

func TestResolveGitHubToken_FlagPriority(t *testing.T) {
	// Flag should have highest priority
	flagToken := "test-flag-token"

	token, source, err := ResolveGitHubToken(flagToken, "")
	if err != nil {
		t.Fatalf("ResolveGitHubToken() error = %v", err)
	}

	if token != flagToken {
		t.Errorf("token = %q, want %q", token, flagToken)
	}

	if source != TokenSourceFlag {
		t.Errorf("source = %v, want %v", source, TokenSourceFlag)
	}
}

func TestResolveGitHubToken_EnvGitHub(t *testing.T) {
	testToken := "test-github-token"
	t.Setenv("GITHUB_TOKEN", testToken)
	t.Setenv("GH_TOKEN", "")

	token, source, err := ResolveGitHubToken("", "")
	if err != nil {
		t.Fatalf("ResolveGitHubToken() error = %v", err)
	}

	if token != testToken {
		t.Errorf("token = %q, want %q", token, testToken)
	}

	if source != TokenSourceEnvGitHub {
		t.Errorf("source = %v, want %v", source, TokenSourceEnvGitHub)
	}
}

func TestResolveGitHubToken_EnvGH(t *testing.T) {
	testToken := "test-gh-token"

	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GH_TOKEN", testToken)

	token, source, err := ResolveGitHubToken("", "")
	if err != nil {
		t.Fatalf("ResolveGitHubToken() error = %v", err)
	}

	if token != testToken {
		t.Errorf("token = %q, want %q", token, testToken)
	}

	if source != TokenSourceEnvGH {
		t.Errorf("source = %v, want %v", source, TokenSourceEnvGH)
	}
}

func TestResolveGitHubToken_NoToken(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GH_TOKEN", "")

	token, source, err := ResolveGitHubToken("", "")

	// If no token from gh CLI, should return error
	if token == "" && source == TokenSourceNone {
		if err == nil {
			t.Error("ResolveGitHubToken() should return error when no token available")
		}
	}
}

func TestResolveGitHubTokenForHost_FlagPriority(t *testing.T) {
	flagToken := "test-host-flag-token"

	token, source, err := ResolveGitHubTokenForHost(flagToken, "", "github.example.com")
	if err != nil {
		t.Fatalf("ResolveGitHubTokenForHost() error = %v", err)
	}

	if token != flagToken {
		t.Errorf("token = %q, want %q", token, flagToken)
	}

	if source != TokenSourceFlag {
		t.Errorf("source = %v, want %v", source, TokenSourceFlag)
	}
}

func TestResolveGitHubTokenForHost_EnvVars(t *testing.T) {
	testToken := "test-env-token"
	t.Setenv("GITHUB_TOKEN", testToken)
	t.Setenv("GH_TOKEN", "")

	token, source, err := ResolveGitHubTokenForHost("", "", "github.example.com")
	if err != nil {
		t.Fatalf("ResolveGitHubTokenForHost() error = %v", err)
	}

	if token != testToken {
		t.Errorf("token = %q, want %q", token, testToken)
	}

	if source != TokenSourceEnvGitHub {
		t.Errorf("source = %v, want %v", source, TokenSourceEnvGitHub)
	}
}

func TestResolveGitHubTokenForHost_GHToken(t *testing.T) {
	testToken := "test-gh-env-token"

	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GH_TOKEN", testToken)

	token, source, err := ResolveGitHubTokenForHost("", "", "github.example.com")
	if err != nil {
		t.Fatalf("ResolveGitHubTokenForHost() error = %v", err)
	}

	if token != testToken {
		t.Errorf("token = %q, want %q", token, testToken)
	}

	if source != TokenSourceEnvGH {
		t.Errorf("source = %v, want %v", source, TokenSourceEnvGH)
	}
}

func TestResolveGitHubTokenForHost_NoToken(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GH_TOKEN", "")

	token, source, err := ResolveGitHubTokenForHost("", "", "nonexistent.host.example.com")

	// If no token available, should return error
	if token == "" && source == TokenSourceNone {
		if err == nil {
			t.Error("ResolveGitHubTokenForHost() should return error when no token available")
		}
	}
}

func TestTokenSourceConstants(t *testing.T) {
	// Verify TokenSource constants are correct
	if TokenSourceFlag != "flag" {
		t.Errorf("TokenSourceFlag = %q, want %q", TokenSourceFlag, "flag")
	}

	if TokenSourceEnvGitHub != "GITHUB_TOKEN" {
		t.Errorf("TokenSourceEnvGitHub = %q, want %q", TokenSourceEnvGitHub, "GITHUB_TOKEN")
	}

	if TokenSourceEnvGH != "GH_TOKEN" {
		t.Errorf("TokenSourceEnvGH = %q, want %q", TokenSourceEnvGH, "GH_TOKEN")
	}

	if TokenSourceGHCLI != "gh-cli" {
		t.Errorf("TokenSourceGHCLI = %q, want %q", TokenSourceGHCLI, "gh-cli")
	}

	if TokenSourceNone != "none" {
		t.Errorf("TokenSourceNone = %q, want %q", TokenSourceNone, "none")
	}
}
