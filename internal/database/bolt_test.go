package database

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/inovacc/clonr/internal/model"
)

func setupTestDB(t *testing.T) (*Bolt, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "clonr-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tmpDir, "test.storage")

	db, err := NewBolt(dbPath)
	if err != nil {
		_ = os.RemoveAll(tmpDir)

		t.Fatalf("failed to create test database: %v", err)
	}

	cleanup := func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close database: %v", err)
		}

		_ = os.RemoveAll(tmpDir)
	}

	return db, cleanup
}

func TestBolt_Ping(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	if err := db.Ping(); err != nil {
		t.Errorf("Ping() error = %v, want nil", err)
	}
}

func TestBolt_SaveRepo(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tests := []struct {
		name    string
		url     string
		path    string
		wantErr bool
	}{
		{
			name:    "valid github url",
			url:     "https://github.com/user/repo",
			path:    "/tmp/repo",
			wantErr: false,
		},
		{
			name:    "valid gitlab url",
			url:     "https://gitlab.com/user/project",
			path:    "/tmp/project",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := url.Parse(tt.url)
			if err != nil {
				t.Fatalf("failed to parse URL: %v", err)
			}

			err = db.SaveRepo(u, tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("SaveRepo() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBolt_RepoExistsByURL(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	testURL := "https://github.com/test/exists"
	u, _ := url.Parse(testURL)

	// Should not exist initially
	exists, err := db.RepoExistsByURL(u)
	if err != nil {
		t.Fatalf("RepoExistsByURL() error = %v", err)
	}

	if exists {
		t.Error("RepoExistsByURL() = true, want false for non-existent repo")
	}

	// Save the repo
	if err := db.SaveRepo(u, "/tmp/exists"); err != nil {
		t.Fatalf("SaveRepo() error = %v", err)
	}

	// Should exist now
	exists, err = db.RepoExistsByURL(u)
	if err != nil {
		t.Fatalf("RepoExistsByURL() error = %v", err)
	}

	if !exists {
		t.Error("RepoExistsByURL() = false, want true for existing repo")
	}
}

func TestBolt_RepoExistsByPath(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	testPath := "/tmp/test-path-exists"
	u, _ := url.Parse("https://github.com/test/path")

	// Should not exist initially
	exists, err := db.RepoExistsByPath(testPath)
	if err != nil {
		t.Fatalf("RepoExistsByPath() error = %v", err)
	}

	if exists {
		t.Error("RepoExistsByPath() = true, want false for non-existent path")
	}

	// Save the repo
	if err := db.SaveRepo(u, testPath); err != nil {
		t.Fatalf("SaveRepo() error = %v", err)
	}

	// Should exist now
	exists, err = db.RepoExistsByPath(testPath)
	if err != nil {
		t.Fatalf("RepoExistsByPath() error = %v", err)
	}

	if !exists {
		t.Error("RepoExistsByPath() = false, want true for existing path")
	}
}

func TestBolt_InsertRepoIfNotExists(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	testURL := "https://github.com/test/insert"
	u, _ := url.Parse(testURL)
	testPath := "/tmp/insert"

	// First insert should succeed
	if err := db.InsertRepoIfNotExists(u, testPath); err != nil {
		t.Fatalf("InsertRepoIfNotExists() first call error = %v", err)
	}

	// Second insert should not error (idempotent)
	if err := db.InsertRepoIfNotExists(u, testPath); err != nil {
		t.Errorf("InsertRepoIfNotExists() second call error = %v, want nil", err)
	}

	// Verify only one repo exists
	repos, err := db.GetAllRepos()
	if err != nil {
		t.Fatalf("GetAllRepos() error = %v", err)
	}

	if len(repos) != 1 {
		t.Errorf("GetAllRepos() returned %d repos, want 1", len(repos))
	}
}

func TestBolt_GetAllRepos(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Initially empty
	repos, err := db.GetAllRepos()
	if err != nil {
		t.Fatalf("GetAllRepos() error = %v", err)
	}

	if len(repos) != 0 {
		t.Errorf("GetAllRepos() returned %d repos, want 0", len(repos))
	}

	// Add some repos
	urls := []string{
		"https://github.com/user/repo1",
		"https://github.com/user/repo2",
		"https://github.com/user/repo3",
	}

	for i, urlStr := range urls {
		u, _ := url.Parse(urlStr)

		if err := db.SaveRepo(u, "/tmp/repo"+string(rune('1'+i))); err != nil {
			t.Fatalf("SaveRepo() error = %v", err)
		}
	}

	// Should have 3 repos
	repos, err = db.GetAllRepos()
	if err != nil {
		t.Fatalf("GetAllRepos() error = %v", err)
	}

	if len(repos) != 3 {
		t.Errorf("GetAllRepos() returned %d repos, want 3", len(repos))
	}
}

func TestBolt_GetRepos_FavoritesOnly(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Add repos
	u1, _ := url.Parse("https://github.com/user/fav")
	u2, _ := url.Parse("https://github.com/user/nofav")

	if err := db.SaveRepo(u1, "/tmp/fav"); err != nil {
		t.Fatalf("SaveRepo() error = %v", err)
	}

	if err := db.SaveRepo(u2, "/tmp/nofav"); err != nil {
		t.Fatalf("SaveRepo() error = %v", err)
	}

	// Mark one as favorite
	if err := db.SetFavoriteByURL(u1.String(), true); err != nil {
		t.Fatalf("SetFavoriteByURL() error = %v", err)
	}

	// Get all repos
	allRepos, err := db.GetRepos(false)
	if err != nil {
		t.Fatalf("GetRepos(false) error = %v", err)
	}

	if len(allRepos) != 2 {
		t.Errorf("GetRepos(false) returned %d repos, want 2", len(allRepos))
	}

	// Get favorites only
	favRepos, err := db.GetRepos(true)
	if err != nil {
		t.Fatalf("GetRepos(true) error = %v", err)
	}

	if len(favRepos) != 1 {
		t.Errorf("GetRepos(true) returned %d repos, want 1", len(favRepos))
	}
}

func TestBolt_SetFavoriteByURL(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	testURL := "https://github.com/test/favorite"
	u, _ := url.Parse(testURL)

	if err := db.SaveRepo(u, "/tmp/favorite"); err != nil {
		t.Fatalf("SaveRepo() error = %v", err)
	}

	// Set as favorite
	if err := db.SetFavoriteByURL(testURL, true); err != nil {
		t.Errorf("SetFavoriteByURL(true) error = %v", err)
	}

	// Verify
	repos, _ := db.GetRepos(true)
	if len(repos) != 1 {
		t.Errorf("Expected 1 favorite, got %d", len(repos))
	}

	// Unset favorite
	if err := db.SetFavoriteByURL(testURL, false); err != nil {
		t.Errorf("SetFavoriteByURL(false) error = %v", err)
	}

	// Verify
	repos, _ = db.GetRepos(true)
	if len(repos) != 0 {
		t.Errorf("Expected 0 favorites, got %d", len(repos))
	}
}

func TestBolt_RemoveRepoByURL(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	testURL := "https://github.com/test/remove"
	u, _ := url.Parse(testURL)

	if err := db.SaveRepo(u, "/tmp/remove"); err != nil {
		t.Fatalf("SaveRepo() error = %v", err)
	}

	// Verify exists
	exists, _ := db.RepoExistsByURL(u)
	if !exists {
		t.Fatal("Repo should exist before removal")
	}

	// Remove
	if err := db.RemoveRepoByURL(u); err != nil {
		t.Errorf("RemoveRepoByURL() error = %v", err)
	}

	// Verify removed
	exists, _ = db.RepoExistsByURL(u)
	if exists {
		t.Error("Repo should not exist after removal")
	}
}

func TestBolt_Config(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Get default config
	cfg, err := db.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig() error = %v", err)
	}

	if cfg == nil {
		t.Fatal("GetConfig() returned nil")
	}

	// Modify and save
	cfg.Editor = "nvim"
	cfg.ServerPort = 9999

	if err := db.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	// Retrieve and verify
	cfg2, err := db.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig() after save error = %v", err)
	}

	if cfg2.Editor != "nvim" {
		t.Errorf("Config.Editor = %q, want %q", cfg2.Editor, "nvim")
	}

	if cfg2.ServerPort != 9999 {
		t.Errorf("Config.ServerPort = %d, want %d", cfg2.ServerPort, 9999)
	}
}

func TestBolt_UpdateRepoTimestamp(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	testURL := "https://github.com/test/timestamp"
	u, _ := url.Parse(testURL)

	if err := db.SaveRepo(u, "/tmp/timestamp"); err != nil {
		t.Fatalf("SaveRepo() error = %v", err)
	}

	// Get initial timestamp
	repos, _ := db.GetAllRepos()
	initialTime := repos[0].UpdatedAt

	// Update timestamp
	if err := db.UpdateRepoTimestamp(testURL); err != nil {
		t.Errorf("UpdateRepoTimestamp() error = %v", err)
	}

	// Verify timestamp changed
	repos, _ = db.GetAllRepos()
	if !repos[0].UpdatedAt.After(initialTime) {
		t.Error("UpdatedAt should be after initial time")
	}
}

func TestBolt_SaveConfig(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	cfg := &model.Config{
		DefaultCloneDir: "/custom/path",
		Editor:          "vim",
		Terminal:        "alacritty",
		MonitorInterval: 600,
		ServerPort:      8080,
	}

	if err := db.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	loaded, err := db.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig() error = %v", err)
	}

	if loaded.DefaultCloneDir != cfg.DefaultCloneDir {
		t.Errorf("DefaultCloneDir = %q, want %q", loaded.DefaultCloneDir, cfg.DefaultCloneDir)
	}

	if loaded.Editor != cfg.Editor {
		t.Errorf("Editor = %q, want %q", loaded.Editor, cfg.Editor)
	}

	if loaded.Terminal != cfg.Terminal {
		t.Errorf("Terminal = %q, want %q", loaded.Terminal, cfg.Terminal)
	}

	if loaded.MonitorInterval != cfg.MonitorInterval {
		t.Errorf("MonitorInterval = %d, want %d", loaded.MonitorInterval, cfg.MonitorInterval)
	}

	if loaded.ServerPort != cfg.ServerPort {
		t.Errorf("ServerPort = %d, want %d", loaded.ServerPort, cfg.ServerPort)
	}
}
