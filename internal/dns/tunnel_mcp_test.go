package dns

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

const routeT1Args = `{"id":"t1","hostname":"nexul.example.com","zone_id":"z1","zone":"example.com","origin_url":"http://web:80"}`

func TestTunnelTools_Errors(t *testing.T) {
	runToolErrors(t, []toolError{
		{name: "create without a name is invalid", tool: "dns_tunnel_create", args: `{}`, want: apperrs.ErrInvalid},
		{name: "the old tunnel_id argument is rejected", tool: "dns_tunnel_delete", args: `{"tunnel_id":"t1"}`, want: apperrs.ErrInvalid},
		{name: "the old service argument is rejected", tool: "dns_tunnel_update", args: `{"id":"t1","service":"http://web:80"}`, want: apperrs.ErrInvalid},
		{name: "live status of an unknown tunnel is not found", tool: "dns_tunnel_list", args: `{"id":"ghost"}`, want: apperrs.ErrNotFound},
		{name: "updating an unknown tunnel is not found", tool: "dns_tunnel_update", args: `{"id":"ghost","rotate_credentials":true}`, want: apperrs.ErrNotFound},
		{name: "a route without a hostname is invalid", tool: "dns_tunnel_update", args: `{"id":"t1","origin_url":"http://web:80"}`, want: apperrs.ErrInvalid},
		{name: "deleting an unknown tunnel is not found", tool: "dns_tunnel_delete", args: `{"id":"ghost"}`, want: apperrs.ErrNotFound},
		{name: "a tunnel the token may not create is forbidden", tool: "dns_tunnel_create", args: `{"name":"edge"}`, want: apperrs.ErrForbidden,
			setup: func(f *toolFakes) { f.tunnels.createErr = apperrs.ErrForbidden }},
		{name: "a tunnel delete the token may not make is forbidden", tool: "dns_tunnel_delete", args: `{"id":"t1"}`,
			want: apperrs.ErrForbidden, setup: func(f *toolFakes) { f.tunnels.deleteErr = apperrs.ErrForbidden }},
		{name: "a failed tunnel read surfaces", tool: "dns_tunnel_list", args: `{}`, want: errBoom,
			setup: func(f *toolFakes) { f.repo.tunnelErr = errBoom }},
		{name: "no Cloudflare connection is invalid", tool: "dns_tunnel_list", args: `{"id":"t1"}`, want: apperrs.ErrInvalid,
			setup: func(f *toolFakes) { f.disconnected = true }},
	})
}

func TestTunnelUpdate_AFailedRouteStopsBeforeRotation(t *testing.T) {
	f := newToolFakes(t)
	f.tunnels.routeErr = apperrs.ErrForbidden
	before := f.repo.tunnels["t1"].Token
	_, err := f.call(t, "dns_tunnel_update", `{"id":"t1","hostname":"nexul.example.com","zone_id":"z1","zone":"example.com","origin_url":"http://web:80","rotate_credentials":true}`)
	require.ErrorIs(t, err, apperrs.ErrForbidden)
	assert.Equal(t, before, f.repo.tunnels["t1"].Token, "rotation never ran")
}

func TestTunnelCreate_NeverReturnsTheToken(t *testing.T) {
	f := newToolFakes(t)
	got, err := f.call(t, "dns_tunnel_create", `{"name":"edge"}`)
	require.NoError(t, err)
	created := got.(tunnelResult)
	assert.Equal(t, "edge", created.Name)
	assert.NotContains(t, asJSON(t, got), "tunnel-token")
	assert.NotContains(t, asJSON(t, got), f.repo.tunnels[created.ID].Token, "the ciphertext stays out of the result too")

	again, err := f.call(t, "dns_tunnel_create", `{"name":"edge"}`)
	require.NoError(t, err)
	assert.Equal(t, created.ID, again.(tunnelResult).ID, "a retry returns the tunnel it already made")
}

func TestTunnelList(t *testing.T) {
	f := newToolFakes(t)
	got, err := f.call(t, "dns_tunnel_list", `{}`)
	require.NoError(t, err)
	page := got.(mcptool.Page[tunnelResult])
	require.Len(t, page.Items, 1)
	assert.Empty(t, page.Items[0].Status, "the list reads only Nexul's records")
	assert.NotContains(t, asJSON(t, got), f.repo.tunnels["t1"].Token)

	got, err = f.call(t, "dns_tunnel_list", `{"id":"t1"}`)
	require.NoError(t, err)
	page = got.(mcptool.Page[tunnelResult])
	require.Len(t, page.Items, 1)
	assert.Equal(t, "healthy", page.Items[0].Status, "an id reads the live status")
}

func TestTunnelUpdate_OmittedRouteFieldsKeepTheirValue(t *testing.T) {
	f := newToolFakes(t)
	_, err := f.call(t, "dns_tunnel_update", routeT1Args)
	require.NoError(t, err)

	got, err := f.call(t, "dns_tunnel_update", `{"id":"t1","origin_url":"http://web:8080"}`)
	require.NoError(t, err)
	res := got.(tunnelUpdateResult)
	assert.Equal(t, []string{"route"}, res.Applied)
	assert.Equal(t, "nexul.example.com", res.Tunnel.Hostname)
	assert.Equal(t, "z1", res.Tunnel.ZoneID)
	assert.Equal(t, "http://web:8080", res.Tunnel.OriginURL)
	assert.Equal(t, routeCall{TunnelID: "t1", Hostname: "nexul.example.com", Service: "http://web:8080"}, f.tunnels.routeCalls[1])
	assert.Len(t, f.records.records["z1"], 4, "re-pointing updates the tunnel's record in place")
}

func TestTunnelUpdate_RotatesAfterRouting(t *testing.T) {
	f := newToolFakes(t)
	before := f.repo.tunnels["t1"].Token
	got, err := f.call(t, "dns_tunnel_update", `{"id":"t1","hostname":"nexul.example.com","zone_id":"z1","zone":"example.com","origin_url":"http://web:80","rotate_credentials":true}`)
	require.NoError(t, err)
	res := got.(tunnelUpdateResult)
	assert.Equal(t, []string{"route", "rotate_credentials"}, res.Applied)
	assert.NotEqual(t, before, f.repo.tunnels["t1"].Token)
	assert.NotContains(t, asJSON(t, got), "rotated-token")
}

func TestTunnelUpdate_SaysWhatAppliedWhenRotationFails(t *testing.T) {
	f := newToolFakes(t)
	f.tunnels.rotateErr = apperrs.ErrForbidden
	_, err := f.call(t, "dns_tunnel_update", `{"id":"t1","hostname":"nexul.example.com","zone_id":"z1","zone":"example.com","origin_url":"http://web:80","rotate_credentials":true}`)
	require.ErrorIs(t, err, apperrs.ErrForbidden)
	assert.Contains(t, err.Error(), "applied [route]")
	assert.Equal(t, "nexul.example.com", f.repo.tunnels["t1"].Hostname, "the route stays applied")
}

func TestTunnelUpdate_NothingToChangeReturnsTheTunnel(t *testing.T) {
	got, err := newToolFakes(t).call(t, "dns_tunnel_update", `{"id":"t1"}`)
	require.NoError(t, err)
	res := got.(tunnelUpdateResult)
	assert.Empty(t, res.Applied)
	assert.Equal(t, "t1", res.Tunnel.ID)
}

func TestTunnelDelete(t *testing.T) {
	f := newToolFakes(t)
	got, err := f.call(t, "dns_tunnel_delete", `{"id":"t1"}`)
	require.NoError(t, err)
	assert.Equal(t, mcptool.Gone("t1"), got)
	assert.Empty(t, f.repo.tunnels)
}
