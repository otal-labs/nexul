package pairing

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/otal-labs/nexul/internal/platform/containerlog"
	"github.com/otal-labs/nexul/internal/platform/httpx"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
	"github.com/otal-labs/nexul/internal/platform/paging"
)

type runCommandRequest struct {
	Command        string `json:"command"`
	Shell          string `json:"shell"`
	Cwd            string `json:"cwd"`
	TimeoutSeconds int    `json:"timeout_seconds"`
}

func (h *Handler) runCommand(w http.ResponseWriter, r *http.Request) {
	var req runCommandRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	job, err := h.svc.RunCommand(r.Context(), actorID(r), r.PathValue("id"), RunCommandInput{
		Command: req.Command, Shell: req.Shell, Cwd: req.Cwd, Timeout: time.Duration(req.TimeoutSeconds) * time.Second, Adapter: AdapterWeb,
	})
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, job)
}

func (h *Handler) listCommands(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	win := paging.Window{Limit: limit, Offset: offset}.Clamped()
	jobs, total, err := h.svc.ListShellJobs(r.Context(), actorID(r), r.PathValue("id"), win)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, mcptool.PageOf(jobs, total, win))
}

func (h *Handler) getCommand(w http.ResponseWriter, r *http.Request) {
	job, err := h.svc.GetShellJob(r.Context(), actorID(r), r.PathValue("jobID"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, job)
}

func (h *Handler) cancelCommand(w http.ResponseWriter, r *http.Request) {
	job, err := h.svc.CancelShellJob(r.Context(), actorID(r), r.PathValue("jobID"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, job)
}

// CommandOutputSocket streams one command's output from its first line to its computer's owner or the person who ran
// it, live while it runs.
func (h *Handler) CommandOutputSocket(w http.ResponseWriter, r *http.Request) {
	userID, jobID := actorID(r), r.PathValue("jobID")
	containerlog.ServeSocket(w, r, func(ctx context.Context, _ int, _ bool) (containerlog.Feed, error) {
		return h.svc.OpenShellOutput(ctx, userID, jobID)
	})
}
