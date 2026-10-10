package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/httpx"
	"github.com/otal-labs/nexul/internal/platform/logging"
	"github.com/otal-labs/nexul/internal/platform/version"
)

// Handler's Routes() is public; ProtectedRoutes() needs RequireAuth.
type Handler struct {
	svc *Service
}

// NewHandler wires the auth REST gateway over the given service.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

type callbackRequest struct {
	Code string `json:"code"`
}

// Routes returns the public auth endpoints; /auth/dev-login only registers with DevLogin enabled.
func (h *Handler) Routes() http.Handler {
	mux := httpx.NewServeMux()
	mux.HandleFunc("GET /auth/github", h.startOAuth(ProviderGitHub))
	mux.HandleFunc("GET /auth/github/manifest/callback", h.returnGitHubManifest)
	mux.HandleFunc("GET /auth/callback", h.callbackGET(ProviderGitHub))
	mux.HandleFunc("POST /auth/callback", h.callbackPOST)
	mux.HandleFunc("GET /auth/google", h.startOAuth(ProviderGoogle))
	mux.HandleFunc("GET /auth/google/callback", h.callbackGET(ProviderGoogle))
	mux.HandleFunc("GET /auth/discord", h.startOAuth(ProviderDiscord))
	mux.HandleFunc("GET /auth/discord/callback", h.callbackGET(ProviderDiscord))
	mux.HandleFunc("POST /api/invitations/oauth", h.startInvitationOAuth)
	mux.HandleFunc("POST /api/invitations/acceptance", h.acceptInvitation)
	mux.HandleFunc("POST /api/invitations/redeem", h.redeemInvitation)
	mux.HandleFunc("GET /api/auth/bootstrap-status", h.bootstrapStatus)
	mux.HandleFunc("POST /api/auth/bootstrap", h.bootstrap)
	mux.HandleFunc("POST /api/auth/bootstrap/verify", h.bootstrapVerify)
	mux.HandleFunc("POST /api/setup/unlock", h.unlockSetup)
	mux.HandleFunc("POST /api/auth/connect-codes/exchange", h.exchangeConnectCode)
	if h.svc.DevLoginEnabled() {
		mux.HandleFunc("GET /auth/dev-login", h.devLogin)
	}
	return mux
}

// bootstrapStatus reports how far first run has got (ADR 0040); public, side-effect free.
func (h *Handler) bootstrapStatus(w http.ResponseWriter, r *http.Request) {
	st, err := h.svc.cfg.Settings.Get(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	open, err := h.svc.SetupOpen(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"configured":         st.Configured(),
		"reconfigurable":     st.Configured() && open,
		"google_configured":  st.ProviderConfigured(ProviderGoogle),
		"discord_configured": st.ProviderConfigured(ProviderDiscord),
		"setup_open":         open,
		"instance_url":       st.InstanceURL,
		"local":              h.svc.cfg.Local,
	})
}

type unlockRequest struct {
	Code string `json:"code"`
}

// unlockSetup trades the setup code for a setup pass; public, throttled per client address.
func (h *Handler) unlockSetup(w http.ResponseWriter, r *http.Request) {
	var req unlockRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	pass, err := h.svc.UnlockSetup(r.Context(), httpx.ClientAddr(r), req.Code)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, pass)
}

// SetupRoutes returns the first-run endpoints behind RequireAuth, mounted at /api/setup.
func (h *Handler) SetupRoutes() http.Handler {
	mux := httpx.NewServeMux()
	mux.HandleFunc("PUT /api/setup/instance-url", h.setSetupInstanceURL)
	mux.HandleFunc("POST /api/setup/github-app/start", h.startGitHubManifest)
	mux.HandleFunc("POST /api/setup/github-app/callback", h.completeGitHubManifest)
	mux.HandleFunc("GET /api/setup/public-address", h.publicAddress)
	return mux
}

type setupInstanceURLRequest struct {
	URL string `json:"url"`
}

func (h *Handler) setSetupInstanceURL(w http.ResponseWriter, r *http.Request) {
	var req setupInstanceURLRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	instanceURL, err := h.svc.SetSetupInstanceURL(r.Context(), currentUserID(r), req.URL)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"instance_url": instanceURL})
}

func (h *Handler) publicAddress(w http.ResponseWriter, r *http.Request) {
	addr, err := h.svc.PublicAddress(r.Context(), currentUserID(r))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, addr)
}

type bootstrapRequest struct {
	InstanceURL  string `json:"instance_url"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	// AppSlug is the GitHub App's URL slug, needed by connectors' install; collected here since Connect predates Settings.
	AppSlug string `json:"app_slug"`
}

// bootstrap stores the GitHub App credentials (and the instance URL when first run has not) on a fresh instance; setup pass only.
func (h *Handler) bootstrap(w http.ResponseWriter, r *http.Request) {
	var req bootstrapRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	st, err := h.svc.Bootstrap(r.Context(), currentUserID(r), req.InstanceURL, req.ClientID, req.ClientSecret, req.AppSlug)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"instance_url":     st.InstanceURL,
		"settings_version": st.SettingsVersion,
		"client_id":        st.GitHubOAuthClientID,
		"configured":       st.Configured(),
	})
}

// bootstrapVerify runs the one GitHub App check named by ?check= (204 or GitHub's verdict) and stores nothing; setup pass only.
func (h *Handler) bootstrapVerify(w http.ResponseWriter, r *http.Request) {
	var req bootstrapRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if err := h.runBootstrapCheck(r.Context(), currentUserID(r), r.URL.Query().Get("check"), req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) runBootstrapCheck(ctx context.Context, callerID, check string, req bootstrapRequest) error {
	if check == "instance_url" {
		return h.svc.VerifyBootstrapInstanceURL(ctx, callerID, req.InstanceURL)
	}
	return h.svc.BootstrapVerify(ctx, callerID, check, req.ClientID, req.ClientSecret, req.AppSlug)
}

// devLogin mints a session for the fixed dev identity, redirects like callbackGET, skips GitHub; dev-build only.
func (h *Handler) devLogin(w http.ResponseWriter, r *http.Request) {
	token, err := h.svc.DevLogin(WithDevice(r.Context(), DeviceFromRequest(r)))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	http.Redirect(w, r, h.spaOrigin(r)+"/login?token="+token, http.StatusFound)
}

// spaOrigin is where the browser lands after login: the split-origin dev override, else the instance URL, else the
// request's own host. r.TLS is nil behind a TLS-terminating proxy, so the instance URL is what keeps https intact.
func (h *Handler) spaOrigin(r *http.Request) string {
	if h.svc.cfg.SPAOrigin != "" {
		return h.svc.cfg.SPAOrigin
	}
	if base := h.svc.InstanceURL(r.Context()); base != "" {
		return strings.TrimSuffix(base, "/")
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

// ProtectedRoutes returns the authenticated auth endpoints, mounted behind RequireAuth at /api/auth.
func (h *Handler) ProtectedRoutes() http.Handler {
	mux := httpx.NewServeMux()
	mux.HandleFunc("GET /api/auth/me", h.me)
	mux.HandleFunc("PUT /api/auth/profile", h.updateProfile)
	mux.HandleFunc("POST /api/auth/onboarding/owner", h.completeOwnerWizard)
	mux.HandleFunc("POST /api/auth/onboarding/profile", h.completeFirstLogin)
	mux.HandleFunc("GET /api/auth/settings", h.getSettings)
	mux.HandleFunc("PUT /api/auth/settings", h.updateSettings)
	mux.HandleFunc("PUT /api/auth/settings/oauth/{provider}", h.updateProviderOAuth)
	mux.HandleFunc("POST /api/auth/connection-token", h.connectionToken)
	mux.HandleFunc("GET /api/auth/tokens", h.listPATs)
	mux.HandleFunc("POST /api/auth/tokens", h.mintPAT)
	mux.HandleFunc("DELETE /api/auth/tokens/{id}", h.revokePAT)
	mux.HandleFunc("GET /api/auth/sessions", h.listSessions)
	mux.HandleFunc("POST /api/auth/connect-codes", h.issueConnectCode)
	mux.HandleFunc("DELETE /api/auth/sessions/current", h.signOutCurrentSession)
	mux.HandleFunc("PUT /api/auth/sessions/current/push-token", h.setPushToken)
	mux.HandleFunc("DELETE /api/auth/sessions/others", h.signOutOtherSessions)
	mux.HandleFunc("DELETE /api/auth/sessions/{id}", h.signOutSession)
	mux.HandleFunc("GET /api/auth/identities", h.listIdentities)
	mux.HandleFunc("POST /api/auth/identities/link", h.startIdentityLink)
	mux.HandleFunc("DELETE /api/auth/identities/{provider}", h.unlinkIdentity)
	mux.HandleFunc("GET /api/auth/members", h.listMembers)
	mux.HandleFunc("GET /api/auth/members/lookup", h.lookupMembers)
	mux.HandleFunc("POST /api/auth/members", h.addMember)
	mux.HandleFunc("DELETE /api/auth/members/{login}", h.removeMember)
	mux.HandleFunc("GET /api/auth/accounts", h.listAccounts)
	mux.HandleFunc("POST /api/auth/accounts/{id}/disable", h.disableAccount)
	mux.HandleFunc("POST /api/auth/accounts/{id}/reactivate", h.reactivateAccount)
	mux.HandleFunc("PATCH /api/auth/accounts/{id}", h.updateAccountStatus)
	mux.HandleFunc("DELETE /api/auth/accounts/{id}", h.removeAccount)
	mux.HandleFunc("POST /api/auth/accounts/{id}/restore", h.restoreAccount)
	return mux
}

// startOAuth mints a CSRF state, stores it in a cookie, and redirects to the authorize endpoint.
func (h *Handler) startOAuth(provider Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		configured, err := h.providerConfigured(r, provider)
		if err != nil {
			httpx.WriteError(w, err)
			return
		}
		if !configured {
			httpx.WriteJSON(w, http.StatusServiceUnavailable, httpx.Envelope{
				Message: string(provider) + " OAuth is not configured (complete the instance bootstrap/settings step)",
				Code:    "NOT_CONFIGURED",
			})
			return
		}
		state, err := h.svc.NewState()
		if err != nil {
			httpx.WriteError(w, err)
			return
		}
		url, err := h.svc.AuthorizeURLFor(r.Context(), provider, state)
		if err != nil {
			httpx.WriteError(w, err)
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:     stateCookie,
			Value:    state,
			Path:     "/",
			MaxAge:   int(stateMaxAge.Seconds()),
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Secure:   h.svc.secureCookie(r.Context(), r.TLS != nil),
		})
		http.Redirect(w, r, url, http.StatusFound)
	}
}

type invitationOAuthRequest struct {
	Provider Provider `json:"provider"`
	Token    string   `json:"token"`
}

func (h *Handler) startInvitationOAuth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	var req invitationOAuthRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	start, err := h.svc.StartInvitationOAuth(ctx, req.Provider, req.Token)
	if err != nil {
		httpx.WriteError(w, classifyInvitationError(err))
		return
	}
	http.SetCookie(w, &http.Cookie{Name: start.CookieName, Value: start.State, Path: "/", MaxAge: int(stateMaxAge.Seconds()), HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: h.svc.secureCookie(ctx, r.TLS != nil)})
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"url": start.URL})
}

type invitationRedeemRequest struct {
	AcceptanceToken string `json:"acceptance_token"`
}

func (h *Handler) redeemInvitation(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	var req invitationRedeemRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	result, err := h.svc.RedeemInvitation(WithDevice(ctx, DeviceFromRequest(r)), req.AcceptanceToken)
	if err != nil {
		httpx.WriteError(w, classifyInvitationError(err))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, result)
}

type invitationAcceptanceRequest struct {
	Token           string `json:"token"`
	AcceptanceToken string `json:"acceptance_token"`
}

func (h *Handler) acceptInvitation(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	var req invitationAcceptanceRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if (req.Token == "" && req.AcceptanceToken == "") || (req.Token != "" && req.AcceptanceToken != "") {
		httpx.WriteError(w, invalidInvitationError())
		return
	}
	var details *InvitationAcceptance
	var err error
	if req.Token != "" {
		rawAuth := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		user, _, authErr := h.svc.authenticate(r.WithContext(ctx), rawAuth)
		if authErr != nil || user == nil {
			httpx.WriteError(w, apperrs.ErrUnauthorized)
			return
		}
		details, err = h.svc.PrepareAuthenticatedAcceptance(ctx, user.ID, req.Token)
	}
	if req.AcceptanceToken != "" {
		details, err = h.svc.GetInvitationAcceptance(ctx, req.AcceptanceToken)
	}
	if err != nil {
		httpx.WriteError(w, classifyInvitationError(err))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, details)
}

func (h *Handler) providerConfigured(r *http.Request, provider Provider) (bool, error) {
	return h.svc.ProviderConfigured(r.Context(), provider)
}

// callbackGET verifies the CSRF cookie, exchanges the code, and redirects to the SPA with a session token in the URL.
// stateless answers a callback this instance never started. GitHub sends one after an App installation when
// "Request user authorization during installation" is on (docs/guide/github-app); its code is not a sign-in this
// instance asked for, so it is never exchanged, and the browser goes back into the app instead of to an error.
func (h *Handler) stateless(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("installation_id") != "" {
		http.Redirect(w, r, h.spaOrigin(r)+"/settings/connectors", http.StatusFound)
		return
	}
	http.Redirect(w, r, h.spaOrigin(r)+"/login", http.StatusFound)
}

// installationCallback claims the installation an install link led to; a failure leaves it unassigned.
func (h *Handler) installationCallback(w http.ResponseWriter, r *http.Request, claims InstallationClaimer) {
	q := r.URL.Query()
	landing, err := claims.ClaimInstallation(r.Context(), q.Get("state"), q.Get("code"), q.Get("installation_id"))
	if err != nil {
		logging.FromCtx(r.Context()).Warn("github installation left unassigned", "installation_id", q.Get("installation_id"), "error", err)
		landing = "/"
	}
	http.Redirect(w, r, h.spaOrigin(r)+landing, http.StatusFound)
}

func (h *Handler) callbackGET(provider Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		r = r.WithContext(ctx)
		state := r.URL.Query().Get("state")
		if state == "" {
			h.stateless(w, r)
			return
		}
		if claims := h.svc.cfg.Installations; claims != nil && r.URL.Query().Get("installation_id") != "" && claims.ClaimsState(state) {
			h.installationCallback(w, r, claims)
			return
		}
		if cookie, cookieErr := r.Cookie(invitationStateCookie(hashCredential(state))); cookieErr == nil && cookie.Value == state {
			h.invitationCallback(w, r, provider, state)
			return
		}
		cookie, err := r.Cookie(stateCookie)
		if err != nil || cookie.Value == "" || cookie.Value != state {
			h.signInFailed(w, r, provider, signInStateMismatch, errors.New("oauth state mismatch"))
			return
		}
		http.SetCookie(w, &http.Cookie{Name: stateCookie, Value: "", MaxAge: -1, Path: "/", HttpOnly: true, Secure: h.svc.secureCookie(ctx, r.TLS != nil)})
		code := r.URL.Query().Get("code")
		if IsLinkState(state) {
			h.identityLinkCallback(w, r, provider, state, code)
			return
		}
		if refusal := r.URL.Query().Get("error"); refusal != "" {
			h.signInFailed(w, r, provider, providerRefusalCode(refusal), fmt.Errorf("provider returned error %q", refusal))
			return
		}
		if code == "" {
			h.signInFailed(w, r, provider, signInFailedCode, errors.New("callback has no code"))
			return
		}
		token, err := h.svc.LoginWith(WithDevice(ctx, DeviceFromRequest(r)), provider, code)
		if err != nil {
			h.signInFailed(w, r, provider, signInErrorCode(err), err)
			return
		}
		http.Redirect(w, r, h.spaOrigin(r)+"/login?token="+token, http.StatusFound)
	}
}

// Reason codes a failed browser sign-in carries back to /login as ?error=; the web sign-in page owns their copy.
const (
	signInInvitationRequired = "invitation_required"
	signInInvitationInvalid  = "invitation_invalid"
	signInAccountDisabled    = "account_disabled"
	signInAccountRemoved     = "account_removed"
	signInAccessDenied       = "access_denied"
	signInProviderError      = "provider_error"
	signInStateMismatch      = "state_mismatch"
	signInNotConfigured      = "not_configured"
	signInFailedCode         = "sign_in_failed"
)

func signInErrorCode(err error) string {
	if errors.Is(err, errInvitationRequired) {
		return signInInvitationRequired
	}
	if errors.Is(err, errAccountDisabled) {
		return signInAccountDisabled
	}
	if errors.Is(err, errAccountRemoved) {
		return signInAccountRemoved
	}
	if errors.Is(err, errOAuthNotConfigured) {
		return signInNotConfigured
	}
	return signInFailedCode
}

func providerRefusalCode(refusal string) string {
	if refusal == "access_denied" {
		return signInAccessDenied
	}
	return signInProviderError
}

// signInFailed answers a browser-navigated callback with a redirect to the sign-in page, never JSON; the real reason is logged, not sent.
func (h *Handler) signInFailed(w http.ResponseWriter, r *http.Request, provider Provider, code string, err error) {
	logging.FromCtx(r.Context()).Warn("sign-in failed", "provider", provider, "reason", code, "error", err)
	http.Redirect(w, r, h.spaOrigin(r)+"/login?error="+code, http.StatusFound)
}

func (h *Handler) invitationCallback(w http.ResponseWriter, r *http.Request, provider Provider, state string) {
	if refusal := r.URL.Query().Get("error"); refusal != "" {
		h.signInFailed(w, r, provider, providerRefusalCode(refusal), fmt.Errorf("provider returned error %q", refusal))
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		h.signInFailed(w, r, provider, signInInvitationInvalid, errors.New("callback has no code"))
		return
	}
	acceptance, err := h.svc.CompleteInvitationOAuth(r.Context(), provider, state, code)
	if err != nil {
		reason := signInErrorCode(err)
		if errors.Is(classifyInvitationError(err), apperrs.ErrNotFound) {
			reason = signInInvitationInvalid
		}
		h.signInFailed(w, r, provider, reason, err)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: invitationStateCookie(hashCredential(state)), Value: "", MaxAge: -1, Path: "/", HttpOnly: true, Secure: h.svc.secureCookie(r.Context(), r.TLS != nil)})
	// The invite page reads an unlabeled fragment as the invitation link itself, not the handoff.
	http.Redirect(w, r, h.spaOrigin(r)+"/invite#acceptance-token="+acceptance, http.StatusFound)
}

// identityLinkCallback lands back on Profile either way: the person is signed in, so an error is a toast there.
func (h *Handler) identityLinkCallback(w http.ResponseWriter, r *http.Request, provider Provider, state, code string) {
	target := h.spaOrigin(r) + "/settings/profile"
	if err := h.svc.CompleteIdentityLink(r.Context(), provider, state, code); err != nil {
		logging.FromCtx(r.Context()).Warn("link identity", "provider", provider, "error", err)
		http.Redirect(w, r, target+"?provider="+string(provider)+"&error="+url.QueryEscape(err.Error()), http.StatusFound)
		return
	}
	http.Redirect(w, r, target+"?provider="+string(provider)+"&linked=1", http.StatusFound)
}

// callbackPOST exchanges a code for a token, returning JSON for programmatic clients; the browser uses the GET flow.
func (h *Handler) callbackPOST(w http.ResponseWriter, r *http.Request) {
	var req callbackRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if req.Code == "" {
		httpx.WriteError(w, fmt.Errorf("%w: code is required", apperrs.ErrInvalid))
		return
	}
	token, err := h.svc.Login(WithDevice(r.Context(), DeviceFromRequest(r)), req.Code)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"token": token})
}

// me returns the current user and the onboarding wizard (if any) that applies.
func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	st, err := h.svc.Me(r.Context(), currentUserID(r))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, st)
}

type profileOverrideRequest struct {
	DisplayName       string `json:"display_name"`
	AvatarOverrideURL string `json:"avatar_override_url"`
}

// updateProfile sets the caller's display/avatar override; userID is from context, never the body.
func (h *Handler) updateProfile(w http.ResponseWriter, r *http.Request) {
	var req profileOverrideRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	user, err := h.svc.UpdateProfileOverride(r.Context(), currentUserID(r), req.DisplayName, req.AvatarOverrideURL)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, user)
}

type ownerWizardRequest struct {
	InstanceURL string `json:"instance_url"`
}

func (h *Handler) completeOwnerWizard(w http.ResponseWriter, r *http.Request) {
	var req ownerWizardRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if err := h.svc.CompleteOwnerWizard(r.Context(), currentUserID(r), req.InstanceURL); err != nil {
		httpx.WriteError(w, err)
		return
	}
	st, err := h.svc.Me(r.Context(), currentUserID(r))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, st)
}

func (h *Handler) completeFirstLogin(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.CompleteFirstLogin(r.Context(), currentUserID(r)); err != nil {
		httpx.WriteError(w, err)
		return
	}
	st, err := h.svc.Me(r.Context(), currentUserID(r))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, st)
}

func (h *Handler) getSettings(w http.ResponseWriter, r *http.Request) {
	st, err := h.svc.cfg.Settings.Get(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"instance_url":     st.InstanceURL,
		"settings_version": st.SettingsVersion,
		"oauth_callback":   oauthCallbackFor(st.InstanceURL),
		// Google/Discord sign-in are optional (ADR 0040): the client ID is public, only the secret's presence is reported.
		"google_oauth_client_id":   st.GoogleOAuthClientID,
		"google_oauth_callback":    callbackFor(st.InstanceURL, ProviderGoogle),
		"google_oauth_configured":  st.ProviderConfigured(ProviderGoogle),
		"discord_oauth_client_id":  st.DiscordOAuthClientID,
		"discord_oauth_callback":   callbackFor(st.InstanceURL, ProviderDiscord),
		"discord_oauth_configured": st.ProviderConfigured(ProviderDiscord),
		// mcp_url reuses ConnectionToken's derivation so settings render a copy-paste config without minting a token.
		"mcp_url": mcpURLFor(st.InstanceURL),
	})
}

type updateSettingsRequest struct {
	InstanceURL string `json:"instance_url"`
}

func (h *Handler) updateSettings(w http.ResponseWriter, r *http.Request) {
	var req updateSettingsRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	st, err := h.svc.UpdateInstanceURL(r.Context(), currentUserID(r), req.InstanceURL)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"instance_url":     st.InstanceURL,
		"settings_version": st.SettingsVersion,
	})
}

type providerOAuthRequest struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

// updateProviderOAuth stores or clears sign-in credentials (instance:write); like bootstrap, never echoes the raw secret.
func (h *Handler) updateProviderOAuth(w http.ResponseWriter, r *http.Request) {
	var req providerOAuthRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	provider := Provider(r.PathValue("provider"))
	st, err := h.svc.SetProviderOAuth(r.Context(), currentUserID(r), provider, req.ClientID, req.ClientSecret)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	clientID, _ := st.OAuthCredentials(provider)
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"provider":   provider,
		"client_id":  clientID,
		"callback":   callbackFor(st.InstanceURL, provider),
		"configured": st.ProviderConfigured(provider),
	})
}

func (h *Handler) connectionToken(w http.ResponseWriter, r *http.Request) {
	ct, err := h.svc.GenerateConnectionToken(r.Context(), currentUserID(r))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, ct)
}

type mintPATRequest struct {
	Name string `json:"name"`
}

// mintPAT returns the raw token exactly once, alongside its metadata; the raw value is never stored or listable again.
func (h *Handler) mintPAT(w http.ResponseWriter, r *http.Request) {
	var req mintPATRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	raw, pat, err := h.svc.MintPAT(r.Context(), currentUserID(r), req.Name)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"token":      raw,
		"id":         pat.ID,
		"name":       pat.Name,
		"prefix":     pat.Prefix,
		"created_at": pat.CreatedAt,
	})
}

func (h *Handler) listPATs(w http.ResponseWriter, r *http.Request) {
	tokens, err := h.svc.ListPATs(r.Context(), currentUserID(r))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"tokens": tokens})
}

func (h *Handler) revokePAT(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.RevokePAT(r.Context(), currentUserID(r), r.PathValue("id")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	tokens, err := h.svc.ListPATs(r.Context(), currentUserID(r))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"tokens": tokens})
}

// requireSession is the gate on the session routes: a personal access token has no device to list or sign out.
func requireSession(r *http.Request) (*Session, error) {
	ses := SessionFromCtx(r.Context())
	if ses == nil {
		return nil, fmt.Errorf("%w: a signed-in device is required", apperrs.ErrForbidden)
	}
	return ses, nil
}

func (h *Handler) listSessions(w http.ResponseWriter, r *http.Request) {
	ses, err := requireSession(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	sessions, err := h.svc.ListSessions(r.Context(), currentUserID(r), ses.ID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"sessions": sessions})
}

func (h *Handler) signOutSession(w http.ResponseWriter, r *http.Request) {
	if _, err := requireSession(r); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if err := h.svc.SignOutSession(r.Context(), currentUserID(r), r.PathValue("id")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) signOutCurrentSession(w http.ResponseWriter, r *http.Request) {
	ses, err := requireSession(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	if err := h.svc.SignOutSession(r.Context(), currentUserID(r), ses.ID); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type pushTokenRequest struct {
	PushToken string `json:"push_token"`
}

// setPushToken registers the calling phone's Expo token on its own session; an empty token clears it.
func (h *Handler) setPushToken(w http.ResponseWriter, r *http.Request) {
	ses, err := requireSession(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	var req pushTokenRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if err := h.svc.SetPushToken(r.Context(), currentUserID(r), ses.ID, req.PushToken); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) signOutOtherSessions(w http.ResponseWriter, r *http.Request) {
	ses, err := requireSession(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	if err := h.svc.SignOutOtherSessions(r.Context(), currentUserID(r), ses.ID); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listIdentities(w http.ResponseWriter, r *http.Request) {
	if _, err := requireSession(r); err != nil {
		httpx.WriteError(w, err)
		return
	}
	identities, err := h.svc.ListIdentities(r.Context(), currentUserID(r))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"identities": identities})
}

type identityLinkRequest struct {
	Provider Provider `json:"provider"`
}

// startIdentityLink answers the authorize URL for the browser to visit; a top-level redirect could not carry the bearer.
func (h *Handler) startIdentityLink(w http.ResponseWriter, r *http.Request) {
	if _, err := requireSession(r); err != nil {
		httpx.WriteError(w, err)
		return
	}
	var req identityLinkRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	authorizeURL, state, err := h.svc.StartIdentityLink(r.Context(), currentUserID(r), req.Provider)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: stateCookie, Value: state, Path: "/", MaxAge: int(stateMaxAge.Seconds()),
		HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: h.svc.secureCookie(r.Context(), r.TLS != nil),
	})
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"url": authorizeURL})
}

func (h *Handler) unlinkIdentity(w http.ResponseWriter, r *http.Request) {
	if _, err := requireSession(r); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if err := h.svc.UnlinkIdentity(r.Context(), currentUserID(r), Provider(r.PathValue("provider"))); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// issueConnectCode needs a signed-in device, so no agent holding a personal access token can sign a phone in.
func (h *Handler) issueConnectCode(w http.ResponseWriter, r *http.Request) {
	if _, err := requireSession(r); err != nil {
		httpx.WriteError(w, err)
		return
	}
	code, err := h.svc.IssueConnectCode(r.Context(), currentUserID(r), h.apiOrigin(r))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, code)
}

// apiOrigin is the address a phone dials: the instance URL, else the host this request reached; never the SPA origin.
func (h *Handler) apiOrigin(r *http.Request) string {
	if base := h.svc.InstanceURL(r.Context()); base != "" {
		return strings.TrimSuffix(base, "/")
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

type exchangeConnectCodeRequest struct {
	Code   string        `json:"code"`
	Device ConnectDevice `json:"device"`
}

// exchangeConnectCode is public: the phone has no session yet, so the code is the whole proof, throttled per address.
func (h *Handler) exchangeConnectCode(w http.ResponseWriter, r *http.Request) {
	var req exchangeConnectCodeRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	token, err := h.svc.ExchangeConnectCode(r.Context(), httpx.ClientAddr(r), req.Code, req.Device)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"token": token, "server_version": version.Version})
}

func (h *Handler) listMembers(w http.ResponseWriter, r *http.Request) {
	members, err := h.svc.ListMembers(r.Context(), currentUserID(r))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string][]string{"members": members})
}

func (h *Handler) lookupMembers(w http.ResponseWriter, r *http.Request) {
	matches, err := h.svc.LookupMembers(r.Context(), currentUserID(r), r.URL.Query().Get("q"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string][]LoginMatch{"matches": matches})
}

type memberRequest struct {
	Login string `json:"login"`
}

func (h *Handler) addMember(w http.ResponseWriter, r *http.Request) {
	var req memberRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if err := h.svc.AddMember(r.Context(), currentUserID(r), req.Login); err != nil {
		httpx.WriteError(w, err)
		return
	}
	members, err := h.svc.ListMembers(r.Context(), currentUserID(r))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string][]string{"members": members})
}

func (h *Handler) removeMember(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.RemoveMember(r.Context(), currentUserID(r), r.PathValue("login")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	members, err := h.svc.ListMembers(r.Context(), currentUserID(r))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string][]string{"members": members})
}

func (h *Handler) listAccounts(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	accounts, err := h.svc.ListAccounts(ctx, currentUserID(r))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	out := make([]accountResponse, 0, len(accounts))
	for _, account := range accounts {
		out = append(out, accountResponse{
			ID: account.ID, Login: account.Login, Name: account.Name,
			AvatarURL: account.AvatarURL, Status: account.AccountStatus, CreatedAt: account.CreatedAt,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"accounts": out})
}

type accountResponse struct {
	ID        string        `json:"id"`
	Login     string        `json:"login"`
	Name      string        `json:"name"`
	AvatarURL string        `json:"avatar_url"`
	Status    AccountStatus `json:"status"`
	CreatedAt time.Time     `json:"created_at"`
}

type updateAccountStatusRequest struct {
	Status AccountStatus `json:"status"`
}

func (h *Handler) updateAccountStatus(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	var req updateAccountStatusRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if err := h.svc.UpdateAccountStatus(ctx, currentUserID(r), r.PathValue("id"), req.Status); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) disableAccount(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	if err := h.svc.DisableAccount(ctx, currentUserID(r), r.PathValue("id")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) reactivateAccount(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	if err := h.svc.ReactivateAccount(ctx, currentUserID(r), r.PathValue("id")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) removeAccount(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	if err := h.svc.RemoveAccount(ctx, currentUserID(r), r.PathValue("id")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) restoreAccount(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	if err := h.svc.RestoreAccount(ctx, currentUserID(r), r.PathValue("id")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func currentUserID(r *http.Request) string {
	if u := UserFromCtx(r.Context()); u != nil {
		return u.ID
	}
	return ""
}

// oauthCallbackFor derives the OAuth callback URL from the instance URL; GitHub needs an exact match to its callback.
func oauthCallbackFor(instanceURL string) string {
	return strings.TrimSuffix(instanceURL, "/") + "/auth/callback"
}
