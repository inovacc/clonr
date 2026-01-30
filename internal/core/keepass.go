package core

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/tobischo/gokeepasslib/v3"
	w "github.com/tobischo/gokeepasslib/v3/wrappers"
)

const (
	keepassDBName      = "clonr.kdbx"
	keepassGroupRoot   = "Clonr"
	keepassGroupTokens = "Profiles"
)

// KeePassManager handles KeePass database operations for token storage
type KeePassManager struct {
	db       *gokeepasslib.Database
	dbPath   string
	password string
}

// NewKeePassManager creates or opens a KeePass database
func NewKeePassManager(masterPassword string) (*KeePassManager, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config directory: %w", err)
	}

	dbPath := filepath.Join(configDir, "clonr", keepassDBName)

	kpm := &KeePassManager{
		dbPath:   dbPath,
		password: masterPassword,
	}

	// Check if database exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		if err := kpm.createDatabase(); err != nil {
			return nil, fmt.Errorf("failed to create database: %w", err)
		}
	} else {
		if err := kpm.loadDatabase(); err != nil {
			return nil, fmt.Errorf("failed to load database: %w", err)
		}
	}

	return kpm, nil
}

// GetKeePassDBPath returns the path to the KeePass database
func GetKeePassDBPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get config directory: %w", err)
	}

	return filepath.Join(configDir, "clonr", keepassDBName), nil
}

// KeePassDBExists checks if the KeePass database exists
func KeePassDBExists() bool {
	dbPath, err := GetKeePassDBPath()
	if err != nil {
		return false
	}

	_, err = os.Stat(dbPath)

	return err == nil
}

func (kpm *KeePassManager) createDatabase() error {
	// Ensure directory exists
	dir := filepath.Dir(kpm.dbPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create new database
	db := gokeepasslib.NewDatabase()
	db.Options = &gokeepasslib.DBOptions{}

	// Set master password
	db.Credentials = gokeepasslib.NewPasswordCredentials(kpm.password)

	// Create root group
	rootGroup := gokeepasslib.NewGroup()
	rootGroup.Name = keepassGroupRoot

	// Create Profiles group for storing tokens
	profilesGroup := gokeepasslib.NewGroup()
	profilesGroup.Name = keepassGroupTokens
	rootGroup.Groups = append(rootGroup.Groups, profilesGroup)

	// Add root group to database
	db.Content.Root.Groups = append(db.Content.Root.Groups, rootGroup)

	kpm.db = db

	// Lock and save
	if err := kpm.db.LockProtectedEntries(); err != nil {
		return fmt.Errorf("failed to lock database: %w", err)
	}

	return kpm.saveDatabase()
}

func (kpm *KeePassManager) loadDatabase() error {
	file, err := os.Open(kpm.dbPath)
	if err != nil {
		return err
	}

	defer func() { _ = file.Close() }()

	db := gokeepasslib.NewDatabase()
	db.Credentials = gokeepasslib.NewPasswordCredentials(kpm.password)

	if err := gokeepasslib.NewDecoder(file).Decode(db); err != nil {
		return fmt.Errorf("failed to decode database (incorrect password?): %w", err)
	}

	if err := db.UnlockProtectedEntries(); err != nil {
		return fmt.Errorf("failed to unlock database: %w", err)
	}

	kpm.db = db

	return nil
}

func (kpm *KeePassManager) saveDatabase() error {
	// Ensure directory exists
	dir := filepath.Dir(kpm.dbPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	// Lock protected entries before saving
	if err := kpm.db.LockProtectedEntries(); err != nil {
		return fmt.Errorf("failed to lock database: %w", err)
	}

	file, err := os.Create(kpm.dbPath)
	if err != nil {
		return err
	}

	defer func() { _ = file.Close() }()

	if err := gokeepasslib.NewEncoder(file).Encode(kpm.db); err != nil {
		return fmt.Errorf("failed to encode database: %w", err)
	}

	// Unlock again for continued use
	if err := kpm.db.UnlockProtectedEntries(); err != nil {
		return fmt.Errorf("failed to unlock database: %w", err)
	}

	return nil
}

func (kpm *KeePassManager) findProfilesGroup() *gokeepasslib.Group {
	if kpm.db == nil || kpm.db.Content == nil {
		return nil
	}

	var findInGroup func(*gokeepasslib.Group) *gokeepasslib.Group

	findInGroup = func(group *gokeepasslib.Group) *gokeepasslib.Group {
		if group.Name == keepassGroupTokens {
			return group
		}

		for i := range group.Groups {
			if found := findInGroup(&group.Groups[i]); found != nil {
				return found
			}
		}

		return nil
	}

	for i := range kpm.db.Content.Root.Groups {
		if found := findInGroup(&kpm.db.Content.Root.Groups[i]); found != nil {
			return found
		}
	}

	return nil
}

// SetProfileToken stores a token for a profile
func (kpm *KeePassManager) SetProfileToken(profileName, host, token string) error {
	group := kpm.findProfilesGroup()
	if group == nil {
		return fmt.Errorf("profiles group not found in database")
	}

	entryKey := fmt.Sprintf("%s:%s", profileName, host)

	// Find existing entry or create new one
	var entry *gokeepasslib.Entry

	for i := range group.Entries {
		if group.Entries[i].GetTitle() == entryKey {
			entry = &group.Entries[i]

			break
		}
	}

	if entry == nil {
		// Create new entry
		newEntry := gokeepasslib.NewEntry()
		newEntry.Values = []gokeepasslib.ValueData{
			{Key: "Title", Value: gokeepasslib.V{Content: entryKey}},
			{Key: "UserName", Value: gokeepasslib.V{Content: profileName}},
			{Key: "Password", Value: gokeepasslib.V{Content: token, Protected: w.NewBoolWrapper(true)}},
			{Key: "URL", Value: gokeepasslib.V{Content: host}},
		}
		group.Entries = append(group.Entries, newEntry)
	} else {
		// Update existing entry
		entry.Values = []gokeepasslib.ValueData{
			{Key: "Title", Value: gokeepasslib.V{Content: entryKey}},
			{Key: "UserName", Value: gokeepasslib.V{Content: profileName}},
			{Key: "Password", Value: gokeepasslib.V{Content: token, Protected: w.NewBoolWrapper(true)}},
			{Key: "URL", Value: gokeepasslib.V{Content: host}},
		}
	}

	return kpm.saveDatabase()
}

// GetProfileToken retrieves a token for a profile
func (kpm *KeePassManager) GetProfileToken(profileName, host string) (string, error) {
	group := kpm.findProfilesGroup()
	if group == nil {
		return "", fmt.Errorf("profiles group not found in database")
	}

	entryKey := fmt.Sprintf("%s:%s", profileName, host)

	for i := range group.Entries {
		if group.Entries[i].GetTitle() == entryKey {
			return group.Entries[i].GetPassword(), nil
		}
	}

	return "", fmt.Errorf("token not found for profile %s", profileName)
}

// DeleteProfileToken removes a token for a profile
func (kpm *KeePassManager) DeleteProfileToken(profileName, host string) error {
	group := kpm.findProfilesGroup()
	if group == nil {
		return fmt.Errorf("profiles group not found in database")
	}

	entryKey := fmt.Sprintf("%s:%s", profileName, host)

	for i := range group.Entries {
		if group.Entries[i].GetTitle() == entryKey {
			// Remove entry by replacing with last and truncating
			group.Entries[i] = group.Entries[len(group.Entries)-1]
			group.Entries = group.Entries[:len(group.Entries)-1]

			return kpm.saveDatabase()
		}
	}

	// Not found is not an error for deletion
	return nil
}

// ListProfiles returns all profile names stored in the database
func (kpm *KeePassManager) ListProfiles() []string {
	group := kpm.findProfilesGroup()
	if group == nil {
		return nil
	}

	profiles := make([]string, 0, len(group.Entries))

	for i := range group.Entries {
		profiles = append(profiles, group.Entries[i].GetContent("UserName"))
	}

	return profiles
}

// ChangePassword changes the master password of the database
func (kpm *KeePassManager) ChangePassword(newPassword string) error {
	kpm.db.Credentials = gokeepasslib.NewPasswordCredentials(newPassword)
	kpm.password = newPassword

	return kpm.saveDatabase()
}

// Close closes the database (no-op but good practice)
func (kpm *KeePassManager) Close() error {
	kpm.db = nil

	return nil
}
