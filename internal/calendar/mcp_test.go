package calendar

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

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

func TestMeetingTools_Refusals(t *testing.T) {
	tests := []struct {
		name, actor, tool, args string
		want                    error
	}{
		{"a create without calendar:write", "bob", "meeting_create",
			`{"workspace_id":"ws-1","title":"Standup","starts_at":"2026-11-02T10:00:00Z","ends_at":"2026-11-02T10:30:00Z"}`, apperrs.ErrForbidden},
		{"a start that is not a time", "alice", "meeting_create",
			`{"workspace_id":"ws-1","title":"Standup","starts_at":"tomorrow","ends_at":"2026-11-02T10:30:00Z"}`, apperrs.ErrInvalid},
		{"an end that is not a time", "alice", "meeting_create",
			`{"workspace_id":"ws-1","title":"Standup","starts_at":"2026-11-02T10:00:00Z","ends_at":"later"}`, apperrs.ErrInvalid},
		{"a repeat end that is not a time", "alice", "meeting_create",
			`{"workspace_id":"ws-1","title":"Standup","starts_at":"2026-11-02T10:00:00Z","ends_at":"2026-11-02T10:30:00Z","repeat_until":"never"}`, apperrs.ErrInvalid},
		{"a list from that is not a time", "alice", "meeting_list", `{"workspace_id":"ws-1","from":"monday"}`, apperrs.ErrInvalid},
		{"a list to that is not a time", "alice", "meeting_list", `{"workspace_id":"ws-1","to":"friday"}`, apperrs.ErrInvalid},
		{"a list outside the workspace", "mallory", "meeting_list", `{"workspace_id":"ws-1"}`, apperrs.ErrNotFound},
		{"an update to a meeting nobody has", "alice", "meeting_update", `{"id":"m-gone","title":"x"}`, apperrs.ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := callTool(as(t, tt.actor), t, newHarness().svc, tt.tool, tt.args)
			require.ErrorIs(t, err, tt.want)
		})
	}
}

func TestMeetingTools_CreateListUpdateReply(t *testing.T) {
	h := newHarness()
	created := mustCall[meetingResult](as(t, "alice"), t, h.svc, "meeting_create",
		`{"workspace_id":"ws-1","title":"Standup","starts_at":"2026-11-02T10:00:00Z","ends_at":"2026-11-02T10:30:00Z","repeats":"weekly","invitee_ids":["bob"]}`)
	assert.Equal(t, RepeatWeekly, created.Repeats)
	require.Len(t, created.Occurrences, 1)
	assert.Equal(t, at("2026-11-02T10:00:00Z"), created.Occurrences[0].Slot, "the next occurrence")

	page := mustCall[mcptool.Page[meetingResult]](as(t, "bob"), t, h.svc, "meeting_list",
		`{"workspace_id":"ws-1","from":"2026-11-01T00:00:00Z","to":"2026-11-20T00:00:00Z"}`)
	require.Equal(t, 1, page.Total)
	assert.Len(t, page.Items[0].Occurrences, 3, "every occurrence in the range")
	next := mustCall[mcptool.Page[meetingResult]](as(t, "bob"), t, h.svc, "meeting_list", `{"workspace_id":"ws-1"}`)
	assert.Len(t, next.Items[0].Occurrences, 1, "without a range, the next one only")

	_, err := callTool(as(t, "bob"), t, h.svc, "meeting_update", `{"id":"`+created.ID+`","reply":"accepted","title":"Mine now"}`)
	require.ErrorIs(t, err, apperrs.ErrInvalid, "a reply goes alone")
	replied := mustCall[meetingResult](as(t, "bob"), t, h.svc, "meeting_update", `{"id":"`+created.ID+`","reply":"accepted"}`)
	assert.Equal(t, ReplyAccepted, replied.Invitees[0].Reply)

	updated := mustCall[meetingResult](as(t, "alice"), t, h.svc, "meeting_update", `{"id":"`+created.ID+`","add_invitee_ids":["dave"],"notes":"Bring updates"}`)
	assert.Equal(t, "Standup", updated.Title, "an omitted field keeps its value")
	assert.Equal(t, "Bring updates", updated.Notes)
	assert.Len(t, updated.Invitees, 2)
	assert.Equal(t, ReplyAccepted, updated.Invitees[0].Reply, "a reply survives a change")

	one := mustCall[meetingResult](as(t, "alice"), t, h.svc, "meeting_update", `{"id":"`+created.ID+`","slot":"2026-11-09T10:00:00Z","scope":"this","cancelled":true}`)
	assert.True(t, one.Cancelled)
	assert.Equal(t, created.ID, one.SeriesID)
	_, err = callTool(as(t, "alice"), t, h.svc, "meeting_update", `{"id":"`+created.ID+`","slot":"next week","scope":"this"}`)
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	_, err = callTool(as(t, "alice"), t, h.svc, "meeting_update", `{"id":"`+created.ID+`","repeats":"weekly","repeat_until":"soon"}`)
	require.ErrorIs(t, err, apperrs.ErrInvalid)
}
