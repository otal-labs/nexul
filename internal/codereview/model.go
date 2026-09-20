package codereview

import "time"

// Status is derived from git provider webhook events (ADR 0034); the latest review wins.
type Status string

const (
	StatusPending          Status = "pending"
	StatusApproved         Status = "approved"
	StatusChangesRequested Status = "changes_requested"
	StatusMerged           Status = "merged"
	StatusClosed           Status = "closed"
)

// CodeReview is one record per PR (repository + PR number), not per ticket (ADR 0034).
type CodeReview struct {
	ID        string    `json:"id"`
	PRNumber  int       `json:"pr_number"`
	Repo      string    `json:"repo"`
	Status    Status    `json:"status"`
	Reviewer  string    `json:"reviewer"`
	CreatedAt time.Time `json:"created_at"`
}
