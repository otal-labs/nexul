package access

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// resourceTypeComputer keys a computer's rules by resource_id = the computer id; they name people only (ADR 0148).
const resourceTypeComputer = "computer"

// SourceNotShared is a computer no rule shares with the person asked about.
const SourceNotShared = "not_shared"

// Computers reads a person's computers and the rules their owner wrote on them, past every memo (ADR 0148).
type Computers interface {
	// ComputerAccess is computerID's owner and the live rule naming userID; ErrNotFound unless userID owns it or one does.
	ComputerAccess(ctx context.Context, computerID, userID string) (ownerID string, allow permissions.Set, err error)
	// SharesWithOwner reports whether personID is an active account sharing a workspace with ownerID.
	SharesWithOwner(ctx context.Context, ownerID, personID string) (bool, error)
	// EndComputerRules deletes every computer rule no longer live, with event's row for each, in one transaction.
	EndComputerRules(ctx context.Context, event func(computerID, ownerID, userID string) eventbus.OutboxEvent) error
}

// SetComputers wires the storage reads a computer check makes.
func (s *Service) SetComputers(c Computers) {
	s.computers = c
}

// ComputerPermissions reads fresh, never the memo, so a revoke holds on the next check; no role or Owner reaches it.
func (s *Service) ComputerPermissions(ctx context.Context, userID, computerID string) (ownerID string, held permissions.Set, err error) {
	if strings.TrimSpace(userID) == "" {
		return "", nil, fmt.Errorf("%w: user is required", apperrs.ErrUnauthorized)
	}
	notFound := fmt.Errorf("%w: computer %s", apperrs.ErrNotFound, computerID)
	if _, viewing := viewedWorkspace(ctx); viewing || s.computers == nil || strings.TrimSpace(computerID) == "" {
		return "", nil, notFound
	}
	ownerID, allow, err := s.computers.ComputerAccess(ctx, computerID, userID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return "", nil, notFound
	}
	if err != nil {
		return "", nil, err
	}
	if ownerID == userID {
		return ownerID, permissions.SetOf(permissions.ComputerActions...), nil
	}
	held = seeWithRun(permissions.SetOf(slices.DeleteFunc(slices.Clone(allow), func(a permissions.Action) bool {
		return !slices.Contains(permissions.ComputerActions, a)
	})...))
	if !held.Has(permissions.ComputersSee) {
		return "", nil, notFound
	}
	return ownerID, held, nil
}

// seeWithRun adds See to a Run or Read logs, which can't be used on a computer one can't see.
func seeWithRun(allow permissions.Set) permissions.Set {
	if allow.Has(permissions.ComputersAgents) || allow.Has(permissions.ComputersCommands) || allow.Has(permissions.ComputersLogs) {
		return allow.With(permissions.ComputersSee)
	}
	return allow
}

// computerRule is a computer rule after a write moved actions to state: Run and Read logs keep See, and See going takes all three.
func computerRule(allow permissions.Set, actions []permissions.Action, state string) permissions.Set {
	if state != StateAllow && slices.Contains(actions, permissions.ComputersSee) {
		return allow.Without(permissions.ComputersAgents).Without(permissions.ComputersCommands).Without(permissions.ComputersLogs)
	}
	return seeWithRun(allow)
}

// checkComputerWrite refuses what a computer rule can't be: one naming a role, or a Deny (ADR 0148).
func checkComputerWrite(to Targets, state string) error {
	if len(to.RoleIDs) > 0 {
		return fmt.Errorf("%w: a computer is shared with named people, never a role", apperrs.ErrInvalid)
	}
	if state == StateDeny || state == stateLift {
		return fmt.Errorf("%w: a computer rule is allow or fallback, never deny; fallback takes the person off it", apperrs.ErrInvalid)
	}
	return nil
}

// ownedComputer is not found to anyone but the owner, a person it is shared with included.
func (s *Service) ownedComputer(ctx context.Context, actorID, computerID string) error {
	ownerID, _, err := s.ComputerPermissions(ctx, actorID, computerID)
	if err != nil {
		return err
	}
	if ownerID != actorID {
		return fmt.Errorf("%w: computer %s", apperrs.ErrNotFound, computerID)
	}
	return nil
}

// checkComputerTarget lets anyone be taken off, but allows only active people who share a workspace with the owner.
func (s *Service) checkComputerTarget(ctx context.Context, ownerID string, w ruleWrite) error {
	if w.targetID == ownerID {
		return fmt.Errorf("%w: you own this computer; name the person to share it with", apperrs.ErrInvalid)
	}
	if len(w.allow) == 0 {
		return nil
	}
	shares, err := s.computers.SharesWithOwner(ctx, ownerID, w.targetID)
	if err != nil {
		return err
	}
	if !shares {
		return fmt.Errorf("%w: %s shares no workspace with you", apperrs.ErrInvalid, w.targetID)
	}
	return nil
}

// computerGrantEvent is the outbox row of a computer rule changing; userID now holds allow.
func computerGrantEvent(computerID, ownerID, userID, actorID string, allow permissions.Set) eventbus.OutboxEvent {
	held := make([]string, 0, len(allow))
	for _, a := range allow {
		held = append(held, string(a))
	}
	return eventbus.OutboxEvent{ID: ids.New(), Topic: TopicComputerGrantChanged, Payload: ComputerGrantEvent{
		ComputerID: computerID, OwnerID: ownerID, UserID: userID, Permissions: held, ActorID: actorID, MembersOnly: true,
	}}
}

// HandleMembershipEnded deletes ended computer rules so none comes back on a rejoin or reactivation; safe to repeat.
func (s *Service) HandleMembershipEnded(ctx context.Context, _ eventbus.Event) error {
	if s.computers == nil {
		return nil
	}
	return s.computers.EndComputerRules(ctx, func(computerID, ownerID, userID string) eventbus.OutboxEvent {
		return computerGrantEvent(computerID, ownerID, userID, "", nil)
	})
}

// explainComputer lets the owner explain anyone and a person it is shared with only themselves.
func (s *Service) explainComputer(ctx context.Context, actorID, computerID, userID string) (*Explanation, error) {
	ownerID, _, err := s.ComputerPermissions(ctx, actorID, computerID)
	if err != nil {
		return nil, err
	}
	if userID != actorID && actorID != ownerID {
		return nil, fmt.Errorf("%w: computer %s", apperrs.ErrNotFound, computerID)
	}
	held, err := s.computerHeld(ctx, ownerID, userID, computerID)
	if err != nil {
		return nil, err
	}
	n := s.namer(ctx)
	out := &Explanation{ResourceType: resourceTypeComputer, ResourceID: computerID, UserID: userID, Effective: make([]Answer, 0, len(permissions.ComputerActions))}
	for _, a := range permissions.ComputerActions {
		verdict := computerVerdict(n, computerID, ownerID, userID, held.Has(a))
		fallback := verdict
		if userID != ownerID {
			fallback = computerVerdict(n, computerID, ownerID, userID, false)
		}
		out.Effective = append(out.Effective, Answer{Action: a, Verdict: verdict, Fallback: fallback})
	}
	return out, nil
}

// computerHeld is what userID holds on a computer of ownerID's; a person no live rule names holds nothing.
func (s *Service) computerHeld(ctx context.Context, ownerID, userID, computerID string) (permissions.Set, error) {
	if userID == ownerID {
		return permissions.SetOf(permissions.ComputerActions...), nil
	}
	_, held, err := s.ComputerPermissions(ctx, userID, computerID)
	if errors.Is(err, apperrs.ErrNotFound) {
		return nil, nil
	}
	return held, err
}

// computerVerdict is "Owner of this computer", "Shared with bob by sam", or "Not shared with bob".
func computerVerdict(n *namer, computerID, ownerID, userID string, allowed bool) Verdict {
	person := n.name(TargetPerson, userID)
	src := NamedSource{
		Source:     Source{Kind: SourceNotShared, EntityType: resourceTypeComputer, EntityID: computerID, TargetType: TargetPerson, TargetID: userID, State: StateDeny},
		TargetName: person,
		Sentence:   "Not shared with " + person,
	}
	if userID == ownerID {
		src.Kind, src.State, src.Sentence = SourceOwner, StateAllow, "Owner of this computer"
	}
	if userID != ownerID && allowed {
		src.Kind, src.State, src.Sentence = SourceRule, StateAllow, fmt.Sprintf("Shared with %s by %s", person, n.name(TargetPerson, ownerID))
	}
	return Verdict{Allowed: allowed, Source: src}
}
