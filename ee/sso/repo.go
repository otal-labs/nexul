package sso

import (
	"context"
	"time"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// Repo stores the instance's one provider; implemented in internal/platform/storage.
type Repo interface {
	// Get returns the zero Settings when the instance has none.
	Get(ctx context.Context) (Settings, error)
	Set(ctx context.Context, s Settings, evts ...eventbus.OutboxEvent) error
	// TurnOnOnly saves s with Only on and, in the same transaction, ends every session not started through single
	// sign-on except keepUsers', each with its session.revoked event.
	TurnOnOnly(ctx context.Context, s Settings, keepUsers []string, evts ...eventbus.OutboxEvent) error
	// Delete removes the settings and every grant.
	Delete(ctx context.Context, evts ...eventbus.OutboxEvent) error
	// SaveGrant stores the person's refresh token, replacing an earlier one, as last confirmed at at.
	SaveGrant(ctx context.Context, userID, refreshToken string, at time.Time) error
	// StaleGrants returns up to limit grants last confirmed before before, oldest first; a grant whose token cannot be
	// read comes back with an empty RefreshToken.
	StaleGrants(ctx context.Context, before time.Time, limit int) ([]StoredGrant, error)
	// ConfirmGrant marks the grant confirmed at at, replacing its refresh token when newRefreshToken is not empty.
	ConfirmGrant(ctx context.Context, userID, newRefreshToken string, at time.Time) error
	// DropGrant deletes the grant and changes no session.
	DropGrant(ctx context.Context, userID string) error
	// RefuseGrant deletes the grant and ends every session of the person's started through single sign-on, each with its
	// session.revoked event, in one transaction; it returns how many ended.
	RefuseGrant(ctx context.Context, userID string) (int, error)
}

// Accounts is how a sign-in finds, links and makes the person's account; implemented in internal/platform/storage, which
// joins a new account to the default workspace the way an invitation does.
type Accounts interface {
	// BySubject finds the account a provider's issuer-scoped subject is already linked to; ErrNotFound when none.
	BySubject(ctx context.Context, provider, key string) (Account, error)
	// ByEmail finds the account with a Google or Discord sign-in on that verified email; ErrNotFound when none.
	ByEmail(ctx context.Context, email string) (Account, error)
	// Link attaches the subject to the account, which then signs in through it.
	Link(ctx context.Context, userID string, id Identity) error
	// Create makes the account with the subject as its sign-in and gives it roleID in the default workspace.
	Create(ctx context.Context, userID string, id Identity, roleID string, at time.Time) error
	// CanJoinAs reports whether roleID is a role of the default workspace that is not its Owner role.
	CanJoinAs(ctx context.Context, roleID string) (bool, error)
	// MoveSignIn moves fromID's single sign-on, its only sign-in, onto toID, then disables fromID and ends its
	// sessions; ErrInvalid when fromID signs in another way too, ErrConflict when toID already has one.
	MoveSignIn(ctx context.Context, fromID, toID, actorID string) error
}

// Moves is what moving people between the provider's protocols reads and writes; implemented in internal/platform/storage.
type Moves interface {
	// OldSignIns lists the active accounts with a single sign-on of another protocol than current and none of current's.
	OldSignIns(ctx context.Context, current string) ([]MoveAccount, error)
	// OnlyOn lists the active accounts whose one sign-in is current's single sign-on.
	OnlyOn(ctx context.Context, current string) ([]MoveAccount, error)
	ExpectedLinks(ctx context.Context) ([]ExpectedLink, error)
	// Expect saves the account's expected sign-in, replacing its earlier one; ErrNotFound for no account, ErrInvalid for
	// a removed one, ErrConflict when it signs in through current already or another account expects that email.
	Expect(ctx context.Context, link ExpectedLink, current string, evts ...eventbus.OutboxEvent) error
	// Unexpect deletes the account's expected sign-in; ErrNotFound when it has none.
	Unexpect(ctx context.Context, userID string, evts ...eventbus.OutboxEvent) error
	// ByExpected finds the account an expected sign-in names for email; ErrNotFound when none.
	ByExpected(ctx context.Context, email string) (Account, error)
	// LinkExpected attaches the subject to the account and deletes its expected sign-in, in one transaction.
	LinkExpected(ctx context.Context, userID string, id Identity, evts ...eventbus.OutboxEvent) error
}
