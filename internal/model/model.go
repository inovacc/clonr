package model

import "time"

type Repository struct {
	// ID is the primary key
	ID uint `gorm:"primaryKey" json:"id"`

	// UID is the unique identifier for the repository
	UID string `json:"uid"`

	// URL is the remote repository URL
	URL string `json:"url"`

	// Path is the local path where the repository was cloned
	Path string `json:"path"`

	// Favorite indicates if the repository is a favorite
	Favorite bool `gorm:"default:false" json:"favorite"`

	// Timestamps for when the repository was last cloned and updated
	ClonedAt time.Time `json:"cloned_at"`

	// UpdatedAt is the last time the repository was checked for updates
	UpdatedAt time.Time `json:"updated_at"`

	// LastChecked is the last time the repository was checked for updates
	LastChecked time.Time `json:"last_checked"`
}
