package gitprovider

import (
	"context"
	"fmt"

	apperrors "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// Gate is the permission check a repository's pull requests pass: the caller needs the action in the workspace of
// the project the repository belongs to (ADR 0087).
type Gate interface {
	RequireRepo(ctx context.Context, owner, name string, action permissions.Action) error
}

func requireRead(ctx context.Context, g Gate, owner, name string) error {
	if g == nil {
		return permissions.Ungated(ctx)
	}
	return g.RequireRepo(ctx, owner, name, permissions.ReposRead)
}

// ListPRs lists pull requests for a repository, validating caller input first
// (ADR 0019: one use-case per capability, shared by the HTTP and MCP adapters).
func ListPRs(ctx context.Context, g Gate, p GitProvider, owner, name string, opts PROpts) ([]*PR, error) {
	if owner == "" || name == "" {
		return nil, fmt.Errorf("%w: owner and repo are required", apperrors.ErrInvalid)
	}
	if err := requireRead(ctx, g, owner, name); err != nil {
		return nil, err
	}
	return p.ListPRs(ctx, owner, name, opts)
}

// GetPR fetches a single pull request, validating caller input first.
func GetPR(ctx context.Context, g Gate, p GitProvider, owner, name string, number int) (*PR, error) {
	if owner == "" || name == "" {
		return nil, fmt.Errorf("%w: owner and repo are required", apperrors.ErrInvalid)
	}
	if number < 1 {
		return nil, fmt.Errorf("%w: number must be a positive integer", apperrors.ErrInvalid)
	}
	if err := requireRead(ctx, g, owner, name); err != nil {
		return nil, err
	}
	return p.GetPR(ctx, owner, name, number)
}

// GetRepo fetches a repository, validating caller input first.
func GetRepo(ctx context.Context, g Gate, p GitProvider, owner, name string) (*Repo, error) {
	if owner == "" || name == "" {
		return nil, fmt.Errorf("%w: owner and repo are required", apperrors.ErrInvalid)
	}
	if err := requireRead(ctx, g, owner, name); err != nil {
		return nil, err
	}
	return p.GetRepo(ctx, owner, name)
}
