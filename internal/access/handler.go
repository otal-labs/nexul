package access

import (
	"fmt"
	"net/http"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/httpx"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/platform/permissions"
)

// Handler adapts the access use-cases to the HTTP/JSON gateway (ADR 0019).
type Handler struct {
	svc *Service
}

// NewHandler wires the access REST gateway over the given service.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Routes returns the permissions REST endpoints.
func (h *Handler) Routes() http.Handler {
	mux := httpx.NewServeMux()
	mux.HandleFunc("GET /api/permissions", h.listGrants)
	mux.HandleFunc("GET /api/permissions/catalog", h.catalog)
	mux.HandleFunc("GET /api/permissions/explain", h.explain)
	mux.HandleFunc("GET /api/permissions/users", h.listUsers)
	mux.HandleFunc("PUT /api/permissions", h.setGrants)
	return mux
}

func (h *Handler) listGrants(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if resourceType := q.Get("resource_type"); resourceType == resourceTypePlay || resourceType == resourceTypeProject || resourceType == resourceTypeChannel {
		resourceID := q.Get("resource_id")
		if resourceID == "" {
			httpx.WriteError(w, fmt.Errorf("%w: resource_id is required", apperrs.ErrInvalid))
			return
		}
		grants, err := h.svc.ListRules(r.Context(), actorID(r), resourceType, resourceID)
		if err != nil {
			httpx.WriteError(w, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"resource_type": resourceType, "resource_id": resourceID, "grants": grants})
		return
	}
	docID := q.Get("doc_id")
	if docID == "" {
		docID = q.Get("resource_id")
	}
	if docID == "" {
		httpx.WriteError(w, fmt.Errorf("%w: doc_id is required", apperrs.ErrInvalid))
		return
	}
	grants, err := h.svc.ListGrants(r.Context(), actorID(r), docID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"doc_id": docID, "grants": grants})
}

// GridRoutes returns a workspace's Access page endpoints, mounted under /api/workspaces/{workspaceID}/access.
func (h *Handler) GridRoutes() http.Handler {
	mux := httpx.NewServeMux()
	mux.HandleFunc("GET /api/workspaces/{workspaceID}/access", h.grid)
	mux.HandleFunc("PUT /api/workspaces/{workspaceID}/access", h.saveGrid)
	return mux
}

func (h *Handler) grid(w http.ResponseWriter, r *http.Request) {
	g, err := h.svc.Grid(r.Context(), actorID(r), r.PathValue("workspaceID"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, g)
}

func (h *Handler) saveGrid(w http.ResponseWriter, r *http.Request) {
	var req GridSave
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if err := h.svc.SaveGrid(r.Context(), actorID(r), r.PathValue("workspaceID"), req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// catalog is the full grid in display order; roles, tokens, and doc grants all pick from it.
func (h *Handler) catalog(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"permissions": permissions.Catalog()})
}

// explain is a person's or, with role_id, a role's answers on a doc, play, or project and where each came from;
// user_id defaults to the caller.
func (h *Handler) explain(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if q.Get("resource_id") == "" {
		httpx.WriteError(w, fmt.Errorf("%w: resource_id is required", apperrs.ErrInvalid))
		return
	}
	if roleID := q.Get("role_id"); roleID != "" {
		out, err := h.svc.ExplainRole(r.Context(), actorID(r), q.Get("resource_type"), q.Get("resource_id"), roleID)
		if err != nil {
			httpx.WriteError(w, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, out)
		return
	}
	out, err := h.svc.Explain(r.Context(), actorID(r), q.Get("resource_type"), q.Get("resource_id"), q.Get("user_id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) listUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.svc.ListUsers(r.Context(), actorID(r))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"users": users})
}

type setGrantsRequest struct {
	ResourceType string   `json:"resource_type"`
	DocIDs       []string `json:"doc_ids"`
	ResourceIDs  []string `json:"resource_ids"`
	UserIDs      []string `json:"user_ids"`
	RoleIDs      []string `json:"role_ids"`
	Actions      []string `json:"actions"`
	Grant        bool     `json:"grant"`
	// State is allow, fallback, or deny; without it grant keeps its older meaning.
	State string `json:"state"`
}

func (r setGrantsRequest) state(resourceType string) string {
	if r.State != "" {
		return r.State
	}
	return GrantState(resourceType, r.Grant)
}

func (h *Handler) setGrants(w http.ResponseWriter, r *http.Request) {
	var req setGrantsRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	actions, err := parseActions(req.Actions)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	resourceType, resourceIDs := req.ResourceType, req.ResourceIDs
	if resourceType == "" || resourceType == resourceTypeDoc {
		resourceType, resourceIDs = resourceTypeDoc, req.DocIDs
	}
	to := Targets{UserIDs: req.UserIDs, RoleIDs: req.RoleIDs}
	if err := h.svc.SetRules(r.Context(), actorID(r), resourceType, resourceIDs, to, actions, req.state(resourceType)); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parseActions(raw []string) ([]permissions.Action, error) {
	actions := make([]permissions.Action, 0, len(raw))
	for _, r := range raw {
		a, ok := permissions.ParseRuleAction(r)
		if !ok {
			return nil, fmt.Errorf("%w: unknown action %s", apperrs.ErrInvalid, r)
		}
		actions = append(actions, a)
	}
	return actions, nil
}

func actorID(r *http.Request) string {
	if a, ok := identity.ActorFromCtx(r.Context()); ok {
		return a.ID
	}
	return ""
}
