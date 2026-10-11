package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/pairing"
)

// TestMigration0095_EachOwnerOfAnOldComputerIsToldOnce upgrades a database holding computers paired through a tunnel, by
// URL and through a runner: the move notice reaches the two owners still on the old paths once each, naming their
// oldest such computer, and neither a second announcement nor a restart writes another; their rows survive whole.
func TestMigration0095_EachOwnerOfAnOldComputerIsToldOnce(t *testing.T) {
	db := migrateBefore(t, "0095")
	_, err := db.Exec(`
INSERT INTO users (id, login, created_at, updated_at) VALUES ('u-alice', 'alice', 1, 1), ('u-bob', 'bob', 1, 1), ('u-sam', 'sam', 1, 1);
INSERT INTO pairing_computers (id, user_id, kind, name, server_url, bearer_token, token_expires_at, harness_version, created_at, updated_at,
  tunnel_id, tunnel_hostname, tunnel_zone_id, tunnel_record_id, tunnel_access_app_id, setup_confirmed_at) VALUES
  ('c-laptop', 'u-alice', 't3code', 'Laptop', 'https://laptop-ab12cd34.example.com', 'sealed', 4102444800, '0.0.40', 10, 10,
   'tun-1', 'laptop-ab12cd34.example.com', 'z1', 'rec-1', 'app-1', 20),
  ('c-vps', 'u-alice', 't3code', 'VPS', 'https://vps.example.com:3773', 'sealed', 4102444800, '0.0.40', 30, 30, '', '', '', '', '', NULL),
  ('c-desk', 'u-bob', 't3code', 'Desk', 'https://desk.example.com:3773', 'sealed', 4102444800, '0.0.40', 40, 40, '', '', '', '', '', NULL),
  ('c-new', 'u-sam', 't3code-v2', 'New', 'http://c-new.nexul-computer.invalid', 'sealed', 4102444800, '0.0.46', 50, 50, '', '', '', '', '', NULL),
  ('c-adding', 'u-sam', 't3code', '', '', '', 0, '', 60, 60, '', '', '', '', '', NULL);`)
	require.NoError(t, err)
	require.NoError(t, Migrate(db), "0095 and every later migration apply on top, as an upgrade would")
	s := New(db, []byte("0123456789abcdef0123456789abcdef"))
	ctx := t.Context()
	first := time.Unix(1_800_000_000, 0).UTC()
	now := first
	svc := pairing.NewService(pairing.Config{Repo: s.Pairing, Now: func() time.Time { return now }})

	require.NoError(t, svc.AnnounceMoves(ctx))
	now = first.Add(48 * time.Hour)
	require.NoError(t, svc.AnnounceMoves(ctx), "a restart announces again")

	assert.Equal(t, 2, count(t, db, `SELECT COUNT(*) FROM outbox WHERE topic = ?`, pairing.TopicMoveRequested), "one notice per owner, not per computer or restart")
	assert.Equal(t, 1, count(t, db, `SELECT COUNT(*) FROM outbox WHERE topic = ? AND json_extract(payload, '$.computer_id') = 'c-laptop'`, pairing.TopicMoveRequested))
	assert.Equal(t, 1, count(t, db, `SELECT COUNT(*) FROM outbox WHERE topic = ? AND json_extract(payload, '$.computer_id') = 'c-desk'`, pairing.TopicMoveRequested))
	assert.Equal(t, 2, count(t, db, `SELECT COUNT(*) FROM pairing_move_notices WHERE notified_at = ?`, first.Unix()), "the receipt keeps its first time")

	laptop, err := s.Pairing.GetComputer(ctx, "u-alice", "c-laptop")
	require.NoError(t, err)
	require.NotNil(t, laptop.Tunnel)
	assert.Equal(t, "rec-1", laptop.Tunnel.RecordID)
	assert.NotNil(t, laptop.SetupConfirmedAt)
}
