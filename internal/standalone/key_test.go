package standalone

import (
	"encoding/json"
	"testing"
	"time"
)

func TestGenerateStandaloneKey(t *testing.T) {
	host := "192.168.1.100"
	port := 50052

	key, config, err := GenerateStandaloneKey(host, port)
	if err != nil {
		t.Fatalf("GenerateStandaloneKey() error = %v", err)
	}

	// Verify key fields
	t.Run("key fields", func(t *testing.T) {
		if key.Version != KeyVersion {
			t.Errorf("key.Version = %d, want %d", key.Version, KeyVersion)
		}
		if key.InstanceID == "" {
			t.Error("key.InstanceID is empty")
		}
		if key.Host != host {
			t.Errorf("key.Host = %s, want %s", key.Host, host)
		}
		if key.Port != port {
			t.Errorf("key.Port = %d, want %d", key.Port, port)
		}
		if key.APIKey == "" {
			t.Error("key.APIKey is empty")
		}
		if key.RefreshToken == "" {
			t.Error("key.RefreshToken is empty")
		}
		if key.EncryptionKeyHint == "" {
			t.Error("key.EncryptionKeyHint is empty")
		}
		if key.CreatedAt.IsZero() {
			t.Error("key.CreatedAt is zero")
		}
		if key.ExpiresAt.IsZero() {
			t.Error("key.ExpiresAt is zero")
		}
		if len(key.Capabilities) == 0 {
			t.Error("key.Capabilities is empty")
		}
	})

	// Verify config fields
	t.Run("config fields", func(t *testing.T) {
		if !config.Enabled {
			t.Error("config.Enabled is false")
		}
		if config.InstanceID != key.InstanceID {
			t.Error("config.InstanceID doesn't match key")
		}
		if config.Port != port {
			t.Errorf("config.Port = %d, want %d", config.Port, port)
		}
		if len(config.APIKeyHash) == 0 {
			t.Error("config.APIKeyHash is empty")
		}
		if len(config.RefreshToken) == 0 {
			t.Error("config.RefreshToken is empty")
		}
		if len(config.Salt) == 0 {
			t.Error("config.Salt is empty")
		}
	})

	// Verify expiration
	t.Run("expiration", func(t *testing.T) {
		expectedExpiry := key.CreatedAt.AddDate(0, 0, DefaultExpirationDays)
		if key.ExpiresAt.Sub(expectedExpiry).Abs() > time.Second {
			t.Errorf("key.ExpiresAt = %v, expected ~%v", key.ExpiresAt, expectedExpiry)
		}
	})
}

func TestSerializeAndParseKey(t *testing.T) {
	key := &StandaloneKey{
		Version:           KeyVersion,
		InstanceID:        "test-instance",
		Host:              "localhost",
		Port:              50052,
		APIKey:            "test-api-key",
		RefreshToken:      "test-refresh-token",
		EncryptionKeyHint: "abcd",
		ExpiresAt:         time.Now().Add(24 * time.Hour),
		CreatedAt:         time.Now(),
		Capabilities:      DefaultCapabilities(),
	}

	data, err := SerializeKey(key)
	if err != nil {
		t.Fatalf("SerializeKey() error = %v", err)
	}

	parsed, err := ParseKey(data)
	if err != nil {
		t.Fatalf("ParseKey() error = %v", err)
	}

	if parsed.InstanceID != key.InstanceID {
		t.Errorf("parsed.InstanceID = %s, want %s", parsed.InstanceID, key.InstanceID)
	}
	if parsed.Host != key.Host {
		t.Errorf("parsed.Host = %s, want %s", parsed.Host, key.Host)
	}
	if parsed.Port != key.Port {
		t.Errorf("parsed.Port = %d, want %d", parsed.Port, key.Port)
	}
}

func TestEncodeAndDecodeSharedKey(t *testing.T) {
	key := &StandaloneKey{
		Version:           KeyVersion,
		InstanceID:        "test-instance",
		Host:              "localhost",
		Port:              50052,
		APIKey:            "test-api-key",
		RefreshToken:      "test-refresh-token",
		EncryptionKeyHint: "abcd",
		ExpiresAt:         time.Now().Add(24 * time.Hour),
		CreatedAt:         time.Now(),
		Capabilities:      DefaultCapabilities(),
	}

	encoded, err := EncodeKeyForSharing(key)
	if err != nil {
		t.Fatalf("EncodeKeyForSharing() error = %v", err)
	}

	// Should have magic prefix
	if len(encoded) <= len(StandaloneKeyMagic)+1 {
		t.Error("encoded key too short")
	}
	if encoded[:len(StandaloneKeyMagic)+1] != StandaloneKeyMagic+":" {
		t.Errorf("encoded key missing magic prefix, got %s", encoded[:len(StandaloneKeyMagic)+1])
	}

	decoded, err := DecodeSharedKey(encoded)
	if err != nil {
		t.Fatalf("DecodeSharedKey() error = %v", err)
	}

	if decoded.InstanceID != key.InstanceID {
		t.Errorf("decoded.InstanceID = %s, want %s", decoded.InstanceID, key.InstanceID)
	}
}

func TestDecodeSharedKeyFromJSON(t *testing.T) {
	key := &StandaloneKey{
		Version:      KeyVersion,
		InstanceID:   "test-instance",
		Host:         "localhost",
		Port:         50052,
		APIKey:       "test-api-key",
		RefreshToken: "test-refresh-token",
	}

	// Test decoding raw JSON
	jsonData, _ := json.Marshal(key)
	decoded, err := DecodeSharedKey(string(jsonData))
	if err != nil {
		t.Fatalf("DecodeSharedKey(json) error = %v", err)
	}

	if decoded.InstanceID != key.InstanceID {
		t.Errorf("decoded.InstanceID = %s, want %s", decoded.InstanceID, key.InstanceID)
	}
}

func TestValidateKey(t *testing.T) {
	tests := []struct {
		name    string
		key     *StandaloneKey
		wantErr bool
	}{
		{
			name: "valid key",
			key: &StandaloneKey{
				Version:      KeyVersion,
				InstanceID:   "test",
				Host:         "localhost",
				Port:         50052,
				APIKey:       "key",
				RefreshToken: "token",
				ExpiresAt:    time.Now().Add(time.Hour),
			},
			wantErr: false,
		},
		{
			name: "missing instance ID",
			key: &StandaloneKey{
				Version: KeyVersion,
				Host:    "localhost",
				Port:    50052,
				APIKey:  "key",
			},
			wantErr: true,
		},
		{
			name: "missing API key",
			key: &StandaloneKey{
				Version:    KeyVersion,
				InstanceID: "test",
				Host:       "localhost",
				Port:       50052,
			},
			wantErr: true,
		},
		{
			name: "missing host",
			key: &StandaloneKey{
				Version:    KeyVersion,
				InstanceID: "test",
				Port:       50052,
				APIKey:     "key",
			},
			wantErr: true,
		},
		{
			name: "invalid port",
			key: &StandaloneKey{
				Version:    KeyVersion,
				InstanceID: "test",
				Host:       "localhost",
				Port:       0,
				APIKey:     "key",
			},
			wantErr: true,
		},
		{
			name: "expired key",
			key: &StandaloneKey{
				Version:    KeyVersion,
				InstanceID: "test",
				Host:       "localhost",
				Port:       50052,
				APIKey:     "key",
				ExpiresAt:  time.Now().Add(-time.Hour),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateKey(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateKey() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestKeyIsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{"future", time.Now().Add(time.Hour), false},
		{"past", time.Now().Add(-time.Hour), true},
		{"zero (no expiry)", time.Time{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := &StandaloneKey{ExpiresAt: tt.expiresAt}
			if got := key.IsExpired(); got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestKeyHasCapability(t *testing.T) {
	key := &StandaloneKey{
		Capabilities: []string{CapabilityProfiles, CapabilityWorkspaces},
	}

	if !key.HasCapability(CapabilityProfiles) {
		t.Error("HasCapability(profiles) = false, want true")
	}
	if !key.HasCapability(CapabilityWorkspaces) {
		t.Error("HasCapability(workspaces) = false, want true")
	}
	if key.HasCapability(CapabilityRepos) {
		t.Error("HasCapability(repos) = true, want false")
	}
}

func TestCreateConnection(t *testing.T) {
	key := &StandaloneKey{
		Version:      KeyVersion,
		InstanceID:   "test-instance",
		Host:         "localhost",
		Port:         50052,
		APIKey:       "dGVzdC1hcGkta2V5", // base58 encoded
		RefreshToken: "dGVzdC1yZWZyZXNo", // base58 encoded
		ExpiresAt:    time.Now().Add(24 * time.Hour),
	}

	conn, err := CreateConnection("home-server", key, "local_password")
	if err != nil {
		t.Fatalf("CreateConnection() error = %v", err)
	}

	if conn.Name != "home-server" {
		t.Errorf("conn.Name = %s, want home-server", conn.Name)
	}
	if conn.InstanceID != key.InstanceID {
		t.Errorf("conn.InstanceID = %s, want %s", conn.InstanceID, key.InstanceID)
	}
	if conn.Host != key.Host {
		t.Errorf("conn.Host = %s, want %s", conn.Host, key.Host)
	}
	if len(conn.APIKeyEncrypted) == 0 {
		t.Error("conn.APIKeyEncrypted is empty")
	}
	if len(conn.LocalPasswordHash) == 0 {
		t.Error("conn.LocalPasswordHash is empty")
	}
	if len(conn.LocalSalt) == 0 {
		t.Error("conn.LocalSalt is empty")
	}
	if conn.SyncStatus != StatusDisconnected {
		t.Errorf("conn.SyncStatus = %s, want %s", conn.SyncStatus, StatusDisconnected)
	}
}

func TestDecryptConnection(t *testing.T) {
	key := &StandaloneKey{
		Version:      KeyVersion,
		InstanceID:   "test-instance",
		Host:         "localhost",
		Port:         50052,
		APIKey:       "dGVzdC1hcGkta2V5",
		RefreshToken: "dGVzdC1yZWZyZXNo",
		ExpiresAt:    time.Now().Add(24 * time.Hour),
	}

	password := "local_password"
	conn, _ := CreateConnection("test", key, password)

	t.Run("correct password", func(t *testing.T) {
		apiKey, refreshToken, err := DecryptConnection(conn, password)
		if err != nil {
			t.Fatalf("DecryptConnection() error = %v", err)
		}
		if len(apiKey) == 0 {
			t.Error("apiKey is empty")
		}
		if len(refreshToken) == 0 {
			t.Error("refreshToken is empty")
		}
	})

	t.Run("wrong password", func(t *testing.T) {
		_, _, err := DecryptConnection(conn, "wrong_password")
		if err == nil {
			t.Error("DecryptConnection() expected error with wrong password")
		}
	})
}

func TestGetLocalIP(t *testing.T) {
	ip, err := GetLocalIP()
	if err != nil {
		t.Fatalf("GetLocalIP() error = %v", err)
	}

	if ip == "" {
		t.Error("GetLocalIP() returned empty string")
	}

	// Should be a valid IPv4 address
	t.Logf("Detected local IP: %s", ip)
}
