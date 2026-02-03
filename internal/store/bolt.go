//go:build bolt

package store

import (
	"encoding/json"
	"errors"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/inovacc/clonr/internal/model"
	"github.com/inovacc/clonr/internal/params"
	"github.com/inovacc/clonr/internal/standalone"
	"go.etcd.io/bbolt"
)

const (
	boltBucketRepos          = "repos"           // key: URL -> Repository JSON
	boltBucketPaths          = "paths"           // key: Path -> URL string
	boltBucketConfig         = "config"          // key: "config" -> Config JSON
	boltBucketProfiles       = "profiles"        // key: name -> Profile JSON
	boltBucketDockerProfiles = "docker_profiles" // key: name -> DockerProfile JSON
	boltBucketWorkspaces     = "workspaces"      // key: name -> Workspace JSON
	boltBucketStandalone     = "standalone"      // key: "config" -> StandaloneConfig, "client:<id>" -> Client, "encryption" -> ServerEncryptionConfig
	boltBucketConnections    = "connections"     // key: name -> StandaloneConnection (destination side)
	boltBucketSyncedData     = "synced_data"     // key: "connection:type:name" -> SyncedData (encrypted until decrypted)
)

type Bolt struct {
	storage *bbolt.DB
}

// NewBolt creates a new Bolt database at the specified path.
// This is primarily exposed for testing purposes.
func NewBolt(path string) (*Bolt, error) {
	instance, err := bbolt.Open(path, 0600, &bbolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, err
	}

	if err := instance.Update(func(tx *bbolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists([]byte(boltBucketRepos)); err != nil {
			return err
		}

		if _, err := tx.CreateBucketIfNotExists([]byte(boltBucketPaths)); err != nil {
			return err
		}

		if _, err := tx.CreateBucketIfNotExists([]byte(boltBucketConfig)); err != nil {
			return err
		}

		if _, err := tx.CreateBucketIfNotExists([]byte(boltBucketProfiles)); err != nil {
			return err
		}

		if _, err := tx.CreateBucketIfNotExists([]byte(boltBucketDockerProfiles)); err != nil {
			return err
		}

		if _, err := tx.CreateBucketIfNotExists([]byte(boltBucketWorkspaces)); err != nil {
			return err
		}

		if _, err := tx.CreateBucketIfNotExists([]byte(boltBucketStandalone)); err != nil {
			return err
		}

		if _, err := tx.CreateBucketIfNotExists([]byte(boltBucketConnections)); err != nil {
			return err
		}

		if _, err := tx.CreateBucketIfNotExists([]byte(boltBucketSyncedData)); err != nil {
			return err
		}

		return nil
	}); err != nil {
		_ = instance.Close()

		return nil, err
	}

	return &Bolt{storage: instance}, nil
}

// Close closes the database.
func (b *Bolt) Close() error {
	return b.storage.Close()
}

func initDB() (Store, error) {
	path := filepath.Join(params.AppdataDir, "clonr.bolt")

	instance, err := bbolt.Open(path, 0600, &bbolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, err
	}

	if err := instance.Update(func(tx *bbolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists([]byte(boltBucketRepos)); err != nil {
			return err
		}

		if _, err := tx.CreateBucketIfNotExists([]byte(boltBucketPaths)); err != nil {
			return err
		}

		if _, err := tx.CreateBucketIfNotExists([]byte(boltBucketConfig)); err != nil {
			return err
		}

		if _, err := tx.CreateBucketIfNotExists([]byte(boltBucketProfiles)); err != nil {
			return err
		}

		if _, err := tx.CreateBucketIfNotExists([]byte(boltBucketDockerProfiles)); err != nil {
			return err
		}

		if _, err := tx.CreateBucketIfNotExists([]byte(boltBucketWorkspaces)); err != nil {
			return err
		}

		if _, err := tx.CreateBucketIfNotExists([]byte(boltBucketStandalone)); err != nil {
			return err
		}

		if _, err := tx.CreateBucketIfNotExists([]byte(boltBucketConnections)); err != nil {
			return err
		}

		if _, err := tx.CreateBucketIfNotExists([]byte(boltBucketSyncedData)); err != nil {
			return err
		}

		return nil
	}); err != nil {
		_ = instance.Close()

		return nil, err
	}

	return &Bolt{storage: instance}, nil
}

func (b *Bolt) Ping() error {
	return b.storage.View(func(tx *bbolt.Tx) error {
		return nil
	})
}

func (b *Bolt) SaveRepo(u *url.URL, path string) error {
	return b.SaveRepoWithWorkspace(u, path, "")
}

func (b *Bolt) SaveRepoWithWorkspace(u *url.URL, path string, workspace string) error {
	if u == nil {
		return errors.New("url is required")
	}

	repo := model.Repository{
		UID:       uuid.New().String(),
		URL:       u.String(),
		Path:      path,
		Workspace: workspace,
		ClonedAt:  time.Now(),
	}

	data, err := json.Marshal(&repo)
	if err != nil {
		return err
	}

	return b.storage.Update(func(tx *bbolt.Tx) error {
		var (
			repos = tx.Bucket([]byte(boltBucketRepos))
			paths = tx.Bucket([]byte(boltBucketPaths))
		)

		if repos.Get([]byte(repo.URL)) != nil {
			return nil
		}

		if paths.Get([]byte(repo.Path)) != nil {
			return nil
		}

		if err := repos.Put([]byte(repo.URL), data); err != nil {
			return err
		}

		if err := paths.Put([]byte(repo.Path), []byte(repo.URL)); err != nil {
			return err
		}

		return nil
	})
}

func (b *Bolt) RepoExistsByURL(u *url.URL) (bool, error) {
	var exists bool

	err := b.storage.View(func(tx *bbolt.Tx) error {
		repos := tx.Bucket([]byte(boltBucketRepos))
		exists = repos.Get([]byte(u.String())) != nil

		return nil
	})

	return exists, err
}

func (b *Bolt) RepoExistsByPath(path string) (bool, error) {
	var exists bool

	err := b.storage.View(func(tx *bbolt.Tx) error {
		paths := tx.Bucket([]byte(boltBucketPaths))
		exists = paths.Get([]byte(path)) != nil

		return nil
	})

	return exists, err
}

func (b *Bolt) InsertRepoIfNotExists(u *url.URL, path string) error {
	if u != nil {
		ex, err := b.RepoExistsByURL(u)
		if err != nil {
			return err
		}

		if ex {
			return nil
		}
	}

	ex, err := b.RepoExistsByPath(path)
	if err != nil {
		return err
	}

	if ex {
		return nil
	}

	return b.SaveRepo(u, path)
}

func (b *Bolt) GetAllRepos() ([]model.Repository, error) {
	var out []model.Repository

	err := b.storage.View(func(tx *bbolt.Tx) error {
		repos := tx.Bucket([]byte(boltBucketRepos))

		return repos.ForEach(func(k, v []byte) error {
			var r model.Repository

			if err := json.Unmarshal(v, &r); err != nil {
				return err
			}

			out = append(out, r)

			return nil
		})
	})

	return out, err
}

func (b *Bolt) GetRepos(workspace string, favoritesOnly bool) ([]model.Repository, error) {
	var out []model.Repository

	err := b.storage.View(func(tx *bbolt.Tx) error {
		repos := tx.Bucket([]byte(boltBucketRepos))

		return repos.ForEach(func(k, v []byte) error {
			var r model.Repository

			if err := json.Unmarshal(v, &r); err != nil {
				return err
			}

			// Filter by workspace if specified
			if workspace != "" && r.Workspace != workspace {
				return nil
			}

			// Filter by favorites if requested
			if favoritesOnly && !r.Favorite {
				return nil
			}

			out = append(out, r)

			return nil
		})
	})

	return out, err
}

func (b *Bolt) SetFavoriteByURL(urlStr string, fav bool) error {
	return b.storage.Update(func(tx *bbolt.Tx) error {
		repos := tx.Bucket([]byte(boltBucketRepos))

		v := repos.Get([]byte(urlStr))

		if v == nil {
			return nil
		}

		var r model.Repository

		if err := json.Unmarshal(v, &r); err != nil {
			return err
		}

		r.Favorite = fav

		data, err := json.Marshal(&r)
		if err != nil {
			return err
		}

		return repos.Put([]byte(urlStr), data)
	})
}

func (b *Bolt) UpdateRepoTimestamp(urlStr string) error {
	return b.storage.Update(func(tx *bbolt.Tx) error {
		repos := tx.Bucket([]byte(boltBucketRepos))

		v := repos.Get([]byte(urlStr))

		if v == nil {
			return nil
		}

		var r model.Repository

		if err := json.Unmarshal(v, &r); err != nil {
			return err
		}

		r.UpdatedAt = time.Now()

		data, err := json.Marshal(&r)
		if err != nil {
			return err
		}

		return repos.Put([]byte(urlStr), data)
	})
}

func (b *Bolt) RemoveRepoByURL(u *url.URL) error {
	return b.storage.Update(func(tx *bbolt.Tx) error {
		repos := tx.Bucket([]byte(boltBucketRepos))
		paths := tx.Bucket([]byte(boltBucketPaths))

		v := repos.Get([]byte(u.String()))
		if v == nil {
			return nil
		}

		var r model.Repository

		if err := json.Unmarshal(v, &r); err != nil {
			return err
		}

		if err := repos.Delete([]byte(u.String())); err != nil {
			return err
		}

		if r.Path != "" {
			_ = paths.Delete([]byte(r.Path))
		}

		return nil
	})
}

func (b *Bolt) GetConfig() (*model.Config, error) {
	var cfg *model.Config

	err := b.storage.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(boltBucketConfig))
		v := bucket.Get([]byte("config"))

		if v == nil {
			// Return default config if not found
			defaultCfg := model.DefaultConfig()
			cfg = &defaultCfg

			return nil
		}

		var c model.Config
		if err := json.Unmarshal(v, &c); err != nil {
			return err
		}

		cfg = &c

		return nil
	})

	return cfg, err
}

func (b *Bolt) SaveConfig(cfg *model.Config) error {
	if cfg == nil {
		return errors.New("config is required")
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}

	return b.storage.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(boltBucketConfig))

		return bucket.Put([]byte("config"), data)
	})
}

// SaveProfile saves or updates a profile
func (b *Bolt) SaveProfile(profile *model.Profile) error {
	if profile == nil {
		return errors.New("profile is required")
	}

	if profile.Name == "" {
		return errors.New("profile name is required")
	}

	data, err := json.Marshal(profile)
	if err != nil {
		return err
	}

	return b.storage.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(boltBucketProfiles))

		return bucket.Put([]byte(profile.Name), data)
	})
}

// GetProfile retrieves a profile by name
func (b *Bolt) GetProfile(name string) (*model.Profile, error) {
	var profile *model.Profile

	err := b.storage.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(boltBucketProfiles))
		v := bucket.Get([]byte(name))

		if v == nil {
			return nil
		}

		var p model.Profile
		if err := json.Unmarshal(v, &p); err != nil {
			return err
		}

		profile = &p

		return nil
	})

	return profile, err
}

// GetActiveProfile retrieves the default profile (called "active" for gRPC compatibility)
func (b *Bolt) GetActiveProfile() (*model.Profile, error) {
	var profile *model.Profile

	err := b.storage.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(boltBucketProfiles))

		return bucket.ForEach(func(k, v []byte) error {
			var p model.Profile
			if err := json.Unmarshal(v, &p); err != nil {
				return err
			}

			if p.Default {
				profile = &p

				return nil
			}

			return nil
		})
	})

	return profile, err
}

// SetActiveProfile sets the default profile by name (called "active" for gRPC compatibility)
func (b *Bolt) SetActiveProfile(name string) error {
	return b.storage.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(boltBucketProfiles))

		// First, verify the profile exists
		v := bucket.Get([]byte(name))
		if v == nil {
			return errors.New("profile not found")
		}

		// Clear default from all profiles and set on the specified one
		if err := bucket.ForEach(func(k, val []byte) error {
			var p model.Profile
			if err := json.Unmarshal(val, &p); err != nil {
				return err
			}

			p.Default = string(k) == name

			if p.Default {
				p.LastUsedAt = time.Now()
			}

			data, err := json.Marshal(&p)
			if err != nil {
				return err
			}

			return bucket.Put(k, data)
		}); err != nil {
			return err
		}

		return nil
	})
}

// ListProfiles retrieves all profiles
func (b *Bolt) ListProfiles() ([]model.Profile, error) {
	var profiles []model.Profile

	err := b.storage.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(boltBucketProfiles))

		return bucket.ForEach(func(k, v []byte) error {
			var p model.Profile
			if err := json.Unmarshal(v, &p); err != nil {
				return err
			}

			profiles = append(profiles, p)

			return nil
		})
	})

	return profiles, err
}

// DeleteProfile removes a profile by name
func (b *Bolt) DeleteProfile(name string) error {
	return b.storage.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(boltBucketProfiles))

		return bucket.Delete([]byte(name))
	})
}

// ProfileExists checks if a profile exists by name
func (b *Bolt) ProfileExists(name string) (bool, error) {
	var exists bool

	err := b.storage.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(boltBucketProfiles))
		exists = bucket.Get([]byte(name)) != nil

		return nil
	})

	return exists, err
}

// Docker profile operations

// SaveDockerProfile saves or updates a docker profile
func (b *Bolt) SaveDockerProfile(profile *model.DockerProfile) error {
	if profile == nil {
		return errors.New("docker profile is required")
	}

	if profile.Name == "" {
		return errors.New("docker profile name is required")
	}

	data, err := json.Marshal(profile)
	if err != nil {
		return err
	}

	return b.storage.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(boltBucketDockerProfiles))

		return bucket.Put([]byte(profile.Name), data)
	})
}

// GetDockerProfile retrieves a docker profile by name
func (b *Bolt) GetDockerProfile(name string) (*model.DockerProfile, error) {
	var profile *model.DockerProfile

	err := b.storage.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(boltBucketDockerProfiles))
		v := bucket.Get([]byte(name))

		if v == nil {
			return nil
		}

		var p model.DockerProfile
		if err := json.Unmarshal(v, &p); err != nil {
			return err
		}

		profile = &p

		return nil
	})

	return profile, err
}

// ListDockerProfiles retrieves all docker profiles
func (b *Bolt) ListDockerProfiles() ([]model.DockerProfile, error) {
	var profiles []model.DockerProfile

	err := b.storage.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(boltBucketDockerProfiles))

		return bucket.ForEach(func(k, v []byte) error {
			var p model.DockerProfile
			if err := json.Unmarshal(v, &p); err != nil {
				return err
			}

			profiles = append(profiles, p)

			return nil
		})
	})

	return profiles, err
}

// DeleteDockerProfile removes a docker profile by name
func (b *Bolt) DeleteDockerProfile(name string) error {
	return b.storage.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(boltBucketDockerProfiles))

		return bucket.Delete([]byte(name))
	})
}

// DockerProfileExists checks if a docker profile exists by name
func (b *Bolt) DockerProfileExists(name string) (bool, error) {
	var exists bool

	err := b.storage.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(boltBucketDockerProfiles))
		exists = bucket.Get([]byte(name)) != nil

		return nil
	})

	return exists, err
}

// SaveWorkspace saves or updates a workspace
func (b *Bolt) SaveWorkspace(workspace *model.Workspace) error {
	if workspace == nil {
		return errors.New("workspace is required")
	}

	if workspace.Name == "" {
		return errors.New("workspace name is required")
	}

	data, err := json.Marshal(workspace)
	if err != nil {
		return err
	}

	return b.storage.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(boltBucketWorkspaces))

		return bucket.Put([]byte(workspace.Name), data)
	})
}

// GetWorkspace retrieves a workspace by name
func (b *Bolt) GetWorkspace(name string) (*model.Workspace, error) {
	var workspace *model.Workspace

	err := b.storage.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(boltBucketWorkspaces))
		v := bucket.Get([]byte(name))

		if v == nil {
			return nil
		}

		var w model.Workspace
		if err := json.Unmarshal(v, &w); err != nil {
			return err
		}

		workspace = &w

		return nil
	})

	return workspace, err
}

// GetActiveWorkspace retrieves the currently active workspace
func (b *Bolt) GetActiveWorkspace() (*model.Workspace, error) {
	var workspace *model.Workspace

	err := b.storage.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(boltBucketWorkspaces))

		return bucket.ForEach(func(k, v []byte) error {
			var w model.Workspace
			if err := json.Unmarshal(v, &w); err != nil {
				return err
			}

			if w.Active {
				workspace = &w

				return nil
			}

			return nil
		})
	})

	return workspace, err
}

// SetActiveWorkspace sets the active workspace by name
func (b *Bolt) SetActiveWorkspace(name string) error {
	return b.storage.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(boltBucketWorkspaces))

		// First, verify the workspace exists
		v := bucket.Get([]byte(name))
		if v == nil {
			return errors.New("workspace not found")
		}

		// Deactivate all workspaces and activate the specified one
		if err := bucket.ForEach(func(k, val []byte) error {
			var w model.Workspace
			if err := json.Unmarshal(val, &w); err != nil {
				return err
			}

			w.Active = string(k) == name

			if w.Active {
				w.UpdatedAt = time.Now()
			}

			data, err := json.Marshal(&w)
			if err != nil {
				return err
			}

			return bucket.Put(k, data)
		}); err != nil {
			return err
		}

		return nil
	})
}

// ListWorkspaces retrieves all workspaces
func (b *Bolt) ListWorkspaces() ([]model.Workspace, error) {
	var workspaces []model.Workspace

	err := b.storage.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(boltBucketWorkspaces))

		return bucket.ForEach(func(k, v []byte) error {
			var w model.Workspace
			if err := json.Unmarshal(v, &w); err != nil {
				return err
			}

			workspaces = append(workspaces, w)

			return nil
		})
	})

	return workspaces, err
}

// DeleteWorkspace removes a workspace by name
func (b *Bolt) DeleteWorkspace(name string) error {
	return b.storage.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(boltBucketWorkspaces))

		return bucket.Delete([]byte(name))
	})
}

// WorkspaceExists checks if a workspace exists by name
func (b *Bolt) WorkspaceExists(name string) (bool, error) {
	var exists bool

	err := b.storage.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(boltBucketWorkspaces))
		exists = bucket.Get([]byte(name)) != nil

		return nil
	})

	return exists, err
}

// GetReposByWorkspace retrieves all repository URLs in a workspace
func (b *Bolt) GetReposByWorkspace(workspace string) ([]string, error) {
	var urls []string

	err := b.storage.View(func(tx *bbolt.Tx) error {
		repos := tx.Bucket([]byte(boltBucketRepos))

		return repos.ForEach(func(k, v []byte) error {
			var r model.Repository

			if err := json.Unmarshal(v, &r); err != nil {
				return err
			}

			if r.Workspace == workspace {
				urls = append(urls, r.URL)
			}

			return nil
		})
	})

	return urls, err
}

// UpdateRepoWorkspace updates the workspace for a repository
func (b *Bolt) UpdateRepoWorkspace(urlStr string, workspace string) error {
	return b.storage.Update(func(tx *bbolt.Tx) error {
		repos := tx.Bucket([]byte(boltBucketRepos))

		v := repos.Get([]byte(urlStr))

		if v == nil {
			return errors.New("repository not found")
		}

		var r model.Repository

		if err := json.Unmarshal(v, &r); err != nil {
			return err
		}

		r.Workspace = workspace
		r.UpdatedAt = time.Now()

		data, err := json.Marshal(&r)
		if err != nil {
			return err
		}

		return repos.Put([]byte(urlStr), data)
	})
}

// Standalone operations

// GetStandaloneConfig retrieves the standalone configuration
func (b *Bolt) GetStandaloneConfig() (*standalone.StandaloneConfig, error) {
	var config *standalone.StandaloneConfig

	err := b.storage.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(boltBucketStandalone))
		v := bucket.Get([]byte("config"))

		if v == nil {
			return nil
		}

		var c standalone.StandaloneConfig
		if err := json.Unmarshal(v, &c); err != nil {
			return err
		}

		config = &c
		return nil
	})

	return config, err
}

// SaveStandaloneConfig saves the standalone configuration
func (b *Bolt) SaveStandaloneConfig(config *standalone.StandaloneConfig) error {
	if config == nil {
		return errors.New("config is required")
	}

	data, err := json.Marshal(config)
	if err != nil {
		return err
	}

	return b.storage.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(boltBucketStandalone))
		return bucket.Put([]byte("config"), data)
	})
}

// DeleteStandaloneConfig removes the standalone configuration
func (b *Bolt) DeleteStandaloneConfig() error {
	return b.storage.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(boltBucketStandalone))
		return bucket.Delete([]byte("config"))
	})
}

// GetStandaloneClients retrieves all connected clients
func (b *Bolt) GetStandaloneClients() ([]standalone.Client, error) {
	var clients []standalone.Client

	err := b.storage.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(boltBucketStandalone))

		return bucket.ForEach(func(k, v []byte) error {
			key := string(k)
			if !strings.HasPrefix(key, "client:") {
				return nil
			}

			var c standalone.Client
			if err := json.Unmarshal(v, &c); err != nil {
				return err
			}

			clients = append(clients, c)
			return nil
		})
	})

	return clients, err
}

// SaveStandaloneClient saves or updates a connected client
func (b *Bolt) SaveStandaloneClient(client *standalone.Client) error {
	if client == nil {
		return errors.New("client is required")
	}

	if client.ID == "" {
		return errors.New("client ID is required")
	}

	data, err := json.Marshal(client)
	if err != nil {
		return err
	}

	return b.storage.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(boltBucketStandalone))
		return bucket.Put([]byte("client:"+client.ID), data)
	})
}

// DeleteStandaloneClient removes a connected client
func (b *Bolt) DeleteStandaloneClient(id string) error {
	return b.storage.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(boltBucketStandalone))
		return bucket.Delete([]byte("client:" + id))
	})
}

// Standalone connection operations (destination side)

// GetStandaloneConnection retrieves a connection by name
func (b *Bolt) GetStandaloneConnection(name string) (*standalone.StandaloneConnection, error) {
	var conn *standalone.StandaloneConnection

	err := b.storage.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(boltBucketConnections))
		v := bucket.Get([]byte(name))

		if v == nil {
			return nil
		}

		var c standalone.StandaloneConnection
		if err := json.Unmarshal(v, &c); err != nil {
			return err
		}

		conn = &c
		return nil
	})

	return conn, err
}

// ListStandaloneConnections retrieves all connections
func (b *Bolt) ListStandaloneConnections() ([]standalone.StandaloneConnection, error) {
	var connections []standalone.StandaloneConnection

	err := b.storage.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(boltBucketConnections))

		return bucket.ForEach(func(k, v []byte) error {
			var c standalone.StandaloneConnection
			if err := json.Unmarshal(v, &c); err != nil {
				return err
			}

			connections = append(connections, c)
			return nil
		})
	})

	return connections, err
}

// SaveStandaloneConnection saves or updates a connection
func (b *Bolt) SaveStandaloneConnection(conn *standalone.StandaloneConnection) error {
	if conn == nil {
		return errors.New("connection is required")
	}

	if conn.Name == "" {
		return errors.New("connection name is required")
	}

	data, err := json.Marshal(conn)
	if err != nil {
		return err
	}

	return b.storage.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(boltBucketConnections))
		return bucket.Put([]byte(conn.Name), data)
	})
}

// DeleteStandaloneConnection removes a connection by name
func (b *Bolt) DeleteStandaloneConnection(name string) error {
	return b.storage.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(boltBucketConnections))
		return bucket.Delete([]byte(name))
	})
}

// Server encryption config

// GetServerEncryptionConfig retrieves the server encryption configuration
func (b *Bolt) GetServerEncryptionConfig() (*standalone.ServerEncryptionConfig, error) {
	var config *standalone.ServerEncryptionConfig

	err := b.storage.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(boltBucketStandalone))
		v := bucket.Get([]byte("encryption"))

		if v == nil {
			return nil
		}

		var c standalone.ServerEncryptionConfig
		if err := json.Unmarshal(v, &c); err != nil {
			return err
		}

		config = &c
		return nil
	})

	return config, err
}

// SaveServerEncryptionConfig saves the server encryption configuration
func (b *Bolt) SaveServerEncryptionConfig(config *standalone.ServerEncryptionConfig) error {
	if config == nil {
		return errors.New("config is required")
	}

	data, err := json.Marshal(config)
	if err != nil {
		return err
	}

	return b.storage.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(boltBucketStandalone))
		return bucket.Put([]byte("encryption"), data)
	})
}

// Synced data operations

// syncedDataKey generates the key for synced data storage
func syncedDataKey(connectionName, dataType, name string) string {
	return connectionName + ":" + dataType + ":" + name
}

// GetSyncedData retrieves synced data by connection, type, and name
func (b *Bolt) GetSyncedData(connectionName, dataType, name string) (*standalone.SyncedData, error) {
	var data *standalone.SyncedData

	err := b.storage.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(boltBucketSyncedData))
		key := syncedDataKey(connectionName, dataType, name)
		v := bucket.Get([]byte(key))

		if v == nil {
			return nil
		}

		var d standalone.SyncedData
		if err := json.Unmarshal(v, &d); err != nil {
			return err
		}

		data = &d
		return nil
	})

	return data, err
}

// ListSyncedData retrieves all synced data for a connection
func (b *Bolt) ListSyncedData(connectionName string) ([]standalone.SyncedData, error) {
	var result []standalone.SyncedData
	prefix := connectionName + ":"

	err := b.storage.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(boltBucketSyncedData))

		return bucket.ForEach(func(k, v []byte) error {
			if !strings.HasPrefix(string(k), prefix) {
				return nil
			}

			var d standalone.SyncedData
			if err := json.Unmarshal(v, &d); err != nil {
				return err
			}

			result = append(result, d)
			return nil
		})
	})

	return result, err
}

// ListSyncedDataByState retrieves all synced data with a specific state
func (b *Bolt) ListSyncedDataByState(state standalone.SyncState) ([]standalone.SyncedData, error) {
	var result []standalone.SyncedData

	err := b.storage.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(boltBucketSyncedData))

		return bucket.ForEach(func(k, v []byte) error {
			var d standalone.SyncedData
			if err := json.Unmarshal(v, &d); err != nil {
				return err
			}

			if d.State == state {
				result = append(result, d)
			}
			return nil
		})
	})

	return result, err
}

// SaveSyncedData saves synced data
func (b *Bolt) SaveSyncedData(data *standalone.SyncedData) error {
	if data == nil {
		return errors.New("data is required")
	}

	if data.ConnectionName == "" || data.DataType == "" || data.Name == "" {
		return errors.New("connection name, data type, and name are required")
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return b.storage.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(boltBucketSyncedData))
		key := syncedDataKey(data.ConnectionName, data.DataType, data.Name)
		return bucket.Put([]byte(key), jsonData)
	})
}

// DeleteSyncedData removes synced data
func (b *Bolt) DeleteSyncedData(connectionName, dataType, name string) error {
	return b.storage.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(boltBucketSyncedData))
		key := syncedDataKey(connectionName, dataType, name)
		return bucket.Delete([]byte(key))
	})
}

// Client registration operations (server side)

// SavePendingRegistration saves a pending client registration
func (b *Bolt) SavePendingRegistration(reg *standalone.ClientRegistration) error {
	if reg == nil {
		return errors.New("registration is required")
	}

	if reg.ClientID == "" {
		return errors.New("client ID is required")
	}

	data, err := json.Marshal(reg)
	if err != nil {
		return err
	}

	return b.storage.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(boltBucketStandalone))
		return bucket.Put([]byte("pending:"+reg.ClientID), data)
	})
}

// GetPendingRegistration retrieves a pending registration by client ID
func (b *Bolt) GetPendingRegistration(clientID string) (*standalone.ClientRegistration, error) {
	var reg *standalone.ClientRegistration

	err := b.storage.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(boltBucketStandalone))
		v := bucket.Get([]byte("pending:" + clientID))

		if v == nil {
			return nil
		}

		var r standalone.ClientRegistration
		if err := json.Unmarshal(v, &r); err != nil {
			return err
		}

		reg = &r
		return nil
	})

	return reg, err
}

// ListPendingRegistrations retrieves all pending client registrations
func (b *Bolt) ListPendingRegistrations() ([]*standalone.ClientRegistration, error) {
	var registrations []*standalone.ClientRegistration

	err := b.storage.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(boltBucketStandalone))

		return bucket.ForEach(func(k, v []byte) error {
			key := string(k)
			if !strings.HasPrefix(key, "pending:") {
				return nil
			}

			var r standalone.ClientRegistration
			if err := json.Unmarshal(v, &r); err != nil {
				return err
			}

			registrations = append(registrations, &r)
			return nil
		})
	})

	return registrations, err
}

// RemovePendingRegistration removes a pending registration
func (b *Bolt) RemovePendingRegistration(clientID string) error {
	return b.storage.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(boltBucketStandalone))
		return bucket.Delete([]byte("pending:" + clientID))
	})
}

// Registered client operations (server side)

// SaveRegisteredClient saves a registered client
func (b *Bolt) SaveRegisteredClient(client *standalone.RegisteredClient) error {
	if client == nil {
		return errors.New("client is required")
	}

	if client.ClientID == "" {
		return errors.New("client ID is required")
	}

	data, err := json.Marshal(client)
	if err != nil {
		return err
	}

	return b.storage.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(boltBucketStandalone))
		return bucket.Put([]byte("registered:"+client.ClientID), data)
	})
}

// GetRegisteredClient retrieves a registered client by ID
func (b *Bolt) GetRegisteredClient(clientID string) (*standalone.RegisteredClient, error) {
	var client *standalone.RegisteredClient

	err := b.storage.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(boltBucketStandalone))
		v := bucket.Get([]byte("registered:" + clientID))

		if v == nil {
			return nil
		}

		var c standalone.RegisteredClient
		if err := json.Unmarshal(v, &c); err != nil {
			return err
		}

		client = &c
		return nil
	})

	return client, err
}

// ListRegisteredClients retrieves all registered clients
func (b *Bolt) ListRegisteredClients() ([]*standalone.RegisteredClient, error) {
	var clients []*standalone.RegisteredClient

	err := b.storage.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(boltBucketStandalone))

		return bucket.ForEach(func(k, v []byte) error {
			key := string(k)
			if !strings.HasPrefix(key, "registered:") {
				return nil
			}

			var c standalone.RegisteredClient
			if err := json.Unmarshal(v, &c); err != nil {
				return err
			}

			clients = append(clients, &c)
			return nil
		})
	})

	return clients, err
}

// DeleteRegisteredClient removes a registered client
func (b *Bolt) DeleteRegisteredClient(clientID string) error {
	return b.storage.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(boltBucketStandalone))
		return bucket.Delete([]byte("registered:" + clientID))
	})
}

// Sealed key operations

// GetSealedKey retrieves the sealed key data from the database
func (b *Bolt) GetSealedKey() (*SealedKeyData, error) {
	var data *SealedKeyData

	err := b.storage.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(boltBucketConfig))
		v := bucket.Get([]byte("sealed_key"))

		if v == nil {
			return nil
		}

		var d SealedKeyData
		if err := json.Unmarshal(v, &d); err != nil {
			return err
		}

		data = &d
		return nil
	})

	return data, err
}

// SaveSealedKey saves or updates the sealed key data in the database
func (b *Bolt) SaveSealedKey(data *SealedKeyData) error {
	if data == nil {
		return errors.New("sealed key data is required")
	}

	// Update last accessed time
	data.LastAccessed = time.Now()

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return b.storage.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(boltBucketConfig))
		return bucket.Put([]byte("sealed_key"), jsonData)
	})
}

// DeleteSealedKey removes the sealed key from the database
func (b *Bolt) DeleteSealedKey() error {
	return b.storage.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(boltBucketConfig))
		return bucket.Delete([]byte("sealed_key"))
	})
}

// HasSealedKey checks if a sealed key exists in the database
func (b *Bolt) HasSealedKey() (bool, error) {
	var exists bool

	err := b.storage.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(boltBucketConfig))
		exists = bucket.Get([]byte("sealed_key")) != nil
		return nil
	})

	return exists, err
}
