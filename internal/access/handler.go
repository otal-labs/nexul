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
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/permissions", h.listGrants)
	mux.HandleFunc("GET /api/permissions/catalog", h.catalog)
	mux.HandleFunc("GET /api/permissions/users", h.listUsers)
	mux.HandleFunc("PUT /api/permissions", h.setGrants)
	return mux
}

func (h *Handler) listGrants(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if q.Get("resource_type") == resourceTypePlay {
		resourceID := q.Get("resource_id")
		if resourceID == "" {
			httpx.WriteError(w, fmt.Errorf("%w: resource_id is required", apperrs.ErrInvalid))
			return
		}
		grants, err := h.svc.ListPlayGrants(r.Context(), actorID(r), resourceID)
		if err != nil {
			httpx.WriteError(w, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"resource_type": resourceTypePlay, "resource_id": resourceID, "grants": grants})
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

// catalog is the full grid in display order; roles, tokens, and doc grants all pick from it.
func (h *Handler) catalog(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"permissions": permissions.Catalog()})
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
	Actions      []string `json:"actions"`
	Grant        bool     `json:"grant"`
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
	if req.ResourceType == resourceTypePlay {
		if err := h.svc.SetPlayGrants(r.Context(), actorID(r), req.ResourceIDs, req.UserIDs, actions, req.Grant); err != nil {
			httpx.WriteError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if err := h.svc.SetGrants(r.Context(), actorID(r), req.DocIDs, req.UserIDs, actions, req.Grant); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parseActions(raw []string) ([]permissions.Action, error) {
	actions := make([]permissions.Action, 0, len(raw))
	for _, r := range raw {
		a, ok := permissions.ParseAction(r)
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
