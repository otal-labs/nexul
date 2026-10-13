package welcome

import (
	"net/http"

	"github.com/otal-labs/nexul/internal/platform/httpx"
	"github.com/otal-labs/nexul/internal/platform/identity"
)

// Handler adapts the welcome use-cases to the HTTP/JSON gateway (ADR 0019); every route acts on the caller's own steps.
type Handler struct {
	svc *Service
}

// NewHandler wires the welcome REST gateway over the given service.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Routes are mounted behind RequireAuth at /api/me.
func (h *Handler) Routes() http.Handler {
	mux := httpx.NewServeMux()
	mux.HandleFunc("GET /api/me/welcome", h.get)
	mux.HandleFunc("POST /api/me/welcome/first-run", h.firstRun)
	mux.HandleFunc("POST /api/me/welcome/{step}/skip", h.skip)
	mux.HandleFunc("DELETE /api/me/welcome/{step}/skip", h.unskip)
	return mux
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	respond(w)(h.svc.Progress(r.Context(), actorID(r)))
}

func (h *Handler) firstRun(w http.ResponseWriter, r *http.Request) {
	respond(w)(h.svc.FinishFirstRun(r.Context(), actorID(r)))
}

func (h *Handler) skip(w http.ResponseWriter, r *http.Request) {
	respond(w)(h.svc.Skip(r.Context(), actorID(r), r.PathValue("step")))
}

func (h *Handler) unskip(w http.ResponseWriter, r *http.Request) {
	respond(w)(h.svc.Unskip(r.Context(), actorID(r), r.PathValue("step")))
}

func respond(w http.ResponseWriter) func(Progress, error) {
	return func(p Progress, err error) {
		if err != nil {
			httpx.WriteError(w, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, p)
	}
}

func actorID(r *http.Request) string {
	if a, ok := identity.ActorFromCtx(r.Context()); ok {
		return a.ID
	}
	return ""
}
