package botwebhook

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

const pngAvatar = "data:image/png;base64,iVBORw0KGgo="

func callTool(ctx context.Context, t *testing.T, s *Service, name, args string) (any, error) {
	t.Helper()
	for _, tool := range MCPTools(s) {
		if tool.Name == name {
			return tool.Call(ctx, json.RawMessage(args))
		}
	}
	t.Fatalf("tool %s not found", name)
	return nil, nil
}

func mustCall[T any](ctx context.Context, t *testing.T, s *Service, name, args string) T {
	t.Helper()
	out, err := callTool(ctx, t, s, name, args)
	require.NoError(t, err)
	got, ok := out.(T)
	require.True(t, ok, "%s returned %T", name, out)
	return got
}

func TestBotwebhookList_UrlOnlyForWriteAndNeverTheAvatarBytes(t *testing.T) {
	s := newTestService(newFakeRepo())
	_, err := s.Create(as(editor), "c-eng", "CI", pngAvatar)
	require.NoError(t, err)

	asReader := mustCall[mcptool.Page[botResult]](as(reader), t, s, "botwebhook_list", `{"conversation_id":"c-eng"}`)
	require.Len(t, asReader.Items, 1)
	assert.Empty(t, asReader.Items[0].URL)
	assert.True(t, asReader.Items[0].HasAvatar)
	raw, err := json.Marshal(asReader)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "/api/botwebhooks/")
	assert.NotContains(t, string(raw), "iVBOR")

	asEditor := mustCall[mcptool.Page[botResult]](as(editor), t, s, "botwebhook_list", `{"conversation_id":"c-eng"}`)
	require.Len(t, asEditor.Items, 1)
	assert.Contains(t, asEditor.Items[0].URL, "/api/botwebhooks/"+asEditor.Items[0].ID+"/")

	_, err = callTool(as(reader), t, s, "botwebhook_list", `{"conversation_id":"c-eng","deleted":true}`)
	require.ErrorIs(t, err, apperrs.ErrForbidden, "the restore list is for editors")
}

func TestBotwebhookUpdate_PatchesOnlyWhatItIsGiven(t *testing.T) {
	s := newTestService(newFakeRepo())
	created := mustCall[botResult](as(editor), t, s, "botwebhook_create", `{"conversation_id":"c-eng","name":"CI","avatar":"`+pngAvatar+`"}`)
	require.NotEmpty(t, created.URL)
	require.True(t, created.HasAvatar)

	renamed := mustCall[botResult](as(editor), t, s, "botwebhook_update", `{"id":"`+created.ID+`","name":"Builds"}`)
	assert.Equal(t, "Builds", renamed.Name)
	assert.True(t, renamed.HasAvatar, "an omitted avatar keeps its value")
	assert.Equal(t, created.URL, renamed.URL, "a rename keeps the URL")

	cleared := mustCall[botResult](as(editor), t, s, "botwebhook_update", `{"id":"`+created.ID+`","avatar":""}`)
	assert.False(t, cleared.HasAvatar, "an empty avatar clears it, unlike an omitted one")
	assert.Equal(t, "Builds", cleared.Name)

	regenerated := mustCall[botResult](as(editor), t, s, "botwebhook_update", `{"id":"`+created.ID+`","regenerate":true}`)
	assert.NotEqual(t, created.URL, regenerated.URL)
}

func TestBotwebhookUpdate_DeleteNeedsItsOwnBitAndRestoreNeedsWrite(t *testing.T) {
	s := newTestService(newFakeRepo())
	b := mustCall[botResult](as(editor), t, s, "botwebhook_create", `{"conversation_id":"c-eng","name":"CI"}`)

	_, err := callTool(as(editor), t, s, "botwebhook_update", `{"id":"`+b.ID+`","deleted":true}`)
	require.ErrorIs(t, err, apperrs.ErrForbidden, "write does not delete")

	deleted := mustCall[botResult](as(everyone), t, s, "botwebhook_update", `{"id":"`+b.ID+`","deleted":true}`)
	assert.True(t, deleted.Deleted)
	assert.Empty(t, deleted.URL)

	restored := mustCall[botResult](as(editor), t, s, "botwebhook_update", `{"id":"`+b.ID+`","deleted":false}`)
	assert.False(t, restored.Deleted)
	assert.NotEqual(t, b.URL, restored.URL, "a restore issues a fresh URL")
}

func TestBotwebhookTools_RejectBadArguments(t *testing.T) {
	s := newTestService(newFakeRepo())
	tests := []struct{ name, tool, args string }{
		{"create without a name", "botwebhook_create", `{"conversation_id":"c-eng"}`},
		{"create with an unknown key", "botwebhook_create", `{"conversation_id":"c-eng","name":"CI","token":"x"}`},
		{"update with a non-boolean deleted", "botwebhook_update", `{"id":"b-1","deleted":"yes"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := callTool(as(editor), t, s, tt.tool, tt.args)
			require.ErrorIs(t, err, apperrs.ErrInvalid)
		})
	}
}
