package main

import (
	"errors"
	"fmt"
	"net/http"
	"path"
	"strings"

	"github.com/otal-labs/nexul/internal/auth"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/httpx"
	"github.com/otal-labs/nexul/internal/platform/identity"
	"github.com/otal-labs/nexul/internal/tenancy"
)

const (
	headerViewAsUser      = "X-View-As-User"
	headerViewAsWorkspace = "X-View-As-Workspace"
	// headerViewAsRefused tells the web to leave the mode; a 401 would sign the viewer out instead.
	headerViewAsRefused = "X-View-As"
)

// viewAsReadPosts are the POSTs that only read, so a view still serves them.
var viewAsReadPosts = routeMux(
	"POST /api/mentions/resolve",
	"POST /api/tickets/labels/colors",
	"POST /api/logs",
)

// viewAsPersonal are the routes that belong to the person being viewed rather than to their roles: sign-in state,
// personal computers (ADR 0146), the inbox and connector sign-ins. GET /api/auth/me is the one exception, so it is
// left out in viewRefusal.
var viewAsPersonal = routeMux(
	"/api/auth", "/api/auth/",
	"/api/pairing", "/api/pairing/",
	"/api/notifications", "/api/notifications/",
	"/api/connectors/{id}/oauth/",
)

func routeMux(patterns ...string) *http.ServeMux {
	mux := http.NewServeMux()
	for _, p := range patterns {
		mux.Handle(p, http.NotFoundHandler())
	}
	return mux
}

func routed(mux *http.ServeMux, r *http.Request) bool {
	_, pattern := mux.Handler(r)
	return pattern != ""
}

// withViewAs serves a browser session request that carries the view-as headers as the viewed member (ADR 0160): the
// request's user becomes the target, every write is refused, and what is personal or in another workspace is not
// found. A request without the headers passes untouched; one from a token is turned away by rejectViewAsOffSession.
func withViewAs(tenancySvc *tenancy.Service, authSvc *auth.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			targetID, workspaceID := r.Header.Get(headerViewAsUser), r.Header.Get(headerViewAsWorkspace)
			viewer := auth.UserFromCtx(r.Context())
			if (targetID == "" && workspaceID == "") || auth.SessionFromCtx(r.Context()) == nil || viewer == nil {
				next.ServeHTTP(w, r)
				return
			}
			if targetID == "" || workspaceID == "" {
				httpx.WriteError(w, fmt.Errorf("%w: %s and %s go together", apperrs.ErrInvalid, headerViewAsUser, headerViewAsWorkspace))
				return
			}
			viewed, err := tenancySvc.CheckViewAs(r.Context(), viewer.ID, workspaceID, targetID)
			if errors.Is(err, apperrs.ErrNotFound) || errors.Is(err, apperrs.ErrForbidden) || errors.Is(err, apperrs.ErrInvalid) {
				w.Header().Set(headerViewAsRefused, "refused")
				httpx.WriteJSON(w, http.StatusForbidden, httpx.Envelope{Message: err.Error(), Code: "FORBIDDEN"})
				return
			}
			if err != nil {
				httpx.WriteError(w, err)
				return
			}
			if refusal := viewRefusal(r, viewed); refusal != nil {
				httpx.WriteError(w, refusal)
				return
			}
			target, err := authSvc.Whoami(r.Context(), targetID)
			if err != nil {
				httpx.WriteError(w, err)
				return
			}
			ctx := auth.WithViewedUser(r.Context(), target, &identity.ViewerRef{ID: viewer.ID, WorkspaceID: workspaceID})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// viewRefusal is why a request made in a view is not served, nil when it is.
func viewRefusal(r *http.Request, viewed *tenancy.ViewedMember) error {
	if inOtherWorkspace(r.URL.Path, viewed.WorkspaceID) {
		return apperrs.ErrNotFound
	}
	if routed(viewAsPersonal, r) && (r.Method != http.MethodGet || path.Clean(r.URL.Path) != "/api/auth/me") {
		return apperrs.ErrNotFound
	}
	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return nil
	}
	if routed(viewAsReadPosts, r) {
		return nil
	}
	return fmt.Errorf("%w: You're viewing as %s; changes are off", apperrs.ErrForbidden, viewed.Name)
}

// inOtherWorkspace is whether urlPath is a workspace's route for any workspace but the one the view is held to.
func inOtherWorkspace(urlPath, workspaceID string) bool {
	segments := strings.Split(path.Clean(urlPath), "/")
	return len(segments) > 3 && segments[1] == "api" && segments[2] == "workspaces" && segments[3] != workspaceID
}

// rejectViewAsOffSession answers 400 to a request that carries the view-as headers without a browser session: a
// personal access token, an automation or an integration never views as anyone.
func rejectViewAsOffSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hasHeaders := r.Header.Get(headerViewAsUser) != "" || r.Header.Get(headerViewAsWorkspace) != ""
		if hasHeaders && auth.SessionFromCtx(r.Context()) == nil {
			httpx.WriteError(w, fmt.Errorf("%w: only a signed-in browser session can view as another member", apperrs.ErrInvalid))
			return
		}
		next.ServeHTTP(w, r)
	})
}
