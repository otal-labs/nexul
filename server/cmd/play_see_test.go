package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/access"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
	"github.com/otal-labs/nexul/internal/platform/permissions"
	"github.com/otal-labs/nexul/internal/plays"
)

// hiddenPlay is the play the reader's role is denied See on, while uPlain, whose role holds no play permission, is
// allowed it by name.
const hiddenPlay = "play-notes"

// playSeeFixture gives the reader's and writer's roles the plays permissions, adds a doc play with an auto play, denies
// role Reader See on it and allows uPlain See and Run on it.
func playSeeFixture(t *testing.T) (permFixture, *plays.Runner) {
	t.Helper()
	f := newPermFixture(t)
	ctx := context.Background()
	for _, id := range []string{"role-reader", "role-writer"} {
		role, err := f.store.Roles.Get(ctx, id)
		require.NoError(t, err)
		role.Permissions = role.Permissions.With(permissions.PlaysRead).With(permissions.PlaysRun).With(permissions.AutoplaysRead)
		require.NoError(t, f.store.Roles.Update(ctx, role))
	}
	now := time.Now().UTC()
	require.NoError(t, f.store.Plays.Create(ctx, &plays.Play{ID: hiddenPlay, WorkspaceID: wsDefault, Label: "Release notes", Type: plays.TypeDoc,
		Instructions: "Write them.", Enabled: true, ExcludedProjectIDs: []string{}, CreatedAt: now, UpdatedAt: now}))
	_, err := f.svc.playsSvc.CreateAutoPlay(as(uOwner), wsDefault, hiddenPlay, plays.AutoPlayInput{Moment: plays.MomentDocCreated, RunOn: plays.RunOnCauser,
		Conditions: plays.Conditions{Match: plays.MatchAll, Groups: []plays.Group{}}, Priority: plays.Priority{Rules: []plays.PriorityRule{}, Otherwise: plays.LevelNormal}})
	require.NoError(t, err)
	a := f.svc.accessSvc
	require.NoError(t, a.SetRules(as(uOwner), uOwner, "play", []string{hiddenPlay}, access.Targets{RoleIDs: []string{"role-reader"}},
		[]permissions.Action{permissions.PlaysRead}, access.StateDeny))
	require.NoError(t, a.SetRules(as(uOwner), uOwner, "play", []string{hiddenPlay}, access.Targets{UserIDs: []string{uPlain}},
		[]permissions.Action{permissions.PlaysRead, permissions.PlaysRun}, access.StateAllow))
	runner := plays.NewRunner(plays.RunnerConfig{Plays: f.store.Plays, Trails: f.store.PlayTrails, Queue: f.store.PlayQueue, Perm: a,
		Targets: playsTargetReader{tickets: f.svc.ticketsSvc, docs: f.svc.docsSvc, workspace: f.svc.workspaceSvc}, Projects: playsProjectLookup{memoriesProjectLookup{projects: f.store.Projects}}})
	return f, runner
}

// serveAs answers one request as user through routes, the way the gateway hands a handler its actor.
func serveAs(routes http.Handler, user, method, path, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	routes.ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(body)).WithContext(as(user)))
	return rec
}

// listsHidden reports whether a JSON list of plays holds the hidden play.
func listsHidden(t *testing.T, raw []byte) bool {
	t.Helper()
	var list []struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(raw, &list), string(raw))
	return slices.ContainsFunc(list, func(p struct {
		ID string `json:"id"`
	}) bool {
		return p.ID == hiddenPlay
	})
}

// TestPlays_SeeHidesItEverywhere: a play denied to a role is gone from the plays list, the run menus, auto plays and
// agents' play lists for its holders, and its link and its queued runs are not found, over HTTP and MCP; a rule naming a
// person shows it to them, and the Owner sees it whatever the rules.
func TestPlays_SeeHidesItEverywhere(t *testing.T) {
	f, runner := playSeeFixture(t)
	s := f.svc
	defs := plays.NewHandler(s.playsSvc).Routes()
	runs := plays.NewRunHandler(runner, nil).Routes()
	playList := tool(t, plays.MCPTools(s.playsSvc), "play_list")
	ws := "/api/workspaces/" + wsDefault + "/plays"

	for user, sees := range map[string]bool{uOwner: true, uWriter: true, uPlain: true, uReader: false} {
		t.Run(user, func(t *testing.T) {
			list := serveAs(defs, user, http.MethodGet, ws, "")
			require.Equal(t, http.StatusOK, list.Code, list.Body.String())
			assert.Equal(t, sees, listsHidden(t, list.Body.Bytes()), "the plays list")

			menu := serveAs(defs, user, http.MethodGet, ws+"/applicable?type=doc&project_id="+pGeneral, "")
			require.Equal(t, http.StatusOK, menu.Code, menu.Body.String())
			assert.Equal(t, sees, listsHidden(t, menu.Body.Bytes()), "the run menus")

			link := http.StatusNotFound
			if sees {
				link = http.StatusOK
			}
			assert.Equal(t, link, serveAs(defs, user, http.MethodGet, ws+"/"+hiddenPlay, "").Code, "its link")

			defined, _ := callPage(t, s.accessSvc, playList, user, `{"workspace_id":"`+wsDefault+`"}`)
			assert.Equal(t, sees, slices.Contains(defined, hiddenPlay), "play_list")
			runnable, _ := callPage(t, s.accessSvc, playList, user, `{"workspace_id":"`+wsDefault+`","type":"doc","project_id":"`+pGeneral+`"}`)
			assert.Equal(t, sees, slices.Contains(runnable, hiddenPlay), "play_list with type")
		})
	}

	t.Run("its auto plays and queued runs", func(t *testing.T) {
		assert.Equal(t, http.StatusOK, serveAs(defs, uWriter, http.MethodGet, ws+"/"+hiddenPlay+"/auto-plays", "").Code)
		assert.Equal(t, http.StatusNotFound, serveAs(defs, uReader, http.MethodGet, ws+"/"+hiddenPlay+"/auto-plays", "").Code)
		byPlay, err := s.playsSvc.AutoPlaysByPlay(as(uReader), wsDefault, []string{hiddenPlay})
		require.NoError(t, err)
		assert.NotContains(t, byPlay, hiddenPlay)
		byPlay, err = s.playsSvc.AutoPlaysByPlay(as(uWriter), wsDefault, []string{hiddenPlay})
		require.NoError(t, err)
		assert.Len(t, byPlay[hiddenPlay], 1)

		assert.Equal(t, http.StatusOK, serveAs(runs, uWriter, http.MethodGet, "/api/plays/queued?play_id="+hiddenPlay, "").Code)
		assert.Equal(t, http.StatusNotFound, serveAs(runs, uReader, http.MethodGet, "/api/plays/queued?play_id="+hiddenPlay, "").Code)
	})

	t.Run("pressing it, over HTTP and MCP", func(t *testing.T) {
		body := `{"target_type":"doc","target_id":"` + f.doc + `"}`
		assert.Equal(t, http.StatusNotFound, serveAs(runs, uReader, http.MethodPost, "/api/plays/"+hiddenPlay+"/run", body).Code)
		_, err := tool(t, plays.RunMCPTools(runner), "play_run").Call(as(uReader), json.RawMessage(`{"play_id":"`+hiddenPlay+`","target_type":"doc","target_id":"`+f.doc+`"}`))
		require.ErrorIs(t, err, apperrs.ErrNotFound)
		_, err = tool(t, plays.MCPTools(s.playsSvc), "play_update").Call(as(uReader), json.RawMessage(`{"workspace_id":"`+wsDefault+`","id":"`+hiddenPlay+`","label":"Notes"}`))
		require.ErrorIs(t, err, apperrs.ErrNotFound)
	})

	t.Run("its rules and live frames", func(t *testing.T) {
		_, err := s.accessSvc.ListRules(as(uReader), uReader, "play", hiddenPlay)
		require.ErrorIs(t, err, apperrs.ErrNotFound, "a play the caller can't see has no rules to them")
		a := liveAudience{access: s.accessSvc, tickets: s.ticketsSvc, chat: s.chatSvc, deploy: s.deploySvc}
		updated := plays.UpdatedEvent{Play: plays.Play{ID: hiddenPlay, WorkspaceID: wsDefault}}
		auto := plays.AutoPlayDeletedEvent{ID: "ap-1", PlayID: hiddenPlay, WorkspaceID: wsDefault}
		for user, sees := range map[string]bool{uOwner: true, uWriter: true, uPlain: true, uReader: false} {
			assert.Equal(t, sees, a.allows(as(user), plays.TopicUpdated, updated), "play.updated to %s", user)
		}
		assert.True(t, a.allows(as(uWriter), plays.TopicAutoPlayDeleted, auto))
		assert.False(t, a.allows(as(uReader), plays.TopicAutoPlayDeleted, auto))
	})
}

// TestPlays_TheStarterReadsTheirRun: a run's starter reads it, and its live frames, with See on its play denied; anyone
// else reads it only when they can see the play.
func TestPlays_TheStarterReadsTheirRun(t *testing.T) {
	f, runner := playSeeFixture(t)
	ctx := context.Background()
	for id, starter := range map[string]string{"tr-reader": uReader, "tr-writer": uWriter} {
		require.NoError(t, f.store.PlayTrails.CreateTrail(ctx, &plays.Trail{
			ID: id, WorkspaceID: wsDefault, PlayID: hiddenPlay, PlayLabel: "Release notes", TargetType: plays.TargetTicket, TargetID: f.ticket.ID,
			ProjectID: pGeneral, StarterID: starter, Via: plays.ViaWeb, State: plays.TrailDone, StartedAt: time.Now(), Activity: []plays.ActivityEntry{},
		}))
	}
	runs := plays.NewRunHandler(runner, nil).Routes()
	trailList := tool(t, plays.RunMCPTools(runner), "trail_list")
	a := liveAudience{access: f.svc.accessSvc, tickets: f.svc.ticketsSvc, chat: f.svc.chatSvc, deploy: f.svc.deploySvc}

	for _, tt := range []struct {
		user  string
		reads []string
	}{
		{uReader, []string{"tr-reader"}},
		{uWriter, []string{"tr-reader", "tr-writer"}},
		{uOwner, []string{"tr-reader", "tr-writer"}},
	} {
		t.Run(tt.user, func(t *testing.T) {
			for _, id := range []string{"tr-reader", "tr-writer"} {
				reads := slices.Contains(tt.reads, id)
				code := http.StatusNotFound
				if reads {
					code = http.StatusOK
				}
				assert.Equal(t, code, serveAs(runs, tt.user, http.MethodGet, "/api/plays/runs/"+id, "").Code, id)
				_, err := trailList.Call(as(tt.user), json.RawMessage(`{"id":"`+id+`"}`))
				assert.Equal(t, reads, err == nil, "trail_list of %s", id)
				frame := plays.RunFrame{TrailID: id, PlayID: hiddenPlay, TargetType: plays.TargetTicket, TargetID: f.ticket.ID,
					StarterID: map[string]string{"tr-reader": uReader, "tr-writer": uWriter}[id], WorkspaceID: wsDefault, ProjectID: pGeneral}
				assert.Equal(t, reads, a.allows(as(tt.user), plays.TopicPlayRun, frame), "the run frame of %s", id)
			}
			listed, _ := callPage(t, f.svc.accessSvc, trailList, tt.user, `{"target_type":"ticket","target_id":"`+f.ticket.ID+`"}`)
			assert.ElementsMatch(t, tt.reads, listed, "trail_list of the ticket")
			rec := serveAs(runs, tt.user, http.MethodGet, "/api/plays/runs?target_type=ticket&target_id="+f.ticket.ID, "")
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			var list []plays.Trail
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))
			got := make([]string, 0, len(list))
			for _, tr := range list {
				got = append(got, tr.ID)
			}
			assert.ElementsMatch(t, tt.reads, got, "the ticket's trails over HTTP")
		})
	}
	_, err := runner.Stop(as(uReader), "tr-writer")
	require.ErrorIs(t, err, apperrs.ErrNotFound, "a run the caller can't read can't be steered")
}

// hiddenCanRun is the hidden play's can_run in a JSON list of plays, nil when the list leaves it out.
func hiddenCanRun(t *testing.T, raw []byte) *bool {
	t.Helper()
	var list []struct {
		ID     string `json:"id"`
		CanRun *bool  `json:"can_run"`
	}
	require.NoError(t, json.Unmarshal(raw, &list), string(raw))
	for _, p := range list {
		if p.ID == hiddenPlay {
			require.NotNil(t, p.CanRun, "a listed play says whether the caller can run it")
			return p.CanRun
		}
	}
	return nil
}

// TestPlays_CanRunFollowsThePlaysOwnRules: a play's own Run allow lets someone whose roles hold no plays:run run it,
// and its Run deny leaves it seen but not runnable, in the plays list, the run menus and play_list alike.
func TestPlays_CanRunFollowsThePlaysOwnRules(t *testing.T) {
	f, _ := playSeeFixture(t)
	s := f.svc
	require.NoError(t, s.accessSvc.SetRules(as(uOwner), uOwner, "play", []string{hiddenPlay}, access.Targets{UserIDs: []string{uWriter}},
		[]permissions.Action{permissions.PlaysRun}, access.StateDeny))
	defs := plays.NewHandler(s.playsSvc).Routes()
	playList := tool(t, plays.MCPTools(s.playsSvc), "play_list")
	ws := "/api/workspaces/" + wsDefault + "/plays"
	yes, no := true, false
	for _, tt := range []struct {
		user         string
		listed, menu *bool
	}{
		{uPlain, &yes, &yes},
		{uWriter, &no, nil},
	} {
		t.Run(tt.user, func(t *testing.T) {
			assert.Equal(t, tt.listed, hiddenCanRun(t, serveAs(defs, tt.user, http.MethodGet, ws, "").Body.Bytes()), "the plays list")
			assert.Equal(t, tt.menu, hiddenCanRun(t, serveAs(defs, tt.user, http.MethodGet, ws+"/applicable?type=doc&project_id="+pGeneral, "").Body.Bytes()), "the run menus")
			assert.Equal(t, tt.listed, hiddenCanRun(t, pageItems(t, s.accessSvc, playList, tt.user, `{"workspace_id":"`+wsDefault+`"}`)), "play_list")
			assert.Equal(t, tt.menu, hiddenCanRun(t, pageItems(t, s.accessSvc, playList, tt.user, `{"workspace_id":"`+wsDefault+`","type":"doc","project_id":"`+pGeneral+`"}`)), "play_list with type")
		})
	}
}

// pageItems is a list tool's page of items as one JSON list.
func pageItems(t *testing.T, a *access.Service, tl mcptool.Tool, user, args string) []byte {
	t.Helper()
	_, page := callPage(t, a, tl, user, args)
	raw, err := json.Marshal(page.Items)
	require.NoError(t, err)
	return raw
}
