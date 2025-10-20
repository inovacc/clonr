package database

import (
	"net/url"
	"path/filepath"
	"time"

	"github.com/dyammarcano/clonr/internal/model"
	"github.com/dyammarcano/clonr/internal/params"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Database struct {
	*gorm.DB
}

func InitDB() (*Database, error) {
	path := filepath.Join(params.AppdataDir, "clonr.db")

	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&model.Repository{}); err != nil {
		return nil, err
	}

	conn := &Database{DB: db}

	if err := conn.Ping(); err != nil {
		return nil, err
	}

	return conn, nil
}

func (d *Database) Ping() error {
	conn, err := d.DB.DB()
	if err != nil {
		return err
	}

	return conn.Ping()
}

func (d *Database) SaveRepo(u *url.URL, path string) error {
	return d.Create(&model.Repository{
		UID:       uuid.NewString(),
		URL:       u.String(),
		Path:      path,
		ClonedAt:  time.Now(),
		UpdatedAt: time.Now(),
	}).Error
}

// RepoExistsByURL returns true if a repository with the given URL already exists.
func (d *Database) RepoExistsByURL(u *url.URL) (bool, error) {
	var count int64
	if err := d.Model(&model.Repository{}).Where("url = ?", u.String()).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (d *Database) GetAllRepos() ([]model.Repository, error) {
	var repos []model.Repository

	return repos, d.Find(&repos).Error
}

func (d *Database) RemoveRepoByURL(u *url.URL) error {
	return d.Where("url = ?", u.String()).Delete(&model.Repository{}).Error
}
