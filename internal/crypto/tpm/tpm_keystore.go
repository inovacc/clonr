package tpm

import (
	"path/filepath"
	"sync"

	"github.com/inovacc/clonr/internal/application"
	"github.com/inovacc/clonr/internal/store"
	"github.com/inovacc/sealbox"
)

var (
	cacheDir string

	// globalDBStore holds the database store for sealed key storage
	globalDBStore store.Store
	dbStoreMu     sync.RWMutex
)

func init() {
	var err error

	cacheDir, err = application.GetApplicationDirectory()
	if err != nil {
		panic(err)
	}

	cacheDir = filepath.Join(cacheDir, ".clonr_sealed_key")
}

// SetDBStore configures the database store for sealed key storage.
// When set, sealed keys are stored in SQLite instead of the filesystem.
// This should be called early in application startup.
func SetDBStore(s store.Store) {
	dbStoreMu.Lock()
	defer dbStoreMu.Unlock()

	globalDBStore = s
}

// GetDBStore returns the configured database store, or nil if not set
func GetDBStore() store.Store {
	dbStoreMu.RLock()
	defer dbStoreMu.RUnlock()

	return globalDBStore
}

// useDBStorage returns true if database storage is configured
func useDBStorage() bool {
	return GetDBStore() != nil
}

// sealboxOpts returns the KeyStore options for clonr (file-based fallback)
func sealboxOpts() []sealbox.KeyStoreOption {
	return []sealbox.KeyStoreOption{
		sealbox.WithStorePath(cacheDir),
	}
}

// SealedKeyStore wraps the sealbox KeyStore for backward compatibility
// Supports both file-based and database-backed storage
type SealedKeyStore struct {
	fileStore *sealbox.FileKeyStore
	dbStore   *DBKeyStore
	useDB     bool
}

// NewSealedKeyStore creates a new sealed key store
// Uses database storage if configured, otherwise falls back to file storage
func NewSealedKeyStore() (*SealedKeyStore, error) {
	if s := GetDBStore(); s != nil {
		keyType := "tpm"
		if !sealbox.IsAvailable() {
			keyType = "software"
		}

		return &SealedKeyStore{
			dbStore: NewDBKeyStore(s, keyType),
			useDB:   true,
		}, nil
	}

	// Fall back to file-based storage
	store, err := sealbox.NewKeyStore(sealboxOpts()...)
	if err != nil {
		return nil, err
	}

	return &SealedKeyStore{fileStore: store, useDB: false}, nil
}

// SaveSealedKey saves the sealed key data
func (s *SealedKeyStore) SaveSealedKey(data *sealbox.SealedData) error {
	if s.useDB {
		return s.dbStore.Save(data)
	}

	return s.fileStore.Save(data)
}

// LoadSealedKey loads the sealed key data
func (s *SealedKeyStore) LoadSealedKey() (*sealbox.SealedData, error) {
	if s.useDB {
		return s.dbStore.Load()
	}

	return s.fileStore.Load()
}

// HasSealedKey checks if a sealed key exists
func (s *SealedKeyStore) HasSealedKey() bool {
	if s.useDB {
		return s.dbStore.Exists()
	}

	return s.fileStore.Exists()
}

// DeleteSealedKey removes the sealed key
func (s *SealedKeyStore) DeleteSealedKey() error {
	if s.useDB {
		return s.dbStore.Delete()
	}

	return s.fileStore.Delete()
}

// GetStorePath returns the path where the sealed key is stored
func (s *SealedKeyStore) GetStorePath() string {
	if s.useDB {
		return s.dbStore.Path()
	}

	return s.fileStore.Path()
}

// InitializeTPMKey creates and seals a new key to the TPM
// This should be run once during setup
func InitializeTPMKey() error {
	if useDBStorage() {
		return initializeTPMKeyWithDB()
	}

	return sealbox.Initialize(sealboxOpts()...)
}

// initializeTPMKeyWithDB initializes TPM key and stores in database
func initializeTPMKeyWithDB() error {
	km, err := sealbox.NewKeyManager()
	if err != nil {
		return err
	}
	defer km.Close()

	sealed, err := km.GenerateAndSealKey()
	if err != nil {
		return err
	}

	dbStore := NewDBKeyStore(GetDBStore(), "tpm")

	return dbStore.Save(sealed)
}

// ResetTPMKey removes the existing TPM-sealed key
func ResetTPMKey() error {
	if useDBStorage() {
		dbStore := NewDBKeyStore(GetDBStore(), "tpm")
		return dbStore.Delete()
	}

	return sealbox.Reset(sealboxOpts()...)
}

// HasTPMKey checks if a TPM-sealed key exists
func HasTPMKey() bool {
	if useDBStorage() {
		dbStore := NewDBKeyStore(GetDBStore(), "tpm")
		return dbStore.Exists()
	}

	return sealbox.HasKey(sealboxOpts()...)
}

// GetTPMKeyStorePath returns the path where the TPM-sealed key is stored
func GetTPMKeyStorePath() (string, error) {
	if useDBStorage() {
		return "database://sealed_keys", nil
	}

	return sealbox.GetKeyStorePath(sealboxOpts()...)
}

// GetTPMSealedMasterKey retrieves the master encryption key from TPM
// Returns the unsealed key that can be used for encryption operations
func GetTPMSealedMasterKey() ([]byte, error) {
	if useDBStorage() {
		return getTPMSealedMasterKeyFromDB()
	}

	return sealbox.GetSealedMasterKey(sealboxOpts()...)
}

// getTPMSealedMasterKeyFromDB retrieves and unseals the master key from database
func getTPMSealedMasterKeyFromDB() ([]byte, error) {
	dbStore := NewDBKeyStore(GetDBStore(), "tpm")

	sealed, err := dbStore.Load()
	if err != nil {
		return nil, err
	}

	km, err := sealbox.NewKeyManager()
	if err != nil {
		return nil, err
	}
	defer km.Close()

	return km.UnsealKey(sealed)
}
