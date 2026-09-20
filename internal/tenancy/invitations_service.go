package tenancy

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
)

type InstanceURLGate interface {
	InstanceURL(ctx context.Context) string
}

type CreateInvitationInput struct {
	Grants        []*InvitationGrant
	ExpiresInDays int
}

type CreatedInvitation struct {
	Invitation *Invitation `json:"invitation"`
	URL        string      `json:"url"`
}

type InvitationGrantPreview struct {
	WorkspaceID   string `json:"workspace_id"`
	WorkspaceName string `json:"workspace_name"`
	RoleID        string `json:"role_id"`
	RoleName      string `json:"role_name"`
}

type InvitationPreview struct {
	InstanceURL string                    `json:"instance_url"`
	ExpiresAt   time.Time                 `json:"expires_at"`
	Grants      []*InvitationGrantPreview `json:"grants"`
}

type InvitationService struct {
	repo        InvitationRepo
	instanceURL InstanceURLGate
	now         func() time.Time
}

func NewInvitationService(repo InvitationRepo, instanceURL InstanceURLGate) *InvitationService {
	return &InvitationService{repo: repo, instanceURL: instanceURL, now: time.Now}
}

func (s *InvitationService) Create(ctx context.Context, actorID string, input CreateInvitationInput) (*CreatedInvitation, error) {
	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		return nil, fmt.Errorf("%w: actor is required", apperrs.ErrInvalid)
	}
	if len(input.Grants) == 0 {
		return nil, fmt.Errorf("%w: at least one workspace grant is required", apperrs.ErrInvalid)
	}
	duration, err := invitationDuration(input.ExpiresInDays)
	if err != nil {
		return nil, err
	}
	baseURL := strings.TrimRight(strings.TrimSpace(s.instanceURL.InstanceURL(ctx)), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("%w: instance URL is not configured", apperrs.ErrInvalid)
	}
	rawToken, err := uuid.NewRandom()
	if err != nil {
		return nil, fmt.Errorf("generate invitation token: %w", err)
	}
	token := rawToken.String()
	tokenHash, err := HashInvitationToken(token)
	if err != nil {
		return nil, err
	}
	now := s.now().UTC()
	invitation := &Invitation{ID: ids.New(), InvitedBy: actorID, CreatedAt: now, ExpiresAt: now.Add(duration), Grants: cloneInvitationGrants(input.Grants)}
	if err := s.repo.Create(ctx, invitation, tokenHash, eventbus.OutboxEvent{ID: ids.New(), Topic: TopicInvitationCreated, Payload: InvitationEvent{InvitationID: invitation.ID, ActorID: actorID}}); err != nil {
		return nil, err
	}
	return &CreatedInvitation{Invitation: invitation, URL: baseURL + "/invite#" + token}, nil
}

func (s *InvitationService) List(ctx context.Context, actorID string) ([]*Invitation, error) {
	return s.repo.List(ctx, strings.TrimSpace(actorID), s.now().UTC())
}

func (s *InvitationService) Revoke(ctx context.Context, actorID, invitationID string) error {
	return s.repo.Revoke(ctx, strings.TrimSpace(actorID), strings.TrimSpace(invitationID), s.now().UTC(), eventbus.OutboxEvent{ID: ids.New(), Topic: TopicInvitationRevoked, Payload: InvitationEvent{InvitationID: invitationID, ActorID: actorID}})
}

func (s *InvitationService) Preview(ctx context.Context, rawToken string) (*InvitationPreview, error) {
	tokenHash, err := HashInvitationToken(rawToken)
	if err != nil {
		return nil, invalidInvitation()
	}
	return s.previewHash(ctx, tokenHash)
}

func (s *InvitationService) previewHash(ctx context.Context, tokenHash string) (*InvitationPreview, error) {
	invitation, err := s.repo.GetByTokenHash(ctx, tokenHash, s.now().UTC())
	if err != nil {
		return nil, invalidInvitationIfNeeded(err)
	}
	return s.preview(ctx, invitation), nil
}

func (s *InvitationService) AcceptanceFromToken(ctx context.Context, rawToken string) (*Invitation, error) {
	tokenHash, err := HashInvitationToken(rawToken)
	if err != nil {
		return nil, invalidInvitation()
	}
	return s.repo.GetByTokenHash(ctx, tokenHash, s.now().UTC())
}

func (s *InvitationService) ResolveInvitation(ctx context.Context, rawToken string) (string, error) {
	invitation, err := s.AcceptanceFromToken(ctx, rawToken)
	if err != nil {
		return "", err
	}
	return invitation.ID, nil
}

func (s *InvitationService) AcceptanceFromHash(ctx context.Context, acceptanceHash string) (*Invitation, error) {
	if !validCredentialHash(acceptanceHash) {
		return nil, invalidInvitation()
	}
	return s.repo.GetByAcceptanceHash(ctx, acceptanceHash, s.now().UTC())
}

func (s *InvitationService) Redeem(ctx context.Context, acceptanceHash string, identity InvitationIdentity) (*InvitationAdmission, error) {
	if !validCredentialHash(acceptanceHash) {
		return nil, invalidInvitation()
	}
	admission, err := s.repo.Redeem(ctx, acceptanceHash, identity, s.now().UTC())
	if err != nil {
		return nil, invalidInvitationIfNeeded(err)
	}
	return admission, nil
}

func (s *InvitationService) preview(ctx context.Context, invitation *Invitation) *InvitationPreview {
	preview := &InvitationPreview{InstanceURL: strings.TrimRight(strings.TrimSpace(s.instanceURL.InstanceURL(ctx)), "/"), ExpiresAt: invitation.ExpiresAt, Grants: make([]*InvitationGrantPreview, 0, len(invitation.Grants))}
	for _, grant := range invitation.Grants {
		preview.Grants = append(preview.Grants, &InvitationGrantPreview{WorkspaceID: grant.WorkspaceID, WorkspaceName: grant.WorkspaceName, RoleID: grant.RoleID, RoleName: grant.RoleName})
	}
	return preview
}

func cloneInvitationGrants(grants []*InvitationGrant) []*InvitationGrant {
	out := make([]*InvitationGrant, 0, len(grants))
	for _, grant := range grants {
		if grant == nil {
			out = append(out, nil)
			continue
		}
		copy := *grant
		copy.Allow = grant.Allow.Actions()
		copy.Deny = grant.Deny.Actions()
		out = append(out, &copy)
	}
	return out
}

func invitationDuration(days int) (time.Duration, error) {
	if days == 0 {
		days = 7
	}
	if days == 1 {
		return 24 * time.Hour, nil
	}
	if days == 7 {
		return 7 * 24 * time.Hour, nil
	}
	return 0, fmt.Errorf("%w: invitation expiry must be one or seven days", apperrs.ErrInvalid)
}

func validCredentialHash(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func invalidInvitation() error {
	return fmt.Errorf("%w: invitation is invalid or expired", apperrs.ErrNotFound)
}

func invalidInvitationIfNeeded(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, apperrs.ErrNotFound) {
		return invalidInvitation()
	}
	return err
}
