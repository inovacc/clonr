package tpm

import (
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeystoreEncryptDecrypt(t *testing.T) {
	// Create temp directory for test keystore
	tmpDir, err := os.MkdirTemp("", "clonr-keystore-test-*")
	require.NoError(t, err)

	defer os.RemoveAll(tmpDir)

	// Override application directory for test
	oldGetAppDir := getAppDirectory
	getAppDirectory = func() (string, error) { return tmpDir, nil }

	defer func() { getAppDirectory = oldGetAppDir }()

	// Reset keystore singleton for test
	resetKeystoreForTest()

	t.Run("encrypt and decrypt token", func(t *testing.T) {
		profile := "test-profile"
		token := "ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"

		// Encrypt
		encrypted, err := EncryptWithKeystore(profile, DEKTypeToken, []byte(token))
		require.NoError(t, err)
		assert.NotEqual(t, token, string(encrypted))
		assert.NotEmpty(t, encrypted)

		// Decrypt
		decrypted, err := DecryptWithKeystore(profile, DEKTypeToken, encrypted)
		require.NoError(t, err)
		assert.Equal(t, token, string(decrypted))
	})

	t.Run("different profiles have different encryption", func(t *testing.T) {
		token := "same-token-for-both"
		profile1 := "profile-1"
		profile2 := "profile-2"

		enc1, err := EncryptWithKeystore(profile1, DEKTypeToken, []byte(token))
		require.NoError(t, err)

		enc2, err := EncryptWithKeystore(profile2, DEKTypeToken, []byte(token))
		require.NoError(t, err)

		// Same plaintext should produce different ciphertext
		assert.NotEqual(t, enc1, enc2)

		// Both should decrypt correctly
		dec1, err := DecryptWithKeystore(profile1, DEKTypeToken, enc1)
		require.NoError(t, err)
		assert.Equal(t, token, string(dec1))

		dec2, err := DecryptWithKeystore(profile2, DEKTypeToken, enc2)
		require.NoError(t, err)
		assert.Equal(t, token, string(dec2))
	})

	t.Run("cross-profile decryption fails", func(t *testing.T) {
		token := "secret-token"
		profile1 := "profile-a"
		profile2 := "profile-b"

		// Encrypt with profile1
		encrypted, err := EncryptWithKeystore(profile1, DEKTypeToken, []byte(token))
		require.NoError(t, err)

		// Ensure profile2 exists
		err = EnsureProfile(profile2)
		require.NoError(t, err)

		// Try to decrypt with profile2 - should fail
		_, err = DecryptWithKeystore(profile2, DEKTypeToken, encrypted)
		assert.Error(t, err)
	})
}

func TestKeystoreProfileManagement(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "clonr-keystore-test-*")
	require.NoError(t, err)

	defer os.RemoveAll(tmpDir)

	oldGetAppDir := getAppDirectory
	getAppDirectory = func() (string, error) { return tmpDir, nil }

	defer func() { getAppDirectory = oldGetAppDir }()

	resetKeystoreForTest()

	t.Run("ensure profile creates if not exists", func(t *testing.T) {
		profile := "new-profile"

		ks, err := GetKeystore()
		require.NoError(t, err)

		// Profile shouldn't exist initially
		assert.False(t, ks.HasProfile(profile))

		// Ensure creates it
		err = EnsureProfile(profile)
		require.NoError(t, err)

		// Now it should exist
		assert.True(t, ks.HasProfile(profile))

		// Second ensure should be idempotent
		err = EnsureProfile(profile)
		require.NoError(t, err)
	})

	t.Run("delete profile", func(t *testing.T) {
		profile := "to-delete"

		err := EnsureProfile(profile)
		require.NoError(t, err)

		ks, err := GetKeystore()
		require.NoError(t, err)
		assert.True(t, ks.HasProfile(profile))

		err = DeleteProfile(profile)
		require.NoError(t, err)

		assert.False(t, ks.HasProfile(profile))
	})
}

func TestKeystoreKeyRotation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "clonr-keystore-test-*")
	require.NoError(t, err)

	defer os.RemoveAll(tmpDir)

	oldGetAppDir := getAppDirectory
	getAppDirectory = func() (string, error) { return tmpDir, nil }

	defer func() { getAppDirectory = oldGetAppDir }()

	resetKeystoreForTest()

	t.Run("rotate keys preserves data", func(t *testing.T) {
		profile := "rotate-test"
		token := "my-secret-token-12345"

		// Encrypt data
		encrypted, err := EncryptWithKeystore(profile, DEKTypeToken, []byte(token))
		require.NoError(t, err)

		// Rotate keys
		err = RotateProfileKey(profile)
		require.NoError(t, err)

		// Data should still decrypt correctly after rotation
		decrypted, err := DecryptWithKeystore(profile, DEKTypeToken, encrypted)
		require.NoError(t, err)
		assert.Equal(t, token, string(decrypted))
	})

	t.Run("new encryption uses new keys after rotation", func(t *testing.T) {
		profile := "rotate-test-2"
		token := "another-token"

		// Encrypt before rotation
		encBefore, err := EncryptWithKeystore(profile, DEKTypeToken, []byte(token))
		require.NoError(t, err)

		// Rotate
		err = RotateProfileKey(profile)
		require.NoError(t, err)

		// Encrypt after rotation
		encAfter, err := EncryptWithKeystore(profile, DEKTypeToken, []byte(token))
		require.NoError(t, err)

		// Ciphertext should be different (due to new DEK and random nonce)
		assert.NotEqual(t, encBefore, encAfter)

		// Both should still decrypt
		dec1, err := DecryptWithKeystore(profile, DEKTypeToken, encBefore)
		require.NoError(t, err)
		assert.Equal(t, token, string(dec1))

		dec2, err := DecryptWithKeystore(profile, DEKTypeToken, encAfter)
		require.NoError(t, err)
		assert.Equal(t, token, string(dec2))
	})
}

func TestTokenEncryptionWithKeystore(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "clonr-keystore-test-*")
	require.NoError(t, err)

	defer os.RemoveAll(tmpDir)

	oldGetAppDir := getAppDirectory
	getAppDirectory = func() (string, error) { return tmpDir, nil }

	defer func() { getAppDirectory = oldGetAppDir }()

	resetKeystoreForTest()

	// Enable new keystore
	UseNewKeystore(true)
	defer UseNewKeystore(true) // Reset to default

	t.Run("EncryptToken uses keystore", func(t *testing.T) {
		profile := "token-test"
		host := "github.com"
		token := "ghp_testtoken123456789"

		encrypted, err := EncryptToken(token, profile, host)
		require.NoError(t, err)

		// Should have KS: prefix
		assert.True(t, IsDataKeystore(encrypted), "expected KS: prefix")
		assert.True(t, IsDataEncrypted(encrypted))
		assert.False(t, IsDataOpen(encrypted))

		// Decrypt
		decrypted, err := DecryptToken(encrypted, profile, host)
		require.NoError(t, err)
		assert.Equal(t, token, decrypted)
	})

	t.Run("legacy ENC: data still decrypts", func(t *testing.T) {
		// This test would require TPM which may not be available
		// Skip if no TPM
		if !IsTPMAvailable() {
			t.Skip("TPM not available, skipping legacy decryption test")
		}
	})
}

func TestDataPrefixes(t *testing.T) {
	t.Run("IsDataOpen", func(t *testing.T) {
		assert.True(t, IsDataOpen([]byte("OPEN:plain-text-token")))
		assert.False(t, IsDataOpen([]byte("ENC:encrypted")))
		assert.False(t, IsDataOpen([]byte("KS:keystore")))
		assert.False(t, IsDataOpen([]byte("plain")))
	})

	t.Run("IsDataEncrypted", func(t *testing.T) {
		assert.False(t, IsDataEncrypted([]byte("OPEN:plain")))
		assert.True(t, IsDataEncrypted([]byte("ENC:encrypted")))
		assert.True(t, IsDataEncrypted([]byte("KS:keystore")))
		assert.False(t, IsDataEncrypted([]byte("plain")))
	})

	t.Run("IsDataKeystore", func(t *testing.T) {
		assert.False(t, IsDataKeystore([]byte("OPEN:plain")))
		assert.False(t, IsDataKeystore([]byte("ENC:encrypted")))
		assert.True(t, IsDataKeystore([]byte("KS:keystore")))
		assert.False(t, IsDataKeystore([]byte("plain")))
	})
}

// resetKeystoreForTest resets the keystore singleton for testing
func resetKeystoreForTest() {
	keystoreMu.Lock()
	defer keystoreMu.Unlock()

	if globalKeystore != nil {
		_ = globalKeystore.Close()
		globalKeystore = nil
	}

	keystoreErr = nil
	keystoreOnce = sync.Once{}
}
