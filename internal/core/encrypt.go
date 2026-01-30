package core

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/term"
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
// Priority: 1. TPM sealed key, 2. File-based key
func getOrCreateKey() ([]byte, error) {
	// Try TPM first if available and sealed key exists
	if key, err := GetTPMSealedMasterKey(); err == nil {
		return key, nil
	}

	// Fall back to file-based key
	return getOrCreateFileKey()
}

// getOrCreateFileKey retrieves the file-based encryption key, creating it if necessary
func getOrCreateFileKey() ([]byte, error) {
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

// HasFileBasedKey checks if a file-based encryption key exists
func HasFileBasedKey() bool {
	keyPath, err := getKeyPath()
	if err != nil {
		return false
	}

	_, err = os.Stat(keyPath)

	return err == nil
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

// ErrKeePassPasswordFailed is returned when KeePass password retrieval fails
var ErrKeePassPasswordFailed = errors.New("failed to get KeePass password")

// GetKeePassPassword gets the KeePass password with TPM fallback
// Priority: 1. TPM sealed key, 2. Environment variable, 3. User prompt
func GetKeePassPassword() (string, error) {
	// Try TPM first if available and sealed key exists
	if tpmPassword, err := GetKeePassPasswordTPM(); err == nil {
		return tpmPassword, nil
	}

	// Fall back to traditional password retrieval
	passphrase, err := getKeePassPassphrase()
	if err != nil {
		return "", fmt.Errorf("error reading KeePass password: %w", err)
	}

	return DeriveKeePassPasswordFromPassphrase(passphrase), nil
}

// getKeePassPassphrase gets the KeePass password from env var or prompts the user
func getKeePassPassphrase() (string, error) {
	// Check for environment variable first
	if password := os.Getenv("CLONR_KEEPASS_PASSWORD"); password != "" {
		return password, nil
	}

	// Fall back to prompting
	return PromptForPassword("Enter KeePass master password: ")
}

// PromptForPassword prompts the user for a password without echoing
func PromptForPassword(prompt string) (string, error) {
	_, _ = fmt.Fprint(os.Stderr, prompt)

	fd := int(os.Stdin.Fd())

	bytePassword, err := term.ReadPassword(fd)

	_, _ = fmt.Fprintln(os.Stderr)

	if err != nil {
		return "", err
	}

	return string(bytePassword), nil
}

// GetKeePassPasswordTPM gets the KeePass password from TPM-sealed key
func GetKeePassPasswordTPM() (string, error) {
	if !IsTPMAvailable() {
		return "", ErrTPMNotAvailable
	}

	store, err := NewSealedKeyStore()
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrKeePassPasswordFailed, err)
	}

	if !store.HasSealedKey() {
		return "", ErrNoSealedKey
	}

	tpm, err := NewTPMKeyManager()
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrKeePassPasswordFailed, err)
	}

	sealedData, err := store.LoadSealedKey()
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrKeePassPasswordFailed, err)
	}

	key, err := tpm.UnsealKey(sealedData)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrKeePassPasswordFailed, err)
	}

	return deriveKeePassPasswordFromKey(key), nil
}

// deriveKeePassPasswordFromKey derives a KeePass-compatible password from raw key bytes
func deriveKeePassPasswordFromKey(key []byte) string {
	var (
		result  []byte
		counter byte = 1
		outLen       = 30 // 30 bytes = 240 bits
	)

	for len(result) < outLen {
		mac := hmac.New(sha256.New, key)
		mac.Write([]byte("keepass"))
		mac.Write([]byte{counter})
		result = append(result, mac.Sum(nil)...)
		counter++
	}

	str := base64.RawURLEncoding.EncodeToString(result[:outLen])
	str = strings.ReplaceAll(strings.ReplaceAll(str, "_", ""), "-", "")

	return str
}

// DeriveKeePassPasswordFromPassphrase derives KeePass password from a user passphrase
func DeriveKeePassPasswordFromPassphrase(passphrase string) string {
	var (
		result  []byte
		counter byte = 1
		outLen       = 30 // 30 bytes = 240 bits
	)

	for len(result) < outLen {
		mac := hmac.New(sha256.New, []byte(passphrase))
		mac.Write([]byte("keepass"))
		mac.Write([]byte{counter})
		result = append(result, mac.Sum(nil)...)
		counter++
	}

	str := base64.RawURLEncoding.EncodeToString(result[:outLen])
	str = strings.ReplaceAll(strings.ReplaceAll(str, "_", ""), "-", "")

	return str
}
