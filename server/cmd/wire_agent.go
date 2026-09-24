package main

import (
	"context"
	"time"

	"github.com/otal-labs/nexul/internal/agent"
	"github.com/otal-labs/nexul/internal/attachments"
	"github.com/otal-labs/nexul/internal/chat"
	"github.com/otal-labs/nexul/internal/docs"
	"github.com/otal-labs/nexul/internal/memories"
	"github.com/otal-labs/nexul/internal/platform/storage"
	"github.com/otal-labs/nexul/internal/tickets"
)

// agentConversations adapts chat.Service to the agent pipeline's Conversations seam (ADR 0017: agent never imports chat).
type agentConversations struct {
	svc *chat.Service
}

func (a agentConversations) GetConversation(ctx context.Context, id string) (agent.Conversation, error) {
	c, err := a.svc.GetConversation(ctx, id)
	if err != nil {
		return agent.Conversation{}, err
	}
	return agent.Conversation{
		ID:             c.ID,
		WorkspaceID:    c.WorkspaceID,
		IsTicketThread: c.Kind == chat.KindTicketThread,
		TicketID:       c.TicketID,
		IsDocThread:    c.Kind == chat.KindDocThread,
		DocID:          c.DocID,
		ProjectID:      c.ProjectID,
		ThreadID:       c.AgentThreadID,
		SyncedAt:       c.AgentSyncedAt,
	}, nil
}

func (a agentConversations) MessagesSince(ctx context.Context, conversationID string, since time.Time) ([]agent.ConversationMessage, error) {
	ms, err := a.svc.ListMessagesSince(ctx, conversationID, since)
	if err != nil {
		return nil, err
	}
	out := make([]agent.ConversationMessage, len(ms))
	for i, m := range ms {
		out[i] = agent.ConversationMessage{AuthorID: m.AuthorID, AuthorKind: string(m.AuthorKind), Body: m.Body, CreatedAt: m.CreatedAt}
	}
	return out, nil
}

func (a agentConversations) SetThread(ctx context.Context, conversationID, threadID string) error {
	return a.svc.SetAgentThread(ctx, conversationID, threadID)
}

func (a agentConversations) MarkSynced(ctx context.Context, conversationID string, at time.Time) error {
	return a.svc.MarkAgentSynced(ctx, conversationID, at)
}

func (a agentConversations) PostAgentReply(ctx context.Context, conversationID, viaUserID, body string) (string, error) {
	m, err := a.svc.PostAgentMessage(ctx, conversationID, viaUserID, body)
	if err != nil {
		return "", err
	}
	return m.ID, nil
}

func (a agentConversations) PostSystemNote(ctx context.Context, conversationID, viaUserID, body string) error {
	_, err := a.svc.PostSystemMessage(ctx, conversationID, viaUserID, body)
	return err
}

func (a agentConversations) PostUserMessage(ctx context.Context, conversationID, userID, body string) error {
	_, err := a.svc.PostMessage(ctx, conversationID, userID, body)
	return err
}

// agentTicketReader adapts tickets to the agent's TicketReader seam: turn context needs only title/body/project.
type agentTicketReader struct {
	svc *tickets.Service
}

func (a agentTicketReader) Get(ctx context.Context, id string) (agent.Ticket, error) {
	t, err := a.svc.Get(ctx, id)
	if err != nil {
		return agent.Ticket{}, err
	}
	return agent.Ticket{ProjectID: t.ProjectID, Title: t.Title, Body: t.Body}, nil
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

func (a agentMemories) ListMemories(ctx context.Context, workspaceID, projectID string) (agent.MemoriesIndex, error) {
	workspaceItems, projectItems, err := a.svc.ListMemoryItems(ctx, workspaceID, projectID)
	if err != nil {
		return agent.MemoriesIndex{}, err
	}
	return agent.MemoriesIndex{Workspace: toAgentMemoryItems(workspaceItems), Project: toAgentMemoryItems(projectItems)}, nil
}

func toAgentMemoryItems(items []memories.MemoryItem) []agent.MemoryItem {
	out := make([]agent.MemoryItem, len(items))
	for i, m := range items {
		out[i] = agent.MemoryItem{Name: m.Title, WhenToUse: m.WhenToUse, AlwaysIncluded: m.AlwaysIncluded, Interview: m.Kind == memories.KindInterview, Body: m.Body}
	}
	return out
}

// agentAttachmentReader adapts attachments to the agent pipeline's (and plays runner's) AttachmentReader
// seam (ADR 0017): ctx already carries the mentioning or run-starting user's actor, so Get's owner check
// runs under their own permissions, the same as agentDocReader above.
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
