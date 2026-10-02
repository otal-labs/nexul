package chat

import (
	"context"
	"errors"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strings"
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
	DocID          string `json:"doc_id,omitempty" jsonschema:"A doc's id, from doc_list, for that doc's thread."`
	TicketID       string `json:"ticket_id,omitempty" jsonschema:"A ticket's id (a UUID, not its key such as REF-102; ticket_get returns the id), for that ticket's thread."`
	ProjectID      string `json:"project_id,omitempty" jsonschema:"A project's id, from project_list, for that project's interview thread."`
}

// ticketKey matches a ticket's human key; chat cannot resolve one, and a missing ticket's thread would list as empty.
var ticketKey = regexp.MustCompile(`(?i)^[a-z][a-z0-9]{1,4}-\d+$`)

type messageListIn struct {
	messageTarget
	mcptool.PageArgs
}

type messagePostIn struct {
	messageTarget
	WorkspaceID string      `json:"workspace_id,omitempty" jsonschema:"The workspace of the doc, ticket, or project; needed only when its thread does not exist yet."`
	Body        string      `json:"body" jsonschema:"The message as markdown. @login mentions a member and @Agent starts an agent turn."`
	File        *noteFileIn `json:"file,omitempty" jsonschema:"A markdown file to carry with the message, which makes the post a note; only a ticket's thread takes one."`
}

type noteFileIn struct {
	Name     string `json:"name" jsonschema:"The file's name, such as handoff.md; it always ends in .md."`
	Markdown string `json:"markdown" jsonschema:"The file's content as markdown."`
}

type noteFileResult struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Markdown string `json:"markdown"`
}

type conversationUpdateIn struct {
	ID              string   `json:"id" jsonschema:"The channel's or voice channel's id, from conversation_list."`
	Name            string   `json:"name,omitempty" jsonschema:"The new name, without the leading #; omit to keep the current one."`
	Private         *bool    `json:"private,omitempty" jsonschema:"true makes the channel private, false makes it public and opens its history to the whole workspace; omit to keep it as it is."`
	MemberIDs       []string `json:"member_ids,omitempty" jsonschema:"With private true: the user ids, besides you, who stay in the channel; everyone else loses it. mention_search finds a person's user id."`
	AddMemberIDs    []string `json:"add_member_ids,omitempty" jsonschema:"User ids of workspace members to add to a private channel."`
	RemoveMemberIDs []string `json:"remove_member_ids,omitempty" jsonschema:"User ids to remove from a private channel; your own id leaves it."`
}

type conversationDeleteIn struct {
	ID string `json:"id" jsonschema:"The channel's or voice channel's id, from conversation_list."`
}

type conversationResult struct {
	ID             string    `json:"id"`
	Kind           Kind      `json:"kind"`
	Name           string    `json:"name,omitempty"`
	TicketID       string    `json:"ticket_id,omitempty"`
	DocID          string    `json:"doc_id,omitempty"`
	ProjectID      string    `json:"project_id,omitempty"`
	ParticipantIDs []string  `json:"participant_ids,omitempty"`
	Private        bool      `json:"private"`
	MemberIDs      []string  `json:"member_ids,omitempty"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type messageResult struct {
	ID             string     `json:"id"`
	ConversationID string     `json:"conversation_id"`
	AuthorID       string     `json:"author_id"`
	AuthorKind     AuthorKind `json:"author_kind"`
	Body           string     `json:"body"`
	Mentions       []Mention  `json:"mentions,omitempty"`
	// File is a note's markdown file; only a note carries one.
	File      *noteFileResult `json:"file,omitempty"`
	EditedAt  *time.Time      `json:"edited_at,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}

// MCPTools returns the chat tools; each acts as the caller the identity context carries.
func MCPTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{conversationListTool(s), conversationUpdateTool(s), conversationDeleteTool(s), messageListTool(s), messagePostTool(s)}
}

func conversationListTool(s *Service) mcptool.Tool {
	return mcptool.New("conversation_list", "List conversations",
		"Lists your conversations in a workspace: every public channel and the private channels you are in, plus the direct messages and threads you take part in. "+
			"Use it to find a conversation's id for message_list or message_post; a doc's, ticket's, or project interview's thread "+
			"can also be reached there by the doc_id, ticket_id, or project_id alone. "+
			"Each item has its kind (channel, dm, ticket_thread, doc_thread, interview_thread, voice_channel), what it belongs to, "+
			"and whether it is private, with a private channel's member_ids.",
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
				out = append(out, toConversationResult(c))
			}
			return mcptool.Paginate(out, in.PageArgs), nil
		})
}

func conversationUpdateTool(s *Service) mcptool.Tool {
	return mcptool.New("conversation_update", "Update channel",
		"Renames a channel or voice channel, makes it private or public, and adds or removes a private channel's members; "+
			"direct messages and doc, ticket, or interview threads have none of these to change. "+
			"Renaming and switching private or public take channels:write; going private keeps you and member_ids and "+
			"everyone else loses the channel, and the workspace's general channel is always public. "+
			"Anyone in a private channel may add workspace members, removing someone else takes channels:write, "+
			"and removing yourself leaves, except as its last member. "+
			"Omitted fields keep their value; the changes apply in the order name, private, add, remove, and an error says which applied. "+
			"Returns the channel as it ends up.",
		mcptool.Hints{Idempotent: true, Local: true},
		func(ctx context.Context, in conversationUpdateIn) (any, error) {
			if len(in.MemberIDs) > 0 && (in.Private == nil || !*in.Private) {
				return nil, fmt.Errorf("%w: member_ids names who stays when private is true; add_member_ids adds people to a private channel", apperrs.ErrInvalid)
			}
			c, err := s.GetConversation(ctx, in.ID)
			if err != nil {
				return nil, err
			}
			steps := []struct {
				field string
				set   bool
				apply func() (*Conversation, error)
			}{
				{"name", in.Name != "", func() (*Conversation, error) { return s.RenameChannel(ctx, in.ID, in.Name) }},
				{"private", in.Private != nil, func() (*Conversation, error) { return s.SetChannelPrivate(ctx, in.ID, *in.Private, in.MemberIDs) }},
				{"add_member_ids", len(in.AddMemberIDs) > 0, func() (*Conversation, error) { return s.AddChannelMembers(ctx, in.ID, in.AddMemberIDs) }},
				{"remove_member_ids", len(in.RemoveMemberIDs) > 0, func() (*Conversation, error) { return s.RemoveChannelMembers(ctx, in.ID, in.RemoveMemberIDs) }},
			}
			var applied []string
			for _, step := range steps {
				if !step.set {
					continue
				}
				next, err := step.apply()
				if err != nil && len(applied) > 0 {
					return nil, fmt.Errorf("%w (already applied: %s)", err, strings.Join(applied, ", "))
				}
				if err != nil {
					return nil, err
				}
				c, applied = next, append(applied, step.field)
			}
			return toConversationResult(c), nil
		})
}

func conversationDeleteTool(s *Service) mcptool.Tool {
	return mcptool.New("conversation_delete", "Delete channel",
		"Deletes a channel or voice channel and every message in it, for good; a voice channel's call ends for everyone in it. "+
			"It takes channels:delete, and the workspace's general channel is never deleted. "+
			"Direct messages and doc, ticket, or interview threads are not deleted here. Returns the deleted id.",
		mcptool.Hints{Idempotent: true},
		func(ctx context.Context, in conversationDeleteIn) (any, error) {
			c, err := s.DeleteChannel(ctx, in.ID)
			if err != nil {
				return nil, err
			}
			return map[string]any{"id": c.ID, "deleted": true}, nil
		})
}

func toConversationResult(c *Conversation) conversationResult {
	out := conversationResult{
		ID: c.ID, Kind: c.Kind, Name: c.Name, TicketID: c.TicketID, DocID: c.DocID, ProjectID: c.ProjectID,
		Private: c.Private, UpdatedAt: c.UpdatedAt,
	}
	if c.Private {
		out.MemberIDs = c.ParticipantIDs
		return out
	}
	out.ParticipantIDs = c.ParticipantIDs
	return out
}

func messageListTool(s *Service) mcptool.Tool {
	return mcptool.New("message_list", "List messages",
		"Lists a conversation's messages newest first, so the first page is the latest; page back with offset for older ones. "+
			"Name the conversation by conversation_id, or by exactly one of doc_id, ticket_id, or project_id for that doc's, ticket's, "+
			"or project interview's thread; a thread nobody has started yet lists as empty and is not created. "+
			"Deleted messages are left out, and a note carries its markdown file in file. Reply with message_post.",
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
			live := make([]*Message, 0, len(ms))
			for _, m := range slices.Backward(ms) {
				if m.DeletedAt == nil {
					live = append(live, m)
				}
			}
			page := mcptool.Paginate(live, in.PageArgs)
			files, err := s.NoteFiles(ctx, page.Items)
			if err != nil {
				return nil, err
			}
			out := mcptool.Page[messageResult]{Items: make([]messageResult, 0, len(page.Items)), Total: page.Total, HasMore: page.HasMore, NextOffset: page.NextOffset}
			for _, m := range page.Items {
				out.Items = append(out.Items, toMessageResult(m, files[m.AttachmentID]))
			}
			return out, nil
		})
}

func messagePostTool(s *Service) mcptool.Tool {
	return mcptool.New("message_post", "Post message",
		"Posts a markdown message as you to a conversation, or to a doc's, ticket's, or project interview's thread, "+
			"starting that thread if it does not exist yet (which needs workspace_id). "+
			"Name the target the same way as message_list: conversation_id, or exactly one of doc_id, ticket_id, or project_id. "+
			"Mentioning @Agent starts an agent turn on your own paired computer. "+
			"With file, the post is a note on a ticket's thread: it shows as the Agent on your behalf, starts no turn, "+
			"needs tickets:write on the ticket, and carries the file; message_list returns it. "+
			"Returns the posted message and its conversation_id.",
		mcptool.Hints{},
		func(ctx context.Context, in messagePostIn) (any, error) {
			caller, err := callerID(ctx)
			if err != nil {
				return nil, err
			}
			id, err := threadFor(ctx, s, in.messageTarget, in.WorkspaceID, caller)
			if err != nil {
				return nil, err
			}
			if in.File == nil {
				m, err := s.PostMessage(ctx, id, caller, in.Body)
				if err != nil {
					return nil, err
				}
				return toMessageResult(m, nil), nil
			}
			m, file, err := s.PostNote(ctx, id, caller, in.Body, NoteFileInput{Name: in.File.Name, Markdown: in.File.Markdown})
			if err != nil {
				return nil, err
			}
			return toMessageResult(m, file), nil
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
	if kind == KindTicketThread && ticketKey.MatchString(id) {
		return "", "", fmt.Errorf("%w: ticket_id takes the ticket's id, not its key %s; ticket_get returns the id", apperrs.ErrInvalid, id)
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

func toMessageResult(m *Message, file *NoteFile) messageResult {
	out := messageResult{
		ID: m.ID, ConversationID: m.ConversationID, AuthorID: m.AuthorID, AuthorKind: m.AuthorKind, Body: m.Body,
		Mentions: m.Mentions, EditedAt: m.EditedAt, CreatedAt: m.CreatedAt,
	}
	if file != nil {
		out.File = &noteFileResult{ID: file.ID, Name: file.Name, Markdown: file.Markdown}
	}
	return out
}

// callerID is the authenticated user; an argument never names who reads or posts.
func callerID(ctx context.Context) (string, error) {
	a, ok := identity.ActorFromCtx(ctx)
	if !ok || a.ID == "" {
		return "", fmt.Errorf("%w: an authenticated user is required", apperrs.ErrUnauthorized)
	}
	return a.ID, nil
}
