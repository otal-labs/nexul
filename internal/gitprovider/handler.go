package gitprovider

import (
	"cmp"
	"fmt"
	"net/http"
	"strconv"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/httpx"
)

// Handler adapts the gitprovider use-cases to the HTTP/JSON gateway (ADR 0019); the browser never talks to MCP directly.
type Handler struct {
	p  GitProvider
	cc ChangeContextReader
}

// NewHandler wires the gitprovider REST gateway over the given provider.
func NewHandler(p GitProvider) *Handler {
	return &Handler{p: p}
}

// WithChangeContext adds the change-context route, which walks a PR or commit back to its tickets and decisions.
func (h *Handler) WithChangeContext(cc ChangeContextReader) *Handler {
	h.cc = cc
	return h
}

// Routes returns the gitprovider REST endpoints.
func (h *Handler) Routes() http.Handler {
	mux := httpx.NewServeMux()
	if h.cc != nil {
		mux.HandleFunc("GET /api/repos/{owner}/{repo}/change-context", h.changeContext)
	}
	mux.HandleFunc("GET /api/repos/{owner}/{repo}/prs", h.listPRs)
	mux.HandleFunc("GET /api/repos/{owner}/{repo}/prs/{number}", h.getPR)
	mux.HandleFunc("GET /api/repos/{owner}/{repo}", h.getRepo)
	return mux
}

func (h *Handler) listPRs(w http.ResponseWriter, r *http.Request) {
	prs, err := ListPRs(r.Context(), h.p, r.PathValue("owner"), r.PathValue("repo"), PROpts{
		State: cmp.Or(r.URL.Query().Get("state"), "open"),
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

// changeContext takes ?pr=<number> or ?commit=<sha>.
func (h *Handler) changeContext(w http.ResponseWriter, r *http.Request) {
	ref := ChangeRef{Owner: r.PathValue("owner"), Repo: r.PathValue("repo"), Commit: r.URL.Query().Get("commit")}
	if raw := r.URL.Query().Get("pr"); raw != "" {
		number, err := strconv.Atoi(raw)
		if err != nil || number < 1 {
			httpx.WriteError(w, fmt.Errorf("%w: pr must be a positive integer", apperrs.ErrInvalid))
			return
		}
		ref.Number = number
	}
	out, err := GetChangeContext(r.Context(), h.p, h.cc, ref)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}
