package gitprovider

import (
	"fmt"
	"net/http"
	"strconv"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/httpx"
)

// Handler adapts the gitprovider use-cases to the HTTP/JSON gateway (ADR 0019); the browser never talks to MCP directly.
type Handler struct {
	p GitProvider
}

// NewHandler wires the gitprovider REST gateway over the given provider.
func NewHandler(p GitProvider) *Handler {
	return &Handler{p: p}
}

// Routes returns the gitprovider REST endpoints.
func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/repos/{owner}/{repo}/prs", h.listPRs)
	mux.HandleFunc("GET /api/repos/{owner}/{repo}/prs/{number}", h.getPR)
	mux.HandleFunc("GET /api/repos/{owner}/{repo}", h.getRepo)
	return mux
}

func (h *Handler) listPRs(w http.ResponseWriter, r *http.Request) {
	prs, err := ListPRs(r.Context(), h.p, r.PathValue("owner"), r.PathValue("repo"), PROpts{
		State: strArg(r.URL.Query().Get("state"), "open"),
	})
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, prs)
}

func (h *Handler) getPR(w http.ResponseWriter, r *http.Request) {
	number, err := strconv.Atoi(r.PathValue("number"))
	if err != nil || number < 1 {
		httpx.WriteError(w, fmt.Errorf("%w: number must be a positive integer", apperrs.ErrInvalid))
		return
	}
	pr, err := GetPR(r.Context(), h.p, r.PathValue("owner"), r.PathValue("repo"), number)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, pr)
}

func (h *Handler) getRepo(w http.ResponseWriter, r *http.Request) {
	repo, err := GetRepo(r.Context(), h.p, r.PathValue("owner"), r.PathValue("repo"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, repo)
}
