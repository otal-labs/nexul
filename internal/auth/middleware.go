package auth

import (
	"context"
	"net/http"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/httpx"
)

type ctxKey string

const userCtxKey ctxKey = "user"

// UserFromCtx returns the authenticated user record set by RequireAuth, or nil when unauthenticated.
func UserFromCtx(ctx context.Context) *User {
	u, _ := ctx.Value(userCtxKey).(*User)
	return u
}

// RequireAuth resolves the full User into context, or 401s.
func (s *Service) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.cfg.Users == nil {
			httpx.WriteError(w, apperrs.ErrUnauthorized)
			return
		}
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		token = strings.TrimSpace(token)
		if token == "" {
			httpx.WriteError(w, apperrs.ErrUnauthorized)
			return
		}
		user, err := s.authenticate(r, token)
		if err != nil {
			httpx.WriteError(w, apperrs.ErrUnauthorized)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userCtxKey, user)))
	})
}

// authenticate resolves a Bearer token (session or PAT) to its user; the PAT path re-reads the store each time.
func (s *Service) authenticate(r *http.Request, token string) (*User, error) {
	if isPAT(token) {
		if s.cfg.PATs == nil {
			return nil, apperrs.ErrUnauthorized
		}
		user, err := s.AuthenticatePAT(r.Context(), token)
		if err != nil {
			return nil, err
		}
		return user, nil
	}
	userID, err := s.Verify(token)
	if err != nil {
		return nil, err
	}
	user, err := s.cfg.Users.GetUserByID(r.Context(), userID)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// RequireWS guards a WS endpoint: a browser can't set Authorization on a WS upgrade, so the SPA uses a query param.
func (s *Service) RequireWS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.cfg.Users == nil {
			httpx.WriteError(w, apperrs.ErrUnauthorized)
			return
		}
		token := strings.TrimSpace(r.URL.Query().Get("token"))
		if token == "" {
			httpx.WriteError(w, apperrs.ErrUnauthorized)
			return
		}
		user, err := s.authenticate(r, token)
		if err != nil {
			httpx.WriteError(w, apperrs.ErrUnauthorized)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userCtxKey, user)))
	})
}
