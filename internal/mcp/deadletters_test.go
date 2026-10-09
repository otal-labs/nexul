package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus/deadletter"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

type fakeDeadLetters struct {
	letters map[string]deadletter.DeadLetter
}

func (f *fakeDeadLetters) Put(_ context.Context, dl deadletter.DeadLetter) error {
	f.letters[dl.ID] = dl
	return nil
}

func (f *fakeDeadLetters) List(_ context.Context, limit, offset int) ([]deadletter.DeadLetter, error) {
	out := make([]deadletter.DeadLetter, 0, len(f.letters))
	for _, dl := range f.letters {
		out = append(out, dl)
	}
	slices.SortFunc(out, func(a, b deadletter.DeadLetter) int { return strings.Compare(a.ID, b.ID) })
	return out[min(offset, len(out)):min(offset+limit, len(out))], nil
}

func (f *fakeDeadLetters) Count(context.Context) (int, error) {
	return len(f.letters), nil
}

func (f *fakeDeadLetters) Get(_ context.Context, id string) (*deadletter.DeadLetter, error) {
	dl, ok := f.letters[id]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	return &dl, nil
}

func (f *fakeDeadLetters) Delete(_ context.Context, id string) error {
	delete(f.letters, id)
	return nil
}

type fakePublisher struct{ topics []string }

func (p *fakePublisher) Publish(_ context.Context, topic string, _ any) error {
	p.topics = append(p.topics, topic)
	return nil
}

func (p *fakePublisher) PublishWithID(_ context.Context, _, topic string, _ any) error {
	p.topics = append(p.topics, topic)
	return nil
}

// holders stands in for access: each user holds the listed actions in some workspace.
type holders map[string]permissions.Set

func (h holders) RequireAnywhere(ctx context.Context, action permissions.Action) error {
	actor, _ := identity.ActorFromCtx(ctx)
	if h[actor.ID].Has(action) {
		return nil
	}
	return fmt.Errorf("%w: %s required", apperrs.ErrForbidden, action)
}

func deadLetterFixture() (map[string]mcptool.Tool, *fakeDeadLetters, *fakePublisher) {
	store := &fakeDeadLetters{letters: map[string]deadletter.DeadLetter{
		"dl-json": {ID: "dl-json", Topic: "doc.created", Payload: []byte(`{"doc":{"id":"d-1"}}`), Error: "boom", Attempts: 3, CreatedAt: time.Unix(10, 0)},
		"dl-text": {ID: "dl-text", Topic: "ticket.created", Payload: []byte("not json"), Error: "bad", Attempts: 1, CreatedAt: time.Unix(20, 0)},
	}}
	pub := &fakePublisher{}
	tools := map[string]mcptool.Tool{}
	for _, tool := range deadLetterTools(store, pub, holders{
		"admin-1":  permissions.SetOf(permissions.InstanceRead, permissions.InstanceWrite),
		"reader-1": permissions.SetOf(permissions.InstanceRead),
	}) {
		tools[tool.Name] = tool
	}
	return tools, store, pub
}

func as(userID string) context.Context {
	return identity.WithActor(context.Background(), identity.Actor{ID: userID})
}

func TestDeadLetters_NeedTheInstancePermissions(t *testing.T) {
	tools, store, pub := deadLetterFixture()
	_, err := tools["dead_letter_list"].Call(as("member-1"), json.RawMessage(`{}`))
	require.ErrorIs(t, err, apperrs.ErrForbidden, "listing needs instance:read")
	_, err = tools["dead_letter_list"].Call(as("reader-1"), json.RawMessage(`{}`))
	require.NoError(t, err)
	_, err = tools["dead_letter_replay"].Call(as("reader-1"), json.RawMessage(`{"id":"dl-json"}`))
	require.ErrorIs(t, err, apperrs.ErrForbidden, "replaying needs instance:write")
	assert.Empty(t, pub.topics)
	assert.Len(t, store.letters, 2, "a refused replay leaves the dead letter in place")
}

func TestDeadLetterList_ShapesPayloadsAsJSON(t *testing.T) {
	tools, _, _ := deadLetterFixture()
	out, err := tools["dead_letter_list"].Call(as("admin-1"), json.RawMessage(`{"limit":10}`))
	require.NoError(t, err)
	page, ok := out.(mcptool.Page[deadLetterResult])
	require.True(t, ok)
	require.Len(t, page.Items, 2)
	for _, dl := range page.Items {
		assert.True(t, json.Valid(dl.Payload), "%s payload is JSON", dl.ID)
		if dl.ID == "dl-text" {
			assert.JSONEq(t, `"not json"`, string(dl.Payload), "a non-JSON payload reads as a string")
		}
	}
}

func TestDeadLetterList_CountsPastAThousand(t *testing.T) {
	tools, store, _ := deadLetterFixture()
	for i := range 1500 {
		id := fmt.Sprintf("dl-%04d", i)
		store.letters[id] = deadletter.DeadLetter{ID: id, Topic: "doc.created", Payload: []byte(`{}`)}
	}

	out, err := tools["dead_letter_list"].Call(as("admin-1"), json.RawMessage(`{"offset":1000,"limit":100}`))

	require.NoError(t, err)
	page := out.(mcptool.Page[deadLetterResult])
	assert.Len(t, page.Items, 100)
	assert.Equal(t, 1502, page.Total)
	assert.True(t, page.HasMore)
	assert.Equal(t, 1100, page.NextOffset)
}

func TestDeadLetterReplay(t *testing.T) {
	tools, store, pub := deadLetterFixture()
	out, err := tools["dead_letter_replay"].Call(as("admin-1"), json.RawMessage(`{"id":"dl-json"}`))
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"id": "dl-json", "replayed": true}, out)
	assert.Equal(t, []string{"doc.created"}, pub.topics)
	assert.NotContains(t, store.letters, "dl-json")

	_, err = tools["dead_letter_replay"].Call(as("admin-1"), json.RawMessage(`{"id":"dl-json"}`))
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	_, err = tools["dead_letter_replay"].Call(as("admin-1"), json.RawMessage(`{}`))
	require.ErrorIs(t, err, apperrs.ErrInvalid)
}
