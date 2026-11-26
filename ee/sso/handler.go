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
	codeLapsed          = "sso_lapsed"
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
	mux.HandleFunc("GET /auth/saml", h.startSAML)
	mux.HandleFunc("POST /auth/saml/acs", h.acs)
	mux.HandleFunc("GET /auth/saml/metadata", h.spMetadata)
	return mux
}

// Routes are the Owner's settings for either protocol (read them, check them, save them, remove them) and moving
// people's sign-ins between accounts.
func (h *Handler) Routes() http.Handler {
	mux := httpx.NewServeMux()
	mux.HandleFunc("GET /api/instance/sso", h.get)
	mux.HandleFunc("PUT /api/instance/sso", h.update)
	mux.HandleFunc("DELETE /api/instance/sso", h.remove)
	mux.HandleFunc("POST /api/instance/sso/verify", h.verify)
	mux.HandleFunc("POST /api/instance/sso/move", h.move)
	mux.HandleFunc("GET /api/instance/sso/moves", h.moves)
	mux.HandleFunc("POST /api/instance/sso/moves", h.linkMatched)
	mux.HandleFunc("PUT /api/instance/sso/expected/{account_id}", h.expect)
	mux.HandleFunc("DELETE /api/instance/sso/expected/{account_id}", h.unexpect)
	mux.HandleFunc("GET /api/instance/saml", h.getSAML)
	mux.HandleFunc("PUT /api/instance/saml", h.updateSAML)
	mux.HandleFunc("DELETE /api/instance/saml", h.removeSAML)
	mux.HandleFunc("POST /api/instance/saml/verify", h.verifySAML)
	mux.HandleFunc("POST /api/instance/saml/rotate", h.rotateSAML)
	mux.HandleFunc("POST /api/instance/saml/retire", h.retireSAML)
	return mux
}

type moveBody struct {
	FromAccountID string `json:"from_account_id"`
	ToAccountID   string `json:"to_account_id"`
}

func (h *Handler) move(w http.ResponseWriter, r *http.Request) {
	var b moveBody
	if err := httpx.DecodeJSON(r, &b); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if err := h.svc.MoveSignIn(r.Context(), b.FromAccountID, b.ToAccountID); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) moves(w http.ResponseWriter, r *http.Request) {
	v, err := h.svc.SignInMoves(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, v)
}

type moveResultJSON struct {
	MovePair
	Moved bool `json:"moved"`
	// Message is why the pair did not move; empty when it moved.
	Message string `json:"message"`
}

// linkMatched moves every pair sent and answers 200 with each pair's outcome, since one that fails leaves the rest moved.
func (h *Handler) linkMatched(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Pairs []MovePair `json:"pairs"`
	}
	if err := httpx.DecodeJSON(r, &b); err != nil {
		httpx.WriteError(w, err)
		return
	}
	results, err := h.svc.LinkMatched(r.Context(), b.Pairs)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	out := make([]moveResultJSON, 0, len(results))
	for _, res := range results {
		row := moveResultJSON{MovePair: res.MovePair, Moved: res.Err == nil}
		if res.Err != nil {
			row.Message = httpx.Message(res.Err)
			logging.FromCtx(r.Context()).Warn("sign-in not moved", "from", res.FromAccountID, "to", res.ToAccountID, "error", res.Err)
		}
		out = append(out, row)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"results": out})
}

func (h *Handler) expect(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Email string `json:"email"`
	}
	if err := httpx.DecodeJSON(r, &b); err != nil {
		httpx.WriteError(w, err)
		return
	}
	link, err := h.svc.ExpectSignIn(r.Context(), r.PathValue("account_id"), b.Email)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, link)
}

func (h *Handler) unexpect(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.RemoveExpected(r.Context(), r.PathValue("account_id")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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
	case errors.Is(err, errLapsed):
		return codeLapsed
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
