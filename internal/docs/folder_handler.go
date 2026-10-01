package docs

import (
	"net/http"

	"github.com/otal-labs/nexul/internal/platform/httpx"
)

type saveFolderRequest struct {
	ProjectID string `json:"project_id"`
	Name      string `json:"name"`
}

type moveDocRequest struct {
	FolderID string `json:"folder_id"`
}

func (h *Handler) listFolders(w http.ResponseWriter, r *http.Request) {
	folders, err := h.svc.ListFolders(r.Context(), r.URL.Query().Get("project_id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, folders)
}

func (h *Handler) createFolder(w http.ResponseWriter, r *http.Request) {
	var req saveFolderRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	f, err := h.svc.CreateFolder(r.Context(), req.ProjectID, req.Name)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, f)
}

func (h *Handler) renameFolder(w http.ResponseWriter, r *http.Request) {
	var req saveFolderRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	f, err := h.svc.RenameFolder(r.Context(), r.PathValue("id"), req.Name)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, f)
}

// deleteFolder moves the folder's docs to the project's default folder; it never deletes a doc.
func (h *Handler) deleteFolder(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteFolder(r.Context(), r.PathValue("id")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) move(w http.ResponseWriter, r *http.Request) {
	var req moveDocRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	d, err := h.svc.MoveToFolder(r.Context(), r.PathValue("id"), req.FolderID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, d)
}
