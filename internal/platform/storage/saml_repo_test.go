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
