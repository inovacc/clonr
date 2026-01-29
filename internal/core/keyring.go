package core

import (
	"context"
	"fmt"
	"time"

	"github.com/zalando/go-keyring"
)

const (
	// keyringService is the service name used for keyring entries
	keyringService = "clonr"

	// keyringTimeout is the timeout for keyring operations
	keyringTimeout = 5 * time.Second
)

// KeyringError represents an error during keyring operations
type KeyringError struct {
	Operation string
	Err       error
}

func (e *KeyringError) Error() string {
	return fmt.Sprintf("keyring %s failed: %v", e.Operation, e.Err)
}

func (e *KeyringError) Unwrap() error {
	return e.Err
}

// keyringKey generates a consistent key format for storing tokens
func keyringKey(profileName, host string) string {
	return fmt.Sprintf("profile:%s:%s", profileName, host)
}

// SetToken stores a token in the system keyring with timeout
func SetToken(profileName, host, token string) error {
	ctx, cancel := context.WithTimeout(context.Background(), keyringTimeout)
	defer cancel()

	errCh := make(chan error, 1)

	go func() {
		key := keyringKey(profileName, host)
		errCh <- keyring.Set(keyringService, key, token)
	}()

	select {
	case err := <-errCh:
		if err != nil {
			return &KeyringError{Operation: "set", Err: err}
		}

		return nil
	case <-ctx.Done():
		return &KeyringError{Operation: "set", Err: ctx.Err()}
	}
}

// GetToken retrieves a token from the system keyring with timeout
func GetToken(profileName, host string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), keyringTimeout)
	defer cancel()

	type result struct {
		token string
		err   error
	}

	resultCh := make(chan result, 1)

	go func() {
		key := keyringKey(profileName, host)

		token, err := keyring.Get(keyringService, key)
		resultCh <- result{token: token, err: err}
	}()

	select {
	case r := <-resultCh:
		if r.err != nil {
			return "", &KeyringError{Operation: "get", Err: r.err}
		}

		return r.token, nil
	case <-ctx.Done():
		return "", &KeyringError{Operation: "get", Err: ctx.Err()}
	}
}

// DeleteToken removes a token from the system keyring with timeout
func DeleteToken(profileName, host string) error {
	ctx, cancel := context.WithTimeout(context.Background(), keyringTimeout)
	defer cancel()

	errCh := make(chan error, 1)

	go func() {
		key := keyringKey(profileName, host)
		errCh <- keyring.Delete(keyringService, key)
	}()

	select {
	case err := <-errCh:
		if err != nil {
			// Ignore "not found" errors when deleting
			if err == keyring.ErrNotFound {
				return nil
			}

			return &KeyringError{Operation: "delete", Err: err}
		}

		return nil
	case <-ctx.Done():
		return &KeyringError{Operation: "delete", Err: ctx.Err()}
	}
}

// IsKeyringAvailable checks if the system keyring is available
func IsKeyringAvailable() bool {
	// Try to set and delete a test key
	testKey := "__clonr_keyring_test__"

	if err := keyring.Set(keyringService, testKey, "test"); err != nil {
		return false
	}

	_ = keyring.Delete(keyringService, testKey)

	return true
}
