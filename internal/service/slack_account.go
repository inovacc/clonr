package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/inovacc/clonr/internal/crypto/tpm"
	"github.com/inovacc/clonr/internal/model"
	"github.com/inovacc/clonr/internal/store"
)

var (
	// ErrSlackAccountNotFound is returned when a Slack account doesn't exist
	ErrSlackAccountNotFound = errors.New("slack account not found")

	// ErrSlackAccountExists is returned when trying to create an account that already exists
	ErrSlackAccountExists = errors.New("slack account already exists")

	// ErrNoActiveSlackAccount is returned when no Slack account is active
	ErrNoActiveSlackAccount = errors.New("no active slack account")

	// ErrSlackTokenNotFound is returned when the token cannot be retrieved
	ErrSlackTokenNotFound = errors.New("slack token not found")
)

// SlackAccountService provides Slack account operations with direct database access.
type SlackAccountService struct {
	store store.Store
}

// NewSlackAccountService creates a new SlackAccountService with direct database access.
func NewSlackAccountService(s store.Store) *SlackAccountService {
	return &SlackAccountService{store: s}
}

// CreateAccount creates a new Slack account with the provided token.
func (sas *SlackAccountService) CreateAccount(name, token string) (*model.SlackAccount, error) {
	exists, err := sas.store.SlackAccountExists(name)
	if err != nil {
		return nil, fmt.Errorf("failed to check account existence: %w", err)
	}

	if exists {
		return nil, ErrSlackAccountExists
	}

	// Encrypt token
	encryptedToken, tokenStorage, err := sas.EncryptToken(token, name)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt token: %w", err)
	}

	// Check if this is the first account (auto-set as default)
	accounts, err := sas.store.ListSlackAccounts()
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}

	isFirstAccount := len(accounts) == 0

	account := &model.SlackAccount{
		Name:              name,
		Default:           isFirstAccount,
		EncryptedBotToken: encryptedToken,
		TokenStorage:      tokenStorage,
		CreatedAt:         time.Now(),
		LastUsedAt:        time.Now(),
	}

	if err := sas.store.SaveSlackAccount(account); err != nil {
		return nil, fmt.Errorf("failed to save account: %w", err)
	}

	return account, nil
}

// CreateAccountWithInfo creates a Slack account with workspace info.
func (sas *SlackAccountService) CreateAccountWithInfo(
	name, token, workspaceID, workspaceName, botUserID, teamID string,
) (*model.SlackAccount, error) {
	exists, err := sas.store.SlackAccountExists(name)
	if err != nil {
		return nil, fmt.Errorf("failed to check account existence: %w", err)
	}

	if exists {
		return nil, ErrSlackAccountExists
	}

	// Encrypt token
	encryptedToken, tokenStorage, err := sas.EncryptToken(token, name)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt token: %w", err)
	}

	// Check if this is the first account (auto-set as default)
	accounts, err := sas.store.ListSlackAccounts()
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}

	isFirstAccount := len(accounts) == 0

	account := &model.SlackAccount{
		Name:              name,
		WorkspaceID:       workspaceID,
		WorkspaceName:     workspaceName,
		BotUserID:         botUserID,
		TeamID:            teamID,
		Default:           isFirstAccount,
		EncryptedBotToken: encryptedToken,
		TokenStorage:      tokenStorage,
		CreatedAt:         time.Now(),
		LastUsedAt:        time.Now(),
	}

	if err := sas.store.SaveSlackAccount(account); err != nil {
		return nil, fmt.Errorf("failed to save account: %w", err)
	}

	return account, nil
}

// GetAccount retrieves a Slack account by name.
func (sas *SlackAccountService) GetAccount(name string) (*model.SlackAccount, error) {
	account, err := sas.store.GetSlackAccount(name)
	if err != nil {
		return nil, err
	}

	if account == nil {
		return nil, ErrSlackAccountNotFound
	}

	return account, nil
}

// GetActiveAccount retrieves the currently active Slack account.
func (sas *SlackAccountService) GetActiveAccount() (*model.SlackAccount, error) {
	account, err := sas.store.GetActiveSlackAccount()
	if err != nil {
		return nil, err
	}

	if account == nil {
		return nil, ErrNoActiveSlackAccount
	}

	return account, nil
}

// SetActiveAccount sets a Slack account as active.
func (sas *SlackAccountService) SetActiveAccount(name string) error {
	exists, err := sas.store.SlackAccountExists(name)
	if err != nil {
		return fmt.Errorf("failed to check account existence: %w", err)
	}

	if !exists {
		return ErrSlackAccountNotFound
	}

	return sas.store.SetActiveSlackAccount(name)
}

// ListAccounts returns all Slack accounts.
func (sas *SlackAccountService) ListAccounts() ([]*model.SlackAccount, error) {
	return sas.store.ListSlackAccounts()
}

// DeleteAccount removes a Slack account.
func (sas *SlackAccountService) DeleteAccount(name string) error {
	account, err := sas.store.GetSlackAccount(name)
	if err != nil {
		return fmt.Errorf("failed to get account: %w", err)
	}

	if account == nil {
		return ErrSlackAccountNotFound
	}

	return sas.store.DeleteSlackAccount(name)
}

// AccountExists checks if a Slack account exists.
func (sas *SlackAccountService) AccountExists(name string) (bool, error) {
	return sas.store.SlackAccountExists(name)
}

// GetDecryptedToken retrieves and decrypts the token for an account.
func (sas *SlackAccountService) GetDecryptedToken(name string) (string, error) {
	account, err := sas.store.GetSlackAccount(name)
	if err != nil {
		return "", fmt.Errorf("failed to get account: %w", err)
	}

	if account == nil {
		return "", ErrSlackAccountNotFound
	}

	return sas.getTokenFromAccount(account)
}

// GetActiveAccountToken retrieves the token for the active account.
func (sas *SlackAccountService) GetActiveAccountToken() (string, error) {
	account, err := sas.store.GetActiveSlackAccount()
	if err != nil {
		return "", err
	}

	if account == nil {
		return "", ErrNoActiveSlackAccount
	}

	return sas.getTokenFromAccount(account)
}

// SaveAccount saves a Slack account to the database.
func (sas *SlackAccountService) SaveAccount(account *model.SlackAccount) error {
	return sas.store.SaveSlackAccount(account)
}

// UpdateAccountWithToken updates an existing account with a new token.
func (sas *SlackAccountService) UpdateAccountWithToken(name, token string) (*model.SlackAccount, error) {
	account, err := sas.store.GetSlackAccount(name)
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	if account == nil {
		return nil, ErrSlackAccountNotFound
	}

	// Encrypt new token
	encryptedToken, tokenStorage, err := sas.EncryptToken(token, name)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt token: %w", err)
	}

	account.EncryptedBotToken = encryptedToken
	account.TokenStorage = tokenStorage
	account.LastUsedAt = time.Now()

	if err := sas.store.SaveSlackAccount(account); err != nil {
		return nil, fmt.Errorf("failed to save account: %w", err)
	}

	return account, nil
}

// EncryptToken encrypts a token for storage in an account.
func (sas *SlackAccountService) EncryptToken(token, name string) ([]byte, model.TokenStorage, error) {
	encryptedToken, err := tpm.EncryptToken(token, name, "slack")
	if err != nil {
		return nil, "", fmt.Errorf("failed to encrypt token: %w", err)
	}

	storageType := model.TokenStorageEncrypted
	if tpm.IsDataOpen(encryptedToken) {
		storageType = model.TokenStorageOpen
	}

	return encryptedToken, storageType, nil
}

// getTokenFromAccount retrieves and decrypts the token from the account.
func (sas *SlackAccountService) getTokenFromAccount(account *model.SlackAccount) (string, error) {
	if len(account.EncryptedBotToken) == 0 {
		return "", ErrSlackTokenNotFound
	}

	token, err := tpm.DecryptToken(account.EncryptedBotToken, account.Name, "slack")
	if err != nil {
		return "", fmt.Errorf("failed to decrypt token: %w", err)
	}

	return token, nil
}
