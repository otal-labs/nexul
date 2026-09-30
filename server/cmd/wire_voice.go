package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/otal-labs/nexul/internal/chat"
	"github.com/otal-labs/nexul/internal/connectors"
	"github.com/otal-labs/nexul/internal/livekit"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/voice"
)

// voiceConversations adapts chat.Service to voice's ConversationChecker seam: voice never imports chat's internals.
type voiceConversations struct {
	svc *chat.Service
}

func (v voiceConversations) IsVoiceChannel(ctx context.Context, conversationID string) (bool, error) {
	c, err := v.svc.GetConversation(ctx, conversationID)
	if err != nil {
		return false, err
	}
	return c.Kind == chat.KindVoiceChannel, nil
}

// voiceCredentials adapts ManualCredentials to voice's CredentialSource seam; built fresh per call, never cached.
type voiceCredentials struct {
	svc *connectors.Service
}

func (v voiceCredentials) LiveKit(ctx context.Context) (livekit.Client, error) {
	fields, err := v.svc.ManualCredentials(ctx, "livekit")
	if err != nil {
		return livekit.Client{}, err
	}
	return livekit.Client{WSURL: fields["ws_url"], APIKey: fields["api_key"], APISecret: fields["api_secret"]}, nil
}

// voiceUserNames adapts auth users to voice's UserNames seam: manual profile name, then provider name, then login.
type voiceUserNames struct {
	users *storage.UsersRepo
}

func (v voiceUserNames) DisplayName(ctx context.Context, userID string) (string, error) {
	u, err := v.users.GetUserByID(ctx, userID)
	if err != nil {
		return "", err
	}
	if u.DisplayName != nil && *u.DisplayName != "" {
		return *u.DisplayName, nil
	}
	if u.Name != "" {
		return u.Name, nil
	}
	return u.Login, nil
}

// voiceRoomCloseHandler ends a deleted voice channel's call, so nobody stays connected to a channel that is gone.
func voiceRoomCloseHandler(svc *voice.Service) eventbus.Handler {
	return func(ctx context.Context, ev eventbus.Event) error {
		var e chat.ConversationDeletedEvent
		if err := json.Unmarshal(ev.Payload, &e); err != nil {
			return apperrs.Fatal(fmt.Errorf("parse %s: %w", chat.TopicConversationDeleted, err))
		}
		if e.Kind != chat.KindVoiceChannel {
			return nil
		}
		return svc.CloseRoom(ctx, e.ConversationID)
	}
}
