package standalone

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateAndExtractArchive(t *testing.T) {
	// Create temp directories
	tempDir := t.TempDir()
	repoDir := filepath.Join(tempDir, "test-repo")
	outputDir := filepath.Join(tempDir, "output")
	archivePath := filepath.Join(tempDir, "test.clonr")

	// Create a test repository structure
	if err := os.MkdirAll(filepath.Join(repoDir, "src"), 0755); err != nil {
		t.Fatalf("Failed to create test dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(repoDir, ".git"), 0755); err != nil {
		t.Fatalf("Failed to create .git dir: %v", err)
	}

	// Create test files
	files := map[string]string{
		"README.md":          "# Test Repository\n",
		"src/main.go":        "package main\n\nfunc main() {}\n",
		"src/util.go":        "package main\n\nfunc util() {}\n",
		".git/config":        "[core]\n\trepositoryformatversion = 0\n",
		".git/HEAD":          "ref: refs/heads/main\n",
		".gitignore":         "*.log\n",
	}

	for name, content := range files {
		path := filepath.Join(repoDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create %s: %v", name, err)
		}
	}

	password := "testpassword123"

	t.Run("create archive", func(t *testing.T) {
		opts := DefaultArchiveOptions()
		opts.Password = password

		manifest, err := CreateRepoArchive(archivePath, []string{repoDir}, opts)
		if err != nil {
			t.Fatalf("CreateRepoArchive() error = %v", err)
		}

		if manifest.Version != ArchiveVersion {
			t.Errorf("manifest.Version = %d, want %d", manifest.Version, ArchiveVersion)
		}
		if len(manifest.Repositories) != 1 {
			t.Errorf("manifest.Repositories length = %d, want 1", len(manifest.Repositories))
		}
		if manifest.Checksum == "" {
			t.Error("manifest.Checksum is empty")
		}

		// Verify archive file exists
		if _, err := os.Stat(archivePath); err != nil {
			t.Errorf("Archive file not created: %v", err)
		}
	})

	t.Run("list archive contents", func(t *testing.T) {
		manifest, err := ListArchiveContents(archivePath, password)
		if err != nil {
			t.Fatalf("ListArchiveContents() error = %v", err)
		}

		if len(manifest.Repositories) != 1 {
			t.Errorf("manifest.Repositories length = %d, want 1", len(manifest.Repositories))
		}

		repo := manifest.Repositories[0]
		if repo.Name != "test-repo" {
			t.Errorf("repo.Name = %s, want test-repo", repo.Name)
		}
	})

	t.Run("extract archive", func(t *testing.T) {
		manifest, err := ExtractRepoArchive(archivePath, outputDir, password)
		if err != nil {
			t.Fatalf("ExtractRepoArchive() error = %v", err)
		}

		if len(manifest.Repositories) != 1 {
			t.Errorf("manifest.Repositories length = %d, want 1", len(manifest.Repositories))
		}

		// Verify extracted files
		for name, expectedContent := range files {
			path := filepath.Join(outputDir, "test-repo", name)
			content, err := os.ReadFile(path)
			if err != nil {
				t.Errorf("Failed to read extracted %s: %v", name, err)
				continue
			}
			if string(content) != expectedContent {
				t.Errorf("Extracted %s content mismatch", name)
			}
		}
	})

	t.Run("wrong password", func(t *testing.T) {
		_, err := ListArchiveContents(archivePath, "wrongpassword")
		if err == nil {
			t.Error("ListArchiveContents() expected error with wrong password")
		}
	})
}

func TestCreateArchiveNoGit(t *testing.T) {
	tempDir := t.TempDir()
	repoDir := filepath.Join(tempDir, "test-repo")
	archivePath := filepath.Join(tempDir, "test.clonr")

	// Create test structure with .git
	if err := os.MkdirAll(filepath.Join(repoDir, ".git", "objects"), 0755); err != nil {
		t.Fatalf("Failed to create dirs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, "file.txt"), []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, ".git", "config"), []byte("config"), 0644); err != nil {
		t.Fatalf("Failed to create .git/config: %v", err)
	}

	opts := DefaultArchiveOptions()
	opts.Password = "testpassword"
	opts.IncludeGitDir = false

	manifest, err := CreateRepoArchive(archivePath, []string{repoDir}, opts)
	if err != nil {
		t.Fatalf("CreateRepoArchive() error = %v", err)
	}

	// Extract and verify .git is not included
	outputDir := filepath.Join(tempDir, "output")
	_, err = ExtractRepoArchive(archivePath, outputDir, "testpassword")
	if err != nil {
		t.Fatalf("ExtractRepoArchive() error = %v", err)
	}

	// .git should not exist
	gitPath := filepath.Join(outputDir, "test-repo", ".git")
	if _, err := os.Stat(gitPath); !os.IsNotExist(err) {
		t.Error(".git directory should not exist when IncludeGitDir=false")
	}

	// file.txt should exist
	filePath := filepath.Join(outputDir, "test-repo", "file.txt")
	if _, err := os.Stat(filePath); err != nil {
		t.Errorf("file.txt should exist: %v", err)
	}

	_ = manifest // Silence unused variable warning
}

func TestExcludePatterns(t *testing.T) {
	tests := []struct {
		path     string
		patterns []string
		want     bool
	}{
		{"node_modules", []string{"node_modules/**"}, true},
		{"node_modules/package/file.js", []string{"node_modules/**"}, true},
		{"src/main.go", []string{"node_modules/**"}, false},
		{"file.pyc", []string{"*.pyc"}, true},
		{"src/file.pyc", []string{"*.pyc"}, true},
		{".env", []string{".env", ".env.*"}, true},
		{".env.local", []string{".env", ".env.*"}, true},
		{"config.env", []string{".env", ".env.*"}, false},
		{"vendor/github.com/pkg", []string{"vendor/**"}, true},
		{"__pycache__/file.py", []string{"__pycache__/**"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := shouldExclude(tt.path, tt.patterns)
			if got != tt.want {
				t.Errorf("shouldExclude(%q, %v) = %v, want %v", tt.path, tt.patterns, got, tt.want)
			}
		})
	}
}

func TestArchiveConstants(t *testing.T) {
	if ArchiveMagic != "CLONR-REPO" {
		t.Errorf("ArchiveMagic = %s, want CLONR-REPO", ArchiveMagic)
	}
	if ArchiveExtension != ".clonr" {
		t.Errorf("ArchiveExtension = %s, want .clonr", ArchiveExtension)
	}
}

func TestDefaultArchiveOptions(t *testing.T) {
	opts := DefaultArchiveOptions()

	if !opts.IncludeGitDir {
		t.Error("IncludeGitDir should be true by default")
	}
	if opts.CompressionLevel != 6 {
		t.Errorf("CompressionLevel = %d, want 6", opts.CompressionLevel)
	}
	if len(opts.ExcludePatterns) == 0 {
		t.Error("ExcludePatterns should not be empty")
	}
}

func TestArchiveEmptyPassword(t *testing.T) {
	tempDir := t.TempDir()
	repoDir := filepath.Join(tempDir, "repo")
	archivePath := filepath.Join(tempDir, "test.clonr")

	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatalf("Failed to create dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, "file.txt"), []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	opts := DefaultArchiveOptions()
	opts.Password = ""

	_, err := CreateRepoArchive(archivePath, []string{repoDir}, opts)
	if err == nil {
		t.Error("CreateRepoArchive() expected error with empty password")
	}
}

func TestArchiveInvalidPath(t *testing.T) {
	tempDir := t.TempDir()
	archivePath := filepath.Join(tempDir, "test.clonr")

	opts := DefaultArchiveOptions()
	opts.Password = "testpassword"

	_, err := CreateRepoArchive(archivePath, []string{"/nonexistent/path"}, opts)
	if err == nil {
		t.Error("CreateRepoArchive() expected error with invalid path")
	}
}
