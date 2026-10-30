package automations

import (
	"net/http"
	"strconv"

	"github.com/otal-labs/nexul/internal/platform/httpx"
)

// versionRoutes registers the version-history endpoints, called only when a VersionsService is wired.
func (h *Handler) versionRoutes(mux *httpx.ServeMux) {
	mux.HandleFunc("POST /api/automations/{id}/versions", h.pushVersion)
	mux.HandleFunc("GET /api/automations/{id}/history", h.history)
	mux.HandleFunc("GET /api/automations/{id}/versions", h.listVersions)
	mux.HandleFunc("GET /api/automations/{id}/versions/diff", h.diffVersions)
	mux.HandleFunc("GET /api/automations/{id}/versions/{versionID}", h.getVersion)
	mux.HandleFunc("POST /api/automations/{id}/versions/{versionID}/merge", h.mergeVersion)
	mux.HandleFunc("POST /api/automations/{id}/versions/{versionID}/rollback", h.rollbackVersion)
}

type pushVersionRequest struct {
	Code    string `json:"code"`
	Source  string `json:"source"`
	Message string `json:"message"`
}

func (h *Handler) pushVersion(w http.ResponseWriter, r *http.Request) {
	var req pushVersionRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	v, err := h.versions.Push(r.Context(), actorID(r), r.PathValue("id"), req.Code, req.Source, req.Message)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, v)
}

func (h *Handler) listVersions(w http.ResponseWriter, r *http.Request) {
	list, err := h.versions.List(r.Context(), actorID(r), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, list)
}

func (h *Handler) history(w http.ResponseWriter, r *http.Request) {
	out, err := h.versions.History(r.Context(), actorID(r), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

// getVersion answers a version id with the whole version, and a number with the history dialog's shape: that sequence with
// its code as the body. An id is a UUID, never all digits.
func (h *Handler) getVersion(w http.ResponseWriter, r *http.Request) {
	if sequence, err := strconv.Atoi(r.PathValue("versionID")); err == nil {
		out, err := h.versions.VersionBody(r.Context(), actorID(r), r.PathValue("id"), sequence)
		if err != nil {
			httpx.WriteError(w, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, out)
		return
	}
	v, err := h.versions.Get(r.Context(), actorID(r), r.PathValue("id"), r.PathValue("versionID"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, v)
}

// diffResponse pairs active/pending versions for the UI to diff client-side; pending is nil when none.
type diffResponse struct {
	Active  *Version `json:"active"`
	Pending *Version `json:"pending"`
}

func (h *Handler) diffVersions(w http.ResponseWriter, r *http.Request) {
	active, pending, err := h.versions.Diff(r.Context(), actorID(r), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, diffResponse{Active: active, Pending: pending})
}

func (h *Handler) mergeVersion(w http.ResponseWriter, r *http.Request) {
	v, err := h.versions.Activate(r.Context(), actorID(r), r.PathValue("id"), r.PathValue("versionID"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, v)
}

func (h *Handler) rollbackVersion(w http.ResponseWriter, r *http.Request) {
	v, err := h.versions.Activate(r.Context(), actorID(r), r.PathValue("id"), r.PathValue("versionID"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, v)
}
