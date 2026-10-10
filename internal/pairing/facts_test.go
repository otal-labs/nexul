package pairing

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/otal-labs/nexul/internal/harness"
	"github.com/otal-labs/nexul/internal/platform/crypto"
	"github.com/otal-labs/nexul/internal/platform/eventbus"
)

// factsFixture is alice's paired laptop, whose T3 Code lists one signed-in provider and one project, counting the reads.
type factsFixture struct {
	*runnerFixture
	reads int
}

func newFactsFixture(t *testing.T) *factsFixture {
	t.Helper()
	sealed, err := crypto.Encrypt(testEncKey, []byte("bearer"))
	require.NoError(t, err)
	f := &factsFixture{runnerFixture: newRunnerFixture(t, Computer{Name: "Laptop", BearerToken: sealed, TokenExpiresAt: testNow.Add(20 * day)})}
	f.exch.ListProvidersFn = func(_ context.Context, s harness.Session) ([]harness.Provider, error) {
		f.reads++
		assert.Equal(t, "http://c-laptop.nexul-computer.invalid", s.ServerURL, "read through the runner")
		return []harness.Provider{{ID: "claudeAgent", Driver: "claudeAgent", Name: "Claude", Version: "2.1.0", SignIn: harness.SignedIn,
			Models: []harness.ProviderModel{{Slug: "claude-opus-5-5", Name: "Claude Opus 5.5", Options: []harness.ModelOption{{ID: "effort"}}}}}}, nil
	}
	f.exch.ListProjectsFn = func(context.Context, harness.Session) ([]harness.Project, error) {
		return []harness.Project{{ID: "p1", Title: "app", Path: "/home/alice/code/app"}}, nil
	}
	return f
}

func reportEvent(t *testing.T, facts map[string]any) eventbus.Event {
	t.Helper()
	raw, err := json.Marshal(map[string]any{"runner_id": "r-laptop", "computer_id": "c-laptop", "user_id": "u-alice", "members_only": true, "facts": facts})
	require.NoError(t, err)
	return eventbus.Event{Topic: "runner.facts_reported", Payload: raw}
}

func laptopReport(state string, freeDisk int64) map[string]any {
	return map[string]any{
		"hostname": "alice-laptop", "os": "linux", "arch": "amd64", "runner_version": "v0.3.40", "cloudflared": "2025.9.1",
		"git_name": "Alice Example", "git_email": "alice@example.com", "free_disk_bytes": freeDisk,
		"t3": map[string]any{"state": state, "install": "service", "port": 3773, "version": "0.0.46"},
	}
}

func (f *factsFixture) stored(t *testing.T) Computer {
	t.Helper()
	c, err := f.repo.GetComputer(t.Context(), "u-alice", "c-laptop")
	require.NoError(t, err)
	return *c
}

// TestHandleFactsReported_StoresTheReportAndWhatT3CodeLists: a report is stored whole with the providers and projects
// T3 Code lists through the runner, and computer.facts_changed names the computer without carrying the facts.
func TestHandleFactsReported_StoresTheReportAndWhatT3CodeLists(t *testing.T) {
	t.Parallel()
	f := newFactsFixture(t)

	require.NoError(t, f.svc.HandleFactsReported(t.Context(), reportEvent(t, laptopReport("answering", 5e10))))

	c := f.stored(t)
	require.NotNil(t, c.Facts)
	assert.Equal(t, Facts{
		Hostname: "alice-laptop", OS: "linux", Arch: "amd64", RunnerVersion: "v0.3.40", Cloudflared: "2025.9.1",
		GitName: "Alice Example", GitEmail: "alice@example.com", FreeDiskBytes: 46 << 30,
		T3: T3Facts{State: "answering", Install: "service", Port: 3773, Version: "0.0.46"},
		Providers: []ProviderFacts{{ID: "claudeAgent", Driver: "claudeAgent", Name: "Claude", Version: "2.1.0", SignIn: harness.SignedIn,
			Models: []ModelFacts{{Slug: "claude-opus-5-5", Name: "Claude Opus 5.5"}}}},
		Projects: []harness.Project{{ID: "p1", Title: "app", Path: "/home/alice/code/app"}},
	}, *c.Facts)
	assert.Equal(t, testNow, *c.FactsAt)
	require.Len(t, f.repo.outbox, 1)
	assert.Equal(t, TopicFactsChanged, f.repo.outbox[0].Topic)
	assert.Equal(t, FactsChangedEvent{ComputerID: "c-laptop", UserID: "u-alice", FactsAt: testNow, MembersOnly: true}, f.repo.outbox[0].Payload)
}

// TestHandleFactsReported_AnUnchangedReportWritesNothing: the 6-hourly report of a computer whose facts stay the same
// writes no row and publishes nothing; the first change writes once.
func TestHandleFactsReported_AnUnchangedReportWritesNothing(t *testing.T) {
	t.Parallel()
	f := newFactsFixture(t)
	report := reportEvent(t, laptopReport("answering", 5e10))

	for range 3 {
		require.NoError(t, f.svc.HandleFactsReported(t.Context(), report))
	}
	require.NoError(t, f.svc.HandleFactsReported(t.Context(), reportEvent(t, laptopReport("answering", 5e10-4096))))
	assert.Equal(t, 1, f.repo.factWrites, "only the first report changed anything; free disk counts in whole GiB")
	assert.Len(t, f.repo.outbox, 1)

	require.NoError(t, f.svc.HandleFactsReported(t.Context(), reportEvent(t, laptopReport("answering", 4e10))))
	assert.Equal(t, 2, f.repo.factWrites, "less free disk is a change")
	assert.Len(t, f.repo.outbox, 2)
}

// TestHandleFactsReported_T3CodeNotAnsweringKeepsItsLastListing: a report saying T3 Code stopped replaces the runner's
// part and keeps the providers and projects last read, without asking T3 Code for them.
func TestHandleFactsReported_T3CodeNotAnsweringKeepsItsLastListing(t *testing.T) {
	t.Parallel()
	f := newFactsFixture(t)
	require.NoError(t, f.svc.HandleFactsReported(t.Context(), reportEvent(t, laptopReport("answering", 5e10))))

	stopped := laptopReport("not_running", 5e10)
	stopped["t3"].(map[string]any)["restart_error"] = "the person's user service manager is not running"
	require.NoError(t, f.svc.HandleFactsReported(t.Context(), reportEvent(t, stopped)))

	c := f.stored(t)
	assert.Equal(t, "not_running", c.Facts.T3.State)
	assert.Equal(t, "the person's user service manager is not running", c.Facts.T3.RestartError, "why the runner could not restart it")
	assert.Len(t, c.Facts.Providers, 1)
	assert.Len(t, c.Facts.Projects, 1)
	assert.Equal(t, 1, f.reads, "a stopped T3 Code is not asked")
}

// TestHandleFactsReported_FailedListingKeepsTheReport: when T3 Code answers the runner but its listing fails, the
// runner's report is stored and nothing is returned for the bus to retry.
func TestHandleFactsReported_FailedListingKeepsTheReport(t *testing.T) {
	t.Parallel()
	f := newFactsFixture(t)
	f.exch.ListProvidersFn = func(context.Context, harness.Session) ([]harness.Provider, error) { return nil, errBoom }

	require.NoError(t, f.svc.HandleFactsReported(t.Context(), reportEvent(t, laptopReport("answering", 5e10))))

	c := f.stored(t)
	assert.Equal(t, "alice-laptop", c.Facts.Hostname)
	assert.Empty(t, c.Facts.Providers)
}

// TestHandleFactsReported_SaveFailureIsReturned: a failed write returns its error so the bus retries the report.
func TestHandleFactsReported_SaveFailureIsReturned(t *testing.T) {
	t.Parallel()
	f := newFactsFixture(t)
	f.repo.saveErr = errBoom

	err := f.svc.HandleFactsReported(t.Context(), reportEvent(t, laptopReport("answering", 5e10)))

	require.ErrorIs(t, err, errBoom)
}

// TestPairComputer_ThroughTheRunner_ReadsWhatT3CodeLists: pairing now reads the providers and projects straight away,
// keeping the runner's last report.
func TestPairComputer_ThroughTheRunner_ReadsWhatT3CodeLists(t *testing.T) {
	t.Parallel()
	f := newFactsFixture(t)
	require.NoError(t, f.svc.HandleFactsReported(t.Context(), reportEvent(t, laptopReport("not_running", 5e10))))
	require.Empty(t, f.stored(t).Facts.Providers)

	_, err := f.svc.PairComputer(t.Context(), "u-alice", "c-laptop", "")
	require.NoError(t, err)

	c := f.stored(t)
	assert.Len(t, c.Facts.Providers, 1)
	assert.Equal(t, "alice-laptop", c.Facts.Hostname, "the runner's part stays")
}
