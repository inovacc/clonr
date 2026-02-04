-- Migration: 004_slack_accounts (down)
-- Description: Remove Slack accounts table

DROP INDEX IF EXISTS idx_slack_accounts_default;
DROP TABLE IF EXISTS slack_accounts;

DELETE FROM schema_migrations WHERE version = 4;
