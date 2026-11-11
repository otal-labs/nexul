package auth

import (
	"context"
	"fmt"
	"net/http"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// errSSOOnly refuses a sign-in the instance has turned off in favour of single sign-on.
var errSSOOnly = fmt.Errorf("%w: this instance signs in through single sign-on only", apperrs.ErrUnauthorized)

// SetSSOOnly wires the enterprise single sign-on switch; unset, every sign-in method stays on.
func (s *Service) SetSSOOnly(only func(context.Context) bool) {
	s.ssoOnly = only
}

func (s *Service) onlySSO(ctx context.Context) bool {
	return s.ssoOnly != nil && s.ssoOnly(ctx)
}

// InstanceOwnerHasPasskey reports whether an active instance Owner holds a passkey, the one way in that stays on when
// single sign-on is the only sign-in.
func (s *Service) InstanceOwnerHasPasskey(ctx context.Context) (bool, error) {
	if s.cfg.Passkeys == nil {
		return false, nil
	}
	owners, err := s.cfg.Users.ListActiveOwnerIDs(ctx)
	if err != nil {
		return false, fmt.Errorf("list owners: %w", err)
	}
	for _, id := range owners {
		owner, err := s.isInstanceOwner(ctx, id)
		if err != nil || !owner {
			continue
		}
		passkeys, err := s.cfg.Passkeys.ListPasskeys(ctx, id)
		if err != nil {
			return false, fmt.Errorf("list passkeys of %s: %w", id, err)
		}
		if len(passkeys) > 0 {
			return true, nil
		}
	}
	return false, nil
}

// SPAOrigin is the web app's origin, where a finished sign-in lands.
func (h *Handler) SPAOrigin(r *http.Request) string {
	return h.spaOrigin(r)
}

// SecureCookie reports whether a sign-in's cookies must be Secure.
func (h *Handler) SecureCookie(r *http.Request) bool {
	return h.svc.secureCookie(r.Context(), r.TLS != nil)
}

// SessionFor starts a session for an account another package verified the sign-in of, on the device r came from.
func (h *Handler) SessionFor(r *http.Request, userID string) (string, error) {
	return h.svc.CreateSession(WithDevice(r.Context(), DeviceFromRequest(r)), userID)
}

// requirePasskeyAllowed lets a passkey sign in unless single sign-on is the only way in and it is not an instance Owner's.
func (s *Service) requirePasskeyAllowed(ctx context.Context, userID string) error {
	if !s.onlySSO(ctx) {
		return nil
	}
	owner, err := s.isInstanceOwner(ctx, userID)
	if err != nil {
		return err
	}
	if !owner {
		return errSSOOnly
	}
	return nil
}
