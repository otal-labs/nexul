package mcp

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus/deadletter"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

type fakeDeadLetters struct {
	letters map[string]deadletter.DeadLetter
}

func (f *fakeDeadLetters) Put(_ context.Context, dl deadletter.DeadLetter) error {
	f.letters[dl.ID] = dl
	return nil
}

func (f *fakeDeadLetters) List(context.Context, int, int) ([]deadletter.DeadLetter, error) {
	out := make([]deadletter.DeadLetter, 0, len(f.letters))
	for _, dl := range f.letters {
		out = append(out, dl)
	}
	return out, nil
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

type admins map[string]bool

func (a admins) CanCreateWorkspace(_ context.Context, userID string) (bool, error) {
	return a[userID], nil
}

func deadLetterFixture() (map[string]mcptool.Tool, *fakeDeadLetters, *fakePublisher) {
	store := &fakeDeadLetters{letters: map[string]deadletter.DeadLetter{
		"dl-json": {ID: "dl-json", Topic: "doc.created", Payload: []byte(`{"doc":{"id":"d-1"}}`), Error: "boom", Attempts: 3, CreatedAt: time.Unix(10, 0)},
		"dl-text": {ID: "dl-text", Topic: "ticket.created", Payload: []byte("not json"), Error: "bad", Attempts: 1, CreatedAt: time.Unix(20, 0)},
	}}
	pub := &fakePublisher{}
	tools := map[string]mcptool.Tool{}
	for _, tool := range deadLetterTools(store, pub, admins{"admin-1": true}) {
		tools[tool.Name] = tool
	}
	return tools, store, pub
}

func as(userID string) context.Context {
	return identity.WithActor(context.Background(), identity.Actor{ID: userID})
}

func TestDeadLetters_AreInstanceAdminOnly(t *testing.T) {
	tools, store, pub := deadLetterFixture()
	tests := []struct {
		name string
		ctx  context.Context
		want error
	}{
		{"no caller is unauthorized", context.Background(), apperrs.ErrUnauthorized},
		{"a member is forbidden", as("member-1"), apperrs.ErrForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tools["dead_letter_list"].Call(tt.ctx, json.RawMessage(`{}`))
			require.ErrorIs(t, err, tt.want)
			_, err = tools["dead_letter_replay"].Call(tt.ctx, json.RawMessage(`{"id":"dl-json"}`))
			require.ErrorIs(t, err, tt.want)
		})
	}
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
