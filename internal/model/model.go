package model

import "time"

type Repository struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UID         string    `json:"uid"`
	URL         string    `json:"url"`
	Path        string    `json:"path"`
	Favorite    bool      `gorm:"default:false" json:"favorite"`
	ClonedAt    time.Time `json:"cloned_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	LastChecked time.Time `json:"last_checked"`
}
