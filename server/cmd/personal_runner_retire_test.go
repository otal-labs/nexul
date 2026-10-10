package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/runner"
)

// installedComputer adds a computer for alice, enrolls its runner with the command's token and connects it the way
// the runner does; the channel closes once the server tells it to uninstall.
func installedComputer(t *testing.T, c privacyCast, srv *httptest.Server, name string) (id, credential string, uninstalled <-chan struct{}) {
	t.Helper()
	id, token := addComputer(t, c, c.alice, name)
	rec := callAPI(t, c.h, http.MethodPost, "/api/runners/enroll", "", `{"token":"`+token+`","machine":"alice-laptop","os":"linux","arch":"amd64"}`)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var enrolled struct {
		Credential string `json:"credential"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &enrolled))

	conn, _, err := websocket.Dial(t.Context(), "ws"+strings.TrimPrefix(srv.URL, "http")+"/ws/runner",
		&websocket.DialOptions{HTTPHeader: http.Header{"Authorization": {"Bearer " + enrolled.Credential}}})
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.CloseNow() })
	done := make(chan struct{})
	go func() {
		for {
			var f runner.Frame
			if err := wsjson.Read(t.Context(), conn, &f); err != nil {
				return
			}
			if f.Type == runner.FrameUninstall {
				close(done)
				return
			}
		}
	}()
	require.Eventually(t, func() bool {
		r, err := c.store.Runners.GetByComputer(t.Context(), id)
		return err == nil && r.Connected
	}, 5*time.Second, 10*time.Millisecond)
	return id, enrolled.Credential, done
}

// requireRetired holds the runner to being gone: told to uninstall while connected, refused as removed on its next
// connect, and no longer enrolled for its computer.
func requireRetired(t *testing.T, c privacyCast, computerID, credential string, uninstalled <-chan struct{}) {
	t.Helper()
	select {
	case <-uninstalled:
	case <-time.After(5 * time.Second):
		t.Fatal("the connected runner was never told to uninstall")
	}
	rec := callAPI(t, c.h, http.MethodGet, "/ws/runner", credential, "")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.JSONEq(t, runner.RemovedRefusal, rec.Body.String(), "an offline runner uninstalls itself on its next connect")
	_, err := c.store.Runners.GetByComputer(t.Context(), computerID)
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

// requireEnrollsNothing: a command minted before the retirement can no longer add a runner.
func requireEnrollsNothing(t *testing.T, c privacyCast, token string) {
	t.Helper()
	rec := callAPI(t, c.h, http.MethodPost, "/api/runners/enroll", "", `{"token":"`+token+`","machine":"alice-desk"}`)
	assert.Equal(t, http.StatusUnauthorized, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), "invalid_code")
}

// TestRemoveComputer_RevokesItsRunner: removing a computer through the route the app uses tombstones its runner's
// credential and tells the runner to uninstall, and a command minted for a computer removed before its install
// enrolls nothing.
func TestRemoveComputer_RevokesItsRunner(t *testing.T) {
	c, _ := newPrivacyCast(t)
	srv := httptest.NewServer(c.h)
	t.Cleanup(srv.Close)
	id, credential, uninstalled := installedComputer(t, c, srv, "Alice laptop")

	rec := callAPI(t, c.h, http.MethodDelete, "/api/pairing/computers/"+id, c.alice, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	requireRetired(t, c, id, credential, uninstalled)

	pending, token := addComputer(t, c, c.alice, "Alice desk")
	rec = callAPI(t, c.h, http.MethodDelete, "/api/pairing/computers/"+pending, c.alice, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	requireEnrollsNothing(t, c, token)
}

// outboxSince returns every outbox payload written after rowid, with its topic.
func outboxSince(t *testing.T, db *sql.DB, rowid int64) string {
	t.Helper()
	rows, err := db.Query(`SELECT topic, payload FROM outbox WHERE rowid > ?`, rowid)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	var out strings.Builder
	for rows.Next() {
		var topic string
		var payload []byte
		require.NoError(t, rows.Scan(&topic, &payload))
		out.WriteString(topic + " " + string(payload) + "\n")
	}
	require.NoError(t, rows.Err())
	return out.String()
}

func lastOutboxRow(t *testing.T, db *sql.DB) int64 {
	t.Helper()
	var rowid int64
	require.NoError(t, db.QueryRow(`SELECT COALESCE(MAX(rowid), 0) FROM outbox`).Scan(&rowid))
	return rowid
}

// requireTellsNothing: what the admin's action wrote for others to read names no computer of alice's.
func requireTellsNothing(t *testing.T, c privacyCast, db *sql.DB, since int64, response string, computerIDs ...string) {
	t.Helper()
	written := outboxSince(t, db, since)
	assert.Contains(t, written, "u-alice", "the account event itself is written")
	for _, id := range computerIDs {
		assert.NotContains(t, response, id, "the admin's response")
		assert.NotContains(t, written, id, "an event the admin's action wrote")
		for _, row := range auditRows(t, c.store) {
			assert.NotContains(t, row.Action, id, "an audit row")
		}
	}
	assert.NotContains(t, response, "Alice laptop")
	assert.NotContains(t, written, "Alice laptop")
}

// TestAccountDisabled_RevokesTheirPersonalRunners: the workspace Owner disabling alice through the app's route retires
// her computer's runner and voids her unused command, and learns nothing about either; reactivating her brings no
// runner back, and her computer's row stays for her to add it again.
func TestAccountDisabled_RevokesTheirPersonalRunners(t *testing.T) {
	c, db := newPrivacyCast(t)
	srv := httptest.NewServer(c.h)
	t.Cleanup(srv.Close)
	id, credential, uninstalled := installedComputer(t, c, srv, "Alice laptop")
	pending, token := addComputer(t, c, c.alice, "Alice desk")
	since := lastOutboxRow(t, db)

	rec := callAPI(t, c.h, http.MethodPatch, "/api/auth/accounts/u-alice", c.owner, `{"status":"disabled"}`)
	require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
	requireRetired(t, c, id, credential, uninstalled)
	requireEnrollsNothing(t, c, token)
	requireTellsNothing(t, c, db, since, rec.Body.String(), id, pending)

	rec = callAPI(t, c.h, http.MethodPatch, "/api/auth/accounts/u-alice", c.owner, `{"status":"active"}`)
	require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
	_, err := c.store.Runners.GetByComputer(t.Context(), id)
	require.ErrorIs(t, err, apperrs.ErrNotFound, "reactivating restores no runner")
	rec = callAPI(t, c.h, http.MethodGet, "/api/pairing/computers", c.alice, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"name":"Alice laptop"`, "her computer's row stays")
	assert.NotContains(t, rec.Body.String(), `"runner":`)
	rec = callAPI(t, c.h, http.MethodPost, "/api/pairing/computers/enrollments", c.alice, `{"id":"`+id+`"}`)
	assert.Equal(t, http.StatusCreated, rec.Code, "she adds the same computer again: %s", rec.Body.String())
}

// TestAccountRemoved_RevokesTheirPersonalRunners: removing alice's account over MCP retires her computer's runner and
// voids her unused command; the tool's answer and what it wrote name neither.
func TestAccountRemoved_RevokesTheirPersonalRunners(t *testing.T) {
	c, db := newPrivacyCast(t)
	srv := httptest.NewServer(c.h)
	t.Cleanup(srv.Close)
	id, credential, uninstalled := installedComputer(t, c, srv, "Alice laptop")
	pending, token := addComputer(t, c, c.alice, "Alice desk")
	since := lastOutboxRow(t, db)

	body := callTool(t, c.h, c.owner, "account_delete", `{"id":"u-alice"}`)
	requireRetired(t, c, id, credential, uninstalled)
	requireEnrollsNothing(t, c, token)
	requireTellsNothing(t, c, db, since, body, id, pending)
}
