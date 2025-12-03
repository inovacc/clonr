//go:build !sqlite

package database

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
	boltBucketRepos  = "repos"  // key: URL -> Repository JSON
	boltBucketPaths  = "paths"  // key: Path -> URL string
	boltBucketConfig = "config" // key: "config" -> Config JSON
)

type Bolt struct {
	db *bbolt.DB
}

func initDB() (Store, error) {
	path := filepath.Join(params.AppdataDir, "clonr.bolt")

	db, err := bbolt.Open(path, 0600, &bbolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, err
	}

	if err := db.Update(func(tx *bbolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists([]byte(boltBucketRepos)); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists([]byte(boltBucketPaths)); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists([]byte(boltBucketConfig)); err != nil {
			return err
		}

		return nil
	}); err != nil {
		_ = db.Close()

		return nil, err
	}

	return &Bolt{db: db}, nil
}

func (b *Bolt) Ping() error {
	return b.db.View(func(tx *bbolt.Tx) error {
		return nil
	})
}

func (b *Bolt) SaveRepo(u *url.URL, path string) error {
	if u == nil {
		return errors.New("url is required")
	}

	repo := model.Repository{
		UID:      uuid.New().String(),
		URL:      u.String(),
		Path:     path,
		ClonedAt: time.Now(),
	}

	data, err := json.Marshal(&repo)
	if err != nil {
		return err
	}

	return b.db.Update(func(tx *bbolt.Tx) error {
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

	err := b.db.View(func(tx *bbolt.Tx) error {
		repos := tx.Bucket([]byte(boltBucketRepos))
		exists = repos.Get([]byte(u.String())) != nil

		return nil
	})

	return exists, err
}

func (b *Bolt) RepoExistsByPath(path string) (bool, error) {
	var exists bool

	err := b.db.View(func(tx *bbolt.Tx) error {
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

	err := b.db.View(func(tx *bbolt.Tx) error {
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

func (b *Bolt) GetRepos(favoritesOnly bool) ([]model.Repository, error) {
	if !favoritesOnly {
		return b.GetAllRepos()
	}

	var out []model.Repository

	err := b.db.View(func(tx *bbolt.Tx) error {
		repos := tx.Bucket([]byte(boltBucketRepos))

		return repos.ForEach(func(k, v []byte) error {
			var r model.Repository

			if err := json.Unmarshal(v, &r); err != nil {
				return err
			}

			if r.Favorite {
				out = append(out, r)
			}

			return nil
		})
	})

	return out, err
}

func (b *Bolt) SetFavoriteByURL(urlStr string, fav bool) error {
	return b.db.Update(func(tx *bbolt.Tx) error {
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
	return b.db.Update(func(tx *bbolt.Tx) error {
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
	return b.db.Update(func(tx *bbolt.Tx) error {
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

	err := b.db.View(func(tx *bbolt.Tx) error {
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

	return b.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(boltBucketConfig))

		return bucket.Put([]byte("config"), data)
	})
}
