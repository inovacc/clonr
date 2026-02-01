//go:build !linux

package tpm

import (
	"github.com/inovacc/sealbox"
)

// Re-export errors for backward compatibility
var (
	ErrTPMNotAvailable = sealbox.ErrTPMNotAvailable
	ErrTPMNotSupported = sealbox.ErrTPMNotSupported
	ErrNoSealedKey     = sealbox.ErrNoSealedKey
	ErrSealFailed      = sealbox.ErrSealFailed
	ErrUnsealFailed    = sealbox.ErrUnsealFailed
)

// SealedData is an alias for backward compatibility
type SealedData = sealbox.SealedData

// TPMKeyManager wraps the sealbox KeyManager for backward compatibility
type TPMKeyManager struct{}

// NewTPMKeyManager creates a new TPM key manager
// On non-Linux platforms, this always returns an error
func NewTPMKeyManager() (*TPMKeyManager, error) {
	return nil, ErrTPMNotSupported
}

// IsTPMAvailable checks if a TPM device is accessible
// On non-Linux platforms, this always returns false
func IsTPMAvailable() bool {
	return sealbox.IsAvailable()
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

// Close is a no-op on unsupported platforms
func (t *TPMKeyManager) Close() error {
	return nil
}
