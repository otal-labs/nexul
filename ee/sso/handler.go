package sso

import (
	"context"
	"errors"
	"net/http"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/httpx"
	"github.com/otal-labs/nexul/internal/platform/logging"
)

const (
	stateCookie = "nexul_sso_state"
	stateMaxAge = 10 * time.Minute
)

// Reason codes a failed sign-in carries back to /login as ?error=; the web sign-in page owns their copy.
const (
	codeAccountDisabled = "account_disabled"
	codeAccountRemoved  = "account_removed"
	codeAccessDenied    = "access_denied"
	codeProviderError   = "provider_error"
	codeStateMismatch   = "state_mismatch"
	codeNotConfigured   = "not_configured"
	codeUnverified      = "sso_email_unverified"
	codeNotReady        = "sso_not_ready"
	codeFailed          = "sso_failed"
)

// Browser is what the sign-in callback takes from auth: the web app's origin, whether cookies must be Secure, and a
// session for an account on the device the request came from.
type Browser interface {
	SPAOrigin(r *http.Request) string
	SecureCookie(r *http.Request) bool
	SSOSessionFor(r *http.Request, userID string) (string, error)
}

// Handler adapts single sign-on to the HTTP/JSON gateway (ADR 0019): the public sign-in routes and the Owner's settings.
type Handler struct {
	svc     *Service
	browser Browser
}

// NewHandler wires the single sign-on gateway over the given service.
func NewHandler(svc *Service, b Browser) *Handler {
	return &Handler{svc: svc, browser: b}
}

// PublicRoutes are the routes that need no session: what the sign-in page shows, the redirect out and the callback.
func (h *Handler) PublicRoutes() http.Handler {
	mux := httpx.NewServeMux()
	mux.HandleFunc("GET /api/auth/sso", h.public)
	mux.HandleFunc("GET /auth/oidc", h.start)
	mux.HandleFunc("GET /auth/oidc/callback", h.callback)
	return mux
}

// Routes are the Owner's settings: read them, save them, remove them.
func (h *Handler) Routes() http.Handler {
	mux := httpx.NewServeMux()
	mux.HandleFunc("GET /api/instance/sso", h.get)
	mux.HandleFunc("PUT /api/instance/sso", h.update)
	mux.HandleFunc("DELETE /api/instance/sso", h.remove)
	mux.HandleFunc("POST /api/instance/sso/verify", h.verify)
	return mux
}

func (h *Handler) public(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, h.svc.Public(r.Context()))
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	v, err := h.svc.View(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, v)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	var p Patch
	if err := httpx.DecodeJSON(r, &p); err != nil {
		httpx.WriteError(w, err)
		return
	}
	v, err := h.svc.Update(r.Context(), p)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, v)
}

// verify runs one named check of a form's values, as the connectors' verify does, and stores nothing.
func (h *Handler) verify(w http.ResponseWriter, r *http.Request) {
	var p Patch
	if err := httpx.DecodeJSON(r, &p); err != nil {
		httpx.WriteError(w, err)
		return
	}
	detail, err := h.svc.Check(r.Context(), r.URL.Query().Get("check"), p)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	if detail == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"detail": detail})
}

func (h *Handler) remove(w http.ResponseWriter, r *http.Request) {
	v, err := h.svc.Remove(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, v)
}

// start sends the browser to the provider with a state kept in a cookie the callback checks.
func (h *Handler) start(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	target, state, err := h.svc.Start(ctx)
	if err != nil {
		h.failed(w, r, signInCode(err), err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: stateCookie, Value: state, Path: "/", MaxAge: int(stateMaxAge.Seconds()),
		HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: h.browser.SecureCookie(r),
	})
	http.Redirect(w, r, target, http.StatusFound)
}

// callback checks the state cookie, completes the sign-in and lands on /login with the session token, as the other
// providers' callbacks do.
func (h *Handler) callback(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	r = r.WithContext(ctx)
	state := r.URL.Query().Get("state")
	cookie, err := r.Cookie(stateCookie)
	if state == "" || err != nil || cookie.Value != state {
		h.failed(w, r, codeStateMismatch, errors.New("sso state mismatch"))
		return
	}
	http.SetCookie(w, &http.Cookie{Name: stateCookie, Value: "", MaxAge: -1, Path: "/", HttpOnly: true, Secure: h.browser.SecureCookie(r)})
	if refusal := r.URL.Query().Get("error"); refusal != "" {
		code := codeProviderError
		if refusal == "access_denied" {
			code = codeAccessDenied
		}
		h.failed(w, r, code, errors.New("provider returned error "+refusal))
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		h.failed(w, r, codeFailed, errors.New("callback has no code"))
		return
	}
	userID, err := h.svc.Complete(ctx, state, code)
	if err != nil {
		h.failed(w, r, signInCode(err), err)
		return
	}
	token, err := h.browser.SSOSessionFor(r, userID)
	if err != nil {
		h.failed(w, r, codeFailed, err)
		return
	}
	http.Redirect(w, r, h.browser.SPAOrigin(r)+"/login?token="+token, http.StatusFound)
}

func signInCode(err error) string {
	switch {
	case errors.Is(err, errDisabled):
		return codeAccountDisabled
	case errors.Is(err, errRemoved):
		return codeAccountRemoved
	case errors.Is(err, errUnverified):
		return codeUnverified
	case errors.Is(err, errNotReady):
		return codeNotReady
	case errors.Is(err, apperrs.ErrInvalid):
		return codeNotConfigured
	}
	return codeFailed
}

// failed answers a browser-navigated callback with a redirect to the sign-in page, never JSON; the real reason is logged, not sent.
func (h *Handler) failed(w http.ResponseWriter, r *http.Request, code string, err error) {
	logging.FromCtx(r.Context()).Warn("single sign-on failed", "reason", code, "error", err)
	http.Redirect(w, r, h.browser.SPAOrigin(r)+"/login?error="+code, http.StatusFound)
}
