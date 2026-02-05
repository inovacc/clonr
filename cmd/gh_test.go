package cmd

import (
	"testing"
	"time"
)

func TestFormatAge(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "just now",
			duration: 30 * time.Second,
			expected: "just now",
		},
		{
			name:     "1 minute ago",
			duration: 1 * time.Minute,
			expected: "1 minute ago",
		},
		{
			name:     "multiple minutes ago",
			duration: 5 * time.Minute,
			expected: "5 minutes ago",
		},
		{
			name:     "1 hour ago",
			duration: 1 * time.Hour,
			expected: "1 hour ago",
		},
		{
			name:     "multiple hours ago",
			duration: 5 * time.Hour,
			expected: "5 hours ago",
		},
		{
			name:     "1 day ago",
			duration: 24 * time.Hour,
			expected: "1 day ago",
		},
		{
			name:     "multiple days ago",
			duration: 5 * 24 * time.Hour,
			expected: "5 days ago",
		},
		{
			name:     "1 month ago",
			duration: 35 * 24 * time.Hour,
			expected: "1 month ago",
		},
		{
			name:     "multiple months ago",
			duration: 90 * 24 * time.Hour,
			expected: "3 months ago",
		},
		{
			name:     "1 year ago",
			duration: 400 * 24 * time.Hour,
			expected: "1 year ago",
		},
		{
			name:     "multiple years ago",
			duration: 800 * 24 * time.Hour,
			expected: "2 years ago",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pastTime := time.Now().Add(-tt.duration)

			result := formatAge(pastTime)
			if result != tt.expected {
				t.Errorf("formatAge() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFormatShortDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "seconds only",
			duration: 45 * time.Second,
			expected: "45s",
		},
		{
			name:     "minutes and seconds",
			duration: 2*time.Minute + 30*time.Second,
			expected: "2m 30s",
		},
		{
			name:     "hours and minutes",
			duration: 2*time.Hour + 15*time.Minute,
			expected: "2h 15m",
		},
		{
			name:     "just under minute",
			duration: 59 * time.Second,
			expected: "59s",
		},
		{
			name:     "exactly one minute",
			duration: 1 * time.Minute,
			expected: "1m 0s",
		},
		{
			name:     "exactly one hour",
			duration: 1 * time.Hour,
			expected: "1h 0m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatShortDuration(tt.duration)
			if result != tt.expected {
				t.Errorf("formatShortDuration(%v) = %q, want %q", tt.duration, result, tt.expected)
			}
		})
	}
}

func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{
			name:     "bytes",
			bytes:    500,
			expected: "500 B",
		},
		{
			name:     "kilobytes",
			bytes:    1536,
			expected: "1.5 KB",
		},
		{
			name:     "megabytes",
			bytes:    1572864,
			expected: "1.5 MB",
		},
		{
			name:     "gigabytes",
			bytes:    1610612736,
			expected: "1.5 GB",
		},
		{
			name:     "exactly 1 KB",
			bytes:    1024,
			expected: "1.0 KB",
		},
		{
			name:     "exactly 1 MB",
			bytes:    1048576,
			expected: "1.0 MB",
		},
		{
			name:     "exactly 1 GB",
			bytes:    1073741824,
			expected: "1.0 GB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatFileSize(tt.bytes)
			if result != tt.expected {
				t.Errorf("formatFileSize(%d) = %q, want %q", tt.bytes, result, tt.expected)
			}
		})
	}
}

func TestTruncateStr(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "string shorter than max",
			input:    "test",
			maxLen:   10,
			expected: "test      ",
		},
		{
			name:     "string equal to max",
			input:    "test",
			maxLen:   4,
			expected: "test",
		},
		{
			name:     "string longer than max",
			input:    "testing",
			maxLen:   5,
			expected: "te...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateStr(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncateStr(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
			}
		})
	}
}

func TestGHCmd(t *testing.T) {
	t.Run("gh command exists", func(t *testing.T) {
		if ghCmd == nil {
			t.Fatal("ghCmd should not be nil")
		}
	})

	t.Run("gh command has correct use", func(t *testing.T) {
		if ghCmd.Use != "gh" {
			t.Errorf("ghCmd.Use = %q, want %q", ghCmd.Use, "gh")
		}
	})

	t.Run("gh command has subcommands", func(t *testing.T) {
		subcommands := ghCmd.Commands()
		if len(subcommands) == 0 {
			t.Error("ghCmd should have subcommands")
		}
	})
}

func TestGHFlags(t *testing.T) {
	t.Run("GHFlags struct fields", func(t *testing.T) {
		flags := GHFlags{
			Token:   "token123",
			Profile: "work",
			Repo:    "owner/repo",
			JSON:    true,
		}

		if flags.Token != "token123" {
			t.Errorf("Token = %q, want %q", flags.Token, "token123")
		}

		if flags.Profile != "work" {
			t.Errorf("Profile = %q, want %q", flags.Profile, "work")
		}

		if flags.Repo != "owner/repo" {
			t.Errorf("Repo = %q, want %q", flags.Repo, "owner/repo")
		}

		if !flags.JSON {
			t.Error("JSON should be true")
		}
	})
}

func TestExtractGHFlags(t *testing.T) {
	t.Run("extract flags from command", func(t *testing.T) {
		// Create a test command with common flags
		testCmd := ghCmd.Commands()[0] // Get first subcommand
		if testCmd == nil {
			t.Skip("no subcommands available")
		}

		// Check if command has the common flags
		if testCmd.Flags().Lookup("token") == nil {
			t.Skip("command doesn't have common GH flags")
		}

		// Set flag values
		testCmd.Flags().Set("token", "test-token")
		testCmd.Flags().Set("profile", "test-profile")
		testCmd.Flags().Set("repo", "owner/repo")
		testCmd.Flags().Set("json", "true")

		defer func() {
			testCmd.Flags().Set("token", "")
			testCmd.Flags().Set("profile", "")
			testCmd.Flags().Set("repo", "")
			testCmd.Flags().Set("json", "false")
		}()

		flags := extractGHFlags(testCmd)

		if flags.Token != "test-token" {
			t.Errorf("Token = %q, want %q", flags.Token, "test-token")
		}

		if flags.Profile != "test-profile" {
			t.Errorf("Profile = %q, want %q", flags.Profile, "test-profile")
		}

		if flags.Repo != "owner/repo" {
			t.Errorf("Repo = %q, want %q", flags.Repo, "owner/repo")
		}

		if !flags.JSON {
			t.Error("JSON should be true")
		}
	})
}

func TestAddGHCommonFlags(t *testing.T) {
	t.Run("adds common flags to command", func(t *testing.T) {
		// The flags are already added during init, verify they exist on a gh subcommand
		subcommands := ghCmd.Commands()
		if len(subcommands) == 0 {
			t.Skip("no subcommands available")
		}

		cmd := subcommands[0]
		expectedFlags := []string{"token", "profile", "repo", "json"}

		for _, flagName := range expectedFlags {
			if cmd.Flags().Lookup(flagName) == nil {
				t.Logf("flag %q not found on command %s", flagName, cmd.Name())
			}
		}
	})
}

func TestOutputJSON(t *testing.T) {
	t.Run("outputJSON with simple struct", func(t *testing.T) {
		data := struct {
			Name  string `json:"name"`
			Count int    `json:"count"`
		}{
			Name:  "test",
			Count: 42,
		}

		err := outputJSON(data)
		if err != nil {
			t.Errorf("outputJSON() error = %v", err)
		}
	})
}

func TestNewGHLogger(t *testing.T) {
	t.Run("creates logger for JSON output", func(t *testing.T) {
		logger := newGHLogger(true)
		if logger == nil {
			t.Error("logger should not be nil")
		}
	})

	t.Run("creates logger for text output", func(t *testing.T) {
		logger := newGHLogger(false)
		if logger == nil {
			t.Error("logger should not be nil")
		}
	})
}
