package pairing

import (
	"errors"
	"net/http"

	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/platform/httpx"
	"github.com/otal-labs/nexul/internal/platform/identity"
)

// Handler adapts the pairing use-cases to the HTTP/JSON gateway (ADR 0019).
type Handler struct {
	svc *Service
}

// NewHandler wires the pairing REST gateway over the given service.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Routes returns the pairing REST endpoints, mounted behind RequireAuth at /api/pairing.
func (h *Handler) Routes() http.Handler {
	mux := httpx.NewServeMux()
	mux.HandleFunc("GET /api/pairing/computers", h.listComputers)
	mux.HandleFunc("POST /api/pairing/computers", h.pair)
	mux.HandleFunc("POST /api/pairing/computers/tunnel", h.createTunnel)
	mux.HandleFunc("GET /api/pairing/computers/{id}/tunnel/status", h.tunnelStatus)
	mux.HandleFunc("GET /api/pairing/computers/{id}/tunnel/token", h.tunnelToken)
	mux.HandleFunc("POST /api/pairing/computers/{id}/pair", h.pairComputer)
	mux.HandleFunc("POST /api/pairing/computers/{id}/repair", h.repair)
	mux.HandleFunc("DELETE /api/pairing/computers/{id}", h.deleteComputer)
	mux.HandleFunc("GET /api/pairing/computers/{id}/mcp-token", h.getMCPToken)
	mux.HandleFunc("POST /api/pairing/computers/{id}/mcp-token", h.mintMCPToken)
	mux.HandleFunc("DELETE /api/pairing/computers/{id}/mcp-token", h.revokeMCPToken)
	mux.HandleFunc("GET /api/pairing/computers/{id}/projects", h.listProjects)
	mux.HandleFunc("GET /api/pairing/computers/{id}/providers", h.listProviders)
	// Read-only on purpose: a setup confirmation is written only through MCP (ADR 0063).
	mux.HandleFunc("GET /api/pairing/computers/{id}/setup", h.getSetup)
	mux.HandleFunc("GET /api/pairing/resolve", h.resolve)
	mux.HandleFunc("GET /api/pairing/defaults", h.getDefaults)
	mux.HandleFunc("PUT /api/pairing/defaults", h.setDefaults)
	mux.HandleFunc("GET /api/pairing/projects/{id}", h.getProjectLink)
	mux.HandleFunc("PUT /api/pairing/projects/{id}", h.setProjectLink)
	mux.HandleFunc("DELETE /api/pairing/projects/{id}", h.clearProjectLink)
	return mux
}

func (h *Handler) listComputers(w http.ResponseWriter, r *http.Request) {
	cs, err := h.svc.ListComputers(r.Context(), actorID(r))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"computers": cs})
}

func (h *Handler) listProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := h.svc.ListProjects(r.Context(), actorID(r), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	if projects == nil {
		projects = []harness.Project{}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"projects": projects})
}

func (h *Handler) listProviders(w http.ResponseWriter, r *http.Request) {
	providers, err := h.svc.ListProviders(r.Context(), actorID(r), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"providers": providers})
}

func (h *Handler) getSetup(w http.ResponseWriter, r *http.Request) {
	setup, err := h.svc.GetSetup(r.Context(), actorID(r), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, setup)
}

type pairRequest struct {
	// Kind defaults to T3 Code: the pair form predates the field, and no other harness ships yet.
	Kind      harness.Kind `json:"kind"`
	Name      string       `json:"name"`
	ServerURL string       `json:"server_url"`
	Token     string       `json:"token"`
}

func (h *Handler) pair(w http.ResponseWriter, r *http.Request) {
	var req pairRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if req.Kind == "" {
		req.Kind = harness.KindT3Code
	}
	c, err := h.svc.Pair(r.Context(), actorID(r), req.Kind, req.Name, req.ServerURL, req.Token)
	if err != nil {
		writePairError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, c)
}

type pairComputerRequest struct {
	Token string `json:"token"`
}

// pairComputer pairs an existing computer at its own address, which for a computer tunnel is its hostname.
func (h *Handler) pairComputer(w http.ResponseWriter, r *http.Request) {
	var req pairComputerRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	c, err := h.svc.PairComputer(r.Context(), actorID(r), r.PathValue("id"), req.Token)
	if err != nil {
		writePairError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, c)
}

// writePairError keys a failure under the input that caused it, so the pairing form shows it on that field.
func writePairError(w http.ResponseWriter, err error) {
	var fe *FieldError
	if errors.As(err, &fe) {
		httpx.WriteFieldError(w, err, fe.Field)
		return
	}
	httpx.WriteError(w, err)
}

type createTunnelRequest struct {
	Kind harness.Kind `json:"kind"`
	Name string       `json:"name"`
	Port int          `json:"port"`
}

func (h *Handler) createTunnel(w http.ResponseWriter, r *http.Request) {
	var req createTunnelRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if req.Kind == "" {
		req.Kind = harness.KindT3Code
	}
	if req.Port == 0 {
		req.Port = DefaultT3CodePort
	}
	c, err := h.svc.CreateComputerTunnel(r.Context(), actorID(r), req.Kind, req.Name, req.Port)
	if err != nil {
		writeTunnelError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, c)
}

func (h *Handler) tunnelStatus(w http.ResponseWriter, r *http.Request) {
	status, err := h.svc.ComputerTunnelStatus(r.Context(), actorID(r), r.PathValue("id"))
	if err != nil {
		writeTunnelError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, status)
}

func (h *Handler) tunnelToken(w http.ResponseWriter, r *http.Request) {
	token, err := h.svc.ComputerTunnelToken(r.Context(), actorID(r), r.PathValue("id"))
	if err != nil {
		writeTunnelError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"token": token})
}

// writeTunnelError adds the missing prerequisite's reason to the error envelope, so the dialog can show its fix.
func writeTunnelError(w http.ResponseWriter, err error) {
	var pe *PrerequisiteError
	if errors.As(err, &pe) {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"message": pe.Error(), "code": "INVALID", "reason": string(pe.Reason)})
		return
	}
	httpx.WriteError(w, err)
}

func (h *Handler) repair(w http.ResponseWriter, r *http.Request) {
	var req pairRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	c, err := h.svc.Repair(r.Context(), actorID(r), r.PathValue("id"), req.Name, req.ServerURL, req.Token)
	if err != nil {
		writePairError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, c)
}

func (h *Handler) deleteComputer(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteComputer(r.Context(), actorID(r), r.PathValue("id")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) getMCPToken(w http.ResponseWriter, r *http.Request) {
	token, err := h.svc.GetMCPToken(r.Context(), actorID(r), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"mcp_token": token})
}

// mintMCPToken is the one response that carries the raw token; nothing lists it again.
func (h *Handler) mintMCPToken(w http.ResponseWriter, r *http.Request) {
	minted, err := h.svc.MintMCPToken(r.Context(), actorID(r), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, minted)
}

func (h *Handler) revokeMCPToken(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.RevokeMCPToken(r.Context(), actorID(r), r.PathValue("id")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

// resolve answers whether the caller can run a play right now, without starting a turn or leaking the bearer token.
func (h *Handler) resolve(w http.ResponseWriter, r *http.Request) {
	target, err := h.svc.PreviewTarget(r.Context(), actorID(r), r.URL.Query().Get("project_id"))
	if err != nil {
		var nc *NotConfiguredError
		if errors.As(err, &nc) {
			httpx.WriteJSON(w, http.StatusOK, map[string]any{"ok": false, "reason": nc.Reason})
			return
		}
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"ok":                 true,
		"computer_id":        target.Computer.ID,
		"harness_project_id": target.HarnessProjectID,
		"provider":           target.Provider,
		"model":              target.Model,
	})
}

func (h *Handler) getDefaults(w http.ResponseWriter, r *http.Request) {
	d, err := h.svc.GetDefaults(r.Context(), actorID(r))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, d)
}

type defaultsRequest struct {
	DefaultComputerID string `json:"default_computer_id"`
	FallbackProjectID string `json:"fallback_project_id"`
	Provider          string `json:"provider"`
	Model             string `json:"model"`
}

func (h *Handler) setDefaults(w http.ResponseWriter, r *http.Request) {
	var req defaultsRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	d, err := h.svc.SetDefaults(r.Context(), actorID(r), Defaults{
		DefaultComputerID: req.DefaultComputerID,
		FallbackProjectID: req.FallbackProjectID,
		Provider:          req.Provider,
		Model:             req.Model,
	})
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, d)
}

func (h *Handler) getProjectLink(w http.ResponseWriter, r *http.Request) {
	link, err := h.svc.GetProjectLink(r.Context(), actorID(r), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, link)
}

type projectLinkRequest struct {
	ComputerID       string `json:"computer_id"`
	HarnessProjectID string `json:"harness_project_id"`
	Provider         string `json:"provider"`
	Model            string `json:"model"`
}

func (h *Handler) setProjectLink(w http.ResponseWriter, r *http.Request) {
	var req projectLinkRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	link, err := h.svc.SetProjectLink(r.Context(), actorID(r), r.PathValue("id"), ProjectLink{
		ComputerID:       req.ComputerID,
		HarnessProjectID: req.HarnessProjectID,
		Provider:         req.Provider,
		Model:            req.Model,
	})
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, link)
}

func (h *Handler) clearProjectLink(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.ClearProjectLink(r.Context(), actorID(r), r.PathValue("id")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "cleared"})
}

// actorID reads the identity injected by the composition root's middleware; pairing never imports auth (ADR 0017).
func actorID(r *http.Request) string {
	if a, ok := identity.ActorFromCtx(r.Context()); ok {
		return a.ID
	}
	return ""
}
