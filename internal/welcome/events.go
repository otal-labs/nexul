package welcome

import (
	"context"
	"encoding/json"
	"fmt"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/welcomestep"
)

// TopicProgressChanged is derived: something one person's progress reads may have changed.
const TopicProgressChanged = "welcome.progress_changed"

func Topics() []eventbus.Topic {
	return []eventbus.Topic{{Name: TopicProgressChanged, Payload: ProgressChangedEvent{}}}
}

// ProgressChangedEvent names only the person, so a client refreshes that one person's progress.
type ProgressChangedEvent struct {
	UserID string `json:"user_id"`
}

func changed(userIDs ...string) []eventbus.OutboxEvent {
	out := make([]eventbus.OutboxEvent, 0, len(userIDs))
	for _, id := range userIDs {
		out = append(out, eventbus.OutboxEvent{ID: ids.New(), Topic: TopicProgressChanged, Payload: ProgressChangedEvent{UserID: id}})
	}
	return out
}

// The fields welcome reads from other domains' payloads; their contracts only grow, so a field read here stays.
type sourcePayload struct {
	UserID            string   `json:"user_id"`
	Client            string   `json:"client"`
	EnrolledBy        string   `json:"enrolled_by"`
	RoleID            string   `json:"role_id"`
	RoleIDsAdded      []string `json:"role_ids_added"`
	WelcomeStepsAdded []string `json:"welcome_steps_added"`
	ResourceType      string   `json:"resource_type"`
	ResourceID        string   `json:"resource_id"`
	Target            string   `json:"target"`
}

func parse(ev eventbus.Event) (sourcePayload, error) {
	var p sourcePayload
	if err := json.Unmarshal(ev.Payload, &p); err != nil {
		return p, apperrs.Fatal(fmt.Errorf("parse %s: %w", ev.Topic, err))
	}
	return p, nil
}

func retryable(err error) error {
	if err == nil {
		return nil
	}
	return apperrs.Retryable(err)
}

// HandlePersonChanged announces the person a computer, GitHub link or membership event names.
func (s *Service) HandlePersonChanged(ctx context.Context, ev eventbus.Event) error {
	p, err := parse(ev)
	if err != nil || p.UserID == "" {
		return err
	}
	return retryable(s.repo.Announce(ctx, changed(p.UserID)...))
}

// HandleRunnerChanged announces whoever enrolled a deploy runner that connected or went away.
func (s *Service) HandleRunnerChanged(ctx context.Context, ev eventbus.Event) error {
	p, err := parse(ev)
	if err != nil || p.EnrolledBy == "" {
		return err
	}
	return retryable(s.repo.Announce(ctx, changed(p.EnrolledBy)...))
}

// HandleSessionCreated records a person's first phone sign-in, which keeps the phone step done for good.
func (s *Service) HandleSessionCreated(ctx context.Context, ev eventbus.Event) error {
	p, err := parse(ev)
	if err != nil || p.Client != "phone" || p.UserID == "" {
		return err
	}
	return retryable(s.repo.MarkDone(ctx, p.UserID, welcomestep.Phone, s.now().UTC(), changed(p.UserID)...))
}

// HandleMemberChanged gives a person the steps of the roles they were just given, and announces them either way,
// since a role taken away takes its steps too.
func (s *Service) HandleMemberChanged(ctx context.Context, ev eventbus.Event) error {
	p, err := parse(ev)
	if err != nil || p.UserID == "" {
		return err
	}
	if len(p.RoleIDsAdded) == 0 {
		return retryable(s.repo.Announce(ctx, changed(p.UserID)...))
	}
	steps, err := s.seams.RoleSteps(ctx, p.RoleIDsAdded)
	if err != nil {
		return retryable(err)
	}
	return retryable(s.give(ctx, []string{p.UserID}, steps))
}

// HandleRoleUpdated gives the steps a role's save ticked to everyone holding it, and announces each holder, since a
// tick taken off takes the step too.
func (s *Service) HandleRoleUpdated(ctx context.Context, ev eventbus.Event) error {
	p, err := parse(ev)
	if err != nil || p.RoleID == "" {
		return err
	}
	holders, err := s.seams.RoleHolders(ctx, p.RoleID)
	if err != nil {
		return retryable(err)
	}
	return retryable(s.give(ctx, holders, p.WelcomeStepsAdded))
}

// HandleGrantChanged announces the people a project's rule moves in or out of it: the person it names, a role's
// holders, or for a save of the Access page every member of the workspace.
func (s *Service) HandleGrantChanged(ctx context.Context, ev eventbus.Event) error {
	p, err := parse(ev)
	if err != nil {
		return err
	}
	var people []string
	switch {
	case p.ResourceType == "workspace":
		people, err = s.seams.WorkspaceMembers(ctx, p.ResourceID)
	case p.ResourceType != "project":
		return nil
	case p.Target == "role":
		people, err = s.seams.RoleHolders(ctx, p.RoleID)
	default:
		people = []string{p.UserID}
	}
	if err != nil {
		return retryable(err)
	}
	return retryable(s.repo.Announce(ctx, changed(people...)...))
}
