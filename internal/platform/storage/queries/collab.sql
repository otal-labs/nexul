-- name: AppendCollabUpdate :execlastid
INSERT INTO collab_updates (doc_id, kind, actor_id, payload, created_at) VALUES (?, ?, ?, ?, ?);

-- name: TouchCollabSession :exec
INSERT INTO collab_sessions (doc_id, last_seq, last_commit_at) VALUES (?, ?, ?)
ON CONFLICT(doc_id) DO UPDATE SET last_seq = excluded.last_seq;

-- name: GetLatestCollabSnapshot :one
SELECT * FROM collab_updates WHERE doc_id = ? AND kind = 'snapshot' ORDER BY seq DESC LIMIT 1;

-- name: ListCollabIncrementsAfter :many
SELECT * FROM collab_updates WHERE doc_id = ? AND kind = 'update' AND seq > ? ORDER BY seq;

-- name: GetCollabSessionSeq :one
SELECT last_seq FROM collab_sessions WHERE doc_id = ?;

-- name: TrimCollabUpdates :exec
DELETE FROM collab_updates WHERE doc_id = ? AND kind = 'update' AND seq <= ?;
