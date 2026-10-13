package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
	"github.com/otal-labs/nexul/internal/platform/welcomestep"
	"github.com/otal-labs/nexul/internal/welcome"
)

var _ welcome.Repo = (*WelcomeRepo)(nil)

// WelcomeRepo keeps each person's Welcome wizard step records (welcome_step_states).
type WelcomeRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

func (r *WelcomeRepo) Records(ctx context.Context, userIDs []string) ([]welcome.Record, error) {
	rows, err := r.q.ListWelcomeStepStates(ctx, idsJSON(userIDs))
	if err != nil {
		return nil, fmt.Errorf("list welcome step states: %w", err)
	}
	out := make([]welcome.Record, 0, len(rows))
	for _, row := range rows {
		out = append(out, welcome.Record{UserID: row.UserID, Step: row.Step, SkippedAt: unixPtrFromNull(row.SkippedAt),
			GivenAt: unixPtrFromNull(row.GivenAt), DoneAt: unixPtrFromNull(row.DoneAt)})
	}
	return out, nil
}

func (r *WelcomeRepo) SetSkipped(ctx context.Context, userID, step string, at *time.Time, events ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).SetWelcomeStepSkipped(ctx, sqlcgen.SetWelcomeStepSkippedParams{UserID: userID, Step: step, SkippedAt: nullUnixPtr(at)})
		if err != nil {
			return fmt.Errorf("set skip of %s for %s: %w", step, userID, classifyWriteErr(err))
		}
		return insertOutboxRows(ctx, tx, events)
	})
}

func (r *WelcomeRepo) Give(ctx context.Context, gifts []welcome.Gift, events ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		for _, g := range gifts {
			err := q.GiveWelcomeStep(ctx, sqlcgen.GiveWelcomeStepParams{UserID: g.UserID, Step: g.Step, GivenAt: nullUnixPtr(g.GivenAt)})
			if err != nil {
				return fmt.Errorf("give %s to %s: %w", g.Step, g.UserID, classifyWriteErr(err))
			}
		}
		return insertOutboxRows(ctx, tx, events)
	})
}

func (r *WelcomeRepo) MarkDone(ctx context.Context, userID, step string, at time.Time, events ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).MarkWelcomeStepDone(ctx, sqlcgen.MarkWelcomeStepDoneParams{UserID: userID, Step: step, DoneAt: nullUnixPtr(&at)})
		if err != nil {
			return fmt.Errorf("mark %s done for %s: %w", step, userID, classifyWriteErr(err))
		}
		return insertOutboxRows(ctx, tx, events)
	})
}

func (r *WelcomeRepo) Announce(ctx context.Context, events ...eventbus.OutboxEvent) error {
	if len(events) == 0 {
		return nil
	}
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		return insertOutboxRows(ctx, tx, events)
	})
}

// peopleSet turns the ids a seam's query returned into the set welcome reads.
func peopleSet(ids []string, err error) (map[string]bool, error) {
	if err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(ids))
	for _, id := range ids {
		out[id] = true
	}
	return out, nil
}

// PeopleWithWorkingLink is the welcome seam for Connect GitHub: a link that still lists the person's repositories.
func (r *GitHubUserLinksRepo) PeopleWithWorkingLink(ctx context.Context, userIDs []string) (map[string]bool, error) {
	return peopleSet(r.q.ListUsersWithWorkingGitHubLink(ctx, idsJSON(userIDs)))
}

// PeopleWithReadyComputer is the welcome seam for Your computer: a paired computer whose setup is confirmed.
func (r *PairingRepo) PeopleWithReadyComputer(ctx context.Context, userIDs []string) (map[string]bool, error) {
	return peopleSet(r.q.ListUsersWithReadyComputer(ctx, idsJSON(userIDs)))
}

// PeopleWithConnectedDeployRunner is the welcome seam for Add a deploy server: a connected deploy runner they enrolled.
func (r *RunnersRepo) PeopleWithConnectedDeployRunner(ctx context.Context, userIDs []string) (map[string]bool, error) {
	return peopleSet(r.q.ListUsersWithConnectedDeployRunner(ctx, idsJSON(userIDs)))
}

// PeopleWithFirstRunDone is the welcome seam for the first run, kept as first_login_done.
func (r *UsersRepo) PeopleWithFirstRunDone(ctx context.Context, userIDs []string) (map[string]bool, error) {
	return peopleSet(r.q.ListUsersWithFirstRunDone(ctx, idsJSON(userIDs)))
}

// PeopleInAProject is the welcome seam for Your projects: in at least one project, as RequireProject decides it.
func (r *AccessRepo) PeopleInAProject(ctx context.Context, userIDs []string) (map[string]bool, error) {
	return peopleSet(r.q.ListUsersInAProject(ctx, idsJSON(userIDs)))
}

// WelcomeStepsOfPeople is the union of the welcome steps each person's roles give, the Owner role's from code.
func (r *RolesRepo) WelcomeStepsOfPeople(ctx context.Context, userIDs []string) (map[string][]string, error) {
	rows, err := r.q.ListRoleWelcomeStepsForUsers(ctx, idsJSON(userIDs))
	if err != nil {
		return nil, fmt.Errorf("list welcome steps of people: %w", err)
	}
	out := map[string][]string{}
	for _, row := range rows {
		steps, err := roleSteps(row.WelcomeSteps, row.IsOwnerRole)
		if err != nil {
			return nil, err
		}
		out[row.UserID] = union(out[row.UserID], steps)
	}
	return out, nil
}

// WelcomeStepsOfRoles is the union of the welcome steps roleIDs give.
func (r *RolesRepo) WelcomeStepsOfRoles(ctx context.Context, roleIDs []string) ([]string, error) {
	rows, err := r.q.ListRoleWelcomeStepsForRoles(ctx, idsJSON(roleIDs))
	if err != nil {
		return nil, fmt.Errorf("list welcome steps of roles: %w", err)
	}
	var out []string
	for _, row := range rows {
		steps, err := roleSteps(row.WelcomeSteps, row.IsOwnerRole)
		if err != nil {
			return nil, err
		}
		out = union(out, steps)
	}
	return out, nil
}

// Holders lists who holds roleID, in any workspace.
func (r *RolesRepo) Holders(ctx context.Context, roleID string) ([]string, error) {
	ids, err := r.q.ListRoleHolders(ctx, roleID)
	if err != nil {
		return nil, fmt.Errorf("list holders of role %s: %w", roleID, err)
	}
	return ids, nil
}

func roleSteps(text string, owner int64) ([]string, error) {
	if owner != 0 {
		return welcomestep.Owner, nil
	}
	var steps []string
	if err := json.Unmarshal([]byte(text), &steps); err != nil {
		return nil, fmt.Errorf("decode welcome steps: %w", err)
	}
	return steps, nil
}

func union(a, b []string) []string {
	for _, s := range b {
		if !slices.Contains(a, s) {
			a = append(a, s)
		}
	}
	return a
}
