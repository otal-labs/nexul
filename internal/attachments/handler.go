package attachments

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strconv"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/httpx"
	"github.com/otal-labs/nexul/internal/platform/logging"
)

// Handler's file bytes are the one non-JSON response in the gateway.
type Handler struct {
	svc *Service
}

// NewHandler wires the attachments REST gateway over the given service.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Routes returns the attachments endpoints: multipart upload, owner-scoped list, raw bytes, delete.
func (h *Handler) Routes() http.Handler {
	mux := httpx.NewServeMux()
	mux.HandleFunc("POST /api/attachments", h.upload)
	mux.HandleFunc("GET /api/attachments", h.list)
	mux.HandleFunc("GET /api/attachments/{id}", h.serve)
	mux.HandleFunc("DELETE /api/attachments/{id}", h.delete)
	return mux
}

func (h *Handler) upload(w http.ResponseWriter, r *http.Request) {
	// The slack over MaxSize covers multipart framing and the owner fields; the use-case enforces the exact file cap.
	r.Body = http.MaxBytesReader(w, r.Body, MaxSize+64<<10)
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		httpx.WriteError(w, invalidUpload(err))
		return
	}
	f, _, err := r.FormFile("file")
	if err != nil {
		httpx.WriteError(w, fmt.Errorf("%w: file is required", apperrs.ErrInvalid))
		return
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			logging.FromCtx(r.Context()).Debug("close uploaded file", "error", cerr)
		}
	}()
	data, err := io.ReadAll(f)
	if err != nil {
		httpx.WriteError(w, invalidUpload(err))
		return
	}
	// name lets pasted screenshots get a readable name; multipart filename is the fallback.
	name := r.FormValue("name")
	if name == "" {
		name = r.MultipartForm.File["file"][0].Filename
	}
	owner := Owner{
		DocID:          r.FormValue("doc_id"),
		TicketID:       r.FormValue("ticket_id"),
		ConversationID: r.FormValue("conversation_id"),
		MemoryID:       r.FormValue("memory_id"),
	}
	a, err := h.svc.Upload(r.Context(), owner, name, data)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, a)
}

func invalidUpload(err error) error {
	var tooBig *http.MaxBytesError
	if errors.As(err, &tooBig) {
		return fmt.Errorf("%w: file exceeds %d bytes", apperrs.ErrInvalid, MaxSize)
	}
	return fmt.Errorf("%w: invalid multipart body", apperrs.ErrInvalid)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	owner := Owner{
		DocID:          r.URL.Query().Get("doc_id"),
		TicketID:       r.URL.Query().Get("ticket_id"),
		ConversationID: r.URL.Query().Get("conversation_id"),
		MemoryID:       r.URL.Query().Get("memory_id"),
	}
	as, err := h.svc.List(r.Context(), owner)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, as)
}

// serve sets nosniff so a stored file can never be re-interpreted as HTML on this origin.
func (h *Handler) serve(w http.ResponseWriter, r *http.Request) {
	a, err := h.svc.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	disposition := "attachment"
	if a.Inline() {
		disposition = "inline"
	}
	w.Header().Set("Content-Type", a.ContentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(a.Data)))
	w.Header().Set("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{"filename": a.Name}))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	// IDs are unique and bytes never change, so the browser may keep a copy for as long as it likes.
	w.Header().Set("Cache-Control", "private, max-age=31536000, immutable")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(a.Data)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), r.PathValue("id")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
