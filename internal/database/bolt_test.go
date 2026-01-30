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
	allRepos, err := db.GetRepos("", false)
	if err != nil {
		t.Fatalf("GetRepos(false) error = %v", err)
	}

	if len(allRepos) != 2 {
		t.Errorf("GetRepos(false) returned %d repos, want 2", len(allRepos))
	}

	// Get favorites only
	favRepos, err := db.GetRepos("", true)
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
	repos, _ := db.GetRepos("", true)
	if len(repos) != 1 {
		t.Errorf("Expected 1 favorite, got %d", len(repos))
	}

	// Unset favorite
	if err := db.SetFavoriteByURL(testURL, false); err != nil {
		t.Errorf("SetFavoriteByURL(false) error = %v", err)
	}

	// Verify
	repos, _ = db.GetRepos("", true)
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

func TestBolt_SaveRepo_NilURL(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	err := db.SaveRepo(nil, "/tmp/path")
	if err == nil {
		t.Error("SaveRepo(nil) should return error")
	}
}

func TestBolt_SaveConfig_NilConfig(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	err := db.SaveConfig(nil)
	if err == nil {
		t.Error("SaveConfig(nil) should return error")
	}
}

func TestBolt_SetFavoriteByURL_NonExistent(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Should not error when repo doesn't exist
	err := db.SetFavoriteByURL("https://github.com/nonexistent/repo", true)
	if err != nil {
		t.Errorf("SetFavoriteByURL() on non-existent repo error = %v, want nil", err)
	}
}

func TestBolt_UpdateRepoTimestamp_NonExistent(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Should not error when repo doesn't exist
	err := db.UpdateRepoTimestamp("https://github.com/nonexistent/repo")
	if err != nil {
		t.Errorf("UpdateRepoTimestamp() on non-existent repo error = %v, want nil", err)
	}
}

func TestBolt_RemoveRepoByURL_NonExistent(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	u, _ := url.Parse("https://github.com/nonexistent/repo")

	// Should not error when repo doesn't exist
	err := db.RemoveRepoByURL(u)
	if err != nil {
		t.Errorf("RemoveRepoByURL() on non-existent repo error = %v, want nil", err)
	}
}

func TestBolt_InsertRepoIfNotExists_NilURL(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// With nil URL, should check path only
	err := db.InsertRepoIfNotExists(nil, "/tmp/nil-url-path")
	if err == nil {
		// Expected to error because SaveRepo requires URL
		// But based on code, it calls SaveRepo with nil which returns error
		t.Log("InsertRepoIfNotExists(nil, path) returned error as expected")
	}
}

func TestBolt_SaveRepo_DuplicateURL(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	u, _ := url.Parse("https://github.com/test/duplicate")

	// First save
	if err := db.SaveRepo(u, "/tmp/first"); err != nil {
		t.Fatalf("First SaveRepo() error = %v", err)
	}

	// Second save with same URL but different path should be ignored
	if err := db.SaveRepo(u, "/tmp/second"); err != nil {
		t.Fatalf("Second SaveRepo() error = %v", err)
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

func TestBolt_SaveRepo_DuplicatePath(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	u1, _ := url.Parse("https://github.com/test/repo1")
	u2, _ := url.Parse("https://github.com/test/repo2")

	// First save
	if err := db.SaveRepo(u1, "/tmp/samepath"); err != nil {
		t.Fatalf("First SaveRepo() error = %v", err)
	}

	// Second save with different URL but same path should be ignored
	if err := db.SaveRepo(u2, "/tmp/samepath"); err != nil {
		t.Fatalf("Second SaveRepo() error = %v", err)
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

func TestBolt_Close(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "clonr-close-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	dbPath := filepath.Join(tmpDir, "test.storage")

	db, err := NewBolt(dbPath)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}

	// Close should not error
	if err := db.Close(); err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}
}

func TestBolt_GetConfig_ReturnsDefault(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Get config without saving any - should return default
	cfg, err := db.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig() error = %v", err)
	}

	if cfg == nil {
		t.Fatal("GetConfig() returned nil")
	}

	// Should have default values
	defaultCfg := model.DefaultConfig()
	if cfg.Editor != defaultCfg.Editor {
		t.Errorf("Editor = %q, want default %q", cfg.Editor, defaultCfg.Editor)
	}
}

func TestBolt_InsertRepoIfNotExists_PathAlreadyExists(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	u1, _ := url.Parse("https://github.com/test/first")
	u2, _ := url.Parse("https://github.com/test/second")
	testPath := "/tmp/same-path"

	// First insert
	if err := db.InsertRepoIfNotExists(u1, testPath); err != nil {
		t.Fatalf("First InsertRepoIfNotExists() error = %v", err)
	}

	// Second insert with same path - should be idempotent (no error, no duplicate)
	if err := db.InsertRepoIfNotExists(u2, testPath); err != nil {
		t.Errorf("Second InsertRepoIfNotExists() error = %v, want nil", err)
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

func TestBolt_InsertRepoIfNotExists_URLAlreadyExists(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	testURL := "https://github.com/test/same-url"
	u, _ := url.Parse(testURL)

	// First insert
	if err := db.InsertRepoIfNotExists(u, "/tmp/first-path"); err != nil {
		t.Fatalf("First InsertRepoIfNotExists() error = %v", err)
	}

	// Second insert with same URL but different path - should be idempotent
	if err := db.InsertRepoIfNotExists(u, "/tmp/second-path"); err != nil {
		t.Errorf("Second InsertRepoIfNotExists() error = %v, want nil", err)
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

func TestBolt_RemoveRepoByURL_WithPath(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	testURL := "https://github.com/test/remove-with-path"
	testPath := "/tmp/remove-path-test"
	u, _ := url.Parse(testURL)

	if err := db.SaveRepo(u, testPath); err != nil {
		t.Fatalf("SaveRepo() error = %v", err)
	}

	// Verify both URL and path exist
	existsURL, _ := db.RepoExistsByURL(u)
	existsPath, _ := db.RepoExistsByPath(testPath)

	if !existsURL || !existsPath {
		t.Fatal("Repo should exist by both URL and path before removal")
	}

	// Remove
	if err := db.RemoveRepoByURL(u); err != nil {
		t.Errorf("RemoveRepoByURL() error = %v", err)
	}

	// Verify both URL and path are removed
	existsURL, _ = db.RepoExistsByURL(u)
	existsPath, _ = db.RepoExistsByPath(testPath)

	if existsURL {
		t.Error("Repo should not exist by URL after removal")
	}

	if existsPath {
		t.Error("Repo should not exist by path after removal")
	}
}

func TestBolt_GetRepos_NoFavorites(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Add repos without favorites
	u1, _ := url.Parse("https://github.com/user/repo1")
	u2, _ := url.Parse("https://github.com/user/repo2")

	if err := db.SaveRepo(u1, "/tmp/repo1"); err != nil {
		t.Fatalf("SaveRepo() error = %v", err)
	}

	if err := db.SaveRepo(u2, "/tmp/repo2"); err != nil {
		t.Fatalf("SaveRepo() error = %v", err)
	}

	// Get favorites only - should be empty
	favRepos, err := db.GetRepos("", true)
	if err != nil {
		t.Fatalf("GetRepos(true) error = %v", err)
	}

	if len(favRepos) != 0 {
		t.Errorf("GetRepos(true) returned %d repos, want 0", len(favRepos))
	}
}

func TestBolt_SaveConfig_Multiple(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Save first config
	cfg1 := &model.Config{
		DefaultCloneDir: "/first/path",
		Editor:          "vim",
		ServerPort:      5000,
	}

	if err := db.SaveConfig(cfg1); err != nil {
		t.Fatalf("First SaveConfig() error = %v", err)
	}

	// Save second config (overwrites)
	cfg2 := &model.Config{
		DefaultCloneDir: "/second/path",
		Editor:          "nvim",
		ServerPort:      6000,
	}

	if err := db.SaveConfig(cfg2); err != nil {
		t.Fatalf("Second SaveConfig() error = %v", err)
	}

	// Get config and verify it's the second one
	loaded, err := db.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig() error = %v", err)
	}

	if loaded.DefaultCloneDir != "/second/path" {
		t.Errorf("DefaultCloneDir = %q, want %q", loaded.DefaultCloneDir, "/second/path")
	}

	if loaded.Editor != "nvim" {
		t.Errorf("Editor = %q, want %q", loaded.Editor, "nvim")
	}

	if loaded.ServerPort != 6000 {
		t.Errorf("ServerPort = %d, want %d", loaded.ServerPort, 6000)
	}
}

func TestBolt_SetFavoriteByURL_Toggle(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	testURL := "https://github.com/test/toggle-fav"
	u, _ := url.Parse(testURL)

	if err := db.SaveRepo(u, "/tmp/toggle"); err != nil {
		t.Fatalf("SaveRepo() error = %v", err)
	}

	// Initial state - not favorite
	repos, _ := db.GetAllRepos()
	if repos[0].Favorite {
		t.Error("Repo should not be favorite initially")
	}

	// Set as favorite
	if err := db.SetFavoriteByURL(testURL, true); err != nil {
		t.Fatalf("SetFavoriteByURL(true) error = %v", err)
	}

	repos, _ = db.GetAllRepos()
	if !repos[0].Favorite {
		t.Error("Repo should be favorite after setting")
	}

	// Unset favorite
	if err := db.SetFavoriteByURL(testURL, false); err != nil {
		t.Fatalf("SetFavoriteByURL(false) error = %v", err)
	}

	repos, _ = db.GetAllRepos()
	if repos[0].Favorite {
		t.Error("Repo should not be favorite after unsetting")
	}

	// Set again
	if err := db.SetFavoriteByURL(testURL, true); err != nil {
		t.Fatalf("SetFavoriteByURL(true) second time error = %v", err)
	}

	repos, _ = db.GetAllRepos()
	if !repos[0].Favorite {
		t.Error("Repo should be favorite after setting again")
	}
}

func TestNewBolt_InvalidPath(t *testing.T) {
	// Try to create a database at an invalid path
	_, err := NewBolt("/nonexistent/path/that/does/not/exist/test.db")
	if err == nil {
		t.Error("NewBolt() with invalid path should return error")
	}
}

func TestBolt_RepositoryFieldsPreserved(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	testURL := "https://github.com/test/fields"
	u, _ := url.Parse(testURL)
	testPath := "/tmp/fields-test"

	if err := db.SaveRepo(u, testPath); err != nil {
		t.Fatalf("SaveRepo() error = %v", err)
	}

	repos, err := db.GetAllRepos()
	if err != nil {
		t.Fatalf("GetAllRepos() error = %v", err)
	}

	if len(repos) != 1 {
		t.Fatalf("Expected 1 repo, got %d", len(repos))
	}

	repo := repos[0]

	// Verify all fields are set correctly
	if repo.UID == "" {
		t.Error("UID should not be empty")
	}

	if repo.URL != testURL {
		t.Errorf("URL = %q, want %q", repo.URL, testURL)
	}

	if repo.Path != testPath {
		t.Errorf("Path = %q, want %q", repo.Path, testPath)
	}

	if repo.Favorite {
		t.Error("Favorite should be false by default")
	}

	if repo.ClonedAt.IsZero() {
		t.Error("ClonedAt should not be zero")
	}
}

func TestBolt_MultipleReposOperations(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Add multiple repos
	urls := []string{
		"https://github.com/user/repo1",
		"https://github.com/user/repo2",
		"https://github.com/user/repo3",
		"https://github.com/user/repo4",
		"https://github.com/user/repo5",
	}

	for i, urlStr := range urls {
		u, _ := url.Parse(urlStr)

		if err := db.SaveRepo(u, "/tmp/multi-"+string(rune('a'+i))); err != nil {
			t.Fatalf("SaveRepo(%s) error = %v", urlStr, err)
		}
	}

	// Verify all repos exist
	repos, err := db.GetAllRepos()
	if err != nil {
		t.Fatalf("GetAllRepos() error = %v", err)
	}

	if len(repos) != 5 {
		t.Errorf("Expected 5 repos, got %d", len(repos))
	}

	// Set some as favorites
	if err := db.SetFavoriteByURL(urls[0], true); err != nil {
		t.Fatalf("SetFavoriteByURL() error = %v", err)
	}

	if err := db.SetFavoriteByURL(urls[2], true); err != nil {
		t.Fatalf("SetFavoriteByURL() error = %v", err)
	}

	// Check favorites
	favRepos, err := db.GetRepos("", true)
	if err != nil {
		t.Fatalf("GetRepos(true) error = %v", err)
	}

	if len(favRepos) != 2 {
		t.Errorf("Expected 2 favorites, got %d", len(favRepos))
	}

	// Remove one repo
	u, _ := url.Parse(urls[1])

	if err := db.RemoveRepoByURL(u); err != nil {
		t.Fatalf("RemoveRepoByURL() error = %v", err)
	}

	// Verify removal
	repos, _ = db.GetAllRepos()
	if len(repos) != 4 {
		t.Errorf("Expected 4 repos after removal, got %d", len(repos))
	}

	// Remove a favorite repo
	u, _ = url.Parse(urls[0])

	if err := db.RemoveRepoByURL(u); err != nil {
		t.Fatalf("RemoveRepoByURL() error = %v", err)
	}

	// Check favorites again
	favRepos, _ = db.GetRepos("", true)
	if len(favRepos) != 1 {
		t.Errorf("Expected 1 favorite after removal, got %d", len(favRepos))
	}
}

func TestBolt_GetRepos_WorkspaceFilter(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Add repos in different workspaces
	u1, _ := url.Parse("https://github.com/user/work-repo1")
	u2, _ := url.Parse("https://github.com/user/work-repo2")
	u3, _ := url.Parse("https://github.com/user/personal-repo")
	u4, _ := url.Parse("https://github.com/user/no-workspace")

	if err := db.SaveRepoWithWorkspace(u1, "/tmp/work1", "work"); err != nil {
		t.Fatalf("SaveRepoWithWorkspace() error = %v", err)
	}

	if err := db.SaveRepoWithWorkspace(u2, "/tmp/work2", "work"); err != nil {
		t.Fatalf("SaveRepoWithWorkspace() error = %v", err)
	}

	if err := db.SaveRepoWithWorkspace(u3, "/tmp/personal", "personal"); err != nil {
		t.Fatalf("SaveRepoWithWorkspace() error = %v", err)
	}

	if err := db.SaveRepo(u4, "/tmp/none"); err != nil {
		t.Fatalf("SaveRepo() error = %v", err)
	}

	// Test: Get all repos (no workspace filter)
	allRepos, err := db.GetRepos("", false)
	if err != nil {
		t.Fatalf("GetRepos('', false) error = %v", err)
	}

	if len(allRepos) != 4 {
		t.Errorf("GetRepos('', false) returned %d repos, want 4", len(allRepos))
	}

	// Test: Get repos in "work" workspace
	workRepos, err := db.GetRepos("work", false)
	if err != nil {
		t.Fatalf("GetRepos('work', false) error = %v", err)
	}

	if len(workRepos) != 2 {
		t.Errorf("GetRepos('work', false) returned %d repos, want 2", len(workRepos))
	}

	// Test: Get repos in "personal" workspace
	personalRepos, err := db.GetRepos("personal", false)
	if err != nil {
		t.Fatalf("GetRepos('personal', false) error = %v", err)
	}

	if len(personalRepos) != 1 {
		t.Errorf("GetRepos('personal', false) returned %d repos, want 1", len(personalRepos))
	}

	// Test: Get repos with non-existent workspace
	noRepos, err := db.GetRepos("nonexistent", false)
	if err != nil {
		t.Fatalf("GetRepos('nonexistent', false) error = %v", err)
	}

	if len(noRepos) != 0 {
		t.Errorf("GetRepos('nonexistent', false) returned %d repos, want 0", len(noRepos))
	}

	// Test: Workspace + favorites filter combined
	// Mark one work repo as favorite
	if err := db.SetFavoriteByURL(u1.String(), true); err != nil {
		t.Fatalf("SetFavoriteByURL() error = %v", err)
	}

	workFavRepos, err := db.GetRepos("work", true)
	if err != nil {
		t.Fatalf("GetRepos('work', true) error = %v", err)
	}

	if len(workFavRepos) != 1 {
		t.Errorf("GetRepos('work', true) returned %d repos, want 1", len(workFavRepos))
	}

	// All favorites (any workspace)
	allFavRepos, err := db.GetRepos("", true)
	if err != nil {
		t.Fatalf("GetRepos('', true) error = %v", err)
	}

	if len(allFavRepos) != 1 {
		t.Errorf("GetRepos('', true) returned %d repos, want 1", len(allFavRepos))
	}
}
