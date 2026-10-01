package templates

import (
	"net/http"

	"github.com/otal-labs/nexul/internal/platform/httpx"
)

// Handler adapts the templates use-cases to the HTTP/JSON gateway (ADR 0019).
type Handler struct {
	svc *Service
}

// NewHandler wires the templates REST gateway over the given service.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Routes returns the templates endpoints: the instance layer by kind, and clone and reset at any layer.
func (h *Handler) Routes() http.Handler {
	mux := httpx.NewServeMux()
	mux.HandleFunc("GET /api/templates", h.list)
	mux.HandleFunc("GET /api/templates/{kind}", h.get)
	mux.HandleFunc("PUT /api/templates/{kind}", h.update)
	mux.HandleFunc("DELETE /api/templates/{kind}", h.resetInstance)
	mux.HandleFunc("POST /api/templates/clone", h.clone)
	mux.HandleFunc("POST /api/templates/reset", h.reset)
	return mux
}

type updateRequest struct {
	Key  string `json:"key"`
	Body string `json:"body"`
}

type cloneRequest struct {
	Kind string   `json:"kind"`
	Key  string   `json:"key"`
	From Location `json:"from"`
	To   Location `json:"to"`
}

type resetRequest struct {
	Kind string   `json:"kind"`
	Key  string   `json:"key"`
	At   Location `json:"at"`
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.List(r.Context())
	respond(w, list, err)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	at := Instance
	if scope := q.Get("scope"); scope != "" {
		at = Location{Scope: Scope(scope), WorkspaceID: q.Get("workspace_id"), ProjectID: q.Get("project_id")}
	}
	t, err := h.svc.Get(r.Context(), r.PathValue("kind"), q.Get("key"), at)
	respond(w, t, err)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	var req updateRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	t, err := h.svc.Update(r.Context(), r.PathValue("kind"), req.Key, Instance, req.Body)
	respond(w, t, err)
}

func (h *Handler) resetInstance(w http.ResponseWriter, r *http.Request) {
	t, err := h.svc.Reset(r.Context(), r.PathValue("kind"), r.URL.Query().Get("key"), Instance)
	respond(w, t, err)
}

func (h *Handler) clone(w http.ResponseWriter, r *http.Request) {
	var req cloneRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	t, err := h.svc.Clone(r.Context(), req.Kind, req.Key, req.From, req.To)
	respond(w, t, err)
}

func (h *Handler) reset(w http.ResponseWriter, r *http.Request) {
	var req resetRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	t, err := h.svc.Reset(r.Context(), req.Kind, req.Key, req.At)
	respond(w, t, err)
}

func respond(w http.ResponseWriter, v any, err error) {
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, v)
}
