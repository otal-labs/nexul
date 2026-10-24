package main

import (
	"context"

	"github.com/otal-labs/nexul/internal/calendar"
	"github.com/otal-labs/nexul/internal/chat"
)

// calendarChannels reads a meeting's voice channel through chat's own read rule, as the caller.
type calendarChannels struct {
	svc *chat.Service
}

func (c calendarChannels) Channel(ctx context.Context, id string) (*calendar.Channel, error) {
	conv, err := c.svc.GetConversation(ctx, id)
	if err != nil {
		return nil, err
	}
	return &calendar.Channel{
		ID: conv.ID, WorkspaceID: conv.WorkspaceID, Name: conv.Name, Voice: conv.Kind == chat.KindVoiceChannel,
		Private: conv.Private, MemberIDs: conv.ParticipantIDs,
	}, nil
}
