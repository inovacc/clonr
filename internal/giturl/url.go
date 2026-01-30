package giturl

import (
	"net/url"
	"strings"
)

// IsURL checks if the given string is a git URL
func IsURL(u string) bool {
	return strings.HasPrefix(u, "git@") || isSupportedProtocol(u)
}

func isSupportedProtocol(u string) bool {
	return strings.HasPrefix(u, "ssh:") ||
		strings.HasPrefix(u, "git+ssh:") ||
		strings.HasPrefix(u, "git:") ||
		strings.HasPrefix(u, "http:") ||
		strings.HasPrefix(u, "git+https:") ||
		strings.HasPrefix(u, "https:")
}

func isPossibleProtocol(u string) bool {
	return isSupportedProtocol(u) ||
		strings.HasPrefix(u, "ftp:") ||
		strings.HasPrefix(u, "ftps:") ||
		strings.HasPrefix(u, "file:")
}

// Parse normalizes git remote urls, including scp-like syntax (git@github.com:owner/repo)
func Parse(rawURL string) (*url.URL, error) {
	if !isPossibleProtocol(rawURL) &&
		strings.ContainsRune(rawURL, ':') &&
		// not a Windows path
		!strings.ContainsRune(rawURL, '\\') {
		// support scp-like syntax for ssh protocol
		rawURL = "ssh://" + strings.Replace(rawURL, ":", "/", 1)
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	switch u.Scheme {
	case "git+https":
		u.Scheme = "https"
	case "git+ssh":
		u.Scheme = "ssh"
	}

	if u.Scheme != "ssh" {
		return u, nil
	}

	if strings.HasPrefix(u.Path, "//") {
		u.Path = strings.TrimPrefix(u.Path, "/")
	}

	u.Host = strings.TrimSuffix(u.Host, ":"+u.Port())

	return u, nil
}

// Simplify strips given URL of extra parts like extra path segments (i.e.,
// anything beyond `/owner/repo`), query strings, or fragments.
//
// This allows cloning from:
//   - (Tree)              github.com/owner/repo/blob/main/foo/bar
//   - (Deep-link to line) github.com/owner/repo/blob/main/foo/bar#L168
//   - (Issue/PR comment)  github.com/owner/repo/pull/999#issue-9999999999
//   - (Commit history)    github.com/owner/repo/commits/main/?author=foo
func Simplify(u *url.URL) *url.URL {
	result := &url.URL{
		Scheme: u.Scheme,
		User:   u.User,
		Host:   u.Host,
		Path:   u.Path,
	}

	pathParts := strings.SplitN(strings.Trim(u.Path, "/"), "/", 3)
	if len(pathParts) <= 2 {
		return result
	}

	result.Path = "/" + strings.Join(pathParts[0:2], "/")

	return result
}

// ExtractOwnerRepo extracts owner and repo name from a URL
func ExtractOwnerRepo(u *url.URL) (owner, repo string, err error) {
	parts := strings.SplitN(strings.Trim(u.Path, "/"), "/", 3)
	if len(parts) < 2 {
		return "", "", &url.Error{Op: "parse", URL: u.String(), Err: errInvalidPath}
	}

	owner = parts[0]
	repo = strings.TrimSuffix(parts[1], ".git")

	return owner, repo, nil
}

var errInvalidPath = &invalidPathError{}

type invalidPathError struct{}

func (e *invalidPathError) Error() string {
	return "invalid path: expected owner/repo"
}
