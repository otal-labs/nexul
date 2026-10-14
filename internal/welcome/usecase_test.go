package welcome

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

var fixedNow = time.Date(2026, 10, 13, 9, 0, 0, 0, time.UTC)

// fakeRepo keeps records by user and step, as welcome_step_states does, and the events written with them.
type fakeRepo struct {
	mu      sync.Mutex
	records map[[2]string]Record
	events  []eventbus.OutboxEvent
	err     error
}

func newFakeRepo() *fakeRepo { return &fakeRepo{records: map[[2]string]Record{}} }

func (f *fakeRepo) Records(_ context.Context, userIDs []string) ([]Record, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []Record
	for key, r := range f.records {
		for _, id := range userIDs {
			if key[0] == id {
				out = append(out, r)
			}
		}
	}
	return out, f.err
}

func (f *fakeRepo) write(userID, step string, change func(*Record), events []eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	r := f.records[[2]string{userID, step}]
	r.UserID, r.Step = userID, step
	change(&r)
	f.records[[2]string{userID, step}] = r
	f.events = append(f.events, events...)
	return nil
}

func (f *fakeRepo) SetSkipped(_ context.Context, userID, step string, at *time.Time, events ...eventbus.OutboxEvent) error {
	return f.write(userID, step, func(r *Record) { r.SkippedAt = at }, events)
}

func (f *fakeRepo) Give(_ context.Context, gifts []Gift, events ...eventbus.OutboxEvent) error {
	for _, g := range gifts {
		if err := f.write(g.UserID, g.Step, func(r *Record) {
			r.SkippedAt = nil
			if g.GivenAt != nil {
				r.GivenAt = g.GivenAt
			}
		}, nil); err != nil {
			return err
		}
	}
	return f.Announce(context.Background(), events...)
}

func (f *fakeRepo) MarkDone(_ context.Context, userID, step string, at time.Time, events ...eventbus.OutboxEvent) error {
	return f.write(userID, step, func(r *Record) {
		if r.DoneAt == nil {
			r.DoneAt = &at
		}
	}, events)
}

func (f *fakeRepo) Announce(_ context.Context, events ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	f.events = append(f.events, events...)
	return nil
}

func (f *fakeRepo) announced() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []string
	for _, e := range f.events {
		out = append(out, e.Payload.(ProgressChangedEvent).UserID)
	}
	return out
}

// world is what the other domains hold, answered through Seams the way the composition root wires them.
type world struct {
	ticked    map[string][]string
	roleSteps map[string][]string
	holders   map[string][]string
	members   map[string][]string
	has       map[string]map[string]bool // seam name -> user -> holds
	err       error
}

func (w *world) set(name string) PeopleWith {
	return func(_ context.Context, userIDs []string) (map[string]bool, error) {
		out := map[string]bool{}
		for _, id := range userIDs {
			out[id] = w.has[name][id]
		}
		return out, w.err
	}
}

func (w *world) seams() Seams {
	return Seams{
		TickedSteps: func(_ context.Context, userIDs []string) (map[string][]string, error) {
			out := map[string][]string{}
			for _, id := range userIDs {
				out[id] = w.ticked[id]
			}
			return out, w.err
		},
		RoleSteps: func(_ context.Context, roleIDs []string) ([]string, error) {
			var out []string
			for _, id := range roleIDs {
				out = append(out, w.roleSteps[id]...)
			}
			return out, w.err
		},
		RoleHolders:      func(_ context.Context, roleID string) ([]string, error) { return w.holders[roleID], w.err },
		WorkspaceMembers: func(_ context.Context, id string) ([]string, error) { return w.members[id], w.err },
		GitHub:           w.set("github"),
		Computer:         w.set("computer"),
		InAProject:       w.set("projects"),
		DeployServer:     w.set("deploy_server"),
		FirstRunDone:     w.set("first_run"),
		MarkFirstRun: func(_ context.Context, userID string) error {
			if w.err != nil {
				return w.err
			}
			w.has["first_run"][userID] = true
			return nil
		},
	}
}

func newWorld() *world {
	return &world{ticked: map[string][]string{}, roleSteps: map[string][]string{}, holders: map[string][]string{}, members: map[string][]string{},
		has: map[string]map[string]bool{"github": {}, "computer": {}, "projects": {}, "deploy_server": {}, "first_run": {}}}
}

func newTestService(repo *fakeRepo, w *world) *Service {
	s := NewService(repo, w.seams())
	s.now = func() time.Time { return fixedNow }
	return s
}

func states(p Progress) map[string]State {
	out := map[string]State{}
	for _, s := range p.Steps {
		out[s.Key] = s.State
	}
	return out
}

func TestProgress_EachStepIsDoneSkippedOrLeft(t *testing.T) {
	skipped := fixedNow.Add(-time.Hour)
	tests := []struct {
		name    string
		ticked  []string
		has     []string
		records []Record
		want    map[string]State
		dot     bool
	}{
		{"steps no role gives are left out", []string{"github"}, nil, nil, map[string]State{"github": Left}, true},
		{"done while the thing exists", []string{"github", "computer", "projects", "deploy_server"},
			[]string{"github", "computer", "projects", "deploy_server"}, nil,
			map[string]State{"github": Done, "computer": Done, "projects": Done, "deploy_server": Done}, false},
		{"a skip silences a step left", []string{"computer"}, nil, []Record{{Step: "computer", SkippedAt: &skipped}}, map[string]State{"computer": Skipped}, false},
		{"done beats a skip", []string{"github"}, []string{"github"}, []Record{{Step: "github", SkippedAt: &skipped}}, map[string]State{"github": Done}, false},
		{"the phone stays done once recorded", []string{"phone"}, nil, []Record{{Step: "phone", DoneAt: &skipped}}, map[string]State{"phone": Done}, false},
		{"projects left lights nothing until given", []string{"projects"}, nil, nil, map[string]State{"projects": Left}, false},
		{"projects left once given lights the dot", []string{"projects"}, nil, []Record{{Step: "projects", GivenAt: &skipped}}, map[string]State{"projects": Left}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, w := newFakeRepo(), newWorld()
			w.ticked["u-sam"] = tt.ticked
			for _, name := range tt.has {
				w.has[name]["u-sam"] = true
			}
			for _, r := range tt.records {
				r.UserID = "u-sam"
				repo.records[[2]string{"u-sam", r.Step}] = r
			}
			got, err := newTestService(repo, w).Progress(t.Context(), "u-sam")
			require.NoError(t, err)
			assert.Equal(t, tt.want, states(got))
			assert.Equal(t, tt.dot, got.Dot)
		})
	}
}

func TestProgress_SaysWhetherTheyAreInAProjectWhenNoRoleGivesProjects(t *testing.T) {
	repo, w := newFakeRepo(), newWorld()
	w.ticked["u-sam"], w.ticked["u-bob"] = []string{"github"}, []string{"github"}
	w.has["projects"]["u-bob"] = true
	got, err := newTestService(repo, w).ProgressFor(t.Context(), []string{"u-sam", "u-bob"})
	require.NoError(t, err)
	assert.False(t, got["u-sam"].InAProject)
	assert.True(t, got["u-bob"].InAProject)
}

func TestProgress_StepsComeInTheWizardsOrderWithWhenTheyWereSkippedOrDone(t *testing.T) {
	repo, w := newFakeRepo(), newWorld()
	w.ticked["u-sam"] = []string{"phone", "computer", "github"}
	connected := fixedNow.Add(-time.Hour)
	repo.records[[2]string{"u-sam", "phone"}] = Record{UserID: "u-sam", Step: "phone", DoneAt: &connected}
	s := newTestService(repo, w)
	_, err := s.Skip(t.Context(), "u-sam", "computer")
	require.NoError(t, err)

	got, err := s.Progress(t.Context(), "u-sam")
	require.NoError(t, err)
	assert.Equal(t, []Step{{Key: "github", State: Left}, {Key: "computer", State: Skipped, At: &fixedNow}, {Key: "phone", State: Done, At: &connected}}, got.Steps)
}

func TestSkip_OnlyTheCallersOwnSteps(t *testing.T) {
	repo, w := newFakeRepo(), newWorld()
	w.ticked["u-sam"] = []string{"github"}
	s := newTestService(repo, w)

	_, err := s.Skip(t.Context(), "", "github")
	require.ErrorIs(t, err, apperrs.ErrUnauthorized)
	_, err = s.Skip(t.Context(), "u-sam", "phone")
	require.ErrorIs(t, err, apperrs.ErrNotFound, "a step no role of theirs gives")
	_, err = s.Unskip(t.Context(), "u-sam", "profile")
	require.ErrorIs(t, err, apperrs.ErrNotFound)
	assert.Empty(t, repo.records)

	got, err := s.Skip(t.Context(), "u-sam", "github")
	require.NoError(t, err)
	assert.Equal(t, map[string]State{"github": Skipped}, states(got))
	got, err = s.Unskip(t.Context(), "u-sam", "github")
	require.NoError(t, err)
	assert.Equal(t, map[string]State{"github": Left}, states(got))
	assert.Equal(t, []string{"u-sam", "u-sam"}, repo.announced())
}

func TestService_FailedReadsAndWritesSayWhatFailed(t *testing.T) {
	boom := errors.New("disk gone")
	repo, w := newFakeRepo(), newWorld()
	w.ticked["u-sam"] = []string{"github"}
	s := newTestService(repo, w)

	w.err = boom
	for name, call := range map[string]func() error{
		"progress":  func() error { _, err := s.Progress(t.Context(), "u-sam"); return err },
		"skip":      func() error { _, err := s.Skip(t.Context(), "u-sam", "github"); return err },
		"first run": func() error { _, err := s.FinishFirstRun(t.Context(), "u-sam"); return err },
	} {
		assert.ErrorIs(t, call(), boom, name)
	}
	w.err = nil
	repo.err = boom
	for name, call := range map[string]func() error{
		"records":   func() error { _, err := s.Progress(t.Context(), "u-sam"); return err },
		"skip":      func() error { _, err := s.Skip(t.Context(), "u-sam", "github"); return err },
		"unskip":    func() error { _, err := s.Unskip(t.Context(), "u-sam", "github"); return err },
		"first run": func() error { _, err := s.FinishFirstRun(t.Context(), "u-sam"); return err },
	} {
		assert.ErrorIs(t, call(), boom, name)
	}
	_, err := s.FinishFirstRun(t.Context(), "")
	require.ErrorIs(t, err, apperrs.ErrUnauthorized)
	_, err = s.Progress(t.Context(), "")
	require.ErrorIs(t, err, apperrs.ErrUnauthorized)
}

func TestFinishFirstRun_RecordsItAndAnnouncesThePerson(t *testing.T) {
	repo, w := newFakeRepo(), newWorld()
	got, err := newTestService(repo, w).FinishFirstRun(t.Context(), "u-sam")
	require.NoError(t, err)
	assert.True(t, got.FirstRunDone)
	assert.Equal(t, []string{"u-sam"}, repo.announced())
}

func event(t *testing.T, topic string, payload any) eventbus.Event {
	t.Helper()
	raw, err := json.Marshal(payload)
	require.NoError(t, err)
	return eventbus.Event{ID: "evt-1", Topic: topic, Payload: raw}
}

// TestGiving_BringsASkippedStepBack: a skip lasts until the step is given again, through a tick a held role gains or
// a role that ticks it being given; only after the first run does the step count as given.
func TestGiving_BringsASkippedStepBack(t *testing.T) {
	tests := []struct {
		name  string
		event eventbus.Event
	}{
		{"a role they hold gains the tick", eventbus.Event{Topic: "role.updated", Payload: json.RawMessage(`{"role_id":"role-dev","workspace_id":"ws-1","welcome_steps_added":["projects"]}`)}},
		{"they are given a role that ticks it", eventbus.Event{Topic: "workspace.member.updated", Payload: json.RawMessage(`{"user_id":"u-sam","workspace_id":"ws-1","role_ids_added":["role-dev"]}`)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, w := newFakeRepo(), newWorld()
			w.ticked["u-sam"] = []string{"projects"}
			w.roleSteps["role-dev"] = []string{"projects"}
			w.holders["role-dev"] = []string{"u-sam"}
			w.has["first_run"]["u-sam"] = true
			s := newTestService(repo, w)
			_, err := s.Skip(t.Context(), "u-sam", "projects")
			require.NoError(t, err)

			handle := s.HandleRoleUpdated
			if tt.event.Topic != "role.updated" {
				handle = s.HandleMemberChanged
			}
			require.NoError(t, handle(t.Context(), tt.event))

			got, err := s.Progress(t.Context(), "u-sam")
			require.NoError(t, err)
			assert.Equal(t, []Step{{Key: "projects", State: Left, Given: true}}, got.Steps)
			assert.True(t, got.Dot, "projects lights the dot once given")
			assert.Equal(t, []string{"u-sam", "u-sam"}, repo.announced())
		})
	}
}

func TestGiving_BeforeTheFirstRunClearsTheSkipButGivesNothing(t *testing.T) {
	repo, w := newFakeRepo(), newWorld()
	w.ticked["u-sam"] = []string{"projects"}
	w.holders["role-dev"] = []string{"u-sam"}
	s := newTestService(repo, w)
	require.NoError(t, s.HandleRoleUpdated(t.Context(), event(t, "role.updated", map[string]any{"role_id": "role-dev", "welcome_steps_added": []string{"projects"}})))

	got, err := s.Progress(t.Context(), "u-sam")
	require.NoError(t, err)
	assert.False(t, got.Dot)
}

func TestHandlers_AnnounceWhoseProgressMayHaveMoved(t *testing.T) {
	tests := []struct {
		name   string
		handle func(*Service) eventbus.Handler
		event  string
		want   []string
	}{
		{"a computer change names its owner", func(s *Service) eventbus.Handler { return s.HandlePersonChanged }, `{"computer_id":"c-1","user_id":"u-sam"}`, []string{"u-sam"}},
		{"a deploy runner names who enrolled it", func(s *Service) eventbus.Handler { return s.HandleRunnerChanged }, `{"runner_id":"r-1","enrolled_by":"u-sam"}`, []string{"u-sam"}},
		{"a runner from before names nobody", func(s *Service) eventbus.Handler { return s.HandleRunnerChanged }, `{"runner_id":"r-1"}`, nil},
		{"a browser session is no phone", func(s *Service) eventbus.Handler { return s.HandleSessionCreated }, `{"user_id":"u-sam","client":"browser"}`, nil},
		{"a role taken away announces its holder", func(s *Service) eventbus.Handler { return s.HandleMemberChanged }, `{"user_id":"u-sam","workspace_id":"ws-1"}`, []string{"u-sam"}},
		{"a role save without new ticks announces its holders", func(s *Service) eventbus.Handler { return s.HandleRoleUpdated }, `{"role_id":"role-dev","welcome_steps_added":[]}`, []string{"u-sam", "u-lena"}},
		{"a project rule for a person", func(s *Service) eventbus.Handler { return s.HandleGrantChanged }, `{"resource_type":"project","resource_id":"p-1","target":"person","user_id":"u-sam"}`, []string{"u-sam"}},
		{"a project rule for a role", func(s *Service) eventbus.Handler { return s.HandleGrantChanged }, `{"resource_type":"project","resource_id":"p-1","target":"role","role_id":"role-dev"}`, []string{"u-sam", "u-lena"}},
		{"an Access page save", func(s *Service) eventbus.Handler { return s.HandleGrantChanged }, `{"resource_type":"workspace","resource_id":"ws-1","target":"role"}`, []string{"u-sam", "u-bob"}},
		{"a doc rule moves nobody's projects", func(s *Service) eventbus.Handler { return s.HandleGrantChanged }, `{"resource_type":"doc","resource_id":"d-1","target":"person","user_id":"u-sam"}`, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, w := newFakeRepo(), newWorld()
			w.holders["role-dev"] = []string{"u-sam", "u-lena"}
			w.members["ws-1"] = []string{"u-sam", "u-bob"}
			s := newTestService(repo, w)
			require.NoError(t, tt.handle(s)(t.Context(), eventbus.Event{ID: "evt-1", Topic: "t", Payload: json.RawMessage(tt.event)}))
			assert.Equal(t, tt.want, repo.announced())
		})
	}
}

func TestHandleSessionCreated_APhoneSignInStaysDone(t *testing.T) {
	repo, w := newFakeRepo(), newWorld()
	w.ticked["u-sam"] = []string{"phone"}
	s := newTestService(repo, w)
	signIn := `{"session_id":"s-1","user_id":"u-sam","client":"phone"}`
	require.NoError(t, s.HandleSessionCreated(t.Context(), eventbus.Event{ID: "evt-1", Payload: json.RawMessage(signIn)}))
	s.now = func() time.Time { return fixedNow.Add(time.Hour) }
	require.NoError(t, s.HandleSessionCreated(t.Context(), eventbus.Event{ID: "evt-2", Payload: json.RawMessage(signIn)}))

	got, err := s.Progress(t.Context(), "u-sam")
	require.NoError(t, err)
	assert.Equal(t, []Step{{Key: "phone", State: Done, At: &fixedNow}}, got.Steps, "the first sign-in's time stays")
}

func TestHandlers_ABadPayloadIsFatalAndAFailedReadRetries(t *testing.T) {
	repo, w := newFakeRepo(), newWorld()
	s := newTestService(repo, w)
	bad := eventbus.Event{ID: "evt-1", Topic: "t", Payload: json.RawMessage(`{`)}
	for _, h := range []eventbus.Handler{s.HandlePersonChanged, s.HandleRunnerChanged, s.HandleSessionCreated, s.HandleMemberChanged, s.HandleRoleUpdated, s.HandleGrantChanged} {
		require.ErrorIs(t, h(t.Context(), bad), apperrs.ErrFatal)
	}
	w.err = errors.New("disk gone")
	for name, run := range map[string]func() error{
		"role holders": func() error {
			return s.HandleRoleUpdated(t.Context(), eventbus.Event{Payload: json.RawMessage(`{"role_id":"role-dev"}`)})
		},
		"role steps": func() error {
			return s.HandleMemberChanged(t.Context(), eventbus.Event{Payload: json.RawMessage(`{"user_id":"u-sam","role_ids_added":["role-dev"]}`)})
		},
		"workspace members": func() error {
			return s.HandleGrantChanged(t.Context(), eventbus.Event{Payload: json.RawMessage(`{"resource_type":"workspace","resource_id":"ws-1"}`)})
		},
	} {
		assert.ErrorIs(t, run(), apperrs.ErrRetryable, name)
	}
}
