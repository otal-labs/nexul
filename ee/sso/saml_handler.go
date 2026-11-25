package sso

import (
	"net/http"

	"github.com/otal-labs/nexul/internal/platform/httpx"
)

// maxSAMLBody caps a SAML settings body, which may carry uploaded metadata.
const maxSAMLBody = maxMetadata + 64<<10

func (h *Handler) getSAML(w http.ResponseWriter, r *http.Request) {
	v, err := h.svc.SAMLView(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, v)
}

func decodeSAMLPatch(w http.ResponseWriter, r *http.Request) (SAMLPatch, error) {
	var p SAMLPatch
	r.Body = http.MaxBytesReader(w, r.Body, maxSAMLBody)
	return p, httpx.DecodeJSON(r, &p)
}

func (h *Handler) updateSAML(w http.ResponseWriter, r *http.Request) {
	p, err := decodeSAMLPatch(w, r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	v, err := h.svc.UpdateSAML(r.Context(), p)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, v)
}

func (h *Handler) verifySAML(w http.ResponseWriter, r *http.Request) {
	p, err := decodeSAMLPatch(w, r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	detail, err := h.svc.CheckSAML(r.Context(), r.URL.Query().Get("check"), p)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	if detail == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"detail": detail})
}

func (h *Handler) removeSAML(w http.ResponseWriter, r *http.Request) {
	v, err := h.svc.RemoveSAML(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, v)
}

// spMetadata serves the service provider metadata an identity provider fetches or the Owner downloads.
func (h *Handler) spMetadata(w http.ResponseWriter, r *http.Request) {
	body, err := h.svc.SPMetadata(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/samlmetadata+xml")
	_, _ = w.Write(body)
}
