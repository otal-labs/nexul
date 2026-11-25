package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/ee/sso"
	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

func TestSSORepo_SAMLSettings_SealTheKeyAndRoundTrip(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	got, err := s.SSO.GetSAML(ctx)
	require.NoError(t, err)
	assert.Equal(t, sso.SAMLSettings{}, got, "a fresh instance has no SAML provider")

	want := sso.SAMLSettings{
		MetadataXML:    "<md:EntityDescriptor/>",
		IdP:            sso.IdPMetadata{EntityID: "https://idp.example.com", SSOURL: "https://idp.example.com/sso", Certificates: []string{"AAA", "BBB"}},
		EmailAttribute: "email", NameAttribute: "name", DisplayName: "Acme SSO", JoinRoleID: "role-member", Only: true,
		SPKey: "-----BEGIN PRIVATE KEY-----\nsecret\n-----END PRIVATE KEY-----\n", SPCertificate: "-----BEGIN CERTIFICATE-----",
		UpdatedBy: "u-1", UpdatedAt: time.Unix(1_000, 0).UTC(),
	}
	require.NoError(t, s.SSO.SetSAML(ctx, want, eventbus.OutboxEvent{ID: "e1", Topic: sso.TopicSAMLChanged, Payload: sso.SAMLChangedEvent{}}))

	got, err = s.SSO.GetSAML(ctx)
	require.NoError(t, err)
	assert.Equal(t, want, got)
	var raw string
	require.NoError(t, s.db.QueryRowContext(ctx, `SELECT sp_key FROM saml_settings`).Scan(&raw))
	assert.NotContains(t, raw, "secret", "the database holds the key sealed, so a copied file carries none")

	want.MetadataXML, want.MetadataURL = "", "https://idp.example.com/metadata"
	require.NoError(t, s.SSO.SetSAML(ctx, want))
	got, err = s.SSO.GetSAML(ctx)
	require.NoError(t, err)
	assert.Equal(t, want, got, "a second save replaces the one row")

	require.NoError(t, s.SSO.DeleteSAML(ctx, eventbus.OutboxEvent{ID: "e2", Topic: sso.TopicSAMLChanged, Payload: sso.SAMLChangedEvent{Removed: true}}))
	got, err = s.SSO.GetSAML(ctx)
	require.NoError(t, err)
	assert.Equal(t, sso.SAMLSettings{}, got)
	pending, err := s.Outbox.Unpublished(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, pending, 2)
}

func TestSSORepo_TurnOnSAMLOnly_EndsEverySessionOutsideSSOButTheKeptUsers(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	for _, u := range []string{"owner", "alice"} {
		_, _, err := s.Users.UpsertUser(ctx, newTestUser(u, "gh-"+u, u))
		require.NoError(t, err)
	}
	now := time.Now().UTC()
	for _, ses := range []auth.Session{{ID: "owner-github", UserID: "owner"}, {ID: "alice-sso", UserID: "alice", SSO: true}, {ID: "alice-github", UserID: "alice"}} {
		ses.TokenHash, ses.CreatedAt, ses.LastActiveAt, ses.ExpiresAt, ses.Client = ses.ID, now, now, now.Add(time.Hour), auth.ClientBrowser
		require.NoError(t, s.Sessions.CreateSession(ctx, &ses))
	}
	st := sso.SAMLSettings{IdP: sso.IdPMetadata{EntityID: "https://idp.example.com"}, DisplayName: "Acme", Only: true, SPKey: "k"}

	require.NoError(t, s.SSO.TurnOnSAMLOnly(ctx, st, []string{"owner"}, eventbus.OutboxEvent{ID: "e1", Topic: sso.TopicSAMLChanged, Payload: sso.SAMLChangedEvent{Only: true}}))

	got, err := s.SSO.GetSAML(ctx)
	require.NoError(t, err)
	assert.True(t, got.Only)
	for user, want := range map[string][]string{"owner": {"owner-github"}, "alice": {"alice-sso"}} {
		sessions, err := s.Sessions.ListSessionsByUser(ctx, user)
		require.NoError(t, err)
		ids := []string{}
		for _, ses := range sessions {
			ids = append(ids, ses.ID)
		}
		assert.Equal(t, want, ids, user)
	}
	pending, err := s.Outbox.Unpublished(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, pending, 2, "the settings change and one session.revoked")
}

func TestSSORepo_RefreshSAML_StoresTheReadOnlyWhileTheSameProviderIsSaved(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	st := sso.SAMLSettings{
		MetadataURL: "https://idp.example.com/metadata", IdP: sso.IdPMetadata{EntityID: "https://idp.example.com", SSOURL: "https://idp.example.com/sso", Certificates: []string{"AAA"}},
		DisplayName: "Acme", SPKey: "k", SPCertificate: "c", SPPreviousCertificate: "old", UpdatedAt: time.Unix(1_000, 0),
	}
	require.NoError(t, s.SSO.SetSAML(ctx, st))
	idp := st.IdP
	idp.Certificates = []string{"AAA", "BBB"}

	require.NoError(t, s.SSO.RefreshSAML(ctx, st.MetadataURL, idp, time.Unix(2_000, 0), "", eventbus.OutboxEvent{ID: "e1", Topic: sso.TopicSAMLChanged, Payload: sso.SAMLChangedEvent{}}))

	got, err := s.SSO.GetSAML(ctx)
	require.NoError(t, err)
	assert.Equal(t, []string{"AAA", "BBB"}, got.IdP.Certificates)
	assert.Equal(t, time.Unix(2_000, 0).UTC(), got.RefreshedAt)
	assert.Equal(t, "old", got.SPPreviousCertificate)

	require.NoError(t, s.SSO.RefreshSAML(ctx, "https://idp.example.com/other", sso.IdPMetadata{EntityID: idp.EntityID}, time.Unix(3_000, 0), "down",
		eventbus.OutboxEvent{ID: "e2", Topic: sso.TopicSAMLChanged, Payload: sso.SAMLChangedEvent{}}))
	got, err = s.SSO.GetSAML(ctx)
	require.NoError(t, err)
	assert.Empty(t, got.RefreshError, "a read of a URL no longer saved is dropped")
	pending, err := s.Outbox.Unpublished(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, pending, 1, "with its event")
}

func TestSSORepo_SAMLRequests_AreTakenOnceAndOnlyWhileFresh(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	at := time.Unix(10_000, 0)
	require.NoError(t, s.SSO.SaveSAMLRequest(ctx, "old", at.Add(-time.Hour), at.Add(-2*time.Hour)))
	require.NoError(t, s.SSO.SaveSAMLRequest(ctx, "stale", at.Add(-11*time.Minute), at.Add(-time.Hour)))
	require.NoError(t, s.SSO.SaveSAMLRequest(ctx, "fresh", at, at.Add(-30*time.Minute)))

	var left int
	require.NoError(t, s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM saml_requests`).Scan(&left))
	assert.Equal(t, 2, left, "saving drops the requests older than its bound")
	stale, err := s.SSO.TakeSAMLRequest(ctx, "stale", at.Add(-10*time.Minute))
	require.NoError(t, err)
	assert.False(t, stale, "a request older than its life is not taken")
	fresh, err := s.SSO.TakeSAMLRequest(ctx, "fresh", at.Add(-10*time.Minute))
	require.NoError(t, err)
	assert.True(t, fresh)
	again, err := s.SSO.TakeSAMLRequest(ctx, "fresh", at.Add(-10*time.Minute))
	require.NoError(t, err)
	assert.False(t, again, "a request is taken once")
	assert.Error(t, s.SSO.SaveSAMLRequest(ctx, "stale", at, at.Add(-time.Hour)), "an id is never kept twice")
}

func TestSSORepo_MoveSignIn_MovesASAMLSignInToo(t *testing.T) {
	t.Parallel()
	s := newTestStore(t)
	ctx := t.Context()
	seedDefaultRoles(t, s)
	_, _, err := s.Users.UpsertUser(ctx, newTestUser("alice", "gh-alice", "alice"))
	require.NoError(t, err)
	id := sso.Identity{Provider: sso.ProtocolSAML, Key: "https://idp.example.com#alice-1", Email: "alice@example.com"}
	require.NoError(t, s.SSO.Create(ctx, "u-dup", id, "role-member", time.Unix(500, 0)))

	require.NoError(t, s.SSO.MoveSignIn(ctx, "u-dup", "alice", "owner"))

	acct, err := s.SSO.BySubject(ctx, sso.ProtocolSAML, id.Key)
	require.NoError(t, err)
	assert.Equal(t, "alice", acct.UserID)
	_, err = s.SSO.BySubject(ctx, sso.ProtocolOIDC, id.Key)
	require.Error(t, err, "a SAML subject is not an OIDC one")
}
