package tickets

import (
	"net/http"

	"github.com/otal-labs/nexul/internal/platform/httpx"
)

func (h *Handler) testTarget(w http.ResponseWriter, r *http.Request) {
	target, err := h.svc.TestTarget(r.Context(), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, target)
}

func (h *Handler) testPass(w http.ResponseWriter, r *http.Request) {
	t, err := h.svc.TestPass(r.Context(), r.PathValue("id"), false)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, t)
}

func (h *Handler) testFail(w http.ResponseWriter, r *http.Request) {
	var req TestReport
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	t, err := h.svc.TestFail(r.Context(), r.PathValue("id"), req, false)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, t)
}
