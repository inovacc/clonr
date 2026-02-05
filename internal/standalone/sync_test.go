package standalone

import (
	"bytes"
	"testing"
)

func TestEncryptForSync(t *testing.T) {
	key := make([]byte, keySize)
	for i := range key {
		key[i] = byte(i)
	}

	data := []byte("test data to sync")

	synced, err := EncryptForSync(data, key)
	if err != nil {
		t.Fatalf("EncryptForSync() error = %v", err)
	}

	if synced.State != SyncStateEncrypted {
		t.Errorf("State = %s, want %s", synced.State, SyncStateEncrypted)
	}

	if len(synced.EncryptedData) == 0 {
		t.Error("EncryptedData is empty")
	}

	if synced.SyncedAt.IsZero() {
		t.Error("SyncedAt is zero")
	}
}

func TestEncryptForSyncInvalidKey(t *testing.T) {
	shortKey := make([]byte, 16) // Too short
	data := []byte("test")

	_, err := EncryptForSync(data, shortKey)
	if err == nil {
		t.Error("EncryptForSync() expected error with short key")
	}
}

func TestDecryptSyncedData(t *testing.T) {
	key := make([]byte, keySize)
	for i := range key {
		key[i] = byte(i)
	}

	originalData := []byte("test data to sync and decrypt")

	synced, err := EncryptForSync(originalData, key)
	if err != nil {
		t.Fatalf("EncryptForSync() error = %v", err)
	}

	decrypted, err := DecryptSyncedData(synced, key)
	if err != nil {
		t.Fatalf("DecryptSyncedData() error = %v", err)
	}

	if !bytes.Equal(decrypted, originalData) {
		t.Errorf("DecryptSyncedData() = %s, want %s", decrypted, originalData)
	}
}

func TestDecryptSyncedDataWrongKey(t *testing.T) {
	key := make([]byte, keySize)

	wrongKey := make([]byte, keySize)
	for i := range wrongKey {
		wrongKey[i] = byte(i + 1) // Different key
	}

	data := []byte("test data")
	synced, _ := EncryptForSync(data, key)

	_, err := DecryptSyncedData(synced, wrongKey)
	if err == nil {
		t.Error("DecryptSyncedData() expected error with wrong key")
	}
}

func TestDecryptAlreadyDecrypted(t *testing.T) {
	synced := &SyncedData{
		State: SyncStateDecrypted,
	}

	key := make([]byte, keySize)

	_, err := DecryptSyncedData(synced, key)
	if err == nil {
		t.Error("DecryptSyncedData() expected error for already decrypted data")
	}
}

func TestCreateSyncPackage(t *testing.T) {
	key := make([]byte, keySize)
	for i := range key {
		key[i] = byte(i)
	}

	items := map[string][]byte{
		"profile1": []byte(`{"name":"test1"}`),
		"profile2": []byte(`{"name":"test2"}`),
	}

	pkg, err := CreateSyncPackage("test-instance", key, items)
	if err != nil {
		t.Fatalf("CreateSyncPackage() error = %v", err)
	}

	if pkg.Version != 1 {
		t.Errorf("Version = %d, want 1", pkg.Version)
	}

	if pkg.InstanceID != "test-instance" {
		t.Errorf("InstanceID = %s, want test-instance", pkg.InstanceID)
	}

	if len(pkg.Items) != 2 {
		t.Errorf("Items count = %d, want 2", len(pkg.Items))
	}

	if pkg.EncryptionKey == "" {
		t.Error("EncryptionKey hint is empty")
	}
}

func TestPendingSyncStore(t *testing.T) {
	store := NewPendingSyncStore()

	// Add items
	item1 := &SyncedData{
		ConnectionName: "conn1",
		DataType:       "profile",
		Name:           "profile1",
		State:          SyncStateEncrypted,
	}
	item2 := &SyncedData{
		ConnectionName: "conn1",
		DataType:       "workspace",
		Name:           "workspace1",
		State:          SyncStateEncrypted,
	}
	item3 := &SyncedData{
		ConnectionName: "conn2",
		DataType:       "profile",
		Name:           "profile2",
		State:          SyncStateDecrypted,
	}

	store.Add(item1)
	store.Add(item2)
	store.Add(item3)

	// Test Get
	t.Run("Get", func(t *testing.T) {
		got := store.Get("conn1", "profile", "profile1")
		if got != item1 {
			t.Error("Get() returned wrong item")
		}

		notFound := store.Get("conn1", "profile", "nonexistent")
		if notFound != nil {
			t.Error("Get() should return nil for nonexistent item")
		}
	})

	// Test List
	t.Run("List", func(t *testing.T) {
		conn1Items := store.List("conn1")
		if len(conn1Items) != 2 {
			t.Errorf("List(conn1) count = %d, want 2", len(conn1Items))
		}

		conn2Items := store.List("conn2")
		if len(conn2Items) != 1 {
			t.Errorf("List(conn2) count = %d, want 1", len(conn2Items))
		}
	})

	// Test ListByState
	t.Run("ListByState", func(t *testing.T) {
		encrypted := store.ListByState(SyncStateEncrypted)
		if len(encrypted) != 2 {
			t.Errorf("ListByState(encrypted) count = %d, want 2", len(encrypted))
		}

		decrypted := store.ListByState(SyncStateDecrypted)
		if len(decrypted) != 1 {
			t.Errorf("ListByState(decrypted) count = %d, want 1", len(decrypted))
		}
	})

	// Test Remove
	t.Run("Remove", func(t *testing.T) {
		store.Remove("conn1", "profile", "profile1")

		got := store.Get("conn1", "profile", "profile1")
		if got != nil {
			t.Error("Remove() did not remove item")
		}
	})
}

func TestEncryptionKeyManager(t *testing.T) {
	t.Run("SetupKey", func(t *testing.T) {
		manager := NewEncryptionKeyManager()

		config, err := manager.SetupKey("testpassword123")
		if err != nil {
			t.Fatalf("SetupKey() error = %v", err)
		}

		if !config.Enabled {
			t.Error("config.Enabled = false, want true")
		}

		if len(config.KeyHash) == 0 {
			t.Error("config.KeyHash is empty")
		}

		if len(config.Salt) == 0 {
			t.Error("config.Salt is empty")
		}

		if config.KeyHint == "" {
			t.Error("config.KeyHint is empty")
		}
	})

	t.Run("SetupKeyTooShort", func(t *testing.T) {
		manager := NewEncryptionKeyManager()

		_, err := manager.SetupKey("short")
		if err == nil {
			t.Error("SetupKey() expected error for short password")
		}
	})

	t.Run("VerifyKey", func(t *testing.T) {
		manager := NewEncryptionKeyManager()
		_, _ = manager.SetupKey("testpassword123")

		if !manager.VerifyKey("testpassword123") {
			t.Error("VerifyKey() returned false for correct password")
		}

		if manager.VerifyKey("wrongpassword") {
			t.Error("VerifyKey() returned true for wrong password")
		}
	})

	t.Run("DeriveKey", func(t *testing.T) {
		manager := NewEncryptionKeyManager()
		_, _ = manager.SetupKey("testpassword123")

		key, err := manager.DeriveKey("testpassword123")
		if err != nil {
			t.Fatalf("DeriveKey() error = %v", err)
		}

		if len(key) != keySize {
			t.Errorf("DeriveKey() key length = %d, want %d", len(key), keySize)
		}

		_, err = manager.DeriveKey("wrongpassword")
		if err == nil {
			t.Error("DeriveKey() expected error for wrong password")
		}
	})

	t.Run("IsConfigured", func(t *testing.T) {
		manager := NewEncryptionKeyManager()

		if manager.IsConfigured() {
			t.Error("IsConfigured() = true before setup")
		}

		_, _ = manager.SetupKey("testpassword123")

		if !manager.IsConfigured() {
			t.Error("IsConfigured() = false after setup")
		}
	})

	t.Run("SaveAndLoadConfig", func(t *testing.T) {
		manager := NewEncryptionKeyManager()
		_, _ = manager.SetupKey("testpassword123")

		data, err := manager.SaveConfig()
		if err != nil {
			t.Fatalf("SaveConfig() error = %v", err)
		}

		newManager := NewEncryptionKeyManager()
		if err := newManager.LoadConfig(data); err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}

		if !newManager.VerifyKey("testpassword123") {
			t.Error("Loaded config doesn't verify correct password")
		}
	})
}

func TestSyncStates(t *testing.T) {
	if SyncStateEncrypted != "encrypted" {
		t.Errorf("SyncStateEncrypted = %s, want encrypted", SyncStateEncrypted)
	}

	if SyncStateDecrypted != "decrypted" {
		t.Errorf("SyncStateDecrypted = %s, want decrypted", SyncStateDecrypted)
	}

	if SyncStatePending != "pending" {
		t.Errorf("SyncStatePending = %s, want pending", SyncStatePending)
	}
}
