-- name: GetGitHubIssueSync :one
SELECT * FROM github_issue_sync WHERE owner = ? AND repo = ?;

-- name: ListGitHubIssueSyncsOfProject :many
SELECT * FROM github_issue_sync WHERE project_id = ? ORDER BY owner, repo;

-- name: ListEnabledGitHubIssueSyncs :many
SELECT * FROM github_issue_sync WHERE enabled = 1 ORDER BY owner, repo;

-- name: UpsertGitHubIssueSync :exec
INSERT INTO github_issue_sync (project_id, owner, repo, label, mode, enabled, last_swept_at) VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(owner, repo) DO UPDATE SET label = excluded.label, mode = excluded.mode, enabled = excluded.enabled,
    last_swept_at = excluded.last_swept_at;

-- name: MarkGitHubIssueSyncSwept :exec
UPDATE github_issue_sync SET last_swept_at = ? WHERE owner = ? AND repo = ?;

-- name: GetGitHubIssueLink :one
SELECT * FROM github_issue_links WHERE owner = ? AND repo = ? AND issue_number = ?;

-- name: UpsertGitHubIssueLink :exec
INSERT INTO github_issue_links (owner, repo, issue_number, issue_id, ticket_id, sync_paused, last_inbound_hash, last_outbound_hash, issue_updated_at, linked_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(owner, repo, issue_number) DO UPDATE SET issue_id = excluded.issue_id, sync_paused = excluded.sync_paused,
    last_inbound_hash = excluded.last_inbound_hash, last_outbound_hash = excluded.last_outbound_hash,
    issue_updated_at = excluded.issue_updated_at;

-- name: DeleteGitHubIssueLink :exec
DELETE FROM github_issue_links WHERE owner = ? AND repo = ? AND issue_number = ?;

-- name: GetGitHubCommentLink :one
SELECT * FROM github_comment_links WHERE github_comment_id = ?;

-- name: InsertGitHubCommentLink :exec
INSERT INTO github_comment_links (github_comment_id, message_id, ticket_id) VALUES (?, ?, ?)
ON CONFLICT(github_comment_id) DO NOTHING;

-- name: DeleteGitHubCommentLink :exec
DELETE FROM github_comment_links WHERE github_comment_id = ?;

-- name: GitHubIdentityUserID :one
SELECT user_id FROM user_identities WHERE provider = 'github' AND lower(login) = lower(?) ORDER BY created_at LIMIT 1;
