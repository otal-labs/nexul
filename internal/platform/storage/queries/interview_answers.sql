-- name: ListInterviewAnswers :many
SELECT * FROM interview_answers WHERE project_id = ? ORDER BY round, id;

-- name: UpsertInterviewAnswer :one
INSERT INTO interview_answers (id, workspace_id, project_id, round, question, selected, free_text, skipped, answered_by, answered_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (project_id, round, question) DO UPDATE SET
    selected = excluded.selected, free_text = excluded.free_text, skipped = excluded.skipped,
    answered_by = excluded.answered_by, answered_at = excluded.answered_at
RETURNING *;

-- name: UpdateInterviewAnswer :one
UPDATE interview_answers SET selected = ?, free_text = ?, skipped = ?, answered_by = ?, answered_at = ?
WHERE project_id = ? AND round = ? AND question = ?
RETURNING *;

-- name: DeleteInterviewAnswer :execrows
DELETE FROM interview_answers WHERE project_id = ? AND round = ? AND question = ?;

-- name: LastInterviewRound :one
SELECT MAX(round) FROM interview_answers WHERE project_id = ?;

-- name: InsertInterviewAnswer :exec
INSERT INTO interview_answers (id, workspace_id, project_id, round, question, options, multi_select, why, selected, free_text, skipped, answered_by, answered_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
