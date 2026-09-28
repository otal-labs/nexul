package main

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/otal-labs/nexul/internal/platform/httpx"
	"github.com/otal-labs/nexul/internal/platform/version"
	"github.com/otal-labs/nexul/internal/runner"
)

// versionResponse is the GET /api/version wire shape; Latest is null and UpdateAvailable false for a dev build
// or when the GitHub lookup fails. Changes lists the releases between this build and Latest, newest first.
type versionResponse struct {
	Version         string          `json:"version"`
	Channel         string          `json:"channel"`
	Latest          *latestVersion  `json:"latest"`
	UpdateAvailable bool            `json:"update_available"`
	Changes         []versionChange `json:"changes"`
}

type versionChange struct {
	Version string   `json:"version"`
	URL     string   `json:"url"`
	Notes   []string `json:"notes"`
}

type latestVersion struct {
	Version string `json:"version"`
	URL     string `json:"url"`
}

// versionHandler serves GET /api/version off the same release client the runner download proxy uses, so the two
// share one 5-minute cache instead of each polling GitHub on its own.
func versionHandler(runnerSvc *runner.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("refresh") == "1" {
			runnerSvc.RefreshReleases()
		}
		resp := versionResponse{Version: version.Version, Channel: version.Channel(), Changes: []versionChange{}}
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
		if resp.UpdateAvailable {
			resp.Changes = versionChanges(r, runnerSvc)
		}
		httpx.WriteJSON(w, http.StatusOK, resp)
	}
}

// instanceUpgradeGetHandler serves GET /api/instance/upgrade: the facts row plus whether an upgrade can start.
// The use-case enforces instance administration, so a non-admin gets 403 here and over MCP alike.
func instanceUpgradeGetHandler(runnerSvc *runner.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("refresh") == "1" {
			runnerSvc.RefreshReleases()
		}
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

// versionChanges never fails the version response: without notes the update still shows, just without its changelog.
func versionChanges(r *http.Request, runnerSvc *runner.Service) []versionChange {
	releases, err := runnerSvc.ReleaseClient().Since(r.Context(), version.Channel(), version.Version)
	if err != nil {
		slog.DebugContext(r.Context(), "list releases since this build", "error", err)
		return []versionChange{}
	}
	changes := make([]versionChange, 0, len(releases))
	for _, rel := range releases {
		notes := rel.Notes()
		if notes == nil {
			notes = []string{}
		}
		changes = append(changes, versionChange{Version: rel.Tag, URL: rel.URL, Notes: notes})
	}
	return changes
}
