-- name: GetSlackConfig :one
SELECT * FROM slack_config WHERE id = 1 LIMIT 1;

-- name: SlackConfigExists :one
SELECT EXISTS(SELECT 1 FROM slack_config WHERE id = 1) AS exists_flag;

-- name: InsertSlackConfig :one
INSERT INTO slack_config (
    id, enabled, workspace_id, workspace_name, encrypted_webhook_url,
    encrypted_bot_token, default_channel, bot_enabled, events, created_at, updated_at
) VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
RETURNING *;

-- name: UpdateSlackConfig :exec
UPDATE slack_config SET
    enabled = ?,
    workspace_id = ?,
    workspace_name = ?,
    encrypted_webhook_url = ?,
    encrypted_bot_token = ?,
    default_channel = ?,
    bot_enabled = ?,
    events = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = 1;

-- name: DeleteSlackConfig :exec
DELETE FROM slack_config WHERE id = 1;

-- name: EnableSlackNotifications :exec
UPDATE slack_config SET enabled = 1, updated_at = CURRENT_TIMESTAMP WHERE id = 1;

-- name: DisableSlackNotifications :exec
UPDATE slack_config SET enabled = 0, updated_at = CURRENT_TIMESTAMP WHERE id = 1;
