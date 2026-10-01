package voice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/livekit"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// fakeConversations is a test double for ConversationChecker.
type fakeConversations struct {
	voiceChannels map[string]bool
	readers       map[string]bool
	err, readsErr error
}

func (f *fakeConversations) IsVoiceChannel(_ context.Context, conversationID string) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	return f.voiceChannels[conversationID], nil
}

func (f *fakeConversations) Reads(_ context.Context, _, userID string) (bool, error) {
	return f.readers[userID], f.readsErr
}

// fakeCredentials is a test double for CredentialSource.
type fakeCredentials struct {
	client livekit.Client
	err    error
}

func (f *fakeCredentials) LiveKit(context.Context) (livekit.Client, error) {
	if f.err != nil {
		return livekit.Client{}, f.err
	}
	return f.client, nil
}

// fakeUserNames is a test double for UserNames.
type fakeUserNames struct {
	names map[string]string
	err   error
}

func (f *fakeUserNames) DisplayName(_ context.Context, userID string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.names[userID], nil
}

// fakePublisher is a test double for Publisher, capturing every publish.
type fakePublisher struct {
	mu     sync.Mutex
	events []OccupancyChangedEvent
}

func (f *fakePublisher) Publish(_ context.Context, topic string, payload any) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if topic != TopicOccupancyChanged {
		return nil
	}
	ev, ok := payload.(OccupancyChangedEvent)
	if !ok {
		return nil
	}
	f.events = append(f.events, ev)
	return nil
}

func (f *fakePublisher) all() []OccupancyChangedEvent {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]OccupancyChangedEvent, len(f.events))
	copy(out, f.events)
	return out
}

func newTestService(t *testing.T, conv *fakeConversations, cred *fakeCredentials, users *fakeUserNames, bus *fakePublisher) *Service {
	t.Helper()
	if conv == nil {
		conv = &fakeConversations{voiceChannels: map[string]bool{"conv-1": true}}
	}
	if cred == nil {
		cred = &fakeCredentials{client: livekit.Client{WSURL: "wss://lk.example.com", APIKey: "key1", APISecret: "secret1"}}
	}
	if users == nil {
		users = &fakeUserNames{names: map[string]string{"u-1": "Ada Lovelace"}}
	}
	if bus == nil {
		bus = &fakePublisher{}
	}
	return NewService(Config{Conversations: conv, Credentials: cred, Users: users, Bus: bus})
}

func TestJoin(t *testing.T) {
	t.Run("empty conversation id is invalid", func(t *testing.T) {
		s := newTestService(t, nil, nil, nil, nil)
		_, err := s.Join(context.Background(), "  ", "u-1")
		if !errors.Is(err, apperrs.ErrInvalid) {
			t.Fatalf("Join() err = %v, want ErrInvalid", err)
		}
	})

	t.Run("empty user id is invalid", func(t *testing.T) {
		s := newTestService(t, nil, nil, nil, nil)
		_, err := s.Join(context.Background(), "conv-1", "  ")
		if !errors.Is(err, apperrs.ErrInvalid) {
			t.Fatalf("Join() err = %v, want ErrInvalid", err)
		}
	})

	t.Run("nonexistent conversation propagates the checker's error", func(t *testing.T) {
		conv := &fakeConversations{err: apperrs.ErrNotFound}
		s := newTestService(t, conv, nil, nil, nil)
		_, err := s.Join(context.Background(), "conv-x", "u-1")
		if !errors.Is(err, apperrs.ErrNotFound) {
			t.Fatalf("Join() err = %v, want ErrNotFound", err)
		}
	})

	t.Run("conversation that isn't a voice channel is invalid", func(t *testing.T) {
		conv := &fakeConversations{voiceChannels: map[string]bool{"conv-1": false}}
		s := newTestService(t, conv, nil, nil, nil)
		_, err := s.Join(context.Background(), "conv-1", "u-1")
		if !errors.Is(err, apperrs.ErrInvalid) {
			t.Fatalf("Join() err = %v, want ErrInvalid", err)
		}
	})

	t.Run("no LiveKit connector configured is a clear 4xx (owner setup needed)", func(t *testing.T) {
		cred := &fakeCredentials{err: apperrs.ErrNotFound}
		s := newTestService(t, nil, cred, nil, nil)
		_, err := s.Join(context.Background(), "conv-1", "u-1")
		if !errors.Is(err, apperrs.ErrInvalid) {
			t.Fatalf("Join() err = %v, want ErrInvalid (not ErrNotFound, distinct from a missing conversation)", err)
		}
	})

	t.Run("mints a token with the resolved display name and room = conversation id", func(t *testing.T) {
		s := newTestService(t, nil, nil, nil, nil)
		tok, err := s.Join(context.Background(), "conv-1", "u-1")
		if err != nil {
			t.Fatalf("Join(): %v", err)
		}
		if tok.WSURL != "wss://lk.example.com" {
			t.Errorf("WSURL = %q, want wss://lk.example.com", tok.WSURL)
		}
		claims, err := parseTestJWT(tok.Token, "secret1")
		if err != nil {
			t.Fatalf("parse minted token: %v", err)
		}
		if claims["sub"] != "u-1" {
			t.Errorf("sub = %v, want u-1", claims["sub"])
		}
		if claims["name"] != "Ada Lovelace" {
			t.Errorf("name = %v, want Ada Lovelace", claims["name"])
		}
		video, _ := claims["video"].(map[string]any)
		if video["room"] != "conv-1" {
			t.Errorf("video.room = %v, want conv-1 (room = conversation id)", video["room"])
		}
	})
}

func TestHandleWebhook(t *testing.T) {
	t.Run("participant_joined adds an occupant and publishes", testHandleWebhookParticipantJoined)
	t.Run("participant_left removes an occupant and publishes", testHandleWebhookParticipantLeft)
	t.Run("room_finished clears a tracked room and publishes", testHandleWebhookRoomFinished)
	t.Run("room_finished on an untracked room is a no-op, no publish", testHandleWebhookRoomFinishedUntracked)
	t.Run("room_started is a no-op, no publish", testHandleWebhookRoomStarted)
}

func testHandleWebhookParticipantJoined(t *testing.T) {
	bus := &fakePublisher{}
	s := newTestService(t, nil, nil, nil, bus)
	if err := s.HandleWebhook(context.Background(), livekit.Event{Type: "participant_joined", Room: "conv-1", ParticipantIdentity: "u-1", ParticipantName: "Ada"}); err != nil {
		t.Fatalf("HandleWebhook: %v", err)
	}
	evts := bus.all()
	if len(evts) != 1 {
		t.Fatalf("published events = %+v, want 1", evts)
	}
	want := OccupancyChangedEvent{ConversationID: "conv-1", Occupants: []Occupant{{Identity: "u-1", Name: "Ada"}}}
	if evts[0].ConversationID != want.ConversationID || len(evts[0].Occupants) != 1 || evts[0].Occupants[0] != want.Occupants[0] {
		t.Errorf("published event = %+v, want %+v", evts[0], want)
	}
}

func testHandleWebhookParticipantLeft(t *testing.T) {
	bus := &fakePublisher{}
	s := newTestService(t, nil, nil, nil, bus)
	if err := s.HandleWebhook(context.Background(), livekit.Event{Type: "participant_joined", Room: "conv-1", ParticipantIdentity: "u-1", ParticipantName: "Ada"}); err != nil {
		t.Fatalf("HandleWebhook(joined): %v", err)
	}
	if err := s.HandleWebhook(context.Background(), livekit.Event{Type: "participant_left", Room: "conv-1", ParticipantIdentity: "u-1"}); err != nil {
		t.Fatalf("HandleWebhook(left): %v", err)
	}
	evts := bus.all()
	if len(evts) != 2 {
		t.Fatalf("published events = %+v, want 2 (joined, then left)", evts)
	}
	if len(evts[1].Occupants) != 0 {
		t.Errorf("published event after leave = %+v, want empty occupants", evts[1])
	}
	if got := s.Occupancy(context.Background()); len(got) != 0 {
		t.Errorf("Occupancy() after last participant leaves = %+v, want empty (room absent)", got)
	}
}

func testHandleWebhookRoomFinished(t *testing.T) {
	bus := &fakePublisher{}
	s := newTestService(t, nil, nil, nil, bus)
	if err := s.HandleWebhook(context.Background(), livekit.Event{Type: "participant_joined", Room: "conv-1", ParticipantIdentity: "u-1", ParticipantName: "Ada"}); err != nil {
		t.Fatalf("HandleWebhook(joined): %v", err)
	}
	if err := s.HandleWebhook(context.Background(), livekit.Event{Type: "room_finished", Room: "conv-1"}); err != nil {
		t.Fatalf("HandleWebhook(room_finished): %v", err)
	}
	if got := s.Occupancy(context.Background()); len(got) != 0 {
		t.Errorf("Occupancy() after room_finished = %+v, want empty", got)
	}
}

func testHandleWebhookRoomFinishedUntracked(t *testing.T) {
	bus := &fakePublisher{}
	s := newTestService(t, nil, nil, nil, bus)
	if err := s.HandleWebhook(context.Background(), livekit.Event{Type: "room_finished", Room: "conv-empty"}); err != nil {
		t.Fatalf("HandleWebhook(room_finished): %v", err)
	}
	if evts := bus.all(); len(evts) != 0 {
		t.Errorf("published events = %+v, want none", evts)
	}
}

func testHandleWebhookRoomStarted(t *testing.T) {
	bus := &fakePublisher{}
	s := newTestService(t, nil, nil, nil, bus)
	if err := s.HandleWebhook(context.Background(), livekit.Event{Type: "room_started", Room: "conv-1"}); err != nil {
		t.Fatalf("HandleWebhook(room_started): %v", err)
	}
	if evts := bus.all(); len(evts) != 0 {
		t.Errorf("published events = %+v, want none", evts)
	}
}

func TestOccupancy_Snapshot(t *testing.T) {
	s := newTestService(t, nil, nil, nil, nil)
	if got := s.Occupancy(context.Background()); len(got) != 0 {
		t.Fatalf("Occupancy() on a fresh service = %+v, want empty", got)
	}
	if err := s.HandleWebhook(context.Background(), livekit.Event{Type: "participant_joined", Room: "conv-1", ParticipantIdentity: "u-1", ParticipantName: "Ada"}); err != nil {
		t.Fatalf("HandleWebhook: %v", err)
	}
	got := s.Occupancy(context.Background())
	want := []Occupant{{Identity: "u-1", Name: "Ada"}}
	if len(got["conv-1"]) != 1 || got["conv-1"][0] != want[0] {
		t.Errorf("Occupancy() = %+v, want conv-1: %+v", got, want)
	}
}

func TestCloseRoom(t *testing.T) {
	t.Run("ends the LiveKit room and empties its occupancy", func(t *testing.T) {
		var deleted []string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var body struct {
				Room string `json:"room"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			deleted = append(deleted, strings.TrimPrefix(r.URL.Path, "/twirp/livekit.RoomService/")+" "+body.Room)
			_, _ = w.Write([]byte(`{}`))
		}))
		defer srv.Close()
		bus := &fakePublisher{}
		s := newTestService(t, nil, &fakeCredentials{client: livekit.Client{WSURL: srv.URL, APIKey: "key1", APISecret: "secret1"}}, nil, bus)
		s.occupancy.join("conv-1", Occupant{Identity: "u-1", Name: "Ada"})

		if err := s.CloseRoom(context.Background(), "conv-1"); err != nil {
			t.Fatalf("CloseRoom: %v", err)
		}
		if len(deleted) != 1 || deleted[0] != "DeleteRoom conv-1" {
			t.Fatalf("LiveKit calls = %v, want one DeleteRoom of conv-1", deleted)
		}
		if got := s.occupancy.room("conv-1"); len(got) != 0 {
			t.Fatalf("occupants after close = %v, want none", got)
		}
		events := bus.all()
		if len(events) != 1 || events[0].ConversationID != "conv-1" || len(events[0].Occupants) != 0 {
			t.Fatalf("published = %+v, want one empty occupancy for conv-1", events)
		}
	})

	t.Run("without a LiveKit connector there is no room to close", func(t *testing.T) {
		s := newTestService(t, nil, &fakeCredentials{err: errNotConfigured}, nil, nil)
		if err := s.CloseRoom(context.Background(), "conv-1"); err != nil {
			t.Fatalf("CloseRoom = %v, want nil", err)
		}
	})

	t.Run("a credentials failure is returned so the event is retried", func(t *testing.T) {
		s := newTestService(t, nil, &fakeCredentials{err: errors.New("database is locked")}, nil, nil)
		if err := s.CloseRoom(context.Background(), "conv-1"); err == nil {
			t.Fatal("CloseRoom = nil, want the credentials error")
		}
	})
}

// fakeLiveKit answers every RoomService call with status and records each as "<method> <room> <identity>".
func fakeLiveKit(t *testing.T, status int) (*fakeCredentials, *[]string) {
	t.Helper()
	var calls []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct{ Room, Identity string }
		_ = json.NewDecoder(r.Body).Decode(&body)
		calls = append(calls, strings.TrimPrefix(r.URL.Path, "/twirp/livekit.RoomService/")+" "+body.Room+" "+body.Identity)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	return &fakeCredentials{client: livekit.Client{WSURL: srv.URL, APIKey: "key1", APISecret: "secret1"}}, &calls
}

func TestRemoveFromCall(t *testing.T) {
	voiceChannel := map[string]bool{"conv-1": true}

	t.Run("disconnects whoever no longer reads the channel and keeps whoever still does", func(t *testing.T) {
		cred, calls := fakeLiveKit(t, http.StatusOK)
		bus := &fakePublisher{}
		s := newTestService(t, &fakeConversations{voiceChannels: voiceChannel, readers: map[string]bool{"u-owner": true}}, cred, nil, bus)
		s.occupancy.join("conv-1", Occupant{Identity: "u-owner", Name: "Owner"})
		s.occupancy.join("conv-1", Occupant{Identity: "u-1", Name: "Ada"})

		require.NoError(t, s.RemoveFromCall(t.Context(), "conv-1", []string{"u-owner", "u-1"}))
		assert.Equal(t, []string{"RemoveParticipant conv-1 u-1"}, *calls)
		assert.Equal(t, []Occupant{{Identity: "u-owner", Name: "Owner"}}, s.occupancy.room("conv-1"))
		assert.Len(t, bus.all(), 1)
	})

	t.Run("a deleted channel is left to CloseRoom", func(t *testing.T) {
		cred, calls := fakeLiveKit(t, http.StatusOK)
		notFound := fmt.Errorf("%w: conversation conv-1", apperrs.ErrNotFound)
		s := newTestService(t, &fakeConversations{err: notFound}, cred, nil, nil)
		require.NoError(t, s.RemoveFromCall(t.Context(), "conv-1", []string{"u-1"}))
		assert.Empty(t, *calls)
	})

	t.Run("without a LiveKit connector nobody is in a call", func(t *testing.T) {
		s := newTestService(t, &fakeConversations{voiceChannels: voiceChannel}, &fakeCredentials{err: errNotConfigured}, nil, nil)
		require.NoError(t, s.RemoveFromCall(t.Context(), "conv-1", []string{"u-1"}))
	})

	t.Run("a failed read check is returned so the event is retried", func(t *testing.T) {
		cred, calls := fakeLiveKit(t, http.StatusOK)
		s := newTestService(t, &fakeConversations{voiceChannels: voiceChannel, readsErr: errors.New("database is locked")}, cred, nil, nil)
		require.Error(t, s.RemoveFromCall(t.Context(), "conv-1", []string{"u-1"}))
		assert.Empty(t, *calls)
	})

	t.Run("a LiveKit outage is retryable", func(t *testing.T) {
		cred, _ := fakeLiveKit(t, http.StatusServiceUnavailable)
		s := newTestService(t, &fakeConversations{voiceChannels: voiceChannel}, cred, nil, nil)
		require.ErrorIs(t, s.RemoveFromCall(t.Context(), "conv-1", []string{"u-1"}), apperrs.ErrRetryable)
	})
}
