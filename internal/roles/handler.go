package roles

import (
	"context"
	"net/http"

	"github.com/otal-labs/nexul/internal/platform/httpx"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

type ctxKey string

const userIDKey ctxKey = "user_id"

// WithUserID lets roles attribute mutations without importing auth (ADR 0017).
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// UserIDFromCtx returns the user id injected by WithUserID, or "".
func UserIDFromCtx(ctx context.Context) string {
	id, _ := ctx.Value(userIDKey).(string)
	return id
}

// Handler is mounted under the tenancy domain's /api/workspaces prefix.
type Handler struct {
	svc *Service
}

// NewHandler wires the roles REST gateway over the given service.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

type roleRequest struct {
	Name    string   `json:"name"`
	Actions []string `json:"actions"`
}

// Routes are wrapped with the auth-user-id injection adapter before mounting behind RequireAuth.
func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/workspaces/{workspaceID}/roles", h.list)
	mux.HandleFunc("POST /api/workspaces/{workspaceID}/roles", h.create)
	mux.HandleFunc("GET /api/workspaces/{workspaceID}/roles/{roleID}", h.get)
	mux.HandleFunc("PATCH /api/workspaces/{workspaceID}/roles/{roleID}", h.update)
	mux.HandleFunc("DELETE /api/workspaces/{workspaceID}/roles/{roleID}", h.delete)
	return mux
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	rs, err := h.svc.List(r.Context(), r.PathValue("workspaceID"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, rs)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req roleRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	role, err := h.svc.Create(r.Context(), r.PathValue("workspaceID"), UserIDFromCtx(r.Context()), req.Name, setFromActions(req.Actions))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, role)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	role, err := h.svc.Get(r.Context(), r.PathValue("workspaceID"), r.PathValue("roleID"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, role)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	var req roleRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	role, err := h.svc.Update(r.Context(), r.PathValue("workspaceID"), r.PathValue("roleID"), UserIDFromCtx(r.Context()), req.Name, setFromActions(req.Actions))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, role)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), r.PathValue("workspaceID"), r.PathValue("roleID"), UserIDFromCtx(r.Context())); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// setFromActions silently drops unknown action names, so a client on an older grid never gets a 400.
func setFromActions(actions []string) permissions.Set {
	out := make([]permissions.Action, 0, len(actions))
	for _, name := range actions {
		if a, ok := permissions.ParseAction(name); ok {
			out = append(out, a)
		}
	}
	return permissions.SetOf(out...)
}
