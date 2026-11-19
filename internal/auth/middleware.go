package auth

import (
	"context"
	"net/http"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/httpx"
	"github.com/otal-labs/nexul/internal/platform/identity"
)

type ctxKey string

const (
	userCtxKey    ctxKey = "user"
	sessionCtxKey ctxKey = "session"
	viewerCtxKey  ctxKey = "viewer"
	grantCtxKey   ctxKey = "grant"
)

// GrantFromCtx returns the MCP client whose sign-in grant the request came in on; "" for any other credential.
func GrantFromCtx(ctx context.Context) string {
	g, _ := ctx.Value(grantCtxKey).(string)
	return g
}

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

// WithViewedUser makes target the request's user for a view-as request (ADR 0160); the real session stays in context.
func WithViewedUser(ctx context.Context, target *User, viewer *identity.ViewerRef) context.Context {
	return context.WithValue(context.WithValue(ctx, userCtxKey, target), viewerCtxKey, viewer)
}

// ViewerFromCtx returns who is viewing as the request's user and the workspace the view is held to; nil outside a view.
func ViewerFromCtx(ctx context.Context) *identity.ViewerRef {
	v, _ := ctx.Value(viewerCtxKey).(*identity.ViewerRef)
	return v
}

// RequireAuth resolves the full User into context, or 401s; a setup pass resolves to the setup identity (setup.go).
func (s *Service) RequireAuth(next http.Handler) http.Handler {
	return s.requireAuth(next, false, func(*http.Request) string { return bearerChallenge })
}

// RequireMCPAuth is RequireAuth for the MCP endpoint, the one resource a sign-in grant is issued for; its 401 names
// the protected-resource metadata, where an MCP client starts sign-in.
func (s *Service) RequireMCPAuth(next http.Handler) http.Handler {
	return s.requireAuth(next, true, func(r *http.Request) string {
		if !s.OAuthEnabled() {
			return bearerChallenge
		}
		return bearerChallenge + `, resource_metadata="` + s.apiOrigin(r) + `/.well-known/oauth-protected-resource/mcp"`
	})
}

const bearerChallenge = `Bearer realm="nexul"`

func (s *Service) requireAuth(next http.Handler, allowGrant bool, challenge func(*http.Request) string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		if s.cfg.Users == nil || token == "" {
			unauthorized(w, challenge(r))
			return
		}
		user, ses, grant, err := s.authenticate(r, token)
		if err != nil || grant != "" && !allowGrant {
			unauthorized(w, challenge(r))
			return
		}
		ctx := withPrincipal(r.Context(), user, ses)
		if grant != "" {
			ctx = context.WithValue(ctx, grantCtxKey, grant)
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// unauthorized carries the Bearer challenge RFC 6750 requires on every 401 of a bearer-protected resource.
func unauthorized(w http.ResponseWriter, challenge string) {
	w.Header().Set("WWW-Authenticate", challenge)
	httpx.WriteError(w, apperrs.ErrUnauthorized)
}

// authenticate resolves a Bearer token (session, PAT or setup pass) to its user, and a sign-in grant to its client;
// every path re-reads its store.
func (s *Service) authenticate(r *http.Request, token string) (*User, *Session, string, error) {
	if isSetupPass(token) {
		user, err := s.authenticateSetupPass(r, token)
		return user, nil, "", err
	}
	if isPAT(token) {
		if s.cfg.PATs == nil {
			return nil, nil, "", apperrs.ErrUnauthorized
		}
		user, pat, err := s.authenticatePAT(r.Context(), token)
		if err != nil {
			return nil, nil, "", err
		}
		return user, nil, pat.OAuthClientID, nil
	}
	user, ses, err := s.AuthenticateSession(r.Context(), token, httpx.ClientAddr(r))
	return user, ses, "", err
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
		user, ses, grant, err := s.authenticate(r, token)
		if err != nil || grant != "" {
			httpx.WriteError(w, apperrs.ErrUnauthorized)
			return
		}
		next.ServeHTTP(w, r.WithContext(withPrincipal(r.Context(), user, ses)))
	})
}
