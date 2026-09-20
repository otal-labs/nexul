package runner

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
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
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/runners", h.list)
	mux.HandleFunc("GET /api/runners/queue", h.queue)
	mux.HandleFunc("GET /api/runners/install", h.install)
	mux.HandleFunc("GET /api/runners/latest-version", h.latestVersion)
	mux.HandleFunc("GET /api/machines", h.listMachines)
	mux.HandleFunc("PATCH /api/machines/{id}", h.updateMachine)
	mux.HandleFunc("POST /api/machines/{id}/discover", h.discoverMachine)
	return mux
}

// PublicRoutes returns the runner download endpoint, auth'd by the runner secret, so it must sit outside RequireAuth.
func (h *HTTPHandler) PublicRoutes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/runners/download/{target}", h.download)
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

func (h *HTTPHandler) install(w http.ResponseWriter, r *http.Request) {
	info, err := h.svc.Install(r.Context(), r.Host, r.TLS != nil)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, info)
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

// download authenticates with the shared runner secret and streams target's binary from the latest release.
func (h *HTTPHandler) download(w http.ResponseWriter, r *http.Request) {
	secret, err := h.svc.repo.Secret(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !constantTimeEqual(token, secret) {
		httpx.WriteError(w, apperrs.ErrUnauthorized)
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
