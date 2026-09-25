package gitprovider

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

func callTool(t *testing.T, p GitProvider, cc ChangeContextReader, name, args string) (any, error) {
	t.Helper()
	for _, tool := range MCPTools(p, cc) {
		if tool.Name == name {
			return tool.Call(t.Context(), json.RawMessage(args))
		}
	}
	t.Fatalf("tool %s not found", name)
	return nil, nil
}

func TestMCPTools_Surface(t *testing.T) {
	var names []string
	for _, tool := range MCPTools(&fakeProvider{}, &fakeChangeReader{}) {
		names = append(names, tool.Name)
		assert.NotEmpty(t, tool.Title, tool.Name)
		assert.NotEmpty(t, tool.Description, tool.Name)
	}
	assert.Equal(t, []string{"pull_request_list", "pull_request_get"}, names)
}

func TestMCPTools_Errors(t *testing.T) {
	tests := []struct {
		name, tool, args string
		provider         *fakeProvider
		want             error
	}{
		{"pull_request_list needs a repo", "pull_request_list", `{"owner":"acme"}`, &fakeProvider{}, apperrs.ErrInvalid},
		{"pull_request_list rejects an unknown state", "pull_request_list", `{"owner":"acme","repo":"app","state":"merged"}`, &fakeProvider{}, apperrs.ErrInvalid},
		{"pull_request_list surfaces a provider failure", "pull_request_list", `{"owner":"acme","repo":"app"}`, &fakeProvider{err: apperrs.ErrUnauthorized}, apperrs.ErrUnauthorized},
		{"pull_request_get rejects the old name key", "pull_request_get", `{"owner":"acme","name":"app","number":7}`, &fakeProvider{}, apperrs.ErrInvalid},
		{"pull_request_get needs a number or a commit", "pull_request_get", `{"owner":"acme","repo":"app"}`, &fakeProvider{}, apperrs.ErrInvalid},
		{"pull_request_get of a commit in no pull request", "pull_request_get", `{"owner":"acme","repo":"app","commit":"abc"}`, &fakeProvider{}, apperrs.ErrNotFound},
		{"pull_request_get of a missing pull request", "pull_request_get", `{"owner":"acme","repo":"app","number":9}`, &fakeProvider{err: apperrs.ErrNotFound}, apperrs.ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := callTool(t, tt.provider, &fakeChangeReader{}, tt.tool, tt.args)
			require.ErrorIs(t, err, tt.want)
		})
	}
}

func TestPullRequestList(t *testing.T) {
	p := &fakeProvider{prs: []*PR{
		{Number: 3, Title: "Add cache", Body: "long body", State: PRStateOpen, LinkedTicketIDs: []string{"t-1"}},
		{Number: 2, Title: "Fix login", State: PRStateOpen},
	}}
	got, err := callTool(t, p, &fakeChangeReader{}, "pull_request_list", `{"owner":"acme","repo":"app","limit":1}`)
	require.NoError(t, err)
	page := got.(mcptool.Page[prSummary])
	assert.Equal(t, PROpts{State: "open", Limit: prScan}, p.listOpts)
	require.Len(t, page.Items, 1)
	assert.Equal(t, 3, page.Items[0].Number)
	assert.Equal(t, []string{"t-1"}, page.Items[0].LinkedTicketIDs)
	assert.True(t, page.HasMore)
	b, err := json.Marshal(got)
	require.NoError(t, err)
	assert.NotContains(t, string(b), "long body", "bodies stay on pull_request_get")

	_, err = callTool(t, p, &fakeChangeReader{}, "pull_request_list", `{"owner":"acme","repo":"app","state":"all"}`)
	require.NoError(t, err)
	assert.Equal(t, "all", p.listOpts.State)
}

func TestPullRequestGet_CarriesTheChangeContext(t *testing.T) {
	tests := []struct{ name, args string }{
		{"by number", `{"owner":"acme","repo":"app","number":7}`},
		{"by a commit it contains", `{"owner":"acme","repo":"app","commit":"abc"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, r := changeFixture()
			got, err := callTool(t, p, r, "pull_request_get", tt.args)
			require.NoError(t, err)
			cc := got.(*ChangeContext)
			assert.Equal(t, 7, cc.PR.Number)
			assert.Len(t, cc.Tickets, 3)
			assert.Len(t, cc.Decisions, 2)
		})
	}
}
