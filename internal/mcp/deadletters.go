package mcp

import (
	"context"
	"encoding/json"
	"time"

	"github.com/otal-labs/nexul/internal/platform/eventbus/deadletter"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// deadLetterResult is a dead letter as the model reads it: the payload as JSON, not base64 bytes.
type deadLetterResult struct {
	ID        string          `json:"id"`
	Topic     string          `json:"topic"`
	Payload   json.RawMessage `json:"payload"`
	Error     string          `json:"error"`
	Attempts  int             `json:"attempts"`
	CreatedAt time.Time       `json:"created_at"`
}

// ponytail: pages in memory over the newest 1,000; add a count query if dead letters ever pile up past that.
const deadLetterScan = 1000

type deadLetterListIn struct {
	mcptool.PageArgs
}

type deadLetterReplayIn struct {
	ID string `json:"id" jsonschema:"The dead letter's id, from dead_letter_list."`
}

// deadLetterTools read and replay every domain's failed events, so both are instance-admin only.
func deadLetterTools(store deadletter.Storer, pub deadletter.Publisher, admin identity.InstanceAdmin) []mcptool.Tool {
	return []mcptool.Tool{
		mcptool.New("dead_letter_list", "List dead letters",
			"Lists events that exhausted their retries or failed permanently, newest first, with the error each one "+
				"hit and its payload. Use it to diagnose a failure the event bus could not process, then "+
				"dead_letter_replay to retry one once the cause is fixed. Instance admins only, because payloads carry "+
				"every domain's data.",
			mcptool.Hints{ReadOnly: true, Local: true},
			func(ctx context.Context, in deadLetterListIn) (any, error) {
				if err := identity.RequireInstanceAdmin(ctx, admin); err != nil {
					return nil, err
				}
				letters, err := store.List(ctx, deadLetterScan, 0)
				if err != nil {
					return nil, err
				}
				return mcptool.Paginate(shapeDeadLetters(letters), in.PageArgs), nil
			}),
		mcptool.New("dead_letter_replay", "Replay dead letter",
			"Republishes a dead letter to its original topic and removes it from the store, so every consumer of "+
				"that topic runs again. Use it only after fixing what made the event fail; dead_letter_list shows the "+
				"error. Instance admins only.",
			mcptool.Hints{},
			func(ctx context.Context, in deadLetterReplayIn) (any, error) {
				if err := identity.RequireInstanceAdmin(ctx, admin); err != nil {
					return nil, err
				}
				if err := deadletter.Replay(ctx, store, pub, in.ID); err != nil {
					return nil, err
				}
				return map[string]any{"id": in.ID, "replayed": true}, nil
			}),
	}
}

// shapeDeadLetters renders payloads as JSON, falling back to a JSON string for a payload that is not JSON.
func shapeDeadLetters(letters []deadletter.DeadLetter) []deadLetterResult {
	out := make([]deadLetterResult, 0, len(letters))
	for _, dl := range letters {
		payload := json.RawMessage(dl.Payload)
		if !json.Valid(payload) {
			payload, _ = json.Marshal(string(dl.Payload)) // marshalling a string cannot fail
		}
		out = append(out, deadLetterResult{
			ID: dl.ID, Topic: dl.Topic, Payload: payload, Error: dl.Error, Attempts: dl.Attempts, CreatedAt: dl.CreatedAt,
		})
	}
	return out
}
