package tpm

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

const (
	// OpenPrefix marks data stored in plain text (no TPM available)
	OpenPrefix = "OPEN:"

	// EncPrefix marks encrypted data (TPM available)
	EncPrefix = "ENC:"
)

var (
	// ErrDecryptionFailed is returned when decryption fails
	ErrDecryptionFailed = errors.New("decryption failed: invalid ciphertext or key")

	// ErrEncryptionFailed is returned when encryption fails
	ErrEncryptionFailed = errors.New("encryption failed")

	// ErrNoEncryption indicates data is stored without encryption
	ErrNoEncryption = errors.New("no encryption available (no TPM)")
)

// getOrCreateKey retrieves the encryption key from TPM, creating it if necessary.
// Returns nil if TPM is not available (data will be stored unencrypted with OPEN: tag).
func getOrCreateKey() ([]byte, error) {
	// Only use TPM - no fallback to file-based encryption
	if !IsTPMAvailable() {
		return nil, ErrNoEncryption
	}

	// Auto-initialize if no sealed key exists
	if !HasTPMKey() {
		if err := InitializeTPMKey(); err != nil {
			return nil, fmt.Errorf("failed to initialize TPM key: %w", err)
		}
	}

	// Get the sealed key
	key, err := GetTPMSealedMasterKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get TPM key: %w", err)
	}

	return key, nil
}

// IsDataOpen checks if the stored data is in plain text (no encryption)
func IsDataOpen(data []byte) bool {
	return strings.HasPrefix(string(data), OpenPrefix)
}

// IsDataEncrypted checks if the stored data is encrypted
func IsDataEncrypted(data []byte) bool {
	return strings.HasPrefix(string(data), EncPrefix)
}

// IsEncryptionAvailable checks if TPM encryption is available
func IsEncryptionAvailable() bool {
	return IsTPMAvailable()
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

// EncryptToken encrypts a token using AES-256-GCM if TPM is available.
// If TPM is not available, stores the token in plain text with OPEN: prefix.
func EncryptToken(token, profileName, host string) ([]byte, error) {
	masterKey, err := getOrCreateKey()
	if err != nil {
		// No TPM available - store as plain text with OPEN: prefix
		if errors.Is(err, ErrNoEncryption) {
			return []byte(OpenPrefix + token), nil
		}

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

	// Prefix encrypted data with ENC:
	result := make([]byte, len(EncPrefix)+len(ciphertext))
	copy(result, EncPrefix)
	copy(result[len(EncPrefix):], ciphertext)

	return result, nil
}

// DecryptToken decrypts a token using AES-256-GCM or returns plain text if OPEN: prefix.
func DecryptToken(ciphertext []byte, profileName, host string) (string, error) {
	data := string(ciphertext)

	// Check for OPEN: prefix (plain text, no encryption)
	if after, ok := strings.CutPrefix(data, OpenPrefix); ok {
		return after, nil
	}

	// Check for ENC: prefix and strip it
	if after, ok := strings.CutPrefix(data, EncPrefix); ok {
		ciphertext = []byte(after)
	}

	// Need TPM to decrypt
	masterKey, err := getOrCreateKey()
	if err != nil {
		// If no TPM and data is encrypted, we can't decrypt
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

// GetKeePassPassword gets the KeePass password from TPM (no fallback)
func GetKeePassPassword() (string, error) {
	return GetKeePassPasswordTPM()
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

// GetKeePassPasswordTPM gets the KeePass password from a TPM-sealed key.
// If no sealed key exists, it will be created silently.
func GetKeePassPasswordTPM() (string, error) {
	if !IsTPMAvailable() {
		return "", ErrTPMNotAvailable
	}

	store, err := NewSealedKeyStore()
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrKeePassPasswordFailed, err)
	}

	// Auto-initialize if no sealed key exists
	if !store.HasSealedKey() {
		if err := InitializeTPMKey(); err != nil {
			return "", fmt.Errorf("%w: failed to initialize TPM key: %v", ErrKeePassPasswordFailed, err)
		}
	}

	instance, err := NewTPMKeyManager()
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrKeePassPasswordFailed, err)
	}

	sealedData, err := store.LoadSealedKey()
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrKeePassPasswordFailed, err)
	}

	key, err := instance.UnsealKey(sealedData)
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
