package gitprovider

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// fakeBus is an in-process bus for tests: it records published events and synchronously dispatches them to subscribed handlers.
type fakeBus struct {
	mu         sync.Mutex
	published  []eventbus.Event
	publishErr error
	handlers   map[string][]eventbus.Handler
}

func newFakeBus() *fakeBus {
	return &fakeBus{handlers: map[string][]eventbus.Handler{}}
}

func (f *fakeBus) Publish(ctx context.Context, topic string, payload any) error {
	f.mu.Lock()
	if f.publishErr != nil {
		f.mu.Unlock()
		return f.publishErr
	}
	b, _ := json.Marshal(payload)
	ev := eventbus.Event{Topic: topic, Payload: b}
	f.published = append(f.published, ev)
	hs := append([]eventbus.Handler(nil), f.handlers[topic]...)
	f.mu.Unlock()
	for _, h := range hs {
		if err := h(ctx, ev); err != nil {
			return err
		}
	}
	return nil
}

func (f *fakeBus) Subscribe(_ context.Context, topic string, h eventbus.Handler) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.handlers[topic] = append(f.handlers[topic], h)
	return nil
}

func (f *fakeBus) Enqueue(context.Context, string, any) error              { return nil }
func (f *fakeBus) Consume(context.Context, string, eventbus.Handler) error { return nil }
func (f *fakeBus) Close() error                                            { return nil }

func (f *fakeBus) lastPublished() eventbus.Event {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.published) == 0 {
		return eventbus.Event{}
	}
	return f.published[len(f.published)-1]
}

// fakeLinker records ticket link calls for the event handlers.
type fakeLinker struct {
	mu      sync.Mutex
	linked  []string
	linkErr error
}

func (f *fakeLinker) LinkPR(_ context.Context, ticketID string, _ PRRef) error {
	if f.linkErr != nil {
		return f.linkErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.linked = append(f.linked, ticketID)
	return nil
}

// fakeProvider is a scriptable GitProvider for use-case and tool tests.
type fakeProvider struct {
	repo         *Repo
	prs          []*PR
	pr           *PR
	err          error
	hookID       string
	installRepos []*Repo
	tree         []TreeEntry
	file         []byte
}

func (f *fakeProvider) GetRepo(context.Context, string, string) (*Repo, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.repo, nil
}

func (f *fakeProvider) ListPRs(context.Context, string, string, PROpts) ([]*PR, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.prs, nil
}

func (f *fakeProvider) GetPR(context.Context, string, string, int) (*PR, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.pr, nil
}

func (f *fakeProvider) CreateWebhook(context.Context, string, string, WebhookConfig) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.hookID, nil
}

func (f *fakeProvider) DeleteWebhook(context.Context, string, string, string) error {
	return f.err
}

func (f *fakeProvider) ListInstallationRepos(context.Context) ([]*Repo, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.installRepos, nil
}

func (f *fakeProvider) GetTree(context.Context, string, string, string) ([]TreeEntry, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.tree, nil
}

func (f *fakeProvider) GetFile(context.Context, string, string, string, string) ([]byte, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.file, nil
}
