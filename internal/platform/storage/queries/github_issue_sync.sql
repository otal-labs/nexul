-- name: GetIssueSync :one
SELECT * FROM github_issue_sync WHERE owner = ? AND repo = ?;

-- name: ListIssueSyncsByProject :many
SELECT * FROM github_issue_sync WHERE project_id = ? ORDER BY owner, repo;

-- name: ListEnabledIssueSyncs :many
SELECT * FROM github_issue_sync WHERE enabled = 1 ORDER BY owner, repo;

-- name: PutIssueSync :exec
INSERT INTO github_issue_sync (project_id, owner, repo, label, mode, enabled, last_swept_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(owner, repo) DO UPDATE SET
  project_id = excluded.project_id, label = excluded.label, mode = excluded.mode,
  enabled = excluded.enabled, last_swept_at = excluded.last_swept_at;

-- name: SetIssueSyncSwept :execrows
UPDATE github_issue_sync SET last_swept_at = ? WHERE owner = ? AND repo = ?;

-- name: GetIssueLink :one
SELECT * FROM github_issue_links WHERE owner = ? AND repo = ? AND issue_number = ?;

-- name: PutIssueLink :exec
INSERT INTO github_issue_links (owner, repo, issue_number, issue_id, ticket_id, sync_paused, last_inbound_hash, last_outbound_hash, issue_updated_at, linked_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(owner, repo, issue_number) DO UPDATE SET
  issue_id = excluded.issue_id, ticket_id = excluded.ticket_id, sync_paused = excluded.sync_paused,
  last_inbound_hash = excluded.last_inbound_hash, last_outbound_hash = excluded.last_outbound_hash,
  issue_updated_at = excluded.issue_updated_at;

-- name: DeleteIssueLink :exec
DELETE FROM github_issue_links WHERE owner = ? AND repo = ? AND issue_number = ?;

-- name: GetIssueCommentLink :one
SELECT * FROM github_comment_links WHERE github_comment_id = ?;

-- name: PutIssueCommentLink :exec
INSERT INTO github_comment_links (github_comment_id, message_id, ticket_id) VALUES (?, ?, ?)
ON CONFLICT(github_comment_id) DO NOTHING;

-- name: DeleteIssueCommentLink :exec
DELETE FROM github_comment_links WHERE github_comment_id = ?;

-- name: DeleteIssueCommentLinksByTicket :exec
DELETE FROM github_comment_links WHERE ticket_id = ?;
