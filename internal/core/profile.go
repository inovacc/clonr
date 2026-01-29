package core

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/inovacc/clonr/internal/grpcclient"
	"github.com/inovacc/clonr/internal/model"
)

var (
	// ErrProfileNotFound is returned when a profile doesn't exist
	ErrProfileNotFound = errors.New("profile not found")

	// ErrProfileExists is returned when trying to create a profile that already exists
	ErrProfileExists = errors.New("profile already exists")

	// ErrNoActiveProfile is returned when no profile is active
	ErrNoActiveProfile = errors.New("no active profile")

	// ErrTokenNotFound is returned when token cannot be retrieved
	ErrTokenNotFound = errors.New("token not found")
)

// ProfileManager handles profile operations
type ProfileManager struct {
	client *grpcclient.Client
}

// NewProfileManager creates a new ProfileManager
func NewProfileManager() (*ProfileManager, error) {
	client, err := grpcclient.GetClient()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}

	return &ProfileManager{client: client}, nil
}

// CreateProfile creates a new profile with OAuth authentication
func (pm *ProfileManager) CreateProfile(ctx context.Context, name, host string, scopes []string) (*model.Profile, string, error) {
	// Check if profile already exists
	exists, err := pm.client.ProfileExists(name) //nolint:contextcheck // client manages its own timeout
	if err != nil {
		return nil, "", fmt.Errorf("failed to check profile existence: %w", err)
	}

	if exists {
		return nil, "", ErrProfileExists
	}

	// Use defaults if not specified
	if host == "" {
		host = model.DefaultHost()
	}

	if len(scopes) == 0 {
		scopes = model.DefaultScopes()
	}

	// Run OAuth flow
	flow := NewOAuthFlow(host, scopes)

	var deviceCode, verificationURL string

	flow.OnDeviceCode(func(code, url string) {
		deviceCode = code
		verificationURL = url
	})

	result, err := flow.Run(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("OAuth authentication failed: %w", err)
	}

	// Try to store token in keyring first
	var tokenStorage model.TokenStorage

	var encryptedToken []byte

	if err := SetToken(name, host, result.Token); err != nil {
		// Keyring not available, use encrypted storage
		encryptedToken, err = EncryptToken(result.Token, name, host)
		if err != nil {
			return nil, "", fmt.Errorf("failed to encrypt token: %w", err)
		}

		tokenStorage = model.TokenStorageInsecure
	} else {
		tokenStorage = model.TokenStorageKeyring
	}

	// Check if this is the first profile (make it active)
	profiles, err := pm.client.ListProfiles() //nolint:contextcheck // client manages its own timeout
	if err != nil {
		return nil, "", fmt.Errorf("failed to list profiles: %w", err)
	}

	isFirstProfile := len(profiles) == 0

	// Create profile
	profile := &model.Profile{
		Name:           name,
		Host:           host,
		User:           result.Username,
		TokenStorage:   tokenStorage,
		Scopes:         result.Scopes,
		Active:         isFirstProfile,
		EncryptedToken: encryptedToken,
		CreatedAt:      time.Now(),
		LastUsedAt:     time.Now(),
	}

	// Save profile
	if err := pm.client.SaveProfile(profile); err != nil { //nolint:contextcheck // client manages its own timeout
		// Clean up token on failure
		if tokenStorage == model.TokenStorageKeyring {
			_ = DeleteToken(name, host)
		}

		return nil, "", fmt.Errorf("failed to save profile: %w", err)
	}

	// Return device code info for UI display (empty if OAuth already completed)
	_ = deviceCode
	_ = verificationURL

	return profile, result.Token, nil
}

// GetProfile retrieves a profile by name
func (pm *ProfileManager) GetProfile(name string) (*model.Profile, error) {
	profile, err := pm.client.GetProfile(name)
	if err != nil {
		return nil, err
	}

	if profile == nil {
		return nil, ErrProfileNotFound
	}

	return profile, nil
}

// GetActiveProfile retrieves the currently active profile
func (pm *ProfileManager) GetActiveProfile() (*model.Profile, error) {
	profile, err := pm.client.GetActiveProfile()
	if err != nil {
		return nil, err
	}

	if profile == nil {
		return nil, ErrNoActiveProfile
	}

	return profile, nil
}

// SetActiveProfile sets the active profile
func (pm *ProfileManager) SetActiveProfile(name string) error {
	// Verify profile exists
	exists, err := pm.client.ProfileExists(name)
	if err != nil {
		return fmt.Errorf("failed to check profile existence: %w", err)
	}

	if !exists {
		return ErrProfileNotFound
	}

	return pm.client.SetActiveProfile(name)
}

// ListProfiles returns all profiles
func (pm *ProfileManager) ListProfiles() ([]model.Profile, error) {
	return pm.client.ListProfiles()
}

// DeleteProfile removes a profile and its stored token
func (pm *ProfileManager) DeleteProfile(name string) error {
	// Get profile first to know where token is stored
	profile, err := pm.client.GetProfile(name)
	if err != nil {
		return fmt.Errorf("failed to get profile: %w", err)
	}

	if profile == nil {
		return ErrProfileNotFound
	}

	// Delete token from keyring if applicable
	// Ignore errors - profile deletion is more important and token might already be deleted
	if profile.TokenStorage == model.TokenStorageKeyring {
		_ = DeleteToken(name, profile.Host)
	}

	return pm.client.DeleteProfile(name)
}

// GetProfileToken retrieves the token for a profile
func (pm *ProfileManager) GetProfileToken(name string) (string, error) {
	profile, err := pm.client.GetProfile(name)
	if err != nil {
		return "", fmt.Errorf("failed to get profile: %w", err)
	}

	if profile == nil {
		return "", ErrProfileNotFound
	}

	return pm.getTokenFromProfile(profile)
}

// GetActiveProfileToken retrieves the token for the active profile
func (pm *ProfileManager) GetActiveProfileToken() (string, error) {
	profile, err := pm.client.GetActiveProfile()
	if err != nil {
		return "", err
	}

	if profile == nil {
		return "", ErrNoActiveProfile
	}

	return pm.getTokenFromProfile(profile)
}

// getTokenFromProfile retrieves the token based on storage type
func (pm *ProfileManager) getTokenFromProfile(profile *model.Profile) (string, error) {
	switch profile.TokenStorage {
	case model.TokenStorageKeyring:
		token, err := GetToken(profile.Name, profile.Host)
		if err != nil {
			return "", fmt.Errorf("failed to get token from keyring: %w", err)
		}

		return token, nil
	case model.TokenStorageInsecure:
		if len(profile.EncryptedToken) == 0 {
			return "", ErrTokenNotFound
		}

		token, err := DecryptToken(profile.EncryptedToken, profile.Name, profile.Host)
		if err != nil {
			return "", fmt.Errorf("failed to decrypt token: %w", err)
		}

		return token, nil
	default:
		return "", ErrTokenNotFound
	}
}

// ValidateProfileToken checks if a profile's token is still valid
func (pm *ProfileManager) ValidateProfileToken(ctx context.Context, name string) (bool, error) {
	profile, err := pm.client.GetProfile(name) //nolint:contextcheck // client manages its own timeout
	if err != nil {
		return false, fmt.Errorf("failed to get profile: %w", err)
	}

	if profile == nil {
		return false, ErrProfileNotFound
	}

	token, err := pm.getTokenFromProfile(profile)
	if err != nil {
		return false, err
	}

	valid, _, err := ValidateToken(ctx, token, profile.Host)
	if err != nil {
		return false, err
	}

	return valid, nil
}

// RefreshProfile refreshes a profile's OAuth token
func (pm *ProfileManager) RefreshProfile(ctx context.Context, name string) error {
	profile, err := pm.client.GetProfile(name) //nolint:contextcheck // client manages its own timeout
	if err != nil {
		return fmt.Errorf("failed to get profile: %w", err)
	}

	if profile == nil {
		return ErrProfileNotFound
	}

	// Run OAuth flow again
	flow := NewOAuthFlow(profile.Host, profile.Scopes)

	result, err := flow.Run(ctx)
	if err != nil {
		return fmt.Errorf("OAuth authentication failed: %w", err)
	}

	// Store new token
	if profile.TokenStorage == model.TokenStorageKeyring {
		if err := SetToken(name, profile.Host, result.Token); err != nil {
			return fmt.Errorf("failed to store token: %w", err)
		}
	} else {
		encryptedToken, err := EncryptToken(result.Token, name, profile.Host)
		if err != nil {
			return fmt.Errorf("failed to encrypt token: %w", err)
		}

		profile.EncryptedToken = encryptedToken
	}

	// Update profile
	profile.User = result.Username
	profile.LastUsedAt = time.Now()

	return pm.client.SaveProfile(profile) //nolint:contextcheck // client manages its own timeout
}
