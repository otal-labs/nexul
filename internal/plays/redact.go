package plays

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/redact"
)

// redactedTrails hides personal access tokens in every trail and run event before they are saved, whichever turn wrote them.
type redactedTrails struct {
	TrailRepo
}

func (r redactedTrails) CreateTrail(ctx context.Context, t *Trail, evts ...eventbus.OutboxEvent) error {
	saved, evts, err := redactTrail(t, evts)
	if err != nil {
		return err
	}
	return r.TrailRepo.CreateTrail(ctx, saved, evts...)
}

func (r redactedTrails) UpdateTrail(ctx context.Context, t *Trail, evts ...eventbus.OutboxEvent) error {
	saved, evts, err := redactTrail(t, evts)
	if err != nil {
		return err
	}
	return r.TrailRepo.UpdateTrail(ctx, saved, evts...)
}

// redactTrail returns redacted copies only where a token was found, leaving the caller's trail as the live run knows it.
func redactTrail(t *Trail, evts []eventbus.OutboxEvent) (*Trail, []eventbus.OutboxEvent, error) {
	out := make([]eventbus.OutboxEvent, 0, len(evts))
	for _, e := range evts {
		payload, found, err := redact.JSON(e.Payload)
		if err != nil {
			return nil, nil, err
		}
		if found {
			e.Payload = payload
		}
		out = append(out, e)
	}
	raw, found, err := redact.JSON(t)
	if err != nil {
		return nil, nil, err
	}
	if !found {
		return t, out, nil
	}
	var saved Trail
	if err := json.Unmarshal(raw, &saved); err != nil {
		return nil, nil, fmt.Errorf("unmarshal redacted trail %s: %w", t.ID, err)
	}
	return &saved, out, nil
}

// redactedThreads hides personal access tokens in the run's thread messages before chat saves them.
type redactedThreads struct {
	Threads
}

func (r redactedThreads) PostMessage(ctx context.Context, conversationID, authorID, body string) (string, error) {
	return r.Threads.PostMessage(ctx, conversationID, authorID, redact.Tokens(body))
}

func (r redactedThreads) PostSystemNote(ctx context.Context, conversationID, viaUserID, body string) error {
	return r.Threads.PostSystemNote(ctx, conversationID, viaUserID, redact.Tokens(body))
}
