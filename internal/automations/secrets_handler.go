package automations

import (
	"net/http"

	"github.com/otal-labs/nexul/internal/platform/httpx"
)

// SecretsHandler adapts the secrets use-cases, mounted separately since secrets aren't scoped to one automation.
type SecretsHandler struct {
	svc *SecretsService
}

// NewSecretsHandler wires the secrets REST gateway over the given service.
func NewSecretsHandler(svc *SecretsService) *SecretsHandler {
	return &SecretsHandler{svc: svc}
}

// Routes returns the workspace secrets REST endpoints (list/set/delete by name).
func (h *SecretsHandler) Routes() http.Handler {
	mux := httpx.NewServeMux()
	mux.HandleFunc("GET /api/automation-secrets", h.list)
	mux.HandleFunc("PUT /api/automation-secrets/{name}", h.set)
	mux.HandleFunc("DELETE /api/automation-secrets/{name}", h.delete)
	return mux
}

type setSecretRequest struct {
	Value string `json:"value"`
}

func (h *SecretsHandler) list(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.List(r.Context(), actorID(r))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, list)
}

func (h *SecretsHandler) set(w http.ResponseWriter, r *http.Request) {
	var req setSecretRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if err := h.svc.Set(r.Context(), actorID(r), r.PathValue("name"), req.Value); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *SecretsHandler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), actorID(r), r.PathValue("name")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
