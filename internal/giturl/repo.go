package giturl

import (
	"fmt"
	"net/url"
	"strings"
)

const defaultHost = "github.com"

// Repository represents a Git repository with owner, name, and host
type Repository struct {
	Owner string
	Name  string
	Host  string
}

// CloneURL returns the clone URL for the repository using the specified protocol
func (r *Repository) CloneURL(protocol string) string {
	if protocol == "ssh" {
		return fmt.Sprintf("git@%s:%s/%s.git", r.Host, r.Owner, r.Name)
	}

	return fmt.Sprintf("https://%s/%s/%s.git", r.Host, r.Owner, r.Name)
}

// FullName returns the "owner/repo" string
func (r *Repository) FullName() string {
	return fmt.Sprintf("%s/%s", r.Owner, r.Name)
}

// ParseRepository parses a repository string into a Repository struct.
// Supports multiple formats:
//   - "repo" (requires currentUser to resolve to currentUser/repo)
//   - "owner/repo"
//   - "https://github.com/owner/repo"
//   - "https://github.com/owner/repo/blob/main/file.go#L10"
//   - "git@github.com:owner/repo.git"
//   - "ssh://git@github.com/owner/repo.git"
func ParseRepository(arg string, currentUser string) (*Repository, error) {
	// Check if it's a URL (contains ":" but not a Windows path)
	isURL := strings.Contains(arg, ":") && !strings.Contains(arg, "\\")

	if isURL {
		return parseRepositoryFromURL(arg)
	}

	// Check if it's owner/repo format
	if strings.Contains(arg, "/") {
		return parseRepositoryFromFullName(arg)
	}

	// It's just a repo name, need currentUser
	if currentUser == "" {
		return nil, fmt.Errorf("cannot resolve repository %q: no authenticated user", arg)
	}

	return &Repository{
		Owner: currentUser,
		Name:  arg,
		Host:  defaultHost,
	}, nil
}

func parseRepositoryFromURL(rawURL string) (*Repository, error) {
	u, err := Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL %q: %w", rawURL, err)
	}

	// Simplify the URL to strip extra path segments
	u = Simplify(u)

	owner, name, err := ExtractOwnerRepo(u)
	if err != nil {
		return nil, fmt.Errorf("invalid repository URL %q: %w", rawURL, err)
	}

	host := u.Hostname()
	if host == "" {
		host = defaultHost
	}

	return &Repository{
		Owner: owner,
		Name:  name,
		Host:  strings.ToLower(strings.TrimPrefix(host, "www.")),
	}, nil
}

func parseRepositoryFromFullName(fullName string) (*Repository, error) {
	// Handle HOST/OWNER/REPO format
	parts := strings.Split(fullName, "/")
	switch len(parts) {
	case 2:
		return &Repository{
			Owner: parts[0],
			Name:  parts[1],
			Host:  defaultHost,
		}, nil
	case 3:
		return &Repository{
			Owner: parts[1],
			Name:  parts[2],
			Host:  strings.ToLower(strings.TrimPrefix(parts[0], "www.")),
		}, nil
	default:
		return nil, fmt.Errorf("invalid repository format %q: expected owner/repo or host/owner/repo", fullName)
	}
}

// BuildCloneURL builds a clone URL from the given repository argument
func BuildCloneURL(arg string, currentUser string) (*url.URL, error) {
	repo, err := ParseRepository(arg, currentUser)
	if err != nil {
		return nil, err
	}

	return url.Parse(repo.CloneURL("https"))
}
