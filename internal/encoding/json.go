// Package encoding provides utilities for encoding and decoding data.
package encoding

import (
	"encoding/json"
	"fmt"
	"os"
)

// LoadJSON reads a JSON file and unmarshals it into the provided value.
// Returns nil, nil if the file does not exist.
// Returns an error for other file access or parsing issues.
func LoadJSON[T any](path string) (*T, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON from %s: %w", path, err)
	}

	return &result, nil
}

// SaveJSON marshals the value to JSON and writes it to the specified path.
// Creates parent directories if they don't exist.
// Uses 0600 permissions for the file.
func SaveJSON[T any](path string, value T) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write file %s: %w", path, err)
	}

	return nil
}

// MustLoadJSON is like LoadJSON but panics on error.
// Use only when the file is expected to exist and be valid.
func MustLoadJSON[T any](path string) *T {
	result, err := LoadJSON[T](path)
	if err != nil {
		panic(fmt.Sprintf("failed to load JSON from %s: %v", path, err))
	}

	return result
}

// ParseJSON unmarshals JSON data into the provided type.
// Returns an error if parsing fails.
func ParseJSON[T any](data []byte) (*T, error) {
	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return &result, nil
}

// ToJSON marshals a value to JSON bytes.
// Returns an error if marshaling fails.
func ToJSON[T any](value T) ([]byte, error) {
	return json.Marshal(value)
}

// ToJSONIndent marshals a value to indented JSON bytes.
// Returns an error if marshaling fails.
func ToJSONIndent[T any](value T) ([]byte, error) {
	return json.MarshalIndent(value, "", "  ")
}
