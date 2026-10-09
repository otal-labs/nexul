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
	// RequireTickets is RequireTicket for many tickets: the ids it lets through, each refused ticket left out.
	RequireTickets(ctx context.Context, ticketIDs []string, action permissions.Action) (map[string]bool, error)
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
		AddedUserIDs: added, RemovedUserIDs: removed, ActorID: actorID(ctx), MembersOnly: c.Private,
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
