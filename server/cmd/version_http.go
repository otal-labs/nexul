package main

import (
	"context"
	"errors"
	"net/http"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/httpx"
	"github.com/otal-labs/nexul/internal/platform/version"
	"github.com/otal-labs/nexul/internal/runner"
)

// upgradeAdminGate is the instance-admin fact requireInstanceAdmin checks; instanceAdminGate is its only
// production implementation, kept as an interface here so tests can supply a fake without a real auth.Service.
type upgradeAdminGate interface {
	CanCreateWorkspace(ctx context.Context, userID string) (bool, error)
}

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

// requireInstanceAdmin gates GET/POST /api/instance/upgrade on the instance-admin fact directly, since
// runner.Service's UpgradeStatus/RequestUpgrade take no actor for that check (UpgradeStatus has none to take;
// RequestUpgrade's actor is provenance, not a gate) — unlike domains that bake the check into their use-case
// (workspace.requireOwner), this one lives at the HTTP boundary. Reads the acting user the same way withUserID
// does (requestUserID: session/PAT first, identity.Actor otherwise) rather than currentUserID's session-only
// view. A non-admin, or no signed-in user, gets 403/401.
func requireInstanceAdmin(gate upgradeAdminGate, w http.ResponseWriter, r *http.Request) (userID string, ok bool) {
	userID = requestUserID(r)
	if err := checkInstanceAdmin(r.Context(), gate, userID); err != nil {
		httpx.WriteError(w, err)
		return "", false
	}
	return userID, true
}

// checkInstanceAdmin is requireInstanceAdmin's status-code-free core, split out so it is testable without a
// real HTTP request or auth.Service.
func checkInstanceAdmin(ctx context.Context, gate upgradeAdminGate, userID string) error {
	if userID == "" {
		return apperrs.ErrUnauthorized
	}
	can, err := gate.CanCreateWorkspace(ctx, userID)
	if err != nil {
		return err
	}
	if !can {
		return apperrs.ErrForbidden
	}
	return nil
}

// instanceUpgradeGetHandler serves GET /api/instance/upgrade: the facts row plus whether an upgrade can start.
func instanceUpgradeGetHandler(runnerSvc *runner.Service, gate upgradeAdminGate) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := requireInstanceAdmin(gate, w, r); !ok {
			return
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
func instanceUpgradePostHandler(runnerSvc *runner.Service, gate upgradeAdminGate) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := requireInstanceAdmin(gate, w, r)
		if !ok {
			return
		}
		upgrade, err := runnerSvc.RequestUpgrade(r.Context(), userID)
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
