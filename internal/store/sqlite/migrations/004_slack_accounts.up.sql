-- Migration: 004_slack_accounts
-- Description: Add Slack accounts table for multi-account support
-- Created: 2026-02-04

-- Slack accounts table (multiple accounts per user)
CREATE TABLE IF NOT EXISTS slack_accounts (
    name TEXT PRIMARY KEY,                   -- Unique account name (e.g., "work", "personal")
    workspace_id TEXT,                       -- Slack workspace ID
    workspace_name TEXT,                     -- Slack workspace name
    bot_user_id TEXT,                        -- Bot user ID in the workspace
    team_id TEXT,                            -- Slack team ID
    is_default INTEGER DEFAULT 0,            -- 0 = not default, 1 = default account
    encrypted_bot_token BLOB,                -- TPM-encrypted bot token (xoxb-...)
    token_storage TEXT DEFAULT 'encrypted',  -- 'encrypted' or 'open'
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_used_at DATETIME
);

-- Index for finding the default account quickly
CREATE INDEX IF NOT EXISTS idx_slack_accounts_default ON slack_accounts(is_default);

-- Record this migration
INSERT INTO schema_migrations (version, description) VALUES (4, 'Slack accounts');
