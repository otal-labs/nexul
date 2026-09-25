package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/httpx"
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
	if h.svc.DevLoginEnabled() {
		mux.HandleFunc("GET /auth/dev-login", h.devLogin)
	}
	return mux
}

// bootstrapStatus reports whether the GitHub OAuth App is configured (ADR 0040); public, side-effect free.
func (h *Handler) bootstrapStatus(w http.ResponseWriter, r *http.Request) {
	st, err := h.svc.cfg.Settings.Get(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	reconfigurable := false
	if st.Configured() {
		if reconfigurable, err = h.svc.BootstrapReconfigurable(r.Context()); err != nil {
			httpx.WriteError(w, err)
			return
		}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]bool{
		"configured":         st.Configured(),
		"reconfigurable":     reconfigurable,
		"google_configured":  st.ProviderConfigured(ProviderGoogle),
		"discord_configured": st.ProviderConfigured(ProviderDiscord),
	})
}

type bootstrapRequest struct {
	InstanceURL  string `json:"instance_url"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	// AppSlug is the GitHub App's URL slug, needed by connectors' install; collected here since Connect predates Settings.
	AppSlug string `json:"app_slug"`
}

// bootstrap stores the instance URL and GitHub App credentials on a fresh instance; public, since there's no user yet.
func (h *Handler) bootstrap(w http.ResponseWriter, r *http.Request) {
	var req bootstrapRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	st, err := h.svc.Bootstrap(r.Context(), req.InstanceURL, req.ClientID, req.ClientSecret, req.AppSlug)
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

// bootstrapVerify runs the one GitHub App check named by ?check= (204 or GitHub's verdict) and stores nothing; public like bootstrap.
func (h *Handler) bootstrapVerify(w http.ResponseWriter, r *http.Request) {
	var req bootstrapRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if err := h.runBootstrapCheck(r.Context(), r.URL.Query().Get("check"), req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) runBootstrapCheck(ctx context.Context, check string, req bootstrapRequest) error {
	if check == "instance_url" {
		return h.svc.VerifyInstanceURL(ctx, req.InstanceURL)
	}
	return h.svc.BootstrapVerify(ctx, check, req.ClientID, req.ClientSecret, req.AppSlug)
}

// devLogin mints a session for the fixed dev identity, redirects like callbackGET, skips GitHub; dev-build only.
func (h *Handler) devLogin(w http.ResponseWriter, r *http.Request) {
	token, err := h.svc.DevLogin(r.Context())
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
	mux.HandleFunc("PATCH /api/auth/settings/mention-chip-template", h.updateMentionChipTemplate)
	mux.HandleFunc("POST /api/auth/connection-token", h.connectionToken)
	mux.HandleFunc("GET /api/auth/tokens", h.listPATs)
	mux.HandleFunc("POST /api/auth/tokens", h.mintPAT)
	mux.HandleFunc("DELETE /api/auth/tokens/{id}", h.revokePAT)
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
	result, err := h.svc.RedeemInvitation(ctx, req.AcceptanceToken)
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
		user, authErr := h.svc.authenticate(r.WithContext(ctx), rawAuth)
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
func (h *Handler) callbackGET(provider Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		state := r.URL.Query().Get("state")
		if state != "" {
			if cookie, cookieErr := r.Cookie(invitationStateCookie(hashCredential(state))); cookieErr == nil && cookie.Value == state {
				code := r.URL.Query().Get("code")
				if code == "" {
					httpx.WriteError(w, invalidInvitationError())
					return
				}
				acceptance, err := h.svc.CompleteInvitationOAuth(ctx, provider, state, code)
				if err != nil {
					httpx.WriteError(w, classifyInvitationError(err))
					return
				}
				http.SetCookie(w, &http.Cookie{Name: invitationStateCookie(hashCredential(state)), Value: "", MaxAge: -1, Path: "/", HttpOnly: true, Secure: h.svc.secureCookie(ctx, r.TLS != nil)})
				http.Redirect(w, r, h.spaOrigin(r)+"/invite#"+acceptance, http.StatusFound)
				return
			}
		}
		cookie, err := r.Cookie(stateCookie)
		if err != nil || cookie.Value == "" || cookie.Value != state {
			httpx.WriteError(w, fmt.Errorf("%w: oauth state mismatch", apperrs.ErrUnauthorized))
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			httpx.WriteError(w, fmt.Errorf("%w: code is required", apperrs.ErrInvalid))
			return
		}
		token, err := h.svc.LoginWith(r.Context(), provider, code)
		if err != nil {
			httpx.WriteError(w, err)
			return
		}
		http.SetCookie(w, &http.Cookie{Name: stateCookie, Value: "", MaxAge: -1, Path: "/", HttpOnly: true, Secure: h.svc.secureCookie(ctx, r.TLS != nil)})
		http.Redirect(w, r, h.spaOrigin(r)+"/login?token="+token, http.StatusFound)
	}
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
	token, err := h.svc.Login(r.Context(), req.Code)
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
		"instance_url":          st.InstanceURL,
		"settings_version":      st.SettingsVersion,
		"oauth_callback":        oauthCallbackFor(st.InstanceURL),
		"mention_chip_template": st.MentionChipTemplate,
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

type updateMentionChipTemplateRequest struct {
	MentionChipTemplate string `json:"mention_chip_template"`
}

// updateMentionChipTemplate sets the mention chip layout; any user reads it, only workspaces:write can change it.
func (h *Handler) updateMentionChipTemplate(w http.ResponseWriter, r *http.Request) {
	var req updateMentionChipTemplateRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	st, err := h.svc.SetMentionChipTemplate(r.Context(), currentUserID(r), req.MentionChipTemplate)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"mention_chip_template": st.MentionChipTemplate,
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

// updateProviderOAuth stores or clears sign-in credentials (owner only); like bootstrap, never echoes the raw secret.
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
			ID: account.ID, Provider: account.Provider, Login: account.Login, Name: account.Name,
			AvatarURL: account.AvatarURL, Status: account.AccountStatus,
			CanCreateWorkspace: account.CanCreateWorkspace, CreatedAt: account.CreatedAt,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"accounts": out})
}

type accountResponse struct {
	ID                 string        `json:"id"`
	Provider           Provider      `json:"provider"`
	Login              string        `json:"login"`
	Name               string        `json:"name"`
	AvatarURL          string        `json:"avatar_url"`
	Status             AccountStatus `json:"status"`
	CanCreateWorkspace bool          `json:"can_create_workspace"`
	CreatedAt          time.Time     `json:"created_at"`
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
