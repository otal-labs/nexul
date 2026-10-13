package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/access"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

// Grid reads the Access page's source for workspaceID in six statements whatever its size.
func (r *AccessRepo) Grid(ctx context.Context, workspaceID string) (*access.GridSource, error) {
	setup, err := r.q.GetAccessSetup(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("get the access setup of workspace %s: %w", workspaceID, notFoundIfNoRows(err))
	}
	src := &access.GridSource{Grid: access.Grid{WorkspaceID: workspaceID, Setup: toGridSetup(setup)}}
	projects, err := r.q.ListAccessGridProjects(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list the projects of workspace %s: %w", workspaceID, err)
	}
	for _, p := range projects {
		src.Projects = append(src.Projects, access.GridProject{ID: p.ID, Name: p.Name, Prefix: p.Prefix})
	}
	roles, err := r.q.ListAccessGridRoles(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list the roles of workspace %s: %w", workspaceID, err)
	}
	src.Roles = make([]access.GridRole, 0, len(roles))
	for _, role := range roles {
		src.Roles = append(src.Roles, access.GridRole{ID: role.ID, Name: role.Name, Holders: int(role.Holders)})
	}
	channels, err := r.q.ListAccessGridChannels(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list the public channels of workspace %s: %w", workspaceID, err)
	}
	for _, c := range channels {
		src.Channels = append(src.Channels, access.GridChannel{ID: c.ID, Name: c.Name, Kind: c.Kind})
	}
	if src.MemberIDs, err = r.q.ListAccessGridMembers(ctx, workspaceID); err != nil {
		return nil, fmt.Errorf("list the members of workspace %s: %w", workspaceID, err)
	}
	rules, err := r.q.ListAccessGridRules(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list the project and channel rules of workspace %s: %w", workspaceID, err)
	}
	if src.Rules, err = toOverwrites(rules); err != nil {
		return nil, err
	}
	return src, nil
}

func toGridSetup(row sqlcgen.GetAccessSetupRow) access.GridSetup {
	setup := access.GridSetup{Finished: row.AccessSetupFinished != 0, FinishedBy: row.AccessSetupBy, Login: row.FinishedByLogin}
	if row.AccessSetupAt != 0 {
		at := time.Unix(row.AccessSetupAt, 0).UTC()
		setup.FinishedAt = &at
	}
	return setup
}

// gridRule is one rule as the bulk statements read it from a JSON array.
type gridRule struct {
	ResourceType string  `json:"resource_type"`
	ResourceID   string  `json:"resource_id"`
	Target       string  `json:"target"`
	UserID       *string `json:"user_id"`
	RoleID       *string `json:"role_id"`
	Allow        string  `json:"allow"`
	Deny         string  `json:"deny"`
}

// SaveGrid upserts the rules with a set and deletes the empty ones, each group in one statement, then marks the setup
// finished and writes the events, all in one transaction.
func (r *AccessRepo) SaveGrid(ctx context.Context, workspaceID string, rules []*access.Overwrite, finishedBy string, events ...eventbus.OutboxEvent) error {
	var kept, emptied []gridRule
	for _, ow := range rules {
		row := gridRule{ResourceType: ow.ResourceType, ResourceID: ow.ResourceID, Target: ow.Target, Allow: setJSON(ow.Allow), Deny: setJSON(ow.Deny)}
		if ow.Target == access.TargetRole {
			row.RoleID = &ow.RoleID
		}
		if ow.Target == access.TargetPerson {
			row.UserID = &ow.UserID
		}
		if len(ow.Allow)+len(ow.Deny) == 0 {
			emptied = append(emptied, row)
			continue
		}
		kept = append(kept, row)
	}
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		if len(kept) > 0 {
			if err := q.UpsertOverwritesFromJSON(ctx, sqlcgen.UpsertOverwritesFromJSONParams{Now: time.Now().Unix(), Rules: rulesJSON(kept)}); err != nil {
				return fmt.Errorf("set the rules of workspace %s: %w", workspaceID, classifyWriteErr(err))
			}
		}
		if len(emptied) > 0 {
			if err := q.DeleteOverwritesFromJSON(ctx, rulesJSON(emptied)); err != nil {
				return fmt.Errorf("delete the emptied rules of workspace %s: %w", workspaceID, err)
			}
		}
		if finishedBy != "" {
			if err := q.FinishAccessSetup(ctx, sqlcgen.FinishAccessSetupParams{AccessSetupBy: finishedBy, AccessSetupAt: time.Now().Unix(), ID: workspaceID}); err != nil {
				return fmt.Errorf("finish the access setup of workspace %s: %w", workspaceID, err)
			}
		}
		return insertOutboxRows(ctx, tx, events)
	})
}

// rulesJSON is rows as one JSON array; a slice of plain fields always encodes.
func rulesJSON(rows []gridRule) string {
	b, _ := json.Marshal(rows)
	return string(b)
}
