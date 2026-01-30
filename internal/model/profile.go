package model

import "time"

// TokenStorage indicates where the token is stored
type TokenStorage string

const (
	// TokenStorageKeyring stores token in system keyring
	TokenStorageKeyring TokenStorage = "keyring"

	// TokenStorageInsecure stores encrypted token in database (fallback)
	TokenStorageInsecure TokenStorage = "insecure_storage"

	// TokenStorageKeePass stores token in KeePass database
	TokenStorageKeePass TokenStorage = "keepass"
)

// Profile represents a GitHub authentication profile
type Profile struct {
	// Name is the unique identifier for this profile
	Name string `json:"name"`

	// Host is the GitHub host (e.g., github.com or enterprise URL)
	Host string `json:"host"`

	// User is the authenticated GitHub username
	User string `json:"user"`

	// TokenStorage indicates where the token is stored (keyring or encrypted)
	TokenStorage TokenStorage `json:"token_storage"`

	// Scopes are the OAuth scopes granted to this token
	Scopes []string `json:"scopes"`

	// Active indicates if this is the currently active profile
	Active bool `json:"active"`

	// EncryptedToken stores the encrypted token when keyring is unavailable
	// This field is only populated when TokenStorage is insecure_storage
	EncryptedToken []byte `json:"encrypted_token,omitempty"`

	// CreatedAt is when the profile was created
	CreatedAt time.Time `json:"created_at"`

	// LastUsedAt is when the profile was last used
	LastUsedAt time.Time `json:"last_used_at"`

	// Workspace is the associated workspace for this profile
	Workspace string `json:"workspace"`
}

// DefaultHost returns the default GitHub host
func DefaultHost() string {
	return "github.com"
}

// DefaultScopes returns the default OAuth scopes for clonr
func DefaultScopes() []string {
	return []string{
		"repo",
		"read:org",
		"gist",
		"read:user",
		"user:email",
	}
}
