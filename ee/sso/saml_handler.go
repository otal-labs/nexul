package sso

import (
	"context"
	"net/http"
	"time"

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

func (h *Handler) rotateSAML(w http.ResponseWriter, r *http.Request) {
	h.answer(w)(h.svc.RotateSAMLCertificate(r.Context()))
}

func (h *Handler) retireSAML(w http.ResponseWriter, r *http.Request) {
	h.answer(w)(h.svc.RetireSAMLCertificate(r.Context()))
}

// answer writes a use-case's view, or its error.
func (h *Handler) answer(w http.ResponseWriter) func(SAMLView, error) {
	return func(v SAMLView, err error) {
		if err != nil {
			httpx.WriteError(w, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, v)
	}
}

// startSAML sends the browser to the provider with a signed AuthnRequest whose id the response must name.
func (h *Handler) startSAML(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	target, err := h.svc.StartSAML(ctx)
	if err != nil {
		h.failed(w, r, signInCode(err), err)
		return
	}
	http.Redirect(w, r, target, http.StatusFound)
}

// acs takes the provider's posted response, the assertion consumer service, and lands on /login with the session token
// as the other callbacks do. The post is cross-site, so no cookie ties it to the browser; its InResponseTo does.
func (h *Handler) acs(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	r = r.WithContext(ctx)
	r.Body = http.MaxBytesReader(w, r.Body, maxSAMLResponse+64<<10)
	if err := r.ParseForm(); err != nil {
		h.failed(w, r, codeFailed, err)
		return
	}
	userID, err := h.svc.CompleteSAML(ctx, r.PostForm.Get("SAMLResponse"))
	if err != nil {
		h.failed(w, r, signInCode(err), err)
		return
	}
	token, err := h.browser.SSOSessionFor(r, userID)
	if err != nil {
		h.failed(w, r, codeFailed, err)
		return
	}
	http.Redirect(w, r, h.browser.SPAOrigin(r)+"/login?token="+token, http.StatusSeeOther)
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
