package pairing

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/harness/harnesstest"
	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// fakeRepo is an in-memory pairing.Repo for use-case tests.
type fakeRepo struct {
	mu           sync.Mutex
	computers    map[string]Computer
	defaults     map[string]Defaults
	projectLinks map[string]ProjectLink
	setups       map[string][]ProviderSetup
	turns        map[string]SetupTurn
	outbox       []eventbus.OutboxEvent
	saveErr      error
	getErr       error
	listSetupErr error
	listTurnsErr error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{computers: map[string]Computer{}, defaults: map[string]Defaults{}, projectLinks: map[string]ProjectLink{}, setups: map[string][]ProviderSetup{}, turns: map[string]SetupTurn{}}
}

func (f *fakeRepo) SaveComputer(_ context.Context, c Computer, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.saveErr != nil {
		return f.saveErr
	}
	if existing, ok := f.computers[c.ID]; ok {
		c.Tunnel = existing.Tunnel
	}
	f.computers[c.ID] = c
	f.outbox = append(f.outbox, evts...)
	return nil
}

func (f *fakeRepo) GetComputer(_ context.Context, userID, id string) (*Computer, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getErr != nil {
		return nil, f.getErr
	}
	c, ok := f.computers[id]
	if !ok || c.UserID != userID {
		return nil, apperrs.ErrNotFound
	}
	copied := c
	return &copied, nil
}

func (f *fakeRepo) ListComputers(_ context.Context, userID string) ([]Computer, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]Computer, 0)
	for _, c := range f.computers {
		if c.UserID == userID {
			out = append(out, c)
		}
	}
	return out, nil
}

func (f *fakeRepo) DeleteComputer(_ context.Context, userID, id string, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	c, ok := f.computers[id]
	if !ok || c.UserID != userID {
		return apperrs.ErrNotFound
	}
	delete(f.computers, id)
	f.outbox = append(f.outbox, evts...)
	return nil
}

func (f *fakeRepo) GetDefaults(_ context.Context, userID string) (Defaults, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.defaults[userID], nil
}

func (f *fakeRepo) SaveDefaults(_ context.Context, d Defaults) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.defaults[d.UserID] = d
	return nil
}

func (f *fakeRepo) GetComputerByID(_ context.Context, id string) (*Computer, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getErr != nil {
		return nil, f.getErr
	}
	c, ok := f.computers[id]
	if !ok {
		return nil, apperrs.ErrNotFound
	}
	copied := c
	return &copied, nil
}

func (f *fakeRepo) GetProjectLink(_ context.Context, projectID string) (ProjectLink, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.projectLinks[projectID], nil
}

func (f *fakeRepo) SaveProjectLink(_ context.Context, link ProjectLink) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.saveErr != nil {
		return f.saveErr
	}
	f.projectLinks[link.ProjectID] = link
	return nil
}

func (f *fakeRepo) DeleteProjectLink(_ context.Context, projectID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.projectLinks, projectID)
	return nil
}

func (f *fakeRepo) SetSetupConfirmedAt(_ context.Context, userID, computerID string, at *time.Time, evt eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.saveErr != nil {
		return f.saveErr
	}
	c, ok := f.computers[computerID]
	if !ok || c.UserID != userID {
		return apperrs.ErrNotFound
	}
	c.SetupConfirmedAt = at
	f.computers[computerID] = c
	f.outbox = append(f.outbox, evt)
	return nil
}

func (f *fakeRepo) ListProviderSetups(_ context.Context, computerID string) ([]ProviderSetup, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.listSetupErr != nil {
		return nil, f.listSetupErr
	}
	return slices.Clone(f.setups[computerID]), nil
}

func (f *fakeRepo) SaveProviderSetup(_ context.Context, computerID string, p ProviderSetup, _ time.Time, evt eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.saveErr != nil {
		return f.saveErr
	}
	rows := slices.DeleteFunc(f.setups[computerID], func(existing ProviderSetup) bool { return existing.Provider == p.Provider })
	f.setups[computerID] = append(rows, p)
	f.outbox = append(f.outbox, evt)
	return nil
}

func (f *fakeRepo) SaveSetupTurn(_ context.Context, t SetupTurn, evts ...eventbus.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.saveErr != nil {
		return f.saveErr
	}
	t.Transcript = slices.Clone(t.Transcript)
	f.turns[t.ID] = t
	f.outbox = append(f.outbox, evts...)
	return nil
}

func (f *fakeRepo) ListLatestSetupTurns(_ context.Context, computerID string) ([]SetupTurnSummary, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.listTurnsErr != nil {
		return nil, f.listTurnsErr
	}
	latest := map[string]SetupTurn{}
	for _, t := range f.turns {
		if t.ComputerID != computerID {
			continue
		}
		if prev, ok := latest[t.Provider]; ok && (prev.StartedAt.After(t.StartedAt) || (prev.StartedAt.Equal(t.StartedAt) && prev.ID > t.ID)) {
			continue
		}
		latest[t.Provider] = t
	}
	out := []SetupTurnSummary{}
	for _, t := range latest {
		out = append(out, SetupTurnSummary{RunID: t.RunID, TurnID: t.ID, Provider: t.Provider, ProviderName: t.ProviderName, Model: t.Model, State: t.State, Status: t.Status, UpdatedAt: t.UpdatedAt})
	}
	slices.SortFunc(out, func(a, b SetupTurnSummary) int { return strings.Compare(a.Provider, b.Provider) })
	return out, nil
}

func (f *fakeRepo) SetSetupMCPToken(_ context.Context, userID, computerID, sealed string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	c, ok := f.computers[computerID]
	if !ok || c.UserID != userID {
		return apperrs.ErrNotFound
	}
	c.SetupMCPToken = sealed
	f.computers[computerID] = c
	return nil
}

// fakeExchanger scripts the pairing half of a harness.Client; the listing half comes from the embedded harnesstest.Client.
type fakeExchanger struct {
	harnesstest.Client
	result      harness.PairResult
	exchangeErr error
	version     string
	versionErr  error
	probedURL   string
	pairedURL   string
}

func (f *fakeExchanger) Pair(_ context.Context, serverURL, _ string) (harness.PairResult, error) {
	f.pairedURL = serverURL
	if f.exchangeErr != nil {
		return harness.PairResult{}, f.exchangeErr
	}
	if f.versionErr != nil {
		return harness.PairResult{}, f.versionErr
	}
	r := f.result
	r.Version = f.version
	return r, nil
}

func (f *fakeExchanger) Version(_ context.Context, serverURL string) (string, error) {
	f.probedURL = serverURL
	if f.versionErr != nil {
		return "", f.versionErr
	}
	return f.version, nil
}

// registry wraps exch as the only T3 client.
func registry(exch *fakeExchanger) harness.Registry {
	return harness.Registry{harness.KindT3Code: exch}
}

var errBoom = errors.New("boom")

// fakeTokens is an in-memory MCPTokens: one active token per user's computer, revoked on replace like auth's.
type fakeTokens struct {
	mu        sync.Mutex
	active    map[string]MCPToken
	seq       int
	revoked   []string
	mintErr   error
	revokeErr error
}

func newFakeTokens() *fakeTokens {
	return &fakeTokens{active: map[string]MCPToken{}}
}

func (f *fakeTokens) MintComputerToken(_ context.Context, userID, computerID, computerName string) (string, *MCPToken, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.mintErr != nil {
		return "", nil, f.mintErr
	}
	if old, ok := f.active[userID+"/"+computerID]; ok {
		f.revoked = append(f.revoked, old.ID)
	}
	f.seq++
	raw := fmt.Sprintf("dep_%043d", f.seq)
	token := MCPToken{ID: fmt.Sprintf("pat-%d", f.seq), Name: "Nexul MCP on " + computerName, Prefix: raw[len(raw)-6:]}
	f.active[userID+"/"+computerID] = token
	return raw, &token, nil
}

func (f *fakeTokens) ComputerToken(_ context.Context, userID, computerID string) (*MCPToken, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	token, ok := f.active[userID+"/"+computerID]
	if !ok {
		return nil, nil
	}
	return &token, nil
}

func (f *fakeTokens) RevokeComputerToken(_ context.Context, userID, computerID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.revokeErr != nil {
		return f.revokeErr
	}
	if old, ok := f.active[userID+"/"+computerID]; ok {
		f.revoked = append(f.revoked, old.ID)
		delete(f.active, userID+"/"+computerID)
	}
	return nil
}
