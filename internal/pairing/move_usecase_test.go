package pairing

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var laptopTunnel = ComputerTunnel{TunnelID: "tun-1", Hostname: "laptop-ab12cd34.example.com", ZoneID: "z1", RecordID: "rec-1", AccessAppID: "app-1"}

// tunnelLaptop is alice's laptop as a tunnel paired it, set up, with 20 days left on its session.
func tunnelLaptop() Computer {
	tunnel := laptopTunnel
	return Computer{Name: "Laptop", ServerURL: "https://" + tunnel.Hostname, BearerToken: "sealed-old", TokenExpiresAt: testNow.Add(20 * day),
		HarnessVersion: "0.0.33", SetupConfirmedAt: &testNow, SetupMCPToken: "sealed-mcp", CreatedAt: testNow.Add(-90 * day), Tunnel: &tunnel}
}

// TestHandleFactsReported_MovesATunnelComputerOntoItsRunner: the first report of a runner enrolled for a tunnel computer
// pairs it through the runner, keeping its id, setup and MCP token, and only then deletes its tunnel, DNS record and
// Access app and forgets them.
func TestHandleFactsReported_MovesATunnelComputerOntoItsRunner(t *testing.T) {
	t.Parallel()
	f := newRunnerFixture(t, tunnelLaptop())

	require.NoError(t, f.svc.HandleFactsReported(t.Context(), factsEvent(t, "answering")))

	stored, err := f.repo.GetComputer(t.Context(), "u-alice", "c-laptop")
	require.NoError(t, err)
	assert.Equal(t, runnerAddress("c-laptop"), stored.ServerURL)
	assert.Equal(t, testNow.Add(30*day), stored.TokenExpiresAt)
	assert.Nil(t, stored.Tunnel, "the tunnel columns are cleared")
	assert.Equal(t, &testNow, stored.SetupConfirmedAt, "setup is kept")
	assert.Equal(t, "sealed-mcp", stored.SetupMCPToken, "the MCP token is kept")
	assert.Equal(t, "Laptop", stored.Name)
	assert.Equal(t, []ComputerTunnel{laptopTunnel}, f.tunnels.deleted, "Cloudflare loses the tunnel, record and Access app")
	assert.Equal(t, []string{TopicComputerPaired, TopicTunnelRemoved, TopicFactsChanged}, topics(f.repo.outbox), "paired first, the tunnel retired after")
}

// TestHandleFactsReported_AFailedMoveLeavesTheTunnelComputerWorking: a pairing through the runner that fails keeps the
// computer on its tunnel, with its session, and asks Cloudflare for nothing.
func TestHandleFactsReported_AFailedMoveLeavesTheTunnelComputerWorking(t *testing.T) {
	t.Parallel()
	before := tunnelLaptop()
	f := newRunnerFixture(t, before)
	f.exch.exchangeErr = errBoom

	require.NoError(t, f.svc.HandleFactsReported(t.Context(), factsEvent(t, "answering")))

	stored, err := f.repo.GetComputer(t.Context(), "u-alice", "c-laptop")
	require.NoError(t, err)
	assert.Equal(t, before.ServerURL, stored.ServerURL)
	assert.Equal(t, "sealed-old", stored.BearerToken)
	assert.Equal(t, before.TokenExpiresAt, stored.TokenExpiresAt)
	assert.Equal(t, &laptopTunnel, stored.Tunnel)
	assert.Empty(t, f.tunnels.deleted)
	assert.Equal(t, []string{TopicFactsChanged}, topics(f.repo.outbox), "only the report itself is stored")
}

// TestHandleFactsReported_ATunnelLeftAfterTheMoveIsRetiredOnTheNextReport: when Cloudflare refuses the teardown the moved
// computer keeps working through its runner and keeps the tunnel, which the next report retires without pairing again.
func TestHandleFactsReported_ATunnelLeftAfterTheMoveIsRetiredOnTheNextReport(t *testing.T) {
	t.Parallel()
	f := newRunnerFixture(t, tunnelLaptop())
	f.tunnels.deleteErr = errBoom

	paired, err := f.svc.PairComputer(t.Context(), "u-alice", "c-laptop", "")

	require.NoError(t, err)
	assert.Equal(t, &laptopTunnel, paired.Tunnel, "the answer says the tunnel is still there")
	stored, err := f.repo.GetComputer(t.Context(), "u-alice", "c-laptop")
	require.NoError(t, err)
	assert.Equal(t, runnerAddress("c-laptop"), stored.ServerURL)
	assert.NotNil(t, stored.Tunnel)

	f.tunnels.deleteErr = nil
	require.NoError(t, f.svc.HandleFactsReported(t.Context(), factsEvent(t, "not_running")))

	stored, err = f.repo.GetComputer(t.Context(), "u-alice", "c-laptop")
	require.NoError(t, err)
	assert.Nil(t, stored.Tunnel)
	assert.Equal(t, 1, f.runners.tokens, "the retry pairs nothing")
}

// TestHandleFactsReported_MovesAComputerPairedByURL: a computer paired at an address moves to its runner the same way,
// with no tunnel to delete.
func TestHandleFactsReported_MovesAComputerPairedByURL(t *testing.T) {
	t.Parallel()
	f := newRunnerFixture(t, Computer{Name: "VPS", ServerURL: "https://vps.example.com:3773", BearerToken: "sealed-old", TokenExpiresAt: testNow.Add(20 * day)})

	require.NoError(t, f.svc.HandleFactsReported(t.Context(), factsEvent(t, "answering")))

	stored, err := f.repo.GetComputer(t.Context(), "u-alice", "c-laptop")
	require.NoError(t, err)
	assert.Equal(t, runnerAddress("c-laptop"), stored.ServerURL)
	assert.Empty(t, f.tunnels.deleted)
	assert.Equal(t, []string{TopicComputerPaired, TopicFactsChanged}, topics(f.repo.outbox))
}

// TestAnnounceMoves_WritesOneReceiptAndNoticePerOwner: each owner the repo names gets a receipt at now with one
// computer.move_requested event, and a failed write stops the sweep with the error.
func TestAnnounceMoves_WritesOneReceiptAndNoticePerOwner(t *testing.T) {
	t.Parallel()
	f := newRunnerFixture(t, tunnelLaptop())
	f.repo.toMove = []Computer{{ID: "c-laptop", UserID: "u-alice"}, {ID: "c-desk", UserID: "u-bob"}}

	require.NoError(t, f.svc.AnnounceMoves(t.Context()))

	assert.Equal(t, map[string]time.Time{"u-alice": testNow, "u-bob": testNow}, f.repo.moveNotices)
	require.Len(t, f.repo.outbox, 2)
	assert.Equal(t, MoveRequestedEvent{ComputerID: "c-desk", UserID: "u-bob", MembersOnly: true}, f.repo.outbox[1].Payload)

	f.repo.saveErr = errBoom
	require.ErrorIs(t, f.svc.AnnounceMoves(t.Context()), errBoom)
}
