package auth

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// maxAvatarOverrideBytes caps a decoded avatar override data URI at ~10MB; there's no file-upload/object-storage here.
const maxAvatarOverrideBytes = 10 * 1024 * 1024

// UpdateProfileOverride sets the caller's display name/avatar override; empty clears back to provider-sourced.
func (s *Service) UpdateProfileOverride(ctx context.Context, userID, displayName, avatarOverrideURL string) (*User, error) {
	displayName = strings.TrimSpace(displayName)
	if avatarOverrideURL != "" {
		if err := validateAvatarOverrideURL(avatarOverrideURL); err != nil {
			return nil, err
		}
	}
	current, err := s.cfg.Users.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user %s: %w", userID, err)
	}
	if deref(current.DisplayName) == displayName && deref(current.AvatarOverrideURL) == avatarOverrideURL {
		return current, nil
	}
	var namePtr, avatarPtr *string
	if displayName != "" {
		namePtr = &displayName
	}
	if avatarOverrideURL != "" {
		avatarPtr = &avatarOverrideURL
	}
	event := eventbus.OutboxEvent{ID: newUserID(), Topic: TopicProfileUpdated, Payload: AccountLifecycleEvent{AccountID: userID}}
	if err := s.cfg.Users.SetProfileOverride(ctx, userID, namePtr, avatarPtr, event); err != nil {
		return nil, fmt.Errorf("set profile override %s: %w", userID, err)
	}
	return s.cfg.Users.GetUserByID(ctx, userID)
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// validateAvatarOverrideURL requires a base64 data: URI decoding to no more than maxAvatarOverrideBytes.
func validateAvatarOverrideURL(raw string) error {
	if !strings.HasPrefix(raw, "data:") {
		return fmt.Errorf("%w: avatar_override_url must be a data: URI", apperrs.ErrInvalid)
	}
	meta, data, ok := strings.Cut(strings.TrimPrefix(raw, "data:"), ",")
	if !ok || !strings.Contains(meta, "base64") {
		return fmt.Errorf("%w: avatar_override_url must be a base64-encoded data: URI", apperrs.ErrInvalid)
	}
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return fmt.Errorf("%w: avatar_override_url has an invalid base64 payload", apperrs.ErrInvalid)
	}
	if len(decoded) > maxAvatarOverrideBytes {
		return fmt.Errorf("%w: avatar image exceeds the %dMB limit", apperrs.ErrInvalid, maxAvatarOverrideBytes/(1024*1024))
	}
	return nil
}

// GetUserByID resolves a user record for consumers of the owner role.
func (s *Service) GetUserByID(ctx context.Context, id string) (*User, error) {
	return s.cfg.Users.GetUserByID(ctx, id)
}

// ListUsers returns every user record, for access's permissions modal user picker (via the access.Users interface).
func (s *Service) ListUsers(ctx context.Context) ([]*User, error) {
	return s.cfg.Users.ListUsers(ctx)
}

// GetAccount returns the caller's own account for an empty or own id; any other account needs accounts:read.
func (s *Service) GetAccount(ctx context.Context, actorID, id string) (*User, error) {
	if id == "" || id == actorID {
		return s.Whoami(ctx, actorID)
	}
	if err := s.requireAnywhere(ctx, actorID, permissions.AccountsRead); err != nil {
		return nil, err
	}
	user, err := s.cfg.Users.GetUserByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get account %s: %w", id, err)
	}
	return user, nil
}

// UpdateAccountStatus moves an account to active or disabled; active reactivates a disabled account and restores a removed one.
func (s *Service) UpdateAccountStatus(ctx context.Context, actorID, targetID string, status AccountStatus) error {
	// The permission check comes first, so a caller without it cannot tell a real account id from an unknown one.
	if err := s.requireAnywhere(ctx, actorID, permissions.AccountsWrite); err != nil {
		return err
	}
	target, err := s.cfg.Users.GetUserByID(ctx, targetID)
	if err != nil {
		return fmt.Errorf("get account %s: %w", targetID, err)
	}
	if status == AccountDisabled && target.AccountStatus == AccountDisabled {
		return nil
	}
	if status == AccountDisabled {
		return s.DisableAccount(ctx, actorID, target.ID)
	}
	if status != AccountActive {
		return fmt.Errorf("%w: account status must be active or disabled", apperrs.ErrInvalid)
	}
	if target.AccountStatus == AccountDisabled {
		return s.ReactivateAccount(ctx, actorID, target.ID)
	}
	if target.AccountStatus == AccountRemoved {
		return s.RestoreAccount(ctx, actorID, target.ID)
	}
	return nil
}

// ListAccounts returns every registered account to a holder of accounts:read.
func (s *Service) ListAccounts(ctx context.Context, actorID string) ([]*User, error) {
	if err := s.requireAnywhere(ctx, actorID, permissions.AccountsRead); err != nil {
		return nil, err
	}
	return s.cfg.Users.ListUsers(ctx)
}

// DisableAccount blocks authentication while preserving the account's memberships and credentials.
func (s *Service) DisableAccount(ctx context.Context, actorID, targetID string) error {
	return s.changeAccountStatus(ctx, actorID, targetID, AccountDisabled, AccountActive, TopicAccountDisabled)
}

// ReactivateAccount restores an intentionally disabled account.
func (s *Service) ReactivateAccount(ctx context.Context, actorID, targetID string) error {
	return s.changeAccountStatus(ctx, actorID, targetID, AccountActive, AccountDisabled, TopicAccountReactivated)
}

// RemoveAccount tombstones an account and removes its access credentials and memberships; it needs accounts:delete.
func (s *Service) RemoveAccount(ctx context.Context, actorID, targetID string) error {
	return s.changeAccountStatus(ctx, actorID, targetID, AccountRemoved, AccountActive, TopicAccountRemoved)
}

// RestoreAccount reactivates a removed account without restoring deleted access.
func (s *Service) RestoreAccount(ctx context.Context, actorID, targetID string) error {
	return s.changeAccountStatus(ctx, actorID, targetID, AccountActive, AccountRemoved, TopicAccountRestored)
}

func (s *Service) changeAccountStatus(ctx context.Context, actorID, targetID string, status, expectedFrom AccountStatus, topic string) error {
	action := permissions.AccountsWrite
	if status == AccountRemoved {
		action = permissions.AccountsDelete
	}
	if err := s.requireAnywhere(ctx, actorID, action); err != nil {
		return err
	}
	if strings.TrimSpace(targetID) == "" {
		return fmt.Errorf("%w: account id is required", apperrs.ErrInvalid)
	}
	target, err := s.cfg.Users.GetUserByID(ctx, targetID)
	if err != nil {
		return fmt.Errorf("get account %s: %w", targetID, err)
	}
	current := target.AccountStatus
	if current == "" {
		current = AccountActive
	}
	if status == AccountRemoved && (current == AccountDisabled || current == AccountActive) {
		expectedFrom = current
	}
	if current != expectedFrom {
		return fmt.Errorf("%w: account transition from %s to %s is not allowed", apperrs.ErrInvalid, current, status)
	}
	payload := AccountLifecycleEvent{AccountID: targetID, ActorID: actorID}
	return s.cfg.Users.SetAccountStatus(ctx, targetID, status, eventbus.OutboxEvent{ID: newUserID(), Topic: topic, Payload: payload})
}

// ListMembers returns the allowlist (accounts:read, ADR 0040).
func (s *Service) ListMembers(ctx context.Context, userID string) ([]string, error) {
	if err := s.requireAnywhere(ctx, userID, permissions.AccountsRead); err != nil {
		return nil, err
	}
	return s.cfg.Allowlist.List(ctx)
}

// AddMember adds a git provider username to the allowlist (accounts:write, ADR 0040).
func (s *Service) AddMember(ctx context.Context, userID, login string) error {
	if err := s.requireAnywhere(ctx, userID, permissions.AccountsWrite); err != nil {
		return err
	}
	login = strings.ToLower(strings.TrimSpace(login))
	if login == "" {
		return fmt.Errorf("%w: username is required", apperrs.ErrInvalid)
	}
	return s.cfg.Allowlist.Add(ctx, login)
}

// RemoveMember removes a username from the allowlist (accounts:delete); the user row is untouched, sessions live until TTL.
func (s *Service) RemoveMember(ctx context.Context, userID, login string) error {
	if err := s.requireAnywhere(ctx, userID, permissions.AccountsDelete); err != nil {
		return err
	}
	return s.cfg.Allowlist.Remove(ctx, strings.ToLower(strings.TrimSpace(login)))
}

// IsLoginAllowlisted reports whether login can sign in, backing tenancy's AllowlistGate seam (ADR 0017).
func (s *Service) IsLoginAllowlisted(ctx context.Context, login string) (bool, error) {
	return s.cfg.Allowlist.Contains(ctx, strings.ToLower(strings.TrimSpace(login)))
}

// UserIDForLogin resolves login to an existing User's id, found=false if none (tenancy's UserLookupGate seam, ADR 0017).
func (s *Service) UserIDForLogin(ctx context.Context, login string) (string, bool, error) {
	u, err := s.cfg.Users.GetUserByLogin(ctx, strings.ToLower(strings.TrimSpace(login)))
	if err != nil {
		if errors.Is(err, apperrs.ErrNotFound) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("resolve user for login %s: %w", login, err)
	}
	return u.ID, true, nil
}
