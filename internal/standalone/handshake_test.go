package standalone

import (
	"strings"
	"testing"
)

func TestGenerateMachineInfo(t *testing.T) {
	info := GenerateMachineInfo("1.0.0")

	if info.ClonrVersion != "1.0.0" {
		t.Errorf("ClonrVersion = %s, want 1.0.0", info.ClonrVersion)
	}

	if info.OS == "" {
		t.Error("OS is empty")
	}

	if info.Arch == "" {
		t.Error("Arch is empty")
	}

	if info.NumCPU <= 0 {
		t.Errorf("NumCPU = %d, want > 0", info.NumCPU)
	}

	if info.GoVersion == "" {
		t.Error("GoVersion is empty")
	}
}

func TestGenerateClientID(t *testing.T) {
	id1 := GenerateClientID()
	id2 := GenerateClientID()

	if id1 == "" {
		t.Error("GenerateClientID() returned empty string")
	}

	if id1 == id2 {
		t.Error("GenerateClientID() returned same ID twice")
	}

	// Should be a valid UUID format
	if len(id1) != 36 {
		t.Errorf("ClientID length = %d, want 36 (UUID format)", len(id1))
	}
}

func TestGenerateChallengeToken(t *testing.T) {
	token1, err := GenerateChallengeToken()
	if err != nil {
		t.Fatalf("GenerateChallengeToken() error = %v", err)
	}

	token2, err := GenerateChallengeToken()
	if err != nil {
		t.Fatalf("GenerateChallengeToken() error = %v", err)
	}

	if token1 == "" {
		t.Error("GenerateChallengeToken() returned empty string")
	}

	if token1 == token2 {
		t.Error("GenerateChallengeToken() returned same token twice")
	}

	// Should be 64 hex characters (32 bytes)
	if len(token1) != 64 {
		t.Errorf("Token length = %d, want 64", len(token1))
	}
}

func TestGenerateClientKey(t *testing.T) {
	fullKey, displayKey, err := GenerateClientKey()
	if err != nil {
		t.Fatalf("GenerateClientKey() error = %v", err)
	}

	if len(fullKey) != ClientKeySize {
		t.Errorf("Full key length = %d, want %d", len(fullKey), ClientKeySize)
	}

	if len(displayKey) != DisplayKeySize*2 {
		t.Errorf("Display key length = %d, want %d", len(displayKey), DisplayKeySize*2)
	}

	// Verify the full key can be derived from display key
	derived := DeriveClientKey(displayKey)
	if len(derived) != ClientKeySize {
		t.Errorf("Derived key length = %d, want %d", len(derived), ClientKeySize)
	}

	// Generate another key pair to ensure uniqueness
	fullKey2, displayKey2, err := GenerateClientKey()
	if err != nil {
		t.Fatalf("GenerateClientKey() second call error = %v", err)
	}

	if displayKey == displayKey2 {
		t.Error("GenerateClientKey() returned same display key twice")
	}

	if string(fullKey) == string(fullKey2) {
		t.Error("GenerateClientKey() returned same full key twice")
	}
}

func TestDeriveClientKey(t *testing.T) {
	displayKey := "0123456789abcdef0123456789abcdef"
	key1 := DeriveClientKey(displayKey)
	key2 := DeriveClientKey(displayKey)

	// Same display key should derive to same full key
	if string(key1) != string(key2) {
		t.Error("DeriveClientKey() returned different keys for same input")
	}

	if len(key1) != ClientKeySize {
		t.Errorf("Key length = %d, want %d", len(key1), ClientKeySize)
	}

	// Different display key should derive to different full key
	key3 := DeriveClientKey("fedcba9876543210fedcba9876543210")
	if string(key1) == string(key3) {
		t.Error("DeriveClientKey() returned same key for different inputs")
	}
}

func TestFormatDisplayKey(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"abcd", "abcd"},
		{"abcdefgh", "abcd-efgh"},
		{"0123456789abcdef", "0123-4567-89ab-cdef"},
		{"0123456789abcdef0123456789abcdef", "0123-4567-89ab-cdef-0123-4567-89ab-cdef"},
		{"ab", "ab"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := FormatDisplayKey(tt.input)
			if result != tt.expected {
				t.Errorf("FormatDisplayKey(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseDisplayKey(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"abcd-efgh", "abcdefgh"},
		{"0123-4567-89ab-cdef", "0123456789abcdef"},
		{"abcd efgh", "abcdefgh"},
		{"abcd - efgh", "abcdefgh"},
		{"abcdefgh", "abcdefgh"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ParseDisplayKey(tt.input)
			if result != tt.expected {
				t.Errorf("ParseDisplayKey(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestHandshakeFlow(t *testing.T) {
	// Create client handshake
	machineInfo := GenerateMachineInfo("1.0.0")
	clientHandshake := NewHandshake("test-client", machineInfo)

	reg := clientHandshake.GetRegistration()
	if reg.ClientID == "" {
		t.Error("ClientID is empty")
	}

	if reg.ClientName != "test-client" {
		t.Errorf("ClientName = %s, want test-client", reg.ClientName)
	}

	if reg.State != HandshakeStateInitiated {
		t.Errorf("State = %s, want %s", reg.State, HandshakeStateInitiated)
	}

	// Set challenge
	clientHandshake.SetChallenge("test-challenge-token")

	if reg.State != HandshakeStateChallenged {
		t.Errorf("State after challenge = %s, want %s", reg.State, HandshakeStateChallenged)
	}

	if reg.ChallengeToken != "test-challenge-token" {
		t.Errorf("ChallengeToken = %s, want test-challenge-token", reg.ChallengeToken)
	}

	// Generate key
	displayKey, err := clientHandshake.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	if displayKey == "" {
		t.Error("DisplayKey is empty")
	}

	if reg.State != HandshakeStateKeyGenerated {
		t.Errorf("State after key generation = %s, want %s", reg.State, HandshakeStateKeyGenerated)
	}

	// Verify display key is formatted
	if !strings.Contains(displayKey, "-") {
		t.Error("DisplayKey should be formatted with dashes")
	}

	// Get full key
	fullKey := clientHandshake.GetFullKey()
	if len(fullKey) != ClientKeySize {
		t.Errorf("Full key length = %d, want %d", len(fullKey), ClientKeySize)
	}

	// Complete handshake
	clientHandshake.Complete()

	if reg.State != HandshakeStateCompleted {
		t.Errorf("State after completion = %s, want %s", reg.State, HandshakeStateCompleted)
	}

	if reg.CompletedAt.IsZero() {
		t.Error("CompletedAt is zero")
	}
}

func TestServerHandshakeFlow(t *testing.T) {
	serverHandshake := NewServerHandshake()

	// Create client registration
	machineInfo := GenerateMachineInfo("1.0.0")
	clientReg := &ClientRegistration{
		ClientID:    GenerateClientID(),
		ClientName:  "test-client",
		MachineInfo: machineInfo,
		State:       HandshakeStateInitiated,
	}

	// Initiate handshake
	challenge, err := serverHandshake.InitiateHandshake(clientReg)
	if err != nil {
		t.Fatalf("InitiateHandshake() error = %v", err)
	}

	if challenge == "" {
		t.Error("Challenge is empty")
	}

	if len(challenge) != 64 {
		t.Errorf("Challenge length = %d, want 64", len(challenge))
	}

	// Get pending client
	pending := serverHandshake.GetPendingClient(clientReg.ClientID)
	if pending == nil {
		t.Fatal("GetPendingClient() returned nil")
	}

	if pending.State != HandshakeStateChallenged {
		t.Errorf("Pending state = %s, want %s", pending.State, HandshakeStateChallenged)
	}

	// List pending clients
	pendingList := serverHandshake.ListPendingClients()
	if len(pendingList) != 1 {
		t.Errorf("ListPendingClients() count = %d, want 1", len(pendingList))
	}

	// Generate a display key (simulate what client would do)
	_, displayKey, _ := GenerateClientKey()

	// Register client with key
	registered, err := serverHandshake.RegisterClient(clientReg.ClientID, displayKey)
	if err != nil {
		t.Fatalf("RegisterClient() error = %v", err)
	}

	if registered.ClientID != clientReg.ClientID {
		t.Errorf("RegisteredClient.ClientID = %s, want %s", registered.ClientID, clientReg.ClientID)
	}

	if registered.ClientName != "test-client" {
		t.Errorf("RegisteredClient.ClientName = %s, want test-client", registered.ClientName)
	}

	if registered.Status != "active" {
		t.Errorf("RegisteredClient.Status = %s, want active", registered.Status)
	}

	if registered.KeyHint == "" {
		t.Error("RegisteredClient.KeyHint is empty")
	}

	if len(registered.EncryptionKeyHash) == 0 {
		t.Error("RegisteredClient.EncryptionKeyHash is empty")
	}

	if len(registered.EncryptionSalt) == 0 {
		t.Error("RegisteredClient.EncryptionSalt is empty")
	}

	// Client should be removed from pending
	pending = serverHandshake.GetPendingClient(clientReg.ClientID)
	if pending != nil {
		t.Error("Client should be removed from pending after registration")
	}
}

func TestServerHandshakeInvalidKey(t *testing.T) {
	serverHandshake := NewServerHandshake()

	clientReg := &ClientRegistration{
		ClientID:   GenerateClientID(),
		ClientName: "test-client",
		State:      HandshakeStateInitiated,
	}

	_, _ = serverHandshake.InitiateHandshake(clientReg)

	// Try to register with invalid key length
	_, err := serverHandshake.RegisterClient(clientReg.ClientID, "short")
	if err == nil {
		t.Error("RegisterClient() should fail with short key")
	}
}

func TestServerHandshakeNonExistentClient(t *testing.T) {
	serverHandshake := NewServerHandshake()

	_, err := serverHandshake.RegisterClient("nonexistent", "0123456789abcdef0123456789abcdef")
	if err == nil {
		t.Error("RegisterClient() should fail for nonexistent client")
	}
}

func TestVerifyClientKey(t *testing.T) {
	// Create a registered client
	_, displayKey, _ := GenerateClientKey()
	fullKey := DeriveClientKey(displayKey)

	salt, _ := GenerateSalt()
	keyHash := HashPassword(displayKey, salt)

	client := &RegisteredClient{
		EncryptionKeyHash: keyHash,
		EncryptionSalt:    salt,
		KeyHint:           ComputeKeyHint(fullKey),
	}

	// Verify with correct key
	if !VerifyClientKey(client, displayKey) {
		t.Error("VerifyClientKey() returned false for correct key")
	}

	// Verify with formatted key
	formattedKey := FormatDisplayKey(displayKey)
	if !VerifyClientKey(client, formattedKey) {
		t.Error("VerifyClientKey() returned false for formatted correct key")
	}

	// Verify with wrong key
	if VerifyClientKey(client, "wrongkeywrongkeywrongkeywrongkey") {
		t.Error("VerifyClientKey() returned true for wrong key")
	}
}

func TestEncryptDecryptForClient(t *testing.T) {
	_, displayKey, _ := GenerateClientKey()

	plaintext := []byte("sensitive data for this client")

	// Encrypt for client
	encrypted, err := EncryptForClient(plaintext, displayKey)
	if err != nil {
		t.Fatalf("EncryptForClient() error = %v", err)
	}

	if len(encrypted.EncryptedData) == 0 {
		t.Error("EncryptedData is empty")
	}

	// Decrypt for client
	decrypted, err := DecryptForClient(encrypted, displayKey)
	if err != nil {
		t.Fatalf("DecryptForClient() error = %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("DecryptForClient() = %s, want %s", decrypted, plaintext)
	}

	// Try with wrong key
	_, err = DecryptForClient(encrypted, "wrongkeywrongkeywrongkeywrongkey")
	if err == nil {
		t.Error("DecryptForClient() should fail with wrong key")
	}
}

func TestClassifyData(t *testing.T) {
	tests := []struct {
		dataType       string
		classification DataClassification
	}{
		{"profile", ClassificationSensitive},
		{"token", ClassificationSensitive},
		{"credential", ClassificationSensitive},
		{"secret", ClassificationSensitive},
		{"repo", ClassificationPublic},
		{"repository", ClassificationPublic},
		{"workspace", ClassificationPublic},
		{"config", ClassificationPublic},
		{"unknown", ClassificationPublic},
		{"", ClassificationPublic},
	}

	for _, tt := range tests {
		t.Run(tt.dataType, func(t *testing.T) {
			result := ClassifyData(tt.dataType)
			if result != tt.classification {
				t.Errorf("ClassifyData(%s) = %s, want %s", tt.dataType, result, tt.classification)
			}
		})
	}
}

func TestHandshakeStates(t *testing.T) {
	// Verify state constants
	if HandshakeStateInitiated != "initiated" {
		t.Errorf("HandshakeStateInitiated = %s, want initiated", HandshakeStateInitiated)
	}

	if HandshakeStateChallenged != "challenged" {
		t.Errorf("HandshakeStateChallenged = %s, want challenged", HandshakeStateChallenged)
	}

	if HandshakeStateKeyGenerated != "key_generated" {
		t.Errorf("HandshakeStateKeyGenerated = %s, want key_generated", HandshakeStateKeyGenerated)
	}

	if HandshakeStateKeyPending != "key_pending" {
		t.Errorf("HandshakeStateKeyPending = %s, want key_pending", HandshakeStateKeyPending)
	}

	if HandshakeStateCompleted != "completed" {
		t.Errorf("HandshakeStateCompleted = %s, want completed", HandshakeStateCompleted)
	}

	if HandshakeStateRejected != "rejected" {
		t.Errorf("HandshakeStateRejected = %s, want rejected", HandshakeStateRejected)
	}
}

func TestRemovePending(t *testing.T) {
	serverHandshake := NewServerHandshake()

	clientReg := &ClientRegistration{
		ClientID:   GenerateClientID(),
		ClientName: "test-client",
		State:      HandshakeStateInitiated,
	}

	_, _ = serverHandshake.InitiateHandshake(clientReg)

	// Verify client is pending
	if serverHandshake.GetPendingClient(clientReg.ClientID) == nil {
		t.Fatal("Client should be pending")
	}

	// Remove pending
	serverHandshake.RemovePending(clientReg.ClientID)

	// Verify client is removed
	if serverHandshake.GetPendingClient(clientReg.ClientID) != nil {
		t.Error("Client should be removed from pending")
	}
}
