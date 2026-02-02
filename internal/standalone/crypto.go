package standalone

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"fmt"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/hkdf"
	"golang.org/x/crypto/pbkdf2"
)

// Crypto constants
const (
	// Key derivation
	pbkdf2Iterations = 100000
	argon2Time       = 1
	argon2Memory     = 64 * 1024 // 64 MB
	argon2Threads    = 4
	argon2KeyLen     = 32

	// Sizes
	saltSize  = 16
	nonceSize = 12
	keySize   = 32

	// HKDF info strings
	hkdfInfoAPIAuth        = "standalone-api-auth"
	hkdfInfoDataEncryption = "standalone-data-encryption"
	hkdfInfoLocalStorage   = "standalone-local-storage"
)

// GenerateRandomBytes generates cryptographically secure random bytes.
func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return b, nil
}

// GenerateSalt generates a random salt for key derivation.
func GenerateSalt() ([]byte, error) {
	return GenerateRandomBytes(saltSize)
}

// DeriveKeyPBKDF2 derives a key using PBKDF2-SHA256.
func DeriveKeyPBKDF2(password string, salt []byte) []byte {
	return pbkdf2.Key([]byte(password), salt, pbkdf2Iterations, keySize, sha256.New)
}

// DeriveKeyArgon2 derives a key using Argon2id (stronger, used for local passwords).
func DeriveKeyArgon2(password string, salt []byte) []byte {
	return argon2.IDKey([]byte(password), salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)
}

// DeriveAPIKey derives an API authentication key using HKDF.
func DeriveAPIKey(masterKey []byte, instanceID string) ([]byte, error) {
	return deriveWithHKDF(masterKey, hkdfInfoAPIAuth, instanceID)
}

// DeriveEncryptionKey derives a data encryption key using HKDF.
func DeriveEncryptionKey(masterKey []byte, instanceID string) ([]byte, error) {
	return deriveWithHKDF(masterKey, hkdfInfoDataEncryption, instanceID)
}

// DeriveLocalStorageKey derives a local storage key using HKDF.
func DeriveLocalStorageKey(localKey []byte, connectionID string) ([]byte, error) {
	return deriveWithHKDF(localKey, hkdfInfoLocalStorage, connectionID)
}

// deriveWithHKDF derives a key using HKDF-SHA256.
func deriveWithHKDF(secret []byte, info, salt string) ([]byte, error) {
	hkdfReader := hkdf.New(sha256.New, secret, []byte(salt), []byte(info))
	key := make([]byte, keySize)
	if _, err := hkdfReader.Read(key); err != nil {
		return nil, fmt.Errorf("HKDF key derivation failed: %w", err)
	}
	return key, nil
}

// Encrypt encrypts data using AES-256-GCM.
// Returns: salt (16) + nonce (12) + ciphertext
func Encrypt(plaintext []byte, password string) ([]byte, error) {
	salt, err := GenerateSalt()
	if err != nil {
		return nil, err
	}

	key := DeriveKeyPBKDF2(password, salt)
	return encryptWithKey(plaintext, key, salt)
}

// EncryptWithKey encrypts data using a pre-derived key.
// Returns: nonce (12) + ciphertext
func EncryptWithKey(plaintext, key []byte) ([]byte, error) {
	return encryptWithKeyNoSalt(plaintext, key)
}

// encryptWithKey encrypts data and prepends salt.
func encryptWithKey(plaintext, key, salt []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce, err := GenerateRandomBytes(nonceSize)
	if err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	// Combine: salt + nonce + ciphertext
	result := make([]byte, 0, len(salt)+nonceSize+len(ciphertext))
	result = append(result, salt...)
	result = append(result, nonce...)
	result = append(result, ciphertext...)

	return result, nil
}

// encryptWithKeyNoSalt encrypts data without prepending salt.
func encryptWithKeyNoSalt(plaintext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce, err := GenerateRandomBytes(nonceSize)
	if err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	// Combine: nonce + ciphertext
	result := make([]byte, 0, nonceSize+len(ciphertext))
	result = append(result, nonce...)
	result = append(result, ciphertext...)

	return result, nil
}

// Decrypt decrypts data encrypted with Encrypt.
// Expects: salt (16) + nonce (12) + ciphertext
func Decrypt(ciphertext []byte, password string) ([]byte, error) {
	minLen := saltSize + nonceSize + 16 // 16 is GCM tag size
	if len(ciphertext) < minLen {
		return nil, fmt.Errorf("ciphertext too short")
	}

	salt := ciphertext[:saltSize]
	key := DeriveKeyPBKDF2(password, salt)

	return decryptWithKey(ciphertext, key)
}

// DecryptWithKey decrypts data using a pre-derived key.
// Expects: nonce (12) + ciphertext
func DecryptWithKey(ciphertext, key []byte) ([]byte, error) {
	return decryptWithKeyNoSalt(ciphertext, key)
}

// decryptWithKey decrypts data that has salt prepended.
func decryptWithKey(data, key []byte) ([]byte, error) {
	nonce := data[saltSize : saltSize+nonceSize]
	ciphertext := data[saltSize+nonceSize:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	return plaintext, nil
}

// decryptWithKeyNoSalt decrypts data without salt prefix.
func decryptWithKeyNoSalt(data, key []byte) ([]byte, error) {
	minLen := nonceSize + 16 // 16 is GCM tag size
	if len(data) < minLen {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce := data[:nonceSize]
	ciphertext := data[nonceSize:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	return plaintext, nil
}

// HashPassword creates an Argon2id hash of a password for verification.
func HashPassword(password string, salt []byte) []byte {
	return DeriveKeyArgon2(password, salt)
}

// VerifyPassword verifies a password against an Argon2id hash.
func VerifyPassword(password string, salt, hash []byte) bool {
	computed := DeriveKeyArgon2(password, salt)
	return subtle.ConstantTimeCompare(computed, hash) == 1
}

// ComputeKeyHint computes a hint (first 4 chars of hex hash) for key verification.
func ComputeKeyHint(key []byte) string {
	hash := sha256.Sum256(key)
	return fmt.Sprintf("%x", hash[:2]) // First 4 hex chars (2 bytes)
}

// SecureZero zeroes out a byte slice to prevent sensitive data from lingering in memory.
func SecureZero(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
