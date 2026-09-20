package voice

import (
	"io"
	"log/slog"
	"net/http"
)

// WebhookHandler verifies LiveKit webhook deliveries and applies them to occupancy; a public route.
type WebhookHandler struct {
	svc         *Service
	credentials CredentialSource
	log         *slog.Logger
}

// NewWebhookHandler wires the receiver; credentials is consulted per request, so no restart is needed after a save.
func NewWebhookHandler(svc *Service, credentials CredentialSource, logger *slog.Logger) *WebhookHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &WebhookHandler{svc: svc, credentials: credentials, log: logger}
}

func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read webhook body", http.StatusBadRequest)
		return
	}
	client, err := h.credentials.LiveKit(r.Context())
	if err != nil {
		// No connector configured; 200 so LiveKit doesn't spend its retry budget on a delivery we can't use.
		h.log.Warn("livekit webhook received with no connector configured", "error", err)
		w.WriteHeader(http.StatusOK)
		return
	}
	ev, err := client.VerifyWebhook(body, r.Header.Get("Authorization"))
	if err != nil {
		h.log.Warn("livekit webhook verification failed", "error", err)
		http.Error(w, "invalid webhook signature", http.StatusUnauthorized)
		return
	}
	if err := h.svc.HandleWebhook(r.Context(), ev); err != nil {
		h.log.Error("livekit webhook handling failed", "error", err)
		http.Error(w, "webhook handling failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
