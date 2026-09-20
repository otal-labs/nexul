package mentions

import (
	"net/http"
	"strconv"

	"github.com/otal-labs/nexul/internal/platform/httpx"
)

// Handler backs both the browser's @ picker/chips and MCP's ops for LLMs.
type Handler struct {
	svc *Service
}

// NewHandler wires the mentions REST gateway over the given service.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Routes returns the mentions REST endpoints.
func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/mentions/search", h.search)
	mux.HandleFunc("POST /api/mentions/resolve", h.resolve)
	return mux
}

type resolveRequest struct {
	Refs []Ref `json:"refs"`
}

// resolve renders one batch of references as chips (one request per document render).
func (h *Handler) resolve(w http.ResponseWriter, r *http.Request) {
	var req resolveRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	chips, err := h.svc.Resolve(r.Context(), req.Refs)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"chips": chips})
}

// search returns the autocomplete entries for the @ picker.
func (h *Handler) search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	results, err := h.svc.Search(r.Context(), q, limit)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"results": results})
}
