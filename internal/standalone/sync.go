package standalone

import (
	"encoding/json"
	"fmt"
	"time"
)

// SyncState represents the encryption state of synced data.
type SyncState string

const (
	// SyncStateEncrypted means data is stored encrypted, awaiting decryption key
	SyncStateEncrypted SyncState = "encrypted"
	// SyncStateDecrypted means data has been decrypted and is usable
	SyncStateDecrypted SyncState = "decrypted"
	// SyncStatePending means sync is in progress
	SyncStatePending SyncState = "pending"
)

// SyncedData represents data synced from a standalone instance.
// Data is stored encrypted until the user provides a decryption key.
type SyncedData struct {
	ID             string    `json:"id"`
	ConnectionName string    `json:"connection_name"`
	InstanceID     string    `json:"instance_id"`
	DataType       string    `json:"data_type"` // "profile", "workspace", "repo", "config"
	Name           string    `json:"name"`      // Original name of the item
	EncryptedData  []byte    `json:"encrypted_data"`
	Nonce          []byte    `json:"nonce"`
	State          SyncState `json:"state"`
	SyncedAt       time.Time `json:"synced_at"`
	DecryptedAt    time.Time `json:"decrypted_at,omitempty"`
	Checksum       string    `json:"checksum"` // For detecting changes
}

// ServerEncryptionConfig holds the server's encryption configuration.
type ServerEncryptionConfig struct {
	Enabled      bool      `json:"enabled"`
	KeyHash      []byte    `json:"key_hash"` // Argon2 hash for verification
	Salt         []byte    `json:"salt"`     // Salt for key derivation
	KeyHint      string    `json:"key_hint"` // First 4 chars of derived key hash
	ConfiguredAt time.Time `json:"configured_at"`
}

// EncryptForSync encrypts data for transmission/storage using the standalone key.
// The data can only be decrypted with the correct decryption key.
func EncryptForSync(data []byte, encryptionKey []byte) (*SyncedData, error) {
	if len(encryptionKey) != keySize {
		return nil, fmt.Errorf("invalid encryption key size: got %d, want %d", len(encryptionKey), keySize)
	}

	// Generate nonce
	nonce, err := GenerateRandomBytes(nonceSize)
	if err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt with the key
	encrypted, err := EncryptWithKey(data, encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt: %w", err)
	}

	return &SyncedData{
		EncryptedData: encrypted,
		Nonce:         nonce,
		State:         SyncStateEncrypted,
		SyncedAt:      time.Now(),
	}, nil
}

// DecryptSyncedData decrypts synced data using the provided key.
func DecryptSyncedData(synced *SyncedData, decryptionKey []byte) ([]byte, error) {
	if synced.State == SyncStateDecrypted {
		return nil, fmt.Errorf("data is already decrypted")
	}

	if len(decryptionKey) != keySize {
		return nil, fmt.Errorf("invalid decryption key size")
	}

	decrypted, err := DecryptWithKey(synced.EncryptedData, decryptionKey)
	if err != nil {
		return nil, fmt.Errorf("decryption failed (wrong key?): %w", err)
	}

	return decrypted, nil
}

// SyncPackage represents a batch of encrypted data to sync.
type SyncPackage struct {
	Version       int                    `json:"version"`
	InstanceID    string                 `json:"instance_id"`
	CreatedAt     time.Time              `json:"created_at"`
	EncryptionKey string                 `json:"encryption_key_hint"` // Hint for verification
	Items         []SyncPackageItem      `json:"items"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// SyncPackageItem is a single item in a sync package.
type SyncPackageItem struct {
	Type          string    `json:"type"` // "profile", "workspace", "repo", "config"
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	EncryptedData []byte    `json:"encrypted_data"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// CreateSyncPackage creates an encrypted sync package from multiple data items.
func CreateSyncPackage(instanceID string, encryptionKey []byte, items map[string][]byte) (*SyncPackage, error) {
	pkg := &SyncPackage{
		Version:       1,
		InstanceID:    instanceID,
		CreatedAt:     time.Now(),
		EncryptionKey: ComputeKeyHint(encryptionKey),
		Items:         make([]SyncPackageItem, 0, len(items)),
	}

	for name, data := range items {
		encrypted, err := EncryptWithKey(data, encryptionKey)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt %s: %w", name, err)
		}

		pkg.Items = append(pkg.Items, SyncPackageItem{
			ID:            name,
			Name:          name,
			EncryptedData: encrypted,
			UpdatedAt:     time.Now(),
		})
	}

	return pkg, nil
}

// PendingSyncStore manages encrypted data awaiting decryption.
type PendingSyncStore struct {
	items map[string]*SyncedData
}

// NewPendingSyncStore creates a new pending sync store.
func NewPendingSyncStore() *PendingSyncStore {
	return &PendingSyncStore{
		items: make(map[string]*SyncedData),
	}
}

// Add adds encrypted data to the pending store.
func (s *PendingSyncStore) Add(synced *SyncedData) {
	key := fmt.Sprintf("%s:%s:%s", synced.ConnectionName, synced.DataType, synced.Name)
	s.items[key] = synced
}

// Get retrieves pending data by connection, type, and name.
func (s *PendingSyncStore) Get(connectionName, dataType, name string) *SyncedData {
	key := fmt.Sprintf("%s:%s:%s", connectionName, dataType, name)
	return s.items[key]
}

// List returns all pending items for a connection.
func (s *PendingSyncStore) List(connectionName string) []*SyncedData {
	var result []*SyncedData
	for _, item := range s.items {
		if item.ConnectionName == connectionName {
			result = append(result, item)
		}
	}
	return result
}

// ListByState returns all items with a specific state.
func (s *PendingSyncStore) ListByState(state SyncState) []*SyncedData {
	var result []*SyncedData
	for _, item := range s.items {
		if item.State == state {
			result = append(result, item)
		}
	}
	return result
}

// Remove removes an item from the pending store.
func (s *PendingSyncStore) Remove(connectionName, dataType, name string) {
	key := fmt.Sprintf("%s:%s:%s", connectionName, dataType, name)
	delete(s.items, key)
}

// DecryptAll attempts to decrypt all pending items with the provided key.
// Returns the number of successfully decrypted items and any errors.
func (s *PendingSyncStore) DecryptAll(connectionName string, decryptionKey []byte) (int, []error) {
	var decrypted int
	var errors []error

	for key, item := range s.items {
		if item.ConnectionName != connectionName {
			continue
		}
		if item.State != SyncStateEncrypted {
			continue
		}

		_, err := DecryptSyncedData(item, decryptionKey)
		if err != nil {
			errors = append(errors, fmt.Errorf("%s: %w", key, err))
			continue
		}

		item.State = SyncStateDecrypted
		item.DecryptedAt = time.Now()
		decrypted++
	}

	return decrypted, errors
}

// EncryptionKeyManager manages encryption keys for standalone mode.
type EncryptionKeyManager struct {
	config *ServerEncryptionConfig
}

// NewEncryptionKeyManager creates a new key manager.
func NewEncryptionKeyManager() *EncryptionKeyManager {
	return &EncryptionKeyManager{}
}

// SetupKey sets up the server encryption key.
func (m *EncryptionKeyManager) SetupKey(password string) (*ServerEncryptionConfig, error) {
	if len(password) < 8 {
		return nil, fmt.Errorf("password must be at least 8 characters")
	}

	salt, err := GenerateSalt()
	if err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	// Derive key and hash for verification
	derivedKey := DeriveKeyArgon2(password, salt)
	keyHash := HashPassword(password, salt)

	m.config = &ServerEncryptionConfig{
		Enabled:      true,
		KeyHash:      keyHash,
		Salt:         salt,
		KeyHint:      ComputeKeyHint(derivedKey),
		ConfiguredAt: time.Now(),
	}

	return m.config, nil
}

// VerifyKey verifies the provided password matches the configured key.
func (m *EncryptionKeyManager) VerifyKey(password string) bool {
	if m.config == nil || !m.config.Enabled {
		return false
	}
	return VerifyPassword(password, m.config.Salt, m.config.KeyHash)
}

// DeriveKey derives the encryption key from the password.
func (m *EncryptionKeyManager) DeriveKey(password string) ([]byte, error) {
	if m.config == nil || !m.config.Enabled {
		return nil, fmt.Errorf("encryption not configured")
	}
	if !m.VerifyKey(password) {
		return nil, fmt.Errorf("invalid password")
	}
	return DeriveKeyArgon2(password, m.config.Salt), nil
}

// IsConfigured returns whether encryption is configured.
func (m *EncryptionKeyManager) IsConfigured() bool {
	return m.config != nil && m.config.Enabled
}

// GetConfig returns the current configuration.
func (m *EncryptionKeyManager) GetConfig() *ServerEncryptionConfig {
	return m.config
}

// LoadConfig loads configuration from JSON.
func (m *EncryptionKeyManager) LoadConfig(data []byte) error {
	var config ServerEncryptionConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}
	m.config = &config
	return nil
}

// SaveConfig returns the configuration as JSON.
func (m *EncryptionKeyManager) SaveConfig() ([]byte, error) {
	if m.config == nil {
		return nil, fmt.Errorf("no configuration to save")
	}
	return json.Marshal(m.config)
}
