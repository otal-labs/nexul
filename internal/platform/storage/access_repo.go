package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/otal-labs/nexul/internal/access"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

var _ access.Repo = (*AccessRepo)(nil)

// AccessRepo persists permission_overwrites: the rules for a person or a role on an entity of any resource type.
type AccessRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
	// hot runs the one-project reads every single check and live frame makes as prepared statements.
	hot *sqlcgen.Queries
}

func (r *AccessRepo) Get(ctx context.Context, resourceType, resourceID, userID string) (*access.Overwrite, error) {
	row, err := r.q.GetOverwrite(ctx, sqlcgen.GetOverwriteParams{
		ResourceType: resourceType, ResourceID: resourceID, UserID: nullString(userID),
	})
	if err != nil {
		return nil, fmt.Errorf("get overwrite %s/%s/%s: %w", resourceType, resourceID, userID, notFoundIfNoRows(err))
	}
	return toOverwrite(row)
}

// Rules returns the rules naming userID or a role they hold on each of resourceIDs that has any, in one read.
func (r *AccessRepo) Rules(ctx context.Context, resourceType string, resourceIDs []string, userID string) (map[string]*access.Rules, error) {
	if len(resourceIDs) == 0 {
		return map[string]*access.Rules{}, nil
	}
	rows, err := r.q.ListOverwriteRules(ctx, sqlcgen.ListOverwriteRulesParams{
		ResourceType: resourceType, UserID: nullString(userID), ResourceIds: resourceIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("get %s rules for %s: %w", resourceType, userID, err)
	}
	return toRules(rows)
}

// RulesOfType returns every rule on a resourceType naming userID or a role they hold, in one read.
func (r *AccessRepo) RulesOfType(ctx context.Context, resourceType, userID string) (map[string]*access.Rules, error) {
	rows, err := r.q.ListOverwriteRulesOfType(ctx, sqlcgen.ListOverwriteRulesOfTypeParams{ResourceType: resourceType, UserID: nullString(userID)})
	if err != nil {
		return nil, fmt.Errorf("list %s rules of %s: %w", resourceType, userID, err)
	}
	return toRules(rows)
}

// OwnRules returns the rules naming userID on workspaceID and on each of projectIDs that has one, keyed by resource
// id, in one read: two index lookups for one project, the person's rules on every workspace and project for a list.
func (r *AccessRepo) OwnRules(ctx context.Context, userID, workspaceID string, projectIDs []string) (map[string]*access.Overwrite, error) {
	var rows []sqlcgen.PermissionOverwrite
	var err error
	if len(projectIDs) <= 1 {
		rows, err = r.hot.ListOwnOverwritesInProject(ctx, sqlcgen.ListOwnOverwritesInProjectParams{UserID: nullString(userID), WorkspaceID: workspaceID, ProjectID: firstOf(projectIDs)})
	}
	if len(projectIDs) > 1 {
		rows, err = r.q.ListOwnOverwrites(ctx, nullString(userID))
	}
	if err != nil {
		return nil, fmt.Errorf("get the rules of %s in workspace %s: %w", userID, workspaceID, err)
	}
	asked := append([]string{workspaceID}, projectIDs...)
	out := make(map[string]*access.Overwrite, len(rows))
	for _, row := range rows {
		ow, err := toOverwrite(row)
		if err != nil {
			return nil, err
		}
		if slices.Contains(asked, ow.ResourceID) {
			out[ow.ResourceID] = ow
		}
	}
	return out, nil
}

// RoleRules returns the rules for roleIDs on each of resourceIDs that has any, keyed by resource id in role order, in
// one read: by the resource for one, by the roles for a list.
func (r *AccessRepo) RoleRules(ctx context.Context, resourceType string, resourceIDs, roleIDs []string) (map[string][]*access.Overwrite, error) {
	var rows []sqlcgen.PermissionOverwrite
	var err error
	if len(resourceIDs) == 1 {
		rows, err = r.hot.ListRoleOverwritesInResource(ctx, sqlcgen.ListRoleOverwritesInResourceParams{ResourceType: resourceType, ResourceID: resourceIDs[0], RoleIds: idsJSON(roleIDs)})
	}
	if len(resourceIDs) > 1 {
		rows, err = r.q.ListRoleOverwrites(ctx, sqlcgen.ListRoleOverwritesParams{RoleIds: idsJSON(roleIDs), ResourceType: resourceType})
	}
	if err != nil {
		return nil, fmt.Errorf("get the %s rules for roles: %w", resourceType, err)
	}
	out := map[string][]*access.Overwrite{}
	for _, row := range rows {
		ow, err := toOverwrite(row)
		if err != nil {
			return nil, err
		}
		if ow.Target == access.TargetRole && slices.Contains(resourceIDs, ow.ResourceID) {
			out[ow.ResourceID] = append(out[ow.ResourceID], ow)
		}
	}
	for _, rules := range out {
		slices.SortFunc(rules, func(a, b *access.Overwrite) int { return strings.Compare(a.RoleID, b.RoleID) })
	}
	return out, nil
}

func firstOf(ids []string) string {
	if len(ids) == 0 {
		return ""
	}
	return ids[0]
}

// toRules groups rows by resource, each entity's role rules in their target order.
func toRules(rows []sqlcgen.PermissionOverwrite) (map[string]*access.Rules, error) {
	out := map[string]*access.Rules{}
	for _, row := range rows {
		ow, err := toOverwrite(row)
		if err != nil {
			return nil, err
		}
		rules := out[ow.ResourceID]
		if rules == nil {
			rules = &access.Rules{}
			out[ow.ResourceID] = rules
		}
		if ow.Target == access.TargetRole {
			rules.Roles = append(rules.Roles, ow)
			continue
		}
		rules.Person = ow
	}
	for _, rules := range out {
		slices.SortFunc(rules.Roles, func(a, b *access.Overwrite) int { return strings.Compare(a.RoleID, b.RoleID) })
	}
	return out, nil
}

func (r *AccessRepo) ListByResource(ctx context.Context, resourceType, resourceID string) ([]*access.Overwrite, error) {
	rows, err := r.q.ListOverwritesByResource(ctx, sqlcgen.ListOverwritesByResourceParams{
		ResourceType: resourceType, ResourceID: resourceID,
	})
	if err != nil {
		return nil, fmt.Errorf("list overwrites %s/%s: %w", resourceType, resourceID, err)
	}
	return toOverwrites(rows)
}

func (r *AccessRepo) Set(ctx context.Context, resourceType, resourceID, userID string, allow, deny permissions.Set, events ...eventbus.OutboxEvent) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		if len(allow) == 0 && len(deny) == 0 {
			n, err := q.DeleteOverwrite(ctx, sqlcgen.DeleteOverwriteParams{
				ResourceType: resourceType, ResourceID: resourceID, UserID: nullString(userID),
			})
			if err != nil {
				return fmt.Errorf("delete overwrite %s/%s/%s: %w", resourceType, resourceID, userID, err)
			}
			if n == 0 {
				return fmt.Errorf("delete overwrite %s/%s/%s: %w", resourceType, resourceID, userID, apperrs.ErrNotFound)
			}
			return insertOutboxRows(ctx, tx, events)
		}
		n, err := q.UpsertOverwrite(ctx, sqlcgen.UpsertOverwriteParams{
			ResourceType: resourceType, ResourceID: resourceID, UserID: nullString(userID),
			Allow: setJSON(allow), Deny: setJSON(deny), Now: time.Now().Unix(),
		})
		if err != nil {
			return fmt.Errorf("set overwrite %s/%s/%s: %w", resourceType, resourceID, userID, classifyWriteErr(err))
		}
		if n == 0 {
			return fmt.Errorf("set overwrite %s/%s/%s: %w", resourceType, resourceID, userID, apperrs.ErrConflict)
		}
		return insertOutboxRows(ctx, tx, events)
	})
}

func (r *AccessRepo) DeleteByResource(ctx context.Context, resourceType, resourceID string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		if err := r.q.WithTx(tx).DeleteOverwritesByResource(ctx, sqlcgen.DeleteOverwritesByResourceParams{
			ResourceType: resourceType, ResourceID: resourceID,
		}); err != nil {
			return fmt.Errorf("delete overwrites %s/%s: %w", resourceType, resourceID, err)
		}
		return nil
	})
}

func (r *AccessRepo) HasAllowAny(ctx context.Context, resourceType, userID string, action permissions.Action) (bool, error) {
	// The quotes make the substring match exact: "docs:read" never matches inside "docs:readx".
	n, err := r.q.CountOverwriteAllowAny(ctx, sqlcgen.CountOverwriteAllowAnyParams{
		ResourceType: resourceType, UserID: nullString(userID), QuotedAction: `"` + string(action) + `"`,
	})
	if err != nil {
		return false, fmt.Errorf("has allow %s for %s/%s: %w", action, resourceType, userID, err)
	}
	return n > 0, nil
}

func toOverwrite(row sqlcgen.PermissionOverwrite) (*access.Overwrite, error) {
	allow, err := parseSet(row.Allow)
	if err != nil {
		return nil, fmt.Errorf("decode allow for %s/%s/%s: %w", row.ResourceType, row.ResourceID, row.Target, err)
	}
	deny, err := parseSet(row.Deny)
	if err != nil {
		return nil, fmt.Errorf("decode deny for %s/%s/%s: %w", row.ResourceType, row.ResourceID, row.Target, err)
	}
	return &access.Overwrite{
		ResourceType: row.ResourceType,
		ResourceID:   row.ResourceID,
		Target:       row.Target,
		UserID:       row.UserID.String,
		RoleID:       row.RoleID.String,
		Allow:        allow,
		Deny:         deny,
		CreatedAt:    time.Unix(row.CreatedAt, 0).UTC(),
		UpdatedAt:    time.Unix(row.UpdatedAt, 0).UTC(),
	}, nil
}

func toOverwrites(rows []sqlcgen.PermissionOverwrite) ([]*access.Overwrite, error) {
	var out []*access.Overwrite
	for _, row := range rows {
		ow, err := toOverwrite(row)
		if err != nil {
			return nil, err
		}
		out = append(out, ow)
	}
	return out, nil
}

// setJSON is the stored form of a permission set; marshalling a string slice cannot fail.
func setJSON(s permissions.Set) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func parseSet(text string) (permissions.Set, error) {
	var s permissions.Set
	if err := json.Unmarshal([]byte(text), &s); err != nil {
		return nil, err
	}
	return s, nil
}
