package tenancy

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

type invitationRepoFake struct {
	created      []*Invitation
	listed       []*Invitation
	byToken      map[string]*Invitation
	byAcceptance map[string]*Invitation
	events       []eventbus.OutboxEvent
	createErr    error
	listErr      error
	previewErr   error
	redeemErr    error
	redeemed     []InvitationIdentity
}

func (f *invitationRepoFake) Create(_ context.Context, invitation *Invitation, tokenHash string, events ...eventbus.OutboxEvent) error {
	if f.createErr != nil {
		return f.createErr
	}
	copy := *invitation
	f.created = append(f.created, &copy)
	f.events = append(f.events, events...)
	_ = tokenHash
	return nil
}

func (f *invitationRepoFake) GetByTokenHash(_ context.Context, tokenHash string, _ time.Time) (*Invitation, error) {
	if f.previewErr != nil {
		return nil, f.previewErr
	}
	invitation, ok := f.byToken[tokenHash]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	return invitation, nil
}

func (f *invitationRepoFake) GetByAcceptanceHash(_ context.Context, acceptanceHash string, _ time.Time) (*Invitation, error) {
	invitation, ok := f.byAcceptance[acceptanceHash]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	return invitation, nil
}

func (f *invitationRepoFake) List(_ context.Context, _ string, _ time.Time) ([]*Invitation, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.listed, nil
}

func (f *invitationRepoFake) Revoke(_ context.Context, _ string, _ string, _ time.Time, events ...eventbus.OutboxEvent) error {
	f.events = append(f.events, events...)
	return nil
}

func (f *invitationRepoFake) Redeem(_ context.Context, _ string, identity InvitationIdentity, _ time.Time, events ...eventbus.OutboxEvent) (*InvitationAdmission, error) {
	if f.redeemErr != nil {
		return nil, f.redeemErr
	}
	f.redeemed = append(f.redeemed, identity)
	f.events = append(f.events, events...)
	return &InvitationAdmission{UserID: identity.ID, Created: true}, nil
}

type invitationURLFake struct{ url string }

func (f invitationURLFake) InstanceURL(context.Context) string { return f.url }

func newInvitationServiceFixture() (*InvitationService, *invitationRepoFake) {
	repo := &invitationRepoFake{byToken: map[string]*Invitation{}, byAcceptance: map[string]*Invitation{}}
	svc := NewInvitationService(repo, invitationURLFake{url: "https://nexul.example"})
	svc.now = func() time.Time { return time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC) }
	return svc, repo
}

func TestInvitationService_Create_OneTimeURLAndLifetime(t *testing.T) {
	t.Parallel()
	svc, repo := newInvitationServiceFixture()
	ctx := t.Context()
	input := CreateInvitationInput{
		Grants:        []*InvitationGrant{{WorkspaceID: "ws-1", RoleID: "role-editor"}},
		ExpiresInDays: 7,
	}

	created, err := svc.Create(ctx, "actor", input)
	require.NoError(t, err)
	require.Len(t, repo.created, 1)
	rawToken := strings.TrimPrefix(created.URL, "https://nexul.example/invite#")
	assert.Equal(t, "https://nexul.example/invite#"+rawToken, created.URL)
	assert.NotEmpty(t, rawToken)
	parsed, err := uuid.Parse(rawToken)
	require.NoError(t, err)
	assert.Equal(t, uuid.Version(4), parsed.Version())
	assert.Equal(t, 7*24*time.Hour, repo.created[0].ExpiresAt.Sub(repo.created[0].CreatedAt))
	assert.NotContains(t, created.URL, repo.created[0].ID)
}

func TestInvitationService_Create_RejectsUnsupportedLifetime(t *testing.T) {
	t.Parallel()
	svc, _ := newInvitationServiceFixture()
	_, err := svc.Create(t.Context(), "actor", CreateInvitationInput{Grants: []*InvitationGrant{{WorkspaceID: "ws-1", RoleID: "role-editor"}}, ExpiresInDays: 2})
	assert.ErrorIs(t, err, apperrs.ErrInvalid)
}

func TestInvitationService_Preview_HidesPermissionOverwrites(t *testing.T) {
	t.Parallel()
	svc, repo := newInvitationServiceFixture()
	invitation := &Invitation{ID: "inv-1", ExpiresAt: svc.now().Add(24 * time.Hour), Grants: []*InvitationGrant{{WorkspaceID: "ws-1", WorkspaceName: "Acme", RoleID: "role-editor", RoleName: "Editor", Allow: permissions.SetOf(permissions.DocsRead), Deny: permissions.SetOf(permissions.DocsDelete)}}}
	repo.byToken["token-hash"] = invitation

	preview, err := svc.Preview(t.Context(), "4d2f3f29-2a43-4ae7-b2d4-0b6f1a7f4c11")
	assert.ErrorIs(t, err, apperrs.ErrNotFound)
	assert.Nil(t, preview)

	preview, err = svc.previewHash(t.Context(), "token-hash")
	require.NoError(t, err)
	require.Len(t, preview.Grants, 1)
	assert.Equal(t, "Acme", preview.Grants[0].WorkspaceName)
	assert.Equal(t, "Editor", preview.Grants[0].RoleName)
}

func TestInvitationService_AcceptanceAndRedeem_UseGenericInvalidErrors(t *testing.T) {
	t.Parallel()
	svc, repo := newInvitationServiceFixture()
	acceptanceHash := strings.Repeat("a", 64)
	repo.byAcceptance[acceptanceHash] = &Invitation{ID: "inv-1", ExpiresAt: svc.now().Add(time.Hour), Grants: []*InvitationGrant{{WorkspaceID: "ws-1", WorkspaceName: "Acme", RoleID: "role-editor", RoleName: "Editor"}}}
	repo.redeemErr = apperrs.ErrNotFound

	_, err := svc.AcceptanceFromHash(t.Context(), acceptanceHash)
	require.NoError(t, err)
	_, err = svc.Redeem(t.Context(), acceptanceHash, InvitationIdentity{ID: "u-1", Provider: "github", ProviderUserID: "p-1"})
	assert.ErrorIs(t, err, apperrs.ErrNotFound)
	assert.NotContains(t, err.Error(), "acceptance-hash")
}

func TestInvitationService_Events_CarryNoCredential(t *testing.T) {
	t.Parallel()
	svc, repo := newInvitationServiceFixture()
	created, err := svc.Create(t.Context(), "actor", CreateInvitationInput{Grants: []*InvitationGrant{{WorkspaceID: "ws-1", RoleID: "role-editor"}}, ExpiresInDays: 1})
	require.NoError(t, err)
	require.NoError(t, svc.Revoke(t.Context(), "actor", created.Invitation.ID))
	for _, event := range repo.events {
		payload, err := json.Marshal(event.Payload)
		require.NoError(t, err)
		assert.NotContains(t, strings.ToLower(string(payload)), "token")
		assert.NotContains(t, strings.ToLower(string(payload)), "hash")
	}
}

func TestInvitationService_List_PropagatesRepositoryError(t *testing.T) {
	t.Parallel()
	svc, repo := newInvitationServiceFixture()
	repo.listErr = errors.New("storage down")
	_, err := svc.List(t.Context(), "actor")
	assert.ErrorIs(t, err, repo.listErr)
}
