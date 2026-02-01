//go:build !sqlite

package store

import (
	"encoding/json"
	"errors"
	"net/url"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/inovacc/clonr/internal/model"
	"github.com/inovacc/clonr/internal/params"
	"go.etcd.io/bbolt"
)

const (
	boltBucketRepos      = "repos"      // key: URL -> Repository JSON
	boltBucketPaths      = "paths"      // key: Path -> URL string
	boltBucketConfig     = "config"     // key: "config" -> Config JSON
	boltBucketProfiles   = "profiles"   // key: name -> Profile JSON
	boltBucketWorkspaces = "workspaces" // key: name -> Workspace JSON
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

		if _, err := tx.CreateBucketIfNotExists([]byte(boltBucketWorkspaces)); err != nil {
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

		if _, err := tx.CreateBucketIfNotExists([]byte(boltBucketWorkspaces)); err != nil {
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

// GetActiveProfile retrieves the currently active profile
func (b *Bolt) GetActiveProfile() (*model.Profile, error) {
	var profile *model.Profile

	err := b.storage.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(boltBucketProfiles))

		return bucket.ForEach(func(k, v []byte) error {
			var p model.Profile
			if err := json.Unmarshal(v, &p); err != nil {
				return err
			}

			if p.Active {
				profile = &p

				return nil
			}

			return nil
		})
	})

	return profile, err
}

// SetActiveProfile sets the active profile by name
func (b *Bolt) SetActiveProfile(name string) error {
	return b.storage.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(boltBucketProfiles))

		// First, verify the profile exists
		v := bucket.Get([]byte(name))
		if v == nil {
			return errors.New("profile not found")
		}

		// Deactivate all profiles and activate the specified one
		if err := bucket.ForEach(func(k, val []byte) error {
			var p model.Profile
			if err := json.Unmarshal(val, &p); err != nil {
				return err
			}

			p.Active = string(k) == name

			if p.Active {
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
