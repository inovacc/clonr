package encoding

import (
	"fmt"
	"os"
	"path/filepath"
)

// FileExists checks if a file exists at the given path.
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// DirExists checks if a directory exists at the given path.
func DirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// EnsureDir creates a directory and all parent directories if they don't exist.
// Uses 0755 permissions.
func EnsureDir(path string) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", path, err)
	}
	return nil
}

// EnsureParentDir ensures the parent directory of a file path exists.
func EnsureParentDir(filePath string) error {
	return EnsureDir(filepath.Dir(filePath))
}

// ReadFile reads the entire contents of a file.
// Returns nil, nil if the file does not exist.
func ReadFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}
	return data, nil
}

// WriteFile writes data to a file with the specified permissions.
// Creates parent directories if they don't exist.
func WriteFile(path string, data []byte, perm os.FileMode) error {
	if err := EnsureParentDir(path); err != nil {
		return err
	}
	if err := os.WriteFile(path, data, perm); err != nil {
		return fmt.Errorf("failed to write file %s: %w", path, err)
	}
	return nil
}

// WriteFileSecure writes data to a file with 0600 permissions (owner read/write only).
// Use for sensitive data like tokens or configs.
func WriteFileSecure(path string, data []byte) error {
	return WriteFile(path, data, 0600)
}
