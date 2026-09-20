package deploy

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// fakeGatewayLookup is an in-memory deploy.GatewayLookup for use-case tests.
type fakeGatewayLookup struct{ networks map[string]bool }

func (f fakeGatewayLookup) NetworkHasGateway(_ context.Context, network string) (bool, error) {
	return f.networks[network], nil
}

// fakeGatewayJoin is an in-memory deploy.GatewayJoin for use-case tests.
type fakeGatewayJoin struct {
	containers map[string]string
	err        error
}

func (f fakeGatewayJoin) GatewayForStackNetwork(_ context.Context, network string) (string, bool, error) {
	if f.err != nil {
		return "", false, f.err
	}
	container, ok := f.containers[network]
	return container, ok, nil
}

// exposureState is one exposure a fakeExposureManager tracks.
type exposureState struct {
	hostname string
	port     int
	network  string
}

// removeExposuresCall records one RemoveExposures invocation, so tests can assert which stack name and
// container ids it was called with.
type removeExposuresCall struct {
	stackName    string
	containerIDs []string
}

// fakeExposureManager is an in-memory deploy.ExposureManager for use-case
// tests: keyed by stack name, mirroring dns's own one-exposure-per-service
// v1 assumption.
type fakeExposureManager struct {
	mu          sync.Mutex
	exposures   map[string]exposureState
	ensureCalls int
	removeCalls int
	removes     []removeExposuresCall
}

func newFakeExposureManager() *fakeExposureManager {
	return &fakeExposureManager{exposures: map[string]exposureState{}}
}

func (f *fakeExposureManager) EnsureExposure(_ context.Context, hostname, service string, port int, network string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ensureCalls++
	f.exposures[service] = exposureState{hostname: hostname, port: port, network: network}
	return nil
}

func (f *fakeExposureManager) RemoveExposures(_ context.Context, stackName string, containerIDs []string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.removeCalls++
	f.removes = append(f.removes, removeExposuresCall{stackName: stackName, containerIDs: containerIDs})
	delete(f.exposures, stackName)
	return nil
}

func (f *fakeExposureManager) get(service string) (exposureState, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	e, ok := f.exposures[service]
	return e, ok
}

// validRunStack is a run-strategy, repo-driven base stack — the shape a
// branch deploy rule needs (a docker network to join and a build source to
// build the pushed ref from).
func validRunStack() Stack {
	return Stack{
		ID:            "svc-1",
		ProjectID:     "proj-1",
		Name:          "api",
		Slug:          "api",
		Machine:       "10.0.0.1:22",
		Strategy:      StrategyRun,
		DockerNetwork: "app-net",
		Env:           map[string]string{"PORT": "8080"},
		BuildSource:   &BuildSource{RepoOwner: "acme", RepoName: "api", Dockerfile: "Dockerfile"},
		Managed:       true,
	}
}

// newBranchTestService wires a deploy.Service with a project store that
// accepts proj-1, for branch deploy rule tests.
func newBranchTestService() (*Service, *fakeStackRepo, *fakeRepo) {
	deploys := newFakeRepo()
	stacks := newFakeStackRepo()
	containers := newFakeContainerRepo()
	projects := newFakeProjects()
	projects.exists["proj-1"] = true
	projects.repos["proj-1"] = []string{"acme/api"}
	s := NewService(deploys, stacks, containers, projects)
	s.now = func() time.Time { return time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC) }
	return s, stacks, deploys
}

func pushEvent(t *testing.T, p PushTrigger) eventbus.Event {
	t.Helper()
	b, err := json.Marshal(p)
	require.NoError(t, err)
	return eventbus.Event{Topic: "git.push", Payload: b}
}

func branchDeletedEvent(t *testing.T, p BranchDeletedTrigger) eventbus.Event {
	t.Helper()
	b, err := json.Marshal(p)
	require.NoError(t, err)
	return eventbus.Event{Topic: "git.branch_deleted", Payload: b}
}

func TestHandlePush_ExactRuleInPlace(t *testing.T) {
	s, stacks, deploys := newBranchTestService()
	base := validRunStack()
	base.BranchDeployRules = []BranchDeployRule{{Pattern: "main", DockerNetwork: "app-net"}}
	require.NoError(t, stacks.Create(context.Background(), &base))

	err := s.HandlePush(context.Background(), pushEvent(t, PushTrigger{Owner: "acme", Repo: "api", Branch: "main", SHA: "sha1"}))
	require.NoError(t, err)

	// No derived clone: the base stack itself is redeployed.
	svcs, err := stacks.ListByProject(context.Background(), "proj-1")
	require.NoError(t, err)
	assert.Len(t, svcs, 1)

	payload := deployRequestedPayload(t, deploys.of(TopicDeployRequested))
	assert.Equal(t, "api", payload.Service)
	assert.Equal(t, "sha1", payload.Ref)
	assert.Equal(t, "build", payload.Kind)
}

func TestHandlePush_WildcardRule_CreatesDerivedClone(t *testing.T) {
	s, stacks, deploys := newBranchTestService()
	base := validRunStack()
	base.BranchDeployRules = []BranchDeployRule{{Pattern: "feature/*", DockerNetwork: "app-net"}}
	require.NoError(t, stacks.Create(context.Background(), &base))

	err := s.HandlePush(context.Background(), pushEvent(t, PushTrigger{Owner: "acme", Repo: "api", Branch: "feature/discord-integration", SHA: "sha1"}))
	require.NoError(t, err)

	derived, err := stacks.GetBySlugAndMachine(context.Background(), "api-feature-discord-integration", base.Machine)
	require.NoError(t, err)
	assert.Equal(t, base.ID, derived.DerivedFrom)
	assert.Equal(t, "feature/discord-integration", derived.Branch)
	assert.Equal(t, base.Env, derived.Env)
	assert.Equal(t, "app-net", derived.DockerNetwork)

	payload := deployRequestedPayload(t, deploys.of(TopicDeployRequested))
	assert.Equal(t, "api-feature-discord-integration", payload.Service)
	assert.Equal(t, "sha1", payload.Ref)

	// The base stack list stays clean; the clone lists under it.
	top, err := s.ListStacks(context.Background(), "proj-1")
	require.NoError(t, err)
	assert.Len(t, top, 1)
	assert.Equal(t, "api", top[0].Name)

	branches, err := s.ListBranchDeployments(context.Background(), base.ID)
	require.NoError(t, err)
	require.Len(t, branches, 1)
	assert.Equal(t, derived.ID, branches[0].ID)
}

func TestHandlePush_WildcardRule_RepeatedPushUpdatesNotDuplicates(t *testing.T) {
	s, stacks, deploys := newBranchTestService()
	base := validRunStack()
	base.BranchDeployRules = []BranchDeployRule{{Pattern: "feature/*", DockerNetwork: "app-net"}}
	require.NoError(t, stacks.Create(context.Background(), &base))

	ev := pushEvent(t, PushTrigger{Owner: "acme", Repo: "api", Branch: "feature/x", SHA: "sha1"})
	require.NoError(t, s.HandlePush(context.Background(), ev))
	// The first deploy is still pending/running, so the second push
	// must not error — it collapses onto the in-flight deploy.
	err := s.HandlePush(context.Background(), pushEvent(t, PushTrigger{Owner: "acme", Repo: "api", Branch: "feature/x", SHA: "sha2"}))
	require.NoError(t, err)

	branches, err := s.ListBranchDeployments(context.Background(), base.ID)
	require.NoError(t, err)
	assert.Len(t, branches, 1, "a repeated push updates the existing clone, not a second one")
	_ = deploys
}

func TestHandlePush_NoMatchingRule_IsNoop(t *testing.T) {
	s, stacks, _ := newBranchTestService()
	base := validRunStack()
	base.BranchDeployRules = []BranchDeployRule{{Pattern: "main", DockerNetwork: "app-net"}}
	require.NoError(t, stacks.Create(context.Background(), &base))

	err := s.HandlePush(context.Background(), pushEvent(t, PushTrigger{Owner: "acme", Repo: "api", Branch: "chore/unrelated", SHA: "sha1"}))
	require.NoError(t, err)

	branches, err := s.ListBranchDeployments(context.Background(), base.ID)
	require.NoError(t, err)
	assert.Empty(t, branches)
}

func TestHandlePush_ImageOnlyStack_RedeploysLastHealthyImage(t *testing.T) {
	s, stacks, deploys := newBranchTestService()
	base := validRunStack()
	// A BuildSource with a repo pointer but no Dockerfile/compose path is
	// "image-only" (ticket 07 Answer): still watched by pushes for matching,
	// but nothing to build from.
	base.BuildSource.Dockerfile = ""
	base.BranchDeployRules = []BranchDeployRule{{Pattern: "main", DockerNetwork: "app-net"}}
	require.NoError(t, stacks.Create(context.Background(), &base))
	require.NoError(t, deploys.Create(context.Background(), &Deploy{
		ID: "d1", StackID: base.ID, Service: base.Name, Image: "ghcr.io/acme/api:v1", Status: StatusHealthy, CreatedAt: time.Now(),
	}))

	err := s.HandlePush(context.Background(), pushEvent(t, PushTrigger{Owner: "acme", Repo: "api", Branch: "main", SHA: "sha1"}))
	require.NoError(t, err)

	payload := deployRequestedPayload(t, deploys.of(TopicDeployRequested))
	assert.Equal(t, "ghcr.io/acme/api:v1", payload.Image)
	assert.Empty(t, payload.Ref)
}

func TestHandlePush_HostnameTemplate_EnsuresExposure(t *testing.T) {
	s, stacks, _ := newBranchTestService()
	exposures := newFakeExposureManager()
	s.SetExposureManager(exposures)
	base := validRunStack()
	base.BranchDeployRules = []BranchDeployRule{{Pattern: "feature/*", DockerNetwork: "app-net", HostnameTemplate: "{branch}.example.com", Port: 8080}}
	require.NoError(t, stacks.Create(context.Background(), &base))

	require.NoError(t, s.HandlePush(context.Background(), pushEvent(t, PushTrigger{Owner: "acme", Repo: "api", Branch: "feature/x", SHA: "sha1"})))

	got, ok := exposures.get("api-feature-x")
	require.True(t, ok)
	assert.Equal(t, "feature-x.example.com", got.hostname)
	assert.Equal(t, 8080, got.port)
	assert.Equal(t, "app-net", got.network)
	assert.Equal(t, 1, exposures.ensureCalls)

	// A second push to the same branch is idempotent: same hostname/port,
	// no duplicate exposure call beyond the no-op check.
	require.NoError(t, s.HandlePush(context.Background(), pushEvent(t, PushTrigger{Owner: "acme", Repo: "api", Branch: "feature/x", SHA: "sha2"})))
	_, ok = exposures.get("api-feature-x")
	require.True(t, ok)
}

func TestHandleBranchDeleted_TearsDownDerivedClone(t *testing.T) {
	s, stacks, _ := newBranchTestService()
	exposures := newFakeExposureManager()
	s.SetExposureManager(exposures)
	base := validRunStack()
	base.BranchDeployRules = []BranchDeployRule{{Pattern: "feature/*", DockerNetwork: "app-net", HostnameTemplate: "{branch}.example.com", Port: 8080}}
	require.NoError(t, stacks.Create(context.Background(), &base))
	require.NoError(t, s.HandlePush(context.Background(), pushEvent(t, PushTrigger{Owner: "acme", Repo: "api", Branch: "feature/x", SHA: "sha1"})))
	_, err := stacks.GetBySlugAndMachine(context.Background(), "api-feature-x", base.Machine)
	require.NoError(t, err)

	err = s.HandleBranchDeleted(context.Background(), branchDeletedEvent(t, BranchDeletedTrigger{Owner: "acme", Repo: "api", Branch: "feature/x"}))
	require.NoError(t, err)

	_, err = stacks.GetBySlugAndMachine(context.Background(), "api-feature-x", base.Machine)
	assert.ErrorIs(t, err, apperrs.ErrNotFound)
	_, ok := exposures.get("api-feature-x")
	assert.False(t, ok, "teardown removes the exposure")
	assert.Equal(t, 1, exposures.removeCalls)
}

func TestHandleBranchDeleted_ExactInPlaceRule_LeavesBaseStackAlone(t *testing.T) {
	s, stacks, _ := newBranchTestService()
	base := validRunStack()
	base.BranchDeployRules = []BranchDeployRule{{Pattern: "main", DockerNetwork: "app-net"}}
	require.NoError(t, stacks.Create(context.Background(), &base))

	err := s.HandleBranchDeleted(context.Background(), branchDeletedEvent(t, BranchDeletedTrigger{Owner: "acme", Repo: "api", Branch: "main"}))
	require.NoError(t, err)

	_, err = stacks.GetByID(context.Background(), base.ID)
	require.NoError(t, err, "an exact in-place rule's base stack is never torn down by a branch delete")
}

func TestHandleBranchDeleted_NothingToTearDown_IsNoop(t *testing.T) {
	s, stacks, _ := newBranchTestService()
	base := validRunStack()
	require.NoError(t, stacks.Create(context.Background(), &base))

	err := s.HandleBranchDeleted(context.Background(), branchDeletedEvent(t, BranchDeletedTrigger{Owner: "acme", Repo: "api", Branch: "feature/never-deployed"}))
	require.NoError(t, err)
}

func TestCheckBranchDeployRules_HostnameTemplateRequiresGateway(t *testing.T) {
	s, _, _ := newBranchTestService()
	s.SetGatewayLookup(fakeGatewayLookup{networks: map[string]bool{"app-net": true}})

	stack := validRunStack()
	stack.ID = ""
	stack.BranchDeployRules = []BranchDeployRule{{Pattern: "feature/*", DockerNetwork: "no-gateway-net", HostnameTemplate: "{branch}.example.com", Port: 8080}}
	_, err := s.CreateStack(context.Background(), stack, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, apperrs.ErrInvalid)

	stack.BranchDeployRules[0].DockerNetwork = "app-net"
	_, err = s.CreateStack(context.Background(), stack, nil)
	require.NoError(t, err)
}

func TestDeleteStack_RemovesExposure(t *testing.T) {
	s, stacks, _ := newBranchTestService()
	exposures := newFakeExposureManager()
	s.SetExposureManager(exposures)
	base := validRunStack()
	require.NoError(t, stacks.Create(context.Background(), &base))
	exposures.exposures[base.Name] = exposureState{hostname: "api.example.com", port: 8080, network: "app-net"}

	require.NoError(t, s.DeleteStack(context.Background(), base.ID))

	_, ok := exposures.get(base.Name)
	assert.False(t, ok)
	assert.Equal(t, 1, exposures.removeCalls)
}
