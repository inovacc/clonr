//go:build !linux

package core

import "errors"

var (
	// ErrTPMNotAvailable is returned when TPM device is not accessible
	ErrTPMNotAvailable = errors.New("TPM device not available")

	// ErrNoSealedKey is returned when no sealed key exists
	ErrNoSealedKey = errors.New("no sealed key found")

	// ErrSealFailed is returned when sealing operation fails
	ErrSealFailed = errors.New("failed to seal key to TPM")

	// ErrUnsealFailed is returned when unsealing operation fails
	ErrUnsealFailed = errors.New("failed to unseal key from TPM")

	// ErrTPMNotSupported is returned on platforms without TPM support
	ErrTPMNotSupported = errors.New("TPM not supported on this platform")
)

// TPMKeyManager handles TPM-based key operations
type TPMKeyManager struct{}

// SealedData represents the data needed to unseal a key
type SealedData struct {
	PublicArea       []byte `json:"public_area"`
	PrivateArea      []byte `json:"private_area"`
	SealedBlob       []byte `json:"sealed_blob"`
	SealedBlobPublic []byte `json:"sealed_blob_public"`
}

// NewTPMKeyManager creates a new TPM key manager
// On non-Linux platforms, this always returns an error
func NewTPMKeyManager() (*TPMKeyManager, error) {
	return nil, ErrTPMNotSupported
}

// IsTPMAvailable checks if a TPM device is accessible
// On non-Linux platforms, this always returns false
func IsTPMAvailable() bool {
	return false
}

// SealKey seals a key to the TPM
// On non-Linux platforms, this always returns an error
func (t *TPMKeyManager) SealKey(key []byte) (*SealedData, error) {
	return nil, ErrTPMNotSupported
}

// UnsealKey unseals a key from the TPM
// On non-Linux platforms, this always returns an error
func (t *TPMKeyManager) UnsealKey(data *SealedData) ([]byte, error) {
	return nil, ErrTPMNotSupported
}

// GenerateAndSealKey generates a random key and seals it to the TPM
// On non-Linux platforms, this always returns an error
func (t *TPMKeyManager) GenerateAndSealKey() (*SealedData, error) {
	return nil, ErrTPMNotSupported
}
