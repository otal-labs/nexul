package chat

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// ponytail: message_list loads a conversation's whole history to page it newest first; add a newest-first paged query if histories grow huge.
const messageScan = math.MaxInt32

type conversationListIn struct {
	WorkspaceID string `json:"workspace_id" jsonschema:"The workspace whose conversations to list."`
	mcptool.PageArgs
}

// messageTarget names a conversation directly, or through the doc, ticket, or project interview it belongs to.
type messageTarget struct {
	ConversationID string `json:"conversation_id,omitempty" jsonschema:"A conversation's id, from conversation_list."`
	DocID          string `json:"doc_id,omitempty" jsonschema:"A doc's id, for that doc's thread."`
	TicketID       string `json:"ticket_id,omitempty" jsonschema:"A ticket's id (a UUID, not its key), for that ticket's thread."`
	ProjectID      string `json:"project_id,omitempty" jsonschema:"A project's id, for that project's interview thread."`
}

type messageListIn struct {
	messageTarget
	mcptool.PageArgs
}

type messagePostIn struct {
	messageTarget
	WorkspaceID string `json:"workspace_id,omitempty" jsonschema:"The workspace of the doc, ticket, or project; needed only when its thread does not exist yet."`
	Body        string `json:"body" jsonschema:"The message as markdown. @login mentions a member and @Agent starts an agent turn."`
}

type conversationResult struct {
	ID             string    `json:"id"`
	Kind           Kind      `json:"kind"`
	Name           string    `json:"name,omitempty"`
	TicketID       string    `json:"ticket_id,omitempty"`
	DocID          string    `json:"doc_id,omitempty"`
	ProjectID      string    `json:"project_id,omitempty"`
	ParticipantIDs []string  `json:"participant_ids,omitempty"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type messageResult struct {
	ID             string     `json:"id"`
	ConversationID string     `json:"conversation_id"`
	AuthorID       string     `json:"author_id"`
	AuthorKind     AuthorKind `json:"author_kind"`
	Body           string     `json:"body"`
	Mentions       []Mention  `json:"mentions,omitempty"`
	EditedAt       *time.Time `json:"edited_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

// MCPTools returns the chat tools; each acts as the caller the identity context carries.
func MCPTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{conversationListTool(s), messageListTool(s), messagePostTool(s)}
}

func conversationListTool(s *Service) mcptool.Tool {
	return mcptool.New("conversation_list", "List conversations",
		"Lists your conversations in a workspace: every channel, plus the direct messages and threads you take part in. "+
			"Use it to find a conversation's id for message_list or message_post; a doc's, ticket's, or project interview's thread "+
			"can also be reached there by the doc_id, ticket_id, or project_id alone. "+
			"Each item has its kind (channel, dm, ticket_thread, doc_thread, interview_thread, voice_channel) and what it belongs to.",
		mcptool.Hints{ReadOnly: true, Local: true},
		func(ctx context.Context, in conversationListIn) (any, error) {
			caller, err := callerID(ctx)
			if err != nil {
				return nil, err
			}
			cs, err := s.ListConversations(ctx, in.WorkspaceID, caller)
			if err != nil {
				return nil, err
			}
			out := make([]conversationResult, 0, len(cs))
			for _, c := range cs {
				out = append(out, conversationResult{
					ID: c.ID, Kind: c.Kind, Name: c.Name, TicketID: c.TicketID, DocID: c.DocID, ProjectID: c.ProjectID,
					ParticipantIDs: c.ParticipantIDs, UpdatedAt: c.UpdatedAt,
				})
			}
			return mcptool.Paginate(out, in.PageArgs), nil
		})
}

func messageListTool(s *Service) mcptool.Tool {
	return mcptool.New("message_list", "List messages",
		"Lists a conversation's messages newest first, so the first page is the latest; page back with offset for older ones. "+
			"Name the conversation by conversation_id, or by exactly one of doc_id, ticket_id, or project_id for that doc's, ticket's, "+
			"or project interview's thread; a thread nobody has started yet lists as empty and is not created. "+
			"Deleted messages are left out. Reply with message_post.",
		mcptool.Hints{ReadOnly: true, Local: true},
		func(ctx context.Context, in messageListIn) (any, error) {
			caller, err := callerID(ctx)
			if err != nil {
				return nil, err
			}
			id, err := existingConversation(ctx, s, in.messageTarget, caller)
			if errors.Is(err, apperrs.ErrNotFound) && in.ConversationID == "" {
				return mcptool.Paginate([]messageResult{}, in.PageArgs), nil
			}
			if err != nil {
				return nil, err
			}
			ms, err := s.ListMessages(ctx, id, caller, messageScan)
			if err != nil {
				return nil, err
			}
			out := make([]messageResult, 0, len(ms))
			for _, m := range slices.Backward(ms) {
				if m.DeletedAt == nil {
					out = append(out, toMessageResult(m))
				}
			}
			return mcptool.Paginate(out, in.PageArgs), nil
		})
}

func messagePostTool(s *Service) mcptool.Tool {
	return mcptool.New("message_post", "Post message",
		"Posts a markdown message as you to a conversation, or to a doc's, ticket's, or project interview's thread, "+
			"starting that thread if it does not exist yet (which needs workspace_id). "+
			"Name the target the same way as message_list: conversation_id, or exactly one of doc_id, ticket_id, or project_id. "+
			"Mentioning @Agent starts an agent turn on your own paired computer. Returns the posted message and its conversation_id.",
		mcptool.Hints{Additive: true},
		func(ctx context.Context, in messagePostIn) (any, error) {
			caller, err := callerID(ctx)
			if err != nil {
				return nil, err
			}
			id, err := threadFor(ctx, s, in.messageTarget, in.WorkspaceID, caller)
			if err != nil {
				return nil, err
			}
			m, err := s.PostMessage(ctx, id, caller, in.Body)
			if err != nil {
				return nil, err
			}
			return toMessageResult(m), nil
		})
}

// target returns the thread kind and id named, an empty kind for a conversation id, or ErrInvalid unless exactly one is set.
func (t messageTarget) target() (Kind, string, error) {
	var kind Kind
	var id string
	set := 0
	for _, c := range []struct {
		kind Kind
		id   string
	}{{"", t.ConversationID}, {KindDocThread, t.DocID}, {KindTicketThread, t.TicketID}, {KindInterviewThread, t.ProjectID}} {
		if c.id != "" {
			kind, id = c.kind, c.id
			set++
		}
	}
	if set != 1 {
		return "", "", fmt.Errorf("%w: pass exactly one of conversation_id, doc_id, ticket_id, or project_id (the project's interview thread)", apperrs.ErrInvalid)
	}
	return kind, id, nil
}

// existingConversation resolves a target without creating anything; a thread nobody started is ErrNotFound.
func existingConversation(ctx context.Context, s *Service, t messageTarget, caller string) (string, error) {
	kind, id, err := t.target()
	if err != nil {
		return "", err
	}
	if kind == "" {
		return conversationByID(ctx, s, id)
	}
	c, err := s.ExistingThread(ctx, kind, id, caller)
	if err != nil {
		return "", err
	}
	return c.ID, nil
}

// threadFor resolves a target, starting the doc's, ticket's, or interview's thread when it does not exist yet.
func threadFor(ctx context.Context, s *Service, t messageTarget, workspaceID, caller string) (string, error) {
	kind, id, err := t.target()
	if err != nil {
		return "", err
	}
	var c *Conversation
	switch kind {
	case "":
		return conversationByID(ctx, s, id)
	case KindDocThread:
		c, err = s.GetOrCreateDocThread(ctx, workspaceID, id, caller)
	case KindTicketThread:
		c, err = s.GetOrCreateTicketThread(ctx, workspaceID, id, caller)
	default:
		c, err = s.GetOrCreateInterviewThread(ctx, workspaceID, id, caller)
	}
	if err != nil {
		return "", err
	}
	return c.ID, nil
}

func conversationByID(ctx context.Context, s *Service, id string) (string, error) {
	c, err := s.GetConversation(ctx, id)
	if err != nil {
		return "", fmt.Errorf("%w; conversation_list lists the conversations you can see", err)
	}
	return c.ID, nil
}

func toMessageResult(m *Message) messageResult {
	return messageResult{
		ID: m.ID, ConversationID: m.ConversationID, AuthorID: m.AuthorID, AuthorKind: m.AuthorKind, Body: m.Body,
		Mentions: m.Mentions, EditedAt: m.EditedAt, CreatedAt: m.CreatedAt,
	}
}

// callerID is the authenticated user; an argument never names who reads or posts.
func callerID(ctx context.Context) (string, error) {
	a, ok := identity.ActorFromCtx(ctx)
	if !ok || a.ID == "" {
		return "", apperrs.ErrUnauthorized
	}
	return a.ID, nil
}
