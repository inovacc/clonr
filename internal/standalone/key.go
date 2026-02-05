package standalone

import (
	"encoding/json"
	"fmt"
	"net"
	"slices"
	"time"

	"github.com/btcsuite/btcutil/base58"
	"github.com/google/uuid"
)

// Magic header for standalone key format
const StandaloneKeyMagic = "CLONR-SYNC"

// GenerateStandaloneKey generates a new standalone key for sync.
// This is called when initializing standalone mode on the source instance.
func GenerateStandaloneKey(host string, port int) (*StandaloneKey, *StandaloneConfig, error) {
	// Generate instance ID
	instanceID := uuid.New().String()

	// Generate master key (32 bytes)
	masterKey, err := GenerateRandomBytes(keySize)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate master key: %w", err)
	}
	defer SecureZero(masterKey)

	// Derive API key from master key
	apiKey, err := DeriveAPIKey(masterKey, instanceID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to derive API key: %w", err)
	}

	// Derive encryption key (for data encryption)
	encKey, err := DeriveEncryptionKey(masterKey, instanceID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to derive encryption key: %w", err)
	}

	// Generate refresh token
	refreshTokenBytes, err := GenerateRandomBytes(keySize)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Generate salt for config storage
	salt, err := GenerateSalt()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	// Hash API key for storage (we don't store the actual key on server)
	apiKeyHash := HashPassword(string(apiKey), salt)

	// Encrypt refresh token for storage
	refreshTokenEncrypted, err := EncryptWithKey(refreshTokenBytes, encKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to encrypt refresh token: %w", err)
	}

	now := time.Now()
	expiresAt := now.AddDate(0, 0, DefaultExpirationDays)

	// Create the key to share with destination instances
	standaloneKey := &StandaloneKey{
		Version:           KeyVersion,
		InstanceID:        instanceID,
		Host:              host,
		Port:              port,
		APIKey:            base58.Encode(apiKey),
		RefreshToken:      base58.Encode(refreshTokenBytes),
		EncryptionKeyHint: ComputeKeyHint(encKey),
		ExpiresAt:         expiresAt,
		CreatedAt:         now,
		Capabilities:      DefaultCapabilities(),
	}

	// Create the config to store locally on source instance
	config := &StandaloneConfig{
		Enabled:      true,
		IsServer:     true, // This instance is a server (accepts client connections)
		InstanceID:   instanceID,
		Port:         port,
		APIKeyHash:   apiKeyHash,
		RefreshToken: refreshTokenEncrypted,
		Salt:         salt,
		CreatedAt:    now,
		ExpiresAt:    expiresAt,
		Capabilities: DefaultCapabilities(),
	}

	return standaloneKey, config, nil
}

// SerializeKey serializes a StandaloneKey to JSON.
func SerializeKey(key *StandaloneKey) ([]byte, error) {
	return json.MarshalIndent(key, "", "  ")
}

// ParseKey parses a StandaloneKey from JSON.
func ParseKey(data []byte) (*StandaloneKey, error) {
	var key StandaloneKey
	if err := json.Unmarshal(data, &key); err != nil {
		return nil, fmt.Errorf("failed to parse standalone key: %w", err)
	}

	// Validate required fields
	if key.InstanceID == "" {
		return nil, fmt.Errorf("invalid key: missing instance_id")
	}

	if key.APIKey == "" {
		return nil, fmt.Errorf("invalid key: missing api_key")
	}

	if key.Host == "" {
		return nil, fmt.Errorf("invalid key: missing host")
	}

	if key.Port == 0 {
		return nil, fmt.Errorf("invalid key: missing port")
	}

	return &key, nil
}

// EncodeKeyForSharing encodes a key with magic header for easy sharing.
func EncodeKeyForSharing(key *StandaloneKey) (string, error) {
	data, err := SerializeKey(key)
	if err != nil {
		return "", err
	}

	encoded := base58.Encode(data)

	return fmt.Sprintf("%s:%s", StandaloneKeyMagic, encoded), nil
}

// DecodeSharedKey decodes a key from the sharing format.
func DecodeSharedKey(encoded string) (*StandaloneKey, error) {
	// Check for magic header
	prefix := StandaloneKeyMagic + ":"
	if len(encoded) <= len(prefix) {
		return nil, fmt.Errorf("invalid key format: too short")
	}

	if encoded[:len(prefix)] != prefix {
		// Try parsing as raw JSON (for file-based import)
		return ParseKey([]byte(encoded))
	}

	// Decode base58
	data := base58.Decode(encoded[len(prefix):])
	if len(data) == 0 {
		return nil, fmt.Errorf("invalid key format: base58 decode failed")
	}

	return ParseKey(data)
}

// ValidateKey validates a standalone key.
func ValidateKey(key *StandaloneKey) error {
	if key.Version > KeyVersion {
		return fmt.Errorf("key version %d is newer than supported version %d", key.Version, KeyVersion)
	}

	if key.InstanceID == "" {
		return fmt.Errorf("missing instance_id")
	}

	if key.APIKey == "" {
		return fmt.Errorf("missing api_key")
	}

	if key.Host == "" {
		return fmt.Errorf("missing host")
	}

	if key.Port <= 0 || key.Port > 65535 {
		return fmt.Errorf("invalid port: %d", key.Port)
	}

	if !key.ExpiresAt.IsZero() && time.Now().After(key.ExpiresAt) {
		return fmt.Errorf("key has expired")
	}

	return nil
}

// IsExpired checks if a key has expired.
func (k *StandaloneKey) IsExpired() bool {
	return !k.ExpiresAt.IsZero() && time.Now().After(k.ExpiresAt)
}

// HasCapability checks if the key has a specific capability.
func (k *StandaloneKey) HasCapability(cap string) bool {

	return slices.Contains(k.Capabilities, cap)
}

// GetLocalIP attempts to determine the local IP address.
func GetLocalIP() (string, error) {
	// Try to connect to a public IP to determine local interface
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		// Fallback to interface enumeration
		return getLocalIPFromInterfaces()
	}

	defer func() { _ = conn.Close() }()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP.String(), nil
}

// getLocalIPFromInterfaces enumerates network interfaces to find a local IP.
func getLocalIPFromInterfaces() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("failed to get network interfaces: %w", err)
	}

	for _, iface := range interfaces {
		// Skip loopback and down interfaces
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP

			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			// Skip IPv6 and loopback
			if ip == nil || ip.IsLoopback() || ip.To4() == nil {
				continue
			}

			return ip.String(), nil
		}
	}

	return "127.0.0.1", nil // Fallback to localhost
}

// RotateKey generates a new API key while keeping the same instance ID.
// This invalidates all existing connections.
func RotateKey(config *StandaloneConfig) (*StandaloneKey, *StandaloneConfig, error) {
	// Get current host (we need to rediscover it)
	host, err := GetLocalIP()
	if err != nil {
		host = "127.0.0.1"
	}

	// Generate new key but keep instance ID
	newKey, newConfig, err := GenerateStandaloneKey(host, config.Port)
	if err != nil {
		return nil, nil, err
	}

	// Preserve instance ID from original config
	newKey.InstanceID = config.InstanceID
	newConfig.InstanceID = config.InstanceID

	return newKey, newConfig, nil
}

// CreateConnection creates a StandaloneConnection from a key and local password.
func CreateConnection(name string, key *StandaloneKey, localPassword string) (*StandaloneConnection, error) {
	if err := ValidateKey(key); err != nil {
		return nil, fmt.Errorf("invalid key: %w", err)
	}

	// Generate salt for local encryption
	localSalt, err := GenerateSalt()
	if err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	// Derive local encryption key from password
	localKey := DeriveKeyArgon2(localPassword, localSalt)

	// Hash password for verification
	passwordHash := HashPassword(localPassword, localSalt)

	// Decode and encrypt API key locally
	apiKeyBytes := base58.Decode(key.APIKey)

	apiKeyEncrypted, err := EncryptWithKey(apiKeyBytes, localKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt API key: %w", err)
	}

	// Decode and encrypt refresh token locally
	refreshTokenBytes := base58.Decode(key.RefreshToken)

	refreshTokenEncrypted, err := EncryptWithKey(refreshTokenBytes, localKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt refresh token: %w", err)
	}

	now := time.Now()

	return &StandaloneConnection{
		Name:                  name,
		InstanceID:            key.InstanceID,
		Host:                  key.Host,
		Port:                  key.Port,
		APIKeyEncrypted:       apiKeyEncrypted,
		RefreshTokenEncrypted: refreshTokenEncrypted,
		LocalPasswordHash:     passwordHash,
		LocalSalt:             localSalt,
		SyncStatus:            StatusDisconnected,
		CreatedAt:             now,
		UpdatedAt:             now,
	}, nil
}

// DecryptConnection decrypts the connection credentials using the local password.
func DecryptConnection(conn *StandaloneConnection, localPassword string) (apiKey, refreshToken []byte, err error) {
	// Verify password
	if !VerifyPassword(localPassword, conn.LocalSalt, conn.LocalPasswordHash) {
		return nil, nil, fmt.Errorf("invalid password")
	}

	// Derive local key
	localKey := DeriveKeyArgon2(localPassword, conn.LocalSalt)

	// Decrypt API key
	apiKey, err = DecryptWithKey(conn.APIKeyEncrypted, localKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decrypt API key: %w", err)
	}

	// Decrypt refresh token
	refreshToken, err = DecryptWithKey(conn.RefreshTokenEncrypted, localKey)
	if err != nil {
		SecureZero(apiKey)
		return nil, nil, fmt.Errorf("failed to decrypt refresh token: %w", err)
	}

	return apiKey, refreshToken, nil
}
