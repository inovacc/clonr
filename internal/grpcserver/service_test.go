package grpcserver

import (
	"context"
	"errors"
	"net/url"
	"testing"

	"github.com/inovacc/clonr/internal/model"
	v1 "github.com/inovacc/clonr/pkg/api/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// mockStore implements store.Store for testing
type mockStore struct {
	pingErr             error
	saveRepoErr         error
	repoExistsByURL     bool
	repoExistsByURLErr  error
	repoExistsByPath    bool
	repoExistsByPathErr error
	insertRepoErr       error
	getAllReposResult   []model.Repository
	getAllReposErr      error
	getReposResult      []model.Repository
	getReposErr         error
	setFavoriteErr      error
	updateTimestampErr  error
	removeRepoErr       error
	getConfigResult     *model.Config
	getConfigErr        error
	saveConfigErr       error

	// Profile fields
	saveProfileErr      error
	getProfileResult    *model.Profile
	getProfileErr       error
	getActiveProfileRes *model.Profile
	getActiveProfileErr error
	setActiveProfileErr error
	listProfilesResult  []model.Profile
	listProfilesErr     error
	deleteProfileErr    error
	profileExistsResult bool
	profileExistsErr    error

	// Workspace fields
	saveWorkspaceErr         error
	getWorkspaceResult       *model.Workspace
	getWorkspaceErr          error
	getActiveWorkspaceRes    *model.Workspace
	getActiveWorkspaceErr    error
	setActiveWorkspaceErr    error
	listWorkspacesResult     []model.Workspace
	listWorkspacesErr        error
	deleteWorkspaceErr       error
	workspaceExistsResult    bool
	workspaceExistsErr       error
	getReposByWorkspaceRes   []string
	getReposByWorkspaceErr   error
	updateRepoWorkspaceErr   error
	saveRepoWithWorkspaceErr error
}

func (m *mockStore) Ping() error {
	return m.pingErr
}

func (m *mockStore) SaveRepo(_ *url.URL, _ string) error {
	return m.saveRepoErr
}

func (m *mockStore) RepoExistsByURL(_ *url.URL) (bool, error) {
	return m.repoExistsByURL, m.repoExistsByURLErr
}

func (m *mockStore) RepoExistsByPath(_ string) (bool, error) {
	return m.repoExistsByPath, m.repoExistsByPathErr
}

func (m *mockStore) InsertRepoIfNotExists(_ *url.URL, _ string) error {
	return m.insertRepoErr
}

func (m *mockStore) GetAllRepos() ([]model.Repository, error) {
	return m.getAllReposResult, m.getAllReposErr
}

func (m *mockStore) GetRepos(_ string, _ bool) ([]model.Repository, error) {
	return m.getReposResult, m.getReposErr
}

func (m *mockStore) SetFavoriteByURL(_ string, _ bool) error {
	return m.setFavoriteErr
}

func (m *mockStore) UpdateRepoTimestamp(_ string) error {
	return m.updateTimestampErr
}

func (m *mockStore) RemoveRepoByURL(_ *url.URL) error {
	return m.removeRepoErr
}

func (m *mockStore) GetConfig() (*model.Config, error) {
	return m.getConfigResult, m.getConfigErr
}

func (m *mockStore) SaveConfig(_ *model.Config) error {
	return m.saveConfigErr
}

func (m *mockStore) Close() error {
	return nil
}

func (m *mockStore) SaveProfile(_ *model.Profile) error {
	return m.saveProfileErr
}

func (m *mockStore) GetProfile(_ string) (*model.Profile, error) {
	return m.getProfileResult, m.getProfileErr
}

func (m *mockStore) GetActiveProfile() (*model.Profile, error) {
	return m.getActiveProfileRes, m.getActiveProfileErr
}

func (m *mockStore) SetActiveProfile(_ string) error {
	return m.setActiveProfileErr
}

func (m *mockStore) ListProfiles() ([]model.Profile, error) {
	return m.listProfilesResult, m.listProfilesErr
}

func (m *mockStore) DeleteProfile(_ string) error {
	return m.deleteProfileErr
}

func (m *mockStore) ProfileExists(_ string) (bool, error) {
	return m.profileExistsResult, m.profileExistsErr
}

func (m *mockStore) SaveRepoWithWorkspace(_ *url.URL, _ string, _ string) error {
	return m.saveRepoWithWorkspaceErr
}

func (m *mockStore) SaveWorkspace(_ *model.Workspace) error {
	return m.saveWorkspaceErr
}

func (m *mockStore) GetWorkspace(_ string) (*model.Workspace, error) {
	return m.getWorkspaceResult, m.getWorkspaceErr
}

func (m *mockStore) GetActiveWorkspace() (*model.Workspace, error) {
	return m.getActiveWorkspaceRes, m.getActiveWorkspaceErr
}

func (m *mockStore) SetActiveWorkspace(_ string) error {
	return m.setActiveWorkspaceErr
}

func (m *mockStore) ListWorkspaces() ([]model.Workspace, error) {
	return m.listWorkspacesResult, m.listWorkspacesErr
}

func (m *mockStore) DeleteWorkspace(_ string) error {
	return m.deleteWorkspaceErr
}

func (m *mockStore) WorkspaceExists(_ string) (bool, error) {
	return m.workspaceExistsResult, m.workspaceExistsErr
}

func (m *mockStore) GetReposByWorkspace(_ string) ([]string, error) {
	return m.getReposByWorkspaceRes, m.getReposByWorkspaceErr
}

func (m *mockStore) UpdateRepoWorkspace(_ string, _ string) error {
	return m.updateRepoWorkspaceErr
}

func TestNewService(t *testing.T) {
	mock := &mockStore{}

	svc := NewService(mock)
	if svc == nil {
		t.Fatal("NewService() returned nil")
	}

	if svc.db != mock {
		t.Error("NewService() did not set db correctly")
	}
}

func TestService_Ping(t *testing.T) {
	tests := []struct {
		name    string
		pingErr error
		wantErr bool
	}{
		{"success", nil, false},
		{"db error", errors.New("db error"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(&mockStore{pingErr: tt.pingErr})

			_, err := svc.Ping(context.Background(), &v1.Empty{})
			if (err != nil) != tt.wantErr {
				t.Errorf("Ping() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestService_SaveRepo(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		path     string
		saveErr  error
		wantErr  bool
		wantCode codes.Code
	}{
		{"success", "https://github.com/user/repo", "/tmp/repo", nil, false, codes.OK},
		{"empty url", "", "/tmp/repo", nil, true, codes.InvalidArgument},
		{"empty path", "https://github.com/user/repo", "", nil, true, codes.InvalidArgument},
		{"invalid url", "://invalid", "/tmp/repo", nil, true, codes.InvalidArgument},
		{"db error", "https://github.com/user/repo", "/tmp/repo", errors.New("db error"), true, codes.Internal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(&mockStore{saveRepoWithWorkspaceErr: tt.saveErr})

			resp, err := svc.SaveRepo(context.Background(), &v1.SaveRepoRequest{
				Url:  tt.url,
				Path: tt.path,
			})
			if (err != nil) != tt.wantErr {
				t.Errorf("SaveRepo() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil {
				st, _ := status.FromError(err)
				if st.Code() != tt.wantCode {
					t.Errorf("SaveRepo() code = %v, want %v", st.Code(), tt.wantCode)
				}
			} else if !resp.GetSuccess() {
				t.Error("SaveRepo() success = false, want true")
			}
		})
	}
}

func TestService_RepoExistsByURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		exists   bool
		dbErr    error
		wantErr  bool
		wantCode codes.Code
	}{
		{"exists", "https://github.com/user/repo", true, nil, false, codes.OK},
		{"not exists", "https://github.com/user/repo", false, nil, false, codes.OK},
		{"empty url", "", false, nil, true, codes.InvalidArgument},
		{"invalid url", "://invalid", false, nil, true, codes.InvalidArgument},
		{"db error", "https://github.com/user/repo", false, errors.New("db error"), true, codes.Internal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(&mockStore{
				repoExistsByURL:    tt.exists,
				repoExistsByURLErr: tt.dbErr,
			})

			resp, err := svc.RepoExistsByURL(context.Background(), &v1.RepoExistsByURLRequest{
				Url: tt.url,
			})
			if (err != nil) != tt.wantErr {
				t.Errorf("RepoExistsByURL() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err == nil && resp.GetExists() != tt.exists {
				t.Errorf("RepoExistsByURL() exists = %v, want %v", resp.GetExists(), tt.exists)
			}
		})
	}
}

func TestService_RepoExistsByPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		exists   bool
		dbErr    error
		wantErr  bool
		wantCode codes.Code
	}{
		{"exists", "/tmp/repo", true, nil, false, codes.OK},
		{"not exists", "/tmp/repo", false, nil, false, codes.OK},
		{"empty path", "", false, nil, true, codes.InvalidArgument},
		{"db error", "/tmp/repo", false, errors.New("db error"), true, codes.Internal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(&mockStore{
				repoExistsByPath:    tt.exists,
				repoExistsByPathErr: tt.dbErr,
			})

			resp, err := svc.RepoExistsByPath(context.Background(), &v1.RepoExistsByPathRequest{
				Path: tt.path,
			})
			if (err != nil) != tt.wantErr {
				t.Errorf("RepoExistsByPath() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err == nil && resp.GetExists() != tt.exists {
				t.Errorf("RepoExistsByPath() exists = %v, want %v", resp.GetExists(), tt.exists)
			}
		})
	}
}

func TestService_InsertRepoIfNotExists(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		path      string
		insertErr error
		wantErr   bool
		wantCode  codes.Code
	}{
		{"success with url", "https://github.com/user/repo", "/tmp/repo", nil, false, codes.OK},
		{"success with path only", "", "/tmp/repo", nil, false, codes.OK},
		{"both empty", "", "", nil, true, codes.InvalidArgument},
		{"invalid url", "://invalid", "/tmp/repo", nil, true, codes.InvalidArgument},
		{"db error", "https://github.com/user/repo", "/tmp/repo", errors.New("db error"), true, codes.Internal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(&mockStore{insertRepoErr: tt.insertErr})

			resp, err := svc.InsertRepoIfNotExists(context.Background(), &v1.InsertRepoIfNotExistsRequest{
				Url:  tt.url,
				Path: tt.path,
			})
			if (err != nil) != tt.wantErr {
				t.Errorf("InsertRepoIfNotExists() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err == nil && !resp.GetInserted() {
				t.Error("InsertRepoIfNotExists() inserted = false, want true")
			}
		})
	}
}

func TestService_GetAllRepos(t *testing.T) {
	repos := []model.Repository{
		{ID: 1, UID: "uid1", URL: "https://github.com/user/repo1"},
		{ID: 2, UID: "uid2", URL: "https://github.com/user/repo2"},
	}

	tests := []struct {
		name    string
		repos   []model.Repository
		dbErr   error
		wantErr bool
		wantLen int
	}{
		{"success", repos, nil, false, 2},
		{"empty", nil, nil, false, 0},
		{"db error", nil, errors.New("db error"), true, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(&mockStore{
				getAllReposResult: tt.repos,
				getAllReposErr:    tt.dbErr,
			})

			resp, err := svc.GetAllRepos(context.Background(), &v1.GetAllReposRequest{})
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAllRepos() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err == nil && len(resp.GetRepositories()) != tt.wantLen {
				t.Errorf("GetAllRepos() len = %d, want %d", len(resp.GetRepositories()), tt.wantLen)
			}
		})
	}
}

func TestService_GetRepos(t *testing.T) {
	repos := []model.Repository{
		{ID: 1, UID: "uid1", URL: "https://github.com/user/repo1", Favorite: true},
	}

	tests := []struct {
		name          string
		favoritesOnly bool
		repos         []model.Repository
		dbErr         error
		wantErr       bool
		wantLen       int
	}{
		{"all repos", false, repos, nil, false, 1},
		{"favorites only", true, repos, nil, false, 1},
		{"db error", false, nil, errors.New("db error"), true, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(&mockStore{
				getReposResult: tt.repos,
				getReposErr:    tt.dbErr,
			})

			resp, err := svc.GetRepos(context.Background(), &v1.GetReposRequest{
				FavoritesOnly: tt.favoritesOnly,
			})
			if (err != nil) != tt.wantErr {
				t.Errorf("GetRepos() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err == nil && len(resp.GetRepositories()) != tt.wantLen {
				t.Errorf("GetRepos() len = %d, want %d", len(resp.GetRepositories()), tt.wantLen)
			}
		})
	}
}

func TestService_SetFavoriteByURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		favorite bool
		dbErr    error
		wantErr  bool
		wantCode codes.Code
	}{
		{"set favorite", "https://github.com/user/repo", true, nil, false, codes.OK},
		{"unset favorite", "https://github.com/user/repo", false, nil, false, codes.OK},
		{"empty url", "", true, nil, true, codes.InvalidArgument},
		{"invalid url", "://invalid", true, nil, true, codes.InvalidArgument},
		{"db error", "https://github.com/user/repo", true, errors.New("db error"), true, codes.Internal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(&mockStore{setFavoriteErr: tt.dbErr})

			resp, err := svc.SetFavoriteByURL(context.Background(), &v1.SetFavoriteRequest{
				Url:      tt.url,
				Favorite: tt.favorite,
			})
			if (err != nil) != tt.wantErr {
				t.Errorf("SetFavoriteByURL() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err == nil && !resp.GetSuccess() {
				t.Error("SetFavoriteByURL() success = false, want true")
			}
		})
	}
}

func TestService_UpdateRepoTimestamp(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		dbErr    error
		wantErr  bool
		wantCode codes.Code
	}{
		{"success", "https://github.com/user/repo", nil, false, codes.OK},
		{"empty url", "", nil, true, codes.InvalidArgument},
		{"db error", "https://github.com/user/repo", errors.New("db error"), true, codes.Internal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(&mockStore{updateTimestampErr: tt.dbErr})

			resp, err := svc.UpdateRepoTimestamp(context.Background(), &v1.UpdateRepoTimestampRequest{
				Url: tt.url,
			})
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateRepoTimestamp() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err == nil && !resp.GetSuccess() {
				t.Error("UpdateRepoTimestamp() success = false, want true")
			}
		})
	}
}

func TestService_RemoveRepoByURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		dbErr    error
		wantErr  bool
		wantCode codes.Code
	}{
		{"success", "https://github.com/user/repo", nil, false, codes.OK},
		{"empty url", "", nil, true, codes.InvalidArgument},
		{"invalid url", "://invalid", nil, true, codes.InvalidArgument},
		{"db error", "https://github.com/user/repo", errors.New("db error"), true, codes.Internal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(&mockStore{removeRepoErr: tt.dbErr})

			resp, err := svc.RemoveRepoByURL(context.Background(), &v1.RemoveRepoByURLRequest{
				Url: tt.url,
			})
			if (err != nil) != tt.wantErr {
				t.Errorf("RemoveRepoByURL() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err == nil && !resp.GetSuccess() {
				t.Error("RemoveRepoByURL() success = false, want true")
			}
		})
	}
}

func TestService_GetConfig(t *testing.T) {
	cfg := &model.Config{
		DefaultCloneDir: "/home/user/repos",
		Editor:          "nvim",
		ServerPort:      8080,
	}

	tests := []struct {
		name    string
		config  *model.Config
		dbErr   error
		wantErr bool
	}{
		{"success", cfg, nil, false},
		{"db error", nil, errors.New("db error"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(&mockStore{
				getConfigResult: tt.config,
				getConfigErr:    tt.dbErr,
			})

			resp, err := svc.GetConfig(context.Background(), &v1.GetConfigRequest{})
			if (err != nil) != tt.wantErr {
				t.Errorf("GetConfig() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err == nil && resp.GetConfig() == nil {
				t.Error("GetConfig() config is nil")
			}
		})
	}
}

func TestService_SaveConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *v1.Config
		dbErr   error
		wantErr bool
	}{
		{"success", &v1.Config{Editor: "vim"}, nil, false},
		{"nil config", nil, nil, true},
		{"db error", &v1.Config{Editor: "vim"}, errors.New("db error"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(&mockStore{saveConfigErr: tt.dbErr})

			resp, err := svc.SaveConfig(context.Background(), &v1.SaveConfigRequest{
				Config: tt.config,
			})
			if (err != nil) != tt.wantErr {
				t.Errorf("SaveConfig() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err == nil && !resp.GetSuccess() {
				t.Error("SaveConfig() success = false, want true")
			}
		})
	}
}

func TestService_ContextCancellation(t *testing.T) {
	svc := NewService(&mockStore{})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Test SaveRepo with canceled context
	_, err := svc.SaveRepo(ctx, &v1.SaveRepoRequest{Url: "https://example.com", Path: "/tmp"})
	if err == nil {
		t.Error("SaveRepo with canceled context should return error")
	}

	st, _ := status.FromError(err)
	if st.Code() != codes.Canceled {
		t.Errorf("SaveRepo canceled context code = %v, want %v", st.Code(), codes.Canceled)
	}
}
