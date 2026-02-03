-- name: GetAllRepos :many
SELECT * FROM repositories ORDER BY updated_at DESC;

-- name: GetRepoByURL :one
SELECT * FROM repositories WHERE url = ? LIMIT 1;

-- name: GetRepoByPath :one
SELECT * FROM repositories WHERE path = ? LIMIT 1;

-- name: GetReposByWorkspace :many
SELECT * FROM repositories WHERE workspace = ? ORDER BY updated_at DESC;

-- name: GetReposByWorkspaceAndFavorites :many
SELECT * FROM repositories
WHERE (workspace = ? OR ? = '')
  AND (favorite = 1 OR ? = 0)
ORDER BY updated_at DESC;

-- name: RepoExistsByURL :one
SELECT EXISTS(SELECT 1 FROM repositories WHERE url = ?) AS exists_flag;

-- name: RepoExistsByPath :one
SELECT EXISTS(SELECT 1 FROM repositories WHERE path = ?) AS exists_flag;

-- name: InsertRepo :one
INSERT INTO repositories (uid, url, path, workspace, favorite, cloned_at, updated_at)
VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
RETURNING *;

-- name: UpdateRepoWorkspace :exec
UPDATE repositories SET workspace = ?, updated_at = CURRENT_TIMESTAMP WHERE url = ?;

-- name: UpdateRepoFavorite :exec
UPDATE repositories SET favorite = ?, updated_at = CURRENT_TIMESTAMP WHERE url = ?;

-- name: UpdateRepoTimestamp :exec
UPDATE repositories SET updated_at = CURRENT_TIMESTAMP WHERE url = ?;

-- name: UpdateRepoLastChecked :exec
UPDATE repositories SET last_checked = CURRENT_TIMESTAMP WHERE url = ?;

-- name: DeleteRepoByURL :exec
DELETE FROM repositories WHERE url = ?;

-- name: DeleteRepoByPath :exec
DELETE FROM repositories WHERE path = ?;
