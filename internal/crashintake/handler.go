package crashintake

import (
	"net/http"

	"github.com/otal-labs/nexul/internal/platform/httpx"
)

// Handler adapts the intake to the HTTP/JSON gateway (ADR 0019), where the Crash intake automation calls it.
type Handler struct {
	svc *Service
}

// NewHandler wires the intake's route over svc.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

type intakeRequest struct {
	Config
	Post
}

// Routes returns the intake endpoint, mounted under /api/crash-reports.
func (h *Handler) Routes() http.Handler {
	mux := httpx.NewServeMux()
	mux.HandleFunc("POST /api/crash-reports", h.intake)
	return mux
}

func (h *Handler) intake(w http.ResponseWriter, r *http.Request) {
	var req intakeRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	res, err := h.svc.Intake(r.Context(), req.Config, req.Post)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}
