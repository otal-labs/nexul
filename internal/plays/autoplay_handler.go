package plays

import (
	"net/http"

	"github.com/otal-labs/nexul/internal/platform/httpx"
)

// autoPlayRequest is a full replace of an auto play's editable fields; a create starts it off whatever enabled says.
type autoPlayRequest struct {
	Enabled           bool       `json:"enabled"`
	Moment            Moment     `json:"moment"`
	MomentStage       *Stage     `json:"moment_stage"`
	Conditions        Conditions `json:"conditions"`
	Priority          Priority   `json:"priority"`
	OnceWithinMinutes int        `json:"once_within_minutes"`
	RunOn             RunOn      `json:"run_on"`
}

func (req autoPlayRequest) input() AutoPlayInput {
	return AutoPlayInput{
		Moment: req.Moment, MomentStage: req.MomentStage, Conditions: req.Conditions, Priority: req.Priority,
		OnceWithinMinutes: req.OnceWithinMinutes, RunOn: req.RunOn,
	}
}

type autoPlayLimits struct {
	DailyCapPerTicket int `json:"daily_cap_per_ticket"`
}

func (h *Handler) listAutoPlays(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListAutoPlays(r.Context(), r.PathValue("workspaceID"), r.PathValue("playID"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, list)
}

func (h *Handler) createAutoPlay(w http.ResponseWriter, r *http.Request) {
	var req autoPlayRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	a, err := h.svc.CreateAutoPlay(r.Context(), r.PathValue("workspaceID"), r.PathValue("playID"), req.input())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, a)
}

func (h *Handler) updateAutoPlay(w http.ResponseWriter, r *http.Request) {
	var req autoPlayRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	a, err := h.svc.UpdateAutoPlay(r.Context(), r.PathValue("workspaceID"), r.PathValue("playID"), r.PathValue("autoPlayID"), req.Enabled, req.input())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, a)
}

func (h *Handler) deleteAutoPlay(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteAutoPlay(r.Context(), r.PathValue("workspaceID"), r.PathValue("playID"), r.PathValue("autoPlayID")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) getAutoPlayLimits(w http.ResponseWriter, r *http.Request) {
	limit, err := h.svc.AutoPlayDailyCap(r.Context(), r.PathValue("workspaceID"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, autoPlayLimits{DailyCapPerTicket: limit})
}

func (h *Handler) setAutoPlayLimits(w http.ResponseWriter, r *http.Request) {
	var req autoPlayLimits
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	limit, err := h.svc.SetAutoPlayDailyCap(r.Context(), r.PathValue("workspaceID"), req.DailyCapPerTicket)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, autoPlayLimits{DailyCapPerTicket: limit})
}
