package tickets

import (
	"fmt"
	"net/http"

	"github.com/otal-labs/nexul/internal/platform/colors"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/httpx"
)

// Handler adapts tickets use-cases to HTTP/JSON, since the browser never talks to MCP directly.
type Handler struct {
	svc *Service
}

// NewHandler wires the tickets REST gateway over the given service.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

type saveTicketRequest struct {
	Title      string `json:"title"`
	Body       string `json:"body"`
	DocID      string `json:"doc_id"`
	Developer  string `json:"developer"`
	Tester     string `json:"tester"`
	ProjectID  string `json:"project_id"`
	CategoryID string `json:"category_id"`
	TypeID     string `json:"type_id"`
}

type updateStatusRequest struct {
	Status Status `json:"status"`
}

type updateTicketRequest struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

type setTypeRequest struct {
	TypeID string `json:"type_id"`
}

type setPersonRequest struct {
	Login string `json:"login"`
}

type updatePositionRequest struct {
	Position int `json:"position"`
}

type labelRequest struct {
	Label string `json:"label"`
}

type linkPRRequest struct {
	Owner  string `json:"owner"`
	Repo   string `json:"repo"`
	Number int    `json:"number"`
	Title  string `json:"title"`
	SHA    string `json:"sha"`
}

type linkBranchRequest struct {
	Owner  string `json:"owner"`
	Repo   string `json:"repo"`
	Branch string `json:"branch"`
}

// ticketLinks is the ticket page's development-section payload.
type ticketLinks struct {
	PRs      []PRLink     `json:"prs"`
	Branches []BranchLink `json:"branches"`
}

type devStatusRequest struct {
	TicketIDs []string `json:"ticket_ids"`
}

type setLabelColorRequest struct {
	ProjectID string `json:"project_id"`
	Color     string `json:"color"`
}

type labelColorsRequest struct {
	ProjectID string   `json:"project_id"`
	Labels    []string `json:"labels"`
}

// Routes returns the tickets REST endpoints.
func (h *Handler) Routes() http.Handler {
	mux := httpx.NewServeMux()
	mux.HandleFunc("POST /api/tickets", h.create)
	mux.HandleFunc("GET /api/tickets", h.list)
	mux.HandleFunc("GET /api/tickets/search", h.search)
	mux.HandleFunc("GET /api/tickets/labels", h.listAllLabels)
	mux.HandleFunc("POST /api/tickets/labels/colors", h.labelColors)
	mux.HandleFunc("PUT /api/tickets/labels/{label}/color", h.setLabelColor)
	mux.HandleFunc("POST /api/tickets/dev-status", h.devStatus)
	mux.HandleFunc("GET /api/tickets/{id}", h.get)
	mux.HandleFunc("PATCH /api/tickets/{id}", h.update)
	mux.HandleFunc("PATCH /api/tickets/{id}/status", h.updateStatus)
	mux.HandleFunc("PATCH /api/tickets/{id}/position", h.updatePosition)
	mux.HandleFunc("PATCH /api/tickets/{id}/type", h.setType)
	mux.HandleFunc("PATCH /api/tickets/{id}/developer", h.setPerson(RoleDeveloper))
	mux.HandleFunc("PATCH /api/tickets/{id}/tester", h.setPerson(RoleTester))
	mux.HandleFunc("GET /api/tickets/{id}/labels", h.listLabels)
	mux.HandleFunc("POST /api/tickets/{id}/labels", h.addLabel)
	mux.HandleFunc("DELETE /api/tickets/{id}/labels/{label}", h.removeLabel)
	mux.HandleFunc("DELETE /api/tickets/{id}", h.delete)
	mux.HandleFunc("GET /api/tickets/{id}/links", h.listLinks)
	mux.HandleFunc("POST /api/tickets/{id}/prs", h.linkPR)
	mux.HandleFunc("POST /api/tickets/{id}/branches", h.linkBranch)
	return mux
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req saveTicketRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	t, err := h.svc.Create(r.Context(), req.ProjectID, req.Title, req.Body, req.DocID, req.Developer, CreateOptions{CategoryID: req.CategoryID, TypeID: req.TypeID, Tester: req.Tester})
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, t)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	if docID := r.URL.Query().Get("doc_id"); docID != "" {
		ts, err := h.svc.ListByDoc(r.Context(), docID)
		if err != nil {
			httpx.WriteError(w, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, ts)
		return
	}
	if projectID := r.URL.Query().Get("project_id"); projectID != "" {
		ts, err := h.svc.ListByProject(r.Context(), projectID)
		if err != nil {
			httpx.WriteError(w, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, ts)
		return
	}
	ts, err := h.svc.List(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, ts)
}

func (h *Handler) search(w http.ResponseWriter, r *http.Request) {
	results, err := h.svc.Search(r.Context(), r.URL.Query().Get("q"), 0)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, results)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	t, err := h.svc.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, t)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	var req updateTicketRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	t, err := h.svc.UpdateTicket(r.Context(), r.PathValue("id"), req.Title, req.Body)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, t)
}

func (h *Handler) updateStatus(w http.ResponseWriter, r *http.Request) {
	var req updateStatusRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if req.Status == "" {
		httpx.WriteError(w, fmt.Errorf("%w: status is required", apperrs.ErrInvalid))
		return
	}
	t, err := h.svc.UpdateStatus(r.Context(), r.PathValue("id"), req.Status)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, t)
}

func (h *Handler) updatePosition(w http.ResponseWriter, r *http.Request) {
	var req updatePositionRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	t, err := h.svc.SetPosition(r.Context(), r.PathValue("id"), req.Position)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, t)
}

func (h *Handler) setType(w http.ResponseWriter, r *http.Request) {
	var req setTypeRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if req.TypeID == "" {
		httpx.WriteError(w, fmt.Errorf("%w: type_id is required", apperrs.ErrInvalid))
		return
	}
	t, err := h.svc.SetType(r.Context(), r.PathValue("id"), req.TypeID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, t)
}

func (h *Handler) setPerson(role Role) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req setPersonRequest
		if err := httpx.DecodeJSON(r, &req); err != nil {
			httpx.WriteError(w, err)
			return
		}
		t, err := h.svc.SetPerson(r.Context(), r.PathValue("id"), role, req.Login)
		if err != nil {
			httpx.WriteError(w, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, t)
	}
}

func (h *Handler) listLabels(w http.ResponseWriter, r *http.Request) {
	labels, err := h.svc.ListLabels(r.Context(), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, labels)
}

func (h *Handler) addLabel(w http.ResponseWriter, r *http.Request) {
	var req labelRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	t, err := h.svc.AddLabel(r.Context(), r.PathValue("id"), req.Label)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, t)
}

func (h *Handler) removeLabel(w http.ResponseWriter, r *http.Request) {
	t, err := h.svc.RemoveLabel(r.Context(), r.PathValue("id"), r.PathValue("label"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, t)
}

func (h *Handler) listAllLabels(w http.ResponseWriter, r *http.Request) {
	labels, err := h.svc.ListAllLabels(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, labels)
}

func (h *Handler) setLabelColor(w http.ResponseWriter, r *http.Request) {
	var req setLabelColorRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	lc, err := h.svc.SetLabelColor(r.Context(), req.ProjectID, r.PathValue("label"), colors.Color(req.Color))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, lc)
}

func (h *Handler) labelColors(w http.ResponseWriter, r *http.Request) {
	var req labelColorsRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	out, err := h.svc.LabelColors(r.Context(), req.ProjectID, req.Labels)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) devStatus(w http.ResponseWriter, r *http.Request) {
	var req devStatusRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	statuses, err := h.svc.DevStatus(r.Context(), req.TicketIDs)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, statuses)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), r.PathValue("id")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listLinks(w http.ResponseWriter, r *http.Request) {
	prs, branches, err := h.svc.ListLinks(r.Context(), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, ticketLinks{PRs: prs, Branches: branches})
}

func (h *Handler) linkPR(w http.ResponseWriter, r *http.Request) {
	var req linkPRRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if err := h.svc.LinkPR(r.Context(), r.PathValue("id"), PRRef(req)); err != nil {
		httpx.WriteError(w, err)
		return
	}
	prs, branches, err := h.svc.ListLinks(r.Context(), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, ticketLinks{PRs: prs, Branches: branches})
}

func (h *Handler) linkBranch(w http.ResponseWriter, r *http.Request) {
	var req linkBranchRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if err := h.svc.LinkBranch(r.Context(), r.PathValue("id"), req.Owner, req.Repo, req.Branch); err != nil {
		httpx.WriteError(w, err)
		return
	}
	prs, branches, err := h.svc.ListLinks(r.Context(), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, ticketLinks{PRs: prs, Branches: branches})
}
