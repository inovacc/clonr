package common

import "strings"

// SanitizeGitURL removes credentials from a git URL
func SanitizeGitURL(rawURL string) string {
	// Handle URLs with embedded credentials like:
	// https://user:token@github.com/owner/repo.git
	if strings.Contains(rawURL, "@") && strings.Contains(rawURL, "://") {
		// Split at ://
		parts := strings.SplitN(rawURL, "://", 2)
		if len(parts) == 2 {
			// Find @ and remove everything before it
			atIdx := strings.Index(parts[1], "@")
			if atIdx != -1 {
				return parts[0] + "://" + parts[1][atIdx+1:]
			}
		}
	}

	return rawURL
}
