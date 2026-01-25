package database

import (
	"net/url"
	"sync"

	"github.com/inovacc/clonr/internal/model"
)

// Store defines the database operations used by the app.
//
//nolint:interfacebloat // all methods are required for database operations
type Store interface {
	Ping() error
	SaveRepo(u *url.URL, path string) error
	RepoExistsByURL(u *url.URL) (bool, error)
	RepoExistsByPath(path string) (bool, error)
	InsertRepoIfNotExists(u *url.URL, path string) error
	GetAllRepos() ([]model.Repository, error)
	GetRepos(favoritesOnly bool) ([]model.Repository, error)
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
