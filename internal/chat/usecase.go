package chat

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// Service is the chat use-case layer (ADR 0019); mutations enqueue events via the transactional outbox.
type Service struct {
	repo      Repo
	docAccess DocAccess
	gate      Gate
	members   Membership
	now       func() time.Time
}

// Membership answers whether a person belongs to a workspace, so a DM only ever joins its own members.
type Membership interface {
	IsMember(ctx context.Context, userID, workspaceID string) (bool, error)
}

// SetMembership wires the membership lookup; unset, only the server's own calls may start a DM.
func (s *Service) SetMembership(m Membership) { s.members = m }

// Gate is the permission check a conversation passes before the caller reads or starts one (the access domain,
// ADR 0042); permissions.Member asks only that the caller belongs to the workspace.
type Gate interface {
	Require(ctx context.Context, workspaceID string, action permissions.Action) error
}

// SetGate wires the permission check; unset, only the server's own calls pass.
func (s *Service) SetGate(g Gate) { s.gate = g }

func (s *Service) require(ctx context.Context, workspaceID string, action permissions.Action) error {
	if s.gate == nil {
		return permissions.Ungated(ctx)
	}
	return s.gate.Require(ctx, workspaceID, action)
}

// readAction is what reading a conversation takes: any member reads channels and DMs they are in, while a ticket
// or interview thread is read by whoever may read the ticket or the project's memories.
func readAction(kind Kind) permissions.Action {
	switch kind {
	case KindTicketThread:
		return permissions.TicketsRead
	case KindInterviewThread:
		return permissions.MemoriesRead
	}
	return permissions.Member
}

// requireRead checks callerID (the actor when empty) may read c: a doc thread takes docs:thread (ADR 0057), and a
// DM they are not part of reads as not found.
func (s *Service) requireRead(ctx context.Context, c *Conversation, callerID string) error {
	if err := s.require(ctx, c.WorkspaceID, readAction(c.Kind)); err != nil {
		return err
	}
	if callerID == "" {
		actor, ok := identity.ActorFromCtx(ctx)
		if !ok {
			return nil
		}
		callerID = actor.ID
	}
	if c.Kind == KindDocThread && !s.canThread(ctx, callerID, c.DocID) {
		return fmt.Errorf("%w: docs:thread required on doc %s", apperrs.ErrForbidden, c.DocID)
	}
	if c.Kind == KindDM && !slices.Contains(c.ParticipantIDs, callerID) {
		return fmt.Errorf("%w: conversation %s", apperrs.ErrNotFound, c.ID)
	}
	return nil
}

// NewService wires the chat use-cases over the given repo.
func NewService(repo Repo) *Service {
	return &Service{repo: repo, now: time.Now}
}

// SetDocAccess wires the access domain's docs:thread check (ADR 0017); a nil checker fails closed.
func (s *Service) SetDocAccess(a DocAccess) {
	s.docAccess = a
}

// CreateChannel creates a workspace-scoped public channel; it takes chat:write in the workspace.
func (s *Service) CreateChannel(ctx context.Context, workspaceID, creatorUserID, name string) (*Conversation, error) {
	return s.createChannelKind(ctx, workspaceID, creatorUserID, name, KindChannel, true)
}

// CreateVoiceChannel creates a public voice channel; internal/voice owns the LiveKit side.
func (s *Service) CreateVoiceChannel(ctx context.Context, workspaceID, creatorUserID, name string) (*Conversation, error) {
	return s.createChannelKind(ctx, workspaceID, creatorUserID, name, KindVoiceChannel, true)
}

// createChannelKind is CreateChannel/CreateVoiceChannel's shared validate-and-create path; gated says whether the
// caller needs chat:write, which a workspace's own #general, made as the workspace is born, does not.
func (s *Service) createChannelKind(ctx context.Context, workspaceID, creatorUserID, name string, kind Kind, gated bool) (*Conversation, error) {
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
	if gated {
		if err := s.require(ctx, workspaceID, permissions.ChatWrite); err != nil {
			return nil, err
		}
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
	_, err := s.createChannelKind(ctx, workspaceID, creatorUserID, GeneralChannelName, KindChannel, false)
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
	if err := s.require(ctx, workspaceID, permissions.TicketsRead); err != nil {
		return nil, err
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
		return existing, s.requireRead(ctx, existing, creatorUserID)
	}
	if !errors.Is(err, apperrs.ErrNotFound) {
		return nil, fmt.Errorf("get interview thread for project %s: %w", projectID, err)
	}
	workspaceID, creatorUserID = strings.TrimSpace(workspaceID), strings.TrimSpace(creatorUserID)
	if workspaceID == "" {
		return nil, fmt.Errorf("%w: workspace id is required", apperrs.ErrInvalid)
	}
	if err := s.require(ctx, workspaceID, permissions.MemoriesRead); err != nil {
		return nil, err
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
	if err := s.requireRead(ctx, c, callerID); err != nil {
		return nil, err
	}
	return c, nil
}

// readableKinds answers, per conversation kind, whether the caller may read it in workspaceID, asking the gate
// once for each thread kind rather than once per conversation.
func (s *Service) readableKinds(ctx context.Context, workspaceID string) (func(Kind) bool, error) {
	allowed := map[permissions.Action]bool{permissions.Member: true}
	for _, action := range []permissions.Action{permissions.TicketsRead, permissions.MemoriesRead} {
		err := s.require(ctx, workspaceID, action)
		if err != nil && !permissions.Refused(err) {
			return nil, err
		}
		allowed[action] = err == nil
	}
	return func(k Kind) bool { return allowed[readAction(k)] }, nil
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
	threads := make([]*Conversation, 0, len(out))
	for ticketID := range out {
		thread, err := s.repo.GetTicketThread(ctx, ticketID)
		if err != nil {
			return nil, fmt.Errorf("has ticket threads: %w", err)
		}
		threads = append(threads, thread)
	}
	readable, err := permissions.Filter(threads, func(c *Conversation) string { return c.WorkspaceID }, func(workspaceID string) error {
		return s.require(ctx, workspaceID, readAction(KindTicketThread))
	})
	if err != nil {
		return nil, fmt.Errorf("has ticket threads: %w", err)
	}
	visible := make(map[string]bool, len(readable))
	for _, c := range readable {
		visible[c.TicketID] = true
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
	readable, err := s.readableKinds(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	out := make([]*Conversation, 0, len(cs))
	for _, c := range cs {
		if !readable(c.Kind) || (c.Kind == KindDocThread && !s.canThread(ctx, userID, c.DocID)) {
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
	if err := s.requireConversation(ctx, conversationID, callerID); err != nil {
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
	if err := s.requireConversation(ctx, conversationID, authorID); err != nil {
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
	if err := s.requireConversation(ctx, conversationID, userID); err != nil {
		return err
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
	if err := s.require(ctx, workspaceID, permissions.Member); err != nil {
		return nil, err
	}
	rows, err := s.repo.UnreadCounts(ctx, workspaceID, userID)
	if err != nil {
		return nil, fmt.Errorf("unread counts for user %s: %w", userID, err)
	}
	readable, err := s.readableKinds(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	counts := make(map[string]int, len(rows))
	for _, r := range rows {
		if !readable(r.Kind) || (r.Kind == KindDocThread && !s.canThread(ctx, userID, r.DocID)) {
			continue
		}
		counts[r.ConversationID] = r.Count
	}
	return counts, nil
}
