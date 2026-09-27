package automations

import (
	"errors"
	"net/http"
	"strings"

	"github.com/otal-labs/nexul/internal/platform/httpx"
)

// HostsHandler adapts the automations host use-cases to the HTTP gateway (ADR 0019).
type HostsHandler struct {
	svc *HostsService
}

// NewHostsHandler wires the automations host endpoints over the given service.
func NewHostsHandler(svc *HostsService) *HostsHandler {
	return &HostsHandler{svc: svc}
}

// Routes returns the endpoints mounted behind the user session: list, enroll and remove.
func (h *HostsHandler) Routes() http.Handler {
	mux := httpx.NewServeMux()
	mux.HandleFunc("GET /api/automation-hosts", h.list)
	mux.HandleFunc("POST /api/automation-hosts/enrollments", h.createEnrollment)
	mux.HandleFunc("DELETE /api/automation-hosts/{id}", h.remove)
	return mux
}

// PublicRoutes are the endpoints a host calls without a user session: enrolling with a code, then polling and
// removing itself with its own credential. They must sit outside RequireAuth.
func (h *HostsHandler) PublicRoutes() http.Handler {
	mux := httpx.NewServeMux()
	mux.HandleFunc("POST /api/automation-hosts/enroll", h.enroll)
	mux.HandleFunc("POST /api/automation-hosts/self/remove", h.removeSelf)
	mux.HandleFunc("GET /api/automation-hosts/self/assignments", h.assignments)
	return mux
}

func (h *HostsHandler) list(w http.ResponseWriter, r *http.Request) {
	hosts, err := h.svc.List(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, hosts)
}

// createHostEnrollmentRequest is the POST /api/automation-hosts/enrollments wire shape.
type createHostEnrollmentRequest struct {
	Name    string `json:"name"`
	Machine string `json:"machine,omitempty"`
}

func (h *HostsHandler) createEnrollment(w http.ResponseWriter, r *http.Request) {
	var req createHostEnrollmentRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	e, err := h.svc.CreateEnrollment(r.Context(), req.Name, req.Machine)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, e)
}

func (h *HostsHandler) enroll(w http.ResponseWriter, r *http.Request) {
	var req HostEnrollRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	enrolled, err := h.svc.Enroll(r.Context(), req)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, enrolled)
}

func (h *HostsHandler) remove(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Remove(r.Context(), r.PathValue("id")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *HostsHandler) removeSelf(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.RemoveSelf(r.Context(), bearerToken(r)); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *HostsHandler) assignments(w http.ResponseWriter, r *http.Request) {
	a, err := h.svc.Assignments(r.Context(), bearerToken(r))
	if errors.Is(err, errHostRemoved) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(HostRemovedRefusal)) // the status alone already refuses; the body is the reason
		return
	}
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, a)
}

func bearerToken(r *http.Request) string {
	return strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
}
