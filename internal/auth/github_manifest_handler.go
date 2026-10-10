package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/httpx"
)

func (h *Handler) startGitHubManifest(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	out, err := h.svc.StartGitHubManifest(ctx, currentUserID(r), strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusOK, out)
}

type manifestCallbackRequest struct {
	State string `json:"state"`
	Code  string `json:"code"`
}

func (h *Handler) completeGitHubManifest(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	var in manifestCallbackRequest
	if err := httpx.DecodeJSON(r, &in); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if err := h.svc.CompleteGitHubManifest(ctx, currentUserID(r), strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "), in.State, in.Code); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) returnGitHubManifest(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if q.Get("code") == "" || len(q.Get("code")) > 256 || q.Get("state") == "" || len(q.Get("state")) > 200 {
		httpx.WriteError(w, fmt.Errorf("%w: invalid GitHub registration callback", apperrs.ErrInvalid))
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	fragment := url.Values{"github_manifest": {"1"}, "code": {q.Get("code")}, "state": {q.Get("state")}}.Encode()
	http.Redirect(w, r, h.spaOrigin(r)+"/setup#"+fragment, http.StatusSeeOther)
}
