package calendar

import (
	"fmt"
	"net/http"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/httpx"
)

// Handler adapts the calendar use-cases to the HTTP/JSON gateway (ADR 0019).
type Handler struct {
	svc *Service
}

// NewHandler wires the calendar gateway over svc.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Routes returns the calendar endpoints, mounted under /api/workspaces/{workspaceID}/meetings and /api/meetings.
func (h *Handler) Routes() http.Handler {
	mux := httpx.NewServeMux()
	mux.HandleFunc("GET /api/workspaces/{workspaceID}/meetings", h.calendar)
	mux.HandleFunc("POST /api/workspaces/{workspaceID}/meetings", h.create)
	mux.HandleFunc("GET /api/meetings/{id}", h.get)
	mux.HandleFunc("PATCH /api/meetings/{id}", h.update)
	mux.HandleFunc("PUT /api/meetings/{id}/reply", h.reply)
	return mux
}

// detailsRequest is a meeting's fields on the wire; an omitted field keeps its value, and an empty repeat_until
// clears the series' end.
type detailsRequest struct {
	Title          *string    `json:"title"`
	Notes          *string    `json:"notes"`
	StartsAt       *time.Time `json:"starts_at"`
	EndsAt         *time.Time `json:"ends_at"`
	Timezone       *string    `json:"timezone"`
	VoiceChannelID *string    `json:"voice_channel_id"`
	Link           *string    `json:"link"`
	Repeats        *Repeat    `json:"repeats"`
	RepeatUntil    *string    `json:"repeat_until"`
	RemindMinutes  *int       `json:"remind_minutes"`
	InviteeIDs     *[]string  `json:"invitee_ids"`
}

func (d detailsRequest) details() (Details, error) {
	out := Details{
		Title: d.Title, Notes: d.Notes, StartsAt: d.StartsAt, EndsAt: d.EndsAt, Timezone: d.Timezone,
		VoiceChannelID: d.VoiceChannelID, Link: d.Link, Repeats: d.Repeats, RemindMinutes: d.RemindMinutes, InviteeIDs: d.InviteeIDs,
	}
	until, err := ParseRepeatUntil(d.RepeatUntil)
	if err != nil {
		return Details{}, err
	}
	out.RepeatUntil = until
	return out, nil
}

// ParseRepeatUntil reads a series' end as RFC 3339; an empty string is the zero time, which clears it.
func ParseRepeatUntil(v *string) (*time.Time, error) {
	if v == nil {
		return nil, nil
	}
	if *v == "" {
		return &time.Time{}, nil
	}
	t, err := time.Parse(time.RFC3339, *v)
	if err != nil {
		return nil, fmt.Errorf("%w: repeat_until is an RFC 3339 time such as 2026-12-31T23:59:59Z", apperrs.ErrInvalid)
	}
	return &t, nil
}

type changeRequest struct {
	detailsRequest
	Cancelled *bool      `json:"cancelled"`
	Slot      *time.Time `json:"slot"`
	Scope     Scope      `json:"scope"`
}

type replyRequest struct {
	Reply Reply `json:"reply"`
}

// calendarResponse is a range of the calendar: the meetings in it and each time they take place.
type calendarResponse struct {
	Meetings    []*Meeting   `json:"meetings"`
	Occurrences []Occurrence `json:"occurrences"`
}

func (h *Handler) calendar(w http.ResponseWriter, r *http.Request) {
	from, err := parseTime(r.URL.Query().Get("from"), "from")
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	to, err := parseTime(r.URL.Query().Get("to"), "to")
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	ms, occurrences, err := h.svc.Calendar(r.Context(), r.PathValue("workspaceID"), from, to)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, calendarResponse{Meetings: ms, Occurrences: occurrences})
}

func parseTime(v, name string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: %s is an RFC 3339 time such as 2026-11-01T00:00:00Z", apperrs.ErrInvalid, name)
	}
	return t, nil
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req detailsRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	d, err := req.details()
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	m, err := h.svc.Create(r.Context(), r.PathValue("workspaceID"), d)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, m)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	m, err := h.svc.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, m)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	var req changeRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	d, err := req.details()
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	m, err := h.svc.Update(r.Context(), r.PathValue("id"), Change{Details: d, Cancelled: req.Cancelled, Slot: req.Slot, Scope: req.Scope})
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, m)
}

func (h *Handler) reply(w http.ResponseWriter, r *http.Request) {
	var req replyRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	m, err := h.svc.Reply(r.Context(), r.PathValue("id"), req.Reply)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, m)
}
