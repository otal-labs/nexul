package sso

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/identity"
)

type fakeMoves struct {
	old, only []MoveAccount
	// links is each expected sign-in by account id; status is each account's status, active when unset.
	links  map[string]ExpectedLink
	status map[string]string
	// current is the protocol the last Expect was told the provider speaks.
	current string
	used    map[string]Identity
	events  []eventbus.OutboxEvent
	err     error
}

func newMoves() *fakeMoves {
	return &fakeMoves{links: map[string]ExpectedLink{}, status: map[string]string{}, used: map[string]Identity{}}
}

func (m *fakeMoves) OldSignIns(context.Context, string) ([]MoveAccount, error) { return m.old, m.err }
func (m *fakeMoves) OnlyOn(context.Context, string) ([]MoveAccount, error)     { return m.only, m.err }
func (m *fakeMoves) ExpectedLinks(context.Context) ([]ExpectedLink, error) {
	out := []ExpectedLink{}
	for _, l := range m.links {
		out = append(out, l)
	}
	return out, m.err
}
func (m *fakeMoves) Expect(_ context.Context, link ExpectedLink, current string, evts ...eventbus.OutboxEvent) error {
	m.links[link.Account.ID], m.current = link, current
	m.events = append(m.events, evts...)
	return nil
}
func (m *fakeMoves) Unexpect(_ context.Context, userID string, evts ...eventbus.OutboxEvent) error {
	delete(m.links, userID)
	m.events = append(m.events, evts...)
	return nil
}
func (m *fakeMoves) ByExpected(_ context.Context, email string) (Account, error) {
	for id, l := range m.links {
		if l.Account.Email == email {
			return Account{UserID: id, Status: m.status[id]}, nil
		}
	}
	return Account{}, apperrs.ErrNotFound
}
func (m *fakeMoves) LinkExpected(_ context.Context, userID string, id Identity, evts ...eventbus.OutboxEvent) error {
	m.used[userID] = id
	delete(m.links, userID)
	m.events = append(m.events, evts...)
	return nil
}

func asHolder(t *testing.T) context.Context {
	t.Helper()
	return identity.WithActor(t.Context(), identity.Actor{ID: "u-helper"})
}

func TestMoveSignIn_AHolderWhoIsNotAnInstanceOwner_NeverTouchesAnOwnersAccount(t *testing.T) {
	t.Parallel()
	for name, pair := range map[string]MovePair{
		"onto an Owner's account": {FromAccountID: "u-dup", ToAccountID: "owner-1"},
		"off an Owner's account":  {FromAccountID: "owner-1", ToAccountID: "u-alice"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			h := newHarness(t, true)

			err := h.svc.MoveSignIn(asHolder(t), pair.FromAccountID, pair.ToAccountID)

			require.ErrorIs(t, err, apperrs.ErrForbidden)
			assert.Empty(t, h.accounts.moved)
			results, err := h.svc.LinkMatched(asHolder(t), []MovePair{pair})
			require.NoError(t, err)
			require.ErrorIs(t, results[0].Err, apperrs.ErrForbidden)
			assert.Empty(t, h.accounts.moved)
		})
	}
	h := newHarness(t, true)
	require.NoError(t, h.svc.MoveSignIn(asActor(t), "u-dup", "owner-1"), "an instance Owner may")
	require.NoError(t, h.svc.MoveSignIn(asHolder(t), "u-dup", "u-alice"), "a holder may on anyone else's")
	assert.Equal(t, []string{"u-dup>owner-1", "u-dup>u-alice"}, h.accounts.moved)
}

func TestExpectSignIn_AHolderWhoIsNotAnInstanceOwner_CannotExpectOneForAnOwner(t *testing.T) {
	t.Parallel()
	h := newHarness(t, true)

	_, err := h.svc.ExpectSignIn(asHolder(t), "owner-1", "owner@example.com")

	require.ErrorIs(t, err, apperrs.ErrForbidden)
	assert.Empty(t, h.moves.links)
	_, err = h.svc.ExpectSignIn(asActor(t), "owner-1", "owner@example.com")
	require.NoError(t, err, "an instance Owner may")
}

func TestLinkMatched_MovesEachPairAndSaysWhichDidNot(t *testing.T) {
	t.Parallel()
	h := newHarness(t, true)

	results, err := h.svc.LinkMatched(asHolder(t), []MovePair{
		{FromAccountID: "u-dup-a", ToAccountID: "u-alice"},
		{FromAccountID: "u-dup-o", ToAccountID: "owner-1"},
		{FromAccountID: "u-bob", ToAccountID: "u-bob"},
		{FromAccountID: "u-dup-c", ToAccountID: "u-carol"},
	})

	require.NoError(t, err)
	require.Len(t, results, 4)
	require.NoError(t, results[0].Err)
	require.ErrorIs(t, results[1].Err, apperrs.ErrForbidden)
	require.ErrorIs(t, results[2].Err, apperrs.ErrInvalid)
	require.NoError(t, results[3].Err)
	assert.Equal(t, []string{"u-dup-a>u-alice", "u-dup-c>u-carol"}, h.accounts.moved)
}

func TestLinkMatched_RefusesBeforeMovingAnything(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		licensed bool
		gate     error
		pairs    []MovePair
		want     error
	}{
		{"without sign_ins:move", true, apperrs.ErrForbidden, []MovePair{{FromAccountID: "a", ToAccountID: "b"}}, apperrs.ErrForbidden},
		{"without a key that unlocks sso", false, nil, []MovePair{{FromAccountID: "a", ToAccountID: "b"}}, apperrs.ErrForbidden},
		{"with no pairs", true, nil, nil, apperrs.ErrInvalid},
		{"with too many pairs", true, nil, make([]MovePair, maxMovePairs+1), apperrs.ErrInvalid},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := newHarness(t, tc.licensed)
			h.svc.cfg.Gate = fakeGate{err: tc.gate}

			_, err := h.svc.LinkMatched(asActor(t), tc.pairs)

			require.ErrorIs(t, err, tc.want)
			assert.Empty(t, h.accounts.moved)
		})
	}
	h := newHarness(t, true)
	h.svc.cfg.InstanceOwners = func(context.Context) ([]string, error) { return nil, errors.New("db down") }
	_, err := h.svc.LinkMatched(asActor(t), []MovePair{{FromAccountID: "a", ToAccountID: "b"}})
	require.Error(t, err)
	assert.Empty(t, h.accounts.moved)
}

func TestSignInMoves_PairsEachOldSignInWithTheAccountItsEmailMade(t *testing.T) {
	t.Parallel()
	h := newHarness(t, true)
	h.moves.old = []MoveAccount{
		{ID: "u-alice", Login: "alice", Email: "Alice@example.com"},
		{ID: "u-bob", Login: "bob", Email: "bob@example.com"},
		{ID: "u-alice2", Login: "alice2", Email: "alice@example.com"},
	}
	h.moves.only = []MoveAccount{{ID: "u-dup", Email: "alice@example.com"}, {ID: "u-carol", Email: "carol@example.com"}}
	h.moves.links["u-bob"] = ExpectedLink{Account: MoveAccount{ID: "u-bob", Email: "robert@example.com"}}

	got, err := h.svc.SignInMoves(asActor(t))

	require.NoError(t, err)
	assert.Equal(t, ProtocolOIDC, got.Protocol)
	assert.Equal(t, "Acme SSO", got.ProviderName)
	require.Len(t, got.People, 3)
	assert.Equal(t, &MoveAccount{ID: "u-dup", Email: "alice@example.com"}, got.People[0].Match, "the email matches ignoring case")
	assert.Nil(t, got.People[1].Match)
	assert.Equal(t, "robert@example.com", got.People[1].ExpectedEmail)
	assert.Nil(t, got.People[2].Match, "one account a first sign-in made matches one person")
	assert.Equal(t, []MoveAccount{{ID: "u-carol", Email: "carol@example.com"}}, got.Unmatched)
	assert.Len(t, got.Expected, 1)
}

func TestSignInMoves_RefusedOrWithNoProvider(t *testing.T) {
	t.Parallel()
	h := newHarness(t, true)
	h.svc.cfg.Gate = fakeGate{err: apperrs.ErrForbidden}
	_, err := h.svc.SignInMoves(asActor(t))
	require.ErrorIs(t, err, apperrs.ErrForbidden)

	h = newHarness(t, true)
	h.repo.st = Settings{}
	h.moves.old = []MoveAccount{{ID: "u-alice"}}
	got, err := h.svc.SignInMoves(asActor(t))
	require.NoError(t, err)
	assert.Empty(t, got.Protocol)
	assert.Empty(t, got.People, "with no provider saved nobody is on the other one")

	h = newHarness(t, true)
	h.moves.err = errors.New("db down")
	_, err = h.svc.SignInMoves(asActor(t))
	require.Error(t, err)
}

func TestExpectSignIn_RefusesWhatCannotBeExpected(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, account, email string
		licensed, noProvider bool
		gate                 error
		want                 error
	}{
		{"without sign_ins:move", "u-bob", "bob@example.com", true, false, apperrs.ErrForbidden, apperrs.ErrForbidden},
		{"without a key that unlocks sso", "u-bob", "bob@example.com", false, false, nil, apperrs.ErrForbidden},
		{"with no account", " ", "bob@example.com", true, false, nil, apperrs.ErrInvalid},
		{"with no email", "u-bob", "", true, false, nil, apperrs.ErrInvalid},
		{"with a name around the email", "u-bob", "Bob <bob@example.com>", true, false, nil, apperrs.ErrInvalid},
		{"with no provider saved", "u-bob", "bob@example.com", true, true, nil, apperrs.ErrInvalid},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := newHarness(t, tc.licensed)
			h.svc.cfg.Gate = fakeGate{err: tc.gate}
			if tc.noProvider {
				h.repo.st = Settings{}
			}

			_, err := h.svc.ExpectSignIn(asActor(t), tc.account, tc.email)

			require.ErrorIs(t, err, tc.want)
			assert.Empty(t, h.moves.links)
		})
	}
}

func TestExpectSignIn_SavesItForTheSavedProtocolWithItsEvent(t *testing.T) {
	t.Parallel()
	h := newHarness(t, true)

	link, err := h.svc.ExpectSignIn(asActor(t), " u-bob ", " bob@example.com ")

	require.NoError(t, err)
	assert.Equal(t, MoveAccount{ID: "u-bob", Email: "bob@example.com"}, link.Account)
	assert.Equal(t, "owner-1", link.CreatedBy)
	assert.Equal(t, ProtocolOIDC, h.moves.current)
	require.Len(t, h.moves.events, 1)
	assert.Equal(t, TopicExpectedChanged, h.moves.events[0].Topic)
	assert.Equal(t, ExpectedChangedEvent{AccountID: "u-bob", Email: "bob@example.com", ChangedBy: "owner-1", ChangedAt: h.moves.events[0].Payload.(ExpectedChangedEvent).ChangedAt}, h.moves.events[0].Payload)
}

func TestRemoveExpected_DeletesItWithItsEvent(t *testing.T) {
	t.Parallel()
	h := newHarness(t, false)
	h.moves.links["u-bob"] = ExpectedLink{Account: MoveAccount{ID: "u-bob", Email: "bob@example.com"}}

	require.ErrorIs(t, h.svc.RemoveExpected(asActor(t), "u-carol"), apperrs.ErrNotFound)
	require.NoError(t, h.svc.RemoveExpected(asActor(t), "u-bob"), "a lapsed key still lets one go")

	assert.Empty(t, h.moves.links)
	require.Len(t, h.moves.events, 1)
	payload := h.moves.events[0].Payload.(ExpectedChangedEvent)
	assert.True(t, payload.Removed)
	assert.Equal(t, "bob@example.com", payload.Email)
	h.svc.cfg.Gate = fakeGate{err: apperrs.ErrForbidden}
	require.ErrorIs(t, h.svc.RemoveExpected(asActor(t), "u-bob"), apperrs.ErrForbidden)
}

func TestComplete_AnExpectedSignIn_BeatsTheEmailMatchAndIsUsed(t *testing.T) {
	t.Parallel()
	h := newHarness(t, true)
	h.accounts.byEmail["alice@example.com"] = Account{UserID: "u-google", Status: "active"}
	h.moves.links["u-old"] = ExpectedLink{Account: MoveAccount{ID: "u-old", Email: "alice@example.com"}}

	userID, err := h.svc.Complete(t.Context(), "state", "code")

	require.NoError(t, err)
	assert.Equal(t, "u-old", userID)
	assert.Equal(t, issuerKey, h.moves.used["u-old"].Key)
	assert.Empty(t, h.moves.links, "using it deletes it")
	assert.Empty(t, h.accounts.linked)
	assert.Empty(t, h.accounts.created)
	require.Len(t, h.moves.events, 1)
	assert.True(t, h.moves.events[0].Payload.(ExpectedChangedEvent).Used)
}

func TestComplete_AnExpectedSignInOntoADisabledAccount_IsRefusedAndKept(t *testing.T) {
	t.Parallel()
	h := newHarness(t, true)
	h.moves.links["u-old"] = ExpectedLink{Account: MoveAccount{ID: "u-old", Email: "alice@example.com"}}
	h.moves.status["u-old"] = "disabled"

	_, err := h.svc.Complete(t.Context(), "state", "code")

	require.ErrorIs(t, err, errDisabled)
	assert.Len(t, h.moves.links, 1)
	assert.Empty(t, h.accounts.created)
}

func TestComplete_WithALapsedKey_NoExpectedSignInIsUsed(t *testing.T) {
	t.Parallel()
	h := newHarness(t, false)
	h.moves.links["u-old"] = ExpectedLink{Account: MoveAccount{ID: "u-old", Email: "alice@example.com"}, CreatedAt: time.Unix(1, 0)}

	_, err := h.svc.Complete(t.Context(), "state", "code")

	require.ErrorIs(t, err, errLapsed)
	assert.Len(t, h.moves.links, 1)
}
