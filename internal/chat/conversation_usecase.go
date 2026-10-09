package chat

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

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
	if err := s.require(ctx, workspaceID, permissions.ChatWrite); err != nil {
		return nil, err
	}
	participants := dedupeParticipants(creatorUserID, participantUserIDs)
	if err := s.requireMembers(ctx, workspaceID, participants); err != nil {
		return nil, err
	}
	return s.createConversation(ctx, &Conversation{
		WorkspaceID: workspaceID,
		Kind:        KindDM,
		CreatedBy:   creatorUserID,
	}, participants)
}

// requireMembers refuses a participant outside the workspace as not found, the same answer for someone who does not
// exist, so a DM never confirms who is registered elsewhere.
func (s *Service) requireMembers(ctx context.Context, workspaceID string, userIDs []string) error {
	if s.members == nil {
		return permissions.Ungated(ctx)
	}
	for _, id := range userIDs {
		ok, err := s.members.IsMember(ctx, id, workspaceID)
		if err != nil {
			return fmt.Errorf("check member %s: %w", id, err)
		}
		if !ok {
			return fmt.Errorf("%w: no member %s in this workspace", apperrs.ErrNotFound, id)
		}
	}
	return nil
}

// GetOrCreateTicketThread lazily creates the thread; a losing race just re-fetches.
func (s *Service) GetOrCreateTicketThread(ctx context.Context, workspaceID, ticketID, creatorUserID string) (*Conversation, error) {
	ticketID = strings.TrimSpace(ticketID)
	if ticketID == "" {
		return nil, fmt.Errorf("%w: ticket id is required", apperrs.ErrInvalid)
	}
	existing, err := s.repo.GetTicketThread(ctx, ticketID)
	if err == nil {
		return existing, s.requireRead(ctx, existing, creatorUserID)
	}
	if !errors.Is(err, apperrs.ErrNotFound) {
		return nil, fmt.Errorf("get ticket thread for ticket %s: %w", ticketID, err)
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, fmt.Errorf("%w: workspace id is required", apperrs.ErrInvalid)
	}
	thread := &Conversation{WorkspaceID: workspaceID, Kind: KindTicketThread, TicketID: ticketID}
	if err := s.requireNewThread(ctx, thread); err != nil {
		return nil, err
	}
	creatorUserID = strings.TrimSpace(creatorUserID)
	if creatorUserID == "" {
		return nil, fmt.Errorf("%w: creator id is required", apperrs.ErrInvalid)
	}
	thread.CreatedBy = creatorUserID
	created, err := s.createConversation(ctx, thread, []string{creatorUserID})
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
		return existing, s.requireRead(ctx, existing, creatorUserID)
	}
	if !errors.Is(err, apperrs.ErrNotFound) {
		return nil, fmt.Errorf("get interview thread for project %s: %w", projectID, err)
	}
	workspaceID, creatorUserID = strings.TrimSpace(workspaceID), strings.TrimSpace(creatorUserID)
	if workspaceID == "" {
		return nil, fmt.Errorf("%w: workspace id is required", apperrs.ErrInvalid)
	}
	thread := &Conversation{WorkspaceID: workspaceID, Kind: KindInterviewThread, ProjectID: projectID}
	if err := s.requireNewThread(ctx, thread); err != nil {
		return nil, err
	}
	if creatorUserID == "" {
		return nil, fmt.Errorf("%w: creator id is required", apperrs.ErrInvalid)
	}
	thread.CreatedBy = creatorUserID
	created, err := s.createConversation(ctx, thread, []string{creatorUserID})
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

// requireNewThread checks a thread about to start: membership of its workspace, then its project.
func (s *Service) requireNewThread(ctx context.Context, thread *Conversation) error {
	if err := s.require(ctx, thread.WorkspaceID, permissions.Member); err != nil {
		return err
	}
	return s.requireGate(ctx, thread)
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
	if err := s.requireRead(ctx, c, callerID); err != nil {
		return nil, err
	}
	return c, nil
}

// listed keeps the conversations the caller reads; membership was checked once, and the doc and ticket threads are
// asked about in one batch each.
func (s *Service) listed(ctx context.Context, userID string, cs []*Conversation) ([]*Conversation, error) {
	threads, err := s.readableThreads(ctx, userID, cs)
	if err != nil {
		return nil, err
	}
	out := make([]*Conversation, 0, len(cs))
	for _, c := range cs {
		if readable, thread := threads[c.ID]; thread {
			if readable {
				out = append(out, c)
			}
			continue
		}
		err := s.reads(ctx, userID, c)
		if err != nil && !permissions.Refused(err) {
			return nil, err
		}
		if err == nil {
			out = append(out, c)
		}
	}
	return out, nil
}

// readableThreads answers every doc, ticket, and interview thread in cs, by conversation id: docs:thread on the doc
// for all doc threads at once, tickets:read through each ticket's project for all ticket threads at once.
func (s *Service) readableThreads(ctx context.Context, userID string, cs []*Conversation) (map[string]bool, error) {
	docs, tickets, err := s.threadTargets(ctx, userID, cs)
	if err != nil {
		return nil, err
	}
	out := map[string]bool{}
	for _, c := range cs {
		switch {
		case c.Kind == KindDocThread:
			out[c.ID] = docs[c.DocID]
		case c.Kind == KindTicketThread && tickets != nil:
			out[c.ID] = tickets[c.TicketID]
		case projectThread(c.Kind):
			err := s.requireGate(ctx, c)
			if err != nil && !permissions.Refused(err) {
				return nil, err
			}
			out[c.ID] = err == nil
		}
	}
	return out, nil
}

// threadTargets is which of cs's thread docs userID holds docs:thread on and which thread tickets they read; tickets
// is nil without a thread gate, leaving each ticket thread to its workspace check.
func (s *Service) threadTargets(ctx context.Context, userID string, cs []*Conversation) (docs, tickets map[string]bool, err error) {
	var docIDs, ticketIDs []string
	for _, c := range cs {
		if c.Kind == KindDocThread {
			docIDs = append(docIDs, c.DocID)
		}
		if c.Kind == KindTicketThread {
			ticketIDs = append(ticketIDs, c.TicketID)
		}
	}
	docs = map[string]bool{}
	if s.docAccess != nil && userID != "" && len(docIDs) > 0 {
		docs = s.docAccess.CanDocs(ctx, userID, "", docIDs, permissions.DocsThread)
	}
	if s.threads == nil || len(ticketIDs) == 0 {
		return docs, nil, nil
	}
	tickets, err = s.threads.RequireTickets(ctx, ticketIDs, permissions.TicketsRead)
	if err != nil {
		return nil, nil, fmt.Errorf("ticket threads: %w", err)
	}
	return docs, tickets, nil
}

// canThread reports whether userID holds docs:thread on docID; a service with no wired DocAccess fails closed.
func (s *Service) canThread(ctx context.Context, userID, docID string) bool {
	if s.docAccess == nil || userID == "" || docID == "" {
		return false
	}
	ok, err := s.docAccess.Can(ctx, userID, docID, permissions.DocsThread)
	return err == nil && ok
}

// requireConversation gates a conversation's messages behind reading it; one that doesn't exist yet is left to
// the caller's own error path.
func (s *Service) requireConversation(ctx context.Context, conversationID, callerID string) error {
	conv, err := s.repo.GetConversation(ctx, conversationID)
	if err != nil {
		return nil
	}
	return s.requireRead(ctx, conv, callerID)
}

// createConversation is the shared create-and-enqueue path for CreateChannel/CreateDM/GetOrCreateTicketThread.
func (s *Service) createConversation(ctx context.Context, c *Conversation, participantIDs []string) (*Conversation, error) {
	now := s.now().UTC()
	c.ID = ids.New()
	c.CreatedAt = now
	c.UpdatedAt = now
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicConversationCreated, Payload: ConversationCreatedEvent{Conversation: *c, MembersOnly: c.MembersOnly()}}
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

// HasTicketThreads batches the ticket-thread-exists check for the board's per-card indicator, leaving out threads
// the caller may not read.
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
	if s.threads != nil {
		visible, err := s.threads.RequireTickets(ctx, slices.Collect(maps.Keys(out)), permissions.TicketsRead)
		if err != nil {
			return nil, fmt.Errorf("has ticket threads: %w", err)
		}
		return visible, nil
	}
	threads := make([]*Conversation, 0, len(out))
	for ticketID := range out {
		thread, err := s.repo.GetTicketThread(ctx, ticketID)
		if err != nil {
			return nil, fmt.Errorf("has ticket threads: %w", err)
		}
		threads = append(threads, thread)
	}
	visible := make(map[string]bool, len(threads))
	for _, c := range threads {
		err := s.requireGate(ctx, c)
		if err != nil && !permissions.Refused(err) {
			return nil, fmt.Errorf("has ticket threads: %w", err)
		}
		if err == nil {
			visible[c.TicketID] = true
		}
	}
	return visible, nil
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
	if err := s.requireRead(ctx, c, ""); err != nil {
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

// MarkAgentSeen records the newest harness turn Nexul saw end on the conversation's agent thread.
func (s *Service) MarkAgentSeen(ctx context.Context, conversationID, marker string) error {
	conversationID = strings.TrimSpace(conversationID)
	if conversationID == "" {
		return fmt.Errorf("%w: conversation id is required", apperrs.ErrInvalid)
	}
	if err := s.repo.SetAgentSeen(ctx, conversationID, marker); err != nil {
		return fmt.Errorf("mark agent seen for conversation %s: %w", conversationID, err)
	}
	return nil
}

// ConversationByAgentThread is the conversation whose agent thread is threadID, read by the server, never a person.
func (s *Service) ConversationByAgentThread(ctx context.Context, threadID string) (*Conversation, error) {
	threadID = strings.TrimSpace(threadID)
	if threadID == "" {
		return nil, fmt.Errorf("%w: agent thread id is required", apperrs.ErrInvalid)
	}
	c, err := s.repo.GetConversationByAgentThread(ctx, threadID)
	if err != nil {
		return nil, fmt.Errorf("get conversation for agent thread %s: %w", threadID, err)
	}
	return c, nil
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
	if err := s.require(ctx, workspaceID, permissions.Member); err != nil {
		return nil, err
	}
	cs, err := s.repo.ListConversationsForUser(ctx, workspaceID, userID)
	if err != nil {
		return nil, fmt.Errorf("list conversations for user %s: %w", userID, err)
	}
	return s.listed(ctx, userID, cs)
}
