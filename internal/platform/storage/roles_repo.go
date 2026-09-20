package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
	"github.com/otal-labs/nexul/internal/roles"
)

var _ roles.Repo = (*RolesRepo)(nil)

// RolesRepo persists the roles domain's Role entity (ticket 08).
type RolesRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

func (r *RolesRepo) Create(ctx context.Context, role *roles.Role) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		err := r.q.WithTx(tx).CreateRole(ctx, sqlcgen.CreateRoleParams{
			ID: role.ID, WorkspaceID: role.WorkspaceID, Name: role.Name,
			Permissions: setJSON(role.Permissions), IsOwnerRole: int64(boolInt(role.IsOwnerRole)),
			CreatedAt: role.CreatedAt.Unix(), UpdatedAt: role.UpdatedAt.Unix(),
		})
		if err != nil {
			return fmt.Errorf("insert role %s: %w", role.ID, classifyWriteErr(err))
		}
		return nil
	})
}

func (r *RolesRepo) Get(ctx context.Context, id string) (*roles.Role, error) {
	row, err := r.q.GetRole(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get role %s: %w", id, notFoundIfNoRows(err))
	}
	return toRole(row)
}

func (r *RolesRepo) List(ctx context.Context, workspaceID string) ([]*roles.Role, error) {
	rows, err := r.q.ListRoles(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list roles for workspace %s: %w", workspaceID, err)
	}
	var out []*roles.Role
	for _, row := range rows {
		role, err := toRole(row)
		if err != nil {
			return nil, err
		}
		out = append(out, role)
	}
	return out, nil
}

func (r *RolesRepo) Update(ctx context.Context, role *roles.Role) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).UpdateRole(ctx, sqlcgen.UpdateRoleParams{
			Name: role.Name, Permissions: setJSON(role.Permissions), UpdatedAt: role.UpdatedAt.Unix(), ID: role.ID,
		})
		if err != nil {
			return fmt.Errorf("update role %s: %w", role.ID, classifyWriteErr(err))
		}
		if n == 0 {
			return fmt.Errorf("update role %s: %w", role.ID, apperrs.ErrNotFound)
		}
		return nil
	})
}

func (r *RolesRepo) Delete(ctx context.Context, id string) error {
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		n, err := r.q.WithTx(tx).DeleteRole(ctx, id)
		if err != nil {
			return fmt.Errorf("delete role %s: %w", id, classifyWriteErr(err))
		}
		if n == 0 {
			return fmt.Errorf("delete role %s: %w", id, apperrs.ErrNotFound)
		}
		return nil
	})
}

func toRole(row sqlcgen.Role) (*roles.Role, error) {
	perms, err := parseSet(row.Permissions)
	if err != nil {
		return nil, fmt.Errorf("decode permissions for role %s: %w", row.ID, err)
	}
	return &roles.Role{
		ID:          row.ID,
		WorkspaceID: row.WorkspaceID,
		Name:        row.Name,
		Permissions: perms,
		IsOwnerRole: row.IsOwnerRole != 0,
		CreatedAt:   time.Unix(row.CreatedAt, 0).UTC(),
		UpdatedAt:   time.Unix(row.UpdatedAt, 0).UTC(),
	}, nil
}
