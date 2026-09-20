package connectors

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/httpx"
)

const oauthStateCookie = "nexul_connector_oauth_state"

type ctxKey string

const userIDKey ctxKey = "user_id"

// WithUserID attaches the authenticated user id so this gateway can attribute connections without importing auth (ADR 0017).
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// UserIDFromCtx returns the user id injected by WithUserID, or "".
func UserIDFromCtx(ctx context.Context) string {
	id, _ := ctx.Value(userIDKey).(string)
	return id
}

// Handler adapts the connectors use-cases to the HTTP/JSON gateway (ADR 0019).
type Handler struct {
	svc *Service
}

// NewHandler wires the connectors REST gateway over the given service.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Routes returns the connectors REST endpoints, mounted behind RequireAuth at /api/connectors.
func (h *Handler) Routes() http.Handler {
	mux := httpx.NewServeMux()
	mux.HandleFunc("GET /api/connectors", h.list)
	mux.HandleFunc("GET /api/connectors/{id}/oauth/start", h.oauthStart)
	mux.HandleFunc("POST /api/connectors/{id}/manual/verify", h.verifyManual)
	mux.HandleFunc("POST /api/connectors/{id}/manual", h.saveManual)
	mux.HandleFunc("POST /api/connectors/{id}/disconnect", h.disconnect)
	mux.HandleFunc("PUT /api/connectors/{id}/app-config", h.setAppConfig)
	return mux
}

// PublicRoutes lives under /auth/: a provider redirect has no Authorization header and would 401 under RequireAuth.
func (h *Handler) PublicRoutes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /auth/connectors/{id}/callback", h.oauthCallback)
	return mux
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	st, err := h.svc.List(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, st)
}

// oauthStart mints a CSRF state carrying the user id, the only way the public callback below learns who connected.
func (h *Handler) oauthStart(w http.ResponseWriter, r *http.Request) {
	connectorID := r.PathValue("id")
	userID := UserIDFromCtx(r.Context())
	if userID == "" {
		httpx.WriteError(w, apperrs.ErrUnauthorized)
		return
	}
	nonce, err := newNonce()
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	state := nonce + ":" + userID
	url, err := h.svc.AuthorizeURL(r.Context(), connectorID, state)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookie,
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil,
	})
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"url": url})
}

// oauthCallback verifies the state cookie, exchanges the code, and redirects to the SPA; failures redirect too.
func (h *Handler) oauthCallback(w http.ResponseWriter, r *http.Request) {
	connectorID := r.PathValue("id")
	cookie, err := r.Cookie(oauthStateCookie)
	if err != nil || cookie.Value == "" || cookie.Value != r.URL.Query().Get("state") {
		h.redirectSPAError(w, r, connectorID, "sign-in state expired or mismatched — try connecting again")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: oauthStateCookie, Value: "", MaxAge: -1, Path: "/", HttpOnly: true})
	_, userID, ok := strings.Cut(cookie.Value, ":")
	if !ok || userID == "" {
		h.redirectSPAError(w, r, connectorID, "sign-in state carried no user — try connecting again")
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		h.redirectSPAError(w, r, connectorID, "the provider sent no authorization code — try connecting again")
		return
	}
	if err := h.svc.CompleteOAuth(r.Context(), connectorID, code, userID); err != nil {
		// Authorized but not installed: forward to the install page so the post-install redirect lands back here.
		var notInstalled *NotInstalledError
		if errors.As(err, &notInstalled) && notInstalled.InstallURL != "" {
			if nonce, nerr := newNonce(); nerr == nil {
				if u, perr := url.Parse(notInstalled.InstallURL); perr == nil {
					state := nonce + ":" + userID
					http.SetCookie(w, &http.Cookie{
						Name:     oauthStateCookie,
						Value:    state,
						Path:     "/",
						HttpOnly: true,
						SameSite: http.SameSiteLaxMode,
						Secure:   r.TLS != nil,
					})
					q := u.Query()
					q.Set("state", state)
					u.RawQuery = q.Encode()
					http.Redirect(w, r, u.String(), http.StatusFound)
					return
				}
			}
		}
		h.redirectSPAError(w, r, connectorID, err.Error())
		return
	}
	http.Redirect(w, r, h.spaBase(r)+"/settings?connector="+url.QueryEscape(connectorID)+"&connected=1", http.StatusFound)
}

// spaBase resolves the configured instance URL, not r.Host: a split-origin dev API origin isn't browser-reachable.
func (h *Handler) spaBase(r *http.Request) string {
	if base := h.svc.InstanceURL(r.Context()); base != "" {
		return base
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

// redirectSPAError lands an OAuth-callback failure back in the SPA settings page, where ConnectorsSection toasts it.
func (h *Handler) redirectSPAError(w http.ResponseWriter, r *http.Request, connectorID, msg string) {
	http.Redirect(w, r,
		h.spaBase(r)+"/settings?connector="+url.QueryEscape(connectorID)+"&error="+url.QueryEscape(msg),
		http.StatusFound)
}

// saveManual is the non-OAuth connect path; Verify runs before anything is stored, so a bad credential never saves.
// verifyManual runs the provider check only (204 or the provider's error), leaving storage to saveManual.
func (h *Handler) verifyManual(w http.ResponseWriter, r *http.Request) {
	if UserIDFromCtx(r.Context()) == "" {
		httpx.WriteError(w, apperrs.ErrUnauthorized)
		return
	}
	var fields map[string]string
	if err := json.NewDecoder(r.Body).Decode(&fields); err != nil {
		httpx.WriteError(w, fmt.Errorf("%w: invalid request body", apperrs.ErrInvalid))
		return
	}
	if key := r.URL.Query().Get("check"); key != "" {
		if err := h.svc.VerifyManualCheck(r.Context(), r.PathValue("id"), fields, key); err != nil {
			httpx.WriteError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if err := h.svc.VerifyManualCredentials(r.Context(), r.PathValue("id"), fields); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) saveManual(w http.ResponseWriter, r *http.Request) {
	connectorID := r.PathValue("id")
	userID := UserIDFromCtx(r.Context())
	if userID == "" {
		httpx.WriteError(w, apperrs.ErrUnauthorized)
		return
	}
	var fields map[string]string
	if err := json.NewDecoder(r.Body).Decode(&fields); err != nil {
		httpx.WriteError(w, fmt.Errorf("%w: invalid request body", apperrs.ErrInvalid))
		return
	}
	st, err := h.svc.SaveManualCredentials(r.Context(), connectorID, userID, fields)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, st)
}

func (h *Handler) disconnect(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Disconnect(r.Context(), r.PathValue("id")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "disconnected"})
}

// setAppConfig stores connectorID's app-level OAuth registration (CN3a); the use-case enforces the owner-only check.
func (h *Handler) setAppConfig(w http.ResponseWriter, r *http.Request) {
	connectorID := r.PathValue("id")
	userID := UserIDFromCtx(r.Context())
	var body struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		BaseURL      string `json:"base_url"`
		AppSlug      string `json:"app_slug"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteError(w, fmt.Errorf("%w: invalid request body", apperrs.ErrInvalid))
		return
	}
	st, err := h.svc.SetAppConfig(r.Context(), userID, connectorID, body.ClientID, body.ClientSecret, body.BaseURL, body.AppSlug)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, st)
}

func newNonce() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate oauth state: %w", err)
	}
	return hex.EncodeToString(b), nil
}
