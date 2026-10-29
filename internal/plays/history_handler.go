package plays

import (
	"fmt"
	"net/http"
	"strconv"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/httpx"
)

// HistoryHandler adapts a play's history to the HTTP/JSON gateway (ADR 0019).
type HistoryHandler struct {
	svc *Service
}

// NewHistoryHandler wires the history gateway over the given service.
func NewHistoryHandler(svc *Service) *HistoryHandler {
	return &HistoryHandler{svc: svc}
}

// Routes are registered as exact patterns beside the run gateway's /api/plays subtree: the two reads need plays:read,
// the restore plays:write.
func (h *HistoryHandler) Routes() http.Handler {
	mux := httpx.NewServeMux()
	mux.HandleFunc("GET /api/plays/{id}/history", h.history)
	mux.HandleFunc("GET /api/plays/{id}/versions/{n}", h.version)
	mux.HandleFunc("POST /api/plays/{id}/versions/{n}/restore", h.restore)
	return mux
}

func (h *HistoryHandler) history(w http.ResponseWriter, r *http.Request) {
	out, err := h.svc.History(r.Context(), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *HistoryHandler) version(w http.ResponseWriter, r *http.Request) {
	n, err := versionNumber(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	out, err := h.svc.Version(r.Context(), r.PathValue("id"), n)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *HistoryHandler) restore(w http.ResponseWriter, r *http.Request) {
	n, err := versionNumber(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	p, err := h.svc.Restore(r.Context(), r.PathValue("id"), n)
	if err != nil {
		writePlayError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, p)
}

func versionNumber(r *http.Request) (int, error) {
	n, err := strconv.Atoi(r.PathValue("n"))
	if err != nil || n < 1 {
		return 0, fmt.Errorf("%w: version must be a number from 1 up", apperrs.ErrInvalid)
	}
	return n, nil
}
