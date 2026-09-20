package repository

import (
	"net/http"

	"github.com/otal-labs/nexul/internal/platform/httpx"
)

// Handler adapts the repository use-cases to the HTTP/JSON gateway (ADR 0019).
type Handler struct {
	s Scanner
}

// NewHandler wires the repository REST gateway over the given scanner.
func NewHandler(s Scanner) *Handler {
	return &Handler{s: s}
}

// Routes returns the repository REST endpoints.
func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/repositories/scan", h.scan)
	mux.HandleFunc("GET /api/repositories", h.list)
	return mux
}

type scanRequest struct {
	Owner string `json:"owner"`
	Name  string `json:"name"`
	Ref   string `json:"ref"`
}

func (h *Handler) scan(w http.ResponseWriter, r *http.Request) {
	var req scanRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	result, err := Scan(r.Context(), h.s, req.Owner, req.Name, req.Ref)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	repos, err := ListRepos(r.Context(), h.s)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"repositories": repos})
}
