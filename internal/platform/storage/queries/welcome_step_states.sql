-- name: ListWelcomeStepStates :many
SELECT * FROM welcome_step_states WHERE user_id IN (SELECT value FROM json_each(sqlc.arg(user_ids)));

-- name: SetWelcomeStepSkipped :exec
INSERT INTO welcome_step_states (user_id, step, skipped_at) VALUES (?, ?, ?)
ON CONFLICT (user_id, step) DO UPDATE SET skipped_at = excluded.skipped_at;

-- name: GiveWelcomeStep :exec
INSERT INTO welcome_step_states (user_id, step, given_at) VALUES (?, ?, ?)
ON CONFLICT (user_id, step) DO UPDATE SET skipped_at = NULL, given_at = COALESCE(excluded.given_at, welcome_step_states.given_at);

-- name: MarkWelcomeStepDone :exec
INSERT INTO welcome_step_states (user_id, step, done_at) VALUES (?, ?, ?)
ON CONFLICT (user_id, step) DO UPDATE SET done_at = COALESCE(welcome_step_states.done_at, excluded.done_at);
