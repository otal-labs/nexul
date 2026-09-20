package voice

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/otal-labs/nexul/internal/livekit"
)

// twirpStub is a minimal stub RoomService server for ListRooms/
// ListParticipants, mirroring internal/livekit's own test helper (a
// different package, so it isn't reusable directly).
func twirpStub(t *testing.T, handlers map[string]func() any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method := r.URL.Path[len("/twirp/livekit.RoomService/"):]
		h, ok := handlers[method]
		if !ok {
			t.Fatalf("unexpected twirp method %q", method)
		}
		_, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(h())
	}))
}

func TestReconcileOnce_PublishesOnlyChangedRooms(t *testing.T) {
	srv := twirpStub(t, map[string]func() any{
		"ListRooms": func() any {
			return map[string]any{"rooms": []map[string]any{{"name": "conv-1", "numParticipants": 1}}}
		},
		"ListParticipants": func() any {
			return map[string]any{"participants": []map[string]any{{"identity": "u-1", "name": "Ada"}}}
		},
	})
	defer srv.Close()

	cred := &fakeCredentials{client: livekit.Client{WSURL: srv.URL, APIKey: "key1", APISecret: "secret1"}}
	bus := &fakePublisher{}
	s := NewService(Config{Conversations: &fakeConversations{}, Credentials: cred, Users: &fakeUserNames{}, Bus: bus})

	s.reconcileOnce(context.Background(), nil)
	evts := bus.all()
	if len(evts) != 1 {
		t.Fatalf("published events after first reconcile = %+v, want 1 (new room)", evts)
	}
	if evts[0].ConversationID != "conv-1" || len(evts[0].Occupants) != 1 {
		t.Errorf("published event = %+v, want conv-1 with 1 occupant", evts[0])
	}

	// A second reconcile against the same server state is a no-op: no drift, no publish.
	s.reconcileOnce(context.Background(), nil)
	if evts := bus.all(); len(evts) != 1 {
		t.Fatalf("published events after unchanged second reconcile = %+v, want still 1", evts)
	}
}

func TestReconcileOnce_NoConnectorConfigured_NoOp(t *testing.T) {
	cred := &fakeCredentials{err: errNotConfigured}
	bus := &fakePublisher{}
	s := NewService(Config{Conversations: &fakeConversations{}, Credentials: cred, Users: &fakeUserNames{}, Bus: bus})

	s.reconcileOnce(context.Background(), nil)
	if evts := bus.all(); len(evts) != 0 {
		t.Errorf("published events with no connector configured = %+v, want none", evts)
	}
}
