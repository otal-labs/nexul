package mentions

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

func mcpService() *Service {
	return New(Config{
		Tickets: &fakeTicketSource{
			tickets: map[string]Ticket{"t-1": {ID: "t-1", Title: "Fix the bug", Status: "open"}},
			search:  []SearchHit{{ID: "t-1", Title: "Fix the bug"}},
		},
		Docs: &fakeDocSource{
			docs:   map[string]Doc{"d-1": {ID: "d-1", Title: "Architecture"}, "d-2": {ID: "d-2", Title: "Secret plan"}},
			search: []SearchHit{{ID: "d-1", Title: "Architecture"}, {ID: "d-2", Title: "Secret plan"}},
		},
		Statuses:    &fakeStatusSource{statuses: map[string]Status{"open": {ID: "open", Name: "Open"}}},
		Access:      &fakeAccessChecker{canOpen: map[string]bool{"d-1": true}},
		Projects:    &fakeProjectSource{projects: map[string]Project{}},
		TicketTypes: &fakeTicketTypeSource{types: map[string]TicketType{}},
	})
}

func callSearch(ctx context.Context, args string) (any, error) {
	return MCPTools(mcpService())[0].Call(ctx, json.RawMessage(args))
}

func asUser() context.Context {
	return identity.WithActor(context.Background(), identity.Actor{ID: "u-1"})
}

func TestMCPTools_Surface(t *testing.T) {
	tools := MCPTools(mcpService())
	require.Len(t, tools, 1)
	assert.Equal(t, "mention_search", tools[0].Name)
	assert.NotEmpty(t, tools[0].Title)
	assert.True(t, tools[0].Hints.ReadOnly)
}

func TestMentionSearch_Errors(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		args    string
		wantErr error
	}{
		{"neither query nor refs", asUser(), `{}`, apperrs.ErrInvalid},
		{"both query and refs", asUser(), `{"query":"fix","refs":[{"type":"doc","id":"d-1"}]}`, apperrs.ErrInvalid},
		{"a ref without an id", asUser(), `{"refs":[{"type":"doc"}]}`, apperrs.ErrInvalid},
		{"a ref of an unknown type", asUser(), `{"refs":[{"type":"user","id":"u-2"}]}`, apperrs.ErrInvalid},
		{"a blank query", asUser(), `{"query":" "}`, apperrs.ErrInvalid},
		{"a search without a caller", context.Background(), `{"query":"fix"}`, apperrs.ErrUnauthorized},
		{"a resolve without a caller", context.Background(), `{"refs":[{"type":"doc","id":"d-1"}]}`, apperrs.ErrUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := callSearch(tt.ctx, tt.args)
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestMentionSearch_Query(t *testing.T) {
	out, err := callSearch(asUser(), `{"query":"a"}`)
	require.NoError(t, err)
	page := out.(mcptool.Page[SearchResult])
	require.Len(t, page.Items, 2, "the doc the caller cannot open is not found")
	assert.Equal(t, "t-1", page.Items[0].ID)
	assert.Equal(t, "Open", page.Items[0].StatusLabel)
	assert.Equal(t, "d-1", page.Items[1].ID)

	out, err = callSearch(asUser(), `{"query":"a","limit":1}`)
	require.NoError(t, err)
	assert.True(t, out.(mcptool.Page[SearchResult]).HasMore)
}

func TestMentionSearch_Refs(t *testing.T) {
	out, err := callSearch(asUser(), `{"refs":[{"type":"ticket","id":"t-1"},{"type":"doc","id":"d-2"},{"type":"ticket","id":"missing"}]}`)
	require.NoError(t, err)
	chips := out.(mcptool.Page[Chip]).Items
	require.Len(t, chips, 2, "a missing target is left out")
	assert.Equal(t, "Fix the bug", chips[0].Title)
	assert.Equal(t, "Open", chips[0].StatusLabel)
	assert.Equal(t, "Secret plan", chips[1].Title)
	assert.False(t, chips[1].CanOpen, "an unopenable doc resolves as an inert chip")
}
