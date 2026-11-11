package branding

import (
	"errors"
	"io"
	"net/http"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/httpx"
)

// Handler adapts the branding use-cases to the HTTP/JSON gateway (ADR 0019).
type Handler struct {
	svc *Service
}

// NewHandler wires the branding REST gateway over the given service.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Routes returns the Owner's endpoints: read the saved branding, change it, replace or remove the logo, clear it all.
func (h *Handler) Routes() http.Handler {
	mux := httpx.NewServeMux()
	mux.HandleFunc("GET /api/instance/branding", h.get)
	mux.HandleFunc("PATCH /api/instance/branding", h.patch)
	mux.HandleFunc("DELETE /api/instance/branding", h.clear)
	mux.HandleFunc("PUT /api/instance/branding/logo", h.setLogo)
	mux.HandleFunc("DELETE /api/instance/branding/logo", h.removeLogo)
	return mux
}

// PublicRoutes returns the two reads a signed-out page needs: the look and the logo.
func (h *Handler) PublicRoutes() http.Handler {
	mux := httpx.NewServeMux()
	mux.HandleFunc("GET /api/branding", h.look)
	mux.HandleFunc("GET /api/branding/logo", h.logo)
	return mux
}

type patchRequest struct {
	ProductName *string `json:"product_name"`
	Accent      *string `json:"accent"`
	ThemeFile   *string `json:"theme_file"`
	RemoveTheme string  `json:"remove_theme"`
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	st, err := h.svc.Get(r.Context())
	h.reply(w, st, err)
}

func (h *Handler) patch(w http.ResponseWriter, r *http.Request) {
	var req patchRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	st, err := h.svc.Update(r.Context(), Patch{ProductName: req.ProductName, Accent: req.Accent, Theme: req.ThemeFile, RemoveTheme: req.RemoveTheme})
	if errors.Is(err, apperrs.ErrInvalid) {
		httpx.WriteFieldError(w, err, fieldOf(req))
		return
	}
	h.reply(w, st, err)
}

// fieldOf names the field a refusal belongs to, by which one the request carried.
func fieldOf(req patchRequest) string {
	switch {
	case req.ThemeFile != nil:
		return "theme_file"
	case req.Accent != nil:
		return "accent"
	case req.ProductName != nil:
		return "product_name"
	}
	return "remove_theme"
}

func (h *Handler) clear(w http.ResponseWriter, r *http.Request) {
	st, err := h.svc.Clear(r.Context())
	h.reply(w, st, err)
}

func (h *Handler) setLogo(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, MaxLogoBytes+1))
	if err != nil || len(data) > MaxLogoBytes {
		httpx.WriteFieldError(w, validateLogo("image/png", make([]byte, MaxLogoBytes+1)), "logo")
		return
	}
	st, err := h.svc.SetLogo(r.Context(), data)
	if errors.Is(err, apperrs.ErrInvalid) {
		httpx.WriteFieldError(w, err, "logo")
		return
	}
	h.reply(w, st, err)
}

func (h *Handler) removeLogo(w http.ResponseWriter, r *http.Request) {
	st, err := h.svc.RemoveLogo(r.Context())
	h.reply(w, st, err)
}

func (h *Handler) reply(w http.ResponseWriter, st Settings, err error) {
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, st)
}

func (h *Handler) look(w http.ResponseWriter, r *http.Request) {
	l, err := h.svc.Look(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-cache")
	httpx.WriteJSON(w, http.StatusOK, l)
}

// logo serves the image sandboxed: an SVG logo can never run a script, even when opened on its own.
func (h *Handler) logo(w http.ResponseWriter, r *http.Request) {
	contentType, data, err := h.svc.Logo(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	hdr := w.Header()
	hdr.Set("Content-Type", contentType)
	hdr.Set("X-Content-Type-Options", "nosniff")
	hdr.Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; sandbox")
	hdr.Set("Cache-Control", "no-cache")
	if r.URL.Query().Get("v") != "" {
		hdr.Set("Cache-Control", "public, max-age=31536000, immutable")
	}
	_, _ = w.Write(data)
}
