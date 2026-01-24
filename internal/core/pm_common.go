package core

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// DetectJiraProject attempts to determine the Jira project key.
// Priority:
//  1. Explicit argument (PROJ or PROJ-123)
//  2. --project flag value
//  3. Config file default project
//  4. Error with guidance
func DetectJiraProject(arg, projectFlag string) (projectKey string, err error) {
	// 1. Check explicit argument
	if arg != "" {
		key := extractJiraProjectKey(arg)
		if key != "" {
			return key, nil
		}

		return "", fmt.Errorf("invalid Jira project key format: %s\n\nExpected format: PROJ or PROJ-123", arg)
	}

	// 2. Check flag
	if projectFlag != "" {
		key := extractJiraProjectKey(projectFlag)
		if key != "" {
			return key, nil
		}

		return "", fmt.Errorf("invalid Jira project key format: %s", projectFlag)
	}

	// 3. Try config file default
	defaultProject, err := GetJiraDefaultProject("")
	if err == nil && defaultProject != "" {
		return defaultProject, nil
	}

	// 4. No project found
	return "", fmt.Errorf(`Jira project key required

Specify a project with:
  * clonr pm jira issues list PROJ
  * clonr pm jira issues list --project PROJ

Or set a default project in ~/.config/clonr/jira.json`)
}

// extractJiraProjectKey extracts the project key from various formats
// Supports: PROJ, PROJ-123, proj, proj-123
func extractJiraProjectKey(s string) string {
	s = strings.TrimSpace(s)

	// Pattern: PROJECT-123 or PROJECT
	pattern := regexp.MustCompile(`^([A-Za-z][A-Za-z0-9]*)(?:-\d+)?$`)
	matches := pattern.FindStringSubmatch(s)

	if len(matches) >= 2 {
		return strings.ToUpper(matches[1])
	}

	return ""
}

// ExtractJiraIssueKey extracts a full issue key (PROJ-123) from input
func ExtractJiraIssueKey(s string) (string, error) {
	s = strings.TrimSpace(s)

	// Pattern: PROJECT-123
	pattern := regexp.MustCompile(`^([A-Za-z][A-Za-z0-9]*)-(\d+)$`)
	matches := pattern.FindStringSubmatch(s)

	if len(matches) == 3 {
		return strings.ToUpper(matches[1]) + "-" + matches[2], nil
	}

	return "", fmt.Errorf("invalid issue key format: %s\n\nExpected format: PROJ-123", s)
}

// DetectZenHubRepo attempts to determine the GitHub repository for ZenHub.
// Priority:
//  1. Explicit argument (owner/repo)
//  2. --repo flag value
//  3. Current directory's git config
//  4. Error with guidance
func DetectZenHubRepo(arg, repoFlag string) (owner, repo string, err error) {
	// Use existing DetectRepository function
	return DetectRepository(arg, repoFlag)
}

// FormatAge formats a time as a human-readable age string
func FormatAge(t time.Time) string {
	d := time.Since(t)

	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		mins := int(d.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}

		return fmt.Sprintf("%d minutes ago", mins)
	case d < 24*time.Hour:
		hours := int(d.Hours())
		if hours == 1 {
			return "1 hour ago"
		}

		return fmt.Sprintf("%d hours ago", hours)
	case d < 30*24*time.Hour:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}

		return fmt.Sprintf("%d days ago", days)
	case d < 365*24*time.Hour:
		months := int(d.Hours() / 24 / 30)
		if months == 1 {
			return "1 month ago"
		}

		return fmt.Sprintf("%d months ago", months)
	default:
		years := int(d.Hours() / 24 / 365)
		if years == 1 {
			return "1 year ago"
		}

		return fmt.Sprintf("%d years ago", years)
	}
}

// FormatDuration formats a duration in a human-readable way
func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}

	if d < time.Hour {
		mins := int(d.Minutes())
		secs := int(d.Seconds()) % 60
		if secs > 0 {
			return fmt.Sprintf("%dm %ds", mins, secs)
		}

		return fmt.Sprintf("%dm", mins)
	}

	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60
	if mins > 0 {
		return fmt.Sprintf("%dh %dm", hours, mins)
	}

	return fmt.Sprintf("%dh", hours)
}

// GetCurrentGitBranch returns the current git branch name
func GetCurrentGitBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not in a git repository or git not installed")
	}

	return strings.TrimSpace(string(output)), nil
}

// GetCurrentWorkingDirectory returns the current working directory
func GetCurrentWorkingDirectory() (string, error) {
	return os.Getwd()
}

// IsJiraProjectKey checks if a string is a valid Jira project key
func IsJiraProjectKey(s string) bool {
	pattern := regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]*$`)
	return pattern.MatchString(s)
}

// IsJiraIssueKey checks if a string is a valid Jira issue key
func IsJiraIssueKey(s string) bool {
	pattern := regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]*-\d+$`)
	return pattern.MatchString(s)
}

// TruncateString truncates a string to a maximum length with ellipsis
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	if maxLen <= 3 {
		return s[:maxLen]
	}

	return s[:maxLen-3] + "..."
}
