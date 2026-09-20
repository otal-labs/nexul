package integrations

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseScope(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    Scope
		wantErr bool
	}{
		{"events read", "events:read", ScopeEventsRead, false},
		{"docs read", "docs:read", ScopeDocsRead, false},
		{"docs write", "docs:write", ScopeDocsWrite, false},
		{"docs delete", "docs:delete", Scope("docs:delete"), false},
		{"tickets write", "tickets:write", ScopeTicketsWrite, false},
		{"deploys write", "deploys:write", Scope("deploys:write"), false},
		{"deploys delete unknown action", "deploys:delete", "", true},
		{"topology write", "topology:write", ScopeTopologyWrite, false},
		{"projects read", "projects:read", ScopeProjectsRead, false},
		{"stacks delete", "stacks:delete", Scope("stacks:delete"), false},
		{"unknown domain", "billing:read", "", true},
		{"excluded domain", "auth:read", "", true},
		{"excluded domain pairing", "pairing:read", "", true},
		{"automations now grantable", "automations:write", Scope("automations:write"), false},
		{"members now grantable", "members:delete", Scope("members:delete"), false},
		{"plays run", "plays:run", Scope("plays:run"), false},
		{"memories clone", "memories:clone", Scope("memories:clone"), false},
		{"docs thread", "docs:thread", Scope("docs:thread"), false},
		{"a verb another domain didn't declare", "plays:clone", "", true},
		{"empty", "", "", true},
		{"whitespace", "  ", "", true},
		{"bogus", "nonsense", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseScope(tt.in)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseScopes(t *testing.T) {
	tests := []struct {
		name    string
		in      []string
		wantLen int
		wantErr bool
	}{
		{"single", []string{"docs:read"}, 1, false},
		{"multiple", []string{"docs:read", "docs:write"}, 2, false},
		{"dedupe", []string{"docs:read", "docs:read"}, 1, false},
		{"empty list", nil, 0, true},
		{"invalid entry", []string{"docs:read", "bogus"}, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseScopes(tt.in)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, got, tt.wantLen)
		})
	}
}

func TestExpandScopes(t *testing.T) {
	tests := []struct {
		name string
		in   []Scope
		want []Scope
	}{
		{
			"tickets write implies read only",
			[]Scope{ScopeTicketsWrite},
			[]Scope{ScopeTicketsWrite, ScopeTicketsRead},
		},
		{
			"docs write implies read only",
			[]Scope{ScopeDocsWrite},
			[]Scope{ScopeDocsWrite, ScopeDocsRead},
		},
		{
			"topology write implies read",
			[]Scope{ScopeTopologyWrite},
			[]Scope{ScopeTopologyWrite, ScopeTopologyRead},
		},
		{
			"stacks delete implies read, not write",
			[]Scope{Scope("stacks:delete")},
			[]Scope{Scope("stacks:delete"), Scope("stacks:read")},
		},
		{
			"stacks write does not imply delete",
			[]Scope{Scope("stacks:write")},
			[]Scope{Scope("stacks:write"), Scope("stacks:read")},
		},
		{
			"read-only stays",
			[]Scope{ScopeEventsRead},
			[]Scope{ScopeEventsRead},
		},
		{
			"already present implied is not duplicated",
			[]Scope{ScopeTicketsWrite, ScopeTicketsRead},
			[]Scope{ScopeTicketsWrite, ScopeTicketsRead},
		},
		{
			"plays run implies plays read",
			[]Scope{Scope("plays:run")},
			[]Scope{Scope("plays:run"), Scope("plays:read")},
		},
		{
			"memories clone implies memories read",
			[]Scope{Scope("memories:clone")},
			[]Scope{Scope("memories:clone"), Scope("memories:read")},
		},
		{
			"docs thread implies docs read",
			[]Scope{Scope("docs:thread")},
			[]Scope{Scope("docs:thread"), ScopeDocsRead},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExpandScopes(tt.in)
			assert.ElementsMatch(t, tt.want, got)
		})
	}
}

func TestHasScopes(t *testing.T) {
	assert.True(t, HasScopes([]Scope{ScopeDocsRead, ScopeDocsWrite}, ScopeDocsRead))
	assert.False(t, HasScopes([]Scope{ScopeDocsRead}, ScopeDocsWrite))
}

func TestScopeAllows(t *testing.T) {
	write := []Scope{ScopeTicketsWrite, ScopeTicketsRead, ScopeProjectsRead}
	read := []Scope{ScopeTicketsRead}
	empty := []Scope{}

	tests := []struct {
		name   string
		method string
		path   string
		scopes []Scope
		want   bool
	}{
		{"method to action: GET is read", "GET", "/api/tickets", read, true},
		{"method to action: HEAD is read", "HEAD", "/api/tickets", read, true},
		{"method to action: POST is write", "POST", "/api/tickets", write, true},
		{"method to action: PUT is write", "PUT", "/api/docs/abc123", []Scope{Scope("docs:write")}, true},
		{"method to action: PATCH is write", "PATCH", "/api/tickets/abc/status", []Scope{ScopeTicketsWrite}, true},
		{"method to action: DELETE is delete", "DELETE", "/api/docs/abc123", []Scope{Scope("docs:delete")}, true},
		{"delete scope does not cover write", "PUT", "/api/docs/abc123", []Scope{Scope("docs:delete")}, false},
		{"write scope does not cover delete", "DELETE", "/api/docs/abc123", []Scope{Scope("docs:write")}, false},
		{"read-only cannot write", "POST", "/api/tickets", read, false},

		{"alias: services routes to stacks", "GET", "/api/services/svc-1", []Scope{Scope("stacks:read")}, true},
		{"alias: services denied without stacks scope", "GET", "/api/services/svc-1", []Scope{ScopeDeploysRead}, false},
		{"alias: categories routes to projects", "GET", "/api/categories", []Scope{ScopeProjectsRead}, true},
		{"alias: ticket-types routes to projects", "GET", "/api/ticket-types", []Scope{ScopeProjectsRead}, true},
		{"alias: statuses routes to projects", "DELETE", "/api/statuses/1", []Scope{Scope("projects:delete")}, true},
		{"roles under workspaces: read", "GET", "/api/workspaces/ws1/roles", []Scope{Scope("workspaces:read")}, true},
		{"roles under workspaces: write", "POST", "/api/workspaces/ws1/roles", []Scope{Scope("workspaces:write")}, true},
		{"roles under workspaces: delete", "DELETE", "/api/workspaces/ws1/roles/r1", []Scope{Scope("workspaces:delete")}, true},

		{"projects read with implied scope", "GET", "/api/projects", write, true},
		{"projects read denied without scope", "GET", "/api/projects", read, false},
		{"projects write always denied for ticket scopes", "POST", "/api/projects", write, false},
		{"events catalog read", "GET", "/api/events/catalog", []Scope{ScopeEventsRead}, true},
		{"events write denied", "POST", "/api/events/catalog", []Scope{ScopeEventsRead}, false},

		{"excluded domain: auth denied even with every scope", "GET", "/api/auth/me", []Scope{
			ScopeEventsRead, ScopeDocsRead, ScopeDocsWrite, ScopeTicketsRead, ScopeTicketsWrite,
			ScopeDeploysRead, ScopeTopologyRead, ScopeTopologyWrite, ScopeProjectsRead,
		}, false},
		{"excluded domain: pairing denied", "GET", "/api/pairing", []Scope{Scope("pairing:read")}, false},
		{"automations grantable: read", "GET", "/api/automations", []Scope{Scope("automations:read")}, true},
		{"automations grantable: delete", "DELETE", "/api/automations/a1", []Scope{Scope("automations:delete")}, true},
		{"integrations grantable: write", "POST", "/api/integrations", []Scope{Scope("integrations:write")}, true},
		{"connectors grantable: read", "GET", "/api/connectors", []Scope{Scope("connectors:read")}, true},
		{"connectors has no delete", "DELETE", "/api/connectors/c1", []Scope{Scope("connectors:read"), Scope("connectors:write")}, false},
		{"alias: agent routes to chat", "POST", "/api/agent/conversations/1/interrupt", []Scope{Scope("chat:write")}, true},
		{"alias: agent denied with its own name", "POST", "/api/agent/conversations/1/interrupt", []Scope{Scope("agent:write")}, false},
		{"alias: automation-secrets routes to automations", "GET", "/api/automation-secrets", []Scope{Scope("automations:read")}, true},
		{"alias: automation-secrets denied with its own name", "GET", "/api/automation-secrets", []Scope{Scope("automation-secrets:read")}, false},
		{"members: write", "POST", "/api/members", []Scope{Scope("members:write")}, true},
		{"roles: write", "POST", "/api/roles", []Scope{Scope("roles:write")}, true},

		{"runners install denylisted even with runners:read", "GET", "/api/runners/install", []Scope{Scope("runners:read")}, false},
		{"runners list still allowed", "GET", "/api/runners", []Scope{Scope("runners:read")}, true},

		{"unknown domain", "GET", "/api/billing", []Scope{ScopeEventsRead}, false},
		{"empty scopes denied", "GET", "/api/tickets", empty, false},
		{"trailing slash normalized", "GET", "/api/tickets/", read, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := scopeAllows(tt.method, tt.path, tt.scopes)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCatalog_CoversEveryGrantableScopeOnce(t *testing.T) {
	seen := map[Scope]bool{}
	for _, info := range Catalog() {
		sc := Scope(info.Value)
		assert.True(t, allScopes[sc], "catalog lists unknown scope %q", sc)
		assert.False(t, seen[sc], "catalog repeats %q", sc)
		assert.NotEmpty(t, info.Label, "%q has no label", sc)
		assert.NotEmpty(t, info.Domain, "%q has no domain", sc)
		assert.NotEmpty(t, info.Action, "%q has no action", sc)
		seen[sc] = true
	}
	assert.Len(t, seen, len(allScopes))
	assert.Len(t, seen, 69)
}

func TestCatalog_EveryValueParses(t *testing.T) {
	for _, info := range Catalog() {
		_, err := ParseScope(string(info.Value))
		assert.NoError(t, err, "catalog value %q does not parse", info.Value)
	}
}
