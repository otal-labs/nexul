// Package voice owns LiveKit interaction for voice channels: join tokens, webhooks, and room occupancy (ADR 0030).
package voice

import (
	"context"

	"github.com/otal-labs/nexul/internal/livekit"
)

// Occupant is one participant currently in a voice channel's room.
type Occupant struct {
	Identity string `json:"identity"`
	Name     string `json:"name"`
}

// ConversationChecker is the consumer-side seam over chat (ADR 0017), confirming a conversation is a voice channel.
type ConversationChecker interface {
	IsVoiceChannel(ctx context.Context, conversationID string) (bool, error)
}

// CredentialSource re-reads the credential per call, no restart needed (ADR 0017).
type CredentialSource interface {
	LiveKit(ctx context.Context) (livekit.Client, error)
}

// UserNames is the consumer-side seam over auth (ADR 0017), resolving a display name for the join token.
type UserNames interface {
	DisplayName(ctx context.Context, userID string) (string, error)
}

// Publisher bridges occupancy changes to the browser via the event bus.
type Publisher interface {
	Publish(ctx context.Context, topic string, payload any) error
}
