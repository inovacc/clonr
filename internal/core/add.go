package core

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/inovacc/clonr/internal/grpcclient"
)

// AddOptions holds optional parameters for adding a repo.
type AddOptions struct {
	Yes  bool   // skip confirmation (handled at CLI level)
	Name string // reserved for future use
}

// AddRepo validates the path is a git repo and registers it in the DB if not present.
func AddRepo(path string, _ AddOptions) (string, error) {
	if path == "" {
		return "", errors.New("path is required")
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("cannot resolve path: %w", err)
	}

	fi, err := os.Stat(abs)
	if err != nil || !fi.IsDir() {
		return "", fmt.Errorf("path does not exist or is not a directory: %s", abs)
	}

	gitDir := filepath.Join(abs, ".git")
	if s, err := os.Stat(gitDir); err != nil || !s.IsDir() {
		return "", fmt.Errorf("not a git repository (missing .git): %s", abs)
	}

	// Try to detect remote.origin.url; it's optional.
	var remote *url.URL

	if out, err := exec.Command("git", "config", "--get", "remote.origin.url").CombinedOutput(); err == nil {
		parsed, perr := url.Parse(bytesTrimSpace(out))
		if perr == nil {
			remote = parsed
		}
	}

	client, err := grpcclient.GetClient()
	if err != nil {
		return "", fmt.Errorf("failed to connect to server: %w", err)
	}

	if err := client.InsertRepoIfNotExists(remote, abs); err != nil {
		return "", err
	}

	if remote != nil {
		return remote.String(), nil
	}

	return abs, nil
}

// bytesTrimSpace is a tiny helper to avoid importing strings for a single use.
func bytesTrimSpace(b []byte) string {
	i := 0
	j := len(b)

	for i < j && (b[i] == ' ' || b[i] == '\n' || b[i] == '\r' || b[i] == '\t') {
		i++
	}

	for i < j && (b[j-1] == ' ' || b[j-1] == '\n' || b[j-1] == '\r' || b[j-1] == '\t') {
		j--
	}

	return string(b[i:j])
}
