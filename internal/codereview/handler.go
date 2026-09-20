package codereview

import (
	"fmt"
	"net/http"
	"strconv"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/httpx"
)

// Handler is read-only, since provider events are the only writer (ADR 0034).
type Handler struct {
	svc *Service
}

// NewHandler wires the codereview REST gateway over the given service.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Routes' List filters by query param (ticket_id or repo+number), so the collection route stays free for /{id}.
func (h *Handler) Routes() http.Handler {
	mux := httpx.NewServeMux()
	mux.HandleFunc("GET /api/reviews", h.list)
	mux.HandleFunc("GET /api/reviews/{id}", h.get)
	return mux
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if ticketID := q.Get("ticket_id"); ticketID != "" {
		rs, err := h.svc.ListByTicket(r.Context(), ticketID)
		if err != nil {
			httpx.WriteError(w, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, rs)
		return
	}
	if repo := q.Get("repo"); repo != "" {
		number, err := strconv.Atoi(q.Get("number"))
		if err != nil || number < 1 {
			httpx.WriteError(w, fmt.Errorf("%w: number must be a positive integer", apperrs.ErrInvalid))
			return
		}
		rs, err := h.svc.ListByPR(r.Context(), repo, number)
		if err != nil {
			httpx.WriteError(w, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, rs)
		return
	}
	httpx.WriteError(w, fmt.Errorf("%w: ticket_id or repo+number is required", apperrs.ErrInvalid))
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	rv, err := h.svc.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, rv)
}
