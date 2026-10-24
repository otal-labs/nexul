package calendar

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
	"github.com/otal-labs/nexul/internal/platform/paging"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// Gate is the workspace permission check; the empty action asks only for membership.
type Gate interface {
	Require(ctx context.Context, workspaceID string, action permissions.Action) error
}

// Channel is a conversation a meeting may take place in, as chat lets the caller read it.
type Channel struct {
	ID          string
	WorkspaceID string
	Name        string
	Voice       bool
	Private     bool
	MemberIDs   []string
}

// Channels reads a conversation as the caller, so a meeting never names a channel its organiser can't see.
type Channels interface {
	Channel(ctx context.Context, id string) (*Channel, error)
}

// Members lists who belongs to a workspace, the people a meeting may invite.
type Members interface {
	ListMemberUserIDs(ctx context.Context, workspaceID string) ([]string, error)
}

// Config wires the Service's seams; the adapters live in the composition root.
type Config struct {
	Repo     Repo
	Gate     Gate
	Channels Channels
	Members  Members
}

// Service is the calendar use-case layer; every change goes out through the outbox.
type Service struct {
	repo     Repo
	gate     Gate
	channels Channels
	members  Members
	now      func() time.Time
}

// NewService wires the calendar use-cases.
func NewService(cfg Config) *Service {
	return &Service{repo: cfg.Repo, gate: cfg.Gate, channels: cfg.Channels, members: cfg.Members, now: time.Now}
}

func actorID(ctx context.Context) string {
	a, _ := identity.ActorFromCtx(ctx)
	return a.ID
}

// Create adds a meeting the caller organises; it takes calendar:write in the workspace.
func (s *Service) Create(ctx context.Context, workspaceID string, d Details) (*Meeting, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, fmt.Errorf("%w: workspace id is required", apperrs.ErrInvalid)
	}
	if err := s.gate.Require(ctx, workspaceID, permissions.CalendarWrite); err != nil {
		return nil, err
	}
	actor := actorID(ctx)
	if actor == "" {
		return nil, fmt.Errorf("%w: a meeting needs a person to organise it", apperrs.ErrUnauthorized)
	}
	now := s.now().UTC()
	m := &Meeting{
		ID: ids.New(), WorkspaceID: workspaceID, OrganiserID: actor, Repeats: RepeatNone, RemindMinutes: DefaultRemind,
		CreatedAt: now, UpdatedAt: now,
	}
	d.apply(m)
	if err := s.check(ctx, m); err != nil {
		return nil, err
	}
	m.planReminder(now, nil)
	ev := MeetingEvent{Meeting: refOf(m), ActorID: actor, UserIDs: m.Audience(), InvitedIDs: m.InviteeIDs()}
	if err := s.repo.Save(ctx, []*Meeting{m}, outboxEvent(TopicMeetingCreated, ev)); err != nil {
		return nil, fmt.Errorf("create meeting: %w", err)
	}
	return m, nil
}

// Get returns one meeting the caller organises or is invited to; to anyone else it does not exist.
func (s *Service) Get(ctx context.Context, id string) (*Meeting, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("%w: meeting id is required", apperrs.ErrInvalid)
	}
	m, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get meeting %s: %w", id, err)
	}
	if err := s.gate.Require(ctx, m.WorkspaceID, permissions.Member); err != nil {
		return nil, err
	}
	if !m.Sees(actorID(ctx)) {
		return nil, fmt.Errorf("get meeting %s: %w", id, apperrs.ErrNotFound)
	}
	return m, nil
}

// Calendar returns the caller's meetings that take place between from and to, and each time they do, in order.
func (s *Service) Calendar(ctx context.Context, workspaceID string, from, to time.Time) ([]*Meeting, []Occurrence, error) {
	if err := ValidRange(from, to); err != nil {
		return nil, nil, err
	}
	if err := s.gate.Require(ctx, workspaceID, permissions.Member); err != nil {
		return nil, nil, err
	}
	ms, err := s.repo.ListVisible(ctx, workspaceID, actorID(ctx), RangeFilter{From: from, To: to})
	if err != nil {
		return nil, nil, fmt.Errorf("list meetings of workspace %s: %w", workspaceID, err)
	}
	byMeeting, err := s.Expand(ctx, ms, from, to)
	if err != nil {
		return nil, nil, err
	}
	var occurrences []Occurrence
	var shown []*Meeting
	for _, m := range ms {
		if len(byMeeting[m.ID]) == 0 {
			continue
		}
		shown = append(shown, m)
		occurrences = append(occurrences, byMeeting[m.ID]...)
	}
	slices.SortStableFunc(occurrences, func(a, b Occurrence) int { return a.StartsAt.Compare(b.StartsAt) })
	return shown, occurrences, nil
}

// Page returns one window of the caller's meetings in a workspace, by start, and how many there are.
func (s *Service) Page(ctx context.Context, workspaceID string, f RangeFilter, w paging.Window) ([]*Meeting, int, error) {
	if !f.From.IsZero() && !f.To.IsZero() && !f.To.After(f.From) {
		return nil, 0, fmt.Errorf("%w: from comes before to", apperrs.ErrInvalid)
	}
	if err := s.gate.Require(ctx, workspaceID, permissions.Member); err != nil {
		return nil, 0, err
	}
	ms, total, err := s.repo.PageVisible(ctx, workspaceID, actorID(ctx), f, w.Clamped())
	if err != nil {
		return nil, 0, fmt.Errorf("list meetings of workspace %s: %w", workspaceID, err)
	}
	return ms, total, nil
}

// Expand returns each meeting's occurrences overlapping [from, to); a zero to gives each its next one only.
func (s *Service) Expand(ctx context.Context, ms []*Meeting, from, to time.Time) (map[string][]Occurrence, error) {
	replaced, err := s.replaced(ctx, ms)
	if err != nil {
		return nil, err
	}
	out := make(map[string][]Occurrence, len(ms))
	for _, m := range ms {
		out[m.ID] = m.Occurrences(from, to, replaced[m.ID])
	}
	return out, nil
}

// replaced maps each repeating meeting to the slots its own changed occurrences stand in for.
func (s *Service) replaced(ctx context.Context, ms []*Meeting) (map[string]map[int64]bool, error) {
	var series []string
	for _, m := range ms {
		if m.Repeating() {
			series = append(series, m.ID)
		}
	}
	out := map[string]map[int64]bool{}
	if len(series) == 0 {
		return out, nil
	}
	rs, err := s.repo.Replacements(ctx, series)
	if err != nil {
		return nil, fmt.Errorf("read changed occurrences: %w", err)
	}
	for _, r := range rs {
		if r.ReplacesAt == nil {
			continue
		}
		if out[r.SeriesID] == nil {
			out[r.SeriesID] = map[int64]bool{}
		}
		out[r.SeriesID][r.ReplacesAt.Unix()] = true
	}
	return out, nil
}

// Update changes a meeting, cancels it or restores it. Its organiser needs calendar:write, anyone else who sees it
// calendar:delete. On a repeating meeting a Slot picks one occurrence and Scope whether the change covers it alone
// or it and every later one. Returns the meeting the change landed on, a new one when it split the series.
func (s *Service) Update(ctx context.Context, id string, c Change) (*Meeting, error) {
	m, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.requireChange(ctx, m); err != nil {
		return nil, err
	}
	if c.Slot == nil || !m.Repeating() {
		return s.changeWhole(ctx, m, c)
	}
	slot := c.Slot.UTC()
	replacements, err := s.repo.Replacements(ctx, []string{m.ID})
	if err != nil {
		return nil, fmt.Errorf("read changed occurrences of meeting %s: %w", m.ID, err)
	}
	if !m.HasSlot(slot, slotSet(replacements)) {
		return nil, fmt.Errorf("%w: %s is not an occurrence of meeting %s; meeting_list lists its occurrences", apperrs.ErrInvalid, slot.Format(time.RFC3339), m.ID)
	}
	switch c.Scope {
	case ScopeThis:
		return s.changeOne(ctx, m, slot, replacements, c)
	case ScopeFollowing:
		if slot.Equal(m.StartsAt) {
			return s.changeWhole(ctx, m, c)
		}
		return s.changeFollowing(ctx, m, slot, replacements, c)
	}
	return nil, fmt.Errorf("%w: a change to one occurrence has scope this or following", apperrs.ErrInvalid)
}

// requireChange lets the organiser change their meeting with calendar:write, and anyone holding calendar:delete.
func (s *Service) requireChange(ctx context.Context, m *Meeting) error {
	if m.OrganiserID == actorID(ctx) {
		err := s.gate.Require(ctx, m.WorkspaceID, permissions.CalendarWrite)
		if err == nil || !permissions.Refused(err) {
			return err
		}
	}
	err := s.gate.Require(ctx, m.WorkspaceID, permissions.CalendarDelete)
	if permissions.Refused(err) {
		return fmt.Errorf("%w: only the organiser or someone holding calendar:delete changes this meeting", apperrs.ErrForbidden)
	}
	return err
}

// changeWhole patches every occurrence of m; a new start moves the occurrences changed on their own along with it.
func (s *Service) changeWhole(ctx context.Context, m *Meeting, c Change) (*Meeting, error) {
	before := m.clone()
	now := s.now().UTC()
	c.apply(m)
	applyCancel(m, c.Cancelled, now)
	if err := s.check(ctx, m); err != nil {
		return nil, err
	}
	m.UpdatedAt = now
	saved := []*Meeting{m}
	evts := []eventbus.OutboxEvent{changeEvent(ctx, before, m)}
	replaced := map[int64]bool{}
	if m.Repeating() {
		rs, err := s.repo.Replacements(ctx, []string{m.ID})
		if err != nil {
			return nil, fmt.Errorf("read changed occurrences of meeting %s: %w", m.ID, err)
		}
		shift := m.StartsAt.Sub(before.StartsAt)
		for _, r := range rs {
			if shift != 0 && r.ReplacesAt != nil {
				// ponytail: a moved series shifts its changed occurrences by the same amount, which a new repeat rule can leave off its slots.
				at := r.ReplacesAt.Add(shift)
				r.ReplacesAt = &at
				r.UpdatedAt = now
				saved = append(saved, r)
			}
		}
		replaced = slotSet(rs)
	}
	m.planReminder(now, replaced)
	if err := s.repo.Save(ctx, saved, evts...); err != nil {
		return nil, fmt.Errorf("update meeting %s: %w", m.ID, err)
	}
	return m, nil
}

// changeOne changes one occurrence of the series m alone, as a meeting of its own standing in for slot.
func (s *Service) changeOne(ctx context.Context, m *Meeting, slot time.Time, replacements []*Meeting, c Change) (*Meeting, error) {
	if c.Repeats != nil && *c.Repeats != RepeatNone {
		return nil, fmt.Errorf("%w: one occurrence can't repeat; change this and the following ones instead", apperrs.ErrInvalid)
	}
	if c.RepeatUntil != nil {
		return nil, fmt.Errorf("%w: one occurrence has no repeat end; change this and the following ones instead", apperrs.ErrInvalid)
	}
	now := s.now().UTC()
	one := m.occurrenceCopy(slot, now)
	one.SeriesID, one.ReplacesAt = m.ID, &slot
	before := one.clone()
	c.apply(one)
	applyCancel(one, c.Cancelled, now)
	if err := s.check(ctx, one); err != nil {
		return nil, err
	}
	one.planReminder(now, nil)
	replaced := slotSet(replacements)
	replaced[slot.Unix()] = true
	m.planReminder(now, replaced)

	topic := TopicMeetingUpdated
	if one.Cancelled() {
		topic = TopicMeetingCancelled
	}
	ev := MeetingEvent{
		Meeting: refOf(one), ActorID: actorID(ctx), UserIDs: union(m.Audience(), one.Audience()),
		InvitedIDs: newlyInvited(before, one),
	}
	if err := s.repo.Save(ctx, []*Meeting{m, one}, outboxEvent(topic, ev)); err != nil {
		return nil, fmt.Errorf("change one occurrence of meeting %s: %w", m.ID, err)
	}
	return one, nil
}

// changeFollowing ends the series m before slot and carries on from there as a new series with the change.
func (s *Service) changeFollowing(ctx context.Context, m *Meeting, slot time.Time, replacements []*Meeting, c Change) (*Meeting, error) {
	now := s.now().UTC()
	before := m.clone()
	rest := m.occurrenceCopy(slot, now)
	rest.Repeats, rest.RepeatUntil = m.Repeats, m.RepeatUntil
	restBefore := rest.clone()
	c.apply(rest)
	applyCancel(rest, c.Cancelled, now)
	if err := s.check(ctx, rest); err != nil {
		return nil, err
	}
	until := slot.Add(-time.Second)
	m.RepeatUntil, m.UpdatedAt = &until, now

	saved := []*Meeting{m, rest}
	shift := rest.StartsAt.Sub(slot)
	kept, moved := map[int64]bool{}, map[int64]bool{}
	for _, r := range replacements {
		if r.ReplacesAt == nil {
			continue
		}
		if r.ReplacesAt.Before(slot) {
			kept[r.ReplacesAt.Unix()] = true
			continue
		}
		at := r.ReplacesAt.Add(shift)
		r.SeriesID, r.ReplacesAt, r.UpdatedAt = rest.ID, &at, now
		moved[at.Unix()] = true
		saved = append(saved, r)
	}
	m.planReminder(now, kept)
	rest.planReminder(now, moved)

	topic := TopicMeetingCreated
	if rest.Cancelled() {
		topic = TopicMeetingCancelled
	}
	restEv := MeetingEvent{
		Meeting: refOf(rest), ActorID: actorID(ctx), UserIDs: union(m.Audience(), rest.Audience()),
		InvitedIDs: newlyInvited(restBefore, rest),
	}
	evts := []eventbus.OutboxEvent{changeEvent(ctx, before, m), outboxEvent(topic, restEv)}
	if err := s.repo.Save(ctx, saved, evts...); err != nil {
		return nil, fmt.Errorf("change meeting %s from %s on: %w", m.ID, slot.Format(time.RFC3339), err)
	}
	return rest, nil
}

// Reply records the caller's answer to a meeting they are invited to.
func (s *Service) Reply(ctx context.Context, id string, reply Reply) (*Meeting, error) {
	if err := ValidReply(reply); err != nil {
		return nil, err
	}
	m, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	actor := actorID(ctx)
	if !m.Invited(actor) {
		return nil, fmt.Errorf("%w: only someone invited to meeting %s replies to it", apperrs.ErrForbidden, m.ID)
	}
	now := s.now().UTC()
	ev := ReplyEvent{MeetingID: m.ID, WorkspaceID: m.WorkspaceID, ActorID: actor, Reply: string(reply), UserIDs: m.Audience()}
	if err := s.repo.SaveReply(ctx, m.ID, actor, reply, now, outboxEvent(TopicMeetingReplied, ev)); err != nil {
		return nil, fmt.Errorf("reply to meeting %s: %w", m.ID, err)
	}
	for i := range m.Invitees {
		if m.Invitees[i].UserID == actor {
			m.Invitees[i].Reply, m.Invitees[i].RepliedAt = reply, &now
		}
	}
	return m, nil
}

// check normalizes m and validates it: its fields, that every invitee belongs to its workspace, and that its voice
// channel is one the organiser sees there, holding every invitee when it is private.
func (s *Service) check(ctx context.Context, m *Meeting) error {
	m.normalize()
	if err := m.validate(); err != nil {
		return err
	}
	if len(m.Invitees) > 0 {
		members, err := s.members.ListMemberUserIDs(ctx, m.WorkspaceID)
		if err != nil {
			return fmt.Errorf("list members of workspace %s: %w", m.WorkspaceID, err)
		}
		for _, id := range m.InviteeIDs() {
			if !slices.Contains(members, id) {
				return fmt.Errorf("%w: invitee %s is not a member of this workspace; account_list lists members", apperrs.ErrInvalid, id)
			}
		}
	}
	if m.VoiceChannelID == "" {
		return nil
	}
	return s.checkChannel(ctx, m)
}

func (s *Service) checkChannel(ctx context.Context, m *Meeting) error {
	ch, err := s.channels.Channel(ctx, m.VoiceChannelID)
	if errors.Is(err, apperrs.ErrNotFound) || permissions.Refused(err) || (err == nil && (!ch.Voice || ch.WorkspaceID != m.WorkspaceID)) {
		return fmt.Errorf("%w: voice_channel_id is not a voice channel you can see in this workspace; conversation_list lists them", apperrs.ErrInvalid)
	}
	if err != nil {
		return fmt.Errorf("read voice channel %s: %w", m.VoiceChannelID, err)
	}
	if !ch.Private {
		return nil
	}
	for _, id := range m.Audience() {
		if !slices.Contains(ch.MemberIDs, id) {
			return fmt.Errorf("%w: #%s is private and not everyone invited is in it; pick another channel or add them to it", apperrs.ErrInvalid, ch.Name)
		}
	}
	return nil
}

// clone copies m deep enough that changing the copy's invitees leaves m's alone.
func (m *Meeting) clone() *Meeting {
	c := *m
	c.Invitees = slices.Clone(m.Invitees)
	return &c
}

// occurrenceCopy is m's occurrence at slot as a meeting of its own, not repeating, with m's invitees and replies.
func (m *Meeting) occurrenceCopy(slot, now time.Time) *Meeting {
	c := m.clone()
	c.ID = ids.New()
	c.StartsAt, c.EndsAt = slot, slot.Add(m.EndsAt.Sub(m.StartsAt))
	c.Repeats, c.RepeatUntil, c.SeriesID, c.ReplacesAt = RepeatNone, nil, "", nil
	c.CreatedAt, c.UpdatedAt = now, now
	return c
}

// applyCancel cancels or restores m; nil leaves it.
func applyCancel(m *Meeting, cancelled *bool, now time.Time) {
	if cancelled == nil {
		return
	}
	if !*cancelled {
		m.CancelledAt = nil
		return
	}
	if m.CancelledAt == nil {
		m.CancelledAt = &now
	}
}

// changeEvent is the event for a meeting going from before to after: cancelled, restored, or changed.
func changeEvent(ctx context.Context, before, after *Meeting) eventbus.OutboxEvent {
	topic := TopicMeetingUpdated
	if !before.Cancelled() && after.Cancelled() {
		topic = TopicMeetingCancelled
	}
	if before.Cancelled() && !after.Cancelled() {
		topic = TopicMeetingRestored
	}
	return outboxEvent(topic, MeetingEvent{
		Meeting: refOf(after), ActorID: actorID(ctx), UserIDs: union(before.Audience(), after.Audience()),
		InvitedIDs: newlyInvited(before, after),
	})
}

func newlyInvited(before, after *Meeting) []string {
	var out []string
	for _, id := range after.InviteeIDs() {
		if !before.Invited(id) {
			out = append(out, id)
		}
	}
	return out
}

func union(a, b []string) []string {
	out := slices.Clone(a)
	for _, id := range b {
		if !slices.Contains(out, id) {
			out = append(out, id)
		}
	}
	return out
}

func slotSet(ms []*Meeting) map[int64]bool {
	out := map[int64]bool{}
	for _, r := range ms {
		if r.ReplacesAt != nil {
			out[r.ReplacesAt.Unix()] = true
		}
	}
	return out
}
