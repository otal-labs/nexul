package composite

import (
	"context"
	"errors"

	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// restoreThenChange restores first, so the other fields apply to a workspace the caller reaches again.
func restoreThenChange(ctx context.Context, w WorkspaceService, limits AutoPlayLimits, accounts GitHubAccounts, setup AccessSetup, userID string, in workspaceUpdateIn) (any, error) {
	restored, err := w.Restore(ctx, userID, in.ID)
	if err != nil {
		return nil, err
	}
	no := false
	out := workspaceUpdateResult{ID: restored.ID, Name: restored.Name, Slug: restored.Slug, Archived: &no}
	if !in.changesSettings() {
		return out, nil
	}
	changed, err := changeSettings(ctx, w, limits, accounts, setup, userID, in)
	if err != nil {
		return nil, appliedFirst("archived", err)
	}
	changed.Archived = &no
	return changed, nil
}

// archiveLast archives after the other fields, which need a workspace the caller still reaches.
func archiveLast(ctx context.Context, w WorkspaceService, userID string, in workspaceUpdateIn, changed workspaceUpdateResult, hadSettings bool) (any, error) {
	archived, err := w.Archive(ctx, userID, in.ID)
	if err != nil && hadSettings {
		return nil, &mcptool.PartialError{Applied: []string{"every other field"}, Err: err}
	}
	if err != nil {
		return nil, err
	}
	yes := true
	changed.ID, changed.Name, changed.Slug, changed.Archived = archived.ID, archived.Name, archived.Slug, &yes
	return changed, nil
}

// appliedFirst puts step at the head of what a failed update already applied.
func appliedFirst(step string, err error) error {
	var partial *mcptool.PartialError
	if errors.As(err, &partial) {
		return &mcptool.PartialError{Applied: append([]string{step}, partial.Applied...), Err: partial.Err}
	}
	return &mcptool.PartialError{Applied: []string{step}, Err: err}
}

// archivedWorkspaces lists what the instance Owner can restore, each as workspace_update archived false names it.
func archivedWorkspaces(ctx context.Context, w WorkspaceService, userID string, page mcptool.PageArgs) (any, error) {
	ws, err := w.ListArchived(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]workspaceResult, 0, len(ws))
	for _, one := range ws {
		out = append(out, workspaceResult{ID: one.ID, Name: one.Name, Slug: one.Slug, ArchivedAt: one.ArchivedAt})
	}
	return mcptool.Paginate(out, page), nil
}
