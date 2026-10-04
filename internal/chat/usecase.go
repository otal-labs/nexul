package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/otal-labs/nexul/internal/harness"
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
	threads   ThreadGate
	standing  Standing
	members   Membership
	noteLive  NoteLive
	now       func() time.Time
}

// NoteLive fences a note file write against the note's live room, then resets the room so the write wins (ADR 0109).
type NoteLive interface {
	Reset(ctx context.Context, messageID string, write func(context.Context) error) error
}

// SetNoteLive wires the live rooms a note file write must win over; without it a write touches only the file.
func (s *Service) SetNoteLive(l NoteLive) { s.noteLive = l }

// Membership answers who belongs to a workspace, so a DM or a private channel only ever holds its own members.
type Membership interface {
	IsMember(ctx context.Context, userID, workspaceID string) (bool, error)
	MemberIDs(ctx context.Context, workspaceID string) ([]string, error)
}

// ThreadGate checks a ticket or interview thread through its project, so threads follow Project access (ADR 0097).
type ThreadGate interface {
	RequireTicket(ctx context.Context, ticketID string, action permissions.Action) error
	RequireProject(ctx context.Context, projectID string, action permissions.Action) error
}

// SetThreadGate wires the per-project thread check; unset, a thread is checked in its workspace instead.
func (s *Service) SetThreadGate(g ThreadGate) { s.threads = g }

// Standing: the Owner reads every private channel (ADR 0098), a Restricted member no public one (ADR 0097).
type Standing interface {
	IsOwner(ctx context.Context, userID, workspaceID string) (bool, error)
	IsRestricted(ctx context.Context, userID, workspaceID string) (bool, error)
}

// SetStanding wires the Owner and Restricted member lookups; unset, nobody is either.
func (s *Service) SetStanding(st Standing) { s.standing = st }

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

// projectThread reports a kind read through its project: a ticket or an interview thread.
func projectThread(kind Kind) bool {
	return kind == KindTicketThread || kind == KindInterviewThread
}

// requireGate is the gate's part of reading c: a ticket or interview thread through its project, else membership.
func (s *Service) requireGate(ctx context.Context, c *Conversation) error {
	switch c.Kind {
	case KindTicketThread:
		if s.threads == nil {
			return s.require(ctx, c.WorkspaceID, permissions.TicketsRead)
		}
		return s.threads.RequireTicket(ctx, c.TicketID, permissions.TicketsRead)
	case KindInterviewThread:
		if s.threads == nil {
			return s.require(ctx, c.WorkspaceID, permissions.MemoriesRead)
		}
		return s.threads.RequireProject(ctx, c.ProjectID, permissions.MemoriesRead)
	}
	return s.require(ctx, c.WorkspaceID, permissions.Member)
}

// requireRead checks callerID (the actor when empty) may read c: the gate's part, then the person's own.
func (s *Service) requireRead(ctx context.Context, c *Conversation, callerID string) error {
	if err := s.requireGate(ctx, c); err != nil {
		return err
	}
	if callerID == "" {
		actor, ok := identity.ActorFromCtx(ctx)
		if !ok {
			return nil
		}
		callerID = actor.ID
	}
	return s.reads(ctx, s.viewer(callerID, c.WorkspaceID), c)
}

// reads is the person's part of reading c: docs:thread on a doc thread, a DM's participants, a channel's standing.
func (s *Service) reads(ctx context.Context, v *viewer, c *Conversation) error {
	if c.Kind == KindDocThread && !s.canThread(ctx, v.userID, c.DocID) {
		return fmt.Errorf("%w: docs:thread required on doc %s", apperrs.ErrForbidden, c.DocID)
	}
	if c.Kind == KindDM && !slices.Contains(c.ParticipantIDs, v.userID) {
		return fmt.Errorf("%w: conversation %s", apperrs.ErrNotFound, c.ID)
	}
	if c.Kind != KindChannel && c.Kind != KindVoiceChannel {
		return nil
	}
	visible, err := v.readsChannel(ctx, c)
	if err != nil {
		return err
	}
	if !visible {
		return fmt.Errorf("%w: conversation %s", apperrs.ErrNotFound, c.ID)
	}
	return nil
}

// viewer is one person in one workspace; their standing is looked up at most once, and only when a channel needs it.
type viewer struct {
	standing            Standing
	userID, workspaceID string
	owner, restricted   *bool
}

func (s *Service) viewer(userID, workspaceID string) *viewer {
	return &viewer{standing: s.standing, userID: userID, workspaceID: workspaceID}
}

// readsChannel: a private channel is read by its members and the Owner, a public one by everyone but a Restricted member.
func (v *viewer) readsChannel(ctx context.Context, c *Conversation) (bool, error) {
	if !c.Private {
		restricted, err := v.memo(ctx, &v.restricted, Standing.IsRestricted)
		return !restricted, err
	}
	if slices.Contains(c.ParticipantIDs, v.userID) {
		return true, nil
	}
	return v.memo(ctx, &v.owner, Standing.IsOwner)
}

func (v *viewer) memo(ctx context.Context, cached **bool, ask func(Standing, context.Context, string, string) (bool, error)) (bool, error) {
	if *cached != nil {
		return **cached, nil
	}
	if v.standing == nil || v.userID == "" {
		return false, nil
	}
	answer, err := ask(v.standing, ctx, v.userID, v.workspaceID)
	if err != nil {
		return false, fmt.Errorf("standing of %s in workspace %s: %w", v.userID, v.workspaceID, err)
	}
	*cached = &answer
	return answer, nil
}

// NewService wires the chat use-cases over the given repo.
func NewService(repo Repo) *Service {
	return &Service{repo: repo, now: time.Now}
}

// SetDocAccess wires the access domain's docs:thread check (ADR 0017); a nil checker fails closed.
func (s *Service) SetDocAccess(a DocAccess) {
	s.docAccess = a
}

// CreateChannel creates a workspace-scoped public channel; it takes channels:write in the workspace.
func (s *Service) CreateChannel(ctx context.Context, workspaceID, creatorUserID, name string) (*Conversation, error) {
	return s.createChannelKind(ctx, &Conversation{WorkspaceID: workspaceID, Kind: KindChannel, Name: name, CreatedBy: creatorUserID}, nil)
}

// CreateVoiceChannel creates a public voice channel; internal/voice owns the LiveKit side.
func (s *Service) CreateVoiceChannel(ctx context.Context, workspaceID, creatorUserID, name string) (*Conversation, error) {
	return s.createChannelKind(ctx, &Conversation{WorkspaceID: workspaceID, Kind: KindVoiceChannel, Name: name, CreatedBy: creatorUserID}, nil)
}

// CreatePrivateChannel creates a text or voice channel whose members are its creator and memberIDs (ADR 0098).
func (s *Service) CreatePrivateChannel(ctx context.Context, workspaceID, creatorUserID, name string, kind Kind, memberIDs []string) (*Conversation, error) {
	if kind != KindChannel && kind != KindVoiceChannel {
		return nil, fmt.Errorf("%w: only a channel or voice channel can be private, not a %s", apperrs.ErrInvalid, kind)
	}
	return s.createChannelKind(ctx, &Conversation{WorkspaceID: workspaceID, Kind: kind, Name: name, CreatedBy: creatorUserID, Private: true}, memberIDs)
}

// createChannelKind is the shared create path; a workspace's own #general needs no channels:write.
func (s *Service) createChannelKind(ctx context.Context, c *Conversation, memberIDs []string) (*Conversation, error) {
	c.WorkspaceID = strings.TrimSpace(c.WorkspaceID)
	if c.WorkspaceID == "" {
		return nil, fmt.Errorf("%w: workspace id is required", apperrs.ErrInvalid)
	}
	c.CreatedBy = strings.TrimSpace(c.CreatedBy)
	if c.CreatedBy == "" {
		return nil, fmt.Errorf("%w: creator id is required", apperrs.ErrInvalid)
	}
	name, err := channelName(c.Name)
	if err != nil {
		return nil, err
	}
	c.Name = name
	if c.General {
		return s.createConversation(ctx, c, []string{c.CreatedBy})
	}
	if err := s.require(ctx, c.WorkspaceID, permissions.ChannelsWrite); err != nil {
		return nil, err
	}
	if !c.Private {
		if err := s.refuseRestricted(ctx, c.CreatedBy, c.WorkspaceID); err != nil {
			return nil, err
		}
		return s.createConversation(ctx, c, []string{c.CreatedBy})
	}
	members := dedupeParticipants(c.CreatedBy, memberIDs)
	if err := s.requireMembers(ctx, c.WorkspaceID, members); err != nil {
		return nil, err
	}
	c.ParticipantIDs = members
	return s.createConversation(ctx, c, members)
}

// refuseRestricted refuses a Restricted member anything that leaves a channel public, which they could not then read.
func (s *Service) refuseRestricted(ctx context.Context, userID, workspaceID string) error {
	v := s.viewer(userID, workspaceID)
	restricted, err := v.memo(ctx, &v.restricted, Standing.IsRestricted)
	if err != nil {
		return err
	}
	if restricted {
		return fmt.Errorf("%w: a member who sees only some projects may only have private channels", apperrs.ErrForbidden)
	}
	return nil
}

// channelName is the one validation a channel's name goes through, on create and on rename.
func channelName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("%w: channel name is required", apperrs.ErrInvalid)
	}
	return name, nil
}

// EnsureGeneralChannel idempotently creates a workspace's #general channel (tenancy's ChannelGate seam).
func (s *Service) EnsureGeneralChannel(ctx context.Context, workspaceID, creatorUserID string) error {
	_, err := s.createChannelKind(ctx, &Conversation{WorkspaceID: workspaceID, Kind: KindChannel, Name: GeneralChannelName, CreatedBy: creatorUserID, General: true}, nil)
	if err != nil && !errors.Is(err, apperrs.ErrConflict) {
		return fmt.Errorf("ensure general channel for workspace %s: %w", workspaceID, err)
	}
	return nil
}

// RenameChannel renames a text or voice channel through the same name rules as creating one; it takes channels:write.
func (s *Service) RenameChannel(ctx context.Context, id, name string) (*Conversation, error) {
	name, err := channelName(name)
	if err != nil {
		return nil, err
	}
	c, err := s.manageableChannel(ctx, id, permissions.ChannelsWrite)
	if err != nil {
		return nil, err
	}
	if c.Name == name {
		return c, nil
	}
	renamed := *c
	renamed.Name, renamed.UpdatedAt = name, s.now().UTC()
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicConversationUpdated, Payload: ConversationUpdatedEvent{
		ConversationID: c.ID, WorkspaceID: c.WorkspaceID, Kind: c.Kind, Name: name, PreviousName: c.Name, ActorID: actorID(ctx),
		MembersOnly: c.Private,
	}}
	if err := s.repo.RenameConversation(ctx, c.ID, name, renamed.UpdatedAt, evt); err != nil {
		return nil, fmt.Errorf("rename %s conversation: %w", c.Kind, err)
	}
	return &renamed, nil
}

// DeleteChannel deletes a text or voice channel with its messages under channels:delete; #general is never deleted.
func (s *Service) DeleteChannel(ctx context.Context, id string) (*Conversation, error) {
	c, err := s.manageableChannel(ctx, id, permissions.ChannelsDelete)
	if err != nil {
		return nil, err
	}
	if c.General {
		return nil, fmt.Errorf("%w: #%s is the workspace's general channel, which can be renamed but not deleted", apperrs.ErrInvalid, c.Name)
	}
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicConversationDeleted, Payload: ConversationDeletedEvent{
		ConversationID: c.ID, WorkspaceID: c.WorkspaceID, Kind: c.Kind, Name: c.Name, ActorID: actorID(ctx),
		Private: c.Private, MemberIDs: c.ParticipantIDs, MembersOnly: c.Private,
	}}
	if err := s.repo.DeleteConversation(ctx, c.ID, evt); err != nil {
		return nil, fmt.Errorf("delete %s conversation: %w", c.Kind, err)
	}
	return c, nil
}

// ReadsDeleted reports whether the caller could read the channel a chat.conversation.deleted event removed.
func (s *Service) ReadsDeleted(ctx context.Context, e ConversationDeletedEvent) bool {
	c := &Conversation{ID: e.ConversationID, WorkspaceID: e.WorkspaceID, Kind: e.Kind, Private: e.Private, ParticipantIDs: e.MemberIDs}
	return s.requireRead(ctx, c, "") == nil
}

// SetChannelPrivate switches a channel private, keeping the caller and keepUserIDs, or public (ADR 0098).
func (s *Service) SetChannelPrivate(ctx context.Context, id string, private bool, keepUserIDs []string) (*Conversation, error) {
	c, err := s.manageableChannel(ctx, id, permissions.ChannelsWrite)
	if err != nil {
		return nil, err
	}
	if c.General {
		return nil, fmt.Errorf("%w: #%s is the workspace's general channel, which is always public", apperrs.ErrInvalid, c.Name)
	}
	if c.Private == private {
		return c, nil
	}
	actor := actorID(ctx)
	if !private {
		if err := s.refuseRestricted(ctx, actor, c.WorkspaceID); err != nil {
			return nil, err
		}
		return s.switchPrivate(ctx, c, false, nil, nil)
	}
	members := cleanIDs(append([]string{actor}, keepUserIDs...))
	if len(members) == 0 {
		return nil, fmt.Errorf("%w: a private channel needs at least one member", apperrs.ErrInvalid)
	}
	if err := s.requireMembers(ctx, c.WorkspaceID, members); err != nil {
		return nil, err
	}
	everyone, err := s.workspaceMemberIDs(ctx, c.WorkspaceID)
	if err != nil {
		return nil, err
	}
	removed := slices.DeleteFunc(everyone, func(id string) bool { return slices.Contains(members, id) })
	return s.switchPrivate(ctx, c, true, members, removed)
}

func (s *Service) switchPrivate(ctx context.Context, c *Conversation, private bool, members, removed []string) (*Conversation, error) {
	switched := *c
	switched.Private, switched.ParticipantIDs, switched.UpdatedAt = private, members, s.now().UTC()
	evt := s.membersChanged(ctx, &switched, members, removed)
	if err := s.repo.SetChannelPrivate(ctx, c.ID, private, members, switched.UpdatedAt, evt); err != nil {
		return nil, fmt.Errorf("set %s conversation private: %w", c.Kind, err)
	}
	return &switched, nil
}

// AddChannelMembers adds workspace members to a private channel; anyone who reads it may (ADR 0098).
func (s *Service) AddChannelMembers(ctx context.Context, id string, userIDs []string) (*Conversation, error) {
	c, err := s.privateChannel(ctx, id)
	if err != nil {
		return nil, err
	}
	added := slices.DeleteFunc(cleanIDs(userIDs), func(id string) bool { return slices.Contains(c.ParticipantIDs, id) })
	if len(added) == 0 {
		return c, nil
	}
	if err := s.requireMembers(ctx, c.WorkspaceID, added); err != nil {
		return nil, err
	}
	updated := *c
	updated.ParticipantIDs = append(slices.Clone(c.ParticipantIDs), added...)
	evt := s.membersChanged(ctx, &updated, added, nil)
	if err := s.repo.AddParticipants(ctx, c.ID, added, s.now().UTC(), evt); err != nil {
		return nil, fmt.Errorf("add members to %s conversation: %w", c.Kind, err)
	}
	return &updated, nil
}

// RemoveChannelMembers removes people from a private channel; removing anyone but yourself takes channels:write.
func (s *Service) RemoveChannelMembers(ctx context.Context, id string, userIDs []string) (*Conversation, error) {
	c, err := s.privateChannel(ctx, id)
	if err != nil {
		return nil, err
	}
	userIDs = cleanIDs(userIDs)
	if len(userIDs) == 0 {
		return nil, fmt.Errorf("%w: name at least one member to remove", apperrs.ErrInvalid)
	}
	actor := actorID(ctx)
	if slices.ContainsFunc(userIDs, func(id string) bool { return id != actor }) {
		if err := s.require(ctx, c.WorkspaceID, permissions.ChannelsWrite); err != nil {
			return nil, err
		}
	}
	removed := slices.DeleteFunc(slices.Clone(userIDs), func(id string) bool { return !slices.Contains(c.ParticipantIDs, id) })
	if len(removed) == 0 {
		return c, nil
	}
	remaining := slices.DeleteFunc(slices.Clone(c.ParticipantIDs), func(id string) bool { return slices.Contains(removed, id) })
	if len(remaining) == 0 {
		return nil, fmt.Errorf("%w: the last member cannot leave #%s; make it public or delete it instead", apperrs.ErrInvalid, c.Name)
	}
	updated := *c
	updated.ParticipantIDs = remaining
	evt := s.membersChanged(ctx, &updated, nil, removed)
	if err := s.repo.RemoveParticipants(ctx, c.ID, removed, evt); err != nil {
		return nil, fmt.Errorf("remove members from %s conversation: %w", c.Kind, err)
	}
	return &updated, nil
}

// privateChannel checks, in order, that the caller reads the conversation and that it is a private channel.
func (s *Service) privateChannel(ctx context.Context, id string) (*Conversation, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("%w: conversation id is required", apperrs.ErrInvalid)
	}
	c, err := s.repo.GetConversation(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get conversation %s: %w", id, err)
	}
	if err := s.requireRead(ctx, c, ""); err != nil {
		return nil, err
	}
	if c.Kind != KindChannel && c.Kind != KindVoiceChannel {
		return nil, fmt.Errorf("%w: only a channel or voice channel has members to manage, not a %s", apperrs.ErrInvalid, c.Kind)
	}
	if !c.Private {
		return nil, fmt.Errorf("%w: #%s is public, so every member reads it; make it private to choose its members", apperrs.ErrInvalid, c.Name)
	}
	return c, nil
}

func (s *Service) membersChanged(ctx context.Context, c *Conversation, added, removed []string) eventbus.OutboxEvent {
	return eventbus.OutboxEvent{ID: ids.New(), Topic: TopicConversationMembersChanged, Payload: ConversationMembersChangedEvent{
		ConversationID: c.ID, WorkspaceID: c.WorkspaceID, Private: c.Private,
		AddedUserIDs: nonNil(added), RemovedUserIDs: nonNil(removed), ActorID: actorID(ctx), MembersOnly: c.Private,
	}}
}

func (s *Service) workspaceMemberIDs(ctx context.Context, workspaceID string) ([]string, error) {
	if s.members == nil {
		return nil, nil
	}
	ids, err := s.members.MemberIDs(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list members of workspace %s: %w", workspaceID, err)
	}
	return ids, nil
}

// cleanIDs trims blanks and drops duplicates, keeping order.
func cleanIDs(in []string) []string {
	out := make([]string, 0, len(in))
	for _, id := range in {
		if id = strings.TrimSpace(id); id != "" && !slices.Contains(out, id) {
			out = append(out, id)
		}
	}
	return out
}

func nonNil(ids []string) []string {
	if ids == nil {
		return []string{}
	}
	return ids
}

// manageableChannel checks, in order, that the caller reads the conversation, that it is a channel, and holds action.
func (s *Service) manageableChannel(ctx context.Context, id string, action permissions.Action) (*Conversation, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("%w: conversation id is required", apperrs.ErrInvalid)
	}
	c, err := s.repo.GetConversation(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get conversation %s: %w", id, err)
	}
	if err := s.requireRead(ctx, c, ""); err != nil {
		return nil, err
	}
	if c.Kind != KindChannel && c.Kind != KindVoiceChannel {
		return nil, fmt.Errorf("%w: only a channel or voice channel can be renamed or deleted, not a %s", apperrs.ErrInvalid, c.Kind)
	}
	if err := s.require(ctx, c.WorkspaceID, action); err != nil {
		return nil, err
	}
	return c, nil
}

func actorID(ctx context.Context) string {
	actor, _ := identity.ActorFromCtx(ctx)
	return actor.ID
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

// listed keeps the conversations the caller reads; membership was checked once, and threads ask their project.
func (s *Service) listed(ctx context.Context, workspaceID, userID string, cs []*Conversation) ([]*Conversation, error) {
	v := s.viewer(userID, workspaceID)
	out := make([]*Conversation, 0, len(cs))
	for _, c := range cs {
		err := s.reads(ctx, v, c)
		if err == nil && projectThread(c.Kind) {
			err = s.requireGate(ctx, c)
		}
		if err != nil && !permissions.Refused(err) {
			return nil, err
		}
		if err == nil {
			out = append(out, c)
		}
	}
	return out, nil
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
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicConversationCreated, Payload: ConversationCreatedEvent{Conversation: *c, MembersOnly: c.membersOnly()}}
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
	// ponytail: one check per thread through its ticket's project; ask once per project if boards get slow.
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
	return s.listed(ctx, workspaceID, userID, cs)
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

// PostAgentMessage persists the Agent's final turn reply with the work it handed off; in-progress text only streams live.
func (s *Service) PostAgentMessage(ctx context.Context, conversationID, viaUserID, body string, handoffs []harness.Handoff) (*Message, error) {
	m, err := s.newMessage(conversationID, viaUserID, body, AuthorAgent)
	if err != nil {
		return nil, err
	}
	if m.Handoffs, err = storedHandoffs(handoffs); err != nil {
		return nil, err
	}
	return s.create(ctx, m)
}

// maxHandoffBytes caps the hand-offs one reply stores, as JSON.
const maxHandoffBytes = 256 << 10

// storedHandoffs is hs as a reply stores them, nil for none, within maxHandoffBytes: the oldest steps go first, then the longest replies.
func storedHandoffs(hs []harness.Handoff) ([]Handoff, error) {
	if len(hs) == 0 {
		return nil, nil
	}
	out := make([]Handoff, 0, len(hs))
	for _, h := range hs {
		out = append(out, NewHandoff(h))
	}
	raw, err := json.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("measure hand-offs: %w", err)
	}
	type stepRef struct {
		handoff, step int
		at            time.Time
		size          int
	}
	var refs []stepRef
	for i, h := range out {
		for j, s := range h.Steps {
			step, err := json.Marshal(s)
			if err != nil {
				return nil, fmt.Errorf("measure a hand-off step: %w", err)
			}
			refs = append(refs, stepRef{i, j, s.At, len(step)})
		}
	}
	slices.SortStableFunc(refs, func(a, b stepRef) int { return a.at.Compare(b.at) })
	size, drop := len(raw), map[[2]int]bool{}
	for _, r := range refs {
		if size <= maxHandoffBytes {
			break
		}
		size -= r.size
		drop[[2]int{r.handoff, r.step}] = true
	}
	for i := range out {
		kept := out[i].Steps[:0]
		for j, s := range out[i].Steps {
			if !drop[[2]int{i, j}] {
				kept = append(kept, s)
			}
		}
		out[i].Steps = kept
	}
	cutReplies(out, size)
	return out, nil
}

// cutReplies cuts out's longest replies to even shares of what the rest leaves under maxHandoffBytes, size being out's JSON length.
func cutReplies(out []Handoff, size int) {
	if size <= maxHandoffBytes {
		return
	}
	lens, order := make([]int, len(out)), make([]int, len(out))
	budget := maxHandoffBytes - size
	for i, h := range out {
		lens[i], order[i] = jsonLen(h.Reply), i
		budget += lens[i]
	}
	slices.SortFunc(order, func(a, b int) int { return lens[a] - lens[b] })
	for k, i := range order {
		share := budget / (len(order) - k)
		if lens[i] > share {
			// An escape never shrinks a byte, so cutting the excess as raw bytes cuts at least as much JSON.
			out[i].Reply = harness.CapBytes(out[i].Reply, max(len(out[i].Reply)-(lens[i]-share), 0))
			lens[i] = share
		}
		budget -= lens[i]
	}
}

func jsonLen(s string) int {
	raw, _ := json.Marshal(s) // marshalling a string cannot fail
	return len(raw)
}

// PostSystemMessage posts an inline system message attributed to the user it's for.
func (s *Service) PostSystemMessage(ctx context.Context, conversationID, viaUserID, body string) (*Message, error) {
	return s.postMessage(ctx, conversationID, viaUserID, body, AuthorSystem)
}

func (s *Service) postMessage(ctx context.Context, conversationID, authorID, body string, kind AuthorKind) (*Message, error) {
	m, err := s.newMessage(conversationID, authorID, body, kind)
	if err != nil {
		return nil, err
	}
	return s.create(ctx, m)
}

func (s *Service) create(ctx context.Context, m *Message) (*Message, error) {
	membersOnly, err := s.MembersOnly(ctx, m.ConversationID)
	if err != nil {
		return nil, fmt.Errorf("post message to conversation %s: %w", m.ConversationID, err)
	}
	if err := s.repo.CreateMessage(ctx, m, messageCreated(m, membersOnly)); err != nil {
		return nil, fmt.Errorf("post message to conversation %s: %w", m.ConversationID, err)
	}
	return m, nil
}

func (s *Service) newMessage(conversationID, authorID, body string, kind AuthorKind) (*Message, error) {
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
	return &Message{
		ID:             ids.New(),
		ConversationID: conversationID,
		AuthorID:       authorID,
		AuthorKind:     kind,
		Body:           body,
		Mentions:       ParseMentions(body),
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

func messageCreated(m *Message, membersOnly bool) eventbus.OutboxEvent {
	return eventbus.OutboxEvent{ID: ids.New(), Topic: TopicMessageCreated, Payload: MessageCreatedEvent{Message: *m, MembersOnly: membersOnly}}
}

// maxNoteFileBytes matches the attachment cap (ADR 0027), since a note's file is stored as an attachment.
const maxNoteFileBytes = 10 << 20

const maxNoteNameLen = 252

// PostNote posts a note: an Agent message on a ticket's thread, on callerID's behalf, carrying a markdown file
// (ADR 0108). It takes tickets:write on the ticket, and starts no turn, since only a person's message does.
func (s *Service) PostNote(ctx context.Context, conversationID, callerID, body string, in NoteFileInput) (*Message, *NoteFile, error) {
	m, err := s.newMessage(conversationID, callerID, body, AuthorAgent)
	if err != nil {
		return nil, nil, err
	}
	c, err := s.noteThread(ctx, m.ConversationID, m.AuthorID)
	if err != nil {
		return nil, nil, err
	}
	file, err := newNoteFile(in)
	if err != nil {
		return nil, nil, err
	}
	m.AttachmentID = file.ID
	if err := s.repo.CreateNote(ctx, m, file, messageCreated(m, c.membersOnly())); err != nil {
		return nil, nil, fmt.Errorf("post note to conversation %s: %w", c.ID, err)
	}
	return m, file, nil
}

// noteThread checks, in order, that callerID reads the conversation, that it is a ticket's thread, and tickets:write.
func (s *Service) noteThread(ctx context.Context, conversationID, callerID string) (*Conversation, error) {
	return s.noteThreadFor(ctx, conversationID, callerID, permissions.TicketsWrite)
}

// noteThreadFor is noteThread with the ticket action to require.
func (s *Service) noteThreadFor(ctx context.Context, conversationID, callerID string, action permissions.Action) (*Conversation, error) {
	c, err := s.repo.GetConversation(ctx, conversationID)
	if err != nil {
		return nil, fmt.Errorf("get conversation %s: %w", conversationID, err)
	}
	if err := s.requireRead(ctx, c, callerID); err != nil {
		return nil, err
	}
	if c.Kind != KindTicketThread {
		return nil, fmt.Errorf("%w: only a ticket's thread takes a note's file, not a %s", apperrs.ErrInvalid, c.Kind)
	}
	if s.threads == nil {
		return c, s.require(ctx, c.WorkspaceID, action)
	}
	return c, s.threads.RequireTicket(ctx, c.TicketID, action)
}

func newNoteFile(in NoteFileInput) (*NoteFile, error) {
	if strings.TrimSpace(in.Markdown) == "" {
		return nil, fmt.Errorf("%w: a note's file needs its markdown", apperrs.ErrInvalid)
	}
	if err := noteFileFits(in.Markdown); err != nil {
		return nil, err
	}
	return &NoteFile{ID: ids.New(), Name: noteFileName(in.Name), Markdown: in.Markdown}, nil
}

func noteFileFits(markdown string) error {
	if len(markdown) > maxNoteFileBytes {
		return fmt.Errorf("%w: a note's file exceeds %d bytes", apperrs.ErrInvalid, maxNoteFileBytes)
	}
	return nil
}

// ReplaceNote replaces a note's file from outside its live room under tickets:write; open editors reload from it.
func (s *Service) ReplaceNote(ctx context.Context, messageID, callerID, markdown string) (*Message, *NoteFile, error) {
	if strings.TrimSpace(markdown) == "" {
		return nil, nil, fmt.Errorf("%w: a note's file needs its markdown", apperrs.ErrInvalid)
	}
	if err := noteFileFits(markdown); err != nil {
		return nil, nil, err
	}
	m, err := s.repo.GetMessage(ctx, strings.TrimSpace(messageID))
	if err != nil {
		return nil, nil, fmt.Errorf("replace note %s: %w", messageID, err)
	}
	c, err := s.noteThread(ctx, m.ConversationID, callerID)
	if err != nil {
		return nil, nil, err
	}
	if m.DeletedAt != nil {
		return nil, nil, fmt.Errorf("%w: message %s was deleted", apperrs.ErrNotFound, m.ID)
	}
	if m.AttachmentID == "" {
		return nil, nil, fmt.Errorf("%w: message %s is not a note, so it has no file to replace", apperrs.ErrInvalid, m.ID)
	}
	files, err := s.repo.ListNoteFiles(ctx, []string{m.AttachmentID})
	if err != nil {
		return nil, nil, fmt.Errorf("replace note %s: %w", m.ID, err)
	}
	if len(files) == 0 {
		return nil, nil, fmt.Errorf("replace note %s: %w", m.ID, apperrs.ErrNotFound)
	}
	write := func(ctx context.Context) error { return s.writeNote(ctx, m, c.membersOnly(), markdown) }
	if err := s.fenceNote(ctx, m.ID, write); err != nil {
		return nil, nil, err
	}
	return m, &NoteFile{ID: files[0].ID, Name: files[0].Name, Markdown: markdown}, nil
}

// fenceNote runs write through the note's live room when one is wired, so open editors cannot overwrite it.
func (s *Service) fenceNote(ctx context.Context, messageID string, write func(context.Context) error) error {
	if s.noteLive == nil {
		return write(ctx)
	}
	return s.noteLive.Reset(ctx, messageID, write)
}

// CommitNote writes a note's live room into its file; joining the room already took tickets:write.
func (s *Service) CommitNote(ctx context.Context, messageID, markdown string) error {
	if err := noteFileFits(markdown); err != nil {
		return err
	}
	m, err := s.repo.GetMessage(ctx, messageID)
	if err != nil {
		return fmt.Errorf("commit note %s: %w", messageID, err)
	}
	if m.DeletedAt != nil || m.AttachmentID == "" {
		return fmt.Errorf("%w: note %s was deleted", apperrs.ErrConflict, messageID)
	}
	membersOnly, err := s.MembersOnly(ctx, m.ConversationID)
	if err != nil {
		return fmt.Errorf("commit note %s: %w", messageID, err)
	}
	return s.writeNote(ctx, m, membersOnly, markdown)
}

// writeNote replaces m's file and publishes chat.message.updated in the same transaction.
func (s *Service) writeNote(ctx context.Context, m *Message, membersOnly bool, markdown string) error {
	m.UpdatedAt = s.now().UTC()
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicMessageUpdated, Payload: MessageUpdatedEvent{Message: *m, MembersOnly: membersOnly}}
	if err := s.repo.ReplaceNoteFile(ctx, m, markdown, m.UpdatedAt, evt); err != nil {
		return fmt.Errorf("replace note %s: %w", m.ID, err)
	}
	return nil
}

// NoteLocked reports a message whose file refuses live edits: one that is deleted or is not a note.
func (s *Service) NoteLocked(ctx context.Context, messageID string) (bool, error) {
	m, err := s.repo.GetMessage(ctx, messageID)
	if err != nil {
		return false, err
	}
	return m.DeletedAt != nil || m.AttachmentID == "", nil
}

// CanJoinNote reports whether userID may join a note's live room under action, checked on the note's ticket.
func (s *Service) CanJoinNote(ctx context.Context, userID, messageID string, action permissions.Action) (bool, error) {
	m, err := s.repo.GetMessage(ctx, messageID)
	if err != nil {
		return false, err
	}
	if m.DeletedAt != nil || m.AttachmentID == "" {
		return false, nil
	}
	if _, err := s.noteThreadFor(ctx, m.ConversationID, userID, action); err != nil {
		return false, err
	}
	return true, nil
}

// noteFileName keeps the base name, so a path never reaches storage, and makes sure it ends in .md.
func noteFileName(name string) string {
	name = path.Base(strings.TrimSpace(strings.ReplaceAll(name, "\\", "/")))
	if name == "." || name == "/" {
		name = "note"
	}
	if strings.EqualFold(path.Ext(name), ".md") {
		name = strings.TrimSuffix(name, path.Ext(name))
	}
	if len(name) > maxNoteNameLen {
		name = name[:maxNoteNameLen]
	}
	return name + ".md"
}

// NoteFiles returns the files of the notes among ms, by attachment id; ms come from a call that already checked the read.
func (s *Service) NoteFiles(ctx context.Context, ms []*Message) (map[string]*NoteFile, error) {
	var attachmentIDs []string
	for _, m := range ms {
		if m.AttachmentID != "" {
			attachmentIDs = append(attachmentIDs, m.AttachmentID)
		}
	}
	files, err := s.repo.ListNoteFiles(ctx, attachmentIDs)
	if err != nil {
		return nil, err
	}
	out := make(map[string]*NoteFile, len(files))
	for _, f := range files {
		out[f.ID] = f
	}
	return out, nil
}

// IsNoteFile reports whether a conversation's file is a note's, which goes only with its message; the server's own seam.
func (s *Service) IsNoteFile(ctx context.Context, conversationID, attachmentID string) (bool, error) {
	return s.repo.IsNoteFile(ctx, conversationID, attachmentID)
}

// MembersOnly reports whether a conversation is read by its members alone; the server's own seam, so unchecked.
func (s *Service) MembersOnly(ctx context.Context, conversationID string) (bool, error) {
	c, err := s.repo.GetConversation(ctx, conversationID)
	if err != nil {
		return false, err
	}
	return c.membersOnly(), nil
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
	membersOnly, err := s.MembersOnly(ctx, current.ConversationID)
	if err != nil {
		return nil, fmt.Errorf("edit message %s: %w", messageID, err)
	}
	now := s.now().UTC()
	updated := *current
	updated.Body = body
	updated.Mentions = ParseMentions(body)
	updated.EditedAt = &now
	updated.UpdatedAt = now
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicMessageUpdated, Payload: MessageUpdatedEvent{Message: updated, MembersOnly: membersOnly}}
	if err := s.repo.UpdateMessage(ctx, messageID, updated.Body, updated.Mentions, now, evt); err != nil {
		return nil, fmt.Errorf("edit message %s: %w", messageID, err)
	}
	return &updated, nil
}

// maxEmojiBytes fits the longest emoji sequences, such as a family joined with skin tones.
const maxEmojiBytes = 64

// React adds or removes callerID's emoji on a message; whoever reads its conversation may react, never on a deleted one.
func (s *Service) React(ctx context.Context, messageID, callerID, emoji string, reacted bool) (*Message, error) {
	messageID = strings.TrimSpace(messageID)
	if messageID == "" {
		return nil, fmt.Errorf("%w: message id is required", apperrs.ErrInvalid)
	}
	callerID = strings.TrimSpace(callerID)
	if callerID == "" {
		return nil, fmt.Errorf("%w: user id is required", apperrs.ErrInvalid)
	}
	emoji = strings.TrimSpace(emoji)
	if emoji == "" || len(emoji) > maxEmojiBytes || strings.ContainsFunc(emoji, unicode.IsSpace) {
		return nil, fmt.Errorf("%w: emoji must be a single emoji such as 👍", apperrs.ErrInvalid)
	}
	current, err := s.repo.GetMessage(ctx, messageID)
	if err != nil {
		return nil, fmt.Errorf("react to message %s: %w", messageID, err)
	}
	if current.DeletedAt != nil {
		return nil, fmt.Errorf("%w: cannot react to a deleted message", apperrs.ErrInvalid)
	}
	if err := s.requireConversation(ctx, current.ConversationID, callerID); err != nil {
		return nil, err
	}
	membersOnly, err := s.MembersOnly(ctx, current.ConversationID)
	if err != nil {
		return nil, fmt.Errorf("react to message %s: %w", messageID, err)
	}
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicMessageReactionsChanged, Payload: MessageReactionsChangedEvent{
		ConversationID: current.ConversationID, MessageID: messageID, UserID: callerID, Emoji: emoji, Reacted: reacted, MembersOnly: membersOnly,
	}}
	if err := s.repo.SetReaction(ctx, messageID, callerID, emoji, reacted, s.now().UTC(), evt); err != nil {
		return nil, fmt.Errorf("react to message %s: %w", messageID, err)
	}
	m, err := s.repo.GetMessage(ctx, messageID)
	if err != nil {
		return nil, fmt.Errorf("react to message %s: %w", messageID, err)
	}
	return m, nil
}

// DeleteMessage soft-deletes a message (author-only, a note under tickets:write); deleting an already-deleted one is a no-op.
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
	if current.AttachmentID != "" {
		return s.deleteNote(ctx, current, authorID)
	}
	if current.AuthorID != authorID {
		return fmt.Errorf("%w: only the author may delete this message", apperrs.ErrForbidden)
	}
	membersOnly, err := s.MembersOnly(ctx, current.ConversationID)
	if err != nil {
		return fmt.Errorf("delete message %s: %w", messageID, err)
	}
	now := s.now().UTC()
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicMessageDeleted, Payload: MessageDeletedEvent{ConversationID: current.ConversationID, MessageID: messageID, DeletedAt: now, MembersOnly: membersOnly}}
	if err := s.repo.DeleteMessage(ctx, messageID, now, evt); err != nil {
		return fmt.Errorf("delete message %s: %w", messageID, err)
	}
	return nil
}

// noteImageRef finds the attachment ids a note's markdown points at, the images pasted into it.
var noteImageRef = regexp.MustCompile(`/api/attachments/([\w-]+)`)

// deleteNote takes tickets:write rather than authorship, and takes the note's file and its images with it.
func (s *Service) deleteNote(ctx context.Context, m *Message, callerID string) error {
	c, err := s.noteThread(ctx, m.ConversationID, callerID)
	if err != nil {
		return err
	}
	files, err := s.repo.ListNoteFiles(ctx, []string{m.AttachmentID})
	if err != nil {
		return fmt.Errorf("delete note %s: %w", m.ID, err)
	}
	var images []string
	for _, f := range files {
		for _, ref := range noteImageRef.FindAllStringSubmatch(f.Markdown, -1) {
			images = append(images, ref[1])
		}
	}
	now := s.now().UTC()
	evt := eventbus.OutboxEvent{ID: ids.New(), Topic: TopicMessageDeleted, Payload: MessageDeletedEvent{ConversationID: c.ID, MessageID: m.ID, DeletedAt: now, MembersOnly: c.membersOnly()}}
	if err := s.repo.DeleteNote(ctx, m, slices.Compact(slices.Sorted(slices.Values(images))), now, evt); err != nil {
		return fmt.Errorf("delete note %s: %w", m.ID, err)
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

// UnreadCounts returns userID's unread message count per conversation their list shows, the nav badge's data source.
func (s *Service) UnreadCounts(ctx context.Context, workspaceID, userID string) (map[string]int, error) {
	listed, err := s.ListConversations(ctx, workspaceID, userID)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.UnreadCounts(ctx, strings.TrimSpace(workspaceID), strings.TrimSpace(userID))
	if err != nil {
		return nil, fmt.Errorf("unread counts for user %s: %w", userID, err)
	}
	shown := make(map[string]bool, len(listed))
	for _, c := range listed {
		shown[c.ID] = true
	}
	counts := make(map[string]int, len(rows))
	for _, r := range rows {
		if shown[r.ConversationID] {
			counts[r.ConversationID] = r.Count
		}
	}
	return counts, nil
}
