package core

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/inovacc/clonr/internal/client/grpc"
	"github.com/inovacc/clonr/internal/crypto/tpm"
	"github.com/inovacc/clonr/internal/model"
)

var (
	// ErrProfileNotFound is returned when a profile doesn't exist
	ErrProfileNotFound = errors.New("profile not found")

	// ErrProfileExists is returned when trying to create a profile that already exists
	ErrProfileExists = errors.New("profile already exists")

	// ErrNoActiveProfile is returned when no profile is active
	ErrNoActiveProfile = errors.New("no active profile")

	// ErrTokenNotFound is returned when the token cannot be retrieved
	ErrTokenNotFound = errors.New("token not found")
)

// ProfileManager handles profile operations
type ProfileManager struct {
	client *grpc.Client
}

// NewProfileManager creates a new ProfileManager
func NewProfileManager() (*ProfileManager, error) {
	client, err := grpc.GetClient()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}

	return &ProfileManager{client: client}, nil
}

// CreateProfile creates a new profile with OAuth authentication
func (pm *ProfileManager) CreateProfile(ctx context.Context, name, host string, scopes []string) (*model.Profile, string, error) {
	// Check if a profile already exists
	exists, err := pm.client.ProfileExists(name) //nolint:contextcheck // a client manages its own timeout
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

	// Encrypt and store token in profile
	encryptedToken, tokenStorage, err := pm.storeToken(name, host, result.Token)
	if err != nil {
		return nil, "", fmt.Errorf("failed to store token: %w", err)
	}

	// Check if this is the first profile (make it active)
	profiles, err := pm.client.ListProfiles() //nolint:contextcheck // a client manages its own timeout
	if err != nil {
		return nil, "", fmt.Errorf("failed to list profiles: %w", err)
	}

	isFirstProfile := len(profiles) == 0

	// Create profile with encrypted token
	profile := &model.Profile{
		Name:           name,
		Host:           host,
		User:           result.Username,
		TokenStorage:   tokenStorage,
		Scopes:         result.Scopes,
		Default:        isFirstProfile,
		EncryptedToken: encryptedToken,
		CreatedAt:      time.Now(),
		LastUsedAt:     time.Now(),
	}

	// Save profile to BoltDB
	if err := pm.client.SaveProfile(profile); err != nil { //nolint:contextcheck // a client manages its own timeout
		return nil, "", fmt.Errorf("failed to save profile: %w", err)
	}

	// Return device code info for UI display (empty if OAuth already completed)
	_ = deviceCode
	_ = verificationURL

	return profile, result.Token, nil
}

// storeToken encrypts and returns the token for storage in BoltDB.
// Returns the encrypted token bytes and storage type (encrypted or open).
func (pm *ProfileManager) storeToken(name, host, token string) ([]byte, model.TokenStorage, error) {
	encryptedToken, err := tpm.EncryptToken(token, name, host)
	if err != nil {
		return nil, "", fmt.Errorf("failed to encrypt token: %w", err)
	}

	// Determine a storage type based on whether data is encrypted or open
	storageType := model.TokenStorageEncrypted
	if tpm.IsDataOpen(encryptedToken) {
		storageType = model.TokenStorageOpen
	}

	return encryptedToken, storageType, nil
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
	// Get a profile first to know where the token is stored
	profile, err := pm.client.GetProfile(name)
	if err != nil {
		return fmt.Errorf("failed to get profile: %w", err)
	}

	if profile == nil {
		return ErrProfileNotFound
	}

	// Token is stored in profile.EncryptedToken, so it gets deleted with the profile
	return pm.client.DeleteProfile(name)
}

// UpdateProfile updates an existing profile
func (pm *ProfileManager) UpdateProfile(profile *model.Profile) error {
	return pm.client.SaveProfile(profile)
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

// getTokenFromProfile retrieves and decrypts the token from the profile
func (pm *ProfileManager) getTokenFromProfile(profile *model.Profile) (string, error) {
	if len(profile.EncryptedToken) == 0 {
		return "", ErrTokenNotFound
	}

	token, err := tpm.DecryptToken(profile.EncryptedToken, profile.Name, profile.Host)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt token: %w", err)
	}

	return token, nil
}

// ValidateProfileToken checks if a profile's token is still valid
func (pm *ProfileManager) ValidateProfileToken(ctx context.Context, name string) (bool, error) {
	profile, err := pm.client.GetProfile(name) //nolint:contextcheck // a client manages its own timeout
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
	profile, err := pm.client.GetProfile(name) //nolint:contextcheck // a client manages its own timeout
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

	// Encrypt and store new token
	encryptedToken, tokenStorage, err := pm.storeToken(name, profile.Host, result.Token)
	if err != nil {
		return fmt.Errorf("failed to encrypt token: %w", err)
	}

	profile.EncryptedToken = encryptedToken
	profile.TokenStorage = tokenStorage

	// Update profile
	profile.User = result.Username
	profile.LastUsedAt = time.Now()

	return pm.client.SaveProfile(profile) //nolint:contextcheck // a client manages its own timeout
}

// AddNotifyChannel adds a notification channel to a profile.
// Sensitive data in the config map is encrypted using the profile's encryption.
func (pm *ProfileManager) AddNotifyChannel(profileName string, channel *model.NotifyChannel) error {
	profile, err := pm.GetProfile(profileName)
	if err != nil {
		return err
	}

	// Encrypt sensitive config values
	encryptedConfig := make(map[string]string)

	for key, value := range channel.Config {
		if isSensitiveKey(key) && value != "" {
			encrypted, encErr := tpm.EncryptToken(value, profileName, string(channel.Type))
			if encErr != nil {
				return fmt.Errorf("failed to encrypt %s: %w", key, encErr)
			}

			// Base64 encode to ensure valid UTF-8 for gRPC transport
			encryptedConfig[key] = base64.StdEncoding.EncodeToString(encrypted)
		} else {
			encryptedConfig[key] = value
		}
	}

	channel.Config = encryptedConfig

	// Check if channel with same ID exists, replace it
	found := false

	for i, ch := range profile.NotifyChannels {
		if ch.ID == channel.ID {
			profile.NotifyChannels[i] = *channel
			found = true

			break
		}
	}

	if !found {
		profile.NotifyChannels = append(profile.NotifyChannels, *channel)
	}

	return pm.client.SaveProfile(profile) //nolint:contextcheck // a client manages its own timeout
}

// RemoveNotifyChannel removes a notification channel from a profile.
func (pm *ProfileManager) RemoveNotifyChannel(profileName, channelID string) error {
	profile, err := pm.GetProfile(profileName)
	if err != nil {
		return err
	}

	newChannels := make([]model.NotifyChannel, 0, len(profile.NotifyChannels))

	for _, ch := range profile.NotifyChannels {
		if ch.ID != channelID {
			newChannels = append(newChannels, ch)
		}
	}

	profile.NotifyChannels = newChannels

	return pm.client.SaveProfile(profile) //nolint:contextcheck // a client manages its own timeout
}

// GetNotifyChannel retrieves a notification channel from a profile.
func (pm *ProfileManager) GetNotifyChannel(profileName, channelID string) (*model.NotifyChannel, error) {
	profile, err := pm.GetProfile(profileName)
	if err != nil {
		return nil, err
	}

	for _, ch := range profile.NotifyChannels {
		if ch.ID == channelID {
			return &ch, nil
		}
	}

	return nil, fmt.Errorf("channel %s not found in profile %s", channelID, profileName)
}

// GetNotifyChannelByType retrieves the first notification channel of a given type from a profile.
func (pm *ProfileManager) GetNotifyChannelByType(profileName string, channelType model.ChannelType) (*model.NotifyChannel, error) {
	profile, err := pm.GetProfile(profileName)
	if err != nil {
		return nil, err
	}

	for _, ch := range profile.NotifyChannels {
		if ch.Type == channelType {
			return &ch, nil
		}
	}

	return nil, nil
}

// DecryptChannelConfig decrypts sensitive values in a channel's config.
func (pm *ProfileManager) DecryptChannelConfig(profileName string, channel *model.NotifyChannel) (map[string]string, error) {
	decrypted := make(map[string]string)

	for key, value := range channel.Config {
		if isSensitiveKey(key) && value != "" {
			// Decode base64 first
			encryptedBytes, err := base64.StdEncoding.DecodeString(value)
			if err != nil {
				// Not base64 encoded, try raw bytes (backwards compatibility)
				encryptedBytes = []byte(value)
			}

			plaintext, err := tpm.DecryptToken(encryptedBytes, profileName, string(channel.Type))
			if err != nil {
				// If decryption fails, the value might not be encrypted
				decrypted[key] = value
			} else {
				decrypted[key] = plaintext
			}
		} else {
			decrypted[key] = value
		}
	}

	return decrypted, nil
}

// isSensitiveKey returns true if the config key contains sensitive data.
func isSensitiveKey(key string) bool {
	sensitiveKeys := []string{
		"token", "secret", "password", "api_key", "webhook_url",
		"bot_token", "client_secret", "hmac_secret", "refresh_token",
		"access_token",
	}

	for _, k := range sensitiveKeys {
		if key == k {
			return true
		}
	}

	return false
}
