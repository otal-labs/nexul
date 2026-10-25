package agent

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/otal-labs/nexul/internal/harness"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/httpx"
	"github.com/otal-labs/nexul/internal/platform/identity"
)

// Handler adapts the agent pipeline's stop, steer and answer controls to the HTTP gateway.
type Handler struct {
	svc *Service
}

// NewHandler wires the agent REST gateway over the given service.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Routes returns the agent REST endpoints.
func (h *Handler) Routes() http.Handler {
	mux := httpx.NewServeMux()
	mux.HandleFunc("POST /api/agent/conversations/{id}/interrupt", h.interrupt)
	mux.HandleFunc("POST /api/agent/conversations/{id}/answer", h.answer)
	mux.HandleFunc("POST /api/agent/conversations/{id}/secret", h.secret)
	mux.HandleFunc("POST /api/agent/conversations/{id}/steer", h.steer)
	return mux
}

// AnswerRequest is the body of the answer route: the question's request id and one answer per question id.
type AnswerRequest struct {
	RequestID string                         `json:"request_id"`
	Answers   map[string]harness.AnswerValue `json:"answers"`
}

func (h *Handler) answer(w http.ResponseWriter, r *http.Request) {
	var req AnswerRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	actor, ok := identity.ActorFromCtx(r.Context())
	if !ok || actor.ID == "" {
		httpx.WriteError(w, fmt.Errorf("%w: an authenticated user is required", apperrs.ErrUnauthorized))
		return
	}
	if err := h.svc.AnswerFromChat(r.Context(), r.PathValue("id"), actor.ID, req.RequestID, harness.QuestionAnswer{Answers: req.Answers}); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// SecretRequest is the body of the secret route: the request's id and the value, empty to decline.
type SecretRequest struct {
	RequestID string `json:"request_id"`
	Secret    string `json:"secret"`
}

func (h *Handler) secret(w http.ResponseWriter, r *http.Request) {
	var req SecretRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	actor, ok := identity.ActorFromCtx(r.Context())
	if !ok || actor.ID == "" {
		httpx.WriteError(w, fmt.Errorf("%w: an authenticated user is required", apperrs.ErrUnauthorized))
		return
	}
	if err := h.svc.AnswerSecret(r.Context(), r.PathValue("id"), actor.ID, req.RequestID, strings.TrimSpace(req.Secret)); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// SteerRequest is the body of the steer route: the queued message to hand to the turn under way.
type SteerRequest struct {
	MessageID string `json:"message_id"`
}

func (h *Handler) steer(w http.ResponseWriter, r *http.Request) {
	var req SteerRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if req.MessageID == "" {
		httpx.WriteError(w, fmt.Errorf("%w: message_id is required", apperrs.ErrInvalid))
		return
	}
	if err := h.svc.Steer(r.Context(), r.PathValue("id"), req.MessageID); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) interrupt(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Interrupt(r.Context(), r.PathValue("id")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
