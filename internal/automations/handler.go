package automations

import (
	"encoding/json"
	"net/http"

	"github.com/otal-labs/nexul/internal/platform/httpx"
	"github.com/otal-labs/nexul/internal/platform/identity"
)

// Handler adapts the automations use-cases to the HTTP gateway (ADR 0019); versions is nil until wired via WithVersions.
type Handler struct {
	svc      *Service
	versions *VersionsService
}

// NewHandler wires the automations REST gateway over the given service.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// WithVersions adds the version-history endpoints to this handler.
func (h *Handler) WithVersions(svc *VersionsService) *Handler {
	h.versions = svc
	return h
}

// Routes returns the automations REST endpoints (CRUD + enable/disable + token mint/revoke).
func (h *Handler) Routes() http.Handler {
	mux := httpx.NewServeMux()
	mux.HandleFunc("GET /api/automations", h.list)
	mux.HandleFunc("POST /api/automations", h.create)
	mux.HandleFunc("GET /api/automations/{id}", h.get)
	mux.HandleFunc("PATCH /api/automations/{id}/config", h.updateConfig)
	mux.HandleFunc("PATCH /api/automations/{id}/enabled", h.setEnabled)
	mux.HandleFunc("DELETE /api/automations/{id}", h.delete)
	mux.HandleFunc("POST /api/automations/{id}/token", h.mintToken)
	mux.HandleFunc("DELETE /api/automations/{id}/token", h.revokeToken)
	if h.versions != nil {
		h.versionRoutes(mux)
	}
	return mux
}

type createRequest struct {
	Name   string   `json:"name"`
	Scopes []string `json:"scopes"`
}

// tokenResponse wraps an automation alongside its raw token, returned once
// at mint time (create and rotate) — never again after this response.
type tokenResponse struct {
	Automation *Automation `json:"automation"`
	Token      string      `json:"token"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	a, token, err := h.svc.Create(r.Context(), actorID(r), req.Name, req.Scopes)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, tokenResponse{Automation: a, Token: token})
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.List(r.Context(), actorID(r))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, list)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	a, err := h.svc.Get(r.Context(), actorID(r), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, a)
}

type updateConfigRequest struct {
	ConfigValues json.RawMessage `json:"config_values"`
}

func (h *Handler) updateConfig(w http.ResponseWriter, r *http.Request) {
	var req updateConfigRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	a, err := h.svc.UpdateConfigValues(r.Context(), actorID(r), r.PathValue("id"), req.ConfigValues)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, a)
}

type setEnabledRequest struct {
	Enabled bool `json:"enabled"`
}

func (h *Handler) setEnabled(w http.ResponseWriter, r *http.Request) {
	var req setEnabledRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	a, err := h.svc.SetEnabled(r.Context(), actorID(r), r.PathValue("id"), req.Enabled)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, a)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), actorID(r), r.PathValue("id")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) mintToken(w http.ResponseWriter, r *http.Request) {
	a, token, err := h.svc.MintToken(r.Context(), actorID(r), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, tokenResponse{Automation: a, Token: token})
}

func (h *Handler) revokeToken(w http.ResponseWriter, r *http.Request) {
	a, err := h.svc.RevokeToken(r.Context(), actorID(r), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, a)
}

func actorID(r *http.Request) string {
	if a, ok := identity.ActorFromCtx(r.Context()); ok {
		return a.ID
	}
	return ""
}
