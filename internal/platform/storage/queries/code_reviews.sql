-- name: CreateCodeReview :exec
INSERT INTO code_reviews (id, pr_number, repo, status, reviewer, created_at) VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(repo, pr_number) DO NOTHING;

-- name: GetCodeReview :one
SELECT * FROM code_reviews WHERE id = ?;

-- name: UpdateCodeReviewStatus :execrows
UPDATE code_reviews SET status = ?, reviewer = ? WHERE id = ?;

-- name: ListCodeReviewsByTicket :many
SELECT r.* FROM code_reviews r
JOIN ticket_pr_links l ON l.pr_owner || '/' || l.pr_repo = r.repo AND l.pr_number = r.pr_number
WHERE l.ticket_id = ? ORDER BY r.created_at;

-- name: ListCodeReviewsByPR :many
SELECT * FROM code_reviews WHERE repo = ? AND pr_number = ? ORDER BY created_at;
