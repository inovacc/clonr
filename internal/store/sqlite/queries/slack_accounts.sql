-- name: GetSlackAccount :one
SELECT * FROM slack_accounts WHERE name = ? LIMIT 1;

-- name: GetActiveSlackAccount :one
SELECT * FROM slack_accounts WHERE is_default = 1 LIMIT 1;

-- name: ListSlackAccounts :many
SELECT * FROM slack_accounts ORDER BY name ASC;

-- name: SlackAccountExists :one
SELECT EXISTS(SELECT 1 FROM slack_accounts WHERE name = ?) AS exists_flag;

-- name: InsertSlackAccount :one
INSERT INTO slack_accounts (
    name, workspace_id, workspace_name, bot_user_id, team_id,
    is_default, encrypted_bot_token, token_storage, created_at, last_used_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, NULL)
RETURNING *;

-- name: UpdateSlackAccount :exec
UPDATE slack_accounts SET
    workspace_id = ?,
    workspace_name = ?,
    bot_user_id = ?,
    team_id = ?,
    encrypted_bot_token = ?,
    token_storage = ?,
    last_used_at = CURRENT_TIMESTAMP
WHERE name = ?;

-- name: UpdateSlackAccountLastUsed :exec
UPDATE slack_accounts SET last_used_at = CURRENT_TIMESTAMP WHERE name = ?;

-- name: SetActiveSlackAccount :exec
UPDATE slack_accounts SET is_default = CASE WHEN name = ? THEN 1 ELSE 0 END;

-- name: ClearActiveSlackAccount :exec
UPDATE slack_accounts SET is_default = 0;

-- name: DeleteSlackAccount :exec
DELETE FROM slack_accounts WHERE name = ?;
