package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
)

// EventWorkspacesRepo finds an event entity's workspace; "" means it is gone or, for a stack or deploy, has no project.
type EventWorkspacesRepo struct {
	q *sqlcgen.Queries
}

func optionalWorkspace(id string, err error) (string, error) {
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("resolve event workspace: %w", err)
	}
	return id, nil
}

func (r *EventWorkspacesRepo) OfProject(ctx context.Context, id string) (string, error) {
	return optionalWorkspace(r.q.EventWorkspaceOfProject(ctx, id))
}

func (r *EventWorkspacesRepo) OfTicket(ctx context.Context, id string) (string, error) {
	return optionalWorkspace(r.q.EventWorkspaceOfTicket(ctx, id))
}

func (r *EventWorkspacesRepo) OfDoc(ctx context.Context, id string) (string, error) {
	return optionalWorkspace(r.q.EventWorkspaceOfDoc(ctx, id))
}

func (r *EventWorkspacesRepo) OfConversation(ctx context.Context, id string) (string, error) {
	return optionalWorkspace(r.q.EventWorkspaceOfConversation(ctx, id))
}

func (r *EventWorkspacesRepo) OfPlay(ctx context.Context, id string) (string, error) {
	return optionalWorkspace(r.q.EventWorkspaceOfPlay(ctx, id))
}

func (r *EventWorkspacesRepo) OfRepo(ctx context.Context, owner, name string) (string, error) {
	return optionalWorkspace(r.q.EventWorkspaceOfRepo(ctx, sqlcgen.EventWorkspaceOfRepoParams{Owner: owner, Name: name}))
}

func (r *EventWorkspacesRepo) OfStack(ctx context.Context, id string) (string, error) {
	return optionalWorkspace(r.q.EventWorkspaceOfStack(ctx, id))
}

func (r *EventWorkspacesRepo) OfDeploy(ctx context.Context, id string) (string, error) {
	return optionalWorkspace(r.q.EventWorkspaceOfDeploy(ctx, id))
}

// ListWorkspaceIDs returns every workspace's id, oldest first.
func (r *EventWorkspacesRepo) ListWorkspaceIDs(ctx context.Context) ([]string, error) {
	ids, err := r.q.ListWorkspaceIDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("list workspace ids: %w", err)
	}
	return ids, nil
}
