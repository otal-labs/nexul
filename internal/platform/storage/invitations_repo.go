package storage

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/ids"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/platform/storage/sqlcgen"
	"github.com/otal-labs/nexul/internal/tenancy"
)

var _ tenancy.InvitationRepo = (*InvitationsRepo)(nil)

type InvitationsRepo struct {
	db *sql.DB
	w  *Serializer
	q  *sqlcgen.Queries
}

func (r *InvitationsRepo) Create(ctx context.Context, invitation *tenancy.Invitation, tokenHash string, events ...eventbus.OutboxEvent) error {
	if invitation == nil || invitation.ID == "" || invitation.InvitedBy == "" || !validTokenHash(tokenHash) || !validInvitationLifetime(invitation.CreatedAt, invitation.ExpiresAt) {
		return fmt.Errorf("%w: invitation fields are required", apperrs.ErrInvalid)
	}
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		if err := validateInvitationPackage(ctx, q, invitation, invitation.InvitedBy); err != nil {
			return err
		}
		if err := insertInvitation(ctx, q, invitation, tokenHash); err != nil {
			return fmt.Errorf("create invitation %s: %w", invitation.ID, classifyWriteErr(err))
		}
		for _, grant := range invitation.Grants {
			if err := q.CreateInvitationGrant(ctx, sqlcgen.CreateInvitationGrantParams{
				InvitationID: invitation.ID,
				WorkspaceID:  grant.WorkspaceID,
				RoleID:       grant.RoleID,
				AllowJson:    setJSON(grant.Allow),
				DenyJson:     setJSON(grant.Deny),
			}); err != nil {
				return fmt.Errorf("create invitation grant %s/%s: %w", invitation.ID, grant.WorkspaceID, classifyWriteErr(err))
			}
		}
		return enqueueInvitationEvents(ctx, tx, events)
	})
}

func (r *InvitationsRepo) GetByTokenHash(ctx context.Context, tokenHash string, now time.Time) (*tenancy.Invitation, error) {
	if !validTokenHash(tokenHash) {
		return nil, invalidInvitation()
	}
	var invitation *tenancy.Invitation
	var invalidated bool
	err := r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		row, err := q.GetInvitationByTokenHash(ctx, tokenHash)
		if errors.Is(err, sql.ErrNoRows) {
			return invalidInvitation()
		}
		if err != nil {
			return fmt.Errorf("get invitation: %w", err)
		}
		if row.RedeemedAt.Valid {
			return invalidInvitation()
		}
		candidate, err := readInvitation(ctx, q, row)
		if err != nil {
			if errors.Is(err, apperrs.ErrInvalid) {
				if deleteErr := deleteInvitation(ctx, q, tx, row.ID, "invalid grant"); deleteErr != nil {
					return deleteErr
				}
				invalidated = true
				return nil
			}
			return err
		}
		if err := validateInvitationPackage(ctx, q, candidate, candidate.InvitedBy); err != nil {
			if deleteErr := deleteInvitation(ctx, q, tx, candidate.ID, "creator authority or grant invalid"); deleteErr != nil {
				return deleteErr
			}
			invalidated = true
			return nil
		}
		if !now.Before(candidate.ExpiresAt) {
			if err := deleteInvitation(ctx, q, tx, candidate.ID, "expired"); err != nil {
				return err
			}
			invalidated = true
			return nil
		}
		invitation = candidate
		return nil
	})
	if err != nil {
		return nil, err
	}
	if invalidated {
		return nil, invalidInvitation()
	}
	return invitation, nil
}

func (r *InvitationsRepo) GetByAcceptanceHash(ctx context.Context, acceptanceHash string, now time.Time) (*tenancy.Invitation, error) {
	if !validTokenHash(acceptanceHash) {
		return nil, invalidInvitation()
	}
	handoff, err := r.q.GetOAuthHandoffByAcceptanceHash(ctx, sql.NullString{String: acceptanceHash, Valid: true})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, invalidInvitation()
	}
	if err != nil {
		return nil, fmt.Errorf("get OAuth handoff by acceptance hash: %w", err)
	}
	if handoff.ExpiresAt <= now.Unix() || handoff.CompletedAt.Valid {
		return nil, invalidInvitation()
	}
	row, err := r.q.GetInvitationByID(ctx, handoff.InvitationID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, invalidInvitation()
	}
	if err != nil {
		return nil, fmt.Errorf("get invitation for acceptance: %w", err)
	}
	if row.RedeemedAt.Valid || row.ExpiresAt <= now.Unix() {
		return nil, invalidInvitation()
	}
	invitation, err := readInvitation(ctx, r.q, row)
	if err != nil {
		if errors.Is(err, apperrs.ErrInvalid) {
			return nil, invalidInvitation()
		}
		return nil, err
	}
	return invitation, nil
}

func (r *InvitationsRepo) List(ctx context.Context, actorID string, now time.Time) ([]*tenancy.Invitation, error) {
	if actorID == "" {
		return nil, fmt.Errorf("%w: actor is required", apperrs.ErrInvalid)
	}
	var invitations []*tenancy.Invitation
	err := r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		rows, err := q.ListManageableInvitationHeaders(ctx, sqlcgen.ListManageableInvitationHeadersParams{ExpiresAt: now.Unix(), UserID: actorID, Limit: maxInvitationList})
		if err != nil {
			return fmt.Errorf("list invitations: %w", err)
		}
		for _, row := range rows {
			if row.RedeemedAt.Valid {
				continue
			}
			invitation, err := readInvitation(ctx, q, row)
			if err != nil {
				if errors.Is(err, apperrs.ErrInvalid) {
					if deleteErr := deleteInvitation(ctx, q, tx, row.ID, "invalid grant"); deleteErr != nil {
						return deleteErr
					}
					continue
				}
				return err
			}
			if !now.Before(invitation.ExpiresAt) || validateInvitationPackage(ctx, q, invitation, invitation.InvitedBy) != nil {
				if err := deleteInvitation(ctx, q, tx, invitation.ID, "expired or invalid"); err != nil {
					return err
				}
				continue
			}
			invitations = append(invitations, invitation)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return invitations, nil
}

func (r *InvitationsRepo) Revoke(ctx context.Context, actorID, invitationID string, now time.Time, events ...eventbus.OutboxEvent) error {
	if actorID == "" || invitationID == "" {
		return fmt.Errorf("%w: actor and invitation are required", apperrs.ErrInvalid)
	}
	return r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		row, err := q.GetInvitationByID(ctx, invitationID)
		if errors.Is(err, sql.ErrNoRows) {
			return apperrs.ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("get invitation %s: %w", invitationID, err)
		}
		if err := requireInvitationManager(ctx, q, actorID, invitationID); err != nil {
			return err
		}
		return revokeInvitationRow(ctx, q, tx, row, now, events)
	})
}

func requireInvitationManager(ctx context.Context, q *sqlcgen.Queries, actorID, invitationID string) error {
	grantRows, err := q.ListInvitationGrantsByInvitation(ctx, invitationID)
	if err != nil {
		return fmt.Errorf("list invitation grants %s: %w", invitationID, err)
	}
	grants := make([]*tenancy.InvitationGrant, 0, len(grantRows))
	for _, grant := range grantRows {
		grants = append(grants, &tenancy.InvitationGrant{WorkspaceID: grant.WorkspaceID, RoleID: grant.RoleID})
	}
	allowed, err := actorCanManage(ctx, q, actorID, grants)
	if err != nil {
		return err
	}
	if len(grants) == 0 || !allowed {
		return fmt.Errorf("%w: members:write required in every invited workspace", apperrs.ErrForbidden)
	}
	return nil
}

func revokeInvitationRow(ctx context.Context, q *sqlcgen.Queries, tx *sql.Tx, row sqlcgen.Invitation, now time.Time, events []eventbus.OutboxEvent) error {
	if row.RedeemedAt.Valid {
		return apperrs.ErrNotFound
	}
	invitation, err := readInvitation(ctx, q, row)
	if err != nil {
		if errors.Is(err, apperrs.ErrInvalid) {
			return deleteInvitation(ctx, q, tx, row.ID, "invalid grant")
		}
		return err
	}
	if !now.Before(invitation.ExpiresAt) {
		return deleteInvitation(ctx, q, tx, row.ID, "expired")
	}
	if err := validateInvitationPackage(ctx, q, invitation, invitation.InvitedBy); err != nil {
		return deleteInvitation(ctx, q, tx, row.ID, "creator authority or grant invalid")
	}
	if err := removeInvitation(ctx, q, row.ID); err != nil {
		return err
	}
	return enqueueInvitationEvents(ctx, tx, events)
}

//nolint:gocyclo // Redemption stays in one transaction so every identity, membership, receipt, and outbox write rolls back together.
func (r *InvitationsRepo) Redeem(ctx context.Context, acceptanceHash string, identity tenancy.InvitationIdentity, now time.Time, events ...eventbus.OutboxEvent) (*tenancy.InvitationAdmission, error) {
	if !validTokenHash(acceptanceHash) || identity.Provider == "" || identity.ProviderUserID == "" || identity.ID == "" || identity.Login == "" {
		return nil, invalidInvitation()
	}
	var admission *tenancy.InvitationAdmission
	var invalidatedInvitationID string
	err := r.w.WithTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		handoff, err := q.GetOAuthHandoffByAcceptanceHash(ctx, sql.NullString{String: acceptanceHash, Valid: true})
		if errors.Is(err, sql.ErrNoRows) {
			return invalidInvitation()
		}
		if err != nil {
			return fmt.Errorf("get OAuth handoff: %w", err)
		}
		if handoff.ExpiresAt <= now.Unix() || handoff.Provider != identity.Provider || !handoff.ProviderUserID.Valid || handoff.ProviderUserID.String != identity.ProviderUserID {
			return invalidInvitation()
		}
		if handoff.CompletedAt.Valid {
			user, err := q.GetUserByProviderForInvitation(ctx, sqlcgen.GetUserByProviderForInvitationParams{Provider: identity.Provider, ProviderUserID: identity.ProviderUserID})
			if err != nil || user.AccountStatus != "active" || !handoff.AdmittedUserID.Valid || handoff.AdmittedUserID.String != user.ID {
				return invalidInvitation()
			}
			admission = &tenancy.InvitationAdmission{UserID: user.ID, Created: false}
			return nil
		}
		invitationRow, err := q.GetInvitationByID(ctx, handoff.InvitationID)
		if errors.Is(err, sql.ErrNoRows) {
			return invalidInvitation()
		}
		if err != nil {
			return fmt.Errorf("get invitation: %w", err)
		}
		if invitationRow.RedeemedAt.Valid {
			return invalidInvitation()
		}
		if invitationRow.ExpiresAt <= now.Unix() {
			if err := deleteInvitation(ctx, q, tx, handoff.InvitationID, "expired"); err != nil {
				return err
			}
			invalidatedInvitationID = handoff.InvitationID
			return nil
		}
		invitation, err := readInvitation(ctx, q, invitationRow)
		if err != nil {
			if deleteErr := deleteInvitation(ctx, q, tx, handoff.InvitationID, "invalid grant"); deleteErr != nil {
				return deleteErr
			}
			invalidatedInvitationID = handoff.InvitationID
			return nil
		}
		if err := validateInvitationPackage(ctx, q, invitation, invitation.InvitedBy); err != nil {
			if deleteErr := deleteInvitation(ctx, q, tx, invitation.ID, "creator authority or grant invalid"); deleteErr != nil {
				return deleteErr
			}
			invalidatedInvitationID = invitation.ID
			return nil
		}
		userRow, existing, err := invitationUser(ctx, q, identity)
		if err != nil {
			return err
		}
		userID := identity.ID
		if existing {
			if userRow.AccountStatus != "active" {
				return invalidInvitation()
			}
			userID = userRow.ID
		}
		redeemedBy := sql.NullString{String: userID, Valid: existing}
		changed, err := q.MarkInvitationRedeemed(ctx, sqlcgen.MarkInvitationRedeemedParams{RedeemedAt: sql.NullInt64{Int64: now.Unix(), Valid: true}, RedeemedBy: redeemedBy, ID: invitation.ID, ExpiresAt: now.Unix()})
		if err != nil {
			return fmt.Errorf("mark invitation redeemed: %w", err)
		}
		if changed != 1 {
			return invalidInvitation()
		}
		if !existing {
			if err := insertInvitationUser(ctx, q, identity, now); err != nil {
				return err
			}
		}
		redemptionEvents := append([]eventbus.OutboxEvent{{ID: ids.New(), Topic: tenancy.TopicInvitationRedeemed, Payload: tenancy.InvitationEvent{InvitationID: invitation.ID, UserID: userID}}}, events...)
		if !existing {
			redemptionEvents = append(redemptionEvents, eventbus.OutboxEvent{ID: ids.New(), Topic: tenancy.TopicAccountAdmitted, Payload: tenancy.InvitationEvent{InvitationID: invitation.ID, UserID: userID}})
		}
		for _, grant := range invitation.Grants {
			_, err := q.GetInvitationMember(ctx, sqlcgen.GetInvitationMemberParams{WorkspaceID: grant.WorkspaceID, UserID: userID})
			if err == nil {
				continue
			}
			if !errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("check existing membership: %w", err)
			}
			if err := q.AddInvitationMember(ctx, sqlcgen.AddInvitationMemberParams{UserID: userID, WorkspaceID: grant.WorkspaceID, RoleID: grant.RoleID, CreatedAt: now.Unix()}); err != nil {
				return fmt.Errorf("add invitation membership: %w", classifyWriteErr(err))
			}
			redemptionEvents = append(redemptionEvents, eventbus.OutboxEvent{ID: ids.New(), Topic: tenancy.TopicWorkspaceMemberAdded, Payload: tenancy.InvitationEvent{InvitationID: invitation.ID, UserID: userID, WorkspaceID: grant.WorkspaceID}})
			if len(grant.Allow) == 0 && len(grant.Deny) == 0 {
				continue
			}
			if err := q.AddInvitationOverwrite(ctx, sqlcgen.AddInvitationOverwriteParams{ResourceID: grant.WorkspaceID, UserID: userID, Allow: setJSON(grant.Allow), Deny: setJSON(grant.Deny), CreatedAt: now.Unix(), UpdatedAt: now.Unix()}); err != nil {
				return fmt.Errorf("add invitation overwrite: %w", err)
			}
		}
		if !existing {
			if _, err := q.SetInvitationRedeemedBy(ctx, sqlcgen.SetInvitationRedeemedByParams{RedeemedBy: sql.NullString{String: userID, Valid: true}, ID: invitation.ID}); err != nil {
				return fmt.Errorf("record invitation redeemer: %w", err)
			}
		}
		completed, err := q.CompleteOAuthHandoffRedemption(ctx, sqlcgen.CompleteOAuthHandoffRedemptionParams{AdmittedUserID: sql.NullString{String: userID, Valid: true}, CompletedAt: sql.NullInt64{Int64: now.Unix(), Valid: true}, AcceptanceHash: sql.NullString{String: acceptanceHash, Valid: true}, ExpiresAt: now.Unix()})
		if err != nil {
			return fmt.Errorf("complete OAuth handoff: %w", err)
		}
		if completed != 1 {
			return invalidInvitation()
		}
		if err := enqueueInvitationEvents(ctx, tx, redemptionEvents); err != nil {
			return err
		}
		admission = &tenancy.InvitationAdmission{UserID: userID, Created: !existing}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if invalidatedInvitationID != "" {
		return nil, invalidInvitation()
	}
	return admission, nil
}

func invitationUser(ctx context.Context, q *sqlcgen.Queries, identity tenancy.InvitationIdentity) (sqlcgen.User, bool, error) {
	row, err := q.GetUserByProviderForInvitation(ctx, sqlcgen.GetUserByProviderForInvitationParams{Provider: identity.Provider, ProviderUserID: identity.ProviderUserID})
	if err == nil {
		return row, true, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return sqlcgen.User{}, false, fmt.Errorf("get invitation user: %w", err)
	}
	return sqlcgen.User{}, false, nil
}

func insertInvitationUser(ctx context.Context, q *sqlcgen.Queries, identity tenancy.InvitationIdentity, now time.Time) error {
	if err := q.InsertUserForInvitation(ctx, sqlcgen.InsertUserForInvitationParams{ID: identity.ID, Provider: identity.Provider, ProviderUserID: identity.ProviderUserID, Login: identity.Login, Name: identity.Name, AvatarUrl: identity.AvatarURL, CreatedAt: now.Unix(), UpdatedAt: now.Unix()}); err != nil {
		return fmt.Errorf("admit invitation user: %w", classifyWriteErr(err))
	}
	return nil
}

func validateInvitationPackage(ctx context.Context, q *sqlcgen.Queries, invitation *tenancy.Invitation, actorID string) error {
	if len(invitation.Grants) == 0 {
		return fmt.Errorf("%w: invitation has no grants", apperrs.ErrInvalid)
	}
	seen := make(map[string]struct{}, len(invitation.Grants))
	for _, grant := range invitation.Grants {
		if grant == nil || grant.WorkspaceID == "" || grant.RoleID == "" {
			return fmt.Errorf("%w: invitation grant is incomplete", apperrs.ErrInvalid)
		}
		if _, ok := seen[grant.WorkspaceID]; ok {
			return fmt.Errorf("%w: invitation contains duplicate workspaces", apperrs.ErrInvalid)
		}
		seen[grant.WorkspaceID] = struct{}{}
		if _, err := q.GetWorkspaceForInvitationGrant(ctx, grant.WorkspaceID); err != nil {
			return fmt.Errorf("%w: workspace is unavailable", apperrs.ErrInvalid)
		}
		role, err := q.GetRoleForInvitationGrant(ctx, sqlcgen.GetRoleForInvitationGrantParams{ID: grant.RoleID, WorkspaceID: grant.WorkspaceID})
		if err != nil {
			return fmt.Errorf("%w: role is unavailable", apperrs.ErrInvalid)
		}
		if role.IsOwnerRole != 0 {
			return fmt.Errorf("%w: Owner cannot be granted by invitation", apperrs.ErrInvalid)
		}
		if err := validateInvitationSets(grant.Allow, grant.Deny); err != nil {
			return err
		}
	}
	allowed, err := actorCanManage(ctx, q, actorID, invitation.Grants)
	if err != nil {
		return err
	}
	if !allowed {
		return fmt.Errorf("%w: members:write required in every invited workspace", apperrs.ErrForbidden)
	}
	return nil
}

func validateInvitationSets(allow, deny permissions.Set) error {
	denied := make(map[permissions.Action]struct{}, len(deny))
	for _, action := range deny {
		if _, ok := permissions.ParseAction(string(action)); !ok {
			return fmt.Errorf("%w: invitation permission set is invalid", apperrs.ErrInvalid)
		}
		denied[action] = struct{}{}
	}
	for _, action := range allow {
		if _, ok := permissions.ParseAction(string(action)); !ok {
			return fmt.Errorf("%w: invitation permission set is invalid", apperrs.ErrInvalid)
		}
		if _, ok := denied[action]; ok {
			return fmt.Errorf("%w: invitation permission set is invalid", apperrs.ErrInvalid)
		}
	}
	return nil
}

func actorCanManage(ctx context.Context, q *sqlcgen.Queries, actorID string, grants []*tenancy.InvitationGrant) (bool, error) {
	for _, grant := range grants {
		row, err := q.GetInvitationCreatorWorkspaceAccess(ctx, sqlcgen.GetInvitationCreatorWorkspaceAccessParams{WorkspaceID: grant.WorkspaceID, UserID: actorID})
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		if err != nil {
			return false, fmt.Errorf("check workspace authority: %w", err)
		}
		if row.IsOwnerRole != 0 {
			continue
		}
		rolePermissions, err := parseSet(row.Permissions)
		if err != nil {
			return false, fmt.Errorf("decode role permissions: %w", err)
		}
		allowed := rolePermissions.Has(permissions.MembersWrite)
		denied := false
		if row.Deny.Valid {
			deny, err := parseSet(row.Deny.String)
			if err != nil {
				return false, fmt.Errorf("decode workspace deny: %w", err)
			}
			if deny.Has(permissions.MembersWrite) {
				allowed = false
				denied = true
			}
		}
		if !denied && row.Allow.Valid {
			allow, err := parseSet(row.Allow.String)
			if err != nil {
				return false, fmt.Errorf("decode workspace allow: %w", err)
			}
			allowed = allowed || allow.Has(permissions.MembersWrite)
		}
		if !allowed {
			return false, nil
		}
	}
	return true, nil
}

func readInvitation(ctx context.Context, q *sqlcgen.Queries, row sqlcgen.Invitation) (*tenancy.Invitation, error) {
	grantRows, err := q.ListInvitationGrantsByInvitation(ctx, row.ID)
	if err != nil {
		return nil, fmt.Errorf("list invitation grants: %w", err)
	}
	invitation := &tenancy.Invitation{ID: row.ID, InvitedBy: row.InvitedBy, CreatedAt: time.Unix(row.CreatedAt, 0).UTC(), ExpiresAt: time.Unix(row.ExpiresAt, 0).UTC()}
	if row.RedeemedAt.Valid {
		redeemedAt := time.Unix(row.RedeemedAt.Int64, 0).UTC()
		invitation.RedeemedAt = &redeemedAt
	}
	if row.RedeemedBy.Valid {
		invitation.RedeemedBy = row.RedeemedBy.String
	}
	for _, grantRow := range grantRows {
		allow, err := parseSet(grantRow.AllowJson)
		if err != nil {
			return nil, fmt.Errorf("decode invitation allow: %w", err)
		}
		deny, err := parseSet(grantRow.DenyJson)
		if err != nil {
			return nil, fmt.Errorf("decode invitation deny: %w", err)
		}
		workspace, err := q.GetWorkspaceForInvitationGrant(ctx, grantRow.WorkspaceID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, fmt.Errorf("%w: invitation workspace is unavailable", apperrs.ErrInvalid)
			}
			return nil, fmt.Errorf("read invitation workspace: %w", err)
		}
		role, err := q.GetRoleForInvitationGrant(ctx, sqlcgen.GetRoleForInvitationGrantParams{ID: grantRow.RoleID, WorkspaceID: grantRow.WorkspaceID})
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, fmt.Errorf("%w: invitation role is unavailable", apperrs.ErrInvalid)
			}
			return nil, fmt.Errorf("read invitation role: %w", err)
		}
		invitation.Grants = append(invitation.Grants, &tenancy.InvitationGrant{WorkspaceID: grantRow.WorkspaceID, WorkspaceName: workspace.Name, RoleID: grantRow.RoleID, RoleName: role.Name, Allow: allow, Deny: deny})
	}
	return invitation, nil
}

func insertInvitation(ctx context.Context, q *sqlcgen.Queries, invitation *tenancy.Invitation, tokenHash string) error {
	return q.CreateInvitation(ctx, sqlcgen.CreateInvitationParams{ID: invitation.ID, TokenHash: tokenHash, InvitedBy: invitation.InvitedBy, CreatedAt: invitation.CreatedAt.Unix(), ExpiresAt: invitation.ExpiresAt.Unix()})
}

func deleteInvitation(ctx context.Context, q *sqlcgen.Queries, tx *sql.Tx, id, reason string) error {
	deleted, err := q.DeleteInvitation(ctx, id)
	if err != nil {
		return fmt.Errorf("delete invitation %s: %w", id, err)
	}
	if deleted == 0 {
		return nil
	}
	if err := insertOutboxRow(ctx, tx, ids.New(), tenancy.TopicInvitationDeleted, tenancy.InvitationEvent{InvitationID: id, Reason: reason}); err != nil {
		return fmt.Errorf("record invitation deletion %s: %w", id, err)
	}
	return nil
}

func removeInvitation(ctx context.Context, q *sqlcgen.Queries, id string) error {
	_, err := q.DeleteInvitation(ctx, id)
	if err != nil {
		return fmt.Errorf("delete invitation %s: %w", id, err)
	}
	return nil
}

func enqueueInvitationEvents(ctx context.Context, tx *sql.Tx, events []eventbus.OutboxEvent) error {
	for _, event := range events {
		if err := insertOutboxRow(ctx, tx, event.ID, event.Topic, event.Payload); err != nil {
			return err
		}
	}
	return nil
}

func validTokenHash(tokenHash string) bool {
	if len(tokenHash) != sha256HexLength {
		return false
	}
	_, err := hex.DecodeString(tokenHash)
	return err == nil
}

func validInvitationLifetime(createdAt, expiresAt time.Time) bool {
	duration := expiresAt.Sub(createdAt)
	return duration == 24*time.Hour || duration == 7*24*time.Hour
}

const sha256HexLength = 64

const maxInvitationList = 100

func invalidInvitation() error {
	return fmt.Errorf("%w: invitation is invalid or expired", apperrs.ErrNotFound)
}
