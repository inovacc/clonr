package model

import "time"

// TokenStorage indicates where the token is stored
type TokenStorage string

const (
	// TokenStorageEncrypted stores TPM-encrypted token in database
	TokenStorageEncrypted TokenStorage = "encrypted"

	// TokenStorageOpen stores plain text token in database (no TPM available)
	TokenStorageOpen TokenStorage = "open"
)

// Profile represents a GitHub authentication profile
type Profile struct {
	// Name is the unique identifier for this profile
	Name string `json:"name"`

	// Host is the GitHub host (e.g., github.com or enterprise URL)
	Host string `json:"host"`

	// User is the authenticated GitHub username
	User string `json:"user"`

	// TokenStorage indicates how the token is stored (encrypted or open)
	TokenStorage TokenStorage `json:"token_storage"`

	// Scopes are the OAuth scopes granted to this token
	Scopes []string `json:"scopes"`

	// Active indicates if this is the currently active profile
	Active bool `json:"active"`

	// EncryptedToken stores the token (with ENC: or OPEN: prefix)
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
