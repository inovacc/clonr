package tpm

import (
	"errors"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/inovacc/clonr/internal/application"
	"github.com/inovacc/sealbox"
)

var (
	// ErrKeystoreNotInitialized is returned when the keystore hasn't been initialized
	ErrKeystoreNotInitialized = errors.New("keystore not initialized")

	// globalKeystore is the singleton keystore instance
	globalKeystore *sealbox.Keystore
	keystoreMu     sync.RWMutex
	keystoreErr    error
	keystoreOnce   sync.Once

	// getAppDirectory is a function variable for testing
	getAppDirectory = application.GetApplicationDirectory
)

// initKeystore initializes the global keystore instance
func initKeystore() (*sealbox.Keystore, error) {
	keystoreOnce.Do(func() {
		appDir, err := getAppDirectory()
		if err != nil {
			keystoreErr = fmt.Errorf("failed to get application directory: %w", err)
			return
		}

		keystorePath := filepath.Join(appDir, ".clonr_keystore")

		// Try TPM first, fall back to password-less mode if unavailable
		var opts []sealbox.KeystoreOption
		if sealbox.IsAvailable() {
			opts = append(opts, sealbox.WithTPMRoot())
		} else {
			// Use a derived key from machine-specific data as fallback
			// This is less secure than TPM but better than nothing
			machineKey := getMachineKey()
			opts = append(opts, sealbox.WithPasswordRoot(machineKey))
		}
		opts = append(opts, sealbox.WithAutoSave())

		globalKeystore, keystoreErr = sealbox.Open(keystorePath, opts...)
	})

	return globalKeystore, keystoreErr
}

// getMachineKey returns a machine-specific key for non-TPM systems
// This provides some protection but is not as secure as TPM
func getMachineKey() []byte {
	// Use application directory as a simple machine identifier
	// In production, could use more robust machine fingerprinting
	appDir, _ := application.GetApplicationDirectory()
	return []byte("clonr-machine-key:" + appDir)
}

// GetKeystore returns the global keystore instance, initializing if needed
func GetKeystore() (*sealbox.Keystore, error) {
	return initKeystore()
}

// EnsureProfile ensures a profile exists in the keystore, creating if needed
func EnsureProfile(name string) error {
	ks, err := GetKeystore()
	if err != nil {
		return err
	}

	if ks.HasProfile(name) {
		return nil
	}

	return ks.CreateProfile(name)
}

// DeleteProfile removes a profile from the keystore
func DeleteProfile(name string) error {
	ks, err := GetKeystore()
	if err != nil {
		return err
	}

	if !ks.HasProfile(name) {
		return nil // Already doesn't exist
	}

	return ks.DeleteProfile(name)
}

// EncryptWithKeystore encrypts data using the keystore for a specific profile and data type
func EncryptWithKeystore(profile, dekType string, plaintext []byte) ([]byte, error) {
	ks, err := GetKeystore()
	if err != nil {
		return nil, err
	}

	// Ensure profile exists
	if !ks.HasProfile(profile) {
		if err := ks.CreateProfile(profile); err != nil {
			return nil, fmt.Errorf("failed to create profile: %w", err)
		}
	}

	return ks.Encrypt(profile, dekType, plaintext)
}

// DecryptWithKeystore decrypts data using the keystore for a specific profile and data type
func DecryptWithKeystore(profile, dekType string, ciphertext []byte) ([]byte, error) {
	ks, err := GetKeystore()
	if err != nil {
		return nil, err
	}

	return ks.Decrypt(profile, dekType, ciphertext)
}

// RotateProfileKey rotates the profile's master key and re-encrypts all DEKs
func RotateProfileKey(profile string) error {
	ks, err := GetKeystore()
	if err != nil {
		return err
	}

	return ks.RotateProfile(profile)
}

// CloseKeystore closes the global keystore
func CloseKeystore() error {
	keystoreMu.Lock()
	defer keystoreMu.Unlock()

	if globalKeystore != nil {
		err := globalKeystore.Close()
		globalKeystore = nil
		return err
	}
	return nil
}

// IsKeystoreAvailable checks if the keystore can be initialized
func IsKeystoreAvailable() bool {
	ks, err := GetKeystore()
	return err == nil && ks != nil && ks.IsOpen()
}
