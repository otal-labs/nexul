package tenancy

import (
	"context"
	"net/http"
	"time"

	"github.com/otal-labs/nexul/internal/platform/httpx"
)

type InvitationHandler struct {
	svc *InvitationService
}

func NewInvitationHandler(svc *InvitationService) *InvitationHandler {
	return &InvitationHandler{svc: svc}
}

func (h *InvitationHandler) Routes() http.Handler {
	mux := httpx.NewServeMux()
	mux.HandleFunc("POST /api/invitations", h.create)
	mux.HandleFunc("GET /api/invitations", h.list)
	mux.HandleFunc("DELETE /api/invitations/{invitationID}", h.revoke)
	return mux
}

func (h *InvitationHandler) PublicRoutes() http.Handler {
	mux := httpx.NewServeMux()
	mux.HandleFunc("POST /api/invitations/preview", h.preview)
	return mux
}

type createInvitationRequest struct {
	Grants        []*InvitationGrant `json:"grants"`
	ExpiresInDays int                `json:"expires_in_days"`
}

type invitationPreviewRequest struct {
	Token string `json:"token"`
}

func (h *InvitationHandler) create(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := invitationRequestContext(r)
	defer cancel()
	var req createInvitationRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	created, err := h.svc.Create(ctx, UserIDFromCtx(r.Context()), CreateInvitationInput{Grants: req.Grants, ExpiresInDays: req.ExpiresInDays})
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, created)
}

func (h *InvitationHandler) list(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := invitationRequestContext(r)
	defer cancel()
	invitations, err := h.svc.List(ctx, UserIDFromCtx(r.Context()))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"invitations": invitations})
}

func (h *InvitationHandler) revoke(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := invitationRequestContext(r)
	defer cancel()
	if err := h.svc.Revoke(ctx, UserIDFromCtx(r.Context()), r.PathValue("invitationID")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *InvitationHandler) preview(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := invitationRequestContext(r)
	defer cancel()
	var req invitationPreviewRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	preview, err := h.svc.Preview(ctx, req.Token)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, preview)
}

func invitationRequestContext(r *http.Request) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), 10*time.Second)
}
