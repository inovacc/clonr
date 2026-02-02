package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/go-github/v82/github"
	"github.com/inovacc/clonr/internal/application"
	"github.com/inovacc/clonr/internal/client/grpc"
)

// Organization represents a GitHub organization with local mirror status
type Organization struct {
	Login       string
	Name        string
	Description string
	URL         string
	RepoCount   int
	IsMirrored  bool
	MirrorPath  string
	LocalRepos  int
}

// ListOrganizationsOptions configures the organization listing
type ListOrganizationsOptions struct {
	IncludeUser bool // Include user's personal repos as pseudo-org
}

// ListOrganizations fetches the user's GitHub organizations and checks mirror status
func ListOrganizations(token string, opts ListOrganizationsOptions) ([]Organization, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create a GitHub client
	client := NewGitHubClient(ctx, token)

	// Get user's organizations
	ghOrgs, _, err := client.Organizations.List(ctx, "", &github.ListOptions{PerPage: 100})
	if err != nil {
		return nil, fmt.Errorf("failed to list organizations: %w", err)
	}

	// Get a clone directory from config
	cloneDir, err := getCloneDir()
	if err != nil {
		return nil, err
	}

	var orgs = make([]Organization, 0)

	// Optionally include user's personal repos
	if opts.IncludeUser {
		user, _, err := client.Users.Get(ctx, "")
		if err == nil && user.Login != nil {
			userOrg := Organization{
				Login:       *user.Login,
				Name:        getUserName(user),
				Description: "Personal repositories",
				URL:         fmt.Sprintf("https://github.com/%s", *user.Login),
			}

			// Get user's repo count
			if user.PublicRepos != nil {
				userOrg.RepoCount = *user.PublicRepos
			}

			if user.TotalPrivateRepos != nil {
				userOrg.RepoCount += int(*user.TotalPrivateRepos)
			}

			// Check mirror status
			userOrg.MirrorPath = filepath.Join(cloneDir, *user.Login)
			userOrg.IsMirrored, userOrg.LocalRepos = checkMirrorStatus(userOrg.MirrorPath)

			orgs = append(orgs, userOrg)
		}
	}

	// Process organizations
	for _, ghOrg := range ghOrgs {
		org := Organization{
			Login: safeString(ghOrg.Login),
			URL:   fmt.Sprintf("https://github.com/%s", safeString(ghOrg.Login)),
		}

		// Fetch full org details to get repo count and description
		fullOrg, _, err := client.Organizations.Get(ctx, org.Login)
		if err == nil {
			org.Name = safeString(fullOrg.Name)

			org.Description = safeString(fullOrg.Description)
			if fullOrg.PublicRepos != nil {
				org.RepoCount = *fullOrg.PublicRepos
			}

			if fullOrg.TotalPrivateRepos != nil {
				org.RepoCount += int(*fullOrg.TotalPrivateRepos)
			}
		}

		if org.Name == "" {
			org.Name = org.Login
		}

		// Check mirror status
		org.MirrorPath = filepath.Join(cloneDir, org.Login)
		org.IsMirrored, org.LocalRepos = checkMirrorStatus(org.MirrorPath)

		orgs = append(orgs, org)
	}

	return orgs, nil
}

// getCloneDir retrieves the clone directory from config
func getCloneDir() (string, error) {
	client, err := grpc.GetClient()
	if err != nil {
		// Fallback to default if server not running
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}

		return filepath.Join(home, application.AppName), nil
	}

	cfg, err := client.GetConfig()
	if err != nil {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}

		return filepath.Join(home, application.AppName), nil
	}

	return cfg.DefaultCloneDir, nil
}

// checkMirrorStatus checks if an organization has been mirrored locally
func checkMirrorStatus(path string) (bool, int) {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return false, 0
	}

	// Count subdirectories (repos)
	entries, err := os.ReadDir(path)
	if err != nil {
		return true, 0
	}

	count := 0

	for _, entry := range entries {
		if entry.IsDir() {
			// Check if it's a git repo (include repos starting with dot like .github)
			gitPath := filepath.Join(path, entry.Name(), ".git")
			if _, err := os.Stat(gitPath); err == nil {
				count++
			}
		}
	}

	return count > 0, count
}

// safeString safely dereferences a string pointer
func safeString(s *string) string {
	if s == nil {
		return ""
	}

	return *s
}

// getUserName gets display name for user
func getUserName(user *github.User) string {
	if user.Name != nil && *user.Name != "" {
		return *user.Name
	}

	if user.Login != nil {
		return *user.Login
	}

	return "Unknown"
}
