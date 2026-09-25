package dns

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// toolFakes backs every dns tool: zone z1 with three records, tunnel t1 ("prod"), and container c-app on host1/net1.
type toolFakes struct {
	repo         *fakeRepo
	records      *fakeProvider
	tunnels      *fakeTunnelProvider
	prov         *fakeProvisioner
	containers   *fakeContainerLookup
	disconnected bool
}

func newToolFakes(t *testing.T) *toolFakes {
	t.Helper()
	f := &toolFakes{
		repo: newFakeRepo(), records: newFakeProvider(), tunnels: newFakeTunnelProvider(),
		prov: &fakeProvisioner{}, containers: newFakeContainerLookup(),
	}
	f.records.records["z1"] = []Record{
		{ID: "r1", ZoneID: "z1", Type: RecordCNAME, Name: "app.example.com", Content: "t1.cfargotunnel.com", TTL: 1, Proxied: true},
		{ID: "r2", ZoneID: "z1", Type: RecordA, Name: "api.example.com", Content: "203.0.113.10", TTL: 300},
		{ID: "r3", ZoneID: "z1", Type: RecordTXT, Name: "api.example.com", Content: "v=spf1 -all", TTL: 300},
	}
	enc, err := encryptTunnelTokenForTest(testKey(), "the-tunnel-secret")
	require.NoError(t, err)
	require.NoError(t, f.repo.SaveTunnel(t.Context(), Tunnel{ID: "t1", Name: "prod", Token: enc}))
	f.tunnels.tunnels["t1"] = &Tunnel{ID: "t1", Name: "prod", Status: "healthy"}
	f.containers.add("app", ExposureTarget{
		ContainerID: "c-app", Name: "app", StackID: "s-app", ProjectID: "p1", Machine: "host1", Networks: []string{"net1"},
	})
	return f
}

func (f *toolFakes) tools() []mcptool.Tool {
	cfg := Config{
		Repo: f.repo, Provider: f.records, TunnelProvider: f.tunnels, Provisioner: f.prov, Containers: f.containers,
		Tokens: &fakeTokenProvider{token: "at"}, EncryptionKey: testKey(),
		Settings: &fakeSettings{instanceURL: "https://deploy.example.com"},
		Now:      func() time.Time { return time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC) },
	}
	if f.disconnected {
		cfg.Provider, cfg.TunnelProvider, cfg.Tokens = nil, nil, &fakeTokenProvider{err: apperrs.ErrNotFound}
	}
	return MCPTools(NewService(cfg))
}

func (f *toolFakes) call(t *testing.T, name, args string) (any, error) {
	t.Helper()
	for _, tool := range f.tools() {
		if tool.Name == name {
			return tool.Call(t.Context(), json.RawMessage(args))
		}
	}
	t.Fatalf("tool %s not found", name)
	return nil, nil
}

// asJSON renders a result as the adapter sends it, so a test asserts on what the model reads.
func asJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return string(b)
}

// toolError is one error-path case: the fakes to break, the call, and the sentinel it must surface.
type toolError struct {
	name  string
	setup func(*toolFakes)
	tool  string
	args  string
	want  error
}

func runToolErrors(t *testing.T, tests []toolError) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newToolFakes(t)
			if tt.setup != nil {
				tt.setup(f)
			}
			_, err := f.call(t, tt.tool, tt.args)
			require.ErrorIs(t, err, tt.want)
		})
	}
}

func TestMCPTools_Surface(t *testing.T) {
	var names []string
	for _, tool := range newToolFakes(t).tools() {
		names = append(names, tool.Name)
		assert.NotEmpty(t, tool.Title, tool.Name)
		assert.GreaterOrEqual(t, strings.Count(tool.Description, ". ")+1, 3, "%s: at least three sentences", tool.Name)
		assert.Equal(t, strings.HasSuffix(tool.Name, "_list"), tool.Hints.ReadOnly, "%s: read-only exactly when it lists", tool.Name)
		for prop, schema := range tool.InputSchema.Properties {
			assert.NotEmpty(t, schema.Description, "%s.%s", tool.Name, prop)
		}
	}
	assert.Equal(t, []string{
		"dns_zone_list", "dns_record_list", "dns_record_create", "dns_record_update", "dns_record_delete",
		"gateway_list", "gateway_create", "gateway_delete",
		"exposure_list", "exposure_create", "exposure_delete",
		"dns_tunnel_list", "dns_tunnel_create", "dns_tunnel_update", "dns_tunnel_delete",
	}, names)
}

func TestRecordTools_Errors(t *testing.T) {
	runToolErrors(t, []toolError{
		{name: "list without a zone is invalid", tool: "dns_record_list", args: `{}`, want: apperrs.ErrInvalid},
		{name: "list with an unknown argument is invalid", tool: "dns_record_list", args: `{"zone_id":"z1","record_id":"r1"}`, want: apperrs.ErrInvalid},
		{name: "create with an unsupported type is invalid", tool: "dns_record_create",
			args: `{"zone_id":"z1","type":"SRV","name":"api","content":"x"}`, want: apperrs.ErrInvalid},
		{name: "update of a record the zone lacks is not found", tool: "dns_record_update",
			args: `{"zone_id":"z1","id":"ghost","content":"x"}`, want: apperrs.ErrNotFound},
		{name: "delete without an id is invalid", tool: "dns_record_delete", args: `{"zone_id":"z1"}`, want: apperrs.ErrInvalid},
		{name: "a token the account refuses is forbidden", tool: "dns_zone_list", args: `{}`, want: apperrs.ErrForbidden,
			setup: func(f *toolFakes) { f.records.zonesErr = apperrs.ErrForbidden }},
		{name: "a record write the token may not make is forbidden", tool: "dns_record_create",
			args: `{"zone_id":"z1","type":"A","name":"api","content":"203.0.113.10"}`, want: apperrs.ErrForbidden,
			setup: func(f *toolFakes) { f.records.createErr = apperrs.ErrForbidden }},
		{name: "a listing the token may not read is unauthorized", tool: "dns_record_list", args: `{"zone_id":"z1"}`,
			want: apperrs.ErrUnauthorized, setup: func(f *toolFakes) { f.records.listErr = apperrs.ErrUnauthorized }},
		{name: "an update the token may not make is forbidden", tool: "dns_record_update", args: `{"zone_id":"z1","id":"r2","ttl":60}`,
			want: apperrs.ErrForbidden, setup: func(f *toolFakes) { f.records.updateErr = apperrs.ErrForbidden }},
		{name: "a delete the token may not make is forbidden", tool: "dns_record_delete", args: `{"zone_id":"z1","id":"r2"}`,
			want: apperrs.ErrForbidden, setup: func(f *toolFakes) { f.records.deleteErr = apperrs.ErrForbidden }},
		{name: "no Cloudflare connection is invalid", tool: "dns_zone_list", args: `{}`, want: apperrs.ErrInvalid,
			setup: func(f *toolFakes) { f.disconnected = true }},
	})
}

func TestMCPTools_NotConnectedNamesTheFix(t *testing.T) {
	f := newToolFakes(t)
	f.disconnected = true
	_, err := f.call(t, "dns_record_list", `{"zone_id":"z1"}`)
	require.ErrorIs(t, err, apperrs.ErrInvalid)
	assert.Contains(t, err.Error(), "Cloudflare is not connected")
}

func TestRecordUpdate_OmittedFieldsKeepTheirValue(t *testing.T) {
	f := newToolFakes(t)
	got, err := f.call(t, "dns_record_update", `{"zone_id":"z1","id":"r1","content":"t2.cfargotunnel.com"}`)
	require.NoError(t, err)
	assert.Equal(t, recordResult{
		ID: "r1", ZoneID: "z1", Type: RecordCNAME, Name: "app.example.com", Content: "t2.cfargotunnel.com", TTL: 1, Proxied: true,
	}, got, "a content change keeps the tunnel CNAME proxied")

	got, err = f.call(t, "dns_record_update", `{"zone_id":"z1","id":"r1","proxied":false}`)
	require.NoError(t, err)
	assert.False(t, got.(recordResult).Proxied, "an explicit false turns the proxy off")
	assert.Equal(t, "t2.cfargotunnel.com", got.(recordResult).Content)
	assert.Equal(t, []string{TopicRecordChanged, TopicRecordChanged}, f.repo.topics())
}

func TestRecordList(t *testing.T) {
	tests := []struct {
		name    string
		args    string
		wantIDs []string
		total   int
	}{
		{"every record", `{"zone_id":"z1"}`, []string{"r1", "r2", "r3"}, 3},
		{"by name, ignoring case and a trailing dot", `{"zone_id":"z1","name":"API.example.com."}`, []string{"r2", "r3"}, 2},
		{"by name and type", `{"zone_id":"z1","name":"api.example.com","type":"TXT"}`, []string{"r3"}, 1},
		{"by id", `{"zone_id":"z1","id":"r2"}`, []string{"r2"}, 1},
		{"a page", `{"zone_id":"z1","limit":1,"offset":1}`, []string{"r2"}, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newToolFakes(t).call(t, "dns_record_list", tt.args)
			require.NoError(t, err)
			page := got.(mcptool.Page[recordResult])
			var ids []string
			for _, r := range page.Items {
				ids = append(ids, r.ID)
				assert.Nil(t, r.Propagated, "propagation is checked only on request")
			}
			assert.Equal(t, tt.wantIDs, ids)
			assert.Equal(t, tt.total, page.Total)
		})
	}
}

func TestRecordList_CheckPropagation(t *testing.T) {
	for _, tt := range []struct {
		name string
		err  error
		want bool
	}{
		{"live", nil, true},
		{"not yet", apperrs.Retryable(errBoom), false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := newToolFakes(t)
			f.records.propagateErr = tt.err
			got, err := f.call(t, "dns_record_list", `{"zone_id":"z1","id":"r2","check_propagation":true}`)
			require.NoError(t, err)
			items := got.(mcptool.Page[recordResult]).Items
			require.Len(t, items, 1)
			require.NotNil(t, items[0].Propagated)
			assert.Equal(t, tt.want, *items[0].Propagated)
		})
	}
}

func TestRecordCreate_DefaultsTTLAndKeepsProxied(t *testing.T) {
	f := newToolFakes(t)
	got, err := f.call(t, "dns_record_create", `{"zone_id":"z1","type":"CNAME","name":"docs","content":"t1.cfargotunnel.com","proxied":true}`)
	require.NoError(t, err)
	rec := got.(recordResult)
	assert.Equal(t, 1, rec.TTL, "an omitted TTL is the provider's automatic one")
	assert.True(t, rec.Proxied)
	assert.Equal(t, []string{TopicRecordChanged}, f.repo.topics())
}

func TestZoneList(t *testing.T) {
	got, err := newToolFakes(t).call(t, "dns_zone_list", `{}`)
	require.NoError(t, err)
	assert.Equal(t, mcptool.Page[Zone]{Items: []Zone{{ID: "z1", Name: "example.com", Status: "active"}}, Total: 1}, got)
}

func TestRecordDelete(t *testing.T) {
	f := newToolFakes(t)
	got, err := f.call(t, "dns_record_delete", `{"zone_id":"z1","id":"r2"}`)
	require.NoError(t, err)
	assert.Equal(t, mcptool.Gone("r2"), got)
	assert.Len(t, f.records.records["z1"], 2)

	_, err = f.call(t, "dns_record_delete", `{"zone_id":"z1","id":"r2"}`)
	require.NoError(t, err, "deleting an absent record is a no-op")
}
