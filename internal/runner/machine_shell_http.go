package runner

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/otal-labs/nexul/internal/platform/containerlog"
	"github.com/otal-labs/nexul/internal/platform/httpx"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
	"github.com/otal-labs/nexul/internal/platform/paging"
	"github.com/otal-labs/nexul/internal/platform/shelljob"
)

type runCommandRequest struct {
	Command        string `json:"command"`
	Shell          string `json:"shell"`
	Cwd            string `json:"cwd"`
	TimeoutSeconds int    `json:"timeout_seconds"`
}

func (h *HTTPHandler) runCommand(w http.ResponseWriter, r *http.Request) {
	var req runCommandRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	job, err := h.svc.RunCommand(r.Context(), r.PathValue("id"), shelljob.Input{
		Command: req.Command, Shell: req.Shell, Cwd: req.Cwd, Timeout: time.Duration(req.TimeoutSeconds) * time.Second,
		Adapter: shelljob.AdapterWeb,
	})
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, job)
}

func (h *HTTPHandler) listCommands(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	win := paging.Window{Limit: limit, Offset: offset}.Clamped()
	jobs, total, err := h.svc.ListShellJobs(r.Context(), r.PathValue("id"), win)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, mcptool.PageOf(jobs, total, win))
}

func (h *HTTPHandler) getCommand(w http.ResponseWriter, r *http.Request) {
	job, err := h.svc.GetShellJob(r.Context(), r.PathValue("jobID"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, job)
}

func (h *HTTPHandler) cancelCommand(w http.ResponseWriter, r *http.Request) {
	job, err := h.svc.CancelShellJob(r.Context(), r.PathValue("jobID"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, job)
}

type setShellRequest struct {
	Enabled bool `json:"enabled"`
}

func (h *HTTPHandler) setShell(w http.ResponseWriter, r *http.Request) {
	var req setShellRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	m, err := h.svc.SetMachineShell(r.Context(), r.PathValue("id"), req.Enabled)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, m)
}

// CommandOutputSocket streams one command on a machine's output from its first line, live while it runs, to a person
// holding Run commands on the machine.
func (h *HTTPHandler) CommandOutputSocket(w http.ResponseWriter, r *http.Request) {
	jobID := r.PathValue("jobID")
	containerlog.ServeSocket(w, r, func(ctx context.Context, _ int, _ bool) (containerlog.Feed, error) {
		return h.svc.OpenShellOutput(ctx, jobID)
	})
}
