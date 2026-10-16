-- name: GetStackCrashTicket :one
SELECT ticket_id FROM stack_crash_tickets WHERE stack_id = ? AND subject = ?;

-- name: SetStackCrashTicket :exec
INSERT INTO stack_crash_tickets (stack_id, subject, ticket_id) VALUES (?, ?, ?)
ON CONFLICT (stack_id, subject) DO UPDATE SET ticket_id = excluded.ticket_id;
