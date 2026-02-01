//go:build linux

package core

import (
	"github.com/inovacc/sealbox"
)

// Re-export errors for backward compatibility
var (
	ErrTPMNotAvailable = sealbox.ErrTPMNotAvailable
	ErrNoSealedKey     = sealbox.ErrNoSealedKey
	ErrSealFailed      = sealbox.ErrSealFailed
	ErrUnsealFailed    = sealbox.ErrUnsealFailed
)

// SealedData is an alias for backward compatibility
type SealedData = sealbox.SealedData

// TPMKeyManager wraps the sealbox KeyManager for backward compatibility
type TPMKeyManager struct {
	km sealbox.KeyManager
}

// NewTPMKeyManager creates a new TPM key manager
func NewTPMKeyManager() (*TPMKeyManager, error) {
	km, err := sealbox.NewKeyManager()
	if err != nil {
		return nil, err
	}

	return &TPMKeyManager{km: km}, nil
}

// IsTPMAvailable checks if a TPM device is accessible
func IsTPMAvailable() bool {
	return sealbox.IsAvailable()
}

// SealKey seals a key to the TPM
func (t *TPMKeyManager) SealKey(key []byte) (*SealedData, error) {
	return t.km.SealKey(key)
}

// UnsealKey unseals a key from the TPM
func (t *TPMKeyManager) UnsealKey(data *SealedData) ([]byte, error) {
	return t.km.UnsealKey(data)
}

// GenerateAndSealKey generates a random key and seals it to the TPM
func (t *TPMKeyManager) GenerateAndSealKey() (*SealedData, error) {
	return t.km.GenerateAndSealKey()
}

// Close releases TPM resources
func (t *TPMKeyManager) Close() error {
	return t.km.Close()
}
