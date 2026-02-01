package core

import (
	"path/filepath"

	"github.com/inovacc/sealbox"
)

var cacheDir string

func init() {
	var err error

	cacheDir, err = GetClonrConfigDir()
	if err != nil {
		panic(err)
	}

	cacheDir = filepath.Join(cacheDir, ".clonr_sealed_key")
}

// sealboxOpts returns the KeyStore options for clonr
func sealboxOpts() []sealbox.KeyStoreOption {
	return []sealbox.KeyStoreOption{
		sealbox.WithStorePath(cacheDir),
	}
}

// SealedKeyStore wraps the sealbox KeyStore for backward compatibility
type SealedKeyStore struct {
	store *sealbox.FileKeyStore
}

// NewSealedKeyStore creates a new sealed key store
func NewSealedKeyStore() (*SealedKeyStore, error) {
	store, err := sealbox.NewKeyStore(sealboxOpts()...)
	if err != nil {
		return nil, err
	}

	return &SealedKeyStore{store: store}, nil
}

// SaveSealedKey saves the sealed key data to disk
func (s *SealedKeyStore) SaveSealedKey(data *sealbox.SealedData) error {
	return s.store.Save(data)
}

// LoadSealedKey loads the sealed key data from disk
func (s *SealedKeyStore) LoadSealedKey() (*sealbox.SealedData, error) {
	return s.store.Load()
}

// HasSealedKey checks if a sealed key exists on disk
func (s *SealedKeyStore) HasSealedKey() bool {
	return s.store.Exists()
}

// DeleteSealedKey removes the sealed key from disk
func (s *SealedKeyStore) DeleteSealedKey() error {
	return s.store.Delete()
}

// GetStorePath returns the path where the sealed key is stored
func (s *SealedKeyStore) GetStorePath() string {
	return s.store.Path()
}

// InitializeTPMKey creates and seals a new key to the TPM
// This should be run once during setup
func InitializeTPMKey() error {
	return sealbox.Initialize(sealboxOpts()...)
}

// ResetTPMKey removes the existing TPM-sealed key
func ResetTPMKey() error {
	return sealbox.Reset(sealboxOpts()...)
}

// HasTPMKey checks if a TPM-sealed key exists
func HasTPMKey() bool {
	return sealbox.HasKey(sealboxOpts()...)
}

// GetTPMKeyStorePath returns the path where the TPM-sealed key is stored
func GetTPMKeyStorePath() (string, error) {
	return sealbox.GetKeyStorePath(sealboxOpts()...)
}

// GetTPMSealedMasterKey retrieves the master encryption key from TPM
// Returns the unsealed key that can be used for encryption operations
func GetTPMSealedMasterKey() ([]byte, error) {
	return sealbox.GetSealedMasterKey(sealboxOpts()...)
}
