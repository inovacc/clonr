package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const (
	// sealedKeyFileName is the name of the file storing the sealed key
	sealedKeyFileName = ".clonr_sealed_key"
)

// SealedKeyStore manages sealed key blobs on disk
type SealedKeyStore struct {
	storePath string
}

// NewSealedKeyStore creates a new sealed key store
func NewSealedKeyStore() (*SealedKeyStore, error) {
	var configDir string

	switch runtime.GOOS {
	case "windows":
		configDir = os.Getenv("LOCALAPPDATA")
		if configDir == "" {
			configDir = os.Getenv("APPDATA")
		}

		configDir = filepath.Join(configDir, "clonr")
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}

		configDir = filepath.Join(home, "Library", "Application Support", "clonr")
	default: // linux and others
		configDir = os.Getenv("XDG_CONFIG_HOME")
		if configDir == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("failed to get home directory: %w", err)
			}

			configDir = filepath.Join(home, ".config", "clonr")
		} else {
			configDir = filepath.Join(configDir, "clonr")
		}
	}

	storePath := filepath.Join(configDir, sealedKeyFileName)

	return &SealedKeyStore{
		storePath: storePath,
	}, nil
}

// SaveSealedKey saves the sealed key data to disk
func (s *SealedKeyStore) SaveSealedKey(data *SealedData) error {
	// Ensure directory exists
	dir := filepath.Dir(s.storePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Marshal the sealed data to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal sealed data: %w", err)
	}

	// Write to file with restricted permissions (owner read/write only)
	if err := os.WriteFile(s.storePath, jsonData, 0600); err != nil {
		return fmt.Errorf("failed to write sealed key file: %w", err)
	}

	return nil
}

// LoadSealedKey loads the sealed key data from disk
func (s *SealedKeyStore) LoadSealedKey() (*SealedData, error) {
	jsonData, err := os.ReadFile(s.storePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNoSealedKey
		}

		return nil, fmt.Errorf("failed to read sealed key file: %w", err)
	}

	var data SealedData
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal sealed data: %w", err)
	}

	return &data, nil
}

// HasSealedKey checks if a sealed key exists on disk
func (s *SealedKeyStore) HasSealedKey() bool {
	_, err := os.Stat(s.storePath)

	return err == nil
}

// DeleteSealedKey removes the sealed key from disk
func (s *SealedKeyStore) DeleteSealedKey() error {
	if err := os.Remove(s.storePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete sealed key: %w", err)
	}

	return nil
}

// GetStorePath returns the path where the sealed key is stored
func (s *SealedKeyStore) GetStorePath() string {
	return s.storePath
}

// InitializeTPMKey creates and seals a new key to the TPM
// This should be run once during setup
func InitializeTPMKey() error {
	if !IsTPMAvailable() {
		return ErrTPMNotAvailable
	}

	tpm, err := NewTPMKeyManager()
	if err != nil {
		return fmt.Errorf("failed to create TPM key manager: %w", err)
	}

	// Generate and seal a new random key
	sealedData, err := tpm.GenerateAndSealKey()
	if err != nil {
		return fmt.Errorf("failed to generate and seal key: %w", err)
	}

	// Save the sealed data to disk
	store, err := NewSealedKeyStore()
	if err != nil {
		return fmt.Errorf("failed to create key store: %w", err)
	}

	if err := store.SaveSealedKey(sealedData); err != nil {
		return fmt.Errorf("failed to save sealed key: %w", err)
	}

	return nil
}

// ResetTPMKey removes the existing TPM-sealed key
func ResetTPMKey() error {
	store, err := NewSealedKeyStore()
	if err != nil {
		return fmt.Errorf("failed to create key store: %w", err)
	}

	return store.DeleteSealedKey()
}

// HasTPMKey checks if a TPM-sealed key exists
func HasTPMKey() bool {
	store, err := NewSealedKeyStore()
	if err != nil {
		return false
	}

	return store.HasSealedKey()
}

// GetTPMKeyStorePath returns the path where the TPM-sealed key is stored
func GetTPMKeyStorePath() (string, error) {
	store, err := NewSealedKeyStore()
	if err != nil {
		return "", err
	}

	return store.GetStorePath(), nil
}

// GetTPMSealedMasterKey retrieves the master encryption key from TPM
// Returns the unsealed key that can be used for encryption operations
func GetTPMSealedMasterKey() ([]byte, error) {
	if !IsTPMAvailable() {
		return nil, ErrTPMNotAvailable
	}

	store, err := NewSealedKeyStore()
	if err != nil {
		return nil, fmt.Errorf("failed to create key store: %w", err)
	}

	if !store.HasSealedKey() {
		return nil, ErrNoSealedKey
	}

	tpm, err := NewTPMKeyManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create TPM key manager: %w", err)
	}

	sealedData, err := store.LoadSealedKey()
	if err != nil {
		return nil, fmt.Errorf("failed to load sealed key: %w", err)
	}

	key, err := tpm.UnsealKey(sealedData)
	if err != nil {
		return nil, fmt.Errorf("failed to unseal key: %w", err)
	}

	return key, nil
}
