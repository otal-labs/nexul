package gitprovider

import (
	"context"
	"fmt"
	"strings"

	apperrors "github.com/otal-labs/nexul/internal/platform/errors"
)

// ChangeDoc is the doc a changed ticket came from.
type ChangeDoc struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// ChangeBug is a bug found after the ticket was done, linked to it by found-in (ADR 0064).
type ChangeBug struct {
	ID    string `json:"id"`
	Key   string `json:"key"`
	Title string `json:"title"`
	Done  bool   `json:"done"`
}

// ChangeTicket is one ticket a pull request is linked to, with the doc it came from and the bugs found after it.
type ChangeTicket struct {
	ID        string      `json:"id"`
	Key       string      `json:"key"`
	Title     string      `json:"title"`
	ProjectID string      `json:"project_id"`
	Done      bool        `json:"done"`
	Doc       *ChangeDoc  `json:"doc"`
	BugsFound []ChangeBug `json:"bugs_found"`
}

// ChangeContext answers "why does this code exist": the PR, its tickets, and the decisions-log entries citing them.
type ChangeContext struct {
	Owner     string         `json:"owner"`
	Repo      string         `json:"repo"`
	PR        *PR            `json:"pr"`
	Tickets   []ChangeTicket `json:"tickets"`
	Decisions []string       `json:"decisions"`
}

// ChangeContextReader is the change context's seam onto tickets, docs, and memories (ADR 0017).
type ChangeContextReader interface {
	TicketsForPR(ctx context.Context, owner, repo string, number int) ([]ChangeTicket, error)
	// DecisionEntries returns the project's decisions-log entries naming any of refs.
	DecisionEntries(ctx context.Context, projectID string, refs []string) ([]string, error)
}

// ChangeRef names the change to explain: a PR number, or a commit SHA resolved to the PR that contains it.
type ChangeRef struct {
	Owner  string
	Repo   string
	Number int
	Commit string
}

// GetChangeContext walks a change back to its why: PR, tickets, their docs and later bugs, and the decisions citing them.
func GetChangeContext(ctx context.Context, p GitProvider, r ChangeContextReader, ref ChangeRef) (*ChangeContext, error) {
	pr, err := resolveChangePR(ctx, p, ref)
	if err != nil {
		return nil, err
	}
	tickets, err := r.TicketsForPR(ctx, ref.Owner, ref.Repo, pr.Number)
	if err != nil {
		return nil, fmt.Errorf("tickets for PR %s/%s#%d: %w", ref.Owner, ref.Repo, pr.Number, err)
	}
	refsByProject := map[string][]string{}
	var projects []string
	for _, t := range tickets {
		if _, seen := refsByProject[t.ProjectID]; !seen {
			projects = append(projects, t.ProjectID)
		}
		refsByProject[t.ProjectID] = append(refsByProject[t.ProjectID], t.Key, "/tickets/"+t.ID)
	}
	decisions := []string{}
	for _, projectID := range projects {
		entries, err := r.DecisionEntries(ctx, projectID, refsByProject[projectID])
		if err != nil {
			return nil, fmt.Errorf("decisions for project %s: %w", projectID, err)
		}
		decisions = append(decisions, entries...)
	}
	if tickets == nil {
		tickets = []ChangeTicket{}
	}
	return &ChangeContext{Owner: ref.Owner, Repo: ref.Repo, PR: pr, Tickets: tickets, Decisions: decisions}, nil
}

func resolveChangePR(ctx context.Context, p GitProvider, ref ChangeRef) (*PR, error) {
	if ref.Owner == "" || ref.Repo == "" {
		return nil, fmt.Errorf("%w: owner and repo are required", apperrors.ErrInvalid)
	}
	commit := strings.TrimSpace(ref.Commit)
	if ref.Number > 0 {
		return p.GetPR(ctx, ref.Owner, ref.Repo, ref.Number)
	}
	if commit == "" {
		return nil, fmt.Errorf("%w: a PR number or a commit SHA is required", apperrors.ErrInvalid)
	}
	prs, err := p.PRsForCommit(ctx, ref.Owner, ref.Repo, commit)
	if err != nil {
		return nil, fmt.Errorf("pull requests for commit %s: %w", commit, err)
	}
	if len(prs) == 0 {
		return nil, fmt.Errorf("%w: no pull request contains commit %s", apperrors.ErrNotFound, commit)
	}
	return prs[0], nil
}
