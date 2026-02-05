package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/inovacc/clonr/internal/model"
)

// formatTokenStorage returns a human-readable string for the token storage type
func formatTokenStorage(ts model.TokenStorage) string {
	switch ts {
	case model.TokenStorageEncrypted:
		return "encrypted (TPM)"
	case model.TokenStorageOpen:
		return "plain text"
	default:
		return string(ts)
	}
}

// promptConfirm asks the user for confirmation and returns true if they confirm
// prompt should include the question (e.g., "Delete this file? [y/N]: ")
func promptConfirm(prompt string) bool {
	_, _ = fmt.Fprint(os.Stdout, prompt)

	var response string

	_, _ = fmt.Scanln(&response)

	return response == "y" || response == "Y"
}

// expandPath expands ~ to the user's home directory and returns an absolute path
func expandPath(path string) (string, error) {
	if len(path) == 0 {
		return "", fmt.Errorf("path is empty")
	}

	// Expand ~ to home directory
	if path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}

		path = filepath.Join(home, path[1:])
	}

	// Make path absolute
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to resolve path: %w", err)
	}

	return absPath, nil
}

// printEmptyResult prints a "no results" message with a create hint
// resourceType: "profiles", "workspaces", etc.
// createCmd: the command to create the resource
func printEmptyResult(resourceType, createCmd string) {
	_, _ = fmt.Fprintf(os.Stdout, "No %s configured.\n", resourceType)
	_, _ = fmt.Fprintf(os.Stdout, "Create one with: %s\n", createCmd)
}

// centerString centers a string in a field of given width
func centerString(s string, width int) string {
	if len(s) >= width {
		return s
	}

	padding := (width - len(s)) / 2

	return fmt.Sprintf("%*s%s%*s", padding, "", s, width-len(s)-padding, "")
}

// truncateString truncates a string to the specified length with ellipsis
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	if maxLen <= 3 {
		return s[:maxLen]
	}

	return s[:maxLen-3] + "..."
}

// boxWidth is the standard width for info boxes
const boxWidth = 64

// printBoxHeader prints the top border of an info box with a title
func printBoxHeader(title string) {
	_, _ = fmt.Fprintln(os.Stdout, "╔══════════════════════════════════════════════════════════════╗")
	_, _ = fmt.Fprintf(os.Stdout, "║%s║\n", centerString(title, boxWidth-2))
	_, _ = fmt.Fprintln(os.Stdout, "╠══════════════════════════════════════════════════════════════╣")
}

// printBoxLine prints a line inside an info box with label and value
func printBoxLine(label, value string) {
	content := fmt.Sprintf("  %s: %s", label, value)

	padding := boxWidth - 2 - len(content)
	if padding < 0 {
		padding = 0
		content = content[:boxWidth-2]
	}

	_, _ = fmt.Fprintf(os.Stdout, "║%s%*s║\n", content, padding, "")
}

// printBoxFooter prints the bottom border of an info box
func printBoxFooter() {
	_, _ = fmt.Fprintln(os.Stdout, "╚══════════════════════════════════════════════════════════════╝")
}

// printInfoBox prints a complete info box with title and key-value pairs
func printInfoBox(title string, items map[string]string, order []string) {
	printBoxHeader(title)

	for _, key := range order {
		if val, ok := items[key]; ok {
			printBoxLine(key, val)
		}
	}

	printBoxFooter()
}
