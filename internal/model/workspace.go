package model

import "time"

// Workspace represents a logical grouping of repositories
type Workspace struct {
	// Name is the unique identifier for this workspace (e.g., "personal", "work")
	Name string `json:"name"`

	// Description is an optional description of the workspace
	Description string `json:"description"`

	// Path is the base clone directory for this workspace
	Path string `json:"path"`

	// Active indicates if this is the currently active workspace
	Active bool `json:"active"`

	// CreatedAt is when the workspace was created
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when the workspace was last updated
	UpdatedAt time.Time `json:"updated_at"`
}

// DefaultWorkspaceName returns the name of the default workspace
func DefaultWorkspaceName() string {
	return "default"
}
