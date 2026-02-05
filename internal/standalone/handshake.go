package standalone

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/google/uuid"
)

// HandshakeState represents the state of the client-server handshake.
type HandshakeState string

const (
	HandshakeStateInitiated    HandshakeState = "initiated"     // Client sent initial request
	HandshakeStateChallenged   HandshakeState = "challenged"    // Server sent challenge
	HandshakeStateKeyGenerated HandshakeState = "key_generated" // Client generated key
	HandshakeStateKeyPending   HandshakeState = "key_pending"   // Waiting for key entry on server
	HandshakeStateCompleted    HandshakeState = "completed"     // Handshake successful
	HandshakeStateRejected     HandshakeState = "rejected"      // Handshake rejected
)

// ClientKeySize is the size of the per-client encryption key (32 bytes = 256 bits).
const ClientKeySize = 32

// DisplayKeySize is the size of the key displayed to user (16 bytes = 32 hex chars).
const DisplayKeySize = 16

// ClientRegistration contains information sent by client during handshake.
type ClientRegistration struct {
	// Identification
	ClientID   string `json:"client_id"`   // Unique client UUID
	ClientName string `json:"client_name"` // Human-readable name

	// Machine information for metrics
	MachineInfo MachineInfo `json:"machine_info"`

	// Handshake state
	State       HandshakeState `json:"state"`
	InitiatedAt time.Time      `json:"initiated_at"`
	CompletedAt time.Time      `json:"completed_at,omitempty"`

	// Challenge data
	ChallengeToken string    `json:"challenge_token,omitempty"`
	ChallengeAt    time.Time `json:"challenge_at,omitempty"`
}

// MachineInfo contains machine-related metrics for identification.
type MachineInfo struct {
	Hostname     string `json:"hostname"`
	OS           string `json:"os"`
	Arch         string `json:"arch"`
	NumCPU       int    `json:"num_cpu"`
	GoVersion    string `json:"go_version"`
	ClonrVersion string `json:"clonr_version"`
}

// RegisteredClient represents a fully registered client on the server.
type RegisteredClient struct {
	// From registration
	ClientID    string      `json:"client_id"`
	ClientName  string      `json:"client_name"`
	MachineInfo MachineInfo `json:"machine_info"`

	// Encryption
	EncryptionKeyHash []byte `json:"encryption_key_hash"` // Argon2 hash for verification
	EncryptionSalt    []byte `json:"encryption_salt"`     // Salt for key derivation
	KeyHint           string `json:"key_hint"`            // First 4 chars for identification

	// Status
	Status       string    `json:"status"` // "active", "suspended", "revoked"
	RegisteredAt time.Time `json:"registered_at"`
	LastSeenAt   time.Time `json:"last_seen_at"`
	SyncCount    int       `json:"sync_count"`

	// Connection info
	LastIP string `json:"last_ip"`
}

// ClientEncryptedData stores data that is encrypted with client-specific key.
type ClientEncryptedData struct {
	ClientID      string    `json:"client_id"`
	DataType      string    `json:"data_type"` // "profile", "config", etc.
	DataID        string    `json:"data_id"`   // Identifier within type
	EncryptedData []byte    `json:"encrypted_data"`
	Nonce         []byte    `json:"nonce"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// GenerateMachineInfo collects current machine information.
func GenerateMachineInfo(clonrVersion string) MachineInfo {
	hostname, _ := os.Hostname()

	return MachineInfo{
		Hostname:     hostname,
		OS:           runtime.GOOS,
		Arch:         runtime.GOARCH,
		NumCPU:       runtime.NumCPU(),
		GoVersion:    runtime.Version(),
		ClonrVersion: clonrVersion,
	}
}

// GenerateClientID generates a unique client identifier.
func GenerateClientID() string {
	return uuid.New().String()
}

// GenerateChallengeToken generates a random challenge token.
func GenerateChallengeToken() (string, error) {
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		return "", fmt.Errorf("failed to generate challenge token: %w", err)
	}

	return hex.EncodeToString(token), nil
}

// GenerateClientKey generates a new client encryption key and returns both
// the full key (for encryption) and display key (for user to enter on server).
func GenerateClientKey() (fullKey []byte, displayKey string, err error) {
	// Generate full key for encryption
	fullKey = make([]byte, ClientKeySize)
	if _, err := rand.Read(fullKey); err != nil {
		return nil, "", fmt.Errorf("failed to generate key: %w", err)
	}

	// Generate display key (shorter, easier to type)
	displayBytes := make([]byte, DisplayKeySize)
	if _, err := rand.Read(displayBytes); err != nil {
		return nil, "", fmt.Errorf("failed to generate display key: %w", err)
	}

	displayKey = hex.EncodeToString(displayBytes)

	// The full key is derived from display key + some fixed derivation
	// This ensures the display key is sufficient to recreate the full key
	fullKey = DeriveKeyArgon2(displayKey, []byte("clonr-client-key"))

	return fullKey, displayKey, nil
}

// DeriveClientKey derives the full encryption key from the display key.
func DeriveClientKey(displayKey string) []byte {
	return DeriveKeyArgon2(displayKey, []byte("clonr-client-key"))
}

// FormatDisplayKey formats a display key for easier reading (groups of 4).
func FormatDisplayKey(key string) string {
	if len(key) <= 4 {
		return key
	}

	var formatted string

	for i, char := range key {
		if i > 0 && i%4 == 0 {
			formatted += "-"
		}

		formatted += string(char)
	}

	return formatted
}

// ParseDisplayKey removes formatting from a display key.
func ParseDisplayKey(formatted string) string {
	var result string

	for _, char := range formatted {
		if char != '-' && char != ' ' {
			result += string(char)
		}
	}

	return result
}

// Handshake manages the client-server handshake process.
type Handshake struct {
	registration *ClientRegistration
	challenge    string
	clientKey    []byte
	displayKey   string
}

// NewHandshake creates a new handshake for a client.
func NewHandshake(clientName string, machineInfo MachineInfo) *Handshake {
	return &Handshake{
		registration: &ClientRegistration{
			ClientID:    GenerateClientID(),
			ClientName:  clientName,
			MachineInfo: machineInfo,
			State:       HandshakeStateInitiated,
			InitiatedAt: time.Now(),
		},
	}
}

// GetRegistration returns the client registration info.
func (h *Handshake) GetRegistration() *ClientRegistration {
	return h.registration
}

// SetChallenge sets the challenge received from server.
func (h *Handshake) SetChallenge(token string) {
	h.challenge = token
	h.registration.ChallengeToken = token
	h.registration.ChallengeAt = time.Now()
	h.registration.State = HandshakeStateChallenged
}

// GenerateKey generates the client encryption key.
// Returns the display key that user needs to enter on the server.
func (h *Handshake) GenerateKey() (string, error) {
	fullKey, displayKey, err := GenerateClientKey()
	if err != nil {
		return "", err
	}

	h.clientKey = fullKey
	h.displayKey = displayKey
	h.registration.State = HandshakeStateKeyGenerated

	return FormatDisplayKey(displayKey), nil
}

// GetFullKey returns the full encryption key (after GenerateKey was called).
func (h *Handshake) GetFullKey() []byte {
	return h.clientKey
}

// Complete marks the handshake as completed.
func (h *Handshake) Complete() {
	h.registration.State = HandshakeStateCompleted
	h.registration.CompletedAt = time.Now()
}

// ServerHandshake manages the server side of the handshake.
type ServerHandshake struct {
	pendingClients map[string]*ClientRegistration
}

// NewServerHandshake creates a new server handshake manager.
func NewServerHandshake() *ServerHandshake {
	return &ServerHandshake{
		pendingClients: make(map[string]*ClientRegistration),
	}
}

// InitiateHandshake processes a client's initial registration request.
// Returns a challenge token for the client.
func (s *ServerHandshake) InitiateHandshake(reg *ClientRegistration) (string, error) {
	// Generate challenge
	challenge, err := GenerateChallengeToken()
	if err != nil {
		return "", err
	}

	// Store pending registration
	reg.ChallengeToken = challenge
	reg.ChallengeAt = time.Now()
	reg.State = HandshakeStateChallenged
	s.pendingClients[reg.ClientID] = reg

	return challenge, nil
}

// GetPendingClient returns a pending client registration.
func (s *ServerHandshake) GetPendingClient(clientID string) *ClientRegistration {
	return s.pendingClients[clientID]
}

// ListPendingClients returns all pending client registrations.
func (s *ServerHandshake) ListPendingClients() []*ClientRegistration {
	var result []*ClientRegistration
	for _, reg := range s.pendingClients {
		result = append(result, reg)
	}

	return result
}

// RegisterClient completes the handshake by registering the client with their key.
func (s *ServerHandshake) RegisterClient(clientID, displayKey string) (*RegisteredClient, error) {
	reg, exists := s.pendingClients[clientID]
	if !exists {
		return nil, fmt.Errorf("no pending registration for client %s", clientID)
	}

	// Parse and validate display key
	cleanKey := ParseDisplayKey(displayKey)
	if len(cleanKey) != DisplayKeySize*2 { // hex encoded
		return nil, fmt.Errorf("invalid key length: expected %d characters", DisplayKeySize*2)
	}

	// Derive the full encryption key
	fullKey := DeriveClientKey(cleanKey)

	// Generate salt and hash for storage
	salt, err := GenerateSalt()
	if err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	keyHash := HashPassword(cleanKey, salt)

	// Create registered client
	client := &RegisteredClient{
		ClientID:          reg.ClientID,
		ClientName:        reg.ClientName,
		MachineInfo:       reg.MachineInfo,
		EncryptionKeyHash: keyHash,
		EncryptionSalt:    salt,
		KeyHint:           ComputeKeyHint(fullKey),
		Status:            "active",
		RegisteredAt:      time.Now(),
		LastSeenAt:        time.Now(),
	}

	// Remove from pending
	delete(s.pendingClients, clientID)

	return client, nil
}

// RemovePending removes a pending registration (timeout or rejection).
func (s *ServerHandshake) RemovePending(clientID string) {
	delete(s.pendingClients, clientID)
}

// VerifyClientKey verifies a client's key against stored hash.
func VerifyClientKey(client *RegisteredClient, displayKey string) bool {
	cleanKey := ParseDisplayKey(displayKey)
	return VerifyPassword(cleanKey, client.EncryptionSalt, client.EncryptionKeyHash)
}

// EncryptForClient encrypts data for a specific client using their key.
func EncryptForClient(data []byte, displayKey string) (*ClientEncryptedData, error) {
	cleanKey := ParseDisplayKey(displayKey)
	fullKey := DeriveClientKey(cleanKey)

	encrypted, err := EncryptWithKey(data, fullKey)
	if err != nil {
		return nil, fmt.Errorf("encryption failed: %w", err)
	}

	return &ClientEncryptedData{
		EncryptedData: encrypted,
		UpdatedAt:     time.Now(),
	}, nil
}

// DecryptForClient decrypts client-specific data using their key.
func DecryptForClient(encData *ClientEncryptedData, displayKey string) ([]byte, error) {
	cleanKey := ParseDisplayKey(displayKey)
	fullKey := DeriveClientKey(cleanKey)

	decrypted, err := DecryptWithKey(encData.EncryptedData, fullKey)
	if err != nil {
		return nil, fmt.Errorf("decryption failed (wrong key?): %w", err)
	}

	return decrypted, nil
}

// DataClassification indicates how data should be stored.
type DataClassification string

const (
	// ClassificationPublic - stored unencrypted (repos, non-sensitive config)
	ClassificationPublic DataClassification = "public"
	// ClassificationSensitive - stored encrypted with client key (tokens, credentials)
	ClassificationSensitive DataClassification = "sensitive"
)

// ClassifyData determines how a data type should be stored.
func ClassifyData(dataType string) DataClassification {
	switch dataType {
	case "profile", "token", "credential", "secret":
		return ClassificationSensitive
	case "repo", "repository", "workspace", "config":
		return ClassificationPublic
	default:
		return ClassificationPublic
	}
}
