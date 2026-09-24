package dns

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// fakeRepo is an in-memory dns.Repo for use-case tests.
type fakeRepo struct {
	mu          sync.Mutex
	hostnames   map[string]*ServiceHostname
	hostErr     error
	tunnels     map[string]*Tunnel
	tunnelErr   error
	gateways    map[string]*Gateway
	gatewayErr  error
	exposures   map[string]*Exposure
	exposureErr error
	outbox      []eventbus.OutboxEvent
	outboxErr   error
	svcToken    *ServiceToken
	svcTokenErr error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		hostnames: map[string]*ServiceHostname{},
		tunnels:   map[string]*Tunnel{},
		gateways:  map[string]*Gateway{},
		exposures: map[string]*Exposure{},
	}
}

func (f *fakeRepo) UpsertServiceHostname(_ context.Context, sh ServiceHostname, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.hostErr != nil {
		return f.hostErr
	}
	f.hostnames[sh.Service] = &sh
	f.outbox = append(f.outbox, marshalEvents(evts)...)
	return nil
}

func (f *fakeRepo) GetServiceHostname(_ context.Context, service string) (*ServiceHostname, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.hostErr != nil {
		return nil, f.hostErr
	}
	sh, ok := f.hostnames[service]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	return sh, nil
}

func (f *fakeRepo) ListServiceHostnames(_ context.Context) ([]*ServiceHostname, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.hostErr != nil {
		return nil, f.hostErr
	}
	out := make([]*ServiceHostname, 0, len(f.hostnames))
	for _, sh := range f.hostnames {
		out = append(out, sh)
	}
	return out, nil
}

func (f *fakeRepo) DeleteServiceHostname(_ context.Context, service string, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.hostErr != nil {
		return f.hostErr
	}
	if _, ok := f.hostnames[service]; !ok {
		return apperrs.ErrNotFound
	}
	delete(f.hostnames, service)
	f.outbox = append(f.outbox, marshalEvents(evts)...)
	return nil
}

func (f *fakeRepo) RecordChanged(_ context.Context, evt eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.outboxErr != nil {
		return f.outboxErr
	}
	f.outbox = append(f.outbox, marshalEvents([]eventbus.OutboxEvent{evt})...)
	return nil
}

func (f *fakeRepo) SaveTunnel(_ context.Context, t Tunnel, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.tunnelErr != nil {
		return f.tunnelErr
	}
	copied := t
	f.tunnels[t.ID] = &copied
	f.outbox = append(f.outbox, marshalEvents(evts)...)
	return nil
}

func (f *fakeRepo) GetTunnel(_ context.Context, tunnelID string) (*Tunnel, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.tunnelErr != nil {
		return nil, f.tunnelErr
	}
	t, ok := f.tunnels[tunnelID]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	copied := *t
	return &copied, nil
}

func (f *fakeRepo) ListTunnels(_ context.Context) ([]*Tunnel, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.tunnelErr != nil {
		return nil, f.tunnelErr
	}
	out := make([]*Tunnel, 0, len(f.tunnels))
	for _, t := range f.tunnels {
		copied := *t
		out = append(out, &copied)
	}
	return out, nil
}

func (f *fakeRepo) DeleteTunnel(_ context.Context, tunnelID string, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.tunnelErr != nil {
		return f.tunnelErr
	}
	if _, ok := f.tunnels[tunnelID]; !ok {
		return apperrs.ErrNotFound
	}
	delete(f.tunnels, tunnelID)
	f.outbox = append(f.outbox, marshalEvents(evts)...)
	return nil
}

func (f *fakeRepo) SaveGateway(_ context.Context, g Gateway, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.gatewayErr != nil {
		return f.gatewayErr
	}
	copied := g
	f.gateways[g.ID] = &copied
	f.outbox = append(f.outbox, marshalEvents(evts)...)
	return nil
}

func (f *fakeRepo) GetGateway(_ context.Context, gatewayID string) (*Gateway, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.gatewayErr != nil {
		return nil, f.gatewayErr
	}
	g, ok := f.gateways[gatewayID]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	copied := *g
	return &copied, nil
}

func (f *fakeRepo) GetGatewayByNetwork(_ context.Context, dockerNetwork string) (*Gateway, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.gatewayErr != nil {
		return nil, f.gatewayErr
	}
	for _, g := range f.gateways {
		if g.DockerNetwork == dockerNetwork {
			copied := *g
			return &copied, nil
		}
	}
	return nil, apperrs.ErrNotFound
}

func (f *fakeRepo) ListGateways(_ context.Context) ([]*Gateway, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.gatewayErr != nil {
		return nil, f.gatewayErr
	}
	out := make([]*Gateway, 0, len(f.gateways))
	for _, g := range f.gateways {
		copied := *g
		out = append(out, &copied)
	}
	return out, nil
}

func (f *fakeRepo) ListGatewaysByMachine(_ context.Context, machine string) ([]*Gateway, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.gatewayErr != nil {
		return nil, f.gatewayErr
	}
	var out []*Gateway
	for _, g := range f.gateways {
		if g.Machine == machine {
			copied := *g
			out = append(out, &copied)
		}
	}
	return out, nil
}

func (f *fakeRepo) DeleteGateway(_ context.Context, gatewayID string, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.gatewayErr != nil {
		return f.gatewayErr
	}
	if _, ok := f.gateways[gatewayID]; !ok {
		return apperrs.ErrNotFound
	}
	delete(f.gateways, gatewayID)
	f.outbox = append(f.outbox, marshalEvents(evts)...)
	return nil
}

func (f *fakeRepo) SaveExposure(_ context.Context, e Exposure, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.exposureErr != nil {
		return f.exposureErr
	}
	copied := e
	f.exposures[e.ID] = &copied
	f.outbox = append(f.outbox, marshalEvents(evts)...)
	return nil
}

func (f *fakeRepo) GetExposure(_ context.Context, exposureID string) (*Exposure, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.exposureErr != nil {
		return nil, f.exposureErr
	}
	e, ok := f.exposures[exposureID]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	copied := *e
	return &copied, nil
}

func (f *fakeRepo) ListExposures(_ context.Context) ([]*Exposure, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.exposureErr != nil {
		return nil, f.exposureErr
	}
	out := make([]*Exposure, 0, len(f.exposures))
	for _, e := range f.exposures {
		copied := *e
		out = append(out, &copied)
	}
	return out, nil
}

func (f *fakeRepo) ListExposuresByGateway(_ context.Context, gatewayID string) ([]*Exposure, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.exposureErr != nil {
		return nil, f.exposureErr
	}
	var out []*Exposure
	for _, e := range f.exposures {
		if e.GatewayID == gatewayID {
			copied := *e
			out = append(out, &copied)
		}
	}
	return out, nil
}

func (f *fakeRepo) ListExposuresByService(_ context.Context, serviceID string) ([]*Exposure, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.exposureErr != nil {
		return nil, f.exposureErr
	}
	var out []*Exposure
	for _, e := range f.exposures {
		if e.ServiceID == serviceID {
			copied := *e
			out = append(out, &copied)
		}
	}
	return out, nil
}

func (f *fakeRepo) ListExposuresByServiceName(_ context.Context, serviceName string) ([]*Exposure, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.exposureErr != nil {
		return nil, f.exposureErr
	}
	var out []*Exposure
	for _, e := range f.exposures {
		if e.Service == serviceName {
			copied := *e
			out = append(out, &copied)
		}
	}
	return out, nil
}

func (f *fakeRepo) DeleteExposure(_ context.Context, exposureID string, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.exposureErr != nil {
		return f.exposureErr
	}
	if _, ok := f.exposures[exposureID]; !ok {
		return apperrs.ErrNotFound
	}
	delete(f.exposures, exposureID)
	f.outbox = append(f.outbox, marshalEvents(evts)...)
	return nil
}

// marshalEvents encodes outbox payloads the way the storage repo does, so
// tests can read them back as JSON.
func marshalEvents(evts []eventbus.OutboxEvent) []eventbus.OutboxEvent {
	out := make([]eventbus.OutboxEvent, 0, len(evts))
	for _, evt := range evts {
		if _, isRaw := evt.Payload.(json.RawMessage); !isRaw {
			b, err := json.Marshal(evt.Payload)
			if err == nil {
				evt.Payload = json.RawMessage(b)
			}
		}
		out = append(out, evt)
	}
	return out
}

func (f *fakeRepo) topics() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []string
	for _, evt := range f.outbox {
		out = append(out, evt.Topic)
	}
	return out
}

// fakeProvider is an in-memory DNSProvider for use-case tests.
type fakeProvider struct {
	mu           sync.Mutex
	zones        []Zone
	records      map[string][]Record
	nextID       int
	verifyErr    error
	zonesErr     error
	listErr      error
	createErr    error
	updateErr    error
	deleteErr    error
	propagateErr error
}

func newFakeProvider() *fakeProvider {
	return &fakeProvider{
		records: map[string][]Record{},
		zones:   []Zone{{ID: "z1", Name: "example.com", Status: "active"}},
	}
}

func (f *fakeProvider) Verify(_ context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.verifyErr
}

func (f *fakeProvider) ListZones(_ context.Context) ([]Zone, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.zonesErr != nil {
		return nil, f.zonesErr
	}
	return f.zones, nil
}

func (f *fakeProvider) ListRecords(_ context.Context, zoneID string) ([]Record, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.records[zoneID], nil
}

func (f *fakeProvider) CreateRecord(_ context.Context, zoneID string, in RecordInput) (*Record, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.createErr != nil {
		return nil, f.createErr
	}
	f.nextID++
	rec := &Record{ID: "r" + itoa(f.nextID), ZoneID: zoneID, Type: in.Type, Name: in.Name, Content: in.Content, TTL: in.TTL}
	f.records[zoneID] = append(f.records[zoneID], *rec)
	return rec, nil
}

func (f *fakeProvider) UpdateRecord(_ context.Context, zoneID, recordID string, in RecordInput) (*Record, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.updateErr != nil {
		return nil, f.updateErr
	}
	for i := range f.records[zoneID] {
		if f.records[zoneID][i].ID == recordID {
			f.records[zoneID][i].Type, f.records[zoneID][i].Name = in.Type, in.Name
			f.records[zoneID][i].Content, f.records[zoneID][i].TTL = in.Content, in.TTL
			f.records[zoneID][i].Proxied = in.Proxied
			r := f.records[zoneID][i]
			return &r, nil
		}
	}
	return nil, apperrs.ErrNotFound
}

func (f *fakeProvider) DeleteRecord(_ context.Context, zoneID, recordID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.deleteErr != nil {
		return f.deleteErr
	}
	for i := range f.records[zoneID] {
		if f.records[zoneID][i].ID == recordID {
			f.records[zoneID] = append(f.records[zoneID][:i], f.records[zoneID][i+1:]...)
			return nil
		}
	}
	return nil
}

func (f *fakeProvider) CheckPropagation(_ context.Context, _ string, _ Record) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.propagateErr
}

// fakeTokenProvider is an in-memory dns.TokenProvider (ticket 03): the
// composition-root seam that replaced dns's own OAuth/credential storage,
// now backed by the connectors service in production.
type fakeTokenProvider struct {
	token string
	err   error
}

func (f *fakeTokenProvider) Token(_ context.Context) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.token, nil
}

// fakeSettings is an in-memory dns.SettingsReader.
type fakeSettings struct {
	instanceURL string
	err         error
}

func (f *fakeSettings) GetInstanceURL(_ context.Context) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.instanceURL, nil
}

// routeCall records one RouteTunnelHostname/RemoveTunnelHostname invocation,
// so tests can assert which hostnames were routed onto which tunnel without
// re-implementing ingress-list bookkeeping in the fake.
type routeCall struct {
	TunnelID, Hostname, Service string
}

// fakeTunnelProvider is an in-memory dns.TunnelProvider for tunnel use-case tests.
type fakeTunnelProvider struct {
	mu          sync.Mutex
	tunnels     map[string]*Tunnel
	nextID      int
	createErr   error
	listErr     error
	getErr      error
	deleteErr   error
	routeErr    error
	removeErr   error
	rotateErr   error
	tokenErr    error
	rotated     string
	routeCalls  []routeCall
	removeCalls []routeCall
}

func newFakeTunnelProvider() *fakeTunnelProvider {
	return &fakeTunnelProvider{tunnels: map[string]*Tunnel{}}
}

func (f *fakeTunnelProvider) CreateTunnel(_ context.Context, name string) (*Tunnel, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.createErr != nil {
		return nil, f.createErr
	}
	f.nextID++
	t := &Tunnel{
		ID: "tunnel-" + itoa(f.nextID), Name: name, AccountID: "acct-1",
		Status: "inactive", Token: "tunnel-token-" + itoa(f.nextID),
	}
	copied := *t
	f.tunnels[t.ID] = &copied
	return t, nil
}

func (f *fakeTunnelProvider) ListTunnels(_ context.Context) ([]Tunnel, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.listErr != nil {
		return nil, f.listErr
	}
	out := make([]Tunnel, 0, len(f.tunnels))
	for _, t := range f.tunnels {
		out = append(out, *t)
	}
	return out, nil
}

func (f *fakeTunnelProvider) GetTunnel(_ context.Context, tunnelID string) (*Tunnel, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getErr != nil {
		return nil, f.getErr
	}
	t, ok := f.tunnels[tunnelID]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	copied := *t
	return &copied, nil
}

func (f *fakeTunnelProvider) DeleteTunnel(_ context.Context, tunnelID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.deleteErr != nil {
		return f.deleteErr
	}
	delete(f.tunnels, tunnelID)
	return nil
}

func (f *fakeTunnelProvider) RouteTunnelHostname(_ context.Context, tunnelID, hostname, service string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.routeCalls = append(f.routeCalls, routeCall{TunnelID: tunnelID, Hostname: hostname, Service: service})
	return f.routeErr
}

// ListTunnelHostnames replays the routes recorded by RouteTunnelHostname minus those removed since.
func (f *fakeTunnelProvider) ListTunnelHostnames(_ context.Context, tunnelID string) ([]TunnelRoute, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.listErr != nil {
		return nil, f.listErr
	}
	removed := map[string]bool{}
	for _, c := range f.removeCalls {
		if c.TunnelID == tunnelID {
			removed[c.Hostname] = true
		}
	}
	var out []TunnelRoute
	for _, c := range f.routeCalls {
		if c.TunnelID == tunnelID && !removed[c.Hostname] {
			out = append(out, TunnelRoute{Hostname: c.Hostname, Service: c.Service})
		}
	}
	return out, nil
}

func (f *fakeTunnelProvider) RemoveTunnelHostname(_ context.Context, tunnelID, hostname string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.removeCalls = append(f.removeCalls, routeCall{TunnelID: tunnelID, Hostname: hostname})
	return f.removeErr
}

func (f *fakeTunnelProvider) RotateTunnelCredentials(_ context.Context, tunnelID string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.rotateErr != nil {
		return "", f.rotateErr
	}
	if f.rotated != "" {
		return f.rotated, nil
	}
	return "rotated-token-" + tunnelID, nil
}

func (f *fakeTunnelProvider) TunnelToken(_ context.Context, tunnelID string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.tokenErr != nil {
		return "", f.tokenErr
	}
	return "current-token-" + tunnelID, nil
}

// fakeProvisioner is an in-memory dns.ServiceProvisioner.
type fakeProvisioner struct {
	mu               sync.Mutex
	calls            []AgentSpec
	result           *AgentProvisioned
	err              error
	deprovisionCalls []string
	deprovisionErr   error
}

func (f *fakeProvisioner) Provision(_ context.Context, in AgentSpec) (*AgentProvisioned, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, in)
	if f.err != nil {
		return nil, f.err
	}
	if f.result != nil {
		copied := *f.result
		return &copied, nil
	}
	return &AgentProvisioned{ServiceID: "svc-" + itoa(len(f.calls))}, nil
}

func (f *fakeProvisioner) Deprovision(_ context.Context, serviceID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deprovisionCalls = append(f.deprovisionCalls, serviceID)
	return f.deprovisionErr
}

// fakeContainerLookup is an in-memory dns.ContainerLookup: containers keyed by id, plus name/stackID
// indexes mirroring what a real deploy-side adapter resolves (ContainerByStackName/ContainerByStackID).
type fakeContainerLookup struct {
	mu          sync.Mutex
	byID        map[string]*ExposureTarget
	byStackName map[string]*ExposureTarget
	byStackID   map[string]*ExposureTarget
	err         error
}

func newFakeContainerLookup() *fakeContainerLookup {
	return &fakeContainerLookup{
		byID:        map[string]*ExposureTarget{},
		byStackName: map[string]*ExposureTarget{},
		byStackID:   map[string]*ExposureTarget{},
	}
}

// add registers target under its container id, and under stackName/stackID as a single-container stack's
// resolution paths (a run stack always has exactly one container, matching real deploy data).
func (f *fakeContainerLookup) add(stackName string, target ExposureTarget) *ExposureTarget {
	f.mu.Lock()
	defer f.mu.Unlock()
	copied := target
	f.byID[target.ContainerID] = &copied
	f.byStackName[stackName] = &copied
	f.byStackID[target.StackID] = &copied
	return &copied
}

func (f *fakeContainerLookup) ContainerByID(_ context.Context, containerID string) (*ExposureTarget, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	target, ok := f.byID[containerID]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	copied := *target
	return &copied, nil
}

func (f *fakeContainerLookup) ContainerByStackName(_ context.Context, stackName string) (*ExposureTarget, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	target, ok := f.byStackName[stackName]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	copied := *target
	return &copied, nil
}

func (f *fakeContainerLookup) ContainerByStackID(_ context.Context, stackID string) (*ExposureTarget, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	target, ok := f.byStackID[stackID]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	copied := *target
	return &copied, nil
}

// fakeRunnerJoiner is an in-memory dns.RunnerJoiner, recording each call for tests to assert against.
type fakeRunnerJoiner struct {
	mu    sync.Mutex
	calls []joinCall
	err   error
}

type joinCall struct {
	Machine, GatewayContainer string
	Networks                  []string
}

func (f *fakeRunnerJoiner) JoinNetworks(_ context.Context, machine, gatewayContainer string, networks []string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, joinCall{Machine: machine, GatewayContainer: gatewayContainer, Networks: networks})
	return f.err
}

// newTestService wires a Service over the given fakes with a fixed clock. A
// nil provider leaves the token-resolution path active (so tests can
// exercise resolveToken); a nil tokens defaults to an always-available fake
// token, since most tests fix Provider directly and never touch it.
func newTestService(repo *fakeRepo, provider *fakeProvider, tokens *fakeTokenProvider) *Service {
	if tokens == nil {
		tokens = &fakeTokenProvider{token: "at"}
	}
	cfg := Config{
		Repo:          repo,
		Tokens:        tokens,
		EncryptionKey: []byte("0123456789abcdef0123456789abcdef"),
		Settings:      &fakeSettings{instanceURL: "https://deploy.example.com"},
		Now:           func() time.Time { return time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC) },
	}
	if provider != nil {
		cfg.Provider = provider
	}
	return NewService(cfg)
}

// newGatewayService wires a Service over fakes with tunnel + provisioner +
// container lookup, the set gateway/exposure use-cases need.
func newGatewayService(repo *fakeRepo, tunnel *fakeTunnelProvider, prov *fakeProvisioner, containers *fakeContainerLookup) *Service {
	return newGatewayServiceWithJoiner(repo, tunnel, prov, containers, nil)
}

// newGatewayServiceWithJoiner is newGatewayService plus a RunnerJoiner, for tests exercising the immediate
// network-join path.
func newGatewayServiceWithJoiner(repo *fakeRepo, tunnel *fakeTunnelProvider, prov *fakeProvisioner, containers *fakeContainerLookup, joiner *fakeRunnerJoiner) *Service {
	cfg := Config{
		Repo:           repo,
		Provider:       newFakeProvider(),
		TunnelProvider: tunnel,
		Provisioner:    prov,
		Containers:     containers,
		EncryptionKey:  []byte("0123456789abcdef0123456789abcdef"),
		Settings:       &fakeSettings{instanceURL: "https://deploy.example.com"},
		Tokens:         &fakeTokenProvider{token: "at"},
		Now:            func() time.Time { return time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC) },
	}
	if joiner != nil {
		cfg.RunnerJoin = joiner
	}
	return NewService(cfg)
}

func itoa(n int) string {
	return string(rune('0' + n))
}

var errBoom = errors.New("boom")

func (f *fakeRepo) SaveAccessServiceToken(_ context.Context, t ServiceToken) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.svcTokenErr != nil {
		return f.svcTokenErr
	}
	f.svcToken = &t
	return nil
}

func (f *fakeRepo) GetAccessServiceToken(_ context.Context) (*ServiceToken, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.svcTokenErr != nil {
		return nil, f.svcTokenErr
	}
	if f.svcToken == nil {
		return nil, apperrs.ErrNotFound
	}
	t := *f.svcToken
	return &t, nil
}

func (f *fakeRepo) DeleteAccessServiceToken(_ context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.svcTokenErr != nil {
		return f.svcTokenErr
	}
	if f.svcToken == nil {
		return apperrs.ErrNotFound
	}
	f.svcToken = nil
	return nil
}

// fakeAccessProvider is an in-memory Cloudflare Access account; err fails every call.
type fakeAccessProvider struct {
	mu      sync.Mutex
	err     error
	apps    map[string]string
	tokens  map[string]string
	minted  int
	deleted []string
}

func newFakeAccessProvider() *fakeAccessProvider {
	return &fakeAccessProvider{apps: map[string]string{}, tokens: map[string]string{}}
}

func (f *fakeAccessProvider) CreateAccessApp(_ context.Context, hostname, serviceTokenID string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return "", f.err
	}
	id := "app-" + hostname
	f.apps[id] = serviceTokenID
	return id, nil
}

func (f *fakeAccessProvider) DeleteAccessApp(_ context.Context, appID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	delete(f.apps, appID)
	return nil
}

func (f *fakeAccessProvider) CreateServiceToken(_ context.Context, _ string) (*ServiceToken, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	f.minted++
	id := "st-" + itoa(f.minted)
	f.tokens[id] = "secret-" + itoa(f.minted)
	return &ServiceToken{ID: id, ClientID: "cid-" + id, ClientSecret: f.tokens[id]}, nil
}

func (f *fakeAccessProvider) RotateServiceToken(_ context.Context, tokenID string) (*ServiceToken, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	if _, ok := f.tokens[tokenID]; !ok {
		return nil, apperrs.ErrNotFound
	}
	f.tokens[tokenID] = "rotated-" + tokenID
	return &ServiceToken{ID: tokenID, ClientID: "cid-" + tokenID, ClientSecret: f.tokens[tokenID]}, nil
}

func (f *fakeAccessProvider) DeleteServiceToken(_ context.Context, tokenID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	delete(f.tokens, tokenID)
	f.deleted = append(f.deleted, tokenID)
	return nil
}
