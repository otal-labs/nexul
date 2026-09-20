package deploy

import (
	"net/http"

	"github.com/otal-labs/nexul/internal/platform/httpx"
)

// Handler adapts the deploy use-cases to the HTTP/JSON gateway (ADR 0019).
// It is mounted by the composition root; the browser never talks to MCP.
type Handler struct {
	svc *Service
}

// NewHandler wires the deploy REST gateway over the given service.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Routes returns the deploy and stack REST endpoints.
func (h *Handler) Routes() http.Handler {
	mux := httpx.NewServeMux()
	mux.HandleFunc("POST /api/deploys", h.deploy)
	mux.HandleFunc("GET /api/deploys", h.list)
	mux.HandleFunc("GET /api/deploys/{id}", h.get)
	mux.HandleFunc("POST /api/deploys/{id}/cancel", h.cancel)
	h.mountStackRoutes(mux, "/api/stacks")
	// /api/services stays mounted through the same handlers for one release (2026-09-10): the pre-rename web
	// build still calls it, and it is dropped once web moves to /api/stacks.
	h.mountStackRoutes(mux, "/api/services")
	mux.HandleFunc("POST /api/machines/{id}/import", h.importMachine)
	return mux
}

// mountStackRoutes registers the stack CRUD + sub-resource endpoints under prefix, so /api/stacks and its
// deprecated /api/services alias share one set of handler functions.
func (h *Handler) mountStackRoutes(mux *httpx.ServeMux, prefix string) {
	mux.HandleFunc("POST "+prefix, h.createStack)
	mux.HandleFunc("GET "+prefix, h.listStacks)
	mux.HandleFunc("GET "+prefix+"/{id}", h.getStack)
	mux.HandleFunc("PATCH "+prefix+"/{id}", h.updateStack)
	mux.HandleFunc("DELETE "+prefix+"/{id}", h.deleteStack)
	mux.HandleFunc("POST "+prefix+"/{id}/rollback", h.rollback)
	mux.HandleFunc("GET "+prefix+"/{id}/deploys", h.listStackDeploys)
	mux.HandleFunc("GET "+prefix+"/{id}/services", h.listServices)
}

func (h *Handler) deploy(w http.ResponseWriter, r *http.Request) {
	var req DeployRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if req.TriggeredBy == "" {
		req.TriggeredBy = triggeredBy(r)
	}
	d, err := h.svc.Deploy(r.Context(), req)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, d)
}

// stackOrServiceID reads the "stack_id" query param, falling back to the deprecated "service_id" alias.
func stackOrServiceID(r *http.Request) string {
	if id := r.URL.Query().Get("stack_id"); id != "" {
		return id
	}
	return r.URL.Query().Get("service_id")
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Query().Get("service") != "":
		ds, err := h.svc.ListByService(r.Context(), r.URL.Query().Get("service"))
		if err != nil {
			httpx.WriteError(w, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, ds)
		return
	case stackOrServiceID(r) != "":
		ds, err := h.svc.ListByStackID(r.Context(), stackOrServiceID(r))
		if err != nil {
			httpx.WriteError(w, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, ds)
		return
	case r.URL.Query().Get("status") != "":
		ds, err := h.svc.ListByStatus(r.Context(), Status(r.URL.Query().Get("status")))
		if err != nil {
			httpx.WriteError(w, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, ds)
		return
	default:
		ds, err := h.svc.List(r.Context())
		if err != nil {
			httpx.WriteError(w, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, ds)
	}
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	d, err := h.svc.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, d)
}

func (h *Handler) cancel(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Cancel(r.Context(), r.PathValue("id")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, map[string]string{"id": r.PathValue("id"), "status": "cancelling"})
}

// createStackRequest is the stack-creation wire shape: a Stack plus the candidate's declared services.
type createStackRequest struct {
	Stack
	Declared map[string]Declared `json:"declared,omitempty"`
	// LinkRepository attaches the build repository to the project first; Deploy enqueues the first deploy at Ref
	// (default: the build source's branch). Both are what the wizard's service step sends.
	LinkRepository bool   `json:"link_repository,omitempty"`
	Deploy         bool   `json:"deploy,omitempty"`
	Ref            string `json:"ref,omitempty"`
}

// createStackResponse is the created stack plus the first deploy when one was asked for.
type createStackResponse struct {
	*Stack
	Deploy *Deploy `json:"deploy,omitempty"`
}

func (h *Handler) createStack(w http.ResponseWriter, r *http.Request) {
	var req createStackRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	opts := CreateStackOptions{LinkRepository: req.LinkRepository, Deploy: req.Deploy, Ref: req.Ref, TriggeredBy: triggeredBy(r)}
	stack, d, err := h.svc.CreateStackWithOptions(r.Context(), req.Stack, req.Declared, opts)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, createStackResponse{Stack: stack, Deploy: d})
}

func (h *Handler) listStacks(w http.ResponseWriter, r *http.Request) {
	stacks, err := h.svc.ListStacks(r.Context(), r.URL.Query().Get("project_id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, stacks)
}

// stackResponse wraps a stack with its branch deployments so the SPA renders both from one fetch.
type stackResponse struct {
	*Stack
	BranchDeployments []*Stack `json:"branch_deployments,omitempty"`
}

func (h *Handler) getStack(w http.ResponseWriter, r *http.Request) {
	stack, err := h.svc.GetStack(r.Context(), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	resp := stackResponse{Stack: stack}
	if stack.DerivedFrom == "" {
		branches, err := h.svc.ListBranchDeployments(r.Context(), stack.ID)
		if err != nil {
			httpx.WriteError(w, err)
			return
		}
		resp.BranchDeployments = branches
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) updateStack(w http.ResponseWriter, r *http.Request) {
	var req Stack
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	req.ID = r.PathValue("id")
	stack, err := h.svc.UpdateStack(r.Context(), req)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, stack)
}

func (h *Handler) deleteStack(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteStack(r.Context(), r.PathValue("id")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"id": r.PathValue("id"), "status": "deleted"})
}

func (h *Handler) rollback(w http.ResponseWriter, r *http.Request) {
	d, err := h.svc.Rollback(r.Context(), r.PathValue("id"), triggeredBy(r))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, d)
}

func (h *Handler) listStackDeploys(w http.ResponseWriter, r *http.Request) {
	ds, err := h.svc.ListByStackID(r.Context(), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, ds)
}

func (h *Handler) importMachine(w http.ResponseWriter, r *http.Request) {
	var req ImportRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	result, err := h.svc.Import(r.Context(), r.PathValue("id"), req)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) listServices(w http.ResponseWriter, r *http.Request) {
	svcs, err := h.svc.ListServices(r.Context(), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, svcs)
}
