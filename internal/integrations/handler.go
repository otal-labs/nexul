package integrations

import (
	"net/http"
	"strconv"

	"github.com/otal-labs/nexul/internal/platform/httpx"
	"github.com/otal-labs/nexul/internal/platform/identity"
)

// Handler adapts integrations use-cases to the HTTP/JSON gateway (ADR 0019); token auth lives in RequireIntegration.
type Handler struct {
	svc *Service
}

// NewHandler wires the integrations REST gateway over the given service.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// ActorID resolves the acting user id, set by withIdentity or by RequireIntegration's install.CreatedBy.
func ActorID(r *http.Request) string {
	if a, ok := identity.ActorFromCtx(r.Context()); ok {
		return a.ID
	}
	return ""
}

type installRequest struct {
	Name       string   `json:"name"`
	TrustTier  string   `json:"trust_tier"`
	WebhookURL string   `json:"webhook_url"`
	Scopes     []string `json:"scopes"`
}

type subscribeRequest struct {
	Topic string `json:"topic"`
}

// Routes returns the integrations REST endpoints.
func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/integrations", h.install)
	mux.HandleFunc("GET /api/integrations", h.list)
	mux.HandleFunc("GET /api/integrations/{id}", h.get)
	mux.HandleFunc("DELETE /api/integrations/{id}", h.revoke)
	mux.HandleFunc("POST /api/integrations/{id}/subscriptions", h.subscribe)
	mux.HandleFunc("GET /api/integrations/{id}/subscriptions", h.listSubscriptions)
	mux.HandleFunc("DELETE /api/integrations/{id}/subscriptions/{topic}", h.unsubscribe)
	mux.HandleFunc("GET /api/integrations/{id}/deliveries", h.listDeliveries)
	mux.HandleFunc("GET /api/integrations/scopes", h.scopes)
	mux.HandleFunc("GET /api/events/catalog", h.catalog)
	mux.HandleFunc("GET /api/audit", h.audit)
	return mux
}

func (h *Handler) install(w http.ResponseWriter, r *http.Request) {
	var req installRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	raw, secret, install, err := h.svc.Install(r.Context(), ActorID(r), req.Name, TrustTier(req.TrustTier), req.WebhookURL, toScopes(req.Scopes))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{
		"install":        install,
		"token":          raw,
		"webhook_secret": secret,
	})
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	installs, err := h.svc.ListInstalls(r.Context(), ActorID(r))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"installs": installs})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	install, err := h.svc.GetInstall(r.Context(), ActorID(r), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, install)
}

func (h *Handler) revoke(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.RevokeInstall(r.Context(), ActorID(r), r.PathValue("id")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) subscribe(w http.ResponseWriter, r *http.Request) {
	var req subscribeRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if err := h.svc.Subscribe(r.Context(), ActorID(r), r.PathValue("id"), req.Topic); err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]string{"status": "subscribed"})
}

func (h *Handler) listSubscriptions(w http.ResponseWriter, r *http.Request) {
	subs, err := h.svc.ListSubscriptions(r.Context(), ActorID(r), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"subscriptions": subs})
}

func (h *Handler) unsubscribe(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Unsubscribe(r.Context(), ActorID(r), r.PathValue("id"), r.PathValue("topic")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listDeliveries(w http.ResponseWriter, r *http.Request) {
	deliveries, err := h.svc.ListDeliveries(r.Context(), ActorID(r), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"deliveries": deliveries})
}

func (h *Handler) catalog(w http.ResponseWriter, r *http.Request) {
	entries, err := h.svc.Catalog(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"catalog": entries})
}

func (h *Handler) scopes(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"scopes": Catalog()})
}

func (h *Handler) audit(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	entries, err := h.svc.ListAudit(r.Context(), ActorID(r), limit)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"audit": entries})
}

func toScopes(raw []string) []Scope {
	out := make([]Scope, len(raw))
	for i, s := range raw {
		out[i] = Scope(s)
	}
	return out
}
