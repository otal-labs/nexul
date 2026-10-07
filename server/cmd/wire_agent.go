package main

import (
	"context"
	"fmt"
	"time"

	"github.com/otal-labs/nexul/internal/agent"
	"github.com/otal-labs/nexul/internal/attachments"
	"github.com/otal-labs/nexul/internal/chat"
	"github.com/otal-labs/nexul/internal/docs"
	"github.com/otal-labs/nexul/internal/docs/richtext"
	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/memories"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/tickets"
	"github.com/otal-labs/nexul/internal/workspace"
)

// agentConversations adapts chat.Service to the agent pipeline's Conversations seam (ADR 0017: agent never imports chat).
type agentConversations struct {
	svc      *chat.Service
	projects *workspace.Service
}

func (a agentConversations) GetConversation(ctx context.Context, id string) (agent.Conversation, error) {
	c, err := a.svc.GetConversation(ctx, id)
	if err != nil {
		return agent.Conversation{}, err
	}
	return agent.Conversation{
		ProjectName:    a.projectName(ctx, c),
		Name:           c.Name,
		ID:             c.ID,
		WorkspaceID:    c.WorkspaceID,
		IsTicketThread: c.Kind == chat.KindTicketThread,
		TicketID:       c.TicketID,
		IsDocThread:    c.Kind == chat.KindDocThread,
		DocID:          c.DocID,
		ProjectID:      c.ProjectID,
		ThreadID:       c.AgentThreadID,
		SyncedAt:       c.AgentSyncedAt,
		SeenMarker:     c.AgentSeen,
	}, nil
}

// projectName reads an interview thread's project name for its session title; a failed read only costs the title.
func (a agentConversations) projectName(ctx context.Context, c *chat.Conversation) string {
	if c.Kind != chat.KindInterviewThread || c.ProjectID == "" || a.projects == nil {
		return ""
	}
	p, err := a.projects.Get(ctx, c.ProjectID)
	if err != nil {
		return ""
	}
	return p.Name
}

func (a agentConversations) MessagesSince(ctx context.Context, conversationID string, since time.Time) ([]agent.ConversationMessage, error) {
	ms, err := a.svc.ListMessagesSince(ctx, conversationID, since)
	if err != nil {
		return nil, err
	}
	files, err := a.svc.NoteFiles(ctx, ms)
	if err != nil {
		return nil, err
	}
	out := make([]agent.ConversationMessage, len(ms))
	for i, m := range ms {
		out[i] = agent.ConversationMessage{AuthorID: m.AuthorID, AuthorKind: string(m.AuthorKind), Body: m.Body, CreatedAt: m.CreatedAt, Via: m.Via, AuthorName: m.AuthorName}
		if f, ok := files[m.AttachmentID]; ok {
			out[i].Note = &agent.NoteFile{Name: f.Name, Markdown: f.Markdown}
		}
	}
	return out, nil
}

func (a agentConversations) SetThread(ctx context.Context, conversationID, threadID string) error {
	return a.svc.SetAgentThread(ctx, conversationID, threadID)
}

func (a agentConversations) MarkSynced(ctx context.Context, conversationID string, at time.Time) error {
	return a.svc.MarkAgentSynced(ctx, conversationID, at)
}

func (a agentConversations) PostAgentReply(ctx context.Context, conversationID, viaUserID, body string, handoffs []harness.Handoff) (string, error) {
	m, err := a.svc.PostAgentMessage(ctx, conversationID, viaUserID, body, handoffs)
	if err != nil {
		return "", err
	}
	return m.ID, nil
}

func (a agentConversations) PostSystemMessage(ctx context.Context, conversationID, viaUserID, body string) error {
	_, err := a.svc.PostSystemMessage(ctx, conversationID, viaUserID, body)
	return err
}

func (a agentConversations) PostUserMessage(ctx context.Context, conversationID, userID, body string) error {
	_, err := a.svc.PostMessage(ctx, conversationID, userID, body)
	return err
}

func (a agentConversations) MarkSeen(ctx context.Context, conversationID, marker string) error {
	return a.svc.MarkAgentSeen(ctx, conversationID, marker)
}

func (a agentConversations) ConversationByThread(ctx context.Context, threadID string) (agent.Conversation, error) {
	c, err := a.svc.ConversationByAgentThread(ctx, threadID)
	if err != nil {
		return agent.Conversation{}, err
	}
	return a.GetConversation(ctx, c.ID)
}

func (a agentConversations) PostHarnessMessage(ctx context.Context, conversationID, userID, body, via, key string, at time.Time) error {
	return a.svc.PostHarnessMessage(ctx, conversationID, userID, body, via, key, at)
}

// agentTicketReader adapts tickets to the agent's TicketReader seam: the key and title name the ticket, the body
// as markdown is where its images are found, and its column's stage says whether it is done.
type agentTicketReader struct {
	svc      *tickets.Service
	projects *storage.ProjectsRepo
	statuses *storage.StatusesRepo
}

func (a agentTicketReader) Get(ctx context.Context, id string) (agent.Ticket, error) {
	t, err := a.svc.Get(ctx, id)
	if err != nil {
		return agent.Ticket{}, err
	}
	body, err := richtext.ToMarkdown(t.Body)
	if err != nil {
		return agent.Ticket{}, err
	}
	prefix := ""
	if p, err := a.projects.Get(ctx, t.ProjectID); err == nil {
		prefix = p.Prefix
	}
	done := false
	if st, err := a.statuses.Get(ctx, string(t.Status)); err == nil {
		done = st.Kind == workspace.StatusKindDone
	}
	return agent.Ticket{ProjectID: t.ProjectID, Key: ticketKey(prefix, t.Number, t.ID), Title: t.Title, Body: body, Done: done}, nil
}

// ticketKey renders PREFIX-NUMBER (ADR 0004), or the id for a ticket whose project has no prefix.
func ticketKey(prefix string, number int, id string) string {
	if prefix == "" {
		return id
	}
	return fmt.Sprintf("%s-%d", prefix, number)
}

// agentDocReader adapts docs to the agent's DocReader seam: turn context needs the title, project, and
// the body as markdown (ADR 0026); ctx already carries the mentioning user's actor, so docs:read is checked.
type agentDocReader struct {
	svc *docs.Service
}

func (a agentDocReader) Get(ctx context.Context, id string) (agent.Doc, error) {
	d, err := a.svc.Get(ctx, id)
	if err != nil {
		return agent.Doc{}, err
	}
	md, err := a.svc.ExportMarkdown(ctx, id)
	if err != nil {
		return agent.Doc{}, err
	}
	return agent.Doc{ProjectID: d.ProjectID, Title: d.Title, BodyMarkdown: md}, nil
}

// agentUserReader adapts auth users to agent's UserReader seam: context lines render a login, not an id.
type agentUserReader struct {
	users *storage.UsersRepo
}

func (a agentUserReader) Login(ctx context.Context, userID string) (string, error) {
	u, err := a.users.GetUserByID(ctx, userID)
	if err != nil {
		return "", err
	}
	return u.Login, nil
}

func (a agentUserReader) UserID(ctx context.Context, login string) (string, error) {
	u, err := a.users.GetUserByLogin(ctx, login)
	if err != nil {
		return "", err
	}
	return u.ID, nil
}

// agentMemories adapts memories.Service to the agent pipeline's MemoriesReader seam (ADR 0017: agent never imports memories).
type agentMemories struct {
	svc *memories.Service
}

func (a agentMemories) ListMemories(ctx context.Context, projectID string) ([]agent.MemoryItem, error) {
	items, err := a.svc.ListMemoryItems(ctx, projectID)
	if err != nil {
		return nil, err
	}
	out := make([]agent.MemoryItem, len(items))
	for i, m := range items {
		out[i] = agent.MemoryItem{ID: m.ID, Name: m.Title, AlwaysIncluded: m.AlwaysIncluded, Interview: m.Kind == memories.KindInterview}
	}
	return out, nil
}

// agentAttachmentReader adapts attachments to the agent pipeline's AttachmentReader seam (ADR 0017): ctx already
// carries the mentioning or run-starting user's actor, so Get's owner check runs under their own permissions.
type agentAttachmentReader struct {
	svc *attachments.Service
}

func (a agentAttachmentReader) Get(ctx context.Context, id string) (agent.StoredAttachment, error) {
	att, err := a.svc.Get(ctx, id)
	if err != nil {
		return agent.StoredAttachment{}, err
	}
	return agent.StoredAttachment{Name: att.Name, MIME: att.ContentType, Bytes: att.Data}, nil
}
