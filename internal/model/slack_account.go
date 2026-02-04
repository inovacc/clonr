package model

import "time"

// SlackAccount represents a named Slack token configuration.
// Users can have multiple Slack accounts and switch between them.
type SlackAccount struct {
	// Name is the unique identifier for this account (e.g., "work", "personal")
	Name string `json:"name"`

	// WorkspaceID is the Slack workspace ID (team ID)
	WorkspaceID string `json:"workspace_id,omitempty"`

	// WorkspaceName is the human-readable workspace name
	WorkspaceName string `json:"workspace_name,omitempty"`

	// BotUserID is the bot user ID in the workspace
	BotUserID string `json:"bot_user_id,omitempty"`

	// TeamID is the Slack team ID (same as workspace ID in most cases)
	TeamID string `json:"team_id,omitempty"`

	// Default indicates if this is the default account used when none specified
	Default bool `json:"default"`

	// EncryptedBotToken stores the TPM-encrypted bot token (xoxb-...)
	EncryptedBotToken []byte `json:"encrypted_bot_token,omitempty"`

	// TokenStorage indicates how the token is stored (encrypted or open)
	TokenStorage TokenStorage `json:"token_storage"`

	// CreatedAt is when the account was created
	CreatedAt time.Time `json:"created_at"`

	// LastUsedAt is when the account was last used
	LastUsedAt time.Time `json:"last_used_at"`
}
