//go:build !bolt

package store

import (
	"net/url"
	"path/filepath"

	"github.com/inovacc/clonr/internal/model"
	"github.com/inovacc/clonr/internal/params"
	"github.com/inovacc/clonr/internal/standalone"
	"github.com/inovacc/clonr/internal/store/sqlite"
)

// SQLiteWrapper wraps the sqlite.Store to implement the Store interface.
type SQLiteWrapper struct {
	store *sqlite.Store
}

func initDB() (Store, error) {
	path := filepath.Join(params.AppdataDir, "clonr.db")
	store, err := sqlite.New(path)
	if err != nil {
		return nil, err
	}
	return &SQLiteWrapper{store: store}, nil
}

func (w *SQLiteWrapper) Ping() error {
	return w.store.Ping()
}

func (w *SQLiteWrapper) SaveRepo(u *url.URL, path string) error {
	return w.store.SaveRepo(u, path)
}

func (w *SQLiteWrapper) SaveRepoWithWorkspace(u *url.URL, path, workspace string) error {
	return w.store.SaveRepoWithWorkspace(u, path, workspace)
}

func (w *SQLiteWrapper) RepoExistsByURL(u *url.URL) (bool, error) {
	return w.store.RepoExistsByURL(u)
}

func (w *SQLiteWrapper) RepoExistsByPath(path string) (bool, error) {
	return w.store.RepoExistsByPath(path)
}

func (w *SQLiteWrapper) InsertRepoIfNotExists(u *url.URL, path string) error {
	return w.store.InsertRepoIfNotExists(u, path)
}

func (w *SQLiteWrapper) GetAllRepos() ([]model.Repository, error) {
	repos, err := w.store.GetAllRepos()
	if err != nil {
		return nil, err
	}
	result := make([]model.Repository, len(repos))
	for i, r := range repos {
		result[i] = *r
	}
	return result, nil
}

func (w *SQLiteWrapper) GetRepos(workspace string, favoritesOnly bool) ([]model.Repository, error) {
	repos, err := w.store.GetRepos(workspace, favoritesOnly)
	if err != nil {
		return nil, err
	}
	result := make([]model.Repository, len(repos))
	for i, r := range repos {
		result[i] = *r
	}
	return result, nil
}

func (w *SQLiteWrapper) SetFavoriteByURL(urlStr string, fav bool) error {
	return w.store.SetFavoriteByURL(urlStr, fav)
}

func (w *SQLiteWrapper) UpdateRepoTimestamp(urlStr string) error {
	return w.store.UpdateRepoTimestamp(urlStr)
}

func (w *SQLiteWrapper) RemoveRepoByURL(u *url.URL) error {
	return w.store.RemoveRepoByURL(u)
}

func (w *SQLiteWrapper) GetConfig() (*model.Config, error) {
	return w.store.GetConfig()
}

func (w *SQLiteWrapper) SaveConfig(cfg *model.Config) error {
	return w.store.SaveConfig(cfg)
}

func (w *SQLiteWrapper) SaveProfile(profile *model.Profile) error {
	return w.store.SaveProfile(profile)
}

func (w *SQLiteWrapper) GetProfile(name string) (*model.Profile, error) {
	return w.store.GetProfile(name)
}

func (w *SQLiteWrapper) GetActiveProfile() (*model.Profile, error) {
	return w.store.GetActiveProfile()
}

func (w *SQLiteWrapper) SetActiveProfile(name string) error {
	return w.store.SetActiveProfile(name)
}

func (w *SQLiteWrapper) ListProfiles() ([]model.Profile, error) {
	profiles, err := w.store.ListProfiles()
	if err != nil {
		return nil, err
	}
	result := make([]model.Profile, len(profiles))
	for i, p := range profiles {
		result[i] = *p
	}
	return result, nil
}

func (w *SQLiteWrapper) DeleteProfile(name string) error {
	return w.store.DeleteProfile(name)
}

func (w *SQLiteWrapper) ProfileExists(name string) (bool, error) {
	return w.store.ProfileExists(name)
}

func (w *SQLiteWrapper) SaveWorkspace(workspace *model.Workspace) error {
	return w.store.SaveWorkspace(workspace)
}

func (w *SQLiteWrapper) GetWorkspace(name string) (*model.Workspace, error) {
	return w.store.GetWorkspace(name)
}

func (w *SQLiteWrapper) GetActiveWorkspace() (*model.Workspace, error) {
	return w.store.GetActiveWorkspace()
}

func (w *SQLiteWrapper) SetActiveWorkspace(name string) error {
	return w.store.SetActiveWorkspace(name)
}

func (w *SQLiteWrapper) ListWorkspaces() ([]model.Workspace, error) {
	workspaces, err := w.store.ListWorkspaces()
	if err != nil {
		return nil, err
	}
	result := make([]model.Workspace, len(workspaces))
	for i, ws := range workspaces {
		result[i] = *ws
	}
	return result, nil
}

func (w *SQLiteWrapper) DeleteWorkspace(name string) error {
	return w.store.DeleteWorkspace(name)
}

func (w *SQLiteWrapper) WorkspaceExists(name string) (bool, error) {
	return w.store.WorkspaceExists(name)
}

func (w *SQLiteWrapper) GetReposByWorkspace(workspace string) ([]string, error) {
	repos, err := w.store.GetReposByWorkspace(workspace)
	if err != nil {
		return nil, err
	}
	result := make([]string, len(repos))
	for i, r := range repos {
		result[i] = r.URL
	}
	return result, nil
}

func (w *SQLiteWrapper) UpdateRepoWorkspace(urlStr, workspace string) error {
	return w.store.UpdateRepoWorkspace(urlStr, workspace)
}

// Standalone operations

func (w *SQLiteWrapper) GetStandaloneConfig() (*standalone.StandaloneConfig, error) {
	return w.store.GetStandaloneConfig()
}

func (w *SQLiteWrapper) SaveStandaloneConfig(config *standalone.StandaloneConfig) error {
	return w.store.SaveStandaloneConfig(config)
}

func (w *SQLiteWrapper) DeleteStandaloneConfig() error {
	return w.store.DeleteStandaloneConfig()
}

func (w *SQLiteWrapper) GetStandaloneClients() ([]standalone.Client, error) {
	clients, err := w.store.GetStandaloneClients()
	if err != nil {
		return nil, err
	}
	result := make([]standalone.Client, len(clients))
	for i, c := range clients {
		result[i] = *c
	}
	return result, nil
}

func (w *SQLiteWrapper) SaveStandaloneClient(client *standalone.Client) error {
	return w.store.SaveStandaloneClient(client)
}

func (w *SQLiteWrapper) DeleteStandaloneClient(id string) error {
	return w.store.DeleteStandaloneClient(id)
}

func (w *SQLiteWrapper) GetStandaloneConnection(name string) (*standalone.StandaloneConnection, error) {
	return w.store.GetStandaloneConnection(name)
}

func (w *SQLiteWrapper) ListStandaloneConnections() ([]standalone.StandaloneConnection, error) {
	conns, err := w.store.ListStandaloneConnections()
	if err != nil {
		return nil, err
	}
	result := make([]standalone.StandaloneConnection, len(conns))
	for i, c := range conns {
		result[i] = *c
	}
	return result, nil
}

func (w *SQLiteWrapper) SaveStandaloneConnection(conn *standalone.StandaloneConnection) error {
	return w.store.SaveStandaloneConnection(conn)
}

func (w *SQLiteWrapper) DeleteStandaloneConnection(name string) error {
	return w.store.DeleteStandaloneConnection(name)
}

func (w *SQLiteWrapper) GetServerEncryptionConfig() (*standalone.ServerEncryptionConfig, error) {
	return w.store.GetServerEncryptionConfig()
}

func (w *SQLiteWrapper) SaveServerEncryptionConfig(config *standalone.ServerEncryptionConfig) error {
	return w.store.SaveServerEncryptionConfig(config)
}

func (w *SQLiteWrapper) GetSyncedData(connectionName, dataType, name string) (*standalone.SyncedData, error) {
	return w.store.GetSyncedData(connectionName, dataType, name)
}

func (w *SQLiteWrapper) ListSyncedData(connectionName string) ([]standalone.SyncedData, error) {
	data, err := w.store.ListSyncedData(connectionName)
	if err != nil {
		return nil, err
	}
	result := make([]standalone.SyncedData, len(data))
	for i, d := range data {
		result[i] = *d
	}
	return result, nil
}

func (w *SQLiteWrapper) ListSyncedDataByState(state standalone.SyncState) ([]standalone.SyncedData, error) {
	data, err := w.store.ListSyncedDataByState(state)
	if err != nil {
		return nil, err
	}
	result := make([]standalone.SyncedData, len(data))
	for i, d := range data {
		result[i] = *d
	}
	return result, nil
}

func (w *SQLiteWrapper) SaveSyncedData(data *standalone.SyncedData) error {
	return w.store.SaveSyncedData(data)
}

func (w *SQLiteWrapper) DeleteSyncedData(connectionName, dataType, name string) error {
	return w.store.DeleteSyncedData(connectionName, dataType, name)
}

func (w *SQLiteWrapper) SavePendingRegistration(reg *standalone.ClientRegistration) error {
	return w.store.SavePendingRegistration(reg)
}

func (w *SQLiteWrapper) GetPendingRegistration(clientID string) (*standalone.ClientRegistration, error) {
	return w.store.GetPendingRegistration(clientID)
}

func (w *SQLiteWrapper) ListPendingRegistrations() ([]*standalone.ClientRegistration, error) {
	return w.store.ListPendingRegistrations()
}

func (w *SQLiteWrapper) RemovePendingRegistration(clientID string) error {
	return w.store.RemovePendingRegistration(clientID)
}

func (w *SQLiteWrapper) SaveRegisteredClient(client *standalone.RegisteredClient) error {
	return w.store.SaveRegisteredClient(client)
}

func (w *SQLiteWrapper) GetRegisteredClient(clientID string) (*standalone.RegisteredClient, error) {
	return w.store.GetRegisteredClient(clientID)
}

func (w *SQLiteWrapper) ListRegisteredClients() ([]*standalone.RegisteredClient, error) {
	return w.store.ListRegisteredClients()
}

func (w *SQLiteWrapper) DeleteRegisteredClient(clientID string) error {
	return w.store.DeleteRegisteredClient(clientID)
}

// Docker profile operations

func (w *SQLiteWrapper) SaveDockerProfile(profile *model.DockerProfile) error {
	return w.store.SaveDockerProfile(profile)
}

func (w *SQLiteWrapper) GetDockerProfile(name string) (*model.DockerProfile, error) {
	return w.store.GetDockerProfile(name)
}

func (w *SQLiteWrapper) ListDockerProfiles() ([]model.DockerProfile, error) {
	profiles, err := w.store.ListDockerProfiles()
	if err != nil {
		return nil, err
	}
	result := make([]model.DockerProfile, len(profiles))
	for i, p := range profiles {
		result[i] = *p
	}
	return result, nil
}

func (w *SQLiteWrapper) DeleteDockerProfile(name string) error {
	return w.store.DeleteDockerProfile(name)
}

func (w *SQLiteWrapper) DockerProfileExists(name string) (bool, error) {
	return w.store.DockerProfileExists(name)
}

// Sealed key operations

func (w *SQLiteWrapper) GetSealedKey() (*SealedKeyData, error) {
	sqliteKey, err := w.store.GetSealedKey()
	if err != nil {
		return nil, err
	}
	if sqliteKey == nil {
		return nil, nil
	}
	return &SealedKeyData{
		SealedData:   sqliteKey.SealedData,
		Version:      sqliteKey.Version,
		KeyType:      sqliteKey.KeyType,
		Metadata:     sqliteKey.Metadata,
		CreatedAt:    sqliteKey.CreatedAt,
		RotatedAt:    sqliteKey.RotatedAt,
		LastAccessed: sqliteKey.LastAccessed,
	}, nil
}

func (w *SQLiteWrapper) SaveSealedKey(data *SealedKeyData) error {
	return w.store.SaveSealedKey(&sqlite.SealedKeyData{
		SealedData:   data.SealedData,
		Version:      data.Version,
		KeyType:      data.KeyType,
		Metadata:     data.Metadata,
		CreatedAt:    data.CreatedAt,
		RotatedAt:    data.RotatedAt,
		LastAccessed: data.LastAccessed,
	})
}

func (w *SQLiteWrapper) DeleteSealedKey() error {
	return w.store.DeleteSealedKey()
}

func (w *SQLiteWrapper) HasSealedKey() (bool, error) {
	return w.store.HasSealedKey()
}
