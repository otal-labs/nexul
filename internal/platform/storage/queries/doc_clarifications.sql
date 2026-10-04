-- name: ListDocClarificationRounds :many
SELECT * FROM doc_clarification_rounds WHERE doc_id = ? ORDER BY round;

-- name: InsertDocClarificationRound :exec
INSERT INTO doc_clarification_rounds (doc_id, round, started_by, trail_id, started_at, running, took_lock)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: UpdateDocClarificationRound :execrows
UPDATE doc_clarification_rounds SET running = ?, anything_else = ?, anything_else_by = ?, anything_else_at = ?,
    anything_else_reply = ?, no_gaps_at = ?, closed_by = ?, closed_at = ?
WHERE doc_id = ? AND round = ?;

-- name: DeleteDocClarificationRound :execrows
DELETE FROM doc_clarification_rounds WHERE doc_id = ? AND round = ?;

-- name: ListDocClarificationQuestions :many
SELECT * FROM doc_clarification_questions WHERE doc_id = ? ORDER BY round, position;

-- name: GetDocClarificationQuestion :one
SELECT * FROM doc_clarification_questions WHERE id = ?;

-- name: InsertDocClarificationQuestion :exec
INSERT INTO doc_clarification_questions (id, doc_id, round, position, question, why, options, multi_select)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateDocClarificationAnswer :execrows
UPDATE doc_clarification_questions SET selected = ?, free_text = ?, skipped = ?, answered_by = ?, answered_at = ?
WHERE id = ?;

-- name: CountPendingDocClarificationQuestions :one
SELECT COUNT(*) FROM doc_clarification_questions
WHERE doc_id = ? AND round = ? AND skipped = 0 AND free_text = '' AND selected = '[]';
