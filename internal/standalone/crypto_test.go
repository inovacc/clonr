package standalone

import (
	"bytes"
	"testing"
)

func TestGenerateRandomBytes(t *testing.T) {
	tests := []struct {
		name    string
		size    int
		wantErr bool
	}{
		{"16 bytes", 16, false},
		{"32 bytes", 32, false},
		{"64 bytes", 64, false},
		{"zero bytes", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateRandomBytes(tt.size)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateRandomBytes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(got) != tt.size {
				t.Errorf("GenerateRandomBytes() got length = %d, want %d", len(got), tt.size)
			}
		})
	}

	// Test randomness - two calls should produce different results
	t.Run("randomness", func(t *testing.T) {
		b1, _ := GenerateRandomBytes(32)

		b2, _ := GenerateRandomBytes(32)
		if bytes.Equal(b1, b2) {
			t.Error("GenerateRandomBytes() produced identical results")
		}
	})
}

func TestEncryptDecrypt(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		password string
	}{
		{"simple text", []byte("hello world"), "password123"},
		{"empty data", []byte(""), "password123"},
		{"binary data", []byte{0x00, 0x01, 0x02, 0xff, 0xfe}, "password123"},
		{"unicode password", []byte("test data"), "密码测试"},
		{"long data", bytes.Repeat([]byte("a"), 10000), "password123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encrypted, err := Encrypt(tt.data, tt.password)
			if err != nil {
				t.Fatalf("Encrypt() error = %v", err)
			}

			// Verify encrypted data is different from original
			if bytes.Equal(encrypted, tt.data) && len(tt.data) > 0 {
				t.Error("Encrypt() produced identical output")
			}

			decrypted, err := Decrypt(encrypted, tt.password)
			if err != nil {
				t.Fatalf("Decrypt() error = %v", err)
			}

			if !bytes.Equal(decrypted, tt.data) {
				t.Errorf("Decrypt() got = %v, want %v", decrypted, tt.data)
			}
		})
	}
}

func TestDecryptWrongPassword(t *testing.T) {
	data := []byte("secret data")
	password := "correct_password"

	encrypted, err := Encrypt(data, password)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	_, err = Decrypt(encrypted, "wrong_password")
	if err == nil {
		t.Error("Decrypt() expected error with wrong password")
	}
}

func TestEncryptWithKey(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	data := []byte("test data")

	encrypted, err := EncryptWithKey(data, key)
	if err != nil {
		t.Fatalf("EncryptWithKey() error = %v", err)
	}

	decrypted, err := DecryptWithKey(encrypted, key)
	if err != nil {
		t.Fatalf("DecryptWithKey() error = %v", err)
	}

	if !bytes.Equal(decrypted, data) {
		t.Errorf("DecryptWithKey() got = %v, want %v", decrypted, data)
	}
}

func TestDeriveKeys(t *testing.T) {
	masterKey := []byte("master_key_for_testing_purposes!")
	instanceID := "test-instance-id"

	t.Run("DeriveAPIKey", func(t *testing.T) {
		key1, err := DeriveAPIKey(masterKey, instanceID)
		if err != nil {
			t.Fatalf("DeriveAPIKey() error = %v", err)
		}

		if len(key1) != keySize {
			t.Errorf("DeriveAPIKey() key size = %d, want %d", len(key1), keySize)
		}

		// Same inputs should produce same output
		key2, _ := DeriveAPIKey(masterKey, instanceID)
		if !bytes.Equal(key1, key2) {
			t.Error("DeriveAPIKey() not deterministic")
		}

		// Different instance ID should produce different key
		key3, _ := DeriveAPIKey(masterKey, "different-instance")
		if bytes.Equal(key1, key3) {
			t.Error("DeriveAPIKey() same output for different instance IDs")
		}
	})

	t.Run("DeriveEncryptionKey", func(t *testing.T) {
		key, err := DeriveEncryptionKey(masterKey, instanceID)
		if err != nil {
			t.Fatalf("DeriveEncryptionKey() error = %v", err)
		}

		if len(key) != keySize {
			t.Errorf("DeriveEncryptionKey() key size = %d, want %d", len(key), keySize)
		}

		// API key and encryption key should be different
		apiKey, _ := DeriveAPIKey(masterKey, instanceID)
		if bytes.Equal(key, apiKey) {
			t.Error("DeriveEncryptionKey() same as API key")
		}
	})
}

func TestPasswordHashing(t *testing.T) {
	password := "test_password"
	salt, _ := GenerateSalt()

	hash := HashPassword(password, salt)

	t.Run("verify correct password", func(t *testing.T) {
		if !VerifyPassword(password, salt, hash) {
			t.Error("VerifyPassword() failed for correct password")
		}
	})

	t.Run("reject wrong password", func(t *testing.T) {
		if VerifyPassword("wrong_password", salt, hash) {
			t.Error("VerifyPassword() accepted wrong password")
		}
	})

	t.Run("reject wrong salt", func(t *testing.T) {
		wrongSalt, _ := GenerateSalt()
		if VerifyPassword(password, wrongSalt, hash) {
			t.Error("VerifyPassword() accepted wrong salt")
		}
	})
}

func TestComputeKeyHint(t *testing.T) {
	key := []byte("test_key_for_hint")
	hint := ComputeKeyHint(key)

	// Hint should be 4 hex characters
	if len(hint) != 4 {
		t.Errorf("ComputeKeyHint() length = %d, want 4", len(hint))
	}

	// Same key should produce same hint
	hint2 := ComputeKeyHint(key)
	if hint != hint2 {
		t.Error("ComputeKeyHint() not deterministic")
	}

	// Different key should produce different hint (most likely)
	hint3 := ComputeKeyHint([]byte("different_key"))
	if hint == hint3 {
		t.Log("Warning: ComputeKeyHint() produced same hint for different keys (possible but unlikely)")
	}
}

func TestSecureZero(t *testing.T) {
	data := []byte("sensitive data")
	SecureZero(data)

	for i, b := range data {
		if b != 0 {
			t.Errorf("SecureZero() byte %d = %d, want 0", i, b)
		}
	}
}
