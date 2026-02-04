package service

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/inovacc/clonr/internal/crypto/tpm"
	"github.com/inovacc/clonr/internal/model"
	"github.com/inovacc/clonr/internal/store"
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

// ProfileService provides profile operations with direct database access.
// This is used by the web server running inside the gRPC server process.
type ProfileService struct {
	store store.Store
}

// NewProfileService creates a new ProfileService with direct database access.
func NewProfileService(s store.Store) *ProfileService {
	return &ProfileService{store: s}
}

// GetProfile retrieves a profile by name.
func (ps *ProfileService) GetProfile(name string) (*model.Profile, error) {
	profile, err := ps.store.GetProfile(name)
	if err != nil {
		return nil, err
	}

	if profile == nil {
		return nil, ErrProfileNotFound
	}

	return profile, nil
}

// GetActiveProfile retrieves the currently active profile.
func (ps *ProfileService) GetActiveProfile() (*model.Profile, error) {
	profile, err := ps.store.GetActiveProfile()
	if err != nil {
		return nil, err
	}

	if profile == nil {
		return nil, ErrNoActiveProfile
	}

	return profile, nil
}

// SetActiveProfile sets the active profile.
func (ps *ProfileService) SetActiveProfile(name string) error {
	exists, err := ps.store.ProfileExists(name)
	if err != nil {
		return fmt.Errorf("failed to check profile existence: %w", err)
	}

	if !exists {
		return ErrProfileNotFound
	}

	return ps.store.SetActiveProfile(name)
}

// ListProfiles returns all profiles.
func (ps *ProfileService) ListProfiles() ([]model.Profile, error) {
	return ps.store.ListProfiles()
}

// SaveProfile saves a profile to the database.
func (ps *ProfileService) SaveProfile(profile *model.Profile) error {
	return ps.store.SaveProfile(profile)
}

// DeleteProfile removes a profile.
func (ps *ProfileService) DeleteProfile(name string) error {
	profile, err := ps.store.GetProfile(name)
	if err != nil {
		return fmt.Errorf("failed to get profile: %w", err)
	}

	if profile == nil {
		return ErrProfileNotFound
	}

	return ps.store.DeleteProfile(name)
}

// ProfileExists checks if a profile exists.
func (ps *ProfileService) ProfileExists(name string) (bool, error) {
	return ps.store.ProfileExists(name)
}

// GetProfileToken retrieves and decrypts the token for a profile.
func (ps *ProfileService) GetProfileToken(name string) (string, error) {
	profile, err := ps.store.GetProfile(name)
	if err != nil {
		return "", fmt.Errorf("failed to get profile: %w", err)
	}

	if profile == nil {
		return "", ErrProfileNotFound
	}

	return ps.getTokenFromProfile(profile)
}

// GetActiveProfileToken retrieves the token for the active profile.
func (ps *ProfileService) GetActiveProfileToken() (string, error) {
	profile, err := ps.store.GetActiveProfile()
	if err != nil {
		return "", err
	}

	if profile == nil {
		return "", ErrNoActiveProfile
	}

	return ps.getTokenFromProfile(profile)
}

// getTokenFromProfile retrieves and decrypts the token from the profile.
func (ps *ProfileService) getTokenFromProfile(profile *model.Profile) (string, error) {
	if len(profile.EncryptedToken) == 0 {
		return "", ErrTokenNotFound
	}

	token, err := tpm.DecryptToken(profile.EncryptedToken, profile.Name, profile.Host)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt token: %w", err)
	}

	return token, nil
}

// EncryptToken encrypts a token for storage in a profile.
func (ps *ProfileService) EncryptToken(token, name, host string) ([]byte, model.TokenStorage, error) {
	encryptedToken, err := tpm.EncryptToken(token, name, host)
	if err != nil {
		return nil, "", fmt.Errorf("failed to encrypt token: %w", err)
	}

	storageType := model.TokenStorageEncrypted
	if tpm.IsDataOpen(encryptedToken) {
		storageType = model.TokenStorageOpen
	}

	return encryptedToken, storageType, nil
}

// AddNotifyChannel adds a notification channel to a profile.
// Sensitive data in the config map is encrypted.
func (ps *ProfileService) AddNotifyChannel(profileName string, channel *model.NotifyChannel) error {
	profile, err := ps.GetProfile(profileName)
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

			encryptedConfig[key] = string(encrypted)
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

	return ps.store.SaveProfile(profile)
}

// RemoveNotifyChannel removes a notification channel from a profile.
func (ps *ProfileService) RemoveNotifyChannel(profileName, channelID string) error {
	profile, err := ps.GetProfile(profileName)
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

	return ps.store.SaveProfile(profile)
}

// GetNotifyChannel retrieves a notification channel from a profile.
func (ps *ProfileService) GetNotifyChannel(profileName, channelID string) (*model.NotifyChannel, error) {
	profile, err := ps.GetProfile(profileName)
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

// GetNotifyChannelByType retrieves the first notification channel of a given type.
func (ps *ProfileService) GetNotifyChannelByType(profileName string, channelType model.ChannelType) (*model.NotifyChannel, error) {
	profile, err := ps.GetProfile(profileName)
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
func (ps *ProfileService) DecryptChannelConfig(profileName string, channel *model.NotifyChannel) (map[string]string, error) {
	decrypted := make(map[string]string)

	for key, value := range channel.Config {
		if isSensitiveKey(key) && value != "" {
			plaintext, err := tpm.DecryptToken([]byte(value), profileName, string(channel.Type))
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

// CreateProfileWithToken creates a new profile with a provided token.
func (ps *ProfileService) CreateProfileWithToken(name, host, username, token, workspace string, scopes []string) (*model.Profile, error) {
	exists, err := ps.store.ProfileExists(name)
	if err != nil {
		return nil, fmt.Errorf("failed to check profile existence: %w", err)
	}

	if exists {
		return nil, ErrProfileExists
	}

	if host == "" {
		host = model.DefaultHost()
	}

	if len(scopes) == 0 {
		scopes = model.DefaultScopes()
	}

	// Encrypt token
	encryptedToken, tokenStorage, err := ps.EncryptToken(token, name, host)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt token: %w", err)
	}

	// Check if this is the first profile
	profiles, err := ps.store.ListProfiles()
	if err != nil {
		return nil, fmt.Errorf("failed to list profiles: %w", err)
	}

	isFirstProfile := len(profiles) == 0

	profile := &model.Profile{
		Name:           name,
		Host:           host,
		User:           username,
		TokenStorage:   tokenStorage,
		Scopes:         scopes,
		Default:        isFirstProfile,
		EncryptedToken: encryptedToken,
		CreatedAt:      time.Now(),
		LastUsedAt:     time.Now(),
		Workspace:      workspace,
	}

	if err := ps.store.SaveProfile(profile); err != nil {
		return nil, fmt.Errorf("failed to save profile: %w", err)
	}

	return profile, nil
}

// isSensitiveKey returns true if the config key contains sensitive data.
func isSensitiveKey(key string) bool {
	sensitiveKeys := []string{
		"token", "secret", "password", "api_key", "webhook_url",
		"bot_token", "client_secret", "hmac_secret",
	}

	return slices.Contains(sensitiveKeys, key)
}
