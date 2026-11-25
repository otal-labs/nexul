package auditlog

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/platform/webhooksig"
)

func TestTest_RefusesWhatCannotBeSentTo(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		unlocked bool
		saved    Settings
		patch    Patch
		want     error
	}{
		{"the key is locked", false, Settings{StreamURL: "https://logs.example.com"}, Patch{}, apperrs.ErrForbidden},
		{"no destination saved or given", true, Settings{}, Patch{}, apperrs.ErrInvalid},
		{"an http destination", true, Settings{}, Patch{StreamURL: ptr("http://logs.example.com")}, apperrs.ErrInvalid},
		{"a secret reference that does not resolve", true, Settings{StreamURL: "https://logs.example.com"}, Patch{StreamSecret: ptr("op://vault/missing/secret")}, apperrs.ErrInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := newHarness(t, tt.unlocked)
			h.repo.st = tt.saved

			_, err := h.svc.Test(asOwner(t), tt.patch)
			require.ErrorIs(t, err, tt.want)
		})
	}
}

func TestTest_SendsOneSignedSampleToTheFormsDestination_AndQueuesNothing(t *testing.T) {
	t.Parallel()
	rc, url := serveReceiver(t, http.StatusAccepted)
	h := newHarness(t, true)
	h.repo.st = Settings{StreamURL: "https://saved.example.com", StreamSecret: "saved"}

	res, err := h.svc.Test(asOwner(t), Patch{StreamURL: ptr(url), StreamSecret: ptr("typed")})
	require.NoError(t, err)

	assert.Equal(t, TestResult{Status: http.StatusAccepted}, res)
	got := rc.received()
	require.Len(t, got, 1)
	assert.Equal(t, "test", got[0].Action)
	assert.Equal(t, "owner-1", got[0].ActorID)
	assert.True(t, webhooksig.Verify("typed", rc.bodies[0], rc.sigs[0]))
	stats, err := h.queue.Stats(t.Context())
	require.NoError(t, err)
	assert.Zero(t, stats.Pending)
	assert.Empty(t, h.repo.events, "a test stores nothing")
}

func TestTest_ReportsTheReceiversRefusal(t *testing.T) {
	t.Parallel()
	_, url := serveReceiver(t, http.StatusForbidden)
	h := newHarness(t, true)
	h.repo.st = Settings{StreamURL: url, StreamSecret: "s3cret"}

	res, err := h.svc.Test(asOwner(t), Patch{})
	require.NoError(t, err)

	assert.Equal(t, TestResult{Status: http.StatusForbidden, Error: "the destination answered 403"}, res)
}

func TestSend_ASecretOrAddressThatCannotBeRead_IsReported(t *testing.T) {
	t.Parallel()
	h := newHarness(t, true)
	h.repo.st = Settings{StreamURL: "https://logs.example.com", StreamSecret: "op://vault/missing/secret"}
	res, err := h.svc.Test(asOwner(t), Patch{})
	require.NoError(t, err)
	assert.Contains(t, res.Error, "resolve the audit stream secret")

	h.repo.st.StreamSecret = "s3cret"
	h.svc.cfg.InstanceURL = func(context.Context) (string, error) { return "", errors.New("database is locked") }
	res, err = h.svc.Test(asOwner(t), Patch{})
	require.NoError(t, err)
	assert.Contains(t, res.Error, "read instance url")
}

func TestRotateSecret_MakesANewSecretShownOnce(t *testing.T) {
	t.Parallel()
	h := newHarness(t, true)
	h.repo.st = Settings{RetentionDays: 90, StreamURL: "https://logs.example.com", StreamSecret: "old", StreamEnabled: true}

	saved, err := h.svc.RotateSecret(asOwner(t))
	require.NoError(t, err)

	assert.Len(t, saved.NewSecret, 64)
	assert.Equal(t, saved.NewSecret, h.repo.st.StreamSecret)
	assert.Equal(t, 90, h.repo.st.RetentionDays)
	require.Len(t, h.repo.events, 1)
	assert.Equal(t, SettingsChangedEvent{RetentionDays: 90, StreamURL: "https://logs.example.com", StreamEnabled: true, ChangedBy: "owner-1", ChangedAt: now},
		h.repo.events[0].Payload)
}

func TestRotateSecretAndRetry_AreRefusedWithoutWhatTheyNeed(t *testing.T) {
	t.Parallel()
	h := newHarness(t, true)
	_, err := h.svc.RotateSecret(asOwner(t))
	require.ErrorIs(t, err, apperrs.ErrInvalid, "no destination to sign for")

	locked := newHarness(t, false)
	locked.repo.st = Settings{StreamURL: "https://logs.example.com"}
	_, err = locked.svc.RotateSecret(asOwner(t))
	require.ErrorIs(t, err, apperrs.ErrForbidden)
	_, err = locked.svc.RetryFailed(asOwner(t))
	require.ErrorIs(t, err, apperrs.ErrForbidden)

	h.svc.cfg.Gate = fakeGate{refuse: permissions.InstanceWrite}
	_, err = h.svc.RotateSecret(asOwner(t))
	require.ErrorIs(t, err, apperrs.ErrForbidden)
	_, err = h.svc.RetryFailed(asOwner(t))
	require.ErrorIs(t, err, apperrs.ErrForbidden)
	_, err = h.svc.Test(asOwner(t), Patch{})
	require.ErrorIs(t, err, apperrs.ErrForbidden)

	h.svc.cfg.Gate = fakeGate{}
	h.repo.err = errors.New("database is locked")
	_, err = h.svc.RotateSecret(asOwner(t))
	require.ErrorIs(t, err, h.repo.err)
	_, err = h.svc.Test(asOwner(t), Patch{})
	require.ErrorIs(t, err, h.repo.err)
}

func TestHandler_StreamRoutes(t *testing.T) {
	t.Parallel()
	_, url := serveReceiver(t, http.StatusOK)
	h := newHarness(t, true)
	h.repo.st = Settings{StreamURL: url, StreamSecret: "s3cret", StreamEnabled: true}
	routes := NewHandler(h.svc).Routes()
	post := func(path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequestWithContext(asOwner(t), http.MethodPost, path, strings.NewReader(body))
		rec := httptest.NewRecorder()
		routes.ServeHTTP(rec, req)
		return rec
	}

	rec := post("/api/instance/audit/stream/test", `{}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.JSONEq(t, `{"status":200}`, rec.Body.String())
	assert.Equal(t, http.StatusBadRequest, post("/api/instance/audit/stream/test", `{`).Code)

	rec = post("/api/instance/audit/stream/retry", "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	rec = post("/api/instance/audit/stream/rotate-secret", "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var saved Saved
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &saved))
	assert.NotEmpty(t, saved.NewSecret)

	h.licenses.on = false
	assert.Equal(t, http.StatusForbidden, post("/api/instance/audit/stream/retry", "").Code)
	assert.Equal(t, http.StatusForbidden, post("/api/instance/audit/stream/rotate-secret", "").Code)
	assert.Equal(t, http.StatusForbidden, post("/api/instance/audit/stream/test", `{}`).Code)
}
