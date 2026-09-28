package push

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

type fakeTokens struct {
	mu      sync.Mutex
	targets []Target
	cleared []string
	listErr error
}

func (f *fakeTokens) ListPushTargets(_ context.Context, userIDs []string) ([]Target, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	var out []Target
	for _, t := range f.targets {
		for _, id := range userIDs {
			if t.UserID == id {
				out = append(out, t)
				break
			}
		}
	}
	return out, nil
}

func (f *fakeTokens) ClearPushToken(_ context.Context, sessionID, _ string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.cleared = append(f.cleared, sessionID)
	return nil
}

type fakeNamer struct {
	names map[string]string
	calls int
}

func (f *fakeNamer) WorkspaceName(_ context.Context, id string) (string, error) {
	f.calls++
	name, ok := f.names[id]
	if !ok {
		return "", apperrs.ErrNotFound
	}
	return name, nil
}

type fakeInstance struct{ url string }

func (f fakeInstance) GetInstanceURL(context.Context) (string, error) { return f.url, nil }

type sentMessage struct {
	To    string `json:"to"`
	Title string `json:"title"`
	Body  string `json:"body"`
	Data  struct {
		NotificationID string `json:"notification_id"`
		Host           string `json:"host"`
	} `json:"data"`
}

// fakeExpo records every batch and answers each message with the ticket its reply function picks.
type fakeExpo struct {
	mu      sync.Mutex
	batches [][]sentMessage
	reply   func(m sentMessage) ticket
	status  int
}

func (f *fakeExpo) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var batch []sentMessage
	if err := json.NewDecoder(r.Body).Decode(&batch); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	f.mu.Lock()
	f.batches = append(f.batches, batch)
	f.mu.Unlock()
	if f.status != 0 {
		w.WriteHeader(f.status)
		return
	}
	tickets := make([]ticket, 0, len(batch))
	for _, m := range batch {
		tk := ticket{Status: "ok"}
		if f.reply != nil {
			tk = f.reply(m)
		}
		tickets = append(tickets, tk)
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{"data": tickets}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (f *fakeExpo) sent() []sentMessage {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []sentMessage
	for _, b := range f.batches {
		out = append(out, b...)
	}
	return out
}

func newSender(t *testing.T, expo *fakeExpo, tokens *fakeTokens, namer *fakeNamer) *Sender {
	t.Helper()
	srv := httptest.NewServer(expo)
	t.Cleanup(srv.Close)
	return New(Config{Tokens: tokens, Workspaces: namer, Instance: fakeInstance{url: "https://nexul.example/"}, Endpoint: srv.URL, Client: srv.Client()})
}

func event(t *testing.T, items ...pushItem) eventbus.Event {
	t.Helper()
	payload, err := json.Marshal(pushRequestedEvent{Notifications: items})
	require.NoError(t, err)
	return eventbus.Event{ID: "evt-1", Topic: "notification.push_requested", Payload: payload}
}

func TestHandleNotificationPushRequested_OnePhone_PostsTheAgreedPayload(t *testing.T) {
	expo := &fakeExpo{}
	tokens := &fakeTokens{targets: []Target{{SessionID: "s-phone", UserID: "u1", Token: "ExponentPushToken[abc]"}}}
	namer := &fakeNamer{names: map[string]string{"ws-1": "Acme"}}
	s := newSender(t, expo, tokens, namer)

	require.NoError(t, s.HandleNotificationPushRequested(t.Context(), event(t, pushItem{ID: "evt:u1", UserID: "u1", WorkspaceID: "ws-1"})))

	sent := expo.sent()
	require.Len(t, sent, 1)
	assert.Equal(t, "ExponentPushToken[abc]", sent[0].To)
	assert.Equal(t, "Nexul", sent[0].Title)
	assert.Equal(t, "New activity in Acme", sent[0].Body)
	assert.Equal(t, "evt:u1", sent[0].Data.NotificationID)
	assert.Equal(t, "https://nexul.example", sent[0].Data.Host, "host is the instance url without its trailing slash")
	assert.Empty(t, tokens.cleared)
}

func TestHandleNotificationPushRequested_NoWorkspace_SaysYourWorkspace(t *testing.T) {
	expo := &fakeExpo{}
	tokens := &fakeTokens{targets: []Target{{SessionID: "s1", UserID: "u1", Token: "tok"}}}
	namer := &fakeNamer{names: map[string]string{}}
	s := newSender(t, expo, tokens, namer)

	require.NoError(t, s.HandleNotificationPushRequested(t.Context(), event(t,
		pushItem{ID: "a:u1", UserID: "u1"},
		pushItem{ID: "b:u1", UserID: "u1", WorkspaceID: "ws-missing"},
		pushItem{ID: "c:u1", UserID: "u1", WorkspaceID: "ws-missing"},
	)))

	sent := expo.sent()
	require.Len(t, sent, 3)
	for _, m := range sent {
		assert.Equal(t, "New activity in your workspace", m.Body)
	}
	assert.Equal(t, 1, namer.calls, "a workspace is looked up once per event, even when the lookup fails")
}

func TestHandleNotificationPushRequested_NoTargets_NeverCallsExpo(t *testing.T) {
	expo := &fakeExpo{}
	s := newSender(t, expo, &fakeTokens{}, &fakeNamer{})

	require.NoError(t, s.HandleNotificationPushRequested(t.Context(), event(t, pushItem{ID: "a:u1", UserID: "u1"})))
	require.NoError(t, s.HandleNotificationPushRequested(t.Context(), event(t)))

	assert.Empty(t, expo.batches)
}

func TestHandleNotificationPushRequested_DeviceNotRegistered_ClearsThatToken(t *testing.T) {
	expo := &fakeExpo{reply: func(m sentMessage) ticket {
		tk := ticket{Status: "ok"}
		if m.To == "gone" {
			tk = ticket{Status: "error", Message: "not registered"}
			tk.Details.Error = "DeviceNotRegistered"
		}
		if m.To == "bad" {
			tk = ticket{Status: "error", Message: "bad token"}
			tk.Details.Error = "InvalidCredentials"
		}
		return tk
	}}
	tokens := &fakeTokens{targets: []Target{
		{SessionID: "s-live", UserID: "u1", Token: "live"},
		{SessionID: "s-gone", UserID: "u1", Token: "gone"},
		{SessionID: "s-bad", UserID: "u1", Token: "bad"},
	}}
	s := newSender(t, expo, tokens, &fakeNamer{})

	require.NoError(t, s.HandleNotificationPushRequested(t.Context(), event(t, pushItem{ID: "a:u1", UserID: "u1"})))

	assert.Equal(t, []string{"s-gone"}, tokens.cleared, "only the unregistered device loses its token; other errors are logged")
}

func TestHandleNotificationPushRequested_ManyPhones_BatchesByHundred(t *testing.T) {
	expo := &fakeExpo{}
	tokens := &fakeTokens{}
	for range 250 {
		tokens.targets = append(tokens.targets, Target{SessionID: "s", UserID: "u1", Token: "tok"})
	}
	s := newSender(t, expo, tokens, &fakeNamer{})

	require.NoError(t, s.HandleNotificationPushRequested(t.Context(), event(t, pushItem{ID: "a:u1", UserID: "u1"})))

	require.Len(t, expo.batches, 3)
	assert.Len(t, expo.batches[0], 100)
	assert.Len(t, expo.batches[1], 100)
	assert.Len(t, expo.batches[2], 50)
}

func TestHandleNotificationPushRequested_ExpoDown_IsRetryable(t *testing.T) {
	expo := &fakeExpo{status: http.StatusBadGateway}
	tokens := &fakeTokens{targets: []Target{{SessionID: "s1", UserID: "u1", Token: "tok"}}}
	s := newSender(t, expo, tokens, &fakeNamer{})

	err := s.HandleNotificationPushRequested(t.Context(), event(t, pushItem{ID: "a:u1", UserID: "u1"}))

	require.ErrorIs(t, err, apperrs.ErrRetryable)
}

func TestHandleNotificationPushRequested_BadPayload_IsFatal(t *testing.T) {
	s := newSender(t, &fakeExpo{}, &fakeTokens{}, &fakeNamer{})

	err := s.HandleNotificationPushRequested(t.Context(), eventbus.Event{ID: "e", Payload: []byte(`{`)})

	require.ErrorIs(t, err, apperrs.ErrFatal)
}

func TestHandleNotificationPushRequested_StoreError_Surfaces(t *testing.T) {
	boom := errors.New("db down")
	s := newSender(t, &fakeExpo{}, &fakeTokens{listErr: boom}, &fakeNamer{})

	err := s.HandleNotificationPushRequested(t.Context(), event(t, pushItem{ID: "a:u1", UserID: "u1"}))

	require.ErrorIs(t, err, boom)
}

func TestNew_Defaults(t *testing.T) {
	s := New(Config{})
	assert.Equal(t, ExpoEndpoint, s.cfg.Endpoint)
	assert.NotNil(t, s.cfg.Client)
	assert.NotNil(t, s.cfg.Logger)
	assert.Equal(t, "", s.host(t.Context()), "no instance gate means no host")
}
