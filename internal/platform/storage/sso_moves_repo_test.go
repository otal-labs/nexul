package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/ee/sso"
	"github.com/otal-labs/nexul/internal/auth"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// seedMoves makes an instance moving from OIDC to SAML: alice (GitHub and OIDC) and bob (OIDC) are still on OIDC, dave
// signs in through both, eve is disabled, and the SAML provider's first sign-ins made u-dup (alice's email) and carol.
func seedMoves(t *testing.T, s *Store) {
	t.Helper()
	ctx := t.Context()
	seedDefaultRoles(t, s)
	for _, u := range []string{"alice", "dave", "eve"} {
		_, _, err := s.Users.UpsertUser(ctx, newTestUser("u-"+u, "gh-"+u, u))
		require.NoError(t, err)
	}
	oidc := func(user string) sso.Identity {
		return sso.Identity{Provider: sso.ProtocolOIDC, Key: "https://old.example.com#" + user, Email: user + "@example.com"}
	}
	saml := func(user, email string) sso.Identity {
		return sso.Identity{Provider: sso.ProtocolSAML, Key: "https://idp.example.com#" + user, Email: email}
	}
	require.NoError(t, s.SSO.Link(ctx, "u-alice", oidc("alice")))
	require.NoError(t, s.SSO.Create(ctx, "u-bob", oidc("bob"), "role-member", time.Unix(100, 0)))
	require.NoError(t, s.SSO.Link(ctx, "u-dave", oidc("dave")))
	require.NoError(t, s.SSO.Link(ctx, "u-dave", saml("dave", "dave@example.com")))
	require.NoError(t, s.SSO.Link(ctx, "u-eve", oidc("eve")))
	require.NoError(t, s.Users.SetAccountStatus(ctx, "u-eve", auth.AccountDisabled))
	require.NoError(t, s.SSO.Create(ctx, "u-dup", saml("dup", "Alice@example.com"), "role-member", time.Unix(200, 0)))
	require.NoError(t, s.SSO.Create(ctx, "u-carol", saml("carol", "carol@example.com"), "role-member", time.Unix(300, 0)))
}

func moveIDs(accounts []sso.MoveAccount) []string {
	out := []string{}
	for _, a := range accounts {
		out = append(out, a.ID)
	}
	return out
}

func TestSSORepo_OldSignInsAndOnlyOn_ListWhoMovesAndWhatTheNewProviderMade(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	seedMoves(t, s)

	old, err := s.SSO.OldSignIns(ctx, sso.ProtocolSAML)
	require.NoError(t, err)
	assert.Equal(t, []string{"u-alice", "u-bob"}, moveIDs(old), "dave signs in through SAML already and eve is disabled")
	assert.Equal(t, "alice@example.com", old[0].Email)
	assert.Equal(t, "Name alice", old[0].Name)

	only, err := s.SSO.OnlyOn(ctx, sso.ProtocolSAML)
	require.NoError(t, err)
	assert.Equal(t, []string{"u-dup", "u-carol"}, moveIDs(only), "dave has another sign-in too")
	assert.Equal(t, time.Unix(200, 0).UTC(), only[0].CreatedAt)

	old, err = s.SSO.OldSignIns(ctx, sso.ProtocolOIDC)
	require.NoError(t, err)
	assert.Equal(t, []string{"u-dup", "u-carol"}, moveIDs(old), "the other way round, the SAML-only accounts are the old ones")
}

func TestSSORepo_Expect_SavesReplacesAndRefuses(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	seedMoves(t, s)
	_, _, err := s.Users.UpsertUser(ctx, newTestUser("u-gone", "gh-gone", "gone"))
	require.NoError(t, err)
	require.NoError(t, s.Users.SetAccountStatus(ctx, "u-gone", auth.AccountRemoved))
	link := func(user, email string) sso.ExpectedLink {
		return sso.ExpectedLink{Account: sso.MoveAccount{ID: user, Email: email}, CreatedBy: "u-owner", CreatedAt: time.Unix(400, 0)}
	}
	evt := eventbus.OutboxEvent{ID: "evt-1", Topic: sso.TopicExpectedChanged, Payload: sso.ExpectedChangedEvent{AccountID: "u-bob"}}

	require.NoError(t, s.SSO.Expect(ctx, link("u-bob", "robert@example.com"), sso.ProtocolSAML, evt))
	require.NoError(t, s.SSO.Expect(ctx, link("u-bob", "bob@example.com"), sso.ProtocolSAML), "a second one replaces the first")

	for name, tc := range map[string]struct {
		link sso.ExpectedLink
		want error
	}{
		"an email another account expects": {link("u-alice", "BOB@example.com"), apperrs.ErrConflict},
		"an account the provider signs in": {link("u-dave", "dave2@example.com"), apperrs.ErrConflict},
		"a removed account":                {link("u-gone", "gone@example.com"), apperrs.ErrInvalid},
		"an unknown account":               {link("u-ghost", "ghost@example.com"), apperrs.ErrNotFound},
	} {
		require.ErrorIs(t, s.SSO.Expect(ctx, tc.link, sso.ProtocolSAML), tc.want, name)
	}
	links, err := s.SSO.ExpectedLinks(ctx)
	require.NoError(t, err)
	require.Len(t, links, 1)
	assert.Equal(t, sso.MoveAccount{ID: "u-bob", Login: "bob@example.com", Name: "", Email: "bob@example.com"}, links[0].Account)
	assert.Equal(t, "u-owner", links[0].CreatedBy)
	acct, err := s.SSO.ByExpected(ctx, "Bob@Example.com")
	require.NoError(t, err)
	assert.Equal(t, sso.Account{UserID: "u-bob", Status: "active"}, acct, "an email matches ignoring case")
	_, err = s.SSO.ByExpected(ctx, "robert@example.com")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	assert.Contains(t, outboxTopics(t, s), sso.TopicExpectedChanged)

	require.NoError(t, s.SSO.Unexpect(ctx, "u-bob"))
	require.ErrorIs(t, s.SSO.Unexpect(ctx, "u-bob"), apperrs.ErrNotFound)
	links, err = s.SSO.ExpectedLinks(ctx)
	require.NoError(t, err)
	assert.Empty(t, links)
}

func TestSSORepo_LinkExpected_LinksTheSubjectAndUsesItUp(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	seedMoves(t, s)
	require.NoError(t, s.SSO.Expect(ctx, sso.ExpectedLink{Account: sso.MoveAccount{ID: "u-bob", Email: "bob@example.com"}, CreatedAt: time.Unix(400, 0)}, sso.ProtocolSAML))
	before := outboxTopics(t, s)
	id := sso.Identity{Provider: sso.ProtocolSAML, Key: "https://idp.example.com#bob", Email: "bob@example.com"}
	used := eventbus.OutboxEvent{ID: "evt-used", Topic: sso.TopicExpectedChanged, Payload: sso.ExpectedChangedEvent{AccountID: "u-bob", Used: true}}

	require.NoError(t, s.SSO.LinkExpected(ctx, "u-bob", id, used))

	acct, err := s.SSO.BySubject(ctx, sso.ProtocolSAML, id.Key)
	require.NoError(t, err)
	assert.Equal(t, "u-bob", acct.UserID)
	_, err = s.SSO.ByExpected(ctx, "bob@example.com")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	assert.Equal(t, append(before, auth.TopicIdentityLinked, sso.TopicExpectedChanged), outboxTopics(t, s))
	require.Error(t, s.SSO.LinkExpected(ctx, "u-bob", id), "a subject links once")
}

func TestSSORepo_MoveSignIn_OntoAnAccountOnTheOtherProtocol_MovesAndDropsItsExpectedSignIn(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	seedMoves(t, s)
	require.NoError(t, s.SSO.Expect(ctx, sso.ExpectedLink{Account: sso.MoveAccount{ID: "u-alice", Email: "alice@example.com"}, CreatedAt: time.Unix(400, 0)}, sso.ProtocolSAML))
	before := outboxTopics(t, s)

	require.NoError(t, s.SSO.MoveSignIn(ctx, "u-dup", "u-alice", "u-owner"))

	acct, err := s.SSO.BySubject(ctx, sso.ProtocolSAML, "https://idp.example.com#dup")
	require.NoError(t, err)
	assert.Equal(t, "u-alice", acct.UserID)
	_, err = s.SSO.BySubject(ctx, sso.ProtocolOIDC, "https://old.example.com#alice")
	require.NoError(t, err, "the old sign-in stays")
	links, err := s.SSO.ExpectedLinks(ctx)
	require.NoError(t, err)
	assert.Empty(t, links)
	assert.Equal(t, append(before, auth.TopicIdentityLinked, auth.TopicAccountDisabled, sso.TopicExpectedChanged), outboxTopics(t, s))
	old, err := s.SSO.OldSignIns(ctx, sso.ProtocolSAML)
	require.NoError(t, err)
	assert.Equal(t, []string{"u-bob"}, moveIDs(old))
	require.ErrorIs(t, s.SSO.MoveSignIn(ctx, "u-carol", "u-dave", "u-owner"), apperrs.ErrConflict, "dave signs in through SAML already")
}
