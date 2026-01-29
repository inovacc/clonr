package core

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

const (
	// keyFileName is the name of the file storing the encryption key
	keyFileName = ".clonr_key"
)

var (
	// ErrDecryptionFailed is returned when decryption fails
	ErrDecryptionFailed = errors.New("decryption failed: invalid ciphertext or key")

	// ErrEncryptionFailed is returned when encryption fails
	ErrEncryptionFailed = errors.New("encryption failed")
)

// getKeyPath returns the path to the encryption key file
func getKeyPath() (string, error) {
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
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}

		configDir = filepath.Join(home, "Library", "Application Support", "clonr")
	default: // linux and others
		configDir = os.Getenv("XDG_CONFIG_HOME")
		if configDir == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", fmt.Errorf("failed to get home directory: %w", err)
			}

			configDir = filepath.Join(home, ".config", "clonr")
		} else {
			configDir = filepath.Join(configDir, "clonr")
		}
	}

	return filepath.Join(configDir, keyFileName), nil
}

// getOrCreateKey retrieves the encryption key, creating it if necessary
func getOrCreateKey() ([]byte, error) {
	keyPath, err := getKeyPath()
	if err != nil {
		return nil, err
	}

	// Try to read existing key
	keyHex, err := os.ReadFile(keyPath)
	if err == nil {
		key, decodeErr := hex.DecodeString(string(keyHex))
		if decodeErr == nil && len(key) == 32 {
			return key, nil
		}
	}

	// Generate new key
	key := make([]byte, 32) // AES-256
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("failed to generate encryption key: %w", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(keyPath), 0700); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Save key with restrictive permissions
	keyHex = []byte(hex.EncodeToString(key))
	if err := os.WriteFile(keyPath, keyHex, 0600); err != nil {
		return nil, fmt.Errorf("failed to save encryption key: %w", err)
	}

	return key, nil
}

// deriveKey creates a profile-specific key from the master key
func deriveKey(masterKey []byte, profileName, host string) []byte {
	suffix := []byte(profileName + ":" + host)
	data := make([]byte, 0, len(masterKey)+len(suffix))
	data = append(data, masterKey...)
	data = append(data, suffix...)

	hash := sha256.Sum256(data)

	return hash[:]
}

// EncryptToken encrypts a token using AES-256-GCM
func EncryptToken(token, profileName, host string) ([]byte, error) {
	masterKey, err := getOrCreateKey()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrEncryptionFailed, err)
	}

	key := deriveKey(masterKey, profileName, host)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrEncryptionFailed, err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrEncryptionFailed, err)
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrEncryptionFailed, err)
	}

	ciphertext := aesGCM.Seal(nonce, nonce, []byte(token), nil)

	return ciphertext, nil
}

// DecryptToken decrypts a token using AES-256-GCM
func DecryptToken(ciphertext []byte, profileName, host string) (string, error) {
	masterKey, err := getOrCreateKey()
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrDecryptionFailed, err)
	}

	key := deriveKey(masterKey, profileName, host)

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrDecryptionFailed, err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrDecryptionFailed, err)
	}

	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", ErrDecryptionFailed
	}

	nonce, ciphertextBytes := ciphertext[:nonceSize], ciphertext[nonceSize:]

	plaintext, err := aesGCM.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", ErrDecryptionFailed
	}

	return string(plaintext), nil
}
