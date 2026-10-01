package docs

import (
	"context"
	"fmt"
	"slices"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/ids"
)

// Watchers lists the people who get a doc's change notifications and whether the caller is one; requires docs:read.
func (s *Service) Watchers(ctx context.Context, docID string) (*Watchers, error) {
	d, err := s.Get(ctx, docID)
	if err != nil {
		return nil, err
	}
	return s.watchersOf(ctx, d.ID)
}

// SetWatching starts or stops the caller watching a doc, needing only docs:read; a stop outlasts their own later edits.
func (s *Service) SetWatching(ctx context.Context, docID string, watching bool) (*Watchers, error) {
	actor, ok := identity.ActorFromCtx(ctx)
	if !ok || actor.ID == "" {
		return nil, fmt.Errorf("%w: an authenticated user is required", apperrs.ErrUnauthorized)
	}
	d, err := s.Get(ctx, docID)
	if err != nil {
		return nil, err
	}
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicWatchersChanged, Payload: WatchersChangedEvent{
		Doc: WatchedDoc{ID: d.ID, ProjectID: d.ProjectID, Title: d.Title}, UserID: actor.ID, Watching: watching,
	}}
	if err := s.repo.SetWatching(ctx, d.ID, actor.ID, watching, s.now().UTC(), evt); err != nil {
		return nil, fmt.Errorf("set watching on doc %s: %w", d.ID, err)
	}
	return s.watchersOf(ctx, d.ID)
}

func (s *Service) watchersOf(ctx context.Context, docID string) (*Watchers, error) {
	ws, err := s.repo.ListWatchers(ctx, docID)
	if err != nil {
		return nil, fmt.Errorf("list watchers of doc %s: %w", docID, err)
	}
	caller := actorID(ctx)
	return &Watchers{
		Watchers: ws,
		Watching: slices.ContainsFunc(ws, func(w *Watcher) bool { return w.UserID == caller }),
	}, nil
}
