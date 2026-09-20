package gitprovider

import (
	"context"
	"fmt"

	apperrors "github.com/otal-labs/nexul/internal/platform/errors"
)

// ListPRs lists pull requests for a repository, validating caller input first
// (ADR 0019: one use-case per capability, shared by the HTTP and MCP adapters).
func ListPRs(ctx context.Context, p GitProvider, owner, name string, opts PROpts) ([]*PR, error) {
	if owner == "" || name == "" {
		return nil, fmt.Errorf("%w: owner and repo are required", apperrors.ErrInvalid)
	}
	return p.ListPRs(ctx, owner, name, opts)
}

// GetPR fetches a single pull request, validating caller input first.
func GetPR(ctx context.Context, p GitProvider, owner, name string, number int) (*PR, error) {
	if owner == "" || name == "" {
		return nil, fmt.Errorf("%w: owner and repo are required", apperrors.ErrInvalid)
	}
	if number < 1 {
		return nil, fmt.Errorf("%w: number must be a positive integer", apperrors.ErrInvalid)
	}
	return p.GetPR(ctx, owner, name, number)
}

// GetRepo fetches a repository, validating caller input first.
func GetRepo(ctx context.Context, p GitProvider, owner, name string) (*Repo, error) {
	if owner == "" || name == "" {
		return nil, fmt.Errorf("%w: owner and repo are required", apperrors.ErrInvalid)
	}
	return p.GetRepo(ctx, owner, name)
}
