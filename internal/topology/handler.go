package topology

import (
	"net/http"

	"github.com/otal-labs/nexul/internal/platform/httpx"
)

// Handler adapts the topology use-cases to the HTTP/JSON gateway (ADR 0019); the browser never talks to MCP.
type Handler struct {
	svc *Service
}

// NewHandler wires the topology REST gateway over the given service.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Routes returns the topology REST endpoints; environment defaults to the canonical one when absent.
func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/topology", h.get)
	mux.HandleFunc("PUT /api/topology", h.update)
	mux.HandleFunc("POST /api/topology/nodes", h.addNode)
	mux.HandleFunc("DELETE /api/topology/nodes/{id}", h.removeNode)
	mux.HandleFunc("POST /api/topology/edges", h.addEdge)
	mux.HandleFunc("DELETE /api/topology/edges/{id}", h.removeEdge)
	return mux
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	c, err := h.svc.Get(r.Context(), environmentParam(r))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, c)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	var req Canvas
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	c, err := h.svc.Update(r.Context(), environmentParam(r), &req)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, c)
}

func (h *Handler) addNode(w http.ResponseWriter, r *http.Request) {
	var req Node
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	c, err := h.svc.AddNode(r.Context(), environmentParam(r), req)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, c)
}

func (h *Handler) removeNode(w http.ResponseWriter, r *http.Request) {
	c, err := h.svc.RemoveNode(r.Context(), environmentParam(r), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, c)
}

func (h *Handler) addEdge(w http.ResponseWriter, r *http.Request) {
	var req Edge
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	c, err := h.svc.AddEdge(r.Context(), environmentParam(r), req)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, c)
}

func (h *Handler) removeEdge(w http.ResponseWriter, r *http.Request) {
	c, err := h.svc.RemoveEdge(r.Context(), environmentParam(r), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, c)
}

func environmentParam(r *http.Request) string {
	if env := r.URL.Query().Get("environment"); env != "" {
		return env
	}
	return DefaultEnvironment
}
