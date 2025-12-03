package database

import (
	"net/url"
	"sync"

	"github.com/inovacc/clonr/internal/model"
)

// Store defines the database operations used by the app.
type Store interface {
	Ping() error
	SaveRepo(u *url.URL, path string) error
	RepoExistsByURL(u *url.URL) (bool, error)
	RepoExistsByPath(path string) (bool, error)
	InsertRepoIfNotExists(u *url.URL, path string) error
	GetAllRepos() ([]model.Repository, error)
	GetRepos(favoritesOnly bool) ([]model.Repository, error)
	SetFavoriteByURL(urlStr string, fav bool) error
	RemoveRepoByURL(u *url.URL) error
	GetConfig() (*model.Config, error)
	SaveConfig(cfg *model.Config) error
}

var (
	once sync.Once
	db   Store
)

func init() {
	once.Do(func() {
		var err error

		db, err = initDB()
		if err != nil {
			panic(err)
		}
	})
}

// GetDB returns the initialized database store.
func GetDB() Store {
	return db
}
