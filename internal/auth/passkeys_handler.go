package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/otal-labs/nexul/internal/platform/httpx"
)

// passkeyBodyMax bounds a browser's passkey response, which the public sign-in finish reads before it knows anyone.
const passkeyBodyMax = 64 << 10

type passkeyFinishRequest struct {
	Ceremony   string          `json:"ceremony"`
	Credential json.RawMessage `json:"credential"`
}

func decodePasskeyFinish(w http.ResponseWriter, r *http.Request) (passkeyFinishRequest, error) {
	r.Body = http.MaxBytesReader(w, r.Body, passkeyBodyMax)
	var req passkeyFinishRequest
	err := httpx.DecodeJSON(r, &req)
	return req, err
}

func passkeyCtx(r *http.Request) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), 10*time.Second)
}

func (h *Handler) startPasskeyLogin(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := passkeyCtx(r)
	defer cancel()
	options, ceremony, err := h.svc.StartPasskeyLogin(WithDevice(ctx, DeviceFromRequest(r)), h.spaOrigin(r))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"ceremony": ceremony, "options": options})
}

func (h *Handler) finishPasskeyLogin(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := passkeyCtx(r)
	defer cancel()
	req, err := decodePasskeyFinish(w, r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	token, err := h.svc.FinishPasskeyLogin(WithDevice(ctx, DeviceFromRequest(r)), h.spaOrigin(r), req.Ceremony, req.Credential)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"token": token})
}

func (h *Handler) listPasskeys(w http.ResponseWriter, r *http.Request) {
	if _, err := requireSession(r); err != nil {
		httpx.WriteError(w, err)
		return
	}
	passkeys, err := h.svc.ListPasskeys(r.Context(), currentUserID(r))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"passkeys": passkeys, "available": h.svc.PasskeysAvailable(h.spaOrigin(r))})
}

func (h *Handler) startPasskeyRegistration(w http.ResponseWriter, r *http.Request) {
	if _, err := requireSession(r); err != nil {
		httpx.WriteError(w, err)
		return
	}
	ctx, cancel := passkeyCtx(r)
	defer cancel()
	options, ceremony, err := h.svc.StartPasskeyRegistration(WithDevice(ctx, DeviceFromRequest(r)), currentUserID(r), h.spaOrigin(r))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"ceremony": ceremony, "options": options})
}

func (h *Handler) addPasskey(w http.ResponseWriter, r *http.Request) {
	if _, err := requireSession(r); err != nil {
		httpx.WriteError(w, err)
		return
	}
	ctx, cancel := passkeyCtx(r)
	defer cancel()
	req, err := decodePasskeyFinish(w, r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	passkey, err := h.svc.FinishPasskeyRegistration(WithDevice(ctx, DeviceFromRequest(r)), currentUserID(r), h.spaOrigin(r), req.Ceremony, req.Credential)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, passkey)
}

func (h *Handler) renamePasskey(w http.ResponseWriter, r *http.Request) {
	if _, err := requireSession(r); err != nil {
		httpx.WriteError(w, err)
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if err := h.svc.RenamePasskey(r.Context(), currentUserID(r), r.PathValue("id"), req.Name); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) deletePasskey(w http.ResponseWriter, r *http.Request) {
	if _, err := requireSession(r); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if err := h.svc.DeletePasskey(r.Context(), currentUserID(r), r.PathValue("id")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
