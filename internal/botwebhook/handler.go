package botwebhook

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/httpx"
	"github.com/otal-labs/nexul/internal/platform/logging"
)

// Handler adapts the bot use-cases to the HTTP/JSON gateway (ADR 0019); the public execute route is not here.
type Handler struct {
	svc *Service
}

// NewHandler wires the bot management gateway over the given service.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

type createRequest struct {
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

// Routes returns the bot management endpoints, mounted under /api/conversations and /api/botwebhooks.
func (h *Handler) Routes() http.Handler {
	mux := httpx.NewServeMux()
	mux.HandleFunc("GET /api/conversations/{id}/botwebhooks", h.list)
	mux.HandleFunc("POST /api/conversations/{id}/botwebhooks", h.create)
	mux.HandleFunc("PATCH /api/botwebhooks/{id}", h.update)
	mux.HandleFunc("DELETE /api/botwebhooks/{id}", h.delete)
	mux.HandleFunc("GET /api/botwebhooks/{id}/avatar", h.avatar)
	return mux
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	bots, err := h.svc.List(r.Context(), r.PathValue("id"), r.URL.Query().Get("deleted") == "true")
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, bots)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	b, err := h.svc.Create(r.Context(), r.PathValue("id"), req.Name, req.Avatar)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, b)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	var req Changes
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	b, err := h.svc.Update(r.Context(), r.PathValue("id"), req)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, b)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	if _, err := h.svc.Delete(r.Context(), r.PathValue("id")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// avatar sets nosniff so an uploaded picture can never be re-interpreted as HTML on this origin.
func (h *Handler) avatar(w http.ResponseWriter, r *http.Request) {
	contentType, data, err := h.svc.Avatar(r.Context(), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	// A message keeps the ?v= it was posted with yet follows the bot's current avatar, so no copy lives long.
	w.Header().Set("Cache-Control", "private, max-age=300")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data) // the status is already sent; a failed write only means the client went away
}

// Discord's per-bot and per-address limits, a minute each.
const (
	postsPerMinute   = 30
	rejectsPerMinute = 60
	executeTimeout   = 10 * time.Second
)

// discordError is the body Discord answers an unknown webhook with, so a sender's own handling of it keeps working.
type discordError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

var unknownWebhook = discordError{Message: "Unknown Webhook", Code: 10015}

type rateLimited struct {
	Message    string  `json:"message"`
	RetryAfter float64 `json:"retry_after"`
	Global     bool    `json:"global"`
}

// ExecuteHandler serves the URL senders post to: no session, the token in the path is the credential, and it never
// passes the audit middleware, which records the raw path.
type ExecuteHandler struct {
	svc     *Service
	log     *slog.Logger
	posts   *limiter
	rejects *limiter
}

// NewExecuteHandler wires the public execute route over the given service.
func NewExecuteHandler(svc *Service, log *slog.Logger) *ExecuteHandler {
	return &ExecuteHandler{svc: svc, log: log, posts: newLimiter(postsPerMinute, time.Minute), rejects: newLimiter(rejectsPerMinute, time.Minute)}
}

// Routes returns the execute route, mounted before authentication.
func (h *ExecuteHandler) Routes() http.Handler {
	mux := httpx.NewServeMux()
	mux.HandleFunc("POST /api/botwebhooks/{id}/{token}", h.execute)
	return mux
}

// execute answers as Discord does: 204, or the message with wait=true; 404 alike for every refused URL; 429 past
// either limit, an address over its rejects answered before any work.
func (h *ExecuteHandler) execute(w http.ResponseWriter, r *http.Request) {
	addr, id := httpx.ClientAddr(r), r.PathValue("id")
	log := h.log.With("trace_id", logging.NewTraceID(), "botwebhook_id", id, "client", addr)
	if q := h.rejects.peek(addr); q.remaining == 0 {
		log.Info("bot post refused: too many rejected requests from this address")
		writeRateLimited(w, q)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), executeTimeout)
	defer cancel()
	b, err := h.svc.Authenticate(ctx, id, r.PathValue("token"))
	if err != nil {
		h.reject(w, log, addr, err)
		return
	}
	p, err := decodePayload(w, r)
	if err != nil {
		h.reject(w, log, addr, err)
		return
	}
	q, ok := h.posts.take(b.ID)
	if !ok {
		log.Info("bot post refused: the bot is over its posts a minute")
		writeRateLimited(w, q)
		return
	}
	m, err := h.svc.Execute(ctx, b, p)
	if err != nil {
		h.posts.give(b.ID)
		h.reject(w, log, addr, err)
		return
	}
	writeQuota(w.Header(), q)
	if wait, _ := strconv.ParseBool(r.URL.Query().Get("wait")); !wait {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, m)
}

// reject answers a refused post, counting a 404 or a 400 against the address; it is logged, never audited.
func (h *ExecuteHandler) reject(w http.ResponseWriter, log *slog.Logger, addr string, err error) {
	if errors.Is(err, apperrs.ErrNotFound) {
		h.rejects.take(addr)
		log.Info("bot post rejected: unknown webhook")
		httpx.WriteJSON(w, http.StatusNotFound, unknownWebhook)
		return
	}
	if errors.Is(err, apperrs.ErrInvalid) {
		h.rejects.take(addr)
		log.Info("bot post rejected: invalid payload", "reason", err.Error())
		httpx.WriteError(w, err)
		return
	}
	log.Error("bot post failed", "error", err)
	httpx.WriteError(w, err)
}

func decodePayload(w http.ResponseWriter, r *http.Request) (Payload, error) {
	var p Payload
	err := json.NewDecoder(http.MaxBytesReader(w, r.Body, MaxBodyBytes)).Decode(&p)
	var tooBig *http.MaxBytesError
	if errors.As(err, &tooBig) {
		return p, fmt.Errorf("%w: the body is over the %d KiB a post takes", apperrs.ErrInvalid, MaxBodyBytes>>10)
	}
	if err != nil {
		return p, fmt.Errorf("%w: the body is not a JSON post", apperrs.ErrInvalid)
	}
	return p, nil
}

func writeQuota(h http.Header, q quota) {
	h.Set("X-RateLimit-Limit", strconv.Itoa(q.limit))
	h.Set("X-RateLimit-Remaining", strconv.Itoa(q.remaining))
	h.Set("X-RateLimit-Reset-After", strconv.FormatFloat(q.resetAfter.Seconds(), 'f', 3, 64))
}

func writeRateLimited(w http.ResponseWriter, q quota) {
	writeQuota(w.Header(), q)
	w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(q.resetAfter.Seconds()))))
	httpx.WriteJSON(w, http.StatusTooManyRequests, rateLimited{Message: "You are being rate limited.", RetryAfter: math.Ceil(q.resetAfter.Seconds()*1000) / 1000})
}
