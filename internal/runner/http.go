package runner

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/otal-labs/nexul/internal/platform/httpx"
)

// HTTPHandler adapts the runner visibility use-cases to the HTTP/JSON gateway (ADR 0019).
type HTTPHandler struct {
	svc *Service
	log *slog.Logger
}

// NewHTTPHandler wires the runner REST gateway over the given service.
func NewHTTPHandler(svc *Service) *HTTPHandler {
	return &HTTPHandler{svc: svc, log: slog.Default()}
}

// Routes returns the runner REST endpoints mounted behind the user session.
func (h *HTTPHandler) Routes() http.Handler {
	mux := httpx.NewServeMux()
	mux.HandleFunc("GET /api/runners", h.list)
	mux.HandleFunc("GET /api/runners/queue", h.queue)
	mux.HandleFunc("GET /api/runners/latest-version", h.latestVersion)
	mux.HandleFunc("POST /api/runners/enrollments", h.createEnrollment)
	mux.HandleFunc("DELETE /api/runners/{id}", h.remove)
	mux.HandleFunc("GET /api/machines", h.listMachines)
	mux.HandleFunc("PATCH /api/machines/{id}", h.updateMachine)
	mux.HandleFunc("POST /api/machines/{id}/discover", h.discoverMachine)
	return mux
}

// PublicRoutes are the endpoints a machine calls without a user session: enrolling with a code, and removing or
// downloading with the runner's own credential. They must sit outside RequireAuth.
func (h *HTTPHandler) PublicRoutes() http.Handler {
	mux := httpx.NewServeMux()
	mux.HandleFunc("GET /api/runners/download/{target}", h.download)
	mux.HandleFunc("POST /api/runners/enroll", h.enroll)
	mux.HandleFunc("POST /api/runners/self/remove", h.removeSelf)
	return mux
}

func (h *HTTPHandler) list(w http.ResponseWriter, r *http.Request) {
	rs, err := h.svc.ListRunners(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, rs)
}

func (h *HTTPHandler) queue(w http.ResponseWriter, r *http.Request) {
	q, err := h.svc.ListQueue(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, q)
}

func (h *HTTPHandler) latestVersion(w http.ResponseWriter, r *http.Request) {
	tag, err := h.svc.LatestVersion(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"version": tag})
}

// createEnrollmentRequest is the POST /api/runners/enrollments wire shape.
type createEnrollmentRequest struct {
	Name    string `json:"name"`
	Machine string `json:"machine,omitempty"`
}

func (h *HTTPHandler) createEnrollment(w http.ResponseWriter, r *http.Request) {
	var req createEnrollmentRequest
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

func (h *HTTPHandler) enroll(w http.ResponseWriter, r *http.Request) {
	var req EnrollRequest
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

func (h *HTTPHandler) remove(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.RemoveRunner(r.Context(), r.PathValue("id")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *HTTPHandler) removeSelf(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.RemoveSelf(r.Context(), bearerToken(r)); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *HTTPHandler) listMachines(w http.ResponseWriter, r *http.Request) {
	ms, err := h.svc.ListMachines(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, ms)
}

// updateMachineRequest is the PATCH /api/machines/{id} wire shape; either field may be left empty to keep it.
type updateMachineRequest struct {
	Name      string `json:"name,omitempty"`
	StackRoot string `json:"stack_root,omitempty"`
}

func (h *HTTPHandler) updateMachine(w http.ResponseWriter, r *http.Request) {
	var req updateMachineRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	m, err := h.svc.UpdateMachine(r.Context(), r.PathValue("id"), req.Name, req.StackRoot)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, m)
}

// discoverMachine dispatches a discover job to the machine and returns the report grouped for the wizard (spec §8).
func (h *HTTPHandler) discoverMachine(w http.ResponseWriter, r *http.Request) {
	grouped, err := h.svc.DiscoverForImport(r.Context(), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, grouped)
}

// download authenticates with a runner's own credential and streams target's binary from the release.
func (h *HTTPHandler) download(w http.ResponseWriter, r *http.Request) {
	cred, err := authenticate(r.Context(), h.svc.repo, bearerToken(r))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	if cred.Revoked {
		httpx.WriteError(w, errRunnerRemoved)
		return
	}
	target := r.PathValue("target")
	asset, err := h.svc.Download(r.Context(), target, r.URL.Query().Get("version"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	defer func() {
		if cerr := asset.Body.Close(); cerr != nil {
			h.log.Debug("close download asset body", "target", target, "error", cerr)
		}
	}()
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", asset.Name))
	if asset.Size > 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(asset.Size, 10))
	}
	if asset.Sha256 != "" {
		w.Header().Set("X-Checksum-Sha256", asset.Sha256)
	}
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, asset.Body); err != nil {
		h.log.Warn("runner download stream failed", "target", target, "error", err)
	}
}
