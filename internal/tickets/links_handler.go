package tickets

import (
	"net/http"

	"github.com/otal-labs/nexul/internal/platform/httpx"
)

type setFoundInRequest struct {
	OriginID      string `json:"origin_id"`
	OriginUnknown bool   `json:"origin_unknown"`
}

type addBlockerRequest struct {
	BlockerID string `json:"blocker_id"`
}

func (h *Handler) ticketLinks(w http.ResponseWriter, r *http.Request) {
	writeLinkSet(w)(h.svc.Links(r.Context(), r.PathValue("id")))
}

func (h *Handler) setFoundIn(w http.ResponseWriter, r *http.Request) {
	var req setFoundInRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	writeLinkSet(w)(h.svc.SetFoundIn(r.Context(), r.PathValue("id"), req.OriginID, req.OriginUnknown))
}

func (h *Handler) removeFoundIn(w http.ResponseWriter, r *http.Request) {
	writeLinkSet(w)(h.svc.RemoveFoundIn(r.Context(), r.PathValue("id")))
}

func (h *Handler) addBlocker(w http.ResponseWriter, r *http.Request) {
	var req addBlockerRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	writeLinkSet(w)(h.svc.AddBlocker(r.Context(), r.PathValue("id"), req.BlockerID))
}

func (h *Handler) removeBlocker(w http.ResponseWriter, r *http.Request) {
	writeLinkSet(w)(h.svc.RemoveBlocker(r.Context(), r.PathValue("id"), r.PathValue("blockerId")))
}

func (h *Handler) unclearedBlockers(w http.ResponseWriter, r *http.Request) {
	out, err := h.svc.UnclearedBlockers(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

// writeLinkSet returns a writer taking a link use-case's two results directly.
func writeLinkSet(w http.ResponseWriter) func(*LinkSet, error) {
	return func(set *LinkSet, err error) {
		if err != nil {
			httpx.WriteError(w, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, set)
	}
}
