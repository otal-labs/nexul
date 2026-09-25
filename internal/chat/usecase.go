package chat

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// Service is the chat use-case layer (ADR 0019); mutations enqueue events via the transactional outbox.
type Service struct {
	repo      Repo
	docAccess DocAccess
	now       func() time.Time
}

// NewService wires the chat use-cases over the given repo.
func NewService(repo Repo) *Service {
	return &Service{repo: repo, now: time.Now}
}

// SetDocAccess wires the access domain's docs:thread check (ADR 0017); a nil checker fails closed.
func (s *Service) SetDocAccess(a DocAccess) {
	s.docAccess = a
}

// CreateChannel creates a workspace-scoped public channel; v1 enforces no access check on it (ADR 0023).
func (s *Service) CreateChannel(ctx context.Context, workspaceID, creatorUserID, name string) (*Conversation, error) {
	return s.createChannelKind(ctx, workspaceID, creatorUserID, name, KindChannel)
}

// CreateVoiceChannel creates a public voice channel; internal/voice owns the LiveKit side.
func (s *Service) CreateVoiceChannel(ctx context.Context, workspaceID, creatorUserID, name string) (*Conversation, error) {
	return s.createChannelKind(ctx, workspaceID, creatorUserID, name, KindVoiceChannel)
}

// createChannelKind is CreateChannel/CreateVoiceChannel's shared validate-and-create path.
func (s *Service) createChannelKind(ctx context.Context, workspaceID, creatorUserID, name string, kind Kind) (*Conversation, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, fmt.Errorf("%w: workspace id is required", apperrs.ErrInvalid)
	}
	creatorUserID = strings.TrimSpace(creatorUserID)
	if creatorUserID == "" {
		return nil, fmt.Errorf("%w: creator id is required", apperrs.ErrInvalid)
	}
	// Lower-cased: the unique index is case-sensitive, so "#General" and "#general" would otherwise coexist.
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return nil, fmt.Errorf("%w: channel name is required", apperrs.ErrInvalid)
	}
	return s.createConversation(ctx, &Conversation{
		WorkspaceID: workspaceID,
		Kind:        kind,
		Name:        name,
		CreatedBy:   creatorUserID,
	}, []string{creatorUserID})
}

// EnsureGeneralChannel idempotently creates a workspace's #general channel (tenancy's ChannelGate seam).
func (s *Service) EnsureGeneralChannel(ctx context.Context, workspaceID, creatorUserID string) error {
	_, err := s.CreateChannel(ctx, workspaceID, creatorUserID, GeneralChannelName)
	if err != nil && !errors.Is(err, apperrs.ErrConflict) {
		return fmt.Errorf("ensure general channel for workspace %s: %w", workspaceID, err)
	}
	return nil
}

// CreateDM creates a direct-message conversation; v1 has no dedup lookup, each call is a new one.
func (s *Service) CreateDM(ctx context.Context, workspaceID, creatorUserID string, participantUserIDs []string) (*Conversation, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, fmt.Errorf("%w: workspace id is required", apperrs.ErrInvalid)
	}
	creatorUserID = strings.TrimSpace(creatorUserID)
	if creatorUserID == "" {
		return nil, fmt.Errorf("%w: creator id is required", apperrs.ErrInvalid)
	}
	participants := dedupeParticipants(creatorUserID, participantUserIDs)
	return s.createConversation(ctx, &Conversation{
		WorkspaceID: workspaceID,
		Kind:        KindDM,
		CreatedBy:   creatorUserID,
	}, participants)
}

// GetOrCreateTicketThread lazily creates the thread; a losing race just re-fetches.
func (s *Service) GetOrCreateTicketThread(ctx context.Context, workspaceID, ticketID, creatorUserID string) (*Conversation, error) {
	ticketID = strings.TrimSpace(ticketID)
	if ticketID == "" {
		return nil, fmt.Errorf("%w: ticket id is required", apperrs.ErrInvalid)
	}
	existing, err := s.repo.GetTicketThread(ctx, ticketID)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, apperrs.ErrNotFound) {
		return nil, fmt.Errorf("get ticket thread for ticket %s: %w", ticketID, err)
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, fmt.Errorf("%w: workspace id is required", apperrs.ErrInvalid)
	}
	creatorUserID = strings.TrimSpace(creatorUserID)
	if creatorUserID == "" {
		return nil, fmt.Errorf("%w: creator id is required", apperrs.ErrInvalid)
	}
	created, err := s.createConversation(ctx, &Conversation{
		WorkspaceID: workspaceID,
		Kind:        KindTicketThread,
		TicketID:    ticketID,
		CreatedBy:   creatorUserID,
	}, []string{creatorUserID})
	if err == nil {
		return created, nil
	}
	if !errors.Is(err, apperrs.ErrConflict) {
		return nil, err
	}
	existing, getErr := s.repo.GetTicketThread(ctx, ticketID)
	if getErr != nil {
		return nil, fmt.Errorf("get ticket thread for ticket %s after conflict: %w", ticketID, getErr)
	}
	return existing, nil
}

// GetOrCreateDocThread lazily creates the thread; a losing race just re-fetches. callerID must already
// hold docs:thread on docID, checked here so the gateway and MCP message tools agree (ADR 0057).
func (s *Service) GetOrCreateDocThread(ctx context.Context, workspaceID, docID, callerID string) (*Conversation, error) {
	docID = strings.TrimSpace(docID)
	if docID == "" {
		return nil, fmt.Errorf("%w: doc id is required", apperrs.ErrInvalid)
	}
	callerID = strings.TrimSpace(callerID)
	if callerID == "" {
		return nil, fmt.Errorf("%w: caller id is required", apperrs.ErrInvalid)
	}
	if !s.canThread(ctx, callerID, docID) {
		return nil, fmt.Errorf("%w: docs:thread required on doc %s", apperrs.ErrForbidden, docID)
	}
	existing, err := s.repo.GetDocThread(ctx, docID)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, apperrs.ErrNotFound) {
		return nil, fmt.Errorf("get doc thread for doc %s: %w", docID, err)
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, fmt.Errorf("%w: workspace id is required", apperrs.ErrInvalid)
	}
	created, err := s.createConversation(ctx, &Conversation{
		WorkspaceID: workspaceID,
		Kind:        KindDocThread,
		DocID:       docID,
		CreatedBy:   callerID,
	}, []string{callerID})
	if err == nil {
		return created, nil
	}
	if !errors.Is(err, apperrs.ErrConflict) {
		return nil, err
	}
	existing, getErr := s.repo.GetDocThread(ctx, docID)
	if getErr != nil {
		return nil, fmt.Errorf("get doc thread for doc %s after conflict: %w", docID, getErr)
	}
	return existing, nil
}

// GetOrCreateInterviewThread lazily creates a project's interview thread; a losing race just re-fetches.
func (s *Service) GetOrCreateInterviewThread(ctx context.Context, workspaceID, projectID, creatorUserID string) (*Conversation, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, fmt.Errorf("%w: project id is required", apperrs.ErrInvalid)
	}
	existing, err := s.repo.GetInterviewThread(ctx, projectID)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, apperrs.ErrNotFound) {
		return nil, fmt.Errorf("get interview thread for project %s: %w", projectID, err)
	}
	workspaceID, creatorUserID = strings.TrimSpace(workspaceID), strings.TrimSpace(creatorUserID)
	if workspaceID == "" {
		return nil, fmt.Errorf("%w: workspace id is required", apperrs.ErrInvalid)
	}
	if creatorUserID == "" {
		return nil, fmt.Errorf("%w: creator id is required", apperrs.ErrInvalid)
	}
	created, err := s.createConversation(ctx, &Conversation{
		WorkspaceID: workspaceID,
		Kind:        KindInterviewThread,
		ProjectID:   projectID,
		CreatedBy:   creatorUserID,
	}, []string{creatorUserID})
	if err == nil {
		return created, nil
	}
	if !errors.Is(err, apperrs.ErrConflict) {
		return nil, err
	}
	existing, getErr := s.repo.GetInterviewThread(ctx, projectID)
	if getErr != nil {
		return nil, fmt.Errorf("get interview thread for project %s after conflict: %w", projectID, getErr)
	}
	return existing, nil
}

// ExistingThread returns a doc, ticket, or interview thread without creating it, ErrNotFound until one exists; doc threads need docs:thread.
func (s *Service) ExistingThread(ctx context.Context, kind Kind, targetID, callerID string) (*Conversation, error) {
	targetID = strings.TrimSpace(targetID)
	if targetID == "" {
		return nil, fmt.Errorf("%w: target id is required", apperrs.ErrInvalid)
	}
	var c *Conversation
	var err error
	switch kind {
	case KindDocThread:
		if !s.canThread(ctx, callerID, targetID) {
			return nil, fmt.Errorf("%w: docs:thread required on doc %s", apperrs.ErrForbidden, targetID)
		}
		c, err = s.repo.GetDocThread(ctx, targetID)
	case KindTicketThread:
		c, err = s.repo.GetTicketThread(ctx, targetID)
	case KindInterviewThread:
		c, err = s.repo.GetInterviewThread(ctx, targetID)
	default:
		return nil, fmt.Errorf("%w: %s conversations are not threads of a doc, ticket, or project", apperrs.ErrInvalid, kind)
	}
	if err != nil {
		return nil, fmt.Errorf("get %s for %s: %w", kind, targetID, err)
	}
	return c, nil
}

// canThread reports whether userID holds docs:thread on docID; a service with no wired DocAccess fails closed.
func (s *Service) canThread(ctx context.Context, userID, docID string) bool {
	if s.docAccess == nil || userID == "" || docID == "" {
		return false
	}
	ok, err := s.docAccess.Can(ctx, userID, docID, permissions.DocsThread)
	return err == nil && ok
}

// requireDocThreadAccess gates a doc_thread conversation's messages behind docs:thread; a conversation that
// isn't a doc thread, or doesn't exist yet, is left to the caller's own error path.
func (s *Service) requireDocThreadAccess(ctx context.Context, conversationID, callerID string) error {
	conv, err := s.repo.GetConversation(ctx, conversationID)
	if err != nil {
		return nil
	}
	if conv.Kind != KindDocThread {
		return nil
	}
	if s.canThread(ctx, callerID, conv.DocID) {
		return nil
	}
	return fmt.Errorf("%w: docs:thread required on doc %s", apperrs.ErrForbidden, conv.DocID)
}

// createConversation is the shared create-and-enqueue path for CreateChannel/CreateDM/GetOrCreateTicketThread.
func (s *Service) createConversation(ctx context.Context, c *Conversation, participantIDs []string) (*Conversation, error) {
	now := s.now().UTC()
	c.ID = ids.New()
	c.CreatedAt = now
	c.UpdatedAt = now
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicConversationCreated, Payload: ConversationCreatedEvent{Conversation: *c}}
	if err := s.repo.CreateConversation(ctx, c, participantIDs, evt); err != nil {
		return nil, fmt.Errorf("create %s conversation: %w", c.Kind, err)
	}
	return c, nil
}

// dedupeParticipants folds creatorUserID into participantUserIDs, trims blanks, and drops duplicates.
func dedupeParticipants(creatorUserID string, participantUserIDs []string) []string {
	seen := map[string]bool{creatorUserID: true}
	out := []string{creatorUserID}
	for _, id := range participantUserIDs {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

// HasTicketThreads batches the ticket-thread-exists check for the board's per-card indicator.
func (s *Service) HasTicketThreads(ctx context.Context, ticketIDs []string) (map[string]bool, error) {
	cleaned := make([]string, 0, len(ticketIDs))
	for _, id := range ticketIDs {
		if id = strings.TrimSpace(id); id != "" {
			cleaned = append(cleaned, id)
		}
	}
	out, err := s.repo.HasTicketThreads(ctx, cleaned)
	if err != nil {
		return nil, fmt.Errorf("has ticket threads: %w", err)
	}
	return out, nil
}

// GetConversation returns one conversation by id.
func (s *Service) GetConversation(ctx context.Context, id string) (*Conversation, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("%w: conversation id is required", apperrs.ErrInvalid)
	}
	c, err := s.repo.GetConversation(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get conversation %s: %w", id, err)
	}
	return c, nil
}

// SetAgentThread persists a conversation's durable T3 thread id.
func (s *Service) SetAgentThread(ctx context.Context, conversationID, threadID string) error {
	conversationID = strings.TrimSpace(conversationID)
	if conversationID == "" {
		return fmt.Errorf("%w: conversation id is required", apperrs.ErrInvalid)
	}
	if err := s.repo.SetAgentThread(ctx, conversationID, threadID); err != nil {
		return fmt.Errorf("set agent thread for conversation %s: %w", conversationID, err)
	}
	return nil
}

// MarkAgentSynced advances how far a conversation's history was sent as turn context.
func (s *Service) MarkAgentSynced(ctx context.Context, conversationID string, at time.Time) error {
	conversationID = strings.TrimSpace(conversationID)
	if conversationID == "" {
		return fmt.Errorf("%w: conversation id is required", apperrs.ErrInvalid)
	}
	if err := s.repo.SetAgentSyncedAt(ctx, conversationID, at.UTC()); err != nil {
		return fmt.Errorf("mark agent synced for conversation %s: %w", conversationID, err)
	}
	return nil
}

// ListConversations returns a user's chat surface: every public channel plus their other conversations.
func (s *Service) ListConversations(ctx context.Context, workspaceID, userID string) ([]*Conversation, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, fmt.Errorf("%w: workspace id is required", apperrs.ErrInvalid)
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, fmt.Errorf("%w: user id is required", apperrs.ErrInvalid)
	}
	cs, err := s.repo.ListConversationsForUser(ctx, workspaceID, userID)
	if err != nil {
		return nil, fmt.Errorf("list conversations for user %s: %w", userID, err)
	}
	out := make([]*Conversation, 0, len(cs))
	for _, c := range cs {
		if c.Kind == KindDocThread && !s.canThread(ctx, userID, c.DocID) {
			continue
		}
		out = append(out, c)
	}
	return out, nil
}

// ListMessages returns a conversation's messages oldest-first; callerID gates a doc thread's messages.
func (s *Service) ListMessages(ctx context.Context, conversationID, callerID string, limit int) ([]*Message, error) {
	conversationID = strings.TrimSpace(conversationID)
	if conversationID == "" {
		return nil, fmt.Errorf("%w: conversation id is required", apperrs.ErrInvalid)
	}
	if err := s.requireDocThreadAccess(ctx, conversationID, callerID); err != nil {
		return nil, err
	}
	if limit < 1 {
		limit = 50
	}
	ms, err := s.repo.ListMessages(ctx, conversationID, limit)
	if err != nil {
		return nil, fmt.Errorf("list messages for conversation %s: %w", conversationID, err)
	}
	return ms, nil
}

// ListMessagesSince returns non-deleted messages created strictly after since, oldest-first.
func (s *Service) ListMessagesSince(ctx context.Context, conversationID string, since time.Time) ([]*Message, error) {
	conversationID = strings.TrimSpace(conversationID)
	if conversationID == "" {
		return nil, fmt.Errorf("%w: conversation id is required", apperrs.ErrInvalid)
	}
	ms, err := s.repo.ListMessagesSince(ctx, conversationID, since)
	if err != nil {
		return nil, fmt.Errorf("list messages since %s for conversation %s: %w", since, conversationID, err)
	}
	return ms, nil
}

// PostMessage posts a markdown message to a conversation, parsing @user/@Agent mentions; a doc thread
// gates the post behind docs:thread so the gateway and MCP tools agree (ADR 0057).
func (s *Service) PostMessage(ctx context.Context, conversationID, authorID, body string) (*Message, error) {
	if err := s.requireDocThreadAccess(ctx, conversationID, authorID); err != nil {
		return nil, err
	}
	return s.postMessage(ctx, conversationID, authorID, body, AuthorUser)
}

// PostAgentMessage persists the Agent's final turn reply; in-progress text only streams live.
func (s *Service) PostAgentMessage(ctx context.Context, conversationID, viaUserID, body string) (*Message, error) {
	return s.postMessage(ctx, conversationID, viaUserID, body, AuthorAgent)
}

// PostSystemMessage posts an inline system note attributed to the user it's for.
func (s *Service) PostSystemMessage(ctx context.Context, conversationID, viaUserID, body string) (*Message, error) {
	return s.postMessage(ctx, conversationID, viaUserID, body, AuthorSystem)
}

func (s *Service) postMessage(ctx context.Context, conversationID, authorID, body string, kind AuthorKind) (*Message, error) {
	conversationID = strings.TrimSpace(conversationID)
	if conversationID == "" {
		return nil, fmt.Errorf("%w: conversation id is required", apperrs.ErrInvalid)
	}
	authorID = strings.TrimSpace(authorID)
	if authorID == "" {
		return nil, fmt.Errorf("%w: author id is required", apperrs.ErrInvalid)
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, fmt.Errorf("%w: message body is required", apperrs.ErrInvalid)
	}
	now := s.now().UTC()
	m := &Message{
		ID:             ids.New(),
		ConversationID: conversationID,
		AuthorID:       authorID,
		AuthorKind:     kind,
		Body:           body,
		Mentions:       ParseMentions(body),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicMessageCreated, Payload: MessageCreatedEvent{Message: *m}}
	if err := s.repo.CreateMessage(ctx, m, evt); err != nil {
		return nil, fmt.Errorf("post message to conversation %s: %w", conversationID, err)
	}
	return m, nil
}

// EditMessage edits a message's body in place; only its author may edit it, and never a deleted one.
func (s *Service) EditMessage(ctx context.Context, messageID, authorID, body string) (*Message, error) {
	messageID = strings.TrimSpace(messageID)
	if messageID == "" {
		return nil, fmt.Errorf("%w: message id is required", apperrs.ErrInvalid)
	}
	authorID = strings.TrimSpace(authorID)
	if authorID == "" {
		return nil, fmt.Errorf("%w: author id is required", apperrs.ErrInvalid)
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, fmt.Errorf("%w: message body is required", apperrs.ErrInvalid)
	}
	current, err := s.repo.GetMessage(ctx, messageID)
	if err != nil {
		return nil, fmt.Errorf("edit message %s: %w", messageID, err)
	}
	if current.DeletedAt != nil {
		return nil, fmt.Errorf("%w: cannot edit a deleted message", apperrs.ErrInvalid)
	}
	if current.AuthorID != authorID {
		return nil, fmt.Errorf("%w: only the author may edit this message", apperrs.ErrForbidden)
	}
	now := s.now().UTC()
	updated := *current
	updated.Body = body
	updated.Mentions = ParseMentions(body)
	updated.EditedAt = &now
	updated.UpdatedAt = now
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicMessageUpdated, Payload: MessageUpdatedEvent{Message: updated}}
	if err := s.repo.UpdateMessage(ctx, messageID, updated.Body, updated.Mentions, now, evt); err != nil {
		return nil, fmt.Errorf("edit message %s: %w", messageID, err)
	}
	return &updated, nil
}

// DeleteMessage soft-deletes a message (author-only); deleting an already-deleted one is a no-op.
func (s *Service) DeleteMessage(ctx context.Context, messageID, authorID string) error {
	messageID = strings.TrimSpace(messageID)
	if messageID == "" {
		return fmt.Errorf("%w: message id is required", apperrs.ErrInvalid)
	}
	authorID = strings.TrimSpace(authorID)
	if authorID == "" {
		return fmt.Errorf("%w: author id is required", apperrs.ErrInvalid)
	}
	current, err := s.repo.GetMessage(ctx, messageID)
	if err != nil {
		return fmt.Errorf("delete message %s: %w", messageID, err)
	}
	if current.DeletedAt != nil {
		return nil
	}
	if current.AuthorID != authorID {
		return fmt.Errorf("%w: only the author may delete this message", apperrs.ErrForbidden)
	}
	now := s.now().UTC()
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicMessageDeleted, Payload: MessageDeletedEvent{ConversationID: current.ConversationID, MessageID: messageID, DeletedAt: now}}
	if err := s.repo.DeleteMessage(ctx, messageID, now, evt); err != nil {
		return fmt.Errorf("delete message %s: %w", messageID, err)
	}
	return nil
}

// MarkRead advances userID's read cursor on a conversation to now.
func (s *Service) MarkRead(ctx context.Context, conversationID, userID string) error {
	conversationID = strings.TrimSpace(conversationID)
	if conversationID == "" {
		return fmt.Errorf("%w: conversation id is required", apperrs.ErrInvalid)
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return fmt.Errorf("%w: user id is required", apperrs.ErrInvalid)
	}
	if err := s.repo.MarkRead(ctx, conversationID, userID, s.now().UTC()); err != nil {
		return fmt.Errorf("mark conversation %s read for user %s: %w", conversationID, userID, err)
	}
	return nil
}

// UnreadCounts returns userID's unread message count per conversation, the nav badge's data source.
func (s *Service) UnreadCounts(ctx context.Context, workspaceID, userID string) (map[string]int, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, fmt.Errorf("%w: workspace id is required", apperrs.ErrInvalid)
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, fmt.Errorf("%w: user id is required", apperrs.ErrInvalid)
	}
	rows, err := s.repo.UnreadCounts(ctx, workspaceID, userID)
	if err != nil {
		return nil, fmt.Errorf("unread counts for user %s: %w", userID, err)
	}
	counts := make(map[string]int, len(rows))
	for _, r := range rows {
		if r.Kind == KindDocThread && !s.canThread(ctx, userID, r.DocID) {
			continue
		}
		counts[r.ConversationID] = r.Count
	}
	return counts, nil
}
