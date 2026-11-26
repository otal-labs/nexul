package sso

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"slices"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// maxMovePairs caps one Link matched; a page of people is far below it.
const maxMovePairs = 200

var errOwnerAccount = fmt.Errorf("%w: only an instance Owner moves a sign-in onto or off an instance Owner's account, or expects one for it", apperrs.ErrForbidden)

// MoveSignIn joins the two accounts a person ends up with when their first sign-in through the provider found no
// account to match: the single sign-on moves from fromID, the account it made, onto toID, and fromID is disabled and
// signed out with everything it made kept. It needs sign_ins:move and a key that unlocks sso.
func (s *Service) MoveSignIn(ctx context.Context, fromID, toID string) error {
	if err := s.requireMoves(ctx); err != nil {
		return err
	}
	guard, err := s.ownerGuard(ctx)
	if err != nil {
		return err
	}
	return s.moveSignIn(ctx, guard, MovePair{FromAccountID: fromID, ToAccountID: toID})
}

// LinkMatched moves each pair's sign-in as MoveSignIn does, each in its own transaction, and says per pair whether it
// moved; one that fails leaves the others moved.
func (s *Service) LinkMatched(ctx context.Context, pairs []MovePair) ([]MoveResult, error) {
	if err := s.requireMoves(ctx); err != nil {
		return nil, err
	}
	if len(pairs) == 0 || len(pairs) > maxMovePairs {
		return nil, fmt.Errorf("%w: send 1 to %d pairs", apperrs.ErrInvalid, maxMovePairs)
	}
	guard, err := s.ownerGuard(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]MoveResult, 0, len(pairs))
	for _, p := range pairs {
		out = append(out, MoveResult{MovePair: p, Err: s.moveSignIn(ctx, guard, p)})
	}
	return out, nil
}

func (s *Service) requireMoves(ctx context.Context) error {
	if err := s.require(ctx, permissions.SignInsMove); err != nil {
		return err
	}
	if !s.unlocked(ctx) {
		return errNeedsLicense
	}
	return nil
}

func (s *Service) moveSignIn(ctx context.Context, guard func(...string) error, p MovePair) error {
	fromID, toID := strings.TrimSpace(p.FromAccountID), strings.TrimSpace(p.ToAccountID)
	if fromID == "" || toID == "" {
		return fmt.Errorf("%w: both accounts are needed", apperrs.ErrInvalid)
	}
	if fromID == toID {
		return fmt.Errorf("%w: single sign-on moves between two different accounts", apperrs.ErrInvalid)
	}
	if err := guard(fromID, toID); err != nil {
		return err
	}
	return s.cfg.Accounts.MoveSignIn(ctx, fromID, toID, actorID(ctx))
}

// ownerGuard returns the check that keeps a holder who is not an instance Owner off every instance Owner's account, since
// moving a sign-in onto one, off one, or expecting one for it would hand them that Owner's way in. The server's own
// calls pass.
func (s *Service) ownerGuard(ctx context.Context) (func(...string) error, error) {
	actor := actorID(ctx)
	if actor == "" {
		return func(...string) error { return nil }, nil
	}
	owners, err := s.cfg.InstanceOwners(ctx)
	if err != nil {
		return nil, fmt.Errorf("list instance owners: %w", err)
	}
	if slices.Contains(owners, actor) {
		return func(...string) error { return nil }, nil
	}
	return func(accountIDs ...string) error {
		for _, id := range accountIDs {
			if slices.Contains(owners, id) {
				return errOwnerAccount
			}
		}
		return nil
	}, nil
}

// SignInMoves reads who still signs in through the other protocol than the saved provider's, each with the account a
// first sign-in through the saved one made under the same email, the accounts whose one sign-in is the saved
// provider's and match nobody, and every expected sign-in waiting. It needs sign_ins:move.
func (s *Service) SignInMoves(ctx context.Context) (SignInMoves, error) {
	if err := s.require(ctx, permissions.SignInsMove); err != nil {
		return SignInMoves{}, err
	}
	pub := s.Public(ctx)
	expected, err := s.cfg.Moves.ExpectedLinks(ctx)
	if err != nil {
		return SignInMoves{}, err
	}
	out := SignInMoves{Protocol: pub.Protocol, ProviderName: pub.Name, Expected: expected}
	if pub.Protocol == "" {
		return out, nil
	}
	old, err := s.cfg.Moves.OldSignIns(ctx, pub.Protocol)
	if err != nil {
		return SignInMoves{}, err
	}
	only, err := s.cfg.Moves.OnlyOn(ctx, pub.Protocol)
	if err != nil {
		return SignInMoves{}, err
	}
	out.People, out.Unmatched = matchOldSignIns(old, only, expected)
	return out, nil
}

// matchOldSignIns pairs each old sign-in with the first unclaimed account only the saved provider signs in whose email
// is the same, ignoring case.
func matchOldSignIns(old, only []MoveAccount, expected []ExpectedLink) ([]OldSignIn, []MoveAccount) {
	waiting := make(map[string]string, len(expected))
	for _, e := range expected {
		waiting[e.Account.ID] = e.Account.Email
	}
	claimed := map[string]bool{}
	people := make([]OldSignIn, 0, len(old))
	for _, o := range old {
		row := OldSignIn{MoveAccount: o, ExpectedEmail: waiting[o.ID]}
		for _, n := range only {
			if !claimed[n.ID] && strings.EqualFold(n.Email, o.Email) {
				claimed[n.ID] = true
				match := n
				row.Match = &match
				break
			}
		}
		people = append(people, row)
	}
	unmatched := make([]MoveAccount, 0, len(only))
	for _, n := range only {
		if !claimed[n.ID] {
			unmatched = append(unmatched, n)
		}
	}
	return people, unmatched
}

// ExpectSignIn saves an expected sign-in: the first sign-in through the saved provider that brings email lands on the
// account instead of matching by email or making one. It replaces the account's earlier one, and needs sign_ins:move,
// a key that unlocks sso and a saved provider the account does not sign in through yet.
func (s *Service) ExpectSignIn(ctx context.Context, accountID, email string) (ExpectedLink, error) {
	if err := s.requireMoves(ctx); err != nil {
		return ExpectedLink{}, err
	}
	accountID, email = strings.TrimSpace(accountID), strings.TrimSpace(email)
	if accountID == "" {
		return ExpectedLink{}, fmt.Errorf("%w: the account is needed", apperrs.ErrInvalid)
	}
	if addr, err := mail.ParseAddress(email); err != nil || addr.Address != email {
		return ExpectedLink{}, fmt.Errorf("%w: email must be an address such as alice@example.com", apperrs.ErrInvalid)
	}
	pub := s.Public(ctx)
	if pub.Protocol == "" {
		return ExpectedLink{}, fmt.Errorf("%w: set up a single sign-on provider first", apperrs.ErrInvalid)
	}
	guard, err := s.ownerGuard(ctx)
	if err != nil {
		return ExpectedLink{}, err
	}
	if err := guard(accountID); err != nil {
		return ExpectedLink{}, err
	}
	link := ExpectedLink{Account: MoveAccount{ID: accountID, Email: email}, CreatedBy: actorID(ctx), CreatedAt: s.cfg.Now().UTC()}
	evt := s.expectedChanged(ExpectedChangedEvent{AccountID: accountID, Email: email, ChangedBy: link.CreatedBy})
	if err := s.cfg.Moves.Expect(ctx, link, pub.Protocol, evt); err != nil {
		return ExpectedLink{}, err
	}
	return link, nil
}

// RemoveExpected deletes the account's expected sign-in, so its first sign-in matches as any other does. It needs
// sign_ins:move and works without a key, so a lapsed license never traps one in place.
func (s *Service) RemoveExpected(ctx context.Context, accountID string) error {
	if err := s.require(ctx, permissions.SignInsMove); err != nil {
		return err
	}
	accountID = strings.TrimSpace(accountID)
	links, err := s.cfg.Moves.ExpectedLinks(ctx)
	if err != nil {
		return err
	}
	i := slices.IndexFunc(links, func(l ExpectedLink) bool { return l.Account.ID == accountID })
	if i < 0 {
		return fmt.Errorf("%w: account %s expects no sign-in", apperrs.ErrNotFound, accountID)
	}
	evt := s.expectedChanged(ExpectedChangedEvent{AccountID: accountID, Email: links[i].Account.Email, Removed: true, ChangedBy: actorID(ctx)})
	return s.cfg.Moves.Unexpect(ctx, accountID, evt)
}

// expected is the account an expected sign-in names for the identity's email, linked to it and the link used; found
// is false when none waits for that email.
func (s *Service) expected(ctx context.Context, id Identity) (userID string, found bool, err error) {
	acct, err := s.cfg.Moves.ByExpected(ctx, id.Email)
	if errors.Is(err, apperrs.ErrNotFound) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	userID, err = admit(acct)
	if err != nil {
		return "", true, err
	}
	evt := s.expectedChanged(ExpectedChangedEvent{AccountID: userID, Email: id.Email, Used: true, ChangedBy: userID})
	if err := s.cfg.Moves.LinkExpected(ctx, userID, id, evt); err != nil {
		return "", true, err
	}
	return userID, true, nil
}

func (s *Service) expectedChanged(e ExpectedChangedEvent) eventbus.OutboxEvent {
	e.ChangedAt = s.cfg.Now().UTC()
	return eventbus.OutboxEvent{ID: ids.New(), Topic: TopicExpectedChanged, Payload: e}
}
