// This file exercises the dial-in transport end to end over a real
// websocket and a real SQLite store, so it lives in an external test
// package (automations_test): internal/platform/storage already imports
// internal/automations for AutomationsRepo, so a white-box test file here
// importing storage back would be an import cycle.
package automations_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/automations"
	"github.com/otal-labs/nexul/internal/platform/eventbus/testutil"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/platform/storage"
)

// allowAllPerm grants every automation.* action — these tests exercise the
// transport, not the access domain's own rules (already covered in
// usecase_test.go).
type allowAllPerm struct{}

func (allowAllPerm) HasPermission(context.Context, string, permissions.Action) bool { return true }

func newTestStore(t *testing.T) (*sql.DB, *storage.Store) {
	t.Helper()
	db, err := storage.OpenDB(filepath.Join(t.TempDir(), "test.db"))
	require.NoError(t, err)
	require.NoError(t, storage.Migrate(db))
	t.Cleanup(func() { _ = db.Close() })
	return db, storage.New(db, []byte("0123456789abcdef0123456789abcdef"))
}

type dialinSetup struct {
	db  *sql.DB
	svc *automations.Service
	dh  *automations.DialinHandler
	srv *httptest.Server
}

func newDialinSetup(t *testing.T) *dialinSetup {
	t.Helper()
	db, store := newTestStore(t)
	svc := automations.NewService(store.Automations, allowAllPerm{})
	dh := automations.NewDialinHandler(svc, automations.DialinConfig{
		Repo:         store.Automations,
		Cursors:      store.AutomationCursors,
		EventLog:     store.AutomationEventLog,
		Runs:         store.AutomationRuns,
		Logger:       testutil.DiscardLogger(),
		PollInterval: 10 * time.Millisecond,
	})
	svc.SetConnectionRegistry(dh)
	srv := httptest.NewServer(dh)
	t.Cleanup(srv.Close)
	return &dialinSetup{db: db, svc: svc, dh: dh, srv: srv}
}

func (s *dialinSetup) wsURL(token string) string {
	return "ws" + strings.TrimPrefix(s.srv.URL, "http") + "/?token=" + token
}

// seedEvent inserts a durable log row directly (bypassing the relay, which
// is irrelevant to automations' own cursor-based reads over the same table).
func (s *dialinSetup) seedEvent(t *testing.T, id, topic, payload string, createdAt time.Time) {
	t.Helper()
	_, err := s.db.Exec(`INSERT INTO outbox (id, topic, payload, created_at, published) VALUES (?, ?, ?, ?, 0)`,
		id, topic, []byte(payload), createdAt.Unix())
	require.NoError(t, err)
}

// dialAnnounce connects, sends the announce frame, and consumes the
// resulting hello — the handshake every scenario starts from.
func dialAnnounce(t *testing.T, url string, announce automations.Frame) *websocket.Conn {
	t.Helper()
	announce.Type = automations.FrameAnnounce
	if announce.Name == "" {
		announce.Name = "Test Automation"
	}
	conn, _, err := websocket.Dial(context.Background(), url, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "") })

	require.NoError(t, wsjson.Write(context.Background(), conn, announce))
	var hello automations.Frame
	require.NoError(t, wsjson.Read(context.Background(), conn, &hello))
	require.Equal(t, automations.FrameHello, hello.Type)
	return conn
}

func TestDialinHandler_ServeHTTP_InvalidToken_Refuses(t *testing.T) {
	setup := newDialinSetup(t)
	resp, err := http.Get(setup.srv.URL + "/?token=bogus")
	require.NoError(t, err)
	defer func() { require.NoError(t, resp.Body.Close()) }()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestDialinHandler_ServeHTTP_NoToken_Refuses(t *testing.T) {
	setup := newDialinSetup(t)
	resp, err := http.Get(setup.srv.URL + "/")
	require.NoError(t, err)
	defer func() { require.NoError(t, resp.Body.Close()) }()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestDialinHandler_ServeHTTP_RevokedToken_Refuses(t *testing.T) {
	setup := newDialinSetup(t)
	created, token, err := setup.svc.Create(context.Background(), "owner", "x", []string{"tickets:read"})
	require.NoError(t, err)
	_, err = setup.svc.RevokeToken(context.Background(), "owner", created.ID)
	require.NoError(t, err)

	resp, err := http.Get(setup.srv.URL + "/?token=" + token)
	require.NoError(t, err)
	defer func() { require.NoError(t, resp.Body.Close()) }()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestDialinHandler_Announce_SyncsCodeAndHelloCarriesConfig(t *testing.T) {
	setup := newDialinSetup(t)
	created, token, err := setup.svc.Create(context.Background(), "owner", "x", []string{"tickets:read"})
	require.NoError(t, err)
	_, err = setup.svc.UpdateConfigValues(context.Background(), "owner", created.ID, json.RawMessage(`{"status":"done"}`))
	require.NoError(t, err)

	conn, _, err := websocket.Dial(context.Background(), setup.wsURL(token), nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "") })

	require.NoError(t, wsjson.Write(context.Background(), conn, automations.Frame{
		Type:          automations.FrameAnnounce,
		Name:          "PR notifier",
		Description:   "notifies on PR open",
		Subscriptions: []string{"ticket.created"},
		ConfigSchema:  json.RawMessage(`{"status":{"type":"string"}}`),
	}))
	var hello automations.Frame
	require.NoError(t, wsjson.Read(context.Background(), conn, &hello))
	assert.Equal(t, automations.FrameHello, hello.Type)
	assert.JSONEq(t, `{"status":"done"}`, string(hello.ConfigValues))

	got, err := setup.svc.Get(context.Background(), "owner", created.ID)
	require.NoError(t, err)
	assert.Equal(t, "PR notifier", got.Name)
	assert.Equal(t, "notifies on PR open", got.Description)
	assert.Equal(t, []string{"ticket.created"}, got.Subscriptions)
}

func TestDialinHandler_AnnounceNotFirst_ClosesConnection(t *testing.T) {
	setup := newDialinSetup(t)
	_, token, err := setup.svc.Create(context.Background(), "owner", "x", []string{"tickets:read"})
	require.NoError(t, err)

	conn, _, err := websocket.Dial(context.Background(), setup.wsURL(token), nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "") })

	// run_started before announce is a protocol violation.
	require.NoError(t, wsjson.Write(context.Background(), conn, automations.Frame{Type: automations.FrameRunStarted, RunID: "r1"}))

	var f automations.Frame
	err = wsjson.Read(context.Background(), conn, &f)
	require.Error(t, err)
}

func TestDialinHandler_DeliversSubscribedEvent_RunSuccess_AdvancesCursor(t *testing.T) {
	setup := newDialinSetup(t)
	created, token, err := setup.svc.Create(context.Background(), "owner", "x", []string{"tickets:read"})
	require.NoError(t, err)
	_, err = setup.svc.SetEnabled(context.Background(), "owner", created.ID, true)
	require.NoError(t, err)

	conn := dialAnnounce(t, setup.wsURL(token), automations.Frame{Subscriptions: []string{"ticket.created"}})

	// Seeded after connect: the automation's cursor was established at
	// connect time on an empty log, so anything published from here on is
	// "after" it.
	setup.seedEvent(t, "ev-1", "ticket.created", `{"id":"t1"}`, time.Now())

	var ev automations.Frame
	require.NoError(t, wsjson.Read(context.Background(), conn, &ev))
	require.Equal(t, automations.FrameEvent, ev.Type)
	assert.Equal(t, "ticket.created", ev.Topic)
	assert.Equal(t, "ev-1", ev.EventID)
	assert.JSONEq(t, `{"id":"t1"}`, string(ev.Payload))
	require.NotEmpty(t, ev.RunID)

	require.NoError(t, wsjson.Write(context.Background(), conn, automations.Frame{Type: automations.FrameRunStarted, RunID: ev.RunID}))
	require.NoError(t, wsjson.Write(context.Background(), conn, automations.Frame{Type: automations.FrameRunLog, RunID: ev.RunID, Log: "handled\n"}))
	require.NoError(t, wsjson.Write(context.Background(), conn, automations.Frame{Type: automations.FrameRunFinished, RunID: ev.RunID, Outcome: automations.OutcomeSuccess}))

	store := storage.New(setup.db, []byte("0123456789abcdef0123456789abcdef"))
	var runs []automations.Run
	require.Eventually(t, func() bool {
		var err error
		runs, err = store.AutomationRuns.ListByAutomation(context.Background(), created.ID, 10)
		return err == nil && len(runs) == 1
	}, 2*time.Second, 10*time.Millisecond)
	assert.Equal(t, automations.RunOutcomeSuccess, runs[0].Outcome)
	assert.Equal(t, "ev-1", runs[0].EventID)
	assert.Equal(t, "handled\n", runs[0].Logs)

	_, ok, err := store.AutomationCursors.Get(context.Background(), created.ID)
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestDialinHandler_DisabledAutomation_NoDelivery(t *testing.T) {
	setup := newDialinSetup(t)
	_, token, err := setup.svc.Create(context.Background(), "owner", "x", []string{"tickets:read"})
	require.NoError(t, err)
	// left disabled

	conn := dialAnnounce(t, setup.wsURL(token), automations.Frame{Subscriptions: []string{"ticket.created"}})
	setup.seedEvent(t, "ev-1", "ticket.created", `{}`, time.Now())

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	var f automations.Frame
	err = wsjson.Read(ctx, conn, &f)
	assert.Error(t, err, "a disabled automation must not receive delivery")
}

func TestDialinHandler_UnsubscribedTopic_NoDelivery(t *testing.T) {
	setup := newDialinSetup(t)
	created, token, err := setup.svc.Create(context.Background(), "owner", "x", []string{"tickets:read"})
	require.NoError(t, err)
	_, err = setup.svc.SetEnabled(context.Background(), "owner", created.ID, true)
	require.NoError(t, err)

	conn := dialAnnounce(t, setup.wsURL(token), automations.Frame{Subscriptions: []string{"ticket.created"}})
	setup.seedEvent(t, "ev-1", "doc.created", `{}`, time.Now())

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	var f automations.Frame
	err = wsjson.Read(ctx, conn, &f)
	assert.Error(t, err, "an event on an unsubscribed topic must not be delivered")
}

// TestDialinHandler_CrashMidRun_RedeliveredOnReconnect is the reconnect/
// cursor-resume regression: a run that never reports a terminal frame
// because the connection dies must not advance the cursor, so the very same
// event is redelivered once the automation reconnects (ADR 0046).
func TestDialinHandler_CrashMidRun_RedeliveredOnReconnect(t *testing.T) {
	setup := newDialinSetup(t)
	created, token, err := setup.svc.Create(context.Background(), "owner", "x", []string{"tickets:read"})
	require.NoError(t, err)
	_, err = setup.svc.SetEnabled(context.Background(), "owner", created.ID, true)
	require.NoError(t, err)

	conn1 := dialAnnounce(t, setup.wsURL(token), automations.Frame{Subscriptions: []string{"ticket.created"}})
	setup.seedEvent(t, "ev-1", "ticket.created", `{"id":"t1"}`, time.Now())

	var ev1 automations.Frame
	require.NoError(t, wsjson.Read(context.Background(), conn1, &ev1))
	require.Equal(t, "ev-1", ev1.EventID)

	require.NoError(t, wsjson.Write(context.Background(), conn1, automations.Frame{Type: automations.FrameRunStarted, RunID: ev1.RunID}))
	// The connection dies mid-run, no close handshake: no run_finished/
	// run_crashed ever arrives, exactly like a killed automation process.
	require.NoError(t, conn1.CloseNow())

	store := storage.New(setup.db, []byte("0123456789abcdef0123456789abcdef"))
	require.Eventually(t, func() bool {
		runs, err := store.AutomationRuns.ListByAutomation(context.Background(), created.ID, 10)
		return err == nil && len(runs) == 1 && runs[0].Outcome == automations.RunOutcomeCrash
	}, 2*time.Second, 10*time.Millisecond)

	// The cursor must not have advanced past ev-1: reconnecting redelivers it.
	conn2 := dialAnnounce(t, setup.wsURL(token), automations.Frame{Subscriptions: []string{"ticket.created"}})
	var ev2 automations.Frame
	require.NoError(t, wsjson.Read(context.Background(), conn2, &ev2))
	assert.Equal(t, "ev-1", ev2.EventID, "the crashed event must be redelivered, not skipped")

	require.NoError(t, wsjson.Write(context.Background(), conn2, automations.Frame{Type: automations.FrameRunStarted, RunID: ev2.RunID}))
	require.NoError(t, wsjson.Write(context.Background(), conn2, automations.Frame{Type: automations.FrameRunFinished, RunID: ev2.RunID, Outcome: automations.OutcomeSuccess}))

	require.Eventually(t, func() bool {
		runs, err := store.AutomationRuns.ListByAutomation(context.Background(), created.ID, 10)
		return err == nil && len(runs) == 2
	}, 2*time.Second, 10*time.Millisecond)
}

func TestDialinHandler_RunCrashedFrame_DoesNotAdvanceCursor(t *testing.T) {
	setup := newDialinSetup(t)
	created, token, err := setup.svc.Create(context.Background(), "owner", "x", []string{"tickets:read"})
	require.NoError(t, err)
	_, err = setup.svc.SetEnabled(context.Background(), "owner", created.ID, true)
	require.NoError(t, err)

	conn := dialAnnounce(t, setup.wsURL(token), automations.Frame{Subscriptions: []string{"ticket.created"}})
	setup.seedEvent(t, "ev-1", "ticket.created", `{}`, time.Now())

	var ev automations.Frame
	require.NoError(t, wsjson.Read(context.Background(), conn, &ev))
	require.NoError(t, wsjson.Write(context.Background(), conn, automations.Frame{Type: automations.FrameRunStarted, RunID: ev.RunID}))
	require.NoError(t, wsjson.Write(context.Background(), conn, automations.Frame{Type: automations.FrameRunCrashed, RunID: ev.RunID, Error: "panic: boom"}))

	store := storage.New(setup.db, []byte("0123456789abcdef0123456789abcdef"))
	require.Eventually(t, func() bool {
		runs, err := store.AutomationRuns.ListByAutomation(context.Background(), created.ID, 10)
		return err == nil && len(runs) == 1 && runs[0].Outcome == automations.RunOutcomeCrash && runs[0].Error == "panic: boom"
	}, 2*time.Second, 10*time.Millisecond)

	// The connect-time cursor (established on the empty log, before ev-1 was
	// seeded) must be unchanged: a crash never advances it past the event it
	// crashed on.
	got, ok, err := store.AutomationCursors.Get(context.Background(), created.ID)
	require.NoError(t, err)
	require.True(t, ok, "connect always establishes a baseline cursor")
	assert.Empty(t, got.ID, "a crash must not advance the cursor past the event it crashed on")
}

func TestDialinHandler_RunFinishedFailure_StillAdvancesCursor(t *testing.T) {
	// A caught exception/falsy return is a clean, reported failure —
	// the event was fully processed, so it must not be redelivered forever.
	setup := newDialinSetup(t)
	created, token, err := setup.svc.Create(context.Background(), "owner", "x", []string{"tickets:read"})
	require.NoError(t, err)
	_, err = setup.svc.SetEnabled(context.Background(), "owner", created.ID, true)
	require.NoError(t, err)

	conn := dialAnnounce(t, setup.wsURL(token), automations.Frame{Subscriptions: []string{"ticket.created"}})
	setup.seedEvent(t, "ev-1", "ticket.created", `{}`, time.Now())

	var ev automations.Frame
	require.NoError(t, wsjson.Read(context.Background(), conn, &ev))
	require.NoError(t, wsjson.Write(context.Background(), conn, automations.Frame{Type: automations.FrameRunFinished, RunID: ev.RunID, Outcome: automations.OutcomeFailure}))

	store := storage.New(setup.db, []byte("0123456789abcdef0123456789abcdef"))
	require.Eventually(t, func() bool {
		_, ok, err := store.AutomationCursors.Get(context.Background(), created.ID)
		return err == nil && ok
	}, 2*time.Second, 10*time.Millisecond)
}

func TestDialinHandler_UnexpectedFrameFromAutomation_LoggedNotFatal(t *testing.T) {
	setup := newDialinSetup(t)
	_, token, err := setup.svc.Create(context.Background(), "owner", "x", []string{"tickets:read"})
	require.NoError(t, err)

	conn := dialAnnounce(t, setup.wsURL(token), automations.Frame{})
	// A second announce is not one of the expected inbound types post-
	// handshake; dispatchFrame reports and continues rather than closing.
	require.NoError(t, wsjson.Write(context.Background(), conn, automations.Frame{Type: automations.FrameAnnounce, Name: "again"}))
	// Connection should still be alive: closing it now should be a clean
	// client-initiated close, not an already-dead one.
	require.NoError(t, conn.Close(websocket.StatusNormalClosure, ""))
}

func TestDialinHandler_RevokeWhileConnected_DisconnectsSocket(t *testing.T) {
	setup := newDialinSetup(t)
	created, token, err := setup.svc.Create(context.Background(), "owner", "x", []string{"tickets:read"})
	require.NoError(t, err)

	conn := dialAnnounce(t, setup.wsURL(token), automations.Frame{})

	_, err = setup.svc.RevokeToken(context.Background(), "owner", created.ID)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	var f automations.Frame
	err = wsjson.Read(ctx, conn, &f)
	require.Error(t, err, "a revoked token must force-close the live connection")
}

func TestDialinHandler_CloseAll_ClosesEveryLiveConnection(t *testing.T) {
	setup := newDialinSetup(t)
	_, token1, err := setup.svc.Create(context.Background(), "owner", "a", []string{"tickets:read"})
	require.NoError(t, err)
	_, token2, err := setup.svc.Create(context.Background(), "owner", "b", []string{"tickets:read"})
	require.NoError(t, err)

	conn1 := dialAnnounce(t, setup.wsURL(token1), automations.Frame{})
	conn2 := dialAnnounce(t, setup.wsURL(token2), automations.Frame{})

	setup.dh.CloseAll("shutdown")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	var f automations.Frame
	require.Error(t, wsjson.Read(ctx, conn1, &f))
	require.Error(t, wsjson.Read(ctx, conn2, &f))
}

func TestDialinHandler_AnnounceBlankAfterTrim_ClosesConnection(t *testing.T) {
	// The wire-level Frame.Validate only rejects an empty string; SyncFromCode
	// applies TrimSpace and rejects whitespace-only names, so an
	// otherwise-well-formed announce can still fail sync (a different branch
	// than "no announce frame at all").
	setup := newDialinSetup(t)
	_, token, err := setup.svc.Create(context.Background(), "owner", "x", []string{"tickets:read"})
	require.NoError(t, err)

	conn, _, err := websocket.Dial(context.Background(), setup.wsURL(token), nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "") })

	require.NoError(t, wsjson.Write(context.Background(), conn, automations.Frame{Type: automations.FrameAnnounce, Name: "   "}))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	var f automations.Frame
	require.Error(t, wsjson.Read(ctx, conn, &f))
}

func TestDialinHandler_NewConnectionReplacesOld(t *testing.T) {
	setup := newDialinSetup(t)
	_, token, err := setup.svc.Create(context.Background(), "owner", "x", []string{"tickets:read"})
	require.NoError(t, err)

	conn1 := dialAnnounce(t, setup.wsURL(token), automations.Frame{})
	conn2 := dialAnnounce(t, setup.wsURL(token), automations.Frame{})
	_ = conn2

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	var f automations.Frame
	err = wsjson.Read(ctx, conn1, &f)
	require.Error(t, err, "the older connection must be replaced, not left dangling")
}
