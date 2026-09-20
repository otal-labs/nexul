package voice

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/otal-labs/nexul/internal/livekit"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// fakeConversations is a test double for ConversationChecker.
type fakeConversations struct {
	voiceChannels map[string]bool
	err           error
}

func (f *fakeConversations) IsVoiceChannel(_ context.Context, conversationID string) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	return f.voiceChannels[conversationID], nil
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
	if got := s.Occupancy(); len(got) != 0 {
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
	if got := s.Occupancy(); len(got) != 0 {
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
	if got := s.Occupancy(); len(got) != 0 {
		t.Fatalf("Occupancy() on a fresh service = %+v, want empty", got)
	}
	if err := s.HandleWebhook(context.Background(), livekit.Event{Type: "participant_joined", Room: "conv-1", ParticipantIdentity: "u-1", ParticipantName: "Ada"}); err != nil {
		t.Fatalf("HandleWebhook: %v", err)
	}
	got := s.Occupancy()
	want := []Occupant{{Identity: "u-1", Name: "Ada"}}
	if len(got["conv-1"]) != 1 || got["conv-1"][0] != want[0] {
		t.Errorf("Occupancy() = %+v, want conv-1: %+v", got, want)
	}
}
