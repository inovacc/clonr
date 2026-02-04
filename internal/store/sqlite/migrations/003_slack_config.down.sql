-- Migration: 003_slack_config (rollback)
-- Description: Remove Slack integration configuration table

DROP TABLE IF EXISTS slack_config;

DELETE FROM schema_migrations WHERE version = 3;
