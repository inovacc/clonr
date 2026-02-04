package store

import (
	"net/url"
	"sync"
	"time"

	"github.com/inovacc/clonr/internal/model"
	"github.com/inovacc/clonr/internal/standalone"
)

// SealedKeyData represents TPM-sealed encryption key data stored in the database
type SealedKeyData struct {
	SealedData   []byte            `json:"sealed_data"`
	Version      int               `json:"version"`
	KeyType      string            `json:"key_type"` // "tpm", "password", "software"
	Metadata     map[string]string `json:"metadata"`
	CreatedAt    time.Time         `json:"created_at"`
	RotatedAt    time.Time         `json:"rotated_at,omitempty"`
	LastAccessed time.Time         `json:"last_accessed,omitempty"`
}

// Store defines the database operations used by the app.
//
//nolint:interfacebloat // all methods are required for database operations
type Store interface {
	Ping() error
	SaveRepo(u *url.URL, path string) error
	SaveRepoWithWorkspace(u *url.URL, path string, workspace string) error
	RepoExistsByURL(u *url.URL) (bool, error)
	RepoExistsByPath(path string) (bool, error)
	InsertRepoIfNotExists(u *url.URL, path string) error
	GetAllRepos() ([]model.Repository, error)
	GetRepos(workspace string, favoritesOnly bool) ([]model.Repository, error)
	SetFavoriteByURL(urlStr string, fav bool) error
	UpdateRepoTimestamp(urlStr string) error
	RemoveRepoByURL(u *url.URL) error
	GetConfig() (*model.Config, error)
	SaveConfig(cfg *model.Config) error

	// Profile operations
	SaveProfile(profile *model.Profile) error
	GetProfile(name string) (*model.Profile, error)
	GetActiveProfile() (*model.Profile, error)
	SetActiveProfile(name string) error
	ListProfiles() ([]model.Profile, error)
	DeleteProfile(name string) error
	ProfileExists(name string) (bool, error)

	// Docker profile operations
	SaveDockerProfile(profile *model.DockerProfile) error
	GetDockerProfile(name string) (*model.DockerProfile, error)
	ListDockerProfiles() ([]model.DockerProfile, error)
	DeleteDockerProfile(name string) error
	DockerProfileExists(name string) (bool, error)

	// Workspace operations
	SaveWorkspace(workspace *model.Workspace) error
	GetWorkspace(name string) (*model.Workspace, error)
	GetActiveWorkspace() (*model.Workspace, error)
	SetActiveWorkspace(name string) error
	ListWorkspaces() ([]model.Workspace, error)
	DeleteWorkspace(name string) error
	WorkspaceExists(name string) (bool, error)
	GetReposByWorkspace(workspace string) ([]string, error)
	UpdateRepoWorkspace(urlStr string, workspace string) error

	// Standalone operations
	GetStandaloneConfig() (*standalone.StandaloneConfig, error)
	SaveStandaloneConfig(config *standalone.StandaloneConfig) error
	DeleteStandaloneConfig() error
	GetStandaloneClients() ([]standalone.Client, error)
	SaveStandaloneClient(client *standalone.Client) error
	DeleteStandaloneClient(id string) error

	// Standalone connections (destination side)
	GetStandaloneConnection(name string) (*standalone.StandaloneConnection, error)
	ListStandaloneConnections() ([]standalone.StandaloneConnection, error)
	SaveStandaloneConnection(conn *standalone.StandaloneConnection) error
	DeleteStandaloneConnection(name string) error

	// Server encryption config
	GetServerEncryptionConfig() (*standalone.ServerEncryptionConfig, error)
	SaveServerEncryptionConfig(config *standalone.ServerEncryptionConfig) error

	// Synced data (encrypted storage)
	GetSyncedData(connectionName, dataType, name string) (*standalone.SyncedData, error)
	ListSyncedData(connectionName string) ([]standalone.SyncedData, error)
	ListSyncedDataByState(state standalone.SyncState) ([]standalone.SyncedData, error)
	SaveSyncedData(data *standalone.SyncedData) error
	DeleteSyncedData(connectionName, dataType, name string) error

	// Client registration (server side)
	SavePendingRegistration(reg *standalone.ClientRegistration) error
	GetPendingRegistration(clientID string) (*standalone.ClientRegistration, error)
	ListPendingRegistrations() ([]*standalone.ClientRegistration, error)
	RemovePendingRegistration(clientID string) error

	// Registered clients (server side)
	SaveRegisteredClient(client *standalone.RegisteredClient) error
	GetRegisteredClient(clientID string) (*standalone.RegisteredClient, error)
	ListRegisteredClients() ([]*standalone.RegisteredClient, error)
	DeleteRegisteredClient(clientID string) error

	// Sealed key operations (TPM-sealed encryption key stored in database)
	GetSealedKey() (*SealedKeyData, error)
	SaveSealedKey(data *SealedKeyData) error
	DeleteSealedKey() error
	HasSealedKey() (bool, error)

	// Slack configuration operations
	GetSlackConfig() (*model.SlackConfig, error)
	SaveSlackConfig(config *model.SlackConfig) error
	DeleteSlackConfig() error
	EnableSlackNotifications() error
	DisableSlackNotifications() error

	// Slack account operations
	SaveSlackAccount(account *model.SlackAccount) error
	GetSlackAccount(name string) (*model.SlackAccount, error)
	GetActiveSlackAccount() (*model.SlackAccount, error)
	SetActiveSlackAccount(name string) error
	ListSlackAccounts() ([]*model.SlackAccount, error)
	DeleteSlackAccount(name string) error
	SlackAccountExists(name string) (bool, error)
}

var (
	once sync.Once
	db   Store
)

// GetDB returns the initialized database store.
func GetDB() Store {
	once.Do(lazyInit)

	return db
}

func lazyInit() {
	instance, err := initDB()
	if err != nil {
		panic(err)
	}

	_ = instance.Ping()
	db = instance
}
