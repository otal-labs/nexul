package main

import (
	"errors"
	"net/http"

	"github.com/otal-labs/nexul/internal/platform/httpx"
	"github.com/otal-labs/nexul/internal/platform/version"
	"github.com/otal-labs/nexul/internal/runner"
)

// versionResponse is the GET /api/version wire shape; Latest is null and UpdateAvailable false for a dev build
// or when the GitHub lookup fails.
type versionResponse struct {
	Version         string         `json:"version"`
	Channel         string         `json:"channel"`
	Latest          *latestVersion `json:"latest"`
	UpdateAvailable bool           `json:"update_available"`
}

type latestVersion struct {
	Version string `json:"version"`
	URL     string `json:"url"`
}

// versionHandler serves GET /api/version off the same release client the runner download proxy uses, so the two
// share one 5-minute cache instead of each polling GitHub on its own.
func versionHandler(runnerSvc *runner.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := versionResponse{Version: version.Version, Channel: version.Channel()}
		if !version.IsRelease() {
			httpx.WriteJSON(w, http.StatusOK, resp)
			return
		}
		rel, err := runnerSvc.ReleaseClient().Latest(r.Context(), version.Channel())
		if err != nil {
			httpx.WriteJSON(w, http.StatusOK, resp)
			return
		}
		resp.Latest = &latestVersion{Version: rel.Tag, URL: rel.URL}
		resp.UpdateAvailable = rel.Tag != version.Version
		httpx.WriteJSON(w, http.StatusOK, resp)
	}
}

// instanceUpgradeGetHandler serves GET /api/instance/upgrade: the facts row plus whether an upgrade can start.
// The use-case enforces instance administration, so a non-admin gets 403 here and over MCP alike.
func instanceUpgradeGetHandler(runnerSvc *runner.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status, err := runnerSvc.UpgradeStatus(r.Context())
		if err != nil {
			httpx.WriteError(w, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, status)
	}
}

// instanceUpgradePostHandler serves POST /api/instance/upgrade: 202 with the new record, or 409 with the
// spec's {"reason": "..."} body (not the generic error envelope) when can_upgrade was false.
func instanceUpgradePostHandler(runnerSvc *runner.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		upgrade, err := runnerSvc.RequestUpgrade(r.Context(), requestUserID(r))
		if err != nil {
			var blocked *runner.UpgradeBlockedError
			if errors.As(err, &blocked) {
				httpx.WriteJSON(w, http.StatusConflict, map[string]string{"reason": blocked.Reason})
				return
			}
			httpx.WriteError(w, err)
			return
		}
		httpx.WriteJSON(w, http.StatusAccepted, upgrade)
	}
}
