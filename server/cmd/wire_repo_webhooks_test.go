package main

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/gitprovider"
	"github.com/otal-labs/nexul/internal/workspace"
)

const testHookURL = "https://nexul.example/hooks/github"

// fakeHookHost records webhook calls; every other GitProvider method panics through the nil embed.
type fakeHookHost struct {
	gitprovider.GitProvider
	hooks   []gitprovider.Webhook
	listErr error
	created []gitprovider.WebhookConfig
	deleted []string
	calls   []string
	prs     map[int]*gitprovider.PR
}

func (f *fakeHookHost) ListWebhooks(context.Context, string, string) ([]gitprovider.Webhook, error) {
	f.calls = append(f.calls, "list")
	return f.hooks, f.listErr
}

func (f *fakeHookHost) CreateWebhook(_ context.Context, _, _ string, cfg gitprovider.WebhookConfig) (string, error) {
	f.calls = append(f.calls, "create")
	f.created = append(f.created, cfg)
	return "1", nil
}

func (f *fakeHookHost) DeleteWebhook(_ context.Context, _, _, id string) error {
	f.calls = append(f.calls, "delete")
	f.deleted = append(f.deleted, id)
	return nil
}

func (f *fakeHookHost) GetPR(_ context.Context, _, _ string, number int) (*gitprovider.PR, error) {
	f.calls = append(f.calls, "get")
	pr, ok := f.prs[number]
	if !ok {
		return nil, errors.New("not found")
	}
	return pr, nil
}

// fakePublisher records published topics in order, failing every publish when err is set.
type fakePublisher struct {
	topics []string
	events []gitprovider.PREvent
	err    error
}

func (f *fakePublisher) Publish(_ context.Context, topic string, payload any) error {
	if f.err != nil {
		return f.err
	}
	f.topics = append(f.topics, topic)
	f.events = append(f.events, payload.(gitprovider.PREvent))
	return nil
}

func newTestRepoWebhooks(host *fakeHookHost, instanceURL string) repoWebhooks {
	return repoWebhooks{
		git:         host,
		instanceURL: func(context.Context) (string, error) { return instanceURL, nil },
		secret:      "derived",
		openPRs:     func(context.Context, string, string) ([]int, error) { return nil, nil },
		bus:         &fakePublisher{},
	}
}

func TestRepoWebhooksCatchUp(t *testing.T) {
	merged := &gitprovider.PR{Number: 1, State: gitprovider.PRStateClosed, Merged: true}
	closed := &gitprovider.PR{Number: 2, State: gitprovider.PRStateClosed}
	open := &gitprovider.PR{Number: 3, State: gitprovider.PRStateOpen}
	tests := []struct {
		name       string
		open       []int
		openErr    error
		prs        map[int]*gitprovider.PR
		publishErr error
		wantErr    bool
		wantTopics []string
	}{
		{"merged and closed PRs publish their outcome, open ones nothing", []int{1, 2, 3},
			nil, map[int]*gitprovider.PR{1: merged, 2: closed, 3: open}, nil, false,
			[]string{gitprovider.TopicPRMerged, gitprovider.TopicPRClosed}},
		{"one unreadable PR still lets the rest catch up", []int{9, 1},
			nil, map[int]*gitprovider.PR{1: merged}, nil, true, []string{gitprovider.TopicPRMerged}},
		{"nothing linked open asks the provider nothing", nil, nil, nil, nil, false, nil},
		{"listing failure asks the provider nothing", nil, errors.New("db"), nil, nil, true, nil},
		{"publish failure is reported", []int{1}, nil, map[int]*gitprovider.PR{1: merged}, errors.New("bus"), true, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host := &fakeHookHost{prs: tt.prs}
			pub := &fakePublisher{err: tt.publishErr}
			h := newTestRepoWebhooks(host, "https://nexul.example")
			h.openPRs = func(_ context.Context, owner, name string) ([]int, error) {
				assert.Equal(t, "acme/app", owner+"/"+name)
				return tt.open, tt.openErr
			}
			h.bus = pub

			err := h.catchUp(t.Context(), "acme", "app")
			if tt.wantErr {
				require.Error(t, err)
			}
			if !tt.wantErr {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.wantTopics, pub.topics)
			assert.Len(t, host.calls, len(tt.open))
			for _, ev := range pub.events {
				assert.Equal(t, "acme", ev.Owner)
				assert.Equal(t, "app", ev.Repo)
			}
		})
	}
}

func TestRepoWebhooksEnsure(t *testing.T) {
	tests := []struct {
		name        string
		instanceURL string
		host        *fakeHookHost
		wantErr     bool
		wantCreated []gitprovider.WebhookConfig
	}{
		{"no hook yet registers one", "https://nexul.example/", &fakeHookHost{},
			false, []gitprovider.WebhookConfig{{URL: testHookURL, Secret: "derived"}}},
		{"own hook already there is left alone", "https://nexul.example",
			&fakeHookHost{hooks: []gitprovider.Webhook{{ID: "7", URL: testHookURL}}}, false, nil},
		{"another instance's hook does not count", "https://nexul.example",
			&fakeHookHost{hooks: []gitprovider.Webhook{{ID: "8", URL: "https://staging.example/hooks/github"}}},
			false, []gitprovider.WebhookConfig{{URL: testHookURL, Secret: "derived"}}},
		{"unset instance url registers nothing", "", &fakeHookHost{}, true, nil},
		{"list failure registers nothing", "https://nexul.example", &fakeHookHost{listErr: errors.New("forbidden")}, true, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := newTestRepoWebhooks(tt.host, tt.instanceURL).ensure(t.Context(), "acme", "app")
			if tt.wantErr {
				require.Error(t, err)
			}
			if !tt.wantErr {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.wantCreated, tt.host.created)
		})
	}
}

func TestRepoWebhooksRemove_DeletesOnlyThisInstancesHook(t *testing.T) {
	host := &fakeHookHost{hooks: []gitprovider.Webhook{
		{ID: "7", URL: testHookURL},
		{ID: "8", URL: "https://staging.example/hooks/github"},
	}}
	require.NoError(t, newTestRepoWebhooks(host, "https://nexul.example").remove(t.Context(), "acme", "app"))
	assert.Equal(t, []string{"7"}, host.deleted)
}

// fakeProjectRepos records attach and detach into the shared call log, so ordering against the host is visible.
type fakeProjectRepos struct {
	workspace.Repo
	addErr error
	calls  *[]string
}

func (f fakeProjectRepos) AddRepo(context.Context, string, workspace.RepoRef) error {
	*f.calls = append(*f.calls, "attach")
	return f.addErr
}

func (f fakeProjectRepos) RemoveRepo(context.Context, string, string) error {
	*f.calls = append(*f.calls, "detach")
	return nil
}

func TestHookedProjects_AddRepo_WebhookFailureStillAttaches(t *testing.T) {
	host := &fakeHookHost{listErr: errors.New("forbidden")}
	p := hookedProjects{Repo: fakeProjectRepos{calls: &host.calls}, hooks: newTestRepoWebhooks(host, "https://nexul.example")}

	require.NoError(t, p.AddRepo(t.Context(), "p1", workspace.RepoRef{Owner: "acme", Name: "app"}))
	assert.Equal(t, []string{"attach", "list"}, host.calls)
}

func TestHookedProjects_AddRepo_CatchesUpAfterRegistering(t *testing.T) {
	host := &fakeHookHost{prs: map[int]*gitprovider.PR{4: {Number: 4, State: gitprovider.PRStateClosed, Merged: true}}}
	hooks := newTestRepoWebhooks(host, "https://nexul.example")
	hooks.openPRs = func(context.Context, string, string) ([]int, error) { return []int{4}, nil }
	p := hookedProjects{Repo: fakeProjectRepos{calls: &host.calls}, hooks: hooks}

	require.NoError(t, p.AddRepo(t.Context(), "p1", workspace.RepoRef{Owner: "acme", Name: "app"}))
	assert.Equal(t, []string{"attach", "list", "create", "get"}, host.calls)
	assert.Equal(t, []string{gitprovider.TopicPRMerged}, hooks.bus.(*fakePublisher).topics)
}

func TestHookedProjects_AddRepo_FailedAttachRegistersNothing(t *testing.T) {
	host := &fakeHookHost{}
	p := hookedProjects{Repo: fakeProjectRepos{addErr: errors.New("conflict"), calls: &host.calls}, hooks: newTestRepoWebhooks(host, "https://nexul.example")}

	require.Error(t, p.AddRepo(t.Context(), "p1", workspace.RepoRef{Owner: "acme", Name: "app"}))
	assert.Equal(t, []string{"attach"}, host.calls)
}

func TestHookedProjects_RemoveRepo_DeletesTheHookBeforeDetaching(t *testing.T) {
	host := &fakeHookHost{hooks: []gitprovider.Webhook{{ID: "7", URL: testHookURL}}}
	p := hookedProjects{Repo: fakeProjectRepos{calls: &host.calls}, hooks: newTestRepoWebhooks(host, "https://nexul.example")}

	require.NoError(t, p.RemoveRepo(t.Context(), "acme", "app"))
	assert.Equal(t, []string{"list", "delete", "detach"}, host.calls)
}
