package main

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/otal-labs/nexul/internal/auth"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/tenancy"
)

type invitationAuthGate struct {
	svc *tenancy.InvitationService
}

func (g invitationAuthGate) GetInvitationByToken(ctx context.Context, rawToken string, _ time.Time) (*auth.InvitationAcceptance, error) {
	invitation, err := g.svc.AcceptanceFromToken(ctx, rawToken)
	if err != nil {
		return nil, err
	}
	return mapInvitationAcceptance(g.svc, ctx, invitation), nil
}

func (g invitationAuthGate) GetInvitationByAcceptance(ctx context.Context, acceptanceHash string, _ time.Time) (*auth.InvitationAcceptance, error) {
	invitation, err := g.svc.AcceptanceFromHash(ctx, acceptanceHash)
	if err != nil {
		return nil, err
	}
	return mapInvitationAcceptance(g.svc, ctx, invitation), nil
}

func (g invitationAuthGate) RedeemInvitation(ctx context.Context, acceptanceHash string, identity auth.InvitationIdentity, _ time.Time, _ ...eventbus.OutboxEvent) (auth.InvitationAdmission, error) {
	invitation, err := g.svc.AcceptanceFromHash(ctx, acceptanceHash)
	if err != nil {
		return auth.InvitationAdmission{}, err
	}
	admission, err := g.svc.Redeem(ctx, acceptanceHash, tenancy.InvitationIdentity{
		ID: identity.ID, Provider: string(identity.Provider), ProviderUserID: identity.ProviderUserID,
		Login: identity.Login, Name: identity.Name, AvatarURL: identity.AvatarURL,
	})
	if err != nil {
		return auth.InvitationAdmission{}, err
	}
	workspaceIDs := make([]string, 0, len(invitation.Grants))
	for _, grant := range invitation.Grants {
		workspaceIDs = append(workspaceIDs, grant.WorkspaceID)
	}
	return auth.InvitationAdmission{UserID: admission.UserID, Created: admission.Created, WorkspaceIDs: workspaceIDs}, nil
}

func mapInvitationAcceptance(svc *tenancy.InvitationService, ctx context.Context, invitation *tenancy.Invitation) *auth.InvitationAcceptance {
	instanceURL := strings.TrimRight(svc.InstanceURL(ctx), "/")
	instanceName := instanceURL
	if parsed, err := url.Parse(instanceURL); err == nil && parsed.Host != "" {
		instanceName = parsed.Host
	}
	details := &auth.InvitationAcceptance{
		InvitationID: invitation.ID,
		InstanceName: instanceName,
		InstanceURL:  instanceURL,
		ExpiresAt:    invitation.ExpiresAt,
		Grants:       make([]auth.InvitationGrant, 0, len(invitation.Grants)),
	}
	for _, grant := range invitation.Grants {
		details.Grants = append(details.Grants, auth.InvitationGrant{
			WorkspaceID: grant.WorkspaceID, WorkspaceName: grant.WorkspaceName,
			RoleID: grant.RoleID, RoleName: grant.RoleName,
			Allow: permissionStrings(grant.Allow), Deny: permissionStrings(grant.Deny),
		})
	}
	return details
}

func permissionStrings(set permissions.Set) []string {
	out := make([]string, len(set))
	for i, action := range set {
		out[i] = string(action)
	}
	return out
}
