package workspace

import (
	"context"
	"net/http"

	"github.com/otal-labs/nexul/internal/platform/colors"
	"github.com/otal-labs/nexul/internal/platform/httpx"
)

type ctxKey string

const userIDKey ctxKey = "user_id"

// WithUserID attaches the authenticated user id so the gateway can owner-gate without importing auth (ADR 0017).
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// UserIDFromCtx returns the user id injected by WithUserID, or "".
func UserIDFromCtx(ctx context.Context) string {
	id, _ := ctx.Value(userIDKey).(string)
	return id
}

// Handler adapts the workspace use-cases to the HTTP/JSON gateway (ADR 0019); the browser never talks to MCP.
type Handler struct {
	svc *Service
}

// NewHandler wires the workspace REST gateway over the given service.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Icon is a pointer to tell "field omitted" (keep current) from `"icon": ""` (clear it explicitly).
type saveProjectRequest struct {
	Name string  `json:"name"`
	Icon *string `json:"icon"`
}

// createProjectRequest is separate from saveProjectRequest since prefix is set once at creation, never renamed.
type createProjectRequest struct {
	Name        string `json:"name"`
	Prefix      string `json:"prefix"`
	WorkspaceID string `json:"workspace_id"`
	Icon        string `json:"icon"`
}

type setPrefixRequest struct {
	Prefix string `json:"prefix"`
}

type reorderProjectsRequest struct {
	IDs         []string `json:"ids"`
	WorkspaceID string   `json:"workspace_id"`
}

type addRepoRequest struct {
	Owner       string `json:"owner"`
	Name        string `json:"name"`
	ConnectorID string `json:"connector_id"`
	Role        string `json:"role"`
}

type setTestsLocationRequest struct {
	TestsLocation string `json:"tests_location"`
}

type saveCategoryRequest struct {
	ProjectID string `json:"project_id"`
	Name      string `json:"name"`
	Color     string `json:"color"`
}

type reorderCategoriesRequest struct {
	ProjectID string   `json:"project_id"`
	IDs       []string `json:"ids"`
}

type saveTicketTypeRequest struct {
	ProjectID string `json:"project_id"`
	Name      string `json:"name"`
	Color     string `json:"color"`
}

type setTicketTypeTemplateRequest struct {
	BodyTemplate string `json:"body_template"`
}

type saveStatusRequest struct {
	ProjectID string `json:"project_id"`
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Icon      string `json:"icon"`
}

// Routes returns the workspace REST endpoints, mounted behind RequireAuth by the composition root.
func (h *Handler) Routes() http.Handler {
	mux := httpx.NewServeMux()
	mux.HandleFunc("GET /api/projects", h.list)
	mux.HandleFunc("POST /api/projects", h.create)
	mux.HandleFunc("POST /api/projects/reorder", h.reorder)
	mux.HandleFunc("GET /api/projects/{id}", h.get)
	mux.HandleFunc("PATCH /api/projects/{id}", h.rename)
	mux.HandleFunc("POST /api/projects/{id}/prefix", h.setPrefix)
	mux.HandleFunc("PUT /api/projects/{id}/tests-location", h.setTestsLocation)
	mux.HandleFunc("DELETE /api/projects/{id}", h.delete)
	mux.HandleFunc("GET /api/projects/{id}/impact", h.impact)
	mux.HandleFunc("GET /api/projects/{id}/repos", h.listRepos)
	mux.HandleFunc("POST /api/projects/{id}/repos", h.addRepo)
	mux.HandleFunc("DELETE /api/projects/repos/{owner}/{name}", h.removeRepo)
	mux.HandleFunc("POST /api/projects/{id}/tickets/{ticketID}", h.moveTicket)

	mux.HandleFunc("GET /api/categories", h.listCategories)
	mux.HandleFunc("POST /api/categories", h.createCategory)
	mux.HandleFunc("POST /api/categories/reorder", h.reorderCategories)
	mux.HandleFunc("GET /api/categories/{id}", h.getCategory)
	mux.HandleFunc("PATCH /api/categories/{id}", h.renameCategory)
	mux.HandleFunc("DELETE /api/categories/{id}", h.deleteCategory)
	mux.HandleFunc("POST /api/categories/{id}/tickets/{ticketID}", h.moveTicketToCategory)
	mux.HandleFunc("DELETE /api/categories/tickets/{ticketID}", h.clearTicketCategory)

	mux.HandleFunc("GET /api/ticket-types", h.listTicketTypes)
	mux.HandleFunc("POST /api/ticket-types", h.createTicketType)
	mux.HandleFunc("POST /api/ticket-types/reorder", h.reorderTicketTypes)
	mux.HandleFunc("GET /api/ticket-types/{id}", h.getTicketType)
	mux.HandleFunc("PATCH /api/ticket-types/{id}", h.renameTicketType)
	mux.HandleFunc("PUT /api/ticket-types/{id}/template", h.setTicketTypeTemplate)
	mux.HandleFunc("DELETE /api/ticket-types/{id}", h.deleteTicketType)

	mux.HandleFunc("GET /api/statuses", h.listStatuses)
	mux.HandleFunc("POST /api/statuses", h.createStatus)
	mux.HandleFunc("POST /api/statuses/reorder", h.reorderStatuses)
	mux.HandleFunc("GET /api/statuses/{id}", h.getStatus)
	mux.HandleFunc("PATCH /api/statuses/{id}", h.renameStatus)
	mux.HandleFunc("DELETE /api/statuses/{id}", h.deleteStatus)
	return mux
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	projects, err := h.svc.List(r.Context(), r.URL.Query().Get("workspace_id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, projects)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req createProjectRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	p, err := h.svc.Create(r.Context(), UserIDFromCtx(r.Context()), req.WorkspaceID, req.Name, req.Prefix, ProjectIcon(req.Icon))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, p)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	p, err := h.svc.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, p)
}

func (h *Handler) rename(w http.ResponseWriter, r *http.Request) {
	var req saveProjectRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	p, err := h.svc.Rename(r.Context(), UserIDFromCtx(r.Context()), r.PathValue("id"), req.Name, (*ProjectIcon)(req.Icon))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, p)
}

func (h *Handler) setPrefix(w http.ResponseWriter, r *http.Request) {
	var req setPrefixRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	p, err := h.svc.SetPrefix(r.Context(), UserIDFromCtx(r.Context()), r.PathValue("id"), req.Prefix)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, p)
}

func (h *Handler) setTestsLocation(w http.ResponseWriter, r *http.Request) {
	var req setTestsLocationRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	p, err := h.svc.SetTestsLocation(r.Context(), UserIDFromCtx(r.Context()), r.PathValue("id"), TestsLocation(req.TestsLocation))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, p)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), UserIDFromCtx(r.Context()), r.PathValue("id")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) impact(w http.ResponseWriter, r *http.Request) {
	impact, err := h.svc.DeleteImpact(r.Context(), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, impact)
}

func (h *Handler) reorder(w http.ResponseWriter, r *http.Request) {
	var req reorderProjectsRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if err := h.svc.Reorder(r.Context(), UserIDFromCtx(r.Context()), req.WorkspaceID, req.IDs); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listRepos(w http.ResponseWriter, r *http.Request) {
	repos, err := h.svc.ListRepos(r.Context(), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, repos)
}

func (h *Handler) addRepo(w http.ResponseWriter, r *http.Request) {
	var req addRepoRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if err := h.svc.AddRepo(r.Context(), UserIDFromCtx(r.Context()), r.PathValue("id"), req.Owner, req.Name, req.ConnectorID, RepoRole(req.Role)); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) removeRepo(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.RemoveRepo(r.Context(), UserIDFromCtx(r.Context()), r.PathValue("owner"), r.PathValue("name")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) moveTicket(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.MoveTicket(r.Context(), r.PathValue("ticketID"), r.PathValue("id")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listCategories(w http.ResponseWriter, r *http.Request) {
	if projectID := r.URL.Query().Get("project_id"); projectID != "" {
		cats, err := h.svc.ListCategoriesByProject(r.Context(), projectID)
		if err != nil {
			httpx.WriteError(w, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, cats)
		return
	}
	cats, err := h.svc.ListCategories(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, cats)
}

func (h *Handler) createCategory(w http.ResponseWriter, r *http.Request) {
	var req saveCategoryRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	c, err := h.svc.CreateCategory(r.Context(), UserIDFromCtx(r.Context()), req.ProjectID, req.Name, colors.Color(req.Color))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, c)
}

func (h *Handler) getCategory(w http.ResponseWriter, r *http.Request) {
	c, err := h.svc.GetCategory(r.Context(), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, c)
}

func (h *Handler) renameCategory(w http.ResponseWriter, r *http.Request) {
	var req saveCategoryRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	c, err := h.svc.RenameCategory(r.Context(), UserIDFromCtx(r.Context()), r.PathValue("id"), req.Name, colors.Color(req.Color))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, c)
}

func (h *Handler) deleteCategory(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteCategory(r.Context(), UserIDFromCtx(r.Context()), r.PathValue("id")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) reorderCategories(w http.ResponseWriter, r *http.Request) {
	var req reorderCategoriesRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if err := h.svc.ReorderCategories(r.Context(), UserIDFromCtx(r.Context()), req.ProjectID, req.IDs); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) moveTicketToCategory(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.MoveTicketToCategory(r.Context(), r.PathValue("ticketID"), r.PathValue("id")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) clearTicketCategory(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.MoveTicketToCategory(r.Context(), r.PathValue("ticketID"), ""); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listTicketTypes(w http.ResponseWriter, r *http.Request) {
	types, err := h.svc.ListTicketTypesByProject(r.Context(), r.URL.Query().Get("project_id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, types)
}

func (h *Handler) createTicketType(w http.ResponseWriter, r *http.Request) {
	var req saveTicketTypeRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	tt, err := h.svc.CreateTicketType(r.Context(), UserIDFromCtx(r.Context()), req.ProjectID, req.Name, colors.Color(req.Color))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, tt)
}

func (h *Handler) getTicketType(w http.ResponseWriter, r *http.Request) {
	tt, err := h.svc.GetTicketType(r.Context(), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, tt)
}

func (h *Handler) renameTicketType(w http.ResponseWriter, r *http.Request) {
	var req saveTicketTypeRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	tt, err := h.svc.RenameTicketType(r.Context(), UserIDFromCtx(r.Context()), r.PathValue("id"), req.Name, colors.Color(req.Color))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, tt)
}

func (h *Handler) setTicketTypeTemplate(w http.ResponseWriter, r *http.Request) {
	var req setTicketTypeTemplateRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	tt, err := h.svc.SetTicketTypeTemplate(r.Context(), UserIDFromCtx(r.Context()), r.PathValue("id"), req.BodyTemplate)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, tt)
}

func (h *Handler) deleteTicketType(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteTicketType(r.Context(), UserIDFromCtx(r.Context()), r.PathValue("id")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) reorderTicketTypes(w http.ResponseWriter, r *http.Request) {
	var req reorderCategoriesRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if err := h.svc.ReorderTicketTypes(r.Context(), UserIDFromCtx(r.Context()), req.ProjectID, req.IDs); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listStatuses(w http.ResponseWriter, r *http.Request) {
	statuses, err := h.svc.ListStatusesByProject(r.Context(), r.URL.Query().Get("project_id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, statuses)
}

func (h *Handler) createStatus(w http.ResponseWriter, r *http.Request) {
	var req saveStatusRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	st, err := h.svc.CreateStatus(r.Context(), UserIDFromCtx(r.Context()), req.ProjectID, req.Name, StatusKind(req.Kind), StatusIcon(req.Icon))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, st)
}

func (h *Handler) getStatus(w http.ResponseWriter, r *http.Request) {
	st, err := h.svc.GetStatus(r.Context(), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, st)
}

func (h *Handler) renameStatus(w http.ResponseWriter, r *http.Request) {
	var req saveStatusRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	st, err := h.svc.RenameStatus(r.Context(), UserIDFromCtx(r.Context()), r.PathValue("id"), req.Name, StatusKind(req.Kind), StatusIcon(req.Icon))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, st)
}

func (h *Handler) deleteStatus(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteStatus(r.Context(), UserIDFromCtx(r.Context()), r.PathValue("id")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) reorderStatuses(w http.ResponseWriter, r *http.Request) {
	var req reorderCategoriesRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if err := h.svc.ReorderStatuses(r.Context(), UserIDFromCtx(r.Context()), req.ProjectID, req.IDs); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
