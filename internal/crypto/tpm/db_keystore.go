package tpm

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/inovacc/clonr/internal/store"
	"github.com/inovacc/sealbox"
)

// DBKeyStore implements a sealed key store backed by database storage
// instead of the filesystem. This provides better integration with
// SQLite and avoids scattered key files.
type DBKeyStore struct {
	store   store.Store
	keyType string // "tpm", "password", "software"
}

// NewDBKeyStore creates a new database-backed sealed key store
func NewDBKeyStore(s store.Store, keyType string) *DBKeyStore {
	if keyType == "" {
		keyType = "tpm"
	}
	return &DBKeyStore{
		store:   s,
		keyType: keyType,
	}
}

// Save stores the sealed data in the database
func (d *DBKeyStore) Save(data *sealbox.SealedData) error {
	if data == nil {
		return errors.New("sealed data is nil")
	}

	// Serialize SealedData to JSON
	sealedBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// Check if we already have a sealed key to preserve created_at
	existing, err := d.store.GetSealedKey()
	var createdAt time.Time
	if err == nil && existing != nil {
		createdAt = existing.CreatedAt
	} else {
		createdAt = time.Now()
	}

	return d.store.SaveSealedKey(&store.SealedKeyData{
		SealedData:   sealedBytes,
		Version:      data.GetVersion(),
		KeyType:      d.keyType,
		Metadata:     nil,
		CreatedAt:    createdAt,
		RotatedAt:    time.Now(),
		LastAccessed: time.Now(),
	})
}

// Load retrieves the sealed data from the database
func (d *DBKeyStore) Load() (*sealbox.SealedData, error) {
	keyData, err := d.store.GetSealedKey()
	if err != nil {
		return nil, err
	}
	if keyData == nil {
		return nil, errors.New("no sealed key found")
	}

	var sealedData sealbox.SealedData
	if err := json.Unmarshal(keyData.SealedData, &sealedData); err != nil {
		return nil, err
	}

	return &sealedData, nil
}

// Exists checks if a sealed key exists in the database
func (d *DBKeyStore) Exists() bool {
	exists, err := d.store.HasSealedKey()
	return err == nil && exists
}

// Delete removes the sealed key from the database
func (d *DBKeyStore) Delete() error {
	return d.store.DeleteSealedKey()
}

// Path returns a description of the storage location (for compatibility)
func (d *DBKeyStore) Path() string {
	return "database://sealed_keys"
}

// GetKeyType returns the key type (tpm, password, software)
func (d *DBKeyStore) GetKeyType() string {
	return d.keyType
}

// GetMetadata returns the sealed key metadata
func (d *DBKeyStore) GetMetadata() (*store.SealedKeyData, error) {
	return d.store.GetSealedKey()
}

// KeyStoreAdapter allows DBKeyStore to be used where FileKeyStore is expected
// by implementing the same interface
type KeyStoreAdapter interface {
	Save(data *sealbox.SealedData) error
	Load() (*sealbox.SealedData, error)
	Exists() bool
	Delete() error
	Path() string
}

// Ensure DBKeyStore implements KeyStoreAdapter
var _ KeyStoreAdapter = (*DBKeyStore)(nil)
