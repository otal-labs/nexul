package voice

import (
	"context"
	"fmt"
	"strings"

	"github.com/otal-labs/nexul/internal/livekit"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// Config wires Service's seams (ADR 0017); adapters live in the composition root, never imported here directly.
type Config struct {
	Conversations ConversationChecker
	Credentials   CredentialSource
	Users         UserNames
	Bus           Publisher
}

// Service is voice's use-case layer: token minting and occupancy tracking against one LiveKit server.
type Service struct {
	conversations ConversationChecker
	credentials   CredentialSource
	users         UserNames
	bus           Publisher
	occupancy     *occupancyStore
}

// NewService wires voice's use-cases over the given seams.
func NewService(cfg Config) *Service {
	return &Service{
		conversations: cfg.Conversations,
		credentials:   cfg.Credentials,
		users:         cfg.Users,
		bus:           cfg.Bus,
		occupancy:     newOccupancyStore(),
	}
}

// JoinToken is what POST /api/voice/{conversationID}/token returns: a short-lived token and its WS URL.
type JoinToken struct {
	WSURL string `json:"ws_url"`
	Token string `json:"token"`
}

// Join mints a room-join token for userID; an unconfigured LiveKit connector is ErrInvalid, not ErrNotFound.
func (s *Service) Join(ctx context.Context, conversationID, userID string) (JoinToken, error) {
	conversationID = strings.TrimSpace(conversationID)
	if conversationID == "" {
		return JoinToken{}, fmt.Errorf("%w: conversation id is required", apperrs.ErrInvalid)
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return JoinToken{}, fmt.Errorf("%w: user id is required", apperrs.ErrInvalid)
	}
	ok, err := s.conversations.IsVoiceChannel(ctx, conversationID)
	if err != nil {
		return JoinToken{}, fmt.Errorf("check voice channel %s: %w", conversationID, err)
	}
	if !ok {
		return JoinToken{}, fmt.Errorf("%w: conversation %s is not a voice channel", apperrs.ErrInvalid, conversationID)
	}
	client, err := s.credentials.LiveKit(ctx)
	if err != nil {
		return JoinToken{}, fmt.Errorf("%w: voice channels need a LiveKit connector configured by the workspace owner: %v", apperrs.ErrInvalid, err)
	}
	name, err := s.users.DisplayName(ctx, userID)
	if err != nil {
		return JoinToken{}, fmt.Errorf("resolve display name for user %s: %w", userID, err)
	}
	token, err := client.MintToken(userID, name, conversationID)
	if err != nil {
		return JoinToken{}, fmt.Errorf("mint livekit token for conversation %s: %w", conversationID, err)
	}
	// Optimistic presence: register now, don't wait on the webhook/poll; a publish failure must not fail the join.
	occupants, changed := s.occupancy.join(conversationID, Occupant{Identity: userID, Name: name})
	_ = s.publishIfChanged(ctx, conversationID, occupants, changed)
	return JoinToken{WSURL: client.WSURL, Token: token}, nil
}

// Leave removes userID from the occupant list immediately, Join's optimistic-presence counterpart.
func (s *Service) Leave(ctx context.Context, conversationID, userID string) error {
	conversationID = strings.TrimSpace(conversationID)
	if conversationID == "" {
		return fmt.Errorf("%w: conversation id is required", apperrs.ErrInvalid)
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return fmt.Errorf("%w: user id is required", apperrs.ErrInvalid)
	}
	occupants, changed := s.occupancy.leave(conversationID, userID)
	return s.publishIfChanged(ctx, conversationID, occupants, changed)
}

// Occupancy returns every voice channel's current occupant list, the initial-render data source.
func (s *Service) Occupancy() map[string][]Occupant {
	return s.occupancy.snapshot()
}

// HandleWebhook applies one verified LiveKit delivery to occupancy and publishes a delta if it changed.
func (s *Service) HandleWebhook(ctx context.Context, ev livekit.Event) error {
	switch ev.Type {
	case "participant_joined":
		occupants, changed := s.occupancy.join(ev.Room, Occupant{Identity: ev.ParticipantIdentity, Name: ev.ParticipantName})
		return s.publishIfChanged(ctx, ev.Room, occupants, changed)
	case "participant_left":
		occupants, changed := s.occupancy.leave(ev.Room, ev.ParticipantIdentity)
		return s.publishIfChanged(ctx, ev.Room, occupants, changed)
	case "room_finished":
		if s.occupancy.clear(ev.Room) {
			return s.publish(ctx, ev.Room, []Occupant{})
		}
	}
	return nil
}

func (s *Service) publishIfChanged(ctx context.Context, room string, occupants []Occupant, changed bool) error {
	if !changed {
		return nil
	}
	return s.publish(ctx, room, occupants)
}

func (s *Service) publish(ctx context.Context, room string, occupants []Occupant) error {
	if s.bus == nil {
		return nil
	}
	return s.bus.Publish(ctx, TopicOccupancyChanged, OccupancyChangedEvent{ConversationID: room, Occupants: occupants})
}
