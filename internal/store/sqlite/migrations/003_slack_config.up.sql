-- Migration: 003_slack_config
-- Description: Add Slack integration configuration table
-- Created: 2026-02-03

-- Slack configuration table (singleton - one row)
CREATE TABLE IF NOT EXISTS slack_config (
    id INTEGER PRIMARY KEY CHECK (id = 1),  -- Singleton constraint
    enabled INTEGER DEFAULT 0,               -- 0 = disabled, 1 = enabled
    workspace_id TEXT,                       -- Slack workspace ID
    workspace_name TEXT,                     -- Slack workspace name
    encrypted_webhook_url BLOB,              -- TPM-encrypted webhook URL
    encrypted_bot_token BLOB,                -- TPM-encrypted bot token
    default_channel TEXT,                    -- Default channel (e.g., "#dev")
    bot_enabled INTEGER DEFAULT 0,           -- 0 = webhook only, 1 = bot enabled
    events TEXT DEFAULT '[]',                -- JSON array of SlackEventConfig
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Record this migration
INSERT INTO schema_migrations (version, description) VALUES (3, 'Slack configuration');
