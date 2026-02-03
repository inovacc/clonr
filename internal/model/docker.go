package model

import "time"

// DockerProfile represents container registry credentials
type DockerProfile struct {
	// Name is the unique identifier for this docker profile
	Name string `json:"name"`

	// Registry is the container registry URL (e.g., "docker.io", "ghcr.io")
	Registry string `json:"registry"`

	// Username is the registry username
	Username string `json:"username"`

	// EncryptedToken stores the password/token encrypted with keystore
	EncryptedToken []byte `json:"encrypted_token,omitempty"`

	// TokenStorage indicates how the token is stored (encrypted or open)
	TokenStorage TokenStorage `json:"token_storage"`

	// CreatedAt is when the profile was created
	CreatedAt time.Time `json:"created_at"`

	// LastUsedAt is when the profile was last used for login
	LastUsedAt time.Time `json:"last_used_at"`
}

// Common container registries
const (
	RegistryDockerHub = "docker.io"
	RegistryGHCR      = "ghcr.io"
	RegistryGCR       = "gcr.io"
	RegistryECR       = "ecr.aws"
	RegistryACR       = "azurecr.io"
)

// DefaultDockerRegistry returns the default registry (Docker Hub)
func DefaultDockerRegistry() string {
	return RegistryDockerHub
}
