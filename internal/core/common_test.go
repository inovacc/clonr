package core

import (
	"archive/zip"
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDotGit(t *testing.T) {
	testdataPath := "testdata/repofake.zip"
	if _, err := os.Stat(testdataPath); os.IsNotExist(err) {
		t.Skip("testdata/repofake.zip not found, skipping test")
	}

	dest, err := unzipHelper(testdataPath, t.TempDir())
	require.NoError(t, err)

	g, err := dotGitCheck(filepath.Join(dest, ".git"))
	require.NoError(t, err)

	t.Log(g)
}

func TestGitHubURL_HTTPS(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantScheme string
		wantHost   string
		wantPath   string
		wantErr    bool
	}{
		{
			name:       "https github",
			input:      "https://github.com/user/repo",
			wantScheme: "https",
			wantHost:   "github.com",
			wantPath:   "/user/repo",
		},
		{
			name:       "https github with .git",
			input:      "https://github.com/user/repo.git",
			wantScheme: "https",
			wantHost:   "github.com",
			wantPath:   "/user/repo.git",
		},
		{
			name:       "http url",
			input:      "http://gitlab.com/user/project",
			wantScheme: "http",
			wantHost:   "gitlab.com",
			wantPath:   "/user/project",
		},
		{
			name:       "https with port",
			input:      "https://github.com:443/user/repo",
			wantScheme: "https",
			wantHost:   "github.com:443",
			wantPath:   "/user/repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := gitHubURL(tt.input)
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.wantScheme, u.Scheme)
			require.Equal(t, tt.wantHost, u.Host)
			require.Equal(t, tt.wantPath, u.Path)
		})
	}
}

func TestGitHubURL_SCPLike(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantScheme string
		wantUser   string
		wantHost   string
		wantPath   string
		wantErr    bool
	}{
		{
			name:       "scp-like github",
			input:      "git@github.com:user/repo.git",
			wantScheme: "ssh",
			wantUser:   "git",
			wantHost:   "github.com",
			wantPath:   "/user/repo.git",
		},
		{
			name:       "scp-like gitlab",
			input:      "git@gitlab.com:group/project.git",
			wantScheme: "ssh",
			wantUser:   "git",
			wantHost:   "gitlab.com",
			wantPath:   "/group/project.git",
		},
		{
			name:       "scp-like bitbucket",
			input:      "git@bitbucket.org:team/repo.git",
			wantScheme: "ssh",
			wantUser:   "git",
			wantHost:   "bitbucket.org",
			wantPath:   "/team/repo.git",
		},
		{
			name:       "scp-like with nested path",
			input:      "git@github.com:org/subgroup/repo.git",
			wantScheme: "ssh",
			wantUser:   "git",
			wantHost:   "github.com",
			wantPath:   "/org/subgroup/repo.git",
		},
		{
			name:    "invalid scp-like missing user",
			input:   "@github.com:user/repo.git",
			wantErr: true,
		},
		{
			name:    "invalid scp-like missing host",
			input:   "git@:user/repo.git",
			wantErr: true,
		},
		{
			name:    "invalid scp-like missing path",
			input:   "git@github.com:",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := gitHubURL(tt.input)
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.wantScheme, u.Scheme)
			require.Equal(t, tt.wantUser, u.User.Username())
			require.Equal(t, tt.wantHost, u.Host)
			require.Equal(t, tt.wantPath, u.Path)
		})
	}
}

func TestGitHubURL_SSH(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantScheme string
		wantHost   string
		wantPath   string
		wantErr    bool
	}{
		{
			name:       "ssh url",
			input:      "ssh://git@github.com/user/repo.git",
			wantScheme: "ssh",
			wantHost:   "github.com",
			wantPath:   "/user/repo.git",
		},
		{
			name:       "git+ssh url",
			input:      "git+ssh://git@github.com/user/repo.git",
			wantScheme: "ssh",
			wantHost:   "github.com",
			wantPath:   "/user/repo.git",
		},
		{
			name:       "ssh with port",
			input:      "ssh://git@github.com:22/user/repo.git",
			wantScheme: "ssh",
			wantHost:   "github.com:22",
			wantPath:   "/user/repo.git",
		},
		{
			name:    "ssh missing host",
			input:   "ssh:///user/repo.git",
			wantErr: true,
		},
		{
			name:    "ssh missing path",
			input:   "ssh://git@github.com",
			wantErr: true,
		},
		{
			name:    "ssh only slash path",
			input:   "ssh://git@github.com/",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := gitHubURL(tt.input)
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.wantScheme, u.Scheme)
			require.Equal(t, tt.wantHost, u.Host)
			require.Equal(t, tt.wantPath, u.Path)
		})
	}
}

func TestGitHubURL_Git(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantScheme string
		wantHost   string
		wantPath   string
		wantErr    bool
	}{
		{
			name:       "git protocol",
			input:      "git://github.com/user/repo.git",
			wantScheme: "git",
			wantHost:   "github.com",
			wantPath:   "/user/repo.git",
		},
		{
			name:    "git protocol missing host",
			input:   "git:///user/repo.git",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := gitHubURL(tt.input)
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.wantScheme, u.Scheme)
			require.Equal(t, tt.wantHost, u.Host)
			require.Equal(t, tt.wantPath, u.Path)
		})
	}
}

func TestGitHubURL_Empty(t *testing.T) {
	u, err := gitHubURL("")
	require.NoError(t, err)
	require.Equal(t, "file", u.Scheme)
	require.Equal(t, ".", u.Path)
}

func TestGitHubURL_Whitespace(t *testing.T) {
	u, err := gitHubURL("   ")
	require.NoError(t, err)
	require.Equal(t, "file", u.Scheme)
	require.Equal(t, ".", u.Path)
}

func TestGitHubURL_InvalidScheme(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"ftp scheme", "ftp://example.com/repo.git"},
		{"file scheme", "file:///path/to/repo"},
		{"unknown scheme", "unknown://host/path"},
		{"missing scheme", "github.com/user/repo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := gitHubURL(tt.input)
			require.Error(t, err)
		})
	}
}

func TestIsSCPLike(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"scp-like github", "git@github.com:user/repo.git", true},
		{"scp-like gitlab", "git@gitlab.com:group/project", true},
		{"scp-like custom", "user@host.example.com:path/to/repo", true},
		{"https url", "https://github.com/user/repo", false},
		{"http url", "http://github.com/user/repo", false},
		{"ssh url", "ssh://git@github.com/user/repo", false},
		{"git url", "git://github.com/user/repo", false},
		{"git+ssh url", "git+ssh://git@github.com/user/repo", false},
		{"no colon", "git@github.com", false},
		{"no at sign", "github.com:user/repo", false},
		{"empty string", "", false},
		{"colon before at", "host:path@user", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSCPLike(tt.input)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestHasSchemePrefix(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"https", "https://github.com/user/repo", true},
		{"http", "http://github.com/user/repo", true},
		{"ssh", "ssh://git@github.com/user/repo", true},
		{"git", "git://github.com/user/repo", true},
		{"git+ssh", "git+ssh://git@github.com/user/repo", true},
		{"HTTPS uppercase", "HTTPS://github.com/user/repo", true},
		{"HTTP uppercase", "HTTP://github.com/user/repo", true},
		{"SSH uppercase", "SSH://git@github.com/user/repo", true},
		{"GIT uppercase", "GIT://github.com/user/repo", true},
		{"GIT+SSH uppercase", "GIT+SSH://git@github.com/user/repo", true},
		{"scp-like", "git@github.com:user/repo", false},
		{"no scheme", "github.com/user/repo", false},
		{"ftp scheme", "ftp://example.com/file", false},
		{"empty string", "", false},
		{"just protocol", "https://", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasSchemePrefix(tt.input)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestDotGitCheck_NotGitRepo(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := dotGitCheck(tmpDir)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not a git repo")
}

func TestDotGitCheck_NoConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, ".git")
	require.NoError(t, os.MkdirAll(gitDir, 0755))

	_, err := dotGitCheck(gitDir)
	require.Error(t, err)
}

func unzipHelper(zipFile, destDir string) (string, error) {
	archive, err := zip.OpenReader(zipFile)
	if err != nil {
		return "", err
	}
	defer func(archive *zip.ReadCloser) {
		if err := archive.Close(); err != nil {
			log.Fatal(err)
		}
	}(archive)

	destPath := filepath.Join(destDir, archive.File[0].Name)

	for _, file := range archive.File {
		filePath := filepath.Join(destDir, file.Name)

		if file.FileInfo().IsDir() {
			_ = os.MkdirAll(filePath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			return "", err
		}

		dstFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return "", err
		}

		fileInArchive, err := file.Open()
		if err != nil {
			return "", err
		}

		if _, err := io.Copy(dstFile, fileInArchive); err != nil {
			return "", err
		}

		if err := dstFile.Close(); err != nil {
			return "", err
		}

		if err := fileInArchive.Close(); err != nil {
			return "", err
		}
	}

	return destPath, nil
}
