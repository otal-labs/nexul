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
	var rows []sqlcgen.PermissionOverwrite
	var err error
	if len(resourceIDs) == 1 {
		rows, err = r.hot.ListOverwriteRulesInResource(ctx, sqlcgen.ListOverwriteRulesInResourceParams{
			ResourceType: resourceType, ResourceID: resourceIDs[0], UserID: nullString(userID),
		})
	}
	if len(resourceIDs) > 1 {
		rows, err = r.q.ListOverwriteRules(ctx, sqlcgen.ListOverwriteRulesParams{
			ResourceType: resourceType, UserID: nullString(userID), ResourceIds: resourceIDs,
		})
	}
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

// OwnRules returns the rules naming userID on each of projectIDs that has one, keyed by project id, in one read: an
// index lookup for one project, the person's rules on every project for a list.
func (r *AccessRepo) OwnRules(ctx context.Context, userID string, projectIDs []string) (map[string]*access.Overwrite, error) {
	var rows []sqlcgen.PermissionOverwrite
	var err error
	if len(projectIDs) == 1 {
		rows, err = r.hot.ListOwnOverwritesInProject(ctx, sqlcgen.ListOwnOverwritesInProjectParams{UserID: nullString(userID), ResourceID: projectIDs[0]})
	}
	if len(projectIDs) > 1 {
		rows, err = r.q.ListOwnOverwrites(ctx, nullString(userID))
	}
	if err != nil {
		return nil, fmt.Errorf("get the project rules of %s: %w", userID, err)
	}
	out := make(map[string]*access.Overwrite, len(rows))
	for _, row := range rows {
		ow, err := toOverwrite(row)
		if err != nil {
			return nil, err
		}
		if slices.Contains(projectIDs, ow.ResourceID) {
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

// StackRuleTurns decides userID's stack rules for action in SQL and hands back only the stacks whose answer turns.
func (r *AccessRepo) StackRuleTurns(ctx context.Context, userID string, action permissions.Action, ruled, readable []string) (allowed, denied []string, err error) {
	rows, err := r.hot.ListStackRuleTurns(ctx, sqlcgen.ListStackRuleTurnsParams{
		QuotedAction: `"` + string(action) + `"`, UserID: nullString(userID), Ruled: idsJSON(ruled), Readable: idsJSON(readable),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("stack rules of %s for %s: %w", userID, action, err)
	}
	allowed, denied = []string{}, []string{}
	for _, row := range rows {
		if held, _ := row.Held.(int64); held == 1 {
			allowed = append(allowed, row.StackID)
			continue
		}
		denied = append(denied, row.StackID)
	}
	return allowed, denied, nil
}

// DocRules returns the rules naming userID or a role they hold on each of docIDs and on its folder, in one read.
func (r *AccessRepo) DocRules(ctx context.Context, docIDs []string, userID string) (map[string]*access.DocRules, error) {
	if len(docIDs) == 0 {
		return map[string]*access.DocRules{}, nil
	}
	rows, err := r.q.ListDocChainRules(ctx, sqlcgen.ListDocChainRulesParams{DocIds: idsJSON(docIDs), UserID: nullString(userID)})
	if err != nil {
		return nil, fmt.Errorf("get doc rules for %s: %w", userID, err)
	}
	byDoc := map[string][]sqlcgen.PermissionOverwrite{}
	for _, row := range rows {
		byDoc[row.DocID] = append(byDoc[row.DocID], sqlcgen.PermissionOverwrite{
			ResourceType: row.ResourceType, ResourceID: row.ResourceID, Target: row.Target, UserID: row.UserID, RoleID: row.RoleID,
			Allow: row.Allow, Deny: row.Deny, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		})
	}
	out := make(map[string]*access.DocRules, len(byDoc))
	for docID, owned := range byDoc {
		rules, err := toRules(owned)
		if err != nil {
			return nil, err
		}
		chain := &access.DocRules{Doc: rules[docID]}
		for id, r := range rules {
			if id != docID {
				chain.Folder = r
			}
		}
		out[docID] = chain
	}
	return out, nil
}

// DocRuleTurns decides userID's folder and doc rules for action in SQL and hands back only those that turn the answer.
func (r *AccessRepo) DocRuleTurns(ctx context.Context, userID string, action permissions.Action, ruled, readable []string) (permissions.DocReach, error) {
	rows, err := r.q.ListDocRuleTurns(ctx, sqlcgen.ListDocRuleTurnsParams{
		QuotedAction: `"` + string(action) + `"`, UserID: nullString(userID), Ruled: idsJSON(ruled), Readable: idsJSON(readable),
	})
	if err != nil {
		return permissions.DocReach{}, fmt.Errorf("doc rules of %s for %s: %w", userID, action, err)
	}
	out := permissions.DocReach{AllowedFolders: []string{}, DeniedFolders: []string{}, Allowed: []string{}, Denied: []string{}}
	for _, row := range rows {
		folder := row.Kind == "folder"
		into := &out.Denied
		if folder {
			into = &out.DeniedFolders
		}
		if state, _ := row.State.(int64); state == 1 {
			into = &out.Allowed
			if folder {
				into = &out.AllowedFolders
			}
		}
		*into = append(*into, row.ResourceID)
	}
	return out, nil
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

func (r *AccessRepo) GetRole(ctx context.Context, resourceType, resourceID, roleID string) (*access.Overwrite, error) {
	row, err := r.q.GetRoleOverwrite(ctx, sqlcgen.GetRoleOverwriteParams{
		ResourceType: resourceType, ResourceID: resourceID, RoleID: nullString(roleID),
	})
	if err != nil {
		return nil, fmt.Errorf("get role overwrite %s/%s/%s: %w", resourceType, resourceID, roleID, notFoundIfNoRows(err))
	}
	return toOverwrite(row)
}

func (r *AccessRepo) Set(ctx context.Context, resourceType, resourceID, userID string, allow, deny permissions.Set, events ...eventbus.OutboxEvent) error {
	return r.set(ctx, resourceType+"/"+resourceID+"/"+userID, allow, deny, events, func(q *sqlcgen.Queries) (int64, error) {
		return q.DeleteOverwrite(ctx, sqlcgen.DeleteOverwriteParams{ResourceType: resourceType, ResourceID: resourceID, UserID: nullString(userID)})
	}, func(q *sqlcgen.Queries) (int64, error) {
		return q.UpsertOverwrite(ctx, sqlcgen.UpsertOverwriteParams{
			ResourceType: resourceType, ResourceID: resourceID, UserID: nullString(userID),
			Allow: setJSON(allow), Deny: setJSON(deny), Now: time.Now().Unix(),
		})
	})
}

func (r *AccessRepo) SetRole(ctx context.Context, resourceType, resourceID, roleID string, allow, deny permissions.Set, events ...eventbus.OutboxEvent) error {
	return r.set(ctx, resourceType+"/"+resourceID+"/role "+roleID, allow, deny, events, func(q *sqlcgen.Queries) (int64, error) {
		return q.DeleteRoleOverwrite(ctx, sqlcgen.DeleteRoleOverwriteParams{ResourceType: resourceType, ResourceID: resourceID, RoleID: nullString(roleID)})
	}, func(q *sqlcgen.Queries) (int64, error) {
		return q.UpsertRoleOverwrite(ctx, sqlcgen.UpsertRoleOverwriteParams{
			ResourceType: resourceType, ResourceID: resourceID, RoleID: nullString(roleID),
			Allow: setJSON(allow), Deny: setJSON(deny), Now: time.Now().Unix(),
		})
	})
}

// set upserts one rule, or deletes it when both sets are empty, with its events in the same transaction.
func (r *AccessRepo) set(ctx context.Context, name string, allow, deny permissions.Set, events []eventbus.OutboxEvent, del, upsert func(*sqlcgen.Queries) (int64, error)) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		if len(allow) == 0 && len(deny) == 0 {
			n, err := del(q)
			if err != nil {
				return fmt.Errorf("delete overwrite %s: %w", name, err)
			}
			if n == 0 {
				return fmt.Errorf("delete overwrite %s: %w", name, apperrs.ErrNotFound)
			}
			return insertOutboxRows(ctx, tx, events)
		}
		n, err := upsert(q)
		if err != nil {
			return fmt.Errorf("set overwrite %s: %w", name, classifyWriteErr(err))
		}
		if n == 0 {
			return fmt.Errorf("set overwrite %s: %w", name, apperrs.ErrConflict)
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
