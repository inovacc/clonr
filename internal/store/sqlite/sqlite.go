// Package sqlite provides SQLite database storage for clonr.
package sqlite

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/inovacc/clonr/internal/model"
	"github.com/inovacc/clonr/internal/store/sqlite/sqlc"
	_ "modernc.org/sqlite" // Pure Go SQLite driver
)

// Pointer conversion helpers for sqlc compatibility
func ptrString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func ptrInt64(i int64) *int64 {
	return &i
}

// Store implements the store.Store interface using SQLite.
type Store struct {
	db      *sql.DB
	queries *sqlc.Queries
	mu      sync.RWMutex
}

var (
	instance *Store
	once     sync.Once
	initErr  error
)

// New creates a new SQLite store with the given database path.
func New(dbPath string) (*Store, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("creating database directory: %w", err)
	}

	// Open database with WAL mode for better concurrency
	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=ON")
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(1) // SQLite doesn't handle multiple writers well
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)

	// Run migrations
	migrator := NewMigrator(db)
	if err := migrator.MigrateUp(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	return &Store{
		db:      db,
		queries: sqlc.New(db),
	}, nil
}

// GetDB returns the singleton SQLite store instance.
func GetDB() *Store {
	once.Do(func() {
		dbPath := getDefaultDBPath()
		instance, initErr = New(dbPath)
	})
	if initErr != nil {
		panic(fmt.Sprintf("failed to initialize SQLite store: %v", initErr))
	}
	return instance
}

// getDefaultDBPath returns the default database path.
func getDefaultDBPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = "."
	}
	return filepath.Join(configDir, "clonr", "clonr.db")
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// Ping checks if the database is accessible.
func (s *Store) Ping() error {
	return s.db.Ping()
}

// ============================================================================
// Repository Operations
// ============================================================================

func (s *Store) SaveRepo(u *url.URL, path string) error {
	return s.SaveRepoWithWorkspace(u, path, "")
}

func (s *Store) SaveRepoWithWorkspace(u *url.URL, path, workspace string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := newContext()
	_, err := s.queries.InsertRepo(ctx, sqlc.InsertRepoParams{
		Uid:       uuid.New().String(),
		Url:       u.String(),
		Path:      path,
		Workspace: ptrString(workspace),
		Favorite:  ptrInt64(0),
	})
	return err
}

func (s *Store) RepoExistsByURL(u *url.URL) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ctx := newContext()
	result, err := s.queries.RepoExistsByURL(ctx, u.String())
	if err != nil {
		return false, err
	}
	return result == 1, nil
}

func (s *Store) RepoExistsByPath(path string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ctx := newContext()
	result, err := s.queries.RepoExistsByPath(ctx, path)
	if err != nil {
		return false, err
	}
	return result == 1, nil
}

func (s *Store) InsertRepoIfNotExists(u *url.URL, path string) error {
	exists, err := s.RepoExistsByURL(u)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return s.SaveRepo(u, path)
}

func (s *Store) GetAllRepos() ([]*model.Repository, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ctx := newContext()
	rows, err := s.queries.GetAllRepos(ctx)
	if err != nil {
		return nil, err
	}

	repos := make([]*model.Repository, 0, len(rows))
	for _, row := range rows {
		repos = append(repos, sqlcRepoToModel(row))
	}
	return repos, nil
}

func (s *Store) GetRepos(workspace string, favoritesOnly bool) ([]*model.Repository, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ctx := newContext()
	favInt := int64(0)
	if favoritesOnly {
		favInt = 1
	}

	rows, err := s.queries.GetReposByWorkspaceAndFavorites(ctx, sqlc.GetReposByWorkspaceAndFavoritesParams{
		Workspace: ptrString(workspace),
		Column2:   workspace,
		Column3:   favInt,
	})
	if err != nil {
		return nil, err
	}

	repos := make([]*model.Repository, 0, len(rows))
	for _, row := range rows {
		repos = append(repos, sqlcRepoToModel(row))
	}
	return repos, nil
}

func (s *Store) GetReposByWorkspace(workspace string) ([]*model.Repository, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ctx := newContext()
	rows, err := s.queries.GetReposByWorkspace(ctx, ptrString(workspace))
	if err != nil {
		return nil, err
	}

	repos := make([]*model.Repository, 0, len(rows))
	for _, row := range rows {
		repos = append(repos, sqlcRepoToModel(row))
	}
	return repos, nil
}

func (s *Store) SetFavoriteByURL(urlStr string, fav bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := newContext()
	favInt := int64(0)
	if fav {
		favInt = 1
	}
	return s.queries.UpdateRepoFavorite(ctx, sqlc.UpdateRepoFavoriteParams{
		Favorite: ptrInt64(favInt),
		Url:      urlStr,
	})
}

func (s *Store) UpdateRepoTimestamp(urlStr string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := newContext()
	return s.queries.UpdateRepoTimestamp(ctx, urlStr)
}

func (s *Store) UpdateRepoWorkspace(urlStr, workspace string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := newContext()
	return s.queries.UpdateRepoWorkspace(ctx, sqlc.UpdateRepoWorkspaceParams{
		Workspace: ptrString(workspace),
		Url:       urlStr,
	})
}

func (s *Store) RemoveRepoByURL(u *url.URL) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := newContext()
	return s.queries.DeleteRepoByURL(ctx, u.String())
}

// ============================================================================
// Configuration Operations
// ============================================================================

func (s *Store) GetConfig() (*model.Config, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ctx := newContext()
	row, err := s.queries.GetConfig(ctx)
	if err != nil {
		return nil, err
	}

	var customEditors []model.Editor
	if row.CustomEditors != nil && *row.CustomEditors != "" {
		if err := json.Unmarshal([]byte(*row.CustomEditors), &customEditors); err != nil {
			customEditors = nil
		}
	}

	return &model.Config{
		DefaultCloneDir: derefString(row.DefaultCloneDir),
		Editor:          derefString(row.Editor),
		Terminal:        derefString(row.Terminal),
		MonitorInterval: int(derefInt64(row.MonitorInterval)),
		ServerPort:      int(derefInt64(row.ServerPort)),
		CustomEditors:   customEditors,
	}, nil
}

func (s *Store) SaveConfig(cfg *model.Config) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := newContext()

	customEditorsJSON, err := json.Marshal(cfg.CustomEditors)
	if err != nil {
		customEditorsJSON = []byte("[]")
	}
	customEditorsStr := string(customEditorsJSON)

	return s.queries.UpdateConfig(ctx, sqlc.UpdateConfigParams{
		DefaultCloneDir: ptrString(cfg.DefaultCloneDir),
		Editor:          ptrString(cfg.Editor),
		Terminal:        ptrString(cfg.Terminal),
		MonitorInterval: ptrInt64(int64(cfg.MonitorInterval)),
		ServerPort:      ptrInt64(int64(cfg.ServerPort)),
		CustomEditors:   &customEditorsStr,
	})
}

// ============================================================================
// Profile Operations
// ============================================================================

func (s *Store) SaveProfile(profile *model.Profile) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := newContext()

	scopesJSON, _ := json.Marshal(profile.Scopes)
	notifyJSON, _ := json.Marshal(profile.NotifyChannels)
	scopesStr := string(scopesJSON)
	notifyStr := string(notifyJSON)
	tokenStorageStr := string(profile.TokenStorage)

	exists, _ := s.queries.ProfileExists(ctx, profile.Name)
	if exists == 1 {
		return s.queries.UpdateProfile(ctx, sqlc.UpdateProfileParams{
			Host:           ptrString(profile.Host),
			Username:       ptrString(profile.User),
			TokenStorage:   ptrString(tokenStorageStr),
			Scopes:         &scopesStr,
			EncryptedToken: profile.EncryptedToken,
			Workspace:      ptrString(profile.Workspace),
			NotifyChannels: &notifyStr,
			Name:           profile.Name,
		})
	}

	isDefault := int64(0)
	if profile.Default {
		isDefault = 1
	}

	_, err := s.queries.InsertProfile(ctx, sqlc.InsertProfileParams{
		Name:           profile.Name,
		Host:           ptrString(profile.Host),
		Username:       ptrString(profile.User),
		TokenStorage:   ptrString(tokenStorageStr),
		Scopes:         &scopesStr,
		IsDefault:      ptrInt64(isDefault),
		EncryptedToken: profile.EncryptedToken,
		Workspace:      ptrString(profile.Workspace),
		NotifyChannels: &notifyStr,
	})
	return err
}

func (s *Store) GetProfile(name string) (*model.Profile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ctx := newContext()
	row, err := s.queries.GetProfile(ctx, name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("profile %q not found", name)
		}
		return nil, err
	}

	return sqlcProfileToModel(row), nil
}

func (s *Store) GetActiveProfile() (*model.Profile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ctx := newContext()
	row, err := s.queries.GetActiveProfile(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no active profile")
		}
		return nil, err
	}

	return sqlcProfileToModel(row), nil
}

func (s *Store) SetActiveProfile(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := newContext()
	return s.queries.SetActiveProfile(ctx, name)
}

func (s *Store) ListProfiles() ([]*model.Profile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ctx := newContext()
	rows, err := s.queries.ListProfiles(ctx)
	if err != nil {
		return nil, err
	}

	profiles := make([]*model.Profile, 0, len(rows))
	for _, row := range rows {
		profiles = append(profiles, sqlcProfileToModel(row))
	}
	return profiles, nil
}

func (s *Store) DeleteProfile(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := newContext()
	return s.queries.DeleteProfile(ctx, name)
}

func (s *Store) ProfileExists(name string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ctx := newContext()
	result, err := s.queries.ProfileExists(ctx, name)
	if err != nil {
		return false, err
	}
	return result == 1, nil
}

// ============================================================================
// Workspace Operations
// ============================================================================

func (s *Store) SaveWorkspace(workspace *model.Workspace) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := newContext()

	exists, _ := s.queries.WorkspaceExists(ctx, workspace.Name)
	if exists == 1 {
		return s.queries.UpdateWorkspace(ctx, sqlc.UpdateWorkspaceParams{
			Description: ptrString(workspace.Description),
			Path:        ptrString(workspace.Path),
			Name:        workspace.Name,
		})
	}

	isActive := int64(0)
	if workspace.Active {
		isActive = 1
	}

	_, err := s.queries.InsertWorkspace(ctx, sqlc.InsertWorkspaceParams{
		Name:        workspace.Name,
		Description: ptrString(workspace.Description),
		Path:        ptrString(workspace.Path),
		IsActive:    ptrInt64(isActive),
	})
	return err
}

func (s *Store) GetWorkspace(name string) (*model.Workspace, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ctx := newContext()
	row, err := s.queries.GetWorkspace(ctx, name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("workspace %q not found", name)
		}
		return nil, err
	}

	return sqlcWorkspaceToModel(row), nil
}

func (s *Store) GetActiveWorkspace() (*model.Workspace, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ctx := newContext()
	row, err := s.queries.GetActiveWorkspace(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no active workspace")
		}
		return nil, err
	}

	return sqlcWorkspaceToModel(row), nil
}

func (s *Store) SetActiveWorkspace(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := newContext()
	return s.queries.SetActiveWorkspace(ctx, name)
}

func (s *Store) ListWorkspaces() ([]*model.Workspace, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ctx := newContext()
	rows, err := s.queries.ListWorkspaces(ctx)
	if err != nil {
		return nil, err
	}

	workspaces := make([]*model.Workspace, 0, len(rows))
	for _, row := range rows {
		workspaces = append(workspaces, sqlcWorkspaceToModel(row))
	}
	return workspaces, nil
}

func (s *Store) DeleteWorkspace(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := newContext()
	return s.queries.DeleteWorkspace(ctx, name)
}

func (s *Store) WorkspaceExists(name string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ctx := newContext()
	result, err := s.queries.WorkspaceExists(ctx, name)
	if err != nil {
		return false, err
	}
	return result == 1, nil
}

// ============================================================================
// Docker Profile Operations
// ============================================================================

func (s *Store) SaveDockerProfile(profile *model.DockerProfile) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := newContext()
	tokenStorageStr := string(profile.TokenStorage)

	exists, _ := s.queries.DockerProfileExists(ctx, profile.Name)
	if exists == 1 {
		return s.queries.UpdateDockerProfile(ctx, sqlc.UpdateDockerProfileParams{
			Registry:       ptrString(profile.Registry),
			Username:       profile.Username,
			EncryptedToken: profile.EncryptedToken,
			TokenStorage:   ptrString(tokenStorageStr),
			Name:           profile.Name,
		})
	}

	_, err := s.queries.InsertDockerProfile(ctx, sqlc.InsertDockerProfileParams{
		Name:           profile.Name,
		Registry:       ptrString(profile.Registry),
		Username:       profile.Username,
		EncryptedToken: profile.EncryptedToken,
		TokenStorage:   ptrString(tokenStorageStr),
	})
	return err
}

func (s *Store) GetDockerProfile(name string) (*model.DockerProfile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ctx := newContext()
	row, err := s.queries.GetDockerProfile(ctx, name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return sqlcDockerProfileToModel(row), nil
}

func (s *Store) ListDockerProfiles() ([]*model.DockerProfile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ctx := newContext()
	rows, err := s.queries.ListDockerProfiles(ctx)
	if err != nil {
		return nil, err
	}

	profiles := make([]*model.DockerProfile, 0, len(rows))
	for _, row := range rows {
		profiles = append(profiles, sqlcDockerProfileToModel(row))
	}
	return profiles, nil
}

func (s *Store) DeleteDockerProfile(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := newContext()
	return s.queries.DeleteDockerProfile(ctx, name)
}

func (s *Store) DockerProfileExists(name string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ctx := newContext()
	result, err := s.queries.DockerProfileExists(ctx, name)
	if err != nil {
		return false, err
	}
	return result == 1, nil
}

// ============================================================================
// Sealed Key Operations
// ============================================================================

// SealedKeyData represents TPM-sealed encryption key data
type SealedKeyData struct {
	SealedData   []byte            `json:"sealed_data"`
	Version      int               `json:"version"`
	KeyType      string            `json:"key_type"`
	Metadata     map[string]string `json:"metadata"`
	CreatedAt    time.Time         `json:"created_at"`
	RotatedAt    time.Time         `json:"rotated_at,omitempty"`
	LastAccessed time.Time         `json:"last_accessed,omitempty"`
}

func (s *Store) GetSealedKey() (*SealedKeyData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ctx := newContext()
	row, err := s.queries.GetSealedKey(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	var metadata map[string]string
	if row.Metadata != nil && *row.Metadata != "" {
		_ = json.Unmarshal([]byte(*row.Metadata), &metadata)
	}

	return &SealedKeyData{
		SealedData:   row.SealedData,
		Version:      int(derefInt64(row.Version)),
		KeyType:      derefString(row.KeyType),
		Metadata:     metadata,
		CreatedAt:    row.CreatedAt,
		RotatedAt:    row.RotatedAt,
		LastAccessed: row.LastAccessed,
	}, nil
}

func (s *Store) SaveSealedKey(data *SealedKeyData) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := newContext()

	var metadataJSON string
	if data.Metadata != nil {
		b, _ := json.Marshal(data.Metadata)
		metadataJSON = string(b)
	}

	version := int64(data.Version)
	return s.queries.InsertSealedKey(ctx, sqlc.InsertSealedKeyParams{
		SealedData: data.SealedData,
		Version:    &version,
		KeyType:    ptrString(data.KeyType),
		Metadata:   ptrString(metadataJSON),
	})
}

func (s *Store) DeleteSealedKey() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := newContext()
	return s.queries.DeleteSealedKey(ctx)
}

func (s *Store) HasSealedKey() (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ctx := newContext()
	result, err := s.queries.SealedKeyExists(ctx)
	if err != nil {
		return false, err
	}
	return result == 1, nil
}
