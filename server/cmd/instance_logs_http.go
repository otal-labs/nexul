package main

import (
	"net/http"

	"github.com/otal-labs/nexul/internal/platform/httpx"
	"github.com/otal-labs/nexul/internal/platform/logstore"
	"github.com/otal-labs/nexul/internal/runner"
)

// instanceLogsHandler serves GET /api/instance/logs: the same read instance_get's logs makes, gated by logs:read in
// the use-case so the browser and MCP answer alike.
func instanceLogsHandler(runnerSvc *runner.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params, err := logstore.ParamsOf(r.URL.Query())
		if err != nil {
			httpx.WriteError(w, err)
			return
		}
		page, err := runnerSvc.InstanceLogs(r.Context(), params)
		if err != nil {
			httpx.WriteError(w, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, page)
	}
}
