//go:build sqlite

package database

import (
	"errors"
	"net/url"
	"path/filepath"
	"time"

	"github.com/inovacc/clonr/internal/model"
	"github.com/inovacc/clonr/internal/params"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// ConfigRecord is the GORM model for configuration
type ConfigRecord struct {
	ID              uint   `gorm:"primaryKey"`
	DefaultCloneDir string `gorm:"column:default_clone_dir"`
	Editor          string `gorm:"column:editor"`
	Terminal        string `gorm:"column:terminal"`
	MonitorInterval int    `gorm:"column:monitor_interval"`
	ServerPort      int    `gorm:"column:server_port"`
}

func (ConfigRecord) TableName() string {
	return "config"
}

type sqliteStore struct {
	*gorm.DB
}

func initDB() (Store, error) {
	path := filepath.Join(params.AppdataDir, "clonr.sqlite")

	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&model.Repository{}, &ConfigRecord{}); err != nil {
		return nil, err
	}

	conn := &sqliteStore{DB: db}

	if err := conn.Ping(); err != nil {
		return nil, err
	}

	return conn, nil
}

func (s *sqliteStore) Ping() error {
	conn, err := s.DB.DB()
	if err != nil {
		return err
	}

	return conn.Ping()
}

func (s *sqliteStore) SaveRepo(u *url.URL, path string) error {
	if u == nil {
		return errors.New("url is required")
	}

	return s.Create(&model.Repository{
		UID:      uuid.New().String(),
		URL:      u.String(),
		Path:     path,
		ClonedAt: time.Now(),
	}).Error
}

func (s *sqliteStore) RepoExistsByURL(u *url.URL) (bool, error) {
	var count int64
	if err := s.Model(&model.Repository{}).Where("url = ?", u.String()).Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (s *sqliteStore) RepoExistsByPath(path string) (bool, error) {
	var count int64
	if err := s.Model(&model.Repository{}).Where("path = ?", path).Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (s *sqliteStore) InsertRepoIfNotExists(u *url.URL, path string) error {
	if u != nil {
		exists, err := s.RepoExistsByURL(u)
		if err != nil {
			return err
		}

		if exists {
			return nil
		}
	}

	repoExists, err := s.RepoExistsByPath(path)
	if err != nil {
		return err
	}

	if repoExists {
		return nil
	}

	return s.SaveRepo(u, path)
}

func (s *sqliteStore) GetAllRepos() ([]model.Repository, error) {
	var repos []model.Repository

	return repos, s.Find(&repos).Error
}

func (s *sqliteStore) GetRepos(favoritesOnly bool) ([]model.Repository, error) {
	var repos []model.Repository

	q := s.Model(&model.Repository{})

	if favoritesOnly {
		q = q.Where("favorite = ?", true)
	}

	return repos, q.Find(&repos).Error
}

func (s *sqliteStore) SetFavoriteByURL(urlStr string, fav bool) error {
	return s.Model(&model.Repository{}).Where("url = ?", urlStr).Update("favorite", fav).Error
}

func (s *sqliteStore) UpdateRepoTimestamp(urlStr string) error {
	return s.Model(&model.Repository{}).Where("url = ?", urlStr).Update("updated_at", time.Now()).Error
}

func (s *sqliteStore) RemoveRepoByURL(u *url.URL) error {
	return s.Where("url = ?", u.String()).Delete(&model.Repository{}).Error
}

func (s *sqliteStore) GetConfig() (*model.Config, error) {
	var record ConfigRecord

	err := s.First(&record).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Return default config if not found
			defaultCfg := model.DefaultConfig()
			return &defaultCfg, nil
		}
		return nil, err
	}

	cfg := &model.Config{
		DefaultCloneDir: record.DefaultCloneDir,
		Editor:          record.Editor,
		Terminal:        record.Terminal,
		MonitorInterval: record.MonitorInterval,
		ServerPort:      record.ServerPort,
	}

	return cfg, nil
}

func (s *sqliteStore) SaveConfig(cfg *model.Config) error {
	if cfg == nil {
		return errors.New("config is required")
	}

	record := ConfigRecord{
		ID:              1, // Always use ID 1 for singleton config
		DefaultCloneDir: cfg.DefaultCloneDir,
		Editor:          cfg.Editor,
		Terminal:        cfg.Terminal,
		MonitorInterval: cfg.MonitorInterval,
		ServerPort:      cfg.ServerPort,
	}

	// Use Save to insert or update
	return s.Save(&record).Error
}
