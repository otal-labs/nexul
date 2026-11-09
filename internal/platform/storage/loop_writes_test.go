package storage

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"modernc.org/sqlite"

	"github.com/otal-labs/nexul/internal/automations"
	"github.com/otal-labs/nexul/internal/platform/wake"
	"github.com/otal-labs/nexul/internal/runner"
)

// tablesRead records every table a statement compiled on the database reads.
type tablesRead struct {
	mu sync.Mutex
	m  map[string]bool
}

func (r *tablesRead) authorize(action sqlite.AuthorizerActionCode, table, _, _, _ string) sqlite.AuthorizerReturnCode {
	if action == sqlite.AuthRead {
		r.mu.Lock()
		r.m[table] = true
		r.mu.Unlock()
	}
	return sqlite.AuthorizerOK
}

func (r *tablesRead) sorted() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, 0, len(r.m))
	for t := range r.m {
		out = append(out, t)
	}
	slices.Sort(out)
	return out
}

type authorizingConnector struct {
	driver.Connector
	read *tablesRead
}

func (c authorizingConnector) Connect(ctx context.Context) (driver.Conn, error) {
	conn, err := c.Connector.Connect(ctx)
	if err != nil {
		return nil, err
	}
	return conn, conn.(sqlite.AuthorizerRegisterer).RegisterAuthorizer(c.read.authorize)
}

// newReadRecordingStore is newTestStore whose connections record the tables every statement reads.
func newReadRecordingStore(t *testing.T) (*Store, *tablesRead) {
	t.Helper()
	data, err := os.ReadFile(migratedTemplate(t))
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "test.db")
	require.NoError(t, os.WriteFile(path, data, 0o600))
	inner, err := sqlite.NewConnector(path + "?" + pragmas)
	require.NoError(t, err)
	read := &tablesRead{m: map[string]bool{}}
	db := sql.OpenDB(authorizingConnector{inner, read})
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	return New(db, testEncKey), read
}

func TestLoopWrites_CoverEveryTableTheLoopsPassReads(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name   string
		writes []TableWrite
		// exempt is a table the pass reads whose writes never give the loop work.
		exempt []string
		pass   func(ctx context.Context, s *Store)
	}{
		{"outbox relay", OutboxRelayWrites, nil, func(ctx context.Context, s *Store) {
			_, _ = s.Outbox.Unpublished(ctx, 100)
		}},
		{"webhook relay", WebhookRelayWrites, nil, func(ctx context.Context, s *Store) {
			_, _ = s.IntegrationDeliveries.Due(ctx, 50, now.Unix())
		}},
		{"automation dial-in", DialinWrites, []string{"automation_cursors"}, func(ctx context.Context, s *Store) {
			_, _ = s.Automations.Get(ctx, "auto-acme")
			_, _, _ = s.AutomationCursors.Get(ctx, "auto-acme")
			_, _ = s.AutomationEventLog.After(ctx, []string{"ticket.created"}, automations.Cursor{}, 50)
		}},
		{"doc settle", SettleWrites, nil, func(ctx context.Context, s *Store) {
			_, _, _ = s.Docs.NextSettleDue(ctx)
		}},
		{"calendar reminders", ReminderWrites, nil, func(ctx context.Context, s *Store) {
			_, _ = s.Meetings.RemindersDue(ctx, now)
			_, _ = s.Meetings.Replacements(ctx, []string{"meeting-standup"})
			_, _, _ = s.Meetings.NextReminder(ctx)
			_, _ = s.Meetings.DueTickets(ctx, "2026-01-01", "2026-12-31")
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, read := newReadRecordingStore(t)
			ctx := t.Context()
			_, err := s.db.ExecContext(ctx, `INSERT INTO meetings (id, workspace_id, organiser_id, title, starts_at, ends_at, repeats, next_remind_at, created_at, updated_at)
				SELECT 'meeting-standup', id, 'alice', 'Standup', ?, ?, 'weekly', ?, ?, ? FROM workspaces LIMIT 1`,
				now.Unix(), now.Add(time.Hour).Unix(), now.Add(-time.Minute).Unix(), now.Unix(), now.Unix())
			require.NoError(t, err)
			stored := storedTables(t, s.db)
			clear(read.m)

			tt.pass(ctx, s)

			watched := map[string]bool{}
			for _, w := range tt.writes {
				watched[w.table] = true
			}
			tables := slices.DeleteFunc(read.sorted(), func(table string) bool { return !stored[table] })
			require.NotEmpty(t, tables)
			for _, table := range tables {
				assert.True(t, watched[table] || slices.Contains(tt.exempt, table), "the pass reads %s, so a write to it must wake the loop", table)
			}
		})
	}
}

// storedTables is every table the schema stores rows in, leaving out table-valued functions such as json_each.
func storedTables(t *testing.T, db *sql.DB) map[string]bool {
	t.Helper()
	rows, err := db.QueryContext(t.Context(), `SELECT name FROM sqlite_schema WHERE type = 'table'`)
	require.NoError(t, err)
	defer func() { require.NoError(t, rows.Close()) }()
	out := map[string]bool{}
	for rows.Next() {
		var name string
		require.NoError(t, rows.Scan(&name))
		out[name] = true
	}
	require.NoError(t, rows.Err())
	return out
}

func TestStore_CommitsTo_AMachineTouchWakesNoLoopButTheMemo(t *testing.T) {
	s := newTestStore(t)
	ctx := t.Context()
	now := time.Now()
	require.NoError(t, s.Machines.Create(ctx, &runner.Machine{ID: "machine-acme", Name: "acme-box", FirstSeen: now, LastSeen: now}))
	loops := map[string]*wake.Broadcast{
		"outbox relay":       s.CommitsTo(OutboxRelayWrites...),
		"webhook relay":      s.CommitsTo(WebhookRelayWrites...),
		"automation dial-in": s.CommitsTo(DialinWrites...),
		"doc settle":         s.CommitsTo(SettleWrites...),
		"calendar reminders": s.CommitsTo(ReminderWrites...),
	}
	waits := map[string]<-chan struct{}{}
	for name, b := range loops {
		waits[name] = b.Next()
	}
	memo := s.Commits().Next()

	require.NoError(t, s.Machines.Touch(ctx, "machine-acme", "acme-box", now))

	assert.True(t, woken(memo), "every commit clears the access memo")
	for name, ch := range waits {
		assert.False(t, woken(ch), "a heartbeat touch woke the %s", name)
	}
}

func TestStore_CommitsTo_WakesOnTheWritesItNamesOnly(t *testing.T) {
	s := newTestStore(t)
	ctx := t.Context()
	inserts := s.CommitsTo(Inserts("outbox"))
	writes := s.CommitsTo(Writes("outbox"))
	write := func(stmt string, fail error) error {
		return s.w.WithTx(ctx, s.db, func(tx *sql.Tx) error {
			if _, err := tx.ExecContext(ctx, stmt); err != nil {
				return err
			}
			return fail
		})
	}

	ins, wr := inserts.Next(), writes.Next()
	require.Error(t, write(`INSERT INTO outbox (id, topic, payload) VALUES ('ev-rolled-back', 'ticket.created', x'7b7d')`, assert.AnError))
	assert.False(t, woken(ins), "a rollback wakes nobody")
	require.NoError(t, write(`INSERT INTO outbox (id, topic, payload) VALUES ('ev-1', 'ticket.created', x'7b7d')`, nil))
	assert.True(t, woken(ins), "an insert wakes an inserts watch")
	assert.True(t, woken(wr), "an insert wakes a writes watch")

	ins, wr = inserts.Next(), writes.Next()
	require.NoError(t, write(`UPDATE outbox SET published = 1 WHERE id = 'ev-1'`, nil))
	assert.False(t, woken(ins), "an update leaves an inserts watch asleep")
	assert.True(t, woken(wr), "an update wakes a writes watch")
}

func woken(ch <-chan struct{}) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}
