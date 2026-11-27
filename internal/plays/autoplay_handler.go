package plays

import (
	"net/http"

	"github.com/otal-labs/nexul/internal/platform/httpx"
	"github.com/otal-labs/nexul/internal/platform/labels"
)

// autoPlayRequest is a full replace of an auto play's editable fields; a create starts it off whatever enabled says.
type autoPlayRequest struct {
	Enabled           bool       `json:"enabled"`
	Moment            Moment     `json:"moment"`
	MomentStage       *Stage     `json:"moment_stage"`
	Schedule          *Schedule  `json:"schedule"`
	Conditions        Conditions `json:"conditions"`
	Priority          Priority   `json:"priority"`
	OnceWithinMinutes int        `json:"once_within_minutes"`
	RunOn             RunOn      `json:"run_on"`
	// RunAsMe picks the caller as the person a run_on person auto play runs as.
	RunAsMe    bool     `json:"run_as_me"`
	MemoryIDs  []string `json:"memory_ids"`
	OneAtATime bool     `json:"one_at_a_time"`
	Resource   string   `json:"resource"`
	// ComputerRule narrows the run-as person's computers; omitted keeps the auto play's.
	ComputerRule *labels.ComputerRule `json:"computer_rule"`
}

func (req autoPlayRequest) input() AutoPlayInput {
	return AutoPlayInput{
		Moment: req.Moment, MomentStage: req.MomentStage, Schedule: req.Schedule, Conditions: req.Conditions, Priority: req.Priority,
		OnceWithinMinutes: req.OnceWithinMinutes, RunOn: req.RunOn, RunAsMe: req.RunAsMe, MemoryIDs: req.MemoryIDs,
		OneAtATime: req.OneAtATime, Resource: req.Resource, ComputerRule: req.ComputerRule,
	}
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
