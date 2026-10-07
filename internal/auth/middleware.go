package auth

import (
	"context"
	"net/http"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/httpx"
)

type ctxKey string

const (
	userCtxKey    ctxKey = "user"
	sessionCtxKey ctxKey = "session"
)

// UserFromCtx returns the authenticated user record set by RequireAuth, or nil when unauthenticated.
func UserFromCtx(ctx context.Context) *User {
	u, _ := ctx.Value(userCtxKey).(*User)
	return u
}

// SessionFromCtx returns the device session the request came in on; nil for a personal access token or setup pass.
func SessionFromCtx(ctx context.Context) *Session {
	s, _ := ctx.Value(sessionCtxKey).(*Session)
	return s
}

func withPrincipal(ctx context.Context, user *User, ses *Session) context.Context {
	ctx = context.WithValue(ctx, userCtxKey, user)
	if ses == nil {
		return ctx
	}
	return context.WithValue(ctx, sessionCtxKey, ses)
}

// RequireAuth resolves the full User into context, or 401s; a setup pass resolves to the setup identity (setup.go).
func (s *Service) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.cfg.Users == nil {
			unauthorized(w)
			return
		}
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		token = strings.TrimSpace(token)
		if token == "" {
			unauthorized(w)
			return
		}
		user, ses, err := s.authenticate(r, token)
		if err != nil {
			unauthorized(w)
			return
		}
		next.ServeHTTP(w, r.WithContext(withPrincipal(r.Context(), user, ses)))
	})
}

// unauthorized carries the Bearer challenge RFC 6750 requires on every 401 of a bearer-protected resource.
func unauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Bearer realm="nexul"`)
	httpx.WriteError(w, apperrs.ErrUnauthorized)
}

// authenticate resolves a Bearer token (session, PAT or setup pass) to its user; every path re-reads its store.
func (s *Service) authenticate(r *http.Request, token string) (*User, *Session, error) {
	if isSetupPass(token) {
		user, err := s.authenticateSetupPass(r, token)
		return user, nil, err
	}
	if isPAT(token) {
		if s.cfg.PATs == nil {
			return nil, nil, apperrs.ErrUnauthorized
		}
		user, err := s.AuthenticatePAT(r.Context(), token)
		return user, nil, err
	}
	return s.AuthenticateSession(r.Context(), token, httpx.ClientAddr(r))
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
		user, ses, err := s.authenticate(r, token)
		if err != nil {
			httpx.WriteError(w, apperrs.ErrUnauthorized)
			return
		}
		next.ServeHTTP(w, r.WithContext(withPrincipal(r.Context(), user, ses)))
	})
}
