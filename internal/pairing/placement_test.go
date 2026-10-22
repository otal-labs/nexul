package pairing

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/platform/crypto"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

var placementNow = time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)

// fakeRemotes is each project's repositories as T3 names a checkout's remote.
type fakeRemotes map[string][]string

func (f fakeRemotes) ProjectRemotes(_ context.Context, projectID string) ([]string, error) {
	return f[projectID], nil
}

type placementFixture struct {
	repo *fakeRepo
	exch *fakeExchanger
	svc  *Service
}

func newPlacementFixture(t *testing.T) *placementFixture {
	t.Helper()
	repo := newFakeRepo()
	exch := &fakeExchanger{}
	exch.ListProvidersFn = func(context.Context, harness.Session) ([]harness.Provider, error) {
		return []harness.Provider{{ID: "claude", Driver: "claude", Name: "Claude"}}, nil
	}
	svc := NewService(Config{
		Repo: repo, Harnesses: registry(exch), EncryptionKey: testEncKey, Now: func() time.Time { return placementNow },
		Projects: fakeProjects{}, Repositories: fakeRemotes{"proj-1": {"github.com/acme/app"}}, Tokens: newFakeTokens(),
	})
	return &placementFixture{repo: repo, exch: exch, svc: svc}
}

// add pairs a computer for u1 that is ready for a Claude run in proj-1, after edit.
func (f *placementFixture) add(t *testing.T, id string, edit func(*Computer)) {
	t.Helper()
	sealed, err := crypto.Encrypt(testEncKey, []byte("bearer-"+id))
	require.NoError(t, err)
	c := Computer{
		ID: id, UserID: "u1", Kind: harness.KindT3Code, Name: id, ServerURL: "https://" + id + ".example.com", BearerToken: sealed,
		TokenExpiresAt: placementNow.Add(30 * 24 * time.Hour), SetupConfirmedAt: &placementNow, Position: len(f.repo.computers),
		RunsAtOnce: 1, ChatProjectID: "t3-chat-" + id,
		Facts: &Facts{
			T3:        T3Facts{State: t3Answering},
			Providers: []ProviderFacts{{ID: "claude", Driver: "claude", Name: "Claude", Models: []ModelFacts{{Slug: "sonnet"}, {Slug: "opus"}}}},
			Projects:  []harness.Project{{ID: "t3-app-" + id, Repository: "github.com/acme/app"}, {ID: "t3-other-" + id}},
		},
	}
	if edit != nil {
		edit(&c)
	}
	f.repo.computers[id] = c
	f.repo.setups[id] = []ProviderSetup{{Provider: "claude", ConfirmedAt: &placementNow}}
}

func TestResolveRun_LandsOnTheFirstReadyComputer(t *testing.T) {
	t.Parallel()
	offline := func(c *Computer) { c.Facts.T3.State = "not_running" }
	tests := []struct {
		name       string
		edits      map[string]func(*Computer)
		noSetup    string
		running    map[string]int
		defaults   Defaults
		link       ProjectLink
		req        RunRequest
		answerless string
		want       string
		checkout   string
		reason     NotConfiguredReason
		passed     []string
	}{
		{name: "the first ready computer takes it", want: "a", checkout: "t3-app-a"},
		{name: "an offline computer is passed over", edits: map[string]func(*Computer){"a": offline}, want: "b", checkout: "t3-app-b",
			passed: []string{"T3 Code isn't answering on a"}},
		{name: "a session that ran out is passed over", edits: map[string]func(*Computer){"a": func(c *Computer) { c.TokenExpiresAt = placementNow.Add(-time.Hour) }},
			want: "b", passed: []string{"a needs pairing again"}},
		{name: "a computer not set up for the provider is passed over", noSetup: "a", want: "b", passed: []string{"a isn't set up for Claude"}},
		{name: "a computer without the model is passed over", req: RunRequest{Model: "opus"},
			edits: map[string]func(*Computer){"a": func(c *Computer) { c.Facts.Providers[0].Models = []ModelFacts{{Slug: "sonnet"}} }},
			want:  "b", passed: []string{"a doesn't offer opus"}},
		{name: "a computer without a checkout of the project is passed over",
			edits: map[string]func(*Computer){"a": func(c *Computer) {
				c.Facts.Projects = []harness.Project{{ID: "t3-x", Repository: "github.com/acme/other"}}
			}},
			want: "b", passed: []string{"a doesn't have this project"}},
		{name: "two checkouts of the project are ambiguous",
			edits: map[string]func(*Computer){"a": func(c *Computer) {
				c.Facts.Projects = append(c.Facts.Projects, harness.Project{ID: "t3-app-copy", Repository: "github.com/acme/app"})
			}},
			want: "b", passed: []string{"a has several checkouts of this project; pick one"}},
		{name: "a picked checkout wins over the remote", link: ProjectLink{Checkouts: []Checkout{{ComputerID: "a", HarnessProjectID: "t3-other-a"}}},
			want: "a", checkout: "t3-other-a"},
		{name: "a picked checkout the computer no longer lists falls back to the remote",
			link: ProjectLink{Checkouts: []Checkout{{ComputerID: "a", HarnessProjectID: "t3-gone"}}}, want: "a", checkout: "t3-app-a"},
		{name: "a queued run passes over a full computer", req: RunRequest{Room: RoomRequired}, running: map[string]int{"a": 1},
			want: "b", passed: []string{"a is full (1 of 1 running)"}},
		{name: "a queued run waits while every computer is full", req: RunRequest{Room: RoomRequired}, running: map[string]int{"a": 1, "b": 1},
			reason: ReasonNoComputerReady, passed: []string{"a is full (1 of 1 running)", "b is full (1 of 1 running)"}},
		{name: "a pressed run with every computer full starts on the first", req: RunRequest{Room: RoomPreferred}, running: map[string]int{"a": 1, "b": 1},
			want: "a", passed: []string{"b is full (1 of 1 running)"}},
		{name: "a chat ignores how busy a computer is", running: map[string]int{"a": 5}, want: "a"},
		{name: "spread takes the computer least busy for its size", req: RunRequest{Room: RoomRequired}, defaults: Defaults{Placement: PlacementSpread},
			edits: map[string]func(*Computer){"a": func(c *Computer) { c.RunsAtOnce = 2 }}, running: map[string]int{"a": 1}, want: "b"},
		{name: "spread goes in order between equally busy computers", req: RunRequest{Room: RoomRequired}, defaults: Defaults{Placement: PlacementSpread}, want: "a"},
		{name: "a project's own list narrows the order", link: ProjectLink{Computers: []string{"b"}}, want: "b"},
		{name: "a project's own placement wins over the person's", req: RunRequest{Room: RoomRequired}, running: map[string]int{"a": 1},
			edits:    map[string]func(*Computer){"a": func(c *Computer) { c.RunsAtOnce = 3 }},
			defaults: Defaults{Placement: PlacementSpread}, link: ProjectLink{Computers: []string{"a", "b"}, Placement: PlacementInOrder}, want: "a"},
		{name: "a chat outside a project runs in the chat folder", req: RunRequest{ProjectID: "-"}, want: "a", checkout: "t3-chat-a"},
		{name: "a computer with no chat folder is passed over", req: RunRequest{ProjectID: "-"},
			edits: map[string]func(*Computer){"a": func(c *Computer) { c.ChatProjectID = "" }}, want: "b", checkout: "t3-chat-b",
			passed: []string{"a has no chat folder"}},
		{name: "a chat in a project no computer has runs in the chat folder", req: RunRequest{Chat: true},
			edits: map[string]func(*Computer){"a": func(c *Computer) { c.Facts.Projects = nil }, "b": func(c *Computer) { c.Facts.Projects = nil }},
			want:  "a", checkout: "t3-chat-a"},
		{name: "a pressed run no computer has the project for asks where", req: RunRequest{Person: true, Room: RoomPreferred},
			edits:  map[string]func(*Computer){"a": func(c *Computer) { c.Facts.Projects = nil }, "b": func(c *Computer) { c.Facts.Projects = nil }},
			reason: ReasonNeedsLocation, passed: []string{"a doesn't have this project", "b doesn't have this project"}},
		{name: "a run pinned to one offline computer is that computer's refusal", req: RunRequest{ComputerIDs: []string{"a"}},
			edits: map[string]func(*Computer){"a": offline}, reason: ReasonOffline, passed: []string{"T3 Code isn't answering on a"}},
		{name: "a computer that does not answer is passed over", answerless: "a", want: "b",
			passed: []string{"T3 Code isn't answering on a"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newPlacementFixture(t)
			f.add(t, "a", tt.edits["a"])
			f.add(t, "b", tt.edits["b"])
			if tt.noSetup != "" {
				f.repo.setups[tt.noSetup] = nil
			}
			f.repo.running = tt.running
			tt.defaults.UserID = "u1"
			require.NoError(t, f.repo.SaveDefaults(t.Context(), tt.defaults))
			tt.link.UserID, tt.link.ProjectID = "u1", "proj-1"
			require.NoError(t, f.repo.SaveProjectLink(t.Context(), tt.link))
			f.exch.ListProvidersFn = func(_ context.Context, s harness.Session) ([]harness.Provider, error) {
				if s.ComputerID == tt.answerless {
					return nil, errBoom
				}
				return []harness.Provider{{ID: "claude", Driver: "claude", Name: "Claude"}}, nil
			}
			req := tt.req
			req.UserID = "u1"
			if req.ProjectID == "" {
				req.ProjectID = "proj-1"
			}
			if req.ProjectID == "-" {
				req.ProjectID = ""
			}

			target, err := f.svc.ResolveRun(t.Context(), req)

			var passed []string
			if tt.reason != "" {
				var nc *NotConfiguredError
				require.ErrorAs(t, err, &nc)
				assert.Equal(t, tt.reason, nc.Reason)
				for _, p := range nc.PassedOver {
					passed = append(passed, p.Text)
				}
				assert.Equal(t, tt.passed, passed)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, target.Computer.ID)
			assert.Equal(t, "bearer-"+tt.want, target.Computer.BearerToken)
			if tt.checkout != "" {
				assert.Equal(t, tt.checkout, target.HarnessProjectID)
			}
			for _, p := range target.PassedOver {
				passed = append(passed, p.Text)
			}
			assert.Equal(t, tt.passed, passed)
		})
	}
}

func TestResolveRun_ModelAndStartInComeFromTheLinkThenTheDefaults(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		defaults     Defaults
		link         ProjectLink
		req          RunRequest
		wantModel    string
		wantOptions  []harness.OptionSetting
		wantWorktree bool
	}{
		{name: "the defaults when the link says nothing", defaults: Defaults{Provider: "claude", Model: "sonnet", StartIn: StartInWorktree},
			wantModel: "sonnet", wantWorktree: true},
		{name: "the link over the defaults", defaults: Defaults{Model: "sonnet", StartIn: StartInWorktree},
			link:      ProjectLink{Model: "opus", ModelOptions: []harness.OptionSetting{{ID: "effort", Value: "high"}}, StartIn: StartInFolder},
			wantModel: "opus", wantOptions: []harness.OptionSetting{{ID: "effort", Value: "high"}}},
		{name: "the run's own pick over both, without the link's options", link: ProjectLink{Model: "opus", ModelOptions: []harness.OptionSetting{{ID: "effort", Value: "high"}}},
			req: RunRequest{Model: "sonnet"}, wantModel: "sonnet"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newPlacementFixture(t)
			f.add(t, "a", nil)
			tt.defaults.UserID = "u1"
			require.NoError(t, f.repo.SaveDefaults(t.Context(), tt.defaults))
			tt.link.UserID, tt.link.ProjectID = "u1", "proj-1"
			require.NoError(t, f.repo.SaveProjectLink(t.Context(), tt.link))
			req := tt.req
			req.UserID, req.ProjectID = "u1", "proj-1"

			target, err := f.svc.ResolveRun(t.Context(), req)

			require.NoError(t, err)
			assert.Equal(t, tt.wantModel, target.Model)
			assert.Equal(t, tt.wantOptions, target.ModelOptions)
			assert.Equal(t, tt.wantWorktree, target.Worktree)
		})
	}
}

func TestResolveRun_APressedRunsPickIsRememberedOnlyWhenTheComputerListsIt(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		list    func(context.Context, harness.Session) ([]harness.Project, error)
		want    error
		reason  NotConfiguredReason
		checked []Checkout
	}{
		{name: "listed", list: listsT3Projects("t3-picked"), checked: []Checkout{{ComputerID: "a", HarnessProjectID: "t3-picked"}}},
		{name: "not listed", list: listsT3Projects("t3-app"), want: apperrs.ErrInvalid},
		{name: "the computer cannot say", list: func(context.Context, harness.Session) ([]harness.Project, error) { return nil, errBoom },
			want: apperrs.ErrInvalid, reason: ReasonOffline},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newPlacementFixture(t)
			f.add(t, "a", nil)
			f.exch.ListProjectsFn = tt.list

			target, err := f.svc.ResolveRun(t.Context(), RunRequest{
				UserID: "u1", ProjectID: "proj-1", ComputerIDs: []string{"a"}, HarnessProjectID: "t3-picked", Person: true, Room: RoomPreferred,
			})

			link, lerr := f.svc.GetProjectLink(t.Context(), "u1", "proj-1")
			require.NoError(t, lerr)
			assert.Equal(t, tt.checked, link.Checkouts)
			if tt.want != nil {
				require.ErrorIs(t, err, tt.want)
				var nc *NotConfiguredError
				if tt.reason != "" {
					require.ErrorAs(t, err, &nc)
					assert.Equal(t, tt.reason, nc.Reason)
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "t3-picked", target.HarnessProjectID)
		})
	}
}

func TestResolveRun_RefusesWhatItCannotPlace(t *testing.T) {
	t.Parallel()
	f := newPlacementFixture(t)
	f.add(t, "a", nil)
	f.repo.computers["theirs"] = Computer{ID: "theirs", UserID: "u2"}

	_, err := f.svc.ResolveRun(t.Context(), RunRequest{UserID: "u1", ProjectID: "proj-1", ComputerIDs: []string{"theirs"}})
	var nc *NotConfiguredError
	require.ErrorAs(t, err, &nc, "another person's computer is not one of theirs")
	assert.Equal(t, ReasonUnpaired, nc.Reason)

	_, err = f.svc.ResolveRun(t.Context(), RunRequest{UserID: "u1", ProjectID: "proj-1", HarnessProjectID: "t3-app-a"})
	require.ErrorIs(t, err, apperrs.ErrInvalid, "a T3 project needs the one computer it is on")

	_, err = f.svc.ResolveRun(t.Context(), RunRequest{UserID: "u3", ProjectID: "proj-1"})
	require.ErrorAs(t, err, &nc)
	assert.Equal(t, ReasonUnpaired, nc.Reason, "a person with no computer")

	_, err = f.svc.ResolveRun(t.Context(), RunRequest{ProjectID: "proj-1"})
	require.ErrorIs(t, err, apperrs.ErrUnauthorized)
}

func TestPreviewRun_SaysWhereARunWouldLandWithoutAskingAComputer(t *testing.T) {
	t.Parallel()
	f := newPlacementFixture(t)
	f.add(t, "a", func(c *Computer) { c.Facts.T3.State = "not_running" })
	f.add(t, "b", nil)
	f.exch.ListProvidersFn = func(context.Context, harness.Session) ([]harness.Provider, error) {
		t.Fatal("a preview asks no computer")
		return nil, nil
	}

	target, err := f.svc.PreviewTarget(t.Context(), "u1", "proj-1")

	require.NoError(t, err)
	assert.Equal(t, "b", target.Computer.ID)
	assert.Empty(t, target.Computer.BearerToken)
	assert.Equal(t, []PassOver{{ComputerID: "a", Name: "a", Why: PassOverOffline, Text: "T3 Code isn't answering on a"}}, target.PassedOver)
}

func TestReorderComputers_TakesEveryOwnComputerOnce(t *testing.T) {
	t.Parallel()
	f := newPlacementFixture(t)
	f.add(t, "a", nil)
	f.add(t, "b", nil)
	f.repo.computers["theirs"] = Computer{ID: "theirs", UserID: "u2"}

	require.NoError(t, f.svc.ReorderComputers(t.Context(), "u1", []string{"b", " a "}))
	listed, err := f.svc.ListComputers(t.Context(), "u1")
	require.NoError(t, err)
	assert.Equal(t, []string{"b", "a"}, []string{listed[0].ID, listed[1].ID})

	for name, ids := range map[string][]string{
		"one left out":       {"b"},
		"one named twice":    {"a", "a"},
		"someone else's one": {"a", "b", "theirs"},
	} {
		require.ErrorIs(t, f.svc.ReorderComputers(t.Context(), "u1", ids), apperrs.ErrInvalid, name)
	}
}

func TestListComputers_CountsTheRunsGoingOnEachOwnComputer(t *testing.T) {
	t.Parallel()
	f := newPlacementFixture(t)
	f.add(t, "a", nil)
	f.add(t, "b", nil)
	f.repo.running = map[string]int{"a": 2}

	listed, err := f.svc.ListComputers(t.Context(), "u1")

	require.NoError(t, err)
	assert.Equal(t, []int{2, 0}, []int{listed[0].Running, listed[1].Running})
}

func TestCheckoutOn_IsTheCheckoutARunThereWouldUse(t *testing.T) {
	t.Parallel()
	f := newPlacementFixture(t)
	f.add(t, "a", nil)
	f.add(t, "b", func(c *Computer) { c.Facts.Projects = nil })

	got, err := f.svc.CheckoutOn(t.Context(), "u1", "proj-1", "a")
	require.NoError(t, err)
	assert.Equal(t, "t3-app-a", got)
	got, err = f.svc.CheckoutOn(t.Context(), "u1", "proj-1", "b")
	require.NoError(t, err)
	assert.Empty(t, got, "a computer without the project has no checkout")
	_, err = f.svc.CheckoutOn(t.Context(), "u2", "proj-1", "a")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
}

func TestSetComputerPlacement_PublishesSoWaitingRunsTryAgain(t *testing.T) {
	t.Parallel()
	bus := &fakeBus{}
	f := newPlacementFixture(t)
	f.svc.bus = bus
	f.add(t, "a", nil)

	_, err := f.svc.SetComputerPlacement(t.Context(), "u1", "a", 3, "t3-chat")
	require.NoError(t, err)
	require.NoError(t, f.svc.ReorderComputers(t.Context(), "u1", []string{"a"}))
	_, err = f.svc.SetDefaults(t.Context(), "u1", Defaults{Placement: PlacementSpread})
	require.NoError(t, err)

	one := PlacementChangedEvent{UserID: "u1", MembersOnly: true}
	assert.Equal(t, []PlacementChangedEvent{one, one, one}, bus.placements, "runs at once, the order and the defaults each wake the queue")
}

func listsT3Projects(ids ...string) func(context.Context, harness.Session) ([]harness.Project, error) {
	return func(context.Context, harness.Session) ([]harness.Project, error) {
		projects := make([]harness.Project, 0, len(ids))
		for _, id := range ids {
			projects = append(projects, harness.Project{ID: id, Title: id})
		}
		return projects, nil
	}
}
